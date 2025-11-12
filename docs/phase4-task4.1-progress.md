# Phase 4 Task 4.1: Egress 策略执行测试 - 进展报告

## 日期
2025-11-12

## 状态
🟡 进行中 - 发现关键问题需要修复

## 已完成工作

### 1. 扩展统计数据结构 ✅

#### 修改文件: `src/agent/pkg/dataplane/dataplane.go`

添加了 Egress 相关统计字段:
```go
type Statistics struct {
    // ... 现有字段
    // Direction-specific statistics (for egress support)
    IngressPackets  uint64
    EgressPackets   uint64
    IngressDenied   uint64
    EgressDenied    uint64
}
```

更新 `GetStatistics()` 方法读取新统计:
```go
stats.IngressPackets = readStat(8)  // STATS_INGRESS_PACKETS
stats.EgressPackets = readStat(9)   // STATS_EGRESS_PACKETS
stats.IngressDenied = readStat(10)  // STATS_INGRESS_DENIED
stats.EgressDenied = readStat(11)   // STATS_EGRESS_DENIED
```

### 2. 扩展 E2E 测试框架 ✅

#### 修改文件: `src/agent/test/e2e/framework.go`

**新增方法**:
- `TryConnectFromServer(port int) bool` - 从 Server 主动连接 Client (测试 Egress)
- `StartTCPServerOnClient(port int)` - 在 Client 侧启动 TCP 服务器

**更新方法**:
- `AssertStatistic()` - 支持新的统计字段 (ingress_packets, egress_packets, ingress_denied, egress_denied)
- `WaitForStatistic()` - 同上

### 3. 创建 Egress 策略测试 ✅

#### 新文件: `src/agent/test/e2e/egress_policy_outbound_test.go`

**测试场景**:

1. **TestE2E_EgressOutboundDeny** ❌ 失败
   - 目的: 测试 Egress DENY 策略阻止 Server 主动出站连接
   - 拓扑: Client 运行 TCP Server (监听 9000), Server 尝试连接 Client
   - 策略: Egress DENY (10.100.0.2 → 10.100.0.1:9000)
   - **问题**: 连接仍然成功, egress_denied = 0

2. **TestE2E_EgressOutboundAllow** ✅ 通过
   - 目的: 测试 Egress ALLOW 策略允许 Server 主动出站连接
   - 策略: Egress ALLOW + Ingress ALLOW (双向)
   - 结果: 连接成功, egress_packets 增加

3. **TestE2E_EgressOutboundDefaultBehavior** ✅ 通过
   - 目的: 测试没有 Egress 策略时的默认行为
   - 结果: 默认 ALLOW (连接成功)

## 发现的关键问题 ⚠️

### 问题: Egress DENY 策略不生效

#### 症状

测试 `TestE2E_EgressOutboundDeny` 失败:

```
统计 (策略创建前): egress_denied=0, egress_packets=0
策略创建: rule_id=5000 10.100.0.2:0 -> 10.100.0.1:9000 proto=tcp action=deny dir=egress
连接测试: connected = true (应该是 false!)
统计 (测试后): egress_denied=0, egress_packets=4, denied_packets=0
```

**问题分析**:
1. 策略成功添加到 `wildcard_policy_map` (日志确认)
2. 但连接仍然成功建立 (应该被拒绝)
3. `egress_denied` 计数器始终为 0
4. `egress_packets = 4` 说明有 4 个 Egress 数据包通过了

#### 可能原因

需要进一步调查以下方面:

1. **策略匹配逻辑**
   - Direction 字段匹配是否正确?
   - 端口匹配 (wildcard src_port=0) 是否工作?
   - IP 地址匹配是否正确?

2. **会话缓存机制**
   - 第一个 SYN 包创建会话时是否正确查找策略?
   - 会话缓存的策略动作是否考虑了方向?

3. **数据结构对齐**
   - Go 侧和 eBPF 侧的结构体布局是否一致?
   - 字节序 (endianness) 是否正确?

4. **Direction 检测**
   - `skb->ingress_ifindex` 检测 Egress 是否准确?
   - 策略中的 direction 字段是否正确传递?

## 下一步行动

### 立即行动

1. **启用 DEBUG 模式** - 查看 eBPF 日志
   ```c
   #define DEBUG_MODE 1  // 在 tc_microsegment.bpf.c 中
   ```
   然后使用 `sudo cat /sys/kernel/debug/tracing/trace_pipe` 查看日志

2. **Dump eBPF Maps** - 验证策略是否正确存储
   ```bash
   sudo bpftool map dump name wildcard_policy_map
   ```

3. **抓包分析** - 查看实际数据包
   ```bash
   sudo tcpdump -i veth-server -nn
   ```

### 调试策略

A. 添加详细日志到测试,打印:
   - 策略详细信息 (IP, port, direction)
   - 每个步骤的统计快照

B. 创建简化测试用例:
   - 单个 Egress DENY 策略
   - 单个 TCP SYN 包
   - 验证是否被拒绝

C. 代码审查:
   - 重新检查 `matches_wildcard()` 函数
   - 验证 `lookup_policy_action()` 的 direction 参数传递
   - 确认策略写入时的数据格式

## 测试命令

```bash
# 清理旧的 pinned maps
sudo rm -rf /sys/fs/bpf/microsegment

# 运行 Egress Outbound 测试
sudo -E /usr/local/go/bin/go test -v ./src/agent/test/e2e \
    -run "TestE2E_EgressOutbound" \
    -timeout 3m

# 只运行失败的测试
sudo -E /usr/local/go/bin/go test -v ./src/agent/test/e2e \
    -run "TestE2E_EgressOutboundDeny" \
    -timeout 2m
```

## 文件清单

### 修改的文件

1. `src/agent/pkg/dataplane/dataplane.go` - 添加 Egress 统计字段
2. `src/agent/test/e2e/framework.go` - 扩展测试框架支持 Egress 测试

### 新增的文件

1. `src/agent/test/e2e/egress_policy_test.go` - 响应流量测试 (设计有问题,可能需要重构)
2. `src/agent/test/e2e/egress_policy_outbound_test.go` - 主动出站连接测试

## 技术要点

### Egress 流量定义

在我们的架构中 (eBPF 附加在 Server veth):
- **Ingress**: Client → Server (进入 Server)
- **Egress**: Server → Client (离开 Server)

### 测试拓扑

```
[Client NS]  <-----------  [Server NS + eBPF]
 TCP Server                主动连接 (Egress)
 监听 9000
```

### Direction 检测

```c
// TC 程序中的方向检测
__u8 direction = (skb->ingress_ifindex != 0) ? POLICY_DIR_INGRESS : POLICY_DIR_EGRESS;
```

- `ingress_ifindex != 0`: Ingress (从外部进入)
- `ingress_ifindex == 0`: Egress (从内部发出)

## 总结

Task 4.1 基础框架已完成,但发现 **Egress DENY 策略不生效** 的关键bug。这需要深入调试 eBPF 策略匹配逻辑才能继续。

当前进度: **60%**
- ✅ 统计数据扩展
- ✅ 测试框架扩展
- ✅ 测试用例创建
- ❌ Egress DENY 策略修复 (阻塞中)
- ⏳ 测试验证和文档

---
生成时间: 2025-11-12T15:25:00+08:00
