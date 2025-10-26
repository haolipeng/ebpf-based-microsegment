# 第4周：高级功能实现

**[⬅️ 第3周](./week3-userspace-control.md)** | **[📚 目录](./README.md)** | **[➡️ 第5周](./week5-testing-optimization.md)**

---

## 📋 学习进度跟踪表

> 💡 **使用说明**：每天学习后，更新下表记录你的进度、遇到的问题和解决方案

| 日期 | 学习内容 | 状态 | 实际耗时 | 遇到的问题 | 解决方案/笔记 |
|------|----------|------|----------|-----------|--------------|
| Day 1-2 | TCP状态机实现 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 3 | LPM Trie IP段匹配 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 4 | Map压力监控 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 5 | 统计与日志功能 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 6-7 | 功能测试 + 周总结 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |

### 📝 本周学习笔记

**重点概念：**
-
-
-

**遇到的难点：**
1.
2.

**解决的关键问题：**
1.
2.

**下周需要重点关注：**
-
-

---

## 5. 第4周：高级功能实现

### 🎯 本周目标

- [ ] 实现TCP状态机跟踪
- [ ] 实现LPM Trie IP段匹配
- [ ] 实现Map压力监控
- [ ] 添加统计与日志功能

### 📊 本周交付物

1. ✅ TCP状态机完整实现
2. ✅ LPM Trie IP段匹配
3. ✅ Map压力监控系统
4. ✅ 详细日志和统计功能

---

### 📅 Day 1-2: TCP状态机实现

#### 🎯 任务目标 (Day 1-2)
- 理解TCP协议状态机 (RFC 793)
- 实现TCP连接跟踪
- 处理SYN/ACK/FIN/RST标志位

#### ✅ 具体任务

**Day 1上午：学习TCP状态机**

📚 **学习资料**:
1. 阅读RFC 793:
   - TCP连接建立 (3次握手)
   - TCP连接终止 (4次挥手)
   - TCP状态转换图
   - 时间: 1.5小时

2. 研究现有实现:
   ```bash
   # 查看Linux conntrack实现
   less /proc/net/nf_conntrack
   
   # 查看TCP状态定义
   grep -r "TCP_" /usr/include/netinet/tcp.h



   - 时间: 1小时

3. 理解关键状态:
   - CLOSED, SYN_SENT, SYN_RECV
   - ESTABLISHED
   - FIN_WAIT1, FIN_WAIT2, CLOSING, TIME_WAIT
   - CLOSE_WAIT, LAST_ACK
   - 时间: 30分钟

**Day 1下午 + Day 2：实现TCP状态机**

更新session_value结构添加TCP状态:

```c
// TCP状态定义
enum tcp_state {
    TCP_NONE = 0,
    TCP_SYN_SENT,
    TCP_SYN_RECV,
    TCP_ESTABLISHED,
    TCP_FIN_WAIT1,
    TCP_FIN_WAIT2,
    TCP_TIME_WAIT,
    TCP_CLOSE,
    TCP_CLOSE_WAIT,
    TCP_LAST_ACK,
    TCP_CLOSING,
    TCP_MAX
};

struct session_value {
    __u64 last_seen;
    __u64 packet_count;
    __u64 byte_count;
    __u8  cached_action;
    __u8  tcp_state;      // 新增TCP状态
    __u32 policy_id;
    __u32 flags;
} __attribute__((packed));
```

实现TCP状态转换函数:

```c
static __always_inline int tcp_state_transition(
    struct session_value *sess, struct tcphdr *tcp, bool is_ingress)
{
    __u8 flags = 0;
    flags |= tcp->syn ? 0x01 : 0;
    flags |= tcp->ack ? 0x02 : 0;
    flags |= tcp->fin ? 0x04 : 0;
    flags |= tcp->rst ? 0x08 : 0;

    __u8 old_state = sess->tcp_state;
    __u8 new_state = old_state;

    switch (old_state) {
        case TCP_NONE:
            // 初始状态
            if (flags == 0x01) {  // SYN
                new_state = TCP_SYN_SENT;
            } else if (flags == 0x03) {  // SYN+ACK
                new_state = TCP_SYN_RECV;
            }
            break;

        case TCP_SYN_SENT:
            if (flags == 0x03) {  // SYN+ACK
                new_state = TCP_SYN_RECV;
            } else if (flags == 0x08) {  // RST
                new_state = TCP_CLOSE;
            }
            break;

        case TCP_SYN_RECV:
            if (flags == 0x02) {  // ACK
                new_state = TCP_ESTABLISHED;
            } else if (flags == 0x08) {  // RST
                new_state = TCP_CLOSE;
            }
            break;

        case TCP_ESTABLISHED:
            if (flags == 0x04 || flags == 0x06) {  // FIN or FIN+ACK
                new_state = is_ingress ? TCP_CLOSE_WAIT : TCP_FIN_WAIT1;
            } else if (flags == 0x08) {  // RST
                new_state = TCP_CLOSE;
            }
            break;

        case TCP_FIN_WAIT1:
            if (flags == 0x02) {  // ACK
                new_state = TCP_FIN_WAIT2;
            } else if (flags == 0x04 || flags == 0x06) {  // FIN or FIN+ACK
                new_state = TCP_CLOSING;
            } else if (flags == 0x08) {  // RST
                new_state = TCP_CLOSE;
            }
            break;

        case TCP_FIN_WAIT2:
            if (flags == 0x04 || flags == 0x06) {  // FIN or FIN+ACK
                new_state = TCP_TIME_WAIT;
            } else if (flags == 0x08) {  // RST
                new_state = TCP_CLOSE;
            }
            break;

        case TCP_CLOSING:
            if (flags == 0x02) {  // ACK
                new_state = TCP_TIME_WAIT;
            } else if (flags == 0x08) {  // RST
                new_state = TCP_CLOSE;
            }
            break;

        case TCP_CLOSE_WAIT:
            if (flags == 0x04 || flags == 0x06) {  // FIN or FIN+ACK
                new_state = TCP_LAST_ACK;
            } else if (flags == 0x08) {  // RST
                new_state = TCP_CLOSE;
            }
            break;

        case TCP_LAST_ACK:
            if (flags == 0x02) {  // ACK
                new_state = TCP_CLOSE;
            } else if (flags == 0x08) {  // RST
                new_state = TCP_CLOSE;
            }
            break;

        case TCP_TIME_WAIT:
            // TIME_WAIT超时后变为CLOSE (由用户态清理)
            break;

        case TCP_CLOSE:
            // 已关闭, 应该被删除
            return -1;
    }

    sess->tcp_state = new_state;

    // 打印状态转换 (调试用)
    if (old_state != new_state) {
        bpf_printk("TCP state: %d -> %d (flags=0x%x)\n",
                   old_state, new_state, flags);
    }

    // TIME_WAIT或CLOSE状态标记为待清理
    if (new_state == TCP_TIME_WAIT || new_state == TCP_CLOSE) {
        return 1;  // 需要清理
    }

    return 0;
}
```

集成到主程序:

```c
SEC("tc")
int microsegment_filter(struct __sk_buff *skb)
{
    // ... 现有代码 ...

    // 如果是TCP协议,更新状态机
    if (key.protocol == IPPROTO_TCP) {
        void *data = (void *)(long)skb->data;
        void *data_end = (void *)(long)skb->data_end;

        struct ethhdr *eth = data;
        if ((void *)(eth + 1) > data_end)
            goto allow;

        struct iphdr *iph = (void *)(eth + 1);
        if ((void *)(iph + 1) > data_end)
            goto allow;

        struct tcphdr *tcph = (void *)iph + (iph->ihl * 4);
        if ((void *)(tcph + 1) > data_end)
            goto allow;

        // 更新TCP状态
        if (sess) {
            int ret = tcp_state_transition(sess, tcph, true);
            if (ret == 1) {
                // 标记为需要清理
                sess->flags |= 0x01;
            } else if (ret < 0) {
                // 删除已关闭的会话
                bpf_map_delete_elem(&session_map, &key);
                return TC_ACT_SHOT;
            }
        }
    }

allow:
    update_stat(STAT_ALLOWED);
    return TC_ACT_OK;
}
```

创建状态机测试脚本 `tests/test_tcp_statemachine.sh`:

```bash
#!/bin/bash
set -e

echo "=== TCP状态机测试 ==="

# 启动 libbpf 加载器
sudo ./microsegment_loader lo &
PID=$!
sleep 2

# 建立TCP连接
echo "测试1: TCP 3次握手"
nc -zv 127.0.0.1 80 2>&1 | head -1

# 查看会话状态
sudo bpftool map dump name session_map

# 查看内核trace日志
sudo cat /sys/kernel/debug/tracing/trace_pipe | grep "TCP state" | head -5 &
TRACE_PID=$!

# 完整连接测试
echo "测试2: 完整TCP连接 (建立+传输+关闭)"
curl -s http://127.0.0.1 >/dev/null

sleep 1

# 停止trace
sudo kill $TRACE_PID 2>/dev/null || true

# 清理
sudo kill $PID 2>/dev/null || true

echo "✓ TCP状态机测试完成"
```

#### 📚 学习资料

1. TCP协议深入:
   - RFC 793全文
   - TCP状态转换图
   - 时间: 2小时

2. 连接跟踪实现:
   - Linux nf_conntrack源码
   - 状态超时处理
   - 时间: 1.5小时

#### ✅ 完成标准 (Day 1-2)

- [ ] TCP状态枚举定义正确
- [ ] 状态转换逻辑符合RFC 793
- [ ] 能正确跟踪TCP 3次握手
- [ ] 能正确跟踪TCP 4次挥手
- [ ] RST包处理正确

---

### 📅 Day 3: IP段匹配 (LPM Trie)

#### 🎯 任务目标
- 理解LPM (Longest Prefix Match) 算法
- 使用BPF_MAP_TYPE_LPM_TRIE实现IP段匹配
- 支持CIDR格式策略

#### ✅ 具体任务

**全天：实现LPM Trie IP段匹配**

定义LPM Trie Map:

```c
// LPM Trie key
struct lpm_key {
    __u32 prefixlen;  // 前缀长度 (0-32)
    __u32 ip;         // IP地址
} __attribute__((packed));

struct ip_range_value {
    __u8 action;
    __u32 priority;
    __u64 hit_count;
} __attribute__((packed));

// IP段匹配Map (LPM Trie)
struct {
    __uint(type, BPF_MAP_TYPE_LPM_TRIE);
    __uint(max_entries, 10000);
    __uint(map_flags, BPF_F_NO_PREALLOC);  // LPM Trie必须
    __type(key, struct lpm_key);
    __type(value, struct ip_range_value);
} ip_range_map SEC(".maps");
```

实现IP段匹配逻辑:

```c
static __always_inline int lookup_ip_range_policy(
    __u32 src_ip, __u32 dst_ip, __u8 *action)
{
    struct lpm_key key;

    // 1. 先查找目的IP段
    key.prefixlen = 32;  // 从最长前缀开始
    key.ip = dst_ip;

    struct ip_range_value *val = bpf_map_lookup_elem(&ip_range_map, &key);
    if (val) {
        *action = val->action;
        __sync_fetch_and_add(&val->hit_count, 1);
        return 0;
    }

    // 2. 查找源IP段
    key.ip = src_ip;

    val = bpf_map_lookup_elem(&ip_range_map, &key);
    if (val) {
        *action = val->action;
        __sync_fetch_and_add(&val->hit_count, 1);
        return 0;
    }

    return -1;  // 未找到
}
```

集成到策略查找流程:

```c
SEC("tc")
int microsegment_filter(struct __sk_buff *skb)
{
    // ... 解析数据包 ...

    // 策略查找顺序:
    // 1. 精确匹配 (policy_map)
    // 2. IP段匹配 (ip_range_map)
    // 3. 默认动作

    __u8 action = ACTION_ALLOW;

    // 1. 精确5元组匹配
    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, &key);
    if (policy) {
        action = policy->action;
        goto apply_action;
    }

    // 2. IP段匹配
    if (lookup_ip_range_policy(key.src_ip, key.dst_ip, &action) == 0) {
        update_stat(STAT_IP_RANGE_HIT);
        goto apply_action;
    }

    // 3. 默认允许
    action = ACTION_ALLOW;

apply_action:
    // ... 执行动作 ...
}
```

添加CLI支持 (ip-range子命令):

```c
int ip_range_add(int argc, char **argv)
{
    const char *cidr = NULL;
    const char *action = "allow";

    // 解析参数...

    // 解析CIDR
    char ip_str[16];
    int prefix_len;
    if (sscanf(cidr, "%[^/]/%d", ip_str, &prefix_len) != 2) {
        fprintf(stderr, "Invalid CIDR format\n");
        return 1;
    }

    if (prefix_len < 0 || prefix_len > 32) {
        fprintf(stderr, "Invalid prefix length\n");
        return 1;
    }

    struct lpm_key key;
    struct ip_range_value value = {0};

    key.prefixlen = prefix_len;
    inet_pton(AF_INET, ip_str, &key.ip);

    value.action = (strcmp(action, "deny") == 0) ? 1 : 0;
    value.priority = 100;

    // 打开Map
    char map_path[256];
    snprintf(map_path, sizeof(map_path), "%s/ip_range_map", cfg.map_dir);
    int map_fd = bpf_obj_get(map_path);
    if (map_fd < 0) {
        perror("Failed to open ip_range_map");
        return 1;
    }

    // 添加规则
    if (bpf_map_update_elem(map_fd, &key, &value, BPF_ANY) < 0) {
        perror("Failed to add IP range rule");
        close(map_fd);
        return 1;
    }

    printf("✓ IP range added: %s (%s)\n", cidr, action);
    close(map_fd);
    return 0;
}
```

测试LPM Trie:

```bash
# 添加IP段规则
sudo ./tc_microsegment_cli ip-range add --cidr 192.168.1.0/24 --action allow
sudo ./tc_microsegment_cli ip-range add --cidr 10.0.0.0/8 --action deny
sudo ./tc_microsegment_cli ip-range add --cidr 172.16.0.0/12 --action allow

# 列出规则
sudo ./tc_microsegment_cli ip-range list

# 测试匹配
# 192.168.1.100 应该匹配 192.168.1.0/24 (allow)
# 10.1.2.3 应该匹配 10.0.0.0/8 (deny)
```

#### 📚 学习资料

1. LPM算法:
   - Longest Prefix Match原理
   - Trie数据结构
   - 时间: 1小时

2. BPF_MAP_TYPE_LPM_TRIE:
   - 内核实现
   - 性能特性
   - 时间: 1小时

#### ✅ 完成标准 (Day 3)

- [ ] LPM Trie Map正确创建
- [ ] CIDR格式解析正确
- [ ] IP段匹配逻辑正确
- [ ] 最长前缀优先
- [ ] CLI工具支持IP段操作

---

### 📅 Day 4: Map压力监控

#### 🎯 任务目标
- 实现Map容量监控
- 实现SYN Flood检测
- 实现降级策略

#### ✅ 具体任务

**全天：实现Map压力监控系统**

定义压力监控结构:

```c
struct map_pressure {
    __u64 total_sessions;
    __u64 evictions;          // LRU淘汰次数
    __u32 pressure_level;     // 0-100
    __u32 syn_sent_count;     // SYN_SENT状态数量
    __u64 last_check_time;
} __attribute__((packed));

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, struct map_pressure);
} pressure_map SEC(".maps");
```

实现压力检查函数:

```c
#define MAX_SESSIONS 100000
#define SYN_FLOOD_THRESHOLD 10000
#define PRESSURE_CHECK_INTERVAL_NS (1000000000ULL)  // 1秒

static __always_inline int check_map_pressure(
    struct map_pressure *pressure)
{
    __u64 now = bpf_ktime_get_ns();

    // 每秒检查一次
    if (now - pressure->last_check_time < PRESSURE_CHECK_INTERVAL_NS)
        return 0;

    pressure->last_check_time = now;

    // 计算压力级别
    __u32 usage = (pressure->total_sessions * 100) / MAX_SESSIONS;
    pressure->pressure_level = usage;

    // 检查SYN Flood
    if (pressure->syn_sent_count > SYN_FLOOD_THRESHOLD) {
        bpf_printk("WARNING: Possible SYN flood detected (%u)\n",
                   pressure->syn_sent_count);
        return 1;  // SYN Flood
    }

    // 压力级别告警
    if (usage > 90) {
        bpf_printk("CRITICAL: Map pressure at %u%%\n", usage);
    } else if (usage > 80) {
        bpf_printk("WARNING: Map pressure at %u%%\n", usage);
    }

    return 0;
}

static __always_inline int handle_map_full_scenario(
    struct __sk_buff *skb,
    struct session_key *key,
    struct session_value *val,
    struct policy_value *policy)
{
    int ret = bpf_map_update_elem(&session_map, key, val, BPF_NOEXIST);

    if (ret == -ENOSPC || ret == -E2BIG) {
        // Map已满, 执行降级策略
        update_stat(STAT_MAP_FULL);

        // 策略1: 如果策略明确允许,放行
        if (policy && policy->action == ACTION_ALLOW)
            return TC_ACT_OK;

        // 策略2: 如果是重要协议 (DNS, ICMP), 放行
        if (key->protocol == IPPROTO_ICMP || key->dst_port == htons(53))
            return TC_ACT_OK;

        // 策略3: 其他情况拒绝 (保守策略)
        bpf_printk("Map full, dropping packet\n");
        return TC_ACT_SHOT;
    }

    return ret;
}
```

集成到主程序:

```c
SEC("tc")
int microsegment_filter(struct __sk_buff *skb)
{
    // ... 现有逻辑 ...

    // 获取压力监控信息
    __u32 pressure_key = 0;
    struct map_pressure *pressure = bpf_map_lookup_elem(&pressure_map, &pressure_key);
    if (pressure) {
        // 检查Map压力
        check_map_pressure(pressure);

        // 更新会话计数
        if (!sess) {
            __sync_fetch_and_add(&pressure->total_sessions, 1);

            // 如果是TCP SYN包
            if (key.protocol == IPPROTO_TCP) {
                void *data = (void *)(long)skb->data;
                void *data_end = (void *)(long)skb->data_end;
                struct ethhdr *eth = data;
                if ((void *)(eth + 1) <= data_end) {
                    struct iphdr *iph = (void *)(eth + 1);
                    if ((void *)(iph + 1) <= data_end) {
                        struct tcphdr *tcph = (void *)iph + (iph->ihl * 4);
                        if ((void *)(tcph + 1) <= data_end && tcph->syn && !tcph->ack) {
                            __sync_fetch_and_add(&pressure->syn_sent_count, 1);
                        }
                    }
                }
            }
        }

        // 如果压力过高,启用降级策略
        if (pressure->pressure_level > 95) {
            // 只允许已建立的连接
            if (!sess) {
                update_stat(STAT_PRESSURE_DROP);
                return TC_ACT_SHOT;
            }
        }
    }

    // 创建新会话时检查容量
    if (!sess) {
        ret = handle_map_full_scenario(skb, &key, &new_sess, policy);
        if (ret < 0) {
            return TC_ACT_SHOT;
        }
    }

    // ... 其余逻辑 ...
}
```

添加监控CLI命令:

```c
int monitor_pressure(int argc, char **argv)
{
    char map_path[256];
    snprintf(map_path, sizeof(map_path), "%s/pressure_map", cfg.map_dir);
    int map_fd = bpf_obj_get(map_path);
    if (map_fd < 0) {
        perror("Failed to open pressure_map");
        return 1;
    }

    printf("=== Map Pressure Monitor ===\n");
    printf("Press Ctrl+C to exit...\n\n");

    while (1) {
        __u32 key = 0;
        struct map_pressure pressure;

        if (bpf_map_lookup_elem(map_fd, &key, &pressure) == 0) {
            printf("\r");
            printf("Sessions: %llu | ", pressure.total_sessions);
            printf("Evictions: %llu | ", pressure.evictions);
            printf("Pressure: %u%% | ", pressure.pressure_level);
            printf("SYN_SENT: %u   ", pressure.syn_sent_count);
            fflush(stdout);
        }

        sleep(1);
    }

    close(map_fd);
    return 0;
}
```

#### 📚 学习资料

1. Map容量管理:
   - LRU淘汰机制
   - 容量规划方法
   - 时间: 1小时

2. DDoS防护:
   - SYN Flood原理
   - 检测和缓解方法
   - 时间: 1.5小时

#### ✅ 完成标准 (Day 4)

- [ ] 压力监控Map正确实现
- [ ] SYN Flood检测工作
- [ ] 降级策略能触发
- [ ] monitor命令实时显示压力

---

### 📅 Day 5: 统计与日志功能

#### 🎯 任务目标
- 完善统计系统
- 添加perf event日志
- 实现Prometheus导出器

#### ✅ 具体任务

**上午：完善统计系统**

添加更多统计维度:

```c
enum {
    STAT_TOTAL = 0,
    STAT_POLICY_HIT,
    STAT_SESSION_HIT,
    STAT_NEW_SESSION,
    STAT_ALLOWED,
    STAT_DENIED,
    STAT_LOGGED,
    STAT_DROPPED,
    STAT_IP_RANGE_HIT,      // 新增
    STAT_MAP_FULL,          // 新增
    STAT_PRESSURE_DROP,     // 新增
    STAT_TCP_ESTABLISHED,   // 新增
    STAT_TCP_CLOSED,        // 新增
    STAT_MAX
};

// 扩展stats_map
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, STAT_MAX);
    __type(key, __u32);
    __type(value, __u64);
} stats_map SEC(".maps");
```

**下午：Prometheus导出器**

创建HTTP服务器 `src/user/prometheus.c`:

```c
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <bpf/bpf.h>
#include "types.h"

#define PORT 9100

void export_metrics(int client_fd, int stats_fd, int session_fd, int policy_fd)
{
    char buffer[4096];
    int len = 0;

    len += snprintf(buffer + len, sizeof(buffer) - len,
                    "HTTP/1.1 200 OK\r\n"
                    "Content-Type: text/plain\r\n\r\n");

    // 读取统计数据
    __u64 stats[STAT_MAX] = {0};
    for (__u32 key = 0; key < STAT_MAX; key++) {
        bpf_map_lookup_elem(stats_fd, &key, &stats[key]);
    }

    // 输出Prometheus格式指标
    len += snprintf(buffer + len, sizeof(buffer) - len,
                    "# HELP microsegment_packets_total Total packets processed\n"
                    "# TYPE microsegment_packets_total counter\n"
                    "microsegment_packets_total{action=\"total\"} %llu\n"
                    "microsegment_packets_total{action=\"allowed\"} %llu\n"
                    "microsegment_packets_total{action=\"denied\"} %llu\n\n",
                    stats[STAT_TOTAL], stats[STAT_ALLOWED], stats[STAT_DENIED]);

    len += snprintf(buffer + len, sizeof(buffer) - len,
                    "# HELP microsegment_sessions_total Total sessions\n"
                    "# TYPE microsegment_sessions_total counter\n"
                    "microsegment_sessions_total %llu\n\n",
                    stats[STAT_NEW_SESSION]);

    len += snprintf(buffer + len, sizeof(buffer) - len,
                    "# HELP microsegment_cache_hits Cache hits\n"
                    "# TYPE microsegment_cache_hits counter\n"
                    "microsegment_cache_hits{type=\"policy\"} %llu\n"
                    "microsegment_cache_hits{type=\"session\"} %llu\n\n",
                    stats[STAT_POLICY_HIT], stats[STAT_SESSION_HIT]);

    // 发送响应
    write(client_fd, buffer, len);
}

int main(int argc, char **argv)
{
    int server_fd, client_fd;
    struct sockaddr_in address;
    int addrlen = sizeof(address);

    // 打开BPF Maps
    int stats_fd = bpf_obj_get("/sys/fs/bpf/stats_map");
    int session_fd = bpf_obj_get("/sys/fs/bpf/session_map");
    int policy_fd = bpf_obj_get("/sys/fs/bpf/policy_map");

    if (stats_fd < 0 || session_fd < 0 || policy_fd < 0) {
        fprintf(stderr, "Failed to open BPF maps\n");
        return 1;
    }

    // 创建socket
    server_fd = socket(AF_INET, SOCK_STREAM, 0);
    int opt = 1;
    setsockopt(server_fd, SOL_SOCKET, SO_REUSEADDR, &opt, sizeof(opt));

    address.sin_family = AF_INET;
    address.sin_addr.s_addr = INADDR_ANY;
    address.sin_port = htons(PORT);

    bind(server_fd, (struct sockaddr *)&address, sizeof(address));
    listen(server_fd, 3);

    printf("Prometheus exporter listening on http://0.0.0.0:%d/metrics\n", PORT);

    while (1) {
        client_fd = accept(server_fd, (struct sockaddr *)&address, (socklen_t*)&addrlen);
        export_metrics(client_fd, stats_fd, session_fd, policy_fd);
        close(client_fd);
    }

    return 0;
}
```

测试Prometheus导出:

```bash
# 启动导出器
sudo ./prometheus_exporter &

# 测试抓取指标
curl http://localhost:9100/metrics

# 配置Prometheus
cat > prometheus.yml <<EOF
scrape_configs:
  - job_name: 'ebpf_microsegment'
    static_configs:
      - targets: ['localhost:9100']
EOF

# 启动Prometheus
prometheus --config.file=prometheus.yml
```

#### 📚 学习资料

1. Prometheus指标规范:
   - Counter vs Gauge
   - 指标命名规范
   - 时间: 1小时

2. HTTP服务器编程:
   - Socket API
   - HTTP协议基础
   - 时间: 1小时

#### ✅ 完成标准 (Day 5)

- [ ] 统计维度完善
- [ ] Prometheus导出器工作
- [ ] 指标格式符合规范
- [ ] 能被Prometheus抓取

---

### 📅 本周总结 (Friday晚上)

#### ✍️ 输出物

创建文档 `docs/week4_summary.md`:

```markdown
# 第4周学习总结

## 完成情况

- [x] TCP状态机实现
- [x] LPM Trie IP段匹配
- [x] Map压力监控
- [x] 统计与日志功能

## 核心收获

### 1. TCP状态机
- 11种TCP状态完整实现
- 符合RFC 793规范
- 能跟踪完整TCP生命周期

### 2. IP段匹配
- LPM Trie实现最长前缀匹配
- 支持CIDR格式策略
- 性能O(log n)

### 3. Map压力监控
- 实时监控Map使用率
- SYN Flood检测
- 自动降级策略

### 4. 可观测性
- 多维度统计指标
- Prometheus导出器
- 实时监控Dashboard

## 功能演示

```bash
# 添加IP段规则
tc_microsegment_cli ip-range add --cidr 192.168.0.0/16 --action allow

# 监控Map压力
tc_microsegment_cli monitor pressure

# 查看Prometheus指标
curl http://localhost:9100/metrics
```



#### 🎯 本周验收标准

**必须完成**:
- [ ] TCP状态机测试通过
- [ ] LPM Trie匹配正确
- [ ] 压力监控能触发告警
- [ ] Prometheus指标可抓取

**加分项**:
- [ ] 完整的状态转换测试
- [ ] SYN Flood防护测试
- [ ] Grafana Dashboard

---

## 6. 第5周：测试与优化

### 🎯 本周目标

- [ ] 编写单元测试
- [ ] 功能测试与bug修复
- [ ] 性能测试与调优
- [ ] 压力测试

### 📊 本周交付物

---

**[⬅️ 第3周](./week3-userspace-control.md)** | **[📚 目录](./README.md)** | **[➡️ 第5周](./week5-testing-optimization.md)**
