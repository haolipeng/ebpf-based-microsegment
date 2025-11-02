# 实施任务：基于标签的策略管理

本文档提供了实施基于标签的策略系统的有序检查清单。任务按天组织，并附有明确的验收标准。

## 实施前准备

- [ ] 查阅 proposal.md 和 design.md 以完全理解需求
- [ ] 创建功能分支：`git checkout -b feature/label-based-policy`
- [ ] 查阅现有代码库结构（`src/agent/pkg/`）

## 第1天：数据模型和存储层

### 任务 1.1：工作负载数据模型
- [ ] 创建 `src/agent/pkg/workload/types.go`
  - [ ] 定义 `Workload` 结构体，包含所有字段（ID、Name、IPs、MACs、Labels 等）
  - [ ] 定义 `WorkloadState` 枚举（running、stopped、paused）
  - [ ] 添加 JSON 标签用于 API 序列化
  - [ ] 添加 DB 标签用于 SQLite 映射
- [ ] 在 `src/agent/pkg/workload/types_test.go` 中创建单元测试
  - [ ] 测试使用有效数据创建工作负载
  - [ ] 测试 JSON 序列化/反序列化
  - [ ] 测试标签操作（添加、更新、删除）

**验收标准**：Workload 结构体编译通过，测试通过，JSON 序列化正常工作

### 任务 1.2：工作负载存储
- [ ] 创建 `src/agent/pkg/workload/storage.go`
  - [ ] 定义 `WorkloadStorage` 接口（Create、Get、Update、Delete、List）
  - [ ] 实现 `SQLiteWorkloadStorage` 结构体
- [ ] 创建数据库模式迁移
  - [ ] 将 `workloads` 表模式添加到迁移文件
  - [ ] 在 `id`、`host_id`、`state` 上创建索引
  - [ ] 测试迁移正常应用
- [ ] 实现 CRUD 操作
  - [ ] `CreateWorkload(w *Workload) error`
  - [ ] `GetWorkload(id string) (*Workload, error)`
  - [ ] `UpdateWorkload(w *Workload) error`
  - [ ] `DeleteWorkload(id string) error`
  - [ ] `ListWorkloads() ([]*Workload, error)`
  - [ ] `ListWorkloadsByLabel(key, value string) ([]*Workload, error)`
- [ ] 在 `src/agent/pkg/workload/storage_test.go` 中创建集成测试
  - [ ] 测试完整的 CRUD 生命周期
  - [ ] 测试并发访问
  - [ ] 测试标签过滤

**验收标准**：所有 CRUD 操作正常工作，数据持久化到 SQLite，测试通过

### 任务 1.3：标签系统
- [ ] 创建 `src/agent/pkg/labels/types.go`
  - [ ] 定义 `LabelDimension` 常量（Role、App、Env、Location）
  - [ ] 定义标签验证函数
  - [ ] 定义保留标签前缀
- [ ] 创建 `src/agent/pkg/labels/validator.go`
  - [ ] 实现 `ValidateLabelKey(key string) error`
  - [ ] 实现 `ValidateLabelValue(value string) error`
  - [ ] 拒绝无效字符，强制执行长度限制
- [ ] 在 `src/agent/pkg/labels/validator_test.go` 中创建单元测试
  - [ ] 测试有效的标签格式
  - [ ] 测试无效格式（特殊字符、过长等）
  - [ ] 测试 Illumio 维度验证

**验收标准**：标签验证正常工作，所有边界情况都被测试覆盖

### 任务 1.4：分组数据模型和存储
- [ ] 创建 `src/agent/pkg/groups/types.go`
  - [ ] 定义 `Group` 结构体
  - [ ] 定义 `LabelSelector` 结构体
  - [ ] 定义 `SelectorOperator` 枚举（=、!=、in、not-in、exists、not-exists）
- [ ] 创建 `src/agent/pkg/groups/storage.go`
  - [ ] 定义 `GroupStorage` 接口
  - [ ] 实现 `SQLiteGroupStorage`
- [ ] 创建数据库模式迁移
  - [ ] 添加 `groups` 表模式
  - [ ] 在 `name` 上创建索引
- [ ] 实现 CRUD 操作
  - [ ] `CreateGroup(g *Group) error`
  - [ ] `GetGroup(name string) (*Group, error)`
  - [ ] `UpdateGroup(g *Group) error`
  - [ ] `DeleteGroup(name string) error`
  - [ ] `ListGroups() ([]*Group, error)`
- [ ] 在 `src/agent/pkg/groups/storage_test.go` 中创建测试

**验收标准**：分组存储在数据库中，选择器正确序列化，测试通过

## 第2天：分组成员解析

### 任务 2.1：选择器评估引擎
- [ ] 创建 `src/agent/pkg/groups/selector.go`
  - [ ] 实现 `EvaluateSelector(wl *Workload, sel LabelSelector) bool`
    - [ ] 处理 `=` 运算符
    - [ ] 处理 `!=` 运算符
    - [ ] 处理 `in` 运算符
    - [ ] 处理 `not-in` 运算符
    - [ ] 处理 `exists` 运算符
    - [ ] 处理 `not-exists` 运算符
- [ ] 在 `src/agent/pkg/groups/selector_test.go` 中创建全面的测试
  - [ ] 测试每个运算符的有效/无效输入
  - [ ] 测试边界情况（空值、nil 标签等）
  - [ ] 测试运算符组合

**验收标准**：所有6个运算符正常工作，100% 测试覆盖率

### 任务 2.2：分组成员解析
- [ ] 创建 `src/agent/pkg/groups/manager.go`
  - [ ] 定义 `GroupManager` 结构体
  - [ ] 实现 `NewGroupManager(storage GroupStorage, workloadMgr *workload.Manager)`
  - [ ] 实现 `ResolveGroupMembers(groupName string) ([]*Workload, error)`
  - [ ] 实现 `IsWorkloadSelected(wl *Workload, selectors []LabelSelector) bool`
- [ ] 添加成员缓存（可选优化）
  - [ ] 缓存分组成员解析结果
  - [ ] 在工作负载/分组更新时使缓存失效
- [ ] 在 `src/agent/pkg/groups/manager_test.go` 中创建测试
  - [ ] 测试0个工作负载的解析
  - [ ] 测试1个匹配工作负载的解析
  - [ ] 测试N个匹配工作负载的解析
  - [ ] 测试复杂选择器组合
  - [ ] 测试性能：100个工作负载 + 10个分组

**验收标准**：分组解析正常工作，100个工作负载的性能 <100ms

### 任务 2.3：自动标记引擎
- [ ] 创建 `src/agent/pkg/labels/autotagger.go`
  - [ ] 定义 `AutoTagger` 结构体
  - [ ] 实现 `InferRoleFromImage(image string) string`
    - [ ] nginx → web
    - [ ] mysql/postgres → db
    - [ ] redis/memcached → cache
    - [ ] rabbitmq/kafka → mq
  - [ ] 实现 `InferRoleFromPorts(ports []uint16) string`
    - [ ] 80/443/8080 → web
    - [ ] 3306/5432/27017 → db
    - [ ] 6379 → cache
  - [ ] 实现 `AutoTagWorkload(wl *Workload) map[string]string`
- [ ] 在 `src/agent/pkg/labels/autotagger_test.go` 中创建测试
  - [ ] 测试基于镜像的推断
  - [ ] 测试基于端口的推断
  - [ ] 测试组合推断
  - [ ] 测试无推断（自定义工作负载）

**验收标准**：对于常见镜像/端口的自动标记正常工作，测试通过

## 第3天：策略编译引擎

### 任务 3.1：策略规则数据模型和存储
- [ ] 创建 `src/agent/pkg/policy/rule.go`
  - [ ] 定义 `PolicyRule` 结构体（name、from_group、to_group、ports、action 等）
  - [ ] 定义 `PortRange` 结构体（protocol、start、end）
- [ ] 扩展 `src/agent/pkg/policy/storage.go`
  - [ ] 添加 `PolicyRuleStorage` 接口
  - [ ] 实现 SQLite 存储方法
- [ ] 创建数据库迁移
  - [ ] 添加 `policy_rules` 表
  - [ ] 添加指向 `groups` 表的外键
  - [ ] 在 `id`、`enabled` 上创建索引
- [ ] 实现 CRUD 操作
  - [ ] `CreatePolicyRule(r *PolicyRule) error`
  - [ ] `GetPolicyRule(id uint32) (*PolicyRule, error)`
  - [ ] `UpdatePolicyRule(r *PolicyRule) error`
  - [ ] `DeletePolicyRule(id uint32) error`
  - [ ] `ListPolicyRules() ([]*PolicyRule, error)`
- [ ] 在 `src/agent/pkg/policy/rule_test.go` 中创建测试

**验收标准**：策略规则存储在数据库中，端口范围正确序列化，测试通过

### 任务 3.2：编译后的策略和溯源
- [ ] 创建 `src/agent/pkg/policy/compiled.go`
  - [ ] 定义 `CompiledPolicy` 结构体（扩展现有的 Policy，添加溯源信息）
  - [ ] 添加字段：`SourcePolicyID`、`FromGroup`、`ToGroup`、`FromWorkloadID`、`ToWorkloadID`
- [ ] 创建数据库迁移
  - [ ] 添加 `policy_compilation` 表
  - [ ] 添加指向 `policies` 和 `policy_rules` 的外键
  - [ ] 为溯源查询添加索引
- [ ] 实现溯源跟踪
  - [ ] `GetCompiledPolicySource(compiledID uint32) (*PolicyRule, error)`
  - [ ] `GetCompiledPoliciesForRule(ruleID uint32) ([]*CompiledPolicy, error)`

**验收标准**：溯源跟踪正常工作，可以将编译后的规则追溯到源规则

### 任务 3.3：策略编译器
- [ ] 创建 `src/agent/pkg/policy/compiler.go`
  - [ ] 定义 `PolicyCompiler` 结构体
  - [ ] 实现 `NewPolicyCompiler(storage Storage, groupMgr *groups.GroupManager)`
  - [ ] 实现 `CompilePolicyRule(ruleID uint32) ([]*CompiledPolicy, error)`
    - [ ] 解析 from_group 成员
    - [ ] 解析 to_group 成员
    - [ ] 生成笛卡尔积（N×M IP 规则）
    - [ ] 分配唯一规则 ID
    - [ ] 存储溯源信息
  - [ ] 实现 `CompileAllPolicies() error`
  - [ ] 实现 `InvalidateCompiledPolicies(ruleID uint32) error`
- [ ] 为大规模扩展添加警告
  - [ ] 如果 N×M > 1000 个编译规则则发出警告
  - [ ] 记录分组大小
- [ ] 在 `src/agent/pkg/policy/compiler_test.go` 中创建测试
  - [ ] 测试 1×1 分组扩展
  - [ ] 测试 N×M 扩展（10个web × 2个db = 20条规则）
  - [ ] 测试多个端口范围
  - [ ] 测试溯源跟踪
  - [ ] 测试性能：10×10 扩展 <500ms

**验收标准**：编译正常工作，溯源被跟踪，达到性能目标

### 任务 3.4：与 PolicyManager 集成
- [ ] 扩展 `src/agent/pkg/policy/manager.go`
  - [ ] 添加 `compiler *PolicyCompiler` 字段
  - [ ] 实现 `AddPolicyRule(r *PolicyRule) error`
    - [ ] 存储规则
    - [ ] 编译为 IP 规则
    - [ ] 为每个编译后的规则调用现有的 `AddPolicy()`
  - [ ] 实现 `DeletePolicyRule(id uint32) error`
    - [ ] 获取编译后的策略
    - [ ] 删除每个编译后的策略
    - [ ] 删除源规则
  - [ ] 实现 `UpdatePolicyRule(r *PolicyRule) error`
    - [ ] 删除旧的编译策略
    - [ ] 重新编译
    - [ ] 添加新的编译策略
- [ ] 创建集成测试
  - [ ] 测试完整生命周期：添加规则 → 验证 eBPF 映射 → 删除规则
  - [ ] 测试并发策略更新

**验收标准**：基于标签的规则正确填充 eBPF 映射，测试通过

## 第4天：API 端点和集成

### 任务 4.1：工作负载 API 处理器
- [ ] 创建 `src/agent/pkg/api/handlers/workload.go`
  - [ ] 定义 `WorkloadHandler` 结构体
  - [ ] 实现 `CreateWorkload(c *gin.Context)`
  - [ ] 实现 `ListWorkloads(c *gin.Context)`
  - [ ] 实现 `GetWorkload(c *gin.Context)`
  - [ ] 实现 `UpdateWorkload(c *gin.Context)`
  - [ ] 实现 `DeleteWorkload(c *gin.Context)`
  - [ ] 实现 `AutoTagWorkload(c *gin.Context)`（POST /workloads/:id/autotag）
- [ ] 在 `src/agent/pkg/api/models/workload.go` 中创建请求/响应模型
- [ ] 将路由添加到 `src/agent/cmd/agent/main.go`
  - [ ] POST /api/v1/workloads
  - [ ] GET /api/v1/workloads
  - [ ] GET /api/v1/workloads/:id
  - [ ] PUT /api/v1/workloads/:id
  - [ ] DELETE /api/v1/workloads/:id
  - [ ] POST /api/v1/workloads/:id/autotag
- [ ] 在 `src/agent/pkg/api/handlers/workload_test.go` 中创建处理器测试

**验收标准**：所有工作负载端点正常工作，返回正确的状态码

### 任务 4.2：分组 API 处理器
- [ ] 创建 `src/agent/pkg/api/handlers/group.go`
  - [ ] 定义 `GroupHandler` 结构体
  - [ ] 实现 `CreateGroup(c *gin.Context)`
  - [ ] 实现 `ListGroups(c *gin.Context)`
  - [ ] 实现 `GetGroup(c *gin.Context)`
  - [ ] 实现 `UpdateGroup(c *gin.Context)`
  - [ ] 实现 `DeleteGroup(c *gin.Context)`
  - [ ] 实现 `GetGroupMembers(c *gin.Context)`（GET /groups/:name/members）
- [ ] 在 `src/agent/pkg/api/models/group.go` 中创建模型
- [ ] 将路由添加到 main.go
  - [ ] POST /api/v1/groups
  - [ ] GET /api/v1/groups
  - [ ] GET /api/v1/groups/:name
  - [ ] PUT /api/v1/groups/:name
  - [ ] DELETE /api/v1/groups/:name
  - [ ] GET /api/v1/groups/:name/members
- [ ] 创建处理器测试

**验收标准**：所有分组端点正常工作，通过 API 的成员解析正常工作

### 任务 4.3：策略规则 API 处理器
- [ ] 创建 `src/agent/pkg/api/handlers/policy_rule.go`
  - [ ] 定义 `PolicyRuleHandler` 结构体
  - [ ] 实现 `CreatePolicyRule(c *gin.Context)`
  - [ ] 实现 `ListPolicyRules(c *gin.Context)`
  - [ ] 实现 `GetPolicyRule(c *gin.Context)`
  - [ ] 实现 `UpdatePolicyRule(c *gin.Context)`
  - [ ] 实现 `DeletePolicyRule(c *gin.Context)`
  - [ ] 实现 `CompilePolicyRule(c *gin.Context)`（POST /policy-rules/:id/compile）
  - [ ] 实现 `GetCompiledPolicies(c *gin.Context)`（GET /policy-rules/:id/compiled）
  - [ ] 实现 `GetPolicySource(c *gin.Context)`（GET /compiled-policies/:id/source）
- [ ] 在 `src/agent/pkg/api/models/policy_rule.go` 中创建模型
- [ ] 将路由添加到 main.go
  - [ ] POST /api/v1/policy-rules
  - [ ] GET /api/v1/policy-rules
  - [ ] GET /api/v1/policy-rules/:id
  - [ ] PUT /api/v1/policy-rules/:id
  - [ ] DELETE /api/v1/policy-rules/:id
  - [ ] POST /api/v1/policy-rules/:id/compile
  - [ ] GET /api/v1/policy-rules/:id/compiled
  - [ ] GET /api/v1/compiled-policies/:id/source
- [ ] 创建处理器测试

**验收标准**：所有策略规则端点正常工作，通过 API 触发编译

### 任务 4.4：端到端集成测试
- [ ] 创建 `src/agent/test/integration/label_policy_test.go`
  - [ ] 测试场景1：创建工作负载 → 创建分组 → 验证成员
  - [ ] 测试场景2：创建策略规则 → 验证数据库中的编译策略
  - [ ] 测试场景3：创建策略规则 → 验证 eBPF 映射已更新
  - [ ] 测试场景4：更新工作负载标签 → 验证分组成员变化
  - [ ] 测试场景5：删除策略规则 → 验证编译策略被删除
  - [ ] 测试场景6：自动标记工作负载 → 验证标签已分配
  - [ ] 测试场景7：复杂选择器（AND/OR）→ 验证正确的成员
- [ ] 在 `src/agent/test/benchmark/label_benchmark_test.go` 中创建性能基准测试
  - [ ] 基准测试分组成员解析（100个工作负载，10个分组）
  - [ ] 基准测试策略编译（10×10 扩展）
  - [ ] 验证：解析 <100ms，编译 <500ms

**验收标准**：所有端到端场景通过，达到性能目标

## 第5天：文档和验证

### 任务 5.1：API 文档
- [ ] 更新 OpenAPI 规范（如果存在）或创建新规范
  - [ ] 记录所有新端点
  - [ ] 包含请求/响应模式
  - [ ] 添加示例负载
- [ ] 在 `docs/label-based-policy-api-examples.md` 中创建 API 使用示例
  - [ ] 示例1：创建带标签的工作负载
  - [ ] 示例2：定义带选择器的分组
  - [ ] 示例3：创建策略规则
  - [ ] 示例4：查询分组成员
  - [ ] 示例5：将编译策略追溯到源

**验收标准**：文档完整，示例可以直接复制粘贴

### 任务 5.2：代码质量
- [ ] 运行 linter：`golangci-lint run ./src/agent/pkg/...`
  - [ ] 修复所有错误
  - [ ] 修复关键警告
- [ ] 检查测试覆盖率：`go test -cover ./src/agent/pkg/...`
  - [ ] 确保新包的覆盖率 >90%
  - [ ] 为未覆盖的分支添加测试
- [ ] 运行所有测试：`go test -v ./src/agent/...`
  - [ ] 确保所有测试通过
  - [ ] 修复任何不稳定的测试

**验收标准**：没有 lint 错误，覆盖率 >90%，所有测试通过

### 任务 5.3：数据库迁移
- [ ] 创建合并的迁移文件
  - [ ] workloads 表模式
  - [ ] groups 表模式
  - [ ] policy_rules 表模式
  - [ ] policy_compilation 表模式
  - [ ] 所有索引和外键
- [ ] 在干净的数据库上测试迁移
- [ ] 测试迁移回滚（如果支持）
- [ ] 在 `docs/label-system-migration.md` 中记录迁移过程

**验收标准**：迁移正常应用，回滚正常工作

### 任务 5.4：OpenSpec 验证
- [ ] 运行 `openspec validate add-label-based-policy --strict`
  - [ ] 修复任何验证错误
  - [ ] 确保所有规范都有场景
  - [ ] 确保所有任务标记为完成
- [ ] 更新提案状态
  - [ ] 将所有范围内的项目标记为已实现
  - [ ] 记录任何范围变更
- [ ] 生成最终报告
  - [ ] 实施摘要
  - [ ] 测试结果
  - [ ] 性能基准
  - [ ] 已知限制

**验收标准**：OpenSpec 验证通过，提案准备归档

## 实施后

### 任务 6.1：代码审查准备
- [ ] 创建带详细描述的拉取请求
- [ ] 包含前后对比示例
- [ ] 引用提案和设计文档
- [ ] 添加 API 响应的截图（如适用）

### 任务 6.2：部署准备
- [ ] 更新部署文档
- [ ] 为现有用户创建迁移指南
- [ ] 准备发布说明
- [ ] 标记发布：`git tag v1.1.0-label-system`

## 成功标准总结

**功能性**：
- ✅ 用户可以通过 API 创建带标签的工作负载
- ✅ 用户可以定义带标签选择器的分组
- ✅ 用户可以创建分组到分组的策略规则
- ✅ 系统将规则编译为基于 IP 的策略
- ✅ 编译后的规则由 eBPF 数据平面强制执行
- ✅ 溯源跟踪正常工作（追踪编译后规则 → 源规则）

**性能**：
- ✅ 100个工作负载 + 10个分组的分组成员解析 <100ms
- ✅ 10×10 扩展的策略编译 <500ms
- ✅ API 响应时间 <50ms（CRUD 操作）
- ✅ 无数据平面延迟影响

**质量**：
- ✅ 新模块的测试覆盖率 >90%
- ✅ 带示例的 API 文档
- ✅ 端到端测试场景通过
- ✅ OpenSpec 验证通过

## 预估时间线

- **第1天**：8小时（数据模型、存储层）
- **第2天**：8小时（分组解析、自动标记）
- **第3天**：8小时（策略编译、集成）
- **第4天**：8小时（API 端点、端到端测试）
- **第5天**：4小时（文档、验证）

**总计**：36小时（全力专注约4.5天）

## 注意事项

- 任务按依赖关系排序（必须在第2天之前完成第1天，依此类推）
- 每个任务都有明确的验收标准
- 测试与实现同步编写（TDD 方法）
- 持续验证性能目标
- OpenSpec 验证确保完整性
