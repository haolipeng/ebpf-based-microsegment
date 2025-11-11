# add-xdp-tc-dual-mode

## Summary
实现 XDP 和 TC eBPF 程序双模式架构，使系统能够在用户态智能检测系统能力，并自动选择最佳数据平面模式（Native XDP → Generic XDP → TCX → Legacy TC），以获得最佳性能和最广泛的兼容性。

## Background
当前系统仅使用 TC (Traffic Control) 作为数据平面 hook 点，虽然稳定可靠，但在高性能场景下存在性能瓶颈。

**现状**：
- ✅ 使用 TC Ingress hook 处理流量 ([tc_microsegment.bpf.c](../../../src/bpf/tc_microsegment.bpf.c))
- ✅ 支持 TCX (kernel >= 6.6) 和 Legacy TC (kernel >= 4.18) 自动降级
- ✅ 会话跟踪、策略匹配、统计收集功能完整
- ❌ 无法利用 XDP 的超低延迟特性
- ❌ 在高流量场景下 CPU 使用率较高
- ❌ 延迟相对较高（30-50μs）

**问题**：
1. TC hook 在网络栈较晚的位置触发，延迟较高
2. 需要 skb 分配开销，影响性能
3. 无法充分利用现代网卡的 XDP 硬件加速能力
4. 在 10Gbps+ 网络环境下性能受限

**技术分析结果**：
- ✅ 当前代码**只使用 Ingress hook**（XDP 完全兼容）
- ✅ 无 skb 元数据依赖（XDP 限制不影响）
- ✅ 所有 Map 类型 XDP 支持
- ✅ 所有 helper 函数 XDP 兼容
- ✅ 95%+ 代码可直接复用

## Goals
1. **实现 XDP 数据平面**：创建 XDP 版本的 eBPF 程序，复用现有策略匹配逻辑
2. **智能模式检测**：自动检测内核版本、XDP 支持、网卡驱动能力
3. **4 层降级策略**：Native XDP → Generic XDP → TCX → Legacy TC
4. **代码最大复用**：通过头文件共享和 Map Pinning 实现代码复用
5. **灵活配置**：支持 auto/force 模式选择，满足不同场景需求

## Non-Goals
- 不实现 XDP Egress（XDP 不支持，未来如需要可用 TC）
- 不修改现有 TC 程序的功能（保持向后兼容）
- 不实现 XDP_TX 或 XDP_REDIRECT（当前只需 PASS/DROP）
- 不支持数据包修改（当前架构无此需求）

## Proposed Changes

### 1. 模式检测和选择（新增 spec: `dataplane-mode-detection`）
- 实现系统能力检测：
  - 内核版本检测
  - XDP 程序类型支持检测
  - Native XDP 驱动测试（尝试实际附加）
  - TCX 支持检测（kernel >= 6.6）
- 实现智能模式选择逻辑
- 支持配置强制模式或自动选择

### 2. XDP eBPF 程序（新增 spec: `xdp-dataplane`）
- 创建 `xdp_microsegment.bpf.c`
- 复用策略匹配逻辑（通过 `#include` 头文件）
- 通过 Map Pinning 共享会话和策略数据
- 实现 XDP 数据包解析（`xdp_md` 上下文）
- 返回 `XDP_PASS` / `XDP_DROP`

### 3. 共享代码架构（修改现有 spec: `ebpf-policy-enforcement`）
- 提取策略匹配逻辑到 `headers/policy_match.h`
- 提取流处理函数到 `headers/flow_helpers.h`
- Map 定义添加 `LIBBPF_PIN_BY_NAME` 支持
- XDP 和 TC 共享同一组 Map

### 4. 用户态加载器重构（修改现有 spec: `agent-initialization`）
- 创建统一的 `DataPlane` 接口
- 实现 `xdp_loader.go` 支持 Native/Generic XDP
- 重构 `tc_loader.go` 提取现有逻辑
- 实现模式选择和降级逻辑
- 配置 bpf2go 生成 XDP Go 绑定

## Design Decisions

### 为什么选择双模式共存而非完全替换 TC？
**选择**：XDP 和 TC 双模式共存，自动选择

**原因**：
- ✅ **兼容性**：不是所有网卡都支持 Native XDP
- ✅ **降级路径**：Generic XDP 性能提升有限，需要 TC 作为兜底
- ✅ **向后兼容**：保持现有 TC 用户不受影响
- ✅ **灵活性**：用户可根据环境强制选择模式

**替代方案**：完全替换为 XDP
- ❌ 兼容性差：旧内核或不支持的网卡无法运行
- ❌ 功能受限：未来如需 Egress 支持，XDP 无法满足

### 如何共享策略匹配逻辑？
**选择**：通过 `#include` 头文件共享

**实现**：
```c
// headers/policy_match.h
static __always_inline __u8 lookup_policy_action(
    struct flow_key *key,
    __u32 *rule_id)
{
    // 精确匹配
    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, key);
    if (policy) {
        *rule_id = policy->rule_id;
        return policy->action;
    }

    // 通配符匹配
    // ...

    return POLICY_ACTION_ALLOW;
}

// xdp_microsegment.bpf.c 和 tc_microsegment.bpf.c 都 #include 此头文件
```

**优势**：
- ✅ 核心逻辑只需维护一份
- ✅ 保证 XDP 和 TC 行为完全一致
- ✅ 编译时内联，无性能开销

### 如何共享 Map 数据？
**选择**：使用 BPF Map Pinning 机制

**实现**：
```c
struct {
    __uint(type, BPF_MAP_TYPE_LRU_HASH);
    __uint(max_entries, MAX_ENTRIES_SESSION);
    __type(key, struct flow_key);
    __type(value, struct session_value);
    __uint(pinning, LIBBPF_PIN_BY_NAME);  // 固定到 /sys/fs/bpf/
} session_map SEC(".maps");
```

**优势**：
- ✅ XDP 和 TC 查看相同的会话状态
- ✅ 无需同步，实时一致
- ✅ 内存开销不增加（只有一份 Map）

### 降级策略的优先级如何确定？
**选择**：性能优先，兼容性兜底

**降级链**：
```
1. Native XDP (最快，300-400% 提升)
   ↓ 驱动不支持
2. Generic XDP (较快，20-30% 提升)
   ↓ 内核 < 4.12
3. TCX (快，kernel >= 6.6)
   ↓ 内核 < 6.6
4. Legacy TC (稳定，kernel >= 4.18)
```

**配置选项**：
```yaml
dataplane:
  mode: auto              # 自动选择
  prefer_xdp: true        # 优先 XDP
  allow_generic_xdp: true # 允许 Generic 回退
```

## Dependencies
- `k8s.io/client-go` 库（已添加，用于 K8s 集成）
- Linux kernel >= 4.18（现有要求）
- Linux kernel >= 4.12（Generic XDP）
- Linux kernel >= 6.6（TCX，可选）
- 网卡驱动支持 XDP（Native XDP，可选）

## Risks and Mitigations

### 风险 1：Native XDP 驱动兼容性
**影响**：不是所有网卡都支持 Native XDP

**缓解措施**：
- 实现 4 层降级策略
- 默认允许 Generic XDP 回退
- 提供驱动兼容性检测和清晰的日志
- 文档说明常见网卡的支持情况

### 风险 2：Map Pinning 路径权限
**影响**：需要 `/sys/fs/bpf/` 挂载且有写权限

**缓解措施**：
- 启动时检查 BPF 文件系统挂载状态
- 提供清晰的错误提示和修复建议
- 文档说明权限要求

### 风险 3：XDP 和 TC 行为不一致
**影响**：策略匹配结果可能不同

**缓解措施**：
- 共享策略匹配逻辑（同一份代码）
- 共享 Map 数据（同一组 Map）
- E2E 测试验证行为一致性
- 单元测试覆盖策略匹配逻辑

### 风险 4：性能提升不明显
**影响**：Generic XDP 性能提升有限（20-30%）

**缓解措施**：
- 性能基准测试明确收益
- 文档说明各模式的性能预期
- 提供性能调优建议
- 推荐在支持的环境使用 Native XDP

## Testing Strategy
1. **单元测试**：
   - 模式检测逻辑测试
   - 配置验证测试
   - Mock 测试框架

2. **集成测试**：
   - 验证 XDP 和 TC 程序都能加载
   - 验证 Map 共享机制
   - 验证降级逻辑

3. **E2E 测试**：
   - 策略匹配行为一致性测试
   - 会话跟踪验证
   - 统计数据准确性

4. **性能测试**：
   - 延迟对比（P50, P99）
   - 吞吐量对比（小包、大包）
   - CPU 使用率对比

## Success Criteria
- [ ] XDP 和 TC 程序都能正常加载和运行
- [ ] 自动检测正确选择最佳模式
- [ ] 策略匹配行为与 TC 完全一致
- [ ] Map 数据正确共享（会话、策略、统计）
- [ ] 所有现有测试通过（无功能回归）
- [ ] Native XDP 延迟降低 50-70%
- [ ] Native XDP 吞吐量提升 2-4 倍
- [ ] 配置文档和故障排查指南完整
- [ ] 性能基准测试报告

## Timeline
- **Week 1**: 模式检测器和配置扩展（已完成 ✅）
- **Week 2**: XDP eBPF 程序和共享头文件
- **Week 3**: XDP 加载器和统一接口
- **Week 4**: 测试、性能验证和文档

## Related Changes
- Depends on: 现有的 TC 数据平面实现
- Related: `add-kubernetes-integration` (K8s 集成可受益于 XDP 性能提升)
- Blocks: 未来的 XDP-based NetworkPolicy 实现

## References
- [XDP 官方文档](https://www.kernel.org/doc/html/latest/networking/xdp-offload.html)
- [Cilium XDP Performance](https://cilium.io/blog/2021/05/11/cilium-110)
- [Cloudflare XDP 实践](https://blog.cloudflare.com/how-to-drop-10-million-packets/)
- [libbpf Map Pinning](https://nakryiko.com/posts/bpf-portability-and-co-re/#bpf-map-pinning)
- 现有 TC 实现分析: [design-docs/analysis/ebpf-tc-comparison.md](../../../design-docs/analysis/ebpf-tc-comparison.md)
