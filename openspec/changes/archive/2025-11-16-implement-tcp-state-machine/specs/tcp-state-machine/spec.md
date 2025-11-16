# Capability: TCP 状态机追踪

## Purpose

TCP 状态机追踪能力在 eBPF 数据平面层实现符合 RFC 793 标准的完整 TCP 连接生命周期追踪。它使系统能够精确识别 TCP 连接的每个阶段，从三次握手到四次挥手，为高级安全策略、异常检测和精细化流量管理提供基础。

## Context

TCP 是互联网最重要的传输层协议，理解 TCP 连接的状态对于网络安全和可观测性至关重要。完整的 TCP 状态机追踪使系统能够：

- **精确的连接追踪**: 区分连接建立、数据传输、正常关闭和异常终止
- **智能会话管理**: 根据 TCP 状态优化会话超时和资源清理
- **增强安全策略**: 支持基于连接状态的细粒度策略（例如，仅允许已建立连接传输数据）
- **异常行为检测**: 识别异常的 TCP 行为模式（例如，无握手的数据传输，SYN 洪水攻击）
- **准确的指标收集**: 区分不同类型的连接终止（正常 FIN vs 异常 RST）

本能力扩展现有的会话追踪基础设施，将简单的会话状态提升为完整的 TCP 状态机。

## ADDED Requirements

### Requirement: TCP 标志位提取和解析

系统必须(SHALL)从 TCP 头部准确提取和解析以下标志位：

- **FIN (Finish)**: 发送方完成数据发送
- **SYN (Synchronize)**: 同步序列号，用于建立连接
- **RST (Reset)**: 重置连接，异常终止
- **PSH (Push)**: 推送数据到应用层
- **ACK (Acknowledgment)**: 确认号有效
- **URG (Urgent)**: 紧急指针有效

#### Scenario: 提取 TCP 标志位

**Given** TCP 数据包到达 eBPF 程序
**When** 系统提取 TCP 标志位
**Then** 必须(SHALL)正确解析所有 6 个基本标志位（FIN、SYN、RST、PSH、ACK、URG）
**And** 提取过程必须(SHALL)包含边界检查防止越界访问
**And** 提取延迟必须(SHALL) < 50 纳秒

#### Scenario: SYN+ACK 组合标志检测

**Given** TCP 数据包包含 SYN=1、ACK=1
**When** 系统解析标志位
**Then** 必须(SHALL)正确识别为 SYN+ACK 组合（标志位 0x12）
**And** 能够区分单独的 SYN（0x02）和 SYN+ACK（0x12）

### Requirement: 完整的 TCP 状态机实现

系统必须(SHALL)实现符合 RFC 793 的完整 TCP 状态机，包括以下 11 种状态：

1. **CLOSED**: 连接关闭或尚未建立
2. **LISTEN**: 服务器被动打开，等待连接请求（eBPF 层可选）
3. **SYN_SENT**: 客户端主动打开，已发送 SYN
4. **SYN_RECV**: 服务器收到 SYN，已发送 SYN+ACK
5. **ESTABLISHED**: 连接已建立，可以传输数据
6. **FIN_WAIT_1**: 主动关闭方，已发送 FIN
7. **FIN_WAIT_2**: 主动关闭方，收到对方 ACK
8. **CLOSE_WAIT**: 被动关闭方，收到 FIN
9. **CLOSING**: 同时关闭，双方都发送 FIN
10. **LAST_ACK**: 被动关闭方，发送 FIN 等待最后 ACK
11. **TIME_WAIT**: 主动关闭方等待 2MSL

#### Scenario: TCP 三次握手 - 客户端视角

**Given** 客户端发起新的 TCP 连接
**When** 客户端发送 SYN 数据包
**Then** 会话的 tcp_state 必须(SHALL)转换为 TCP_STATE_SYN_SENT

**When** 服务器响应 SYN+ACK 数据包
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_ESTABLISHED
**And** session_state 必须(SHALL)同步更新为 SESSION_STATE_ESTABLISHED

#### Scenario: TCP 三次握手 - 服务器视角

**Given** 服务器收到客户端的 SYN 数据包
**When** eBPF 程序处理 SYN 数据包
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_SYN_RECV

**When** 服务器发送 SYN+ACK 后收到客户端的 ACK
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_ESTABLISHED
**And** session_state 必须(SHALL)同步更新为 SESSION_STATE_ESTABLISHED

#### Scenario: TCP 四次挥手 - 主动关闭方

**Given** TCP 连接处于 ESTABLISHED 状态
**When** 主动关闭方发送 FIN 数据包
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_FIN_WAIT_1
**And** session_state 必须(SHALL)转换为 SESSION_STATE_CLOSING

**When** 收到对方的 ACK 数据包
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_FIN_WAIT_2

**When** 收到对方的 FIN 数据包
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_TIME_WAIT
**And** 应该(SHOULD)发送最后的 ACK（TCP 栈处理）

**When** 超时（2MSL，通常 2 分钟）
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_CLOSED
**And** session_state 必须(SHALL)转换为 SESSION_STATE_CLOSED
**Note**: 超时处理由用户空间负责

#### Scenario: TCP 四次挥手 - 被动关闭方

**Given** TCP 连接处于 ESTABLISHED 状态
**When** 被动关闭方收到对方的 FIN 数据包
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_CLOSE_WAIT
**And** session_state 必须(SHALL)转换为 SESSION_STATE_CLOSING

**When** 被动关闭方发送自己的 FIN 数据包
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_LAST_ACK

**When** 收到对方的最后 ACK
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_CLOSED
**And** session_state 必须(SHALL)转换为 SESSION_STATE_CLOSED

#### Scenario: TCP 同时关闭

**Given** TCP 连接处于 ESTABLISHED 状态
**When** 双方几乎同时发送 FIN 数据包
**Then** 主动方的 tcp_state 必须(SHALL)从 FIN_WAIT_1 转换为 TCP_STATE_CLOSING

**When** 双方都收到对方的 FIN 并发送 ACK
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_TIME_WAIT

#### Scenario: TCP 异常关闭 - RST 重置

**Given** TCP 连接处于任意非 CLOSED 状态
**When** 收到 RST 数据包
**Then** tcp_state 必须(SHALL)立即转换为 TCP_STATE_CLOSED
**And** session_state 必须(SHALL)立即转换为 SESSION_STATE_CLOSED
**And** 必须(SHALL)增加 STATS_TCP_RST_RECEIVED 计数器
**And** 应该(SHOULD)设置 SESSION_FLAG_RST_SEEN 标志

### Requirement: 双向状态追踪

系统必须(SHALL)根据数据包方向（客户端到服务器 vs 服务器到客户端）进行状态转换：

- 区分主动关闭方和被动关闭方
- 根据方向设置正确的状态（FIN_WAIT_1 vs CLOSE_WAIT）
- 分别追踪双向的 FIN 标志

#### Scenario: 区分主动和被动关闭

**Given** TCP 连接处于 ESTABLISHED 状态
**When** 客户端方向的数据包带有 FIN 标志（客户端主动关闭）
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_FIN_WAIT_1
**And** SESSION_FLAG_CLIENT_FIN 标志必须(SHALL)被设置

**Given** TCP 连接处于 ESTABLISHED 状态
**When** 服务器方向的数据包带有 FIN 标志（服务器主动关闭）
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_CLOSE_WAIT（从客户端视角）
**And** SESSION_FLAG_SERVER_FIN 标志必须(SHALL)被设置

### Requirement: 状态转换验证

系统必须(SHALL)验证状态转换的合法性，检测并处理无效的状态转换：

- 验证 RFC 793 定义的合法转换路径
- 检测异常转换（例如，从 CLOSED 直接到 ESTABLISHED）
- 记录无效转换尝试
- 在检测到无效转换时保持当前状态

#### Scenario: 检测无效状态转换

**Given** TCP 连接处于 SYN_SENT 状态
**When** 收到既无 SYN 也无 ACK 的纯数据包
**Then** tcp_state 必须(SHALL)保持在 SYN_SENT 状态（不转换）
**And** 应该(SHOULD)设置 SESSION_FLAG_INVALID 标志
**And** 必须(SHALL)增加 STATS_TCP_INVALID_TRANSITION 计数器

#### Scenario: 允许宽容的状态转换

**Given** 网络存在数据包乱序或丢失
**When** 收到的数据包标志与当前状态不完全匹配，但仍在合理范围内
**Then** 系统应该(SHOULD)采取宽容策略，允许状态前进
**And** 不应该(SHOULD NOT)因轻微的异常就将连接标记为无效

### Requirement: 会话状态同步

系统必须(SHALL)在 TCP 状态转换时同步更新会话状态（SESSION_STATE_*）：

| TCP 状态 | 会话状态 |
|---------|---------|
| CLOSED, LISTEN | N/A（无会话）|
| SYN_SENT, SYN_RECV | SESSION_STATE_NEW |
| ESTABLISHED | SESSION_STATE_ESTABLISHED |
| FIN_WAIT_1, FIN_WAIT_2, CLOSE_WAIT, CLOSING, LAST_ACK | SESSION_STATE_CLOSING |
| TIME_WAIT, CLOSED（关闭后）| SESSION_STATE_CLOSED |

#### Scenario: 会话状态自动同步

**Given** TCP 连接从 SYN_SENT 转换到 ESTABLISHED
**When** tcp_state 更新为 TCP_STATE_ESTABLISHED
**Then** session->state 必须(SHALL)自动更新为 SESSION_STATE_ESTABLISHED

**Given** TCP 连接从 ESTABLISHED 转换到 FIN_WAIT_1
**When** tcp_state 更新为 TCP_STATE_FIN_WAIT_1
**Then** session->state 必须(SHALL)自动更新为 SESSION_STATE_CLOSING

### Requirement: 性能优化

TCP 状态机实现必须(SHALL)满足严格的性能要求：

- **快速路径优化**: ESTABLISHED 状态且无 FIN/RST 标志的数据包处理延迟 < 0.2 μs
- **慢速路径优化**: 涉及状态转换的数据包处理延迟 < 1.0 μs
- **零数据包丢弃**: 在正常负载下（100K pps）不得因 TCP 状态机处理导致数据包丢弃
- **内联函数**: 所有状态机函数必须使用 `__always_inline` 强制内联

#### Scenario: ESTABLISHED 状态快速路径性能

**Given** TCP 连接处于 ESTABLISHED 状态
**When** 收到不包含 FIN 或 RST 标志的 ACK 数据包
**Then** update_tcp_state() 函数必须(SHALL)在 < 0.2 微秒内完成
**And** 必须(SHALL)使用快速路径跳过复杂的状态转换逻辑
**And** 必须(SHALL)直接返回，不执行状态更新

#### Scenario: 状态转换性能

**Given** TCP 连接处于 SYN_SENT 状态
**When** 收到 SYN+ACK 数据包触发状态转换
**Then** update_tcp_state() 函数必须(SHALL)在 < 1.0 微秒内完成
**And** 包含状态更新、会话状态同步和统计计数器更新

#### Scenario: eBPF 验证器兼容性

**Given** TCP 状态机代码编译为 eBPF 字节码
**When** 加载到 Linux 内核（版本 ≥ 5.10）
**Then** 必须(SHALL)通过 eBPF 验证器检查
**And** 不得(SHALL NOT)超过指令数量限制
**And** 不得(SHALL NOT)超过栈空间限制（512 字节）
**And** 不得(SHALL NOT)包含无界循环

### Requirement: 统计和可观测性

系统必须(SHALL)提供 TCP 状态相关的统计计数器：

- **STATS_TCP_SYN_SENT**: 当前处于 SYN_SENT 状态的连接数
- **STATS_TCP_SYN_RECV**: 当前处于 SYN_RECV 状态的连接数
- **STATS_TCP_ESTABLISHED**: 当前处于 ESTABLISHED 状态的连接数
- **STATS_TCP_FIN_WAIT**: 当前处于 FIN_WAIT_* 状态的连接数
- **STATS_TCP_CLOSE_WAIT**: 当前处于 CLOSE_WAIT 状态的连接数
- **STATS_TCP_TIME_WAIT**: 当前处于 TIME_WAIT 状态的连接数
- **STATS_TCP_RST_RECEIVED**: 收到的 RST 数据包总数
- **STATS_TCP_INVALID_TRANSITION**: 检测到的无效状态转换总数

#### Scenario: 统计计数器更新

**Given** TCP 连接从 SYN_SENT 转换到 ESTABLISHED
**When** 状态转换完成
**Then** STATS_TCP_ESTABLISHED 计数器必须(SHALL)递增 1
**And** 统计数据必须(SHALL)可通过 bpftool 或用户空间 API 查询

#### Scenario: 异常统计收集

**Given** 系统检测到无效的 TCP 状态转换
**When** 无效转换被识别
**Then** STATS_TCP_INVALID_TRANSITION 计数器必须(SHALL)递增 1
**And** 统计数据必须(SHALL)用于异常检测和告警

### Requirement: 调试支持 SHALL 通过条件编译提供

系统必须(SHALL)提供条件编译的调试支持：

- 通过 `DEBUG_MODE` 编译标志控制调试日志
- 在状态转换时输出详细日志（旧状态、新状态、TCP 标志）
- 记录无效状态转换尝试
- 在生产模式下完全禁用调试代码（零性能开销）

#### Scenario: 调试模式日志输出

**Given** 编译时 DEBUG_MODE=1
**When** TCP 状态从 SYN_SENT 转换到 ESTABLISHED
**Then** 系统必须(SHALL)通过 bpf_printk() 输出状态转换日志
**And** 日志格式必须(SHALL)包含：旧状态、新状态、TCP 标志、方向
**And** 日志应该(SHOULD)可通过 /sys/kernel/debug/tracing/trace_pipe 查看

#### Scenario: 生产模式零开销

**Given** 编译时 DEBUG_MODE=0
**When** 处理任何 TCP 数据包
**Then** 不得(SHALL NOT)执行任何 bpf_printk() 调用
**And** 不得(SHALL NOT)增加任何调试相关的处理延迟

### Requirement: 流事件扩展

系统必须(SHALL)在流事件中包含 TCP 状态信息：

- 在 flow_event 结构中添加 tcp_state 和 tcp_flags 字段
- 在重要状态转换时发送流事件（NEW、ESTABLISHED、CLOSED）
- 允许用户空间程序监控 TCP 连接生命周期

#### Scenario: 流事件包含 TCP 状态

**Given** TCP 连接建立成功（ESTABLISHED）
**When** 系统发送 FLOW_EVENT_NEW 事件到用户空间
**Then** flow_event 必须(SHALL)包含 tcp_state 字段（值为 TCP_STATE_ESTABLISHED）
**And** flow_event 必须(SHALL)包含触发事件的 tcp_flags 字段

**Given** TCP 连接关闭（CLOSED）
**When** 系统发送 FLOW_EVENT_CLOSED 事件
**Then** flow_event 必须(SHALL)包含最终的 tcp_state 字段（TCP_STATE_CLOSED）
**And** flow_event 应该(SHOULD)包含关闭原因（正常 FIN vs 异常 RST）

### Requirement: 异常处理和降级

系统必须(SHALL)优雅地处理错误和异常情况：

- NULL 指针检查（session 或 tcph）
- 边界检查失败时跳过状态机更新
- 无效状态转换时保持当前状态
- 任何错误不得影响数据包的转发决策

#### Scenario: NULL 指针保护

**Given** update_tcp_state() 被调用
**When** session 参数为 NULL 或 tcph 参数为 NULL
**Then** 函数必须(SHALL)立即返回 -1（错误）
**And** 不得(SHALL NOT)尝试访问 NULL 指针
**And** 不得(SHALL NOT)导致内核崩溃或数据包丢弃

#### Scenario: 错误不影响转发

**Given** TCP 状态机处理过程中发生错误
**When** 错误被检测到（例如，无效状态转换）
**Then** 系统必须(SHALL)继续处理数据包
**And** 数据包转发决策不得(SHALL NOT)受状态机错误影响
**And** 策略执行必须(SHALL)正常进行

## MODIFIED Requirements

*本能力为新增能力，无修改已有要求*

## REMOVED Requirements

*本能力为新增能力，无删除已有要求*

## Data Structures

### 扩展的 session_value 结构

```c
struct session_value {
    __u64 created_ts;         // 会话创建时间戳（纳秒）
    __u64 last_seen_ts;       // 最后数据包时间戳
    __u64 packets_to_server;  // 客户端->服务器数据包数
    __u64 packets_to_client;  // 服务器->客户端数据包数
    __u64 bytes_to_server;    // 客户端->服务器字节数
    __u64 bytes_to_client;    // 服务器->客户端字节数
    __u8  state;              // 会话状态（SESSION_STATE_*）
    __u8  tcp_state;          // TCP 状态（TCP_STATE_*）- 已存在
    __u8  policy_action;      // 匹配的策略动作
    __u8  flags;              // 会话标志位（新增）
    __u32 pad;                // 对齐填充
};
```

### 会话标志位定义（新增）

```c
#define SESSION_FLAG_CLIENT_FIN  0x01  // 客户端已发送 FIN
#define SESSION_FLAG_SERVER_FIN  0x02  // 服务器已发送 FIN
#define SESSION_FLAG_RST_SEEN    0x04  // 看到 RST 标志
#define SESSION_FLAG_INVALID     0x80  // 检测到无效状态转换
```

### TCP 标志位定义

```c
#define TCP_FIN  0x01  // Finish
#define TCP_SYN  0x02  // Synchronize
#define TCP_RST  0x04  // Reset
#define TCP_PSH  0x08  // Push
#define TCP_ACK  0x10  // Acknowledgment
#define TCP_URG  0x20  // Urgent

// 常用标志组合
#define TCP_FLAGS_SYN_ACK  (TCP_SYN | TCP_ACK)  // 0x12
#define TCP_FLAGS_FIN_ACK  (TCP_FIN | TCP_ACK)  // 0x11
```

### 扩展的流事件结构

```c
struct flow_event {
    // 现有字段...
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;
    __u8  event_type;
    __u8  direction;
    __u8  padding;
    __u64 packet_count;
    __u64 byte_count;
    __u64 timestamp_ns;
    __u32 policy_id;
    __u8  policy_action;
    __u8  state;

    // 新增字段
    __u8  tcp_state;          // TCP 状态机状态
    __u8  tcp_flags;          // 触发事件的 TCP 标志
    __u16 reserved;
} __attribute__((packed));
```

## Implementation Notes

### 核心函数

1. **extract_tcp_flags(struct tcphdr *tcph)**:
   - 从 TCP 头部提取标志位字节
   - 包含边界检查
   - 延迟 < 50 ns

2. **update_tcp_state(session, tcph, is_client_to_server)**:
   - 实现完整的状态机转换逻辑
   - 使用 switch 语句处理所有状态
   - 包含快速路径优化（ESTABLISHED、RST）
   - 同步更新 session_state

3. **validate_state_transition(old, new, flags)** (可选):
   - 验证状态转换合法性
   - 用于调试和异常检测
   - 可在生产中禁用

### 性能优化技术

1. **内联函数**: 所有函数使用 `__always_inline`
2. **分支预测**: 使用 `__builtin_expect()` 提示编译器
3. **快速路径**: ESTABLISHED 状态无状态转换时直接返回
4. **本地变量**: 减少内存访问，使用寄存器
5. **条件编译**: 调试代码通过宏控制

### eBPF 验证器兼容性

1. **避免循环**: 使用 switch 而非循环
2. **栈限制**: 局部变量 < 64 字节
3. **边界检查**: 所有指针访问前检查
4. **指令限制**: 保持代码简洁，避免复杂嵌套

### 集成点

- **TC 程序**: 在会话更新后调用 `update_tcp_state()`
- **XDP 程序**: 在会话创建/更新后调用 `update_tcp_state()`
- **统计更新**: 在状态转换后更新相应计数器
- **流事件**: 在重要状态转换时发送事件

## Performance Characteristics

| 操作 | 目标延迟 | 实际延迟（预期）|
|------|---------|----------------|
| 标志位提取 | < 50 ns | ~30 ns |
| ESTABLISHED 快速路径 | < 0.2 μs | ~0.15 μs |
| 状态转换（慢速路径）| < 1.0 μs | ~0.8 μs |
| RST 快速路径 | < 0.3 μs | ~0.25 μs |

**内存开销**:
- session_value 结构大小保持不变（64 字节）
- 新增统计计数器：8 个（64 字节）
- 总额外内存 < 100 字节

**CPU 开销**:
- 快速路径（> 95% 数据包）：+ 0.15 μs
- 慢速路径（< 5% 数据包）：+ 0.8 μs
- 平均影响：< 0.2 μs/包

## Related Capabilities

- **session-tracking**: 扩展会话追踪，增加完整 TCP 状态机
- **policy-enforcement**: 未来支持基于 TCP 状态的策略
- **statistics-reporting**: 新增 TCP 状态统计指标
- **flow-management**: 流事件包含 TCP 状态信息

## Future Enhancements

### 短期（1-2 个月）
- 基于 TCP 状态的策略匹配（例如，仅允许 ESTABLISHED 状态传输数据）
- 半开连接检测和清理（SYN_SENT 超时）
- SYN 洪水攻击检测

### 长期（3-6 个月）
- TCP 序列号追踪（高级功能，性能影响较大）
- TCP 选项解析（MSS、窗口缩放、时间戳）
- IPv6 TCP 状态追踪
- TCP 重传检测
