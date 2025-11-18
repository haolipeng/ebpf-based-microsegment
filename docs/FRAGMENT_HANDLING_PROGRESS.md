# 分片数据包处理功能 - 实现进度

**文档版本**: v1.1
**创建时间**: 2025-11-18
**最后更新**: 2025-11-18
**当前完成度**: 约 95%

---

## ✅ 已完成的工作

### 1. 核心头文件 (100%)

#### `src/bpf/headers/fragment_tracking.h`
- ✅ 分片数据结构定义
  - `struct frag_key`: 分片标识符（src_ip, dst_ip, frag_id, protocol）
  - `struct frag_value`: 首片缓存信息（完整5元组、策略动作、时间戳）
  - `struct frag_config`: 分片处理配置（模式、日志、超时）
  - `enum frag_stats_key`: 统计类型枚举

- ✅ IPv4 分片检测函数
  - `is_ipv4_fragment()`: 检测是否为 IPv4 分片
  - `is_ipv4_first_fragment()`: 检测是否为首片
  - `is_ipv4_subsequent_fragment()`: 检测是否为后续片段
  - `extract_ipv4_frag_key()`: 提取分片键

- ✅ IPv6 分片检测函数
  - `is_ipv6_fragment()`: 检测是否包含分片扩展头
  - `is_ipv6_first_fragment()`: 检测是否为首片
  - `is_ipv6_subsequent_fragment()`: 检测是否为后续片段
  - `extract_ipv6_frag_key()`: 提取分片键

- ✅ 分片处理模式定义
  - `FRAG_MODE_STRICT` (0): 丢弃所有分片（最安全）
  - `FRAG_MODE_NORMAL` (1): 允许首片（匹配策略），拒绝后续片段（推荐）
  - `FRAG_MODE_PERMISSIVE` (2): 允许首片和后续片段（最不安全）

#### `src/bpf/headers/fragment_handler.h`
- ✅ 分片处理核心逻辑
  - `update_frag_stats()`: 更新分片统计
  - `handle_ipv4_fragment()`: IPv4 分片处理函数
  - `handle_ipv6_fragment()`: IPv6 分片处理函数

### 2. TC 程序集成 (100%)

#### `src/bpf/tc_microsegment.bpf.c`
- ✅ 添加分片 eBPF Maps
  - `frag_state_map`: LRU_HASH，存储首片信息
  - `frag_config_map`: 分片配置
  - `frag_stats_map`: Per-CPU 统计

- ✅ 引入分片处理头文件
  - `fragment_tracking.h`
  - `fragment_handler.h`

- ✅ 主处理逻辑集成
  - 在慢速路径（新会话）中添加分片检测
  - **后续片段处理**：
    - 查找首片缓存
    - 根据配置模式决定允许/拒绝
    - 更新统计并立即返回
  - **首片处理**：
    - 继续正常的 NAT 检测和策略匹配
    - 在创建会话后缓存分片状态
    - 更新统计

- ✅ 编译测试通过
  - TC 程序已成功编译
  - 仅有警告，无错误

### 3. XDP 程序集成 (100%)

#### `src/bpf/xdp_microsegment.bpf.c`
- ✅ 添加分片 eBPF Maps（与 TC 共享）
  - `frag_state_map`: 已添加并 pin
  - `frag_config_map`: 已添加并 pin
  - `frag_stats_map`: 已添加并 pin

- ✅ 引入头文件
  - `fragment_tracking.h`: 已添加
  - `fragment_handler.h`: 已添加

- ✅ 主处理逻辑集成
  - 在慢速路径中添加分片检测逻辑
  - 后续片段处理：查找缓存并应用策略
  - 首片处理：继续策略匹配，然后缓存结果
  - 编译测试通过

### 4. 用户态 API 实现 (100%)

#### `src/agent/pkg/dataplane/fragment.go` (新文件，245 行)
- ✅ 数据结构定义
  - `FragmentMode`: 分片处理模式枚举
  - `FragmentConfig`: 分片配置结构
  - `FragmentStats`: 分片统计结构
  - `fragConfigBPF`: BPF map 对应结构

- ✅ API 方法实现
  - `SetFragmentConfig()`: 配置分片处理模式
  - `GetFragmentConfig()`: 获取当前配置
  - `GetFragmentStats()`: 获取分片统计
  - `ResetFragmentStats()`: 重置统计计数器

- ✅ Helper 方法
  - `getFragmentConfigMap()`: 获取配置 map
  - `getFragmentStatsMap()`: 获取统计 map
  - `readFragmentStat()`: 读取 per-CPU 统计

### 5. DataPlaneMaps 和 Loaders 更新 (100%)

#### `src/agent/pkg/dataplane/interface.go`
- ✅ 添加分片 Maps 到 DataPlaneMaps
  - `FragStateMap`: 分片状态表
  - `FragConfigMap`: 分片配置表
  - `FragStatsMap`: 分片统计表

#### `src/agent/pkg/dataplane/tc_loader.go`
- ✅ 更新 `GetMaps()` 方法
  - 返回分片相关的 maps

#### `src/agent/pkg/dataplane/xdp_loader.go`
- ✅ 更新 `GetMaps()` 方法
  - 返回分片相关的 maps

---

## 🔨 剩余工作

### 4. XDP 程序完整集成 (40% 完成)

**需要完成的步骤**：

1. **引入 fragment_handler.h**
   - 在 `flow_processing.h` 之后添加：
     ```c
     #include "headers/fragment_handler.h"
     ```

2. **在主函数中集成分片处理**
   - 找到 `xdp_microsegment_prog()` 函数
   - 在慢速路径中（policy lookup 之前）添加与 TC 相同的分片检测逻辑
   - 参考 TC 程序的第 609-736 行

3. **缓存首片策略动作**
   - 在创建会话后添加分片状态缓存
   - 参考 TC 程序的第 789-803 行

### 5. 用户态 API 实现 (0%)

**文件位置**: `src/agent/pkg/dataplane/fragment.go`（新文件）

**需要实现的功能**：

```go
// FragmentConfig - 分片配置结构
type FragmentConfig struct {
    Mode       FragmentMode `json:"mode"`        // STRICT/NORMAL/PERMISSIVE
    LogEvents  bool         `json:"log_events"`  // 是否记录分片事件
    TimeoutNs  uint64       `json:"timeout_ns"`  // 分片超时时间（纳秒）
}

// FragmentMode - 分片处理模式
type FragmentMode uint8
const (
    FragmentModeStrict FragmentMode = 0
    FragmentModeNormal FragmentMode = 1
    FragmentModePermissive FragmentMode = 2
)

// FragmentStats - 分片统计
type FragmentStats struct {
    FirstFragments       uint64 `json:"first_fragments"`
    SubsequentFragments  uint64 `json:"subsequent_fragments"`
    FragmentsAllowed     uint64 `json:"fragments_allowed"`
    FragmentsDenied      uint64 `json:"fragments_denied"`
    FragmentsTimeout     uint64 `json:"fragments_timeout"`
    CacheHits            uint64 `json:"cache_hits"`
    CacheMisses          uint64 `json:"cache_misses"`
    IPv4Fragments        uint64 `json:"ipv4_fragments"`
    IPv6Fragments        uint64 `json:"ipv6_fragments"`
}

// API 方法
func (dp *DataPlane) SetFragmentConfig(config *FragmentConfig) error
func (dp *DataPlane) GetFragmentConfig() (*FragmentConfig, error)
func (dp *DataPlane) GetFragmentStats() (*FragmentStats, error)
func (dp *DataPlane) ResetFragmentStats() error
```

### 6. DataPlaneMaps 更新 (0%)

**文件**: `src/agent/pkg/dataplane/interface.go`

**需要添加**：
```go
type DataPlaneMaps struct {
    // ... existing maps ...
    FragStateMap      *ebpf.Map  // Fragment state tracking
    FragConfigMap     *ebpf.Map  // Fragment configuration
    FragStatsMap      *ebpf.Map  // Fragment statistics
}
```

### 7. Loader 更新 (0%)

**文件**:
- `src/agent/pkg/dataplane/tc_loader.go`
- `src/agent/pkg/dataplane/xdp_loader.go`

**需要更新 GetMaps() 方法**：
```go
return &DataPlaneMaps{
    // ... existing maps ...
    FragStateMap:  l.objs.FragStateMap,
    FragConfigMap: l.objs.FragConfigMap,
    FragStatsMap:  l.objs.FragStatsMap,
}, nil
```

### 8. 分片超时清理 (0%)

**实现方式**：类似 `session.TimeoutManager`

**新文件**: `src/agent/pkg/fragment/cleaner.go`

**功能**：
- 定期扫描 `frag_state_map`
- 删除超时的分片条目（基于 timestamp）
- 更新 `FRAG_STAT_FRAGMENTS_TIMEOUT` 统计

### 9. 测试脚本 (0%)

**文件**: `tests/fragment/fragment_test.sh`

**测试场景**：
1. **首片测试**：发送首片，验证策略匹配
2. **后续片段测试**：发送后续片段，验证缓存查找
3. **模式测试**：测试 STRICT/NORMAL/PERMISSIVE 三种模式
4. **超时测试**：验证分片超时清理
5. **统计测试**：验证统计计数器正确性

---

## 📝 下一步操作指南

### 步骤 1: 完成 XDP 程序集成

1. 编辑 `src/bpf/xdp_microsegment.bpf.c`
2. 在 `#include "headers/flow_processing.h"` 之后添加：
   ```c
   #include "headers/fragment_handler.h"
   ```
3. 找到 `xdp_microsegment_prog()` 函数
4. 复制 TC 程序的分片处理代码（609-736 行和 789-803 行）
5. 调整为 XDP 上下文（`struct xdp_md *ctx` 而不是 `struct __sk_buff *skb`）

### 步骤 2: 编译测试

```bash
cd /home/work/ebpf-based-microsegment/src/agent
go generate ./pkg/dataplane
```

### 步骤 3: 实现用户态 API

1. 创建 `src/agent/pkg/dataplane/fragment.go`
2. 实现上述 API 方法
3. 更新 `interface.go` 添加分片 maps
4. 更新 loaders 的 `GetMaps()` 方法

### 步骤 4: 实现分片清理器

1. 创建 `src/agent/pkg/fragment/` 目录
2. 实现 `cleaner.go`
3. 在 Agent 启动时初始化

### 步骤 5: 编写测试

1. 创建测试脚本
2. 验证各种场景

---

## 技术细节

### 分片检测流程

```
数据包到达
    ↓
提取 flow key
    ↓
检测是否为分片 ←─────────────────┐
    ↓                            │
    ├─ 不是分片 ────→ 正常处理    │
    │                            │
    ├─ 是首片 ─────────→ 继续策略匹配
    │                       ↓
    │                  创建会话
    │                       ↓
    │                  缓存分片状态 ─┘
    │
    └─ 是后续片段 ───→ 查找缓存
                         ↓
                    ├─ 缓存命中 ──→ 应用策略
                    └─ 缓存未命中 ─→ 拒绝（安全）
```

### 分片缓存结构

```c
// 分片键 (用于查找)
struct frag_key {
    src_ip[4], dst_ip[4],  // IP 地址
    frag_id,               // 分片 ID
    protocol,              // 协议
    ip_version             // IP 版本
};

// 分片值 (缓存内容)
struct frag_value {
    complete_key,   // 完整 5 元组 (含端口)
    timestamp,      // 时间戳 (用于超时)
    policy_action   // 策略动作 (ALLOW/DENY)
};
```

### 性能考虑

- **快速路径不受影响**：分片检测只在新会话（慢速路径）中进行
- **后续片段快速处理**：单次 map 查找即可决定动作
- **共享缓存**：TC 和 XDP 共享 `frag_state_map`，首片在任一程序处理后，后续片段在另一程序也能查到
- **LRU 自动淘汰**：使用 LRU_HASH 自动清理旧条目

---

## 参考文档

- [eBPF 微隔离路线图](./EBPF_MICROSEGMENTATION_ROADMAP.md) - 第 84-116 行
- [ZFW 分片处理分析](./zfw-analysis/zfw-deep-dive.md) - 第 914-933 行
- RFC 791 (IPv4): IP Fragmentation
- RFC 8200 (IPv6): Fragment Extension Header
