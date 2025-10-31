# NeuVector dp 组件编译指南

> 本文档详细说明如何编译 NeuVector 的 dp（Data Plane）组件

---

## 目录

- [组件简介](#组件简介)
- [环境要求](#环境要求)
- [安装编译依赖](#安装编译依赖)
- [编译步骤](#编译步骤)
- [验证编译](#验证编译)
- [依赖库详解](#依赖库详解)
- [常见问题](#常见问题)

---

## 组件简介

NeuVector dp（Data Plane）是一个高性能的网络数据包处理引擎，主要功能包括：

- **深度包检测 (DPI)**：支持多种应用层协议识别
  - HTTP, HTTPS, gRPC
  - MySQL, PostgreSQL, MongoDB, Redis
  - Kafka, Cassandra, ZooKeeper
  - SSH, DNS, DHCP, NTP
  - 等 20+ 种协议

- **策略执行**：基于策略进行流量过滤和控制
- **会话跟踪**：TCP/UDP 连接跟踪和状态管理
- **高性能正则匹配**：使用 HyperScan 引擎进行高速模式匹配
- **统计和日志**：提供详细的流量统计和日志记录

### 架构特点

- 基于 **Netfilter Queue** 进行数据包拦截
- 使用 **RCU (Read-Copy-Update)** 实现无锁数据结构
- 采用 **jemalloc** 优化内存分配性能
- 集成 **HyperScan** 高性能正则匹配引擎

---

## 环境要求

### 操作系统
- Ubuntu 22.04+ (推荐)
- CentOS 8+ / RHEL 8+
- Debian 11+

### 硬件要求
- CPU: x86_64 架构
- 内存: 至少 2GB（推荐 4GB+）
- 磁盘: 至少 1GB 可用空间

### 内核版本
- Linux Kernel ≥ 4.14（Netfilter Queue 支持）
- 建议 ≥ 5.10（更好的性能和稳定性）

### 编译工具链
- GCC ≥ 7.0
- GNU Make ≥ 4.0
- pkg-config

---

## 安装编译依赖

### 一键安装脚本

```bash
#!/bin/bash
# install_dp_deps.sh - 一键安装 dp 编译依赖

set -e

echo "=== 安装 NeuVector dp 编译依赖 ==="

# 更新包管理器
sudo apt-get update

# 基础编译工具
echo "[1/7] 安装基础编译工具..."
sudo apt-get install -y build-essential pkg-config

# RCU (Read-Copy-Update) 库
echo "[2/7] 安装 liburcu (RCU 无锁数据结构)..."
sudo apt-get install -y liburcu-dev

# Netfilter 相关库
echo "[3/7] 安装 Netfilter 库..."
sudo apt-get install -y \
    libnfnetlink-dev \
    libnetfilter-queue-dev

# 数据包捕获库
echo "[4/7] 安装 libpcap (数据包捕获)..."
sudo apt-get install -y libpcap-dev

# 正则表达式库
echo "[5/7] 安装正则表达式库..."
sudo apt-get install -y libpcre2-dev

# JSON 和内存管理库
echo "[6/7] 安装 JSON 和内存管理库..."
sudo apt-get install -y \
    libjansson-dev \
    libjemalloc-dev

# HyperScan 高性能正则匹配引擎
echo "[7/7] 安装 HyperScan..."
sudo apt-get install -y libhyperscan-dev

echo ""
echo "✅ 所有依赖安装完成！"
echo ""
echo "验证安装："
dpkg -l | grep -E "liburcu|libnfnetlink|libnetfilter-queue|libpcap|libpcre2|libjansson|libjemalloc|libhyperscan" | awk '{print $2, $3}'
```

### 分步安装

#### 第一步：基础编译工具

```bash
sudo apt-get update
sudo apt-get install -y build-essential pkg-config
```

#### 第二步：RCU 库

```bash
# RCU (Read-Copy-Update) 库 - 用于无锁数据结构
sudo apt-get install -y liburcu-dev
```

#### 第三步：Netfilter 相关库

```bash
# Netfilter 库 - 用于内核通信和数据包队列处理
sudo apt-get install -y \
    libnfnetlink-dev \
    libnetfilter-queue-dev
```

#### 第四步：其他依赖库

```bash
# 数据包捕获、正则匹配、JSON 解析、内存分配
sudo apt-get install -y \
    libpcap-dev \
    libpcre2-dev \
    libjansson-dev \
    libjemalloc-dev \
    libhyperscan-dev
```

---

## 编译步骤

### 第一步：进入源码目录

```bash
cd /home/work/ebpf-based-microsegment/source-references/neuvector/dp
```

### 第二步：清理旧编译（可选）

```bash
# 如果之前编译过，建议先清理
make clean
```

### 第三步：编译

```bash
# 执行编译
make

# 编译过程会经历以下阶段：
# 1. 编译 utils 工具库
# 2. 编译 dpi/parsers 协议解析器
# 3. 编译 dpi/sig 签名匹配模块
# 4. 编译主程序并链接所有库
```

### 第四步：查看编译结果

```bash
# 查看生成的 dp 二进制文件
ls -lh dp

# 输出示例:
# -rwxr-xr-x 1 root root 1.4M Oct 29 19:07 dp
```

---

## 验证编译

### 检查二进制文件

```bash
# 查看文件类型
file dp

# 输出示例:
# dp: ELF 64-bit LSB executable, x86-64, version 1 (SYSV),
# dynamically linked, interpreter /lib64/ld-linux-x86-64.so.2,
# BuildID[sha1]=..., for GNU/Linux 3.2.0, not stripped
```

### 检查依赖库

```bash
# 查看所有依赖库
ldd dp

# 重点检查以下库是否正确链接
ldd dp | grep -E "liburcu|libnetfilter|libhs|libjansson|libjemalloc|libpcre2"

# 输出示例:
# liburcu.so.8 => /lib/x86_64-linux-gnu/liburcu.so.8 (0x...)
# liburcu-cds.so.8 => /lib/x86_64-linux-gnu/liburcu-cds.so.8 (0x...)
# libnetfilter_queue.so.1 => /lib/x86_64-linux-gnu/libnetfilter_queue.so.1 (0x...)
# libhs.so.5 => /lib/x86_64-linux-gnu/libhs.so.5 (0x...)
# libjansson.so.4 => /lib/x86_64-linux-gnu/libjansson.so.4 (0x...)
# libjemalloc.so.2 => /lib/x86_64-linux-gnu/libjemalloc.so.2 (0x...)
# libpcre2-8.so.0 => /lib/x86_64-linux-gnu/libpcre2-8.so.0 (0x...)
```

### 检查符号表

```bash
# 查看导出的符号（确认关键函数存在）
nm dp | grep -E "dpi_|nfq_|rcu_"

# 应该能看到 DPI、NFQ、RCU 相关的符号
```

---

## 依赖库详解

### 核心依赖库说明

| 库 | 版本要求 | 用途 | 关键功能 |
|---|---------|------|---------|
| **liburcu** | ≥ 0.13 | RCU 无锁数据结构 | 高性能并发读写、会话表管理 |
| **libnfnetlink** | ≥ 1.0 | Netfilter 内核通信 | 内核消息传递、配置管理 |
| **libnetfilter-queue** | ≥ 1.0 | 数据包队列处理 | 拦截数据包、判决处理 |
| **libpcap** | ≥ 1.10 | 数据包捕获 | 网络嗅探、包解析 |
| **libpcre2** | ≥ 10.39 | 正则表达式匹配 | URL 匹配、内容过滤 |
| **libjansson** | ≥ 2.13 | JSON 解析 | 配置文件解析、API 通信 |
| **libjemalloc** | ≥ 5.2 | 内存分配器 | 高性能内存管理、减少碎片 |
| **libhyperscan** | ≥ 5.4 | 高性能正则匹配引擎 | 多模式匹配、DPI 加速 |

### 各库详细说明

#### 1. liburcu (Userspace RCU)

**作用**：
- 提供 RCU（Read-Copy-Update）无锁数据结构
- 支持高并发读操作，写操作不阻塞读
- 用于实现高性能的会话表、策略表

**在 dp 中的应用**：
- 会话跟踪的连接表
- 策略规则的查找表
- 统计数据的并发更新

**官方网站**：https://liburcu.org/

#### 2. libnfnetlink + libnetfilter-queue

**作用**：
- Netfilter 框架的用户态接口
- 拦截内核转发的数据包
- 将数据包判决结果返回内核

**在 dp 中的应用**：
- 注册 NFQUEUE 队列
- 接收数据包进行 DPI 检测
- 返回 ACCEPT/DROP 判决

**数据流**：
```
内核 iptables/nftables 规则
    ↓ (NFQUEUE target)
NFQUEUE (queue 0-65535)
    ↓
libnetfilter_queue (用户态)
    ↓
dp 进行 DPI 检测
    ↓
返回判决 (NF_ACCEPT/NF_DROP)
    ↓
内核继续处理数据包
```

#### 3. libhyperscan

**作用**：
- Intel 开发的高性能正则匹配引擎
- 支持多模式并行匹配（一次扫描匹配多个规则）
- 使用 SIMD 指令加速（SSE/AVX）

**性能优势**：
- 比 PCRE 快 10-100 倍
- 支持数千条规则同时匹配
- 专为 DPI 场景优化

**在 dp 中的应用**：
- WAF 规则匹配（SQL 注入、XSS 检测）
- 协议识别（基于特征签名）
- 恶意流量检测

**官方网站**：https://www.hyperscan.io/

#### 4. libjemalloc

**作用**：
- Facebook 开发的高性能内存分配器
- 减少内存碎片
- 提供详细的内存统计

**性能优势**：
- 比 glibc malloc 快 2-3 倍
- 多线程场景下扩展性好
- 内存碎片更少

---

## 常见问题

### 问题 1：找不到 libhs

**错误信息**：
```
/usr/bin/ld: cannot find -lhs
collect2: error: ld returned 1 exit status
```

**原因**：未安装 HyperScan 库

**解决方法**：
```bash
sudo apt-get install -y libhyperscan-dev
```

---

### 问题 2：内核头文件缺失

**错误信息**：
```
fatal error: linux/netfilter.h: No such file or directory
```

**原因**：缺少内核头文件

**解决方法**：
```bash
sudo apt-get install -y linux-headers-$(uname -r)
```

---

### 问题 3：pkg-config 找不到库

**错误信息**：
```
Package libnfnetlink was not found in the pkg-config search path
```

**原因**：pkg-config 路径配置问题

**解决方法**：
```bash
# 检查 pkg-config 是否能找到库
pkg-config --cflags libnfnetlink

# 如果失败，手动设置路径
export PKG_CONFIG_PATH=/usr/lib/x86_64-linux-gnu/pkgconfig:$PKG_CONFIG_PATH

# 或者永久设置（添加到 ~/.bashrc）
echo 'export PKG_CONFIG_PATH=/usr/lib/x86_64-linux-gnu/pkgconfig:$PKG_CONFIG_PATH' >> ~/.bashrc
source ~/.bashrc
```

---

### 问题 4：编译警告变成错误

**错误信息**：
```
error: unused variable 'xxx' [-Werror=unused-variable]
```

**原因**：Makefile 中使用了 `-Werror` 标志（警告视为错误）

**解决方法 1**（临时）：
```bash
# 禁用优化模式（允许警告）
make DISABLE_OPT=1
```

**解决方法 2**（修改 Makefile）：
```bash
# 编辑 Makefile.rule，移除 -Werror
sed -i 's/-Werror//g' Makefile.rule
make
```

---

### 问题 5：链接错误 - 符号未定义

**错误信息**：
```
undefined reference to `hs_compile'
```

**原因**：库的链接顺序不正确或缺少库

**解决方法**：
```bash
# 检查是否安装了对应的开发库（-dev）
dpkg -l | grep hyperscan

# 如果只有运行时库，需要安装开发库
sudo apt-get install -y libhyperscan-dev
```

---

### 问题 6：架构不兼容

**错误信息**：
```
/usr/lib/gcc/.../ld: skipping incompatible /usr/lib/libxxx.so
```

**原因**：32位库和64位编译器不兼容

**解决方法**：
```bash
# 确保安装的是 amd64 架构的库
sudo apt-get install -y libhyperscan-dev:amd64
```

---

## 编译选项说明

### Makefile 变量

```bash
# 禁用优化（用于调试）
make DISABLE_OPT=1

# 指定编译器
make CC=gcc-10

# 详细输出
make V=1

# 并行编译（加速）
make -j$(nproc)

# 仅编译特定子模块
make -C utils
make -C dpi
```

---

## 编译输出文件

编译成功后，会生成以下文件：

```
dp/
├── dp                    # 主可执行文件 (1.4M)
├── .objs/               # 目标文件目录
│   ├── ctrl.o
│   ├── main.o
│   ├── nfq.o
│   └── ...
├── .deps/               # 依赖文件目录
│   ├── ctrl.d
│   └── ...
├── utils/.objs/         # utils 模块目标文件
├── dpi/.objs/           # DPI 模块目标文件
└── ...
```

---

## 下一步

编译完成后，你可以：

1. **运行 dp 组件**（需要 root 权限和 iptables 配置）
   ```bash
   # 配置 iptables NFQUEUE 规则
   sudo iptables -A FORWARD -j NFQUEUE --queue-num 0

   # 运行 dp
   sudo ./dp
   ```

2. **分析源代码**
   - 阅读 `main.c` - 程序入口和初始化
   - 阅读 `nfq.c` - Netfilter Queue 处理逻辑
   - 阅读 `dpi/` - DPI 引擎实现

3. **参考 dp 设计自己的 eBPF 微隔离系统**
   - 学习 DPI 实现模式
   - 参考会话跟踪机制
   - 借鉴策略匹配算法

4. **开始 eBPF 学习**
   - 查看 [docs/weekly-guide/week1-environment-and-basics.md](weekly-guide/week1-environment-and-basics.md)
   - 学习 eBPF + TC 实现数据包过滤

---

## 参考资源

### 官方文档
- NeuVector GitHub: https://github.com/neuvector/neuvector
- Netfilter Queue: https://netfilter.org/projects/libnetfilter_queue/
- HyperScan: https://www.hyperscan.io/

### 相关文档
- [ZFW 架构分析](zfw-architecture-analysis.md) - 另一个基于 eBPF 的防火墙
- [eBPF TC 实施指南](../design-docs/implementation/ebpf-tc-implementation.md) - 6周学习计划
- [OpenSpec 工作流程](OpenSpec-Workflow-Guide.md) - 项目规格管理

---

**最后更新**：2025-10-29
**维护者**：项目团队
