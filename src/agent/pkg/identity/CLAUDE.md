[上级索引](../CLAUDE.md) > **identity**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# identity

## 架构定位

身份管理器 | 输入: 标签集合（map[string]string） | 输出: 数字身份（uint32）、IP-身份映射、策略匹配

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| cache.go | 身份缓存，维护 ID -> Identity 和 Hash -> Identity 映射 | `NewCache()`, `Lookup()`, `Allocate()`, `Release()` |
| policy.go | 基于身份的策略匹配逻辑 | `MatchIdentityPolicy()`, `AllowsIdentity()` |
| types.go | 身份数据类型定义 | `Identity`, `NumericIdentity`, `IPIdentityMetadata` |

## 核心功能

- **身份分配**: 为标签集合分配唯一的数字 ID
- **哈希索引**: 基于标签哈希快速查找身份
- **双向查询**: 支持 ID -> Identity 和 LabelHash -> Identity
- **引用计数**: 跟踪身份使用情况，支持安全回收
- **事件监听**: 身份变化时通知订阅者

## 数据结构

身份包含：ID（数字身份 1-65535）、Labels（标签集合）、LabelHash（标签哈希用于快速比较）、RefCount（引用计数）

## 应用场景

- **策略优化**: 将基于标签的策略编译为基于 ID 的策略，提升 eBPF 匹配性能
- **IP-身份映射**: 通过 IPCache 维护 IP -> Identity 映射，实现 IP 无关的策略
- **身份继承**: Pod 继承 Namespace/Service 的身份
