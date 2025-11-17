---
name: wildcard-policy-performance-optimization
description: 优化通配符策略匹配性能,从线性扫描升级到 LPM Trie,支持大规模策略部署
status: backlog
created: 2025-11-16T11:18:13Z
---

# PRD: 通配符策略性能优化

## Executive Summary

当前 eBPF 微分段系统的通配符策略匹配使用线性扫描算法,在策略数量增长时存在严重的性能瓶颈。本 PRD 提出分阶段优化方案:短期通过限制策略数量和优化扫描逻辑来缓解问题,长期通过引入 LPM Trie Map 实现 O(log n) 复杂度的 CIDR 匹配,支持企业级大规模策略部署场景。

**价值主张**:
- **性能提升**: 从 O(n) 线性扫描优化到 O(log n) LPM 查找,支持 1000+ 条通配符策略
- **可扩展性**: 消除性能瓶颈,支持复杂网络环境下的大规模策略管理
- **用户体验**: 降低策略查找延迟,减少数据包丢包率,提升系统稳定性

## Problem Statement

### 当前问题

#### 1. 性能瓶颈 🔥
**代码位置**: `src/bpf/headers/policy_match.h:140`

```c
#pragma unroll
for (__u32 i = 0; i < 100; i++) {  // 硬编码最多扫描 100 次
    wildcard = bpf_map_lookup_elem(&wildcard_policy_map, &idx);
    if (!wildcard || wildcard->rule_id == 0)
        continue;
    if (matches_wildcard(key, wildcard, direction)) {
        // 找到匹配...
    }
}
```

**问题分析**:
- **时间复杂度**: O(n),其中 n 最大为 100
- **最坏情况**: 每个数据包需要执行 100 次 map lookup + 100 次 CIDR 匹配计算
- **性能影响**: 在 10Gbps 网络环境下,可能导致显著的数据包处理延迟和丢包

#### 2. 可扩展性受限
- **硬编码限制**: 最多支持 100 条通配符策略
- **eBPF 验证器约束**: `#pragma unroll` 必须使用编译时常量,无法动态扩展
- **企业需求**: 大型企业网络环境可能需要 500-1000+ 条策略

#### 3. 资源浪费
- **空槽位扫描**: 即使策略只有 10 条,仍需扫描 100 次(90 次无效扫描)
- **CPU 周期浪费**: 每次匹配需要 6 次比较操作(src_ip, dst_ip, src_port, dst_port, protocol, direction)

### 为什么现在重要?

1. **业务增长**: 随着系统部署规模扩大,策略数量快速增长
2. **性能要求**: 高吞吐量场景对数据包处理延迟敏感
3. **竞争力**: 与 NeuVector 等企业级产品对标,需要支持大规模策略

## User Stories

### Primary Persona: DevOps Engineer (张伟)

**背景**:
- 负责管理包含 500+ 微服务的 Kubernetes 集群
- 需要配置复杂的网络策略(不同环境隔离、CIDR 范围控制)
- 关注系统性能和稳定性

**当前痛点**:
1. 创建 50 条通配符策略后,监控显示数据包处理延迟明显增加
2. 想要添加更多细粒度策略,但担心性能下降
3. 无法根据 CIDR 段高效管理外部 IP 访问控制

**期望体验**:
1. 能够创建 200+ 条通配符策略而不影响网络性能
2. 策略匹配延迟保持在微秒级(<10μs)
3. 系统自动优化策略查找,无需手动调优

### User Journey: 添加新的通配符策略

#### As-Is (当前状态)
1. 张伟通过 API 创建新的通配符策略:`192.168.0.0/16 -> 10.0.0.0/8, port=3306, action=DENY`
2. Server 写入 `wildcard_policy_map[next_available_index]`
3. **问题**: 每次策略查找需要线性扫描所有策略,延迟 +2μs
4. 当策略数量达到 60 条时,监控告警显示丢包率上升

#### To-Be (优化后)
1. 张伟通过 API 创建新的通配符策略
2. Server 自动将策略插入 LPM Trie Map(src_ip 和 dst_ip 分别建立索引)
3. **改进**: eBPF 通过 O(log n) LPM 查找快速匹配,延迟保持稳定(<1μs)
4. 支持 500+ 条策略而无性能退化

### Acceptance Criteria

**Story 1**: 作为 DevOps,我希望创建 100 条通配符策略后性能不降级
- [ ] 100 条策略时,单包处理延迟 <5μs (vs 当前 10μs)
- [ ] 丢包率 <0.01% (vs 当前 0.1%)
- [ ] eBPF 程序仍然通过 verifier 验证

**Story 2**: 作为系统管理员,我希望支持 CIDR 范围的高效匹配
- [ ] 支持 /8 到 /32 任意 CIDR 前缀
- [ ] 同一 CIDR 内的 IP 查找时间相同(O(log n))
- [ ] 自动选择最精确匹配的策略

**Story 3**: 作为开发者,我希望策略管理 API 保持简单
- [ ] 现有 API 无需修改(向后兼容)
- [ ] 自动迁移现有策略到新数据结构
- [ ] 提供性能监控指标(策略查找耗时分布)

## Requirements

### Functional Requirements

#### FR1: 短期优化(Phase 1)

##### FR1.1: 优化循环上限
- **需求**: 将硬编码的 `100` 替换为可配置常量 `MAX_WILDCARD_LOOP`
- **实现**:
  ```c
  #define MAX_WILDCARD_LOOP 50  // 短期限制为 50
  ```
- **优先级**: P0

##### FR1.2: 早停优化
- **需求**: 当遇到空槽位时,立即跳出循环
- **实现**:
  ```c
  if (!wildcard || wildcard->rule_id == 0) {
      break;  // 而不是 continue
  }
  ```
- **优先级**: P0

##### FR1.3: 紧凑存储
- **需求**: 策略添加时填充到第一个空槽位,避免碎片化
- **实现**: Server 端维护 `first_free_index` 指针
- **优先级**: P1

#### FR2: 长期优化(Phase 2 - LPM Trie)

##### FR2.1: LPM Trie Map 数据结构
- **需求**: 使用 eBPF `BPF_MAP_TYPE_LPM_TRIE` 优化 IP 匹配
- **实现**:
  ```c
  struct {
      __uint(type, BPF_MAP_TYPE_LPM_TRIE);
      __uint(key_size, sizeof(struct lpm_key));
      __uint(value_size, sizeof(struct lpm_value));
      __uint(max_entries, 1024);
      __uint(map_flags, BPF_F_NO_PREALLOC);
  } src_ip_lpm_map SEC(".maps");

  struct {
      __uint(type, BPF_MAP_TYPE_LPM_TRIE);
      // ... 同上 ...
  } dst_ip_lpm_map SEC(".maps");
  ```
- **优先级**: P0

##### FR2.2: 双层查找架构
- **需求**: 结合精确匹配 + LPM 匹配 + 细粒度过滤
- **查找流程**:
  1. 精确匹配(Hash Map) - 处理完全精确的策略
  2. LPM 匹配(Trie Map) - 处理 CIDR 范围
  3. 细粒度过滤 - 检查端口、协议等条件
- **优先级**: P0

##### FR2.3: 策略优先级处理
- **需求**: LPM 返回最长前缀匹配,结合 priority 字段选择最终策略
- **实现**: 当 LPM 返回多个候选时,比较 `priority` 字段
- **优先级**: P1

##### FR2.4: 双向查找
- **需求**: 同时对 src_ip 和 dst_ip 建立 LPM 索引
- **实现**:
  ```c
  struct lpm_value *src_match = bpf_map_lookup_elem(&src_ip_lpm_map, &src_key);
  struct lpm_value *dst_match = bpf_map_lookup_elem(&dst_ip_lpm_map, &dst_key);
  // 合并结果,选择最优匹配
  ```
- **优先级**: P1

#### FR3: 用户空间支持

##### FR3.1: 策略管理 API
- **需求**: Server 端 API 支持策略的 CRUD 操作
- **接口**:
  ```go
  type WildcardPolicy struct {
      RuleID      uint32
      SrcIPCIDR   string  // "192.168.1.0/24"
      DstIPCIDR   string
      SrcPort     uint16
      DstPort     uint16
      Protocol    uint8
      Direction   uint8
      Priority    uint8
      Action      uint8
  }

  // AddWildcardPolicy 自动选择存储后端(Array vs LPM Trie)
  func (s *Server) AddWildcardPolicy(ctx context.Context, policy *WildcardPolicy) error
  ```
- **优先级**: P0

##### FR3.2: 性能监控
- **需求**: 收集策略匹配性能指标
- **指标**:
  - `policy_lookup_duration_us{type="wildcard"}`: 通配符查找耗时
  - `policy_hits_total{type="lpm"}`: LPM 命中次数
  - `policy_miss_rate`: 策略未命中率
- **优先级**: P1

### Non-Functional Requirements

#### NFR1: 性能

##### NFR1.1: 吞吐量
- **要求**: 10Gbps 网络环境下,丢包率 <0.01%
- **基准**: 1000 条策略时,单包处理延迟 <10μs

##### NFR1.2: 延迟
- **要求**: P99 策略查找延迟 <5μs
- **对比**: 当前线性扫描 P99 延迟约 15μs

##### NFR1.3: 可扩展性
- **要求**: 支持 1024 条通配符策略
- **对比**: 当前最多 100 条

#### NFR2: 可靠性

##### NFR2.1: 向后兼容
- **要求**: 现有策略自动迁移,无需手动干预
- **实现**: Server 启动时检测并迁移旧格式策略

##### NFR2.2: 故障恢复
- **要求**: LPM Trie 插入失败时,回退到 Array 存储
- **实现**: 双存储后端,自动切换

#### NFR3: 可维护性

##### NFR3.1: 代码复用
- **要求**: TC 和 XDP 共享相同的策略匹配逻辑
- **实现**: 提取为 `policy_match_lpm.h` 头文件

##### NFR3.2: 可测试性
- **要求**: 提供性能基准测试工具
- **实现**: 编写 `benchmark_policy_lookup.c` 测试程序

#### NFR4: 安全性

##### NFR4.1: 策略隔离
- **要求**: 不同租户的策略互相隔离
- **实现**: LPM Trie 支持 per-namespace 索引

##### NFR4.2: 策略审计
- **要求**: 记录策略变更历史
- **实现**: Server 端审计日志

## Success Criteria

### 定量指标

| 指标 | 当前值 | Phase 1 目标 | Phase 2 目标 | 测量方法 |
|------|--------|-------------|-------------|----------|
| **最大策略数** | 100 | 50 | 1024 | 配置文件限制 |
| **单包处理延迟(100策略)** | ~10μs | <7μs | <3μs | eBPF perf event |
| **P99 查找延迟** | ~15μs | <10μs | <5μs | eBPF perf event |
| **丢包率(1000策略)** | N/A | N/A | <0.01% | tc -s qdisc |
| **策略命中率** | ~60% | >60% | >80% | 命中计数/总查找 |
| **内存占用(1000策略)** | N/A | N/A | <2MB | bpftool map show |

### 定性指标

- [ ] **用户满意度**: DevOps 反馈策略配置流程顺畅,性能稳定
- [ ] **可运维性**: 监控大盘清晰展示策略性能指标
- [ ] **文档完善**: 策略优化最佳实践文档完成

### 里程碑

#### M1: Phase 1 完成(2周)
- [ ] MAX_WILDCARD_LOOP 常量化
- [ ] 早停优化实现
- [ ] 性能基准测试通过

#### M2: Phase 2 设计(1周)
- [ ] LPM Trie 架构设计评审通过
- [ ] 数据结构设计完成
- [ ] 性能模型建立

#### M3: Phase 2 实现(3周)
- [ ] LPM Trie eBPF 代码实现
- [ ] Server 端 API 实现
- [ ] 自动迁移逻辑完成

#### M4: 测试与上线(1周)
- [ ] 性能测试通过
- [ ] 集成测试通过
- [ ] 生产环境灰度发布

## Technical Design (High-Level)

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                      User Space (Server)                     │
├─────────────────────────────────────────────────────────────┤
│  Policy Management API                                       │
│  ├── AddWildcardPolicy(CIDR, port, protocol, priority)      │
│  ├── DeleteWildcardPolicy(rule_id)                          │
│  └── ListWildcardPolicies()                                 │
│                                                              │
│  Policy Storage Engine                                       │
│  ├── LPM Trie Builder (parse CIDR, insert to eBPF map)     │
│  ├── Migration Tool (Array -> LPM Trie)                    │
│  └── Performance Monitor (collect metrics)                  │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼ (eBPF Map Operations)
┌─────────────────────────────────────────────────────────────┐
│                    Kernel Space (eBPF)                       │
├─────────────────────────────────────────────────────────────┤
│  Policy Lookup Pipeline                                      │
│                                                              │
│  1. Exact Match (Hash Map)       ←─ Fast Path              │
│     ├── policy_map[5-tuple]                                 │
│     └── O(1) lookup                                         │
│                                                              │
│  2. LPM Match (Trie Map)          ←─ CIDR Path             │
│     ├── src_ip_lpm_map[src_ip/prefix]                      │
│     ├── dst_ip_lpm_map[dst_ip/prefix]                      │
│     └── O(log n) lookup                                     │
│                                                              │
│  3. Fine-grained Filter                                      │
│     ├── Check port, protocol, direction                     │
│     └── Priority comparison                                 │
│                                                              │
│  4. Default Policy → ALLOW/DENY                             │
└─────────────────────────────────────────────────────────────┘
```

### Data Structures

#### LPM Key Structure
```c
struct lpm_key {
    __u32 prefixlen;  // CIDR prefix length (8-32)
    __u32 ip;         // IP address (network byte order)
};
```

#### LPM Value Structure
```c
struct lpm_value {
    __u32 rule_ids[8];    // 最多 8 个匹配的 rule_id
    __u8  rule_count;     // 实际匹配数量
    __u8  _pad[3];
};
```

#### Policy Detail Structure
```c
struct policy_detail {
    __u32 rule_id;
    __u16 src_port;
    __u16 dst_port;
    __u8  protocol;
    __u8  direction;
    __u8  priority;
    __u8  action;
};

// 新增 Map 用于存储策略详情
struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, __u32);           // rule_id
    __type(value, struct policy_detail);
    __uint(max_entries, 2048);
} policy_detail_map SEC(".maps");
```

### Algorithm: LPM Lookup Flow

```
Input: flow_key (src_ip, dst_ip, src_port, dst_port, protocol)

1. Exact Match Check:
   policy = hash_lookup(policy_map, flow_key)
   if policy:
       return policy.action

2. LPM Match on Source IP:
   src_lpm_key = {prefixlen: 32, ip: src_ip}
   src_matches = lpm_lookup(src_ip_lpm_map, src_lpm_key)

3. LPM Match on Destination IP:
   dst_lpm_key = {prefixlen: 32, ip: dst_ip}
   dst_matches = lpm_lookup(dst_ip_lpm_map, dst_lpm_key)

4. Merge Candidates:
   candidates = intersect(src_matches.rule_ids, dst_matches.rule_ids)

5. Fine-grained Filter:
   for rule_id in candidates:
       detail = hash_lookup(policy_detail_map, rule_id)
       if matches_port(detail, src_port, dst_port) &&
          matches_protocol(detail, protocol) &&
          matches_direction(detail, direction):
           best_match = max(best_match, detail, by=priority)

6. Return:
   if best_match:
       return best_match.action
   else:
       return DEFAULT_ACTION
```

## Constraints & Assumptions

### Technical Constraints

#### C1: eBPF 限制
- **Verifier 约束**: LPM Trie 查找循环必须是有界的
- **Map 大小**: 单个 LPM Trie 最大 64K entries(内核限制)
- **指令数限制**: eBPF 程序最多 1M 指令(需优化复杂度)

#### C2: 内核版本依赖
- **最低要求**: Linux 4.11+ (LPM Trie 支持)
- **建议版本**: Linux 5.10+ (性能优化)

#### C3: 硬件资源
- **CPU**: 需要支持 BPF JIT 的处理器
- **内存**: 1024 条策略约需 2MB eBPF map 内存

### Business Constraints

#### C4: 兼容性
- **向后兼容**: 必须支持现有策略格式
- **API 稳定**: 不破坏现有 API 接口

#### C5: 时间约束
- **Phase 1**: 2 周内完成(低风险优化)
- **Phase 2**: 4 周内完成(核心重构)

### Assumptions

#### A1: 使用场景
- **假设**: 大多数策略使用 CIDR 范围(非精确 IP)
- **验证**: 分析生产环境策略分布

#### A2: 性能要求
- **假设**: P99 延迟 <10μs 可满足业务需求
- **验证**: 与 DevOps 团队确认 SLA

#### A3: 策略分布
- **假设**: 策略优先级分布相对均匀
- **验证**: 监控策略命中率分布

## Out of Scope

### 明确不包含的功能

#### OS1: IPv6 支持
- **说明**: 当前仅支持 IPv4,IPv6 支持留待后续版本
- **原因**: LPM Trie 对 IPv6 的支持需要额外的 key 结构设计

#### OS2: 端口范围匹配
- **说明**: 不支持 `src_port: 8000-9000` 范围匹配
- **原因**: 端口范围需要额外的 interval tree 数据结构

#### OS3: 正则表达式匹配
- **说明**: 不支持基于字符串的正则匹配(如 hostname)
- **原因**: eBPF 中正则匹配性能开销过大

#### OS4: 动态策略更新
- **说明**: 策略更新需要重新加载 eBPF 程序
- **原因**: LPM Trie 的结构化特性导致在线更新复杂

#### OS5: 多租户隔离
- **说明**: Phase 2 不包含 per-namespace 策略隔离
- **原因**: 需要 eBPF map 嵌套或多副本,复杂度高

## Dependencies

### External Dependencies

#### D1: eBPF 工具链
- **依赖**: libbpf >= 0.6, bpftool, clang >= 12
- **风险**: 旧版本内核可能不支持 LPM Trie
- **缓解**: 提供版本检测和降级方案

#### D2: 内核 BPF Subsystem
- **依赖**: Kernel BPF JIT, Map Pinning
- **风险**: 内核 bug 可能导致 LPM 查找错误
- **缓解**: 充分测试,收集内核版本兼容性矩阵

### Internal Dependencies

#### D3: Server 端重构
- **依赖**: Server Policy Manager 模块需要扩展
- **负责人**: Server 团队
- **时间**: Phase 2 启动前完成 API 设计

#### D4: 测试基础设施
- **依赖**: 性能测试环境(10Gbps 网卡)
- **负责人**: QA 团队
- **时间**: Phase 2 开发期间

## Implementation Phases

### Phase 1: 短期优化(2 周)

#### Week 1: 代码优化
- [ ] 替换魔法数字为常量
- [ ] 实现早停逻辑
- [ ] 紧凑存储实现
- [ ] 单元测试

#### Week 2: 测试与部署
- [ ] 性能基准测试
- [ ] 集成测试
- [ ] 文档更新
- [ ] 灰度发布

### Phase 2: LPM Trie 重构(4 周)

#### Week 1: 设计与原型
- [ ] LPM Trie 架构设计评审
- [ ] 数据结构定义
- [ ] 原型实现与验证

#### Week 2-3: 核心开发
- [ ] eBPF LPM 查找逻辑实现
- [ ] Server 端 LPM 管理 API
- [ ] 策略迁移工具
- [ ] 单元测试与集成测试

#### Week 4: 测试与上线
- [ ] 性能压测
- [ ] 兼容性测试
- [ ] 文档完善
- [ ] 生产环境部署

## Risks & Mitigation

### Risk Matrix

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|---------|
| eBPF Verifier 拒绝 LPM 代码 | 高 | 中 | 提前原型验证,准备降级方案 |
| LPM 性能不如预期 | 高 | 低 | 性能建模,benchmark 对比 |
| 策略迁移失败 | 中 | 低 | 双写验证,回滚机制 |
| 内核兼容性问题 | 中 | 中 | 版本检测,Feature Flag 控制 |
| 开发时间超期 | 中 | 中 | 分阶段交付,核心功能优先 |

### Mitigation Strategies

#### M1: eBPF Verifier 风险
- **策略**: 提前在多个内核版本上验证原型代码
- **Plan B**: 保留 Array 存储作为 fallback

#### M2: 性能风险
- **策略**: 建立性能模型,预测 LPM 查找开销
- **Plan B**: 混合使用 Array + LPM(热点策略用 Array)

#### M3: 兼容性风险
- **策略**: 使用 Feature Flag 控制 LPM 启用
- **Plan B**: 提供配置选项允许禁用 LPM

## Appendix

### A. Performance Modeling

#### 当前性能分析

**线性扫描成本**:
```
假设:
- 策略数量: n
- 每次 map lookup: 50ns
- 每次 CIDR 比较: 20ns
- 总耗时: n * (50ns + 20ns) = n * 70ns

当 n=100:
  最坏情况: 100 * 70ns = 7μs
  平均情况: 50 * 70ns = 3.5μs
```

**LPM Trie 预期性能**:
```
假设:
- Trie 深度: log2(n)
- 每次 LPM lookup: 100ns (比 hash 稍慢)
- 细粒度过滤: 8 个候选 * 50ns = 400ns
- 总耗时: log2(n) * 100ns + 400ns

当 n=1000:
  log2(1000) ≈ 10
  总耗时: 10 * 100ns + 400ns = 1.4μs
```

**性能提升**:
- 100 条策略: 7μs → 1.4μs (5x 提升)
- 1000 条策略: N/A → 1.4μs (支持大规模)

### B. Reference Materials

#### Linux Kernel LPM Trie Documentation
- https://www.kernel.org/doc/html/latest/bpf/map_lpm_trie.html

#### eBPF Performance Best Practices
- https://www.brendangregg.com/blog/2019-01-01/learn-ebpf-tracing.html

#### CIDR Matching Algorithms
- "IP Lookup in Hardware for High-Speed Routing" (IEEE Paper)

### C. Glossary

- **LPM**: Longest Prefix Match(最长前缀匹配)
- **CIDR**: Classless Inter-Domain Routing(无类别域间路由)
- **Trie**: 前缀树数据结构
- **eBPF**: Extended Berkeley Packet Filter
- **Verifier**: eBPF 内核验证器,确保程序安全性
- **JIT**: Just-In-Time 编译,将 eBPF 字节码编译为本地机器码
