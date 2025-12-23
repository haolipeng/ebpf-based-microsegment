[上级索引](../CLAUDE.md) > **identity**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# identity

## 架构定位

安全身份管理 | 输入: 工作负载标签集合 | 输出: 数值身份 ID、标签哈希、引用计数

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| allocator.go | 身份分配和管理 | `Identity`, `NumericIdentity`, `ComputeLabelHash()`, `IdentityStorage` |

## 核心功能

- **身份分配**: 标签集合到数值 ID 的映射
- **哈希计算**: 确定性的标签集合哈希
- **保留身份**: Unknown、Host、World 等系统保留身份
- **引用计数**: 身份的使用跟踪

## 保留身份常量

| 值 | 名称 | 用途 |
|----|------|------|
| 0 | IdentityUnknown | 未知身份 |
| 1 | IdentityHost | 主机身份 |
| 2 | IdentityWorld | 外部世界 |
| 3 | IdentityUnmanaged | 未纳管实体 |
| 256+ | 用户分配 | 工作负载身份 |

## 与 Agent 的关系

Server 端的 identity 包与 Agent 端的 identity 包共享相同的身份常量和接口定义。

