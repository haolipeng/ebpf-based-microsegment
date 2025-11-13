# 自动化策略生成（学习模式）

## 概述

手工配置策略是微隔离产品最大的痛点之一。传统方式需要：
- 深入理解应用架构和通信模式
- 手工编写数百甚至数千条策略规则
- 持续维护和更新策略

**学习模式(Learning Mode)** 通过观察实际流量,结合 LLM 的智能分析,可以自动生成符合最小权限原则的策略,大幅降低配置工作量。

---

## 核心理念

### 工作流程

```
📊 观察阶段 (7-30天)
    ↓
  只记录,不阻断
    ↓
收集所有 Flow 数据
    ↓
📈 分析阶段 (AI处理)
    ↓
识别稳定通信模式
    ↓
过滤噪声和异常流量
    ↓
生成策略草稿
    ↓
👤 审核阶段 (人工)
    ↓
查看推荐策略
    ↓
调整/批准
    ↓
🚀 部署阶段
    ↓
一键应用策略
    ↓
🔄 持续优化
    ↓
监控效果,定期调整
```

---

## 场景 1: 观察模式 + 自动策略生成

### 阶段 1: 启动观察模式

```
用户: "我想为新部署的应用配置微隔离策略"

AI: 好的!我推荐使用 **学习模式** 来自动生成策略。

学习模式工作原理:
1. 📊 观察期(7-30天): 记录所有流量,不阻断
2. 🤖 AI 分析: 识别稳定的通信模式
3. 📋 生成策略: 自动创建最小权限策略
4. 👤 人工审核: 您可以调整后再部署

推荐观察期:
- 7 天: 适用于流量模式简单且稳定的应用
- 14 天: 【推荐】适用于大多数场景
- 30 天: 适用于有周期性业务的应用(如月末结算)

您的应用类型: [Web服务/API服务/批处理/其他]
选择观察期: [7天/14天/30天/自定义]
```

**用户选择**: 14天

```
AI: 已为您启动 14 天观察期 ✅

配置详情:
- 工作负载范围: 所有标签包含 app=myapp 的工作负载 (8 个 Pod)
- 观察开始时间: 2025-01-13 10:00:00
- 预计结束时间: 2025-01-27 10:00:00
- 策略模式: LOG (记录但不阻断)

系统将自动:
✅ 记录所有入站和出站流量
✅ 每天生成流量摘要报告
✅ 提前 24 小时通知观察期即将结束
✅ 在第 14 天自动生成策略推荐

进度追踪:
- Web Dashboard: https://console.example.com/learning-mode/session-001
- API: GET /api/v1/ai/learning-sessions/session-001

您可以随时:
- 查看实时流量: /learning-sessions/session-001/flows
- 提前结束观察: /learning-sessions/session-001/stop
- 延长观察期: /learning-sessions/session-001/extend

祝观察顺利! 🚀
```

### 阶段 2: 观察期间 - 每日摘要

```
📊 学习模式 - 第 3 天摘要报告

会话: session-001 (app=myapp)
时间: 2025-01-15

【今日流量统计】
- 总 Flow 数: 12,450
- 唯一源 IP: 15
- 唯一目标 IP: 23
- 协议分布: TCP 95%, UDP 4%, ICMP 1%

【检测到的通信模式】
稳定模式 (连续 3 天):
1. myapp-web → myapp-api:8080 (TCP) - 5,200 次/天
2. myapp-api → postgres-db:5432 (TCP) - 3,800 次/天
3. myapp-api → redis-cache:6379 (TCP) - 2,100 次/天
4. myapp-web → cdn.example.com:443 (TCP) - 1,500 次/天

临时模式 (仅今天):
5. myapp-api → external-api.partner.com:443 - 15 次 ⚠️
   → LLM 判断: 可能是测试流量或偶发调用,继续观察

异常流量 (需注意):
6. myapp-web → 185.220.101.x:6667 - 3 次 🔴
   → LLM 判断: 目标 IP 在威胁情报库中,疑似恶意连接
   → 建议: 立即调查,不要等观察期结束

【工作负载发现】
新增工作负载:
- myapp-worker-1 (role=worker) - 今日首次出现
  流量模式: 仅访问 redis-cache 和 message-queue

【趋势分析】
- 流量增长: +15% (符合业务增长)
- 模式稳定度: 85% (高,有利于策略生成)
- 预计策略数量: 8-12 条

下次报告: 2025-01-16
查看详情: https://console.example.com/learning-mode/session-001
```

### 阶段 3: 观察期结束 - 策略推荐

```
🎉 学习模式观察期结束

会话: session-001 (app=myapp)
观察期: 2025-01-13 至 2025-01-27 (14 天)

【数据收集汇总】
- 总 Flow 数: 168,420
- 观察天数: 14 天
- 覆盖工作负载: 9 个 (1 个中途加入)
- 数据完整性: 99.2% ✅

---

📋 AI 生成的策略推荐

共生成 12 条策略 (3 组)

═══════════════════════════════════════
组 1: 前端 → API 通信
═══════════════════════════════════════

【策略 1】Allow Web to API
```yaml
from:
  selector: role=web,app=myapp
to:
  selector: role=api,app=myapp
ports:
  - protocol: tcp
    port: 8080
action: allow
direction: egress
priority: 100
```

📊 证据:
- 观察到流量: 72,800 次 (平均 5,200 次/天)
- 出现频率: 14/14 天 (100%)
- 时间模式: 工作时间高峰,夜间也有少量
- 流量特征: 小数据包,高频,短连接
- 置信度: ⭐⭐⭐⭐⭐ 极高

💡 LLM 判断:
这是核心业务流量,典型的前后端分离架构。
建议: 必须放行 ✅

---

【策略 2】Block Web Direct to DB
```yaml
from:
  selector: role=web,app=myapp
to:
  selector: role=db
action: deny
direction: any
priority: 100
```

📊 证据:
- 观察到流量: 0 次
- LLM 判断: 虽然未观察到违规流量,但根据最佳实践,
  应主动阻止前端直接访问数据库

💡 LLM 判断:
预防性策略,符合三层架构原则。
建议: 建议添加 ✅

---

═══════════════════════════════════════
组 2: API → 后端服务通信
═══════════════════════════════════════

【策略 3】Allow API to Database
```yaml
from:
  selector: role=api,app=myapp
to:
  selector: role=db,name=postgres-primary
ports:
  - protocol: tcp
    port: 5432
action: allow
direction: egress
priority: 100
```

📊 证据:
- 观察到流量: 53,200 次 (平均 3,800 次/天)
- 出现频率: 14/14 天 (100%)
- 时间模式: 与 Web 流量高度相关 (r=0.95)
- 流量特征: 中等数据包,事务型
- 置信度: ⭐⭐⭐⭐⭐ 极高

💡 LLM 判断:
核心数据访问路径,必须放行 ✅

---

【策略 4】Allow API to Cache
```yaml
from:
  selector: role=api,app=myapp
to:
  selector: role=cache,name=redis-primary
ports:
  - protocol: tcp
    port: 6379
action: allow
direction: egress
priority: 100
```

📊 证据:
- 观察到流量: 29,400 次 (平均 2,100 次/天)
- 出现频率: 14/14 天 (100%)
- 流量特征: 小数据包,极高频,亚毫秒级
- 置信度: ⭐⭐⭐⭐⭐ 极高

💡 LLM 判断:
缓存访问,提升性能的关键路径 ✅

---

【策略 5】Allow API to External Partner API (需审核)
```yaml
from:
  selector: role=api,app=myapp
to:
  ip: 203.0.113.50/32  # external-api.partner.com
ports:
  - protocol: tcp
    port: 443
action: allow
direction: egress
priority: 100
```

📊 证据:
- 观察到流量: 180 次 (平均 12.8 次/天)
- 出现频率: 11/14 天 (78%)
- 流量特征: 低频,白天出现,HTTPS
- 置信度: ⭐⭐⭐⚪⚪ 中等

⚠️  LLM 判断:
这是外部 API 调用,流量较少且不是每天都有。
可能场景:
1. 第三方服务集成 (如支付/短信网关)
2. 数据同步任务
3. 测试代码未清理

建议:
- 如果是必需的第三方服务,放行 ✅
- 如果是测试代码,删除并阻断 ❌
- 如果不确定,可以先设为 'log' 模式观察

👤 需要您确认: 这个外部 API 调用是否必需?
[必需,放行 / 不必需,阻断 / 不确定,继续观察]

---

═══════════════════════════════════════
组 3: Worker 后台任务
═══════════════════════════════════════

【策略 6】Allow Worker to Message Queue
```yaml
from:
  selector: role=worker,app=myapp
to:
  selector: role=mq,name=rabbitmq
ports:
  - protocol: tcp
    port: 5672
action: allow
direction: any
priority: 100
```

📊 证据:
- 观察到流量: 8,640 次 (工作负载从第 3 天才出现)
- 出现频率: 12/12 天 (100%,从出现开始)
- 流量特征: 长连接,双向通信
- 置信度: ⭐⭐⭐⭐⚪ 高

💡 LLM 判断:
后台任务消费队列消息,标准架构 ✅

---

═══════════════════════════════════════
组 4: 默认拒绝策略
═══════════════════════════════════════

【策略 7】Default Deny All (推荐)
```yaml
from:
  selector: app=myapp
to:
  selector: "*"
action: deny
direction: any
priority: 50  # 较低优先级,作为兜底
```

💡 LLM 判断:
最佳实践: 默认拒绝所有未明确允许的流量。
这是零信任架构的基础原则。

建议: 强烈建议添加 ✅

---

═══════════════════════════════════════
未观察到但建议添加的策略
═══════════════════════════════════════

【策略 8】Allow Health Checks (推荐)
```yaml
from:
  selector: role=monitoring
to:
  selector: app=myapp
ports:
  - protocol: tcp
    port: 8080  # /health endpoint
action: allow
priority: 100
```

💡 LLM 判断:
虽然观察期未看到 Prometheus 抓取指标,但根据您的环境配置,
检测到 Prometheus 已部署。建议预先添加此策略,避免监控失效。

---

【策略 9】Block Outbound Internet (推荐)
```yaml
from:
  selector: app=myapp
to:
  ip: 0.0.0.0/0  # 所有外网 IP
exclude:
  - 203.0.113.50/32  # 已允许的 Partner API
action: deny
direction: egress
priority: 60
```

💡 LLM 判断:
数据泄露防护。除了明确需要的外部服务,阻止所有出站互联网流量。

---

═══════════════════════════════════════
📊 策略集摘要
═══════════════════════════════════════

总计: 12 条策略

分类:
- ✅ 高置信度 (必须): 7 条
- ⚠️  中等置信度 (需确认): 1 条 (策略 5)
- 💡 最佳实践 (推荐): 4 条

预计影响:
- 允许的 Flow: 168,240 (99.9%)
- 阻断的 Flow: 180 (0.1%, 都是不确定的外部调用)

安全评分: 90/100 (优秀)
- ✅ 实施最小权限原则
- ✅ 默认拒绝策略
- ✅ 分层架构保护
- ⚠️  有 1 条策略需要人工确认

═══════════════════════════════════════
下一步操作
═══════════════════════════════════════

选项 A: 一键部署 (推荐)
- 自动应用所有高置信度策略
- 中等置信度策略设为 'log' 模式
- 预计部署时间: < 30 秒

选项 B: 逐条审核
- 在 Web UI 中查看每条策略详情
- 手工调整后部署

选项 C: 导出配置
- 下载 YAML 文件
- 经过团队评审后手工部署

选项 D: 延长观察期
- 再观察 7-30 天
- 收集更多数据提高置信度

您的选择: [A/B/C/D]
```

### 阶段 4: 部署后监控

**用户选择**: A (一键部署)

```
🚀 正在部署策略...

进度:
[████████████████████] 100%

✅ 部署完成!

部署摘要:
- 成功应用策略: 11 条
- Log 模式策略: 1 条 (策略 5,等待确认)
- 失败: 0 条

当前策略生效中...

---

📊 部署后首次监控报告 (1 小时后)

时间: 2025-01-27 11:00

【策略效果】
- 允许的流量: 5,245 次 ✅
- 阻断的流量: 0 次 ✅
- 记录的流量: 1 次 (策略 5,外部 API)

【业务影响】
- ❌ 零误报(没有业务流量被错误阻断)
- ✅ 应用响应时间: 正常 (p99: 50ms)
- ✅ 错误率: 0.1% (基线水平)

【安全提升】
新增防护:
- ✅ 阻止前端直接访问数据库(虽然未发生,但已预防)
- ✅ 阻止未授权的外网访问
- ✅ 最小权限原则: 仅放行必要流量

【待处理事项】
⚠️  策略 5 (外部 API) 等待确认:
- 过去 1 小时触发 1 次
- 请确认是否需要放行: [需要/不需要]

---

📈 7 天后效果评估

部署时间: 2025-01-27
评估时间: 2025-02-03

【策略有效性】
好消息! 所有策略工作正常 🎉

详细统计:
策略            | 命中次数  | 阻断次数 | 误报
----------------|----------|---------|------
Web to API      | 36,400   | 0       | 0
API to DB       | 26,600   | 0       | 0
API to Cache    | 14,700   | 0       | 0
Default Deny    | 0        | 45      | 0 ✅

【阻断分析】
Default Deny 策略阻断了 45 次访问:
- 32 次: 外部 IP 扫描尝试 (恶意流量) 🔴
- 8 次: 开发环境误配置尝试访问生产 🟡
- 5 次: 未知来源 UDP 流量 ⚪

LLM 判断: 这 45 次阻断都是不合规流量,策略正确 ✅

【优化建议】
当前策略集运行良好,无需调整。

建议每月重新运行学习模式,以适应:
- 新功能上线
- 架构变化
- 业务增长

下次学习模式推荐时间: 2025-02-27
```

---

## 场景 2: 策略有效性持续评估

### 自动评估触发

系统自动在以下时机评估策略:
1. **部署后 1 小时** - 快速检查是否有误报
2. **部署后 24 小时** - 首日完整评估
3. **部署后 7 天** - 一周效果评估
4. **每月 1 日** - 定期健康检查
5. **检测到异常时** - 即时评估

### 评估报告示例

```
📊 策略有效性月度评估报告

策略集: myapp-policies (12 条策略)
评估时间: 2025-02-01
部署天数: 35 天

═══════════════════════════════════════
📈 整体健康度
═══════════════════════════════════════

评分: 88/100 (良好)

评分构成:
- 有效性: 95/100 (优秀) ✅
  → 所有策略都有实际流量命中
  → 无冗余策略

- 准确性: 90/100 (优秀) ✅
  → 误报率: 0%
  → 漏报率: < 0.1%

- 覆盖率: 85/100 (良好) ⚠️
  → 87% 的流量被显式策略覆盖
  → 13% 的流量走默认策略

- 维护性: 80/100 (良好) ⚪
  → 策略数量适中 (12 条)
  → 但有 2 条策略可以合并

═══════════════════════════════════════
🔍 策略详细分析
═══════════════════════════════════════

【策略 1: Web to API】
状态: ✅ 健康
- 命中次数: 185,240 (平均 5,292/天)
- 趋势: 稳定增长 +5%/周 (符合业务增长)
- 误报: 0 次
- 建议: 无需调整

---

【策略 3: API to Database】
状态: ⚠️  需注意
- 命中次数: 142,800
- 趋势: 快速增长 +15%/周 ⚠️
- 分析: 数据库查询量增长快于业务流量

🔍 LLM 深度分析:
问题: API → DB 流量增长(+15%/周)快于 Web → API (+5%/周)
可能原因:
1. 出现 N+1 查询问题
2. 缓存命中率下降
3. 新功能增加了数据库调用

建议:
1. 检查最近的代码变更
2. 分析慢查询日志
3. 优化 ORM 查询或增加缓存

预计影响:
- 1 个月后数据库将达到容量上限
- 建议 2 周内优化

已创建工单: PERF-2025-0201-001

---

【策略 5: API to External Partner】
状态: ℹ️  使用率低
- 命中次数: 156 (平均 4.5/天)
- 趋势: 稳定
- 分析: 低频调用,符合预期

💡 LLM 建议:
这是第三方 API 调用,使用频率很低。
建议添加告警:
- 如果调用失败率 > 10%,通知团队
- 如果调用频率突增 10 倍,检查是否异常

---

【策略 7: Default Deny】
状态: ✅ 有效防护
- 阻断次数: 1,240
- 趋势: 稳定

阻断流量分类:
- 67%: 外部扫描和攻击尝试 🔴
- 18%: 开发环境误配置 🟡
- 10%: 未注册的新服务尝试通信 ⚪
- 5%: 其他

🛡️  安全价值:
过去 35 天,成功阻止 831 次恶意访问尝试,
包括:
- 端口扫描: 456 次
- SQL 注入尝试: 89 次 (针对 3306 端口)
- SSH 暴力破解: 156 次
- 其他: 130 次

═══════════════════════════════════════
🎯 优化建议
═══════════════════════════════════════

【建议 1: 合并相似策略】
优先级: 低
策略 3 和策略 6 (Worker to DB) 可以合并:
```yaml
from:
  selector: app=myapp,role in (api,worker)
to:
  selector: role=db,name=postgres-primary
```
好处: 减少策略数量,提升可维护性

【建议 2: 添加速率限制】
优先级: 中
针对策略 1 (Web to API),建议添加速率限制:
- 每 IP 最多 1000 req/min
- 防止 DDoS 和滥用
- 需要部署 rate-limiter 组件

【建议 3: 细化 Default Deny】
优先级: 低
当前 Default Deny 会阻断所有未知流量。
建议改为:
- 内网流量: log (记录但不阻断,便于发现新服务)
- 外网流量: deny (阻断,防止数据泄露)

【建议 4: 添加告警规则】
优先级: 高 ⚠️
当前缺少自动告警,建议添加:
- 策略阻断次数 > 100/小时 → P2 告警
- 新流量模式出现 → P3 通知
- 策略命中率突变 → P2 告警

═══════════════════════════════════════
📅 下次评估
═══════════════════════════════════════

- 定期评估: 2025-03-01
- 或任何时候运行: /policies/evaluate

建议:
每 3 个月重新运行学习模式,保持策略最新。

是否应用上述优化建议? [应用全部/选择性应用/查看详情/忽略]
```

---

## 技术实现要点

### 1. 数据收集

```go
type LearningSession struct {
    ID              string    `json:"id"`
    Name            string    `json:"name"`
    WorkloadScope   string    `json:"workload_scope"` // 标签选择器
    StartTime       time.Time `json:"start_time"`
    EndTime         time.Time `json:"end_time"`
    Duration        int       `json:"duration"` // 天数
    Status          string    `json:"status"`   // active/completed/cancelled

    // 统计
    TotalFlows      int64     `json:"total_flows"`
    UniqueSourceIPs int       `json:"unique_source_ips"`
    UniqueDestIPs   int       `json:"unique_dest_ips"`

    // 配置
    MinOccurrence   int       `json:"min_occurrence"` // 最少出现次数
    MinFrequency    float64   `json:"min_frequency"`  // 最小频率 (0-1)

    // 结果
    GeneratedPolicies []Policy `json:"generated_policies"`
}
```

### 2. 模式识别算法

```go
type FlowPattern struct {
    SourceSelector  string
    DestSelector    string
    Protocol        string
    Port            int

    // 统计特征
    TotalCount      int
    DaysObserved    int
    Frequency       float64  // 出现频率
    AvgPerDay       float64
    StdDev          float64

    // 时间特征
    HourlyDistribution [24]int
    WeekdayDistribution [7]int

    // 置信度
    Confidence      float64  // 0-1
}

func (s *LearningSession) AnalyzePatterns() []FlowPattern {
    // 1. 从 SQLite 加载所有 Flow 数据
    flows := s.LoadFlows()

    // 2. 按 5-tuple 分组
    grouped := GroupFlows(flows)

    // 3. 计算统计特征
    patterns := []FlowPattern{}
    for key, flowList := range grouped {
        pattern := CalculatePattern(flowList)

        // 4. 过滤不稳定的模式
        if pattern.Frequency >= s.MinFrequency &&
           pattern.TotalCount >= s.MinOccurrence {
            patterns = append(patterns, pattern)
        }
    }

    // 5. 排序(按置信度)
    sort.Slice(patterns, func(i, j int) bool {
        return patterns[i].Confidence > patterns[j].Confidence
    })

    return patterns
}
```

### 3. LLM Prompt 设计

```python
POLICY_GENERATION_PROMPT = """
你是一个微隔离策略生成专家。基于观察到的流量模式,生成符合最小权限原则的策略。

观察数据:
- 观察期: {duration} 天
- 总流量: {total_flows} 条
- 工作负载: {workloads}

检测到的流量模式:
{patterns}

现有架构信息:
- 应用类型: {app_type}
- 部署环境: {environment}
- 合规要求: {compliance}

任务:
1. 为每个稳定的流量模式生成 ALLOW 策略
2. 识别应该阻止但未观察到的流量,生成预防性 DENY 策略
3. 添加默认拒绝策略
4. 为低置信度的模式提供人工审核建议

输出格式(JSON):
{
  "policies": [
    {
      "policy": {...},
      "evidence": {
        "flow_count": 1234,
        "days_observed": 14,
        "confidence": 0.95
      },
      "rationale": "解释为什么需要这条策略",
      "requires_review": false
    }
  ],
  "summary": {
    "total_policies": 10,
    "high_confidence": 7,
    "needs_review": 3
  }
}

重要原则:
- 最小权限: 只放行必要的流量
- 具体优于宽泛: 优先使用具体的标签选择器
- 安全优先: 有疑问时宁可阻断,让用户审核
- 最佳实践: 参考 NIST, CIS 等标准
"""
```

### 4. API 接口

```go
// POST /api/v1/ai/learning-sessions
type CreateLearningSessionRequest struct {
    Name            string `json:"name"`
    WorkloadScope   string `json:"workload_scope"`
    Duration        int    `json:"duration"` // 天数
    MinOccurrence   int    `json:"min_occurrence"`
}

// GET /api/v1/ai/learning-sessions/:id
// GET /api/v1/ai/learning-sessions/:id/patterns
// POST /api/v1/ai/learning-sessions/:id/generate-policies
// POST /api/v1/ai/learning-sessions/:id/deploy-policies

// GET /api/v1/ai/policies/:id/evaluation
type PolicyEvaluationResponse struct {
    PolicyID        string              `json:"policy_id"`
    HealthScore     int                 `json:"health_score"`
    HitCount        int64               `json:"hit_count"`
    BlockCount      int64               `json:"block_count"`
    FalsePositives  int                 `json:"false_positives"`
    Trend           string              `json:"trend"`
    Recommendations []Recommendation    `json:"recommendations"`
}
```

---

## 用户体验优化

### 1. 进度可视化

```
观察期进度:

[████████░░░░░░░░░░░░] 40% (6/14 天)

流量收集:
✅ 已收集 85,240 条 Flow
✅ 识别 8 个稳定模式
⏳ 需要更多数据: 3 个临时模式

下一步: 2025-01-21 将生成中期报告
```

### 2. 交互式策略审核

```
策略 #5: Allow API to External Partner

📊 证据:
[████████░░░░░░░░░░░░] 置信度: 65%

详细数据:
- 流量次数: 180
- 出现天数: 11/14 天
- 流量曲线: [显示图表]

⚠️  需要您的决定:
这个外部 API 是否必需?

[ ] 必需,放行
[ ] 不必需,阻断
[ ] 不确定,告诉我更多信息

[如选择"更多信息"]

💡 这可能是什么?
我查看了您的代码仓库(如果集成),发现:
- 文件: src/services/payment.go
- 代码: http.Post("https://external-api.partner.com/webhook"...)
- 注释: "通知合作伙伴订单状态"

判断: 这是必需的业务功能 ✅

建议: 选择"必需,放行"
```

### 3. 一键回滚

```
❌ 策略部署后发现问题?

快速回滚:
[点击回滚到部署前状态]

预计影响:
- 删除 12 条新策略
- 恢复 LOG 模式
- 回滚时间: < 5 秒

所有流量数据已保留,可以稍后重新分析。
```

---

## 实施建议

### 1. MVP 功能

- ✅ 基础数据收集
- ✅ 简单模式识别(频率统计)
- ✅ LLM 策略生成
- ✅ 人工审核界面

### 2. 高级功能

- 时间序列分析
- 异常流量过滤
- 多租户支持
- A/B 测试(金丝雀部署)

### 3. 性能考虑

- 大规模环境(10K+ Flow/天): 使用采样
- 实时分析: 流式处理 vs 批量处理
- 存储优化: 定期归档老数据

---

**下一篇**: [智能运维与故障排查 →](04-intelligent-operations.md)
