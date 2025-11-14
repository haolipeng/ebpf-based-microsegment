# Proposal: XDP 流事件推送实现

## 概述

当前 XDP eBPF 程序在创建新会话后未推送流事件到 Ring Buffer,导致控制平面无法感知 XDP 模式下的新连接。本提案旨在实现 XDP 程序的流事件推送功能,与 TC 程序保持一致。

## 动机

### 问题描述

1. **事件缺失**: XDP 程序创建新会话后有明确的 TODO 标记 (`xdp_microsegment.bpf.c:194`),未推送流事件
2. **功能不一致**: TC 程序已完整实现流事件推送,但 XDP 程序缺失此功能
3. **可见性缺失**: 在 XDP 模式下运行时,控制平面无法感知新连接,影响:
   - 流量可视化不完整
   - 审计日志缺失 XDP 层数据
   - 无法进行实时监控和告警

### 当前实现状态

**TC 程序 (已实现)**:
- ✅ 定义了 `push_flow_event()` 辅助函数
- ✅ 在 `create_session()` 中推送 `FLOW_EVENT_NEW` 事件
- ✅ 包含完整的 5-tuple、统计信息、策略上下文

**XDP 程序 (未实现)**:
- ✅ 定义了 Ring Buffer map (`flow_events`)
- ❌ 未实现 `push_flow_event()` 辅助函数
- ❌ 创建会话后未推送事件 (有 TODO 标记)
- ⚠️ 字节统计缺失 (`bytes_to_server = 0`)

## 目标

1. **实现 XDP 流事件推送**: 在 XDP 程序中添加 `push_flow_event()` 函数并调用
2. **补全字节统计**: 从 XDP context 计算数据包长度
3. **保持一致性**: 与 TC 程序的事件格式和行为保持一致
4. **验证集成**: 确保 Go Flow Collector 能正确接收 XDP 事件

## 非目标

- 不修改 `struct flow_event` 定义(保持 48 字节)
- 不实现连接关闭事件推送(留待后续提案)
- 不修改 Ring Buffer 大小或配置
- 不改变 TC 程序现有实现

## 影响范围

### 修改的文件

1. **`src/bpf/xdp_microsegment.bpf.c`**:
   - 添加 `push_flow_event_xdp()` 辅助函数
   - 在创建会话后调用推送事件
   - 补全数据包长度计算

### 不修改的文件

- `src/bpf/headers/common_types.h` (event 定义不变)
- `src/bpf/tc_microsegment.bpf.c` (TC 实现不变)
- Go Flow Collector 代码 (已支持处理事件)

## 实现方案

### 1. 添加 XDP 流事件推送函数

在 `xdp_microsegment.bpf.c` 中添加辅助函数:

```c
// Helper: Push flow event to user-space via Ring Buffer (XDP version)
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
{
    // 实现与 TC 相同,但适配 XDP context
    struct flow_event *event = bpf_ringbuf_reserve(&flow_events, sizeof(*event), 0);
    if (!event) {
        return -1;
    }

    // 填充事件字段 (与 TC 完全一致)
    event->src_ip = key->src_ip;
    event->dst_ip = key->dst_ip;
    event->src_port = key->src_port;
    event->dst_port = key->dst_port;
    event->protocol = key->protocol;
    event->event_type = event_type;
    event->direction = direction;
    event->packet_count = packet_count;
    event->byte_count = byte_count;
    event->timestamp_ns = timestamp_ns;
    event->policy_id = policy_id;
    event->policy_action = policy_action;
    event->state = state;
    event->padding = 0;
    event->reserved = 0;

    bpf_ringbuf_submit(event, 0);
    return 0;
}
```

### 2. 补全字节统计

在创建会话前计算数据包长度:

```c
// 计算数据包长度
__u32 packet_len = (void *)(long)ctx->data_end - (void *)(long)ctx->data;

struct session_value new_session = {
    // ...
    .bytes_to_server = packet_len,  // 修复: 原为 0
    .bytes_to_client = 0,
};
```

### 3. 调用事件推送

在会话创建成功后推送事件:

```c
int ret = bpf_map_update_elem(&session_map, &key, &new_session, BPF_NOEXIST);
if (ret == 0) {
    update_stats(STATS_NEW_SESSIONS);
    update_stats(STATS_ACTIVE_SESSIONS);

    // 推送流事件到 Ring Buffer
    push_flow_event_xdp(
        &key,
        now,
        1,              // First packet
        packet_len,     // Packet bytes
        FLOW_EVENT_NEW,
        action,
        matched_rule_id,
        FLOW_STATE_ACTIVE,
        POLICY_DIR_INGRESS  // XDP 仅支持 ingress
    );
}
```

## 验证方案

### 编译验证

```bash
make bpf
# 确保编译通过,无 eBPF 验证器错误
```

### 功能测试

1. **启动 XDP 模式 Agent**:
   ```bash
   sudo ./bin/microsegment-agent -c config/agent-xdp.yaml
   ```

2. **生成测试流量**:
   ```bash
   curl http://test-server
   ping target-host
   ```

3. **验证事件接收**:
   - 检查 Agent 日志是否有 "Flow event received" 消息
   - 验证事件包含正确的 5-tuple 和统计信息
   - 确认 `byte_count` 不为 0

### 性能测试

- 使用 `iperf3` 生成高流量,验证无性能回归
- 检查 Ring Buffer 无异常丢包
- 对比 XDP 和 TC 模式的事件推送性能

## 风险和依赖

### 风险

1. **性能影响**: Ring Buffer 操作可能增加 XDP 数据路径延迟
   - **缓解**: 使用非阻塞 reserve/submit,满时静默丢弃

2. **eBPF 验证器**: 复杂的辅助函数可能触发验证器限制
   - **缓解**: 标记 `__always_inline`,与 TC 验证成功的实现保持一致

### 依赖

- 无外部依赖
- 依赖现有的 `struct flow_event` 定义
- 依赖 Go Flow Collector 已有的事件处理能力

## 替代方案

### 方案 A: 共享 TC 的 push_flow_event 函数

将 `push_flow_event()` 移到 `headers/flow_processing.h`,TC 和 XDP 共享。

**优点**: 消除代码重复
**缺点**:
- 需要处理不同 context 类型 (struct __sk_buff vs struct xdp_md)
- 可能需要宏或复杂的类型转换
- 增加维护复杂度

**决定**: 不采用,保持 TC/XDP 独立实现更清晰

### 方案 B: 延迟到用户态推送

在 Go Flow Collector 中周期性扫描 session_map,构造事件。

**优点**: eBPF 代码更简单
**缺点**:
- 无法获取首包时间戳
- 增加用户态 CPU 开销
- 延迟大,实时性差

**决定**: 不采用,Ring Buffer 是标准做法

## 时间线

- **Week 1**: 实现 `push_flow_event_xdp()` 和字节统计
- **Week 1**: 编译验证和基础功能测试
- **Week 1**: 性能测试和文档更新

**总计**: 1 周(约 8 小时工作量)

## 成功标准

1. ✅ XDP 程序成功推送流事件,无 TODO 标记
2. ✅ Go Flow Collector 接收到 XDP 事件
3. ✅ 字节统计不为 0
4. ✅ 编译通过,无 eBPF 验证器错误
5. ✅ 性能测试无回归 (延迟 <5% 增加)
6. ✅ Ring Buffer 无异常丢包

## 参考资料

- TC 实现: `src/bpf/tc_microsegment.bpf.c:137-179`
- Flow 事件定义: `src/bpf/headers/common_types.h:129-148`
- Flow Collector: `src/agent/pkg/flow/collector.go`
- 现有规范: `openspec/specs/flow-management/spec.md`
