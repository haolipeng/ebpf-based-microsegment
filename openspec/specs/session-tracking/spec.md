# Capability: 会话跟踪

## Purpose

会话跟踪能力使用 eBPF 技术为网络流提供有状态的连接跟踪。它使用 LRU（最近最少使用）哈希映射在内核空间维护会话状态，实现新连接与现有连接的高效识别，并支持会话感知的策略执行。

## Context

会话跟踪是微隔离的基础，因为它允许系统：
- 区分新连接和已建立的连接
- 跟踪连接生命周期（创建、活动、关闭）
- 维护每个会话的统计信息（数据包、字节）
- 支持 TCP 状态机跟踪
- 启用有状态防火墙功能

## Requirements

### Requirement: 会话标识

系统必须(SHALL)使用由以下组成的 5 元组键标识唯一的网络流：
- 源 IP 地址（IPv4）
- 目标 IP 地址（IPv4）
- 源端口
- 目标端口
- 协议（TCP、UDP、ICMP 等）

#### Scenario: TCP 连接跟踪

**Given** 来自 192.168.1.10:45678 到 10.0.0.5:80 的 TCP 数据包到达
**When** eBPF 程序提取流密钥
**Then** 流密钥必须(SHALL)包含 src_ip=192.168.1.10、dst_ip=10.0.0.5、src_port=45678、dst_port=80、protocol=6

#### Scenario: UDP 流跟踪

**Given** 来自 172.16.0.20:53123 到 8.8.8.8:53 的 UDP 数据包到达
**When** eBPF 程序提取流密钥
**Then** 流密钥必须(SHALL)包含 src_ip=172.16.0.20、dst_ip=8.8.8.8、src_port=53123、dst_port=53、protocol=17

### Requirement: 会话存储

系统必须(SHALL)在具有以下特征的 eBPF LRU_HASH 映射中存储会话信息：
- 最大条目数：100,000 个会话
- 满时自动驱逐最近最少使用的条目
- O(1) 查找和更新操作

#### Scenario: 新会话创建

**Given** 会话映射中不存在 5 元组的数据包到达
**When** eBPF 程序创建新的会话条目
**Then** 会话条目必须(SHALL)存储在 LRU_HASH 映射中，并带有初始状态和时间戳

#### Scenario: 负载下的会话驱逐

**Given** 会话映射包含 100,000 个条目（满容量）
**And** 新流到达
**When** 系统尝试创建新会话
**Then** 最近最少使用的会话必须(SHALL)被自动驱逐
**And** 新会话必须(SHALL)成功存储

### Requirement: 会话状态跟踪

系统必须(SHALL)为每个会话维护以下状态信息：
- 创建时间戳（纳秒）
- 最后一次看到的时间戳（纳秒）
- 数据包计数器（到服务器、到客户端）
- 字节计数器（到服务器、到客户端）
- 会话状态（NEW、ESTABLISHED、CLOSING、CLOSED）
- TCP 状态机状态（如果是 TCP 协议）
- 匹配的策略操作（ALLOW、DENY、LOG）
- 会话标志

#### Scenario: 双向流量计费

**Given** 客户端和服务器之间建立的会话
**When** 来自客户端到服务器的数据包到达（100 字节）
**Then** 会话的 packets_to_server 计数器必须(SHALL)增加 1
**And** 会话的 bytes_to_server 计数器必须(SHALL)增加 100
**And** last_seen_ts 必须(SHALL)更新为当前时间

**When** 来自服务器到客户端的数据包到达（200 字节）
**Then** 会话的 packets_to_client 计数器必须(SHALL)增加 1
**And** 会话的 bytes_to_client 计数器必须(SHALL)增加 200
**And** last_seen_ts 必须(SHALL)更新为当前时间

#### Scenario: 会话状态转换

**Given** 建立新的 TCP 连接
**When** 第一个 SYN 数据包到达
**Then** 会话状态必须(SHALL)为 SESSION_STATE_NEW
**And** tcp_state 必须(SHALL)为 TCP_STATE_SYN_SENT

**When** 后续数据包确认连接
**Then** 会话状态必须(SHALL)转换为 SESSION_STATE_ESTABLISHED
**And** tcp_state 必须(SHALL)转换为 TCP_STATE_ESTABLISHED

### Requirement: TCP 状态机

对于 TCP 连接，系统必须(SHALL)跟踪具有以下状态的 TCP 状态机：
- TCP_STATE_CLOSED
- TCP_STATE_SYN_SENT
- TCP_STATE_SYN_RECV
- TCP_STATE_ESTABLISHED
- TCP_STATE_FIN_WAIT
- TCP_STATE_CLOSE_WAIT
- TCP_STATE_CLOSING
- TCP_STATE_TIME_WAIT

#### Scenario: TCP 三次握手

**Given** 客户端发起 TCP 连接
**When** SYN 数据包到达
**Then** tcp_state 必须(SHALL)为 TCP_STATE_SYN_SENT

**When** SYN-ACK 数据包到达
**Then** tcp_state 必须(SHALL)为 TCP_STATE_SYN_RECV

**When** ACK 数据包完成握手
**Then** tcp_state 必须(SHALL)为 TCP_STATE_ESTABLISHED

#### Scenario: TCP 连接终止

**Given** 已建立的 TCP 连接
**When** 检测到 FIN 数据包
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_FIN_WAIT

**When** 双方完成 FIN 握手
**Then** tcp_state 必须(SHALL)转换为 TCP_STATE_TIME_WAIT

### Requirement: 会话查找性能

系统必须(SHALL)提供高性能的会话查找：
- 查找延迟：< 1 微秒（平均）
- 支持并发查找（无锁读取）
- 最小的 CPU 开销

#### Scenario: 热路径性能

**Given** 具有高流量的已建立会话（10,000 pps）
**When** 此会话的后续数据包到达
**Then** 会话查找必须(SHALL)在 < 1 微秒内完成
**And** 不得(SHALL NOT)由于会话查找开销而发生数据包丢弃

### Requirement: 会话时间戳管理

系统必须(SHALL)维护准确的时间戳信息：
- 使用单调时钟（bpf_ktime_get_ns()）
- 在会话建立时记录创建时间
- 在每个数据包上更新 last_seen 时间

#### Scenario: 会话年龄计算

**Given** 会话在时间戳 T0 创建
**And** 当前时间为 T1
**When** 用户空间查询会话年龄
**Then** 年龄必须(SHALL)计算为（T1 - created_ts）

#### Scenario: 空闲会话检测

**Given** 会话的 last_seen_ts = T0
**And** 当前时间为 T1
**And**（T1 - T0）> idle_timeout（例如，5 分钟）
**When** 系统检查空闲会话
**Then** 会话应当(SHOULD)标记为适合清理

### Requirement: 方向感知

系统必须(SHALL)分别跟踪每个方向的数据包和字节计数器：
- 客户端到服务器方向
- 服务器到客户端方向

这使得能够：
- 检测非对称流量
- 按方向分析带宽使用
- 基于流量模式进行异常检测

#### Scenario: 非对称流量检测

**Given** 会话的 packets_to_server = 1000 和 packets_to_client = 10
**When** 用户空间代理分析会话
**Then** 它必须(SHALL)将其识别为非对称流量
**And** 它可以(MAY)触发潜在异常的警报

## Data Structures

### 流密钥结构

```c
struct flow_key {
    __u32 src_ip;      // 源 IP 地址
    __u32 dst_ip;      // 目标 IP 地址
    __u16 src_port;    // 源端口
    __u16 dst_port;    // 目标端口
    __u8  protocol;    // 协议（TCP=6、UDP=17、ICMP=1 等）
    __u8  pad[3];      // 对齐填充
} __attribute__((packed));
```

### 会话值结构

```c
struct session_value {
    __u64 created_ts;         // 会话创建时间戳（ns）
    __u64 last_seen_ts;       // 最后一个数据包时间戳（ns）
    __u64 packets_to_server;  // 从客户端到服务器的数据包
    __u64 packets_to_client;  // 从服务器到客户端的数据包
    __u64 bytes_to_server;    // 从客户端到服务器的字节
    __u64 bytes_to_client;    // 从服务器到客户端的字节
    __u8  state;              // 会话状态（enum session_state）
    __u8  tcp_state;          // TCP 状态机（enum tcp_state）
    __u8  policy_action;      // 匹配的策略操作
    __u8  flags;              // 会话标志
    __u32 pad;                // 填充
};
```

### 会话状态枚举

```c
enum session_state {
    SESSION_STATE_NEW = 0,
    SESSION_STATE_ESTABLISHED,
    SESSION_STATE_CLOSING,
    SESSION_STATE_CLOSED,
};
```

### TCP 状态枚举

```c
enum tcp_state {
    TCP_STATE_CLOSED = 0,
    TCP_STATE_SYN_SENT,
    TCP_STATE_SYN_RECV,
    TCP_STATE_ESTABLISHED,
    TCP_STATE_FIN_WAIT,
    TCP_STATE_CLOSE_WAIT,
    TCP_STATE_CLOSING,
    TCP_STATE_TIME_WAIT,
};
```

## Implementation Notes

- 会话映射实现为 `BPF_MAP_TYPE_LRU_HASH` 以实现自动内存管理
- 最大容量设置为 100,000 个会话（可通过 `MAX_ENTRIES_SESSION` 配置）
- 时间戳使用来自内核单调时钟的纳秒精度
- 当前实现专注于 IPv4；IPv6 支持计划在未来版本中实现
- ICMP 和其他没有端口号的协议使用 src_port=0 和 dst_port=0

## Performance Characteristics

- **查找延迟**：< 1 微秒（平均）
- **每个会话的内存**：64 字节
- **总内存**：~6.4 MB（100,000 个会话）
- **并发访问**：读取无锁，更新每个条目锁定
- **驱逐开销**：由于 LRU 实现为 O(1)

## Related Capabilities

- **Policy Matching**：使用会话信息来缓存策略决策
- **Policy Enforcement**：应用会话状态中的缓存操作
- **Statistics Reporting**：聚合会话级统计信息
- **Dataplane Performance**：会话缓存对热路径优化至关重要

