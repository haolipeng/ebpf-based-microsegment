# Demo 1: Hello Packet Counter

[⬅️ 返回主目录](../README.md) | [➡️ 下一个: Demo 2](../02-5tuple-extractor/README.md)

---

## 📋 学习目标

通过这个最简单的 eBPF 程序,你将学习:

1. ✅ **TC Hook 基础** - 理解 `SEC("tc")` 和数据包拦截点
2. ✅ **eBPF Map** - 使用 `BPF_MAP_TYPE_ARRAY` 存储数据
3. ✅ **返回值语义** - `TC_ACT_OK` vs `TC_ACT_SHOT`
4. ✅ **原子操作** - 使用 `__sync_fetch_and_add()` 安全地更新计数器
5. ✅ **调试技巧** - 使用 `bpf_printk()` 和 `bpftool`

**难度**: ⭐☆☆☆☆ (入门级)
**学习时间**: 2-3 小时
**代码量**: ~80 行

---

## 🎯 程序功能

这个 eBPF 程序会:
- 统计通过网卡的**每个数据包**
- 将计数存储在 eBPF Map 中
- 允许所有数据包正常通过 (不影响网络功能)
- 打印调试信息到内核日志

---

## 🔧 前置知识

### 什么是 TC (Traffic Control)?

TC 是 Linux 内核的流量控制子系统,可以在网卡的 **ingress** (入站) 和 **egress** (出站) 路径上附加 eBPF 程序。

```
Ingress (入站):  外部网络 → 网卡 → [eBPF 程序] → 内核协议栈
Egress (出站):   内核协议栈 → [eBPF 程序] → 网卡 → 外部网络
```

### TC eBPF 返回值

- `TC_ACT_OK (0)`: **允许数据包继续传递** (最常用)
- `TC_ACT_SHOT (2)`: **丢弃数据包** (用于过滤)
- `TC_ACT_REDIRECT`: 重定向到其他接口
- `TC_ACT_PIPE`: 传递给下一个 TC 过滤器

---

## 📖 代码解析

### 1. eBPF Map 定义

```c
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);  // Map 类型: 数组
    __uint(max_entries, 1);             // 最大条目数: 1 (我们只需要一个计数器)
    __type(key, __u32);                 // Key 类型: 32位无符号整数 (索引)
    __type(value, __u64);               // Value 类型: 64位无符号整数 (计数器)
} packet_counter SEC(".maps");
```

**关键点**:
- `BPF_MAP_TYPE_ARRAY`: 固定大小的数组,通过整数索引访问 (O(1) 时间复杂度)
- `max_entries = 1`: 只有一个槽位 (索引 0)
- `__u64`: 64位计数器可以存储 18,446,744,073,709,551,615 个数据包

### 2. 主程序逻辑

```c
SEC("tc")
int tc_hello_packet(struct __sk_buff *skb)
{
    __u32 key = 0;  // 数组索引 (我们只有一个元素)

    // 从 Map 中查找计数器
    __u64 *count = bpf_map_lookup_elem(&packet_counter, &key);

    if (count) {
        // 原子递增 (线程安全)
        __sync_fetch_and_add(count, 1);

        // 调试输出
        bpf_printk("Hello! Packet #%llu received (len=%d bytes)\n",
                   *count, skb->len);
    }

    return TC_ACT_OK;  // 允许数据包通过
}
```

**关键点**:
- `SEC("tc")`: 标记这是一个 TC eBPF 程序
- `struct __sk_buff *skb`: 数据包上下文,包含元数据 (长度、协议等)
- `bpf_map_lookup_elem()`: 从 Map 中读取值 (返回指针或 NULL)
- `__sync_fetch_and_add()`: 原子操作,避免多核竞争
- `bpf_printk()`: 输出到 `/sys/kernel/debug/tracing/trace_pipe`

### 3. 为什么需要原子操作?

eBPF 程序可能在**多个 CPU 核心上同时执行**。如果不使用原子操作:

```c
// ❌ 错误: 非原子操作
*count = *count + 1;

// 可能发生竞争条件:
// CPU 0: 读取 count = 100
// CPU 1: 读取 count = 100
// CPU 0: 写入 count = 101
// CPU 1: 写入 count = 101  <- 丢失了一次递增!

// ✅ 正确: 原子操作
__sync_fetch_and_add(count, 1);
```

---

## 🚀 运行步骤

### Step 1: 编译 eBPF 程序

```bash
cd demos/01-hello-packet
make
```

**输出**:
```
Compiling eBPF program...
✓ Compiled successfully: hello.bpf.o
```

**幕后发生了什么**:
- `clang` 将 C 代码编译成 eBPF 字节码
- 生成 `hello.bpf.o` (ELF 格式目标文件)
- 包含 eBPF 程序和 Map 定义

### Step 2: 加载到内核

```bash
sudo make load
```

**输出**:
```
Loading eBPF program to interface lo...
✓ eBPF program loaded on lo

View packet count with:
  sudo bpftool map dump name packet_counter

View debug logs with:
  sudo cat /sys/kernel/debug/tracing/trace_pipe
```

**幕后发生了什么**:
- 创建 `clsact` qdisc (TC 队列规则)
- 将 eBPF 程序附加到 `lo` (loopback) 网卡的 **ingress** 路径
- 内核验证程序安全性 (eBPF 验证器)
- 程序开始拦截数据包

### Step 3: 查看初始状态

```bash
sudo bpftool map dump name packet_counter
```

**输出**:
```
key: 00 00 00 00  value: 00 00 00 00 00 00 00 00
Found 1 element
```

解释: 计数器初始值为 0 (8 字节全为 0)

### Step 4: 生成测试流量

```bash
ping -c 5 127.0.0.1
```

这会发送 5 个 ICMP 包到 loopback 接口。

### Step 5: 查看更新后的计数

```bash
sudo bpftool map dump name packet_counter
```

**输出**:
```
key: 00 00 00 00  value: 0a 00 00 00 00 00 00 00
Found 1 element
```

解释: `0x0a` = 10 (5 个 ping 请求 + 5 个 ping 响应)

### Step 6: 查看调试日志

```bash
sudo cat /sys/kernel/debug/tracing/trace_pipe
```

**输出**:
```
<idle>-0       [000] d.s11  1234.567890: bpf_trace_printk: Hello! Packet #1 received (len=98 bytes)
<idle>-0       [001] d.s11  1234.567901: bpf_trace_printk: Hello! Packet #2 received (len=98 bytes)
ping-12345     [002] d.s11  1234.567912: bpf_trace_printk: Hello! Packet #3 received (len=98 bytes)
...
```

### Step 7: 运行自动化测试

```bash
./test.sh
```

这个脚本会自动执行上述步骤并验证结果。

### Step 8: 卸载程序

```bash
sudo make unload
```

---

## 🔍 验证和调试

### 查看加载的 eBPF 程序

```bash
sudo bpftool prog show
```

**输出示例**:
```
123: sched_cls  name tc_hello_packet  tag a1b2c3d4e5f6a7b8  gpl
        loaded_at 2025-01-15T10:30:00+0000  uid 0
        xlated 96B  jited 64B  memlock 4096B  map_ids 45
```

### 查看 Map 信息

```bash
sudo bpftool map show
```

**输出示例**:
```
45: array  name packet_counter  flags 0x0
        key 4B  value 8B  max_entries 1  memlock 4096B
```

### 查看 TC 过滤器

```bash
sudo tc filter show dev lo ingress
```

**输出示例**:
```
filter protocol all pref 49152 bpf chain 0
filter protocol all pref 49152 bpf chain 0 handle 0x1 hello.bpf.o:[tc] direct-action not_in_hw id 123 tag a1b2c3d4e5f6a7b8
```

---

## ❓ 常见问题

### Q1: 为什么计数是 10 而不是 5?

**A**: ping 会产生 **双向流量**:
- 5 个 ICMP Echo Request (发出)
- 5 个 ICMP Echo Reply (收到)

我们的程序附加在 **ingress** (入站),只统计进入的包:
- 在 `lo` 上: 所有发出的包也会"进入" (loopback 特性)
- 因此统计了 5 + 5 = 10 个包

### Q2: 编译失败: "vmlinux.h: No such file"

**A**: 检查 BTF 支持:
```bash
ls /sys/kernel/btf/vmlinux
```

如果文件不存在,你的内核不支持 BTF。解决方法:
- 升级内核到 5.10+
- 或使用预生成的 vmlinux.h

### Q3: 加载失败: "Operation not permitted"

**A**: eBPF 需要 root 权限:
```bash
sudo make load  # 注意 sudo
```

### Q4: bpf_printk() 没有输出?

**A**: 检查 debugfs 是否挂载:
```bash
sudo mount -t debugfs none /sys/kernel/debug
sudo cat /sys/kernel/debug/tracing/trace_pipe
```

### Q5: 为什么用 loopback 接口?

**A**: Loopback (`lo`) 的优点:
- ✅ 总是存在,无需物理网卡
- ✅ 流量可控,不影响外部网络
- ✅ 可以用 `ping 127.0.0.1` 轻松测试

对于生产环境,会使用真实网卡 (如 `eth0`)。

---

## 🎓 进阶练习

### 练习 1: 统计入站和出站流量

修改程序,使用两个计数器分别统计 ingress 和 egress:

```c
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 2);  // 索引 0 = ingress, 索引 1 = egress
    __type(key, __u32);
    __type(value, __u64);
} packet_counter SEC(".maps");
```

提示: 可以通过 `skb->ingress_ifindex` 判断方向。

### 练习 2: 按协议统计

扩展为按协议分类统计 (TCP/UDP/ICMP/其他):

```c
enum {
    PROTO_TCP = 0,
    PROTO_UDP = 1,
    PROTO_ICMP = 2,
    PROTO_OTHER = 3,
    PROTO_MAX = 4
};

struct {
    __uint(max_entries, PROTO_MAX);
    // ...
} proto_counter SEC(".maps");
```

提示: 需要解析以太网和 IP 头来获取协议类型。

### 练习 3: 字节计数

除了数据包数量,还统计总字节数:

```c
struct counter_value {
    __u64 packets;
    __u64 bytes;
};
```

提示: `skb->len` 包含数据包长度。

---

## 📚 相关资源

### eBPF 基础
- [eBPF 官方文档](https://ebpf.io/what-is-ebpf/)
- [BPF 验证器指南](https://docs.kernel.org/bpf/verifier.html)

### TC Hook
- [Linux TC 手册](https://man7.org/linux/man-pages/man8/tc-bpf.8.html)
- [TC eBPF 示例](https://github.com/xdp-project/xdp-tutorial)

### 工具使用
- [bpftool 文档](https://github.com/libbpf/bpftool)
- [libbpf API](https://github.com/libbpf/libbpf)

---

## ✅ 知识检查点

完成这个 Demo 后,你应该能回答:

- [ ] TC Hook 和 XDP Hook 的区别是什么?
- [ ] `BPF_MAP_TYPE_ARRAY` 的查找时间复杂度是多少?
- [ ] 为什么需要 `__sync_fetch_and_add()` 而不是普通的 `+=`?
- [ ] `TC_ACT_OK` 和 `TC_ACT_SHOT` 分别表示什么?
- [ ] `bpf_printk()` 的输出去哪里了?
- [ ] 如何用 `bpftool` 查看 eBPF 程序和 Map?

---

## 🚀 下一步

恭喜完成 Demo 1! 🎉

现在你已经理解了 eBPF 的基本工作流程。下一个 Demo 会更进一步,学习如何**解析数据包内容**并提取 5-tuple 信息。

➡️ [继续学习 Demo 2: 5-Tuple Extractor](../02-5tuple-extractor/README.md)

---

[⬅️ 返回主目录](../README.md) | [➡️ 下一个: Demo 2](../02-5tuple-extractor/README.md)
