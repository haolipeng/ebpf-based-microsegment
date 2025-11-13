# 智能策略管理

## 概述

策略管理是微隔离产品的核心功能，但传统的策略配置方式存在以下痛点：
- **配置复杂**: 需要理解 5-tuple、标签选择器、方向等专业概念
- **语法错误**: JSON/YAML 配置容易出现格式错误
- **冲突难查**: 多条策略可能产生冲突，难以人工排查
- **最佳实践**: 缺乏行业经验的用户不知道如何配置安全策略

通过集成 LLM 技术，可以将策略管理从"专家系统"变为"智能助手"，大幅降低使用门槛。

---

## 场景 1: 自然语言策略生成

### 功能描述

用户使用自然语言描述安全需求，LLM 自动生成对应的策略配置，无需手工编写 JSON/YAML。

### 用户场景

#### 场景 1.1: 简单的访问控制

**用户输入**:
```
"禁止开发环境访问生产数据库"
```

**LLM 生成**:
```json
{
  "id": "policy-auto-001",
  "name": "Block Dev to Prod DB",
  "from": {
    "selector": "env=dev"
  },
  "to": {
    "selector": "role=db,env=prod"
  },
  "action": "deny",
  "direction": "any",
  "priority": 100,
  "description": "自动生成: 禁止开发环境访问生产数据库"
}
```

#### 场景 1.2: 复杂的多端口策略

**用户输入**:
```
"允许前端服务访问 API 网关的 HTTP 和 HTTPS 端口，但是拒绝其他所有端口"
```

**LLM 生成**:
```json
[
  {
    "id": "policy-auto-002",
    "name": "Allow Frontend to API Gateway - HTTP/HTTPS",
    "from": {
      "selector": "role=frontend"
    },
    "to": {
      "selector": "role=api-gateway"
    },
    "ports": [
      {"protocol": "tcp", "port": 80},
      {"protocol": "tcp", "port": 443}
    ],
    "action": "allow",
    "priority": 100
  },
  {
    "id": "policy-auto-003",
    "name": "Block Frontend to API Gateway - Other Ports",
    "from": {
      "selector": "role=frontend"
    },
    "to": {
      "selector": "role=api-gateway"
    },
    "ports": [
      {"protocol": "tcp", "port": 0}  // 通配符
    ],
    "action": "deny",
    "priority": 50
  }
]
```

#### 场景 1.3: 基于时间的策略

**用户输入**:
```
"只允许工作时间（周一到周五 9:00-18:00）访问财务系统"
```

**LLM 响应**:
```
当前系统不支持基于时间的策略。我可以为您生成基础的访问控制策略，
并建议您通过以下方式实现时间限制：

1. 【推荐】使用外部调度系统（如 Kubernetes CronJob）定时启用/禁用策略
2. 在应用层实现时间验证
3. 等待产品后续版本支持时间条件

是否生成基础访问控制策略？
```

### 技术实现要点

#### Prompt 设计

```python
SYSTEM_PROMPT = """
你是一个微隔离策略配置专家。用户会用自然语言描述安全需求，你需要生成符合以下格式的策略配置：

策略格式：
{
  "id": "policy-xxx",
  "name": "策略名称（英文，简洁）",
  "from": {"selector": "标签选择器"},
  "to": {"selector": "标签选择器"},
  "ports": [{"protocol": "tcp/udp", "port": 端口号}],  // 可选
  "action": "allow/deny/log",
  "direction": "any/ingress/egress",
  "priority": 数字（0-1000，越大越优先）
}

标签选择器语法：
- role=web                    # 精确匹配
- env in (prod, staging)      # 集合匹配
- app!=test                   # 不等于
- location exists             # 键存在

重要规则：
1. 如果用户需求模糊，追问明确细节
2. 生成的策略必须符合 JSON Schema 验证
3. 为复杂需求生成多条策略时，注意优先级设置
4. 如果需求超出系统能力，明确告知限制
5. 总是生成 description 字段说明策略来源

当前系统能力：
- 支持: TCP/UDP/ICMP 协议, 标签选择器, 方向控制
- 不支持: 时间条件, FQDN 域名, 应用层协议识别
"""

USER_PROMPT = """
用户需求: {user_input}

当前环境信息:
- 已注册工作负载: {workload_list}
- 已有标签: {label_list}
- 已有策略数量: {policy_count}

请生成策略配置（JSON 格式）。
"""
```

#### 验证层设计

```go
// 策略验证器
type PolicyValidator struct {
    workloadMgr *workload.Manager
    policyMgr   *policy.Manager
}

func (v *PolicyValidator) ValidateAIGeneratedPolicy(policy *Policy) error {
    // 1. JSON Schema 验证
    if err := v.validateSchema(policy); err != nil {
        return fmt.Errorf("schema validation failed: %w", err)
    }

    // 2. 标签选择器验证
    if err := v.validateSelector(policy.From.Selector); err != nil {
        return fmt.Errorf("invalid from selector: %w", err)
    }
    if err := v.validateSelector(policy.To.Selector); err != nil {
        return fmt.Errorf("invalid to selector: %w", err)
    }

    // 3. 检查是否匹配到工作负载
    fromWorkloads := v.workloadMgr.FindByLabels(policy.From.Selector)
    toWorkloads := v.workloadMgr.FindByLabels(policy.To.Selector)
    if len(fromWorkloads) == 0 {
        return fmt.Errorf("warning: from selector matches 0 workloads")
    }
    if len(toWorkloads) == 0 {
        return fmt.Errorf("warning: to selector matches 0 workloads")
    }

    // 4. 冲突检测
    if conflicts := v.policyMgr.DetectConflicts(policy); len(conflicts) > 0 {
        return fmt.Errorf("conflicts with existing policies: %v", conflicts)
    }

    return nil
}
```

### API 接口设计

```go
// POST /api/v1/ai/policies/generate
type GeneratePolicyRequest struct {
    UserInput string `json:"user_input" binding:"required"`
    DryRun    bool   `json:"dry_run"`  // 是否仅验证不部署
}

type GeneratePolicyResponse struct {
    Policies    []Policy           `json:"policies"`
    Warnings    []string           `json:"warnings"`
    Suggestions []string           `json:"suggestions"`
    Explanation string             `json:"explanation"`
    DryRunResult *DryRunResult     `json:"dry_run_result,omitempty"`
}

type DryRunResult struct {
    AffectedWorkloads []string `json:"affected_workloads"`
    EstimatedImpact   string   `json:"estimated_impact"`
    FlowsToBeBlocked  int      `json:"flows_to_be_blocked"`
}
```

### 用户体验优化

#### 1. 交互式确认

```
用户: "禁止开发环境访问生产数据库"

AI: 我为您生成了以下策略：

📋 策略预览:
- 名称: Block Dev to Prod DB
- 规则: env=dev → role=db,env=prod
- 动作: DENY
- 方向: ANY

⚠️ 影响评估:
- 将影响 12 个工作负载
- 预计阻断 5 个现有 Flow
- 可能影响开发环境的数据库迁移脚本

💡 建议:
1. 考虑为迁移脚本添加例外策略
2. 建议先在测试环境验证

是否确认部署? [是/否/修改]
```

#### 2. 渐进式提问

```
用户: "配置 Web 服务的访问控制"

AI: 好的，我需要更多信息来生成准确的策略：

1. Web 服务需要访问哪些后端服务?
   □ 数据库
   □ 缓存(Redis/Memcached)
   □ API 服务
   □ 消息队列
   □ 其他: _______

2. 是否允许 Web 服务访问外部互联网?
   ○ 是，无限制
   ○ 是，但仅允许特定域名
   ○ 否，完全禁止

3. 安全策略级别:
   ○ 严格（默认拒绝，白名单）
   ○ 宽松（默认允许，黑名单）
   ○ 观察模式（仅记录，不阻断）
```

---

## 场景 2: 策略冲突检测与修复建议

### 功能描述

自动分析策略集，识别以下问题：
- **直接冲突**: 相同条件下有 ALLOW 和 DENY 规则
- **优先级混乱**: 低优先级规则被高优先级规则覆盖
- **冗余规则**: 完全重复的策略
- **无效规则**: 永远不会匹配到流量的策略

### 冲突类型分析

#### 类型 1: 直接冲突

**场景**:
```json
// 策略 A
{
  "from": {"selector": "role=web"},
  "to": {"selector": "role=db"},
  "ports": [{"protocol": "tcp", "port": 3306}],
  "action": "allow",
  "priority": 100
}

// 策略 B
{
  "from": {"selector": "role=web,env=dev"},
  "to": {"selector": "role=db,env=prod"},
  "ports": [{"protocol": "tcp", "port": 3306}],
  "action": "deny",
  "priority": 200
}
```

**LLM 分析**:
```
🔴 检测到策略冲突

冲突描述:
策略 A 允许所有 Web 服务访问数据库，但策略 B 禁止开发环境访问生产数据库。
由于策略 B 的选择器更具体且优先级更高，它将覆盖策略 A 的部分规则。

影响范围:
- 开发环境 Web 服务(role=web,env=dev) → 生产数据库(role=db,env=prod): DENY ✅
- 生产环境 Web 服务(role=web,env=prod) → 生产数据库(role=db,env=prod): ALLOW ✅

建议:
1. 【推荐】修改策略 A 的 from 选择器，明确排除开发环境:
   "from": {"selector": "role=web,env=prod"}

2. 或者添加注释说明这是有意的分层策略设计
```

#### 类型 2: 被覆盖的规则

**场景**:
```json
// 策略 A (优先级 50)
{
  "from": {"selector": "role=web"},
  "to": {"selector": "*"},
  "action": "allow"
}

// 策略 B (优先级 100)
{
  "from": {"selector": "*"},
  "to": {"selector": "*"},
  "action": "deny"
}
```

**LLM 分析**:
```
⚠️ 检测到策略被覆盖

问题:
策略 A (优先级 50) 永远不会生效，因为策略 B (优先级 100) 的通配符规则会先匹配所有流量。

建议:
1. 【推荐】调整优先级，将更具体的规则设置为更高优先级:
   - 策略 A 优先级改为 100
   - 策略 B 优先级改为 50

2. 或者修改策略 B 的选择器，避免过于宽泛
```

#### 类型 3: 完全重复

**场景**:
```json
// 策略 A 和策略 B 完全相同
```

**LLM 分析**:
```
ℹ️ 检测到冗余策略

策略 policy-001 和 policy-045 完全相同，建议删除其中一个。

如果您想保留两个策略用于不同目的（如审计），建议：
1. 在 description 字段中说明保留原因
2. 或使用标签(tags)字段标记策略用途
```

### 智能修复建议

#### 自动修复方案

```go
type ConflictResolution struct {
    ConflictType    string          `json:"conflict_type"`
    AffectedPolicies []string       `json:"affected_policies"`
    Severity        string          `json:"severity"` // critical/warning/info
    AutoFixAvailable bool           `json:"auto_fix_available"`
    FixOptions      []FixOption     `json:"fix_options"`
}

type FixOption struct {
    OptionID    string   `json:"option_id"`
    Description string   `json:"description"`
    Changes     []Change `json:"changes"`
    Pros        []string `json:"pros"`
    Cons        []string `json:"cons"`
}
```

**示例输出**:
```json
{
  "conflict_type": "priority_conflict",
  "affected_policies": ["policy-001", "policy-045"],
  "severity": "warning",
  "auto_fix_available": true,
  "fix_options": [
    {
      "option_id": "swap_priority",
      "description": "交换优先级，让更具体的规则优先",
      "changes": [
        {
          "policy_id": "policy-001",
          "field": "priority",
          "old_value": 50,
          "new_value": 100
        },
        {
          "policy_id": "policy-045",
          "field": "priority",
          "old_value": 100,
          "new_value": 50
        }
      ],
      "pros": [
        "保留所有现有策略",
        "符合最佳实践（具体规则优先）"
      ],
      "cons": [
        "可能影响其他依赖优先级的策略"
      ]
    },
    {
      "option_id": "refine_selector",
      "description": "细化选择器，明确区分规则适用范围",
      "changes": [
        {
          "policy_id": "policy-001",
          "field": "from.selector",
          "old_value": "role=web",
          "new_value": "role=web,env=prod"
        }
      ],
      "pros": [
        "语义更清晰",
        "避免未来类似冲突"
      ],
      "cons": [
        "需要确保所有 Web 工作负载都有 env 标签"
      ]
    }
  ]
}
```

### API 接口

```go
// POST /api/v1/ai/policies/analyze-conflicts
type AnalyzeConflictsRequest struct {
    PolicyIDs []string `json:"policy_ids"` // 为空则分析所有策略
}

type AnalyzeConflictsResponse struct {
    Conflicts      []ConflictResolution `json:"conflicts"`
    Summary        string               `json:"summary"`
    OverallHealth  string               `json:"overall_health"` // healthy/warning/critical
}

// POST /api/v1/ai/policies/apply-fix
type ApplyFixRequest struct {
    ConflictID string `json:"conflict_id"`
    OptionID   string `json:"option_id"`
}
```

---

## 场景 3: 策略模板推荐

### 功能描述

基于工作负载的类型、标签和行业最佳实践，自动推荐合适的策略模板。

### 推荐触发时机

1. **新工作负载注册时**: 自动推荐初始策略
2. **标签变更时**: 检查是否有新的适用模板
3. **手动触发**: 用户主动请求推荐
4. **定期审查**: 每周生成策略健康报告

### 推荐逻辑

#### 1. 基于角色的模板

**检测到**: `role=web`, `app=nginx`

**LLM 推荐**:
```yaml
推荐策略包: Web 服务标准安全策略

包含 5 条策略:

1. 允许入站 HTTP/HTTPS 流量
   from: * → to: role=web
   ports: 80, 443
   action: allow
   理由: Web 服务需要对外提供服务

2. 允许访问后端 API
   from: role=web → to: role=api
   ports: 8080
   action: allow
   理由: 典型的前后端分离架构

3. 禁止直接访问数据库
   from: role=web → to: role=db
   action: deny
   理由: 违反最佳实践，应通过 API 层访问

4. 允许访问缓存服务
   from: role=web → to: role=cache
   ports: 6379, 11211
   action: allow
   理由: 常见的性能优化手段

5. 记录所有出站互联网流量
   from: role=web → to: 0.0.0.0/0
   action: log
   理由: 审计和异常检测需求

是否应用此模板? [全部应用/选择性应用/忽略]
```

#### 2. 基于行业的模板

**检测到**: `compliance=pci-dss`, `role=db`

**LLM 推荐**:
```yaml
推荐策略包: PCI-DSS 合规 - 数据库保护

包含 8 条策略:

1. 禁止任意来源访问数据库
   from: * → to: role=db,compliance=pci-dss
   action: deny
   priority: 50
   理由: PCI-DSS 1.2.1 要求

2. 仅允许应用层访问
   from: role=api → to: role=db,compliance=pci-dss
   ports: 3306, 5432
   action: allow
   priority: 100
   理由: 最小权限原则

3. 禁止 SSH 直接访问
   from: * → to: role=db,compliance=pci-dss
   ports: 22
   action: deny
   理由: PCI-DSS 2.2.4 要求

... (省略其他策略)

合规性说明:
✅ 满足 PCI-DSS 1.2.1 (网络隔离)
✅ 满足 PCI-DSS 2.2.4 (禁用不必要服务)
✅ 满足 PCI-DSS 10.2.5 (访问日志)

是否应用此模板? [应用/查看更多/忽略]
```

#### 3. 基于流量模式的推荐

**系统检测**: 观察到工作负载 `web-pod-1` 过去 7 天的流量模式

**LLM 分析**:
```
📊 流量模式分析报告

工作负载: web-pod-1 (role=web, env=prod)
观察期: 2025-01-06 ~ 2025-01-13

检测到的通信模式:
1. api-service:8080 (TCP) - 12,450 次连接
2. redis-cluster:6379 (TCP) - 8,234 次连接
3. postgres-db:5432 (TCP) - 3,567 次连接 ⚠️
4. external-api.example.com:443 (TCP) - 234 次连接

⚠️ 发现潜在问题:
- Web 服务直接访问 PostgreSQL，违反三层架构最佳实践
- 建议通过 API 层访问数据库

🎯 推荐策略:

方案 A: 严格模式（推荐）
- 允许访问 api-service:8080
- 允许访问 redis-cluster:6379
- 禁止访问 postgres-db:5432
- 允许访问 external-api.example.com:443

方案 B: 宽松模式（保留现状）
- 允许所有观察到的流量
- 添加日志记录用于审计

方案 C: 渐进式迁移
- 第一周: 允许所有流量 + 日志
- 第二周: 警告直接数据库访问
- 第三周: 阻断直接数据库访问

选择方案: [A/B/C]
```

### 模板库结构

```yaml
# templates/web-service-standard.yaml
metadata:
  id: web-service-standard
  name: Web 服务标准安全策略
  category: role-based
  applicable_labels:
    - role=web
    - role=frontend
  difficulty: beginner
  popularity: 95%

policies:
  - name: Allow Inbound HTTP/HTTPS
    from: {selector: "*"}
    to: {selector: "role=web"}
    ports: [{protocol: tcp, port: 80}, {protocol: tcp, port: 443}]
    action: allow
    priority: 100
    rationale: Web 服务需要对外提供服务

  - name: Block Direct Database Access
    from: {selector: "role=web"}
    to: {selector: "role=db"}
    action: deny
    priority: 100
    rationale: 违反最佳实践，应通过 API 层访问

  # ... 更多策略

best_practices:
  - 前端服务不应直接访问数据库
  - 使用 API 网关统一管理后端访问
  - 记录所有出站互联网流量用于审计

references:
  - https://owasp.org/www-project-secure-coding-practices/
  - https://www.cisecurity.org/benchmark/kubernetes
```

### API 接口

```go
// GET /api/v1/ai/policies/recommend
type RecommendPoliciesRequest struct {
    WorkloadID string `json:"workload_id"` // 可选，为空则推荐全局优化
    Basis      string `json:"basis"`       // role/compliance/traffic_pattern/all
}

type RecommendPoliciesResponse struct {
    Templates     []PolicyTemplate `json:"templates"`
    Explanation   string           `json:"explanation"`
    EstimatedImpact string         `json:"estimated_impact"`
}

type PolicyTemplate struct {
    ID            string     `json:"id"`
    Name          string     `json:"name"`
    Policies      []Policy   `json:"policies"`
    Rationale     string     `json:"rationale"`
    Pros          []string   `json:"pros"`
    Cons          []string   `json:"cons"`
    ApplicableTo  []string   `json:"applicable_to"` // 工作负载 ID 列表
}
```

---

## 场景 4: 对话式策略配置

### 功能描述

通过多轮对话，引导用户完成复杂的策略配置，类似于"策略配置向导"。

### 对话流程示例

```
用户: "我想配置微服务的安全策略"

AI: 好的! 我来帮您逐步配置。首先，请描述您的应用架构:

1. 您的应用由哪些层次组成?
   □ 前端层 (Web/移动端)
   □ API 网关层
   □ 业务服务层
   □ 数据层 (数据库/缓存)
   □ 其他: _______

---

用户: "前端、API网关、业务服务、数据库四层"

AI: 很好，这是典型的四层架构 👍

第二步，请告诉我各层的具体信息:

【前端层】
- 服务名称: [例如: web-frontend]
- 监听端口: [例如: 80, 443]
- 使用的标签: [例如: role=web, tier=frontend]

【API 网关层】
- 服务名称: _______
- 监听端口: _______
- 使用的标签: _______

... (后续层次)

---

用户: "前端是 nginx，监听 80 和 443，标签 role=web"
用户: "API 网关是 Kong，监听 8080，标签 role=api-gateway"
用户: "业务服务有多个，端口 8080-8090，标签 role=service"
用户: "数据库是 PostgreSQL，端口 5432，标签 role=db"

AI: 收到! 根据您提供的信息，我建议以下安全策略:

📋 策略组 1: 前端访问控制
1. 允许互联网 → Nginx (80,443) ✅
2. 允许 Nginx → Kong (8080) ✅
3. 禁止 Nginx → Service (8080-8090) ❌
4. 禁止 Nginx → PostgreSQL (5432) ❌

📋 策略组 2: API 网关访问控制
5. 允许 Kong → Service (8080-8090) ✅
6. 禁止 Kong → PostgreSQL (5432) ❌

📋 策略组 3: 业务服务访问控制
7. 允许 Service → PostgreSQL (5432) ✅
8. 禁止 Service → 互联网 ❌ (除非必要)

📋 策略组 4: 数据库保护
9. 禁止任意来源 → PostgreSQL ❌
10. 例外: Service → PostgreSQL (已在策略 7 覆盖) ✅

⚙️ 高级选项:
- 是否允许健康检查流量? [是/否]
- 是否允许 Prometheus 监控? [是/否]
- 是否需要审计日志? [全部/仅拒绝/关闭]

请回答上述问题，或直接输入 '确认' 部署基础策略。

---

用户: "允许 Prometheus，需要审计日志，仅拒绝的"

AI: 已更新配置 ✅

添加策略:
11. 允许 Prometheus → * (端口 /metrics) ✅
12. 所有 DENY 动作添加 LOG 标记 📝

最终策略摘要:
- 总共 12 条策略
- 预计影响 23 个工作负载
- 预计阻断 5 个现有 Flow (都是不合规的)

风险评估: 🟢 低风险
- 所有正常业务流量都已放行
- 阻断的流量都是违反架构原则的异常流量

是否确认部署? [确认/修改/取消]

---

用户: "确认"

AI: 正在部署策略... ⏳

✅ 部署完成!

后续步骤:
1. 监控流量日志，确认没有误报
2. 7 天后查看 "策略有效性报告"
3. 如有问题，随时输入 '/help' 或 '回滚策略'

🎉 您的微服务现在受到微隔离保护!
```

### 会话状态管理

```go
type ConversationState struct {
    SessionID   string                 `json:"session_id"`
    UserID      string                 `json:"user_id"`
    Step        int                    `json:"step"`
    Context     map[string]interface{} `json:"context"`
    History     []Message              `json:"history"`
    DraftPolicy []Policy               `json:"draft_policy"`
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
}

type Message struct {
    Role      string    `json:"role"` // user/assistant
    Content   string    `json:"content"`
    Timestamp time.Time `json:"timestamp"`
}
```

---

## 场景 5: What-If 模拟分析

### 功能描述

在部署策略前，模拟其影响，预测可能的风险和影响范围。

### 分析维度

1. **流量影响**: 哪些现有 Flow 会被阻断
2. **工作负载影响**: 哪些服务会受影响
3. **业务风险**: 是否影响关键业务路径
4. **合规性**: 是否满足合规要求
5. **性能影响**: 策略数量对性能的影响

### 示例输出

```
用户: "What-If: 删除策略 policy-123"

AI 分析中... 🔍

📊 影响评估报告

【基本信息】
策略名称: Allow Web to Cache
策略内容: role=web → role=cache (port 6379)
创建时间: 2025-01-01
使用频率: 高 (平均每天 450 次命中)

【流量影响】
⚠️ 将影响 450+ 个每日流量
- 10.0.1.5:随机 → 10.0.2.10:6379 (约 200 次/天)
- 10.0.1.6:随机 → 10.0.2.10:6379 (约 150 次/天)
- 10.0.1.7:随机 → 10.0.2.10:6379 (约 100 次/天)

【工作负载影响】
❌ 以下服务将无法访问 Redis:
- web-frontend-1 (role=web, env=prod) - 🔴 生产环境
- web-frontend-2 (role=web, env=prod) - 🔴 生产环境
- web-frontend-3 (role=web, env=staging) - 🟡 预发布环境

【业务风险】
🔴 高风险
- Web 服务依赖 Redis 缓存，删除策略将导致:
  1. 所有缓存读写失败
  2. 数据库负载增加 300%
  3. 用户响应时间从 50ms 增加到 500ms
  4. 可能触发数据库连接池耗尽

【依赖分析】
此策略是关键路径的一部分:
Internet → Web (policy-001) → Cache (policy-123 ⚠️) → Data

删除后:
- Web 服务无法访问缓存
- 将回退到直接访问数据库 (如果允许)
- 或直接返回错误 (如果数据库也被阻断)

【替代方案】
如果您想修改此策略，建议:

方案 A: 细化而非删除
- 将策略改为仅允许生产环境访问:
  from: role=web,env=prod → to: role=cache

方案 B: 添加例外
- 保留此策略，添加新策略限制特定来源

方案 C: 分阶段迁移
- Week 1: 将 action 改为 'log' (记录但放行)
- Week 2: 观察日志，确认无业务影响
- Week 3: 安全删除

【历史数据】
过去 7 天的流量统计:
- 命中次数: 3,150 次
- 阻断次数: 67 次 (违规访问尝试)
- 平均响应时间: 0.8ms

⚠️ 强烈建议: 不要删除此策略

是否仍要继续? [取消/继续/选择方案A/B/C]
```

### API 接口

```go
// POST /api/v1/ai/policies/what-if
type WhatIfRequest struct {
    Action     string   `json:"action"` // add/delete/modify
    PolicyID   string   `json:"policy_id,omitempty"`
    NewPolicy  *Policy  `json:"new_policy,omitempty"`
    TimeRange  string   `json:"time_range"` // 分析历史数据的时间范围
}

type WhatIfResponse struct {
    Impact           Impact    `json:"impact"`
    Risk             string    `json:"risk"` // low/medium/high/critical
    Recommendations  []string  `json:"recommendations"`
    Alternatives     []Alternative `json:"alternatives"`
}

type Impact struct {
    AffectedFlows      int      `json:"affected_flows"`
    AffectedWorkloads  []string `json:"affected_workloads"`
    BusinessImpact     string   `json:"business_impact"`
    PerformanceImpact  string   `json:"performance_impact"`
}
```

---

## 实施建议

### 1. 开发优先级

**第一阶段** (MVP):
- ✅ 自然语言策略生成
- ✅ 基础验证层

**第二阶段**:
- ✅ 策略冲突检测
- ✅ 模板推荐

**第三阶段**:
- ✅ 对话式配置
- ✅ What-If 分析

### 2. Prompt 工程最佳实践

- 使用 Few-Shot Learning 提供示例
- 结构化输出（JSON Schema）
- 添加验证层防止幻觉
- 持续收集用户反馈优化 Prompt

### 3. 用户体验设计

- 提供"专家模式"和"向导模式"切换
- 所有 AI 生成内容都可手工编辑
- 保留操作历史，支持回滚
- 明确标注 AI 生成的内容

---

**下一篇**: [智能流量分析 →](02-intelligent-traffic-analysis.md)
