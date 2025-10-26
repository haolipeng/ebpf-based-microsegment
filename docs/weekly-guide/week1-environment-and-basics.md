# 第1周：环境准备 + eBPF基础学习

**[⬅️ 返回目录](./README.md)** | **[➡️ 第2周](./week2-basic-framework.md)**

---

## 📋 学习进度跟踪表

> 💡 **使用说明**：每天学习后，更新下表记录你的进度、遇到的问题和解决方案

| 日期 | 学习内容 | 状态 | 实际耗时 | 遇到的问题 | 解决方案/笔记 |
|------|----------|------|----------|-----------|--------------|
| Day 1 | 环境搭建 + 理论学习 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 2 | Hello World eBPF程序 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 3 | 数据包解析基础 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 4 | BPF Map统计 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 5 | 五元组匹配 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 6-7 | 综合练习 + 周总结 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |

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

## 2. 第1周：环境准备 + eBPF基础学习

### 🎯 本周目标

- [ ] 完成开发环境搭建
- [ ] 掌握eBPF基本原理和TC hook机制
- [ ] 实现并运行Hello World eBPF程序
- [ ] 完成基础数据包解析demo

### 📊 本周交付物

1. ✅ 可用的eBPF开发环境
2. ✅ Hello World eBPF程序 (TC hook)
3. ✅ 数据包解析demo (输出IP/端口信息)
4. ✅ 学习笔记和总结文档

---

### 📅 Day 1: 环境搭建 + 理论学习（上午）

#### 🎯 任务目标
- 搭建完整的eBPF开发环境
- 理解eBPF的基本概念和工作原理

#### ✅ 具体任务

**上午 (3-4小时)：环境搭建**

```bash
# 1. 检查系统环境
uname -r  # 确保 >= 5.10

# 2. 安装依赖工具
sudo apt-get update
sudo apt-get install -y \
    clang \
    llvm \
    libbpf-dev \
    linux-headers-$(uname -r) \
    linux-tools-$(uname -r) \
    build-essential \
    pkg-config \
    git

# 3. 验证工具
bpftool version
clang --version  # >= 11

# 4. 创建项目目录
mkdir -p ~/ebpf-microsegment/{src/bpf,src/user,tests,scripts}
cd ~/ebpf-microsegment
```

**下午 (3-4小时)：理论学习**

📚 **学习资料**：
1. 阅读 [BPF and XDP Reference Guide](https://docs.cilium.io/en/stable/bpf/)
   - 重点：Section 1-3 (eBPF基础架构)
   - 时间：1.5小时

2. 观看视频：[eBPF Introduction](https://www.youtube.com/watch?v=lrSExTfS-iQ)
   - Brendan Gregg的入门讲解
   - 时间：1小时

3. 阅读内核文档：
   ```bash
   # 下载内核文档
   git clone --depth 1 https://github.com/torvalds/linux.git
   cd linux/Documentation/bpf/
   # 阅读 bpf_design_QA.rst 和 libbpf/README.rst
   ```
   - 时间：1小时

#### 📝 学习重点

- **eBPF核心概念**：
  - Verifier如何验证程序安全性
  - Map类型和用途（HASH, ARRAY, LRU_HASH）

- **TC hook机制**：
  - Ingress/Egress区别
  - 与XDP的对比
  - 返回值含义（TC_ACT_OK, TC_ACT_SHOT等）

#### ✅ 完成标准

- [ ] 所有工具安装成功并可运行
- [ ] 理解eBPF的Verifier、Map概念
- [ ] 能绘制出eBPF数据包处理流程图

---

### 📅 Day 2: Hello World eBPF程序

#### 🎯 任务目标
- 编写并运行第一个TC eBPF程序
- 掌握eBPF程序的编译和加载流程
- 学习使用 libbpf + skeleton 进行优雅的程序管理

#### ✅ 具体任务

**上午 (3-4小时)：编写Hello World程序**

创建文件 `src/bpf/hello.bpf.c`:

```c
// hello.bpf.c - 第一个TC eBPF程序，演示 __sk_buff 的使用
#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

SEC("tc")
int hello_world(struct __sk_buff *skb)
{
    // 1. 访问 __sk_buff 的基本字段（无需边界检查）
    bpf_printk("=== eBPF Packet Info ===\n");
    bpf_printk("Packet len=%d bytes\n", skb->len);
    bpf_printk("Interface ifindex=%d\n", skb->ifindex);
    bpf_printk("Protocol=0x%x\n", bpf_ntohs(skb->protocol));

    // 2. 访问数据包内容（需要边界检查）
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // 3. 解析以太网头
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end) {
        bpf_printk("Packet too short for ethernet header\n");
        return TC_ACT_OK;
    }

    // 4. 如果是 IPv4 包，显示更多信息
    if (eth->h_proto == bpf_htons(ETH_P_IP)) {
        struct iphdr *ip = (void *)(eth + 1);
        if ((void *)(ip + 1) > data_end) {
            return TC_ACT_OK;
        }

        bpf_printk("IPv4: protocol=%d\n", ip->protocol);
        bpf_printk("IPv4: saddr=%pI4, daddr=%pI4\n", &ip->saddr, &ip->daddr);
    }

    return TC_ACT_OK;  // 放行数据包
}

char LICENSE[] SEC("license") = "GPL";
```

**代码说明：**
- `skb->len`, `skb->ifindex` 等字段可以直接访问
- `skb->data` 和 `skb->data_end` 必须转为指针后使用
- 每次解析协议头前，必须进行边界检查：`if ((void *)(header + 1) > data_end)`
- `%pI4` 是 bpf_printk 的格式化符，用于打印 IPv4 地址

**下午 (3-4小时)：编写用户态加载程序**

创建用户态加载器 `src/user/hello_loader.c`:

```c
// hello_loader.c - 用户态加载器
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <signal.h>
#include <errno.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>
#include <net/if.h>
#include "hello.skel.h"  // 自动生成的skeleton

static volatile bool exiting = false;

static void sig_handler(int sig)
{
    exiting = true;
}

int main(int argc, char **argv)
{
    struct hello_bpf *skel;
    int ifindex, err;
    DECLARE_LIBBPF_OPTS(bpf_tc_hook, hook, .attach_point = BPF_TC_INGRESS);
    DECLARE_LIBBPF_OPTS(bpf_tc_opts, opts, .handle = 1, .priority = 1);

    if (argc < 2) {
        fprintf(stderr, "Usage: %s <interface>\n", argv[0]);
        fprintf(stderr, "Example: %s lo\n", argv[0]);
        return 1;
    }

    // 1. 获取网卡索引
    ifindex = if_nametoindex(argv[1]);
    if (!ifindex) {
        fprintf(stderr, "Invalid interface: %s\n", argv[1]);
        return 1;
    }

    // 2. 打开并加载eBPF程序（一行代码完成！）
    skel = hello_bpf__open_and_load();
    if (!skel) {
        fprintf(stderr, "Failed to open/load BPF skeleton\n");
        return 1;
    }

    printf("✓ eBPF程序加载成功\n");

    // 3. 创建TC hook
    hook.ifindex = ifindex;
    err = bpf_tc_hook_create(&hook);
    if (err && err != -EEXIST) {
        fprintf(stderr, "Failed to create TC hook: %d\n", err);
        goto cleanup;
    }

    // 4. 附加程序到TC ingress
    opts.prog_fd = bpf_program__fd(skel->progs.hello_world);
    err = bpf_tc_attach(&hook, &opts);
    if (err) {
        fprintf(stderr, "Failed to attach TC program: %d\n", err);
        goto cleanup;
    }

    printf("✓ eBPF程序已附加到 %s (ingress)\n", argv[1]);
    printf("✓ 查看日志: sudo cat /sys/kernel/debug/tracing/trace_pipe\n");
    printf("✓ 按 Ctrl+C 退出并自动卸载...\n\n");

    // 5. 等待用户信号
    signal(SIGINT, sig_handler);
    signal(SIGTERM, sig_handler);

    while (!exiting) {
        sleep(1);
    }

    printf("\n正在卸载程序...\n");

    // 6. 自动清理（detach + destroy）
    opts.flags = opts.prog_fd = opts.prog_id = 0;
    bpf_tc_detach(&hook, &opts);
    bpf_tc_hook_destroy(&hook);

cleanup:
    hello_bpf__destroy(skel);
    printf("✓ 清理完成\n");
    return err;
}
```

**创建完整的Makefile:**

```makefile
CLANG ?= clang
BPFTOOL ?= bpftool
CC ?= gcc

# libbpf路径（根据实际情况调整）
LIBBPF_DIR = /usr
INCLUDES = -I$(LIBBPF_DIR)/include -I.
LIBS = -L$(LIBBPF_DIR)/lib64 -lbpf -lelf -lz

BPF_CFLAGS = -g -O2 -target bpf -D__TARGET_ARCH_x86

# 编译eBPF程序
hello.bpf.o: src/bpf/hello.bpf.c
	$(CLANG) $(BPF_CFLAGS) -c $< -o $@

# 生成skeleton头文件
hello.skel.h: hello.bpf.o
	$(BPFTOOL) gen skeleton $< > $@

# 编译用户态程序
hello_loader: src/user/hello_loader.c hello.skel.h
	$(CC) -g -Wall -o $@ src/user/hello_loader.c $(INCLUDES) $(LIBS)

all: hello_loader

clean:
	rm -f *.o *.skel.h hello_loader
```

**一键编译和测试:**

```bash
# 1. 编译（一条命令搞定所有）
make all

# 2. 运行（自动加载+附加，Ctrl+C自动卸载）
sudo ./hello_loader lo

# 3. 在另一个终端查看日志
sudo cat /sys/kernel/debug/tracing/trace_pipe

# 4. 在第三个终端生成测试流量
ping 127.0.0.1 -c 5

# 5. 你应该能在日志终端看到类似输出：
# === eBPF Packet Info ===
# Packet len=98 bytes
# Interface ifindex=1
# Protocol=0x800
# IPv4: protocol=1
# IPv4: saddr=127.0.0.1, daddr=127.0.0.1

# 6. 按 Ctrl+C 即可自动卸载，无需手动清理！
```

**实战练习：**

修改 `hello.bpf.c`，尝试以下任务来熟悉 `__sk_buff`：

```c
// 练习1: 统计不同协议的数据包
// 提示: 使用 skb->protocol 区分 ETH_P_IP, ETH_P_IPV6, ETH_P_ARP

// 练习2: 过滤大数据包
// 提示: if (skb->len > 1500) return TC_ACT_SHOT;

// 练习3: 显示 VLAN 信息
// 提示: 使用 skb->vlan_present 和 skb->vlan_tci

// 练习4: 显示网卡入方向
// 提示: 使用 skb->ingress_ifindex
```

#### 📚 学习资料

1. **深入理解 `__sk_buff` 结构体**（重要！⭐）

`__sk_buff` 是 TC eBPF 程序的核心上下文结构体，类似于内核中的 `sk_buff`，但经过简化和安全封装。

**核心字段解析：**

```c
struct __sk_buff {
    // === 基本数据包信息 ===
    __u32 len;              // 数据包总长度（包括所有协议头）
    __u32 pkt_type;         // 数据包类型（PACKET_HOST, PACKET_BROADCAST等）
    __u32 mark;             // SKB标记（可用于策略路由）
    __u32 queue_mapping;    // 队列映射
    __u32 protocol;         // 以太网协议类型（如 ETH_P_IP = 0x0800）
    __u32 vlan_present;     // 是否有VLAN标签
    __u32 vlan_tci;         // VLAN标签控制信息
    __u32 vlan_proto;       // VLAN协议
    __u32 priority;         // 数据包优先级

    // === 网络接口信息 ===
    __u32 ifindex;          // 入接口索引（ingress）或出接口索引（egress）
    __u32 ingress_ifindex;  // 入接口索引（0表示从本地生成）

    // === 数据访问（重要！）===
    __u32 data;             // 数据包起始位置（指向L2以太网头）
    __u32 data_end;         // 数据包结束位置
    // 注意：data 和 data_end 是 __u32，但在代码中会被转为指针

    // === TCP/IP相关 ===
    __u32 napi_id;          // NAPI ID

    // === 连接跟踪 ===
    __u32 family;           // 协议族（AF_INET, AF_INET6）
    __u32 remote_ip4;       // 远程IPv4地址（仅限连接跟踪）
    __u32 local_ip4;        // 本地IPv4地址
    __u32 remote_ip6[4];    // 远程IPv6地址
    __u32 local_ip6[4];     // 本地IPv6地址
    __u32 remote_port;      // 远程端口（网络字节序）
    __u32 local_port;       // 本地端口

    // === 时间戳 ===
    __u64 tstamp;           // 时间戳（纳秒）

    // === Wire Length ===
    __u32 wire_len;         // 线上长度（包括被截断的部分）

    // === GRO相关 ===
    __u32 gso_segs;         // GSO段数

    // === Hardware Offload ===
    __u32 hwtstamp;         // 硬件时间戳
};
```

**最常用的字段：**

| 字段 | 用途 | 示例 |
|------|------|------|
| `len` | 数据包总长度 | 过滤大包/小包 |
| `data` | 数据起始指针 | 解析协议头 |
| `data_end` | 数据结束指针 | **边界检查必需** |
| `protocol` | L3协议类型 | 区分IPv4/IPv6/ARP |
| `ifindex` | 网络接口索引 | 区分不同网卡 |
| `mark` | SKB标记 | 策略路由、连接跟踪 |

**数据包访问模式（核心！）：**

```c
SEC("tc")
int demo_packet_access(struct __sk_buff *skb)
{
    // 1. 将 __u32 转换为实际指针
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // 2. 解析以太网头（MUST 边界检查！）
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return TC_ACT_OK;  // 数据包太短，放行

    // 3. 检查协议类型
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return TC_ACT_OK;  // 不是IPv4，放行

    // 4. 解析IP头（MUST 边界检查！）
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return TC_ACT_OK;

    // 5. 现在可以安全访问IP头字段
    bpf_printk("Packet len=%d, proto=%d, src=%pI4\n",
               skb->len, ip->protocol, &ip->saddr);

    return TC_ACT_OK;
}
```

**Verifier 边界检查要求：**

```c
// ❌ 错误：直接访问会被 Verifier 拒绝
struct ethhdr *eth = (void *)(long)skb->data;
__u16 proto = eth->h_proto;  // ERROR: invalid access to packet

// ✅ 正确：先检查边界
struct ethhdr *eth = (void *)(long)skb->data;
if ((void *)(eth + 1) > (void *)(long)skb->data_end)
    return TC_ACT_OK;
__u16 proto = eth->h_proto;  // OK: bounds checked
```

**返回值类型（TC 程序）：**

```c
#define TC_ACT_OK           0   // 放行数据包，继续处理
#define TC_ACT_SHOT         2   // 丢弃数据包
#define TC_ACT_STOLEN       4   // 数据包被窃取（不再处理）
#define TC_ACT_REDIRECT     7   // 重定向到另一接口
```

**`__sk_buff` 快速参考表：**

| 分类 | 字段 | 类型 | 说明 | 需要边界检查？ |
|------|------|------|------|---------------|
| **包信息** | `len` | u32 | 数据包总长度 | ❌ |
| | `protocol` | u32 | 以太网协议类型 | ❌ |
| | `pkt_type` | u32 | 数据包类型 | ❌ |
| | `mark` | u32 | SKB标记 | ❌ |
| **接口** | `ifindex` | u32 | 网卡索引 | ❌ |
| | `ingress_ifindex` | u32 | 入接口索引 | ❌ |
| **数据访问** | `data` | u32 | 数据起始（需转指针） | ✅ 必需！ |
| | `data_end` | u32 | 数据结束（需转指针） | ✅ 必需！ |
| **VLAN** | `vlan_present` | u32 | 是否有VLAN | ❌ |
| | `vlan_tci` | u32 | VLAN标签 | ❌ |
| **时间** | `tstamp` | u64 | 时间戳（纳秒） | ❌ |
| **连接跟踪** | `remote_ip4` | u32 | 远程IPv4 | ❌ |
| | `remote_port` | u32 | 远程端口 | ❌ |

**常见协议类型常量：**

```c
#define ETH_P_IP    0x0800  // IPv4
#define ETH_P_IPV6  0x86DD  // IPv6
#define ETH_P_ARP   0x0806  // ARP
```

**边界检查标准模板：**

```c
// 模板：解析任何协议头
struct XXX_hdr *hdr = (void *)(previous_hdr + 1);  // 或 = data
if ((void *)(hdr + 1) > data_end)  // 检查能否完整读取该头部
    return TC_ACT_OK;              // 无法读取，放行
// 现在可以安全访问 hdr->xxx
```

2. 阅读 libbpf-bootstrap 示例：
   ```bash
   git clone https://github.com/libbpf/libbpf-bootstrap.git
   cd libbpf-bootstrap/examples/c
   # 研究 tc.bpf.c，重点看 __sk_buff 的使用
   ```
   - 时间：1小时

3. 理解TC程序加载过程：
   - 阅读 `man tc-bpf`
   - 理解 clsact qdisc的作用
   - 时间：30分钟

#### ✅ 完成标准

- [ ] Hello World程序成功编译
- [ ] 程序成功加载到TC hook
- [ ] 能在trace_pipe中看到输出
- [ ] 理解 `__sk_buff` 结构体及其核心字段
- [ ] 掌握 `data` 和 `data_end` 的边界检查模式
- [ ] 能手动卸载程序

---

### 📅 Day 3: 数据包解析基础

#### 🎯 任务目标
- 掌握如何在eBPF中解析以太网/IP/TCP头部
- 理解指针边界检查的重要性

#### ✅ 具体任务

**全天 (6-8小时)：实现数据包解析demo**

创建文件 `src/bpf/parse_packet.bpf.c`:

```c
#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <linux/in.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

SEC("tc")
int parse_packet(struct __sk_buff *skb)
{
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // 1. 解析以太网头
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return TC_ACT_OK;

    // 检查是否为IP协议
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return TC_ACT_OK;

    // 2. 解析IP头
    struct iphdr *iph = (void *)(eth + 1);
    if ((void *)(iph + 1) > data_end)
        return TC_ACT_OK;

    __u32 src_ip = bpf_ntohl(iph->saddr);
    __u32 dst_ip = bpf_ntohl(iph->daddr);

    // 3. 解析TCP/UDP头
    if (iph->protocol == IPPROTO_TCP) {
        struct tcphdr *tcph = (void *)iph + (iph->ihl * 4);
        if ((void *)(tcph + 1) > data_end)
            return TC_ACT_OK;

        __u16 src_port = bpf_ntohs(tcph->source);
        __u16 dst_port = bpf_ntohs(tcph->dest);

        bpf_printk("TCP: %pI4:%d -> %pI4:%d\n",
                   &iph->saddr, src_port, &iph->daddr, dst_port);
    }
    else if (iph->protocol == IPPROTO_UDP) {
        struct udphdr *udph = (void *)iph + (iph->ihl * 4);
        if ((void *)(udph + 1) > data_end)
            return TC_ACT_OK;

        __u16 src_port = bpf_ntohs(udph->source);
        __u16 dst_port = bpf_ntohs(udph->dest);

        bpf_printk("UDP: %pI4:%d -> %pI4:%d\n",
                   &iph->saddr, src_port, &iph->daddr, dst_port);
    }

    return TC_ACT_OK;
}

char LICENSE[] SEC("license") = "GPL";
```

创建加载器 `src/user/parse_loader.c`:

```c
#include <stdio.h>
#include <signal.h>
#include <unistd.h>
#include <net/if.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>
#include "parse_packet.skel.h"

static volatile bool exiting = false;
static void sig_handler(int sig) { exiting = true; }

int main(int argc, char **argv) {
    struct parse_packet_bpf *skel;
    int ifindex, err;
    DECLARE_LIBBPF_OPTS(bpf_tc_hook, hook, .attach_point = BPF_TC_INGRESS);
    DECLARE_LIBBPF_OPTS(bpf_tc_opts, opts, .handle = 1, .priority = 1);

    if (argc < 2) {
        fprintf(stderr, "Usage: %s <interface>\n", argv[0]);
        return 1;
    }

    ifindex = if_nametoindex(argv[1]);
    skel = parse_packet_bpf__open_and_load();
    if (!skel) return 1;

    hook.ifindex = ifindex;
    bpf_tc_hook_create(&hook);
    opts.prog_fd = bpf_program__fd(skel->progs.parse_packet);
    bpf_tc_attach(&hook, &opts);

    printf("✓ Packet parser started. Press Ctrl+C to exit.\n");
    signal(SIGINT, sig_handler);
    while (!exiting) sleep(1);

    opts.flags = opts.prog_fd = opts.prog_id = 0;
    bpf_tc_detach(&hook, &opts);
    bpf_tc_hook_destroy(&hook);
    parse_packet_bpf__destroy(skel);
    return 0;
}
```

测试步骤:
```bash
# 1. 编译
make parse_packet.bpf.o
bpftool gen skeleton parse_packet.bpf.o > parse_packet.skel.h
gcc -o parse_loader src/user/parse_loader.c -lbpf -lelf -lz

# 2. 运行加载器
sudo ./parse_loader lo

# 3. 在另一终端生成多种流量测试
# TCP流量
curl http://example.com

# UDP流量
dig @8.8.8.8 google.com

# 4. 在第三个终端观察解析输出
sudo cat /sys/kernel/debug/tracing/trace_pipe
```

#### 📚 学习资料

1. 深入理解网络协议头：
   - 阅读 `/usr/include/linux/if_ether.h`
   - 阅读 `/usr/include/linux/ip.h`
   - 阅读 `/usr/include/linux/tcp.h`
   - 时间：1小时

2. 理解字节序转换：
   - `bpf_ntohs()`, `bpf_ntohl()` 的必要性
   - 网络字节序 vs 主机字节序
   - 时间：30分钟

3. 指针边界检查：
   - 为什么每次指针移动都要检查 `> data_end`
   - Verifier如何验证内存访问安全
   - 时间：30分钟

#### ✅ 完成标准

- [ ] 能正确解析以太网/IP/TCP/UDP头
- [ ] 能输出源IP、目的IP、源端口、目的端口
- [ ] 理解并正确使用指针边界检查
- [ ] 通过不同协议流量测试

---

### 📅 Day 4: 引入BPF Map

#### 🎯 任务目标
- 掌握BPF Map的创建和使用
- 实现基础的数据包计数功能

#### ✅ 具体任务

**全天 (6-8小时)：实现统计Map**

创建文件 `src/bpf/stats_counter.bpf.c`:

```c
#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <bpf/bpf_helpers.h>

// 统计类型
enum {
    STAT_TOTAL_PACKETS = 0,
    STAT_TCP_PACKETS,
    STAT_UDP_PACKETS,
    STAT_ICMP_PACKETS,
    STAT_OTHER_PACKETS,
    STAT_MAX
};

// 定义Per-CPU Array Map用于统计
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, STAT_MAX);
    __type(key, __u32);
    __type(value, __u64);
} stats_map SEC(".maps");

SEC("tc")
int count_packets(struct __sk_buff *skb)
{
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // 更新总包数
    __u32 key = STAT_TOTAL_PACKETS;
    __u64 *count = bpf_map_lookup_elem(&stats_map, &key);
    if (count)
        __sync_fetch_and_add(count, 1);

    // 解析协议
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return TC_ACT_OK;

    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return TC_ACT_OK;

    struct iphdr *iph = (void *)(eth + 1);
    if ((void *)(iph + 1) > data_end)
        return TC_ACT_OK;

    // 按协议统计
    switch (iph->protocol) {
        case IPPROTO_TCP:
            key = STAT_TCP_PACKETS;
            break;
        case IPPROTO_UDP:
            key = STAT_UDP_PACKETS;
            break;
        case IPPROTO_ICMP:
            key = STAT_ICMP_PACKETS;
            break;
        default:
            key = STAT_OTHER_PACKETS;
    }

    count = bpf_map_lookup_elem(&stats_map, &key);
    if (count)
        __sync_fetch_and_add(count, 1);

    return TC_ACT_OK;
}

char LICENSE[] SEC("license") = "GPL";
```

创建 libbpf 加载器 `src/user/stats_loader.c`:

```c
#include <stdio.h>
#include <signal.h>
#include <unistd.h>
#include <net/if.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>
#include "stats_counter.skel.h"

static volatile bool exiting = false;
static void sig_handler(int sig) { exiting = true; }

int main(int argc, char **argv) {
    struct stats_counter_bpf *skel;
    int ifindex, err;
    DECLARE_LIBBPF_OPTS(bpf_tc_hook, hook, .attach_point = BPF_TC_INGRESS);
    DECLARE_LIBBPF_OPTS(bpf_tc_opts, opts, .handle = 1, .priority = 1);

    if (argc < 2) {
        fprintf(stderr, "Usage: %s <interface>\n", argv[0]);
        return 1;
    }

    ifindex = if_nametoindex(argv[1]);
    skel = stats_counter_bpf__open_and_load();
    if (!skel) {
        fprintf(stderr, "Failed to load skeleton\n");
        return 1;
    }

    hook.ifindex = ifindex;
    bpf_tc_hook_create(&hook);
    opts.prog_fd = bpf_program__fd(skel->progs.count_packets);
    bpf_tc_attach(&hook, &opts);

    printf("✓ Stats counter started on %s\n", argv[1]);
    printf("Press Ctrl+C to exit...\n\n");

    // 3. 循环读取统计
    while (1) {
        sleep(2);

        printf("\033[2J\033[H");  // 清屏
        printf("=== eBPF Packet Statistics ===\n\n");

        __u64 values[5] = {0};
        __u32 key;
        int num_cpus = libbpf_num_possible_cpus();
        __u64 *percpu_values = calloc(num_cpus, sizeof(__u64));

        // 读取各项统计
        const char *stat_names[] = {
            "Total", "TCP", "UDP", "ICMP", "Other"
        };

        for (int i = 0; i < 5; i++) {
            key = i;
            err = bpf_map_lookup_elem(bpf_map__fd(skel->maps.stats_map),
                                      &key, percpu_values);
            if (err == 0) {
                for (int cpu = 0; cpu < num_cpus; cpu++)
                    values[i] += percpu_values[cpu];
            }

            printf("%-10s: %llu packets\n", stat_names[i], values[i]);
        }

        free(percpu_values);
    }

    // 清理
    opts.flags = opts.prog_fd = opts.prog_id = 0;
    bpf_tc_detach(&hook, &opts);
    bpf_tc_hook_destroy(&hook);
    stats_counter_bpf__destroy(skel);
    return 0;
}
```

更新Makefile添加skeleton生成:

```makefile
stats_counter.bpf.o: src/bpf/stats_counter.bpf.c
	$(CLANG) $(BPF_CFLAGS) -c $< -o $@

stats_counter.skel.h: stats_counter.bpf.o
	$(BPFTOOL) gen skeleton $< > $@

stats_loader: src/user/stats_loader.c stats_counter.skel.h
	$(CC) -g -Wall -I. $< -lbpf -lelf -lz -o $@
```

测试:
```bash
# 编译（自动生成 skeleton）
make stats_counter.bpf.o
bpftool gen skeleton stats_counter.bpf.o > stats_counter.skel.h
gcc -o stats_loader src/user/stats_loader.c -I. -lbpf -lelf -lz

# 运行（自动加载+显示统计，Ctrl+C 自动卸载）
sudo ./stats_loader lo

# 在另一终端生成流量
ping 127.0.0.1 &
curl http://localhost &
```

#### 📚 学习资料

1. 深入理解BPF Map:
   - 阅读内核文档 `linux/Documentation/bpf/maps.rst`
   - 重点理解 PERCPU_ARRAY 的优势（无锁）
   - 时间：1.5小时

2. libbpf skeleton机制:
   - 阅读 [libbpf README](https://github.com/libbpf/libbpf/blob/master/README.md)
   - 理解skeleton如何简化用户态代码
   - 时间：1小时

#### ✅ 完成标准

- [ ] 成功创建和使用BPF Map
- [ ] 用户态程序能正确读取统计数据
- [ ] 理解Per-CPU Map的无锁优势
- [ ] 能实时显示数据包统计

---

### 📅 Day 5: 实现5元组匹配demo

#### 🎯 任务目标
- 实现基于5元组的数据包过滤
- 理解HASH Map的使用

#### ✅ 具体任务

**全天 (6-8小时)：实现5元组策略匹配**

创建文件 `src/bpf/五tuple_filter.bpf.c`:

```c
#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <linux/if_ether.h>
#include <linux/ip.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <bpf/bpf_helpers.h>
#include <bpf/bpf_endian.h>

// 5元组key
struct flow_key {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;
} __attribute__((packed));

// 策略动作
enum {
    ACTION_ALLOW = 0,
    ACTION_DENY = 1,
};

// 策略value
struct policy_value {
    __u8 action;
    __u64 packet_count;
    __u64 byte_count;
} __attribute__((packed));

// 策略Map
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 10000);
    __type(key, struct flow_key);
    __type(value, struct policy_value);
} policy_map SEC(".maps");

// 统计Map
struct {
    __uint(type, BPF_MAP_TYPE_PERCPU_ARRAY);
    __uint(max_entries, 3);
    __type(key, __u32);
    __type(value, __u64);
} stats_map SEC(".maps");

enum {
    STAT_TOTAL = 0,
    STAT_ALLOWED = 1,
    STAT_DENIED = 2,
};

static __always_inline void update_stat(__u32 stat_key)
{
    __u64 *count = bpf_map_lookup_elem(&stats_map, &stat_key);
    if (count)
        __sync_fetch_and_add(count, 1);
}

SEC("tc")
int filter_packets(struct __sk_buff *skb)
{
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    update_stat(STAT_TOTAL);

    // 解析以太网头
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return TC_ACT_OK;

    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return TC_ACT_OK;

    // 解析IP头
    struct iphdr *iph = (void *)(eth + 1);
    if ((void *)(iph + 1) > data_end)
        return TC_ACT_OK;

    // 构建5元组key
    struct flow_key key = {
        .src_ip = iph->saddr,
        .dst_ip = iph->daddr,
        .protocol = iph->protocol,
    };

    // 解析端口
    void *l4 = (void *)iph + (iph->ihl * 4);

    if (iph->protocol == IPPROTO_TCP) {
        struct tcphdr *tcph = l4;
        if ((void *)(tcph + 1) > data_end)
            return TC_ACT_OK;
        key.src_port = tcph->source;
        key.dst_port = tcph->dest;
    }
    else if (iph->protocol == IPPROTO_UDP) {
        struct udphdr *udph = l4;
        if ((void *)(udph + 1) > data_end)
            return TC_ACT_OK;
        key.src_port = udph->source;
        key.dst_port = udph->dest;
    }
    else {
        // 其他协议：ICMP等
        key.src_port = 0;
        key.dst_port = 0;
    }

    // 查找策略
    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, &key);
    if (policy) {
        // 更新策略统计
        __sync_fetch_and_add(&policy->packet_count, 1);
        __sync_fetch_and_add(&policy->byte_count, skb->len);

        if (policy->action == ACTION_DENY) {
            update_stat(STAT_DENIED);
            bpf_printk("DENY: %pI4:%d -> %pI4:%d\n",
                       &key.src_ip, bpf_ntohs(key.src_port),
                       &key.dst_ip, bpf_ntohs(key.dst_port));
            return TC_ACT_SHOT;  // 丢弃
        }
    }

    update_stat(STAT_ALLOWED);
    return TC_ACT_OK;  // 放行
}

char LICENSE[] SEC("license") = "GPL";
```

创建策略管理工具 `src/user/policy_mgmt.c`:

```c
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <arpa/inet.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>

struct flow_key {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;
} __attribute__((packed));

struct policy_value {
    __u8 action;
    __u64 packet_count;
    __u64 byte_count;
} __attribute__((packed));

int main(int argc, char **argv)
{
    if (argc < 2) {
        printf("Usage: %s <add|list|del> [args...]\n", argv[0]);
        printf("  add <src_ip> <dst_ip> <dst_port> <proto> <allow|deny>\n");
        printf("  list\n");
        return 1;
    }

    // 打开Map
    int map_fd = bpf_obj_get("/sys/fs/bpf/policy_map");
    if (map_fd < 0) {
        perror("bpf_obj_get");
        return 1;
    }

    if (strcmp(argv[1], "add") == 0) {
        if (argc != 7) {
            printf("Usage: add <src_ip> <dst_ip> <dst_port> <proto> <allow|deny>\n");
            return 1;
        }

        struct flow_key key = {0};
        struct policy_value value = {0};

        inet_pton(AF_INET, argv[2], &key.src_ip);
        inet_pton(AF_INET, argv[3], &key.dst_ip);
        key.dst_port = htons(atoi(argv[4]));

        if (strcmp(argv[5], "tcp") == 0)
            key.protocol = 6;
        else if (strcmp(argv[5], "udp") == 0)
            key.protocol = 17;

        value.action = (strcmp(argv[6], "deny") == 0) ? 1 : 0;

        if (bpf_map_update_elem(map_fd, &key, &value, BPF_ANY) < 0) {
            perror("bpf_map_update_elem");
            return 1;
        }

        printf("✓ Policy added\n");
    }
    else if (strcmp(argv[1], "list") == 0) {
        struct flow_key key, next_key;
        struct policy_value value;

        printf("=== Policy List ===\n");
        printf("%-15s %-15s %-6s %-5s %-6s %10s %10s\n",
               "SRC_IP", "DST_IP", "PORT", "PROTO", "ACTION", "PACKETS", "BYTES");

        memset(&key, 0, sizeof(key));
        while (bpf_map_get_next_key(map_fd, &key, &next_key) == 0) {
            bpf_map_lookup_elem(map_fd, &next_key, &value);

            char src_ip[16], dst_ip[16];
            inet_ntop(AF_INET, &next_key.src_ip, src_ip, sizeof(src_ip));
            inet_ntop(AF_INET, &next_key.dst_ip, dst_ip, sizeof(dst_ip));

            printf("%-15s %-15s %-6d %-5d %-6s %10llu %10llu\n",
                   src_ip, dst_ip, ntohs(next_key.dst_port),
                   next_key.protocol,
                   value.action == 1 ? "DENY" : "ALLOW",
                   value.packet_count, value.byte_count);

            key = next_key;
        }
    }

    return 0;
}
```

创建 libbpf 加载器 `src/user/fivetuple_loader.c`:

```c
#include <stdio.h>
#include <signal.h>
#include <unistd.h>
#include <net/if.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>
#include "fivetuple_filter.skel.h"

static volatile bool exiting = false;
static void sig_handler(int sig) { exiting = true; }

int main(int argc, char **argv) {
    struct fivetuple_filter_bpf *skel;
    int ifindex, err;
    DECLARE_LIBBPF_OPTS(bpf_tc_hook, hook, .attach_point = BPF_TC_INGRESS);
    DECLARE_LIBBPF_OPTS(bpf_tc_opts, opts, .handle = 1, .priority = 1);

    if (argc < 2) {
        fprintf(stderr, "Usage: %s <interface>\n", argv[0]);
        return 1;
    }

    ifindex = if_nametoindex(argv[1]);
    skel = fivetuple_filter_bpf__open_and_load();
    if (!skel) return 1;

    hook.ifindex = ifindex;
    bpf_tc_hook_create(&hook);
    opts.prog_fd = bpf_program__fd(skel->progs.filter_packets);
    bpf_tc_attach(&hook, &opts);

    // Pin maps 供外部访问
    bpf_map__pin(skel->maps.policy_map, "/sys/fs/bpf/policy_map");
    bpf_map__pin(skel->maps.stats_map, "/sys/fs/bpf/stats_map");

    printf("✓ 5-tuple filter started on %s\n", argv[1]);
    printf("Press Ctrl+C to exit...\n");

    signal(SIGINT, sig_handler);
    while (!exiting) sleep(1);

    opts.flags = opts.prog_fd = opts.prog_id = 0;
    bpf_tc_detach(&hook, &opts);
    bpf_tc_hook_destroy(&hook);
    fivetuple_filter_bpf__destroy(skel);
    return 0;
}
```

测试流程:
```bash
# 1. 编译
make fivetuple_filter.bpf.o
bpftool gen skeleton fivetuple_filter.bpf.o > fivetuple_filter.skel.h
gcc -o fivetuple_loader src/user/fivetuple_loader.c -lbpf -lelf -lz
gcc -o policy_mgmt src/user/policy_mgmt.c -lbpf

# 2. 运行 libbpf 加载器（后台）
sudo ./fivetuple_loader lo &
LOADER_PID=$!
sleep 2

# 3. 添加策略
# 拒绝到本地22端口的SSH连接
sudo ./policy_mgmt add 0.0.0.0 127.0.0.1 22 tcp deny

# 允许到本地80端口的HTTP连接
sudo ./policy_mgmt add 0.0.0.0 127.0.0.1 80 tcp allow

# 4. 列出策略
sudo ./policy_mgmt list

# 5. 测试
# 应该被拒绝
telnet 127.0.0.1 22

# 应该被允许
curl http://127.0.0.1

# 6. 查看统计
sudo bpftool map dump name stats_map

# 7. 清理
sudo kill $LOADER_PID
```

#### 📚 学习资料

1. HASH Map深入理解:
   - 时间复杂度 O(1)
   - 哈希冲突处理
   - 与LRU_HASH的区别
   - 时间：1小时

2. 策略匹配优化:
   - 通配符匹配的实现方式
   - LPM Trie用于IP段匹配
   - 时间：1小时

#### ✅ 完成标准

- [ ] 成功实现5元组策略匹配
- [ ] 能动态添加/删除/查看策略
- [ ] 策略能正确执行（允许/拒绝）
- [ ] 能统计每条策略的包数和字节数

---

### 📅 本周总结 (Friday晚上)

#### ✍️ 输出物

创建学习总结文档 `docs/week1_summary.md`:

```markdown
# 第1周学习总结

## 完成情况

- [x] 开发环境搭建
- [x] Hello World eBPF程序
- [x] 数据包解析demo
- [x] BPF Map统计功能
- [x] 5元组策略匹配

## 核心收获

### 1. eBPF基础概念
- Verifier的安全验证机制
- 指针边界检查的必要性

### 2. TC Hook机制
- Ingress/Egress的区别
- clsact qdisc的作用
- 返回值的含义

### 3. BPF Map类型
- PERCPU_ARRAY: 无锁统计
- HASH: O(1)查找
- Map的pin机制用于持久化

### 4. 遇到的问题和解决

**问题1**: Verifier报错 "invalid access to packet"
- **原因**: 没有检查指针边界
- **解决**: 每次指针操作前都检查 `> data_end`

**问题2**: Map lookup总是返回NULL
- **原因**: Map没有正确pin到bpffs
- **解决**: 使用 `bpftool map pin` 命令

## 下周计划

- 实现会话跟踪（LRU_HASH）
- 添加TCP状态机
- 完善策略管理CLI工具
- 性能测试和优化
```

#### 🎯 本周验收标准

**必须完成**:
- [ ] 所有Day 1-5的任务都打勾
- [ ] 能独立编译、加载、测试eBPF程序
- [ ] 5元组策略匹配demo能正常工作
- [ ] 提交学习总结文档

**加分项**:
- [ ] 写了详细的代码注释
- [ ] 绘制了eBPF数据流图
- [ ] 对比了不同Map类型的性能

---

## 2.6 开发环境准备

#### 系统要求
```bash
# 检查内核版本 (需要 >= 5.10)
uname -r

# 检查eBPF支持
zgrep CONFIG_BPF /proc/config.gz
zgrep CONFIG_BPF_SYSCALL /proc/config.gz
```

#### 安装依赖
```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y \
    clang \
    llvm \
    libbpf-dev \
    linux-headers-$(uname -r) \
    build-essential \
    pkg-config

# 安装bpftool
sudo apt-get install -y linux-tools-$(uname -r)
```

### 2.3 项目结构

```
ebpf-microsegment/
├── src/
│   ├── bpf/
│   │   ├── tc_microsegment.bpf.c    # eBPF程序
│   │   ├── common.h                 # 公共头文件
│   │   └── maps.h                   # Map定义
│   ├── user/
│   │   ├── main.c                   # 主程序
│   │   ├── policy.c                 # 策略管理
│   │   ├── session.c                # 会话管理
│   │   └── stats.c                  # 统计信息
│   └── include/
│       └── types.h                  # 类型定义
├── tests/
│   ├── unit/                        # 单元测试
│   └── integration/                 # 集成测试
├── scripts/
│   ├── setup.sh                     # 环境设置
│   ├── load.sh                      # 加载eBPF程序
│   └── unload.sh                    # 卸载eBPF程序
├── Makefile
└── README.md
```

### 2.4 基础Makefile

```makefile
CLANG ?= clang
LLC ?= llc
BPFTOOL ?= bpftool

ARCH := $(shell uname -m | sed 's/x86_64/x86/' | sed 's/aarch64/arm64/')

BPF_CFLAGS = -g -O2 -target bpf -D__TARGET_ARCH_$(ARCH)
BPF_INCLUDES = -I/usr/include/$(shell uname -m)-linux-gnu

BPF_SRC = src/bpf/tc_microsegment.bpf.c
BPF_OBJ = tc_microsegment.bpf.o
BPF_SKEL = tc_microsegment.skel.h

USER_SRC = src/user/main.c src/user/policy.c src/user/session.c src/user/stats.c
USER_BIN = tc_microsegment

.PHONY: all clean

all: $(BPF_OBJ) $(BPF_SKEL) $(USER_BIN)

# 编译eBPF程序
$(BPF_OBJ): $(BPF_SRC)
	$(CLANG) $(BPF_CFLAGS) $(BPF_INCLUDES) -c $< -o $@

# 生成skeleton
$(BPF_SKEL): $(BPF_OBJ)
	$(BPFTOOL) gen skeleton $< > $@

# 编译用户态程序
$(USER_BIN): $(USER_SRC) $(BPF_SKEL)
	$(CC) -g -Wall -I. $(USER_SRC) -lbpf -lelf -lz -o $@

clean:
	rm -f $(BPF_OBJ) $(BPF_SKEL) $(USER_BIN)
```

### 2.5 验证清单

- [ ] eBPF程序成功编译
- [ ] 用户态程序成功编译
- [ ] TC hook成功附加
- [ ] 数据包能被eBPF程序处理
- [ ] 基础策略匹配工作正常

---


---

**[⬅️ 返回目录](./README.md)** | **[➡️ 第2周](./week2-basic-framework.md)**
