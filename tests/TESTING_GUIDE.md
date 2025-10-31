# 测试指南

本文档提供详细的测试步骤，帮助验证 eBPF 微隔离系统的功能。

## 环境要求

- **操作系统**: Linux (内核 5.10+)
- **权限**: root 权限（eBPF 程序需要 CAP_BPF 或 root）
- **依赖**:
  - Go 1.23+
  - Clang/LLVM 14+
  - Linux 内核头文件
  - bpftool（可选，用于调试）
  - jq（可选，用于美化 JSON 输出）

## 快速测试（推荐）

### 步骤 1: 启动代理

在**终端 1**中运行：

```bash
cd /home/work/ebpf-based-microsegment
sudo ./bin/microsegment-agent --interface lo --log-level info
```

你应该看到类似输出：

```
INFO[2025-10-31 13:56:00] Starting microsegmentation agent on interface lo
INFO[2025-10-31 13:56:00] ✓ Data plane initialized
INFO[2025-10-31 13:56:00] ✓ Policy manager initialized
INFO[2025-10-31 13:56:00] ✓ API server started on http://127.0.0.1:8080
INFO[2025-10-31 13:56:00] ✓ Agent running. Press Ctrl+C to exit
```

### 步骤 2: 运行自动化测试

在**终端 2**中运行：

```bash
cd /home/work/ebpf-based-microsegment
./test_api.sh
```

测试脚本将自动执行 10 个测试场景：

1. ✓ 健康检查
2. ✓ 系统状态
3. ✓ 列出策略（初始为空）
4. ✓ 创建策略
5. ✓ 列出策略（有数据）
6. ✓ 查询特定策略
7. ✓ 生成流量
8. ✓ 统计信息
9. ✓ 更新策略
10. ✓ 删除策略

---

## 手动测试（详细）

如果你想手动测试每个端点，可以使用 `curl` 命令：

### 1. 健康检查

```bash
# 简单健康检查
curl http://localhost:8080/api/v1/health

# 详细系统状态
curl http://localhost:8080/api/v1/status | jq
```

**预期输出**:

```json
{
  "status": "healthy",
  "dataplane": "active",
  "uptime_seconds": 42,
  "statistics": {
    "total_packets": 0,
    "allowed_packets": 0,
    "denied_packets": 0,
    "new_sessions": 0,
    "policy_hits": 0,
    "policy_misses": 0
  }
}
```

### 2. 策略管理

#### 创建策略

允许 SSH 流量：

```bash
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "rule_id": 100,
    "src_ip": "0.0.0.0/0",
    "dst_ip": "0.0.0.0/0",
    "dst_port": 22,
    "protocol": "tcp",
    "action": "allow",
    "priority": 100
  }' | jq
```

拒绝 HTTPS 流量：

```bash
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "rule_id": 101,
    "src_ip": "0.0.0.0/0",
    "dst_ip": "127.0.0.1",
    "dst_port": 443,
    "protocol": "tcp",
    "action": "deny",
    "priority": 200
  }' | jq
```

#### 列出所有策略

```bash
curl http://localhost:8080/api/v1/policies | jq
```

#### 查询特定策略

```bash
curl http://localhost:8080/api/v1/policies/100 | jq
```

#### 更新策略

```bash
curl -X PUT http://localhost:8080/api/v1/policies/100 \
  -H "Content-Type: application/json" \
  -d '{
    "rule_id": 100,
    "src_ip": "0.0.0.0/0",
    "dst_ip": "0.0.0.0/0",
    "dst_port": 22,
    "protocol": "tcp",
    "action": "deny",
    "priority": 100
  }' | jq
```

#### 删除策略

```bash
curl -X DELETE http://localhost:8080/api/v1/policies/101
```

### 3. 统计查询

#### 所有统计

```bash
curl http://localhost:8080/api/v1/stats | jq
```

**预期输出**:

```json
{
  "total_packets": 150,
  "allowed_packets": 148,
  "denied_packets": 2,
  "new_sessions": 5,
  "closed_sessions": 0,
  "active_sessions": 5,
  "policy_hits": 10,
  "policy_misses": 140
}
```

#### 数据包统计

```bash
curl http://localhost:8080/api/v1/stats/packets | jq
```

**预期输出**:

```json
{
  "total_packets": 150,
  "allowed_packets": 148,
  "denied_packets": 2,
  "allow_rate": 0.9867,
  "deny_rate": 0.0133
}
```

#### 会话统计

```bash
curl http://localhost:8080/api/v1/stats/sessions | jq
```

#### 策略统计

```bash
curl http://localhost:8080/api/v1/stats/policies | jq
```

**预期输出**:

```json
{
  "policy_hits": 10,
  "policy_misses": 140,
  "hit_rate": 0.0667
}
```

### 4. 生成测试流量

生成 ICMP 流量：

```bash
ping -c 100 127.0.0.1
```

生成 TCP 流量（使用 `nc` 或 `telnet`）：

```bash
# 安装 netcat 如果没有
sudo apt-get install -y netcat

# 启动监听服务器（终端 1）
nc -l 8888

# 连接（终端 2）
echo "test" | nc 127.0.0.1 8888
```

---

## eBPF 调试（可选）

### 查看加载的 eBPF 程序

```bash
sudo bpftool prog list
```

输出示例：

```
34: tc  name tc_microsegment_filter  tag 1a2b3c4d5e6f7g8h  gpl
    loaded_at 2025-10-31T13:56:00+0000  uid 0
    xlated 1024B  jited 768B  memlock 4096B  map_ids 10,11,12,13
```

### 查看 eBPF Maps

#### 会话 Map

```bash
sudo bpftool map dump name session_map | head -20
```

#### 策略 Map

```bash
sudo bpftool map dump name policy_map
```

#### 统计 Map

```bash
sudo bpftool map dump name stats_map
```

### 查看 eBPF 程序日志

```bash
# 实时查看 eBPF 内核日志（如果启用了 DEBUG_MODE）
sudo cat /sys/kernel/debug/tracing/trace_pipe
```

---

## 性能测试

### 使用内置性能测试工具

```bash
# 构建性能测试工具
make perf-test

# 运行性能测试
sudo ./bin/perf-test -duration 10 -workers 4
```

**预期输出**:

```
====== Performance Test Results ======
Duration:         10.00 seconds
Workers:          4
Total Packets:    500000
Throughput:       50000 pps
Avg Latency:      5.2 μs
P50 Latency:      4.8 μs
P99 Latency:      12.3 μs
======================================
✓ Performance target met: < 10μs
```

### 手动性能测试

使用 `hping3` 生成高速流量：

```bash
# 安装 hping3
sudo apt-get install -y hping3

# 生成 10000 个 TCP SYN 包
sudo hping3 -S -p 80 -c 10000 --faster 127.0.0.1
```

---

## 验证清单

测试完成后，确认以下项目：

- [ ] ✓ 代理启动成功，无错误日志
- [ ] ✓ API 健康检查返回 200 OK
- [ ] ✓ 可以创建、查询、更新、删除策略
- [ ] ✓ 策略立即生效（无需重启）
- [ ] ✓ 统计数据实时更新
- [ ] ✓ 数据包处理延迟 < 10μs (P99)
- [ ] ✓ 会话跟踪工作正常
- [ ] ✓ ALLOW/DENY 动作正确执行
- [ ] ✓ eBPF 程序正确附加到网络接口
- [ ] ✓ 代理可以优雅关闭 (Ctrl+C)

---

## 常见问题排查

### 问题 1: 权限错误

**错误信息**:

```
Error: permission denied: loading eBPF programs requires CAP_BPF
```

**解决方案**:

```bash
sudo ./bin/microsegment-agent
```

### 问题 2: 接口不存在

**错误信息**:

```
Error: interface eth0 not found
```

**解决方案**:

```bash
# 查看可用网络接口
ip link show

# 使用 lo 接口测试
sudo ./bin/microsegment-agent --interface lo
```

### 问题 3: 端口已被占用

**错误信息**:

```
Error: bind: address already in use
```

**解决方案**:

```bash
# 使用其他端口
sudo ./bin/microsegment-agent --api-port 8081

# 或杀死占用端口的进程
sudo lsof -ti:8080 | xargs sudo kill -9
```

### 问题 4: eBPF 程序附加失败

**错误信息**:

```
Error: attaching TC program: operation not supported
```

**解决方案**:

检查内核版本（需要 5.10+）：

```bash
uname -r

# 如果内核太旧，升级内核或使用 Docker 测试
```

---

## 下一步

测试通过后，你可以：

1. **继续开发剩余功能**（配置管理、Swagger 文档等）
2. **编写单元测试和集成测试**
3. **部署到生产环境**
4. **性能调优**

参考文档：

- [架构概览](docs/ARCHITECTURE_OVERVIEW.md)
- [实现总结](IMPLEMENTATION_SUMMARY.txt)
- [OpenSpec 变更](openspec/changes/add-control-plane-api/)

---

## 测试报告模板

完成测试后，可以使用以下模板记录结果：

```
测试日期: 2025-10-31
测试环境: Linux 6.4.0, Go 1.23.0
测试人员: [你的名字]

功能测试:
  ✓ 健康检查: PASS
  ✓ 策略 CRUD: PASS
  ✓ 统计查询: PASS
  ✓ 流量处理: PASS

性能测试:
  - 平均延迟: 5.2μs ✓
  - P99 延迟: 12.3μs (目标: <10μs) ✗
  - 吞吐量: 50K pps ✓

问题:
  1. P99 延迟略高于目标，需要进一步优化

建议:
  1. 启用 CPU 亲和性绑定
  2. 调整 eBPF Map 大小
```

---

**祝测试顺利！** 🧪✨

