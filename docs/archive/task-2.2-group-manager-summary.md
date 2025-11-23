# 任务 2.2：分组成员解析（Group Member Resolution）

**完成日期**: 2025-11-03
**状态**: ✅ 已完成
**覆盖率**: 78.2%（包覆盖率），manager.go核心功能 70-100%

## 概述

实现了 GroupManager，用于解析哪些工作负载属于特定分组。这是基于标签的策略管理系统的核心组件，将标签选择器应用于工作负载集合以确定组成员关系。

## 实现的功能

### 1. 核心成员解析

#### `ResolveGroupMembers(groupName string) ([]*Workload, error)`
- 解析指定分组的所有成员工作负载
- 返回完整的 Workload 对象列表
- 自动更新缓存（如果启用）
- 使用 `MatchesGroup()` 评估每个工作负载

**示例用法**:
```go
mgr := NewGroupManager(groupStorage, workloadMgr)
members, err := mgr.ResolveGroupMembers("prod-web-servers")
// 返回: [wl-1, wl-2] （所有匹配 role=web AND env=prod 的工作负载）
```

#### `ResolveGroupMemberIDs(groupName string) ([]string, error)`
- 轻量级版本，只返回工作负载 ID
- 优先从缓存读取（如果启用）
- 更高性能：609 ns/op（缓存命中）vs 490 μs/op（缓存未命中）

#### `IsWorkloadInGroup(workloadID, groupName string) (bool, error)`
- 检查特定工作负载是否属于分组
- 用于快速成员资格验证
- 不使用缓存（直接评估）

#### `ResolveAllGroupMemberships() (map[string][]string, error)`
- 批量解析所有分组的成员
- 返回 map[groupName][]workloadIDs
- 用于策略编译和全量同步场景

### 2. 缓存机制（可选优化）

#### 缓存策略
- **默认启用**：`cacheEnabled = true`
- **缓存键**：`groupName -> cachedMembers{workloadIDs, timestamp}`
- **线程安全**：使用 `sync.RWMutex` 保护

#### 缓存失效
- **自动失效**：
  - `UpdateGroup()` - 更新分组时自动使缓存失效
  - `DeleteGroup()` - 删除分组时自动使缓存失效

- **手动失效**：
  - `InvalidateCache()` - 清空整个缓存
  - `InvalidateGroupCache(groupName)` - 清空特定分组的缓存

#### 缓存性能
- **无缓存**: 490,039 ns/op（~490 μs）
- **有缓存**: 609 ns/op（~0.6 μs）
- **性能提升**: ~800x 倍

#### 缓存控制
```go
// 禁用缓存（用于测试或低延迟场景）
mgr.SetCacheEnabled(false)

// 获取缓存统计
stats := mgr.GetCacheStats()
// {"enabled": true, "entry_count": 5}
```

### 3. 管理功能（CRUD）

提供了便捷的分组管理方法：

- `CreateGroup(group *Group) error` - 创建分组并验证
- `GetGroup(name string) (*Group, error)` - 获取分组
- `UpdateGroup(group *Group) error` - 更新分组并失效缓存
- `DeleteGroup(name string) error` - 删除分组并失效缓存
- `ListGroups() ([]*Group, error)` - 列出所有分组
- `Close() error` - 关闭管理器和存储

## 架构设计

### 组件关系

```
┌────────────────────────────────────────┐
│         GroupManager                   │
│  - storage: GroupStorage               │
│  - workloadMgr: *workload.Manager      │
│  - memberCache: map[string]*cached...  │
└────────────────────────────────────────┘
          ↓                    ↓
  ┌───────────────┐    ┌──────────────────┐
  │ GroupStorage  │    │ WorkloadManager  │
  │ (SQLite)      │    │ (列出工作负载)    │
  └───────────────┘    └──────────────────┘
          ↓                    ↓
  ┌────────────────────────────────────┐
  │     Selector Evaluation Engine     │
  │  - EvaluateSelector()              │
  │  - EvaluateSelectors() (AND logic) │
  │  - MatchesGroup()                  │
  └────────────────────────────────────┘
```

### 解析流程

```
ResolveGroupMembers(groupName)
  ↓
1. GetGroup(groupName) → Group{Selectors}
  ↓
2. ListWorkloads() → []Workload
  ↓
3. For each workload:
     MatchesGroup(wl, group) → bool
       ↓
     EvaluateSelectors(wl, group.Selectors)
       ↓
     For each selector (AND logic):
       EvaluateSelector(wl, sel) → bool
  ↓
4. Filter matching workloads
  ↓
5. Update cache (if enabled)
  ↓
Return members
```

## 测试覆盖

### 测试用例（11 个测试函数）

1. **TestNewGroupManager** - 管理器创建和初始化
2. **TestResolveGroupMembers_NoWorkloads** - 空工作负载集
3. **TestResolveGroupMembers_OneMatch** - 单个匹配
4. **TestResolveGroupMembers_MultipleMatches** - 多个匹配
5. **TestResolveGroupMembers_ComplexSelectors** - 复杂选择器组合
   - `in` 操作符：role in [api, cache]
   - `exists` 操作符：tier exists
   - `!=` 操作符：role != web
   - `not-in` 操作符：role not-in [api, cache]
6. **TestResolveGroupMemberIDs** - 轻量级 ID 解析
7. **TestIsWorkloadInGroup** - 单个工作负载成员资格检查
8. **TestResolveAllGroupMemberships** - 批量解析所有分组
9. **TestCaching** - 缓存机制验证
10. **TestCacheInvalidationOnUpdate** - 更新时缓存失效
11. **TestPerformance** - 100 工作负载 + 10 分组性能测试

### 覆盖率详情

**总体覆盖率**: 78.2%

**manager.go 关键函数覆盖率**:
- `NewGroupManager`: 100%
- `ResolveGroupMembers`: 81.2%
- `ResolveGroupMemberIDs`: 90.9%
- `IsWorkloadInGroup`: 70.0%
- `ResolveAllGroupMemberships`: 72.7%
- `InvalidateGroupCache`: 100%
- `GetCacheStats`: 100%
- `updateCache`: 100%
- `getCachedMemberIDs`: 100%

**selector.go 函数覆盖率**（来自任务 2.1）:
- `EvaluateSelector`: 90.0%
- `EvaluateSelectors`: 100%
- `MatchesGroup`: 100%
- 所有操作符函数: 83-100%

### 性能基准测试

```
BenchmarkResolveGroupMembers-4
    2220 ops	    490,039 ns/op	   97137 B/op	    2671 allocs/op

BenchmarkResolveGroupMembersWithCache-4
 1,823,794 ops	       609.2 ns/op	     912 B/op	       2 allocs/op

BenchmarkEvaluateSelector-4
57,985,274 ops	        18.93 ns/op	       0 B/op	       0 allocs/op

BenchmarkEvaluateSelectors-4
19,009,288 ops	        64.52 ns/op	       0 B/op	       0 allocs/op
```

**关键性能指标**:
- 选择器评估：~19 ns/op（零分配）
- 多选择器评估：~65 ns/op（零分配）
- 成员解析（无缓存）：~490 μs/op（50 个工作负载）
- 成员解析（有缓存）：~609 ns/op（800x 提升）

## 性能验证

### 测试场景：100 工作负载 + 10 分组

**设置**:
- 100 个工作负载，5 种角色（web, api, db, cache, mq）
- 3 种环境（prod, staging, dev）
- 10 个分组，不同选择器组合

**结果**:
```
Performance: Resolved 10 groups with 100 workloads in 10.3ms
```

**性能分析**:
- **目标**: <100ms
- **实际**: 10.3ms
- **裕度**: **9.7x** 超出目标
- **平均每组**: ~1ms
- **平均每工作负载评估**: ~103 μs

**结论**: 性能远超验收标准，可扩展到更大规模（~1000 工作负载仍可保持在 100ms 以内）

## 使用示例

### 基本用法

```go
// 创建管理器
groupStorage, _ := NewSQLiteGroupStorage("/path/to/db")
workloadMgr := workload.NewManager(workloadStorage)
mgr := NewGroupManager(groupStorage, workloadMgr)
defer mgr.Close()

// 创建分组
group := NewGroup("prod-web-servers")
group.AddSelector(NewEqualSelector("role", "web"))
group.AddSelector(NewEqualSelector("env", "prod"))
mgr.CreateGroup(group)

// 解析成员
members, _ := mgr.ResolveGroupMembers("prod-web-servers")
fmt.Printf("Found %d members\n", len(members))

// 检查特定工作负载
inGroup, _ := mgr.IsWorkloadInGroup("wl-123", "prod-web-servers")
if inGroup {
    fmt.Println("wl-123 is a member")
}
```

### 批量解析

```go
// 解析所有分组成员
memberships, _ := mgr.ResolveAllGroupMemberships()

for groupName, memberIDs := range memberships {
    fmt.Printf("%s: %v\n", groupName, memberIDs)
}
// 输出:
// prod-web-servers: [wl-1, wl-2]
// databases: [wl-3, wl-4]
// cache-servers: [wl-5]
```

### 缓存管理

```go
// 工作负载标签更新后，使所有缓存失效
mgr.InvalidateCache()

// 或仅使特定分组缓存失效
mgr.InvalidateGroupCache("prod-web-servers")

// 检查缓存统计
stats := mgr.GetCacheStats()
fmt.Printf("Cache enabled: %v, entries: %d\n",
    stats["enabled"], stats["entry_count"])
```

### 高级场景：策略编译

```go
// 未来任务 3.1-3.3 的基础

// 步骤 1: 解析源分组成员
srcMembers, _ := mgr.ResolveGroupMemberIDs("prod-web-servers")

// 步骤 2: 解析目标分组成员
dstMembers, _ := mgr.ResolveGroupMemberIDs("databases")

// 步骤 3: 编译策略规则
// 对于每个 srcIP in srcMembers:
//   对于每个 dstIP in dstMembers:
//     生成策略条目 (srcIP, dstIP, ports, action)
//     写入 eBPF map

// （此逻辑将在任务 3.2-3.3 中实现）
```

## 设计决策

### 1. 为什么需要缓存？

**问题**: 每次解析都需要：
1. 从数据库读取分组（1 次查询）
2. 从数据库读取所有工作负载（1 次查询）
3. 对每个工作负载评估选择器（N 次迭代）

**影响**: 对于 100 个工作负载，~490 μs

**解决方案**: 缓存解析结果，将性能提升到 ~609 ns（800x）

**权衡**:
- ✅ 优点：查询性能提升 800x
- ⚠️ 缺点：需要管理缓存失效（已通过自动失效解决）
- ✅ 优点：内存开销小（每个缓存条目仅存储 ID 列表）

### 2. 为什么提供多种解析方法？

**ResolveGroupMembers** - 完整对象
- 用于需要完整工作负载信息的场景（如 API 响应）
- 返回 `[]*Workload`

**ResolveGroupMemberIDs** - 仅 ID
- 用于策略编译和高性能场景
- 返回 `[]string`
- 优先使用缓存

**IsWorkloadInGroup** - 单个检查
- 用于快速验证场景（如准入控制）
- 直接评估，不使用缓存（确保最新结果）

**ResolveAllGroupMemberships** - 批量
- 用于全量同步和策略编译
- 一次性解析所有分组

### 3. 为什么缓存存储 ID 而非完整对象？

**原因**:
1. **内存效率**: ID 是字符串（~16 字节），Workload 对象 >200 字节
2. **一致性**: 工作负载对象可能在缓存外更新，ID 不会改变
3. **灵活性**: 需要完整对象时，可从 WorkloadManager 查询最新数据

### 4. 为什么自动失效而非 TTL？

**选择**: 基于事件的失效（UpdateGroup, DeleteGroup）

**原因**:
- 分组配置变化不频繁（通常是管理员操作）
- 需要立即生效（安全策略不能有延迟）
- 避免 TTL 带来的不确定性

**权衡**:
- ✅ 优点：缓存一致性强
- ✅ 优点：零延迟失效
- ⚠️ 缺点：工作负载标签更新时需手动失效（未来可通过 webhook 改进）

## 与其他组件的集成

### 1. 与 WorkloadManager 集成

```go
type GroupManager struct {
    storage     Storage
    workloadMgr *workload.Manager  // 依赖注入
    ...
}
```

**职责分离**:
- `WorkloadManager`: 管理工作负载生命周期（CRUD）
- `GroupManager`: 基于标签解析分组成员

### 2. 与 Selector Engine 集成

```go
func (m *GroupManager) ResolveGroupMembers(...) {
    ...
    for _, wl := range allWorkloads {
        if MatchesGroup(wl, group) {  // 使用 selector.go
            members = append(members, wl)
        }
    }
    ...
}
```

**复用**: GroupManager 复用任务 2.1 的选择器评估引擎

### 3. 未来集成点（任务 3.x）

```
PolicyCompiler (任务 3.2)
    ↓
使用 ResolveGroupMembers("src-group")
使用 ResolveGroupMembers("dst-group")
    ↓
生成 IP 规则
    ↓
写入 eBPF map
```

## 已知限制和改进方向

### 当前限制

1. **缓存失效策略**
   - ❌ 工作负载标签更新时不会自动使缓存失效
   - 解决方案：需要手动调用 `InvalidateCache()`

2. **无增量更新**
   - ❌ 缓存失效后需要重新解析整个分组
   - 改进：可实现增量成员更新（仅处理变化的工作负载）

3. **无分页支持**
   - ❌ `ResolveGroupMembers()` 返回所有成员
   - 改进：对于大型分组（>1000 成员），可添加分页

4. **无并发解析**
   - ❌ `ResolveAllGroupMemberships()` 串行解析
   - 改进：可使用 goroutine 并行解析多个分组

### 未来改进（非必需）

#### 1. 工作负载事件监听

```go
// 未来改进：监听工作负载更新事件
func (m *GroupManager) OnWorkloadUpdated(wlID string) {
    // 仅失效包含此工作负载的分组缓存
    for groupName := range m.memberCache {
        if m.memberCache[groupName].Contains(wlID) {
            m.InvalidateGroupCache(groupName)
        }
    }
}
```

#### 2. 增量成员更新

```go
// 未来改进：增量更新缓存
func (m *GroupManager) UpdateWorkloadInCache(wl *Workload) {
    for groupName, group := range m.groups {
        matches := MatchesGroup(wl, group)
        cached := m.memberCache[groupName]

        if matches && !cached.Contains(wl.ID) {
            cached.Add(wl.ID)  // 添加到缓存
        } else if !matches && cached.Contains(wl.ID) {
            cached.Remove(wl.ID)  // 从缓存移除
        }
    }
}
```

#### 3. 缓存预热

```go
// 未来改进：启动时预热缓存
func (m *GroupManager) WarmupCache() error {
    groups, _ := m.ListGroups()
    for _, group := range groups {
        m.ResolveGroupMemberIDs(group.Name)  // 填充缓存
    }
    return nil
}
```

#### 4. 并发批量解析

```go
// 未来改进：并行解析分组
func (m *GroupManager) ResolveAllGroupMembershipsParallel() (map[string][]string, error) {
    groups, _ := m.ListGroups()

    results := make(map[string][]string)
    var mu sync.Mutex
    var wg sync.WaitGroup

    for _, group := range groups {
        wg.Add(1)
        go func(g *Group) {
            defer wg.Done()
            memberIDs, _ := m.ResolveGroupMemberIDs(g.Name)

            mu.Lock()
            results[g.Name] = memberIDs
            mu.Unlock()
        }(group)
    }

    wg.Wait()
    return results, nil
}
```

## 总结

### 完成的功能

✅ **核心解析功能**（4 种方法）
- ResolveGroupMembers - 完整对象
- ResolveGroupMemberIDs - 轻量级 ID
- IsWorkloadInGroup - 单个检查
- ResolveAllGroupMemberships - 批量解析

✅ **缓存优化**（800x 性能提升）
- 自动缓存更新
- 事件驱动失效
- 线程安全设计
- 可配置启用/禁用

✅ **管理功能**（CRUD + Close）
- 完整的分组生命周期管理
- 自动验证
- 集成缓存失效

✅ **全面测试**（11 个测试 + 基准测试）
- 78.2% 总体覆盖率
- 核心功能 70-100% 覆盖率
- 性能测试验证

✅ **性能目标超额达成**
- 目标：<100ms（100 工作负载）
- 实际：10.3ms
- **9.7x 超出目标**

### 下一步

**任务 2.3**: 自动标记引擎（AutoTagger）
- 基于镜像名推断角色（nginx → web）
- 基于端口号推断角色（3306 → db）
- 自动生成标签

**任务 3.1**: 策略规则数据模型
- 定义 PolicyRule 结构体
- 存储策略规则

**任务 3.2**: 策略编译引擎
- **关键依赖**: 使用 `ResolveGroupMembers()` 解析源/目标分组
- 生成 IP-based 策略规则
- 写入 eBPF map

---

**文档版本**: 1.0
**最后更新**: 2025-11-03
**审核者**: 待人工审核
