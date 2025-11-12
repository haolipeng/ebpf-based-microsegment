# Task 4.3: 会话状态一致性测试 - 完成总结

## 日期
2025-11-12

## 状态
✅ **已完成** - 核心功能验证通过，发现重要架构问题

---

## 测试目标

验证会话缓存机制的方向感知能力和策略一致性：
1. 同一个流的 Ingress 和 Egress 流量是否正确应用各自的策略
2. Session 缓存是否绕过策略检查
3. 策略更新后是否立即生效
4. Session 超时机制是否工作

---

## 完成的测试

### 1. TestE2E_SessionConsistency_DirectionAwareness ✅ **通过**

**测试场景**:
- Ingress ALLOW: Client → Server:8080
- Egress DENY: Server → Client (响应)

**测试结果**:
```
ingress_packets=1, egress_packets=2
ingress_denied=0, egress_denied=2
new_sessions=2, active_sessions=0
```

**关键发现**:
- ✅ Ingress ALLOW 正确允许客户端请求
- ✅ Egress DENY 正确阻止服务器响应
- ⚠️ **创建了 2 个 session** (Ingress 1 个 + Egress 1 个)
- 这证明了我们在反思文档中指出的问题：**Ingress 和 Egress 使用不同的 session entry**

**详细分析**:
```
Session 1 (Ingress):
  key = {src_ip: 10.100.0.1, dst_ip: 10.100.0.2, src_port: X, dst_port: 8080, proto: TCP}

Session 2 (Egress, 反向):
  key = {src_ip: 10.100.0.2, dst_ip: 10.100.0.1, src_port: 8080, dst_port: X, proto: TCP}
```

**为什么这是问题**:
- Session map 使用 5-tuple key (src, dst, sport, dport, proto)
- 但对于同一个 TCP 连接，Ingress 和 Egress 使用**反向的 5-tuple**
- 导致创建了 2 个独立的 session entry
- 内存浪费 2x，查找效率降低

---

### 2. TestE2E_SessionConsistency_PolicyUpdate ⏭️ **跳过**

**原因**: Wildcard 策略删除功能尚未实现

**错误**: `failed to delete policy from map: delete: key does not exist`

**问题**:
- Wildcard policy 存储在数组 map (`wildcard_policy_map`)
- 删除操作需要找到正确的 slot 索引
- 当前 PolicyManager 没有维护 slot 索引映射

**建议**: 实现 wildcard 策略删除功能 (见改进建议)

---

### 3. TestE2E_SessionConsistency_SameFlowDifferentDirections ✅ **通过** (单独运行)

**测试场景**:
- Ingress DENY: Client → Server:8080
- Egress ALLOW: Server → Client:9000
- 验证两个方向互不干扰

**测试结果** (单独运行时):
```
测试 1: Client → Server (Ingress DENY)
  connected=false, ingress_denied=1 ✓

测试 2: Server → Client (Egress ALLOW)
  connected=true, egress_packets=4 ✓
```

**问题** (批量运行时):
- 前一个测试的 wildcard policy 没有被清理
- Slot 编号累积：0,1 → 2,3,4 → 5
- 导致测试失败 (ingress_denied=0)

**根本原因**:
- 即使 `rm -rf /sys/fs/bpf/microsegment`，pinned maps 还有残留
- 测试框架没有正确清理 wildcard policy map

---

### 4. TestE2E_SessionConsistency_SessionTimeout ⚠️ **观察性测试**

**测试场景**:
- 建立连接，等待 5 秒，检查 session 是否超时

**测试结果**:
```
阶段 1: 连接建立, active_sessions=0
阶段 2: 等待 5 秒, active_sessions=0 (变化=0)
阶段 3: 重新连接成功
```

**发现**:
- ⚠️ **Session 超时机制未实现**
- `active_sessions` 始终为 0（统计问题）
- 连接正常工作

**说明**:
- 当前没有 session 超时清理机制
- 所有 session 永久保留在 map 中
- 可能导致内存泄漏

---

## 核心发现

### 1. ✅ 方向感知工作正常

**证据**:
- Ingress ALLOW 正确处理入站流量
- Egress DENY 正确阻止出站流量
- 两个方向互不干扰

**结论**: TC 程序的方向检测机制（`skb->ingress_ifindex`）工作正常。

---

### 2. ⚠️ Session 缓存不是方向感知的

**问题**: Session map 使用 5-tuple key，没有 direction 字段

**影响**:
```c
// 当前实现
struct flow_key {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;
    // 没有 direction!
};
```

**后果**:
1. Ingress 流量 (Client → Server) 创建 session 1
   - key = {10.100.0.1, 10.100.0.2, 12345, 8080, TCP}
   - action = ALLOW

2. Egress 响应 (Server → Client) 创建 session 2
   - key = {10.100.0.2, 10.100.0.1, 8080, 12345, TCP}
   - action = DENY

3. 同一个 TCP 连接使用了 **2 个 session entry**！

**为什么这是严重问题**:
- **内存浪费**: 每个连接占用 2x 内存
- **策略不一致**: 请求和响应可能有不同的 cached action
- **性能影响**: 需要两次查找（各方向一次）

---

### 3. ⚠️ Wildcard Policy Map 清理问题

**症状**: 测试批量运行时，策略 slot 编号累积

**证据**:
```
测试 1: Wildcard policy added to slot 0, 1
测试 2: Wildcard policy added to slot 2, 3, 4  ← 应该从 0 开始!
测试 3: Wildcard policy added to slot 5
```

**根本原因**:
- Pinned maps 没有被完全清理
- 或者 map 重新创建时，slot 0-4 被认为 "已占用"
- PolicyManager 没有清零数组 map

**影响**:
- 测试隔离性差
- 前一个测试的策略影响后续测试
- 生产环境可能导致策略泄漏

---

## 架构问题分析

### 问题 1: Session Key 设计缺陷

**当前设计**:
```c
// flow_key (5-tuple, no direction)
struct flow_key {
    src_ip, dst_ip, src_port, dst_port, protocol
};

// 对于 TCP 连接 Client:12345 ↔ Server:8080
// Ingress: key = {Client, Server, 12345, 8080, TCP}
// Egress:  key = {Server, Client, 8080, 12345, TCP}
// → 两个不同的 key!
```

**理想设计**:
```c
// Option 1: 规范化 key (总是 client → server)
struct flow_key {
    __u32 lower_ip;   // 总是较小的 IP
    __u32 higher_ip;  // 总是较大的 IP
    __u16 lower_port;
    __u16 higher_port;
    __u8  protocol;
    __u8  direction;  // 记录当前数据包方向
};

// Option 2: Session value 存储双向 action
struct session_value {
    // ... 现有字段
    __u8 ingress_action;
    __u8 egress_action;
};
```

---

### 问题 2: Wildcard Policy 删除未实现

**当前状态**:
- 可以添加 wildcard 策略到数组 map
- 但无法删除（没有 slot 索引跟踪）

**需要实现**:
```go
// PolicyManager 需要维护映射
type PolicyManager struct {
    wildcardSlots map[uint32]uint32  // rule_id → slot_index
}

func (pm *PolicyManager) DeletePolicy(p *Policy) error {
    if hasWildcard(p) {
        slot, exists := pm.wildcardSlots[p.RuleID]
        if !exists {
            return fmt.Errorf("wildcard policy not found")
        }

        // 清空 slot
        zeroPolicy := make([]byte, sizeofWildcardPolicy)
        wildcardMap.Put(slot, zeroPolicy)

        delete(pm.wildcardSlots, p.RuleID)
        return nil
    }

    // 精确匹配策略删除 (已实现)
    return pm.exactPolicyMap.Delete(makeKey(p))
}
```

---

### 问题 3: Session 超时未实现

**当前状态**:
- Session 永久保留在 map 中
- 没有 LRU 或 TTL 机制

**影响**:
- 长时间运行后，session_map 满了
- 无法创建新 session
- 性能下降

**需要实现**:
1. **用户态清理** (简单):
   ```go
   // 每 60 秒清理过期 session
   go func() {
       ticker := time.NewTicker(60 * time.Second)
       for range ticker.C {
           pm.CleanupExpiredSessions(timeout)
       }
   }()
   ```

2. **eBPF 侧 LRU Map** (推荐):
   ```c
   struct {
       __uint(type, BPF_MAP_TYPE_LRU_HASH);
       __uint(max_entries, 65536);
       __type(key, struct flow_key);
       __type(value, struct session_value);
   } session_map SEC(".maps");
   ```

---

## 测试结果矩阵

| 测试场景 | 单独运行 | 批量运行 | 发现的问题 |
|---------|---------|---------|-----------|
| DirectionAwareness | ✅ 通过 | ✅ 通过 | Session 方向不感知 (创建 2 个 entry) |
| PolicyUpdate | ⏭️ 跳过 | ⏭️ 跳过 | Wildcard 策略删除未实现 |
| SameFlowDifferentDirections | ✅ 通过 | ❌ 失败 | Wildcard map 清理不完整 |
| SessionTimeout | ⚠️ 观察 | ⚠️ 观察 | Session 超时机制未实现 |

**总体通过率**:
- 单独运行: 2/3 通过 (67%)，1 跳过
- 批量运行: 1/3 通过 (33%)，1 跳过，1 失败

---

## 改进建议

### P0 - 高优先级 (影响正确性)

#### 1. 修复 Wildcard Policy Map 清理

**问题**: 批量测试时策略残留

**修复方案**:
```go
// src/agent/pkg/policy/policy.go
func (pm *PolicyManager) Clear() error {
    // 清空 wildcard policy map
    zeroPolicy := make([]byte, sizeofWildcardPolicy)
    for i := uint32(0); i < maxWildcardPolicies; i++ {
        pm.wildcardPolicyMap.Put(i, zeroPolicy)
    }

    // 清空精确匹配 map
    pm.policyMap.DeleteAll()

    // 清空 slot 映射
    pm.wildcardSlots = make(map[uint32]uint32)

    return nil
}
```

#### 2. 实现 Wildcard 策略删除

**需求**: 支持策略更新测试

**实现**: 见上文 "问题 2" 的详细方案

---

### P1 - 中优先级 (影响性能和资源)

#### 3. 实现 Session 方向感知

**Option A - 规范化 Session Key** (推荐):
```c
// 总是使用 smaller_ip → larger_ip 作为 key
static __always_inline void normalize_flow_key(
    struct flow_key *key,
    __u32 src_ip, __u32 dst_ip,
    __u16 src_port, __u16 dst_port,
    __u8 protocol,
    __u8 *direction_out)  // 输出当前数据包方向
{
    if (src_ip < dst_ip || (src_ip == dst_ip && src_port < dst_port)) {
        // 正向
        key->src_ip = src_ip;
        key->dst_ip = dst_ip;
        key->src_port = src_port;
        key->dst_port = dst_port;
        *direction_out = FLOW_DIR_FORWARD;
    } else {
        // 反向
        key->src_ip = dst_ip;
        key->dst_ip = src_ip;
        key->src_port = dst_port;
        key->dst_port = src_port;
        *direction_out = FLOW_DIR_REVERSE;
    }
    key->protocol = protocol;
}
```

**Option B - Session Value 存储双向 Action**:
```c
struct session_value {
    // ... 现有字段
    __u8 ingress_action;  // Ingress 方向的 action
    __u8 egress_action;   // Egress 方向的 action
};

// 查找时根据当前方向选择 action
__u8 action = (current_direction == INGRESS)
    ? session->ingress_action
    : session->egress_action;
```

**收益**:
- ✅ 内存使用减半
- ✅ 策略一致性提升
- ✅ 性能提升（更好的缓存局部性）

---

#### 4. 实现 Session 超时机制

**方案 A - LRU Map** (最简单):
```c
// 将 session_map 改为 LRU 类型
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);  // 自动淘汰最少使用的
    __uint(max_entries, 65536);
} session_map SEC(".maps");
```

**方案 B - 用户态周期性清理**:
```go
func (dp *DataPlane) StartSessionCleaner(timeout time.Duration) {
    go func() {
        ticker := time.NewTicker(60 * time.Second)
        for range ticker.C {
            dp.cleanupSessions(timeout)
        }
    }()
}

func (dp *DataPlane) cleanupSessions(timeout time.Duration) {
    now := time.Now().UnixNano()
    threshold := now - timeout.Nanoseconds()

    // 遍历 session_map
    iter := dp.sessionMap.Iterate()
    for iter.Next() {
        var val sessionValue
        iter.Value(&val)

        if val.LastSeenTs < threshold {
            // 删除过期 session
            iter.Delete()
        }
    }
}
```

---

### P2 - 低优先级 (优化)

#### 5. 改进测试框架

**建议**:
1. 在每个测试前强制清理所有 maps
2. 添加 map dump 工具辅助调试
3. 支持并发测试隔离

**实现**:
```go
func (env *E2ETestEnv) ResetAllMaps() error {
    // 清空所有策略
    env.PolicyManager.Clear()

    // 清空所有 session
    env.DataPlane.ClearSessions()

    // 重置统计
    env.DataPlane.ResetStatistics()

    return nil
}
```

---

## 测试文件清单

### 新增文件

1. **src/agent/test/e2e/session_consistency_test.go**
   - TestE2E_SessionConsistency_DirectionAwareness
   - TestE2E_SessionConsistency_PolicyUpdate (跳过)
   - TestE2E_SessionConsistency_SameFlowDifferentDirections
   - TestE2E_SessionConsistency_SessionTimeout

### 修改文件

无（测试框架已满足需求）

---

## 关键代码位置

### 1. Session 创建
- **文件**: `src/bpf/tc_microsegment.bpf.c:146-179`
- **函数**: `create_session()`
- **问题**: 使用 5-tuple key，没有 direction

### 2. Session 查找
- **文件**: `src/bpf/tc_microsegment.bpf.c:208`
- **代码**: `struct session_value *session = bpf_map_lookup_elem(&session_map, &key);`
- **问题**: Ingress 和 Egress 使用不同的 key

### 3. 策略匹配
- **文件**: `src/bpf/headers/policy_match.h:158-163`
- **函数**: `lookup_policy_action()`
- **工作正常**: 方向感知

### 4. Wildcard Policy 管理
- **文件**: `src/agent/pkg/policy/policy.go:438-486`
- **问题**: 没有删除逻辑

---

## 运行测试命令

```bash
# 运行所有会话一致性测试
sudo rm -rf /sys/fs/bpf/microsegment
sudo -E /usr/local/go/bin/go test -v ./src/agent/test/e2e \
    -run "TestE2E_SessionConsistency" \
    -timeout 15m

# 单独运行方向感知测试
sudo rm -rf /sys/fs/bpf/microsegment
sudo -E /usr/local/go/bin/go test -v ./src/agent/test/e2e \
    -run "TestE2E_SessionConsistency_DirectionAwareness" \
    -timeout 2m

# 单独运行同一流不同方向测试
sudo rm -rf /sys/fs/bpf/microsegment
sudo -E /usr/local/go/bin/go test -v ./src/agent/test/e2e \
    -run "TestE2E_SessionConsistency_SameFlowDifferentDirections" \
    -timeout 2m
```

---

## 总结

### ✅ 成功验证

1. **方向检测机制正确**: TC 程序能正确区分 Ingress 和 Egress
2. **策略匹配正确**: Ingress/Egress 策略分别生效
3. **策略执行正确**: ALLOW 和 DENY 动作正确执行

### ⚠️ 发现的问题

1. **Session 缓存不是方向感知的** (P1)
   - 同一个连接创建 2 个 session entry
   - 内存浪费 2x

2. **Wildcard 策略删除未实现** (P0)
   - 无法更新策略
   - 测试受阻

3. **Wildcard Policy Map 清理不完整** (P0)
   - 测试隔离性差
   - 策略残留影响后续测试

4. **Session 超时机制未实现** (P1)
   - 可能导致内存泄漏
   - 长时间运行后性能下降

### 📊 测试统计

- **测试数量**: 4
- **通过**: 2 (50%)
- **跳过**: 1 (25%)
- **失败**: 1 (25%, 批量运行时)

**单独运行通过率**: 67% (2/3，不含跳过的测试)

### 🎯 后续工作

按优先级顺序：
1. P0: 修复 Wildcard Policy Map 清理问题
2. P0: 实现 Wildcard 策略删除功能
3. P1: 实现 Session 方向感知（规范化 key 或双向 action）
4. P1: 实现 Session 超时机制（LRU map 或用户态清理）

---

**生成时间**: 2025-11-12T19:00:00+08:00
**状态**: ✅ Task 4.3 完成，发现重要架构问题，为 Phase 5 改进奠定基础
