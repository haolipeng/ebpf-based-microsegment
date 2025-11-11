# Spec: xdp-dataplane

## ADDED Requirements

### Requirement: XDP 程序结构和入口
XDP eBPF 程序 SHALL 定义正确的程序入口,接受 `xdp_md` 上下文,并返回有效的 XDP 动作码。

#### Scenario: 正常数据包处理
- **GIVEN** 收到合法的 IPv4 TCP 数据包
- **WHEN** XDP 程序 `xdp_microsegment_prog()` 被调用
- **THEN** 程序成功解析数据包
- **AND** 执行策略匹配
- **AND** 返回 `XDP_PASS` 或 `XDP_DROP`

#### Scenario: 无效数据包安全处理
- **GIVEN** 收到格式无效的数据包 (头部不完整)
- **WHEN** XDP 程序解析数据包失败
- **THEN** 程序返回 `XDP_PASS` (安全放行)
- **AND** 不会崩溃或越界访问

---

### Requirement: 数据包解析和边界检查
XDP 程序 SHALL 正确解析以太网、IP 和传输层协议头部,并严格验证所有内存访问的边界。

#### Scenario: 解析 IPv4 + TCP 数据包
- **GIVEN** 数据包包含完整的 Ethernet + IPv4 + TCP 头部
- **WHEN** 调用 `parse_packet(ctx, &key)`
- **THEN** 成功解析所有头部
- **AND** 提取五元组: 协议=TCP, 源IP, 目的IP, 源端口, 目的端口
- **AND** 生成标准化的 `flow_key` 结构
- **AND** 返回 0 (成功)

#### Scenario: 解析 IPv6 + UDP 数据包
- **GIVEN** 数据包包含完整的 Ethernet + IPv6 + UDP 头部
- **WHEN** 调用 `parse_packet(ctx, &key)`
- **THEN** 成功解析所有头部
- **AND** 提取五元组 (IPv6 地址)
- **AND** 返回 0 (成功)

#### Scenario: 边界检查阻止越界访问
- **GIVEN** 数据包头部不完整 (data_end - data < 完整头部大小)
- **WHEN** 尝试访问头部字段
- **THEN** 边界检查失败
- **AND** 解析函数返回 -1
- **AND** 程序返回 `XDP_PASS`

#### Scenario: 不支持的协议安全处理
- **GIVEN** 数据包是非 IP 协议 (例如 ARP)
- **WHEN** 解析以太网头部后检测到非 IP 协议
- **THEN** 解析函数返回 -1
- **AND** 程序返回 `XDP_PASS`

---

### Requirement: Map 定义和 Pinning
XDP 程序 SHALL 定义所有必需的 eBPF Map,并通过 Pinning 机制与 TC 程序共享数据。

#### Scenario: Map Pinning 到文件系统
- **GIVEN** XDP 程序定义了 `session_map`,标记为 `LIBBPF_PIN_BY_NAME`
- **WHEN** 加载 XDP 程序
- **THEN** Map 被固定到 `/sys/fs/bpf/tc-global/session_map`
- **AND** TC 程序和用户态程序都能访问此 Map

#### Scenario: 共享会话数据
- **GIVEN** TC 程序创建了一个会话记录
- **AND** XDP 程序和 TC 程序都使用 pinned `session_map`
- **WHEN** XDP 程序查询相同的流键
- **THEN** 查询到 TC 创建的会话记录
- **AND** 会话数据完全一致

#### Scenario: 所有必需 Map 都已定义
- **WHEN** 检查 XDP 程序的 Map 定义
- **THEN** 存在以下 Map:
  - `session_map` (LRU_HASH, 1048576 entries)
  - `policy_map` (HASH, 65536 entries)
  - `wildcard_policy_map` (HASH, 4096 entries)
  - `stats_map` (PERCPU_HASH, 65536 entries)
- **AND** 所有 Map 都标记为 `LIBBPF_PIN_BY_NAME`

---

### Requirement: 策略匹配逻辑复用
XDP 程序 SHALL 通过 `#include` 头文件复用 TC 程序的策略匹配逻辑,确保行为完全一致。

#### Scenario: 精确策略匹配
- **GIVEN** `policy_map` 中存在精确匹配的规则: src=10.0.1.10, dst=10.0.2.20 → DROP
- **WHEN** XDP 程序调用 `lookup_policy_action(&key, &rule_id)`
- **THEN** 返回 `POLICY_ACTION_DROP`
- **AND** rule_id 被设置为对应的规则 ID
- **AND** 结果与 TC 程序完全一致

#### Scenario: 通配符策略匹配
- **GIVEN** `policy_map` 中无精确匹配
- **AND** `wildcard_policy_map` 中存在通配符规则: dst=10.0.2.0/24 → DROP
- **WHEN** 调用 `lookup_policy_action(&key, &rule_id)`
- **THEN** 匹配通配符规则,返回 `POLICY_ACTION_DROP`
- **AND** 结果与 TC 程序完全一致

#### Scenario: 默认允许策略
- **GIVEN** 没有匹配的精确规则或通配符规则
- **WHEN** 调用 `lookup_policy_action(&key, &rule_id)`
- **THEN** 返回 `POLICY_ACTION_ALLOW`
- **AND** rule_id = 0
- **AND** 结果与 TC 程序完全一致

---

### Requirement: 会话跟踪和缓存
XDP 程序 SHALL 实现会话跟踪功能,复用已有的策略决策,减少重复的策略匹配开销。

#### Scenario: 会话命中
- **GIVEN** `session_map` 中存在会话记录 (action=ALLOW, rule_id=123)
- **WHEN** 同一流的后续数据包到达
- **THEN** 直接从会话中获取决策 action=ALLOW
- **AND** 不执行策略匹配
- **AND** 更新会话的 last_seen 时间戳

#### Scenario: 新会话创建
- **GIVEN** `session_map` 中不存在该流的会话记录
- **WHEN** 执行策略匹配,结果为 DROP
- **THEN** 创建新会话记录: action=DROP, rule_id=456, last_seen=当前时间
- **AND** 插入到 `session_map`
- **AND** 后续数据包可以直接使用此会话

#### Scenario: 会话 LRU 淘汰
- **GIVEN** `session_map` 已满 (1048576 entries)
- **WHEN** 插入新会话
- **THEN** 最久未使用的会话被自动淘汰
- **AND** 新会话成功插入

---

### Requirement: 统计数据收集
XDP 程序 SHALL 收集每个策略规则的数据包和字节统计,并使用 Per-CPU Map 避免锁竞争。

#### Scenario: 更新规则统计
- **GIVEN** 数据包匹配规则 ID=123
- **WHEN** 执行动作后更新统计
- **THEN** `stats_map[123].packets` 增加 1
- **AND** `stats_map[123].bytes` 增加数据包大小 (L3 层)

#### Scenario: Per-CPU 统计无竞争
- **GIVEN** 使用 `BPF_MAP_TYPE_PERCPU_HASH`
- **WHEN** 多个 CPU 并发更新同一规则的统计
- **THEN** 每个 CPU 维护独立的计数器
- **AND** 用户态读取时自动聚合所有 CPU 的计数
- **AND** 无需锁同步

#### Scenario: 统计数据与 TC 一致性
- **GIVEN** XDP 和 TC 都处理流量并更新统计
- **WHEN** 用户态读取统计数据
- **THEN** 统计数据累加正确
- **AND** 反映两个数据平面的总流量

---

### Requirement: 动作执行
XDP 程序 SHALL 根据策略匹配结果正确执行 PASS 或 DROP 动作。

#### Scenario: ALLOW 策略返回 XDP_PASS
- **GIVEN** 策略匹配结果为 `POLICY_ACTION_ALLOW`
- **WHEN** 执行动作
- **THEN** XDP 程序返回 `XDP_PASS`
- **AND** 数据包继续网络栈处理

#### Scenario: DROP 策略返回 XDP_DROP
- **GIVEN** 策略匹配结果为 `POLICY_ACTION_DROP`
- **WHEN** 执行动作
- **THEN** XDP 程序返回 `XDP_DROP`
- **AND** 数据包在网络栈最早位置被丢弃
- **AND** 不消耗后续网络栈处理资源

#### Scenario: 默认策略为 ALLOW
- **GIVEN** 解析失败或无匹配策略
- **WHEN** 执行动作
- **THEN** 默认返回 `XDP_PASS` (安全地放行)

---

### Requirement: bpf2go 代码生成
XDP 程序 SHALL 配置 bpf2go 工具生成 Go 语言绑定,以便用户态程序加载和操作。

#### Scenario: 生成 XDP Go 绑定
- **GIVEN** `generate.go` 中配置了 XDP bpf2go 指令
- **WHEN** 运行 `go generate`
- **THEN** 生成以下文件:
  - `xdp_bpfeb.go` (大端架构)
  - `xdp_bpfel.go` (小端架构)
  - `xdp_bpfeb.o` (大端 eBPF 对象)
  - `xdp_bpfel.o` (小端 eBPF 对象)
- **AND** 生成的 Go 代码能成功编译

#### Scenario: 导出必要的类型定义
- **GIVEN** bpf2go 配置了 `-type flow_key -type session_value -type policy_value`
- **WHEN** 生成 Go 绑定
- **THEN** Go 代码中包含对应的结构体定义
- **AND** 用户态程序能使用这些类型操作 Map

---

### Requirement: 错误处理和健壮性
XDP 程序 SHALL 正确处理所有错误情况,不能因异常输入而崩溃或产生未定义行为。

#### Scenario: Map 查询失败的默认行为
- **GIVEN** `bpf_map_lookup_elem()` 返回 NULL (查询失败)
- **WHEN** 处理返回值
- **THEN** 使用安全的默认行为 (ALLOW 或继续处理)
- **AND** 程序不会崩溃

#### Scenario: Map 更新失败的处理
- **GIVEN** `bpf_map_update_elem()` 返回错误
- **WHEN** 创建新会话失败
- **THEN** 记录失败但继续处理数据包
- **AND** 返回正确的动作码 (不阻塞流量)

#### Scenario: 恶意构造的数据包
- **GIVEN** 数据包头部字段异常 (如非法的协议号)
- **WHEN** 解析数据包
- **THEN** 边界检查通过但逻辑检测到异常
- **AND** 安全地返回 `XDP_PASS`
- **AND** 不会触发漏洞或越界访问

---

## MODIFIED Requirements

### Requirement: TC 程序 Map 定义 (修改以支持 Pinning)
现有 TC 程序的 Map 定义 SHALL 添加 `LIBBPF_PIN_BY_NAME` 标记,以便与 XDP 程序共享数据。

#### Scenario: TC Map 支持 Pinning
- **GIVEN** TC 程序的 `session_map` 定义
- **WHEN** 添加 `__uint(pinning, LIBBPF_PIN_BY_NAME)`
- **THEN** 加载 TC 程序时 Map 被固定到 `/sys/fs/bpf/tc-global/session_map`
- **AND** 功能与之前完全一致 (无功能回归)

**Migration**:
- 修改 `src/bpf/tc_microsegment.bpf.c` 的 Map 定义
- 添加 `__uint(pinning, LIBBPF_PIN_BY_NAME)` 字段
- 测试验证 TC 功能不受影响

---

### Requirement: TC 策略匹配逻辑提取 (重构以支持代码共享)
现有 TC 程序的策略匹配逻辑 SHALL 提取到独立的头文件中,以便 XDP 程序复用。

#### Scenario: TC 程序使用共享头文件
- **GIVEN** 策略匹配逻辑已提取到 `headers/policy_match.h`
- **WHEN** TC 程序 `#include "headers/policy_match.h"`
- **AND** 调用 `lookup_policy_action(&key, &rule_id)`
- **THEN** 功能与原始实现完全一致
- **AND** 所有现有测试通过

**Migration**:
- 创建 `src/bpf/headers/policy_match.h`
- 将 `lookup_policy_action()` 等函数移到头文件
- 修改 TC 程序为 `#include` 方式
- 运行现有测试验证无功能回归
