# 第3周：用户态控制程序

**[⬅️ 第2周](./week2-basic-framework.md)** | **[📚 目录](./README.md)** | **[➡️ 第4周](./week4-advanced-features.md)**

---

## 📋 学习进度跟踪表

> 💡 **使用说明**：每天学习后，更新下表记录你的进度、遇到的问题和解决方案

| 日期 | 学习内容 | 状态 | 实际耗时 | 遇到的问题 | 解决方案/笔记 |
|------|----------|------|----------|-----------|--------------|
| Day 1-2 | libbpf skeleton集成 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 3 | CLI工具开发 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 4 | JSON配置文件支持 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 5 | 策略CRUD + 热更新 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |
| Day 6-7 | 集成测试 + 周总结 | ⬜ 未开始<br>🔄 进行中<br>✅ 已完成 | ___h | | |

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

2. 对比传统方式 vs skeleton方式:
   - 传统: 手动加载.o文件, 手动获取Map FD
   - Skeleton: 自动生成结构体, 类型安全
   - 时间: 30分钟

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

---

**[⬅️ 第2周](./week2-basic-framework.md)** | **[📚 目录](./README.md)** | **[➡️ 第4周](./week4-advanced-features.md)**
