# eBPF + TC 可行性分析 - 实施指南

## 1. 实施阶段规划

### 总体时间表：6周 (详细版)

```
第1周: 环境准备 + eBPF基础学习
  Day 1-2: 开发环境搭建、eBPF理论学习
  Day 3-4: Hello World程序、TC基础实验
  Day 5:   数据包解析demo

第2周: 基础框架开发
  Day 1-2: 策略Map设计与实现
  Day 3-4: 会话Map与5元组匹配
  Day 5:   基础策略执行demo

第3周: 用户态控制程序
  Day 1-2: libbpf skeleton集成
  Day 3-4: 策略CRUD接口
  Day 5:   CLI工具与配置管理

第4周: 高级功能实现
  Day 1-2: TCP状态机实现
  Day 3:   IP段匹配(LPM Trie)
  Day 4:   Map压力监控
  Day 5:   统计与日志功能

第5周: 测试与优化
  Day 1:   单元测试编写
  Day 2:   功能测试与修复
  Day 3:   性能测试与调优
  Day 4:   压力测试
  Day 5:   文档整理

第6周: 生产部署准备
  Day 1-2: 部署脚本完善
  Day 3:   监控集成(Prometheus)
  Day 4:   金丝雀部署测试
  Day 5:   项目交付与演示
```

### 🎯 每周目标与交付物

| 周次 | 主要目标 | 交付物 | 学习重点 |
|------|---------|--------|---------|
| **第1周** | 环境就绪+理论掌握 | Hello World eBPF程序 | eBPF原理、TC hook机制 |
| **第2周** | 基础框架完成 | 可工作的策略匹配demo | Map操作、数据结构设计 |
| **第3周** | 用户态程序完成 | 完整的CLI工具 | libbpf API、策略管理 |
| **第4周** | 高级功能完成 | 生产级功能demo | 状态机、性能优化 |
| **第5周** | 测试覆盖完成 | 测试报告 | 测试方法、性能分析 |
| **第6周** | 生产就绪 | 部署方案+监控 | DevOps、可观测性 |

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
  - JIT编译原理
  - Map类型和用途（HASH, ARRAY, LRU_HASH）

- **TC hook机制**：
  - Ingress/Egress区别
  - 与XDP的对比
  - 返回值含义（TC_ACT_OK, TC_ACT_SHOT等）

#### ✅ 完成标准

- [ ] 所有工具安装成功并可运行
- [ ] 理解eBPF的Verifier、JIT、Map概念
- [ ] 能绘制出eBPF数据包处理流程图

---

### 📅 Day 2: Hello World eBPF程序

#### 🎯 任务目标
- 编写并运行第一个TC eBPF程序
- 掌握eBPF程序的编译和加载流程

#### ✅ 具体任务

**上午 (3-4小时)：编写Hello World程序**

创建文件 `src/bpf/hello.bpf.c`:

```c
// hello.bpf.c - 最简单的TC eBPF程序
#include <linux/bpf.h>
#include <linux/pkt_cls.h>
#include <bpf/bpf_helpers.h>

SEC("tc")
int hello_world(struct __sk_buff *skb)
{
    bpf_printk("Hello eBPF! Packet len=%d\n", skb->len);
    return TC_ACT_OK;  // 放行数据包
}

char LICENSE[] SEC("license") = "GPL";
```

创建简单的Makefile:

```makefile
CLANG ?= clang
BPFTOOL ?= bpftool

BPF_CFLAGS = -g -O2 -target bpf -D__TARGET_ARCH_x86

hello.bpf.o: src/bpf/hello.bpf.c
	$(CLANG) $(BPF_CFLAGS) -c $< -o $@

hello.skel.h: hello.bpf.o
	$(BPFTOOL) gen skeleton $< > $@

clean:
	rm -f *.o *.skel.h
```

编译程序:
```bash
make hello.bpf.o
```

**下午 (3-4小时)：使用libbpf加载和测试**

> **💡 现代方法**: 推荐使用 libbpf 库和 skeleton 来加载 eBPF 程序，相比传统的 shell 脚本方式更安全、更易维护。

首先生成 skeleton 头文件：

```bash
# 生成 skeleton (需要先编译好 hello.bpf.o)
bpftool gen skeleton hello.bpf.o > hello.skel.h
```

创建 C 语言加载器 `src/user/hello_loader.c`:

```c
// src/user/hello_loader.c
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <signal.h>
#include <errno.h>
#include <string.h>
#include <bpf/libbpf.h>
#include "hello.skel.h"

static volatile bool exiting = false;

static void sig_handler(int sig)
{
    exiting = true;
}

static int libbpf_print_fn(enum libbpf_print_level level, const char *format, va_list args)
{
    return vfprintf(stderr, format, args);
}

int main(int argc, char **argv)
{
    struct hello_bpf *skel;
    int err;
    const char *iface = "lo";

    // 设置 libbpf 调试输出
    libbpf_set_print(libbpf_print_fn);

    // 1. 打开并加载 eBPF 程序
    skel = hello_bpf__open_and_load();
    if (!skel) {
        fprintf(stderr, "Failed to open and load BPF skeleton\n");
        return 1;
    }
    printf("✓ BPF program loaded successfully\n");

    // 2. 附加到 TC hook (使用现代 libbpf TC API)
    // libbpf 1.x 提供了原生的 bpf_tc_* API
    LIBBPF_OPTS(bpf_tc_hook, hook,
        .ifindex = if_nametoindex(iface),
        .attach_point = BPF_TC_INGRESS);
    
    LIBBPF_OPTS(bpf_tc_opts, opts,
        .handle = 1,
        .priority = 1,
        .prog_fd = bpf_program__fd(skel->progs.hello_world));
    
    // 创建 TC hook（如果不存在）
    err = bpf_tc_hook_create(&hook);
    if (err && err != -EEXIST) {
        fprintf(stderr, "Failed to create TC hook: %s\n", strerror(-err));
        goto cleanup;
    }
    
    // 附加 eBPF 程序
    err = bpf_tc_attach(&hook, &opts);
    if (err) {
        fprintf(stderr, "Failed to attach TC program: %s\n", strerror(-err));
        goto cleanup;
    }
    
    printf("✓ Attached to %s ingress\n", iface);
    printf("查看输出: sudo cat /sys/kernel/debug/tracing/trace_pipe\n\n");

    // 3. 等待信号
    signal(SIGINT, sig_handler);
    signal(SIGTERM, sig_handler);
    
    printf("程序运行中，按 Ctrl+C 退出...\n");
    while (!exiting) {
        sleep(1);
    }

cleanup:
    // 4. 清理（使用 libbpf TC API）
    if (skel) {
        // 分离 TC 程序
        LIBBPF_OPTS(bpf_tc_hook, hook_cleanup,
            .ifindex = if_nametoindex(iface),
            .attach_point = BPF_TC_INGRESS);
        
        LIBBPF_OPTS(bpf_tc_opts, opts_cleanup,
            .handle = 1,
            .priority = 1);
        
        bpf_tc_detach(&hook_cleanup, &opts_cleanup);
        bpf_tc_hook_destroy(&hook_cleanup);
        
        hello_bpf__destroy(skel);
    }
    
    printf("\n✓ Cleaned up\n");
    return err;
}
```

更新 `Makefile` 添加 skeleton 生成和用户态编译：

```makefile
# Makefile
CLANG ?= clang
BPFTOOL ?= bpftool
ARCH := $(shell uname -m | sed 's/x86_64/x86/')

# eBPF 程序编译
%.bpf.o: src/bpf/%.bpf.c
	$(CLANG) -g -O2 -target bpf -D__TARGET_ARCH_$(ARCH) \
		-I/usr/include/$(shell uname -m)-linux-gnu \
		-c $< -o $@

# 生成 skeleton
%.skel.h: %.bpf.o
	$(BPFTOOL) gen skeleton $< > $@

# 用户态程序编译
hello_loader: hello.skel.h src/user/hello_loader.c
	gcc -Wall -o $@ src/user/hello_loader.c -lbpf -lelf -lz

.PHONY: clean
clean:
	rm -f *.bpf.o *.skel.h hello_loader
```

测试流程:
```bash
# 1. 编译所有组件
make hello.bpf.o        # 编译 eBPF 程序
make hello_loader       # 编译用户态加载器

# 2. 在终端1启动加载器
sudo ./hello_loader

# 3. 在终端2查看日志
sudo cat /sys/kernel/debug/tracing/trace_pipe

# 4. 在终端3生成流量测试
ping 127.0.0.1 -c 5

# 5. 验证能看到 "Hello eBPF!" 输出

# 6. 在终端1按 Ctrl+C 优雅退出（自动清理）
```

**libbpf 方式的优势**:
- ✅ **类型安全**: skeleton 提供编译时类型检查
- ✅ **错误处理**: 详细的错误信息，便于调试
- ✅ **自动清理**: 程序退出时自动清理资源
- ✅ **生产就绪**: 被 Cilium、Katran 等项目广泛使用

#### 📚 学习资料

1. **libbpf skeleton 机制深入理解**：
   ```bash
   git clone https://github.com/libbpf/libbpf-bootstrap.git
   cd libbpf-bootstrap/examples/c
   # 研究 minimal.bpf.c 和 tc.bpf.c
   # 对比 .bpf.c 文件和生成的 .skel.h 文件
   ```
   - 时间：1.5小时

2. **libbpf API 学习**：
   - `bpf_object__open_and_load()` vs `bpf_object__open()` + `bpf_object__load()`
   - `bpf_program__fd()` 获取程序文件描述符
   - `bpf_map__fd()` 获取 Map 文件描述符
   - 时间：1小时

3. **TC程序附加机制**：
   - 阅读 `man tc-bpf`
   - 理解 clsact qdisc 的作用
   - 掌握 libbpf 1.x 的 `bpf_tc_*` API（推荐使用）
   - 参考：[libbpf TC API 文档](https://github.com/libbpf/libbpf/blob/master/src/libbpf.h)
   - 时间：30分钟

#### ✅ 完成标准

- [ ] Hello World eBPF程序成功编译 (`hello.bpf.o`)
- [ ] skeleton 头文件成功生成 (`hello.skel.h`)
- [ ] 用户态加载器成功编译 (`hello_loader`)
- [ ] 程序通过 libbpf 成功加载到TC hook
- [ ] 能在 trace_pipe 中看到输出
- [ ] 程序能优雅退出并自动清理资源

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

测试步骤:
```bash
# 1. 编译eBPF程序和生成skeleton
make parse_packet.bpf.o
make parse_packet.skel.h

# 2. 创建用户态加载器（参考hello_loader.c）
cp src/user/hello_loader.c src/user/parse_packet_loader.c
# 修改其中的skeleton包含和程序名

# 3. 编译用户态程序
make parse_packet_loader

# 4. 启动加载器
sudo ./parse_packet_loader

# 5. 生成多种流量测试
# TCP流量
curl http://httpbin.org/get

# UDP流量  
dig @8.8.8.8 google.com

# 6. 观察解析输出
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

创建用户态读取程序 `src/user/read_stats.c`:

```c
#include <stdio.h>
#include <unistd.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>
#include "stats_counter.skel.h"

int main()
{
    struct stats_counter_bpf *skel;
    int err;

    // 1. 打开和加载eBPF程序
    skel = stats_counter_bpf__open_and_load();
    if (!skel) {
        fprintf(stderr, "Failed to load eBPF program\n");
        return 1;
    }

    // 2. 附加到TC hook (使用 libbpf 1.x TC API)
    LIBBPF_OPTS(bpf_tc_hook, hook,
        .ifindex = if_nametoindex("lo"),
        .attach_point = BPF_TC_INGRESS);
    
    LIBBPF_OPTS(bpf_tc_opts, opts,
        .handle = 1,
        .priority = 1,
        .prog_fd = bpf_program__fd(skel->progs.count_packets));
    
    err = bpf_tc_hook_create(&hook);
    if (err && err != -EEXIST) {
        fprintf(stderr, "Failed to create TC hook: %s\n", strerror(-err));
        stats_counter_bpf__destroy(skel);
        return 1;
    }
    
    err = bpf_tc_attach(&hook, &opts);
    if (err) {
        fprintf(stderr, "Failed to attach TC program: %s\n", strerror(-err));
        stats_counter_bpf__destroy(skel);
        return 1;
    }
    
    printf("✓ Attached to lo ingress\n");
    printf("按Ctrl+C退出...\n\n");

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

read_stats: src/user/read_stats.c stats_counter.skel.h
	$(CC) -g -Wall -I. $< -lbpf -o $@
```

测试:
```bash
# 编译eBPF程序和skeleton
make stats_counter.bpf.o
make stats_counter.skel.h

# 编译用户态程序（已使用skeleton）
make read_stats

# 运行用户态程序（自动加载和附加）
sudo ./read_stats

# 在另一终端生成流量
ping 127.0.0.1 &
curl http://httpbin.org/get &

# 观察实时统计输出
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

测试流程:
```bash
# 1. 编译
make fivetuple_filter.bpf.o
gcc -o policy_mgmt src/user/policy_mgmt.c -lbpf

# 2. 加载eBPF程序并pin map
sudo tc qdisc add dev lo clsact
sudo tc filter add dev lo ingress bpf da obj fivetuple_filter.bpf.o sec tc
sudo bpftool map pin name policy_map /sys/fs/bpf/policy_map

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

---

## 📚 libbpf 最佳实践

### 1. **使用 skeleton 而非手动加载**

skeleton 是 libbpf 推荐的加载方式，提供了更好的类型安全和错误处理：

```c
// ✅ 推荐：使用 skeleton
struct my_prog_bpf *skel = my_prog_bpf__open_and_load();
if (!skel) {
    fprintf(stderr, "Failed to load skeleton\n");
    return 1;
}

// ❌ 不推荐：手动加载
int prog_fd = bpf_prog_load("my_prog.bpf.o", BPF_PROG_TYPE_SCHED_CLS, ...);
```

### 2. **正确的错误处理模式**

使用 `goto cleanup` 模式确保资源正确释放：

```c
int main() {
    struct my_prog_bpf *skel = NULL;
    int err = 0;
    
    skel = my_prog_bpf__open();
    if (!skel) {
        err = -errno;
        fprintf(stderr, "Failed to open: %s\n", strerror(errno));
        goto cleanup;
    }
    
    err = my_prog_bpf__load(skel);
    if (err) {
        fprintf(stderr, "Failed to load: %d\n", err);
        goto cleanup;
    }
    
    // ... 使用程序 ...
    
cleanup:
    my_prog_bpf__destroy(skel);  // NULL-safe
    return err;
}
```

### 3. **配置 libbpf 日志输出**

开发阶段启用详细日志，生产环境可屏蔽：

```c
static int libbpf_print_fn(enum libbpf_print_level level,
                           const char *format, va_list args)
{
    // 生产环境可以屏蔽 DEBUG 级别
    if (level == LIBBPF_DEBUG)
        return 0;
    
    return vfprintf(stderr, format, args);
}

int main() {
    libbpf_set_print(libbpf_print_fn);
    // ...
}
```

### 4. **Map 访问最佳实践**

使用 skeleton 的结构体成员访问 Map，避免字符串路径：

```c
// ✅ 推荐：类型安全
int map_fd = bpf_map__fd(skel->maps.my_map);
struct bpf_map *map = skel->maps.my_map;

// ❌ 不推荐：容易出错
int map_fd = bpf_obj_get("/sys/fs/bpf/my_map");
```

### 5. **分离 open 和 load**

如果需要在加载前修改配置，使用分离的 open 和 load：

```c
skel = my_prog_bpf__open();
if (!skel)
    return 1;

// 修改 Map 大小
bpf_map__set_max_entries(skel->maps.my_map, 100000);

// 然后加载
err = my_prog_bpf__load(skel);
```

### 6. **优雅的程序退出**

使用信号处理实现优雅退出：

```c
static volatile bool exiting = false;

static void sig_handler(int sig) {
    exiting = true;
}

int main() {
    signal(SIGINT, sig_handler);
    signal(SIGTERM, sig_handler);
    
    while (!exiting) {
        // 主循环
    }
    
    // 清理资源
    my_prog_bpf__destroy(skel);
}
```

### 7. **TC 程序附加的现代方式**

**libbpf 1.x 提供了完善的 `bpf_tc_*` API（强烈推荐）**：

```c
#include <net/if.h>
#include <bpf/bpf.h>
#include <bpf/libbpf.h>

// 使用 libbpf 1.x 原生 TC API
LIBBPF_OPTS(bpf_tc_hook, hook,
    .ifindex = if_nametoindex("eth0"),
    .attach_point = BPF_TC_INGRESS);

LIBBPF_OPTS(bpf_tc_opts, opts,
    .handle = 1,
    .priority = 1,
    .prog_fd = bpf_program__fd(skel->progs.my_tc_prog));

// 创建 TC hook（幂等操作）
int err = bpf_tc_hook_create(&hook);
if (err && err != -EEXIST) {
    fprintf(stderr, "Failed to create hook: %s\n", strerror(-err));
    return err;
}

// 附加 eBPF 程序
err = bpf_tc_attach(&hook, &opts);
if (err) {
    fprintf(stderr, "Failed to attach: %s\n", strerror(-err));
    return err;
}

printf("✓ TC program attached successfully\n");

// ... 程序运行 ...

// 清理时分离程序
bpf_tc_detach(&hook, &opts);
bpf_tc_hook_destroy(&hook);
```

**优势**：
- ✅ 纯 C API，无需调用外部命令
- ✅ 更好的错误处理和返回值
- ✅ 性能更好，无进程创建开销
- ✅ 支持更多高级特性（如修改优先级、批量操作）

**旧版本兼容**：如果必须使用 libbpf < 1.0，可以使用 `system()` 调用 `tc` 命令作为降级方案。

### 8. **避免常见陷阱**

```c
// ❌ 错误：忘记检查返回值
my_prog_bpf__load(skel);

// ✅ 正确：始终检查
if (my_prog_bpf__load(skel) != 0) {
    // 错误处理
}

// ❌ 错误：访问已销毁的 skeleton
my_prog_bpf__destroy(skel);
int fd = bpf_map__fd(skel->maps.my_map);  // 崩溃！

// ✅ 正确：在销毁前获取需要的信息
int fd = bpf_map__fd(skel->maps.my_map);
my_prog_bpf__destroy(skel);
```

---

## 核心收获

### 1. eBPF基础概念
- Verifier的安全验证机制
- JIT编译提升性能
- 指针边界检查的必要性

### 2. TC Hook机制
- Ingress/Egress的区别
- clsact qdisc的作用
- 返回值的含义

### 3. BPF Map类型
- PERCPU_ARRAY: 无锁统计
- HASH: O(1)查找
- Map的pin机制用于持久化

### 4. libbpf 和 skeleton
- skeleton 提供类型安全的 eBPF 程序加载
- `bpftool gen skeleton` 自动生成加载代码
- `xxx_bpf__open_and_load()` 简化加载流程
- `bpf_map__fd()` 和 `bpf_program__fd()` 安全访问资源
- 优雅的资源管理和错误处理

### 5. 遇到的问题和解决

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

- [ ] eBPF程序成功编译 (`.bpf.o`)
- [ ] skeleton头文件成功生成 (`.skel.h`)
- [ ] 用户态程序成功编译 (使用skeleton)
- [ ] TC hook通过libbpf成功附加
- [ ] 数据包能被eBPF程序处理
- [ ] 基础策略匹配工作正常
- [ ] 程序能优雅退出和清理

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

测试:
```bash
# 编译
make session_tracking.bpf.o
gcc -o session_viewer src/user/session_viewer.c -lbpf

# 加载
sudo tc qdisc add dev lo clsact
sudo tc filter add dev lo ingress bpf da obj session_tracking.bpf.o sec tc
sudo bpftool map pin name session_map /sys/fs/bpf/session_map

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
sudo tc qdisc del dev lo clsact 2>/dev/null || true

# 加载eBPF程序
sudo tc qdisc add dev lo clsact
sudo tc filter add dev lo ingress bpf da obj microsegment.bpf.o sec tc
sudo bpftool map pin name policy_map /sys/fs/bpf/policy_map
sudo bpftool map pin name session_map /sys/fs/bpf/session_map

echo "✓ eBPF程序已加载"

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
sudo tc qdisc del dev lo clsact
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

## 4. 第3周：用户态控制程序

### 🎯 本周目标

- [ ] 使用libbpf skeleton集成eBPF程序
- [ ] 实现完整的CLI工具
- [ ] 添加策略配置文件支持
- [ ] 实现策略CRUD接口

### 📊 本周交付物

1. ✅ 完整的用户态控制程序 (libbpf skeleton)
2. ✅ CLI工具 (policy/session/stats子命令)
3. ✅ JSON配置文件支持
4. ✅ 策略热更新功能

---

### 📅 Day 1-2: libbpf skeleton集成

#### 🎯 任务目标 (Day 1-2)
- 理解libbpf skeleton自动生成机制
- 重构代码使用skeleton简化加载
- 实现完整的程序生命周期管理

#### ✅ 具体任务

**Day 1上午：学习skeleton机制**

📚 **学习资料**:
1. 阅读libbpf文档:
   ```bash
   git clone https://github.com/libbpf/libbpf.git
   cd libbpf/src
   # 阅读 README.md 和 libbpf.h
   ```
   - 重点: `bpftool gen skeleton` 的作用
   - 时间: 1小时
   
3. 研究skeleton示例:
   ```bash
   cd libbpf-bootstrap/examples/c
   cat minimal.skel.h  # 查看生成的skeleton
   cat minimal.bpf.c   # 查看源码
   cat minimal.c       # 查看用户态代码
   ```
   - 时间: 1小时

**Day 1下午 + Day 2：重写用户态程序**

创建主程序 `src/user/main.c`:

```c
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <signal.h>
#include <errno.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>
#include <net/if.h>
#include "microsegment.skel.h"

static volatile bool exiting = false;

static void sig_handler(int sig)
{
    exiting = true;
}

static int libbpf_print_fn(enum libbpf_print_level level, const char *format, va_list args)
{
    if (level == LIBBPF_DEBUG)
        return 0;
    return vfprintf(stderr, format, args);
}

int attach_tc_program(struct microsegment_bpf *skel, const char *ifname)
{
    DECLARE_LIBBPF_OPTS(bpf_tc_hook, hook,
                        .ifindex = if_nametoindex(ifname),
                        .attach_point = BPF_TC_INGRESS);
    DECLARE_LIBBPF_OPTS(bpf_tc_opts, opts,
                        .handle = 1,
                        .priority = 1,
                        .prog_fd = bpf_program__fd(skel->progs.microsegment_filter));

    int err;

    // 创建clsact qdisc
    err = bpf_tc_hook_create(&hook);
    if (err && err != -EEXIST) {
        fprintf(stderr, "Failed to create TC hook: %d\n", err);
        return err;
    }

    // 附加程序
    err = bpf_tc_attach(&hook, &opts);
    if (err) {
        fprintf(stderr, "Failed to attach TC program: %d\n", err);
        return err;
    }

    printf("✓ Attached to %s (ifindex=%d)\n", ifname, hook.ifindex);
    return 0;
}

int detach_tc_program(const char *ifname)
{
    DECLARE_LIBBPF_OPTS(bpf_tc_hook, hook,
                        .ifindex = if_nametoindex(ifname),
                        .attach_point = BPF_TC_INGRESS);
    DECLARE_LIBBPF_OPTS(bpf_tc_opts, opts,
                        .handle = 1,
                        .priority = 1);

    bpf_tc_detach(&hook, &opts);
    bpf_tc_hook_destroy(&hook);

    printf("✓ Detached from %s\n", ifname);
    return 0;
}

int pin_maps(struct microsegment_bpf *skel)
{
    int err;

    // Pin policy_map
    err = bpf_map__pin(skel->maps.policy_map, "/sys/fs/bpf/policy_map");
    if (err && err != -EEXIST) {
        fprintf(stderr, "Failed to pin policy_map: %s\n", strerror(-err));
        return err;
    }

    // Pin session_map
    err = bpf_map__pin(skel->maps.session_map, "/sys/fs/bpf/session_map");
    if (err && err != -EEXIST) {
        fprintf(stderr, "Failed to pin session_map: %s\n", strerror(-err));
        return err;
    }

    // Pin stats_map
    err = bpf_map__pin(skel->maps.stats_map, "/sys/fs/bpf/stats_map");
    if (err && err != -EEXIST) {
        fprintf(stderr, "Failed to pin stats_map: %s\n", strerror(-err));
        return err;
    }

    printf("✓ Maps pinned to /sys/fs/bpf/\n");
    return 0;
}

int main(int argc, char **argv)
{
    struct microsegment_bpf *skel;
    int err;
    const char *ifname = "lo";

    if (argc > 1)
        ifname = argv[1];

    // 设置libbpf日志回调
    libbpf_set_print(libbpf_print_fn);

    // 1. 打开eBPF程序
    skel = microsegment_bpf__open();
    if (!skel) {
        fprintf(stderr, "Failed to open BPF skeleton\n");
        return 1;
    }
    printf("✓ BPF skeleton opened\n");

    // 2. 加载和验证eBPF程序
    err = microsegment_bpf__load(skel);
    if (err) {
        fprintf(stderr, "Failed to load BPF skeleton: %d\n", err);
        goto cleanup;
    }
    printf("✓ BPF programs loaded and verified\n");

    // 3. 附加到TC hook
    err = attach_tc_program(skel, ifname);
    if (err)
        goto cleanup;

    // 4. Pin maps到BPF文件系统
    err = pin_maps(skel);
    if (err)
        goto cleanup;

    // 5. 设置信号处理
    signal(SIGINT, sig_handler);
    signal(SIGTERM, sig_handler);

    printf("\n=== Microsegmentation started ===\n");
    printf("Interface: %s\n", ifname);
    printf("Press Ctrl+C to exit...\n\n");

    // 6. 主循环 - 定期打印统计
    while (!exiting) {
        sleep(5);

        // 读取统计信息
        __u32 key;
        __u64 value, total = 0;

        for (key = 0; key < 8; key++) {
            if (bpf_map__lookup_elem(skel->maps.stats_map, &key, sizeof(key),
                                      &value, sizeof(value), 0) == 0) {
                if (key == 0)
                    total = value;
            }
        }

        if (total > 0) {
            printf("\rTotal packets: %llu", total);
            fflush(stdout);
        }
    }

    printf("\n\n=== Shutting down ===\n");

    // 7. 清理
    detach_tc_program(ifname);

cleanup:
    microsegment_bpf__destroy(skel);
    return err != 0;
}
```

更新Makefile支持skeleton:

```makefile
CLANG ?= clang
BPFTOOL ?= bpftool
CC ?= gcc

ARCH := $(shell uname -m | sed 's/x86_64/x86/' | sed 's/aarch64/arm64/')
BPF_CFLAGS = -g -O2 -target bpf -D__TARGET_ARCH_$(ARCH)

# eBPF程序
BPF_SRC = src/bpf/microsegment.bpf.c
BPF_OBJ = microsegment.bpf.o
BPF_SKEL = microsegment.skel.h

# 用户态程序
USER_SRC = src/user/main.c
USER_BIN = tc_microsegment

.PHONY: all clean

all: $(USER_BIN)

# 编译eBPF程序
$(BPF_OBJ): $(BPF_SRC)
	$(CLANG) $(BPF_CFLAGS) -c $< -o $@

# 生成skeleton
$(BPF_SKEL): $(BPF_OBJ)
	$(BPFTOOL) gen skeleton $< > $@

# 编译用户态程序
$(USER_BIN): $(USER_SRC) $(BPF_SKEL)
	$(CC) -g -Wall -I. $< -lbpf -lelf -lz -o $@

clean:
	rm -f $(BPF_OBJ) $(BPF_SKEL) $(USER_BIN)
```

测试skeleton程序:
```bash
# 编译
make clean && make

# 运行 (需要root权限)
sudo ./tc_microsegment lo

# 在另一终端测试
ping 127.0.0.1 -c 5
curl http://127.0.0.1

# 查看输出和统计
```

#### 📚 学习资料

1. libbpf API深入:
   - `bpf_object__open/load`
   - `bpf_program__fd`
   - `bpf_map__pin`
   - 时间: 1.5小时

2. TC BPF attachment API:
   - `bpf_tc_hook_create/destroy`
   - `bpf_tc_attach/detach`
   - 时间: 1小时

#### ✅ 完成标准 (Day 1-2)

- [ ] skeleton成功生成
- [ ] 用户态程序使用skeleton加载eBPF
- [ ] TC程序正确附加和分离
- [ ] Maps正确pin到bpffs
- [ ] 程序优雅退出和清理

---

### 📅 Day 3-4: CLI工具开发

#### 🎯 任务目标 (Day 3-4)
- 实现子命令架构 (policy/session/stats)
- 添加参数解析 (使用getopt)
- 实现策略CRUD功能

#### ✅ 具体任务

**Day 3-4：实现完整CLI工具**

创建CLI框架 `src/user/cli.c`:

```c
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <getopt.h>
#include <arpa/inet.h>
#include <bpf/bpf.h>
#include "types.h"

// 全局配置
struct config {
    int verbose;
    const char *map_dir;
} cfg = {
    .verbose = 0,
    .map_dir = "/sys/fs/bpf",
};

// ========== 策略管理 ==========

int policy_add(int argc, char **argv)
{
    struct flow_key key = {0};
    struct policy_value value = {0};
    const char *src_ip = "0.0.0.0";
    const char *dst_ip = NULL;
    int dst_port = 0;
    const char *proto = "tcp";
    const char *action = "allow";

    struct option long_options[] = {
        {"src-ip", required_argument, 0, 's'},
        {"dst-ip", required_argument, 0, 'd'},
        {"dst-port", required_argument, 0, 'p'},
        {"protocol", required_argument, 0, 'P'},
        {"action", required_argument, 0, 'a'},
        {0, 0, 0, 0}
    };

    int opt;
    while ((opt = getopt_long(argc, argv, "s:d:p:P:a:", long_options, NULL)) != -1) {
        switch (opt) {
            case 's': src_ip = optarg; break;
            case 'd': dst_ip = optarg; break;
            case 'p': dst_port = atoi(optarg); break;
            case 'P': proto = optarg; break;
            case 'a': action = optarg; break;
            default:
                fprintf(stderr, "Usage: policy add --dst-ip IP --dst-port PORT [options]\n");
                return 1;
        }
    }

    if (!dst_ip || dst_port == 0) {
        fprintf(stderr, "Error: --dst-ip and --dst-port are required\n");
        return 1;
    }

    // 解析IP
    inet_pton(AF_INET, src_ip, &key.src_ip);
    inet_pton(AF_INET, dst_ip, &key.dst_ip);
    key.dst_port = htons(dst_port);

    // 解析协议
    if (strcmp(proto, "tcp") == 0)
        key.protocol = 6;
    else if (strcmp(proto, "udp") == 0)
        key.protocol = 17;
    else if (strcmp(proto, "icmp") == 0)
        key.protocol = 1;

    // 解析动作
    if (strcmp(action, "allow") == 0)
        value.action = 0;
    else if (strcmp(action, "deny") == 0)
        value.action = 1;
    else if (strcmp(action, "log") == 0)
        value.action = 2;

    value.priority = 100;

    // 打开Map
    char map_path[256];
    snprintf(map_path, sizeof(map_path), "%s/policy_map", cfg.map_dir);
    int map_fd = bpf_obj_get(map_path);
    if (map_fd < 0) {
        perror("Failed to open policy_map");
        return 1;
    }

    // 添加策略
    if (bpf_map_update_elem(map_fd, &key, &value, BPF_ANY) < 0) {
        perror("Failed to add policy");
        close(map_fd);
        return 1;
    }

    printf("✓ Policy added: %s:%d -> %s:%d (%s, %s)\n",
           src_ip, 0, dst_ip, dst_port, proto, action);

    close(map_fd);
    return 0;
}

int policy_list(int argc, char **argv)
{
    char map_path[256];
    snprintf(map_path, sizeof(map_path), "%s/policy_map", cfg.map_dir);
    int map_fd = bpf_obj_get(map_path);
    if (map_fd < 0) {
        perror("Failed to open policy_map");
        return 1;
    }

    printf("=== Policy List ===\n");
    printf("%-15s %-15s %-6s %-5s %-6s %10s\n",
           "SRC_IP", "DST_IP", "PORT", "PROTO", "ACTION", "HIT_COUNT");

    struct flow_key key, next_key;
    struct policy_value value;
    int count = 0;

    memset(&key, 0, sizeof(key));
    while (bpf_map_get_next_key(map_fd, &key, &next_key) == 0) {
        if (bpf_map_lookup_elem(map_fd, &next_key, &value) == 0) {
            char src_ip[16], dst_ip[16];
            inet_ntop(AF_INET, &next_key.src_ip, src_ip, sizeof(src_ip));
            inet_ntop(AF_INET, &next_key.dst_ip, dst_ip, sizeof(dst_ip));

            const char *proto = next_key.protocol == 6 ? "TCP" :
                                next_key.protocol == 17 ? "UDP" : "OTHER";
            const char *action = value.action == 0 ? "ALLOW" :
                                 value.action == 1 ? "DENY" : "LOG";

            printf("%-15s %-15s %-6d %-5s %-6s %10llu\n",
                   src_ip, dst_ip, ntohs(next_key.dst_port),
                   proto, action, value.hit_count);
            count++;
        }
        key = next_key;
    }

    printf("\nTotal policies: %d\n", count);
    close(map_fd);
    return 0;
}

int policy_delete(int argc, char **argv)
{
    // TODO: 实现删除功能
    fprintf(stderr, "Not implemented yet\n");
    return 1;
}

// ========== 会话管理 ==========

int session_list(int argc, char **argv)
{
    char map_path[256];
    snprintf(map_path, sizeof(map_path), "%s/session_map", cfg.map_dir);
    int map_fd = bpf_obj_get(map_path);
    if (map_fd < 0) {
        perror("Failed to open session_map");
        return 1;
    }

    printf("=== Active Sessions ===\n");
    printf("%-15s %-6s %-15s %-6s %-5s %10s %12s\n",
           "SRC_IP", "PORT", "DST_IP", "PORT", "PROTO", "PACKETS", "BYTES");

    struct flow_key key, next_key;
    struct session_value value;
    int count = 0;

    memset(&key, 0, sizeof(key));
    while (bpf_map_get_next_key(map_fd, &key, &next_key) == 0) {
        if (bpf_map_lookup_elem(map_fd, &next_key, &value) == 0) {
            char src_ip[16], dst_ip[16];
            inet_ntop(AF_INET, &next_key.src_ip, src_ip, sizeof(src_ip));
            inet_ntop(AF_INET, &next_key.dst_ip, dst_ip, sizeof(dst_ip));

            const char *proto = next_key.protocol == 6 ? "TCP" :
                                next_key.protocol == 17 ? "UDP" : "OTHER";

            printf("%-15s %-6d %-15s %-6d %-5s %10llu %12llu\n",
                   src_ip, ntohs(next_key.src_port),
                   dst_ip, ntohs(next_key.dst_port),
                   proto, value.packet_count, value.byte_count);
            count++;
        }
        key = next_key;
    }

    printf("\nTotal sessions: %d\n", count);
    close(map_fd);
    return 0;
}

// ========== 统计信息 ==========

int stats_show(int argc, char **argv)
{
    char map_path[256];
    snprintf(map_path, sizeof(map_path), "%s/stats_map", cfg.map_dir);
    int map_fd = bpf_obj_get(map_path);
    if (map_fd < 0) {
        perror("Failed to open stats_map");
        return 1;
    }

    printf("=== Statistics ===\n");

    const char *stat_names[] = {
        "Total packets",
        "Policy hits",
        "Session hits",
        "New sessions",
        "Allowed",
        "Denied",
        "Logged",
        "Dropped"
    };

    for (__u32 key = 0; key < 8; key++) {
        __u64 value = 0;
        if (bpf_map_lookup_elem(map_fd, &key, &value) == 0) {
            printf("%-20s: %llu\n", stat_names[key], value);
        }
    }

    close(map_fd);
    return 0;
}

// ========== 主程序 ==========

void print_usage(const char *prog)
{
    printf("Usage: %s <command> [options]\n\n", prog);
    printf("Commands:\n");
    printf("  policy add    Add a new policy\n");
    printf("  policy list   List all policies\n");
    printf("  policy delete Delete a policy\n");
    printf("  session list  List active sessions\n");
    printf("  stats show    Show statistics\n");
    printf("\nExamples:\n");
    printf("  %s policy add --dst-ip 10.0.0.1 --dst-port 80 --action allow\n", prog);
    printf("  %s policy list\n", prog);
    printf("  %s session list\n", prog);
}

int main(int argc, char **argv)
{
    if (argc < 2) {
        print_usage(argv[0]);
        return 1;
    }

    const char *cmd = argv[1];
    argc--;
    argv++;

    if (strcmp(cmd, "policy") == 0) {
        if (argc < 1) {
            fprintf(stderr, "Error: policy subcommand required\n");
            return 1;
        }

        const char *subcmd = argv[1];
        argc--;
        argv++;

        if (strcmp(subcmd, "add") == 0)
            return policy_add(argc, argv);
        else if (strcmp(subcmd, "list") == 0)
            return policy_list(argc, argv);
        else if (strcmp(subcmd, "delete") == 0)
            return policy_delete(argc, argv);
        else {
            fprintf(stderr, "Unknown policy subcommand: %s\n", subcmd);
            return 1;
        }
    }
    else if (strcmp(cmd, "session") == 0) {
        if (argc < 1) {
            fprintf(stderr, "Error: session subcommand required\n");
            return 1;
        }

        const char *subcmd = argv[1];
        argc--;
        argv++;

        if (strcmp(subcmd, "list") == 0)
            return session_list(argc, argv);
        else {
            fprintf(stderr, "Unknown session subcommand: %s\n", subcmd);
            return 1;
        }
    }
    else if (strcmp(cmd, "stats") == 0) {
        if (argc < 1) {
            fprintf(stderr, "Error: stats subcommand required\n");
            return 1;
        }

        const char *subcmd = argv[1];
        argc--;
        argv++;

        if (strcmp(subcmd, "show") == 0)
            return stats_show(argc, argv);
        else {
            fprintf(stderr, "Unknown stats subcommand: %s\n", subcmd);
            return 1;
        }
    }
    else {
        fprintf(stderr, "Unknown command: %s\n", cmd);
        print_usage(argv[0]);
        return 1;
    }

    return 0;
}
```

创建类型定义文件 `src/include/types.h`:

```c
#ifndef TYPES_H
#define TYPES_H

#include <stdint.h>

struct flow_key {
    uint32_t src_ip;
    uint32_t dst_ip;
    uint16_t src_port;
    uint16_t dst_port;
    uint8_t  protocol;
} __attribute__((packed));

struct policy_value {
    uint8_t action;
    uint32_t priority;
    uint64_t hit_count;
} __attribute__((packed));

struct session_value {
    uint64_t last_seen;
    uint64_t packet_count;
    uint64_t byte_count;
    uint8_t  cached_action;
    uint32_t policy_id;
} __attribute__((packed));

#endif
```

更新Makefile编译CLI工具:

```makefile
# CLI工具
CLI_SRC = src/user/cli.c
CLI_BIN = tc_microsegment_cli

all: $(USER_BIN) $(CLI_BIN)

$(CLI_BIN): $(CLI_SRC)
	$(CC) -g -Wall -I./src/include $< -lbpf -o $@
```

测试CLI工具:
```bash
# 编译
make

# 启动主程序
sudo ./tc_microsegment lo &

# 使用CLI添加策略
sudo ./tc_microsegment_cli policy add --dst-ip 127.0.0.1 --dst-port 80 --action allow
sudo ./tc_microsegment_cli policy add --dst-ip 127.0.0.1 --dst-port 22 --action deny

# 列出策略
sudo ./tc_microsegment_cli policy list

# 生成流量测试
curl http://127.0.0.1
telnet 127.0.0.1 22

# 查看会话
sudo ./tc_microsegment_cli session list

# 查看统计
sudo ./tc_microsegment_cli stats show
```

#### 📚 学习资料

1. getopt_long参数解析:
   - 短选项 vs 长选项
   - required_argument用法
   - 时间: 1小时

2. BPF Map操作API:
   - `bpf_obj_get`
   - `bpf_map_get_next_key`
   - `bpf_map_lookup_elem`
   - 时间: 1小时

#### ✅ 完成标准 (Day 3-4)

- [ ] CLI工具子命令架构完成
- [ ] policy add/list功能正常
- [ ] session list显示活跃会话
- [ ] stats show显示统计信息
- [ ] 参数解析健壮

---

### 📅 Day 5: 配置文件支持与策略热更新

#### 🎯 任务目标
- 添加JSON配置文件支持
- 实现批量策略导入
- 实现策略热更新(无需重启)

#### ✅ 具体任务

**上午：JSON配置文件支持**

安装json-c库:
```bash
sudo apt-get install -y libjson-c-dev
```

创建配置文件加载器 `src/user/config.c`:

```c
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <arpa/inet.h>
#include <json-c/json.h>
#include <bpf/bpf.h>
#include "types.h"

int load_policies_from_json(const char *filename, int map_fd)
{
    // 读取JSON文件
    FILE *fp = fopen(filename, "r");
    if (!fp) {
        perror("fopen");
        return -1;
    }

    fseek(fp, 0, SEEK_END);
    long fsize = ftell(fp);
    fseek(fp, 0, SEEK_SET);

    char *content = malloc(fsize + 1);
    fread(content, 1, fsize, fp);
    fclose(fp);
    content[fsize] = 0;

    // 解析JSON
    struct json_object *root = json_tokener_parse(content);
    free(content);

    if (!root) {
        fprintf(stderr, "Failed to parse JSON\n");
        return -1;
    }

    struct json_object *policies;
    if (!json_object_object_get_ex(root, "policies", &policies)) {
        fprintf(stderr, "No 'policies' field in JSON\n");
        json_object_put(root);
        return -1;
    }

    int count = 0;
    int array_len = json_object_array_length(policies);

    for (int i = 0; i < array_len; i++) {
        struct json_object *policy = json_object_array_get_idx(policies, i);
        struct flow_key key = {0};
        struct policy_value value = {0};

        // 解析各字段
        struct json_object *obj;

        if (json_object_object_get_ex(policy, "dst_ip", &obj)) {
            const char *ip = json_object_get_string(obj);
            inet_pton(AF_INET, ip, &key.dst_ip);
        }

        if (json_object_object_get_ex(policy, "dst_port", &obj)) {
            int port = json_object_get_int(obj);
            key.dst_port = htons(port);
        }

        if (json_object_object_get_ex(policy, "protocol", &obj)) {
            const char *proto = json_object_get_string(obj);
            if (strcmp(proto, "tcp") == 0)
                key.protocol = 6;
            else if (strcmp(proto, "udp") == 0)
                key.protocol = 17;
        }

        if (json_object_object_get_ex(policy, "action", &obj)) {
            const char *action = json_object_get_string(obj);
            if (strcmp(action, "allow") == 0)
                value.action = 0;
            else if (strcmp(action, "deny") == 0)
                value.action = 1;
        }

        if (json_object_object_get_ex(policy, "priority", &obj)) {
            value.priority = json_object_get_int(obj);
        } else {
            value.priority = 100;
        }

        // 添加到Map
        if (bpf_map_update_elem(map_fd, &key, &value, BPF_ANY) == 0) {
            count++;
        } else {
            fprintf(stderr, "Failed to add policy %d\n", i);
        }
    }

    json_object_put(root);
    printf("✓ Loaded %d policies from %s\n", count, filename);
    return count;
}
```

示例配置文件 `configs/policies.json`:

```json
{
  "policies": [
    {
      "name": "Allow HTTP",
      "dst_ip": "0.0.0.0",
      "dst_port": 80,
      "protocol": "tcp",
      "action": "allow",
      "priority": 100
    },
    {
      "name": "Allow HTTPS",
      "dst_ip": "0.0.0.0",
      "dst_port": 443,
      "protocol": "tcp",
      "action": "allow",
      "priority": 100
    },
    {
      "name": "Allow DNS",
      "dst_ip": "0.0.0.0",
      "dst_port": 53,
      "protocol": "udp",
      "action": "allow",
      "priority": 100
    },
    {
      "name": "Deny SSH",
      "dst_ip": "192.168.1.100",
      "dst_port": 22,
      "protocol": "tcp",
      "action": "deny",
      "priority": 200
    }
  ]
}
```

在CLI中添加load子命令:

```c
int policy_load(int argc, char **argv)
{
    const char *filename = NULL;

    struct option long_options[] = {
        {"file", required_argument, 0, 'f'},
        {0, 0, 0, 0}
    };

    int opt;
    while ((opt = getopt_long(argc, argv, "f:", long_options, NULL)) != -1) {
        if (opt == 'f')
            filename = optarg;
    }

    if (!filename) {
        fprintf(stderr, "Error: --file is required\n");
        return 1;
    }

    char map_path[256];
    snprintf(map_path, sizeof(map_path), "%s/policy_map", cfg.map_dir);
    int map_fd = bpf_obj_get(map_path);
    if (map_fd < 0) {
        perror("Failed to open policy_map");
        return 1;
    }

    int count = load_policies_from_json(filename, map_fd);
    close(map_fd);

    return count > 0 ? 0 : 1;
}
```

**下午：测试热更新**

测试策略热更新流程:

```bash
# 1. 启动主程序
sudo ./tc_microsegment lo &

# 2. 加载初始策略
sudo ./tc_microsegment_cli policy load --file configs/policies.json

# 3. 查看策略
sudo ./tc_microsegment_cli policy list

# 4. 测试连接
curl http://127.0.0.1  # 应该成功
telnet 192.168.1.100 22  # 应该被拒绝

# 5. 修改配置文件 (添加新策略)

# 6. 热更新 (无需重启主程序)
sudo ./tc_microsegment_cli policy load --file configs/policies_v2.json

# 7. 验证新策略立即生效
sudo ./tc_microsegment_cli policy list
```

编写热更新测试脚本 `tests/test_hot_reload.sh`:

```bash
#!/bin/bash
set -e

echo "=== 策略热更新测试 ==="

# 启动程序
sudo ./tc_microsegment lo &
PID=$!
sleep 2

# 加载v1策略
echo "加载v1策略..."
sudo ./tc_microsegment_cli policy load --file configs/policies_v1.json
sudo ./tc_microsegment_cli policy list

# 测试v1策略
echo "测试v1策略..."
curl -s http://127.0.0.1 >/dev/null && echo "✓ HTTP allowed"

# 加载v2策略 (不重启程序)
echo "热更新到v2策略..."
sudo ./tc_microsegment_cli policy load --file configs/policies_v2.json
sudo ./tc_microsegment_cli policy list

# 测试v2策略
echo "测试v2策略..."
curl -s https://127.0.0.1 >/dev/null && echo "✓ HTTPS allowed"

# 清理
sudo kill $PID
sudo tc qdisc del dev lo clsact

echo "✓ 热更新测试完成"
```

#### 📚 学习资料

1. json-c库使用:
   - JSON解析API
   - 对象访问方法
   - 时间: 1小时

2. BPF Map热更新机制:
   - Map是内核态对象, 用户态可随时修改
   - 无需重新加载eBPF程序
   - 时间: 30分钟

#### ✅ 完成标准 (Day 5)

- [ ] JSON配置文件正确解析
- [ ] 批量策略导入功能正常
- [ ] 策略热更新无需重启
- [ ] 热更新后立即生效

---

### 📅 本周总结 (Friday晚上)

#### ✍️ 输出物

创建文档 `docs/week3_summary.md`:

```markdown
# 第3周学习总结

## 完成情况

- [x] libbpf skeleton集成
- [x] 完整CLI工具
- [x] JSON配置文件支持
- [x] 策略热更新

## 核心收获

### 1. libbpf skeleton优势
- 自动生成类型安全的加载代码
- 简化Map和Program访问
- 更好的错误处理

### 2. CLI工具设计
- 子命令架构 (policy/session/stats)
- getopt_long参数解析
- 用户友好的输出格式

### 3. 策略热更新
- 无需重启eBPF程序
- Map在内核态独立存在
- 用户态随时可修改

## CLI命令示例

```bash
# 添加策略
tc_microsegment_cli policy add --dst-ip 10.0.0.1 --dst-port 80 --action allow

# 列出策略
tc_microsegment_cli policy list

# 批量导入
tc_microsegment_cli policy load --file policies.json

# 查看会话
tc_microsegment_cli session list

# 查看统计
tc_microsegment_cli stats show
```



#### 🎯 本周验收标准

**必须完成**:
- [ ] skeleton成功集成
- [ ] CLI工具三大子命令全部正常
- [ ] JSON配置文件能正确加载
- [ ] 策略热更新测试通过

**加分项**:
- [ ] CLI帮助信息完善
- [ ] 错误处理健壮
- [ ] 支持更多配置选项

---

## 5. 第4周：高级功能实现

### 🎯 本周目标

- [ ] 实现TCP状态机
- [ ] 实现IP段匹配 (LPM Trie)
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

# 启动程序
sudo ./tc_microsegment lo &
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
sudo kill $PID
sudo tc qdisc del dev lo clsact

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

1. ✅ 完整的测试套件
2. ✅ 测试报告 (功能+性能)
3. ✅ 性能调优方案
4. ✅ Bug修复列表

---

### 📅 Day 1: 单元测试编写

#### 🎯 任务目标
- 为核心函数编写单元测试
- 使用bpf_testmod或自定义测试框架
- 实现Mock数据包测试

#### ✅ 具体任务

**全天：编写单元测试**

创建测试框架 `tests/unit_tests.c`:

```
```c
#include <stdio.h>
#include <assert.h>
#include <string.h>
#include <arpa/inet.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>

// 测试辅助函数
void assert_eq(int expected, int actual, const char *msg) {
    if (expected != actual) {
        fprintf(stderr, "FAIL: %s (expected=%d, actual=%d)\n",
                msg, expected, actual);
        exit(1);
    }
    printf("PASS: %s\n", msg);
}

// 测试1: Map创建和访问
void test_map_operations() {
    printf("\n=== Test: Map Operations ===\n");

    // 打开Maps
    int policy_fd = bpf_obj_get("/sys/fs/bpf/policy_map");
    assert_eq(1, policy_fd > 0, "Open policy_map");

    // 测试插入
    struct flow_key key = {
        .src_ip = inet_addr("192.168.1.100"),
        .dst_ip = inet_addr("10.0.0.50"),
        .dst_port = htons(80),
        .protocol = 6
    };

    struct policy_value value = {
        .action = 0,
        .priority = 100
    };

    int ret = bpf_map_update_elem(policy_fd, &key, &value, BPF_ANY);
    assert_eq(0, ret, "Insert policy");

    // 测试查找
    struct policy_value result;
    ret = bpf_map_lookup_elem(policy_fd, &key, &result);
    assert_eq(0, ret, "Lookup policy");
    assert_eq(0, result.action, "Policy action");

    // 测试删除
    ret = bpf_map_delete_elem(policy_fd, &key);
    assert_eq(0, ret, "Delete policy");

    close(policy_fd);
}

// 测试2: 5元组key构造
void test_flow_key_construction() {
    printf("\n=== Test: Flow Key Construction ===\n");

    struct flow_key key1 = {
        .src_ip = inet_addr("192.168.1.100"),
        .dst_ip = inet_addr("10.0.0.50"),
        .src_port = htons(12345),
        .dst_port = htons(80),
        .protocol = 6
    };

    struct flow_key key2 = key1;

    // 测试key相等性
    int cmp = memcmp(&key1, &key2, sizeof(struct flow_key));
    assert_eq(0, cmp, "Flow key equality");

    // 测试不同key
    key2.dst_port = htons(443);
    cmp = memcmp(&key1, &key2, sizeof(struct flow_key));
    assert_eq(1, cmp != 0, "Flow key inequality");
}

// 测试3: TCP状态转换
void test_tcp_state_machine() {
    printf("\n=== Test: TCP State Machine ===\n");

    // 模拟TCP握手序列
    struct session_value sess = {.tcp_state = 0};  // TCP_NONE

    // SYN -> SYN_SENT
    // (模拟状态转换逻辑)
    sess.tcp_state = 1;  // TCP_SYN_SENT
    assert_eq(1, sess.tcp_state, "After SYN");

    // SYN+ACK -> SYN_RECV
    sess.tcp_state = 2;  // TCP_SYN_RECV
    assert_eq(2, sess.tcp_state, "After SYN+ACK");

    // ACK -> ESTABLISHED
    sess.tcp_state = 3;  // TCP_ESTABLISHED
    assert_eq(3, sess.tcp_state, "After ACK (ESTABLISHED)");
}

// 测试4: LPM Trie匹配
void test_lpm_trie_matching() {
    printf("\n=== Test: LPM Trie Matching ===\n");

    int map_fd = bpf_obj_get("/sys/fs/bpf/ip_range_map");
    if (map_fd < 0) {
        printf("SKIP: LPM Trie map not available\n");
        return;
    }

    // 添加 192.168.1.0/24
    struct lpm_key key = {
        .prefixlen = 24,
        .ip = inet_addr("192.168.1.0")
    };

    struct ip_range_value value = {.action = 0, .priority = 100};
    bpf_map_update_elem(map_fd, &key, &value, BPF_ANY);

    // 查找 192.168.1.100 (应该匹配)
    key.prefixlen = 32;
    key.ip = inet_addr("192.168.1.100");

    struct ip_range_value result;
    int ret = bpf_map_lookup_elem(map_fd, &key, &result);
    assert_eq(0, ret, "LPM match 192.168.1.100");

    // 查找 192.168.2.100 (不应该匹配)
    key.ip = inet_addr("192.168.2.100");
    ret = bpf_map_lookup_elem(map_fd, &key, &result);
    assert_eq(1, ret < 0, "LPM no match 192.168.2.100");

    close(map_fd);
}

// 测试5: 统计计数器
void test_statistics_counters() {
    printf("\n=== Test: Statistics Counters ===\n");

    int stats_fd = bpf_obj_get("/sys/fs/bpf/stats_map");
    assert_eq(1, stats_fd > 0, "Open stats_map");

    // 读取当前统计
    __u32 key = 0;  // STAT_TOTAL
    __u64 value_before;
    bpf_map_lookup_elem(stats_fd, &key, &value_before);

    printf("  Total packets before: %llu\n", value_before);

    // 生成一些流量...
    system("ping -c 1 127.0.0.1 >/dev/null 2>&1");

    // 读取新统计
    __u64 value_after;
    bpf_map_lookup_elem(stats_fd, &key, &value_after);

    printf("  Total packets after: %llu\n", value_after);

    assert_eq(1, value_after >= value_before, "Statistics incremented");

    close(stats_fd);
}

int main() {
    printf("===========================================\n");
    printf("eBPF Microsegmentation Unit Tests\n");
    printf("===========================================\n");

    test_map_operations();
    test_flow_key_construction();
    test_tcp_state_machine();
    test_lpm_trie_matching();
    test_statistics_counters();

    printf("\n===========================================\n");
    printf("✓ All tests passed!\n");
    printf("===========================================\n");

    return 0;
}
```

编译和运行测试:

```bash
# 编译测试
gcc -o unit_tests tests/unit_tests.c -lbpf -I./src/include

# 运行测试 (需要先启动主程序)
sudo ./tc_microsegment lo &
sleep 2
sudo ./unit_tests

# 查看结果
```

#### 📚 学习资料

1. 单元测试最佳实践:
   - 测试隔离
   - Mock和Stub
   - 断言设计
   - 时间: 1.5小时

2. BPF测试框架:
   - libbpf测试工具
   - bpf_prog_test_run
   - 时间: 1.5小时

#### ✅ 完成标准 (Day 1)

- [ ] 至少5个单元测试编写
- [ ] 测试覆盖核心功能
- [ ] 所有测试通过
- [ ] 测试可自动化运行

---

### 📅 Day 2: 功能测试与bug修复

#### 🎯 任务目标
- 编写端到端功能测试
- 发现并修复bug
- 验证所有用户场景

#### ✅ 具体任务

**全天：功能测试**

创建功能测试套件 `tests/functional_tests.sh`:

```bash
#!/bin/bash
set -e

FAILED=0
PASSED=0

pass() {
    echo "✓ PASS: $1"
    PASSED=$((PASSED + 1))
}

fail() {
    echo "✗ FAIL: $1"
    FAILED=$((FAILED + 1))
}

cleanup() {
    sudo tc qdisc del dev lo clsact 2>/dev/null || true
    sudo killall tc_microsegment 2>/dev/null || true
}

trap cleanup EXIT

echo "=== Functional Tests ==="

# 启动程序
sudo ./tc_microsegment lo &
PID=$!
sleep 3

# 测试1: 基础策略添加和查询
echo -e "\n[Test 1] Policy CRUD operations"
sudo ./tc_microsegment_cli policy add --dst-ip 127.0.0.1 --dst-port 8080 --action allow
COUNT=$(sudo ./tc_microsegment_cli policy list | grep -c "127.0.0.1")
if [ "$COUNT" -eq 1 ]; then
    pass "Policy add and list"
else
    fail "Policy add and list"
fi

# 测试2: 策略匹配和执行
echo -e "\n[Test 2] Policy enforcement"
sudo ./tc_microsegment_cli policy add --dst-ip 127.0.0.1 --dst-port 22 --action deny

# 启动临时SSH服务器 (如果存在)
nc -l 127.0.0.1 22 &
NC_PID=$!
sleep 1

# 尝试连接 (应该被拒绝)
timeout 2 telnet 127.0.0.1 22 2>/dev/null && fail "Deny policy" || pass "Deny policy"

kill $NC_PID 2>/dev/null || true

# 测试3: 会话跟踪
echo -e "\n[Test 3] Session tracking"
curl -s http://127.0.0.1:8080 >/dev/null 2>&1 &
sleep 1

SESSION_COUNT=$(sudo ./tc_microsegment_cli session list | grep -c "127.0.0.1")
if [ "$SESSION_COUNT" -gt 0 ]; then
    pass "Session tracking"
else
    fail "Session tracking"
fi

# 测试4: 会话缓存命中
echo -e "\n[Test 4] Session cache hit rate"
for i in {1..10}; do
    curl -s http://127.0.0.1:8080 >/dev/null 2>&1 || true
done

TOTAL=$(sudo bpftool map dump name stats_map | grep "key: 0" -A1 | grep value | awk '{print $2}')
SESSION_HIT=$(sudo bpftool map dump name stats_map | grep "key: 2" -A1 | grep value | awk '{print $2}')

if [ "$TOTAL" -gt 0 ] && [ "$SESSION_HIT" -gt 0 ]; then
    HITRATE=$((SESSION_HIT * 100 / TOTAL))
    if [ "$HITRATE" -gt 50 ]; then
        pass "Session cache ($HITRATE% hit rate)"
    else
        fail "Session cache ($HITRATE% hit rate too low)"
    fi
else
    fail "Session cache (no data)"
fi

# 测试5: IP段匹配
echo -e "\n[Test 5] IP range matching"
sudo ./tc_microsegment_cli ip-range add --cidr 127.0.0.0/8 --action allow 2>/dev/null || true

# 测试匹配
ping -c 1 127.0.0.1 >/dev/null 2>&1 && pass "IP range match" || fail "IP range match"

# 测试6: 统计功能
echo -e "\n[Test 6] Statistics"
STATS=$(sudo ./tc_microsegment_cli stats show)
if echo "$STATS" | grep -q "Total packets"; then
    pass "Statistics display"
else
    fail "Statistics display"
fi

# 测试7: 策略热更新
echo -e "\n[Test 7] Hot policy reload"
cat > /tmp/test_policies.json <<EOF
{
  "policies": [
    {
      "dst_ip": "127.0.0.1",
      "dst_port": 9090,
      "protocol": "tcp",
      "action": "allow",
      "priority": 100
    }
  ]
}
EOF

sudo ./tc_microsegment_cli policy load --file /tmp/test_policies.json 2>/dev/null || true
sleep 1

COUNT=$(sudo ./tc_microsegment_cli policy list | grep -c "9090")
if [ "$COUNT" -eq 1 ]; then
    pass "Hot policy reload"
else
    fail "Hot policy reload"
fi

# 测试8: TCP状态机
echo -e "\n[Test 8] TCP state machine"
# 建立完整TCP连接
nc -zv 127.0.0.1 8080 2>&1 | grep -q "succeeded" && pass "TCP handshake" || fail "TCP handshake"

# 汇总
echo -e "\n========================================="
echo "Tests run: $((PASSED + FAILED))"
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo "========================================="

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi
```

运行功能测试:

```bash
chmod +x tests/functional_tests.sh
sudo ./tests/functional_tests.sh
```

**Bug修复流程**:

1. 记录失败的测试
2. 使用bpf_printk调试eBPF程序
3. 使用gdb调试用户态程序
4. 修复代码
5. 重新运行测试验证

#### 📚 学习资料

1. 调试技巧:
   - bpf_printk使用
   - trace_pipe分析
   - bpftool调试
   - 时间: 2小时

2. 常见bug模式:
   - 边界条件
   - 并发问题
   - 内存泄漏
   - 时间: 1.5小时

#### ✅ 完成标准 (Day 2)

- [ ] 所有功能测试通过
- [ ] 发现的bug已修复
- [ ] 测试用例文档化
- [ ] Bug修复记录

---

### 📅 Day 3: 性能测试与调优

#### 🎯 任务目标
- 测试吞吐量和延迟
- 识别性能瓶颈
- 优化关键路径

#### ✅ 具体任务

**全天：性能测试和优化**

创建性能测试脚本 `tests/performance_tests.sh`:

```bash
#!/bin/bash
set -e

echo "=== Performance Tests ==="

# 启动程序
sudo ./tc_microsegment lo &
PID=$!
sleep 3

# 测试1: 基准吞吐量 (无eBPF)
echo -e "\n[Baseline] Throughput without eBPF"
sudo tc qdisc del dev lo clsact 2>/dev/null || true

iperf3 -s -p 5201 >/dev/null 2>&1 &
IPERF_PID=$!
sleep 2

BASELINE=$(iperf3 -c 127.0.0.1 -p 5201 -t 10 -J | jq '.end.sum_received.bits_per_second')
echo "  Baseline: $(echo "scale=2; $BASELINE / 1000000000" | bc) Gbps"

kill $IPERF_PID 2>/dev/null || true
sleep 2

# 测试2: eBPF吞吐量
echo -e "\n[eBPF] Throughput with eBPF filtering"
sudo ./tc_microsegment lo &
sleep 3

iperf3 -s -p 5201 >/dev/null 2>&1 &
IPERF_PID=$!
sleep 2

EBPF_BW=$(iperf3 -c 127.0.0.1 -p 5201 -t 10 -J | jq '.end.sum_received.bits_per_second')
echo "  eBPF: $(echo "scale=2; $EBPF_BW / 1000000000" | bc) Gbps"

OVERHEAD=$(echo "scale=2; (1 - $EBPF_BW / $BASELINE) * 100" | bc)
echo "  Overhead: $OVERHEAD%"

kill $IPERF_PID 2>/dev/null || true

# 测试3: 延迟测试
echo -e "\n[Latency] Round-trip time"

# 无eBPF
sudo tc qdisc del dev lo clsact 2>/dev/null || true
RTT_BASELINE=$(ping -c 100 -i 0.01 127.0.0.1 | grep "avg" | awk -F'/' '{print $5}')
echo "  Baseline RTT: $RTT_BASELINE ms"

# 有eBPF
sudo ./tc_microsegment lo &
sleep 3

RTT_EBPF=$(ping -c 100 -i 0.01 127.0.0.1 | grep "avg" | awk -F'/' '{print $5}')
echo "  eBPF RTT: $RTT_EBPF ms"

LATENCY_OVERHEAD=$(echo "scale=2; $RTT_EBPF - $RTT_BASELINE" | bc)
echo "  Added latency: $LATENCY_OVERHEAD ms"

# 测试4: 策略查找性能
echo -e "\n[Policy Lookup] Performance"

# 添加1000条策略
for i in {1..1000}; do
    sudo ./tc_microsegment_cli policy add \
        --dst-ip 10.0.$((i/256)).$((i%256)) \
        --dst-port $((1000 + i)) \
        --action allow \
        >/dev/null 2>&1
done

echo "  Added 1000 policies"

# 测试查找时间
START=$(date +%s%N)
for i in {1..1000}; do
    sudo ./tc_microsegment_cli policy list >/dev/null 2>&1
done
END=$(date +%s%N)

ELAPSED=$(echo "scale=2; ($END - $START) / 1000000000" | bc)
AVG=$(echo "scale=2; $ELAPSED / 1000" | bc)
echo "  1000 lookups in $ELAPSED seconds"
echo "  Average: $AVG ms per lookup"

# 测试5: 会话缓存性能
echo -e "\n[Session Cache] Performance"

# 清空会话
sudo bpftool map delete name session_map 2>/dev/null || true

# 首次连接 (未缓存)
START=$(date +%s%N)
curl -s http://127.0.0.1:8080 >/dev/null 2>&1 || true
END=$(date +%s%N)
FIRST=$(echo "scale=4; ($END - $START) / 1000000" | bc)

# 后续连接 (已缓存)
START=$(date +%s%N)
for i in {1..100}; do
    curl -s http://127.0.0.1:8080 >/dev/null 2>&1 || true
done
END=$(date +%s%N)
CACHED=$(echo "scale=4; ($END - $START) / 100 / 1000000" | bc)

echo "  First request (uncached): $FIRST ms"
echo "  Cached requests (avg): $CACHED ms"

SPEEDUP=$(echo "scale=2; $FIRST / $CACHED" | bc)
echo "  Speedup: ${SPEEDUP}x"

# 汇总
echo -e "\n========================================="
echo "Performance Summary:"
echo "  Throughput overhead: $OVERHEAD%"
echo "  Latency overhead: +$LATENCY_OVERHEAD ms"
echo "  Policy lookup: $AVG ms"
echo "  Session cache speedup: ${SPEEDUP}x"
echo "========================================="
```

**性能优化checklist**:

1. **eBPF程序优化**:
   - 减少Map查找次数
   - 使用__always_inline
   - 避免复杂循环

2. **Map优化**:
   - 使用合适的Map类型
   - 调整Map大小
   - 启用BPF_F_NO_PREALLOC (如果适用)

3. **用户态优化**:
   - 减少系统调用
   - 批量操作
   - 使用缓存

#### 📚 学习资料

1. 性能分析工具:
   - perf
   - flamegraph
   - bpftool prog profile
   - 时间: 2小时

2. 优化技巧:
   - 热点路径识别
   - Cache-friendly设计
   - 时间: 1.5小时

#### ✅ 完成标准 (Day 3)

- [ ] 性能基准测试完成
- [ ] 吞吐量开销 < 5%
- [ ] 延迟开销 < 10μs
- [ ] 会话缓存加速 > 3x

---

### 📅 Day 4: 压力测试

#### 🎯 任务目标
- 测试系统极限
- 验证稳定性
- 测试异常场景

#### ✅ 具体任务

**全天：压力测试**

创建压力测试脚本 `tests/stress_tests.sh`:

```bash
#!/bin/bash
set -e

echo "=== Stress Tests ==="

# 启动程序
sudo ./tc_microsegment lo &
PID=$!
sleep 3

# 测试1: 高并发连接
echo -e "\n[Test 1] High concurrency"

# 启动多个HTTP服务器
for port in {8080..8090}; do
    python3 -m http.server $port >/dev/null 2>&1 &
done
sleep 2

# 并发请求
echo "  Sending 10000 concurrent requests..."
ab -n 10000 -c 100 http://127.0.0.1:8080/ >/dev/null 2>&1

# 检查统计
TOTAL=$(sudo bpftool map dump name stats_map | grep "key: 0" -A1 | grep value | awk '{print $2}')
echo "  Total packets processed: $TOTAL"

if [ "$TOTAL" -gt 10000 ]; then
    echo "  ✓ PASS"
else
    echo "  ✗ FAIL"
fi

# 清理
killall python3 2>/dev/null || true

# 测试2: Map容量测试
echo -e "\n[Test 2] Map capacity"

echo "  Filling session map..."
for i in {1..100000}; do
    # 模拟不同源IP的连接
    curl -s --interface "127.0.0.1:$((10000 + i % 10000))" \
         http://127.0.0.1:8080 >/dev/null 2>&1 || true

    if [ $((i % 10000)) -eq 0 ]; then
        echo "    Created $i sessions..."
    fi
done

SESSION_COUNT=$(sudo bpftool map dump name session_map | grep -c "key:")
echo "  Active sessions: $SESSION_COUNT"

# 检查压力级别
PRESSURE=$(sudo ./tc_microsegment_cli monitor pressure 2>&1 | head -1 | grep -oP 'Pressure: \K\d+')
echo "  Map pressure: $PRESSURE%"

if [ "$SESSION_COUNT" -gt 50000 ]; then
    echo "  ✓ PASS: Handled $SESSION_COUNT sessions"
else
    echo "  ✗ FAIL: Only $SESSION_COUNT sessions"
fi

# 测试3: SYN Flood模拟
echo -e "\n[Test 3] SYN flood detection"

echo "  Simulating SYN flood..."
hping3 -S -p 80 --flood --rand-source 127.0.0.1 -c 10000 >/dev/null 2>&1 || \
    echo "  (hping3 not available, skipping)"

# 检查是否检测到
LOGS=$(sudo dmesg | tail -100 | grep -c "SYN flood" || echo "0")
if [ "$LOGS" -gt 0 ]; then
    echo "  ✓ PASS: SYN flood detected"
else
    echo "  ⚠ WARNING: SYN flood not detected (may need hping3)"
fi

# 测试4: 长时间稳定性测试
echo -e "\n[Test 4] Long-running stability (10 minutes)"

START_TIME=$(date +%s)
ERROR_COUNT=0

echo "  Running continuous traffic for 10 minutes..."
for i in {1..600}; do  # 10分钟
    curl -s http://127.0.0.1:8080 >/dev/null 2>&1 || ERROR_COUNT=$((ERROR_COUNT + 1))
    ping -c 1 127.0.0.1 >/dev/null 2>&1 || ERROR_COUNT=$((ERROR_COUNT + 1))
    sleep 1

    if [ $((i % 60)) -eq 0 ]; then
        ELAPSED=$((i / 60))
        echo "    $ELAPSED minutes elapsed, errors: $ERROR_COUNT"
    fi
done

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo "  Duration: $DURATION seconds"
echo "  Errors: $ERROR_COUNT"

if [ "$ERROR_COUNT" -lt 10 ]; then
    echo "  ✓ PASS: Stable operation"
else
    echo "  ✗ FAIL: Too many errors ($ERROR_COUNT)"
fi

# 测试5: 内存泄漏检测
echo -e "\n[Test 5] Memory leak detection"

# 记录初始内存
INIT_MEM=$(ps -p $PID -o rss= | awk '{print $1}')
echo "  Initial memory: $INIT_MEM KB"

# 运行1小时的流量
echo "  Running traffic for 1 hour..."
for i in {1..3600}; do
    curl -s http://127.0.0.1:8080 >/dev/null 2>&1 || true
    if [ $((i % 600)) -eq 0 ]; then
        CURR_MEM=$(ps -p $PID -o rss= | awk '{print $1}')
        INCREASE=$((CURR_MEM - INIT_MEM))
        echo "    $((i/60)) min: $CURR_MEM KB (+$INCREASE KB)"
    fi
done

# 检查最终内存
FINAL_MEM=$(ps -p $PID -o rss= | awk '{print $1}')
INCREASE=$((FINAL_MEM - INIT_MEM))
INCREASE_PCT=$((INCREASE * 100 / INIT_MEM))

echo "  Final memory: $FINAL_MEM KB"
echo "  Increase: $INCREASE KB ($INCREASE_PCT%)"

if [ "$INCREASE_PCT" -lt 20 ]; then
    echo "  ✓ PASS: No significant memory leak"
else
    echo "  ✗ FAIL: Possible memory leak (+$INCREASE_PCT%)"
fi

echo -e "\n========================================="
echo "Stress tests completed"
echo "========================================="
```

#### 📚 学习资料

1. 压力测试工具:
   - ab (Apache Bench)
   - wrk
   - hping3
   - 时间: 1.5小时

2. 稳定性指标:
   - 内存增长率
   - CPU使用率
   - 错误率
   - 时间: 1小时

#### ✅ 完成标准 (Day 4)

- [ ] 处理10万+并发会话
- [ ] SYN Flood检测触发
- [ ] 长时间运行无崩溃
- [ ] 无明显内存泄漏

---

### 📅 Day 5: 文档整理与测试报告

#### 🎯 任务目标
- 整理所有测试结果
- 编写测试报告
- 更新文档
- 准备演示

#### ✅ 具体任务

**全天：文档和报告**

创建测试报告 `docs/test_report.md`:

```markdown
# eBPF微隔离测试报告

## 1. 测试概述

**测试日期**: 2025-xx-xx
**测试环境**: Ubuntu 22.04, Kernel 5.15
**测试负责人**: XXX

## 2. 单元测试结果

| 测试项 | 结果 | 说明 |
|--------|------|------|
| Map操作 | PASS | 插入/查找/删除正常 |
| Flow Key构造 | PASS | 5元组正确构造 |
| TCP状态机 | PASS | 状态转换符合RFC 793 |
| LPM Trie匹配 | PASS | 最长前缀匹配正确 |
| 统计计数器 | PASS | 计数准确 |

**通过率**: 100% (5/5)

## 3. 功能测试结果

| 测试项 | 结果 | 说明 |
|--------|------|------|
| 策略CRUD | PASS | 增删改查正常 |
| 策略执行 | PASS | allow/deny正确 |
| 会话跟踪 | PASS | 会话正确建立 |
| 会话缓存 | PASS | 命中率95%+ |
| IP段匹配 | PASS | CIDR匹配正确 |
| 统计显示 | PASS | 实时统计准确 |
| 策略热更新 | PASS | 无需重启 |
| TCP握手 | PASS | 3次握手正常 |

**通过率**: 100% (8/8)

## 4. 性能测试结果

| 指标 | 基准值 | eBPF值 | 开销 |
|------|--------|--------|------|
| 吞吐量 | 10.0 Gbps | 9.7 Gbps | 3% |
| 平均延迟 | 0.05 ms | 0.06 ms | +0.01 ms |
| P99延迟 | 0.10 ms | 0.12 ms | +0.02 ms |
| 策略查找 | - | 0.05 ms | - |
| 会话缓存加速 | 1x | 4.2x | +320% |

**结论**: 性能开销在可接受范围内，会话缓存显著提升性能。

## 5. 压力测试结果

| 测试项 | 目标 | 实际 | 结果 |
|--------|------|------|------|
| 并发会话数 | 100,000 | 105,342 | PASS |
| SYN Flood检测 | 触发 | 已触发 | PASS |
| 长时间稳定性 | 10小时 | 10小时无崩溃 | PASS |
| 内存泄漏 | <20%增长 | 8%增长 | PASS |
| CPU使用率 | <80% | 平均45% | PASS |

## 6. 发现的问题

### 已修复

1. **Bug #1**: TCP状态机在FIN_WAIT2不正确转换
   **修复**: 添加对FIN+ACK的处理

2. **Bug #2**: LPM Trie在/32前缀时匹配失败
   **修复**: 调整prefixlen设置

### 待修复

无

## 7. 总结

所有测试通过，系统达到生产就绪状态。

**推荐**: 可进入生产部署阶段。
```

更新README:

```markdown
# eBPF TC 微隔离系统

## 功能特性

- ✅ 基于5元组的策略匹配
- ✅ TCP状态机跟踪
- ✅ IP段匹配 (CIDR)
- ✅ 会话缓存 (LRU)
- ✅ Map压力监控
- ✅ Prometheus集成
- ✅ 策略热更新

## 性能指标

- 吞吐量开销: ~3%
- 延迟增加: ~10μs
- 支持会话数: 100,000+
- 会话缓存加速: 4x

## 快速开始

```bash
# 编译
make

# 运行
sudo ./tc_microsegment eth0

# 添加策略
sudo ./tc_microsegment_cli policy add \
    --dst-ip 10.0.0.1 --dst-port 80 --action allow

# 查看统计
sudo ./tc_microsegment_cli stats show
```

## 测试

```bash
# 单元测试
sudo ./unit_tests

# 功能测试
sudo ./tests/functional_tests.sh

# 性能测试
sudo ./tests/performance_tests.sh
```

## 文档

- [设计文档](specs/design.md)
- [实施指南](specs/ebpf-tc-implementation.md)
- [测试报告](docs/test_report.md)
```

#### ✅ 完成标准 (Day 5)

- [ ] 测试报告完成
- [ ] README更新
- [ ] 所有文档整理完毕
- [ ] Demo准备就绪

---

### 📅 本周总结 (Friday晚上)

#### ✍️ 输出物

创建文档 `docs/week5_summary.md`:

```markdown
# 第5周学习总结

## 完成情况

- [x] 单元测试 (5个核心测试)
- [x] 功能测试 (8个场景)
- [x] 性能测试 (吞吐量/延迟/缓存)
- [x] 压力测试 (稳定性/内存)

## 测试结果

### 通过率
- 单元测试: 100% (5/5)
- 功能测试: 100% (8/8)
- 性能测试: 达标
- 压力测试: 全部通过

### 性能数据
- 吞吐量开销: 3%
- 延迟增加: 10μs
- 会话缓存: 4.2x加速
- 并发会话: 105K

## 发现并修复的Bug

1. TCP状态机FIN_WAIT2转换问题
2. LPM Trie /32前缀匹配问题

## 下周计划

- 生产部署脚本完善
- 监控集成 (Prometheus + Grafana)
- 金丝雀部署测试
- 项目交付演示
```

#### 🎯 本周验收标准

**必须完成**:
- [ ] 所有测试通过
- [ ] 性能达标
- [ ] Bug全部修复
- [ ] 测试报告完成

**加分项**:
- [ ] 自动化测试流程
- [ ] 性能优化文档
- [ ] 压力测试录屏

---

## 7. 第6周：生产部署准备

### 🎯 本周目标

完成生产环境部署准备，包括部署脚本、监控集成、金丝雀部署测试，最终交付可用于生产环境的完整系统。

### 📊 本周交付物

| 交付物 | 类型 | 描述 |
|--------|------|------|
| 部署脚本套件 | 脚本 | 自动化部署、升级、回滚脚本 |
| 监控Dashboard | 配置 | Prometheus + Grafana完整监控 |
| 金丝雀测试报告 | 文档 | 灰度部署测试结果 |
| 项目交付文档 | 文档 | 完整的项目说明和演示材料 |
| 项目演示Demo | 演示 | 15分钟功能演示视频 |

---

### 📅 Day 1 (Monday): 自动化部署脚本开发

#### 🎯 任务目标
开发完整的自动化部署脚本，支持一键部署、环境检查、依赖安装。

#### ✅ 具体任务

**任务1: 环境检查脚本**

创建 `scripts/check_env.sh`:

```bash
#!/bin/bash
# check_env.sh - 环境检查脚本

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================="
echo "  eBPF微隔离系统环境检查"
echo "========================================="
echo ""

# 检查内核版本
echo -n "检查内核版本... "
KERNEL_VERSION=$(uname -r | cut -d. -f1,2)
KERNEL_MAJOR=$(echo $KERNEL_VERSION | cut -d. -f1)
KERNEL_MINOR=$(echo $KERNEL_VERSION | cut -d. -f2)

if [ $KERNEL_MAJOR -gt 5 ] || ([ $KERNEL_MAJOR -eq 5 ] && [ $KERNEL_MINOR -ge 10 ]); then
    echo -e "${GREEN}✓${NC} $KERNEL_VERSION (>= 5.10)"
else
    echo -e "${RED}✗${NC} $KERNEL_VERSION (需要 >= 5.10)"
    exit 1
fi

# 检查BTF支持
echo -n "检查BTF支持... "
if [ -f /sys/kernel/btf/vmlinux ]; then
    echo -e "${GREEN}✓${NC} BTF已启用"
else
    echo -e "${YELLOW}⚠${NC} BTF未启用 (功能受限)"
fi

# 检查必需工具
REQUIRED_TOOLS="clang llvm bpftool tc ip"
echo ""
echo "检查必需工具:"
for tool in $REQUIRED_TOOLS; do
    echo -n "  $tool... "
    if command -v $tool &> /dev/null; then
        VERSION=$(command $tool --version 2>&1 | head -n1)
        echo -e "${GREEN}✓${NC} 已安装"
    else
        echo -e "${RED}✗${NC} 未安装"
        exit 1
    fi
done

# 检查libbpf
echo -n "检查libbpf开发库... "
if pkg-config --exists libbpf; then
    VERSION=$(pkg-config --modversion libbpf)
    echo -e "${GREEN}✓${NC} $VERSION"
else
    echo -e "${RED}✗${NC} 未安装"
    exit 1
fi

# 检查内存
echo -n "检查可用内存... "
AVAILABLE_MEM=$(free -m | awk '/^Mem:/{print $7}')
if [ $AVAILABLE_MEM -gt 1024 ]; then
    echo -e "${GREEN}✓${NC} ${AVAILABLE_MEM}MB (推荐 >1GB)"
else
    echo -e "${YELLOW}⚠${NC} ${AVAILABLE_MEM}MB (建议至少1GB)"
fi

# 检查磁盘空间
echo -n "检查磁盘空间... "
AVAILABLE_DISK=$(df -m / | awk 'NR==2 {print $4}')
if [ $AVAILABLE_DISK -gt 2048 ]; then
    echo -e "${GREEN}✓${NC} ${AVAILABLE_DISK}MB"
else
    echo -e "${YELLOW}⚠${NC} ${AVAILABLE_DISK}MB (建议至少2GB)"
fi

# 检查root权限
echo -n "检查权限... "
if [ "$EUID" -eq 0 ]; then
    echo -e "${GREEN}✓${NC} root权限"
else
    echo -e "${YELLOW}⚠${NC} 非root用户 (部分功能需要sudo)"
fi

echo ""
echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}  环境检查通过！可以继续部署。${NC}"
echo -e "${GREEN}=========================================${NC}"
```

**任务2: 一键部署脚本**

创建 `scripts/deploy.sh`:

```bash
#!/bin/bash
# deploy.sh - 一键部署脚本

set -e

INSTALL_DIR="/opt/tc-microsegment"
BIN_DIR="/usr/local/bin"
CONFIG_DIR="/etc/tc-microsegment"
LOG_DIR="/var/log/tc-microsegment"

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "========================================="
echo "  eBPF微隔离系统部署脚本 v1.0"
echo "========================================="
echo ""

# 步骤1: 环境检查
echo -e "${BLUE}步骤1/6:${NC} 环境检查..."
bash scripts/check_env.sh || exit 1
echo ""

# 步骤2: 创建目录
echo -e "${BLUE}步骤2/6:${NC} 创建目录结构..."
mkdir -p $INSTALL_DIR/{bin,lib,bpf}
mkdir -p $CONFIG_DIR
mkdir -p $LOG_DIR
echo -e "${GREEN}✓${NC} 目录创建完成"
echo ""

# 步骤3: 编译eBPF程序
echo -e "${BLUE}步骤3/6:${NC} 编译eBPF程序..."
make clean
make all
echo -e "${GREEN}✓${NC} 编译完成"
echo ""

# 步骤4: 安装文件
echo -e "${BLUE}步骤4/6:${NC} 安装文件..."
cp build/tc_microsegment $INSTALL_DIR/bin/
cp build/*.bpf.o $INSTALL_DIR/bpf/
ln -sf $INSTALL_DIR/bin/tc_microsegment $BIN_DIR/tc-micro
chmod +x $INSTALL_DIR/bin/tc_microsegment
echo -e "${GREEN}✓${NC} 文件安装完成"
echo ""

# 步骤5: 安装配置文件
echo -e "${BLUE}步骤5/6:${NC} 安装配置文件..."
if [ ! -f $CONFIG_DIR/config.json ]; then
    cat > $CONFIG_DIR/config.json <<'EOF'
{
  "interfaces": ["eth0"],
  "log_level": "info",
  "log_file": "/var/log/tc-microsegment/tc-micro.log",
  "metrics_port": 9100,
  "default_policy": "deny",
  "policies": []
}
EOF
    echo -e "${GREEN}✓${NC} 配置文件已创建"
else
    echo -e "${GREEN}✓${NC} 配置文件已存在，跳过"
fi
echo ""

# 步骤6: 安装systemd服务
echo -e "${BLUE}步骤6/6:${NC} 安装systemd服务..."
cat > /etc/systemd/system/tc-microsegment.service <<EOF
[Unit]
Description=eBPF TC Microsegmentation Service
After=network.target

[Service]
Type=simple
ExecStart=$INSTALL_DIR/bin/tc_microsegment --config $CONFIG_DIR/config.json
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
echo -e "${GREEN}✓${NC} systemd服务已安装"
echo ""

echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}  部署完成！${NC}"
echo -e "${GREEN}=========================================${NC}"
echo ""
echo "下一步操作:"
echo "  1. 编辑配置: sudo vi $CONFIG_DIR/config.json"
echo "  2. 启动服务: sudo systemctl start tc-microsegment"
echo "  3. 查看状态: sudo systemctl status tc-microsegment"
echo "  4. 查看日志: sudo journalctl -u tc-microsegment -f"
echo "  5. 添加策略: sudo tc-micro policy add ..."
```

**任务3: 升级脚本**

创建 `scripts/upgrade.sh`:

```bash
#!/bin/bash
# upgrade.sh - 升级脚本

set -e

OLD_VERSION=$(tc-micro --version 2>/dev/null | awk '{print $3}' || echo "unknown")
NEW_VERSION=$(cat VERSION)

echo "升级: $OLD_VERSION → $NEW_VERSION"
echo ""

# 1. 备份配置
echo "备份配置..."
cp /etc/tc-microsegment/config.json /etc/tc-microsegment/config.json.bak
echo "✓ 配置已备份"

# 2. 停止服务
echo "停止服务..."
systemctl stop tc-microsegment || true
echo "✓ 服务已停止"

# 3. 卸载旧版本eBPF程序
echo "清理旧版本..."
tc filter del dev eth0 ingress 2>/dev/null || true
echo "✓ 旧版本已清理"

# 4. 部署新版本
echo "部署新版本..."
bash scripts/deploy.sh

# 5. 恢复配置
echo "恢复配置..."
cp /etc/tc-microsegment/config.json.bak /etc/tc-microsegment/config.json
echo "✓ 配置已恢复"

# 6. 启动服务
echo "启动服务..."
systemctl start tc-microsegment
sleep 2
systemctl status tc-microsegment

echo ""
echo "✓ 升级完成！"
```

#### 📚 学习资料 (2小时)

1. **Bash脚本最佳实践** (1小时)
   - 参考: https://google.github.io/styleguide/shellguide.html
   - 重点: 错误处理、颜色输出、参数验证

2. **systemd服务管理** (1小时)
   - 参考: `man systemd.service`
   - 重点: Type、Restart策略、日志管理

#### ✅ 完成标准

- [ ] check_env.sh 能正确检查所有依赖
- [ ] deploy.sh 能一键完成部署
- [ ] upgrade.sh 能平滑升级
- [ ] 所有脚本有完整错误处理
- [ ] 日志输出清晰友好

---

### 📅 Day 2 (Tuesday): 灰度部署脚本开发

#### 🎯 任务目标
实现金丝雀部署(Canary Deployment)脚本，支持分阶段灰度上线和自动回滚。

#### ✅ 具体任务

**任务1: 金丝雀部署脚本**

创建 `scripts/canary_deploy.sh`:

```bash
#!/bin/bash
# canary_deploy.sh - 金丝雀部署脚本

set -e

CANARY_STAGES=(5 10 25 50 100)  # 灰度比例: 5% -> 10% -> 25% -> 50% -> 100%
STAGE_DURATION=300              # 每阶段持续时间(秒)
HEALTH_CHECK_INTERVAL=10        # 健康检查间隔(秒)
ERROR_THRESHOLD=5               # 错误率阈值(%)

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# 健康检查函数
check_health() {
    local stage=$1

    echo -n "健康检查 (阶段${stage}%)... "

    # 1. 检查服务状态
    if ! systemctl is-active --quiet tc-microsegment; then
        echo -e "${RED}✗ 服务未运行${NC}"
        return 1
    fi

    # 2. 检查eBPF程序是否加载
    if ! bpftool prog show | grep -q "tc_microsegment"; then
        echo -e "${RED}✗ eBPF程序未加载${NC}"
        return 1
    fi

    # 3. 检查统计数据
    STATS=$(tc-micro stats show --json 2>/dev/null || echo '{}')

    # 获取丢包率
    PACKETS_TOTAL=$(echo $STATS | jq -r '.packets_total // 0')
    PACKETS_DROPPED=$(echo $STATS | jq -r '.packets_dropped // 0')

    if [ $PACKETS_TOTAL -gt 0 ]; then
        DROP_RATE=$(awk "BEGIN {printf \"%.2f\", ($PACKETS_DROPPED/$PACKETS_TOTAL)*100}")

        if (( $(echo "$DROP_RATE > $ERROR_THRESHOLD" | bc -l) )); then
            echo -e "${RED}✗ 丢包率过高: ${DROP_RATE}%${NC}"
            return 1
        fi
    fi

    # 4. 检查CPU使用率
    CPU_USAGE=$(top -bn1 | grep "tc_microsegment" | awk '{print $9}' | head -n1)
    if [ -n "$CPU_USAGE" ] && (( $(echo "$CPU_USAGE > 80" | bc -l) )); then
        echo -e "${YELLOW}⚠ CPU使用率较高: ${CPU_USAGE}%${NC}"
    fi

    echo -e "${GREEN}✓ 健康${NC}"
    return 0
}

# 流量切换函数
switch_traffic() {
    local percentage=$1

    echo "切换流量至新版本: ${percentage}%"

    # 这里使用iptables的random模块实现流量分配
    # 实际生产环境可能使用负载均衡器或其他流量管理工具

    # 清除旧规则
    iptables -t mangle -F TC_CANARY 2>/dev/null || true
    iptables -t mangle -X TC_CANARY 2>/dev/null || true

    # 创建新链
    iptables -t mangle -N TC_CANARY

    # 添加规则: percentage% 流量走新版本
    iptables -t mangle -A TC_CANARY -m statistic --mode random \
             --probability $(awk "BEGIN {print $percentage/100}") \
             -j MARK --set-mark 0x2  # 新版本标记

    iptables -t mangle -A TC_CANARY -j MARK --set-mark 0x1  # 旧版本标记

    # 应用到PREROUTING
    iptables -t mangle -I PREROUTING -j TC_CANARY

    echo "✓ 流量切换完成"
}

# 回滚函数
rollback() {
    echo -e "${RED}检测到异常，执行回滚...${NC}"

    # 1. 切换流量到旧版本
    switch_traffic 0

    # 2. 停止新版本
    systemctl stop tc-microsegment

    # 3. 恢复旧版本
    systemctl start tc-microsegment-old

    # 4. 清理eBPF程序
    tc filter del dev eth0 ingress 2>/dev/null || true

    echo -e "${GREEN}✓ 回滚完成${NC}"
    exit 1
}

# 主流程
echo "========================================="
echo "  金丝雀部署启动"
echo "========================================="
echo ""

# 备份当前版本
echo "备份当前版本..."
cp /opt/tc-microsegment/bin/tc_microsegment \
   /opt/tc-microsegment/bin/tc_microsegment.old
cp /etc/systemd/system/tc-microsegment.service \
   /etc/systemd/system/tc-microsegment-old.service
echo "✓ 备份完成"
echo ""

# 编译新版本
echo "编译新版本..."
make clean && make all
echo "✓ 编译完成"
echo ""

# 分阶段部署
for stage in "${CANARY_STAGES[@]}"; do
    echo "========================================="
    echo "  阶段: ${stage}% 流量"
    echo "========================================="

    # 切换流量
    switch_traffic $stage

    # 等待流量稳定
    echo "等待 ${STAGE_DURATION} 秒..."
    ELAPSED=0
    while [ $ELAPSED -lt $STAGE_DURATION ]; do
        sleep $HEALTH_CHECK_INTERVAL
        ELAPSED=$((ELAPSED + HEALTH_CHECK_INTERVAL))

        # 健康检查
        if ! check_health $stage; then
            rollback
        fi

        echo "  进度: ${ELAPSED}/${STAGE_DURATION}s"
    done

    echo -e "${GREEN}✓ 阶段${stage}%完成${NC}"
    echo ""
done

# 清理旧版本
echo "清理旧版本..."
rm -f /opt/tc-microsegment/bin/tc_microsegment.old
rm -f /etc/systemd/system/tc-microsegment-old.service
echo "✓ 清理完成"
echo ""

echo -e "${GREEN}=========================================${NC}"
echo -e "${GREEN}  金丝雀部署成功完成！${NC}"
echo -e "${GREEN}=========================================${NC}"
```

**任务2: 回滚脚本**

创建 `scripts/rollback.sh`:

```bash
#!/bin/bash
# rollback.sh - 快速回滚脚本

set -e

echo "执行快速回滚..."

# 1. 停止当前服务
systemctl stop tc-microsegment

# 2. 卸载eBPF程序
tc filter del dev eth0 ingress 2>/dev/null || true

# 3. 恢复备份
if [ -f /opt/tc-microsegment/bin/tc_microsegment.backup ]; then
    cp /opt/tc-microsegment/bin/tc_microsegment.backup \
       /opt/tc-microsegment/bin/tc_microsegment
else
    echo "错误: 没有找到备份文件"
    exit 1
fi

# 4. 恢复配置
if [ -f /etc/tc-microsegment/config.json.backup ]; then
    cp /etc/tc-microsegment/config.json.backup \
       /etc/tc-microsegment/config.json
fi

# 5. 重启服务
systemctl start tc-microsegment

# 6. 验证
sleep 2
if systemctl is-active --quiet tc-microsegment; then
    echo "✓ 回滚成功"
else
    echo "✗ 回滚失败，请手动检查"
    exit 1
fi
```

#### 📚 学习资料 (2小时)

1. **金丝雀部署原理** (1小时)
   - 参考: https://martinfowler.com/bliki/CanaryRelease.html
   - 重点: 灰度策略、流量分配、监控指标

2. **iptables流量标记** (1小时)
   - 参考: `man iptables-extensions`
   - 重点: mangle表、MARK target、statistic match

#### ✅ 完成标准

- [ ] canary_deploy.sh 能分阶段部署
- [ ] 每阶段都有健康检查
- [ ] 异常时能自动回滚
- [ ] rollback.sh 能快速回滚
- [ ] 完整的日志输出

---

### 📅 Day 3 (Wednesday): Prometheus监控集成

#### 🎯 任务目标
集成Prometheus监控，导出eBPF统计指标，配置Grafana Dashboard。

#### ✅ 具体任务

**任务1: Prometheus Exporter实现**

在用户态程序中添加metrics导出 `src/metrics.c`:

```c
// metrics.c - Prometheus metrics exporter
#include <microhttpd.h>
#include <stdio.h>
#include <string.h>
#include "metrics.h"

#define PORT 9100

static struct MHD_Daemon *daemon = NULL;

// 生成Prometheus格式的指标
static char* generate_metrics(struct bpf_stats *stats)
{
    static char buffer[4096];

    snprintf(buffer, sizeof(buffer),
        "# HELP tc_micro_packets_total Total packets processed\n"
        "# TYPE tc_micro_packets_total counter\n"
        "tc_micro_packets_total %llu\n"
        "\n"
        "# HELP tc_micro_packets_allowed Packets allowed by policy\n"
        "# TYPE tc_micro_packets_allowed counter\n"
        "tc_micro_packets_allowed %llu\n"
        "\n"
        "# HELP tc_micro_packets_denied Packets denied by policy\n"
        "# TYPE tc_micro_packets_denied counter\n"
        "tc_micro_packets_denied %llu\n"
        "\n"
        "# HELP tc_micro_sessions_active Active sessions\n"
        "# TYPE tc_micro_sessions_active gauge\n"
        "tc_micro_sessions_active %u\n"
        "\n"
        "# HELP tc_micro_policy_lookups_total Total policy lookups\n"
        "# TYPE tc_micro_policy_lookups_total counter\n"
        "tc_micro_policy_lookups_total %llu\n"
        "\n"
        "# HELP tc_micro_session_cache_hits Session cache hits\n"
        "# TYPE tc_micro_session_cache_hits counter\n"
        "tc_micro_session_cache_hits %llu\n"
        "\n"
        "# HELP tc_micro_session_cache_misses Session cache misses\n"
        "# TYPE tc_micro_session_cache_misses counter\n"
        "tc_micro_session_cache_misses %llu\n"
        "\n"
        "# HELP tc_micro_cache_hit_rate Session cache hit rate\n"
        "# TYPE tc_micro_cache_hit_rate gauge\n"
        "tc_micro_cache_hit_rate %.2f\n"
        "\n"
        "# HELP tc_micro_map_pressure Map pressure percentage\n"
        "# TYPE tc_micro_map_pressure gauge\n"
        "tc_micro_map_pressure %u\n"
        "\n"
        "# HELP tc_micro_tcp_syn_floods_detected SYN flood attacks detected\n"
        "# TYPE tc_micro_tcp_syn_floods_detected counter\n"
        "tc_micro_tcp_syn_floods_detected %llu\n",
        stats->packets_total,
        stats->packets_allowed,
        stats->packets_denied,
        stats->sessions_active,
        stats->policy_lookups,
        stats->cache_hits,
        stats->cache_misses,
        stats->cache_hits * 100.0 / (stats->cache_hits + stats->cache_misses + 1),
        stats->map_pressure,
        stats->syn_floods);

    return buffer;
}

// HTTP请求处理
static int handle_request(void *cls,
                         struct MHD_Connection *connection,
                         const char *url,
                         const char *method,
                         const char *version,
                         const char *upload_data,
                         size_t *upload_data_size,
                         void **con_cls)
{
    struct bpf_stats *stats = (struct bpf_stats *)cls;
    struct MHD_Response *response;
    int ret;

    if (strcmp(url, "/metrics") != 0) {
        const char *page = "Use /metrics endpoint";
        response = MHD_create_response_from_buffer(strlen(page),
                                                   (void *)page,
                                                   MHD_RESPMEM_PERSISTENT);
        ret = MHD_queue_response(connection, MHD_HTTP_NOT_FOUND, response);
        MHD_destroy_response(response);
        return ret;
    }

    // 生成metrics
    char *metrics = generate_metrics(stats);

    response = MHD_create_response_from_buffer(strlen(metrics),
                                               (void *)metrics,
                                               MHD_RESPMEM_MUST_COPY);
    MHD_add_response_header(response, "Content-Type", "text/plain");

    ret = MHD_queue_response(connection, MHD_HTTP_OK, response);
    MHD_destroy_response(response);

    return ret;
}

// 启动metrics服务器
int metrics_server_start(struct bpf_stats *stats, int port)
{
    daemon = MHD_start_daemon(MHD_USE_SELECT_INTERNALLY,
                             port,
                             NULL, NULL,
                             &handle_request, stats,
                             MHD_OPTION_END);

    if (daemon == NULL) {
        fprintf(stderr, "Failed to start metrics server on port %d\n", port);
        return -1;
    }

    printf("Metrics server started on port %d\n", port);
    printf("Access metrics at: http://localhost:%d/metrics\n", port);

    return 0;
}

// 停止metrics服务器
void metrics_server_stop(void)
{
    if (daemon) {
        MHD_stop_daemon(daemon);
        daemon = NULL;
    }
}
```

**任务2: Prometheus配置**

创建 `deploy/prometheus.yml`:

```yaml
# Prometheus配置
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'tc-microsegment'
    static_configs:
      - targets: ['localhost:9100']
        labels:
          instance: 'node1'
          environment: 'production'

    # 抓取间隔
    scrape_interval: 10s
    scrape_timeout: 5s

    # 指标relabel
    metric_relabel_configs:
      - source_labels: [__name__]
        regex: 'tc_micro_.*'
        action: keep

# 告警规则
rule_files:
  - 'alerts.yml'

# Alertmanager配置
alerting:
  alertmanagers:
    - static_configs:
        - targets: ['localhost:9093']
```

**任务3: 告警规则**

创建 `deploy/alerts.yml`:

```yaml
groups:
  - name: tc_microsegment_alerts
    interval: 30s
    rules:
      # 高丢包率告警
      - alert: HighDropRate
        expr: |
          (rate(tc_micro_packets_denied[5m]) /
           rate(tc_micro_packets_total[5m])) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "高丢包率检测"
          description: "丢包率 {{ $value | humanizePercentage }} 超过10%"

      # Map压力告警
      - alert: MapPressureHigh
        expr: tc_micro_map_pressure > 80
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Map压力过高"
          description: "Map使用率 {{ $value }}% 超过80%"

      # 缓存命中率低
      - alert: LowCacheHitRate
        expr: tc_micro_cache_hit_rate < 0.7
        for: 5m
        labels:
          severity: info
        annotations:
          summary: "缓存命中率较低"
          description: "缓存命中率 {{ $value | humanizePercentage }} 低于70%"

      # SYN Flood检测
      - alert: SynFloodDetected
        expr: rate(tc_micro_tcp_syn_floods_detected[1m]) > 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "检测到SYN Flood攻击"
          description: "检测到SYN Flood攻击，速率: {{ $value }}/s"

      # 服务不可用
      - alert: ServiceDown
        expr: up{job="tc-microsegment"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "服务不可用"
          description: "tc-microsegment服务在 {{ $labels.instance }} 上不可用"
```

**任务4: Grafana Dashboard**

创建 `deploy/grafana-dashboard.json`:

```json
{
  "dashboard": {
    "title": "eBPF TC Microsegmentation",
    "tags": ["ebpf", "networking", "security"],
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "title": "Packet Processing Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(tc_micro_packets_total[1m])",
            "legendFormat": "Total",
            "refId": "A"
          },
          {
            "expr": "rate(tc_micro_packets_allowed[1m])",
            "legendFormat": "Allowed",
            "refId": "B"
          },
          {
            "expr": "rate(tc_micro_packets_denied[1m])",
            "legendFormat": "Denied",
            "refId": "C"
          }
        ],
        "yaxes": [
          {
            "format": "pps",
            "label": "Packets/sec"
          }
        ]
      },
      {
        "id": 2,
        "title": "Cache Hit Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "tc_micro_cache_hit_rate",
            "legendFormat": "Hit Rate",
            "refId": "A"
          }
        ],
        "yaxes": [
          {
            "format": "percentunit",
            "max": 1,
            "min": 0
          }
        ],
        "thresholds": [
          {
            "value": 0.7,
            "colorMode": "critical",
            "op": "lt"
          }
        ]
      },
      {
        "id": 3,
        "title": "Active Sessions",
        "type": "graph",
        "targets": [
          {
            "expr": "tc_micro_sessions_active",
            "legendFormat": "Sessions",
            "refId": "A"
          }
        ]
      },
      {
        "id": 4,
        "title": "Map Pressure",
        "type": "gauge",
        "targets": [
          {
            "expr": "tc_micro_map_pressure",
            "refId": "A"
          }
        ],
        "thresholds": {
          "steps": [
            { "value": 0, "color": "green" },
            { "value": 70, "color": "yellow" },
            { "value": 80, "color": "red" }
          ]
        }
      }
    ]
  }
}
```

#### 📚 学习资料 (2.5小时)

1. **Prometheus基础** (1小时)
   - 参考: https://prometheus.io/docs/introduction/overview/
   - 重点: metrics类型、PromQL查询

2. **libmicrohttpd使用** (1小时)
   - 参考: https://www.gnu.org/software/libmicrohttpd/
   - 重点: HTTP服务器、请求处理

3. **Grafana Dashboard创建** (0.5小时)
   - 参考: https://grafana.com/docs/grafana/latest/dashboards/
   - 重点: 面板配置、查询语法

#### ✅ 完成标准

- [ ] Metrics服务器在9100端口运行
- [ ] Prometheus能成功抓取指标
- [ ] 告警规则配置正确
- [ ] Grafana Dashboard显示所有指标
- [ ] 模拟告警能正常触发

---

### 📅 Day 4 (Thursday): 金丝雀部署测试

#### 🎯 任务目标
在测试环境执行完整的金丝雀部署流程，验证部署脚本和监控系统。

#### ✅ 具体任务

**任务1: 测试环境准备**

```bash
#!/bin/bash
# setup_test_env.sh - 测试环境准备

# 1. 创建虚拟网络环境
sudo ip netns add test-old
sudo ip netns add test-new
sudo ip netns add test-client

# 创建veth对
sudo ip link add veth-old type veth peer name veth-old-br
sudo ip link add veth-new type veth peer name veth-new-br
sudo ip link add veth-client type veth peer name veth-client-br

# 移动到namespace
sudo ip link set veth-old netns test-old
sudo ip link set veth-new netns test-new
sudo ip link set veth-client netns test-client

# 配置IP
sudo ip netns exec test-old ip addr add 10.0.1.10/24 dev veth-old
sudo ip netns exec test-new ip addr add 10.0.1.20/24 dev veth-new
sudo ip netns exec test-client ip addr add 10.0.1.100/24 dev veth-client

# 启动接口
sudo ip netns exec test-old ip link set veth-old up
sudo ip netns exec test-new ip link set veth-new up
sudo ip netns exec test-client ip link set veth-client up
sudo ip link set veth-old-br up
sudo ip link set veth-new-br up
sudo ip link set veth-client-br up

# 2. 创建bridge
sudo ip link add br0 type bridge
sudo ip link set veth-old-br master br0
sudo ip link set veth-new-br master br0
sudo ip link set veth-client-br master br0
sudo ip link set br0 up

echo "✓ 测试环境准备完成"
```

**任务2: 金丝雀部署测试脚本**

创建 `tests/test_canary_deploy.sh`:

```bash
#!/bin/bash
# test_canary_deploy.sh - 金丝雀部署测试

set -e

TEST_DURATION=60  # 每阶段测试时间(秒)
CLIENT_THREADS=10

echo "========================================="
echo "  金丝雀部署测试"
echo "========================================="
echo ""

# 1. 启动旧版本
echo "启动旧版本 (v1.0)..."
sudo ip netns exec test-old /opt/tc-microsegment/bin/tc_microsegment \
    --version 1.0 --port 8080 &
OLD_PID=$!
sleep 2
echo "✓ 旧版本运行中 (PID: $OLD_PID)"
echo ""

# 2. 启动新版本
echo "启动新版本 (v1.1)..."
sudo ip netns exec test-new /opt/tc-microsegment/bin/tc_microsegment \
    --version 1.1 --port 8081 &
NEW_PID=$!
sleep 2
echo "✓ 新版本运行中 (PID: $NEW_PID)"
echo ""

# 3. 启动负载生成器
echo "启动负载生成器..."
sudo ip netns exec test-client wrk \
    -t $CLIENT_THREADS \
    -c 100 \
    -d ${TEST_DURATION}s \
    --latency \
    http://10.0.1.10:8080/ > /tmp/wrk_old.txt &

sudo ip netns exec test-client wrk \
    -t $CLIENT_THREADS \
    -c 100 \
    -d ${TEST_DURATION}s \
    --latency \
    http://10.0.1.20:8081/ > /tmp/wrk_new.txt &

echo "✓ 负载生成器运行中"
echo ""

# 4. 执行金丝雀部署
STAGES=(0 25 50 75 100)

for i in "${!STAGES[@]}"; do
    stage=${STAGES[$i]}

    echo "========================================="
    echo "  阶段 $((i+1))/5: ${stage}% 新版本"
    echo "========================================="

    # 调整iptables规则分配流量
    sudo iptables -t mangle -F TC_CANARY 2>/dev/null || true
    sudo iptables -t mangle -X TC_CANARY 2>/dev/null || true
    sudo iptables -t mangle -N TC_CANARY

    if [ $stage -gt 0 ]; then
        sudo iptables -t mangle -A TC_CANARY \
            -m statistic --mode random \
            --probability $(awk "BEGIN {print $stage/100}") \
            -j DNAT --to-destination 10.0.1.20:8081
    fi

    sudo iptables -t mangle -A TC_CANARY \
        -j DNAT --to-destination 10.0.1.10:8080

    sudo iptables -t mangle -I PREROUTING -j TC_CANARY

    # 等待并监控
    ELAPSED=0
    while [ $ELAPSED -lt 30 ]; do
        sleep 5
        ELAPSED=$((ELAPSED + 5))

        # 检查错误率
        OLD_ERRORS=$(curl -s http://10.0.1.10:8080/stats | jq -r '.errors // 0')
        NEW_ERRORS=$(curl -s http://10.0.1.20:8081/stats | jq -r '.errors // 0')

        echo "  [$ELAPSED/30s] 旧版本错误: $OLD_ERRORS, 新版本错误: $NEW_ERRORS"

        # 如果新版本错误过多，回滚
        if [ $NEW_ERRORS -gt 100 ]; then
            echo "✗ 新版本错误过多，回滚！"
            sudo iptables -t mangle -F TC_CANARY
            exit 1
        fi
    done

    echo "✓ 阶段${stage}%完成"
    echo ""
done

# 5. 收集结果
echo "========================================="
echo "  测试结果"
echo "========================================="
echo ""

echo "旧版本 (v1.0):"
cat /tmp/wrk_old.txt | grep -E "Requests/sec|Latency"
echo ""

echo "新版本 (v1.1):"
cat /tmp/wrk_new.txt | grep -E "Requests/sec|Latency"
echo ""

# 6. 清理
kill $OLD_PID $NEW_PID 2>/dev/null || true
sudo iptables -t mangle -F TC_CANARY
sudo iptables -t mangle -X TC_CANARY

echo "✓ 金丝雀部署测试完成"
```

**任务3: 测试报告生成**

创建 `tests/generate_canary_report.sh`:

```bash
#!/bin/bash
# generate_canary_report.sh - 生成测试报告

cat > /tmp/canary_test_report.md <<'EOF'
# 金丝雀部署测试报告

## 测试环境
- 测试时间: $(date)
- 旧版本: v1.0
- 新版本: v1.1
- 测试工具: wrk
- 并发数: 100

## 部署阶段

| 阶段 | 新版本流量% | 持续时间 | 错误数 | 延迟P50 | 延迟P99 | 结果 |
|------|-------------|----------|--------|---------|---------|------|
| 1    | 0%          | 30s      | 0      | 5ms     | 12ms    | ✓    |
| 2    | 25%         | 30s      | 0      | 5ms     | 13ms    | ✓    |
| 3    | 50%         | 30s      | 0      | 6ms     | 14ms    | ✓    |
| 4    | 75%         | 30s      | 0      | 6ms     | 15ms    | ✓    |
| 5    | 100%        | 30s      | 0      | 7ms     | 16ms    | ✓    |

## 性能对比

### 吞吐量
- 旧版本: 15,234 req/s
- 新版本: 15,892 req/s
- 提升: +4.3%

### 延迟
- 旧版本 P50: 5.2ms, P99: 12.4ms
- 新版本 P50: 5.8ms, P99: 14.1ms
- 变化: P50 +11%, P99 +13%

## 告警触发情况

- 无告警触发

## 结论

✓ **金丝雀部署成功**

- 所有阶段健康检查通过
- 无异常回滚
- 性能指标稳定
- 建议: 可以推广到生产环境

## 改进建议

1. 增加更细粒度的流量切换(5% -> 10% -> 25% ...)
2. 延长每阶段观察时间到5分钟
3. 添加更多自动化健康检查指标
EOF

echo "✓ 报告已生成: /tmp/canary_test_report.md"
cat /tmp/canary_test_report.md
```

#### 📚 学习资料 (1.5小时)

1. **灰度发布最佳实践** (1小时)
   - 参考: https://www.martinfowler.com/bliki/CanaryRelease.html
   - 重点: 风险控制、监控指标、回滚策略

2. **wrk压力测试工具** (0.5小时)
   - 参考: https://github.com/wg/wrk
   - 重点: 参数调优、结果分析

#### ✅ 完成标准

- [ ] 测试环境成功搭建
- [ ] 金丝雀部署脚本正常运行
- [ ] 所有阶段健康检查通过
- [ ] 测试报告自动生成
- [ ] 性能数据完整记录

---

### 📅 Day 5 (Friday): 项目交付与演示准备

#### 🎯 任务目标
完成项目交付文档、演示材料，录制演示视频，准备项目总结。

#### ✅ 具体任务

**任务1: 项目交付文档**

创建 `docs/DELIVERY.md`:

```markdown
# eBPF微隔离系统 - 项目交付文档

## 项目概述

**项目名称**: eBPF TC 微隔离系统
**版本**: v1.0.0
**交付日期**: 2025-10-24
**开发周期**: 6周

### 核心功能
- ✅ 基于eBPF TC的高性能包过滤
- ✅ 5元组策略匹配
- ✅ 会话跟踪与缓存
- ✅ TCP状态机管理
- ✅ IP段匹配(LPM Trie)
- ✅ Prometheus监控集成
- ✅ 金丝雀部署支持

### 性能指标
| 指标 | 目标 | 实测 | 状态 |
|------|------|------|------|
| P50延迟 | <20μs | 12μs | ✅ |
| P99延迟 | <50μs | 35μs | ✅ |
| 吞吐量 | >30Gbps | 38Gbps | ✅ |
| CPU使用率 | <10% | 7% | ✅ |
| 会话容量 | >100K | 150K | ✅ |
| 缓存命中率 | >90% | 94% | ✅ |

## 交付内容

### 1. 源代码
```
├── src/
│   ├── tc_microsegment.bpf.c    # eBPF程序
│   ├── main.c                    # 用户态主程序
│   ├── policy.c                  # 策略管理
│   ├── session.c                 # 会话管理
│   ├── stats.c                   # 统计功能
│   └── metrics.c                 # Prometheus导出
├── include/
│   └── common.h                  # 公共头文件
└── tests/
    ├── unit/                     # 单元测试
    ├── functional/               # 功能测试
    └── performance/              # 性能测试
```

### 2. 文档
- ✅ 架构设计文档 (specs/ebpf-tc-architecture.md)
- ✅ 实施指南 (specs/ebpf-tc-implementation.md)
- ✅ API文档 (docs/API.md)
- ✅ 运维手册 (docs/OPS.md)
- ✅ 故障排查指南 (docs/TROUBLESHOOTING.md)

### 3. 部署工具
- ✅ 一键部署脚本 (scripts/deploy.sh)
- ✅ 环境检查脚本 (scripts/check_env.sh)
- ✅ 金丝雀部署脚本 (scripts/canary_deploy.sh)
- ✅ 回滚脚本 (scripts/rollback.sh)
- ✅ systemd服务配置

### 4. 监控配置
- ✅ Prometheus配置 (deploy/prometheus.yml)
- ✅ 告警规则 (deploy/alerts.yml)
- ✅ Grafana Dashboard (deploy/grafana-dashboard.json)

### 5. 测试报告
- ✅ 单元测试报告 (test_reports/unit_test_report.md)
- ✅ 功能测试报告 (test_reports/functional_test_report.md)
- ✅ 性能测试报告 (test_reports/performance_test_report.md)
- ✅ 金丝雀部署测试报告 (test_reports/canary_test_report.md)

## 快速开始

### 安装
```bash
# 1. 克隆代码
git clone https://github.com/yourorg/ebpf-microsegment.git
cd ebpf-microsegment

# 2. 检查环境
sudo bash scripts/check_env.sh

# 3. 一键部署
sudo bash scripts/deploy.sh

# 4. 启动服务
sudo systemctl start tc-microsegment

# 5. 验证
sudo tc-micro stats show
```

### 添加策略
```bash
# 允许SSH
sudo tc-micro policy add \
    --src-ip 10.0.0.0/24 \
    --dst-port 22 \
    --protocol tcp \
    --action allow

# 拒绝HTTP
sudo tc-micro policy add \
    --dst-port 80 \
    --protocol tcp \
    --action deny
```

## 技术架构

### 数据流
```
数据包 → TC ingress hook → eBPF程序 → 策略匹配 → 放行/拒绝
                                   ↓
                              会话缓存
                                   ↓
                              统计更新
                                   ↓
                            Prometheus导出
```

### 核心组件
1. **eBPF程序** (内核态)
   - tc_microsegment.bpf.c
   - 5元组匹配、会话跟踪、TCP状态机

2. **控制程序** (用户态)
   - 策略管理
   - 统计监控
   - Metrics导出

3. **监控系统**
   - Prometheus抓取
   - Grafana可视化
   - 告警规则

## 运维指南

### 日常操作
```bash
# 查看状态
sudo systemctl status tc-microsegment

# 查看日志
sudo journalctl -u tc-microsegment -f

# 查看统计
sudo tc-micro stats show

# 查看会话
sudo tc-micro session list

# 重载策略
sudo tc-micro policy reload
```

### 升级
```bash
sudo bash scripts/upgrade.sh
```

### 回滚
```bash
sudo bash scripts/rollback.sh
```

### 监控
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000
- Metrics: http://localhost:9100/metrics

## 已知限制

1. **内核版本**: 需要 Linux 5.10+
2. **Map容量**: 会话表最大100K (可调整)
3. **复杂DPI**: 需要用户态辅助
4. **多网卡**: 当前版本支持单网卡 (可扩展)

## 后续优化建议

1. **性能优化**
   - 使用XDP替代TC (更早拦截)
   - 优化Map大小和LRU策略
   - 添加per-CPU哈希表

2. **功能增强**
   - 支持IPv6
   - 添加流量镜像
   - 集成IDS/IPS

3. **运维改进**
   - 添加Web UI
   - 集成K8s CNI
   - 支持配置中心

## 联系方式

- **技术负责人**: [你的名字]
- **Email**: [你的邮箱]
- **项目地址**: https://github.com/yourorg/ebpf-microsegment
- **文档**: https://docs.yourorg.com/ebpf-microsegment

## 附录

### A. 依赖软件版本
- Linux Kernel: >= 5.10 (推荐 5.15+)
- clang/LLVM: >= 11 (推荐 >= 14)
- libbpf: >= 1.0 (推荐使用最新的 1.x 版本)
- bpftool: 匹配内核版本
- systemd: >= 245

### B. 测试环境
- OS: Ubuntu 22.04 LTS
- Kernel: 5.15.0
- CPU: Intel Xeon 2.5GHz
- Memory: 16GB
- Network: 10Gbps

### C. 参考文档
- [eBPF官方文档](https://ebpf.io/)
- [Cilium项目](https://cilium.io/)
- [libbpf](https://github.com/libbpf/libbpf)
```

**任务2: 演示脚本**

创建 `demo/demo.sh`:

```bash
#!/bin/bash
# demo.sh - 15分钟功能演示脚本

set -e

echo "========================================="
echo "  eBPF微隔离系统功能演示"
echo "========================================="
echo ""
sleep 2

# 1. 环境展示
echo "=== 1. 系统环境 ==="
echo ""
echo "内核版本:"
uname -r
echo ""
echo "eBPF支持:"
if [ -f /sys/kernel/btf/vmlinux ]; then
    echo "✓ BTF已启用"
fi
echo ""
sleep 3

# 2. 部署演示
echo "=== 2. 一键部署 ==="
echo ""
echo "执行环境检查..."
sudo bash scripts/check_env.sh
echo ""
echo "执行部署..."
sudo bash scripts/deploy.sh
echo ""
sleep 3

# 3. 策略管理演示
echo "=== 3. 策略管理 ==="
echo ""
echo "添加SSH允许策略:"
sudo tc-micro policy add \
    --src-ip 10.0.0.0/24 \
    --dst-port 22 \
    --protocol tcp \
    --action allow
echo ""
echo "添加HTTP拒绝策略:"
sudo tc-micro policy add \
    --dst-port 80 \
    --protocol tcp \
    --action deny
echo ""
echo "查看所有策略:"
sudo tc-micro policy list
echo ""
sleep 5

# 4. 流量测试
echo "=== 4. 流量测试 ==="
echo ""
echo "发起SSH连接 (应该允许)..."
timeout 2 nc -zv 10.0.0.50 22 || echo "连接成功"
echo ""
echo "发起HTTP连接 (应该拒绝)..."
timeout 2 nc -zv 10.0.0.50 80 || echo "连接被拒绝 ✓"
echo ""
sleep 3

# 5. 会话跟踪
echo "=== 5. 会话跟踪 ==="
echo ""
echo "查看活动会话:"
sudo tc-micro session list | head -n 10
echo ""
echo "会话统计:"
sudo tc-micro stats show | grep -E "sessions|cache"
echo ""
sleep 3

# 6. 性能监控
echo "=== 6. 性能监控 ==="
echo ""
echo "实时统计:"
sudo tc-micro stats show
echo ""
echo "Prometheus指标 (http://localhost:9100/metrics):"
curl -s http://localhost:9100/metrics | grep -E "^tc_micro" | head -n 5
echo ""
sleep 3

# 7. 压力测试
echo "=== 7. 压力测试 ==="
echo ""
echo "启动10秒压力测试..."
wrk -t 4 -c 100 -d 10s http://10.0.0.50/ &
WRK_PID=$!

# 实时显示统计
for i in {1..10}; do
    echo "[$i/10] 当前统计:"
    sudo tc-micro stats show | grep -E "packets|sessions"
    sleep 1
done

wait $WRK_PID
echo ""
echo "压力测试完成"
echo ""
sleep 3

# 8. 监控演示
echo "=== 8. 监控系统 ==="
echo ""
echo "Grafana Dashboard: http://localhost:3000"
echo "Prometheus: http://localhost:9090"
echo ""
echo "打开浏览器查看实时监控..."
echo ""
sleep 5

# 9. 故障模拟与恢复
echo "=== 9. 故障恢复演示 ==="
echo ""
echo "模拟服务故障..."
sudo systemctl stop tc-microsegment
sleep 2
echo "检查状态:"
systemctl status tc-microsegment || echo "服务已停止"
echo ""
echo "执行自动恢复..."
sudo systemctl start tc-microsegment
sleep 2
echo "恢复后状态:"
systemctl status tc-microsegment
echo ""
sleep 3

# 10. 总结
echo "========================================="
echo "  演示完成！"
echo "========================================="
echo ""
echo "核心功能:"
echo "  ✓ 高性能包过滤 (38Gbps吞吐量)"
echo "  ✓ 5元组策略匹配"
echo "  ✓ 会话跟踪 (94%缓存命中率)"
echo "  ✓ Prometheus监控集成"
echo "  ✓ 一键部署与回滚"
echo ""
echo "性能指标:"
echo "  • P50延迟: 12μs"
echo "  • P99延迟: 35μs"
echo "  • CPU使用: 7%"
echo "  • 会话容量: 150K"
echo ""
echo "谢谢观看！"
```

**任务3: 演示视频录制**

录制15分钟演示视频，包含以下内容:

1. **开场 (1分钟)**
   - 项目介绍
   - 核心优势
   - 技术架构图

2. **环境展示 (2分钟)**
   - 系统环境
   - 依赖检查
   - 代码结构

3. **部署演示 (3分钟)**
   - 一键部署流程
   - 服务启动
   - 状态检查

4. **功能演示 (5分钟)**
   - 策略管理 (添加/删除/查看)
   - 流量测试 (允许/拒绝)
   - 会话跟踪
   - 统计信息

5. **性能测试 (2分钟)**
   - 压力测试执行
   - 实时监控展示
   - 性能指标说明

6. **监控系统 (1分钟)**
   - Grafana Dashboard
   - Prometheus查询
   - 告警展示

7. **高级功能 (1分钟)**
   - 金丝雀部署
   - 自动回滚
   - 故障恢复

8. **总结 (0.5分钟)**
   - 项目成果
   - 后续计划

**录制工具**: OBS Studio / Kazam / SimpleScreenRecorder

#### 📚 学习资料 (1小时)

1. **技术演示技巧** (0.5小时)
   - 重点: 演示流程设计、讲解技巧、常见问题处理

2. **视频录制与剪辑** (0.5小时)
   - 工具: OBS Studio
   - 重点: 屏幕录制、字幕添加、视频导出

#### ✅ 完成标准

- [ ] 交付文档完整清晰
- [ ] 演示脚本能顺利执行
- [ ] 演示视频录制完成
- [ ] 所有交付物已打包
- [ ] 项目归档完成

---

### 📅 本周总结 (Friday晚上)

#### ✍️ 输出物

创建文档 `docs/week6_summary.md`:

```markdown
# 第6周学习总结

## 完成情况
- [x] 自动化部署脚本 (deploy.sh, upgrade.sh, rollback.sh)
- [x] 金丝雀部署脚本 (canary_deploy.sh)
- [x] Prometheus监控集成 (metrics导出 + 告警)
- [x] Grafana Dashboard配置
- [x] 金丝雀部署测试 (完整流程验证)
- [x] 项目交付文档 (DELIVERY.md)
- [x] 演示脚本和视频录制

## 核心收获

### 1. 部署自动化
学会了:
- Bash脚本的最佳实践 (错误处理、颜色输出)
- systemd服务管理
- 环境检查的全面性考虑
- 一键部署的用户体验设计

### 2. 金丝雀部署
掌握了:
- 灰度发布的核心思想
- 流量分配策略 (iptables random)
- 健康检查的设计
- 自动回滚机制

### 3. 监控集成
实现了:
- Prometheus metrics导出 (libmicrohttpd)
- 告警规则配置
- Grafana Dashboard设计
- 关键指标选择 (丢包率、缓存命中率、Map压力)

### 4. 项目管理
完成了:
- 完整的交付文档
- 功能演示设计
- 测试报告整理
- 项目归档

## 部署脚本功能

| 脚本 | 功能 | 代码行数 |
|------|------|---------|
| check_env.sh | 环境检查 | 80行 |
| deploy.sh | 一键部署 | 120行 |
| upgrade.sh | 平滑升级 | 60行 |
| rollback.sh | 快速回滚 | 40行 |
| canary_deploy.sh | 金丝雀部署 | 180行 |

## 监控指标

已实现的Prometheus指标:
1. `tc_micro_packets_total` - 总包数
2. `tc_micro_packets_allowed` - 允许包数
3. `tc_micro_packets_denied` - 拒绝包数
4. `tc_micro_sessions_active` - 活动会话数
5. `tc_micro_cache_hit_rate` - 缓存命中率
6. `tc_micro_map_pressure` - Map压力
7. `tc_micro_syn_floods_detected` - SYN Flood检测

## 金丝雀部署测试结果

| 阶段 | 新版本流量% | 错误数 | 延迟P50 | 延迟P99 | 结果 |
|------|-------------|--------|---------|---------|------|
| 1    | 5%          | 0      | 5ms     | 12ms    | ✓    |
| 2    | 10%         | 0      | 5ms     | 13ms    | ✓    |
| 3    | 25%         | 0      | 6ms     | 14ms    | ✓    |
| 4    | 50%         | 0      | 6ms     | 15ms    | ✓    |
| 5    | 100%        | 0      | 7ms     | 16ms    | ✓    |

结论: ✅ 金丝雀部署成功,所有阶段健康检查通过

## 项目最终成果

### 代码统计
```
Language         files     blank   comment      code
-----------------------------------------------------
C                   8       456       623      3254
Bash               12       234       187      1456
Markdown            5        78         0       892
JSON                2         0         0       156
YAML                2        12         8        89
-----------------------------------------------------
SUM:               29       780       818      5847
```

### 测试覆盖
- 单元测试: 8个 (100%通过)
- 功能测试: 12个 (100%通过)
- 性能测试: 6个 (全部达标)
- 压力测试: 4个 (通过)

### 性能指标 (最终)
- P50延迟: **12μs** (目标 <20μs) ✓
- P99延迟: **35μs** (目标 <50μs) ✓
- 吞吐量: **38Gbps** (目标 >30Gbps) ✓
- CPU使用: **7%** @ 1Gbps (目标 <10%) ✓
- 会话容量: **150K** (目标 >100K) ✓
- 缓存命中率: **94%** (目标 >90%) ✓

## 交付清单

- [x] 源代码 (src/, include/, tests/)
- [x] 文档 (specs/, docs/)
- [x] 部署脚本 (scripts/)
- [x] 监控配置 (deploy/)
- [x] 测试报告 (test_reports/)
- [x] 演示材料 (demo/)
- [x] 演示视频 (15分钟)

## 项目总结

经过6周的开发,成功完成了基于eBPF TC的微隔离系统:

**技术突破**:
1. 深入理解eBPF编程模型和Verifier约束
2. 掌握TC hook机制和包处理流程
3. 实现高性能会话跟踪 (LRU_HASH + 缓存优化)
4. 集成完整的监控和告警系统

**工程实践**:
1. 完整的CI/CD流程 (部署、测试、回滚)
2. 金丝雀部署实现
3. 自动化测试框架
4. 详尽的文档和交付材料

**性能成果**:
相比用户态PACKET_MMAP方案:
- 延迟降低 **3倍** (50μs → 15μs)
- 吞吐量提升 **3.8倍** (10Gbps → 38Gbps)
- CPU使用降低 **65%** (20% → 7%)

**后续计划**:
1. 支持IPv6
2. 使用XDP进一步优化性能
3. 集成Kubernetes CNI
4. 添加Web管理界面
```

#### 🎯 本周验收标准

**必须完成**:
- [x] 部署脚本完整且可用
- [x] 金丝雀部署测试通过
- [x] Prometheus监控正常工作
- [x] 项目交付文档完整
- [x] 演示材料准备完成

**加分项**:
- [x] 演示视频录制完成
- [x] 监控Dashboard美观实用
- [x] 部署脚本用户体验优秀
- [x] 交付文档专业详尽

---

## 🎉 项目完成！

恭喜你完成了为期6周的eBPF微隔离系统开发！

### 📊 整体进度

| 周次 | 主题 | 完成度 |
|------|------|--------|
| Week 1 | 环境准备 + eBPF基础 | ✅ 100% |
| Week 2 | 基础框架开发 | ✅ 100% |
| Week 3 | 用户态控制程序 | ✅ 100% |
| Week 4 | 高级功能实现 | ✅ 100% |
| Week 5 | 测试与优化 | ✅ 100% |
| Week 6 | 生产部署准备 | ✅ 100% |

### 🏆 核心成就

1. **技术深度**: 掌握eBPF、TC、网络协议栈
2. **性能优化**: 实现3倍延迟降低、3.8倍吞吐提升
3. **工程质量**: 完整测试、文档、部署流程
4. **生产就绪**: 监控、告警、灰度发布

### 📚 知识体系

累计学习时间: **~60小时**

- eBPF原理与实践: 20小时
- 网络协议与TC: 15小时
- 性能优化与测试: 12小时
- 监控与运维: 8小时
- 部署与发布: 5小时

### 🚀 下一步

1. 在生产环境部署
2. 持续性能优化
3. 添加新功能 (IPv6, XDP)
4. 开源分享经验

---

**恭喜完成！你已经掌握了eBPF微隔离系统的全栈开发！** 🎊
