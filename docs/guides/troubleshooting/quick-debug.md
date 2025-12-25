# eBPF 微隔离系统 - 快速调试指南

> 本文档提供快速启动、调试和问题排查的实用指南

## 目录

1. [快速启动](#1-快速启动)
2. [调试工具使用](#2-调试工具使用)
3. [常见问题排查](#3-常见问题排查)
4. [性能分析](#4-性能分析)
5. [日志分析技巧](#5-日志分析技巧)

---

## 1. 快速启动

### 1.1 编译项目

```bash
# 编译 eBPF 程序（需要 clang/llvm）
cd src/bpf
make

# 编译 Agent
cd ../agent
go build -o agent cmd/main.go
```

### 1.2 启动 Agent

```bash
# 以 root 权限运行（eBPF 需要 CAP_BPF 或 root）
sudo ./agent --iface eth0 --api-port 8080

# 查看启动日志
# - 应该看到 "eBPF program loaded successfully"
# - 应该看到 "API server started on :8080"
```

### 1.3 验证系统运行

```bash
# 1. 检查 eBPF 程序是否加载
sudo bpftool prog list | grep tc_microsegment

# 2. 检查 TC filter 是否附加
sudo tc filter show dev eth0 ingress

# 3. 检查 API 是否响应
curl http://localhost:8080/api/v1/health

# 4. 查看统计信息
curl http://localhost:8080/api/v1/statistics
```

### 1.4 添加测试策略

```bash
# 允许 10.0.0.1 访问 10.0.0.2 的 80 端口
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "source_cidr": "10.0.0.1/32",
    "dest_cidr": "10.0.0.2/32",
    "dest_port": 80,
    "protocol": "TCP",
    "action": "allow"
  }'

# 查看策略列表
curl http://localhost:8080/api/v1/policies | jq
```

---

## 2. 调试工具使用

### 2.1 bpftool - eBPF 调试利器

#### 2.1.1 查看加载的 eBPF 程序

```bash
# 列出所有 eBPF 程序
sudo bpftool prog list

# 查看 tc_microsegment 程序详情
sudo bpftool prog show name tc_microsegment_filter

# 输出示例：
# 123: sched_cls  name tc_microsegment_filter  tag abc123...
#     loaded_at 2024-01-01T10:00:00+0000  uid 0
#     xlated 2048B  jited 1234B  memlock 4096B  map_ids 10,11,12,13,14
```

#### 2.1.2 查看 eBPF Maps

```bash
# 列出所有 maps
sudo bpftool map list

# 查看 policy_map 的内容（精确匹配策略）
sudo bpftool map dump name policy_map

# 输出示例：
# key: 0a 00 00 01 0a 00 00 02 00 50 06 00  # 10.0.0.1 -> 10.0.0.2:80/TCP
# value: 01 00 00 00                        # action=1 (allow)

# 查看 session_map（会话缓存）
sudo bpftool map dump name session_map | head -20

# 查看统计信息（stats_map）
sudo bpftool map dump name stats_map
```

#### 2.1.3 实时监控 Map 变化

```bash
# 实时监控 session_map 的插入/删除
sudo bpftool map event_pipe name session_map

# 在另一个终端触发流量，观察会话创建
```

#### 2.1.4 导出 eBPF 程序字节码

```bash
# 导出 BPF 字节码（用于离线分析）
sudo bpftool prog dump xlated name tc_microsegment_filter

# 导出 JIT 编译后的机器码
sudo bpftool prog dump jited name tc_microsegment_filter
```

### 2.2 tc (Traffic Control) 工具

#### 2.2.1 查看 TC filter

```bash
# 查看 ingress 方向的 TC filter
sudo tc filter show dev eth0 ingress

# 输出示例：
# filter protocol all pref 1 bpf chain 0
# filter protocol all pref 1 bpf chain 0 handle 0x1 tc_microsegment.bpf.o:[tc] direct-action

# 查看详细信息（包括 eBPF 程序 ID）
sudo tc filter show dev eth0 ingress -s
```

#### 2.2.2 手动删除 TC filter（调试用）

```bash
# 删除 ingress 上的所有 filter
sudo tc filter del dev eth0 ingress

# 删除 qdisc（会自动删除所有 filter）
sudo tc qdisc del dev eth0 clsact
```

### 2.3 curl - API 调试

#### 2.3.1 查看策略

```bash
# 获取所有策略
curl http://localhost:8080/api/v1/policies | jq

# 获取特定策略
curl http://localhost:8080/api/v1/policies/policy-uuid | jq
```

#### 2.3.2 添加策略

```bash
# 添加精确匹配策略
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "source_cidr": "192.168.1.100/32",
    "dest_cidr": "10.0.0.50/32",
    "dest_port": 443,
    "protocol": "TCP",
    "action": "allow"
  }' | jq

# 添加通配符策略（支持 CIDR 段）
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "source_cidr": "192.168.1.0/24",
    "dest_cidr": "10.0.0.0/24",
    "dest_port": 0,
    "protocol": "TCP",
    "action": "deny"
  }' | jq
```

#### 2.3.3 删除策略

```bash
# 删除指定策略
curl -X DELETE http://localhost:8080/api/v1/policies/policy-uuid
```

#### 2.3.4 查看统计信息

```bash
# 获取全局统计
curl http://localhost:8080/api/v1/statistics | jq

# 输出示例：
# {
#   "total_packets": 123456,
#   "total_bytes": 987654321,
#   "allowed_packets": 100000,
#   "denied_packets": 23456,
#   "session_hits": 120000,
#   "session_misses": 3456
# }
```

### 2.4 日志调试

#### 2.4.1 查看 Agent 日志

```bash
# 如果使用 systemd 运行
sudo journalctl -u microsegment-agent -f

# 如果直接运行，重定向到文件
sudo ./agent --iface eth0 2>&1 | tee agent.log

# 过滤特定日志（如策略相关）
tail -f agent.log | grep "policy"
```

#### 2.4.2 eBPF 程序日志（bpf_printk）

```bash
# eBPF 程序中使用 bpf_printk() 输出的日志
sudo cat /sys/kernel/debug/tracing/trace_pipe

# 示例输出：
# tc_microsegment_filter: packet from 10.0.0.1:1234 to 10.0.0.2:80
# tc_microsegment_filter: session hit, action=1
```

**注意**：`bpf_printk()` 性能开销大，仅用于开发调试，生产环境应移除。

---

## 3. 常见问题排查

### 3.1 eBPF 程序加载失败

#### 症状
```
Error: failed to load eBPF objects: field tc_microsegment_filter: program tc_microsegment_filter: load program: invalid argument
```

#### 排查步骤

1. **检查内核版本**
```bash
uname -r
# 需要 >= 5.10（推荐 5.15+）
```

2. **检查 eBPF 日志**
```bash
# 使用 cilium/ebpf 库的详细错误信息
# Agent 启动时应该输出 verifier 错误

# 或手动加载并查看 verifier 日志
sudo bpftool prog load tc_microsegment.bpf.o /sys/fs/bpf/test type tc
```

3. **常见原因**
   - **循环次数超限**：eBPF 限制循环 < 1024 次（WILDCARD_POLICY_MAX_ENTRIES）
   - **栈空间超限**：eBPF 栈 < 512 字节，检查局部变量
   - **不安全的内存访问**：所有指针访问需要边界检查
   - **helper 函数使用错误**：如在不允许的上下文中使用 `bpf_get_current_pid_tgid`

### 3.2 TC filter 附加失败

#### 症状
```
Error: failed to attach TC program: file exists
```

#### 解决方法

```bash
# 1. 检查是否已有 filter 附加
sudo tc filter show dev eth0 ingress

# 2. 删除旧的 filter
sudo tc filter del dev eth0 ingress

# 3. 如果仍失败，删除 qdisc 重建
sudo tc qdisc del dev eth0 clsact
sudo tc qdisc add dev eth0 clsact

# 4. 重新启动 Agent
sudo ./agent --iface eth0
```

### 3.3 策略不生效

#### 症状
- 添加了策略，但流量仍被 deny/allow

#### 排查步骤

1. **确认策略是否写入 eBPF map**
```bash
# 检查 policy_map（精确匹配）
sudo bpftool map dump name policy_map

# 检查 wildcard_policy_map（通配符）
sudo bpftool map dump name wildcard_policy_map
```

2. **确认五元组是否匹配**
```bash
# 例如：源 IP=10.0.0.1, 目的 IP=10.0.0.2, 目的端口=80, 协议=TCP
# policy_map key 格式：src_ip(4B) + dst_ip(4B) + dst_port(2B) + protocol(1B) + padding(1B)
# 应为：0a000001 0a000002 0050 06 00

# 使用 bpftool 查看二进制内容
sudo bpftool map lookup name policy_map key hex 0a 00 00 01 0a 00 00 02 00 50 06 00
```

3. **检查会话缓存是否干扰**
```bash
# 如果旧会话缓存了相反的决策，新策略不会立即生效
# 解决方法：清空 session_map 或等待 LRU 淘汰

# 手动清空 session_map（会导致短暂性能下降）
sudo bpftool map delete name session_map key <5-tuple>
# 或重启 Agent（会重新加载所有 map）
```

4. **查看统计信息确认流量是否经过**
```bash
curl http://localhost:8080/api/v1/statistics | jq

# 如果 total_packets 不增长，说明 TC 未捕获流量
# 如果 denied_packets 不符合预期，检查策略匹配逻辑
```

### 3.4 性能下降

#### 症状
- CPU 使用率高
- 丢包率增加

#### 排查步骤

1. **检查通配符策略数量**
```bash
# 通配符策略使用线性查找，数量应 < 100
curl http://localhost:8080/api/v1/policies | jq '[.[] | select(.source_cidr | endswith("/32") | not)] | length'
```

2. **检查会话命中率**
```bash
# session_hits 应该 >> session_misses
curl http://localhost:8080/api/v1/statistics | jq '.session_hits, .session_misses'

# 命中率 < 95% 说明会话淘汰过快，考虑增加 SESSION_MAP_MAX_ENTRIES
```

3. **使用 perf 分析 eBPF 程序热点**
```bash
# 采样 eBPF 程序执行
sudo perf record -e bpf:bpf_prog_run -a -g -- sleep 10
sudo perf report

# 查看哪个 helper 函数耗时最多
```

### 3.5 API 请求失败

#### 症状
```
curl: (7) Failed to connect to localhost port 8080
```

#### 排查步骤

1. **检查 Agent 是否运行**
```bash
ps aux | grep agent
```

2. **检查端口是否监听**
```bash
sudo netstat -tlnp | grep 8080
# 或
sudo ss -tlnp | grep 8080
```

3. **检查防火墙规则**
```bash
sudo iptables -L -n | grep 8080
```

4. **查看 Agent 启动日志**
```bash
# 应该看到 "API server started on :8080"
```

---

## 4. 性能分析

### 4.1 统计信息解读

```bash
curl http://localhost:8080/api/v1/statistics | jq
```

**关键指标**：

| 指标 | 含义 | 理想值 |
|------|------|--------|
| `total_packets` | 总处理包数 | 持续增长 |
| `allowed_packets` | 允许的包数 | 根据策略 |
| `denied_packets` | 拒绝的包数 | 根据策略 |
| `session_hits` | 会话缓存命中 | > 95% |
| `session_misses` | 会话缓存未命中 | < 5% |
| `policy_lookups` | 策略查找次数 | = session_misses |

**计算会话命中率**：
```bash
curl -s http://localhost:8080/api/v1/statistics | jq '(.session_hits / (.session_hits + .session_misses)) * 100'
```

### 4.2 eBPF 性能分析

#### 4.2.1 使用 bpftool 查看程序运行时间

```bash
# 查看程序统计（需要内核 5.15+）
sudo bpftool prog show name tc_microsegment_filter --json | jq '.run_time_ns, .run_cnt'

# 计算平均执行时间
# avg_time = run_time_ns / run_cnt
```

#### 4.2.2 使用 perf 分析

```bash
# 记录 10 秒的 eBPF 事件
sudo perf record -e bpf:bpf_prog_run -a -- sleep 10

# 查看报告
sudo perf report

# 导出火焰图（需要安装 flamegraph 工具）
sudo perf script | stackcollapse-perf.pl | flamegraph.pl > bpf_flamegraph.svg
```

### 4.3 识别性能瓶颈

#### 4.3.1 通配符策略过多

**症状**：`session_misses` 延迟高

**解决**：
- 将常用的通配符策略转换为精确匹配策略
- 减少 `WILDCARD_POLICY_MAX_ENTRIES` 以降低循环次数
- 优化通配符策略顺序（将高频匹配的放前面）

#### 4.3.2 会话淘汰过快

**症状**：`session_hits / total_packets < 0.95`

**解决**：
```c
// 在 src/bpf/tc_microsegment.bpf.c 中增加会话表大小
#define SESSION_MAP_MAX_ENTRIES 1000000  // 从 100000 增加到 100 万
```

#### 4.3.3 Map 查找冲突

**症状**：`policy_map` 查找延迟高（HASH 冲突）

**解决**：
```c
// 增加 policy_map 的桶数量
#define POLICY_MAP_MAX_ENTRIES 100000  // 从 10000 增加
```

---

## 5. 日志分析技巧

### 5.1 Agent 日志关键字

| 关键字 | 含义 | 示例 |
|--------|------|------|
| `eBPF program loaded` | 程序加载成功 | `INFO: eBPF program loaded successfully` |
| `failed to load eBPF` | 加载失败 | `ERROR: failed to load eBPF objects: ...` |
| `TC program attached` | TC 附加成功 | `INFO: TC program attached to eth0` |
| `Policy added` | 策略添加 | `INFO: Policy added: 10.0.0.1/32 -> 10.0.0.2/32:80` |
| `Policy deleted` | 策略删除 | `INFO: Policy deleted: uuid-xxx` |
| `API server started` | API 启动 | `INFO: API server started on :8080` |

### 5.2 过滤有用的日志

```bash
# 只看错误
tail -f agent.log | grep ERROR

# 只看策略变更
tail -f agent.log | grep -E "Policy (added|deleted)"

# 只看 eBPF 相关
tail -f agent.log | grep -i ebpf

# 只看性能统计
tail -f agent.log | grep -i statistics
```

### 5.3 eBPF trace_pipe 日志

```bash
# 实时查看 eBPF 程序输出（如果启用了 bpf_printk）
sudo cat /sys/kernel/debug/tracing/trace_pipe | grep tc_microsegment

# 示例输出：
# <...>-12345 [001] .... 123.456: bpf_trace_printk: packet: 10.0.0.1:1234 -> 10.0.0.2:80
# <...>-12345 [001] .... 123.457: bpf_trace_printk: session hit, action=1
```

**过滤特定 IP**：
```bash
sudo cat /sys/kernel/debug/tracing/trace_pipe | grep "10.0.0.1"
```

### 5.4 分析拒绝原因

```bash
# 在 eBPF 代码中添加调试输出（开发环境）
// src/bpf/tc_microsegment.bpf.c

if (action == ACTION_DENY) {
    bpf_printk("DENY: %pI4:%d -> %pI4:%d",
               &flow_key.src_ip, bpf_ntohs(flow_key.src_port),
               &flow_key.dst_ip, bpf_ntohs(flow_key.dst_port));
}
```

然后：
```bash
# 触发被拒绝的流量
curl http://10.0.0.2:80

# 查看日志
sudo cat /sys/kernel/debug/tracing/trace_pipe | grep DENY
```

---

## 6. 调试工作流示例

### 场景：新增策略不生效

```bash
# 1. 添加策略
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "source_cidr": "192.168.1.100/32",
    "dest_cidr": "10.0.0.50/32",
    "dest_port": 443,
    "protocol": "TCP",
    "action": "allow"
  }'

# 2. 确认策略已添加
curl http://localhost:8080/api/v1/policies | jq '.[] | select(.source_cidr == "192.168.1.100/32")'

# 3. 检查 eBPF map
sudo bpftool map dump name policy_map | grep -A1 "c0 a8 01 64"  # 192.168.1.100

# 4. 清除可能的旧会话
# (可选) 如果有旧会话缓存了 deny 决策

# 5. 测试流量
# 从 192.168.1.100 访问 10.0.0.50:443

# 6. 检查统计
curl http://localhost:8080/api/v1/statistics | jq '.allowed_packets, .denied_packets'

# 7. 查看 eBPF 日志（如果启用）
sudo cat /sys/kernel/debug/tracing/trace_pipe | grep "192.168.1.100"
```

---

## 7. 快速参考

### 7.1 常用命令速查

```bash
# 查看 eBPF 程序
sudo bpftool prog list | grep tc_microsegment

# 查看 TC filter
sudo tc filter show dev eth0 ingress

# 查看策略 map
sudo bpftool map dump name policy_map

# 查看会话 map
sudo bpftool map dump name session_map

# 查看统计
curl localhost:8080/api/v1/statistics | jq

# 查看所有策略
curl localhost:8080/api/v1/policies | jq

# 实时日志
sudo cat /sys/kernel/debug/tracing/trace_pipe | grep tc_microsegment

# Agent 日志
sudo journalctl -u microsegment-agent -f
```

### 7.2 故障排查清单

- [ ] eBPF 程序是否加载？(`bpftool prog list`)
- [ ] TC filter 是否附加？(`tc filter show`)
- [ ] 策略是否写入 map？(`bpftool map dump`)
- [ ] API 是否响应？(`curl /health`)
- [ ] 统计是否增长？(`curl /statistics`)
- [ ] 会话命中率是否 > 95%？
- [ ] 是否有旧会话缓存干扰？
- [ ] 内核版本是否 >= 5.10？
- [ ] 是否以 root 权限运行？

---

## 8. 进阶调试

### 8.1 使用 strace 跟踪系统调用

```bash
# 跟踪 Agent 的 BPF 系统调用
sudo strace -e bpf,ioctl ./agent --iface eth0 2>&1 | tee strace.log

# 过滤 BPF map 操作
grep "bpf(BPF_MAP" strace.log
```

### 8.2 使用 bpftrace 编写自定义探针

```bash
# 统计每种 eBPF helper 调用次数
sudo bpftrace -e 'kprobe:__bpf_prog_run { @[comm] = count(); }'

# 监控 map 更新
sudo bpftrace -e 'tracepoint:bpf:bpf_map_update_elem { printf("%s updated map\n", comm); }'
```

### 8.3 内核调试选项

编译内核时启用：
```
CONFIG_BPF_JIT_ALWAYS_ON=y
CONFIG_BPF_SYSCALL=y
CONFIG_DEBUG_INFO_BTF=y  # 用于 CO-RE
CONFIG_DEBUG_INFO=y       # 用于调试符号
```

---

## 总结

本文档提供了快速调试 eBPF 微隔离系统的实用指南，涵盖：

1. **快速启动**：编译、运行、验证
2. **调试工具**：bpftool、tc、curl、日志
3. **常见问题**：加载失败、策略不生效、性能下降
4. **性能分析**：统计指标、瓶颈识别
5. **日志分析**：关键字、过滤技巧

**配合使用**：建议结合 [code-architecture-guide.md](code-architecture-guide.md) 深入理解代码逻辑。

**反馈**：遇到文档未涵盖的问题？欢迎提交 issue！
