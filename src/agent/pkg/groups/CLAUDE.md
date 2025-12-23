[上级索引](../CLAUDE.md) > **groups**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# groups

## 架构定位

工作负载分组管理器 | 输入: 分组定义（标签选择器）、工作负载列表 | 输出: 分组成员列表（匹配的工作负载 ID 和 IP）

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| manager.go | 分组管理器，解析分组成员并缓存 | `NewGroupManager()`, `ResolveGroupMembers()`, `InvalidateCache()` |
| selector.go | 标签选择器逻辑（matchLabels、matchExpressions） | `MatchLabels()`, `Evaluate()` |
| storage.go | 分组持久化接口（SQLite） | `CreateGroup()`, `GetGroup()`, `ListGroups()`, `DeleteGroup()` |
| types.go | 分组数据类型定义 | `Group`, `Selector`, `SelectorOperator` |

## 核心功能

- **标签选择器**: 支持 K8s 风格的 matchLabels 和 matchExpressions
- **动态成员**: 工作负载标签变化时自动更新分组成员
- **缓存优化**: 缓存分组成员列表，减少计算开销
- **操作符支持**: In、NotIn、Exists、DoesNotExist

## 选择器操作符

| 操作符 | 描述 | 示例 |
|--------|------|------|
| In | 标签值在列表中 | `tier: In [frontend, backend]` |
| NotIn | 标签值不在列表中 | `env: NotIn [dev, test]` |
| Exists | 标签键存在 | `deprecated: Exists` |
| DoesNotExist | 标签键不存在 | `legacy: DoesNotExist` |

## 应用场景

- **策略简化**: 策略规则引用分组而非具体 IP（如 `allow from group:frontend to group:backend`）
- **动态环境**: Pod IP 频繁变化的环境
- **标签分段**: 基于标签的网络分段
