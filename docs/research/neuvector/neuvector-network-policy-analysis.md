# NeuVector 网络策略配置规则深度分析

> 本文档基于 NeuVector 开源项目代码分析，深入解析其网络策略配置规则的设计与实现。

---

## 目录

1. [概述](#概述)
2. [核心数据结构](#核心数据结构)
3. [策略配置类型](#策略配置类型)
4. [策略动作与模式](#策略动作与模式)
5. [策略规则处理流程](#策略规则处理流程)
6. [高级特性](#高级特性)
7. [策略优先级与排序](#策略优先级与排序)
8. [实现要点总结](#实现要点总结)

---

## 概述

NeuVector 是一个容器安全平台，提供零信任网络安全解决方案。其网络策略系统采用分层设计，支持从学习模式到强制执行的渐进式安全策略管理。

### 架构特点

- **分层策略模型**: 高层抽象规则 → 组级IP策略 → 具体IP规则
- **多维度匹配**: 支持组、IP地址、端口、应用协议、FQDN等
- **动态行为调整**: 根据策略模式、命名空间边界等动态调整策略行为
- **自学习能力**: 从实际流量中学习并生成策略规则
- **联邦管理**: 支持跨集群的统一策略管理

### 核心文件位置

```
neuvector/
├── agent/policy/
│   ├── network.go          # 网络策略处理核心逻辑
│   ├── type.go            # 策略数据结构定义
│   └── process.go         # 进程策略处理
├── controller/rest/
│   └── policy.go          # REST API策略管理
├── controller/cache/
│   └── policy.go          # 策略缓存管理
└── share/
    └── clus_apis.go       # 集群策略数据结构
```

---

## 核心数据结构

### 1. 高层策略规则 (CLUSPolicyRule)

位置: `share/clus_apis.go:1245`

```go
type CLUSPolicyRule struct {
    ID             uint32      // Policy rule ID
    Comment        string      // Rule description or comment
    From           string      // Source group name
    To             string      // Destination group name
    FromHost       string      // Source host ID (for managed hosts)
    ToHost         string      // Destination host ID (for managed hosts)
    Ports          string      // Port list in free-style format
    Applications   []uint32    // List of application IDs
    Action         string      // Policy action: "allow" or "deny"
    Disable        bool        // Whether the rule is disabled
    CreatedAt      time.Time   // Rule creation timestamp
    LastModAt      time.Time   // Last modification timestamp
    CfgType        TCfgType    // Configuration type
    Priority       uint32      // Rule priority (0-100)
    MatchCntr      uint64      // Match counter for statistics
    LastMatchAt    time.Time   // Last match timestamp
}
```

**字段说明:**

- **ID**: 策略规则唯一标识符
  - `1 ~ 99,999`: 用户创建的规则
  - `100,000 ~ 2,147,483,647`: 学习规则
  - `2,147,483,648 ~ 2,147,583,647`: 联邦规则
  - `>= 4,000,000,000`: 基础规则(CRD)

- **From/To**: 组名，支持特殊值
  - `external`: 外部地址
  - `nodes`: 所有节点
  - `Workload`: 所有容器
  - `Host:<id/ip>`: 特定主机
  - `Workload:<ip>`: 特定容器IP
  - `nv.fqdn.<domain>`: FQDN地址组

- **Ports**: 端口规范，支持多种格式
  - 单个端口: `80`
  - 端口列表: `80, 8080, 443`
  - 端口范围: `8500-8508`
  - 指定协议: `tcp/443`, `udp/53`
  - 特殊值: `any`, `icmp`

- **Applications**: 应用协议ID列表
  - 空数组表示"任意应用"
  - 支持HTTP、MySQL、Redis等常见应用协议

- **CfgType**: 配置来源类型
  - `Learned`: 学习模式自动生成
  - `UserCreated`: 用户手动创建
  - `GroundCfg`: 基础配置(来自CRD)
  - `FederalCfg`: 联邦配置
  - `SystemDefined`: 系统定义

### 2. 运行时组级IP策略 (CLUSGroupIPPolicy)

位置: `share/clus_apis.go:1359`

```go
type CLUSGroupIPPolicy struct {
    ID     uint32              // Policy ID from CLUSPolicyRule
    From   []*CLUSWorkloadAddr // Source workload address list
    To     []*CLUSWorkloadAddr // Destination workload address list
    Action uint8               // Policy action code
}
```

**作用**: 将高层策略规则转换为包含具体工作负载地址的运行时策略。

### 3. 工作负载地址 (CLUSWorkloadAddr)

位置: `share/clus_apis.go:1345`

```go
type CLUSWorkloadAddr struct {
    WlID         string                    // Workload ID
    PolicyMode   string                    // Policy mode: Discover/Monitor/Protect
    Domain       string                    // Namespace or domain
    PlatformRole string                    // Platform role (e.g., core system container)
    LocalIP      []net.IP                  // Local IP addresses (bridge network)
    GlobalIP     []net.IP                  // Global IP addresses (overlay network)
    NatIP        []net.IP                  // NAT IP addresses or address ranges
    LocalPortApp []CLUSPortApp             // Local port and application mappings
    NatPortApp   []CLUSPortApp             // NAT port and application mappings
    Ports        map[string]CLUSMappedPort // Port mapping (container -> host)
    Apps         map[string]CLUSApp        // Detected applications on ports
}
```

**地址类型说明:**

- **LocalIP**: 容器的本地网络IP (如 Docker bridge 网络)
- **GlobalIP**: 容器的全局网络IP (如 Kubernetes overlay 网络)
- **NatIP**:
  - 主机模式容器: 主机的IP地址
  - 地址组: IP范围 (成对存储: [起始IP, 结束IP])

### 4. 端口与应用配置 (CLUSPortApp)

位置: `share/clus_apis.go:1339`

```go
type CLUSPortApp struct {
    Ports       string // Port specification: "80", "8080-8090", "tcp/443"
    Application uint32 // Application protocol ID
    CheckApp    bool   // Whether to perform application-layer inspection
}
```

### 5. 策略引擎数据结构

位置: `agent/policy/type.go:34`

```go
type Engine struct {
    NetworkPolicy     map[string]*WorkloadIPPolicyInfo  // Network policies per workload
    ProcessPolicy     map[string]*share.CLUSProcessProfile // Process policies per group
    DlpWlRulesInfo    map[string]*dp.DPWorkloadDlpRule  // DLP rules per workload
    DlpBldInfo        *DlpBuildInfo                      // DLP build information
    HostID            string                             // Enforcer host ID
    HostIPs           utils.Set                          // Host IP addresses
    TunnelIP          []net.IPNet                        // Tunnel IP networks
    PolicyAddrMap     map[string]share.CLUSSubnet        // Policy address map (containers)
    HostPolicyAddrMap map[string]share.CLUSSubnet        // Host-mode container addresses
    PolTimerWheel     *utils.TimerWheel                  // Timer for unknown IP caching
    PolDomNBEMap      map[string]bool                    // Namespace boundary enforcement per domain
    Mutex             sync.Mutex                         // Concurrency control
}
```

### 6. 工作负载IP策略信息

位置: `agent/policy/type.go:17`

```go
type WorkloadIPPolicyInfo struct {
    RuleMap    map[string]*dp.DPPolicyIPRule  // Rule deduplication map
    Policy     dp.DPWorkloadIPPolicy           // Data plane policy structure
    Configured bool                            // Whether policy is configured
    SkipPush   bool                            // Skip pushing to data plane
    HostMode   bool                            // Host-mode container flag
    CapIntcp   bool                            // Interception capability
    PolVer     uint16                          // Policy version
    Nbe        bool                            // Namespace boundary enforcement
}
```

---

## 策略配置类型

### 配置类型枚举 (TCfgType)

位置: `share/types.go`

```go
type TCfgType int

const (
    Learned       TCfgType = iota  // Auto-generated from learning mode
    UserCreated                    // User-created rules
    GroundCfg                      // Ground truth config (CRD)
    FederalCfg                     // Federal configuration
    SystemDefined                  // System-defined rules
)
```

### 配置类型特性对比

| 配置类型 | ID范围 | 可编辑 | 可删除 | 优先级 | 用途 |
|---------|--------|--------|--------|--------|------|
| Federal | 2,147,483,648 ~ 2,147,583,647 | ✓ (仅FedAdmin) | ✓ (仅FedAdmin) | 最高 | 多集群统一策略 |
| Ground | ≥ 4,000,000,000 | ✗ | ✗ | 高 | 基础配置(CRD) |
| UserCreated | 1 ~ 99,999 | ✓ | ✓ | 中 | 用户手动创建 |
| Learned | 100,000 ~ 2,147,483,647 | ✗ | ✓ | 低 | 学习模式生成 |

### 配置类型API映射

位置: `controller/rest/policy.go:30`

```go
var cfgTypeMap2Api = map[share.TCfgType]string{
    share.Learned:       api.CfgTypeLearned,      // "learned"
    share.UserCreated:   api.CfgTypeUserCreated,  // "user_created"
    share.GroundCfg:     api.CfgTypeGround,       // "ground"
    share.FederalCfg:    api.CfgTypeFederal,      // "federal"
    share.SystemDefined: api.CfgSystemDefined,    // "system_defined"
}
```

---

## 策略动作与模式

### 1. 策略动作类型

位置: `defs.h` (C宏定义)

```c
// Policy action codes
#define DP_POLICY_ACTION_OPEN       0  // Open policy - no restriction
#define DP_POLICY_ACTION_LEARN      1  // Learn mode - record but don't block
#define DP_POLICY_ACTION_ALLOW      2  // Allow traffic
#define DP_POLICY_ACTION_VIOLATE    3  // Violation - record but don't block
#define DP_POLICY_ACTION_DENY       4  // Deny traffic
#define DP_POLICY_ACTION_CHECK_APP  5  // Check application protocol
#define DP_POLICY_ACTION_CHECK_NBE  6  // Check namespace boundary enforcement
```

**动作说明:**

- **OPEN**: 完全开放，不做任何限制
- **LEARN**: 学习模式，记录流量用于策略学习
- **ALLOW**: 明确允许通信
- **VIOLATE**: 违规记录，记录但不阻止(用于监控模式)
- **DENY**: 拒绝通信，阻止流量
- **CHECK_APP**: 需要检查应用层协议才能决定动作
- **CHECK_NBE**: 需要检查命名空间边界强制规则

### 2. 策略模式 (Policy Mode)

位置: `share/types.go`

```go
const (
    PolicyModeLearn    = "Discover"  // Discovery mode
    PolicyModeEvaluate = "Monitor"   // Monitor mode
    PolicyModeEnforce  = "Protect"   // Protection mode
)
```

**模式对比:**

| 模式 | 英文名 | 行为 | 适用场景 |
|------|--------|------|----------|
| Discover | Learn | 学习所有流量，生成策略规则 | 初始部署，了解应用通信模式 |
| Monitor | Evaluate | 检测违规但不阻止，记录告警 | 策略验证，确保不会误阻止 |
| Protect | Enforce | 严格执行策略，阻止违规流量 | 生产环境，强制安全策略 |

### 3. 默认动作映射

位置: `agent/policy/network.go:1021`

```go
func policyModeToDefaultAction(mode string, capIntcp bool) uint8 {
    switch mode {
    case share.PolicyModeLearn:    // Discover mode
        return C.DP_POLICY_ACTION_LEARN
    case share.PolicyModeEvaluate: // Monitor mode
        return C.DP_POLICY_ACTION_VIOLATE
    case share.PolicyModeEnforce:  // Protect mode
        if capIntcp {
            return C.DP_POLICY_ACTION_DENY
        } else {
            return C.DP_POLICY_ACTION_VIOLATE
        }
    }
    // For workloads with unknown/empty mode, use OPEN to reduce false violations
    return C.DP_POLICY_ACTION_OPEN
}
```

**逻辑说明:**

- **Discover模式**: 默认动作为LEARN，记录所有流量
- **Monitor模式**: 默认动作为VIOLATE，记录违规但不阻止
- **Protect模式**:
  - 可拦截容器: 默认DENY
  - 不可拦截容器: 降级为VIOLATE
- **未知模式**: 使用OPEN，避免误报

---

## 策略规则处理流程

### 1. 端口规范化处理

位置: `controller/rest/policy.go:209`

#### 支持的端口格式

```
示例输入                      标准化输出
--------------------------------------------------
"80"                      →  "tcp/80"
"80, 443, 8080"           →  "tcp/80,tcp/443,tcp/8080"
"8500-8508"               →  "tcp/8500-8508"
"tcp/443"                 →  "tcp/443"
"udp/53"                  →  "udp/53"
"tcp/any"                 →  "tcp/any"
"any"                     →  "any"
"icmp"                    →  "icmp"
"80, tcp/443, udp/53"     →  "icmp,any,tcp/80,tcp/443,udp/53"
```

#### 端口排序规则

```go
// Port sorting order in portRangeSorter.Less()
// 1. By protocol (TCP < UDP)
// 2. By lower bound
// 3. By upper bound
```

输出格式: `icmp` → `any` → `tcp/any` → `udp/any` → 按协议和端口排序的具体端口

### 2. 应用协议处理

位置: `controller/rest/policy.go:315`

```go
func appNames2IDs(apps []string) []uint32 {
    if len(apps) == 0 {
        return []uint32{}  // Empty array means "any application"
    }

    var ids []uint32 = make([]uint32, 0)
    for _, app := range apps {
        if strings.EqualFold(app, api.PolicyAppAny) {
            return []uint32{}  // "any" means no restriction
        }
        if id := common.GetAppIDByName(app); id != 0 {
            ids = append(ids, id)
        }
    }

    return ids
}
```

**应用协议示例:**

- HTTP, HTTPS
- MySQL, PostgreSQL, MongoDB
- Redis, Memcached
- DNS, DHCP
- Kafka, RabbitMQ
- 等

### 3. 策略规则创建流程

位置: `agent/policy/network.go:426`

```go
func (e *Engine) createWorkloadRule(from, to *share.CLUSWorkloadAddr,
    policy *share.CLUSGroupIPPolicy, pInfo *WorkloadIPPolicyInfo,
    ingress, sameHost bool)
```

#### 处理步骤

```
1. 动作调整 (adjustAction)
   ├─ 根据源/目标策略模式调整
   ├─ 考虑容器拦截能力
   └─ 应用命名空间边界强制

2. 确定规则方向和FQDN
   ├─ Ingress: 入向规则
   ├─ Egress: 出向规则
   └─ 提取FQDN域名(如果有)

3. 创建IP规则
   ├─ 处理LocalIP (本地网络)
   ├─ 处理GlobalIP (全局网络)
   ├─ 处理NatIP (NAT/地址组)
   ├─ 同主机特殊处理
   └─ 主机模式容器特殊处理

4. 生成具体IP规则
   └─ 调用 createIPRule() 生成最终规则
```

### 4. 策略动作动态调整

位置: `agent/policy/network.go:366`

#### 4.1 标准模式动作调整

```go
func adjustAction(action uint8, from, to *share.CLUSWorkloadAddr, id uint32) uint8
```

**调整矩阵:**

| 源模式 | 目标模式 | 规则动作 | 调整后动作 | 说明 |
|--------|----------|----------|------------|------|
| Discover | * | DENY | VIOLATE | 学习模式不阻止 |
| Discover | * | ALLOW (学习规则) | LEARN | 记录流量 |
| Monitor | * | DENY | VIOLATE | 监控模式不阻止 |
| Monitor | Discover | ALLOW (学习规则) | LEARN | 帮助目标学习 |
| Protect | Discover | ALLOW (学习规则) | LEARN | 帮助目标学习 |
| Protect | Discover/Monitor | DENY | VIOLATE | 避免阻止非保护模式 |

#### 4.2 严格模式动作调整

位置: `agent/policy/network.go:309`

当 `StrictGroupMode = true` 时启用严格模式:

```go
func adjustActionStrict(action uint8, from, to *share.CLUSWorkloadAddr, id uint32) uint8
```

**严格模式规则:**

- **Protect → Protect**: 严格执行DENY
- **Protect → Discover/Monitor**: DENY降级为VIOLATE
- **Discover/Monitor → Protect**: DENY按原样执行
- **其他组合**: 与标准模式相同

#### 4.3 命名空间边界强制 (NBE)

位置: `agent/policy/network.go:284`

```go
func adjustActionNs(action uint8, from, to *share.CLUSWorkloadAddr, id uint32) uint8
```

**跨命名空间判定:**

标记为跨命名空间(`CHECK_NBE`)的情况:
- 源或目标为`external`
- 源或目标为地址组
- 源或目标为FQDN地址
- 源或目标为`Host:<ip>`格式
- 源或目标为`Workload:<ip>`格式
- 源和目标在不同的命名空间(domain)

**同命名空间判定:**

保持原动作的情况:
- 源或目标为`nodes`
- 源或目标为`Host:<hostid>`格式
- 源或目标为`Workload:ingress`
- 源或目标为系统核心容器
- 源和目标在同一命名空间

### 5. IP规则生成

位置: `agent/policy/network.go:178`

```go
func createIPRule(from, to, fromR, toR net.IP, portApps []share.CLUSPortApp,
    action uint8, pInfo *WorkloadIPPolicyInfo, ctx *ruleContext)
```

#### 规则去重键格式

```
Key = <FromIP><ToIP><Port><FQDN><Direction><FromRange><ToRange>

示例:
- Egress: 10.0.1.5 → 10.0.2.3:80
  Key: "10.0.1.510.0.2.3tcp/800"

- Ingress with range: 192.168.0.1-192.168.0.255 → 10.0.1.5:443
  Key: "192.168.0.110.0.1.5tcp/4431192.168.0.255"
```

#### 应用层检查规则

```go
if !pInfo.HostMode && portApp.CheckApp {
    appRule := &dp.DPPolicyApp{
        App:    portApp.Application,
        Action: action,
        RuleID: id,
    }
    rule.Apps = append(rule.Apps, appRule)
    rule.Action = C.DP_POLICY_ACTION_CHECK_APP
}
```

**逻辑:**

- 主机模式容器: 仅检查端口，不检查应用
- 普通容器: 支持应用层深度检查
- 多个应用: 合并到同一IP规则的Apps列表

---

## 高级特性

### 1. FQDN (域名) 策略支持

位置: `agent/policy/network.go:30`

#### FQDN解析流程

```go
func getFqdnIP(name string) []net.IP
```

```
1. 检查FQDN缓存
   └─ 命中: 返回缓存的IP列表

2. 处理通配符域名
   ├─ "*.example.com" → 返回 [0.0.0.0]
   └─ 在数据平面进行通配符匹配

3. DNS解析
   ├─ 成功: 缓存IPv4地址列表
   ├─ 失败: 缓存 [0.0.0.0] 作为占位符
   └─ 标记为已使用

4. 定期清理
   └─ 删除未使用的FQDN条目
```

#### FQDN生命周期管理

```go
// Before policy calculation
fqdnInfoPrePolicyCalc()  // Mark all FQDN entries as unused

// During policy calculation
getFqdnIP(name)          // Mark accessed FQDN as used

// After policy calculation
fqdnInfoPostPolicyCalc() // Delete unused FQDN entries
```

### 2. 未知IP处理机制

位置: `agent/policy/network.go:1182`

#### 缓存策略

```go
type unknown_ip_cache struct {
    timerTask string            // Timer task ID
    desc      unknown_ip_desc   // Source and destination IPs
    polver    uint16            // Policy version
    start_hit time.Time         // First hit timestamp
    last_hit  time.Time         // Last hit timestamp
    try_cnt   uint8             // Remaining retry count
}
```

#### 处理逻辑

```
1. 检测未知IP流量
   ├─ IP不在PolicyAddrMap中
   └─ 策略动作为VIOLATE或DENY

2. 查找缓存
   ├─ 缓存存在
   │   ├─ 策略版本匹配 && 时间 < 60s: 临时放行 (OPEN)
   │   └─ 否则: 递减重试次数
   └─ 缓存不存在: 创建缓存，临时放行

3. 缓存超时 (10分钟)
   └─ 自动删除缓存条目

4. 重试次数上限
   ├─ 内部主机IP: 3次
   ├─ 外部IP: 2次
   └─ 其他未知IP: 10次
```

**目的**: 避免因策略更新延迟导致的误报，给予新容器/IP一定的宽限期。

### 3. 命名空间边界强制 (NBE)

位置: `agent/policy/network.go:1223`

#### NBE检查流程

```go
func policy_chk_nbe(pInfo *WorkloadIPPolicyInfo, conn *dp.Connection,
    policyId uint32, taction uint8) (uint32, uint8, bool)
```

```
1. 判断是否为NBE规则
   └─ action == CHECK_NBE

2. 应用NBE策略
   ├─ NBE启用
   │   ├─ 设置跨命名空间标志
   │   ├─ 策略ID清零 (ID=0表示NBE违规)
   │   └─ 根据默认动作调整
   │       ├─ Discover → VIOLATE
   │       ├─ Monitor → VIOLATE
   │       └─ Protect → DENY
   └─ NBE未启用
       └─ 允许通信 (ALLOW)

3. 标记同域内的NBE流量
   ├─ Ingress且非外部流量且无Ingress策略
   └─ Egress且非外部流量且无Egress策略
```

**应用场景**: 实现Kubernetes命名空间隔离，防止跨命名空间的未授权访问。

### 4. 策略地址映射

位置: `agent/policy/network.go:882`

#### 地址映射构建

```go
// Add local IPs of containers in Monitor/Protect mode
func addWlLocalAddrToPolicyAddrMap(from *share.CLUSWorkloadAddr,
    newPolicyAddrMap map[string]share.CLUSSubnet)

// Add global IPs
func addWlGlobalAddrToPolicyAddrMap(from *share.CLUSWorkloadAddr,
    newPolicyAddrMap map[string]share.CLUSSubnet)

// Add host-mode container IPs
func addWlHostModeAddrToPolicyAddrMap(from *share.CLUSWorkloadAddr,
    newHostPolicyAddrMap map[string]share.CLUSSubnet)
```

#### 地址映射用途

```
1. 未知IP判断
   └─ 判断IP是否属于已管理的容器/主机

2. 策略计算优化
   └─ 快速判断是否需要生成策略规则

3. 违规检测
   └─ 区分真正的违规和策略更新延迟
```

### 5. 主机模式容器处理

位置: `agent/policy/network.go:450`

#### 特殊处理逻辑

```go
if pInfo.HostMode {
    // Source IP: Use host IPs, not container IPs
    if len(from.NatIP) == 0 {
        return  // Skip if no host IP
    }
    fromIPList = from.NatIP  // Multiple host interfaces possible

    // Destination IP: Include both local and global IPs
    for _, ipTo := range to.GlobalIP {
        createIPRule(ipFrom, ipTo, nil, nil, to.LocalPortApp, action, pInfo, ctx)
    }
}
```

**主机模式特点:**

- 共享主机网络栈
- 不检查源IP (出向)
- 不检查目标IP (入向)
- 仅检查端口，不检查应用层
- 需要处理主机的多个网络接口

### 6. 同主机通信优化

位置: `agent/policy/network.go:463`

```go
if sameHost {
    // Add rules for local IP to local IP communication
    for _, ipFrom := range fromIPList {
        for _, ipTo := range to.LocalIP {
            createIPRule(ipFrom, ipTo, nil, nil, to.LocalPortApp, action, pInfo, ctx)
        }
    }

    // Add rules for all host IPs
    for _, addr := range e.HostIPs.ToSlice() {
        createIPRule(ipFrom, net.ParseIP(addr.(string)), nil, nil,
            to.NatPortApp, action, pInfo, ctx)
    }

    // Host-mode: add loopback rule
    if pInfo.HostMode {
        createIPRule(ipFrom, utils.IPv4Loopback, nil, nil,
            to.NatPortApp, action, pInfo, ctx)
    }
}
```

**优化目标**: 确保同主机上的容器间通信覆盖所有可能的网络路径。

---

## 策略优先级与排序

### 1. 规则优先级顺序

位置: `controller/rest/policy.go:579`

策略规则按以下顺序排列(优先级从高到低):

```
1. Federal Rules (联邦规则)
   ├─ ID范围: 2,147,483,648 ~ 2,147,583,647
   ├─ 仅FedAdmin可编辑
   └─ 用于多集群统一策略

2. Ground Rules (基础规则)
   ├─ ID范围: >= 4,000,000,000
   ├─ 来自CRD配置
   └─ 不可编辑和删除

3. User-Created Rules (用户规则)
   ├─ ID范围: 1 ~ 99,999
   ├─ 用户手动创建
   └─ 可编辑和删除

4. Learned Rules (学习规则)
   ├─ ID范围: 100,000 ~ 2,147,483,647
   ├─ 学习模式自动生成
   └─ 仅可删除，不可编辑
```

### 2. 规则移动限制

位置: `controller/rest/policy.go:583`

```go
func moveRuleID(crhs []*share.CLUSRuleHead, id uint32,
    ruleCfgType share.TCfgType, after *int) error
```

**移动规则:**

- **Ground规则**: 不允许移动
- **Federal规则**: 只能在Federal规则区域内移动
- **其他规则**: 只能在Federal和Ground规则之后移动

**移动参数:**

- `after = nil`: 移动到末尾
- `after = 0`: 移动到开头(受规则类型限制)
- `after = +id`: 移动到规则id之后
- `after = -id`: 移动到规则id之前

### 3. 规则插入位置

位置: `controller/rest/policy.go:712`

```go
func insertPolicyRule(scope string, w http.ResponseWriter,
    r *http.Request, insert *api.RESTPolicyRuleInsert,
    acc *access.AccessControl, login *loginSession) error
```

**插入约束:**

```
Scope=fed:
  ├─ 只能插入Federal规则
  └─ 可插入位置: Federal规则区域内任意位置

Scope=local:
  ├─ 只能插入User-Created规则
  └─ 可插入位置: Ground规则之后，任意位置
```

### 4. 规则替换逻辑

位置: `controller/rest/policy.go:971`

```go
func replacePolicyRule(scope string, w http.ResponseWriter,
    r *http.Request, rules []*api.RESTPolicyRule,
    delRuleIDs utils.Set, acc *access.AccessControl) error
```

**替换行为:**

| 调用者 | Scope | 保留规则 | 替换规则 | 删除规则 |
|--------|-------|----------|----------|----------|
| FedAdmin | fed | Local规则 | Federal规则 | 未提交的Federal规则 |
| Admin | local | Federal + Ground | User + Learned | 未提交的User + 可删除的Learned |
| NamespaceUser | local | Federal + Ground + 不可访问的规则 | 可访问的User + Learned | 未提交的可访问规则 |

**重要特性:**

- 新学习规则保护: 在GET和PATCH之间新生成的学习规则会被追加到末尾
- 只读规则保护: 命名空间用户无法修改只读规则，尝试修改会报错
- 原子操作: 整个替换操作使用事务，要么全部成功，要么全部失败

---

## 实现要点总结

### 1. 分层设计

```
REST API层 (policy.go)
    ↓ 规则验证、标准化
Controller层 (cache/policy.go)
    ↓ 规则缓存、转换
Agent策略引擎层 (agent/policy/network.go)
    ↓ 策略计算、规则生成
Data Plane层 (dp)
    ↓ 策略执行、流量过滤
```

### 2. 策略计算关键步骤

```go
// Main policy calculation entry point
func (e *Engine) UpdateNetworkPolicy(ps []share.CLUSGroupIPPolicy,
    newPolicy map[string]*WorkloadIPPolicyInfo) utils.Set
```

```
1. FQDN预处理
   └─ fqdnInfoPrePolicyCalc()

2. 解析组级IP策略
   └─ e.parseGroupIPPolicy(ps, newPolicy, ...)
       ├─ 构建地址映射表 (addrMap)
       ├─ 填充容器地址信息
       ├─ 生成Egress规则
       └─ 生成Ingress规则

3. FQDN后处理
   └─ fqdnInfoPostPolicyCalc(e.HostID)

4. 推送到数据平面
   ├─ 更新容器策略
   ├─ 更新主机模式容器策略
   └─ 更新策略地址映射

5. 更新引擎状态
   └─ e.NetworkPolicy = newPolicy
```

### 3. 性能优化技巧

#### 规则去重

```go
// Use rule key for deduplication
key := fmt.Sprintf("%v%v%s%s%d", from, to, port, fqdn, direction)
if existRule, ok := pInfo.RuleMap[key]; ok {
    // Merge applications instead of creating duplicate rule
    existRule.Apps = append(existRule.Apps, newApp)
    return
}
```

#### 同主机检测

```go
// Avoid cross-host communication checks for same-host workloads
var sameHost bool = false
if isSameHostEP(to.WlID, e.HostID) {
    sameHost = true
} else if _, ok := workloadPolicyMap[to.WlID]; ok {
    sameHost = true
}
```

#### 策略方向优化

```go
// Skip unnecessary rules based on ApplyDir flag
if pInfo.Policy.ApplyDir & C.DP_POLICY_APPLY_EGRESS > 0 {
    // Generate egress rules
} else {
    // Only generate rules to external/address groups
    if to.WlID == share.CLUSWLExternal ||
       to.WlID == share.CLUSWLAddressGroup {
        // Generate limited egress rules
    }
}
```

### 4. 并发安全

```go
// Engine uses mutex for concurrent access
type Engine struct {
    NetworkPolicy map[string]*WorkloadIPPolicyInfo
    Mutex         sync.Mutex
}

// All public methods acquire lock
func (e *Engine) GetNetworkPolicy() map[string]*WorkloadIPPolicyInfo {
    e.Mutex.Lock()
    defer e.Mutex.Unlock()
    return e.NetworkPolicy
}
```

### 5. 错误处理

```go
// Graceful degradation for non-interceptable containers
if !pInfo.CapIntcp && action == C.DP_POLICY_ACTION_DENY {
    action = C.DP_POLICY_ACTION_VIOLATE
}

// Handle missing IPs for host-mode containers
if pInfo.HostMode {
    if len(from.NatIP) == 0 {
        return  // Silently skip instead of logging errors
    }
}
```

### 6. 可观测性

```go
// Policy rule statistics
type CLUSPolicyRule struct {
    MatchCntr   uint64    // Match counter
    LastMatchAt time.Time // Last match timestamp
}

// Unknown IP caching for debugging
type unknown_ip_cache struct {
    start_hit time.Time  // First hit time
    last_hit  time.Time  // Last hit time
    try_cnt   uint8      // Retry count
}
```

---

## 总结

NeuVector 的网络策略系统是一个设计精良、功能强大的零信任网络安全解决方案:

### 核心优势

1. **灵活的策略模型**
   - 支持组、IP、端口、应用、FQDN等多维度匹配
   - 分层设计便于管理和扩展

2. **渐进式安全策略**
   - Discover → Monitor → Protect 渐进模式
   - 降低误报，平滑过渡到强制执行

3. **智能策略调整**
   - 根据策略模式动态调整行为
   - 命名空间边界强制
   - 未知IP临时放行机制

4. **企业级特性**
   - 多集群联邦管理
   - 基于CRD的GitOps支持
   - 精细的权限控制(RBAC)

5. **高性能实现**
   - 规则去重
   - 同主机优化
   - 高效的数据平面集成

### 适用场景

- **Kubernetes/容器环境**: 零信任网络安全
- **微服务架构**: 服务间通信控制
- **多租户平台**: 命名空间隔离
- **合规性要求**: 网络流量审计和控制

### 参考实现价值

对于 eBPF-based-microsegment 项目:

1. **策略模型设计**: 可借鉴分层策略设计和多维度匹配
2. **动作调整机制**: 参考模式组合的动作调整逻辑
3. **FQDN支持**: 学习域名解析和缓存机制
4. **优化技巧**: 规则去重、同主机检测等性能优化方法
5. **错误处理**: 优雅降级和未知IP处理策略

---

**文档版本**: 1.0
**基于代码**: NeuVector (开源版本)
**分析日期**: 2025-11
**分析文件**:
- `agent/policy/network.go` (1562 lines)
- `agent/policy/type.go` (70 lines)
- `controller/rest/policy.go` (2380 lines)
- `share/clus_apis.go` (相关部分)
