# TCP 重组实现指南

## 文档概览

**目的**: 提供阶段 1 和阶段 2 的详细实现指南，包含完整代码示例和集成步骤。

**状态**: 实现指南
**日期**: 2025-11-19
**相关文档**:
- [TCP 重组必要性分析](./TCP_REASSEMBLY_ANALYSIS.md)
- [TCP 重组解决方案对比](./TCP_REASSEMBLY_SOLUTIONS.md)
- [应用层协议检测实现方案](./APPLICATION_LAYER_PROTOCOL_DETECTION.md)

---

## 目录

1. [实施准备](#1-实施准备)
2. [阶段 1 实现：仅首包检测](#2-阶段-1-实现仅首包检测)
3. [阶段 2 实现：序列号跟踪](#3-阶段-2-实现序列号跟踪)
4. [监控指标实现](#4-监控指标实现)
5. [测试方案](#5-测试方案)
6. [部署步骤](#6-部署步骤)
7. [故障排查](#7-故障排查)

---

## 1. 实施准备

### 1.1 前置条件

确认以下代码已就绪：

- ✅ `src/bpf/headers/common_types.h` - session_value 定义
- ✅ `src/bpf/headers/flow_processing.h` - 数据包解析
- ✅ `src/bpf/tc_microsegment.bpf.c` - TC 程序
- ✅ `src/bpf/xdp_microsegment.bpf.c` - XDP 程序
- ✅ Session map 和 Flow events ring buffer

### 1.2 开发环境设置

```bash
# 1. 确认 eBPF 开发环境
clang --version  # >= 10.0
llvm --version   # >= 10.0

# 2. 安装 libbpf 开发库
sudo apt-get install libbpf-dev

# 3. 验证 eBPF 程序可编译
cd src/bpf
make clean
make
```

### 1.3 文件清单

**需要创建/修改的文件**:

```
src/bpf/headers/
├── app_protocol_types.h          # 新建
├── app_protocol_detection.h      # 新建
├── app_protocol_http.h           # 新建
├── app_protocol_dns.h            # 新建
├── app_protocol_ssh.h            # 新建
├── app_protocol_mysql.h          # 新建
├── app_protocol_redis.h          # 新建
└── common_types.h                # 修改：添加协议字段

src/bpf/
├── tc_microsegment.bpf.c         # 修改：集成协议检测
└── xdp_microsegment.bpf.c        # 修改：集成协议检测

src/agent/pkg/protocol/
├── types.go                      # 新建
├── detector.go                   # 新建
└── stats.go                      # 新建
```

---

## 2. 阶段 1 实现：仅首包检测

### 2.1 步骤 1：扩展 session_value

**文件**: `src/bpf/headers/common_types.h`

```c
// 在 session_value 结构体中添加协议检测字段
struct session_value {
    // ===== 现有字段 =====
    __u64 created_ts;
    __u64 last_seen_ts;
    __u64 packets_to_server;
    __u64 packets_to_client;
    __u64 bytes_to_server;
    __u64 bytes_to_client;

    __u32 tcp_seq_client;
    __u32 tcp_seq_server;
    __u32 tcp_ack_client;
    __u32 tcp_ack_server;
    __u16 tcp_window_size;
    __u8  tcp_retrans_count;

    __u8  state;
    __u8  tcp_state;
    __u8  policy_action;
    __u8  flags;

    // ===== 新增：协议检测字段 =====
    __u8  app_protocol;          // 应用层协议 (enum app_protocol)
    __u8  proto_confidence;      // 检测置信度 0-100
    __u16 proto_flags;           // 协议特性标志 (PROTO_FLAG_*)
    __u32 proto_first_seen_ts;   // 首次检测时间（秒）
    __u32 proto_payload_bytes;   // 已检查的 payload 字节数

    __u8  pad[2];                // 对齐填充
} __attribute__((packed));
```

### 2.2 步骤 2：定义协议枚举和标志

**文件**: `src/bpf/headers/app_protocol_types.h` (新建)

```c
#ifndef __APP_PROTOCOL_TYPES_H__
#define __APP_PROTOCOL_TYPES_H__

// 应用层协议标识符
enum app_protocol {
    APP_PROTO_UNKNOWN = 0,

    // Web 协议
    APP_PROTO_HTTP = 1,
    APP_PROTO_HTTPS = 2,

    // 基础设施
    APP_PROTO_DNS = 3,
    APP_PROTO_SSH = 4,

    // 数据库
    APP_PROTO_MYSQL = 5,
    APP_PROTO_POSTGRESQL = 6,
    APP_PROTO_REDIS = 7,
    APP_PROTO_MONGODB = 8,

    // 消息队列
    APP_PROTO_KAFKA = 9,
    APP_PROTO_RABBITMQ = 10,

    // RPC
    APP_PROTO_GRPC = 11,

    // 其他
    APP_PROTO_FTP = 12,
    APP_PROTO_SMTP = 13,

    APP_PROTO_MAX = 100,
};

// 协议特性标志
#define PROTO_FLAG_ENCRYPTED     0x0001  // 加密流量
#define PROTO_FLAG_CLEARTEXT     0x0002  // 明文流量
#define PROTO_FLAG_BINARY        0x0004  // 二进制协议
#define PROTO_FLAG_TEXT          0x0008  // 文本协议
#define PROTO_FLAG_REQUEST       0x0010  // 请求方向
#define PROTO_FLAG_RESPONSE      0x0020  // 响应方向
#define PROTO_FLAG_OUT_OF_ORDER  0x0040  // 检测到乱序
#define PROTO_FLAG_TUNNEL        0x0080  // 潜在隧道

// 协议检测配置
struct proto_detect_config {
    __u8  enabled;               // 全局启用/禁用
    __u8  sampling_interval;     // 0 = 每个包, N = 每 N 个包
    __u16 max_payload_bytes;     // 最多检查的字节数 (默认: 128)
    __u8  confidence_threshold;  // 最低置信度报告 (0-100, 默认: 70)
    __u8  reserved[3];
} __attribute__((packed));

#endif // __APP_PROTOCOL_TYPES_H__
```

### 2.3 步骤 3：实现核心检测框架

**文件**: `src/bpf/headers/app_protocol_detection.h` (新建)

```c
#ifndef __APP_PROTOCOL_DETECTION_H__
#define __APP_PROTOCOL_DETECTION_H__

#include "app_protocol_types.h"
#include "app_protocol_http.h"
#include "app_protocol_dns.h"
#include "app_protocol_ssh.h"
#include "app_protocol_mysql.h"
#include "app_protocol_redis.h"

// 端口启发式检测
static __always_inline __u8 guess_protocol_by_port(__u16 port, __u8 l4_proto)
{
    __u16 port_h = bpf_ntohs(port);

    if (l4_proto == IPPROTO_TCP) {
        switch (port_h) {
        case 80:
        case 8080:
        case 8000:
            return APP_PROTO_HTTP;
        case 443:
        case 8443:
            return APP_PROTO_HTTPS;
        case 22:
            return APP_PROTO_SSH;
        case 3306:
            return APP_PROTO_MYSQL;
        case 5432:
            return APP_PROTO_POSTGRESQL;
        case 6379:
            return APP_PROTO_REDIS;
        case 27017:
            return APP_PROTO_MONGODB;
        case 9092:
            return APP_PROTO_KAFKA;
        default:
            return APP_PROTO_UNKNOWN;
        }
    } else if (l4_proto == IPPROTO_UDP) {
        if (port_h == 53)
            return APP_PROTO_DNS;
    }

    return APP_PROTO_UNKNOWN;
}

// 主协议检测入口
static __always_inline int detect_app_protocol_first_packet(
    void *payload_start,
    void *data_end,
    struct flow_key *key,
    struct session_value *session,
    struct proto_detect_config *config)
{
    // 检查配置
    if (!config || !config->enabled)
        return 0;

    // 跳过已高置信度检测的会话
    if (session->proto_confidence >= 90)
        return 0;

    // 仅在前几个包检测
    __u64 total_packets = session->packets_to_server + session->packets_to_client;
    if (total_packets > 5)
        return 0;

    // 检查 payload 是否存在
    if (!payload_start || payload_start >= data_end)
        return 0;

    __u32 payload_len = data_end - payload_start;
    if (payload_len == 0)
        return 0;

    // 阶段 1: 端口启发式（首次）
    if (session->app_protocol == APP_PROTO_UNKNOWN) {
        __u8 guessed = guess_protocol_by_port(key->dst_port, key->protocol);
        if (guessed != APP_PROTO_UNKNOWN) {
            session->app_protocol = guessed;
            session->proto_confidence = 60;  // 低置信度
            session->proto_first_seen_ts = bpf_ktime_get_ns() / 1000000000;
        }
    }

    // 限制检查长度
    __u32 inspect_len = payload_len;
    if (config->max_payload_bytes > 0 && inspect_len > config->max_payload_bytes)
        inspect_len = config->max_payload_bytes;

    session->proto_payload_bytes += inspect_len;

    __u8 detected_proto = APP_PROTO_UNKNOWN;
    __u8 confidence = 0;
    __u16 flags = 0;

    // 阶段 2: Payload 特征匹配
    if (key->protocol == IPPROTO_TCP) {
        // 尝试 HTTP 检测
        if (detect_http(payload_start, data_end, inspect_len, &confidence, &flags)) {
            detected_proto = APP_PROTO_HTTP;
        }
        // 尝试 HTTPS/TLS 检测
        else if (detect_tls(payload_start, data_end, inspect_len, &confidence, &flags)) {
            detected_proto = APP_PROTO_HTTPS;
        }
        // 尝试 SSH 检测
        else if (detect_ssh(payload_start, data_end, inspect_len, &confidence, &flags)) {
            detected_proto = APP_PROTO_SSH;
        }
        // 尝试 MySQL 检测
        else if (detect_mysql(payload_start, data_end, inspect_len, &confidence, &flags)) {
            detected_proto = APP_PROTO_MYSQL;
        }
        // 尝试 Redis 检测
        else if (detect_redis(payload_start, data_end, inspect_len, &confidence, &flags)) {
            detected_proto = APP_PROTO_REDIS;
        }
    }
    else if (key->protocol == IPPROTO_UDP) {
        // 尝试 DNS 检测
        if (detect_dns(payload_start, data_end, inspect_len, &confidence, &flags)) {
            detected_proto = APP_PROTO_DNS;
        }
    }

    // 更新会话（如果检测成功且置信度更高）
    if (detected_proto != APP_PROTO_UNKNOWN && confidence > session->proto_confidence) {
        session->app_protocol = detected_proto;
        session->proto_confidence = confidence;
        session->proto_flags = flags;

        if (session->proto_first_seen_ts == 0)
            session->proto_first_seen_ts = bpf_ktime_get_ns() / 1000000000;
    }

    return 0;
}

// 辅助函数：获取 TCP payload 起始位置
static __always_inline void *get_tcp_payload_start(
    struct tcphdr *tcph,
    void *data_end)
{
    if ((void *)(tcph + 1) > data_end)
        return NULL;

    __u8 tcp_hdr_len = tcph->doff * 4;
    void *payload = (void *)tcph + tcp_hdr_len;

    if (payload >= data_end)
        return NULL;

    return payload;
}

// 辅助函数：获取 UDP payload 起始位置
static __always_inline void *get_udp_payload_start(
    struct udphdr *udph,
    void *data_end)
{
    if ((void *)(udph + 1) > data_end)
        return NULL;

    return (void *)(udph + 1);
}

#endif // __APP_PROTOCOL_DETECTION_H__
```

### 2.4 步骤 4：实现 HTTP 检测器

**文件**: `src/bpf/headers/app_protocol_http.h` (新建)

```c
#ifndef __APP_PROTOCOL_HTTP_H__
#define __APP_PROTOCOL_HTTP_H__

// HTTP 方法检测
static __always_inline bool detect_http(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    // 需要至少 4 字节
    if (payload_start + 4 > data_end)
        return false;

    char *data = (char *)payload_start;
    bool is_http = false;

    // 检查 HTTP 方法
    // GET
    if (data[0] == 'G' && data[1] == 'E' && data[2] == 'T' && data[3] == ' ') {
        is_http = true;
        *confidence = 95;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_REQUEST;
    }
    // POST (需要 5 字节)
    else if (payload_start + 5 <= data_end &&
             data[0] == 'P' && data[1] == 'O' && data[2] == 'S' &&
             data[3] == 'T' && data[4] == ' ') {
        is_http = true;
        *confidence = 95;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_REQUEST;
    }
    // PUT
    else if (data[0] == 'P' && data[1] == 'U' && data[2] == 'T' && data[3] == ' ') {
        is_http = true;
        *confidence = 95;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_REQUEST;
    }
    // HEAD
    else if (payload_start + 5 <= data_end &&
             data[0] == 'H' && data[1] == 'E' && data[2] == 'A' &&
             data[3] == 'D' && data[4] == ' ') {
        is_http = true;
        *confidence = 95;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_REQUEST;
    }
    // DELETE
    else if (payload_start + 7 <= data_end &&
             data[0] == 'D' && data[1] == 'E' && data[2] == 'L' &&
             data[3] == 'E' && data[4] == 'T' && data[5] == 'E' && data[6] == ' ') {
        is_http = true;
        *confidence = 95;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_REQUEST;
    }
    // HTTP 响应 (需要 8 字节)
    else if (payload_start + 8 <= data_end &&
             data[0] == 'H' && data[1] == 'T' && data[2] == 'T' && data[3] == 'P' &&
             data[4] == '/' && data[5] == '1' && data[6] == '.') {
        is_http = true;
        *confidence = 90;
        *flags = PROTO_FLAG_CLEARTEXT | PROTO_FLAG_TEXT | PROTO_FLAG_RESPONSE;
    }

    return is_http;
}

// TLS/SSL 检测
static __always_inline bool detect_tls(
    void *payload_start,
    void *data_end,
    __u32 inspect_len,
    __u8 *confidence,
    __u16 *flags)
{
    // 需要至少 5 字节用于 TLS 记录头
    if (payload_start + 5 > data_end)
        return false;

    __u8 *data = (__u8 *)payload_start;

    // TLS 记录头:
    // - ContentType (1 字节): 0x16=Handshake, 0x17=AppData
    // - Version (2 字节): 0x0301=TLS1.0, 0x0303=TLS1.2
    // - Length (2 字节)

    __u8 content_type = data[0];
    __u8 version_major = data[1];
    __u8 version_minor = data[2];

    // 检查内容类型
    bool valid_ct = (content_type == 0x14 ||  // ChangeCipherSpec
                     content_type == 0x15 ||  // Alert
                     content_type == 0x16 ||  // Handshake
                     content_type == 0x17);   // Application Data

    // 检查版本 (TLS 1.0 - 1.3)
    bool valid_ver = (version_major == 0x03 &&
                      version_minor >= 0x00 && version_minor <= 0x04);

    if (valid_ct && valid_ver) {
        *confidence = 98;
        *flags = PROTO_FLAG_ENCRYPTED | PROTO_FLAG_BINARY;

        // 区分请求/响应（通过握手类型）
        if (content_type == 0x16 && payload_start + 6 <= data_end) {
            __u8 handshake_type = data[5];
            if (handshake_type == 0x01)  // ClientHello
                *flags |= PROTO_FLAG_REQUEST;
            else if (handshake_type == 0x02)  // ServerHello
                *flags |= PROTO_FLAG_RESPONSE;
        }

        return true;
    }

    return false;
}

#endif // __APP_PROTOCOL_HTTP_H__
```

*（其他协议检测器如 DNS、SSH、MySQL、Redis 的实现类似，见 APPLICATION_LAYER_PROTOCOL_DETECTION.md）*

### 2.5 步骤 5：集成到 TC 程序

**文件**: `src/bpf/tc_microsegment.bpf.c`

```c
// 在文件开头添加
#include "headers/app_protocol_types.h"
#include "headers/app_protocol_detection.h"

// 添加协议检测配置 map
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct proto_detect_config);
    __uint(pinning, LIBBPF_PIN_BY_NAME);
} proto_detect_config_map SEC(".maps");

// 添加协议统计 map
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 128);
    __type(key, __u32);
    __type(value, __u64);
    __uint(pinning, LIBBPF_PIN_BY_NAME);
} proto_stats_map SEC(".maps");

// 在主 TC handler 中（在会话创建/更新后）
SEC("tc/microsegment")
int tc_microsegment_handler(struct __sk_buff *skb)
{
    // ... 现有的 L2/L3/L4 解析 ...

    // ... 现有的 session 查找/创建 ...

    // ===== 新增：协议检测 =====
    if (session && session->state == SESSION_STATE_ACTIVE) {
        // 加载协议检测配置
        __u32 config_key = 0;
        struct proto_detect_config *proto_config =
            bpf_map_lookup_elem(&proto_detect_config_map, &config_key);

        // 获取 payload 起始位置
        void *payload = NULL;
        if (key.protocol == IPPROTO_TCP) {
            struct tcphdr *tcph = l4_header;
            payload = get_tcp_payload_start(tcph, data_end);
        } else if (key.protocol == IPPROTO_UDP) {
            struct udphdr *udph = l4_header;
            payload = get_udp_payload_start(udph, data_end);
        }

        // 执行协议检测
        detect_app_protocol_first_packet(payload, data_end, &key, session, proto_config);

        // 更新协议统计
        if (session->app_protocol != APP_PROTO_UNKNOWN) {
            __u32 proto_key = session->app_protocol;
            __u64 *proto_count = bpf_map_lookup_elem(&proto_stats_map, &proto_key);
            if (proto_count) {
                __sync_fetch_and_add(proto_count, 1);
            }
        }
    }

    // ... 现有的策略匹配 ...
    // ... 现有的动作执行 ...

    return TC_ACT_OK;
}
```

---

## 3. 阶段 2 实现：序列号跟踪

### 3.1 步骤 1：添加序列号比较函数

**文件**: `src/bpf/headers/app_protocol_detection.h`（在现有基础上添加）

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

static __always_inline bool tcp_seq_gt(__u32 a, __u32 b)
{
    return ((__s32)(a - b)) > 0;
}
```

### 3.2 步骤 2：实现序列号验证

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

    // 选择方向的序列号字段
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

### 3.3 步骤 3：增强协议检测

```c
// 带序列号检查的协议检测
static __always_inline int detect_app_protocol_with_seq_check(
    void *payload_start,
    void *data_end,
    struct flow_key *key,
    struct session_value *session,
    struct proto_detect_config *config,
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

    // 序列号验证（仅 TCP）
    if (key->protocol == IPPROTO_TCP) {
        if (!tcp_seq_is_in_order(session, tcp_seq, payload_len, is_client_to_server)) {
            // 乱序或重传，跳过检测
            update_stats(STATS_PROTO_SKIP_OUT_OF_ORDER);
            return 0;
        }
    }

    // 执行协议检测（与阶段 1 相同）
    return detect_app_protocol_first_packet(
        payload_start, data_end, key, session, config);
}
```

### 3.4 步骤 4：集成到 TC 程序

**文件**: `src/bpf/tc_microsegment.bpf.c`（更新协议检测调用）

```c
// 在 TC handler 中
if (session && session->state == SESSION_STATE_ACTIVE) {
    __u32 config_key = 0;
    struct proto_detect_config *proto_config =
        bpf_map_lookup_elem(&proto_detect_config_map, &config_key);

    void *payload = NULL;
    __u32 tcp_seq = 0;

    if (key.protocol == IPPROTO_TCP) {
        struct tcphdr *tcph = l4_header;
        payload = get_tcp_payload_start(tcph, data_end);
        tcp_seq = bpf_ntohl(tcph->seq);  // 提取序列号
    } else if (key.protocol == IPPROTO_UDP) {
        struct udphdr *udph = l4_header;
        payload = get_udp_payload_start(udph, data_end);
    }

    // 判断方向（根据现有逻辑）
    bool is_client_to_server = /* 根据 flow direction 判断 */;

    // 调用带序列号检查的检测
    detect_app_protocol_with_seq_check(
        payload, data_end, &key, session, proto_config,
        tcp_seq, is_client_to_server);

    // 更新统计
    if (session->app_protocol != APP_PROTO_UNKNOWN) {
        __u32 proto_key = session->app_protocol;
        __u64 *proto_count = bpf_map_lookup_elem(&proto_stats_map, &proto_key);
        if (proto_count) {
            __sync_fetch_and_add(proto_count, 1);
        }
    }
}
```

---

## 4. 监控指标实现

### 4.1 步骤 1：定义统计键

**文件**: `src/bpf/headers/common_types.h`（添加到现有 stats_key）

```c
enum stats_key {
    // ... 现有统计 ...

    // 协议检测统计（新增）
    STATS_PROTO_DETECTED = 20,         // 成功检测的流
    STATS_PROTO_UNKNOWN,               // 未检测的流
    STATS_PROTO_FIRST_PKT_TOO_SMALL,   // 首包太小
    STATS_PROTO_SPLIT_HEADER,          // 疑似分割的头部
    STATS_PROTO_SKIP_OUT_OF_ORDER,     // 因乱序跳过的检测

    // TCP 行为统计（新增）
    STATS_TCP_OUT_OF_ORDER = 30,      // 乱序数据包
    STATS_TCP_RETRANS,                 // 重传
    STATS_TCP_SEQ_GAP,                 // 序列号间隙

    // 每协议计数（新增）
    STATS_PROTO_HTTP = 40,             // HTTP 流
    STATS_PROTO_HTTPS,                 // HTTPS 流
    STATS_PROTO_DNS,                   // DNS 流
    STATS_PROTO_SSH,                   // SSH 流
    STATS_PROTO_MYSQL,                 // MySQL 流
    STATS_PROTO_REDIS,                 // Redis 流
    STATS_PROTO_POSTGRESQL,            // PostgreSQL 流
    STATS_PROTO_MONGODB,               // MongoDB 流

    STATS_MAX = 100,
};
```

### 4.2 步骤 2：用户态统计收集

**文件**: `src/agent/pkg/protocol/stats.go`（新建）

```go
package protocol

import (
    "fmt"
    "github.com/cilium/ebpf"
)

type ProtocolStats struct {
    // 检测统计
    TotalFlows       uint64
    DetectedFlows    uint64
    UnknownFlows     uint64
    SkippedOOO       uint64

    // TCP 行为
    OutOfOrderPkts   uint64
    RetransPkts      uint64

    // 每协议计数
    ProtocolCounts   map[string]uint64
}

func CollectProtocolStats(statsMap *ebpf.Map) (*ProtocolStats, error) {
    stats := &ProtocolStats{
        ProtocolCounts: make(map[string]uint64),
    }

    // 读取检测统计
    stats.DetectedFlows = readStat(statsMap, STATS_PROTO_DETECTED)
    stats.UnknownFlows = readStat(statsMap, STATS_PROTO_UNKNOWN)
    stats.SkippedOOO = readStat(statsMap, STATS_PROTO_SKIP_OUT_OF_ORDER)
    stats.TotalFlows = stats.DetectedFlows + stats.UnknownFlows

    // 读取 TCP 行为
    stats.OutOfOrderPkts = readStat(statsMap, STATS_TCP_OUT_OF_ORDER)
    stats.RetransPkts = readStat(statsMap, STATS_TCP_RETRANS)

    // 读取每协议计数
    protocols := []struct {
        key  uint32
        name string
    }{
        {STATS_PROTO_HTTP, "HTTP"},
        {STATS_PROTO_HTTPS, "HTTPS"},
        {STATS_PROTO_DNS, "DNS"},
        {STATS_PROTO_SSH, "SSH"},
        {STATS_PROTO_MYSQL, "MySQL"},
        {STATS_PROTO_REDIS, "Redis"},
    }

    for _, p := range protocols {
        count := readStat(statsMap, p.key)
        if count > 0 {
            stats.ProtocolCounts[p.name] = count
        }
    }

    return stats, nil
}

func readStat(statsMap *ebpf.Map, key uint32) uint64 {
    var values []uint64
    if err := statsMap.Lookup(&key, &values); err != nil {
        return 0
    }

    // 聚合 per-CPU 值
    total := uint64(0)
    for _, v := range values {
        total += v
    }
    return total
}

func (s *ProtocolStats) Print() {
    fmt.Println("=== 协议检测统计 ===")
    fmt.Printf("总流量:       %d\n", s.TotalFlows)
    if s.TotalFlows > 0 {
        fmt.Printf("已检测:       %d (%.1f%%)\n",
                   s.DetectedFlows,
                   float64(s.DetectedFlows)/float64(s.TotalFlows)*100)
        fmt.Printf("未检测:       %d (%.1f%%)\n",
                   s.UnknownFlows,
                   float64(s.UnknownFlows)/float64(s.TotalFlows)*100)
    }
    fmt.Printf("乱序跳过:     %d\n", s.SkippedOOO)

    fmt.Println("\n=== TCP 行为 ===")
    fmt.Printf("乱序包:       %d\n", s.OutOfOrderPkts)
    fmt.Printf("重传包:       %d\n", s.RetransPkts)

    fmt.Println("\n=== 协议分布 ===")
    for proto, count := range s.ProtocolCounts {
        if s.TotalFlows > 0 {
            fmt.Printf("%-12s: %d (%.1f%%)\n",
                       proto, count,
                       float64(count)/float64(s.TotalFlows)*100)
        }
    }
}
```

---

## 5. 测试方案

### 5.1 单元测试

**测试 HTTP 检测**:

```bash
# 创建测试 PCAP
echo -ne 'GET / HTTP/1.1\r\nHost: example.com\r\n\r\n' | \
    nc -l 8080 > /dev/null &

curl http://localhost:8080/

# 验证检测
sudo bpftool map dump name session_map | grep app_protocol
```

### 5.2 集成测试

**文件**: `test/integration/protocol_detection_test.go`

```go
func TestHTTPDetection(t *testing.T) {
    // 启动 Agent
    agent := setupTestAgent(t)
    defer agent.Close()

    // 生成 HTTP 流量
    resp, err := http.Get("http://test-server:80/api/test")
    require.NoError(t, err)
    defer resp.Body.Close()

    // 等待检测
    time.Sleep(100 * time.Millisecond)

    // 查询会话
    sessions := agent.GetSessions()

    // 验证检测结果
    found := false
    for _, sess := range sessions {
        if sess.DstPort == 80 {
            assert.Equal(t, protocol.AppProtoHTTP, sess.AppProtocol)
            assert.GreaterOrEqual(t, sess.ProtoConfidence, uint8(90))
            found = true
        }
    }
    assert.True(t, found, "HTTP session not detected")
}
```

### 5.3 性能测试

```bash
# 使用 tcpreplay 回放流量
sudo tcpreplay -i eth0 -M 10000 test/pcaps/http-traffic.pcap

# 监控性能
sudo bpftool prog profile id <prog_id> duration 10

# 检查延迟
sudo bpftool prog dump xlated id <prog_id>
```

---

## 6. 部署步骤

### 6.1 编译

```bash
cd src/bpf
make clean
make

# 验证
ls -l tc_microsegment.bpf.o xdp_microsegment.bpf.o
```

### 6.2 部署到测试环境

```bash
# 1. 停止现有 Agent
sudo systemctl stop microsegment-agent

# 2. 备份现有程序
sudo cp /opt/microsegment/tc_microsegment.bpf.o \
        /opt/microsegment/tc_microsegment.bpf.o.backup

# 3. 部署新程序
sudo cp src/bpf/tc_microsegment.bpf.o /opt/microsegment/
sudo cp src/bpf/xdp_microsegment.bpf.o /opt/microsegment/

# 4. 初始化配置
cat > /tmp/proto_config.json <<EOF
{
  "enabled": true,
  "sampling_interval": 1,
  "max_payload_bytes": 128,
  "confidence_threshold": 70
}
EOF

# 5. 启动 Agent
sudo systemctl start microsegment-agent

# 6. 验证加载
sudo bpftool prog list | grep microsegment
sudo bpftool map list | grep proto
```

### 6.3 监控

```bash
# 实时监控统计
watch -n 1 'sudo bpftool map dump name proto_stats_map'

# 查看日志
sudo journalctl -u microsegment-agent -f | grep protocol
```

---

## 7. 故障排查

### 7.1 常见问题

**问题 1: 无法检测协议**

```bash
# 检查配置
sudo bpftool map dump name proto_detect_config_map

# 检查 payload
sudo tcpdump -i eth0 -X -s 200 'tcp port 80'

# 验证 session
sudo bpftool map dump name session_map | grep app_protocol
```

**问题 2: Verifier 错误**

```bash
# 查看详细错误
sudo bpftool prog load tc_microsegment.bpf.o \
    /sys/fs/bpf/tc_microsegment 2>&1 | less

# 简化检测逻辑或使用 tail calls
```

**问题 3: 性能下降**

```bash
# 检查 CPU 使用率
top -H -p $(pgrep microsegment-agent)

# 分析热点
sudo perf record -ag -- sleep 10
sudo perf report
```

### 7.2 调试技巧

**启用 debug 日志**:

```c
// 在 eBPF 代码中
#define DEBUG 1

#if DEBUG
#define debug_log(fmt, ...) \
    bpf_printk(fmt, ##__VA_ARGS__)
#else
#define debug_log(fmt, ...)
#endif

// 使用
debug_log("Detected protocol: %d, confidence: %d\n",
          session->app_protocol, session->proto_confidence);
```

**查看日志**:

```bash
sudo cat /sys/kernel/debug/tracing/trace_pipe | grep bpf_trace_printk
```

---

**文档版本**: 1.0
**最后更新**: 2025-11-19
**下次审查**: 实现完成后
