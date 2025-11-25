# eBPF Agent 层代码审核报告

**审核日期**: 2025年1月
**审核人**: eBPF 技术专家
**审核范围**: `/src/agent` - 所有 eBPF 相关代码
**状态**: 已完成

---

## 目录

1. [执行摘要](#执行摘要)
2. [严重问题](#严重问题)
3. [中等优先级问题](#中等优先级问题)
4. [轻微问题](#轻微问题)
5. [附录：审核文件清单](#附录审核文件清单)

---

## 执行摘要

本文档对 agent 层代码进行了全面审核，重点关注 eBPF 实现的质量、安全性和性能。审核共发现 **18 个问题**，按严重程度分布如下：

| 严重程度 | 数量 | 影响 |
|----------|------|------|
| 严重 | 4 | 内存耗尽、数据损坏、安全绕过 |
| 中等 | 6 | 资源泄漏、竞态条件、功能缺陷 |
| 轻微 | 8 | 代码质量、日志、可维护性 |

**主要发现**:
- 策略编译器存在组合爆炸风险
- 进程监控使用了不安全的结构体大小假设
- 默认允许策略可能违反零信任原则
- XDP 模式缺少出站策略支持

### CRITICAL-002: 进程监控使用不安全的结构体大小

**位置**: `pkg/process/monitor.go:231`

**严重程度**: 🔴 严重

**问题描述**:
进程监控使用 `unsafe.Sizeof()` 来确定从 eBPF ring buffer 接收的事件的预期大小。由于不同的内存对齐规则，Go 结构体布局可能与 C 结构体布局不同。

**问题代码**:
```go
// monitor.go 第 231 行
func (m *ProcessMonitor) parseProcessEvent(rawData []byte) (*ProcessEvent, error) {
    expectedSize := int(unsafe.Sizeof(ProcessEvent{}))
    if len(rawData) < expectedSize {
        return nil, fmt.Errorf("无效的事件大小: 收到 %d, 预期 %d", len(rawData), expectedSize)
    }
    // ...
}
```

**影响**:
- 大小不匹配时的静默数据损坏
- 流事件中的进程信息错误
- 如果进程匹配失败可能导致策略绕过

**修复方案**:

```go
// 定义与 C 结构体匹配的显式大小常量
// 必须与 process_monitor.h 中的 struct process_event 匹配
const ProcessEventSize = 4 + 16 + 8 + 64 + 4 // PID + Comm + ExecTime + ContainerID + Flags = 96 字节

// ProcessEvent 表示 eBPF 进程事件结构
// 重要：此结构必须与 process_monitor.h 中的 C 定义匹配
type ProcessEvent struct {
    PID         uint32      // 4 字节
    Comm        [16]byte    // 16 字节
    ExecTime    uint64      // 8 字节
    ContainerID [64]byte    // 64 字节
    Flags       uint32      // 4 字节
}

// 添加编译时大小验证
func init() {
    // 验证 Go 结构体大小与预期的 C 结构体大小匹配
    goSize := int(unsafe.Sizeof(ProcessEvent{}))
    if goSize != ProcessEventSize {
        panic(fmt.Sprintf(
            "ProcessEvent 大小不匹配: Go=%d, 预期=%d。请检查结构定义是否与 C 头文件匹配。",
            goSize, ProcessEventSize,
        ))
    }
}

func (m *ProcessMonitor) parseProcessEvent(rawData []byte) (*ProcessEvent, error) {
    if len(rawData) < ProcessEventSize {
        return nil, fmt.Errorf("无效的事件大小: 收到 %d, 预期 %d", len(rawData), ProcessEventSize)
    }
    // ... 其余解析逻辑
}
```

**额外建议**:
添加构建时验证脚本，比较 Go 结构体布局与 C 头文件：

```bash
#!/bin/bash
# scripts/verify-struct-sizes.sh
# 作为 CI/CD 流水线的一部分运行

go run ./tools/verify-struct-sizes/main.go
```

---

### CRITICAL-003: 进程监控在 Select 中使用阻塞读取

**位置**: `pkg/process/monitor.go:167-202`

**严重程度**: 🔴 严重

**问题描述**:
收集循环在 `select` 语句的 `default` 分支中放置了阻塞的 `ringBuf.Read()` 调用。这会阻止对 context 取消的及时响应。

**问题代码**:
```go
func (m *ProcessMonitor) collectLoop() {
    defer m.wg.Done()

    for {
        select {
        case <-m.ctx.Done():
            log.Println("[Process Monitor] 收集循环已停止")
            return
        default:
            // 这会阻塞！Read 返回之前不会注意到 Context 取消
            record, err := m.ringBuf.Read()
            // ...
        }
    }
}
```

**影响**:
- 优雅关闭可能无限期挂起
- 资源清理延迟
- Agent 重启时间过长

**修复方案**:

```go
func (m *ProcessMonitor) collectLoop() {
    defer m.wg.Done()
    log.Println("[Process Monitor] 启动事件收集循环...")

    // 创建单独的 goroutine 用于 ring buffer 读取
    recordCh := make(chan ringbuf.Record, 100)
    errCh := make(chan error, 1)

    go func() {
        for {
            record, err := m.ringBuf.Read()
            if err != nil {
                if err == ringbuf.ErrClosed {
                    close(recordCh)
                    return
                }
                select {
                case errCh <- err:
                default:
                }
                continue
            }
            select {
            case recordCh <- record:
            case <-m.ctx.Done():
                return
            }
        }
    }()

    for {
        select {
        case <-m.ctx.Done():
            log.Println("[Process Monitor] 收集循环被 context 停止")
            return

        case record, ok := <-recordCh:
            if !ok {
                log.Println("[Process Monitor] Ring buffer 已关闭")
                return
            }

            event, err := m.parseProcessEvent(record.RawSample)
            if err != nil {
                log.Printf("[Process Monitor] 解析进程事件错误: %v", err)
                m.incrementDropped()
                continue
            }

            if err := m.processEvent(event); err != nil {
                log.Printf("[Process Monitor] 处理事件错误: %v", err)
                m.incrementDropped()
                continue
            }
            m.incrementProcessed()

        case err := <-errCh:
            log.Printf("[Process Monitor] Ring buffer 错误: %v", err)
            m.incrementDropped()
        }
    }
}

// 更新 Stop 方法，首先关闭 ring buffer
func (m *ProcessMonitor) Stop() error {
    log.Println("[Process Monitor] 正在停止进程监控...")

    // 首先关闭 ring buffer 以解除读取阻塞
    if m.ringBuf != nil {
        if err := m.ringBuf.Close(); err != nil {
            log.Printf("[Process Monitor] 关闭 ring buffer 错误: %v", err)
        }
    }

    // 然后取消 context
    m.cancel()

    // 带超时等待 goroutine
    done := make(chan struct{})
    go func() {
        m.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        log.Println("[Process Monitor] 进程监控已成功停止")
    case <-time.After(5 * time.Second):
        log.Println("[Process Monitor] 等待 goroutine 停止超时")
    }

    return nil
}
```

---

### CRITICAL-004: XDP 模式缺少出站支持

**位置**: `pkg/dataplane/xdp_loader.go`

**严重程度**: 🔴 严重

**问题描述**:
XDP (eXpress Data Path) 只能在驱动层处理入站流量。在纯 XDP 模式下运行时，无法执行出站策略，造成重大安全漏洞。

**影响**:
- XDP 模式下出站策略被静默忽略
- 即使有出站拒绝策略，数据泄露仍然可能发生
- 安全态势比预期弱

**修复方案选项 1**: 混合模式（推荐）

```go
// pkg/dataplane/xdp_loader.go

type XDPLoader struct {
    mode         DataPlaneMode
    iface        string
    ifaceIdx     int
    objs         *xdpbpfObjects
    xdpLink      link.Link
    tcEgressLink link.Link          // 新增：混合模式的 TC 出站
    tcObjs       *bpfObjects        // 新增：出站用的 TC 对象
    pinConfig    *MapPinConfig
    hybridMode   bool               // 新增：启用混合模式
}

func NewXDPLoader(mode DataPlaneMode, iface string, ifaceIdx int) (*XDPLoader, error) {
    // ...现有验证...

    return &XDPLoader{
        mode:       mode,
        iface:      iface,
        ifaceIdx:   ifaceIdx,
        pinConfig:  DefaultMapPinConfig(),
        hybridMode: true, // 默认启用以确保安全
    }, nil
}

func (l *XDPLoader) Load() error {
    // 为入站加载 XDP（现有逻辑）
    if err := l.loadXDP(); err != nil {
        return err
    }

    // 在混合模式下为出站加载 TC
    if l.hybridMode {
        if err := l.loadTCEgress(); err != nil {
            log.Warnf("混合模式下加载 TC 出站失败: %v", err)
            log.Warn("出站策略将不会被执行！")
            // 不失败 - 为了向后兼容继续仅入站模式
        }
    }

    return nil
}

func (l *XDPLoader) loadTCEgress() error {
    // 加载 TC BPF 对象
    tcObjs := &bpfObjects{}
    opts := &ebpf.CollectionOptions{
        Maps: ebpf.MapOptions{
            PinPath: l.pinConfig.PinPath, // 与 XDP 共享 maps
        },
    }

    if err := loadBpfObjects(tcObjs, opts); err != nil {
        return fmt.Errorf("加载 TC eBPF 对象失败: %w", err)
    }
    l.tcObjs = tcObjs

    // 设置 clsact qdisc
    qdisc := &netlink.GenericQdisc{
        QdiscAttrs: netlink.QdiscAttrs{
            LinkIndex: l.ifaceIdx,
            Handle:    netlink.MakeHandle(0xffff, 0),
            Parent:    netlink.HANDLE_CLSACT,
        },
        QdiscType: "clsact",
    }

    if err := netlink.QdiscAdd(qdisc); err != nil && !isFileExistsError(err) {
        return fmt.Errorf("添加 clsact qdisc 失败: %w", err)
    }

    // 附加 TC 出站过滤器
    // ...（类似 tc_loader.go 中的 attachLegacyTCEgress）

    log.Infof("✓ TC 出站程序已附加到 %s（XDP 混合模式）", l.iface)
    return nil
}
```

**修复方案选项 2**: 清晰的文档和验证

```go
// pkg/dataplane/manager.go

func (m *Manager) Load() error {
    mode := SelectBestMode(m.caps, m.config)

    // 警告 XDP 出站限制
    if IsXDPMode(mode) {
        log.Warn("======================================================")
        log.Warn("已选择 XDP 模式 - 出站策略将不会被执行")
        log.Warn("要完整执行策略，请使用 TC 模式或混合模式")
        log.Warn("======================================================")
    }

    // ... 现有逻辑
}

// 添加方法检查是否支持出站
func (m *Manager) SupportsEgress() bool {
    return IsTCMode(m.currentMode) ||
           (IsXDPMode(m.currentMode) && m.loader.(*XDPLoader).hybridMode)
}
```

---

## 中等优先级问题

### MEDIUM-001: TC Loader 出站附加失败时资源泄漏

**位置**: `pkg/dataplane/tc_loader.go:74-78`

**严重程度**: 🟡 中等

**问题描述**:
当 TCX 出站附加失败时，代码分离了入站但没有关闭 eBPF 对象，导致资源泄漏。

**问题代码**:
```go
if err := l.attachTCXEgress(); err != nil {
    l.detachTCXIngress() // 清理入站
    return err           // 但 l.objs 没有关闭！
}
```

**修复方案**:
```go
func (l *TCLoader) Load() error {
    // ... 加载对象 ...

    switch l.mode {
    case ModeTCX:
        if err := l.attachTCXIngress(); err != nil {
            l.objs.Close() // 入站失败时关闭
            return err
        }
        if err := l.attachTCXEgress(); err != nil {
            l.detachTCXIngress()
            l.objs.Close() // 出站失败时关闭
            l.objs = nil
            return err
        }
        return nil
    // ... ModeLegacyTC 类似处理
    }
}
```

---

### MEDIUM-002: Map Pinning 目录权限过于宽松

**位置**: `pkg/dataplane/map_pinning.go:65`

**严重程度**: 🟡 中等

**问题描述**:
BPF pin 目录以 `0755` 权限创建，允许任何用户读取目录内容。

**修复方案**:
```go
// 使用更严格的权限
const bpfPinDirMode = 0700

func EnsurePinPath(pinPath string) error {
    if err := EnsureBPFFS(); err != nil {
        return err
    }

    if err := os.MkdirAll(pinPath, bpfPinDirMode); err != nil {
        return fmt.Errorf("创建 pin 目录 %s 失败: %w", pinPath, err)
    }

    // 验证权限未被 umask 修改
    if err := os.Chmod(pinPath, bpfPinDirMode); err != nil {
        log.Warnf("设置 %s 权限失败: %v", pinPath, err)
    }

    log.Debugf("Pin 目录就绪: %s（模式: %o）", pinPath, bpfPinDirMode)
    return nil
}
```

---

### MEDIUM-003: Map Pin 操作的竞态条件

**位置**: `pkg/dataplane/map_pinning.go:114-124`

**严重程度**: 🟡 中等

**问题描述**:
删除旧 pin 和创建新 pin 之间存在 TOCTOU（检查时间/使用时间）竞态。

**修复方案**:
```go
func PinMap(m *ebpf.Map, pinPath, mapName string) error {
    if err := EnsurePinPath(pinPath); err != nil {
        return err
    }

    path := GetPinnedMapPath(pinPath, mapName)
    tempPath := path + ".tmp"

    log.Debugf("正在 pin map 到: %s", path)

    // 首先 pin 到临时路径
    if err := m.Pin(tempPath); err != nil {
        return fmt.Errorf("pin map %s 到临时路径失败: %w", mapName, err)
    }

    // 原子重命名（在同一文件系统上）
    if err := os.Rename(tempPath, path); err != nil {
        os.Remove(tempPath) // 清理临时文件
        return fmt.Errorf("重命名 pin %s 失败: %w", mapName, err)
    }

    log.Infof("✓ 已 pin map: %s -> %s", mapName, path)
    return nil
}
```

---

### MEDIUM-004: 默认 ALLOW 策略的安全风险

**位置**: `src/bpf/headers/indexed_policy_match_v3.h:356-360`

**严重程度**: 🟡 中等

**问题描述**:
当没有策略匹配时，流量默认被允许。这违反了零信任原则。

**修复方案**:

添加可配置的默认动作：

```c
// 在 common_types.h 中
struct global_config {
    __u8 default_action;  // POLICY_ACTION_ALLOW 或 POLICY_ACTION_DENY
    __u8 log_misses;      // 记录策略未命中
    __u8 reserved[6];
};

// 在 indexed_policy_match_v3.h 中
static __always_inline __u8 get_default_action() {
    __u32 key = 0;
    struct global_config *config = bpf_map_lookup_elem(&global_config_map, &key);
    if (config) {
        return config->default_action;
    }
    return POLICY_ACTION_ALLOW; // 向后兼容的默认值
}

static __always_inline __u8 lookup_policy_action_indexed_v3(...) {
    // ... 现有查找逻辑 ...

    if (best_match) {
        update_stats(STATS_POLICY_HITS);
        *rule_id = best_match->rule_id;
        return best_match->action;
    }

    // 可配置的默认动作
    update_stats(STATS_POLICY_MISSES);
    *rule_id = 0;
    return get_default_action();
}
```

Go 侧配置：
```go
// pkg/dataplane/config.go
type GlobalConfig struct {
    DefaultAction PolicyAction `json:"default_action"` // "allow" 或 "deny"
    LogMisses     bool         `json:"log_misses"`
}

func (dp *DataPlane) SetDefaultAction(action PolicyAction) error {
    config := globalConfigBPF{
        DefaultAction: uint8(action),
        LogMisses:     1,
    }
    key := uint32(0)
    return dp.globalConfigMap.Update(&key, &config, ebpf.UpdateAny)
}
```

---

### MEDIUM-005: 统计重置部分失败

**位置**: `pkg/dataplane/nat.go:182-186`, `pkg/dataplane/fragment.go:184-187`

**严重程度**: 🟡 中等

**问题描述**:
统计重置在单个更新失败时继续执行，可能导致不一致状态。

**修复方案**:
```go
func (dp *DataPlane) ResetNATStats() error {
    natStatsMap, err := dp.getNATStatsMap()
    if err != nil {
        return fmt.Errorf("获取 NAT 统计 map 失败: %w", err)
    }

    var errs []error
    zeroValue := natStatsValue{Count: 0}

    for i := uint32(0); i < natStatsMax; i++ {
        if err := natStatsMap.Update(&i, &zeroValue, ebpf.UpdateAny); err != nil {
            errs = append(errs, fmt.Errorf("统计项 %d: %w", i, err))
        }
    }

    if len(errs) > 0 {
        // 记录所有错误但返回聚合错误
        for _, e := range errs {
            log.Warnf("重置 NAT 统计失败: %v", e)
        }
        return fmt.Errorf("部分重置失败: %d/%d 项统计失败", len(errs), natStatsMax)
    }

    log.Info("NAT 统计已成功重置")
    return nil
}
```

---

### MEDIUM-006: Flow 监控缺少优雅关闭

**位置**: `pkg/dataplane/dataplane.go:240-301`

**严重程度**: 🟡 中等

**问题描述**:
`MonitorFlowEvents()` 运行无限循环但没有 context 支持，难以优雅关闭。

**修复方案**:
```go
func (dp *DataPlane) MonitorFlowEvents(ctx context.Context) error {
    log.Info("启动流事件监控")

    for {
        select {
        case <-ctx.Done():
            log.Info("流事件监控被 context 停止")
            return ctx.Err()
        default:
        }

        record, err := dp.rbReader.Read()
        if err != nil {
            if errors.Is(err, ringbuf.ErrClosed) {
                log.Info("Ring buffer 已关闭")
                return nil
            }
            log.Errorf("从 ring buffer 读取失败: %v", err)
            continue
        }

        // ... 处理记录 ...
    }
}

// 替代方案：在 goroutine 中运行并使用 channel
func (dp *DataPlane) StartFlowMonitor(ctx context.Context) <-chan *flow.FlowEvent {
    events := make(chan *flow.FlowEvent, 1000)

    go func() {
        defer close(events)
        for {
            record, err := dp.rbReader.Read()
            if err != nil {
                if errors.Is(err, ringbuf.ErrClosed) {
                    return
                }
                continue
            }

            event, err := flow.ParseFlowEvent(record.RawSample)
            if err != nil {
                continue
            }

            select {
            case events <- event:
            case <-ctx.Done():
                return
            default:
                // Channel 已满，丢弃事件
                log.Warn("流事件 channel 已满，丢弃事件")
            }
        }
    }()

    return events
}
```

---

## 轻微问题

### MINOR-001: IPv6 扩展头未处理

**位置**: `src/bpf/tc_microsegment.bpf.c:206-207`

**严重程度**: 🟢 轻微

**问题描述**:
IPv6 扩展头未被解析；代码假设 TCP 头紧跟在 IPv6 基本头之后。

**影响**: 带有扩展头的 IPv6 数据包可能被错误处理。

**修复方案**:
```c
// 添加扩展头解析函数
static __always_inline void *skip_ipv6_ext_headers(
    struct ipv6hdr *ip6h,
    void *data_end,
    __u8 *next_proto)
{
    void *ptr = (void *)(ip6h + 1);
    __u8 nexthdr = ip6h->nexthdr;

    // 处理最多 4 个扩展头（验证器限制）
    #pragma unroll
    for (int i = 0; i < 4; i++) {
        if (nexthdr == IPPROTO_TCP || nexthdr == IPPROTO_UDP ||
            nexthdr == IPPROTO_ICMPV6) {
            *next_proto = nexthdr;
            return ptr;
        }

        // 处理常见扩展头
        if (nexthdr == IPPROTO_HOPOPTS || nexthdr == IPPROTO_ROUTING ||
            nexthdr == IPPROTO_DSTOPTS) {
            struct ipv6_opt_hdr *ext = ptr;
            if ((void *)(ext + 1) > data_end)
                return NULL;

            nexthdr = ext->nexthdr;
            ptr = (void *)ext + (ext->hdrlen + 1) * 8;

            if (ptr > data_end)
                return NULL;
        } else if (nexthdr == IPPROTO_FRAGMENT) {
            // 分片头是 8 字节
            struct ipv6_frag_hdr *frag = ptr;
            if ((void *)(frag + 1) > data_end)
                return NULL;

            nexthdr = frag->nexthdr;
            ptr = (void *)(frag + 1);
        } else {
            // 未知扩展头
            break;
        }
    }

    *next_proto = nexthdr;
    return ptr;
}
```

---

### MINOR-002: 进程监控日志洪泛

**位置**: `pkg/process/monitor.go:280`

**严重程度**: 🟢 轻微

**修复方案**:
```go
func (m *ProcessMonitor) processEvent(event *ProcessEvent) error {
    info := event.ToProcessInfo()

    // ... 路径解析 ...

    m.cache.Set(info)

    // 使用 Debug 级别而不是 Printf
    log.WithFields(log.Fields{
        "pid":       info.PID,
        "comm":      info.Comm,
        "path":      info.Path,
        "container": info.ContainerID,
    }).Debug("已缓存进程")

    return nil
}
```

---

### MINOR-003: Process 包使用标准 log

**位置**: `pkg/process/monitor.go:8`

**严重程度**: 🟢 轻微

**修复方案**:
```go
import (
    // 移除: "log"
    log "github.com/sirupsen/logrus"  // 使用结构化日志
)
```

---

### MINOR-004: GetProcessInfo 返回 interface{}

**位置**: `pkg/process/monitor.go:125`

**严重程度**: 🟢 轻微

**修复方案**:
创建共享类型包：

```go
// pkg/types/process.go
package types

type ProcessInfo struct {
    PID          uint32
    Comm         string
    Path         string
    ContainerID  string
    ExecTime     uint64
    CachedTime   time.Time
    PathResolved bool
}

// pkg/process/monitor.go
func (m *ProcessMonitor) GetProcessInfo(pid uint32) (*types.ProcessInfo, bool) {
    // ... 实现
}
```

---

### MINOR-005: EnsurePinPath 失败应该是致命错误

**位置**: `pkg/dataplane/tc_loader.go:46`

**严重程度**: 🟢 轻微

**修复方案**:
```go
func (l *TCLoader) Load() error {
    if err := EnsurePinPath(l.pinConfig.PinPath); err != nil {
        return fmt.Errorf("BPF 文件系统不可用: %w", err)
    }
    // ... 继续加载
}
```

---

### MINOR-006: 静默跳过没有 IP 的工作负载

**位置**: `pkg/policy/compiler.go:96-98`

**严重程度**: 🟢 轻微

**修复方案**:
```go
func (pc *PolicyCompiler) CompilePolicyRule(ruleID uint32) (*CompilationResult, error) {
    // ... 现有代码 ...

    skippedWorkloads := 0

    for _, fromMember := range fromMembers {
        for _, toMember := range toMembers {
            srcIP := ""
            if len(fromMember.IPs) > 0 {
                srcIP = fromMember.IPs[0].String()
            }
            dstIP := ""
            if len(toMember.IPs) > 0 {
                dstIP = toMember.IPs[0].String()
            }

            if srcIP == "" || dstIP == "" {
                skippedWorkloads++
                continue
            }
            // ... 创建策略
        }
    }

    if skippedWorkloads > 0 {
        result.AddWarning("由于缺少 IP 地址，跳过了 %d 个工作负载对", skippedWorkloads)
        log.Warnf("策略规则 %d: 跳过 %d 个工作负载对（无 IP）", ruleID, skippedWorkloads)
    }
}
```

---

### MINOR-007: 硬编码的 eBPF Map 限制

**位置**: `src/bpf/headers/common_types.h:7-9`

**严重程度**: 🟢 轻微

**修复方案**:
通过构建标志使限制可配置：

```c
// common_types.h
#ifndef MAX_ENTRIES_SESSION
#define MAX_ENTRIES_SESSION 100000
#endif

#ifndef MAX_ENTRIES_POLICY
#define MAX_ENTRIES_POLICY 10000
#endif

#ifndef MAX_ENTRIES_WILDCARD_POLICY
#define MAX_ENTRIES_WILDCARD_POLICY 1000
#endif
```

使用自定义限制构建：
```bash
make BPF_CFLAGS="-DMAX_ENTRIES_SESSION=200000 -DMAX_ENTRIES_POLICY=20000"
```

---

### MINOR-008: 分片超时可能过短

**位置**: `src/bpf/headers/fragment_tracking.h:43`

**严重程度**: 🟢 轻微

**修复方案**:
使超时可配置：

```go
// pkg/dataplane/fragment.go
const (
    DefaultFragmentTimeoutNs = 30 * time.Second
    MaxFragmentTimeoutNs     = 5 * time.Minute
    MinFragmentTimeoutNs     = 5 * time.Second
)

func (dp *DataPlane) SetFragmentTimeout(timeout time.Duration) error {
    if timeout < MinFragmentTimeoutNs || timeout > MaxFragmentTimeoutNs {
        return fmt.Errorf("超时必须在 %v 和 %v 之间",
            MinFragmentTimeoutNs, MaxFragmentTimeoutNs)
    }

    config, err := dp.GetFragmentConfig()
    if err != nil {
        return err
    }

    config.TimeoutNs = uint64(timeout.Nanoseconds())
    return dp.SetFragmentConfig(config)
}
```

---

## 附录：审核文件清单

| 文件 | 行数 | 发现问题数 |
|------|------|------------|
| `pkg/dataplane/manager.go` | 290 | 0 |
| `pkg/dataplane/tc_loader.go` | 376 | 2 |
| `pkg/dataplane/xdp_loader.go` | 226 | 1 |
| `pkg/dataplane/dataplane.go` | 414 | 1 |
| `pkg/dataplane/map_pinning.go` | 209 | 2 |
| `pkg/dataplane/nat.go` | 251 | 1 |
| `pkg/dataplane/fragment.go` | 241 | 2 |
| `pkg/dataplane/capability.go` | 149 | 0 |
| `pkg/policy/compiler.go` | 262 | 2 |
| `pkg/policy/policy.go` | 500+ | 1 |
| `pkg/process/monitor.go` | 358 | 4 |
| `src/bpf/tc_microsegment.bpf.c` | 838 | 1 |
| `src/bpf/headers/common_types.h` | 274 | 1 |
| `src/bpf/headers/indexed_policy_match_v3.h` | 363 | 1 |
| `src/bpf/headers/fragment_tracking.h` | 353 | 1 |

---

## 变更历史

| 日期 | 版本 | 作者 | 变更内容 |
|------|------|------|----------|
| 2025-01 | 1.0 | eBPF 技术专家 | 初始审核 |

---

*文档结束*
