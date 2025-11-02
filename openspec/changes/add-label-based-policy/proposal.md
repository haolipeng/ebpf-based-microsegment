# 提案：添加基于标签的策略管理

## 概述
为 eBPF 微隔离平台实现基于标签的策略管理系统，支持 Illumio 风格的工作负载抽象和组到组的策略规则。

## 为什么

当前策略系统仅限于基于 IP 的五元组规则（SrcIP/DstIP/Port/Protocol），存在显著的运维挑战：

1. **缺乏工作负载身份**：策略绑定到 IP，当容器/Pod 重启后获得新 IP 时会失效
2. **手动管理**：运维人员必须手动跟踪哪个 IP 属于哪个应用角色
3. **缺乏语义**：无法表达"所有 Web 服务器可以访问所有数据库"这样的意图
4. **可扩展性问题**：N 个服务 × M 个服务 = N×M 条策略规则需要手动管理
5. **缺乏云原生集成**：无法利用 Kubernetes 标签、命名空间或服务元数据

**业务影响**：
- 竞品（Illumio、蔷薇灵动）使用基于标签的策略作为核心差异化功能
- 基于标签的策略将运维复杂度降低 10-100 倍
- 支持工作负载级别隔离的零信任架构

## 变更内容

本提案在现有 eBPF 数据平面之上添加**三层抽象**：

```
第 1 层（用户意图）：    PolicyRule: "web-tier" → "db-tier" on tcp/3306 = allow
第 2 层（编译）：        解析组，展开为 IP（N × M 笛卡尔积）
第 3 层（执行）：        CompiledPolicy: 10.0.1.10 → 10.0.2.20 on tcp/3306 = allow
第 4 层（数据平面）：    现有 eBPF maps（无需修改）
```

### 核心能力

**1. 工作负载抽象**
- 使用唯一 ID 跟踪运行中的容器/进程
- 分配任意键值标签（role=web, app=frontend, env=prod 等）
- 存储工作负载元数据（IP、MAC、端口、镜像、命名空间）

**2. 标签系统**
- 支持 Illumio 的四维模型作为推荐结构：
  - **Role（角色）**：技术角色（web, api, db, cache, mq, worker）
  - **App（应用）**：业务应用（frontend, backend, auth, payment）
  - **Env（环境）**：环境（prod, staging, dev, test）
  - **Location（位置）**：物理/逻辑位置（region, AZ, datacenter）
- 允许超出这四个维度的自定义标签
- 基础自动打标：从镜像名称推断角色（nginx → web）和端口（3306 → db）

**3. 组系统**
- 通过标签选择器定义组（类似 Kubernetes）
- 支持的操作符：`=`, `!=`, `in`, `not-in`, `exists`, `not-exists`
- 动态成员解析（当标签变化时工作负载自动加入/离开组）
- 示例：组 "web-prod" 选择满足 `role=web AND env=prod` 的工作负载

**4. 基于组的策略规则**
- 用户在组之间定义策略：`web-tier → db-tier`
- 指定网络约束：端口（tcp/3306）、协议
- 设置动作：allow、deny 或 log
- 系统自动编译为基于 IP 的规则

**5. 策略编译引擎**
- 解析组成员：哪些工作负载属于每个组？
- 将组到组规则展开为 IP 到 IP 规则（笛卡尔积）
- 跟踪溯源：哪条编译规则来自哪个高级策略？
- 当工作负载/组变化时增量重新编译

### 示例工作流

**之前（基于 IP）**：
```
# 运维人员必须手动跟踪 IP 并创建 N×M 条规则
Policy 1: 10.0.1.10 → 10.0.2.20 tcp/3306 allow  # web1 → db1
Policy 2: 10.0.1.10 → 10.0.2.21 tcp/3306 allow  # web1 → db2
Policy 3: 10.0.1.11 → 10.0.2.20 tcp/3306 allow  # web2 → db1
Policy 4: 10.0.1.11 → 10.0.2.21 tcp/3306 allow  # web2 → db2
# 当容器重启获得新 IP 时，所有规则都必须更新！
```

**之后（基于标签）**：
```
# 定义工作负载（第 2 阶段将支持自动发现）
Workload web1:  IP=10.0.1.10, labels={role:web, app:frontend, env:prod}
Workload web2:  IP=10.0.1.11, labels={role:web, app:frontend, env:prod}
Workload db1:   IP=10.0.2.20, labels={role:db, app:backend, env:prod}
Workload db2:   IP=10.0.2.21, labels={role:db, app:backend, env:prod}

# 定义组
Group "web-frontend": role=web AND app=frontend AND env=prod
Group "mysql-databases": role=db AND app=backend AND env=prod

# 定义一条策略规则
PolicyRule "allow-web-to-db": web-frontend → mysql-databases tcp/3306 allow

# 系统自动编译为 4 条 IP 规则（与之前相同）
# 但当 IP 变化时，规则自动更新！
```

## 范围

### 范围内（第 1 阶段 - MVP）
- [x] 工作负载数据模型（ID、IP、标签、元数据）
- [x] 工作负载 CRUD API 和存储（SQLite）
- [x] 标签系统（任意键值标签）
- [x] Illumio 四维标签作为推荐结构
- [x] 基础自动打标（从镜像/端口推断角色）
- [x] 带标签选择器的组定义
- [x] 选择器操作符：`=`, `!=`, `in`, `not-in`, `exists`, `not-exists`
- [x] 组成员解析引擎
- [x] 组 CRUD API
- [x] PolicyRule 数据模型（组到组规则）
- [x] PolicyRule CRUD API
- [x] 策略编译引擎（组 → IP 展开）
- [x] 溯源跟踪（编译规则 → 源策略）
- [x] 数据库模式扩展（workloads, groups, policy_rules, policy_compilation 表）
- [x] 与现有 PolicyManager 集成（编译规则 → eBPF maps）

**预计工作量**：3-4 天

### 范围外（未来提案）
- [ ] 容器运行时自动发现（Docker API、containerd API）
- [ ] Kubernetes 集成（Pod 发现、标签同步、命名空间映射）
- [ ] 高级自动打标（K8s 标签、注解、服务网格集成）
- [ ] 学习模式（观察流量，建议组/策略）
- [ ] 策略推荐引擎
- [ ] 带标签的流量可视化
- [ ] 分布式存储（etcd/Consul 用于多代理部署）
- [ ] 标签/组管理的 Web UI
- [ ] 高级选择器操作符（regex、contains、prefix）
- [ ] 策略模拟（假设分析）

## 设计原则

### 1. 向后兼容
- 保留现有 `policies` 表和 Policy 结构
- 基于 IP 的策略继续正常工作
- 新的基于标签的策略是附加的，不是替换

### 2. 控制平面/数据平面分离
- **控制平面**：标签、组、高级策略（所有复杂性在这里）
- **数据平面**：仅编译的 IP 规则（保持简单和快速）
- **好处**：数据平面性能不变（<10μs 包处理）

### 3. 存储抽象
- 为所有实体定义存储接口
- SQLite 作为 MVP 实现
- 未来：在不改变业务逻辑的情况下切换到 etcd/Consul

### 4. 渐进式采用
- 用户可以从基于 IP 的策略开始
- 逐步添加标签和组
- 混合使用基于 IP 和基于标签的策略
- 无需强制迁移

### 5. 简单优先
- 从相等操作符（`=`, `!=`, `in`）开始
- 仅在用户反馈验证后添加复杂性（regex、contains）
- 避免过早优化

## 非目标
- 本提案不打算：
  - 替换现有的基于 IP 的策略系统
  - 要求 Kubernetes 或容器运行时
  - 实现策略学习或自动生成（单独提案）
  - 构建 UI 组件（API 优先方法）
  - 支持多租户（MVP 为单一命名空间）

## 成功标准
1. **功能性**：
   - 用户可以通过 API 创建带标签的工作负载
   - 用户可以用选择器定义组
   - 用户可以创建组到组的策略规则
   - 系统编译规则为基于 IP 的规则
   - 编译规则被 eBPF 数据平面正确执行
   - 溯源跟踪有效（可以追溯编译规则到源头）

2. **性能**：
   - 对于 100 个工作负载 + 10 个组，组成员解析 < 100ms
   - 对于 10×10 组展开，策略编译 < 500ms
   - API 响应时间 < 50ms（CRUD 操作）
   - 数据平面延迟无影响（<10μs 不变）

3. **质量**：
   - 新模块的测试覆盖率 > 90%
   - 带示例的 API 文档
   - E2E 测试场景：创建工作负载 → 组 → 策略 → 验证 eBPF 规则

## 依赖项
- 现有：PolicyManager, SQLiteStorage, eBPF maps
- 新增：无外部依赖（纯 Go + SQLite）

## 风险和缓解

| 风险 | 影响 | 缓解措施 |
|------|--------|------------|
| 策略爆炸（N×M 规则） | 高 | 组大小 >100 时警告用户，实现分页 |
| 编译性能 | 中 | 缓存组成员，增量重新编译 |
| 存储可扩展性（SQLite） | 低 | 抽象接口，计划 etcd 迁移（第 2 阶段） |
| API 复杂性 | 低 | 从简单开始，基于反馈迭代 |

## 考虑的替代方案

### 替代方案 1：基于 IP 集的组
- 思路：将组存储为静态 IP 集，无标签
- **拒绝理由**：不解决动态 IP 问题，仍需手动管理

### 替代方案 2：在 eBPF 中实现标签
- 思路：在 eBPF maps 中存储标签，在内核中匹配
- **拒绝理由**：增加数据平面复杂性，性能影响，map 空间有限

### 替代方案 3：外部策略引擎（OPA）
- 思路：使用 Open Policy Agent 进行策略评估
- **拒绝理由**：增加依赖、延迟、复杂性。对于网络策略，标签更简单。

### 选择的方案：控制平面标签 + 编译规则
- **优点**：数据平面保持简单和快速
- **优点**：灵活的标签模型，易于扩展
- **优点**：无外部依赖
- **缺点**：编译开销（对于控制平面可接受）

## 参考资料
- MVP 计划：`/docs/microsegmentation-mvp-implementation-plan.md`（第 4 周：标签系统）
- NeuVector 分析：`/docs/neuvector-analysis/neuvector-agent-dp-policy-flow.md`
- Illumio 模型：四维标签（Role/App/Env/Location）
- 研究报告：Agent 研究输出（全面的标签系统分析）

## 后续步骤
1. 审查并批准此提案
2. 审查详细的 design.md
3. 按照 tasks.md 实施任务
4. 使用 `openspec validate add-label-based-policy --strict` 验证
5. 实施完成后归档提案
