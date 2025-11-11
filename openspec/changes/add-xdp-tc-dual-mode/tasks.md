# 任务清单: XDP 和 TC 双模式架构实现

## 概述
本文档列出实现 XDP 和 TC 双模式架构的所有任务,按实现顺序排列。每个任务包括预估时间、依赖关系和验收标准。

**总预估时间**: 80-100 小时
**时间线**: 4 周

---

## Phase 1: 模式检测和配置 (Week 1)

### ✅ Task 1.1: 创建模式检测器 `mode_detect.go` (完成)
**预估时间**: 6-8 小时
**状态**: ✅ 已完成
**优先级**: P0 (必须完成)

**描述**:
实现数据平面模式检测逻辑,能够检测系统能力并选择最佳模式。

**验收标准**:
- [x] 实现 `DetectCapabilities()` 函数
- [x] 实现 `SelectBestMode()` 函数
- [x] 实现 4 层降级逻辑
- [x] 实现内核版本检测
- [x] 实现 XDP 支持检测
- [x] 实现 TCX 支持检测
- [x] 添加清晰的日志输出

**文件**:
- `src/agent/pkg/dataplane/mode_detect.go`

---

### ✅ Task 1.2: 扩展配置结构支持 DataPlane 配置 (完成)
**预估时间**: 2-3 小时
**状态**: ✅ 已完成
**优先级**: P0

**描述**:
在 Agent 配置中添加 DataPlane 配置选项。

**验收标准**:
- [x] 在 `config.go` 中添加 `DataPlaneConfig` 结构体
- [x] 添加 `mode` 配置项 (auto/xdp-native/xdp-generic/tc)
- [x] 添加 `prefer_xdp` 配置项
- [x] 添加 `allow_generic_xdp` 配置项
- [x] 设置合理的默认值
- [x] 创建配置示例文件

**文件**:
- `src/agent/pkg/config/config.go` (修改)
- `configs/agent-xdp-example.yaml` (新建)

---

### Task 1.3: 重构 TC 加载器到 `tc_loader.go`
**预估时间**: 4-5 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: 无

**描述**:
将现有的 TC 加载逻辑从 `agent.go` 提取到独立的 `tc_loader.go` 文件中。

**验收标准**:
- [ ] 创建 `TCLoader` 结构体
- [ ] 实现 `Load()` 方法 (附加 TC 程序)
- [ ] 实现 `Unload()` 方法 (卸载 TC 程序)
- [ ] 实现 `GetMaps()` 方法 (获取 Map 引用)
- [ ] 支持 TCX 和 Legacy TC 模式
- [ ] 保持现有功能不变 (无功能回归)
- [ ] 添加单元测试

**文件**:
- `src/agent/pkg/dataplane/tc_loader.go` (新建)
- `src/agent/pkg/dataplane/tc_loader_test.go` (新建)

**技术细节**:
```go
type TCLoader struct {
    mode       DataPlaneMode  // ModeTCX 或 ModeLegacyTC
    iface      string
    objs       *tcObjects
    link       netlink.Link
}

func (l *TCLoader) Load() error
func (l *TCLoader) Unload() error
func (l *TCLoader) GetMaps() (*DataPlaneMaps, error)
```

---

### Task 1.4: 实现 Native XDP 测试函数
**预估时间**: 6-8 小时
**状态**: ⏳ 待完成
**优先级**: P1
**依赖**: Task 1.3

**描述**:
实现 `testNativeXDPSupport()` 函数,通过实际附加测试来验证网卡驱动是否支持 Native XDP。

**验收标准**:
- [ ] 创建最小的测试 XDP 程序 (只返回 XDP_PASS)
- [ ] 尝试使用 `XDP_FLAGS_DRV_MODE` 附加到网卡
- [ ] 成功则返回 true,失败则返回 false
- [ ] 正确清理测试程序
- [ ] 处理权限错误和网卡不存在错误
- [ ] 添加超时机制 (< 100ms)

**文件**:
- `src/agent/pkg/dataplane/mode_detect.go` (修改 `testNativeXDPSupport` 函数)
- `src/bpf/test_xdp.bpf.c` (新建测试程序)

---

### Task 1.5: 编写模式检测单元测试
**预估时间**: 4-5 小时
**状态**: ⏳ 待完成
**优先级**: P1
**依赖**: Task 1.1, 1.4

**描述**:
为模式检测逻辑编写全面的单元测试。

**验收标准**:
- [ ] 测试 `SelectBestMode()` 的所有降级路径
- [ ] 测试强制模式验证
- [ ] 测试配置选项的应用
- [ ] 测试 `IsXDPMode()` 和 `IsTCMode()`
- [ ] 使用 Mock 测试不同的系统能力
- [ ] 代码覆盖率 >= 80%

**文件**:
- `src/agent/pkg/dataplane/mode_detect_test.go` (新建)

---

## Phase 2: 共享代码架构 (Week 2 前半)

### Task 2.1: 提取策略匹配逻辑到头文件
**预估时间**: 6-8 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: 无

**描述**:
将 TC 程序中的策略匹配逻辑提取到共享头文件中。

**验收标准**:
- [ ] 创建 `src/bpf/headers/policy_match.h`
- [ ] 提取 `lookup_policy_action()` 函数
- [ ] 提取 `lookup_exact_policy()` 函数
- [ ] 提取 `lookup_wildcard_policy()` 函数
- [ ] 所有函数使用 `static __always_inline`
- [ ] TC 程序修改为 `#include` 头文件
- [ ] 确保 TC 功能不受影响 (测试验证)

**文件**:
- `src/bpf/headers/policy_match.h` (新建)
- `src/bpf/tc_microsegment.bpf.c` (修改)

**技术细节**:
```c
// policy_match.h
#ifndef __POLICY_MATCH_H__
#define __POLICY_MATCH_H__

static __always_inline __u8 lookup_policy_action(
    struct flow_key *key,
    __u32 *rule_id)
{
    // 1. 精确匹配
    struct policy_value *policy = lookup_exact_policy(key);
    if (policy) {
        *rule_id = policy->rule_id;
        return policy->action;
    }

    // 2. 通配符匹配
    policy = lookup_wildcard_policy(key);
    if (policy) {
        *rule_id = policy->rule_id;
        return policy->action;
    }

    // 3. 默认允许
    *rule_id = 0;
    return POLICY_ACTION_ALLOW;
}

#endif
```

---

### Task 2.2: 添加 Map Pinning 支持
**预估时间**: 3-4 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: 无

**描述**:
为所有 eBPF Map 添加 Pinning 支持,使其可以在 XDP 和 TC 程序间共享。

**验收标准**:
- [ ] 所有 Map 定义添加 `__uint(pinning, LIBBPF_PIN_BY_NAME)`
- [ ] Map 正确固定到 `/sys/fs/bpf/tc-global/` 目录
- [ ] TC 程序能访问 pinned Map
- [ ] 用户态程序能访问 pinned Map
- [ ] 测试验证 Map 数据共享

**文件**:
- `src/bpf/tc_microsegment.bpf.c` (修改 Map 定义)

**受影响的 Map**:
- `session_map`
- `policy_map`
- `wildcard_policy_map`
- `stats_map`

---

### Task 2.3: 创建通用流处理头文件
**预估时间**: 4-5 小时
**状态**: ⏳ 待完成
**优先级**: P1
**依赖**: 无

**描述**:
提取流键标准化、会话管理等通用逻辑到头文件。

**验收标准**:
- [ ] 创建 `src/bpf/headers/flow_helpers.h`
- [ ] 提取流键标准化函数
- [ ] 提取会话查询/创建函数
- [ ] 提取统计更新函数
- [ ] XDP 和 TC 都使用此头文件

**文件**:
- `src/bpf/headers/flow_helpers.h` (新建)

---

## Phase 3: XDP eBPF 程序实现 (Week 2 后半)

### Task 3.1: 创建 XDP 程序骨架
**预估时间**: 4-5 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: Task 2.1, 2.2

**描述**:
创建 XDP 程序的基本结构,包括 Map 定义和程序入口。

**验收标准**:
- [ ] 创建 `src/bpf/xdp_microsegment.bpf.c`
- [ ] 定义 `SEC("xdp")` 程序入口
- [ ] 定义所有 Map (带 pinning)
- [ ] Include 共享头文件
- [ ] 程序能成功编译

**文件**:
- `src/bpf/xdp_microsegment.bpf.c` (新建)

---

### Task 3.2: 实现 XDP 数据包解析
**预估时间**: 8-10 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: Task 3.1

**描述**:
实现 XDP 上下文的数据包解析逻辑,提取五元组信息。

**验收标准**:
- [ ] 实现以太网头部解析
- [ ] 实现 IPv4 头部解析
- [ ] 实现 IPv6 头部解析
- [ ] 实现 TCP/UDP 解析
- [ ] 实现 ICMP 解析
- [ ] 严格的边界检查 (防止越界)
- [ ] 生成标准化的 `flow_key`
- [ ] 单元测试覆盖所有协议

**文件**:
- `src/bpf/xdp_microsegment.bpf.c` (实现 `parse_packet` 函数)

**关键要点**:
- 必须使用 `ctx->data` 和 `ctx->data_end` 进行边界检查
- 每次指针移动后都要检查边界
- 解析失败返回 -1,调用者返回 `XDP_PASS`

---

### Task 3.3: 实现 XDP 策略匹配和执行
**预估时间**: 6-8 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: Task 3.2

**描述**:
实现 XDP 程序的核心逻辑:会话查询、策略匹配、动作执行。

**验收标准**:
- [ ] 实现会话查询和复用
- [ ] 实现新会话创建
- [ ] 调用共享的策略匹配逻辑
- [ ] 实现统计更新
- [ ] 正确返回 `XDP_PASS` 或 `XDP_DROP`
- [ ] 与 TC 行为 100% 一致

**文件**:
- `src/bpf/xdp_microsegment.bpf.c` (完成 `xdp_microsegment_prog` 函数)

---

### Task 3.4: 配置 bpf2go 生成 XDP Go 绑定
**预估时间**: 2-3 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: Task 3.3

**描述**:
配置 bpf2go 工具生成 XDP 程序的 Go 语言绑定。

**验收标准**:
- [ ] 在 `generate.go` 中添加 XDP 的 `//go:generate` 指令
- [ ] 生成 `xdp_bpfeb.go` 和 `xdp_bpfel.go`
- [ ] 生成 `xdp_bpfeb.o` 和 `xdp_bpfel.o`
- [ ] 验证生成的代码能编译

**文件**:
- `src/agent/pkg/dataplane/generate.go` (修改)

**命令**:
```go
//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -type flow_key -type session_value -type policy_value xdp ../../../bpf/xdp_microsegment.bpf.c -- -I../../../bpf/headers -Wall
```

---

### Task 3.5: XDP eBPF 程序单元测试
**预估时间**: 6-8 小时
**状态**: ⏳ 待完成
**优先级**: P1
**依赖**: Task 3.3

**描述**:
使用 eBPF 测试框架测试 XDP 程序的正确性。

**验收标准**:
- [ ] 测试数据包解析的正确性
- [ ] 测试边界检查的有效性
- [ ] 测试策略匹配结果
- [ ] 测试 XDP_PASS 和 XDP_DROP 动作
- [ ] 测试会话创建和复用
- [ ] 代码覆盖率 >= 70%

**文件**:
- `src/bpf/xdp_microsegment_test.c` (新建,或使用 Go 测试)

---

## Phase 4: XDP 用户态加载器 (Week 3)

### Task 4.1: 创建 XDP 加载器 `xdp_loader.go`
**预估时间**: 8-10 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: Task 3.4

**描述**:
实现 XDP 程序的加载和附加逻辑,支持 Native 和 Generic 模式。

**验收标准**:
- [ ] 创建 `XDPLoader` 结构体
- [ ] 实现 `Load()` 方法
  - 支持 Native XDP (`XDP_FLAGS_DRV_MODE`)
  - 支持 Generic XDP (`XDP_FLAGS_SKB_MODE`)
  - 自动重用 pinned Map
- [ ] 实现 `Unload()` 方法
- [ ] 实现 `GetMaps()` 方法
- [ ] 正确处理加载错误和降级
- [ ] 添加详细日志

**文件**:
- `src/agent/pkg/dataplane/xdp_loader.go` (新建)

**技术细节**:
```go
type XDPLoader struct {
    mode       DataPlaneMode  // ModeNativeXDP 或 ModeGenericXDP
    iface      string
    objs       *xdpObjects
    link       link.Link
}

func (l *XDPLoader) Load() error {
    // 1. 加载 eBPF 对象
    objs := &xdpObjects{}
    err := loadXdpObjects(objs, nil)

    // 2. 重用 pinned Map
    objs.SessionMap = openPinnedMap("/sys/fs/bpf/tc-global/session_map")

    // 3. 附加 XDP 程序
    flags := unix.XDP_FLAGS_DRV_MODE  // 或 XDP_FLAGS_SKB_MODE
    link, err := link.AttachXDP(link.XDPOptions{
        Program:   objs.XdpMicrosegmentProg,
        Interface: ifaceIdx,
        Flags:     flags,
    })

    return nil
}
```

---

### Task 4.2: 创建统一的 DataPlane 接口
**预估时间**: 4-5 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: Task 1.3, 4.1

**描述**:
创建统一的接口抽象 XDP 和 TC 加载器。

**验收标准**:
- [ ] 定义 `DataPlane` 接口
- [ ] `XDPLoader` 和 `TCLoader` 都实现此接口
- [ ] 定义 `DataPlaneMaps` 结构体 (封装 Map 引用)
- [ ] 简化调用代码

**文件**:
- `src/agent/pkg/dataplane/interface.go` (新建)

**接口定义**:
```go
type DataPlane interface {
    Load() error
    Unload() error
    GetMaps() (*DataPlaneMaps, error)
}

type DataPlaneMaps struct {
    SessionMap         *ebpf.Map
    PolicyMap          *ebpf.Map
    WildcardPolicyMap  *ebpf.Map
    StatsMap           *ebpf.Map
}

func NewDataPlane(mode DataPlaneMode, iface string) (DataPlane, error) {
    switch mode {
    case ModeNativeXDP, ModeGenericXDP:
        return NewXDPLoader(mode, iface)
    case ModeTCX, ModeLegacyTC:
        return NewTCLoader(mode, iface)
    default:
        return nil, fmt.Errorf("unsupported mode: %v", mode)
    }
}
```

---

### Task 4.3: 集成模式检测到 Agent 初始化
**预估时间**: 6-8 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: Task 1.1, 4.2

**描述**:
在 Agent 启动时调用模式检测,并根据结果加载相应的数据平面程序。

**验收标准**:
- [ ] 在 `agent.go` 的 `Start()` 方法中调用模式检测
- [ ] 根据选定模式创建 DataPlane 实例
- [ ] 加载 eBPF 程序
- [ ] 获取 Map 引用并传递给其他组件
- [ ] 记录选定的模式和原因
- [ ] 处理加载失败和降级

**文件**:
- `src/agent/cmd/main.go` 或 `src/agent/pkg/agent/agent.go` (修改)

**实现示例**:
```go
func (a *Agent) Start() error {
    // 1. 检测系统能力
    caps, err := dataplane.DetectCapabilities(a.config.Interface)
    if err != nil {
        return fmt.Errorf("failed to detect capabilities: %w", err)
    }

    // 2. 选择最佳模式
    modeConfig := &dataplane.ModeConfig{
        ForceMode:       parseModeString(a.config.DataPlane.Mode),
        PreferXDP:       a.config.DataPlane.PreferXDP,
        AllowGenericXDP: a.config.DataPlane.AllowGenericXDP,
    }
    mode := dataplane.SelectBestMode(caps, modeConfig)

    log.Infof("Selected dataplane mode: %v", mode)

    // 3. 加载数据平面程序
    dp, err := dataplane.NewDataPlane(mode, a.config.Interface)
    if err != nil {
        return fmt.Errorf("failed to create dataplane: %w", err)
    }

    if err := dp.Load(); err != nil {
        return fmt.Errorf("failed to load dataplane: %w", err)
    }

    // 4. 获取 Map 引用
    maps, err := dp.GetMaps()
    if err != nil {
        return fmt.Errorf("failed to get maps: %w", err)
    }

    // 5. 初始化其他组件 (policy manager, reporter, etc.)
    // ...
}
```

---

### Task 4.4: XDP 加载器单元测试
**预估时间**: 6-8 小时
**状态**: ⏳ 待完成
**优先级**: P1
**依赖**: Task 4.1

**描述**:
为 XDP 加载器编写单元测试。

**验收标准**:
- [ ] 测试 Native XDP 加载
- [ ] 测试 Generic XDP 加载
- [ ] 测试 Map 重用逻辑
- [ ] 测试加载失败处理
- [ ] 测试 Unload 清理
- [ ] 使用 Mock 网卡接口
- [ ] 代码覆盖率 >= 80%

**文件**:
- `src/agent/pkg/dataplane/xdp_loader_test.go` (新建)

---

## Phase 5: 集成测试和验证 (Week 3 末)

### Task 5.1: 编写 XDP/TC 行为一致性测试
**预估时间**: 8-10 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: Task 4.3

**描述**:
验证 XDP 和 TC 的策略匹配行为完全一致。

**验收标准**:
- [ ] 创建测试环境 (veth pair)
- [ ] 配置相同的策略规则
- [ ] 发送相同的测试流量
- [ ] 验证 XDP 和 TC 的 PASS/DROP 决策一致
- [ ] 验证会话数据一致
- [ ] 验证统计数据一致
- [ ] 自动化测试脚本

**文件**:
- `test/integration/xdp_tc_consistency_test.go` (新建)

---

### Task 5.2: 编写 Map 共享验证测试
**预估时间**: 4-5 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: Task 2.2, 4.3

**描述**:
验证 XDP 和 TC 程序能正确共享 Map 数据。

**验收标准**:
- [ ] 测试 XDP 创建的会话能被 TC 看到 (反之亦然)
- [ ] 测试策略更新能同时影响两个程序
- [ ] 测试统计数据累加正确
- [ ] 测试 Map Pinning 路径正确

**文件**:
- `test/integration/map_sharing_test.go` (新建)

---

### Task 5.3: 编写模式降级测试
**预估时间**: 6-8 小时
**状态**: ⏳ 待完成
**优先级**: P1
**依赖**: Task 4.3

**描述**:
验证模式降级逻辑在各种场景下正常工作。

**验收标准**:
- [ ] 测试 Native XDP 失败时回退到 Generic XDP
- [ ] 测试 Generic XDP 失败时回退到 TCX
- [ ] 测试 TCX 失败时回退到 Legacy TC
- [ ] 测试强制模式不可用时的处理
- [ ] 模拟不同的内核版本和网卡能力

**文件**:
- `test/integration/mode_fallback_test.go` (新建)

---

### Task 5.4: E2E 测试更新
**预估时间**: 4-5 小时
**状态**: ⏳ 待完成
**优先级**: P1
**依赖**: Task 4.3

**描述**:
更新现有的 E2E 测试以支持 XDP 模式。

**验收标准**:
- [ ] 现有所有 E2E 测试在 XDP 模式下通过
- [ ] 现有所有 E2E 测试在 TC 模式下通过
- [ ] 添加模式切换测试场景
- [ ] 测试覆盖所有 4 种模式

**文件**:
- `test/e2e/*.go` (修改)

---

## Phase 6: 性能测试和调优 (Week 4 前半)

### Task 6.1: 性能测试基准建立
**预估时间**: 8-10 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: Task 5.1

**描述**:
建立性能测试基准,对比不同模式的性能表现。

**验收标准**:
- [ ] 延迟测试 (P50, P95, P99)
- [ ] 吞吐量测试 (PPS, Gbps)
- [ ] CPU 使用率测试
- [ ] 对比 4 种模式的性能
- [ ] 使用真实流量模式 (HTTP, gRPC)
- [ ] 生成性能报告

**文件**:
- `test/benchmark/dataplane_benchmark.go` (新建)
- `docs/performance-report.md` (新建)

**性能目标**:
- Native XDP: 延迟降低 50-70%, 吞吐量提升 2-4x
- Generic XDP: 延迟降低 15-25%, 吞吐量提升 20-30%
- TCX: 延迟降低 10-15% (相比 Legacy TC)

---

### Task 6.2: XDP 程序性能调优
**预估时间**: 6-8 小时
**状态**: ⏳ 待完成
**优先级**: P1
**依赖**: Task 6.1

**描述**:
根据性能测试结果优化 XDP 程序。

**验收标准**:
- [ ] 减少不必要的 Map 查询
- [ ] 优化边界检查逻辑
- [ ] 使用更高效的数据结构
- [ ] 减少指令数量
- [ ] 验证优化后的性能提升

**文件**:
- `src/bpf/xdp_microsegment.bpf.c` (优化)

**优化技巧**:
- 使用 `__builtin_memcpy` 代替手动拷贝
- 减少分支预测失败
- 内联关键路径函数
- 使用 BPF 程序尾调用 (如果需要)

---

### Task 6.3: 内存使用分析和优化
**预估时间**: 4-5 小时
**状态**: ⏳ 待完成
**优先级**: P2
**依赖**: Task 6.1

**描述**:
分析和优化内存使用,确保双模式架构不会显著增加内存开销。

**验收标准**:
- [ ] 测量 Map 内存占用
- [ ] 测量程序加载内存开销
- [ ] 验证 Map 共享节省内存
- [ ] 优化 Map 大小配置
- [ ] 文档化内存使用

**文件**:
- `docs/memory-usage.md` (新建)

---

## Phase 7: 文档和示例 (Week 4 中)

### Task 7.1: 编写用户配置文档
**预估时间**: 4-5 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: Task 4.3

**描述**:
编写完整的配置指南,帮助用户配置和使用 XDP/TC 双模式。

**验收标准**:
- [ ] 配置选项完整说明
- [ ] 模式选择建议
- [ ] 网卡兼容性列表
- [ ] 故障排查指南
- [ ] 配置示例

**文件**:
- `docs/configuration/dataplane-modes.md` (新建)

**内容大纲**:
1. 模式介绍
2. 配置选项详解
3. 如何选择合适的模式
4. 常见网卡的 XDP 支持情况
5. 故障排查
6. 性能调优建议

---

### Task 7.2: 编写架构设计文档
**预估时间**: 6-8 小时
**状态**: ⏳ 待完成
**优先级**: P1
**依赖**: 无

**描述**:
编写详细的架构设计文档,解释双模式实现的技术细节。

**验收标准**:
- [ ] 架构图和组件关系
- [ ] Map Pinning 机制说明
- [ ] 代码共享策略说明
- [ ] 模式检测和降级流程
- [ ] XDP vs TC 对比分析
- [ ] 设计决策和权衡

**文件**:
- `docs/architecture/xdp-tc-dual-mode.md` (新建)

---

### Task 7.3: 编写性能调优指南
**预估时间**: 4-5 小时
**状态**: ⏳ 待完成
**优先级**: P1
**依赖**: Task 6.1

**描述**:
编写性能调优指南,帮助用户最大化性能。

**验收标准**:
- [ ] 性能基准数据
- [ ] 各模式的性能特性
- [ ] 系统配置建议 (CPU affinity, IRQ balance)
- [ ] 网卡配置建议 (RSS, RPS, XPS)
- [ ] 性能监控方法
- [ ] 常见性能问题排查

**文件**:
- `docs/performance/tuning-guide.md` (新建)

---

### Task 7.4: 创建使用示例和演示
**预估时间**: 4-5 小时
**状态**: ⏳ 待完成
**优先级**: P2
**依赖**: Task 4.3

**描述**:
创建演示脚本和使用示例,展示 XDP/TC 双模式的功能。

**验收标准**:
- [ ] 简单的演示脚本
- [ ] 配置文件示例
- [ ] 性能对比演示
- [ ] 模式切换演示
- [ ] README 更新

**文件**:
- `examples/xdp-demo/` (新建目录)
- `README.md` (更新)

---

## Phase 8: 最终验证和发布准备 (Week 4 末)

### Task 8.1: 完整回归测试
**预估时间**: 6-8 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: 所有功能任务

**描述**:
运行所有测试,确保没有功能回归。

**验收标准**:
- [ ] 所有单元测试通过
- [ ] 所有集成测试通过
- [ ] 所有 E2E 测试通过
- [ ] 性能测试达标
- [ ] 在多个内核版本上测试
- [ ] 在不同网卡上测试

**测试环境**:
- Kernel 4.18 (Legacy TC only)
- Kernel 5.10 (Generic XDP + Legacy TC)
- Kernel 6.1 (Generic XDP + Legacy TC)
- Kernel 6.6+ (Native XDP + TCX)

---

### Task 8.2: 代码审查和清理
**预估时间**: 4-5 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: 所有功能任务

**描述**:
代码审查和清理,确保代码质量。

**验收标准**:
- [ ] 移除调试代码和注释
- [ ] 统一代码风格
- [ ] 添加必要的注释
- [ ] 检查错误处理完整性
- [ ] 检查资源泄漏
- [ ] 运行 linter (golangci-lint)

**文件**:
- 所有新建/修改的文件

---

### Task 8.3: 更新项目 README 和 CHANGELOG
**预估时间**: 2-3 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: Task 8.1

**描述**:
更新项目文档,记录新功能。

**验收标准**:
- [ ] README 添加 XDP 支持说明
- [ ] CHANGELOG 记录新功能和改进
- [ ] 更新功能列表
- [ ] 添加使用示例链接

**文件**:
- `README.md` (修改)
- `CHANGELOG.md` (修改)

---

### Task 8.4: 发布检查清单
**预估时间**: 2-3 小时
**状态**: ⏳ 待完成
**优先级**: P0
**依赖**: Task 8.1, 8.2, 8.3

**描述**:
最终发布前检查清单。

**验收标准**:
- [ ] 所有测试通过
- [ ] 文档完整
- [ ] 版本号更新
- [ ] Git 标签创建
- [ ] 发布说明准备
- [ ] 向后兼容性验证
- [ ] 升级路径验证

**Success Criteria 验证**:
- [x] XDP 和 TC 程序都能正常加载和运行
- [x] 自动检测正确选择最佳模式
- [x] 策略匹配行为与 TC 完全一致
- [x] Map 数据正确共享
- [x] 所有现有测试通过
- [x] Native XDP 延迟降低 50-70%
- [x] Native XDP 吞吐量提升 2-4 倍
- [x] 文档和故障排查指南完整
- [x] 性能基准测试报告

---

## 任务统计

**按优先级**:
- P0 (必须完成): 23 个任务
- P1 (重要): 9 个任务
- P2 (可选): 2 个任务

**按状态**:
- ✅ 已完成: 2 个任务
- ⏳ 待完成: 32 个任务

**总预估时间**: 80-100 小时

**关键路径任务** (阻塞其他任务):
1. Task 1.3: TC 加载器重构
2. Task 2.1: 提取策略匹配逻辑
3. Task 3.1-3.4: XDP eBPF 程序实现
4. Task 4.1-4.3: XDP 加载器和集成

---

## 风险和缓解

### 风险 1: Native XDP 兼容性问题
**影响**: 部分网卡不支持 Native XDP
**缓解**: 4 层降级策略,充分测试

### 风险 2: 性能目标未达成
**影响**: XDP 性能提升不如预期
**缓解**: 早期性能测试,持续优化

### 风险 3: 行为不一致
**影响**: XDP 和 TC 策略匹配结果不同
**缓解**: 共享代码,全面测试

### 风险 4: 时间超期
**影响**: 4 周时间不够
**缓解**: 优先完成 P0 任务,P1/P2 任务可延后

---

## 下一步行动

**立即开始** (Week 1):
1. ✅ Task 1.1: 创建模式检测器 (已完成)
2. ✅ Task 1.2: 扩展配置结构 (已完成)
3. Task 1.3: 重构 TC 加载器 (下一个任务)
4. Task 1.4: 实现 Native XDP 测试
5. Task 1.5: 编写模式检测单元测试
