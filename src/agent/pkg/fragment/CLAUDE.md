[上级索引](../CLAUDE.md) > **fragment**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# fragment

## 架构定位

IP 分片清理器 | 输入: eBPF fragment_state_map（分片 ID -> 分片状态） | 输出: 清理超时分片条目，维护分片统计

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| cleaner.go | 分片清理器，定期扫描和清理超时分片 | `NewFragmentCleaner()`, `Start()`, `Stop()`, `scanLoop()` |
| types.go | 分片数据类型定义 | `FragmentKey`, `FragmentState`, `FragmentCleanerConfig` |

## 核心功能

- **超时清理**: 定期扫描 fragment_state_map，删除超过 TTL 的分片
- **统计追踪**: 记录清理的分片数、当前分片数、错误数
- **可配置**: 清理间隔、分片超时时间可配置
- **优雅关闭**: 确保清理任务完成后再退出

## 配置参数

| 参数 | 描述 | 默认值 |
|------|------|--------|
| ScanInterval | 扫描间隔 | 30s |
| FragmentTTL | 分片超时时间 | 60s |
| Enabled | 是否启用清理 | true |

## 应用场景

- 防止大量分片攻击导致的内存耗尽
- 清理因网络丢包导致的不完整分片
- 维护分片缓存在合理范围内
