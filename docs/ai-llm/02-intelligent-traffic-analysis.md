# 智能流量分析

## 概述

流量分析是微隔离产品的核心价值之一,但传统的流量分析工具面临以下挑战:
- **数据量大**: 每天产生数万甚至数百万条 Flow 记录
- **模式复杂**: 需要专业知识才能识别异常流量
- **上下文缺失**: 原始数据难以直接关联到业务含义
- **响应滞后**: 人工分析速度慢,难以实时响应

通过集成 LLM 技术,可以将海量流量数据转化为可理解的安全洞察,实现从"数据展示"到"智能分析"的跨越。

---

## 场景 1: 异常流量检测与解释

### 功能描述

结合历史基线、策略规则和业务上下文,自动识别异常流量模式,并提供人类可读的安全解释和应对建议。

### 异常类型分类

#### 类型 1: 时间异常

**检测条件**: 非工作时间的异常流量

**示例**:
```
📊 异常流量告警

时间: 2025-01-13 03:15:22
严重级别: ⚠️ 中等

异常描述:
检测到开发环境工作负载在凌晨 3 点访问生产数据库

详细信息:
- 源: dev-pod-5 (10.0.1.15) - role=api, env=dev
- 目标: postgres-prod (10.0.2.10:5432) - role=db, env=prod
- 协议: TCP
- 数据量: 2.3 GB (上传 2.1 GB, 下载 200 MB)
- 持续时间: 45 分钟

🔍 LLM 分析:

【异常维度】
1. ⏰ 时间异常 (高风险)
   - 正常工作时间: 9:00-18:00
   - 当前时间: 凌晨 3:15
   - 该工作负载过去 30 天在凌晨无任何访问记录

2. 🌍 环境隔离违规 (严重)
   - 开发环境不应访问生产数据库
   - 违反策略: isolation-policy-001
   - 但流量被放行,说明存在策略漏洞

3. 📈 流量异常 (高风险)
   - 平均每次访问数据量: 10 MB
   - 本次数据量: 2.3 GB (230 倍异常)
   - 上传数据远大于下载,疑似数据导出

4. 🔄 行为异常 (可疑)
   - 该 Pod 通常每天访问数据库 5-10 次
   - 本次是单次长连接,持续 45 分钟
   - 类似于数据库备份或批量导出操作

【威胁评估】
🔴 高风险场景:
- 潜在的数据泄露或未授权导出
- 可能是被入侵账户的恶意行为
- 违反数据合规要求 (生产数据不得进入开发环境)

🟡 合理场景:
- 开发人员执行紧急数据迁移任务
- 自动化备份脚本配置错误
- 数据库同步任务时间配置错误

【关联事件】
在同一时间段检测到:
- ❌ 该 Pod 的 SSH 登录失败尝试 (15 次)
- ⚠️ 该 Pod 尝试访问外部 IP 45.33.x.x (已被阻断)
- ℹ️  该 Pod 所在节点 CPU 使用率异常升高

综合判断: 🔴 疑似安全事件,建议立即调查

【推荐行动】
紧急措施 (立即执行):
1. 隔离工作负载 dev-pod-5
2. 暂停该 Pod 的所有出站流量
3. 保留当前容器镜像用于取证分析
4. 检查近期部署和代码变更记录

调查步骤:
1. 查看该 Pod 的进程列表和网络连接
2. 审计数据库访问日志,确认导出了哪些数据
3. 检查该 Pod 的启动命令和环境变量
4. 询问开发团队是否有授权的维护任务

长期改进:
1. 添加策略: 完全禁止开发环境访问生产数据库
2. 实施数据脱敏: 开发环境使用脱敏后的数据
3. 启用数据库审计日志
4. 部署 DLP (数据防泄漏) 系统

是否执行推荐的紧急措施? [执行/仅告警/忽略]
```

#### 类型 2: 流量模式异常

**检测条件**: 与历史基线偏差显著

**示例**:
```
📊 流量模式异常

工作负载: web-frontend-1
时间: 2025-01-13 14:30:00
严重级别: 🟡 警告

🔍 LLM 分析:

【流量统计对比】
指标                | 过去 7 天平均 | 当前值    | 偏差
--------------------|--------------|----------|-------
出站连接数           | 150 /小时    | 850 /小时 | +467%
数据传输量           | 500 MB/小时  | 3.2 GB/小时| +540%
目标 IP 唯一数       | 5            | 45        | +800%
平均连接时长         | 2.5 秒       | 0.3 秒    | -88%

【异常特征】
1. 扫描行为特征 ⚠️
   - 短时间内连接大量不同 IP
   - 连接时长极短 (< 0.5 秒)
   - 大部分连接被拒绝 (90% 失败率)
   - 目标端口集中在 80, 443, 8080, 3306, 6379

2. 目标分析 🎯
   - 40% 目标 IP 位于内网其他网段
   - 30% 目标 IP 是外部公网地址
   - 20% 目标 IP 是已知的恶意 IP (参考威胁情报)
   - 10% 目标 IP 无法解析

3. 时间线分析 📈
   - 异常开始时间: 14:15:23
   - 触发事件: 该 Pod 重启
   - 持续时间: 15 分钟至今
   - 趋势: 连接频率逐渐增加

【威胁评估】
🔴 疑似场景:
- 容器被入侵,正在执行网络扫描
- 加密货币挖矿程序的 C&C 通信尝试
- 蠕虫病毒横向传播

🟡 可能场景:
- 应用程序配置错误,重试逻辑失控
- 服务发现机制故障
- 负载均衡健康检查异常

【根因分析】
检查该 Pod 的最近变更:
- 镜像版本: myapp:v2.3.5 (1 小时前部署) ⚠️
- 配置变更: 环境变量 API_ENDPOINTS 被修改 ⚠️
- 代码变更: 提交 abc123f "优化服务发现逻辑"

🔍 发现问题:
查看代码变更 abc123f:
- 新增服务发现功能,尝试自动发现后端服务
- 实现有缺陷: 扫描整个 /16 网段寻找服务
- 缺少速率限制和错误处理

判断: 🟢 配置错误,非恶意行为

【推荐行动】
立即措施:
1. 回滚到上一版本 myapp:v2.3.4
2. 或修复配置: 指定明确的服务端点列表

长期改进:
1. 代码审查: 禁止在生产环境执行网络扫描
2. 策略限制: 限制 Pod 的出站连接速率
3. 监控告警: 设置出站连接数阈值告警

已自动创建工单: #TICKET-2025-0113-001
分配给: DevOps Team

是否立即回滚? [回滚/修复配置/仅监控]
```

#### 类型 3: 协议异常

**检测条件**: 协议使用模式不符合预期

**示例**:
```
📊 协议使用异常

工作负载: api-service-2
时间: 2025-01-13 16:45:00
严重级别: 🟡 警告

🔍 LLM 分析:

【协议分布变化】
协议    | 过去 7 天 | 当前     | 变化
--------|----------|----------|--------
HTTP    | 95%      | 60%      | -35%
HTTPS   | 5%       | 10%      | +5%
DNS     | 0.1%     | 0.1%     | 0%
ICMP    | 0%       | 30%      | +30% ⚠️

【异常分析】
1. ICMP 流量激增 ⚠️
   - 该服务历史上从不发送 ICMP 包
   - 当前每秒发送 500+ ICMP Echo Request
   - 目标: 内网多个 IP 段

2. HTTP 流量占比下降
   - 正常业务流量减少 35%
   - 可能被 ICMP 流量"挤占"带宽

3. 目标分析
   - ICMP 目标是随机 IP
   - 未遵循服务依赖关系
   - 疑似网络可达性探测或 DDoS 准备

【威胁评估】
🔴 疑似攻击准备:
- ICMP 扫描常用于网络侦察
- 为后续攻击收集存活主机信息
- 可能是 APT 攻击的早期阶段

🟡 运维排查:
- 运维人员安装了网络诊断工具
- 但忘记卸载或停止

【容器检查】
执行: kubectl exec api-service-2 -- ps aux

发现异常进程:
- /tmp/.hidden/nmap -sn 10.0.0.0/16  ⚠️
- /tmp/.hidden/masscan --rate 10000  🔴

判断: 🔴 容器已被入侵

【推荐行动】
紧急隔离 (自动执行):
1. ✅ 已自动阻断该 Pod 所有出站流量
2. ✅ 已通知安全团队
3. ⏳ 等待人工介入...

下一步:
1. 不要删除 Pod,保留现场用于取证
2. 导出容器文件系统: docker export > forensics.tar
3. 分析入侵路径:
   - 检查应用程序漏洞
   - 审计最近的部署记录
   - 扫描镜像仓库是否被投毒

安全加固:
1. 启用 Pod Security Policy,禁止运行未签名的二进制文件
2. 实施运行时安全策略 (Falco/Tracee)
3. 网络策略: 默认拒绝所有 ICMP 流量

事件已上报: SOC-INCIDENT-2025-0113-CRITICAL

安全团队将在 15 分钟内介入。
```

### 数据来源

流量分析依赖以下数据源:

```go
type TrafficAnalysisContext struct {
    // 当前流量数据
    CurrentFlow     Flow

    // 历史基线 (从 SQLite 查询)
    HistoricalBaseline Baseline

    // 工作负载上下文
    SourceWorkload  Workload
    DestWorkload    Workload

    // 策略上下文
    ApplicablePolicies []Policy

    // 威胁情报
    ThreatIntel     []ThreatIndicator

    // 业务上下文
    ServiceDependency DependencyGraph

    // 近期事件
    RelatedEvents   []Event
}

type Baseline struct {
    Workload        string
    TimeRange       string // "7d", "30d"

    // 流量统计
    AvgConnections  float64
    AvgBytes        float64
    AvgDuration     float64
    UniqueDestIPs   int

    // 协议分布
    ProtocolDist    map[string]float64

    // 时间模式
    HourlyPattern   [24]float64
    WeekdayPattern  [7]float64
}
```

### API 接口

```go
// POST /api/v1/ai/traffic/analyze-anomaly
type AnalyzeAnomalyRequest struct {
    FlowID      string `json:"flow_id"`
    WorkloadID  string `json:"workload_id"`
    TimeRange   string `json:"time_range"` // 分析窗口
    Depth       string `json:"depth"`      // quick/standard/deep
}

type AnalyzeAnomalyResponse struct {
    IsAnomaly       bool                `json:"is_anomaly"`
    Severity        string              `json:"severity"` // info/warning/critical
    AnomalyTypes    []string            `json:"anomaly_types"`
    Explanation     string              `json:"explanation"`
    ThreatAssessment ThreatAssessment   `json:"threat_assessment"`
    Recommendations []Recommendation    `json:"recommendations"`
    RelatedEvents   []Event             `json:"related_events"`
}

type ThreatAssessment struct {
    LikelyScenarios   []Scenario `json:"likely_scenarios"`
    MaliciousProb     float64    `json:"malicious_probability"` // 0-1
    ImpactScore       int        `json:"impact_score"`          // 0-100
    ConfidenceLevel   string     `json:"confidence_level"`      // low/medium/high
}
```

---

## 场景 2: 智能依赖图分析

### 功能描述

分析 Application Dependency Map (应用依赖图),自动识别架构反模式、性能瓶颈和安全风险。

### 分析维度

#### 1. 架构反模式检测

**示例 1: 跳层访问**

```
🏗️ 架构分析报告

检测到反模式: 跳层访问 (Layer Bypass)

【问题描述】
前端服务直接访问数据库,绕过了 API 层:

web-frontend → API Service (✅ 正常路径)
web-frontend → PostgreSQL (❌ 反模式)

【影响分析】
1. 安全风险 🔴
   - 前端直接持有数据库凭证
   - 增加凭证泄露风险
   - 绕过 API 层的权限检查

2. 维护成本 🟡
   - 业务逻辑分散在前端和 API 两处
   - 数据库 schema 变更需要同步修改前端
   - 增加代码复杂度

3. 性能问题 🟡
   - 前端无法利用 API 层的缓存
   - 增加数据库连接数压力

【根因分析】
查看代码提交历史:
- 提交: def456g (2 周前)
- 作者: developer@example.com
- 说明: "紧急修复: 优化查询性能"
- 变更: 前端增加直接数据库查询,绕过"慢"的 API

判断: 🔴 为了短期性能牺牲了架构原则

【推荐方案】
短期 (本周):
1. 保留前端 → 数据库连接,但添加审计日志
2. 添加告警: 当直连查询数超过阈值时告警

中期 (2 周内):
1. 优化 API 层查询性能 (根因: N+1 查询问题)
2. 在 API 层添加结果缓存
3. 前端逐步迁移到 API,测试性能是否满足要求

长期 (1 个月):
1. 完全移除前端 → 数据库连接
2. 添加策略强制执行架构约束
3. 代码审查流程: 禁止跳层访问

预计工作量: 2-3 个开发日
预计性能提升: 响应时间减少 40% (通过缓存)

是否创建改进任务? [创建 Jira Ticket/稍后/忽略]
```

**示例 2: 循环依赖**

```
🏗️ 架构分析报告

检测到反模式: 循环依赖 (Circular Dependency)

【问题描述】
检测到服务间循环依赖:

service-a → service-b → service-c → service-a ⭕

【详细路径】
1. user-service (10.0.1.5) → order-service (10.0.1.6:8080)
2. order-service (10.0.1.6) → payment-service (10.0.1.7:8080)
3. payment-service (10.0.1.7) → user-service (10.0.1.5:8080) ⚠️

【影响分析】
1. 可靠性风险 🔴
   - 任何一个服务故障会导致整个循环瘫痪
   - 级联故障风险高
   - 难以实施熔断保护

2. 启动顺序问题 🔴
   - 无法确定正确的启动顺序
   - 冷启动时可能死锁
   - K8s 滚动更新可能失败

3. 性能问题 🟡
   - 请求可能在服务间反复跳转
   - 延迟累积
   - 难以设置合理的超时时间

【根因分析】
调用链分析:
1. 用户下单 → order-service.CreateOrder()
2. 需要验证支付 → payment-service.ValidatePayment()
3. 支付服务需要查询用户等级 → user-service.GetUserLevel() ⚠️

问题: payment-service 不应该回调 user-service

【推荐方案】
方案 A: 数据传递 (推荐)
- order-service 调用 payment-service 时,直接传递 user_level 参数
- 优点: 简单,无需架构变更
- 缺点: user_level 可能不是最新的

方案 B: 事件驱动
- 使用消息队列(Kafka/RabbitMQ)解耦
- user-service 发布 UserLevelChanged 事件
- payment-service 订阅并缓存用户等级
- 优点: 彻底解耦
- 缺点: 引入新组件,复杂度增加

方案 C: 数据冗余
- payment-service 维护用户等级副本(Redis)
- 定期同步或通过 CDC (Change Data Capture)
- 优点: 高性能
- 缺点: 数据一致性需要保证

推荐: 方案 A (短期) + 方案 B (长期)

预计工作量: 3-5 个开发日
预计可靠性提升: MTBF 提升 300%

是否创建改进任务? [创建/稍后/忽略]
```

#### 2. 性能瓶颈识别

```
⚡ 性能瓶颈分析

工作负载: api-gateway
时间: 2025-01-13 全天

🔍 LLM 分析:

【瓶颈识别】
检测到 api-gateway 是系统的性能瓶颈:

指标                | 值          | 评估
--------------------|-------------|--------
入站连接数           | 15,000/秒   | 🔴 高负载
出站连接数           | 45,000/秒   | 🔴 极高
平均响应时间         | 450ms       | 🟡 偏慢
P99 响应时间         | 2.3秒       | 🔴 差
错误率              | 3.5%        | 🟡 偏高
下游服务调用数       | 平均 3 次/请求| 🔴 过多

【问题分析】
1. N+1 查询问题 🔴
   - 每个请求平均调用 3 个后端服务
   - 串行调用,延迟累积
   - 建议: 实施并行调用或 GraphQL 联邦

2. 缺少缓存层 🟡
   - 50% 的请求是重复查询
   - 建议: 引入 Redis 缓存热点数据

3. 无连接池复用 🔴
   - 每次请求建立新连接
   - TCP 握手开销: ~50ms
   - 建议: 启用 HTTP Keep-Alive

4. 单点瓶颈 🔴
   - 所有流量经过单一 api-gateway
   - 建议: 水平扩展或按业务线拆分

【优化建议】
快速优化 (本周可完成):
1. 启用 HTTP Keep-Alive (预计减少 30% 延迟)
2. 增加 api-gateway 副本数: 2 → 5
3. 添加 Redis 缓存层

中期优化 (2-4 周):
1. 重构为并行调用下游服务
2. 实施请求聚合 (GraphQL DataLoader 模式)
3. 按业务域拆分 gateway

预期效果:
- P99 延迟: 2.3s → 500ms (78% 提升)
- 吞吐量: 15K QPS → 50K QPS (233% 提升)
- 错误率: 3.5% → 0.5% (86% 降低)

成本: +3 个 api-gateway Pod (约 $50/月)
ROI: 用户体验显著提升,减少客户流失

是否执行快速优化? [执行/查看详细方案/忽略]
```

#### 3. 安全风险识别

```
🔒 安全风险分析

依赖图中检测到 5 个安全风险:

【风险 1: 过度暴露】
严重级别: 🔴 高

问题:
database-primary (role=db) 被 12 个不同服务直接访问,包括:
- ✅ api-service (合理)
- ✅ batch-job (合理)
- ❌ frontend-web (不合理,应通过 API)
- ❌ admin-tool (不合理,应使用只读副本)
- ❌ monitoring-agent (不合理,应通过 metrics endpoint)
- ... (其他 7 个)

影响:
- 数据库凭证分散在 12 个地方
- 任何一个服务被入侵都会危及数据库
- 违反最小权限原则

建议:
1. 仅允许 api-service 和 batch-job 访问主库
2. 其他服务:
   - frontend-web → 改为调用 API
   - admin-tool → 使用只读副本
   - monitoring → 使用 postgres_exporter

---

【风险 2: 跨信任域通信】
严重级别: 🟡 中

问题:
检测到跨网络区域的直接通信:

DMZ Zone (public-facing) → Internal Zone (database)

详细:
- nginx-ingress (DMZ) → internal-api (Internal) ✅ 正常
- nginx-ingress (DMZ) → postgres-db (Internal) ❌ 违规

影响:
- DMZ 区域服务被入侵后,可直接访问内网数据库
- 违反纵深防御原则

建议:
添加边界策略:
```yaml
- from: {selector: "zone=dmz"}
  to: {selector: "zone=internal,role=db"}
  action: deny
```

---

【风险 3: 未加密敏感通信】
严重级别: 🟡 中

问题:
支付服务与其他服务的通信未加密:

payment-service ←→ order-service (HTTP, 未加密)
payment-service ←→ user-service (HTTP, 未加密)

影响:
- 支付数据可能被中间人窃取
- 违反 PCI-DSS 4.1 要求

建议:
1. 启用 TLS (mTLS)
2. 或部署 Service Mesh (Istio/Linkerd)
3. 强制: 所有涉及支付的服务必须使用 HTTPS

---

【风险 4: 默认允许出站**
严重级别: 🟡 中

问题:
85% 的工作负载(17/20)缺少出站流量限制:
- 默认允许访问任意外部 IP
- 存在数据泄露风险

建议:
实施默认拒绝策略:
```yaml
- from: {selector: "*"}
  to: {selector: "0.0.0.0/0"}
  action: deny
  priority: 10

# 然后为需要外网访问的服务添加白名单
```

---

【风险 5: 遗留服务暴露】
严重级别: ℹ️ 低

问题:
检测到 3 个遗留服务仍可被访问,但已 6 个月无流量:
- legacy-api-v1 (last_seen: 2024-07-10)
- old-payment-gateway (last_seen: 2024-08-15)
- deprecated-auth-service (last_seen: 2024-06-01)

建议:
1. 确认这些服务是否还需要
2. 如不需要,下线并删除
3. 如需保留,添加严格的访问限制

---

📊 安全评分: 65/100 (中等)

改进后预期评分: 90/100

是否生成安全加固方案? [生成/稍后/忽略]
```

### API 接口

```go
// GET /api/v1/ai/dependencies/analyze
type AnalyzeDependenciesRequest struct {
    TimeRange   string   `json:"time_range"` // 分析时间窗口
    Focus       string   `json:"focus"`      // architecture/performance/security/all
}

type AnalyzeDependenciesResponse struct {
    DependencyGraph  Graph                `json:"dependency_graph"`
    Issues           []ArchitectureIssue  `json:"issues"`
    Recommendations  []Recommendation     `json:"recommendations"`
    HealthScore      int                  `json:"health_score"` // 0-100
}

type ArchitectureIssue struct {
    Type         string   `json:"type"` // layer_bypass/circular_dependency/bottleneck/security_risk
    Severity     string   `json:"severity"`
    Description  string   `json:"description"`
    AffectedNodes []string `json:"affected_nodes"`
    Solutions    []Solution `json:"solutions"`
}
```

---

## 场景 3: Top Talkers 根因分析

### 功能描述

自动分析流量排行榜(Top Talkers),解释为什么某些工作负载流量异常,并提供优化建议。

### 分析示例

```
📊 Top Talkers 分析报告

时间范围: 2025-01-13 00:00 - 23:59

🏆 流量排行 (按字节数)

排名 | 工作负载 | 流量 | 占比 | 趋势 | 分析
-----|---------|------|------|------|------
1    | cache-redis-1 | 156 GB | 45% | ↑ +300% | 🔍 异常
2    | api-gateway | 82 GB | 23% | → 正常 | ✅ 正常
3    | db-postgres-primary | 55 GB | 16% | ↑ +50% | 🔍 需注意
4    | log-collector | 28 GB | 8% | → 正常 | ✅ 正常
5    | file-storage | 18 GB | 5% | ↓ -20% | ✅ 正常

---

🔍 #1 异常分析: cache-redis-1

【流量激增 +300%】

LLM 根因分析:

1. 时间线分析 📈
   ```
   00:00-08:00: 15 GB/h (正常)
   08:00-10:00: 20 GB/h (正常,工作时间开始)
   10:00-10:15: 突增至 180 GB/h ⚠️
   10:15-23:59: 持续 150 GB/h (异常持续)
   ```

2. 关联事件 🔗
   - 10:05: db-postgres-primary 重启 (维护窗口)
   - 10:08: cache 未命中率从 5% 激增至 45% ⚠️
   - 10:10: 所有应用服务开始大量查询 cache
   - 10:15: cache 开始大量从 DB 回填数据

3. 根因定位 🎯
   数据库重启 → 触发 cache 失效(设计缺陷) → 缓存雪崩 → 大量回源

   详细原因:
   - Redis 使用了基于 DB 连接的缓存失效机制
   - DB 重启导致 Redis 误判所有缓存过期
   - 所有应用同时回源重建缓存
   - 缺少缓存预热和限流保护

4. 影响评估 💥
   - ✅ 业务未中断 (cache 和 DB 扛住了压力)
   - 🟡 响应延迟增加 3 倍 (10:10-10:30)
   - 🟡 DB CPU 使用率 100% (10 分钟)
   - 💰 流量成本增加 $150 (跨 AZ 流量)

5. 预防措施 🛡️
   短期 (本周):
   - 修复缓存失效逻辑,不依赖 DB 连接状态
   - 实施缓存预热脚本,在 DB 重启后批量回填

   中期 (2 周):
   - 实施二级缓存(本地 + 远程)
   - 添加限流保护,防止缓存雪崩
   - 数据库重启前发送通知,应用提前准备

   长期 (1 个月):
   - 实施缓存一致性协议(如 Lease 机制)
   - 部署缓存观测性工具(命中率/失效原因)
   - 制定缓存灾难恢复预案

6. 成本优化 💰
   通过以上改进,预计:
   - 减少 80% 的缓存回填流量
   - 节约跨 AZ 流量成本 $400/月
   - 提升响应速度 50%

---

🔍 #3 需注意: db-postgres-primary

【流量增长 +50%】

LLM 趋势分析:

1. 增长模式 📊
   - 过去 7 天平均: 37 GB/天
   - 今天: 55 GB/天 (+48%)
   - 趋势: 连续 5 天增长,每天 +8%

2. 增长来源 🔍
   分解流量来源:
   - api-service: +5 GB (正常业务增长)
   - batch-job: +8 GB (新增夜间报表任务) ✅
   - data-sync: +5 GB (新增数据同步)

   主要贡献: batch-job 夜间报表

3. 容量预测 📈
   按当前趋势,预计:
   - 7 天后: 75 GB/天 (接近 80 GB 容量阈值)
   - 14 天后: 100 GB/天 (超出容量 25%) ⚠️

4. 推荐行动 💡
   立即行动:
   - ✅ 容量预警已发送给 DBA 团队
   - ⏳ 等待人工决策...

   可选方案:
   A. 垂直扩展: 升级 DB 实例 (成本 +$200/月)
   B. 优化查询: 优化 batch-job 的 SQL (开发 2-3 天)
   C. 读写分离: 报表查询走只读副本 (开发 1 周)

   推荐: 方案 C (长期收益最大)

---

📈 总体趋势分析

总流量: 339 GB/天 (过去 7 天平均: 280 GB/天,+21%)

增长驱动因素:
1. 业务增长: +10% (正常)
2. 缓存雪崩: +8% (偶发)
3. 新功能上线: +3% (batch-job)

成本影响:
- 跨 AZ 流量费用: $450/天 (↑ $80)
- 数据库 IOPS: 接近容量上限

💡 优化建议:
1. 【高优先级】修复缓存雪崩问题 (节约 $400/月)
2. 【中优先级】实施读写分离 (提升容量 100%)
3. 【低优先级】优化跨 AZ 流量路由 (节约 $100/月)

是否生成优化工单? [生成/稍后/忽略]
```

### API 接口

```go
// GET /api/v1/ai/top-talkers/analyze
type AnalyzeTopTalkersRequest struct {
    TimeRange   string `json:"time_range"`
    Limit       int    `json:"limit"`      // Top N
    SortBy      string `json:"sort_by"`    // bytes/packets/connections
}

type AnalyzeTopTalkersResponse struct {
    TopTalkers   []TopTalker `json:"top_talkers"`
    Insights     []Insight   `json:"insights"`
    Trends       Trend       `json:"trends"`
    CostAnalysis CostAnalysis `json:"cost_analysis"`
}

type TopTalker struct {
    WorkloadID   string  `json:"workload_id"`
    Bytes        int64   `json:"bytes"`
    Percentage   float64 `json:"percentage"`
    Trend        string  `json:"trend"` // increasing/stable/decreasing
    Analysis     string  `json:"analysis"` // LLM 生成的分析
    IsAnomaly    bool    `json:"is_anomaly"`
    RootCause    string  `json:"root_cause,omitempty"`
}
```

---

## 实施建议

### 1. 数据准备

**历史基线建立**:
- 至少收集 7 天的流量数据
- 计算各工作负载的正常基线
- 建立时间模式(小时/星期几)
- 存储在时序数据库(可选)

```sql
-- 基线计算示例
CREATE TABLE flow_baselines (
    workload_id VARCHAR(255),
    metric_name VARCHAR(50),
    hour_of_day INT,
    day_of_week INT,
    avg_value FLOAT,
    stddev_value FLOAT,
    p95_value FLOAT,
    p99_value FLOAT,
    updated_at TIMESTAMP
);
```

### 2. 实时处理流程

```
Flow Event → Baseline Comparison → Anomaly Score → LLM Analysis → Alert/Report
    ↓              ↓                      ↓              ↓              ↓
  SQLite      Time Series DB         Scoring        OpenAI API    Notification
```

### 3. Prompt 设计要点

- 提供完整的上下文(历史数据、策略、工作负载信息)
- 使用结构化输出(JSON Schema)
- 要求 LLM 给出置信度评分
- Few-Shot Learning: 提供异常分析示例

### 4. 性能优化

- 仅对异常流量调用 LLM(节约成本)
- 批量分析(而非单条)
- 缓存常见分析结果
- 使用较小模型处理简单情况

---

**下一篇**: [自动化策略生成 →](03-automated-policy-generation.md)
