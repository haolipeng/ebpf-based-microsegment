# XDP Flow Events Specification

## Purpose
定义 XDP eBPF 程序的流事件推送能力,确保 XDP 模式下的网络流量可见性与 TC 模式一致。

## ADDED Requirements

### Requirement: XDP 流事件推送
XDP eBPF 程序 SHALL 在检测到新连接时推送流事件到 Ring Buffer。

#### Scenario: XDP 新连接事件推送
- **GIVEN** XDP 程序创建新会话成功
- **WHEN** `bpf_map_update_elem(&session_map, &key, &new_session, BPF_NOEXIST)` 返回 0
- **THEN** 必须调用 `push_flow_event_xdp()` 推送 FLOW_EVENT_NEW 事件
- **AND** 事件必须包含 5-tuple (src_ip, dst_ip, src_port, dst_port, protocol)
- **AND** 事件必须包含初始统计 (packet_count=1, byte_count=数据包长度)
- **AND** 事件必须包含策略上下文 (policy_id, policy_action)
- **AND** 事件方向必须为 POLICY_DIR_INGRESS (XDP 仅支持 ingress)

#### Scenario: XDP 流事件结构一致性
- **THEN** XDP 推送的 flow_event 必须与 TC 程序使用相同的结构体定义
- **AND** 事件大小必须为 48 字节
- **AND** 所有字段必须按相同顺序和类型填充
- **AND** padding 和 reserved 字段必须设置为 0

#### Scenario: XDP Ring Buffer 非阻塞操作
- **WHEN** 调用 `bpf_ringbuf_reserve(&flow_events, sizeof(*event), 0)` 时
- **THEN** 必须使用非阻塞模式 (flags=0)
- **WHEN** Ring Buffer 满时 (reserve 返回 NULL)
- **THEN** 必须静默丢弃事件 (返回 -1)
- **AND** 不能阻塞数据包处理
- **AND** 不能影响后续数据包的正常处理

---

### Requirement: XDP 字节统计补全
XDP eBPF 程序 SHALL 正确计算并记录数据包字节数。

#### Scenario: XDP 数据包长度计算
- **GIVEN** XDP context 提供 `data` 和 `data_end` 指针
- **WHEN** 创建新会话时
- **THEN** 必须计算数据包长度: `packet_len = ctx->data_end - ctx->data`
- **AND** 必须设置 `new_session.bytes_to_server = packet_len`
- **AND** `bytes_to_server` 不能为 0 (除非是空数据包)

#### Scenario: XDP 会话字节统计
- **WHEN** 更新现有会话时
- **THEN** 必须累加数据包长度: `session->bytes_to_server += packet_len`
- **AND** 字节统计必须在推送流事件前更新

---

### Requirement: XDP 流事件辅助函数
XDP eBPF 程序 SHALL 实现 `push_flow_event_xdp()` 辅助函数。

#### Scenario: 函数签名和参数
- **THEN** 函数签名必须为:
  ```c
  static __always_inline int push_flow_event_xdp(
      struct flow_key *key,
      __u64 timestamp_ns,
      __u64 packet_count,
      __u64 byte_count,
      __u8 event_type,
      __u8 policy_action,
      __u32 policy_id,
      __u8 state,
      __u8 direction)
  ```
- **AND** 必须标记为 `__always_inline` 以满足 eBPF 验证器
- **AND** 返回值: 0 表示成功,-1 表示失败

#### Scenario: 事件字段填充
- **WHEN** 推送流事件时
- **THEN** 必须从 `struct flow_key` 复制所有 5-tuple 字段
- **AND** 必须设置 `event_type` (FLOW_EVENT_NEW 等)
- **AND** 必须设置 `direction` (POLICY_DIR_INGRESS)
- **AND** 必须设置 `timestamp_ns` 为当前时间戳
- **AND** 必须设置 `packet_count` 和 `byte_count`
- **AND** 必须设置 `policy_id`, `policy_action`, `state`
- **AND** 必须将 `padding` 和 `reserved` 字段清零

#### Scenario: Ring Buffer 提交
- **WHEN** 成功 reserve Ring Buffer 空间后
- **THEN** 必须调用 `bpf_ringbuf_submit(event, 0)` 提交事件
- **AND** 必须使用非阻塞提交 (flags=0)

---

## MODIFIED Requirements

### Requirement: Flow 数据收集 (flow-management)
系统 SHALL 扩展流数据收集能力以支持 XDP 程序。

#### Scenario: XDP 和 TC 事件一致性
- **THEN** XDP 程序推送的流事件 SHALL 与 TC 程序推送的事件格式一致
- **AND** Flow Collector SHALL 能无差别处理 XDP 和 TC 的事件
- **AND** XDP 事件的 `direction` 字段 SHALL 始终为 POLICY_DIR_INGRESS

#### Scenario: XDP 流事件验证
- **WHEN** Flow Collector 接收到事件时
- **THEN** SHALL 验证事件大小为 48 字节
- **AND** SHALL 验证 5-tuple 字段非零 (对于 TCP/UDP)
- **AND** SHALL 验证 `byte_count` 大于 0 (对于非空数据包)

---

## 实现注意事项

### eBPF 验证器要求
- 所有辅助函数必须标记 `__always_inline`
- 必须对所有指针进行边界检查
- Ring Buffer 操作必须检查返回值

### 性能考虑
- Ring Buffer 操作应在 "慢路径"(新会话创建)执行
- 热路径 (现有会话匹配) 不应增加事件推送开销
- 非阻塞操作避免影响数据包转发性能

### 调试支持
- 在 DEBUG_MODE 下,Ring Buffer reserve 失败时打印警告
- 记录 Ring Buffer 丢包次数 (可选,通过统计计数器)
