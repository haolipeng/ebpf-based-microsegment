# NeuVector 网络拓扑功能深度调研报告

## 一、产品调研

### 1.1 功能概述

NeuVector 的网络拓扑(Network Activity Map)是其容器安全平台的核心可视化功能,提供了实时的容器间网络连接可视化和应用行为发现能力。

### 1.2 核心产品特性

#### 1.2.1 自动发现与实时可视化
- **零配置发现**: 部署 Enforcer 后自动发现容器基础设施
- **实时更新**: 新的网络流立即反映在拓扑图中,无延迟
- **应用行为学习**: 通过观察容器间的 conversations(网络连接)来学习应用行为

#### 1.2.2 三种安全模式的可视化
NeuVector 的拓扑图根据不同模式显示不同的状态:

1. **Discover 模式** (发现模式)
   - 绿色连线:合法流量
   - 学习阶段,不阻断

2. **Monitor 模式** (监控模式)
   - 绿色连线:允许的流量
   - 黄色/红色连线:违规流量(仅告警,不阻断)

3. **Protect 模式** (保护模式)
   - 绿色连线:允许的流量
   - 红色 X 标记:被阻断的违规流量

#### 1.2.3 交互式拓扑图
- **点击连线查看详情**: 显示应用层协议、客户端/服务端 IP、端口、规则 ID、动作(allow/deny)
- **点击节点查看详情**: 显示容器/服务信息、关联的流量统计
- **违规高亮**: 红色/黄色标记醒目显示安全违规

#### 1.2.4 Layer 7 深度包检测(DPI)
- **协议识别**: 自动识别 HTTP, HTTPS, gRPC, MySQL, PostgreSQL, Redis, MongoDB, Kafka 等 20+ 种协议
- **应用层可见性**: 不仅显示 IP:Port,还显示应用层协议和元数据
- **加密流量检测**: 即使是 HTTPS/TLS 加密流量也能检测数据外泄企图

### 1.3 关键产品优势

#### 对比现有实现的优势:

| 维度 | 我们的实现 | NeuVector |
|------|-----------|-----------|
| **数据粒度** | Flow 记录聚合 | Conversation + DPI (Layer 7) |
| **实时性** | 轮询 API (秒级) | 实时推送 (毫秒级) |
| **安全状态** | 基础的 ALLOW/DENY | Discover/Monitor/Protect 三态,违规可视化 |
| **交互性** | 基础节点/边查看 | 丰富的上下文信息,协议级详情 |
| **自动化** | 手动创建规则 | 自动学习 + 微分段策略生成 |
| **可视化元素** | 节点大小/颜色编码 | 多状态连线(绿/黄/红/X)、动态标注 |

#### NeuVector 的核心竞争力:

1. **零信任微分段自动化**: 从 Discover 模式学习后自动生成白名单策略
2. **实时威胁检测**: DPI 能力可检测 SQL 注入、XSS、敏感数据外泄等
3. **East-West 流量完整可见**: 不仅外部流量,容器间横向流量完全可见
4. **一体化安全**: 拓扑图不仅是可视化,直接关联策略引擎和告警

---

## 二、技术架构调研

### 2.1 数据采集架构

#### 2.1.1 Enforcer 组件(数据源)
```
Container Host
├── NeuVector Enforcer (DaemonSet)
    ├── eBPF Probes
    │   ├── TCP Connection Tracking
    │   ├── Packet Capture
    │   └── Process Monitoring
    ├── Deep Packet Inspection Engine
    │   ├── Protocol Parsers (20+ protocols)
    │   ├── Pattern Matching (Hyperscan)
    │   └── Threat Detection Rules
    └── Data Reporter
        └── gRPC -> Controller
```

#### 2.1.2 数据流
```
1. Kernel Space (eBPF)
   ↓ Connection Events
2. Enforcer User Space (DPI)
   ↓ Enriched Conversations (Layer 7)
3. Controller (Aggregation)
   ↓ Graph Data + Policy Evaluation
4. Manager (UI)
   ↓ Real-time Visualization
```

### 2.2 核心数据结构

#### 2.2.1 Conversation 对象
从源代码推测的数据结构:

```go
type Conversation struct {
    // Connection Identity
    ClientWorkloadID  string    // Source container/workload
    ServerWorkloadID  string    // Dest container/workload
    ClientIP          string
    ServerIP          string
    ServerPort        uint16
    Protocol          string    // tcp/udp/icmp

    // Layer 7 Info
    Application       string    // http, grpc, mysql, etc.
    ApplicationProto  string    // HTTP/1.1, HTTP/2, TLS 1.3

    // State & Metrics
    State             string    // ACTIVE, CLOSED
    FirstSeenAt       time.Time
    LastSeenAt        time.Time
    BytesClientToServer uint64
    BytesServerToClient uint64
    PacketsCount        uint64

    // Security Context
    PolicyAction      string    // allow, deny, violate
    PolicyMode        string    // discover, monitor, protect
    RuleID            uint32
    Violations        []Violation

    // Labels (K8s metadata)
    ClientLabels      map[string]string
    ServerLabels      map[string]string
}
```

#### 2.2.2 Graph 数据结构
```go
type NetworkGraph struct {
    Nodes []GraphNode
    Edges []GraphEdge
}

type GraphNode struct {
    ID          string  // workload/service/external ID
    Type        string  // container, pod, service, host, external
    Name        string
    DisplayName string

    // Security State
    PolicyMode  string  // discover, monitor, protect
    State       string  // running, stopped

    // K8s Context
    Namespace   string
    Labels      map[string]string

    // Metrics
    ActiveConnections int
    Violations        int
}

type GraphEdge struct {
    Source      string  // Node ID
    Destination string  // Node ID

    // Connection Attributes
    Application string
    Ports       []PortInfo

    // Security Status
    Status      string  // normal, warning, violation, blocked
    PolicyAction string
    RuleIDs     []uint32

    // Metrics
    BytesTotal  uint64
    PacketsTotal uint64
    Sessions    int

    // Timestamps
    FirstSeen   time.Time
    LastSeen    time.Time
}
```

### 2.3 API 设计

#### 2.3.1 REST API Endpoints
根据 NeuVector 架构推测的关键 API:

```
GET /v1/conversation
  - Query params:
    - from: 时间范围开始
    - to: 时间范围结束
    - scope: namespace/cluster
    - mode: discover/monitor/protect
  - Response: Conversation[]

GET /v1/graph
  - Query params:
    - scope: namespace/cluster/global
    - depth: 1/2/3 (关系深度)
  - Response: NetworkGraph

GET /v1/conversation/:id
  - Response: ConversationDetail (包含完整 DPI 信息)

WebSocket /v1/conversation/stream
  - Real-time conversation updates
  - Event types: new, update, close, violation
```

#### 2.3.2 数据聚合策略
```
Enforcer (每台 Host)
  └─> 原始 Conversations (每秒数千条)
      └─> Controller Aggregation Layer
          ├─> 按 5-tuple 聚合
          ├─> 应用 K8s 标签
          ├─> Layer 7 协议识别
          └─> 策略评估 (allow/deny/violate)
              └─> Graph Builder
                  ├─> 节点去重 (按 workload/service)
                  ├─> 边聚合 (按 src-dst pair)
                  └─> 计算指标 (bytes/packets/sessions)
                      └─> Manager API Response
```

### 2.4 前端技术栈

#### 2.4.1 可视化库选择
NeuVector Manager 使用的可视化技术(基于社区反馈和截图分析):

**可能的技术栈**:
- **D3.js** 或 **Cytoscape.js**: 用于力导向图
- **AngularJS** (早期版本) 或 **React** (新版本): UI 框架
- **WebSocket**: 实时数据推送

**关键可视化特性**:
```javascript
// 节点渲染
Node Visualization:
  - Size: 根据连接数/流量动态调整
  - Color: 根据 PolicyMode (Discover=蓝, Monitor=黄, Protect=绿)
  - Badge: 显示违规计数
  - Icon: 区分 Container/Pod/Service/Host/External

// 边渲染
Edge Visualization:
  - Line Style:
    - 实线: 允许的流量
    - 虚线: 潜在威胁
    - 红X: 被阻断
  - Color:
    - 绿色: Normal
    - 黄色: Warning (Monitor mode violation)
    - 红色: Blocked (Protect mode violation)
  - Width: 根据流量大小
  - Animation: 流量方向动画

// 交互
Interactions:
  - Click Node: 显示侧边栏详情
  - Click Edge: 显示连接详情弹窗
  - Hover: 显示快速信息 Tooltip
  - Drag: 重新布局
  - Filter: 按namespace/mode/protocol过滤
  - Search: 快速定位节点
  - Zoom/Pan: 大规模拓扑导航
```

#### 2.4.2 布局算法
```javascript
// Force-Directed Layout (力导向布局)
Layout Algorithm:
  - 节点之间的斥力 (防止重叠)
  - 边的引力 (相关节点聚集)
  - 中心引力 (防止分散)
  - 碰撞检测

// 分组优化
Grouping Strategy:
  - 按 Namespace 分区
  - 按 Service 聚类
  - External 节点放置在外圈
```

### 2.5 性能优化

#### 2.5.1 数据采样与聚合
```
Enforcer Level:
  - 每5秒上报一次聚合的 conversation 快照
  - 只上报状态变化的 conversation

Controller Level:
  - In-memory graph cache (TTL: 5min)
  - 增量更新而非全量重建
  - 只下发用户可见范围(namespace)的数据

Frontend Level:
  - 虚拟化渲染 (只渲染视口内节点)
  - 节点数 > 100 时启用简化模式
  - WebWorker 处理图布局计算
```

#### 2.5.2 扩展性设计
```
Small Cluster (< 50 nodes):
  - 全图渲染
  - 1秒更新频率

Medium Cluster (50-500 nodes):
  - 按 Namespace 分页
  - 5秒更新频率
  - 节点聚合 (多个 Pod -> Service 级别)

Large Cluster (> 500 nodes):
  - 必须选择 Namespace/Service
  - 10秒更新频率
  - 高级聚合 (Service -> Application)
```

---

## 三、关键技术亮点总结

### 3.1 架构层面

1. **eBPF + DPI 双引擎**:
   - eBPF 提供高性能连接跟踪
   - DPI 引擎提供应用层可见性
   - 两者结合实现 Zero Trust 微分段

2. **事件驱动架构**:
   - Enforcer 产生事件流
   - Controller 聚合 + 策略引擎
   - Manager WebSocket 实时推送

3. **分布式数据采集**:
   - DaemonSet 部署在每个节点
   - 去中心化采集,中心化聚合
   - 避免单点性能瓶颈

### 3.2 数据模型层面

1. **Conversation 中心模型**:
   - 不是简单的 Flow 记录
   - 包含完整的应用上下文 (Layer 7)
   - 关联安全策略和威胁情报

2. **双向图模型**:
   - Nodes: Workload/Service/External
   - Edges: Conversations with policy context
   - 支持多维度查询 (时间/空间/安全状态)

3. **元数据丰富**:
   - K8s 原生集成 (Labels/Annotations)
   - 自动服务发现
   - 动态更新

### 3.3 可视化层面

1. **多状态可视化**:
   - 不仅仅是连接关系
   - 安全状态一目了然
   - 违规实时高亮

2. **交互式探索**:
   - 点击/悬停/拖拽/缩放
   - 上下文丰富的详情面板
   - 支持复杂过滤和搜索

3. **实时性**:
   - WebSocket 推送
   - 增量更新
   - 毫秒级延迟

---

## 四、改进建议(针对我们的实现)

### 4.1 短期优化 (1-2周)

#### 4.1.1 数据模型增强
```typescript
// 当前: Flow
// 建议: 升级为 Conversation

interface EnhancedFlow extends Flow {
  // 新增字段
  application?: string        // DPI 识别的协议
  policyMode: 'discover' | 'monitor' | 'protect'
  violations: Violation[]
  conversationState: 'active' | 'closed' | 'timeout'
}
```

#### 4.1.2 可视化状态增强
```typescript
// 当前: 基础节点/边
// 建议: 多状态可视化

Edge Status:
  - normal: 绿色实线
  - warning: 黄色虚线 (Monitor 模式违规)
  - blocked: 红色 X 标记 (Protect 模式阻断)
  - learning: 蓝色虚线 (Discover 模式)

Node Badge:
  - violations count
  - mode indicator
```

#### 4.1.3 交互增强
```
1. 边点击显示协议详情
   - 当前: 简单信息
   - 建议: DPI 数据 (如 HTTP method/path, SQL queries)

2. 节点上下文菜单
   - 查看关联策略
   - 快速切换安全模式
   - 跳转到日志/告警

3. 时间轴回放
   - 查看历史拓扑状态
   - 违规事件重现
```

### 4.2 中期优化 (1-2月)

#### 4.2.1 实时推送
```go
// WebSocket 推送架构
type GraphUpdate struct {
    Type      string      // add/update/delete
    NodeID    string
    EdgeID    string
    Delta     interface{} // 只推送变化
    Timestamp time.Time
}

// 前端增量更新
onGraphUpdate(update) {
    if (update.Type === 'add') {
        addNodeToGraph(update.NodeID, update.Delta)
    } else if (update.Type === 'update') {
        updateNode(update.NodeID, update.Delta)
    }
    // 避免全图重绘
}
```

#### 4.2.2 Layer 7 协议识别
```
选项1: 集成 gopacket + Layer 7 解析器
  - HTTP/HTTPS (TLS SNI)
  - gRPC (proto detection)
  - Database protocols (MySQL/PostgreSQL/Redis)

选项2: 使用 BPF + Socket Filter
  - eBPF 程序捕获应用层数据
  - User-space 解析器

选项3: Sidecar + Service Mesh 集成
  - 从 Envoy/Istio 获取 Layer 7 metrics
  - 更丰富的应用层可见性
```

#### 4.2.3 安全模式支持
```go
type PolicyMode string

const (
    ModeDiscover PolicyMode = "discover"  // 学习模式
    ModeMonitor  PolicyMode = "monitor"   // 监控告警
    ModeProtect  PolicyMode = "protect"   // 阻断保护
)

// Agent 实现
func (a *Agent) evaluateFlow(flow *Flow) PolicyAction {
    mode := a.getPolicyMode(flow.Labels)

    switch mode {
    case ModeDiscover:
        a.learnPolicy(flow)  // 自动学习白名单
        return ActionAllow
    case ModeMonitor:
        if a.matchPolicy(flow) {
            return ActionAllow
        }
        a.reportViolation(flow)  // 告警但不阻断
        return ActionAllow
    case ModeProtect:
        if a.matchPolicy(flow) {
            return ActionAllow
        }
        a.reportViolation(flow)
        return ActionDeny  // 阻断
    }
}
```

### 4.3 长期优化 (3-6月)

#### 4.3.1 智能分组与聚类
```
当前: IP/Label 视图
建议:
  - Application 视图 (多服务聚合为应用)
  - Namespace 视图 (K8s 原生)
  - Security Zone 视图 (DMZ/Internal/External)
  - Custom Group 视图 (用户自定义)
```

#### 4.3.2 威胁检测集成
```
在拓扑图上直接显示:
  - DPI 检测到的威胁 (SQL injection, XSS)
  - 异常连接 (从未见过的通信)
  - 数据外泄企图 (大量数据到 External)
  - 横向移动检测 (Container -> Host)
```

#### 4.3.3 策略自动生成
```
从 Discover 模式自动生成策略:
  1. 观察7天流量
  2. 生成白名单规则
  3. UI 中预览和编辑
  4. 一键切换到 Protect 模式
```

---

## 五、技术实现路线图

### Phase 1: 基础增强 (Week 1-2)
- [ ] 数据模型扩展 (EnhancedFlow)
- [ ] 多状态边可视化 (绿/黄/红/X)
- [ ] 改进 Tooltip 和详情面板
- [ ] 添加违规高亮

### Phase 2: 实时能力 (Week 3-4)
- [ ] WebSocket 推送架构
- [ ] 增量图更新算法
- [ ] 前端性能优化 (虚拟化渲染)

### Phase 3: DPI 集成 (Month 2)
- [ ] HTTP/HTTPS 协议识别
- [ ] gRPC 支持
- [ ] Database 协议解析
- [ ] Layer 7 数据展示

### Phase 4: 安全增强 (Month 3)
- [ ] 三模式策略引擎
- [ ] 自动策略学习
- [ ] 威胁检测可视化
- [ ] 告警集成

### Phase 5: 高级特性 (Month 4-6)
- [ ] 多视图支持 (App/NS/Zone)
- [ ] 时间轴回放
- [ ] 策略模拟器
- [ ] 合规性报告

---

## 六、参考资料

### 6.1 官方文档
- NeuVector Documentation: https://open-docs.neuvector.com/
- NeuVector GitHub: https://github.com/neuvector/neuvector

### 6.2 技术博客
- "Deep Packet Inspection for Container Security"
- "Zero Trust Microsegmentation with NeuVector"
- "eBPF-based Network Monitoring"

### 6.3 相关技术
- Cilium (eBPF + Network Policy)
- Istio/Envoy (Service Mesh + Layer 7)
- Falco (eBPF + Security Events)

---

## 七、结论

NeuVector 的网络拓扑功能是一个**产品级、企业级**的实现,核心优势在于:

1. **深度集成**: 不是单纯的可视化工具,而是与安全策略引擎深度集成
2. **Layer 7 可见性**: DPI 能力提供了应用层洞察
3. **实时性**: 毫秒级更新,真正的实时监控
4. **自动化**: 从发现到保护的自动化流程
5. **可操作性**: 不仅看到,还能直接操作 (切换模式、生成策略)

我们当前的实现是一个**良好的MVP**,但要达到 NeuVector 的水平,需要在以下方面持续投入:
- 数据丰富度 (Layer 7 DPI)
- 实时性 (WebSocket 推送)
- 安全集成 (策略引擎)
- 可操作性 (自动化工作流)

建议采用**渐进式演进策略**,按照上述路线图分阶段实现,最终达到企业级网络拓扑可视化和零信任微分段平台的目标。
