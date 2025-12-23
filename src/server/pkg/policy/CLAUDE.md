[上级索引](../CLAUDE.md) > **policy**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# policy

## 架构定位

策略压缩存储 | 输入: 策略规则（通配符策略） | 输出: 紧凑的槽位数组（用于 eBPF Map 同步）

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| compact_storage.go | 紧凑策略存储管理 | `CompactPolicyStorage`, `AddPolicy()`, `RemovePolicy()`, `GetCompactList()` |

## 核心功能

- **槽位复用**: 维护 first_free_index 指针，最小化策略数组空隙
- **空闲列表**: 立即回收已删除策略的槽位
- **紧凑输出**: 提供无空隙的策略列表用于 eBPF 同步
- **线程安全**: Mutex 保护的并发操作

## 设计目标

优化 eBPF Map 同步效率：
- eBPF 通配符策略 Map 是数组类型
- 空隙会导致 eBPF 扫描浪费
- 紧凑存储减少 Agent 下发的策略条目数

## 与 Agent 的关系

Server 管理策略的中心化存储，Agent 从 Server 同步策略后编译为 eBPF Map 条目。

