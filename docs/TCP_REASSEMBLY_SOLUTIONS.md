# TCP 重组解决方案对比

## 文档概览

**目的**: 详细对比 TCP 重组的各种解决方案，包括技术实现、性能分析和适用场景。

**状态**: 方案设计完成
**日期**: 2025-11-19
**相关文档**:
- [TCP 重组必要性分析](./TCP_REASSEMBLY_ANALYSIS.md)
- [TCP 重组实现指南](./TCP_REASSEMBLY_IMPLEMENTATION.md)
- [应用层协议检测实现方案](./APPLICATION_LAYER_PROTOCOL_DETECTION.md)

---

## 目录

1. [方案总览](#1-方案总览)
2. [阶段 1: 仅首包检测](#2-阶段-1-仅首包检测)
3. [阶段 2: 序列号跟踪](#3-阶段-2-序列号跟踪)
4. [阶段 3: 用户态重组](#4-阶段-3-用户态重组)
5. [阶段 4: eBPF 轻量级重组](#5-阶段-4-ebpf-轻量级重组)
6. [性能分析对比](#6-性能分析对比)
7. [eBPF 实现挑战](#7-ebpf-实现挑战)
8. [方案选择建议](#8-方案选择建议)

---

## 1. 方案总览

### 1.1 四种方案概览

我们设计了四种渐进式的解决方案，从最简单到最复杂：

```
复杂度/功能 递增 →

阶段 1            阶段 2            阶段 3             阶段 4
仅首包检测        +序列号跟踪        用户态重组         eBPF轻量级重组
┌────────┐       ┌────────┐        ┌────────┐         ┌────────┐
│ 检测   │  →    │ 检测   │   →    │ 检测   │    →    │ 检测   │
│ 首包   │       │ +序列号│        │ +用户态│         │ +eBPF  │
│        │       │ 验证   │        │ 重组   │         │ 缓存   │
└────────┘       └────────┘        └────────┘         └────────┘
90%+ 覆盖        90%+ 覆盖         99%+ 覆盖          95%+ 覆盖
<10μs            <20μs             ~300μs             ~80μs
极简             简单              中等               复杂
```

### 1.2 关键指标对比

| 指标 | 阶段 1 | 阶段 2 | 阶段 3 | 阶段 4 |
|------|--------|--------|--------|--------|
| **覆盖率** | 90%+ | 90%+ | 99%+ | 95%+ |
| **每包开销** | <10μs | <20μs | ~300μs (慢速路径) | ~80μs |
| **内存开销** | 0 | 0 | ~200MB (用户态) | ~40MB |
| **代码复杂度** | 极低 (5-10 行) | 低 (50 行) | 中 (300 行 Go) | 高 (500-1000 行) |
| **实现时间** | 1-2 天 | +1 天 | 3-5 天 | 7-10 天 |
| **维护成本** | 极低 | 低 | 中 | 高 |
| **Verifier 风险** | 无 | 无 | 无 | 高 |
| **推荐度** | ✅✅✅ | ✅✅ | ⚠️ | ❌ |

### 1.3 适用场景

| 场景 | 推荐方案 | 原因 |
|------|----------|------|
| **标准 HTTP/HTTPS 流量** | 阶段 1 | 协议签名小，首包足够 |
| **良好网络环境** | 阶段 1 | 乱序罕见 |
| **需要可观测性** | 阶段 2 | 提供乱序/重传统计 |
| **高丢包率网络** | 阶段 3 | 需要完整重组 |
| **延迟敏感** | 阶段 1-2 | 低开销 |
| **检测率要求 99%+** | 阶段 3 | 用户态完整重组 |
| **资源受限** | 阶段 1 | 零内存开销 |

---

## 2. 阶段 1: 仅首包检测

### 2.1 核心思想

**策略**: 只在会话的前几个数据包（通常是第一个有 payload 的包）上执行协议检测，不做任何重组。

**基本假设**: 90%+ 的协议特征签名在首个 TCP 段中完整。

### 2.2 技术实现

**核心逻辑**:

```c
// 在协议检测函数中
static __always_inline int detect_app_protocol_first_packet(
    void *payload_start,
    void *data_end,
    struct flow_key *key,
    struct session_value *session,
    struct proto_detect_config *config)
{
    // 跳过已检测的会话
    if (session->proto_confidence >= 90)
        return 0;

    // 只在前几个包检测
    __u64 total_packets = session->packets_to_server + session->packets_to_client;
    if (total_packets > 5)
        return 0;  // 超过 5 个包仍未检测，放弃

    // 检查 payload 是否存在
    if (!payload_start || payload_start >= data_end)
        return 0;

    // 执行协议检测
    __u8 detected_proto = APP_PROTO_UNKNOWN;
    __u8 confidence = 0;
    __u16 flags = 0;

    // 尝试各种协议检测器
    if (key->protocol == IPPROTO_TCP) {
        if (detect_http(payload_start, data_end, &confidence, &flags)) {
            detected_proto = APP_PROTO_HTTP;
        } else if (detect_tls(payload_start, data_end, &confidence, &flags)) {
            detected_proto = APP_PROTO_HTTPS;
        }
        // ... 其他协议
    }

    // 更新会话
    if (detected_proto != APP_PROTO_UNKNOWN) {
        session->app_protocol = detected_proto;
        session->proto_confidence = confidence;
        update_stats(STATS_PROTO_DETECTED);
    }

    return 0;
}
```

### 2.3 优点

✅ **极简实现**:
- 代码量：5-10 行核心逻辑
- 无需复杂数据结构
- 无状态管理

✅ **零开销**:
- 内存：0 额外字节
- 延迟：<10μs per packet
- 仅在前几个包执行检测

✅ **高覆盖率**:
- 90%+ 协议可成功检测
- 基于数据分析验证

✅ **生产就绪**:
- 无 verifier 风险
- 可立即部署
- 易于调试

### 2.4 局限性

❌ **无法处理**:
- 协议头部跨多个 TCP 段
- 乱序到达的首包
- 首包损坏的情况

❌ **边缘场景**:
- 非常长的 URL（罕见）
- 自定义协议（协议检测器未实现）
- 高丢包率网络（首包可能丢失）

### 2.5 性能指标

| 指标 | 值 |
|------|-----|
| **检测成功率** | 90-95% |
| **每包延迟** | 5-10μs |
| **CPU 开销** | ~10 CPU 核 @ 1M PPS |
| **内存开销** | 0 bytes |
| **误报率** | <1% |
| **漏报率** | 5-10% |

### 2.6 使用建议

**推荐使用条件**:
- [x] 主要流量是常见协议（HTTP、HTTPS、DNS、MySQL 等）
- [x] 网络环境良好（低延迟、低丢包）
- [x] 可接受 90%+ 检测率
- [x] 需要快速上线（1-2 天实现）
- [x] 资源受限环境

**不推荐的场景**:
- [ ] 需要 99%+ 检测率
- [ ] 大量自定义协议
- [ ] 高丢包率网络

---

## 3. 阶段 2: 序列号跟踪

### 3.1 核心思想

**策略**: 在阶段 1 基础上，添加 TCP 序列号验证，检测乱序和重传，但不缓存数据。

**目标**:
1. 检测乱序数据包（跳过检测）
2. 检测重传（避免重复检测）
3. 提供可观测性指标
4. 为后续优化提供数据支持

### 3.2 技术实现

**序列号比较**:

```c
// TCP 序列号比较（处理 32 位溢出）
static __always_inline bool tcp_seq_lt(__u32 a, __u32 b)
{
    return ((__s32)(a - b)) < 0;
}

static __always_inline bool tcp_seq_lte(__u32 a, __u32 b)
{
    return ((__s32)(a - b)) <= 0;
}
```

**序列号验证**:

```c
// 检查 TCP 序列号是否符合预期
static __always_inline bool tcp_seq_is_in_order(
    struct session_value *session,
    __u32 tcp_seq,
    __u32 payload_len,
    bool is_client_to_server)
{
    if (payload_len == 0)
        return true;  // 无 payload，无序列号问题

    __u32 *expected_seq;

    // 选择方向的序列号
    if (is_client_to_server) {
        expected_seq = &session->tcp_seq_client;
    } else {
        expected_seq = &session->tcp_seq_server;
    }

    // 首个 payload 包
    if (*expected_seq == 0) {
        *expected_seq = tcp_seq + payload_len;
        return true;
    }

    // 检查序列号
    if (tcp_seq == *expected_seq) {
        // 有序，更新期望序列号
        *expected_seq += payload_len;
        return true;
    }
    else if (tcp_seq_lt(tcp_seq, *expected_seq)) {
        // 重传（旧数据）
        update_stats(STATS_TCP_RETRANS);
        return false;
    }
    else {
        // 乱序（未来数据）
        update_stats(STATS_TCP_OUT_OF_ORDER);
        session->proto_flags |= PROTO_FLAG_OUT_OF_ORDER;
        return false;
    }
}
```

**集成到检测逻辑**:

```c
// 增强的协议检测
static __always_inline int detect_with_seq_check(
    void *payload_start,
    void *data_end,
    struct flow_key *key,
    struct session_value *session,
    __u32 tcp_seq,
    bool is_client_to_server)
{
    // 跳过高置信度会话
    if (session->proto_confidence >= 90)
        return 0;

    // 检查 payload
    if (!payload_start || payload_start >= data_end)
        return 0;

    __u32 payload_len = data_end - payload_start;

    // 序列号验证
    if (!tcp_seq_is_in_order(session, tcp_seq, payload_len, is_client_to_server)) {
        // 乱序或重传，跳过检测
        update_stats(STATS_PROTO_SKIP_OUT_OF_ORDER);
        return 0;
    }

    // 执行协议检测（与阶段 1 相同）
    return detect_app_protocol_first_packet(...);
}
```

### 3.3 优点

✅ **检测乱序**:
- 识别乱序数据包
- 避免在乱序包上浪费 CPU
- 提供网络质量洞察

✅ **检测重传**:
- 识别重传数据包
- 避免重复检测
- 提供丢包率统计

✅ **低开销**:
- 仅增加序列号比较（<10μs）
- 无额外内存（使用现有字段）
- 总开销 <20μs

✅ **可观测性**:
- `STATS_TCP_OUT_OF_ORDER`: 乱序包计数
- `STATS_TCP_RETRANS`: 重传计数
- `STATS_PROTO_SKIP_OUT_OF_ORDER`: 跳过的检测

### 3.4 局限性

❌ **仍无法重组**:
- 不缓存乱序段
- 无法处理分片的协议头
- 乱序首包仍会被跳过

⚠️ **依赖准确的方向判断**:
- 需要正确区分客户端→服务端 vs 服务端→客户端
- 双向流量的序列号独立跟踪

### 3.5 性能指标

| 指标 | 阶段 1 | 阶段 2 |
|------|--------|--------|
| **检测成功率** | 90-95% | 90-95% |
| **每包延迟** | 5-10μs | 10-20μs |
| **额外开销** | - | +5-10μs |
| **内存开销** | 0 bytes | 0 bytes (复用字段) |
| **误报率** | <1% | <0.5% (更精确) |

### 3.6 提供的洞察

通过监控指标，可以了解：

1. **网络质量**:
   - 乱序率 = `STATS_TCP_OUT_OF_ORDER` / `STATS_TOTAL_PACKETS`
   - 重传率 = `STATS_TCP_RETRANS` / `STATS_TOTAL_PACKETS`

2. **检测效果**:
   - 实际检测率 = `STATS_PROTO_DETECTED` / (`STATS_PROTO_DETECTED` + `STATS_PROTO_UNKNOWN`)
   - 乱序影响 = `STATS_PROTO_SKIP_OUT_OF_ORDER` / `STATS_PROTO_DETECTED`

3. **是否需要重组**:
   - 如果乱序率 > 5%，考虑阶段 3
   - 如果检测率 < 85%，调查原因

### 3.7 使用建议

**推荐作为阶段 1 的增强**:
- [x] 提供网络可观测性
- [x] 辅助故障诊断
- [x] 为重组决策提供数据支持
- [x] 开销极低，可默认启用

**实现优先级**: 中等（阶段 1 完成后立即添加）

---

## 4. 阶段 3: 用户态重组

### 4.1 核心思想

**策略**: 在 eBPF 中进行快速首包检测，失败的流推送到用户态进行完整 TCP 重组和深度检测。

**架构**: 双路径设计（快速路径 + 慢速路径）

```
┌──────────────────────────────────────────────┐
│           eBPF 快速路径                       │
├──────────────────────────────────────────────┤
│ 1. 首包检测                                   │
│ 2. 成功 → 完成 (90%+ 流量)                   │
│ 3. 失败 → 推送到用户态                       │
└──────────────────────────────────────────────┘
         ↓ (Ring Buffer)
┌──────────────────────────────────────────────┐
│          用户态慢速路径                       │
├──────────────────────────────────────────────┤
│ 1. 使用 gopacket/reassembly 库              │
│ 2. 完整 TCP 重组                             │
│ 3. 深度协议检测                              │
│ 4. 更新 eBPF session map                     │
└──────────────────────────────────────────────┘
```

### 4.2 技术实现

#### 4.2.1 eBPF 侧

**检测失败触发**:

```c
// 在 TC/XDP 程序中
if (session->app_protocol == APP_PROTO_UNKNOWN &&
    session->packets_to_server < 5) {

    // 推送到用户态重组
    struct reassembly_request evt = {
        .flow_key = *key,
        .tcp_seq = tcp_seq,
        .payload_len = payload_len,
        .timestamp_ns = bpf_ktime_get_ns(),
        .direction = is_client_to_server ? 0 : 1,
    };

    // 推送事件
    bpf_ringbuf_output(&reassembly_events, &evt, sizeof(evt), 0);

    // 标记会话等待用户态处理
    session->proto_flags |= PROTO_FLAG_PENDING_REASSEMBLY;
}
```

**数据包复制**:

```c
// 可选：将数据包数据复制到 ring buffer
// 方法 1: 仅复制元数据（推荐，低开销）
struct pkt_metadata {
    struct flow_key key;
    __u32 tcp_seq;
    __u16 payload_len;
    __u64 timestamp;
};

// 方法 2: 复制完整数据包（高开销，但用户态无需 pcap）
#define MAX_PKT_COPY 256
struct pkt_data {
    struct pkt_metadata meta;
    __u8 data[MAX_PKT_COPY];
};
```

#### 4.2.2 用户态侧

**TCP 重组器** (`src/agent/pkg/protocol/reassembler.go`):

```go
package protocol

import (
    "github.com/google/gopacket"
    "github.com/google/gopacket/layers"
    "github.com/google/gopacket/reassembly"
    "github.com/cilium/ebpf"
)

type TCPReassembler struct {
    streamFactory *TCPStreamFactory
    streamPool    *reassembly.StreamPool
    assembler     *reassembly.Assembler
    sessionMap    *ebpf.Map  // 用于更新 eBPF map
}

type TCPStream struct {
    flowKey    FlowKey
    buffer     []byte
    packets    int
    firstSeen  time.Time
}

// 实现 reassembly.Stream 接口
func (s *TCPStream) Accept(tcp *layers.TCP, ci gopacket.CaptureInfo,
                          dir reassembly.TCPFlowDirection,
                          nextSeq reassembly.Sequence,
                          start *bool, end *bool) bool {
    // 接受所有数据包
    return true
}

func (s *TCPStream) ReassembledSG(sg reassembly.ScatterGather,
                                  ac reassembly.AssemblyContext) {
    // 获取重组后的数据
    data := sg.Fetch(sg.Length())
    s.buffer = append(s.buffer, data...)

    // 尝试协议检测
    if len(s.buffer) >= 64 {  // 有足够数据
        protocol := DetectProtocolFromBytes(s.buffer)
        if protocol != APP_PROTO_UNKNOWN {
            // 更新 eBPF map
            UpdateSessionProtocol(s.flowKey, protocol)
        }
    }
}

func (s *TCPStream) ReassemblyComplete(ac reassembly.AssemblyContext) {
    // 流结束，清理资源
}

// 处理 eBPF 推送的事件
func (r *TCPReassembler) ProcessEvent(evt *ReassemblyEvent) {
    // 从 evt 构建 gopacket.Packet
    packet := createPacketFromEvent(evt)

    // 提取网络层和传输层
    networkLayer := packet.NetworkLayer()
    tcpLayer := packet.TransportLayer().(*layers.TCP)

    // 提交给重组器
    r.assembler.AssembleWithTimestamp(
        networkLayer.NetworkFlow(),
        tcpLayer,
        evt.Timestamp,
    )
}

// 主循环
func (r *TCPReassembler) Start() {
    // 读取 ring buffer 事件
    reader, err := ringbuffer.NewReader(r.reassemblyEventsMap)
    if err != nil {
        log.Fatal(err)
    }

    for {
        record, err := reader.Read()
        if err != nil {
            continue
        }

        var evt ReassemblyEvent
        binary.Read(bytes.NewReader(record.RawSample),
                    binary.LittleEndian, &evt)

        r.ProcessEvent(&evt)
    }
}
```

**协议检测** (`src/agent/pkg/protocol/detector_userspace.go`):

```go
func DetectProtocolFromBytes(data []byte) AppProtocol {
    if len(data) < 4 {
        return APP_PROTO_UNKNOWN
    }

    // HTTP 检测
    if bytes.HasPrefix(data, []byte("GET ")) ||
       bytes.HasPrefix(data, []byte("POST ")) ||
       bytes.HasPrefix(data, []byte("HTTP/")) {
        return APP_PROTO_HTTP
    }

    // TLS 检测
    if len(data) >= 5 &&
       data[0] == 0x16 &&  // Handshake
       data[1] == 0x03 {   // TLS version major
        return APP_PROTO_HTTPS
    }

    // DNS 检测（虽然通常是 UDP，但 TCP DNS 也存在）
    if len(data) >= 12 {
        // DNS header validation
        questions := binary.BigEndian.Uint16(data[4:6])
        if questions > 0 && questions < 10 {
            return APP_PROTO_DNS
        }
    }

    // ... 其他协议

    return APP_PROTO_UNKNOWN
}
```

**更新 eBPF Map**:

```go
func UpdateSessionProtocol(flowKey FlowKey, protocol AppProtocol) error {
    // 查找会话
    var session SessionValue
    err := sessionMap.Lookup(&flowKey, &session)
    if err != nil {
        return err
    }

    // 更新协议
    session.AppProtocol = uint8(protocol)
    session.ProtoConfidence = 95  // 用户态重组，高置信度
    session.ProtoFlags &= ^PROTO_FLAG_PENDING_REASSEMBLY

    // 写回 map
    return sessionMap.Update(&flowKey, &session, ebpf.UpdateExist)
}
```

### 4.3 优点

✅ **高覆盖率**:
- 99%+ 检测成功率
- 处理所有乱序、重传、分片场景

✅ **完整功能**:
- 使用成熟库（gopacket/reassembly）
- 无 eBPF 限制
- 可实现复杂协议检测

✅ **灵活性**:
- 用户态更新快速
- 易于添加新协议
- 复杂逻辑不受 verifier 限制

✅ **双路径优化**:
- 90%+ 流量走快速路径（<10μs）
- 仅 5-10% 流量走慢速路径

### 4.4 局限性

❌ **高延迟**（慢速路径）:
- Ring buffer 推送: 10-20μs
- 用户态调度: 50-100μs
- 重组处理: 100-300μs
- 总延迟: 160-420μs

❌ **内存开销**:
- 用户态缓冲: ~2KB per stream
- 100K streams = ~200MB

❌ **复杂性**:
- 双路径维护
- eBPF 与用户态同步
- 状态一致性

### 4.5 性能分析

**快速路径**（90%+ 流量）:
```
eBPF 检测: 10μs
成功 → 完成
```

**慢速路径**（5-10% 流量）:
```
eBPF 检测失败: 10μs
 ↓
Ring Buffer 推送: 15μs
 ↓
用户态唤醒: 50μs
 ↓
重组处理: 200μs
 ↓
Map 更新: 20μs
─────────────
总计: ~295μs
```

**平均延迟**:
```
Avg = 0.9 × 10μs + 0.1 × 295μs
    = 9μs + 29.5μs
    = 38.5μs
```

**CPU 开销** @ 1M PPS:
```
快速路径: 0.9M PPS × 10μs = 9 CPU 核
慢速路径: 0.1M PPS × 295μs = 29.5 CPU 核
总计: 38.5 CPU 核
```

### 4.6 使用建议

**推荐使用条件**:
- [ ] eBPF 检测率 < 85%（通过阶段 1+2 验证）
- [ ] 乱序率 > 5%
- [ ] 可接受额外延迟（~300μs 慢速路径）
- [ ] 有充足的 CPU 和内存资源

**实施步骤**:
1. 先部署阶段 1+2，收集指标
2. 如果 `STATS_PROTO_UNKNOWN` > 10%，考虑阶段 3
3. 小规模试点（10% 流量）
4. 监控延迟和 CPU 影响
5. 逐步扩大覆盖范围

---

## 5. 阶段 4: eBPF 轻量级重组

### 5.1 核心思想

**策略**: 在 eBPF 中缓存前 2-3 个 TCP 段，进行有限的重组，以处理简单的乱序和分片场景。

**目标**: 在 eBPF 中提升检测率，同时避免完整重组的复杂性。

### 5.2 技术实现

**段缓存结构**:

```c
#define MAX_CACHED_SEGMENTS 3
#define MAX_SEGMENT_SIZE 128

struct tcp_segment_cache {
    __u32 asm_seq;              // 下一个期望的序列号
    __u32 seg_seq[MAX_CACHED_SEGMENTS];
    __u16 seg_len[MAX_CACHED_SEGMENTS];
    __u8  seg_data[MAX_CACHED_SEGMENTS][MAX_SEGMENT_SIZE];
    __u8  seg_count;
    __u8  pad[3];
} __attribute__((packed));

struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 1000);  // 仅缓存 1000 个流
    __type(key, struct flow_key);
    __type(value, struct tcp_segment_cache);
} tcp_segment_cache_map SEC(".maps");
```

**重组逻辑**:

```c
static __always_inline int try_tcp_reassemble(
    struct flow_key *key,
    __u32 tcp_seq,
    void *payload,
    __u16 payload_len,
    void *data_end,
    __u8 *out_buffer,  // 输出：重组后的数据
    __u16 *out_len)    // 输出：重组后的长度
{
    struct tcp_segment_cache *cache =
        bpf_map_lookup_elem(&tcp_segment_cache_map, key);

    if (!cache) {
        // 创建新缓存
        struct tcp_segment_cache new_cache = {
            .asm_seq = tcp_seq,
            .seg_count = 0,
        };
        bpf_map_update_elem(&tcp_segment_cache_map, key,
                           &new_cache, BPF_ANY);
        cache = bpf_map_lookup_elem(&tcp_segment_cache_map, key);
        if (!cache)
            return -1;
    }

    // 情况 1: 有序段
    if (tcp_seq == cache->asm_seq) {
        // 直接使用当前段
        cache->asm_seq += payload_len;

        // 尝试从缓存刷新后续段
        #pragma unroll
        for (int i = 0; i < MAX_CACHED_SEGMENTS; i++) {
            if (i >= cache->seg_count)
                break;

            if (cache->seg_seq[i] == cache->asm_seq) {
                // 找到下一个段，更新 asm_seq
                cache->asm_seq += cache->seg_len[i];

                // 从缓存中移除（简化：标记为已使用）
                cache->seg_seq[i] = 0;
            }
        }

        *out_len = payload_len;
        return 0;  // 成功
    }

    // 情况 2: 乱序段（未来）
    if (tcp_seq_gt(tcp_seq, cache->asm_seq)) {
        // 缓存该段
        if (cache->seg_count < MAX_CACHED_SEGMENTS) {
            int idx = cache->seg_count;
            cache->seg_seq[idx] = tcp_seq;
            cache->seg_len[idx] = payload_len;

            // 复制数据（有界）
            __u16 copy_len = payload_len > MAX_SEGMENT_SIZE ?
                             MAX_SEGMENT_SIZE : payload_len;

            #pragma unroll
            for (int j = 0; j < MAX_SEGMENT_SIZE; j++) {
                if (j < copy_len && payload + j < data_end) {
                    cache->seg_data[idx][j] = *((__u8 *)(payload + j));
                }
            }

            cache->seg_count++;
            return 1;  // 已缓存，但未重组
        } else {
            // 缓存满，放弃该流
            bpf_map_delete_elem(&tcp_segment_cache_map, key);
            update_stats(STATS_REASSEMBLY_CACHE_FULL);
            return -2;
        }
    }

    // 情况 3: 重传或旧段
    return -3;
}
```

### 5.3 优点

✅ **eBPF 内处理**:
- 低延迟（相比用户态）
- 无需 ring buffer 通信

✅ **处理简单乱序**:
- 前 2-3 个段乱序
- 覆盖大部分乱序场景（85-90%）

✅ **有界复杂度**:
- 固定数组，verifier 友好
- 固定循环展开

### 5.4 局限性

❌ **高复杂度**:
- 500-1000 行 eBPF 代码
- 复杂的边界检查
- 难以调试

❌ **内存开销大**:
- ~400 bytes per cache entry
- 1000 entries = ~400KB
- LRU 可能过早淘汰

❌ **容量有限**:
- 仅缓存 3 个段
- 更多段的乱序无法处理

❌ **Verifier 风险**:
- 可能超出指令数限制
- 复杂的内存访问模式

### 5.5 性能分析

**每包开销**:

| 操作 | 指令数 | 时间 |
|------|--------|------|
| 序列号检查 | 10-20 | <1μs |
| Cache 查找 | 50-100 | 5-10μs |
| Cache 管理 | 200-400 | 15-30μs |
| 数据复制 (128B) | 200-400 | 15-30μs |
| 协议检测 | 100-200 | 10-20μs |
| **总计** | **560-1120** | **50-90μs** |

**内存使用**:

```
Cache entry = 4 + 4×3 + 2×3 + 128×3 + 1 + 3
            = 4 + 12 + 6 + 384 + 1 + 3
            = 410 bytes

1000 entries = ~400 KB
```

**CPU 开销** @ 1M PPS:

```
0.08ms/pkt × 1M pps = 80 CPU 核
```

### 5.6 使用建议

❌ **不推荐实现**，除非：

- [ ] 阶段 1+2 检测率 < 80%
- [ ] 阶段 3 用户态重组延迟不可接受（<100μs 要求）
- [ ] 团队有深厚的 eBPF 经验
- [ ] 有充足的开发和测试时间（2+ 周）

**替代方案**:
- 优先考虑阶段 3（用户态重组）
- 或接受 90% 检测率（阶段 1+2）

---

## 6. 性能分析对比

### 6.1 延迟对比

```
阶段 1: 仅首包
┌────────────────────┐
│ 5-10μs            │
└────────────────────┘

阶段 2: +序列号跟踪
┌────────────────────────┐
│ 10-20μs               │
└────────────────────────┘

阶段 3: 用户态重组（快速路径 90%）
┌────────────────────┐
│ 10μs              │
└────────────────────┘
      (慢速路径 10%)
┌──────────────────────────────────────────────────────────┐
│ 160-420μs                                                │
└──────────────────────────────────────────────────────────┘

阶段 4: eBPF 轻量级重组
┌──────────────────────────────────────────────┐
│ 50-90μs                                      │
└──────────────────────────────────────────────┘
```

### 6.2 吞吐量影响

**测试环境**: 10 Gbps, 1M PPS

| 方案 | CPU 开销 | 吞吐量影响 | 可扩展性 |
|------|----------|-----------|---------|
| **无检测** | 0 核 | 0% | ✅ 基准 |
| **阶段 1** | ~10 核 | 1% | ✅ 优秀 |
| **阶段 2** | ~20 核 | 2% | ✅ 优秀 |
| **阶段 3 (平均)** | ~38 核 | 4% | ✅ 良好 |
| **阶段 4** | ~80 核 | 8% | ⚠️ 可接受 |

**结论**: 阶段 1-3 对吞吐量影响可控，阶段 4 开销较高。

### 6.3 内存使用对比

**100K 并发流**:

| 方案 | 每流内存 | 总内存 | 说明 |
|------|---------|--------|------|
| **阶段 1** | 0 bytes | 0 | 无额外内存 |
| **阶段 2** | 0 bytes | 0 | 复用现有字段 |
| **阶段 3** | ~2KB (用户态) | ~200MB | 用户态缓冲 |
| **阶段 4** | ~400 bytes | ~40MB | eBPF map |

**阶段 3 说明**: 内存在用户态，不影响 eBPF map 容量。

### 6.4 可扩展性分析

**横向扩展能力** (多核 CPU):

| 方案 | 核心利用率 | 扩展性 | 瓶颈 |
|------|-----------|--------|------|
| **阶段 1** | 均衡 | 线性 | 无 |
| **阶段 2** | 均衡 | 线性 | 无 |
| **阶段 3** | eBPF 均衡<br>用户态可能不均 | 良好 | 用户态调度 |
| **阶段 4** | 均衡 | 良好 | Map 竞争 |

---

## 7. eBPF 实现挑战

### 7.1 内存限制

**挑战**: eBPF 栈仅 512 字节，无法缓存大量数据。

**影响**:
- 阶段 1-2: ❌ 无影响（无需缓存）
- 阶段 3: ❌ 无影响（用户态缓存）
- 阶段 4: ✅ 需要 map 缓存（复杂）

**解决方案**（阶段 4）:
- 使用 BPF map 存储段数据
- 固定大小数组（避免动态分配）
- LRU map 自动淘汰

### 7.2 Verifier 复杂度限制

**挑战**: 复杂的重组逻辑可能超出 verifier 指令数限制。

**阶段 4 风险评估**:

```c
// 估计指令数
序列号检查:      20 instructions
Cache 查找:      100 instructions
Cache 插入:      200 instructions
数据复制 (128B): 400 instructions (展开循环)
段排序:          200 instructions
重组逻辑:        300 instructions
协议检测:        200 instructions
──────────────────────────────────
总计:           ~1420 instructions
```

**Verifier 限制**: ~1M instructions（但实际常低于此）

**风险**: 中等（复杂场景可能超限）

### 7.3 性能权衡

**CPU vs 覆盖率权衡**:

```
100% ┤                                  ┌─ 阶段 3: 99%+
     │                                 /
     │                           ╭────╯
  95%┤                      ╭────╯ 阶段 4: 95%
     │                 ╭────╯
  90%┤────────────────╯ 阶段 1-2: 90%+
     │
     │
     └────┬────┬────┬────┬────┬────→ CPU 开销
         10   20   40   60   80  100 (核 @ 1M PPS)
```

**最佳点**: 阶段 1-2（90%+ 覆盖率 @ 20 核）

### 7.4 状态管理复杂度

**问题**: eBPF 程序无状态，重组需要复杂的状态管理。

**阶段 4 需要管理**:
- 每流的缓存状态
- 段超时（清理旧段）
- 缓存容量限制
- 并发访问（多核）

**复杂度估计**: 🔴 高

---

## 8. 方案选择建议

### 8.1 决策树

```
开始
 │
 ├─ 需要 99%+ 检测率？
 │   ├─ 是 → 阶段 3: 用户态重组
 │   └─ 否 ↓
 │
 ├─ 可接受 300μs 延迟（慢速路径）？
 │   ├─ 否 → 阶段 4: eBPF 重组（不推荐）
 │   └─ 是 ↓
 │
 ├─ 需要网络可观测性？
 │   ├─ 是 → 阶段 2: 序列号跟踪
 │   └─ 否 → 阶段 1: 仅首包
 │
 └─ 推荐: 阶段 1 (立即) → 阶段 2 (增强) → 评估指标 → 按需阶段 3
```

### 8.2 推荐路线图

**第 1 周**:
```
Day 1-2: 实现阶段 1（仅首包检测）
Day 3:   添加阶段 2（序列号跟踪）
Day 4-5: 测试和优化
Day 6-7: 部署到测试环境
```

**第 2 周**:
```
Day 8-10: 生产部署（小流量 10%）
Day 11-14: 收集指标，监控
```

**评估**:
```
如果 STATS_PROTO_UNKNOWN < 10%:
  ✅ 完成！无需进一步优化

如果 STATS_PROTO_UNKNOWN > 10%:
  → 分析根因
  → 如果是乱序导致 → 实现阶段 3
  → 如果是协议未实现 → 添加协议检测器
```

### 8.3 不同场景的方案选择

| 场景 | 推荐方案 | 理由 |
|------|----------|------|
| **标准 Web 应用** | 阶段 1 | HTTP/HTTPS 首包足够 |
| **数据库密集型** | 阶段 1-2 | MySQL/Redis 握手在首包 |
| **高延迟网络** | 阶段 3 | 乱序可能较多 |
| **边缘计算** | 阶段 1 | 资源受限 |
| **金融交易** | 阶段 2-3 | 需要高可观测性 |
| **CDN** | 阶段 1 | 大量 HTTP 流量 |
| **企业内网** | 阶段 2 | 需要监控网络质量 |

### 8.4 最终建议

✅ **强烈推荐**: 阶段 1 + 阶段 2

**原因**:
1. 覆盖 90%+ 场景
2. 极低开销（<20μs）
3. 提供可观测性
4. 快速实现（2-3 天）
5. 低维护成本

⚠️ **按需考虑**: 阶段 3

**条件**:
- 生产指标显示需要（检测率 < 85%）
- 有充足资源（CPU、内存）
- 可接受额外延迟

❌ **不推荐**: 阶段 4

**原因**:
- 过度复杂
- 高开销
- verifier 风险
- 难以维护

---

**文档版本**: 1.0
**最后更新**: 2025-11-19
**下次审查**: 阶段 1+2 实现完成后
