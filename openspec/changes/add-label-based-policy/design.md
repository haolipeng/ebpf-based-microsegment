# 设计：基于标签的策略管理系统

## 概述

本文档提供了实现基于标签的策略管理系统的详细技术设计，该系统受 Illumio 以工作负载为中心的分段模型启发。

### 架构层次

```
┌─────────────────────────────────────────────────────────────┐
│  用户 / API 层                                              │
│  - 创建工作负载，分配标签                                      │
│  - 通过选择器定义组                                           │
│  - 创建组到组的策略规则                                        │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│  控制平面（本设计）                                           │
│  ┌────────────┐  ┌────────────┐  ┌─────────────────┐       │
│  │ 工作负载   │  │ 组         │  │ 策略            │       │
│  │ 管理器     │→ │ 解析器     │→ │ 编译器          │       │
│  └────────────┘  └────────────┘  └─────────────────┘       │
│       ↓               ↓                    ↓                │
│  ┌────────────────────────────────────────────────┐         │
│  │         SQLite 存储                            │         │
│  │  workloads | groups | policy_rules |           │         │
│  │  policies  | policy_compilation                │         │
│  └────────────────────────────────────────────────┘         │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│  数据平面（现有，无变更）                                      │
│  - eBPF TC 程序                                              │
│  - 用于策略的哈希/数组映射                                     │
│  - 会话跟踪                                                  │
│  - 包处理（<10μs）                                           │
└─────────────────────────────────────────────────────────────┘
```

---

## 1. 数据模型

### 1.1 工作负载实体

表示正在运行的容器、进程或虚拟机。

```go
// Package: src/agent/pkg/workload
type Workload struct {
    // 身份
    ID           string            `json:"id" db:"id"`
    Name         string            `json:"name" db:"name"`
    HostID       string            `json:"host_id" db:"host_id"`

    // 网络（在数据库中 JSON 序列化）
    IPs          []net.IP          `json:"ips" db:"ips"`
    MACs         []string          `json:"macs" db:"macs"`
    Ports        []uint16          `json:"ports,omitempty" db:"ports"`

    // 标签（系统的核心）
    Labels       map[string]string `json:"labels" db:"labels"`

    // 用于自动标记的元数据
    Image        string            `json:"image,omitempty" db:"image"`
    Namespace    string            `json:"namespace,omitempty" db:"namespace"`
    ServiceName  string            `json:"service_name,omitempty" db:"service_name"`
    PodName      string            `json:"pod_name,omitempty" db:"pod_name"`

    // 状态
    State        WorkloadState     `json:"state" db:"state"`
    CreatedAt    time.Time         `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time         `json:"updated_at" db:"updated_at"`
}

type WorkloadState string
const (
    WorkloadRunning WorkloadState = "running"
    WorkloadStopped WorkloadState = "stopped"
    WorkloadPaused  WorkloadState = "paused"
)
```

**设计理由**：
- `Labels` 是 map[string]string 以实现最大灵活性
- 元数据字段（`Image`、`Namespace`）支持自动标记而无需外部依赖
- `IPs` 是切片以支持多宿主工作负载和 IPv6
- `State` 允许过滤（例如，仅匹配运行中的工作负载）

### 1.2 标签维度（Illumio 模型）

推荐的标签结构，系统不强制执行。

```go
// Package: src/agent/pkg/labels
type LabelDimension string

const (
    // Role: 工作负载的技术角色
    LabelRole     LabelDimension = "role"
    // 示例: "web", "api", "db", "cache", "mq", "worker", "gateway"

    // App: 业务应用
    LabelApp      LabelDimension = "app"
    // 示例: "frontend", "backend", "auth", "payment", "analytics"

    // Env: 部署环境
    LabelEnv      LabelDimension = "env"
    // 示例: "prod", "staging", "dev", "test", "qa"

    // Location: 物理或逻辑位置
    LabelLocation LabelDimension = "loc"
    // 示例: "us-west-2", "eu-central-1", "dc-1", "az-a", "edge"
)

// 帮助函数，验证标签是否为维度标签
func IsDimensionLabel(key string) bool {
    return key == string(LabelRole) ||
           key == string(LabelApp) ||
           key == string(LabelEnv) ||
           key == string(LabelLocation)
}
```

**使用示例**：
```go
workload := &Workload{
    ID:   "container-abc123",
    Name: "nginx-web-1",
    IPs:  []net.IP{net.ParseIP("10.0.1.10")},
    Labels: map[string]string{
        "role":    "web",        // 维度标签
        "app":     "frontend",   // 维度标签
        "env":     "prod",       // 维度标签
        "loc":     "us-west-2",  // 维度标签
        "version": "v2.1.0",     // 自定义标签
        "team":    "platform",   // 自定义标签
    },
}
```

### 1.3 组实体

通过标签选择器定义一组工作负载。

```go
// Package: src/agent/pkg/groups
type Group struct {
    Name        string          `json:"name" db:"name"`
    Description string          `json:"description,omitempty" db:"description"`
    Selectors   []LabelSelector `json:"selectors" db:"selectors"`
    CreatedAt   time.Time       `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

type LabelSelector struct {
    Key      string              `json:"key"`
    Operator SelectorOperator    `json:"operator"`
    Values   []string            `json:"values"`
}

type SelectorOperator string
const (
    OpEqual       SelectorOperator = "="          // key = value
    OpNotEqual    SelectorOperator = "!="         // key != value
    OpIn          SelectorOperator = "in"         // key in [v1, v2, ...]
    OpNotIn       SelectorOperator = "not-in"     // key not in [...]
    OpExists      SelectorOperator = "exists"     // key exists（任何值）
    OpNotExists   SelectorOperator = "not-exists" // key does not exist
)
```

**组示例**：

```go
// 示例 1: 生产环境 web 前端
{
    Name: "web-frontend-prod",
    Description: "生产环境 web 前端服务器",
    Selectors: []LabelSelector{
        {Key: "role", Operator: "=", Values: []string{"web"}},
        {Key: "app", Operator: "=", Values: []string{"frontend"}},
        {Key: "env", Operator: "=", Values: []string{"prod"}},
    },
}
// 匹配: role=web AND app=frontend AND env=prod

// 示例 2: 除开发环境外的所有数据库
{
    Name: "databases",
    Description: "所有数据库工作负载（非开发环境）",
    Selectors: []LabelSelector{
        {Key: "role", Operator: "in", Values: []string{"db", "cache"}},
        {Key: "env", Operator: "!=", Values: []string{"dev"}},
    },
}
// 匹配: (role=db OR role=cache) AND env!=dev

// 示例 3: 任何 nginx 容器
{
    Name: "nginx-servers",
    Description: "所有基于 nginx 的工作负载",
    Selectors: []LabelSelector{
        {Key: "image", Operator: "contains", Values: []string{"nginx"}},
    },
}
// 注意: "contains" 操作符是未来的工作
```

### 1.4 策略规则实体

组之间的高级策略规则。

```go
// Package: src/agent/pkg/policy
type PolicyRule struct {
    ID          uint32      `json:"id" db:"id"`
    Name        string      `json:"name" db:"name"`
    Description string      `json:"description,omitempty" db:"description"`

    // 源和目标（组名称）
    FromGroup   string      `json:"from_group" db:"from_group"`
    ToGroup     string      `json:"to_group" db:"to_group"`

    // 网络约束
    Ports       []PortRange `json:"ports" db:"ports"`
    Protocols   []string    `json:"protocols" db:"protocols"`

    // 策略决策
    Action      string      `json:"action" db:"action"` // "allow", "deny", "log"
    Priority    uint16      `json:"priority" db:"priority"`
    Enabled     bool        `json:"enabled" db:"enabled"`

    CreatedAt   time.Time   `json:"created_at" db:"created_at"`
    UpdatedAt   time.Time   `json:"updated_at" db:"updated_at"`
}

type PortRange struct {
    Protocol string `json:"protocol"` // "tcp", "udp", "icmp"
    Start    uint16 `json:"start"`
    End      uint16 `json:"end"`      // 如果 Start == End，单个端口
}
```

**策略规则示例**：

```go
{
    ID:   1001,
    Name: "allow-web-to-db",
    Description: "允许 web 前端访问 MySQL 数据库",
    FromGroup: "web-frontend-prod",
    ToGroup: "mysql-databases",
    Ports: []PortRange{
        {Protocol: "tcp", Start: 3306, End: 3306},
    },
    Protocols: []string{"tcp"},
    Action: "allow",
    Priority: 100,
    Enabled: true,
}
```

### 1.5 编译策略实体

低级别基于 IP 的规则（数据平面格式）。

```go
// Package: src/agent/pkg/policy
type CompiledPolicy struct {
    // 数据平面规则（5元组）
    RuleID      uint32    `json:"rule_id" db:"rule_id"`
    SrcIP       string    `json:"src_ip" db:"src_ip"`
    DstIP       string    `json:"dst_ip" db:"dst_ip"`
    SrcPort     uint16    `json:"src_port" db:"src_port"`
    DstPort     uint16    `json:"dst_port" db:"dst_port"`
    Protocol    string    `json:"protocol" db:"protocol"`
    Action      string    `json:"action" db:"action"`
    Priority    uint16    `json:"priority" db:"priority"`

    // 来源（可追溯性）
    SourcePolicyID uint32 `json:"source_policy_id" db:"source_policy_id"`
    FromGroup      string `json:"from_group" db:"from_group"`
    ToGroup        string `json:"to_group" db:"to_group"`
    FromWorkloadID string `json:"from_workload_id" db:"from_workload_id"`
    ToWorkloadID   string `json:"to_workload_id" db:"to_workload_id"`

    CompiledAt  time.Time `json:"compiled_at" db:"compiled_at"`
}
```

---

## 2. 核心算法

### 2.1 组成员解析

**目标**：确定哪些工作负载属于某个组。

```go
// Package: src/agent/pkg/groups
func (gm *GroupManager) ResolveGroupMembers(groupName string) ([]*workload.Workload, error) {
    // 1. 获取组定义
    group, err := gm.storage.GetGroup(groupName)
    if err != nil {
        return nil, err
    }

    // 2. 获取所有工作负载（可以用过滤器优化）
    allWorkloads, err := gm.workloadMgr.ListWorkloads()
    if err != nil {
        return nil, err
    }

    // 3. 过滤匹配选择器的工作负载
    var members []*workload.Workload
    for _, wl := range allWorkloads {
        if wl.State != workload.WorkloadRunning {
            continue // 仅匹配运行中的工作负载
        }

        if IsWorkloadSelected(wl, group.Selectors) {
            members = append(members, wl)
        }
    }

    // 4. 缓存成员关系以提高性能（可选）
    if err := gm.cacheMembership(groupName, members); err != nil {
        log.Warnf("缓存成员关系失败: %v", err)
    }

    return members, nil
}

func IsWorkloadSelected(wl *workload.Workload, selectors []LabelSelector) bool {
    // 所有选择器必须匹配（AND 逻辑）
    for _, sel := range selectors {
        if !EvaluateSelector(wl, sel) {
            return false
        }
    }
    return true
}

func EvaluateSelector(wl *workload.Workload, sel LabelSelector) bool {
    value, exists := wl.Labels[sel.Key]

    switch sel.Operator {
    case OpEqual:
        return exists && value == sel.Values[0]

    case OpNotEqual:
        return !exists || value != sel.Values[0]

    case OpIn:
        if !exists {
            return false
        }
        for _, v := range sel.Values {
            if value == v {
                return true
            }
        }
        return false

    case OpNotIn:
        if !exists {
            return true // 键不存在，所以不在列表中
        }
        for _, v := range sel.Values {
            if value == v {
                return false // 在列表中找到，失败
            }
        }
        return true

    case OpExists:
        return exists

    case OpNotExists:
        return !exists

    default:
        log.Errorf("未知操作符: %s", sel.Operator)
        return false
    }
}
```

**复杂度**：
- **时间**：O(W × S)，其中 W = 工作负载数，S = 每组的选择器数
- **空间**：O(M)，其中 M = 匹配的工作负载数
- **优化**：按标签键索引工作负载，首次不匹配时提前退出

### 2.2 策略编译

**目标**：将组到组的规则扩展为 IP 到 IP 的规则。

```go
// Package: src/agent/pkg/policy
func (pc *PolicyCompiler) CompilePolicyRule(ruleID uint32) ([]*CompiledPolicy, error) {
    // 1. 获取策略规则
    rule, err := pc.storage.GetPolicyRule(ruleID)
    if err != nil {
        return nil, err
    }

    if !rule.Enabled {
        return nil, nil // 跳过禁用的规则
    }

    // 2. 解析组成员
    srcWorkloads, err := pc.groupMgr.ResolveGroupMembers(rule.FromGroup)
    if err != nil {
        return nil, fmt.Errorf("解析 from_group '%s': %w", rule.FromGroup, err)
    }

    dstWorkloads, err := pc.groupMgr.ResolveGroupMembers(rule.ToGroup)
    if err != nil {
        return nil, fmt.Errorf("解析 to_group '%s': %w", rule.ToGroup, err)
    }

    // 3. 笛卡尔积扩展
    var compiledPolicies []*CompiledPolicy
    ruleIDCounter := uint32(10000 + ruleID*1000) // 避免冲突

    for _, srcWL := range srcWorkloads {
        for _, dstWL := range dstWorkloads {
            for _, portRange := range rule.Ports {
                for _, srcIP := range srcWL.IPs {
                    for _, dstIP := range dstWL.IPs {
                        cp := &CompiledPolicy{
                            RuleID:         ruleIDCounter,
                            SrcIP:          srcIP.String() + "/32",
                            DstIP:          dstIP.String() + "/32",
                            SrcPort:        0, // 源通常为"任意"
                            DstPort:        portRange.Start,
                            Protocol:       portRange.Protocol,
                            Action:         rule.Action,
                            Priority:       rule.Priority,
                            SourcePolicyID: rule.ID,
                            FromGroup:      rule.FromGroup,
                            ToGroup:        rule.ToGroup,
                            FromWorkloadID: srcWL.ID,
                            ToWorkloadID:   dstWL.ID,
                            CompiledAt:     time.Now(),
                        }
                        compiledPolicies = append(compiledPolicies, cp)
                        ruleIDCounter++
                    }
                }
            }
        }
    }

    log.Infof("编译规则 %d: %d 个源工作负载 × %d 个目标工作负载 = %d 条 IP 规则",
        ruleID, len(srcWorkloads), len(dstWorkloads), len(compiledPolicies))

    return compiledPolicies, nil
}

// 编译所有策略规则
func (pc *PolicyCompiler) CompileAllPolicies() error {
    rules, err := pc.storage.ListPolicyRules()
    if err != nil {
        return err
    }

    // 1. 清除旧的编译策略
    if err := pc.storage.ClearCompiledPolicies(); err != nil {
        return err
    }

    // 2. 编译每条规则
    for _, rule := range rules {
        compiledPolicies, err := pc.CompilePolicyRule(rule.ID)
        if err != nil {
            log.Errorf("编译规则 %d 失败: %v", rule.ID, err)
            continue
        }

        // 3. 存储编译的策略
        for _, cp := range compiledPolicies {
            if err := pc.storage.SaveCompiledPolicy(cp); err != nil {
                return err
            }

            // 4. 安装到 eBPF 映射（现有的 PolicyManager）
            ebpfPolicy := &Policy{
                RuleID:   cp.RuleID,
                SrcIP:    cp.SrcIP,
                DstIP:    cp.DstIP,
                DstPort:  cp.DstPort,
                Protocol: cp.Protocol,
                Action:   cp.Action,
                Priority: cp.Priority,
            }
            if err := pc.policyMgr.AddPolicy(ebpfPolicy); err != nil {
                return err
            }
        }
    }

    return nil
}
```

**复杂度**：
- **时间**：O(N × M × P × I)，其中 N=源工作负载，M=目标工作负载，P=端口，I=每个工作负载的 IP 数
- **空间**：O(N × M × P × I) 条编译规则
- **示例**：10 个 web × 5 个 db × 1 个端口 × 每个 1 个 IP = 50 条规则

**优化策略**：
1. **CIDR 聚合**：如果 IP 是连续的，合并为 CIDR 块
2. **端口范围**：使用端口范围而不是单个端口
3. **增量编译**：当工作负载更改时仅重新编译受影响的规则
4. **批处理**：批量将编译的规则发送到 eBPF（减少系统调用）

### 2.3 自动标记

**目标**：从工作负载元数据推断标签。

```go
// Package: src/agent/pkg/labels
type AutoTagger struct {
    rules []TaggingRule
}

type TaggingRule struct {
    Name      string
    Dimension LabelDimension
    Priority  int
    Extractor func(*workload.Workload) string
}

func NewAutoTagger() *AutoTagger {
    return &AutoTagger{
        rules: []TaggingRule{
            // 从镜像推断角色
            {
                Name:      "infer-role-from-image",
                Dimension: LabelRole,
                Priority:  100,
                Extractor: func(wl *workload.Workload) string {
                    img := strings.ToLower(wl.Image)
                    switch {
                    case strings.Contains(img, "nginx"), strings.Contains(img, "httpd"):
                        return "web"
                    case strings.Contains(img, "mysql"), strings.Contains(img, "postgres"):
                        return "db"
                    case strings.Contains(img, "redis"), strings.Contains(img, "memcached"):
                        return "cache"
                    case strings.Contains(img, "rabbitmq"), strings.Contains(img, "kafka"):
                        return "mq"
                    default:
                        return ""
                    }
                },
            },
            // 从端口推断角色
            {
                Name:      "infer-role-from-ports",
                Dimension: LabelRole,
                Priority:  50,
                Extractor: func(wl *workload.Workload) string {
                    for _, port := range wl.Ports {
                        switch port {
                        case 80, 443, 8080, 8443:
                            return "web"
                        case 3306, 5432, 27017:
                            return "db"
                        case 6379, 11211:
                            return "cache"
                        }
                    }
                    return ""
                },
            },
            // 从命名空间推断环境
            {
                Name:      "infer-env-from-namespace",
                Dimension: LabelEnv,
                Priority:  100,
                Extractor: func(wl *workload.Workload) string {
                    ns := strings.ToLower(wl.Namespace)
                    switch {
                    case strings.Contains(ns, "prod"), strings.Contains(ns, "production"):
                        return "prod"
                    case strings.Contains(ns, "stag"), strings.Contains(ns, "staging"):
                        return "staging"
                    case strings.Contains(ns, "dev"), strings.Contains(ns, "development"):
                        return "dev"
                    default:
                        return ""
                    }
                },
            },
        },
    }
}

func (at *AutoTagger) InferLabels(wl *workload.Workload) map[string]string {
    inferred := make(map[string]string)

    // 按优先级排序规则（从高到低）
    sort.Slice(at.rules, func(i, j int) bool {
        return at.rules[i].Priority > at.rules[j].Priority
    })

    for _, rule := range at.rules {
        dimKey := string(rule.Dimension)

        // 如果维度已有值则跳过（更高优先级的规则已匹配）
        if _, exists := inferred[dimKey]; exists {
            continue
        }

        // 执行提取器
        if value := rule.Extractor(wl); value != "" {
            inferred[dimKey] = value
        }
    }

    return inferred
}

// 将自动标记应用于工作负载（与现有标签合并）
func (at *AutoTagger) ApplyAutoTags(wl *workload.Workload) {
    inferred := at.InferLabels(wl)

    // 与现有标签合并（现有标签优先）
    for key, value := range inferred {
        if _, exists := wl.Labels[key]; !exists {
            wl.Labels[key] = value
        }
    }
}
```

**使用**：
```go
wl := &workload.Workload{
    ID:        "container-123",
    Name:      "nginx-web-1",
    Image:     "nginx:1.21",
    Ports:     []uint16{80, 443},
    Namespace: "production",
    Labels:    map[string]string{},
}

autoTagger := NewAutoTagger()
autoTagger.ApplyAutoTags(wl)

// 结果: wl.Labels = {"role": "web", "env": "prod"}
```

---

## 3. 数据库模式

### 3.1 新表

```sql
-- 工作负载表
CREATE TABLE workloads (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    host_id TEXT NOT NULL,
    ips TEXT NOT NULL,                      -- JSON: ["10.0.1.10", "10.0.1.11"]
    macs TEXT NOT NULL,                     -- JSON: ["00:11:22:33:44:55"]
    ports TEXT,                             -- JSON: [80, 443, 8080]
    labels TEXT NOT NULL DEFAULT '{}',      -- JSON: {"role":"web","app":"frontend"}
    image TEXT,
    namespace TEXT,
    service_name TEXT,
    pod_name TEXT,
    state TEXT NOT NULL DEFAULT 'running',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_workload_host ON workloads(host_id);
CREATE INDEX idx_workload_state ON workloads(state);
CREATE INDEX idx_workload_namespace ON workloads(namespace);

-- 组表
CREATE TABLE groups (
    name TEXT PRIMARY KEY,
    description TEXT,
    selectors TEXT NOT NULL,                -- JSON: [{"key":"role","operator":"=","values":["web"]}]
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 策略规则表（高级别，基于组）
CREATE TABLE policy_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    from_group TEXT NOT NULL,
    to_group TEXT NOT NULL,
    ports TEXT NOT NULL,                    -- JSON: [{"protocol":"tcp","start":3306,"end":3306}]
    protocols TEXT NOT NULL,                -- JSON: ["tcp", "udp"]
    action TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 100,
    enabled BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (from_group) REFERENCES groups(name) ON DELETE CASCADE,
    FOREIGN KEY (to_group) REFERENCES groups(name) ON DELETE CASCADE
);

CREATE INDEX idx_policy_from_group ON policy_rules(from_group);
CREATE INDEX idx_policy_to_group ON policy_rules(to_group);
CREATE INDEX idx_policy_enabled ON policy_rules(enabled);

-- 策略编译映射（来源追踪）
CREATE TABLE policy_compilation (
    compiled_policy_id INTEGER PRIMARY KEY,
    source_policy_id INTEGER NOT NULL,
    from_group TEXT,
    to_group TEXT,
    from_workload_id TEXT,
    to_workload_id TEXT,
    compiled_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (compiled_policy_id) REFERENCES policies(rule_id) ON DELETE CASCADE,
    FOREIGN KEY (source_policy_id) REFERENCES policy_rules(id) ON DELETE CASCADE,
    FOREIGN KEY (from_workload_id) REFERENCES workloads(id) ON DELETE SET NULL,
    FOREIGN KEY (to_workload_id) REFERENCES workloads(id) ON DELETE SET NULL
);

CREATE INDEX idx_compilation_source ON policy_compilation(source_policy_id);

-- 组成员缓存（可选，用于性能）
CREATE TABLE group_membership (
    group_name TEXT NOT NULL,
    workload_id TEXT NOT NULL,
    computed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_name, workload_id),
    FOREIGN KEY (group_name) REFERENCES groups(name) ON DELETE CASCADE,
    FOREIGN KEY (workload_id) REFERENCES workloads(id) ON DELETE CASCADE
);
```

### 3.2 现有表（不变）

```sql
-- 编译的策略（数据平面规则）- 现有，无变更
CREATE TABLE policies (
    rule_id INTEGER PRIMARY KEY,
    src_ip TEXT NOT NULL,
    dst_ip TEXT NOT NULL,
    src_port INTEGER NOT NULL,
    dst_port INTEGER NOT NULL,
    protocol TEXT NOT NULL,
    action TEXT NOT NULL,
    priority INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**注意**：现有的 `policies` 表被重用于编译规则。新的 `policy_compilation` 表将它们链接到源 PolicyRules。

---

## 4. API 设计

### 4.1 工作负载管理

```
POST   /api/v1/workloads
Body: {"id": "container-123", "name": "nginx-1", "ips": ["10.0.1.10"], "labels": {"role":"web"}}
Response: 201 Created

GET    /api/v1/workloads
Query: ?label=role:web&state=running
Response: 200 OK, [...]

GET    /api/v1/workloads/:id
Response: 200 OK, {...}

PUT    /api/v1/workloads/:id
Body: {"labels": {"role":"web","app":"frontend"}}
Response: 200 OK

DELETE /api/v1/workloads/:id
Response: 204 No Content

POST   /api/v1/workloads/:id/labels
Body: {"role": "web", "app": "frontend"}
Response: 200 OK

DELETE /api/v1/workloads/:id/labels/:key
Response: 204 No Content
```

### 4.2 组管理

```
POST   /api/v1/groups
Body: {"name": "web-tier", "selectors": [{"key":"role","operator":"=","values":["web"]}]}
Response: 201 Created

GET    /api/v1/groups
Response: 200 OK, [...]

GET    /api/v1/groups/:name
Response: 200 OK, {...}

GET    /api/v1/groups/:name/members
Response: 200 OK, [{"id":"wl1",...}, {"id":"wl2",...}]

PUT    /api/v1/groups/:name
Body: {"selectors": [...]}
Response: 200 OK

DELETE /api/v1/groups/:name
Response: 204 No Content
```

### 4.3 策略规则管理

```
POST   /api/v1/policy-rules
Body: {"name":"allow-web-to-db", "from_group":"web-tier", "to_group":"db-tier", ...}
Response: 201 Created

GET    /api/v1/policy-rules
Response: 200 OK, [...]

GET    /api/v1/policy-rules/:id
Response: 200 OK, {...}

PUT    /api/v1/policy-rules/:id
Body: {"enabled": false}
Response: 200 OK

DELETE /api/v1/policy-rules/:id
Response: 204 No Content

POST   /api/v1/policy-rules/:id/compile
Response: 200 OK, {"compiled_count": 50}

POST   /api/v1/policy-rules/compile-all
Response: 200 OK, {"total_compiled": 250}
```

### 4.4 查询 API

```
GET    /api/v1/compiled-policies
Query: ?source_policy_id=1001
Response: 200 OK, [...]

GET    /api/v1/compiled-policies/:id/source
Response: 200 OK, {"policy_rule": {...}, "from_workload": {...}, "to_workload": {...}}
```

---

## 5. 模块结构

```
src/agent/pkg/
├── workload/                   # 新增
│   ├── workload.go            # Workload 数据模型
│   ├── manager.go             # WorkloadManager 实现
│   ├── storage.go             # WorkloadStorage 接口 + SQLite 实现
│   └── manager_test.go
│
├── labels/                     # 新增
│   ├── label.go               # 标签维度
│   ├── selector.go            # LabelSelector 逻辑
│   ├── autotagger.go          # 自动标记规则
│   └── autotagger_test.go
│
├── groups/                     # 新增
│   ├── group.go               # Group 数据模型
│   ├── manager.go             # GroupManager 实现
│   ├── resolver.go            # 成员解析
│   ├── storage.go             # GroupStorage 接口 + SQLite 实现
│   └── resolver_test.go
│
├── policy/                     # 现有，扩展
│   ├── policy.go              # 现有 Policy 结构（编译规则）
│   ├── manager.go             # 现有 PolicyManager
│   ├── storage.go             # 现有 SQLiteStorage（扩展模式）
│   ├── rule.go                # 新增: PolicyRule（基于组）
│   ├── compiler.go            # 新增: PolicyCompiler
│   ├── rule_storage.go        # 新增: PolicyRuleStorage
│   └── compiler_test.go       # 新增
│
└── api/                        # 现有，扩展
    ├── handlers/
    │   ├── workload.go        # 新增: Workload CRUD 处理器
    │   ├── group.go           # 新增: Group CRUD 处理器
    │   ├── policy_rule.go     # 新增: PolicyRule CRUD 处理器
    │   └── query.go           # 新增: 查询/来源追踪处理器
    └── router.go              # 修改: 注册新路由
```

---

## 6. 权衡与决策

### 6.1 控制平面 vs 数据平面标签

**决策**：标签仅存在于控制平面，数据平面看到编译的 IP 规则。

**替代方案**：
1. 在 eBPF 映射中存储标签，在内核中匹配
2. 在 eBPF 中使用 IP 集，在用户空间解析标签

**理由**：
- **优点**：数据平面保持简单和快速（<10μs）
- **优点**：无 eBPF 映射大小限制
- **优点**：灵活的标签模型，易于扩展
- **缺点**：编译开销（控制平面可接受）

### 6.2 SQLite vs 分布式存储

**决策**：MVP 使用 SQLite，为未来的 etcd/Consul 抽象接口。

**理由**：
- **优点**：无外部依赖
- **优点**：部署简单
- **优点**：足以应对单代理部署
- **缺点**：不适合多代理集群（在第 2 阶段解决）

### 6.3 急切编译 vs 延迟编译

**决策**：急切编译（每次更改时重新编译）。

**替代方案**：
1. 延迟：数据平面查询时按需编译
2. 增量：仅重新编译受影响的规则

**理由**：
- **优点**：实现简单
- **优点**：数据平面始终拥有最新规则
- **优点**：易于调试（显式编译步骤）
- **缺点**：编译延迟（在第 2 阶段通过增量优化）

### 6.4 选择器操作符

**决策**：从 6 个操作符开始：`=`、`!=`、`in`、`not-in`、`exists`、`not-exists`。

**延迟**：`contains`、`prefix`、`regex`（当用户反馈验证需要时添加）。

**理由**：
- **优点**：涵盖 90% 的用例
- **优点**：实现和测试简单
- **缺点**：稍后可能需要添加更多操作符（可接受，接口可扩展）

### 6.5 自动标记范围

**决策**：MVP 中的基本自动标记（镜像/端口 → 角色，命名空间 → 环境）。

**延迟**：Kubernetes 标签同步，服务网格集成。

**理由**：
- **优点**：无需外部依赖即可获得即时价值
- **优点**：减少手动标记负担
- **缺点**：准确性有限（在第 2 阶段通过 K8s 集成解决）

---

## 7. 性能考虑

### 7.1 可扩展性目标

- 每个代理 1,000 个工作负载
- 100 个组
- 100 个策略规则
- 10,000 条编译的 IP 规则（典型：10 条规则 × 10 个源 × 10 个目标 = 1,000）

### 7.2 瓶颈

1. **组解析**：每组 O(W × S)
   - **缓解**：在 `group_membership` 表中缓存成员关系，在工作负载/组更改时使缓存失效

2. **策略编译**：O(N × M × P) 笛卡尔积
   - **缓解**：当组大小 >100 时发出警告，实现分页，批量更新 eBPF

3. **SQLite 写入**：顺序写入可能很慢
   - **缓解**：对批量操作使用事务，使用 WAL 模式实现并发

### 7.3 优化策略

1. **成员缓存**：存储计算的组成员关系，仅在更改时重新计算
2. **增量编译**：跟踪脏组，仅重新编译受影响的策略（第 2 阶段）
3. **CIDR 聚合**：将相邻 IP 合并到 CIDR 块（第 2 阶段）
4. **批量 eBPF 更新**：批量更新映射以减少系统调用开销

---

## 8. 迁移路径

### 8.1 向后兼容性

- 现有基于 IP 的策略继续工作
- `policies` 表不变
- 现有 PolicyManager API 不变
- 新 API 是附加的，不破坏兼容性

### 8.2 采用策略

**步骤 1**：用户继续使用基于 IP 的策略
**步骤 2**：用户注册工作负载并分配标签
**步骤 3**：用户基于标签定义组
**步骤 4**：用户创建基于组的策略规则
**步骤 5**：系统编译规则，用户验证正确性
**步骤 6**：用户禁用旧的基于 IP 的策略，依赖基于标签的策略

### 8.3 回滚

如果出现问题，用户可以：
1. 禁用基于标签的策略规则（设置 `enabled=false`）
2. 回退到现有的基于 IP 的策略
3. 无需数据平面更改（编译的规则格式相同）

---

## 9. 测试策略

### 9.1 单元测试

- `labels/selector_test.go`：测试所有选择器操作符
- `groups/resolver_test.go`：使用复杂选择器测试成员解析
- `policy/compiler_test.go`：使用各种组大小测试策略编译
- `labels/autotagger_test.go`：测试自动标记规则

### 9.2 集成测试

- 端到端流程：创建工作负载 → 组 → 策略规则 → 验证编译规则
- 测试标签更改时的组成员更新
- 测试组更改时的策略重新编译
- 测试来源追踪（将编译规则追溯到源）

### 9.3 性能测试

- 使用 1,000 个工作负载对组解析进行基准测试
- 使用 10×10 组对策略编译进行基准测试
- 测量负载下的 API 响应时间
- 验证数据平面延迟不变

---

## 10. 未来增强（第 2 阶段+）

### 10.1 容器运行时发现
- 从 Docker API / containerd API 自动发现容器
- 自动注册为工作负载
- 从容器元数据同步标签

### 10.2 Kubernetes 集成
- 监视 Kubernetes pods、服务、命名空间
- 从 K8s 元数据同步标签
- 将 K8s 命名空间映射到 `env` 维度
- 将 K8s 服务映射到 `app` 维度

### 10.3 高级功能
- 学习模式：观察流量，建议组/策略
- 策略模拟：应用前的假设分析
- 带标签的流量可视化
- 策略推荐引擎
- 用于多代理集群的分布式存储（etcd）

---

## 参考

- MVP 计划：`/docs/microsegmentation-mvp-implementation-plan.md`
- NeuVector 分析：`/docs/neuvector-analysis/neuvector-agent-dp-policy-flow.md`
- Illumio 文档：四维度标签模型
- 代理研究：全面的标签系统分析（先前的研究输出）
