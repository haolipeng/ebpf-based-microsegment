[上级索引](../CLAUDE.md) > **testutil**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# testutil

## 架构定位

eBPF 和网络测试工具 | 输入: 测试参数（IP、端口、协议） | 输出: 测试数据包、Map 条目、网络连接

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| ebpf.go | eBPF Map 测试辅助函数 | `NewFlowKey()`, `NewPolicyValue()`, `InsertPolicy()`, `LookupSession()` |
| network.go | 网络测试工具（创建监听器、连接） | `CreateTestListener()`, `CreateTestConnection()`, `WaitForConnection()` |
| traffic.go | 流量生成工具（发送 TCP/UDP 包） | `SendTCPTraffic()`, `SendUDPTraffic()`, `GenerateTraffic()` |

## 核心功能

- **FlowKey 构建**: 从 IP/Port/Protocol 生成 FlowKey（与 PolicyManager 一致）
- **Map 操作**: 简化 eBPF Map 的插入、查询、删除
- **数据包生成**: 构建符合策略的测试数据包
- **网络模拟**: 创建本地 TCP/UDP 监听器和连接

## 测试场景支持

- **策略执行测试**: 验证 eBPF 程序正确应用策略
- **会话跟踪测试**: 验证 session_map 正确记录连接
- **通配符测试**: 验证 CIDR/端口范围匹配
- **性能测试**: 高并发流量生成
