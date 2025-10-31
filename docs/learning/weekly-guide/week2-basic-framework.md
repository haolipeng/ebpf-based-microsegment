# 第2周：基础框架开发

**[⬅️ 第1周](./week1-environment-and-basics.md)** | **[📚 目录](./README.md)** | **[➡️ 第3周](./week3-userspace-control.md)**

---

## 📋 学习进度跟踪表

> 💡 **使用说明**：每天学习后，更新下表记录你的进度、遇到的问题和解决方案

| 日期 | 学习内容 | 状态 | 实际耗时 | 遇到的问题 | 解决方案/笔记 |
|------|----------|------|----------|-----------|--------------|
| Day 1-2 | 会话跟踪实现 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 3 | 策略Map设计 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 4 | 策略匹配和执行 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 5 | 集成测试 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 6-7 | 项目重构 + 周总结 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |

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

## 3. 第2周：基础框架开发

### 🎯 本周目标

- [ ] 实现会话跟踪（LRU_HASH Map）
- [ ] 实现完整的策略Map设计
- [ ] 完成基础策略执行逻辑
- [ ] 建立项目代码结构

### 📊 本周交付物

1. ✅ 可工作的会话跟踪系统
2. ✅ 基于5元组的完整策略匹配
3. ✅ 策略执行demo (allow/deny/log)
4. ✅ 规范的项目代码结构

---

### 📅 Day 1-2: 会话跟踪实现

#### 🎯 任务目标 (Day 1-2)
- 理解会话跟踪的必要性
- 实现LRU_HASH Map管理会话
- 实现新建会话和查找会话逻辑

#### ✅ 具体任务

**Day 1上午：学习LRU机制**

📚 **学习资料**:
1. 理解LRU (Least Recently Used) 算法
   - 为什么需要LRU：Map容量有限，自动淘汰
   - LRU vs HASH的区别
   - 时间：1小时

2. 阅读内核文档:
   ```bash
   # 查看BPF_MAP_TYPE_LRU_HASH文档
   grep -A 20 "BPF_MAP_TYPE_LRU_HASH" linux/include/uapi/linux/bpf.h
   ```
   - 时间：30分钟

3. 会话跟踪的概念:
   - 连接跟踪 (conntrack) 的作用
   - 有状态防火墙 vs 无状态防火墙
   - 时间：30分钟

**Day 1下午 + Day 2全天：实现会话跟踪**

创建文件 `src/bpf/session_tracking.bpf.c`:

```c
#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// 会话key (5元组)
struct session_key {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;
} __attribute__((packed));

// 会话value
struct session_value {
    __u64 last_seen;      // 最后活跃时间
    __u64 packet_count;   // 数据包计数
    __u64 byte_count;     // 字节计数
    __u32 flags;          // 标志位
} __attribute__((packed));

// 会话超时时间 (纳秒)
#define SESSION_TIMEOUT_NS (60ULL * 1000000000ULL)  // 60秒

// 会话Map (使用LRU自动淘汰)
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 100000);  // 最多10万并发会话
    __type(key, struct session_key);
    __type(value, struct session_value);
} session_map SEC(".maps");

// 统计Map
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 5);
    __type(key, __u32);
    __type(value, __u64);
} stats_map SEC(".maps");

enum {
    STAT_TOTAL_PACKETS = 0,
    STAT_NEW_SESSIONS = 1,
    STAT_EXISTING_SESSIONS = 2,
    STAT_SESSION_TIMEOUTS = 3,
    STAT_ALLOWED = 4,
};

static __always_inline void update_stat(__u32 key, __u64 val)
{
    __u64 *count = bpf_map_lookup_elem(&stats_map, &key);
    if (count)
        __sync_fetch_and_add(count, val);
}

static __always_inline int parse_packet(struct __sk_buff *skb, struct session_key *key)
{
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // 解析以太网头
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return -1;

    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return -1;

    // 解析IP头
    struct iphdr *iph = (void *)(eth + 1);
    if ((void *)(iph + 1) > data_end)
        return -1;

    key->src_ip = iph->saddr;
    key->dst_ip = iph->daddr;
    key->protocol = iph->protocol;

    // 解析传输层端口
    void *l4 = (void *)iph + (iph->ihl * 4);

    if (iph->protocol == IPPROTO_TCP) {
        struct tcphdr *tcph = l4;
        if ((void *)(tcph + 1) > data_end)
            return -1;
        key->src_port = tcph->source;
        key->dst_port = tcph->dest;
    }
    else if (iph->protocol == IPPROTO_UDP) {
        struct udphdr *udph = l4;
        if ((void *)(udph + 1) > data_end)
            return -1;
        key->src_port = udph->source;
        key->dst_port = udph->dest;
    }
    else {
        key->src_port = 0;
        key->dst_port = 0;
    }

    return 0;
}

SEC("tc")
int track_session(struct __sk_buff *skb)
{
    struct session_key key = {0};

    update_stat(STAT_TOTAL_PACKETS, 1);

    // 解析数据包
    if (parse_packet(skb, &key) < 0)
        return TC_ACT_OK;

    __u64 now = bpf_ktime_get_ns();

    // 查找现有会话
    struct session_value *sess = bpf_map_lookup_elem(&session_map, &key);

    if (sess) {
        // 现有会话
        update_stat(STAT_EXISTING_SESSIONS, 1);

        // 检查超时
        if (now - sess->last_seen > SESSION_TIMEOUT_NS) {
            update_stat(STAT_SESSION_TIMEOUTS, 1);
            bpf_map_delete_elem(&session_map, &key);

            // 创建新会话
            struct session_value new_sess = {
                .last_seen = now,
                .packet_count = 1,
                .byte_count = skb->len,
            };
            bpf_map_update_elem(&session_map, &key, &new_sess, BPF_ANY);
            update_stat(STAT_NEW_SESSIONS, 1);
        }
        else {
            // 更新现有会话
            sess->last_seen = now;
            __sync_fetch_and_add(&sess->packet_count, 1);
            __sync_fetch_and_add(&sess->byte_count, skb->len);
        }
    }
    else {
        // 新会话
        update_stat(STAT_NEW_SESSIONS, 1);

        struct session_value new_sess = {
            .last_seen = now,
            .packet_count = 1,
            .byte_count = skb->len,
        };
        bpf_map_update_elem(&session_map, &key, &new_sess, BPF_ANY);

        bpf_printk("NEW SESSION: %pI4:%d -> %pI4:%d (proto=%d)\n",
                   &key.src_ip, bpf_ntohs(key.src_port),
                   &key.dst_ip, bpf_ntohs(key.dst_port),
                   key.protocol);
    }

    update_stat(STAT_ALLOWED, 1);
    return TC_ACT_OK;
}

char LICENSE[] SEC("license") = "GPL";
```

创建用户态工具 `src/user/session_viewer.c`:

```c
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <signal.h>
#include <arpa/inet.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>
#include <time.h>

struct session_key {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;
} __attribute__((packed));

struct session_value {
    __u64 last_seen;
    __u64 packet_count;
    __u64 byte_count;
    __u32 flags;
} __attribute__((packed));

static int running = 1;

void sigint_handler(int sig)
{
    running = 0;
}

const char* proto_name(__u8 proto)
{
    switch (proto) {
        case 6: return "TCP";
        case 17: return "UDP";
        case 1: return "ICMP";
        default: return "OTHER";
    }
}

int main(int argc, char **argv)
{
    if (argc < 2) {
        printf("Usage: %s <session_map_path>\n", argv[0]);
        printf("Example: %s /sys/fs/bpf/session_map\n", argv[0]);
        return 1;
    }

    // 打开Map
    int map_fd = bpf_obj_get(argv[1]);
    if (map_fd < 0) {
        perror("bpf_obj_get");
        return 1;
    }

    signal(SIGINT, sigint_handler);

    printf("查看活跃会话 (按Ctrl+C退出)...\n\n");

    while (running) {
        system("clear");

        printf("=== Active Sessions ===\n");
        printf("%-15s %-6s %-15s %-6s %-5s %10s %12s %10s\n",
               "SRC_IP", "PORT", "DST_IP", "PORT", "PROTO", "PACKETS", "BYTES", "AGE");

        struct session_key key, next_key;
        struct session_value value;
        int count = 0;
        __u64 now_ns = time(NULL) * 1000000000ULL;

        __builtin_memset(&key, 0, sizeof(key));
        while (bpf_map_get_next_key(map_fd, &key, &next_key) == 0) {
            if (bpf_map_lookup_elem(map_fd, &next_key, &value) == 0) {
                char src_ip[16], dst_ip[16];
                inet_ntop(AF_INET, &next_key.src_ip, src_ip, sizeof(src_ip));
                inet_ntop(AF_INET, &next_key.dst_ip, dst_ip, sizeof(dst_ip));

                __u64 age_sec = (now_ns - value.last_seen) / 1000000000ULL;

                printf("%-15s %-6d %-15s %-6d %-5s %10llu %12llu %8llus\n",
                       src_ip, ntohs(next_key.src_port),
                       dst_ip, ntohs(next_key.dst_port),
                       proto_name(next_key.protocol),
                       value.packet_count, value.byte_count, age_sec);

                count++;
            }
            key = next_key;
        }

        printf("\nTotal active sessions: %d\n", count);

        sleep(2);
    }

    return 0;
}
```

创建 libbpf 加载器 `src/user/session_loader.c`:

```c
#include <stdio.h>
#include <stdlib.h>
#include <signal.h>
#include <unistd.h>
#include <net/if.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>
#include "session_tracking.skel.h"

static volatile bool exiting = false;

static void sig_handler(int sig) {
    exiting = true;
}

int main(int argc, char **argv) {
    struct session_tracking_bpf *skel;
    int ifindex, err;
    DECLARE_LIBBPF_OPTS(bpf_tc_hook, hook, .attach_point = BPF_TC_INGRESS);
    DECLARE_LIBBPF_OPTS(bpf_tc_opts, opts, .handle = 1, .priority = 1);

    if (argc < 2) {
        fprintf(stderr, "Usage: %s <interface>\n", argv[0]);
        return 1;
    }

    ifindex = if_nametoindex(argv[1]);
    if (!ifindex) {
        fprintf(stderr, "Invalid interface\n");
        return 1;
    }

    // 打开并加载
    skel = session_tracking_bpf__open_and_load();
    if (!skel) {
        fprintf(stderr, "Failed to load skeleton\n");
        return 1;
    }

    // 创建 TC hook
    hook.ifindex = ifindex;
    err = bpf_tc_hook_create(&hook);
    if (err && err != -EEXIST) {
        fprintf(stderr, "Failed to create TC hook\n");
        goto cleanup;
    }

    // 附加程序
    opts.prog_fd = bpf_program__fd(skel->progs.track_session);
    err = bpf_tc_attach(&hook, &opts);
    if (err) {
        fprintf(stderr, "Failed to attach\n");
        goto cleanup;
    }

    // Pin maps
    bpf_map__pin(skel->maps.session_map, "/sys/fs/bpf/session_map");

    printf("✓ Session tracking started on %s\n", argv[1]);
    printf("Press Ctrl+C to exit...\n");

    signal(SIGINT, sig_handler);
    while (!exiting) sleep(1);

    // 清理
    opts.flags = opts.prog_fd = opts.prog_id = 0;
    bpf_tc_detach(&hook, &opts);
    bpf_tc_hook_destroy(&hook);

cleanup:
    session_tracking_bpf__destroy(skel);
    return 0;
}
```

测试:
```bash
# 编译（包含 skeleton 生成）
make session_tracking.bpf.o
bpftool gen skeleton session_tracking.bpf.o > session_tracking.skel.h
gcc -o session_loader src/user/session_loader.c -lbpf -lelf -lz
gcc -o session_viewer src/user/session_viewer.c -lbpf

# 运行（自动加载+附加，Ctrl+C 自动卸载）
sudo ./session_loader lo

# 生成流量
ping 127.0.0.1 &
curl http://example.com &

# 查看会话
sudo ./session_viewer /sys/fs/bpf/session_map
```

#### 📚 学习资料

1. 深入理解连接跟踪:
   - Linux conntrack子系统
   - Netfilter框架
   - 时间：1.5小时

2. LRU Map性能特性:
   - 淘汰算法实现
   - 性能开销分析
   - 时间：1小时

#### ✅ 完成标准 (Day 1-2)

- [ ] LRU_HASH Map正确创建
- [ ] 能跟踪新建会话
- [ ] 能更新现有会话
- [ ] 能自动淘汰超时会话
- [ ] session_viewer能实时显示活跃会话

---

### 📅 Day 3-4: 策略Map完善与会话关联

#### 🎯 任务目标 (Day 3-4)
- 将策略匹配与会话跟踪结合
- 实现"首包查策略,后续包查会话"的快速路径
- 完善策略数据结构

#### ✅ 具体任务

**Day 3-4：实现策略+会话混合架构**

创建完整的 `src/bpf/microsegment.bpf.c`:

```c
#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// ========== 数据结构定义 ==========

struct flow_key {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;
} __attribute__((packed));

enum policy_action {
    ACTION_ALLOW = 0,
    ACTION_DENY = 1,
    ACTION_LOG = 2,
};

struct policy_value {
    __u8 action;
    __u32 priority;
    __u64 hit_count;
} __attribute__((packed));

struct session_value {
    __u64 last_seen;
    __u64 packet_count;
    __u64 byte_count;
    __u8  cached_action;  // 缓存的策略动作
    __u32 policy_id;
} __attribute__((packed));

// ========== Map定义 ==========

// 策略Map
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, struct flow_key);
    __type(value, struct policy_value);
} policy_map SEC(".maps");

// 会话Map (使用LRU)
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, 100000);
    __type(key, struct flow_key);
    __type(value, struct session_value);
} session_map SEC(".maps");

// 统计Map
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 8);
    __type(key, __u32);
    __type(value, __u64);
} stats_map SEC(".maps");

enum {
    STAT_TOTAL = 0,
    STAT_POLICY_HIT = 1,
    STAT_SESSION_HIT = 2,
    STAT_NEW_SESSION = 3,
    STAT_ALLOWED = 4,
    STAT_DENIED = 5,
    STAT_LOGGED = 6,
    STAT_DROPPED = 7,
};

// ========== Helper Functions ==========

static __always_inline void update_stat(__u32 key)
{
    __u64 *count = bpf_map_lookup_elem(&stats_map, &key);
    if (count)
        __sync_fetch_and_add(count, 1);
}

static __always_inline int parse_packet(struct __sk_buff *skb, struct flow_key *key)
{
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return -1;

    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return -1;

    struct iphdr *iph = (void *)(eth + 1);
    if ((void *)(iph + 1) > data_end)
        return -1;

    key->src_ip = iph->saddr;
    key->dst_ip = iph->daddr;
    key->protocol = iph->protocol;

    void *l4 = (void *)iph + (iph->ihl * 4);

    if (iph->protocol == IPPROTO_TCP) {
        struct tcphdr *tcph = l4;
        if ((void *)(tcph + 1) > data_end)
            return -1;
        key->src_port = tcph->source;
        key->dst_port = tcph->dest;
    }
    else if (iph->protocol == IPPROTO_UDP) {
        struct udphdr *udph = l4;
        if ((void *)(udph + 1) > data_end)
            return -1;
        key->src_port = udph->source;
        key->dst_port = udph->dest;
    }
    else {
        key->src_port = 0;
        key->dst_port = 0;
    }

    return 0;
}

// ========== Main Program ==========

SEC("tc")
int microsegment_filter(struct __sk_buff *skb)
{
    struct flow_key key = {0};
    __u8 action = ACTION_ALLOW;
    __u64 now = bpf_ktime_get_ns();

    update_stat(STAT_TOTAL);

    // 1. 解析数据包
    if (parse_packet(skb, &key) < 0)
        return TC_ACT_OK;

    // 2. 快速路径: 查找会话缓存
    struct session_value *sess = bpf_map_lookup_elem(&session_map, &key);
    if (sess) {
        update_stat(STAT_SESSION_HIT);

        // 更新会话统计
        sess->last_seen = now;
        __sync_fetch_and_add(&sess->packet_count, 1);
        __sync_fetch_and_add(&sess->byte_count, skb->len);

        // 使用缓存的动作
        action = sess->cached_action;

        if (action == ACTION_DENY) {
            update_stat(STAT_DENIED);
            return TC_ACT_SHOT;
        }

        update_stat(STAT_ALLOWED);
        return TC_ACT_OK;
    }

    // 3. 慢速路径: 查找策略
    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, &key);
    if (policy) {
        update_stat(STAT_POLICY_HIT);
        __sync_fetch_and_add(&policy->hit_count, 1);
        action = policy->action;
    }

    // 4. 创建新会话 (缓存策略决策)
    struct session_value new_sess = {
        .last_seen = now,
        .packet_count = 1,
        .byte_count = skb->len,
        .cached_action = action,
        .policy_id = policy ? policy->priority : 0,
    };
    bpf_map_update_elem(&session_map, &key, &new_sess, BPF_ANY);
    update_stat(STAT_NEW_SESSION);

    bpf_printk("NEW: %pI4:%d->%pI4:%d action=%d\n",
               &key.src_ip, bpf_ntohs(key.src_port),
               &key.dst_ip, bpf_ntohs(key.dst_port), action);

    // 5. 执行动作
    switch (action) {
        case ACTION_DENY:
            update_stat(STAT_DENIED);
            return TC_ACT_SHOT;

        case ACTION_LOG:
            update_stat(STAT_LOGGED);
            // 记录但放行
            return TC_ACT_OK;

        default:
            update_stat(STAT_ALLOWED);
            return TC_ACT_OK;
    }
}

char LICENSE[] SEC("license") = "GPL";
```

性能测试脚本 `tests/bench_session_cache.sh`:

```bash
#!/bin/bash
set -e

echo "=== Session Cache Performance Test ==="

# 1. 添加一条策略
sudo ./policy_mgmt add 0.0.0.0 127.0.0.1 8080 tcp allow

# 2. 首次请求 (慢速路径 - 查策略)
echo "首次请求 (查策略):"
time curl -s http://127.0.0.1:8080 >/dev/null

# 3. 后续请求 (快速路径 - 查会话)
echo "后续请求x10 (查会话缓存):"
time for i in {1..10}; do
    curl -s http://127.0.0.1:8080 >/dev/null
done

# 4. 查看统计
echo -e "\n=== 统计 ==="
sudo bpftool map dump name stats_map | \
    awk '/key:/{k=$2} /value:/{v=$2; print "STAT_"k": "v}'
```

#### 📚 学习资料

1. 快速路径优化技巧:
   - 会话缓存的命中率
   - Cache-friendly数据结构
   - 时间：1小时

2. 策略决策缓存:
   - First-packet处理
   - 后续包零查找开销
   - 时间：1小时

#### ✅ 完成标准 (Day 3-4)

- [ ] 策略Map和会话Map正确关联
- [ ] 首包查策略,后续包查会话
- [ ] 会话缓存策略动作
- [ ] 性能测试显示明显的缓存加速

---

### 📅 Day 5: 基础策略执行demo与集成测试

#### 🎯 任务目标
- 完善策略执行逻辑 (allow/deny/log)
- 编写集成测试用例
- 性能基准测试

#### ✅ 具体任务

**上午：完善策略管理工具**

更新 `src/user/policy_mgmt.c` 添加更多功能:

```c
// 添加批量导入功能
int load_policies_from_json(int map_fd, const char *filename)
{
    FILE *fp = fopen(filename, "r");
    if (!fp) {
        perror("fopen");
        return -1;
    }

    // 简化版JSON解析 (生产环境应使用 json-c 等库)
    char line[256];
    int count = 0;

    while (fgets(line, sizeof(line), fp)) {
        if (strstr(line, "src_ip") && fgets(line, sizeof(line), fp)) {
            // 解析并添加策略
            // TODO: 完整实现
            count++;
        }
    }

    fclose(fp);
    printf("✓ Loaded %d policies\n", count);
    return count;
}
```

**下午：集成测试**

创建测试套件 `tests/integration_test.sh`:

```bash
#!/bin/bash
set -e

echo "=== 微隔离集成测试 ==="

# 清理环境
sudo killall microsegment_loader 2>/dev/null || true
sleep 1

# 启动 libbpf 加载器（后台运行）
sudo ./microsegment_loader lo &
LOADER_PID=$!
sleep 2

echo "✓ eBPF程序已加载（PID: $LOADER_PID）"

# 测试1: 默认放行
echo -e "\n[Test 1] 默认放行策略"
curl -s http://127.0.0.1 >/dev/null && echo "✓ PASS" || echo "✗ FAIL"

# 测试2: 显式允许
echo -e "\n[Test 2] 显式允许规则"
sudo ./policy_mgmt add 0.0.0.0 127.0.0.1 80 tcp allow
curl -s http://127.0.0.1:80 >/dev/null && echo "✓ PASS" || echo "✗ FAIL"

# 测试3: 拒绝规则
echo -e "\n[Test 3] 拒绝规则"
sudo ./policy_mgmt add 0.0.0.0 127.0.0.1 22 tcp deny
timeout 2 telnet 127.0.0.1 22 2>/dev/null && echo "✗ FAIL" || echo "✓ PASS (正确拒绝)"

# 测试4: 会话缓存
echo -e "\n[Test 4] 会话缓存性能"
for i in {1..100}; do
    curl -s http://127.0.0.1:80 >/dev/null
done

# 查看会话命中率
TOTAL=$(sudo bpftool map dump name stats_map | grep "key: 0" -A1 | grep value | awk '{print $2}')
SESSION_HIT=$(sudo bpftool map dump name stats_map | grep "key: 2" -A1 | grep value | awk '{print $2}')
HITRATE=$((SESSION_HIT * 100 / TOTAL))
echo "会话命中率: $HITRATE%"
[ $HITRATE -gt 90 ] && echo "✓ PASS" || echo "✗ FAIL"

# 清理
sudo kill $LOADER_PID 2>/dev/null || true
echo -e "\n=== 所有测试完成 ==="
```

运行测试:
```bash
chmod +x tests/integration_test.sh
sudo ./tests/integration_test.sh
```

#### ✅ 完成标准 (Day 5)

- [ ] 集成测试全部通过
- [ ] 会话缓存命中率 > 90%
- [ ] 策略动作正确执行
- [ ] 无内存泄漏或崩溃

---

### 📅 本周总结 (Friday晚上)

#### ✍️ 输出物

创建文档 `docs/week2_summary.md`:

```markdown
# 第2周学习总结

## 完成情况

- [x] 会话跟踪(LRU_HASH)
- [x] 策略Map设计
- [x] 策略+会话混合架构
- [x] 基础策略执行demo
- [x] 集成测试套件

## 核心收获

### 1. 会话跟踪机制
- LRU Map自动淘汰最久未使用的会话
- 会话缓存大幅提升性能 (首包查策略,后续查会话)
- 会话超时机制 (60秒)

### 2. 快速路径优化
- Session hit rate > 90% (首包后全部命中缓存)
- 避免重复策略查找
- Per-packet开销降低到 < 1μs

### 3. Map设计模式
- **策略Map**: HASH, 精确匹配, 手动管理
- **会话Map**: LRU_HASH, 自动淘汰, 缓存决策

## 性能测试结果

| 指标 | 数值 |
|------|------|
| 会话缓存命中率 | 95%+ |
| 首包延迟 (查策略) | ~20μs |
| 后续包延迟 (查会话) | ~5μs |
| 并发会话数 | 10万 |

## 下周计划

- 实现用户态控制程序 (libbpf skeleton)
- 完善CLI工具
- 添加配置文件支持
- 实现策略热更新
```

#### 🎯 本周验收标准

**必须完成**:
- [ ] 会话跟踪功能正常
- [ ] 策略+会话混合架构工作
- [ ] 集成测试全部通过
- [ ] 会话缓存命中率 > 90%

**加分项**:
- [ ] 性能测试报告
- [ ] 代码注释完整
- [ ] 绘制数据流程图

---


---

**[⬅️ 第1周](./week1-environment-and-basics.md)** | **[📚 目录](./README.md)** | **[➡️ 第3周](./week3-userspace-control.md)**
