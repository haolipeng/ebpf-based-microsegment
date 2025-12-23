[上级索引](../CLAUDE.md) > **conntrack**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# conntrack

## 架构定位

同步内核 conntrack 表到 eBPF Map，支持 NAT 场景下的正确策略匹配。
**输入**: Netlink conntrack 事件（新建/更新/删除连接）
**输出**: eBPF conntrack_cache_map 更新（原始五元组 -> NAT 后五元组映射）

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| syncer.go | Conntrack 同步器主体，监听内核事件 | `NewConntrackSyncer()`, `Start()`, `Stop()` |
| converter.go | Conntrack 条目转换为 eBPF Map 格式 | `ConvertToMapKey()`, `ConvertToMapValue()` |
| types.go | 数据类型定义（缓存键值、配置、统计） | `ConntrackCacheKey`, `ConntrackCacheValue`, `SyncConfig` |

## 核心功能

- **实时同步**: 监听内核 conntrack 事件，实时更新 eBPF Map
- **NAT 支持**: 记录 SNAT/DNAT 转换关系，确保策略正确应用
- **自动清理**: 定期清理过期的 conntrack 条目
- **统计追踪**: 记录同步成功/失败次数、当前条目数

## 应用场景

- 容器使用 SNAT 访问外部服务
- K8s Service 使用 DNAT 负载均衡
- Pod 间通信经过 NAT 转换
