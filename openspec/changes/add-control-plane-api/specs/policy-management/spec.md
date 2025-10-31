# Capability: 策略管理

## ADDED Requirements

### Requirement: 线程安全策略访问
策略管理系统必须(SHALL)为并发操作提供线程安全访问。

#### Scenario: 并发策略读取
- **WHEN** 多个 goroutine 同时读取策略
- **THEN** 所有读取必须(MUST)成功完成
- **AND** 所有读取必须(MUST)返回一致的数据

#### Scenario: 并发策略写入
- **WHEN** 多个 goroutine 同时更新策略
- **THEN** 所有更新必须(MUST)原子地应用
- **AND** 不得(MUST NOT)丢失任何更新

#### Scenario: 读写同步
- **WHEN** 在更新发生时正在读取策略
- **THEN** 读取不得(MUST NOT)看到部分更新
- **AND** 读取不得(MUST NOT)不必要地阻塞

---

### Requirement: 策略生命周期管理
策略管理器必须(SHALL)跟踪策略的完整生命周期。

#### Scenario: 策略创建时间戳
- **WHEN** 创建新策略
- **THEN** 必须(MUST)记录创建时间戳
- **AND** 时间戳必须(MUST)可通过 API 访问

#### Scenario: 策略更新时间戳
- **WHEN** 更新策略
- **THEN** 必须(MUST)记录最后更新时间戳
- **AND** 保留原始创建时间戳

---

### Requirement: 策略 ID 管理
策略管理器必须(SHALL)分配和管理唯一的策略标识符。

#### Scenario: 自动生成策略 ID
- **WHEN** 创建策略时未指定显式 ID
- **THEN** 系统必须(MUST)生成唯一 ID
- **AND** 必须(MUST)将 ID 返回给客户端

#### Scenario: 显式策略 ID
- **WHEN** 使用显式 ID 创建策略
- **THEN** 如果尚未使用，则使用提供的 ID
- **WHEN** ID 已被使用
- **THEN** 返回冲突错误

#### Scenario: ID 持久化
- **THEN** 策略 ID 必须(MUST)在 API 重启后保持稳定
- **AND** ID 必须(MUST)在系统内唯一

---

### Requirement: 策略验证
策略管理器必须(SHALL)在应用之前验证所有策略规则。

#### Scenario: 必需字段验证
- **WHEN** 策略缺少必需字段
- **THEN** 返回列出缺少字段的验证错误

#### Scenario: IP 地址格式验证
- **WHEN** 策略包含无效的 IP 地址格式
- **THEN** 返回验证错误

#### Scenario: CIDR 表示法验证
- **WHEN** 策略使用 CIDR 表示法（例如，"10.0.0.0/24"）
- **THEN** 验证网络地址和前缀长度
- **WHEN** CIDR 无效
- **THEN** 返回验证错误

#### Scenario: 端口范围验证
- **WHEN** 策略端口超出 0-65535 范围
- **THEN** 返回验证错误

#### Scenario: 协议验证
- **WHEN** 策略协议不是"tcp"、"udp"、"icmp"或"any"
- **THEN** 返回验证错误

#### Scenario: 操作验证
- **WHEN** 策略操作不是"allow"、"deny"或"log"
- **THEN** 返回验证错误

---

### Requirement: 策略查询和过滤
策略管理器必须(SHALL)支持查询和过滤策略。

#### Scenario: 列出所有策略
- **WHEN** 查询所有策略
- **THEN** 返回活跃策略的完整列表
- **AND** 包括所有策略元数据

#### Scenario: 按源 IP 过滤
- **WHEN** 使用源 IP 过滤器查询策略
- **THEN** 仅返回匹配的策略

#### Scenario: 按操作过滤
- **WHEN** 使用操作过滤器查询策略
- **THEN** 仅返回具有指定操作的策略

#### Scenario: 按优先级排序
- **WHEN** 查询策略
- **THEN** 允许按优先级排序（升序/降序）

