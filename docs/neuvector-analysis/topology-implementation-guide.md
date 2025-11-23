# 网络拓扑可视化 - 实现指南

> 基于 NeuVector 源码分析的实现参考，专注于核心数据结构和实现细节

---

## 1. 核心数据结构

### 1.1 图的边类型

```go
// connect.go:29-37
const (
    dummyEP    = "Dummy"      // 虚拟端点，用于节点属性存储
    policyLink = "policy"     // 策略学习边
    graphLink  = "graph"      // 流量会话边
    attrLink   = "attr"       // 节点属性边
)
```

**三种边的用途**：

| 边类型 | 源 → 目标 | 属性类型 | 用途 |
|--------|----------|---------|------|
| `policy` | workload → workload | `polAttr` | 存储学习到的策略（端口、应用） |
| `graph` | workload → workload | `graphAttr` | 存储流量会话统计 |
| `attr` | workload → "Dummy" | `nodeAttr` | 存储节点元信息 |

### 1.2 流量会话数据结构

```go
// connect.go:51-85 - 详细条目
type graphEntry struct {
    bytes        uint64    // 累计字节数
    sessions     uint32    // 累计会话数
    server       uint32    // 服务端应用 ID
    threatID     uint32    // 威胁 ID
    dlpID        uint32    // DLP 规则 ID
    wafID        uint32    // WAF 规则 ID
    mappedPort   uint16    // 映射端口（NAT）
    severity     uint8     // 威胁严重性
    dlpSeverity  uint8
    wafSeverity  uint8
    policyAction uint8     // 策略动作
    policyID     uint32    // 匹配的策略 ID
    last         uint32    // 最后见时间（Unix 时间戳）
    xff          uint8     // X-Forwarded-For 标记
    toSidecar    uint8     // Sidecar 流量标记
    fqdn         string    // 服务端域名（出站方向）
    nbe          uint8     // 命名空间边界标记
}

// connect.go:71-77 - 条目索引 Key
type graphKey struct {
    port        uint16    // 端口
    ipproto     uint8     // 协议 (TCP=6, UDP=17)
    application uint32    // 应用 ID (HTTP, MySQL 等)
    cip         uint32    // 客户端 IP (uint32 形式)
    sip         uint32    // 服务端 IP (uint32 形式)
}

// connect.go:79-85 - 会话聚合属性
type graphAttr struct {
    bytes        uint64                    // 总字节数
    sessions     uint32                    // 总会话数
    severity     uint8                     // 最高威胁等级
    policyAction uint8                     // 聚合策略动作
    entries      map[graphKey]*graphEntry  // 详细条目 (5 元组索引)
}
```

### 1.3 策略学习数据结构

```go
// connect.go:87-92
type polAttr struct {
    ports        utils.Set  // 学习到的端口集合 (string: "tcp/80")
    portsSeen    utils.Set  // 已见过的端口（用于去重）
    apps         utils.Set  // 学习到的应用集合 (uint32)
    lastRecalcAt int64      // 上次重算时间
}
```

### 1.4 节点属性

```go
// connect.go:94-103
type nodeAttr struct {
    external bool     // 外部节点（公网 IP）
    workload bool     // 工作负载（容器/Pod）
    host     bool     // 主机节点
    managed  bool     // 被管理的节点
    addrgrp  bool     // 地址组
    ipsvcgrp bool     // IP 服务组
    hostID   string   // 所属主机 ID
    alias    string   // 用户设置的别名
}
```

---

## 2. 数据流架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                         数据采集层                                   │
├─────────────────────────────────────────────────────────────────────┤
│  eBPF/DP 层                                                         │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ DPMsgSession (C 结构体)                                      │   │
│  │ - 5 元组 (srcIP, dstIP, srcPort, dstPort, proto)            │   │
│  │ - 统计 (bytes, packets, sessions)                           │   │
│  │ - 应用识别 (Application ID)                                  │   │
│  │ - 策略匹配 (PolicyId, PolicyAction)                          │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              ↓                                       │
│                     Unix Socket (Binary)                             │
│                              ↓                                       │
│  Agent 层                                                            │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ CLUSConnection (Protobuf)                                    │   │
│  │ - 工作负载 ID (ClientWL, ServerWL)                           │   │
│  │ - 网络信息 (IP, Port, Proto)                                 │   │
│  │ - 统计数据 (Bytes, Sessions)                                 │   │
│  │ - 策略信息 (PolicyId, PolicyAction)                          │   │
│  │ - 元数据 (FQDN, Xff, Nbe)                                    │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              ↓                                       │
│                      gRPC (批量上报)                                 │
│                              ↓                                       │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                         图数据存储层                                 │
├─────────────────────────────────────────────────────────────────────┤
│  Controller 层                                                       │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ Graph (内存图数据库)                                         │   │
│  │                                                              │   │
│  │   Node: workload-A          Node: workload-B                │   │
│  │   ┌──────────────┐          ┌──────────────┐                │   │
│  │   │ ins:         │          │ ins:         │                │   │
│  │   │   policy: {} │◄─────────│   policy: {} │                │   │
│  │   │   graph: {}  │◄─────────│   graph: {}  │                │   │
│  │   │ outs:        │          │ outs:        │                │   │
│  │   │   policy: {} │─────────►│   policy: {} │                │   │
│  │   │   graph: {}  │─────────►│   graph: {}  │                │   │
│  │   │   attr: {}   │          │   attr: {}   │                │   │
│  │   └──────────────┘          └──────────────┘                │   │
│  │                                                              │   │
│  │ 边属性:                                                      │   │
│  │   graphLink: graphAttr (bytes, sessions, entries[])         │   │
│  │   policyLink: polAttr (ports, apps)                         │   │
│  │   attrLink: nodeAttr (type, alias)                          │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────┐
│                         API 层                                       │
├─────────────────────────────────────────────────────────────────────┤
│  REST API                                                            │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │ GET /v1/conversation_endpoint     获取所有端点              │   │
│  │ GET /v1/conversation              获取所有会话              │   │
│  │ POST /v1/conversation             查询会话详情              │   │
│  └─────────────────────────────────────────────────────────────┘   │
│                              ↓                                       │
│  响应格式:                                                           │
│  {                                                                   │
│    "endpoints": [                                                    │
│      {"id": "...", "name": "...", "kind": "workload|host|external"}│
│    ],                                                                │
│    "conversations": [                                                │
│      {                                                               │
│        "from": {...}, "to": {...},                                  │
│        "bytes": 12345, "sessions": 100,                             │
│        "applications": ["HTTP", "MySQL"],                           │
│        "ports": ["tcp/80", "tcp/3306"],                             │
│        "policy_action": "allow|deny|learn",                         │
│        "entries": [...]                                              │
│      }                                                               │
│    ]                                                                 │
│  }                                                                   │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 3. 核心算法

### 3.1 连接数据处理 (addConnectToGraph)

```go
// learn.go:166-455 - 核心逻辑简化版
func addConnectToGraph(conn *CLUSConnection, ca, sa *nodeAttr, stip *serverTip) {
    // 1. 添加节点属性
    if wlGraph.Attr(conn.ClientWL, attrLink, dummyEP) == nil {
        wlGraph.AddLink(conn.ClientWL, attrLink, dummyEP, ca)
    }
    if wlGraph.Attr(conn.ServerWL, attrLink, dummyEP) == nil {
        wlGraph.AddLink(conn.ServerWL, attrLink, dummyEP, sa)
    }

    // 2. 策略学习（根据 PolicyAction）
    switch conn.PolicyAction {
    case DP_POLICY_ACTION_LEARN:
        // 学习端口或应用规则
        learnPolicyRule(conn)
    case DP_POLICY_ACTION_ALLOW:
        // 尝试用应用规则替换端口规则
        upgradeToAppRule(conn)
    case DP_POLICY_ACTION_VIOLATE, DP_POLICY_ACTION_DENY:
        // 记录违规
        violationUpdate(conn)
    }

    // 3. 构建索引 Key (5 元组)
    gkey := graphKey{
        ipproto:     uint8(conn.IPProto),
        port:        stip.wlPort,
        application: conn.Application,
        cip:         IPv42Int(conn.ClientIP),
        sip:         IPv42Int(conn.ServerIP),
    }

    // 4. 获取或创建会话属性
    var attr *graphAttr
    if a := wlGraph.Attr(conn.ClientWL, graphLink, conn.ServerWL); a != nil {
        attr = a.(*graphAttr)
    } else {
        attr = &graphAttr{entries: make(map[graphKey]*graphEntry)}
    }

    // 5. 更新统计数据（累加）
    attr.bytes += conn.Bytes
    attr.sessions += conn.Sessions
    if conn.Severity > attr.severity {
        attr.severity = conn.Severity
    }

    // 6. 更新或创建详细条目
    if ge, ok := attr.entries[gkey]; ok {
        // 累加已有条目
        ge.bytes += conn.Bytes
        ge.sessions += conn.Sessions
        ge.last = conn.LastSeenAt
    } else {
        // 创建新条目
        attr.entries[gkey] = &graphEntry{
            bytes:        conn.Bytes,
            sessions:     conn.Sessions,
            policyAction: conn.PolicyAction,
            policyID:     conn.PolicyId,
            last:         conn.LastSeenAt,
            fqdn:         conn.FQDN,
        }
    }

    // 7. 清理过期条目（保持最多 1000 条）
    removeOldEntries(attr, 1000, 100)

    // 8. 更新图
    wlGraph.AddLink(conn.ClientWL, graphLink, conn.ServerWL, attr)
}
```

### 3.2 条目清理策略

```go
// learn.go:89-125
func removeOldEntries(a *graphAttr, max, count int) bool {
    total := len(a.entries)
    if total <= max {
        return false
    }

    // 1. 转换为切片并按时间排序
    entries := make([]*GraphSyncEntry, len(a.entries))
    k := 0
    for key, entry := range a.entries {
        entries[k] = graphEntry2Sync(&key, entry)
        k++
    }
    sort.Slice(entries, func(i, j int) bool {
        return entries[i].Last < entries[j].Last  // 按最后见时间升序
    })

    // 2. 删除最旧的 count 条
    for i := 0; i < count; i++ {
        gkey, _ := graphSync2Entry(entries[i])
        delete(a.entries, *gkey)
    }

    // 3. 重算聚合数据
    recalcConversation(a)
    return true
}
```

### 3.3 策略学习逻辑

```go
// 端口规则 → 应用规则 升级
func upgradeToAppRule(conn *CLUSConnection) {
    attr := wlGraph.Attr(conn.ClientWL, policyLink, conn.ServerWL)
    if attr == nil || conn.Application == 0 {
        return
    }

    polAttr := attr.(*polAttr)
    port := fmt.Sprintf("%s/%d", getProtoName(conn.IPProto), conn.ServerPort)

    // 如果已有端口规则，且检测到应用，则升级
    if polAttr.ports.Contains(port) && !polAttr.apps.Contains(conn.Application) {
        // 添加应用规则
        polAttr.apps.Add(conn.Application)
        learnAppPort(conn.ClientWL, conn.ServerWL, &conn.Application, nil)

        // 删除端口规则
        polAttr.ports.Remove(port)
        unlearnAppPort(conn.ClientWL, conn.ServerWL, nil, &port)
    }
}
```

---

## 4. API 实现

### 4.1 REST 端点

```go
// conver.go - API 路由注册（推测）
router.GET("/v1/conversation_endpoint", handlerConverEndpointList)
router.GET("/v1/conversation_endpoint/:id", handlerConverEndpointShow)
router.PATCH("/v1/conversation_endpoint/:id", handlerConverEndpointConfig)
router.DELETE("/v1/conversation_endpoint/:id", handlerConverEndpointDelete)
router.GET("/v1/conversation", handlerConverList)
router.POST("/v1/conversation", handlerConverShow)
router.DELETE("/v1/conversation", handlerConverDeleteAll)
router.DELETE("/v1/conversation/:from/:to", handlerConverDelete)
```

### 4.2 获取端点列表

```go
// conver.go:20-78
func handlerConverEndpointList(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    acc, login := getAccessControl(w, r, "")

    query := restParseQuery(r)

    // 获取视图类型（pod 或 workload）
    var view string
    if value, ok := query.pairs[api.QueryKeyView]; ok && value == api.QueryValueViewPod {
        view = api.QueryValueViewPod
    }

    // 从缓存获取端点
    eps := cacher.GetAllConverEndpoints(view, acc)

    // 分页和过滤
    var resp api.RESTConversationEndpointData
    if len(query.filters) > 0 {
        rf := restNewFilter(&api.RESTConversationEndpoint{}, query.filters)
        for _, ep := range eps[query.start:] {
            if rf.Filter(ep) {
                resp.Endpoints = append(resp.Endpoints, ep)
                if query.limit > 0 && len(resp.Endpoints) >= query.limit {
                    break
                }
            }
        }
    } else {
        resp.Endpoints = eps[query.start:end]
    }

    restRespSuccess(w, r, &resp, acc, login, nil, "Get conversation endpoint list")
}
```

### 4.3 获取会话列表

```go
// conver.go:158-250
func handlerConverList(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    acc, login := getAccessControl(w, r, "")
    query := restParseQuery(r)

    // 支持按组和命名空间过滤
    var groupFilter, domainFilter string
    for _, f := range query.filters {
        if f.tag == api.FilterByGroup {
            groupFilter = f.value
        } else if f.tag == api.FilterByDomain {
            domainFilter = f.value
        }
    }

    // 获取会话和端点
    convers, endpoints := cacher.GetAllApplicationConvers(groupFilter, domainFilter, acc)

    // 构建响应
    resp := api.RESTConversationsVerboseData{
        Endpoints: endpoints,
        Convers:   make([]*api.RESTConversation, len(convers)),
    }

    epMap := make(map[string]*api.RESTConversationEndpoint)
    for _, ep := range endpoints {
        epMap[ep.ID] = ep
    }

    for i, conver := range convers {
        resp.Convers[i] = &api.RESTConversation{
            From:                   epMap[conver.From],
            To:                     epMap[conver.To],
            RESTConversationReport: conver.RESTConversationReport,
        }
    }

    restRespSuccess(w, r, &resp, acc, login, nil, "Get conversation list")
}
```

---

## 5. 你的项目实现建议

### 5.1 简化的数据结构

```go
// topology/types.go

// 会话统计
type SessionStats struct {
    Bytes       uint64    `json:"bytes"`
    Packets     uint64    `json:"packets"`
    Sessions    uint32    `json:"sessions"`
    LastSeenAt  time.Time `json:"last_seen_at"`
    FirstSeenAt time.Time `json:"first_seen_at"`
}

// 会话 Key（5 元组）
type SessionKey struct {
    SrcIP    string `json:"src_ip"`
    DstIP    string `json:"dst_ip"`
    SrcPort  uint16 `json:"src_port"`
    DstPort  uint16 `json:"dst_port"`
    Protocol uint8  `json:"protocol"`
}

// 会话详情
type SessionEntry struct {
    Key          SessionKey  `json:"key"`
    Stats        SessionStats `json:"stats"`
    Application  string      `json:"application,omitempty"`
    PolicyID     uint32      `json:"policy_id,omitempty"`
    PolicyAction string      `json:"policy_action"`
}

// 拓扑边（聚合）
type TopologyEdge struct {
    From         string         `json:"from"`
    To           string         `json:"to"`
    TotalBytes   uint64         `json:"total_bytes"`
    TotalSessions uint32        `json:"total_sessions"`
    LastSeenAt   time.Time      `json:"last_seen_at"`
    Applications []string       `json:"applications"`
    Ports        []string       `json:"ports"`
    PolicyAction string         `json:"policy_action"`
    Entries      []SessionEntry `json:"entries,omitempty"` // 详情查询时返回
}

// 拓扑节点
type TopologyNode struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Kind        string            `json:"kind"` // workload, host, external
    Labels      map[string]string `json:"labels,omitempty"`
    Namespace   string            `json:"namespace,omitempty"`
}
```

### 5.2 简化的图实现

```go
// topology/graph.go

type NetworkGraph struct {
    mu    sync.RWMutex
    nodes map[string]*TopologyNode
    edges map[string]*TopologyEdge  // key: "from->to"
}

func NewNetworkGraph() *NetworkGraph {
    return &NetworkGraph{
        nodes: make(map[string]*TopologyNode),
        edges: make(map[string]*TopologyEdge),
    }
}

func (g *NetworkGraph) AddSession(sess *SessionEntry, from, to *TopologyNode) {
    g.mu.Lock()
    defer g.mu.Unlock()

    // 添加节点
    if _, ok := g.nodes[from.ID]; !ok {
        g.nodes[from.ID] = from
    }
    if _, ok := g.nodes[to.ID]; !ok {
        g.nodes[to.ID] = to
    }

    // 添加或更新边
    edgeKey := fmt.Sprintf("%s->%s", from.ID, to.ID)
    if edge, ok := g.edges[edgeKey]; ok {
        // 累加统计
        edge.TotalBytes += sess.Stats.Bytes
        edge.TotalSessions += sess.Stats.Sessions
        if sess.Stats.LastSeenAt.After(edge.LastSeenAt) {
            edge.LastSeenAt = sess.Stats.LastSeenAt
        }
        // 添加应用和端口
        edge.Applications = addUnique(edge.Applications, sess.Application)
        edge.Ports = addUnique(edge.Ports, fmt.Sprintf("%s/%d",
            getProtoName(sess.Key.Protocol), sess.Key.DstPort))
        edge.Entries = append(edge.Entries, *sess)
    } else {
        g.edges[edgeKey] = &TopologyEdge{
            From:          from.ID,
            To:            to.ID,
            TotalBytes:    sess.Stats.Bytes,
            TotalSessions: sess.Stats.Sessions,
            LastSeenAt:    sess.Stats.LastSeenAt,
            Applications:  []string{sess.Application},
            Ports:         []string{fmt.Sprintf("%s/%d", getProtoName(sess.Key.Protocol), sess.Key.DstPort)},
            PolicyAction:  sess.PolicyAction,
            Entries:       []SessionEntry{*sess},
        }
    }
}

func (g *NetworkGraph) GetNodes() []*TopologyNode {
    g.mu.RLock()
    defer g.mu.RUnlock()

    nodes := make([]*TopologyNode, 0, len(g.nodes))
    for _, n := range g.nodes {
        nodes = append(nodes, n)
    }
    return nodes
}

func (g *NetworkGraph) GetEdges() []*TopologyEdge {
    g.mu.RLock()
    defer g.mu.RUnlock()

    edges := make([]*TopologyEdge, 0, len(g.edges))
    for _, e := range g.edges {
        edges = append(edges, e)
    }
    return edges
}

func (g *NetworkGraph) CleanupStale(maxAge time.Duration) int {
    g.mu.Lock()
    defer g.mu.Unlock()

    now := time.Now()
    removed := 0

    for key, edge := range g.edges {
        if now.Sub(edge.LastSeenAt) > maxAge {
            delete(g.edges, key)
            removed++
        }
    }

    // 清理孤立节点
    usedNodes := make(map[string]bool)
    for _, edge := range g.edges {
        usedNodes[edge.From] = true
        usedNodes[edge.To] = true
    }
    for id := range g.nodes {
        if !usedNodes[id] {
            delete(g.nodes, id)
        }
    }

    return removed
}
```

### 5.3 REST API 实现

```go
// api/topology.go

func (h *Handler) GetTopologyNodes(c *gin.Context) {
    nodes := h.graph.GetNodes()
    c.JSON(http.StatusOK, gin.H{
        "nodes": nodes,
    })
}

func (h *Handler) GetTopologyEdges(c *gin.Context) {
    edges := h.graph.GetEdges()

    // 过滤参数
    namespace := c.Query("namespace")
    if namespace != "" {
        filtered := make([]*TopologyEdge, 0)
        for _, e := range edges {
            if h.nodeInNamespace(e.From, namespace) || h.nodeInNamespace(e.To, namespace) {
                filtered = append(filtered, e)
            }
        }
        edges = filtered
    }

    c.JSON(http.StatusOK, gin.H{
        "edges": edges,
    })
}

func (h *Handler) GetTopology(c *gin.Context) {
    nodes := h.graph.GetNodes()
    edges := h.graph.GetEdges()

    // 不返回详细条目，减少响应大小
    for _, e := range edges {
        e.Entries = nil
    }

    c.JSON(http.StatusOK, gin.H{
        "nodes": nodes,
        "edges": edges,
    })
}

func (h *Handler) GetEdgeDetail(c *gin.Context) {
    from := c.Param("from")
    to := c.Param("to")

    edge := h.graph.GetEdge(from, to)
    if edge == nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "edge not found"})
        return
    }

    c.JSON(http.StatusOK, edge)
}
```

---

## 6. 前端实现参考

### 6.1 使用 D3.js 力导向图

```javascript
// topology.js

async function fetchTopology() {
    const response = await fetch('/api/v1/topology');
    return response.json();
}

function renderTopology(data) {
    const { nodes, edges } = data;

    // 转换为 D3 格式
    const d3Nodes = nodes.map(n => ({
        id: n.id,
        name: n.name,
        kind: n.kind,
        namespace: n.namespace
    }));

    const d3Links = edges.map(e => ({
        source: e.from,
        target: e.to,
        bytes: e.total_bytes,
        sessions: e.total_sessions,
        apps: e.applications
    }));

    // 创建 SVG
    const svg = d3.select('#topology')
        .append('svg')
        .attr('width', width)
        .attr('height', height);

    // 力导向模拟
    const simulation = d3.forceSimulation(d3Nodes)
        .force('link', d3.forceLink(d3Links).id(d => d.id).distance(100))
        .force('charge', d3.forceManyBody().strength(-300))
        .force('center', d3.forceCenter(width/2, height/2));

    // 渲染边
    const link = svg.selectAll('.link')
        .data(d3Links)
        .enter()
        .append('line')
        .attr('class', 'link')
        .attr('stroke-width', d => Math.log(d.bytes + 1) / 10)
        .attr('stroke', '#999')
        .on('click', (event, d) => showEdgeDetail(d));

    // 渲染节点
    const node = svg.selectAll('.node')
        .data(d3Nodes)
        .enter()
        .append('g')
        .attr('class', 'node')
        .call(d3.drag()
            .on('start', dragStarted)
            .on('drag', dragged)
            .on('end', dragEnded));

    node.append('circle')
        .attr('r', 10)
        .attr('fill', d => getNodeColor(d.kind));

    node.append('text')
        .text(d => d.name)
        .attr('dx', 12)
        .attr('dy', 4);

    // 更新位置
    simulation.on('tick', () => {
        link.attr('x1', d => d.source.x)
            .attr('y1', d => d.source.y)
            .attr('x2', d => d.target.x)
            .attr('y2', d => d.target.y);

        node.attr('transform', d => `translate(${d.x},${d.y})`);
    });
}

function getNodeColor(kind) {
    const colors = {
        'workload': '#4CAF50',
        'host': '#2196F3',
        'external': '#FF9800'
    };
    return colors[kind] || '#9E9E9E';
}

async function showEdgeDetail(edge) {
    const detail = await fetch(`/api/v1/topology/edge/${edge.source.id}/${edge.target.id}`);
    const data = await detail.json();

    // 显示详情面板
    document.getElementById('edge-detail').innerHTML = `
        <h3>${edge.source.name} → ${edge.target.name}</h3>
        <p>Bytes: ${formatBytes(data.total_bytes)}</p>
        <p>Sessions: ${data.total_sessions}</p>
        <p>Applications: ${data.applications.join(', ')}</p>
        <p>Ports: ${data.ports.join(', ')}</p>
        <table>
            <tr><th>Src IP</th><th>Dst IP</th><th>Port</th><th>Bytes</th></tr>
            ${data.entries.map(e => `
                <tr>
                    <td>${e.key.src_ip}</td>
                    <td>${e.key.dst_ip}</td>
                    <td>${e.key.dst_port}</td>
                    <td>${formatBytes(e.stats.bytes)}</td>
                </tr>
            `).join('')}
        </table>
    `;
}
```

---

## 7. 关键设计决策

| 设计点 | NeuVector 方案 | 推荐方案 |
|--------|---------------|---------|
| 图存储 | 内存 Map | 内存 Map（小规模）/ Redis（大规模） |
| 数据聚合 | 三层：entry → attr → graph | 两层：entry → edge 即可 |
| 条目上限 | 每边 1000 条，LRU 清理 | 可调整，按时间或数量 |
| 更新频率 | Agent 10s 批量上报 | 5-10s 批量上报 |
| 过期清理 | 5 分钟无活动 | 可配置 |
| API 分页 | 支持 start/limit | 必须支持 |

---

**文档整理时间**: 2025-11-22
**参考版本**: NeuVector v5.x
