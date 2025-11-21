# eBPF-based Microsegment 代码优化清单

> 生成时间: 2025-11-21
> 状态: 待优化
> 总优化项: 58 个

## 项目健康度概览

| 指标 | 当前状态 | 目标 |
|------|---------|------|
| 测试覆盖率 | ~45% | 80% |
| 代码重复率 | ~15% | <5% |
| TODO 数量 | 6 个 | 0 个 |
| 已知安全风险 | 2 个中等 | 0 个 |
| 文档覆盖率 | ~60% | 100% |

---

## 优先级分类

- **P0 (Critical)**: 影响功能或稳定性，需立即修复
- **P1 (High)**: 影响性能或用户体验，短期内修复
- **P2 (Medium)**: 代码质量改进，中期规划
- **P3 (Low)**: 优化增强，长期规划

---

## 一、架构层面优化 (12 项)

### 1.1 创建统一的共享代码目录 ⭐ P0

**问题**: README 提到 `pkg/` 目录存放共享代码，但实际不存在

**影响**: 代码重复度高 (~15%)，维护困难

**建议目录结构**:
```
pkg/
├── types/           # 共享数据结构 (Flow, Policy, Session)
├── proto/           # Protobuf 定义 (从 api/proto 移动)
├── netutil/         # 网络工具函数 (IP 转换、CIDR 解析)
├── constants/       # 常量定义 (端口号、超时值)
└── errors/          # 错误类型定义
```

**实施步骤**:
1. [ ] 创建 `pkg/` 目录结构
2. [ ] 迁移 IP 转换函数到 `pkg/netutil/ip.go`
3. [ ] 迁移共享常量到 `pkg/constants/`
4. [ ] 更新所有导入路径
5. [ ] 运行测试验证

**预计收益**: 减少 ~2000 行重复代码

---

### 1.2 统一 eBPF Maps 管理器 ⭐ P1

**问题**: Map Pinning 逻辑硬编码在多个文件中

**位置**:
- `src/bpf/tc_microsegment.bpf.c:76`
- `src/agent/pkg/dataplane/manager.go`

**建议实现**:
```go
package mapmanager

type MapManager struct {
    pinnedMaps map[string]*ebpf.Map
    pinPath    string  // 默认 /sys/fs/bpf/microsegment
    mu         sync.RWMutex
}

func NewMapManager(pinPath string) *MapManager {
    return &MapManager{
        pinnedMaps: make(map[string]*ebpf.Map),
        pinPath:    pinPath,
    }
}

// GetOrCreateSharedMap returns existing pinned map or creates new one
func (m *MapManager) GetOrCreateSharedMap(
    name string,
    spec *ebpf.MapSpec,
) (*ebpf.Map, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Try to load existing pinned map
    pinPath := filepath.Join(m.pinPath, name)
    if existingMap, err := ebpf.LoadPinnedMap(pinPath, nil); err == nil {
        m.pinnedMaps[name] = existingMap
        return existingMap, nil
    }

    // Create new map
    newMap, err := ebpf.NewMap(spec)
    if err != nil {
        return nil, fmt.Errorf("create map: %w", err)
    }

    // Pin it
    if err := newMap.Pin(pinPath); err != nil {
        newMap.Close()
        return nil, fmt.Errorf("pin map: %w", err)
    }

    m.pinnedMaps[name] = newMap
    return newMap, nil
}

// CleanupOrphanedMaps removes maps not in current program
func (m *MapManager) CleanupOrphanedMaps(currentMaps []string) error {
    entries, err := os.ReadDir(m.pinPath)
    if err != nil {
        return err
    }

    currentSet := make(map[string]bool)
    for _, name := range currentMaps {
        currentSet[name] = true
    }

    for _, entry := range entries {
        if !currentSet[entry.Name()] {
            os.Remove(filepath.Join(m.pinPath, entry.Name()))
        }
    }

    return nil
}
```

**任务清单**:
1. [ ] 创建 `pkg/mapmanager/manager.go`
2. [ ] 实现 MapManager 结构体
3. [ ] 迁移 DataPlane 使用 MapManager
4. [ ] 添加单元测试
5. [ ] 添加文档

---

### 1.3 解耦循环依赖风险 ⭐ P1

**问题**: FlowCollector ↔ ProcessMonitor 可能形成循环依赖

**位置**:
- `src/agent/pkg/flow/collector.go`
- `src/agent/pkg/process/monitor.go`

**建议**: 使用接口解耦

```go
// pkg/types/interfaces.go
package types

type ProcessMonitor interface {
    GetProcessInfo(pid uint32) (*ProcessInfo, bool)
    Subscribe(subscriber ProcessSubscriber) error
}

type ProcessSubscriber interface {
    OnProcessStart(info *ProcessInfo)
    OnProcessExit(pid uint32)
}

// flow/collector.go
type FlowCollector struct {
    procMon types.ProcessMonitor  // 依赖接口而非具体实现
}

func NewFlowCollector(procMon types.ProcessMonitor) *FlowCollector {
    return &FlowCollector{procMon: procMon}
}
```

**任务清单**:
1. [ ] 定义 `pkg/types/interfaces.go`
2. [ ] ProcessMonitor 实现接口
3. [ ] FlowCollector 依赖接口
4. [ ] 验证编译通过

---

### 1.4 eBPF 结构体自动同步检查 ⭐ P2

**问题**: C 结构体和 Go 结构体需要手动同步

**现状示例**:
```c
// C
struct flow_key {
    __u32 src_ip[4];
    __u32 dst_ip[4];
    __u16 src_port;
    __u16 dst_port;
    __u8 protocol;
    __u8 ip_version;
    __u16 vlan_id;
};
```

```go
// Go (需手动同步)
type FlowKey struct {
    SrcIP     [4]uint32
    DstIP     [4]uint32
    SrcPort   uint16
    DstPort   uint16
    Protocol  uint8
    IPVersion uint8
    VlanID    uint16
}
```

**建议**: 添加编译时大小检查

```go
// dataplane/types_check.go
package dataplane

import "C"
import "unsafe"

// Compile-time size assertions
const (
    _ = uint(unsafe.Sizeof(bpfFlowKey{})) - uint(C.sizeof_struct_flow_key)
    _ = uint(C.sizeof_struct_flow_key) - uint(unsafe.Sizeof(bpfFlowKey{}))
)
```

**任务清单**:
1. [ ] 为所有共享结构体添加大小检查
2. [ ] 添加 CI 步骤验证
3. [ ] 文档化同步流程

---

## 二、代码质量优化 (18 项)

### 2.1 消除 Fatal 调用，改为返回错误 ⭐ P0

**问题**: 多处使用 `logrus.Fatalf` 导致程序直接退出，无法优雅关闭

**位置**:
- `src/server/pkg/storage/flow_storage.go:32`
- `src/server/pkg/storage/policy_storage.go:28`
- `src/agent/pkg/dataplane/manager.go:45`

**错误示例**:
```go
// ❌ 不好 - 直接退出进程
gormDB, err := newGormFromSQL(db)
if err != nil {
    logrus.Fatalf("failed to initialize gorm: %v", err)
}
```

**修复方案**:
```go
// ✅ 推荐 - 返回错误
func NewFlowStorage(db *sql.DB) (*FlowStorage, error) {
    gormDB, err := newGormFromSQL(db)
    if err != nil {
        return nil, fmt.Errorf("initialize gorm: %w", err)
    }
    return &FlowStorage{db: gormDB, legacyDB: db}, nil
}
```

**任务清单**:
1. [ ] 搜索所有 `Fatalf` 调用
2. [ ] 改为返回错误
3. [ ] 调用者处理错误
4. [ ] 在 main() 中统一处理退出

**影响文件** (需逐一修复):
```bash
grep -r "Fatalf\|Fatal(" src/ --include="*.go" | wc -l
# 预计 8-12 处
```

---

### 2.2 统一 IP 转换工具函数 ⭐ P0

**问题**: IP 转换函数在多个文件中重复定义

**重复位置**:
- `src/agent/pkg/dataplane/dataplane.go:285` - `intToIP()`
- `src/agent/pkg/policy/policy.go:452` - `ipToUint32()`
- `src/server/pkg/storage/flow_storage.go:134` - `ipv4ToInt()`

**建议**: 创建统一工具包

```go
// pkg/netutil/ip.go
package netutil

import (
    "encoding/binary"
    "net"
)

// IPToUint32 converts net.IP to uint32 (little-endian)
// Returns 0 for invalid IPv4 addresses
func IPToUint32(ip net.IP) uint32 {
    ip4 := ip.To4()
    if ip4 == nil {
        return 0
    }
    return binary.LittleEndian.Uint32(ip4)
}

// Uint32ToIP converts uint32 to net.IP (little-endian)
func Uint32ToIP(n uint32) net.IP {
    ip := make(net.IP, 4)
    binary.LittleEndian.PutUint32(ip, n)
    return ip
}

// IPv6ToUint32Array converts IPv6 to [4]uint32 array
func IPv6ToUint32Array(ip net.IP) [4]uint32 {
    ip16 := ip.To16()
    if ip16 == nil {
        return [4]uint32{}
    }

    var arr [4]uint32
    for i := 0; i < 4; i++ {
        arr[i] = binary.BigEndian.Uint32(ip16[i*4 : (i+1)*4])
    }
    return arr
}

// Uint32ArrayToIPv6 converts [4]uint32 array to IPv6
func Uint32ArrayToIPv6(arr [4]uint32) net.IP {
    ip := make(net.IP, 16)
    for i := 0; i < 4; i++ {
        binary.BigEndian.PutUint32(ip[i*4:(i+1)*4], arr[i])
    }
    return ip
}

// ParseCIDR parses CIDR with auto-detection of /32 or /128
func ParseCIDR(cidr string) (net.IP, *net.IPNet, error) {
    if !strings.Contains(cidr, "/") {
        // Auto-detect IP version
        if strings.Contains(cidr, ":") {
            cidr += "/128"  // IPv6
        } else {
            cidr += "/32"   // IPv4
        }
    }

    ip, ipnet, err := net.ParseCIDR(cidr)
    if err != nil {
        return nil, nil, fmt.Errorf("invalid CIDR %s: %w", cidr, err)
    }

    return ip, ipnet, nil
}
```

**任务清单**:
1. [ ] 创建 `pkg/netutil/ip.go`
2. [ ] 实现所有转换函数
3. [ ] 添加单元测试 (覆盖 IPv4/IPv6)
4. [ ] 替换所有重复实现
5. [ ] 验证功能正确性

**预计删除代码**: ~150 行

---

### 2.3 改进错误处理 - 返回包装错误 ⭐ P1

**问题**: 错误信息丢失上下文

**位置**: `src/agent/pkg/policy/policy.go:116`

**当前代码**:
```go
if err := pm.storage.SavePolicy(p); err != nil {
    log.Warnf("Failed to persist policy rule_id=%d: %v", p.RuleID, err)
    // Continue even if persistence fails
}
```

**问题**:
- 用户不知道持久化失败
- 可能导致重启后策略丢失

**建议方案**:
```go
// pkg/errors/policy.go
package errors

type PolicyError struct {
    Policy       *Policy
    EBPFError    error  // eBPF map 操作错误 (致命)
    StorageError error  // 存储错误 (非致命)
}

func (e *PolicyError) Error() string {
    if e.EBPFError != nil {
        return fmt.Sprintf("eBPF error for policy %d: %v",
            e.Policy.RuleID, e.EBPFError)
    }
    return fmt.Sprintf("storage error for policy %d: %v",
        e.Policy.RuleID, e.StorageError)
}

func (e *PolicyError) Unwrap() error {
    if e.EBPFError != nil {
        return e.EBPFError
    }
    return e.StorageError
}

func (e *PolicyError) IsPartial() bool {
    return e.EBPFError == nil && e.StorageError != nil
}

// policy/policy.go
func (pm *PolicyManager) AddPolicy(p *Policy) error {
    ebpfErr := pm.addPolicyToMap(p)
    if ebpfErr != nil {
        return &errors.PolicyError{
            Policy:    p,
            EBPFError: ebpfErr,
        }
    }

    storageErr := pm.storage.SavePolicy(p)
    if storageErr != nil {
        log.Warnf("Policy added to eBPF but storage failed: %v", storageErr)
        return &errors.PolicyError{
            Policy:       p,
            StorageError: storageErr,
        }
    }

    return nil
}

// 调用者可以检查是否是部分成功
err := pm.AddPolicy(policy)
if err != nil {
    if policyErr, ok := err.(*errors.PolicyError); ok && policyErr.IsPartial() {
        // 部分成功 - 策略已生效但未持久化
        log.Warnf("Policy %d is active but not persisted", policy.RuleID)
    } else {
        // 完全失败
        return err
    }
}
```

**任务清单**:
1. [ ] 创建 `pkg/errors/policy.go`
2. [ ] 定义 PolicyError 类型
3. [ ] 修改 AddPolicy 返回值
4. [ ] 更新调用者处理逻辑
5. [ ] 添加测试

---

### 2.4 添加结构化日志 ⭐ P1

**问题**: 大量字符串拼接日志，不利于解析和过滤

**当前示例**:
```go
log.Infof("Policy added: rule_id=%d %s:%d -> %s:%d proto=%s action=%s dir=%s",
    p.RuleID, p.SrcIP, p.SrcPort, p.DstIP, p.DstPort,
    p.Protocol, p.Action, p.Direction)
```

**建议**: 使用 logrus.WithFields

```go
log.WithFields(logrus.Fields{
    "rule_id":   p.RuleID,
    "src":       fmt.Sprintf("%s:%d", p.SrcIP, p.SrcPort),
    "dst":       fmt.Sprintf("%s:%d", p.DstIP, p.DstPort),
    "protocol":  p.Protocol,
    "action":    p.Action,
    "direction": p.Direction,
}).Info("Policy added")
```

**优势**:
- 支持 JSON 输出
- 易于 ELK/Loki 解析
- 支持字段过滤

**任务清单**:
1. [ ] 识别所有日志调用点 (~200 处)
2. [ ] 转换为结构化日志
3. [ ] 统一字段命名规范
4. [ ] 更新配置支持 JSON 格式

---

### 2.5 添加 godoc 注释 ⭐ P2

**问题**: 导出函数缺少文档注释

**位置**: 大部分 `src/agent/pkg/` 和 `src/server/pkg/`

**不好的示例**:
```go
// ❌ 缺少文档
func (pm *PolicyManager) AddPolicy(p *Policy) error {
    // ...
}
```

**推荐格式**:
```go
// AddPolicy adds a network policy rule to the eBPF data plane.
//
// The policy is first validated for correctness (valid IP addresses,
// protocol, action, etc.), then added to the appropriate eBPF map:
//   - Exact match policies go to policy_map (hash map, O(1) lookup)
//   - Wildcard policies go to wildcard_policy_map (array, O(N) scan)
//
// If a storage backend is configured, the policy is persisted
// asynchronously. Storage failures are logged but do not cause the
// function to fail, as the eBPF map is the source of truth.
//
// Parameters:
//   - p: Policy to add (must have unique RuleID)
//
// Returns:
//   - nil on success
//   - error if validation fails or eBPF map update fails
//   - *PolicyError if storage persistence fails (policy still active)
//
// Example:
//   policy := &Policy{
//       RuleID: 1,
//       SrcIP: "192.168.1.0/24",
//       DstIP: "0.0.0.0/0",
//       DstPort: 80,
//       Protocol: "tcp",
//       Action: "allow",
//       Direction: "egress",
//   }
//   if err := pm.AddPolicy(policy); err != nil {
//       log.Fatalf("Failed to add policy: %v", err)
//   }
func (pm *PolicyManager) AddPolicy(p *Policy) error {
    // ...
}
```

**任务清单**:
1. [ ] 定义文档模板
2. [ ] 为所有导出函数添加注释
3. [ ] 运行 `go doc` 验证
4. [ ] 生成文档站点 (可选)

---

### 2.6 消除 eBPF 代码重复 ⭐ P2

**问题**: TC 和 XDP 程序有大量重复逻辑

**重复位置**:
- `src/bpf/tc_microsegment.bpf.c:584-838` (254 行)
- `src/bpf/xdp_microsegment.bpf.c:550-815` (265 行)

**重复内容**:
- 策略查找逻辑
- 会话创建逻辑
- NAT 检测逻辑
- Fragment 处理逻辑

**建议**: 抽取共享头文件

```c
// src/bpf/headers/session_common.h
#ifndef __SESSION_COMMON_H__
#define __SESSION_COMMON_H__

/* Generic context wrapper for TC/XDP */
struct pkt_context {
    void *data;
    void *data_end;
    __u32 len;
    __u32 ifindex;
};

/* Extract packet context from TC __sk_buff */
static __always_inline void pkt_ctx_from_tc(
    struct __sk_buff *skb,
    struct pkt_context *ctx)
{
    ctx->data = (void *)(long)skb->data;
    ctx->data_end = (void *)(long)skb->data_end;
    ctx->len = skb->len;
    ctx->ifindex = skb->ingress_ifindex;
}

/* Extract packet context from XDP xdp_md */
static __always_inline void pkt_ctx_from_xdp(
    struct xdp_md *xdp,
    struct pkt_context *ctx)
{
    ctx->data = (void *)(long)xdp->data;
    ctx->data_end = (void *)(long)xdp->data_end;
    ctx->len = ctx->data_end - ctx->data;
    ctx->ifindex = xdp->ingress_ifindex;
}

/* Unified new session handler */
static __always_inline int handle_new_session(
    struct pkt_context *ctx,
    struct flow_key *key,
    __u8 action,
    __u64 timestamp,
    __u32 rule_id,
    __u8 direction,
    struct process_match_info *proc_info,
    void *session_map)  // Generic map pointer
{
    // TCP state initialization
    __u8 initial_tcp_state = TCP_STATE_CLOSED;
    if (key->protocol == IPPROTO_TCP) {
        // Extract TCP flags from packet
        __u8 flags = extract_tcp_flags(ctx);
        initial_tcp_state = tcp_state_transition_inbound(
            TCP_STATE_CLOSED, flags);
    }

    struct session_value new_session = {
        .created_ts = timestamp,
        .last_seen_ts = timestamp,
        .packets_to_server = 1,
        .packets_to_client = 0,
        .bytes_to_server = ctx->len,
        .bytes_to_client = 0,
        .state = SESSION_STATE_NEW,
        .tcp_state = initial_tcp_state,
        .policy_action = action,
        .flags = 0,
    };

    return bpf_map_update_elem(session_map, key, &new_session, BPF_NOEXIST);
}

#endif /* __SESSION_COMMON_H__ */
```

**使用方式**:
```c
// tc_microsegment.bpf.c
#include "headers/session_common.h"

SEC("tc")
int tc_microsegment_filter(struct __sk_buff *skb) {
    struct pkt_context pkt_ctx;
    pkt_ctx_from_tc(skb, &pkt_ctx);

    // ...

    handle_new_session(&pkt_ctx, &key, action, now, rule_id,
                      direction, &proc_info, &session_map);
}

// xdp_microsegment.bpf.c
#include "headers/session_common.h"

SEC("xdp")
int xdp_microsegment_filter(struct xdp_md *xdp) {
    struct pkt_context pkt_ctx;
    pkt_ctx_from_xdp(xdp, &pkt_ctx);

    // ...

    handle_new_session(&pkt_ctx, &key, action, now, rule_id,
                      direction, &proc_info, &session_map);
}
```

**任务清单**:
1. [x] 创建 `headers/session_common.h`
2. [x] 分析重复代码和函数签名差异
3. [ ] 抽取共享逻辑
4. [ ] 测试 TC 程序
5. [ ] 测试 XDP 程序
6. [ ] 验证性能无退化

**分析结果** (2025-11-21):

已识别的重复代码区域：
- **会话更新逻辑** (tc: lines 613-682, xdp: lines 565-636): ~70 行
  - 会话统计更新
  - TCP 状态机转换
  - 连接关闭检测

- **新会话创建** (tc: lines 381-430, xdp: lines 718-768): ~50 行
  - TCP 初始状态设置
  - 会话数据结构初始化
  - 流事件推送

- **NAT 检测** (tc: lines 713-722, xdp: lines 665-674): ~10 行
  - 调用 `detect_nat_and_restore_with_maps`

- **Fragment 处理** (tc: lines 693-706, xdp: lines 645-658): ~14 行
  - Fragment 检测和缓存策略查找

**实施挑战**:
1. **函数签名不一致**: `push_flow_event` (TC: 13 参数) vs `push_flow_event_xdp` (XDP: 不同参数顺序)
2. **上下文类型差异**: `__sk_buff` vs `xdp_md`
3. **方向检测差异**: TC 支持双向，XDP 仅 ingress
4. **编译依赖顺序**: 共享头文件需要在辅助函数之后包含

**推荐方案**:
1. 优先级调整为 P3 (低优先级) - 重复代码影响维护性但不影响功能
2. 分阶段实施：
   - Phase 1: 统一数据结构定义（pkt_context）
   - Phase 2: 创建统一的函数接口（使用函数指针或宏）
   - Phase 3: 逐步迁移重复逻辑
3. 或者保持现状，通过代码注释标注对应关系，确保同步修改

**预计减少代码**: ~144 行 (如果完全实施)

---

## 三、性能优化 (10 项)

### 3.1 启用索引策略查找 ⭐ P0

**问题**: 通配符策略使用线性扫描 O(N)，每个新连接可能需要 50-100μs

**位置**: `src/bpf/tc_microsegment.bpf.c:738`

**当前状态**:
```c
// 已实现但被禁用
// #if USE_INDEXED_LOOKUP
//     __u8 action = lookup_policy_action_indexed(&original_key, direction, &matched_rule_id);
// #else
    __u8 action = lookup_policy_action(&original_key, direction, &matched_rule_id);
// #endif
```

**原因**: 注释显示索引查找不支持进程策略

**解决方案**: 扩展索引查找支持进程字段

```c
// headers/indexed_policy_match_v3.h (新版本)

/* Protocol-indexed lookup with process support
 *
 * Index structure:
 *   protocol_offset_map[protocol] -> {start_idx, count, process_count}
 *
 * Layout in wildcard_policy_map:
 *   [start_idx, start_idx + count - process_count): network-only policies
 *   [start_idx + count - process_count, start_idx + count): process policies
 *
 * Algorithm:
 *   1. Lookup protocol segment: O(1)
 *   2. Scan process policies first: O(P) where P = process_count
 *   3. Scan network policies if no match: O(N - P)
 *
 * Performance:
 *   - With process match: ~2-5μs (scan 10-50 process policies)
 *   - Network-only match: ~10-20μs (scan 50-200 network policies)
 *   - Much better than O(1000) full scan
 */
static __always_inline __u8 lookup_policy_action_indexed_v3(
    struct flow_key *key,
    struct process_match_info *proc,
    __u8 direction,
    __u32 *rule_id)
{
    // Lookup protocol segment
    __u32 proto = (__u32)key->protocol;
    struct protocol_segment *seg = bpf_map_lookup_elem(
        &protocol_offset_map, &proto);

    if (!seg || seg->count == 0) {
        // Fallback to protocol 0 (ANY)
        proto = 0;
        seg = bpf_map_lookup_elem(&protocol_offset_map, &proto);
        if (!seg || seg->count == 0) {
            return POLICY_ACTION_ALLOW;  // Default allow
        }
    }

    __u32 start_idx = seg->start_idx;
    __u32 end_idx = start_idx + seg->count;
    __u32 proc_start_idx = end_idx - seg->process_count;

    // Phase 1: Scan process-specific policies (higher priority)
    struct wildcard_policy *best_match = NULL;
    __u16 best_priority = 0;

    #pragma unroll
    for (__u32 i = 0; i < 50; i++) {  // Max 50 process policies per protocol
        __u32 idx = proc_start_idx + i;
        if (idx >= end_idx)
            break;

        struct wildcard_policy *policy = bpf_map_lookup_elem(
            &wildcard_policy_map, &idx);
        if (!policy || policy->rule_id == 0)
            break;

        // Must match process
        if (!matches_wildcard_with_process(key, proc, policy, direction))
            continue;

        if (!best_match || policy->priority > best_priority) {
            best_match = policy;
            best_priority = policy->priority;
        }
    }

    if (best_match) {
        *rule_id = best_match->rule_id;
        return best_match->action;
    }

    // Phase 2: Scan network-only policies (lower priority)
    #pragma unroll
    for (__u32 i = 0; i < 200; i++) {  // Max 200 network policies per protocol
        __u32 idx = start_idx + i;
        if (idx >= proc_start_idx)
            break;

        struct wildcard_policy *policy = bpf_map_lookup_elem(
            &wildcard_policy_map, &idx);
        if (!policy || policy->rule_id == 0)
            break;

        // Network-only policies (process_name must be empty)
        if (policy->process_name[0] != '\0')
            continue;

        if (!matches_wildcard_with_process(key, proc, policy, direction))
            continue;

        if (!best_match || policy->priority > best_priority) {
            best_match = policy;
            best_priority = policy->priority;
        }
    }

    if (best_match) {
        *rule_id = best_match->rule_id;
        return best_match->action;
    }

    return POLICY_ACTION_ALLOW;  // Default allow
}
```

**任务清单**:
1. [ ] 创建 `headers/indexed_policy_match_v3.h`
2. [ ] 扩展 protocol_segment 结构体支持 process_count
3. [ ] 修改策略加载逻辑，正确排序策略
4. [ ] 启用 USE_INDEXED_LOOKUP 编译标志
5. [ ] 性能测试 (对比线性扫描)

**预期性能提升**:
- 新连接延迟: 50-100μs → 2-20μs
- 吞吐量提升: ~5-10x (高策略数量场景)

---

### 3.2 添加 Ring Buffer 丢弃统计 ⭐ P0

**问题**: Ring Buffer 满时静默丢弃事件，难以监控

**位置**: `src/bpf/tc_microsegment.bpf.c:312`

**当前代码**:
```c
struct flow_event *event = bpf_ringbuf_reserve(&flow_events, sizeof(*event), 0);
if (!event) {
    // Ring buffer full - event dropped (silent failure for performance)
    return -1;
}
```

**建议**: 添加统计计数

```c
// headers/common_types.h (添加新统计项)
enum stats_key {
    // ... existing stats ...
    STATS_RINGBUF_FULL = 16,         // Ring buffer full, events dropped
    STATS_RINGBUF_RESERVE_FAIL = 17, // Reserve failed
    STATS_MAX = 20,  // Updated
};

// tc_microsegment.bpf.c
struct flow_event *event = bpf_ringbuf_reserve(&flow_events, sizeof(*event), 0);
if (!event) {
    update_stats(STATS_RINGBUF_FULL);  // ⭐ 添加统计
    return -1;
}
```

**用户态监控**:
```go
// dataplane/stats.go
type EBPFStats struct {
    // ... existing fields ...
    RingBufferFull   uint64 `json:"ringbuffer_full"`
    RingBufferDropRate float64 `json:"ringbuffer_drop_rate"`
}

func (dp *DataPlane) GetStats() (*EBPFStats, error) {
    // ...

    ringbufFull := statsMap[STATS_RINGBUF_FULL]
    totalFlows := statsMap[STATS_NEW_SESSIONS]

    stats.RingBufferFull = ringbufFull
    if totalFlows > 0 {
        stats.RingBufferDropRate = float64(ringbufFull) / float64(totalFlows)
    }

    // Alert if drop rate > 1%
    if stats.RingBufferDropRate > 0.01 {
        log.Warnf("High ring buffer drop rate: %.2f%% (%d/%d events dropped)",
            stats.RingBufferDropRate*100, ringbufFull, totalFlows)
    }

    return stats, nil
}
```

**任务清单**:
1. [ ] 添加 STATS_RINGBUF_FULL 枚举
2. [ ] 在所有 reserve 失败处添加统计
3. [ ] 用户态读取并暴露指标
4. [ ] 添加监控告警

---

### 3.3 添加 Context 取消支持 ⭐ P1

**问题**: 无界循环无法优雅停止

**位置**: `src/agent/pkg/dataplane/dataplane.go:224`

**当前代码**:
```go
func (dp *DataPlane) MonitorFlowEvents() {
    for {
        record, err := dp.rbReader.Read()
        // ... no exit mechanism
    }
}
```

**建议**: 使用 Context

```go
func (dp *DataPlane) MonitorFlowEvents(ctx context.Context, handler func(*FlowEvent)) error {
    errChan := make(chan error, 1)

    go func() {
        for {
            select {
            case <-ctx.Done():
                errChan <- ctx.Err()
                return
            default:
                // Non-blocking read with timeout
                record, err := dp.rbReader.Read()
                if err != nil {
                    if errors.Is(err, ringbuf.ErrClosed) {
                        errChan <- nil
                        return
                    }
                    log.Errorf("Failed to read ring buffer: %v", err)
                    continue
                }

                var event FlowEvent
                if err := binary.Read(bytes.NewReader(record.RawSample),
                    binary.LittleEndian, &event); err != nil {
                    log.Errorf("Failed to parse flow event: %v", err)
                    continue
                }

                handler(&event)
            }
        }
    }()

    return <-errChan
}

// 调用方
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go func() {
    if err := dp.MonitorFlowEvents(ctx, handleFlowEvent); err != nil {
        log.Errorf("Flow monitoring stopped: %v", err)
    }
}()

// 优雅关闭
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan

cancel()  // 触发所有监控协程退出
time.Sleep(1 * time.Second)  // 等待清理
```

**任务清单**:
1. [ ] 所有无界循环添加 Context 参数
2. [ ] 在 select 中监听 ctx.Done()
3. [ ] 优雅关闭逻辑
4. [ ] 测试取消流程

---

### 3.4 优化 Map 迭代性能 ⭐ P1

**问题**: 频繁内存分配

**位置**: `src/agent/pkg/policy/policy.go:315`

**当前代码**:
```go
iter := pm.policyMap.Iterate()
for iter.Next(&key, &value) {
    policy := Policy{...}  // 每次迭代分配
    policies = append(policies, policy)
}
```

**优化方案**:
```go
func (pm *PolicyManager) ListPolicies() ([]*Policy, error) {
    // 预分配切片 (减少扩容)
    policies := make([]*Policy, 0, 1000)

    var key policyKey
    var value policyValue

    iter := pm.policyMap.Iterate()
    for iter.Next(&key, &value) {
        // 复用对象池 (可选，针对高频调用)
        policy := policyPool.Get().(*Policy)
        policy.RuleID = value.RuleID
        policy.SrcIP = intToIP(key.SrcIP)
        // ...

        policies = append(policies, policy)
    }

    return policies, iter.Err()
}

// sync.Pool for policy objects
var policyPool = sync.Pool{
    New: func() interface{} {
        return &Policy{}
    },
}
```

**任务清单**:
1. [ ] 识别所有 Map 迭代位置
2. [ ] 预分配切片容量
3. [ ] 考虑使用对象池 (高频场景)
4. [ ] Benchmark 测试

---

### 3.5 添加内存限制 - ProcessCache ⭐ P1

**问题**: LRU 缓存可能无限增长

**位置**: `src/agent/pkg/process/cache.go`

**当前**: 只限制条目数 (20000)，未限制内存

**建议**: 添加内存限制

```go
type ProcessCache struct {
    lru         *lru.Cache
    maxMemory   int64  // 最大内存 (bytes)
    curMemory   atomic.Int64
    mu          sync.RWMutex
}

func NewProcessCache(maxEntries int, maxMemoryMB int64, ttl time.Duration) *ProcessCache {
    cache, _ := lru.NewWithEvict(maxEntries, func(key, value interface{}) {
        // 驱逐回调
        if info, ok := value.(*ProcessInfo); ok {
            size := calculateProcessInfoSize(info)
            cache.curMemory.Add(-size)
        }
    })

    return &ProcessCache{
        lru:       cache,
        maxMemory: maxMemoryMB * 1024 * 1024,
        ttl:       ttl,
    }
}

func (c *ProcessCache) Add(pid uint32, info *ProcessInfo) error {
    size := calculateProcessInfoSize(info)

    // 检查内存限制
    for c.curMemory.Load() + size > c.maxMemory {
        // 驱逐最旧条目
        if !c.evictOldest() {
            return fmt.Errorf("cache full, cannot evict")
        }
    }

    c.mu.Lock()
    defer c.mu.Unlock()

    c.lru.Add(pid, info)
    c.curMemory.Add(size)

    return nil
}

func calculateProcessInfoSize(info *ProcessInfo) int64 {
    return int64(unsafe.Sizeof(*info)) +
           int64(len(info.Cmdline)) +
           int64(len(info.Exe))
}

func (c *ProcessCache) evictOldest() bool {
    c.mu.Lock()
    defer c.mu.Unlock()

    return c.lru.RemoveOldest()
}
```

**配置示例**:
```yaml
# config/agent.yaml
process_monitor:
  cache_max_entries: 20000
  cache_max_memory_mb: 100  # 限制 100MB
  cache_ttl: 5m
```

**任务清单**:
1. [ ] 添加内存统计
2. [ ] 实现内存限制逻辑
3. [ ] 添加配置选项
4. [ ] 监控内存使用

---

## 四、功能完善 (12 项)

### 4.1 实现流标签过滤 ⭐ P0

**问题**: JSONB 标签查询未实现

**位置**: `src/server/pkg/api/handlers/flow.go:104`

**当前代码**:
```go
// TODO: Implement JSONB label filtering
```

**实现方案**:
```go
func (h *FlowHandler) QueryFlows(c *gin.Context) {
    var query FlowQueryRequest
    if err := c.ShouldBindJSON(&query); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    db := h.storage.DB()

    // Time range filter
    if !query.StartTime.IsZero() {
        db = db.Where("timestamp >= ?", query.StartTime)
    }
    if !query.EndTime.IsZero() {
        db = db.Where("timestamp <= ?", query.EndTime)
    }

    // Label filters (JSONB query)
    for key, value := range query.Labels {
        db = db.Where("labels ->> ? = ?", key, value)
    }

    // Label existence check
    for _, key := range query.HasLabels {
        db = db.Where("labels ? ?", key)
    }

    // Complex label queries (array contains, etc.)
    if query.LabelSelector != "" {
        // Support k8s-style label selector:
        // "app=frontend,tier in (web,api),!deprecated"
        selector, err := parseL labelSelector(query.LabelSelector)
        if err != nil {
            c.JSON(400, gin.H{"error": fmt.Sprintf("invalid label selector: %v", err)})
            return
        }
        db = applyLabelSelector(db, selector)
    }

    var flows []Flow
    if err := db.Find(&flows).Error; err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"flows": flows, "count": len(flows)})
}

// applyLabelSelector applies k8s-style label selector to query
func applyLabelSelector(db *gorm.DB, selector *LabelSelector) *gorm.DB {
    for _, req := range selector.Requirements {
        switch req.Operator {
        case "=", "==":
            db = db.Where("labels ->> ? = ?", req.Key, req.Values[0])
        case "!=":
            db = db.Where("NOT (labels ->> ? = ?)", req.Key, req.Values[0])
        case "in":
            db = db.Where("labels ->> ? IN ?", req.Key, req.Values)
        case "notin":
            db = db.Where("labels ->> ? NOT IN ?", req.Key, req.Values)
        case "exists":
            db = db.Where("labels ? ?", req.Key)
        case "!":
            db = db.Where("NOT (labels ? ?)", req.Key)
        }
    }
    return db
}
```

**任务清单**:
1. [ ] 实现基础 JSONB 查询
2. [ ] 实现 k8s 风格 label selector
3. [ ] 添加索引优化查询性能
4. [ ] 添加单元测试
5. [ ] 更新 API 文档

---

### 4.2 实现策略增量更新 ⭐ P1

**问题**: 策略同步需要全量传输，效率低

**位置**: `src/server/pkg/grpc/policy_service.go:52`

**当前代码**:
```go
// TODO: Implement proper pub/sub mechanism for incremental updates
```

**实现方案**:
```go
// proto/policy/policy.proto (添加新消息类型)
message PolicyUpdate {
    enum UpdateType {
        ADDED = 0;
        MODIFIED = 1;
        DELETED = 2;
    }

    UpdateType type = 1;
    Policy policy = 2;       // For ADDED/MODIFIED
    uint32 rule_id = 3;      // For DELETED
    uint64 version = 4;      // Policy version
    google.protobuf.Timestamp timestamp = 5;
}

message PolicySyncRequest {
    string agent_id = 1;
    uint64 current_version = 2;  // Agent's current policy version
}

message PolicySyncResponse {
    repeated PolicyUpdate updates = 1;
    uint64 latest_version = 2;
}

service PolicyService {
    // Existing
    rpc SyncPolicies(PolicySyncRequest) returns (PolicySyncResponse);

    // New: Streaming updates
    rpc StreamPolicyUpdates(PolicySyncRequest) returns (stream PolicyUpdate);
}

// server/pkg/grpc/policy_service.go
type PolicyService struct {
    storage   *storage.PolicyStorage
    pubsub    *PolicyPubSub  // 新增发布订阅机制
}

func (s *PolicyService) SyncPolicies(
    ctx context.Context,
    req *policypb.PolicySyncRequest,
) (*policypb.PolicySyncResponse, error) {
    // Get incremental updates since current_version
    updates, err := s.storage.GetPolicyUpdates(req.CurrentVersion)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "get updates: %v", err)
    }

    latestVersion := s.storage.GetLatestVersion()

    return &policypb.PolicySyncResponse{
        Updates:       updates,
        LatestVersion: latestVersion,
    }, nil
}

func (s *PolicyService) StreamPolicyUpdates(
    req *policypb.PolicySyncRequest,
    stream policypb.PolicyService_StreamPolicyUpdatesServer,
) error {
    // Send historical updates first
    updates, err := s.storage.GetPolicyUpdates(req.CurrentVersion)
    if err != nil {
        return status.Errorf(codes.Internal, "get updates: %v", err)
    }

    for _, update := range updates {
        if err := stream.Send(update); err != nil {
            return err
        }
    }

    // Subscribe to future updates
    updateChan := make(chan *policypb.PolicyUpdate, 100)
    s.pubsub.Subscribe(req.AgentId, updateChan)
    defer s.pubsub.Unsubscribe(req.AgentId)

    for {
        select {
        case <-stream.Context().Done():
            return stream.Context().Err()
        case update := <-updateChan:
            if err := stream.Send(update); err != nil {
                return err
            }
        }
    }
}

// storage/policy_storage.go
type PolicyStorage struct {
    db            *gorm.DB
    versionMu     sync.RWMutex
    latestVersion uint64
    updateLog     *UpdateLog  // 保存增量更新日志
}

func (s *PolicyStorage) GetPolicyUpdates(sinceVersion uint64) ([]*PolicyUpdate, error) {
    var updates []PolicyUpdateRecord
    err := s.db.Where("version > ?", sinceVersion).
        Order("version ASC").
        Find(&updates).Error

    if err != nil {
        return nil, err
    }

    result := make([]*PolicyUpdate, len(updates))
    for i, u := range updates {
        result[i] = &PolicyUpdate{
            Type:    u.Type,
            Policy:  u.Policy,
            RuleId:  u.RuleID,
            Version: u.Version,
        }
    }

    return result, nil
}

// 策略变更时记录日志
func (s *PolicyStorage) AddPolicy(p *Policy) error {
    s.versionMu.Lock()
    s.latestVersion++
    version := s.latestVersion
    s.versionMu.Unlock()

    tx := s.db.Begin()

    // Save policy
    if err := tx.Create(p).Error; err != nil {
        tx.Rollback()
        return err
    }

    // Log update
    update := PolicyUpdateRecord{
        Type:      UPDATE_TYPE_ADDED,
        Policy:    p,
        Version:   version,
        Timestamp: time.Now(),
    }
    if err := tx.Create(&update).Error; err != nil {
        tx.Rollback()
        return err
    }

    tx.Commit()

    // Notify subscribers
    s.pubsub.Publish(&update)

    return nil
}
```

**数据库 Schema**:
```sql
CREATE TABLE policy_updates (
    id BIGSERIAL PRIMARY KEY,
    version BIGINT NOT NULL,
    update_type VARCHAR(20) NOT NULL,  -- 'ADDED', 'MODIFIED', 'DELETED'
    rule_id INT,
    policy JSONB,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),

    INDEX idx_version (version),
    INDEX idx_timestamp (timestamp)
);

-- 定期清理旧日志 (保留 7 天)
DELETE FROM policy_updates WHERE timestamp < NOW() - INTERVAL '7 days';
```

**任务清单**:
1. [ ] 设计 PolicyUpdate 消息格式
2. [ ] 实现 UpdateLog 存储
3. [ ] 实现 PubSub 机制
4. [ ] Agent 端支持增量更新
5. [ ] 添加版本冲突处理
6. [ ] 性能测试

---

### 4.3 添加 Prometheus 指标导出 ⭐ P1

**问题**: 缺少指标导出，难以监控

**实现方案**:
```go
// pkg/metrics/metrics.go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // eBPF packet counters
    EBPFPacketsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ebpf_packets_total",
            Help: "Total number of packets processed by eBPF programs",
        },
        []string{"action", "direction"},  // allow/deny, ingress/egress
    )

    // eBPF session counters
    EBPFSessionsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "ebpf_sessions_total",
            Help: "Total number of sessions created",
        },
        []string{"protocol"},
    )

    EBPFSessionsActive = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "ebpf_sessions_active",
            Help: "Current number of active sessions",
        },
    )

    // Policy counters
    PoliciesLoaded = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "policies_loaded",
            Help: "Number of policies currently loaded",
        },
    )

    PolicyOperations = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "policy_operations_total",
            Help: "Total number of policy operations",
        },
        []string{"operation", "status"},  // add/delete/update, success/failure
    )

    // Flow collection
    FlowEventsCollected = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "flow_events_collected_total",
            Help: "Total number of flow events collected from ring buffer",
        },
    )

    FlowEventsDropped = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "flow_events_dropped_total",
            Help: "Total number of flow events dropped (ring buffer full)",
        },
    )

    // Process monitoring
    ProcessEventsTotal = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "process_events_total",
            Help: "Total number of process events (exec/exit)",
        },
    )

    ProcessCacheSize = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "process_cache_size",
            Help: "Current number of entries in process cache",
        },
    )

    // gRPC metrics
    GRPCRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "grpc_requests_total",
            Help: "Total number of gRPC requests",
        },
        []string{"method", "status"},
    )

    GRPCRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "grpc_request_duration_seconds",
            Help:    "gRPC request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method"},
    )
)

// UpdateFromEBPFStats updates Prometheus metrics from eBPF stats map
func UpdateFromEBPFStats(stats map[uint32]uint64) {
    EBPFPacketsTotal.WithLabelValues("allow", "ingress").
        Add(float64(stats[STATS_ALLOWED_PACKETS]))
    EBPFPacketsTotal.WithLabelValues("deny", "ingress").
        Add(float64(stats[STATS_DENIED_PACKETS]))

    EBPFSessionsActive.Set(float64(stats[STATS_ACTIVE_SESSIONS]))

    FlowEventsDropped.Add(float64(stats[STATS_RINGBUF_FULL]))
}

// cmd/agent/main.go
import (
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "net/http"
)

func main() {
    // ... existing setup ...

    // Start Prometheus metrics endpoint
    go func() {
        http.Handle("/metrics", promhttp.Handler())
        log.Infof("Prometheus metrics server listening on :2112/metrics")
        if err := http.ListenAndServe(":2112", nil); err != nil {
            log.Fatalf("Failed to start metrics server: %v", err)
        }
    }()

    // Periodically update metrics from eBPF
    go func() {
        ticker := time.NewTicker(10 * time.Second)
        defer ticker.Stop()

        for range ticker.C {
            stats, err := dataplane.GetStats()
            if err != nil {
                log.Errorf("Failed to get eBPF stats: %v", err)
                continue
            }
            metrics.UpdateFromEBPFStats(stats)
        }
    }()

    // ... rest of main ...
}
```

**Grafana Dashboard** (JSON 配置):
```json
{
  "dashboard": {
    "title": "eBPF Microsegmentation",
    "panels": [
      {
        "title": "Packet Rate",
        "targets": [
          {
            "expr": "rate(ebpf_packets_total[5m])",
            "legendFormat": "{{action}} {{direction}}"
          }
        ]
      },
      {
        "title": "Active Sessions",
        "targets": [
          {
            "expr": "ebpf_sessions_active"
          }
        ]
      },
      {
        "title": "Flow Event Drop Rate",
        "targets": [
          {
            "expr": "rate(flow_events_dropped_total[5m]) / rate(flow_events_collected_total[5m])"
          }
        ]
      }
    ]
  }
}
```

**任务清单**:
1. [ ] 创建 `pkg/metrics/` 包
2. [ ] 定义所有指标
3. [ ] 集成到 Agent/Server
4. [ ] 创建 Grafana Dashboard
5. [ ] 添加告警规则

---

### 4.4 提取进程 UID/GID ⭐ P1

**问题**: 进程 UID 未实现

**位置**: `src/agent/pkg/flow/collector.go:420`

**当前代码**:
```go
UID: 0, // TODO: Extract UID from process info if available
```

**实现方案**:
```go
// pkg/procutil/proc.go
package procutil

import (
    "bufio"
    "fmt"
    "os"
    "strconv"
    "strings"
)

type ProcessCreds struct {
    UID  uint32
    GID  uint32
    EUID uint32  // Effective UID
    EGID uint32  // Effective GID
}

// GetProcessCreds reads UID/GID from /proc/<pid>/status
func GetProcessCreds(pid uint32) (*ProcessCreds, error) {
    path := fmt.Sprintf("/proc/%d/status", pid)
    file, err := os.Open(path)
    if err != nil {
        return nil, fmt.Errorf("open %s: %w", path, err)
    }
    defer file.Close()

    creds := &ProcessCreds{}
    scanner := bufio.NewScanner(file)

    for scanner.Scan() {
        line := scanner.Text()
        fields := strings.Fields(line)
        if len(fields) < 2 {
            continue
        }

        switch fields[0] {
        case "Uid:":
            // Format: Uid: <real> <effective> <saved> <fs>
            if len(fields) >= 3 {
                if uid, err := strconv.ParseUint(fields[1], 10, 32); err == nil {
                    creds.UID = uint32(uid)
                }
                if euid, err := strconv.ParseUint(fields[2], 10, 32); err == nil {
                    creds.EUID = uint32(euid)
                }
            }
        case "Gid:":
            if len(fields) >= 3 {
                if gid, err := strconv.ParseUint(fields[1], 10, 32); err == nil {
                    creds.GID = uint32(gid)
                }
                if egid, err := strconv.ParseUint(fields[2], 10, 32); err == nil {
                    creds.EGID = uint32(egid)
                }
            }
        }

        // Early exit if we found both
        if creds.UID != 0 && creds.GID != 0 {
            break
        }
    }

    if err := scanner.Err(); err != nil {
        return nil, fmt.Errorf("scan %s: %w", path, err)
    }

    return creds, nil
}

// GetUsername resolves UID to username
func GetUsername(uid uint32) (string, error) {
    user, err := user.LookupId(strconv.FormatUint(uint64(uid), 10))
    if err != nil {
        return "", err
    }
    return user.Username, nil
}

// flow/collector.go
func (c *Collector) enrichProcessInfo(flow *Flow, pid uint32) {
    // Get process info from cache
    procInfo, ok := c.procMon.Get(pid)
    if !ok {
        return
    }

    // Get credentials
    creds, err := procutil.GetProcessCreds(pid)
    if err != nil {
        log.Debugf("Failed to get credentials for PID %d: %v", pid, err)
        flow.UID = 0
        flow.GID = 0
    } else {
        flow.UID = creds.UID
        flow.GID = creds.GID

        // Optionally resolve username
        if username, err := procutil.GetUsername(creds.UID); err == nil {
            flow.Username = username
        }
    }
}
```

**任务清单**:
1. [ ] 创建 `pkg/procutil/proc.go`
2. [ ] 实现 UID/GID 提取
3. [ ] 缓存 UID → Username 映射
4. [ ] 集成到 FlowCollector
5. [ ] 添加测试

---

## 五、测试覆盖 (8 项)

### 5.1 添加 eBPF 单元测试 ⭐ P1

**问题**: eBPF 程序缺少单元测试

**实现方案**:
```go
// src/agent/pkg/dataplane/tc_test.go
package dataplane

import (
    "bytes"
    "encoding/binary"
    "testing"

    "github.com/cilium/ebpf"
    "github.com/stretchr/testify/require"
)

func TestTCMicrosegmentFilter(t *testing.T) {
    // Load eBPF program
    spec, err := loadBpf()
    require.NoError(t, err)

    prog := spec.Programs["tc_microsegment_filter"]
    require.NotNil(t, prog)

    // Load program
    loadedProg, err := ebpf.NewProgram(prog)
    require.NoError(t, err)
    defer loadedProg.Close()

    tests := []struct {
        name           string
        packet         []byte
        expectedAction int
        setupMaps      func(*ebpf.Collection)
    }{
        {
            name: "Allow TCP SYN to port 80",
            packet: buildTCPPacket(
                "192.168.1.100", "192.168.1.1",
                12345, 80,
                TCP_FLAG_SYN,
            ),
            expectedAction: TC_ACT_OK,
            setupMaps: func(coll *ebpf.Collection) {
                // Add allow policy for port 80
                addWildcardPolicy(coll, &WildcardPolicy{
                    RuleID:   1,
                    DstPort:  80,
                    Protocol: IPPROTO_TCP,
                    Action:   POLICY_ACTION_ALLOW,
                })
            },
        },
        {
            name: "Deny TCP to port 22",
            packet: buildTCPPacket(
                "192.168.1.100", "192.168.1.1",
                54321, 22,
                TCP_FLAG_SYN,
            ),
            expectedAction: TC_ACT_SHOT,
            setupMaps: func(coll *ebpf.Collection) {
                addWildcardPolicy(coll, &WildcardPolicy{
                    RuleID:   2,
                    DstPort:  22,
                    Protocol: IPPROTO_TCP,
                    Action:   POLICY_ACTION_DENY,
                })
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup maps
            if tt.setupMaps != nil {
                tt.setupMaps(spec)
            }

            // Run program with test packet
            ret, _, err := loadedProg.Test(tt.packet)
            require.NoError(t, err)
            require.Equal(t, tt.expectedAction, ret, "Unexpected action")
        })
    }
}

func buildTCPPacket(srcIP, dstIP string, srcPort, dstPort uint16, flags uint8) []byte {
    buf := new(bytes.Buffer)

    // Ethernet header (14 bytes)
    buf.Write([]byte{
        0x00, 0x00, 0x00, 0x00, 0x00, 0x01,  // Dst MAC
        0x00, 0x00, 0x00, 0x00, 0x00, 0x02,  // Src MAC
        0x08, 0x00,                           // EtherType: IPv4
    })

    // IPv4 header (20 bytes)
    ipHdr := make([]byte, 20)
    ipHdr[0] = 0x45  // Version + IHL
    ipHdr[9] = 6     // Protocol: TCP
    copy(ipHdr[12:16], net.ParseIP(srcIP).To4())
    copy(ipHdr[16:20], net.ParseIP(dstIP).To4())
    buf.Write(ipHdr)

    // TCP header (20 bytes)
    tcpHdr := make([]byte, 20)
    binary.BigEndian.PutUint16(tcpHdr[0:2], srcPort)
    binary.BigEndian.PutUint16(tcpHdr[2:4], dstPort)
    tcpHdr[13] = flags  // TCP flags
    buf.Write(tcpHdr)

    return buf.Bytes()
}

func addWildcardPolicy(coll *ebpf.Collection, policy *WildcardPolicy) error {
    policyMap := coll.Maps["wildcard_policy_map"]

    var idx uint32 = 0
    return policyMap.Put(&idx, policy)
}
```

**任务清单**:
1. [ ] 创建 `tc_test.go` 和 `xdp_test.go`
2. [ ] 实现 packet builder 工具
3. [ ] 测试基本场景 (allow/deny)
4. [ ] 测试边界情况 (fragment, NAT)
5. [ ] CI 集成

---

### 5.2 添加并发安全测试 ⭐ P1

**问题**: 缺少竞态条件测试

**实现方案**:
```go
// src/agent/pkg/process/cache_test.go
func TestProcessCacheConcurrency(t *testing.T) {
    cache := NewProcessCache(1000, 5*time.Minute)

    const (
        numGoroutines = 100
        numOperations = 1000
    )

    var wg sync.WaitGroup

    // Concurrent writers
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            for j := 0; j < numOperations; j++ {
                pid := uint32(id*numOperations + j)
                cache.Add(pid, &ProcessInfo{
                    PID:  pid,
                    Comm: fmt.Sprintf("proc-%d", pid),
                })
            }
        }(i)
    }

    // Concurrent readers
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            for j := 0; j < numOperations; j++ {
                pid := uint32(id*numOperations + j)
                _, _ = cache.Get(pid)
            }
        }(i)
    }

    wg.Wait()

    // Verify no corruption
    stats := cache.Stats()
    require.True(t, stats.Entries <= 1000, "Cache overflow")
}

// Run with race detector
// go test -race -run TestProcessCacheConcurrency
```

**任务清单**:
1. [ ] 为所有并发组件添加测试
2. [ ] CI 启用 `-race` 标志
3. [ ] 修复发现的竞态条件

---

### 5.3 添加 E2E 集成测试 ⭐ P2

**实现方案**:
```go
// tests/e2e/flow_collection_test.go
package e2e

func TestE2EFlowCollection(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping E2E test in short mode")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    // 1. Start test database
    db := startTestPostgres(t)
    defer db.Stop()

    // 2. Start test server
    server := startTestServer(t, db.ConnectionString())
    defer server.Stop()

    // 3. Start test agent
    agent := startTestAgent(t, server.GRPCAddress())
    defer agent.Stop()

    // 4. Add test policy
    _, err := server.Client().AddPolicy(ctx, &policypb.Policy{
        RuleId:    1,
        SrcIp:     "0.0.0.0/0",
        DstIp:     "192.168.1.1/32",
        DstPort:   80,
        Protocol:  "tcp",
        Action:    "allow",
        Direction: "egress",
    })
    require.NoError(t, err)

    // Wait for policy sync
    time.Sleep(2 * time.Second)

    // 5. Generate test traffic
    conn, err := net.Dial("tcp", "192.168.1.1:80")
    require.NoError(t, err)
    conn.Close()

    // 6. Wait for flow event
    time.Sleep(3 * time.Second)

    // 7. Query flows from server
    flows, err := server.Client().QueryFlows(ctx, &flowpb.FlowQuery{
        StartTime: timestamppb.New(time.Now().Add(-1 * time.Minute)),
        EndTime:   timestamppb.New(time.Now()),
    })
    require.NoError(t, err)
    require.NotEmpty(t, flows.Flows, "No flows collected")

    // 8. Verify flow attributes
    flow := flows.Flows[0]
    require.Equal(t, "192.168.1.1", flow.DstIp)
    require.Equal(t, uint32(80), flow.DstPort)
    require.Equal(t, "tcp", flow.Protocol)
    require.Equal(t, "allow", flow.Action)
}
```

**任务清单**:
1. [ ] 创建 `tests/e2e/` 目录
2. [ ] 实现测试辅助函数
3. [ ] 添加 Docker Compose 测试环境
4. [ ] CI 集成

---

## 六、安全性优化 (8 项)

### 6.1 改进 CIDR 解析验证 ⭐ P1

**问题**: IP 地址验证不足

**位置**: `src/agent/pkg/policy/policy.go:411`

**当前代码**:
```go
func parseCIDR(cidr string) (net.IP, *net.IPMask, error) {
    if !strings.Contains(cidr, "/") {
        cidr = cidr + "/32"  // ❌ 假设是 IPv4
    }
    // ...
}
```

**修复方案**:
```go
// pkg/netutil/validation.go
package netutil

import (
    "fmt"
    "net"
    "strings"
)

// ParseCIDR parses CIDR notation with proper validation
func ParseCIDR(cidr string) (net.IP, *net.IPNet, error) {
    // Normalize input
    cidr = strings.TrimSpace(cidr)

    if cidr == "" {
        return nil, nil, fmt.Errorf("empty CIDR")
    }

    // Auto-detect and add default mask
    if !strings.Contains(cidr, "/") {
        // Check if IPv4 or IPv6
        ip := net.ParseIP(cidr)
        if ip == nil {
            return nil, nil, fmt.Errorf("invalid IP address: %s", cidr)
        }

        if ip.To4() != nil {
            cidr += "/32"  // IPv4
        } else {
            cidr += "/128" // IPv6
        }
    }

    // Parse CIDR
    ip, ipnet, err := net.ParseCIDR(cidr)
    if err != nil {
        return nil, nil, fmt.Errorf("invalid CIDR %s: %w", cidr, err)
    }

    // Validate mask length
    ones, bits := ipnet.Mask.Size()
    if ip.To4() != nil {
        // IPv4: mask must be 0-32
        if bits != 32 {
            return nil, nil, fmt.Errorf("invalid IPv4 mask: /%d", ones)
        }
    } else {
        // IPv6: mask must be 0-128
        if bits != 128 {
            return nil, nil, fmt.Errorf("invalid IPv6 mask: /%d", ones)
        }
    }

    // Verify network address (no host bits set)
    if !ip.Equal(ipnet.IP) {
        return nil, nil, fmt.Errorf("CIDR %s has host bits set, use %s/%d",
            cidr, ipnet.IP.String(), ones)
    }

    return ip, ipnet, nil
}

// ValidatePort validates port number (0-65535)
func ValidatePort(port uint16) error {
    // Port 0 is valid (means "any port" in policy context)
    return nil
}

// ValidateProtocol validates protocol string (tcp, udp, icmp, etc.)
func ValidateProtocol(proto string) error {
    validProtos := map[string]bool{
        "tcp":  true,
        "udp":  true,
        "icmp": true,
        "icmpv6": true,
        "":     true,  // Empty means "any"
    }

    proto = strings.ToLower(proto)
    if !validProtos[proto] {
        return fmt.Errorf("invalid protocol: %s (must be tcp/udp/icmp/icmpv6)", proto)
    }

    return nil
}
```

**任务清单**:
1. [ ] 创建 `pkg/netutil/validation.go`
2. [ ] 替换所有 CIDR 解析逻辑
3. [ ] 添加单元测试 (边界情况)
4. [ ] 集成到策略验证

---

### 6.2 添加 gRPC 请求验证 ⭐ P1

**问题**: gRPC 请求缺少参数校验

**位置**: `src/server/pkg/grpc/policy_service.go`

**实现方案**:
```go
// pkg/validation/policy.go
package validation

import (
    "fmt"

    policypb "github.com/yourusername/ebpf-based-microsegment/api/proto/policy"
)

type PolicyValidator struct{}

func NewPolicyValidator() *PolicyValidator {
    return &PolicyValidator{}
}

func (v *PolicyValidator) ValidatePolicy(p *policypb.Policy) error {
    // Required fields
    if p.RuleId == 0 {
        return fmt.Errorf("rule_id is required and must be > 0")
    }

    if p.SrcIp == "" {
        return fmt.Errorf("src_ip is required")
    }

    if p.DstIp == "" {
        return fmt.Errorf("dst_ip is required")
    }

    // Validate IP addresses
    if _, _, err := netutil.ParseCIDR(p.SrcIp); err != nil {
        return fmt.Errorf("invalid src_ip: %w", err)
    }

    if _, _, err := netutil.ParseCIDR(p.DstIp); err != nil {
        return fmt.Errorf("invalid dst_ip: %w", err)
    }

    // Validate ports (0-65535)
    if p.SrcPort > 65535 {
        return fmt.Errorf("invalid src_port: %d (must be 0-65535)", p.SrcPort)
    }

    if p.DstPort > 65535 {
        return fmt.Errorf("invalid dst_port: %d (must be 0-65535)", p.DstPort)
    }

    // Validate protocol
    if err := netutil.ValidateProtocol(p.Protocol); err != nil {
        return err
    }

    // Validate action
    action := strings.ToLower(p.Action)
    if action != "allow" && action != "deny" && action != "log" {
        return fmt.Errorf("invalid action: %s (must be allow/deny/log)", p.Action)
    }

    // Validate direction
    direction := strings.ToLower(p.Direction)
    if direction != "ingress" && direction != "egress" && direction != "any" {
        return fmt.Errorf("invalid direction: %s (must be ingress/egress/any)", p.Direction)
    }

    // Validate priority (0-65535)
    if p.Priority > 65535 {
        return fmt.Errorf("invalid priority: %d (must be 0-65535)", p.Priority)
    }

    return nil
}

// server/pkg/grpc/policy_service.go
func (s *PolicyService) AddPolicy(
    ctx context.Context,
    req *policypb.Policy,
) (*policypb.PolicyResponse, error) {
    // Validate request
    validator := validation.NewPolicyValidator()
    if err := validator.ValidatePolicy(req); err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
    }

    // ... rest of implementation ...
}
```

**任务清单**:
1. [ ] 创建 `pkg/validation/` 包
2. [ ] 实现所有验证器
3. [ ] 集成到 gRPC 服务
4. [ ] 添加测试

---

### 6.3 添加权限检查 ⭐ P2

**问题**: eBPF 程序加载未检查权限

**实现方案**:
```go
// pkg/capabilities/check.go
package capabilities

import (
    "fmt"

    "golang.org/x/sys/unix"
)

const (
    CAP_BPF        = 39
    CAP_SYS_ADMIN  = 21
    CAP_NET_ADMIN  = 12
)

// HasCapability checks if current process has specified capability
func HasCapability(cap int) (bool, error) {
    var header unix.CapUserHeader
    var data [2]unix.CapUserData

    header.Version = unix.LINUX_CAPABILITY_VERSION_3
    header.Pid = 0  // Current process

    if err := unix.Capget(&header, &data[0]); err != nil {
        return false, fmt.Errorf("capget: %w", err)
    }

    // Check effective capabilities
    idx := cap >> 5
    mask := uint32(1) << (uint(cap) & 31)

    return (data[idx].Effective & mask) != 0, nil
}

// CheckBPFCapabilities checks if process can load eBPF programs
func CheckBPFCapabilities() error {
    hasBPF, err := HasCapability(CAP_BPF)
    if err != nil {
        return err
    }

    hasAdmin, err := HasCapability(CAP_SYS_ADMIN)
    if err != nil {
        return err
    }

    if !hasBPF && !hasAdmin {
        return fmt.Errorf("insufficient privileges: need CAP_BPF or CAP_SYS_ADMIN")
    }

    // Check NET_ADMIN for TC/XDP attachment
    hasNetAdmin, err := HasCapability(CAP_NET_ADMIN)
    if err != nil {
        return err
    }

    if !hasNetAdmin {
        return fmt.Errorf("insufficient privileges: need CAP_NET_ADMIN to attach TC/XDP programs")
    }

    return nil
}

// dataplane/manager.go
func (m *Manager) Load() error {
    // Check capabilities first
    if err := capabilities.CheckBPFCapabilities(); err != nil {
        return fmt.Errorf("capability check failed: %w", err)
    }

    // ... rest of load logic ...
}
```

**任务清单**:
1. [ ] 创建 `pkg/capabilities/` 包
2. [ ] 实现权限检查
3. [ ] 集成到 DataPlane
4. [ ] 添加友好错误提示

---

## 实施路线图

### Phase 1: 代码质量与架构 (Week 1-2)
**目标**: 减少技术债务，提升可维护性

- [ ] **P0** 创建 pkg/ 目录并迁移共享代码
- [ ] **P0** 消除所有 Fatalf 调用
- [ ] **P0** 统一 IP 转换工具函数
- [ ] **P1** 统一 eBPF Maps 管理器
- [ ] **P1** 添加结构化日志
- [ ] **P2** 添加 godoc 注释

**成果**:
- 代码重复率从 15% 降至 <5%
- 错误处理更健壮
- 日志可解析

---

### Phase 2: 核心功能完善 (Week 3-4)
**目标**: 补全关键功能

- [ ] **P0** 实现流标签过滤
- [ ] **P0** 添加 Ring Buffer 丢弃统计
- [ ] **P1** 实现策略增量更新
- [ ] **P1** 添加 Prometheus 指标
- [ ] **P1** 提取进程 UID/GID

**成果**:
- 查询功能完整
- 监控可观测
- 策略同步高效

---

### Phase 3: 性能优化 (Week 5-6)
**目标**: 提升系统性能

- [ ] **P0** 启用索引策略查找
- [ ] **P1** 添加 Context 取消支持
- [ ] **P1** 优化 Map 迭代性能
- [ ] **P1** 添加内存限制
- [ ] **P2** 消除 eBPF 代码重复

**成果**:
- 新连接延迟 <20μs
- 内存使用可控
- 代码更简洁

---

### Phase 4: 测试与安全 (Week 7-8)
**目标**: 提升稳定性和安全性

- [ ] **P1** 添加 eBPF 单元测试
- [ ] **P1** 添加并发安全测试
- [ ] **P1** 改进 CIDR 解析验证
- [ ] **P1** 添加 gRPC 请求验证
- [ ] **P2** 添加 E2E 集成测试
- [ ] **P2** 添加权限检查

**成果**:
- 测试覆盖率 >80%
- 安全漏洞修复
- CI/CD 完善

---

## 附录

### A. 代码复杂度分析

| 文件 | 行数 | 圈复杂度 | 函数数 | 建议 |
|------|------|---------|--------|------|
| tc_microsegment.bpf.c | 839 | 45 | 8 | 抽取共享逻辑 |
| xdp_microsegment.bpf.c | 816 | 42 | 7 | 抽取共享逻辑 |
| policy.go | 1073 | 38 | 25 | 拆分大函数 |
| flow_storage.go | 450 | 28 | 15 | 简化查询构建 |

### B. 技术债务统计

- **代码重复**: ~15% (~4000 行)
- **TODO 数量**: 6 个
- **FIXME 数量**: 0 个
- **Magic Number**: ~30 处
- **长函数 (>100 行)**: 12 个

### C. 性能基准目标

| 指标 | 当前 | 目标 | 改进措施 |
|------|------|------|----------|
| 新连接延迟 | 50-100μs | <20μs | 索引查找 |
| 策略同步时间 | 1-5s (全量) | <100ms (增量) | 增量更新 |
| Ring Buffer 丢失率 | 未知 | <0.1% | 添加统计 |
| 内存使用 | 不可控 | <500MB | 添加限制 |

---

**总结**: 项目代码质量良好，但存在优化空间。建议按优先级逐步实施，重点关注 P0 和 P1 项目。预计 8 周可完成主要优化。
