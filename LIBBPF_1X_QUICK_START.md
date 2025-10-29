# libbpf 1.x 快速入门指南

> 本指南帮助你快速上手 libbpf 1.x 的现代 TC API

## 🚀 快速开始

### 1. 安装 libbpf 1.x

```bash
# Ubuntu 22.04+
sudo apt-get update
sudo apt-get install -y libbpf-dev

# 验证版本（应该 >= 1.0）
pkg-config --modversion libbpf
```

### 2. 基本代码模板

```c
#include <stdio.h>
#include <unistd.h>
#include <signal.h>
#include <net/if.h>
#include <bpf/libbpf.h>
#include <bpf/bpf.h>
#include "my_prog.skel.h"

static volatile bool exiting = false;

static void sig_handler(int sig) {
    exiting = true;
}

int main() {
    struct my_prog_bpf *skel;
    int err;
    const char *ifname = "eth0";

    // 1. 加载 eBPF 程序
    skel = my_prog_bpf__open_and_load();
    if (!skel) {
        fprintf(stderr, "Failed to load BPF skeleton\n");
        return 1;
    }

    // 2. 附加到 TC (使用 libbpf 1.x API)
    LIBBPF_OPTS(bpf_tc_hook, hook,
        .ifindex = if_nametoindex(ifname),
        .attach_point = BPF_TC_INGRESS);
    
    LIBBPF_OPTS(bpf_tc_opts, opts,
        .handle = 1,
        .priority = 1,
        .prog_fd = bpf_program__fd(skel->progs.my_tc_prog));
    
    // 创建 hook
    err = bpf_tc_hook_create(&hook);
    if (err && err != -EEXIST) {
        fprintf(stderr, "Failed to create hook: %s\n", strerror(-err));
        goto cleanup;
    }
    
    // 附加程序
    err = bpf_tc_attach(&hook, &opts);
    if (err) {
        fprintf(stderr, "Failed to attach: %s\n", strerror(-err));
        goto cleanup;
    }
    
    printf("✓ Program attached to %s\n", ifname);

    // 3. 运行
    signal(SIGINT, sig_handler);
    while (!exiting) {
        sleep(1);
    }

cleanup:
    // 4. 清理
    bpf_tc_detach(&hook, &opts);
    bpf_tc_hook_destroy(&hook);
    my_prog_bpf__destroy(skel);
    return 0;
}
```

### 3. Makefile

```makefile
CLANG ?= clang
BPFTOOL ?= bpftool
ARCH := $(shell uname -m | sed 's/x86_64/x86/')

# 编译 eBPF 程序
%.bpf.o: %.bpf.c
	$(CLANG) -g -O2 -target bpf -D__TARGET_ARCH_$(ARCH) \
		-c $< -o $@

# 生成 skeleton
%.skel.h: %.bpf.o
	$(BPFTOOL) gen skeleton $< > $@

# 编译用户态程序
my_prog: my_prog.skel.h main.c
	gcc -Wall -o $@ main.c -lbpf -lelf -lz

clean:
	rm -f *.o *.skel.h my_prog
```

## 📚 核心 API 参考

### TC Hook 操作

```c
// 创建 TC hook
int bpf_tc_hook_create(struct bpf_tc_hook *hook);

// 销毁 TC hook
int bpf_tc_hook_destroy(struct bpf_tc_hook *hook);

// 附加程序
int bpf_tc_attach(const struct bpf_tc_hook *hook, 
                  struct bpf_tc_opts *opts);

// 分离程序
int bpf_tc_detach(const struct bpf_tc_hook *hook,
                  const struct bpf_tc_opts *opts);
```

### 配置选项

```c
// Hook 配置
LIBBPF_OPTS(bpf_tc_hook, hook,
    .ifindex = if_nametoindex("eth0"),  // 网卡索引
    .attach_point = BPF_TC_INGRESS);    // 或 BPF_TC_EGRESS

// 程序选项
LIBBPF_OPTS(bpf_tc_opts, opts,
    .handle = 1,         // TC handle
    .priority = 1,       // 优先级（数字越小优先级越高）
    .prog_fd = prog_fd); // 程序文件描述符
```

## 🔍 常见问题

### Q1: 如何处理错误？

```c
int err = bpf_tc_attach(&hook, &opts);
if (err) {
    // libbpf 返回负的 errno
    fprintf(stderr, "Error: %s\n", strerror(-err));
}
```

### Q2: 如何附加到多个网卡？

```c
const char *ifaces[] = {"eth0", "eth1", "eth2"};
for (int i = 0; i < 3; i++) {
    LIBBPF_OPTS(bpf_tc_hook, hook,
        .ifindex = if_nametoindex(ifaces[i]),
        .attach_point = BPF_TC_INGRESS);
    
    // ... 创建和附加 ...
}
```

### Q3: 如何同时附加到 ingress 和 egress？

```c
// Ingress
LIBBPF_OPTS(bpf_tc_hook, hook_in,
    .ifindex = ifindex,
    .attach_point = BPF_TC_INGRESS);
bpf_tc_hook_create(&hook_in);
bpf_tc_attach(&hook_in, &opts);

// Egress
LIBBPF_OPTS(bpf_tc_hook, hook_eg,
    .ifindex = ifindex,
    .attach_point = BPF_TC_EGRESS);
bpf_tc_hook_create(&hook_eg);
bpf_tc_attach(&hook_eg, &opts);
```

### Q4: 如何设置优先级？

```c
// 优先级数字越小，执行越早
LIBBPF_OPTS(bpf_tc_opts, opts_high_prio,
    .priority = 1);  // 高优先级

LIBBPF_OPTS(bpf_tc_opts, opts_low_prio,
    .priority = 100); // 低优先级
```

## ⚡ 性能优化建议

1. **使用 Per-CPU Map** 避免锁竞争
2. **使用 LRU Map** 自动管理内存
3. **避免在 eBPF 中使用循环** 或使用有界循环
4. **使用内联函数** 减少函数调用开销
5. **批量操作** 而非单个元素操作

## 🔗 相关资源

- [libbpf 官方文档](https://github.com/libbpf/libbpf)
- [libbpf API 参考](https://libbpf.readthedocs.io/)
- [libbpf-bootstrap 示例](https://github.com/libbpf/libbpf-bootstrap)
- [eBPF 文档](https://ebpf.io/what-is-ebpf/)

## 📝 版本兼容性

| libbpf 版本 | TC API 支持 | 推荐使用 |
|------------|------------|---------|
| < 0.6      | ❌ 无      | ❌ 不推荐 |
| 0.6 - 0.8  | ⚠️ 实验性  | ⚠️ 谨慎使用 |
| >= 1.0     | ✅ 完整支持 | ✅ 强烈推荐 |

---

**最后更新**: 2025-10-29  
**适用版本**: libbpf >= 1.0

