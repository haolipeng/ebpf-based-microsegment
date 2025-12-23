[上级索引](../CLAUDE.md) > **ipcache**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# ipcache

## 架构定位

IP 缓存管理器 | 输入: IP/CIDR 前缀、NumericIdentity、元数据 | 输出: eBPF ipcache_map 更新（LPM Trie）、IP-身份变化事件

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| ipcache.go | IP 缓存管理器，维护映射并同步到 BPF Map | `NewIPCache()`, `Upsert()`, `Delete()`, `Lookup()`, `SyncToBPFMap()` |

## 核心功能

- **LPM 匹配**: 使用 Longest Prefix Match Trie 实现 CIDR 高效查找
- **动态更新**: IP-身份映射变化时实时同步到 eBPF
- **IPv4/IPv6**: 统一处理（IPv4 存储为 IPv4-mapped IPv6）
- **事件监听**: 映射变化时通知订阅者（如策略编译器）

## 数据结构

- **IPCacheKey**: PrefixLen（CIDR 前缀长度 0-128）、IP（IPv6 地址，IPv4 映射）
- **IPCacheValue**: Identity（NumericIdentity）、Pad（对齐填充）

## 应用场景

- **工作负载身份**: Pod IP -> Identity 映射
- **网段策略**: 整个子网映射到同一身份（如 10.0.0.0/24 -> frontend）
- **外部服务**: 外部 IP 映射到特殊身份（如 internet、cluster-external）
