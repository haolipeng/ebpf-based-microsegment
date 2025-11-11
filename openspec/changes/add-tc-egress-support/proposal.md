# 提案: 添加 TC Egress Hook 支持

## 元数据

- **变更 ID**: add-tc-egress-support
- **标题**: 添加 TC Egress Hook 支持以实现双向流量控制
- **状态**: proposed
- **创建日期**: 2025-11-11
- **优先级**: high
- **类型**: enhancement / security
- **相关变更**: add-xdp-tc-dual-mode (依赖)

## 问题陈述

### 当前限制

当前微隔离系统**仅实现了 TC ingress hook**，存在严重的安全盲点：

1. **单向控制**: 只能控制进入容器/工作负载的流量（ingress）
2. **无 egress 策略**: 无法控制容器主动发起的连接（egress）
3. **零信任违背**: 无法实现"默认拒绝，显式允许"的完整零信任模型
4. **合规风险**: 不符合严格的微隔离产品要求

### 安全影响

**攻击场景示例**:
```
场景 1: 数据外泄
- Pod A 被攻破，攻击者想将数据发送到外部恶意服务器
- 当前系统: ✅ Pod A 可以自由发起任何 egress 连接（只要目标允许 ingress）
- 理想系统: ❌ 应该拒绝 Pod A 到未授权目标的 egress 流量

场景 2: 横向移动
- Pod A 被攻破，攻击者扫描内网其他服务
- 当前系统: ✅ Pod A 可以主动连接任何内部服务
- 理想系统: ❌ 应该只允许 Pod A 访问其授权的服务列表

场景 3: C&C 通信
- 恶意软件尝试连接到命令控制服务器
- 当前系统: ✅ 可以建立连接（只要 C&C 服务器允许入站）
- 理想系统: ❌ 应该阻止所有未授权的外部连接
```

### 业界标准

参考 NeuVector 等成熟微隔离产品，**双向流量控制是基本要求**：
- **Ingress 策略**: 控制谁可以访问我
- **Egress 策略**: 控制我可以访问谁

## 解决方案

### 核心方案: 添加 TC Egress Hook

在当前 TC 模式下，添加 **TC Egress Hook**，实现真正的双向策略控制。

#### 技术实现

**TC 模式扩展** (TCX + Legacy TC):
```go
// 当前实现 (仅 ingress)
link.AttachTCX(link.TCXOptions{
    Program: l.objs.TcMicrosegmentFilter,
    Attach:  ebpf.AttachTCXIngress,  // ❌ 仅 ingress
})

// 新实现 (双向)
// 1. Ingress Hook
ingressLink, err := link.AttachTCX(link.TCXOptions{
    Program: l.objs.TcIngressFilter,  // 或复用同一个 program
    Attach:  ebpf.AttachTCXIngress,
})

// 2. Egress Hook ✅
egressLink, err := link.AttachTCX(link.TCXOptions{
    Program: l.objs.TcEgressFilter,   // 或复用同一个 program
    Attach:  ebpf.AttachTCXEgress,    // ✅ 新增 egress
})
```

#### XDP 模式说明

**XDP 本身不支持 egress**（仅在数据包接收路径工作），但可以结合使用：
- **XDP**: 高性能 ingress 处理
- **TC Egress**: Egress 流量控制

未来可以考虑 XDP + TC Egress 混合模式。

## 设计概述

### 架构变更

```
┌─────────────────────────────────────────────────────────────┐
│                   数据平面架构 (TC 模式)                      │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  Container/Pod                                                │
│  ┌──────────────────────────────────┐                       │
│  │                                  │                       │
│  │  Application                     │                       │
│  │                                  │                       │
│  └──────────────────────────────────┘                       │
│            │                    ▲                            │
│            │ Egress             │ Ingress                    │
│            ▼                    │                            │
│  ┌─────────────────────────────────────────┐                │
│  │    Network Namespace (eth0/veth)        │                │
│  └─────────────────────────────────────────┘                │
│            │                    ▲                            │
│            │                    │                            │
│  ┌─────────▼────────┐  ┌───────┴──────────┐                │
│  │  TC Egress Hook  │  │  TC Ingress Hook │                │
│  │  ✅ 新增          │  │  ✅ 已存在        │                │
│  └──────────────────┘  └──────────────────┘                │
│            │                    ▲                            │
│            │                    │                            │
│            ▼                    │                            │
│  ┌──────────────────────────────────────────┐               │
│  │      eBPF 策略匹配引擎 (共享 Maps)        │               │
│  │  - policy_map                            │               │
│  │  - wildcard_policy_map                   │               │
│  │  - session_map                           │               │
│  │  - stats_map                             │               │
│  └──────────────────────────────────────────┘               │
│                                                               │
└─────────────────────────────────────────────────────────────┘
```

### 策略匹配增强

为策略添加 **方向字段**:

```c
// 策略定义增强
struct policy_key {
    __u32 src_ip;
    __u32 dst_ip;
    __u16 dst_port;
    __u8  protocol;
    __u8  direction;  // ✅ 新增: 0=any, 1=ingress, 2=egress
} __attribute__((packed));
```

### 流事件增强

流事件记录方向信息:

```c
// 流事件增强
struct flow_event {
    struct flow_key key;
    __u64 timestamp;
    __u32 policy_id;
    __u8  action;       // 0=deny, 1=allow
    __u8  direction;    // ✅ 新增: 1=ingress, 2=egress
    __u8  reserved[2];
} __attribute__((packed));
```

## 成功标准

### 功能要求

1. **✅ Egress 策略执行**: TC Egress Hook 成功拦截并执行 egress 策略
2. **✅ 双向策略匹配**: 同一个 5-tuple 可以配置不同的 ingress/egress 策略
3. **✅ 会话状态共享**: Ingress 和 Egress 共享同一个 session_map
4. **✅ 流事件区分**: Flow events 能够区分 ingress/egress 方向
5. **✅ 向后兼容**: 不破坏现有的 TC ingress 功能

### 性能要求

1. **延迟**: Egress 处理延迟 < 5μs (与 ingress 相当)
2. **吞吐量**: 不影响现有吞吐量性能
3. **资源**: 内存增长 < 20% (主要是双倍的 eBPF 程序)

### 测试要求

1. **单元测试**: TCLoader egress attach/detach 测试
2. **集成测试**: 双向流量控制端到端测试
3. **性能测试**: Egress hook 性能基准测试
4. **安全测试**: 验证 egress 策略能够阻止未授权连接

## 实施计划

### Phase 1: eBPF 程序扩展
- [ ] Task 1.1: 添加 direction 字段到 policy_key
- [ ] Task 1.2: 修改策略匹配逻辑支持方向过滤
- [ ] Task 1.3: 添加 direction 到 flow_event
- [ ] Task 1.4: 更新统计指标（区分 ingress/egress）

### Phase 2: TC Loader 扩展
- [ ] Task 2.1: TCLoader 支持 AttachTCXEgress
- [ ] Task 2.2: Legacy TC 支持 egress filter 添加
- [ ] Task 2.3: 生命周期管理（同时加载/卸载 ingress+egress）
- [ ] Task 2.4: 单元测试

### Phase 3: 策略管理扩展
- [ ] Task 3.1: Policy 结构添加 direction 字段
- [ ] Task 3.2: PolicyManager 支持方向感知策略
- [ ] Task 3.3: API 扩展（policy CRUD 支持 direction）
- [ ] Task 3.4: 策略验证逻辑更新

### Phase 4: 集成测试
- [ ] Task 4.1: Egress 策略执行测试
- [ ] Task 4.2: 双向策略冲突测试
- [ ] Task 4.3: 会话状态一致性测试
- [ ] Task 4.4: 性能回归测试

### Phase 5: 文档与示例
- [ ] Task 5.1: 更新架构文档
- [ ] Task 5.2: Egress 策略配置示例
- [ ] Task 5.3: 安全最佳实践文档
- [ ] Task 5.4: API 文档更新

## 依赖关系

### 前置依赖
- **add-xdp-tc-dual-mode**: 必须完成（TC 基础设施已就绪）

### 后续变更
- **add-xdp-tc-hybrid-mode** (可选): XDP ingress + TC egress 混合模式
- **add-label-based-policy** (独立): 基于标签的策略（可以同时进行）

## 风险与缓解

### 技术风险

**风险 1: 性能影响**
- **描述**: 双倍的 eBPF hook 可能影响性能
- **缓解**:
  - 共享策略匹配逻辑（复用代码）
  - 性能测试验证
  - 提供配置选项（可仅启用 ingress）

**风险 2: 策略复杂度**
- **描述**: 双向策略可能导致配置错误
- **缓解**:
  - 提供清晰的 API 和文档
  - 策略验证工具
  - 合理的默认配置

**风险 3: 兼容性**
- **描述**: 可能与旧版本配置不兼容
- **缓解**:
  - 向后兼容（默认 direction=any）
  - 平滑迁移路径
  - 版本检查和警告

## 替代方案

### 方案 1: NeuVector 风格的 veth pair
- **描述**: 使用 veth pair，在两端都用 TC ingress hook
- **优点**: 两个方向都使用同一个 ingress 程序
- **缺点**:
  - 需要创建额外的 veth pair（侵入性强）
  - 增加网络拓扑复杂度
  - 不适合我们的架构（我们直接 attach 到容器 netns）

### 方案 2: XDP + TC Egress 混合
- **描述**: Ingress 用 XDP，Egress 用 TC
- **优点**: 发挥 XDP 高性能优势
- **缺点**:
  - 模式切换复杂
  - 需要两套不同的 eBPF 程序
  - 可作为未来优化方向

### 推荐方案: 直接 TC Egress Hook
- **理由**:
  - 实现简单，架构清晰
  - 性能足够（TC 本身已经很快）
  - 与现有 TC 模式一致
  - 易于维护和测试

## 参考资料

1. **NeuVector 实现**: `source-references/neuvector/agent/pipe/tc.go`
2. **TCX 文档**: Linux kernel 6.6+ TCX (Traffic Control eXpress)
3. **eBPF 最佳实践**: Cilium eBPF library documentation
4. **零信任网络**: NIST SP 800-207 Zero Trust Architecture

## 批准与验证

### 需要批准
- [ ] 架构评审
- [ ] 安全评审
- [ ] 性能评审

### 验证清单
- [ ] 所有测试通过（单元 + 集成 + 性能）
- [ ] 文档完整
- [ ] 向后兼容性验证
- [ ] 安全测试验证 egress 策略有效性

---

**提案人**: Claude Code AI Assistant
**日期**: 2025-11-11
**版本**: 1.0
