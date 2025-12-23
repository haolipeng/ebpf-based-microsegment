[上级索引](../CLAUDE.md) > **session**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# session

## 架构定位

会话超时管理器 | 输入: eBPF session_map（五元组 -> 会话状态） | 输出: 清理超时会话、维护会话统计

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| timeout_manager.go | 会话超时管理器，定期扫描和清理 | `NewTimeoutManager()`, `Start()`, `Stop()`, `scanLoop()` |
| types.go | 会话数据类型定义 | `SessionKey`, `SessionValue`, `SessionTimeoutConfig` |

## 核心功能

- **超时清理**: 定期扫描 session_map，删除超过 idle timeout 的会话
- **统计追踪**: 记录清理的会话数、当前活跃会话数
- **可配置**: 扫描间隔、超时时间可配置
- **协议感知**: TCP、UDP、ICMP 使用不同的超时时间

## 配置参数

| 参数 | 描述 | 默认值 |
|------|------|--------|
| ScanInterval | 扫描间隔 | 60s |
| TCPTimeout | TCP 空闲超时 | 3600s |
| UDPTimeout | UDP 空闲超时 | 300s |
| ICMPTimeout | ICMP 超时 | 30s |
| Enabled | 是否启用清理 | true |

## 应用场景

- **防止内存泄漏**: 清理长时间不活跃的连接
- **会话限制**: 维护活跃会话数在可控范围
- **资源回收**: 释放 LRU Hash Map 的空间
