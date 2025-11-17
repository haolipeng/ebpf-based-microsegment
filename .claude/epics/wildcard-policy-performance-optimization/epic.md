---
name: wildcard-policy-performance-optimization
status: backlog
created: 2025-11-16T12:31:36Z
progress: 0%
prd: .claude/prds/wildcard-policy-performance-optimization.md
github: https://github.com/haolipeng/ebpf-based-microsegment/issues/8
---

# Epic: 通配符策略性能优化

## Overview

优化 eBPF 通配符策略匹配性能,从当前的线性扫描 O(n) 升级到 LPM Trie O(log n) 查找。分两个阶段实施:
- **Phase 1 (短期)**: 优化现有线性扫描逻辑,限制策略数量到 50 条,添加早停优化
- **Phase 2 (长期)**: 引入 eBPF LPM Trie Map,支持 1000+ 条策略,实现 CIDR 高效匹配

**技术价值**:
- 性能提升 5x: 100 条策略时延迟从 10μs → 2μs
- 可扩展性: 支持 100 → 1024 条策略
- 丢包率降低: <0.01% (当前 0.1%)

## Architecture Decisions

### AD1: 分阶段实施策略
**决策**: 采用两阶段渐进式优化,而非一次性重构
**理由**:
- 降低风险: Phase 1 为低风险优化,快速见效
- 验证价值: 先验证性能提升效果,再投入 LPM 重构
- 向后兼容: 保留现有 Array Map 作为 fallback

### AD2: LPM Trie 双索引架构
**决策**: 对 src_ip 和 dst_ip 分别建立 LPM Trie 索引
**理由**:
- 灵活性: 支持单向和双向 CIDR 匹配
- 性能: 两次 O(log n) 查找仍优于 O(n) 扫描
- 简化: 避免复合键的复杂性

### AD3: 混合查找策略
**决策**: 精确匹配(Hash) → LPM 匹配(Trie) → 细粒度过滤
**理由**:
- Fast Path: 精确匹配 O(1) 处理常见情况
- CIDR Path: LPM O(log n) 处理范围匹配
- 优先级: 最后应用 priority 字段选择最优策略

### AD4: 用户空间抽象层
**决策**: Server 端自动选择存储后端(Array vs LPM Trie)
**理由**:
- API 兼容: 用户无需关心底层实现
- 平滑迁移: 自动迁移现有策略
- Feature Flag: 可配置启用/禁用 LPM

## Technical Approach

### eBPF Data Plane

#### Phase 1: 线性扫描优化
**文件**: `src/bpf/headers/policy_match.h`

**变更**:
1. 常量化循环上限:
   ```c
   #define MAX_WILDCARD_LOOP 50  // 替代硬编码 100
   ```

2. 早停优化:
   ```c
   if (!wildcard || wildcard->rule_id == 0) {
       break;  // 立即退出,不再扫描空槽位
   }
   ```

3. 紧凑存储支持:
   - Server 端维护 `first_free_index`
   - 策略删除时回收空槽位

#### Phase 2: LPM Trie 查找
**新文件**: `src/bpf/headers/policy_match_lpm.h`

**核心数据结构**:
```c
// LPM Key
struct lpm_key {
    __u32 prefixlen;  // CIDR prefix (8-32)
    __u32 ip;
};

// LPM Value: 候选 rule_id 列表
struct lpm_value {
    __u32 rule_ids[8];
    __u8  rule_count;
};

// Policy Detail Map
struct policy_detail {
    __u32 rule_id;
    __u16 src_port, dst_port;
    __u8  protocol, direction, priority, action;
};
```

**eBPF Maps**:
```c
BPF_MAP_TYPE_LPM_TRIE: src_ip_lpm_map, dst_ip_lpm_map
BPF_MAP_TYPE_HASH: policy_detail_map
```

**查找流程**:
```c
1. Exact Match: hash_lookup(policy_map, 5-tuple)
2. LPM Match:
   - src_matches = lpm_lookup(src_ip_lpm_map, src_ip)
   - dst_matches = lpm_lookup(dst_ip_lpm_map, dst_ip)
3. Fine-grained Filter:
   - Intersect src_matches & dst_matches
   - Check port, protocol, direction
   - Select max(priority)
4. Default: ALLOW
```

### Server Control Plane

#### Policy Manager 扩展
**文件**: `src/server/pkg/policy/wildcard_manager.go`

**核心功能**:
```go
type WildcardPolicyManager struct {
    useLPM bool  // Feature flag
    compactIndex map[uint32]uint32  // rule_id -> array_index
}

// AddPolicy: 自动选择存储后端
func (m *WildcardPolicyManager) AddPolicy(policy *WildcardPolicy) error {
    if m.useLPM {
        return m.addToLPMTrie(policy)
    }
    return m.addToArray(policy)
}

// MigrateToLPM: 迁移现有策略
func (m *WildcardPolicyManager) MigrateToLPM() error {
    for _, policy := range m.listArrayPolicies() {
        m.addToLPMTrie(policy)
    }
}
```

#### 性能监控
**新增 Metrics**:
```go
policy_lookup_duration_us{type="wildcard|lpm"}
policy_hits_total{type="exact|wildcard|lpm"}
policy_miss_rate
lpm_trie_entries{map="src_ip|dst_ip"}
```

### Testing Strategy

#### 单元测试
- `policy_match_test.c`: eBPF 逻辑单元测试
- `wildcard_manager_test.go`: Server 端单元测试

#### 性能基准测试
- `benchmark_policy_lookup.c`: 对比 Array vs LPM 性能
- 测试场景: 10/50/100/500/1000 条策略
- 指标: P50/P95/P99 延迟, 吞吐量

#### 集成测试
- 端到端测试: API → Server → eBPF
- 策略迁移测试: Array → LPM 无缝切换
- 兼容性测试: 多内核版本验证

## Implementation Strategy

### Development Phases

#### Phase 1: 短期优化 (Week 1-2)
**目标**: 降低当前性能问题紧迫性

**关键活动**:
1. Week 1: 代码优化
   - 常量化 + 早停逻辑
   - Server 端紧凑存储
   - 单元测试
2. Week 2: 验证与部署
   - 性能基准测试
   - 灰度发布
   - 监控验证

**交付物**:
- 优化后的 `policy_match.h`
- 性能提升报告
- 部署文档

#### Phase 2: LPM Trie 重构 (Week 3-6)
**目标**: 实现企业级可扩展性

**关键活动**:
1. Week 3: 设计与原型
   - LPM 架构设计评审
   - eBPF 原型验证(verifier 通过)
   - 数据结构定义
2. Week 4-5: 核心开发
   - eBPF LPM 查找逻辑
   - Server 端 LPM 管理 API
   - 自动迁移工具
   - 单元 + 集成测试
3. Week 6: 测试与上线
   - 性能压测(1000 条策略)
   - 多内核兼容性测试
   - 生产环境灰度发布

**交付物**:
- `policy_match_lpm.h` (eBPF)
- `wildcard_manager.go` (Server)
- 迁移工具 + 文档
- 性能测试报告

### Risk Mitigation

#### R1: eBPF Verifier 拒绝 LPM 代码
**概率**: 中 | **影响**: 高
**缓解**:
- 提前在多内核版本验证原型
- 保留 Array Map 作为 fallback
- Feature Flag 控制启用

#### R2: LPM 性能不达预期
**概率**: 低 | **影响**: 高
**缓解**:
- 性能建模预测
- Benchmark 对比验证
- 混合策略(热点用 Array,冷数据用 LPM)

#### R3: 策略迁移失败
**概率**: 低 | **影响**: 中
**缓解**:
- 双写验证(同时写 Array + LPM)
- 回滚机制
- 详细的迁移日志

## Tasks Created
- [ ] #9 - 策略自动迁移工具 (parallel: true)
- [ ] #10 - 性能压测与多内核兼容性测试 (parallel: false)
- [ ] #11 - LPM Trie eBPF 数据结构与 Map 定义 (parallel: true)
- [ ] #12 - eBPF LPM 查找逻辑实现 (parallel: false)
- [ ] #14 - Server 端 LPM 管理 API 开发 (parallel: true)
- [ ] #15 - Phase 2 生产环境部署 (parallel: false)
- [ ] #18 - eBPF 循环常量化与早停优化 (parallel: true)
- [ ] #19 - Server 端紧凑存储实现 (parallel: true)
- [ ] #20 - Phase 1 部署与监控验证 (parallel: false)
- [ ] #21 - 性能基准测试框架搭建 (parallel: true)

Total tasks: 10
Parallel tasks: 6
Sequential tasks: 4
## Dependencies

### External Dependencies

#### D1: eBPF 工具链
- **依赖**: libbpf >= 0.6, bpftool, clang >= 12
- **风险**: 旧版本内核可能不支持 LPM Trie
- **缓解**: 版本检测 + Feature Flag 控制
- **负责人**: DevOps Team

#### D2: Linux Kernel
- **依赖**: Kernel >= 4.11 (LPM Trie 支持)
- **建议**: Kernel >= 5.10 (性能优化)
- **验证**: 多内核版本兼容性测试
- **负责人**: Infrastructure Team

### Internal Dependencies

#### D3: 测试基础设施
- **依赖**: 10Gbps 网卡性能测试环境
- **时间**: Phase 2 开发期间
- **负责人**: QA Team

### Prerequisite Work

#### P1: 现有代码审查
- 审查 `policy_match.h` 当前实现
- 识别重构影响范围
- **估算**: 1 day

#### P2: 性能基线建立
- 收集当前策略查找延迟数据
- 建立性能监控大盘
- **估算**: 2 days

## Success Criteria (Technical)

### Performance Benchmarks

| 指标 | 当前 | Phase 1 目标 | Phase 2 目标 |
|------|------|-------------|-------------|
| 最大策略数 | 100 | 50 | 1024 |
| 单包延迟(100策略) | ~10μs | <7μs | <3μs |
| P99 查找延迟 | ~15μs | <10μs | <5μs |
| 丢包率(1000策略) | N/A | N/A | <0.01% |
| 策略命中率 | ~60% | >60% | >80% |

### Quality Gates

#### Phase 1 验收标准
- [ ] 50 条策略时延迟 <7μs
- [ ] eBPF verifier 验证通过
- [ ] 单元测试覆盖率 >80%
- [ ] 无性能回退(vs baseline)

#### Phase 2 验收标准
- [ ] 1000 条策略支持
- [ ] P99 延迟 <5μs
- [ ] 多内核版本兼容(4.11, 5.4, 5.10, 5.15)
- [ ] 自动迁移成功率 100%
- [ ] 集成测试通过

### Acceptance Criteria

#### 功能性
- [ ] 支持 CIDR /8 到 /32 任意前缀
- [ ] 精确匹配与 LPM 匹配混合工作
- [ ] 策略优先级正确处理
- [ ] 向后兼容现有 API

#### 性能
- [ ] 100 条策略时单包延迟 <5μs
- [ ] 1000 条策略时丢包率 <0.01%
- [ ] 策略添加/删除操作 <100ms

#### 可运维性
- [ ] 日志记录策略变更
- [ ] 支持策略导出/导入
- [ ] 故障回滚机制完善

## Estimated Effort

### Resource Requirements
- **eBPF Engineer**: 1 FTE (6 weeks)
- **Backend Engineer**: 0.5 FTE (4 weeks)
- **QA Engineer**: 0.3 FTE (2 weeks)

### Timeline
- **Phase 1**: 2 weeks
- **Phase 2**: 4 weeks
- **Total**: 6 weeks (1.5 months)

### Critical Path
1. eBPF LPM 原型验证 (Week 3) - **关键**
2. Server 端 API 开发 (Week 4)
3. 性能压测 (Week 6) - **关键**

### Risk Buffer
- **建议**: 预留 10% 时间缓冲(+0.6 week)
- **应对**: eBPF verifier 问题、性能调优

---

## Notes

**简化策略**:
- 聚焦核心价值: 性能优化,而非功能扩展
- 复用现有: 保留 Hash Map 精确匹配,只优化通配符部分
- 渐进式: 两阶段交付,降低风险
- 最小化任务: 10 个任务覆盖完整实施路径

**优化亮点**:
- 无需重写整个策略引擎,只优化瓶颈部分
- Feature Flag 控制,可随时回滚
- 自动迁移,用户无感知
- 监控先行,数据驱动决策
