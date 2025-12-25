# NeuVector 策略引擎深度分析

> 本文档分析 NeuVector 的网络策略计算、规则 ID 分配和图数据库实现

---

## 📚 文档概述

本文档覆盖三个核心模块：
1. **agent/policy/network.go** - 网络策略计算引擎
2. **controller/ruleid/rule_uuid.go** - 规则 UUID 管理
3. **controller/graph/graph.go** - 图数据库实现

---

## 1. 网络策略计算引擎 (network.go)

### 1.1 核心数据结构

```go
// 工作负载 IP 策略信息
type WorkloadIPPolicyInfo struct {
    RuleMap    map[string]*dp.DPPolicyIPRule  // 规则去重映射
    Policy     dp.DPWorkloadIPPolicy          // 下发给 DP 的策略
    Configured bool                            // 是否已配置
    SkipPush   bool                            // 是否跳过推送
    HostMode   bool                            // 是否是主机模式容器
    CapIntcp   bool                            // 是否可拦截
    PolVer     uint16                          // 策略版本
    Nbe        bool                            // 命名空间边界强制
}

// 策略引擎
type Engine struct {
    NetworkPolicy     map[string]*WorkloadIPPolicyInfo  // workload ID -> 策略
    ProcessPolicy     map[string]*share.CLUSProcessProfile
    HostID            string
    HostIPs           utils.Set
    PolicyAddrMap     map[string]share.CLUSSubnet      // 策略地址映射
    HostPolicyAddrMap map[string]share.CLUSSubnet
    PolTimerWheel     *utils.TimerWheel
    PolDomNBEMap      map[string]bool                  // 命名空间边界映射
}
```

### 1.2 策略模式 (Policy Mode)

NeuVector 支持三种策略模式：

| 模式 | 说明 | 默认动作 |
|------|------|---------|
| `Learn` | 学习模式 | `LEARN` - 记录但允许 |
| `Evaluate` | 评估模式 | `VIOLATE` - 记录违规但不阻断 |
| `Enforce` | 强制模式 | `DENY` - 阻断未授权流量 |

```go
// 模式到默认动作的映射
func policyModeToDefaultAction(mode string, capIntcp bool) uint8 {
    switch mode {
    case share.PolicyModeLearn:
        return C.DP_POLICY_ACTION_LEARN
    case share.PolicyModeEvaluate:
        return C.DP_POLICY_ACTION_VIOLATE
    case share.PolicyModeEnforce:
        if capIntcp {
            return C.DP_POLICY_ACTION_DENY
        } else {
            return C.DP_POLICY_ACTION_VIOLATE  // 无法拦截时降级为违规
        }
    }
    return C.DP_POLICY_ACTION_OPEN  // 未知模式则放行
}
```

### 1.3 动作调整逻辑 (Action Adjustment)

这是 NeuVector 的核心创新点 - **混合模式下的智能动作调整**：

```go
func adjustAction(action uint8, from, to *share.CLUSWorkloadAddr, id uint32) uint8 {
    fromMode := from.PolicyMode
    toMode := to.PolicyMode

    switch fromMode {
    case share.PolicyModeLearn:
        // 学习模式：DENY -> VIOLATE (不阻断)
        if action == C.DP_POLICY_ACTION_DENY {
            adjustedAction = C.DP_POLICY_ACTION_VIOLATE
        } else if id >= share.PolicyLearnedIDBase && id < share.PolicyFedRuleIDBase {
            // 学习到的规则标记为 LEARN
            adjustedAction = C.DP_POLICY_ACTION_LEARN
        }
    case share.PolicyModeEvaluate:
        // 评估模式：DENY -> VIOLATE
        if action == C.DP_POLICY_ACTION_DENY {
            adjustedAction = C.DP_POLICY_ACTION_VIOLATE
        }
    case share.PolicyModeEnforce:
        // 强制模式：目标是学习模式时，标记为 LEARN
        if toMode == share.PolicyModeLearn && id >= share.PolicyLearnedIDBase {
            adjustedAction = C.DP_POLICY_ACTION_LEARN
        }
    }
    return adjustedAction
}
```

**关键洞察**：
- 动作不仅取决于规则本身，还取决于 **源和目标的模式组合**
- 学习模式的工作负载永远不会被阻断（DENY → VIOLATE）
- 支持**渐进式部署**：可以逐步从学习切换到强制

### 1.4 命名空间边界强制 (Namespace Boundary Enforcement)

```go
func adjustActionNs(action uint8, from, to *share.CLUSWorkloadAddr, id uint32) uint8 {
    // 跨命名空间流量检查
    if fromDomain != toDomain {  // 不同 namespace
        adjustedAction = C.DP_POLICY_ACTION_CHECK_NBE
    }

    // 特殊情况：外部流量、地址组、FQDN 都视为跨命名空间
    if to.WlID == share.CLUSWLExternal ||
       to.WlID == share.CLUSWLAddressGroup ||
       isWorkloadFqdn(to.WlID) {
        adjustedAction = C.DP_POLICY_ACTION_CHECK_NBE
    }
    return adjustedAction
}
```

### 1.5 IP 规则创建流程

```go
func createIPRule(from, to, fromR, toR net.IP, portApps []share.CLUSPortApp,
                  action uint8, pInfo *WorkloadIPPolicyInfo, ctx *ruleContext) {
    // 1. 生成唯一 key 用于去重
    key := fmt.Sprintf("%v%v%s%s%d", from, to, ap, ctx.fqdn, ingress)

    // 2. 检查规则是否已存在
    if existRule, ok := pInfo.RuleMap[key]; ok {
        // 已存在则合并应用层规则
        if existRule.Action == C.DP_POLICY_ACTION_CHECK_APP {
            existRule.Apps = append(existRule.Apps, appRule)
        }
        return
    }

    // 3. 解析端口范围
    proto, p, pr, err := utils.ParsePortRangeLink(ap)

    // 4. 创建规则
    rule := dp.DPPolicyIPRule{
        ID:      id,
        SrcIP:   from,
        DstIP:   to,
        SrcIPR:  fromR,   // IP 范围结束
        DstIPR:  toR,
        Port:    p,
        PortR:   pr,      // 端口范围结束
        IPProto: proto,
        Action:  action,
        Ingress: ctx.ingress,
        Fqdn:    ctx.fqdn,
        Vhost:   ctx.vhost,
    }

    // 5. 添加应用层检测
    if portApp.CheckApp {
        rule.Apps = append(rule.Apps, &dp.DPPolicyApp{
            App:    portApp.Application,
            Action: action,
            RuleID: id,
        })
        rule.Action = C.DP_POLICY_ACTION_CHECK_APP
    }

    pInfo.Policy.IPRules = append(pInfo.Policy.IPRules, &rule)
    pInfo.RuleMap[key] = &rule
}
```

### 1.6 FQDN 处理

```go
// FQDN 信息缓存
var fqdnMap map[string]*fqdnInfo = make(map[string]*fqdnInfo)

type fqdnInfo struct {
    ips  []net.IP
    used bool  // 标记是否仍在使用
}

// 获取 FQDN 对应的 IP（带缓存和 DNS 解析）
func getFqdnIP(name string) []net.IP {
    if info, ok := fqdnMap[name]; ok {
        info.used = true
        return info.ips
    }

    // 通配符处理
    if strings.HasPrefix(name, "*") {
        ret = append(ret, net.IPv4zero)  // 占位符
    } else {
        ips, err := utils.ResolveIP(name)  // DNS 解析
        // ...
    }
    fqdnMap[name] = &fqdnInfo{ips: ret, used: true}
    return ret
}

// 策略计算后清理未使用的 FQDN
func fqdnInfoPostPolicyCalc(hid string) {
    for name, info := range fqdnMap {
        if !info.used {
            dp.DPCtrlDeleteFqdn(del)  // 通知 DP 删除
            delete(fqdnMap, name)
        }
    }
}
```

### 1.7 未知 IP 缓存机制

处理策略计算延迟导致的"未知 IP"问题：

```go
type unknown_ip_cache struct {
    timerTask string
    desc      unknown_ip_desc
    polver    uint16           // 策略版本
    start_hit time.Time
    last_hit  time.Time
    try_cnt   uint8            // 重试次数
}

const UNKN_IP_CACHE_TIMEOUT = time.Duration(time.Second * 600)
const UNKN_IP_TRY_COUNT uint8 = 10

func policy_chk_unknown_ip(pInfo *WorkloadIPPolicyInfo, srcip, dstip net.IP,
                           iptype string, ext bool, action *uint8, aTimerWheel *utils.TimerWheel) {
    // 如果 IP 不在策略地址映射中，临时放行
    if !exist {
        *action = C.DP_POLICY_ACTION_OPEN
        add_unkn_ip_cache(&uip_desc, pInfo.PolVer, iptype, ext, aTimerWheel)
    } else {
        // 已缓存的未知 IP，检查重试次数
        if try_cnt > 0 {
            try_cnt--
            *action = C.DP_POLICY_ACTION_OPEN
        }
    }
}
```

**设计目的**：
- 新创建的工作负载可能在策略计算完成前就有流量
- 通过临时放行避免误阻断
- 通过重试次数限制防止永久绕过

---

## 2. 规则 ID 分配 (rule_uuid.go)

### 2.1 规则 ID 范围分配

```go
const DefaultGroupRuleID uint32 = 0
const PolicyLearnedIDBase = 10000      // 学习到的规则: 10000-99999
const PolicyFedRuleIDBase = 100000     // 联邦规则: 100000-109999
const PolicyFedRuleIDMax = 110000
const PolicyGroundRuleIDBase = 110000  // Ground 规则: 110000-119999
const PolicyGroundRuleIDMax = 120000
```

**ID 范围设计**：
```
0          - 默认规则
1-9999     - 用户自定义规则
10000-99999    - 学习到的规则 (PolicyLearnedIDBase)
100000-109999  - 联邦规则 (多集群策略)
110000-119999  - Ground 规则 (基础设施策略)
```

### 2.2 UUID 管理（进程规则）

```go
type uuidPRuleCache struct {
    rwMutex sync.RWMutex

    // 待处理队列
    pendingProcProfile_u utils.Set  // 更新队列
    pendingProcProfile_d utils.Set  // 删除队列

    // 缓存
    pGrpUuidMap map[string]utils.Set  // group -> Set(uuid)
    pMap        *share.ProcRuleMap    // uuid -> rule
}

// 生成新 UUID（排除保留前缀）
func NewUuid() string {
    for cnt < 255 {
        id = uuid.New().String()
        if !strings.HasPrefix(id, share.CLUSReservedUuidPrefix) {
            return id
        }
        cnt += 1
    }
    return ""
}
```

### 2.3 系统保留 UUID

```go
const (
    CLUSReservedUuidNotAllowed     = "reserved-not-allowed"
    CLUSReservedUuidTunnelProc     = "reserved-tunnel"
    CLUSReservedUuidRootEscalation = "reserved-root-escalation"
    CLUSReservedUuidRiskyApp       = "reserved-risky-app"
    CLUSReservedUuidDockerCp       = "reserved-docker-cp"
)
```

---

## 3. 图数据库 (graph.go)

### 3.1 数据结构

```go
// 边（链接）- 指向目标节点集合
type graphLink struct {
    ends map[string]interface{}  // 目标节点名 -> 属性
}

// 节点 - 包含入边和出边
type graphNode struct {
    ins  map[string]*graphLink  // 入边: link名 -> graphLink
    outs map[string]*graphLink  // 出边: link名 -> graphLink
}

// 图
type Graph struct {
    nodes            map[string]*graphNode  // 节点名 -> graphNode
    cbNewLink        NewLinkCallback
    cbDelNode        DelNodeCallback
    cbDelLink        DelLinkCallback
    cbUpdateLinkAttr UpdateLinkAttrCallback
}
```

### 3.2 图结构示意

```
        graphLink(policy)
    ┌─────────────────────┐
    │  ends:              │
    │    "workload-B" → attr1
    │    "workload-C" → attr2
    └─────────────────────┘
           ▲
           │
    graphNode(workload-A)
    ┌─────────────────────┐
    │  ins:               │
    │    "policy" → graphLink
    │    "graph"  → graphLink
    │  outs:              │
    │    "policy" → graphLink ──→ [workload-B, workload-C]
    │    "attr"   → graphLink
    └─────────────────────┘
```

### 3.3 核心操作

```go
// 添加边（带属性）
func (g *Graph) AddLink(src, link, dst string, attr interface{}) {
    // 1. 创建或获取源节点
    if gn, ok = g.nodes[src]; !ok {
        gn = &graphNode{
            ins:  make(map[string]*graphLink),
            outs: make(map[string]*graphLink),
        }
        g.nodes[src] = gn
    }

    // 2. 创建或获取出边
    if gl, ok = gn.outs[link]; !ok {
        gl = &graphLink{ends: make(map[string]interface{})}
        gn.outs[link] = gl
    }

    // 3. 添加目标节点
    gl.ends[dst] = attr

    // 4. 同时更新目标节点的入边
    // ... (对称操作)

    // 5. 触发回调
    if newlink && g.cbNewLink != nil {
        g.cbNewLink(src, link, dst)
    }
}

// 查询连通节点（BFS）
func (g *Graph) Connected(node string, cb ConnectedNodeCallback) utils.Set {
    ret := utils.NewSet()
    ret.Add(node)
    q := []string{node}

    for len(q) > 0 {
        node, q = q[0], q[1:]  // 出队

        both := g.Both(node)   // 获取所有相邻节点
        for n := range both.Iter() {
            if cb != nil && cb(n.(string)) {
                if !ret.Contains(n) {
                    ret.Add(n)
                    q = append(q, n.(string))
                }
            }
        }
    }
    return ret
}
```

### 3.4 图查询方法

| 方法 | 说明 |
|------|------|
| `Ins(node)` | 获取所有指向该节点的节点 |
| `Outs(node)` | 获取该节点指向的所有节点 |
| `Both(node)` | 获取所有相邻节点（入+出） |
| `InsByLink(node, link)` | 按边类型过滤入边 |
| `OutsByLink(node, link)` | 按边类型过滤出边 |
| `Connected(node, cb)` | BFS 获取所有连通节点 |
| `NoIn()` | 获取没有入边的节点（源节点） |
| `NoOut()` | 获取没有出边的节点（汇节点） |

### 3.5 用途

在 NeuVector 中，图数据库用于：

1. **策略依赖关系**
   - 节点：工作负载、组、策略规则
   - 边类型：`policy`（策略关联）、`member`（组成员）

2. **网络拓扑**
   - 节点：工作负载
   - 边类型：`graph`（网络连接）
   - 属性：连接统计信息

3. **组关系**
   - 节点：组、工作负载
   - 边类型：`member`（成员关系）

---

## 4. 对你项目的启示

### 4.1 策略模式设计

你的项目可以借鉴 NeuVector 的三模式设计：

```go
// 建议的策略模式
type PolicyMode string

const (
    PolicyModeLearn    PolicyMode = "learn"    // 只记录，不阻断
    PolicyModeMonitor  PolicyMode = "monitor"  // 记录违规，不阻断
    PolicyModeEnforce  PolicyMode = "enforce"  // 强制执行
)

// 动作决策表
func (m PolicyMode) DefaultAction(rule *PolicyRule) Action {
    if m == PolicyModeLearn {
        return ActionLearn
    }
    if m == PolicyModeMonitor && rule.Action == ActionDeny {
        return ActionViolate  // 记录但不阻断
    }
    return rule.Action
}
```

### 4.2 规则 ID 范围设计

```go
const (
    UserRuleIDMin     uint32 = 1
    UserRuleIDMax     uint32 = 9999
    LearnedRuleIDMin  uint32 = 10000
    LearnedRuleIDMax  uint32 = 99999
    SystemRuleIDMin   uint32 = 100000
    SystemRuleIDMax   uint32 = 199999
)

func AllocateRuleID(ruleType RuleType) uint32 {
    switch ruleType {
    case RuleTypeUser:
        return allocateInRange(UserRuleIDMin, UserRuleIDMax)
    case RuleTypeLearned:
        return allocateInRange(LearnedRuleIDMin, LearnedRuleIDMax)
    }
}
```

### 4.3 未知 IP 处理

你的 eBPF 数据平面也需要处理策略延迟问题：

```go
type UnknownIPCache struct {
    entries  map[string]*UnknownIPEntry
    timeout  time.Duration
    maxRetry int
}

func (c *UnknownIPCache) ShouldAllow(srcIP, dstIP net.IP, policyVer uint16) bool {
    key := makeKey(srcIP, dstIP)
    if entry, ok := c.entries[key]; ok {
        if entry.retryCount > 0 && entry.policyVer == policyVer {
            entry.retryCount--
            return true  // 临时放行
        }
    }
    return false
}
```

### 4.4 图数据库简化实现

如果你只需要拓扑功能，可以简化：

```go
type SimpleGraph struct {
    nodes map[string]map[string]interface{}  // src -> dst -> attr
}

func (g *SimpleGraph) AddEdge(src, dst string, attr interface{}) {
    if g.nodes[src] == nil {
        g.nodes[src] = make(map[string]interface{})
    }
    g.nodes[src][dst] = attr
}

func (g *SimpleGraph) GetConnected(node string) []string {
    result := []string{}
    if neighbors, ok := g.nodes[node]; ok {
        for dst := range neighbors {
            result = append(result, dst)
        }
    }
    return result
}
```

---

## 5. 关键设计模式总结

| 模式 | NeuVector 实现 | 目的 |
|------|---------------|------|
| **混合模式动作调整** | `adjustAction()` | 支持渐进式部署 |
| **命名空间边界强制** | `adjustActionNs()` | 跨 NS 流量控制 |
| **未知 IP 缓存** | `unknown_ip_cache` | 处理策略延迟 |
| **FQDN 缓存和清理** | `fqdnMap` | 支持基于域名的策略 |
| **规则去重** | `RuleMap[key]` | 防止重复规则 |
| **分层 ID 分配** | ID 范围划分 | 区分规则来源 |
| **图数据库** | 多重有向图 | 存储拓扑和策略关系 |

---

## 6. 参考文件

| 文件 | 位置 | 说明 |
|------|------|------|
| network.go | agent/policy/network.go | 网络策略计算 |
| type.go | agent/policy/type.go | 策略类型定义 |
| rule_uuid.go | controller/ruleid/rule_uuid.go | 规则 UUID 管理 |
| graph.go | controller/graph/graph.go | 图数据库 |
| clus_apis.go | share/clus_apis.go | 共享常量定义 |

---

**文档整理时间**: 2025-11-22
**NeuVector 版本**: v5.x
**分析者**: eBPF 微隔离项目组
