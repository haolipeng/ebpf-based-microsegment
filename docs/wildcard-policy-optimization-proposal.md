# 通配符策略性能优化方案

**文档版本**: v1.0
**创建日期**: 2025-11-17
**状态**: 提案 (Proposal)
**作者**: Claude AI Assistant

---

## 📋 目录

1. [执行摘要](#执行摘要)
2. [问题分析](#问题分析)
3. [优化方案一：提高扫描上限](#优化方案一提高扫描上限)
4. [优化方案二：分层索引架构](#优化方案二分层索引架构)
5. [性能影响分析](#性能影响分析)
6. [实施计划](#实施计划)
7. [风险评估](#风险评估)
8. [参考资料](#参考资料)

---

## 执行摘要

### 背景
当前通配符策略匹配系统采用线性扫描方式，虽然对第一个数据包的查找时间控制在 100 微秒以内，但存在以下关键限制：

- **扫描上限**: 仅能扫描前 50 个通配符策略槽位
- **容量浪费**: 虽然数组容量为 1000 个槽位，但实际只能利用 5%
- **扩展受限**: 无法支持大规模通配符策略部署

### 优化目标
1. **提高扫描上限**: 从 50 增加到 200-500，提升容量利用率至 20%-50%
2. **降低查找延迟**: 通过分层索引将平均查找时间从 50 微秒降低到 10-20 微秒
3. **保持兼容性**: 确保与现有会话缓存机制无缝集成

### 预期收益
- **容量提升**: 4-10倍通配符策略容量
- **性能提升**: 2-5倍平均查找性能
- **功能增强**: 支持更复杂的微隔离场景

---

## 问题分析

### 当前架构概览

```
┌─────────────────────────────────────────────────────────┐
│                   Policy Lookup Flow                    │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐    Fast Path (99%+)                  │
│  │ New Packet   │ ──────────────────► Session Cache    │
│  └──────────────┘         < 1 μs                        │
│         │                                                │
│         │ First Packet                                  │
│         ▼                                                │
│  ┌──────────────┐    O(1) Hash Lookup                  │
│  │ Exact Match  │ ──────────────────► policy_map       │
│  │  (HASH map)  │         < 1 μs                        │
│  └──────────────┘                                        │
│         │ No Match                                       │
│         ▼                                                │
│  ┌──────────────┐    O(n) Linear Scan                  │
│  │ Wildcard     │ ──────────────────► wildcard_policy  │
│  │ Match        │      < 100 μs           _map[1000]   │
│  │ (ARRAY map)  │    ⚠️ Only scan [0-49]               │
│  └──────────────┘                                        │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 核心瓶颈

#### 1. eBPF 验证器限制

**问题根源**: `src/bpf/headers/policy_match.h:24`
```c
/* Maximum number of wildcard policy entries to scan
 * Reduced from 100 to 50 to improve performance
 * This value should be aligned with server-side compact storage implementation
 */
#define MAX_WILDCARD_LOOP 50
```

**技术背景**:
- eBPF 验证器要求循环必须可验证（有明确的上界）
- `#pragma unroll` 编译指令将循环展开为顺序指令
- 展开后的指令数量不能超过 eBPF 程序的复杂度上限（BPF_COMPLEXITY_LIMIT_INSNS）
- Linux 内核版本不同，复杂度限制也不同：
  - Linux 5.2+: 1,000,000 指令
  - Linux 4.14-5.1: 100,000 指令

**当前代码**: `src/bpf/headers/policy_match.h:161-189`
```c
// Linear scan all wildcard policies with support for IP ranges, port ranges, etc.
struct wildcard_policy *wildcard;
struct wildcard_policy *best_match = NULL;
__u8 best_priority = 0;

// Use #pragma unroll to satisfy eBPF verifier requirements
// Scan at most MAX_WILDCARD_LOOP (50) wildcard policies
#pragma unroll
for (__u32 i = 0; i < MAX_WILDCARD_LOOP; i++) {
    __u32 idx = i;
    if (idx >= MAX_ENTRIES_WILDCARD_POLICY)
        break;

    wildcard = bpf_map_lookup_elem(&wildcard_policy_map, &idx);

    if (!wildcard)
        continue;

    // Early stop optimization: immediately stop scanning when empty slot is encountered
    // Assume compact storage of policies (guaranteed by Server side), no valid policies after empty slot
    if (wildcard->rule_id == 0)
        break;

    // Check if it matches (including direction match)
    if (!matches_wildcard(key, wildcard, direction))
        continue;

    // If matched, select policy with highest priority
    // Higher priority value = higher priority (e.g., priority=100 > priority=50)
    if (!best_match || wildcard->priority > best_priority) {
        best_match = wildcard;
        best_priority = wildcard->priority;
    }
}
```

#### 2. 容量利用率低

**数据对比**:
| 指标 | 当前值 | 容量 | 利用率 |
|-----|-------|------|--------|
| 数组总容量 | 1,000 槽位 | 100% | - |
| 可扫描范围 | 50 槽位 | 5% | ⚠️ **极低** |
| 浪费容量 | 950 槽位 | 95% | - |

**影响**:
- 即使插入了第 51-1000 个策略，eBPF 程序也无法扫描到
- 用户空间可以插入策略，但 eBPF 数据平面不会应用这些策略
- 可能导致安全漏洞（策略未生效但用户认为已生效）

#### 3. 线性扫描性能

**时间复杂度**: O(n)，其中 n = 策略数量（当前最多 50）

**最坏情况延迟**:
```
假设每次 matches_wildcard() 耗时 2 微秒:
- 扫描 50 个策略: 50 × 2 = 100 微秒
- 扫描 200 个策略: 200 × 2 = 400 微秒  ⚠️ 超出目标 (< 100 μs)
- 扫描 500 个策略: 500 × 2 = 1000 微秒 = 1 毫秒  ❌ 不可接受
```

**早停优化效果有限**:
- 依赖策略紧凑存储（无空洞）
- 如果所有策略都匹配协议但不匹配 IP，仍需全扫描
- 最坏情况下仍是 O(n)

---

## 优化方案一：提高扫描上限

### 设计目标
在不引入复杂索引结构的前提下，通过优化 eBPF 代码复杂度，安全地提高 `MAX_WILDCARD_LOOP` 上限。

### 技术方案

#### 方案 1A: 渐进式提升（保守方案）

**核心思路**: 逐步提高循环上限，通过测试验证 eBPF 验证器接受程度。

**实施步骤**:

1. **代码优化**: 简化 `matches_wildcard()` 函数，减少每次迭代的指令数

   **当前代码**: `src/bpf/headers/policy_match.h:43-88`（45 行）

   **优化点**:
   ```c
   // Before: Multiple independent checks
   if (wildcard->direction != POLICY_DIR_ANY &&
       wildcard->direction != direction)
       return false;

   if (wildcard->src_port != 0 && key->src_port != wildcard->src_port)
       return false;

   // After: Combine checks to reduce branches
   if ((wildcard->direction != POLICY_DIR_ANY && wildcard->direction != direction) ||
       (wildcard->src_port != 0 && key->src_port != wildcard->src_port))
       return false;
   ```

2. **分阶段测试**:
   - Phase 1: 提升到 100 (2x)
   - Phase 2: 提升到 200 (4x)
   - Phase 3: 提升到 500 (10x)

   **验证方法**:
   ```bash
   # 编译并验证 eBPF 程序
   make clean && make

   # 检查验证器输出
   sudo bpftool prog load obj/tc_microsegment.bpf.o /sys/fs/bpf/test_prog type tc

   # 查看验证器统计
   sudo bpftool prog show | grep -A 5 tc_microsegment
   ```

3. **性能基准测试**:
   ```bash
   # 测试不同策略数量下的延迟
   go test -bench=BenchmarkWildcardLookup -benchmem
   ```

#### 方案 1B: 动态循环边界（激进方案）

**核心思路**: 使用有界循环（bounded loop）+ 动态边界检查，最大化利用容量。

**关键技术**:
```c
// Dynamic loop bound stored in eBPF map (controlled by user-space)
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 1);
    __type(key, __u32);
    __type(value, __u32);  // actual_wildcard_count
} wildcard_count_map SEC(".maps");

static __always_inline __u8 lookup_policy_action(...) {
    // ... exact match logic ...

    // Get actual wildcard policy count from map
    __u32 zero = 0;
    __u32 *count = bpf_map_lookup_elem(&wildcard_count_map, &zero);
    __u32 max_loop = count ? *count : MAX_WILDCARD_LOOP;

    // Clamp to absolute maximum (for verifier)
    if (max_loop > MAX_WILDCARD_LOOP)
        max_loop = MAX_WILDCARD_LOOP;

    // Bounded loop with dynamic limit
    #pragma unroll
    for (__u32 i = 0; i < MAX_WILDCARD_LOOP; i++) {
        if (i >= max_loop)  // Dynamic early exit
            break;

        // ... wildcard matching logic ...
    }
}
```

**优势**:
- 用户空间可以根据实际策略数量调整扫描范围
- 避免扫描空槽位，提升性能
- 与紧凑存储机制配合更高效

**劣势**:
- 每次查找需要额外的 map 查询（约 0.5 微秒）
- 代码复杂度增加

### 内核版本兼容性

| Linux 内核版本 | 指令数上限 | 预估支持的 MAX_WILDCARD_LOOP |
|--------------|-----------|----------------------------|
| 4.14 - 5.1   | 100,000   | 100-150                    |
| 5.2 - 5.9    | 1,000,000 | 500-1000                   |
| 5.10+        | 1,000,000 | 500-1000                   |

**检测方法**:
```bash
# 查看内核版本
uname -r

# 查看 eBPF 复杂度上限 (需要 bpftool)
sudo bpftool feature probe kernel | grep complexity
```

### 实现清单

**修改文件**:
1. `src/bpf/headers/policy_match.h`:
   - 修改 `MAX_WILDCARD_LOOP` 定义
   - 优化 `matches_wildcard()` 函数
   - （可选）添加 `wildcard_count_map`

2. `src/bpf/headers/common_types.h`:
   - （可选）添加 `wildcard_count_map` 声明

3. `src/agent/pkg/policy/policy.go`:
   - （可选）在 `addWildcardPolicy()` 中更新计数器
   - 添加 `CompactWildcardPolicies()` 函数

4. 测试文件:
   - 添加大规模通配符策略测试
   - 性能基准测试

**代码示例**:

```go
// src/agent/pkg/policy/policy.go

// CompactWildcardPolicies re-organizes wildcard policies to remove gaps
// This ensures eBPF can use early-stop optimization effectively
func (pm *PolicyManager) CompactWildcardPolicies() error {
    log.Info("Compacting wildcard policies...")

    // 1. Read all non-empty policies
    var activePolicies []struct {
        Index  uint32
        Policy WildcardPolicyStruct
    }

    for i := uint32(0); i < 1000; i++ {
        var existing WildcardPolicyStruct
        if err := pm.wildcardPolicyMap.Lookup(&i, &existing); err == nil {
            if existing.RuleID != 0 {
                activePolicies = append(activePolicies, struct{
                    Index  uint32
                    Policy WildcardPolicyStruct
                }{Index: i, Policy: existing})
            }
        }
    }

    log.Infof("Found %d active wildcard policies", len(activePolicies))

    // 2. Clear all slots
    zeroPolicy := WildcardPolicyStruct{}
    for i := uint32(0); i < 1000; i++ {
        pm.wildcardPolicyMap.Put(&i, &zeroPolicy)
    }

    // 3. Re-insert in compact form (starting from index 0)
    for idx, item := range activePolicies {
        newIdx := uint32(idx)
        if err := pm.wildcardPolicyMap.Put(&newIdx, &item.Policy); err != nil {
            log.Warnf("Failed to compact policy at index %d: %v", newIdx, err)
        }
    }

    log.Infof("Compacted %d wildcard policies to slots [0-%d]",
        len(activePolicies), len(activePolicies)-1)

    return nil
}
```

### 性能预期

**容量提升**:
| 方案 | 扫描上限 | 容量利用率 | 提升倍数 |
|-----|---------|-----------|---------|
| 当前 | 50 | 5% | 1x |
| Phase 1 | 100 | 10% | 2x |
| Phase 2 | 200 | 20% | 4x |
| Phase 3 | 500 | 50% | 10x |

**延迟影响**（假设每次匹配 2 微秒）:
| 策略数 | 平均扫描 | 平均延迟 | 最坏延迟 | 是否可接受 |
|-------|---------|---------|---------|----------|
| 50    | 25      | 50 μs   | 100 μs  | ✅ 优秀 |
| 100   | 50      | 100 μs  | 200 μs  | ✅ 良好 |
| 200   | 100     | 200 μs  | 400 μs  | ⚠️ 可接受 |
| 500   | 250     | 500 μs  | 1000 μs | ❌ 需配合方案二 |

**结论**: 单纯提高扫描上限到 200 是安全的，超过 200 需要配合分层索引优化。

---

## 优化方案二：分层索引架构

### 设计目标
通过引入分层索引结构，将通配符策略查找从 O(n) 降低到 O(log n) 或 O(1)，支持 500+ 通配符策略。

### 核心思想

**策略分类**: 根据通配符类型将策略分组到不同的索引桶

```
通配符策略类型分类:
├── L1: 按协议分类
│   ├── TCP only (protocol=6, src_port=*, dst_port=specific)
│   ├── UDP only (protocol=17, src_port=*, dst_port=specific)
│   ├── ANY protocol (protocol=0)
│   └── ... (最多 256 个协议)
│
├── L2: 按端口范围分类
│   ├── Well-known ports (0-1023)
│   ├── Registered ports (1024-49151)
│   ├── Dynamic ports (49152-65535)
│   └── ANY port (port=0)
│
└── L3: 按 CIDR 前缀长度分类
    ├── /32 (host specific)
    ├── /24 (subnet)
    ├── /16 (large subnet)
    └── /0 (any IP)
```

### 架构设计

#### 方案 2A: 协议优先索引（推荐）

**数据结构**:

```c
// Protocol-indexed wildcard policy buckets
// Each protocol has its own array of policies
struct {
    __uint(type, BPF_MAP_TYPE_HASH_OF_MAPS);
    __uint(max_entries, 256);  // 256 protocols (0-255)
    __type(key, __u8);         // protocol number
    __type(value, __u32);      // inner map ID
} protocol_index_map SEC(".maps");

// Inner map template: per-protocol wildcard policies
struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 100);  // 100 policies per protocol
    __type(key, __u32);
    __type(value, struct wildcard_policy);
} protocol_bucket_template SEC(".maps");

// Metadata: track how many policies per protocol
struct protocol_metadata {
    __u32 policy_count;        // Number of policies for this protocol
    __u32 reserved[3];
};

struct {
    __uint(type, BPF_MAP_TYPE_ARRAY);
    __uint(max_entries, 256);
    __type(key, __u8);         // protocol number
    __type(value, struct protocol_metadata);
} protocol_metadata_map SEC(".maps");
```

**查找流程**:

```c
static __always_inline __u8 lookup_policy_action_v2(
    struct flow_key *key,
    __u8 direction,
    __u32 *rule_id)
{
    // Fast path: exact match (unchanged)
    __u8 action = lookup_exact_policy(key, direction, rule_id);
    if (*rule_id != 0)
        return action;

    // Slow path v2: Protocol-indexed wildcard lookup

    // Step 1: Get metadata for this protocol
    __u8 proto = key->protocol;
    struct protocol_metadata *meta =
        bpf_map_lookup_elem(&protocol_metadata_map, &proto);

    if (!meta || meta->policy_count == 0) {
        // No policies for this protocol, try ANY protocol (0)
        proto = 0;
        meta = bpf_map_lookup_elem(&protocol_metadata_map, &proto);
        if (!meta || meta->policy_count == 0) {
            // No matching policies
            update_stats(STATS_POLICY_MISSES);
            return POLICY_ACTION_ALLOW;
        }
    }

    // Step 2: Get the policy bucket for this protocol
    void *bucket_map = bpf_map_lookup_elem(&protocol_index_map, &proto);
    if (!bucket_map) {
        update_stats(STATS_POLICY_MISSES);
        return POLICY_ACTION_ALLOW;
    }

    // Step 3: Scan policies within this protocol bucket (much smaller!)
    struct wildcard_policy *best_match = NULL;
    __u8 best_priority = 0;

    __u32 max_scan = meta->policy_count;
    if (max_scan > 100)
        max_scan = 100;  // Cap at bucket size

    #pragma unroll
    for (__u32 i = 0; i < 100; i++) {  // Reduced loop size!
        if (i >= max_scan)
            break;

        struct wildcard_policy *wildcard =
            bpf_map_lookup_elem(bucket_map, &i);

        if (!wildcard || wildcard->rule_id == 0)
            break;

        if (matches_wildcard(key, wildcard, direction)) {
            if (!best_match || wildcard->priority > best_priority) {
                best_match = wildcard;
                best_priority = wildcard->priority;
            }
        }
    }

    if (best_match) {
        update_stats(STATS_POLICY_HITS);
        *rule_id = best_match->rule_id;
        return best_match->action;
    }

    // No match
    update_stats(STATS_POLICY_MISSES);
    return POLICY_ACTION_ALLOW;
}
```

**性能分析**:

假设有 500 个通配符策略，分布如下:
- TCP 协议: 300 个策略
- UDP 协议: 150 个策略
- ANY 协议: 50 个策略

**当前方案**:
- 扫描 500 个策略（但受限于 MAX_WILDCARD_LOOP=50，实际只扫描 50 个）
- 平均延迟: 50 次比较 × 2 μs = **100 μs**

**优化后方案**:
- 对于 TCP 流，只扫描 TCP 桶（300 个策略）+ ANY 桶（50 个策略）
- 但每个桶最多扫描 100 个
- 实际扫描: 100 (TCP 桶) + 50 (ANY 桶) = 150 次比较
- 平均延迟: 150 次 × 2 μs = **300 μs**

**优化效果**:
- ✅ 容量提升: 50 → 500 (10x)
- ⚠️ 延迟增加: 100 μs → 300 μs (3x)
- 💡 需要进一步优化: 缩小桶大小

#### 方案 2B: 多级索引（高级方案）

**核心思想**: 协议 × 端口范围 的二维索引

```
索引键 = (protocol, dst_port_bucket)

其中 dst_port_bucket:
- 0: port=0 (ANY)
- 1: port=1-1023 (well-known)
- 2: port=1024-49151 (registered)
- 3: port=49152-65535 (dynamic)
```

**数据结构**:

```c
// Composite index key: protocol + port bucket
struct composite_key {
    __u8  protocol;        // 0-255
    __u8  dst_port_bucket; // 0-3
    __u16 pad;
} __attribute__((packed));

// Two-level index
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __uint(max_entries, 1024);  // 256 protocols × 4 buckets
    __type(key, struct composite_key);
    __type(value, struct bucket_pointer);  // Pointer to policy array
} composite_index_map SEC(".maps");

struct bucket_pointer {
    __u32 start_idx;  // Starting index in global policy array
    __u32 count;      // Number of policies in this bucket
};
```

**查找流程**:

```c
static inline __u8 get_port_bucket(__u16 port) {
    if (port == 0) return 0;  // ANY
    if (port <= 1023) return 1;  // well-known
    if (port <= 49151) return 2; // registered
    return 3;  // dynamic
}

static __always_inline __u8 lookup_policy_action_v3(...) {
    // ... exact match ...

    // Build composite key
    struct composite_key ckey = {
        .protocol = key->protocol,
        .dst_port_bucket = get_port_bucket(key->dst_port),
        .pad = 0,
    };

    // Lookup bucket
    struct bucket_pointer *bucket =
        bpf_map_lookup_elem(&composite_index_map, &ckey);

    if (!bucket) {
        // Try ANY protocol bucket
        ckey.protocol = 0;
        bucket = bpf_map_lookup_elem(&composite_index_map, &ckey);
    }

    if (!bucket || bucket->count == 0) {
        return POLICY_ACTION_ALLOW;  // No match
    }

    // Scan only policies in this bucket (much smaller!)
    __u32 max_scan = bucket->count;
    if (max_scan > 50)  // Cap to keep verifier happy
        max_scan = 50;

    #pragma unroll
    for (__u32 i = 0; i < 50; i++) {
        if (i >= max_scan)
            break;

        __u32 idx = bucket->start_idx + i;
        // ... lookup and match ...
    }

    return POLICY_ACTION_ALLOW;
}
```

**性能分析**:

假设 500 个策略均匀分布在 256 × 4 = 1024 个桶中:
- 平均每个桶: 500 / 1024 ≈ **0.5 个策略**
- 最坏情况（热门桶，如 TCP:80）: 假设 50 个策略

**查找延迟**:
- 索引查找: 1 次 hash lookup ≈ **0.5 μs**
- 桶内扫描: 平均 0.5 个策略 × 2 μs ≈ **1 μs**
- **总延迟: 1.5 μs** ✅ 极优

**极端情况**（所有策略都是 TCP:80）:
- 索引查找: **0.5 μs**
- 桶内扫描: 50 个策略 × 2 μs = **100 μs**
- **总延迟: 100.5 μs** ✅ 仍可接受

### 用户空间实现

**策略插入时的索引维护**:

```go
// src/agent/pkg/policy/indexed_policy_manager.go

type IndexedPolicyManager struct {
    *PolicyManager
    protocolIndex map[uint8]*ProtocolBucket
}

type ProtocolBucket struct {
    Protocol uint8
    Policies []*wildcard_policy
    MapID    uint32  // eBPF inner map ID
}

func (ipm *IndexedPolicyManager) AddWildcardPolicyIndexed(p *Policy) error {
    // Parse policy
    proto := parseProtocol(p.Protocol)

    // Get or create protocol bucket
    bucket, exists := ipm.protocolIndex[proto]
    if !exists {
        bucket = &ProtocolBucket{
            Protocol: proto,
            Policies: make([]*wildcard_policy, 0, 100),
        }

        // Create inner map for this protocol
        innerMapSpec := &ebpf.MapSpec{
            Type:       ebpf.Array,
            KeySize:    4,
            ValueSize:  uint32(unsafe.Sizeof(wildcard_policy{})),
            MaxEntries: 100,
        }
        innerMap, err := ebpf.NewMap(innerMapSpec)
        if err != nil {
            return fmt.Errorf("failed to create inner map: %w", err)
        }
        bucket.MapID = uint32(innerMap.FD())

        // Add to protocol index map
        if err := ipm.protocolIndexMap.Put(&proto, &bucket.MapID); err != nil {
            return fmt.Errorf("failed to add to protocol index: %w", err)
        }

        ipm.protocolIndex[proto] = bucket
    }

    // Check bucket capacity
    if len(bucket.Policies) >= 100 {
        return fmt.Errorf("protocol bucket %d is full (max 100 policies)", proto)
    }

    // Add policy to bucket
    wildcard := buildWildcardPolicy(p)
    bucket.Policies = append(bucket.Policies, wildcard)

    // Update eBPF map
    idx := uint32(len(bucket.Policies) - 1)
    innerMapFD := int(bucket.MapID)
    innerMap := ebpf.NewMapFromFD(innerMapFD)
    if err := innerMap.Put(&idx, wildcard); err != nil {
        return fmt.Errorf("failed to add to bucket map: %w", err)
    }

    // Update metadata
    meta := &protocol_metadata{
        policy_count: uint32(len(bucket.Policies)),
    }
    if err := ipm.protocolMetadataMap.Put(&proto, meta); err != nil {
        return fmt.Errorf("failed to update metadata: %w", err)
    }

    log.Infof("Added wildcard policy to protocol %d bucket (index %d)", proto, idx)
    return nil
}
```

### 实现复杂度评估

| 方案 | eBPF 代码复杂度 | 用户空间复杂度 | 内存开销 | 维护成本 |
|-----|----------------|---------------|---------|---------|
| 方案 2A (协议索引) | 中 (新增 200 行) | 中 (新增 300 行) | 中 (+256 maps) | 中 |
| 方案 2B (多级索引) | 高 (新增 300 行) | 高 (新增 500 行) | 高 (+1024 buckets) | 高 |

**推荐**: 先实现方案 2A，验证效果后考虑是否需要方案 2B。

---

## 性能影响分析

### 基准测试设计

```go
// src/agent/pkg/policy/benchmark_test.go

func BenchmarkWildcardLookup(b *testing.B) {
    testCases := []struct {
        name           string
        policyCount    int
        matchPosition  string  // "first", "middle", "last", "none"
    }{
        {"50_policies_first_match", 50, "first"},
        {"50_policies_last_match", 50, "last"},
        {"100_policies_middle_match", 100, "middle"},
        {"200_policies_none_match", 200, "none"},
        {"500_policies_first_match", 500, "first"},
    }

    for _, tc := range testCases {
        b.Run(tc.name, func(b *testing.B) {
            // Setup: insert policies
            pm := setupPolicyManager()
            insertWildcardPolicies(pm, tc.policyCount)

            // Prepare test packet
            flowKey := generateTestFlow(tc.matchPosition)

            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                // Simulate eBPF lookup
                _ = lookupPolicyAction(flowKey)
            }
        })
    }
}
```

### 预期性能对比表

| 方案 | 最大策略数 | 平均查找延迟 | P99 延迟 | 内存占用 |
|-----|-----------|------------|---------|---------|
| **当前** | 50 | 50 μs | 100 μs | 40 KB |
| **方案 1A (Loop=200)** | 200 | 200 μs | 400 μs | 160 KB |
| **方案 1B (动态边界)** | 200 | 100 μs | 200 μs | 164 KB |
| **方案 2A (协议索引)** | 500 | 20 μs | 100 μs | 200 KB |
| **方案 2B (多级索引)** | 1000 | 5 μs | 50 μs | 500 KB |

**注**: 延迟数据基于理论计算，实际需要基准测试验证。

### 吞吐量影响

**会话缓存减轻影响**:
- 99%+ 数据包使用会话缓存，延迟 < 1 微秒
- 只有第一个数据包会经历通配符查找
- 假设平均会话持续时间 100 个数据包

**整体影响计算**:
```
假设:
- 数据包速率: 100,000 pps
- 新会话率: 1,000 sessions/s (1%)
- 通配符查找率: 10% (其他 90% 有精确匹配)

当前方案 (50 μs):
  新会话处理时间 = 1,000 × 0.1 × 50 μs = 5 ms/s = 0.5% CPU

方案 2A (20 μs):
  新会话处理时间 = 1,000 × 0.1 × 20 μs = 2 ms/s = 0.2% CPU

优化收益: 60% CPU 减少 ✅
```

---

## 实施计划

### 阶段划分

#### Phase 1: 基础优化（2-3 天）
**目标**: 提升扫描上限到 100-200

**任务**:
1. ✅ 代码审计: 分析 `matches_wildcard()` 指令开销
2. ✅ 优化 eBPF 代码: 减少分支，合并检查
3. ✅ 渐进式测试: 50 → 100 → 150 → 200
4. ✅ 验证器兼容性测试: 多内核版本验证
5. ✅ 性能基准测试: 对比延迟变化

**交付物**:
- ✅ 更新的 `policy_match.h` (MAX_WILDCARD_LOOP=200)
- ✅ 验证报告 (`docs/phase1-validation-report.md`)
- ✅ 性能测试结果

#### Phase 2: 协议索引（1 周）
**目标**: 实现方案 2A（协议优先索引）

**任务**:
1. ✅ 设计索引数据结构
2. ✅ eBPF 实现:
   - 新增 `protocol_index_map`
   - 新增 `protocol_metadata_map`
   - 修改 `lookup_policy_action()`
3. ✅ 用户空间实现:
   - 新增 `IndexedPolicyManager`
   - 策略插入/删除索引维护
4. ✅ 单元测试 + 集成测试
5. ✅ E2E 测试: 500 个通配符策略场景

**交付物**:
- ✅ 新文件: `src/bpf/headers/indexed_policy_match.h`
- ✅ 新文件: `src/agent/pkg/policy/indexed_policy_manager.go`
- ✅ 更新文档: `openspec/specs/policy-matching/spec.md`

#### Phase 3: 性能优化与调优（3-5 天）
**目标**: 精细调优，达到最佳性能

**任务**:
1. ✅ 压力测试: 1000+ 策略场景
2. ✅ 热点分析: 使用 `perf` 分析瓶颈
3. ✅ 策略分布优化: 自动重平衡桶
4. ✅ 内存优化: 减少 map 内存占用
5. ✅ 监控仪表板: 展示索引命中率

**交付物**:
- ✅ 性能调优报告
- ✅ 最佳实践文档
- ✅ Grafana 监控面板

#### Phase 4: 生产部署（1 周）
**目标**: 金丝雀发布，逐步推广

**任务**:
1. ✅ 功能开关: 支持新旧版本切换
2. ✅ 灰度发布: 10% → 50% → 100%
3. ✅ 监控告警: 实时监控异常
4. ✅ 回滚方案: 快速回退机制
5. ✅ 文档更新: 用户手册 + 运维指南

**交付物**:
- ✅ 生产部署清单
- ✅ 监控告警规则
- ✅ 用户迁移指南

### 里程碑

| 里程碑 | 日期 | 关键指标 |
|-------|------|---------|
| M1: 扫描上限达到 200 | Week 1 | MAX_WILDCARD_LOOP=200 验证通过 |
| M2: 协议索引 MVP | Week 2 | 支持 500 个策略，平均延迟 < 50 μs |
| M3: 性能优化完成 | Week 3 | P99 延迟 < 100 μs, 内存 < 500 KB |
| M4: 生产就绪 | Week 4 | 金丝雀部署成功，无回滚 |

---

## 风险评估

### 技术风险

#### 风险 1: eBPF 验证器拒绝
**概率**: 中
**影响**: 高
**缓解措施**:
- 渐进式提升循环上限，每次提升后验证
- 准备多内核版本测试环境
- 保留降级方案（回退到 MAX_WILDCARD_LOOP=50）

#### 风险 2: 性能回退
**概率**: 低
**影响**: 高
**缓解措施**:
- 严格的性能基准测试（自动化 CI）
- P99 延迟告警阈值（超过 200 μs 自动告警）
- 功能开关快速回滚

#### 风险 3: 内存占用过高
**概率**: 中
**影响**: 中
**缓解措施**:
- 内存使用监控（prometheus metrics）
- 动态调整桶大小（根据实际策略分布）
- 支持配置最大内存上限

### 运维风险

#### 风险 4: 索引维护复杂度增加
**概率**: 高
**影响**: 中
**缓解措施**:
- 自动化索引重建工具
- 索引一致性检查（定期扫描）
- 详细的运维文档和故障排查指南

#### 风险 5: 升级兼容性
**概率**: 中
**影响**: 中
**缓解措施**:
- 数据格式版本化
- 平滑升级脚本（自动迁移旧数据）
- 支持新旧版本共存（过渡期）

### 风险矩阵

| 风险 | 概率 | 影响 | 优先级 | 缓解状态 |
|-----|------|------|-------|---------|
| eBPF 验证器拒绝 | 中 | 高 | P0 | ✅ 已规划 |
| 性能回退 | 低 | 高 | P0 | ✅ 已规划 |
| 内存占用过高 | 中 | 中 | P1 | ✅ 已规划 |
| 索引维护复杂 | 高 | 中 | P1 | ✅ 已规划 |
| 升级兼容性 | 中 | 中 | P2 | ✅ 已规划 |

---

## 参考资料

### 相关文档
1. [通配符策略实现提案](../openspec/changes/archive/2025-11-02-add-wildcard-policy-matching/proposal.md)
2. [策略匹配规范](../openspec/specs/policy-matching/spec.md)
3. [性能优化报告](./performance.md)

### 代码位置
1. eBPF 通配符匹配: `src/bpf/headers/policy_match.h`
2. 通配符策略结构: `src/bpf/headers/common_types.h:127-146`
3. Go 策略管理器: `src/agent/pkg/policy/policy.go`

### 外部资料
1. [eBPF 验证器限制](https://www.kernel.org/doc/html/latest/bpf/verifier.html)
2. [BPF Map Types](https://docs.kernel.org/bpf/maps.html)
3. [Cilium eBPF 最佳实践](https://docs.cilium.io/en/stable/bpf/)

---

## 附录

### A. 性能测试脚本

```bash
#!/bin/bash
# Performance benchmark for wildcard policy optimization

set -e

echo "=== Wildcard Policy Performance Test ==="

# Test configurations
POLICY_COUNTS=(50 100 200 500 1000)
MAX_WILDCARD_LOOPS=(50 100 200 500)

for loop in "${MAX_WILDCARD_LOOPS[@]}"; do
    echo ""
    echo "Testing MAX_WILDCARD_LOOP=$loop"

    # Update eBPF code
    sed -i "s/#define MAX_WILDCARD_LOOP.*/#define MAX_WILDCARD_LOOP $loop/" \
        src/bpf/headers/policy_match.h

    # Rebuild
    make clean && make || {
        echo "❌ Build failed with MAX_WILDCARD_LOOP=$loop"
        continue
    }

    for count in "${POLICY_COUNTS[@]}"; do
        if [ $count -gt $loop ]; then
            echo "⏭️  Skipping $count policies (exceeds loop limit)"
            continue
        fi

        echo "  Testing with $count policies..."

        # Run benchmark
        go test -bench=BenchmarkWildcardLookup/$count \
                -benchtime=10s \
                -benchmem \
                -cpuprofile=cpu_${loop}_${count}.prof \
                ./pkg/policy/
    done
done

echo ""
echo "✅ All tests completed. Results saved to *.prof files"
```

### B. 监控指标定义

```yaml
# Prometheus metrics for wildcard policy performance

metrics:
  - name: wildcard_policy_lookup_duration_microseconds
    type: histogram
    help: "Wildcard policy lookup latency in microseconds"
    buckets: [1, 5, 10, 20, 50, 100, 200, 500, 1000]
    labels:
      - protocol
      - match_result  # "hit" or "miss"

  - name: wildcard_policy_scan_count
    type: histogram
    help: "Number of policies scanned per lookup"
    buckets: [1, 10, 25, 50, 100, 200, 500]

  - name: wildcard_policy_bucket_utilization
    type: gauge
    help: "Number of policies in each protocol bucket"
    labels:
      - protocol

  - name: wildcard_policy_index_hit_rate
    type: gauge
    help: "Protocol index hit rate (0.0-1.0)"
```

### C. 故障排查清单

**问题**: eBPF 程序加载失败

```bash
# Step 1: Check verifier log
sudo bpftool prog load obj/tc_microsegment.bpf.o /sys/fs/bpf/test type tc 2>&1 | tail -100

# Step 2: Check instruction count
sudo bpftool prog show | grep tc_microsegment

# Step 3: Reduce MAX_WILDCARD_LOOP
sed -i 's/MAX_WILDCARD_LOOP 200/MAX_WILDCARD_LOOP 50/' src/bpf/headers/policy_match.h
make clean && make

# Step 4: Check kernel version compatibility
uname -r
```

**问题**: 通配符策略不生效

```bash
# Step 1: Check policy insertion
sudo bpftool map dump name wildcard_policy_map | head -50

# Step 2: Check metadata
sudo bpftool map dump name protocol_metadata_map

# Step 3: Enable eBPF debug output
echo 1 > /sys/kernel/debug/tracing/events/bpf_trace/bpf_trace_printk/enable
sudo cat /sys/kernel/debug/tracing/trace_pipe | grep wildcard

# Step 4: Verify policy compaction
curl -X POST http://localhost:8080/api/v1/policies/compact
```

---

**文档结束**

如有疑问或建议，请联系开发团队或提交 Issue。
