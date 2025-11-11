# 任务清单: TC Egress Hook 支持

## 元数据

- **变更 ID**: add-tc-egress-support
- **创建日期**: 2025-11-11
- **预计工期**: 10-14 天
- **优先级**: High

## 任务概览

本变更将为微隔离系统添加 TC Egress Hook 支持，实现真正的双向流量控制（Ingress + Egress）。

### 依赖关系
- **前置依赖**: `add-xdp-tc-dual-mode` 必须完成
- **阻塞变更**: 无

---

## Phase 1: eBPF 程序扩展 (预计 2-3 天)

### Task 1.1: 添加方向字段到策略 Key

**描述**: 修改 `policy_match.h`，为策略匹配添加方向感知能力。

**文件**:
- `src/bpf/policy_match.h`

**实施步骤**:
1. 添加方向常量定义:
   ```c
   #define DIR_ANY     0  // 双向
   #define DIR_INGRESS 1  // Ingress
   #define DIR_EGRESS  2  // Egress
   ```

2. 修改 `policy_key` 结构:
   ```c
   struct policy_key {
       __u32 src_ip;
       __u32 dst_ip;
       __u16 dst_port;
       __u8  protocol;
       __u8  direction;  // ✅ 新增
   } __attribute__((packed));
   ```

3. 修改 `wildcard_key` 结构:
   ```c
   struct wildcard_key {
       __u32 dst_ip;
       __u16 dst_port;
       __u8  protocol;
       __u8  direction;  // ✅ 新增
   } __attribute__((packed));
   ```

**验收标准**:
- [ ] 头文件编译无错误
- [ ] 结构体大小正确 (policy_key: 12 bytes, wildcard_key: 8 bytes)
- [ ] 常量定义清晰

**状态**: ⏳ 待开始

---

### Task 1.2: 修改策略匹配逻辑支持方向过滤

**描述**: 更新 `tc_microsegment.bpf.c` 的策略匹配逻辑，支持基于方向的策略匹配。

**文件**:
- `src/bpf/tc_microsegment.bpf.c`

**实施步骤**:
1. 在数据包处理函数中检测方向:
   ```c
   SEC("tc")
   int tc_microsegment_filter(struct __sk_buff *skb) {
       // 检测方向: ingress_ifindex != 0 表示 ingress
       __u8 direction = (skb->ingress_ifindex != 0) ? DIR_INGRESS : DIR_EGRESS;

       // ...
   }
   ```

2. 策略查找时包含方向:
   ```c
   struct policy_key key = {
       .src_ip = pkt.src_ip,
       .dst_ip = pkt.dst_ip,
       .dst_port = pkt.dst_port,
       .protocol = pkt.protocol,
       .direction = direction,  // ✅
   };

   struct policy_value *policy = bpf_map_lookup_elem(&policy_map, &key);
   ```

3. 如果没有方向特定策略，尝试 `DIR_ANY`:
   ```c
   if (!policy) {
       key.direction = DIR_ANY;
       policy = bpf_map_lookup_elem(&policy_map, &key);
   }
   ```

4. 同样更新通配符策略匹配逻辑。

**验收标准**:
- [ ] eBPF 程序编译成功
- [ ] 策略匹配逻辑正确（先匹配方向特定策略，再回退到 DIR_ANY）
- [ ] 代码注释清晰

**状态**: ⏳ 待开始

---

### Task 1.3: 添加方向到流事件

**描述**: 修改 `flow_processing.h` 和流事件发送逻辑，记录流量方向。

**文件**:
- `src/bpf/flow_processing.h`
- `src/bpf/tc_microsegment.bpf.c`

**实施步骤**:
1. 修改 `flow_event` 结构:
   ```c
   struct flow_event {
       struct flow_key key;
       __u64 timestamp;
       __u32 policy_id;
       __u8  action;       // 0=deny, 1=allow
       __u8  direction;    // ✅ 新增
       __u8  reserved[2];
   } __attribute__((packed));
   ```

2. 更新流事件发送函数:
   ```c
   static __always_inline void
   emit_flow_event(struct flow_key *key, __u8 direction, __u8 action, __u32 policy_id) {
       struct flow_event event = {
           .key = *key,
           .timestamp = bpf_ktime_get_ns(),
           .policy_id = policy_id,
           .action = action,
           .direction = direction,  // ✅
       };

       bpf_ringbuf_output(&flow_events, &event, sizeof(event), 0);
   }
   ```

3. 在所有调用 `emit_flow_event` 的地方传递 direction 参数。

**验收标准**:
- [ ] 头文件编译成功
- [ ] Flow event 结构体大小正确（48 bytes）
- [ ] 流事件包含正确的 direction 信息

**状态**: ⏳ 待开始

---

### Task 1.4: 更新统计指标（分方向统计）

**描述**: 添加 Ingress/Egress 分方向的统计指标。

**文件**:
- `src/bpf/tc_microsegment.bpf.c`
- `src/bpf/policy_match.h` (统计常量)

**实施步骤**:
1. 添加新的统计指标常量:
   ```c
   #define STAT_INGRESS_PACKETS  8
   #define STAT_EGRESS_PACKETS   9
   #define STAT_INGRESS_DENIED   10
   #define STAT_EGRESS_DENIED    11
   ```

2. 在数据包处理时更新分方向统计:
   ```c
   // 更新总流量统计
   if (direction == DIR_INGRESS) {
       increment_stat(STAT_INGRESS_PACKETS);
   } else {
       increment_stat(STAT_EGRESS_PACKETS);
   }

   // 更新拒绝统计
   if (action == ACTION_DENY) {
       if (direction == DIR_INGRESS) {
           increment_stat(STAT_INGRESS_DENIED);
       } else {
           increment_stat(STAT_EGRESS_DENIED);
       }
   }
   ```

**验收标准**:
- [ ] 新统计指标定义正确
- [ ] 统计更新逻辑准确
- [ ] 不影响现有统计指标

**状态**: ⏳ 待开始

---

## Phase 2: TC Loader 扩展 (预计 2-3 天)

### Task 2.1: TCLoader 支持 AttachTCXEgress

**描述**: 扩展 `TCLoader` 支持 TCX Egress Hook attach/detach。

**文件**:
- `src/agent/pkg/dataplane/tc_loader.go`

**实施步骤**:
1. 修改 `TCLoader` 结构:
   ```go
   type TCLoader struct {
       mode        DataPlaneMode
       iface       string
       ifaceIdx    int
       objs        *bpfObjects
       ingressLink link.Link    // 已存在
       egressLink  link.Link    // ✅ 新增
       maps        *DataPlaneMaps
   }
   ```

2. 实现 `attachTCXEgress()` 方法:
   ```go
   func (l *TCLoader) attachTCXEgress() error {
       link, err := link.AttachTCX(link.TCXOptions{
           Program:   l.objs.TcMicrosegmentFilter,
           Attach:    ebpf.AttachTCXEgress,  // ✅
           Interface: l.ifaceIdx,
       })
       if err != nil {
           return fmt.Errorf("attaching TCX egress: %w", err)
       }
       l.egressLink = link
       return nil
   }
   ```

3. 实现 `detachTCXEgress()` 方法:
   ```go
   func (l *TCLoader) detachTCXEgress() error {
       if l.egressLink != nil {
           if err := l.egressLink.Close(); err != nil {
               return fmt.Errorf("detaching TCX egress: %w", err)
           }
           l.egressLink = nil
       }
       return nil
   }
   ```

4. 更新 `Load()` 方法调用 `attachTCXEgress()`。
5. 更新 `Unload()` 方法调用 `detachTCXEgress()`。

**验收标准**:
- [ ] TCX Egress Hook 成功 attach 到网卡
- [ ] Detach 时正确清理
- [ ] 错误处理完善
- [ ] 日志输出清晰

**状态**: ⏳ 待开始

---

### Task 2.2: Legacy TC 支持 Egress Filter

**描述**: 为 Legacy TC 模式添加 Egress Filter 支持（使用 netlink API）。

**文件**:
- `src/agent/pkg/dataplane/tc_loader.go`

**实施步骤**:
1. 实现 `attachLegacyTCEgress()` 方法:
   ```go
   func (l *TCLoader) attachLegacyTCEgress() error {
       // 1. 确保 clsact qdisc 存在
       qdisc := &netlink.Clsact{
           QdiscAttrs: netlink.QdiscAttrs{
               LinkIndex: l.ifaceIdx,
               Handle:    netlink.MakeHandle(0xffff, 0),
               Parent:    netlink.HANDLE_CLSACT,
           },
       }
       if err := netlink.QdiscAdd(qdisc); err != nil && !os.IsExist(err) {
           return fmt.Errorf("adding clsact qdisc: %w", err)
       }

       // 2. 添加 egress filter
       filter := &netlink.BpfFilter{
           FilterAttrs: netlink.FilterAttrs{
               LinkIndex: l.ifaceIdx,
               Parent:    netlink.HANDLE_MIN_EGRESS,  // ✅ Egress parent
               Handle:    1,
               Protocol:  unix.ETH_P_ALL,
               Priority:  1,
           },
           Fd:           l.objs.TcMicrosegmentFilter.FD(),
           Name:         "tc/egress",
           DirectAction: true,
       }

       if err := netlink.FilterAdd(filter); err != nil {
           return fmt.Errorf("adding egress filter: %w", err)
       }

       return nil
   }
   ```

2. 实现 `detachLegacyTCEgress()` 方法:
   ```go
   func (l *TCLoader) detachLegacyTCEgress() error {
       // 删除 egress filter
       // ...
   }
   ```

3. 更新 `Load()` 和 `Unload()` 方法。

**验收标准**:
- [ ] Legacy TC Egress Filter 成功添加
- [ ] Detach 时正确删除
- [ ] 与 Ingress Filter 共存
- [ ] 测试在旧内核（< 6.6）上工作

**状态**: ⏳ 待开始

---

### Task 2.3: 生命周期管理（同时加载/卸载 Ingress+Egress）

**描述**: 确保 Ingress 和 Egress Hook 的生命周期一致性。

**文件**:
- `src/agent/pkg/dataplane/tc_loader.go`

**实施步骤**:
1. 更新 `Load()` 方法:
   ```go
   func (l *TCLoader) Load() error {
       // 1. 加载 eBPF 程序
       // ...

       // 2. Attach Ingress
       if err := l.attachIngress(); err != nil {
           return fmt.Errorf("attaching ingress: %w", err)
       }

       // 3. Attach Egress
       if err := l.attachEgress(); err != nil {
           l.detachIngress()  // ✅ 清理 ingress
           return fmt.Errorf("attaching egress: %w", err)
       }

       // 4. 初始化 Maps
       // ...

       log.Infof("✓ TC dataplane loaded (ingress+egress)")
       return nil
   }
   ```

2. 更新 `Unload()` 方法:
   ```go
   func (l *TCLoader) Unload() error {
       var errs []error

       // 1. Detach Egress
       if err := l.detachEgress(); err != nil {
           errs = append(errs, err)
       }

       // 2. Detach Ingress
       if err := l.detachIngress(); err != nil {
           errs = append(errs, err)
       }

       // 3. Close objects
       // ...

       return errors.Join(errs...)
   }
   ```

3. 实现 `Reload()` 方法支持热重载。

**验收标准**:
- [ ] Load 失败时正确清理已加载的部分
- [ ] Unload 清理所有资源（即使部分失败）
- [ ] Reload 保持连接状态

**状态**: ⏳ 待开始

---

### Task 2.4: TC Loader 单元测试

**描述**: 为 TCLoader 的 Egress 功能编写单元测试。

**文件**:
- `src/agent/pkg/dataplane/tc_loader_test.go`

**实施步骤**:
1. 测试 TCX Egress attach/detach:
   ```go
   func TestTCLoaderTCXEgress(t *testing.T) {
       // 需要 root 权限和真实网卡
       if os.Geteuid() != 0 {
           t.Skip("Requires root")
       }

       loader, _ := NewTCLoader(ModeTCX, "lo", 1)
       defer loader.Unload()

       err := loader.Load()
       assert.NoError(t, err)
       assert.NotNil(t, loader.egressLink)
   }
   ```

2. 测试 Legacy TC Egress attach/detach。

3. 测试双向 Hook 加载失败时的清理。

4. 测试 Unload 幂等性。

**验收标准**:
- [ ] 所有测试通过
- [ ] 测试覆盖率 > 80%
- [ ] 测试在 CI 中可运行（可能需要特权容器）

**状态**: ⏳ 待开始

---

## Phase 3: 策略管理扩展 (预计 2 天)

### Task 3.1: Policy 结构添加 Direction 字段

**描述**: 为 `Policy` 结构添加 `Direction` 字段，支持方向感知策略。

**文件**:
- `src/agent/pkg/policy/policy.go`

**实施步骤**:
1. 修改 `Policy` 结构:
   ```go
   type Policy struct {
       RuleID    uint32
       SrcIP     string
       DstIP     string
       DstPort   uint16
       Protocol  string
       Action    string
       Direction string  // ✅ 新增: "any", "ingress", "egress"
   }
   ```

2. 添加方向常量:
   ```go
   const (
       DirectionAny     = "any"
       DirectionIngress = "ingress"
       DirectionEgress  = "egress"
   )
   ```

3. 添加转换函数:
   ```go
   func (p *Policy) GetDirectionValue() uint8 {
       switch strings.ToLower(p.Direction) {
       case DirectionIngress:
           return 1  // DIR_INGRESS
       case DirectionEgress:
           return 2  // DIR_EGRESS
       default:
           return 0  // DIR_ANY
       }
   }
   ```

4. 添加验证函数:
   ```go
   func (p *Policy) Validate() error {
       // 验证 direction
       if p.Direction != "" &&
          p.Direction != DirectionAny &&
          p.Direction != DirectionIngress &&
          p.Direction != DirectionEgress {
           return fmt.Errorf("invalid direction: %s", p.Direction)
       }
       // ... 其他验证
       return nil
   }
   ```

**验收标准**:
- [ ] Policy 结构包含 Direction 字段
- [ ] Direction 验证逻辑正确
- [ ] 转换函数返回正确的 eBPF 值
- [ ] 向后兼容（Direction 为空时默认 "any"）

**状态**: ⏳ 待开始

---

### Task 3.2: PolicyManager 支持方向感知策略

**描述**: 更新 `PolicyManager` 的 `AddPolicy()` 方法支持 direction。

**文件**:
- `src/agent/pkg/policy/manager.go`

**实施步骤**:
1. 更新 `AddPolicy()` 方法:
   ```go
   func (pm *PolicyManager) AddPolicy(policy *Policy) error {
       // 验证策略
       if err := policy.Validate(); err != nil {
           return err
       }

       // 默认 direction
       if policy.Direction == "" {
           policy.Direction = DirectionAny
       }

       direction := policy.GetDirectionValue()

       // 构造 eBPF policy key
       key := bpfPolicyKey{
           SrcIP:     ipToBinary(policy.SrcIP),
           DstIP:     ipToBinary(policy.DstIP),
           DstPort:   policy.DstPort,
           Protocol:  protocolToBinary(policy.Protocol),
           Direction: direction,  // ✅
       }

       value := bpfPolicyValue{
           RuleID: policy.RuleID,
           Action: actionToBinary(policy.Action),
       }

       // 写入 policy_map
       if err := pm.policyMap.Put(&key, &value); err != nil {
           return fmt.Errorf("adding policy to map: %w", err)
       }

       log.Infof("Policy added: rule_id=%d, direction=%s",
                 policy.RuleID, policy.Direction)
       return nil
   }
   ```

2. 更新 `DeletePolicy()` 方法（需要 direction 才能唯一确定 key）。

3. 更新 `ListPolicies()` 方法返回 direction。

**验收标准**:
- [ ] AddPolicy 正确处理 direction
- [ ] DeletePolicy 需要 direction 参数
- [ ] ListPolicies 返回 direction
- [ ] 向后兼容旧配置

**状态**: ⏳ 待开始

---

### Task 3.3: API 扩展（Policy CRUD 支持 Direction）

**描述**: 更新 Policy API 请求/响应结构，支持 direction 参数。

**文件**:
- `src/agent/pkg/api/handlers.go`
- `src/agent/pkg/api/models.go`

**实施步骤**:
1. 更新 `CreatePolicyRequest`:
   ```go
   type CreatePolicyRequest struct {
       SrcIP     string `json:"src_ip" binding:"required"`
       DstIP     string `json:"dst_ip" binding:"required"`
       DstPort   uint16 `json:"dst_port"`
       Protocol  string `json:"protocol" binding:"required"`
       Action    string `json:"action" binding:"required"`
       Direction string `json:"direction"`  // ✅ 新增
   }
   ```

2. 更新 `handleCreatePolicy`:
   ```go
   func (s *Server) handleCreatePolicy(c *gin.Context) {
       var req CreatePolicyRequest
       if err := c.ShouldBindJSON(&req); err != nil {
           c.JSON(400, gin.H{"error": err.Error()})
           return
       }

       // 验证 direction
       direction := req.Direction
       if direction == "" {
           direction = policy.DirectionAny
       }
       if direction != policy.DirectionAny &&
          direction != policy.DirectionIngress &&
          direction != policy.DirectionEgress {
           c.JSON(400, gin.H{"error": "invalid direction"})
           return
       }

       // 创建策略
       p := &policy.Policy{
           RuleID:    generateRuleID(),
           SrcIP:     req.SrcIP,
           DstIP:     req.DstIP,
           DstPort:   req.DstPort,
           Protocol:  req.Protocol,
           Action:    req.Action,
           Direction: direction,
       }

       if err := s.policyManager.AddPolicy(p); err != nil {
           c.JSON(500, gin.H{"error": err.Error()})
           return
       }

       c.JSON(200, gin.H{"rule_id": p.RuleID})
   }
   ```

3. 更新 `PolicyResponse` 包含 direction。

4. 更新 Flow API 响应包含 direction。

**验收标准**:
- [ ] POST /api/v1/policies 接受 direction 参数
- [ ] GET /api/v1/policies 返回 direction
- [ ] DELETE /api/v1/policies/:id 需要 direction
- [ ] GET /api/v1/flows 返回 direction
- [ ] API 文档更新

**状态**: ⏳ 待开始

---

### Task 3.4: 策略验证逻辑更新

**描述**: 更新策略验证逻辑，处理 direction 字段。

**文件**:
- `src/agent/pkg/policy/validator.go` (如果存在)
- `src/agent/pkg/policy/policy.go`

**实施步骤**:
1. 验证 direction 取值合法。
2. 验证策略冲突（同一 5-tuple + direction 不能有多个策略）。
3. 添加警告：如果 ingress 和 egress 策略不一致。

**验收标准**:
- [ ] Direction 验证正确
- [ ] 冲突检测工作
- [ ] 警告信息清晰

**状态**: ⏳ 待开始

---

## Phase 4: 集成测试 (预计 2-3 天)

### Task 4.1: Egress 策略执行测试

**描述**: 端到端测试 Egress 策略能够正确阻止出站流量。

**文件**:
- `tests/integration/egress_policy_test.go` (新建)

**测试场景**:
1. **场景 1: Egress Deny**
   - 添加 egress deny 策略
   - 尝试从容器发起连接
   - 验证被阻止
   - 验证统计 (egress_denied++)

2. **场景 2: Egress Allow**
   - 添加 egress allow 策略
   - 从容器发起连接
   - 验证成功
   - 验证统计 (egress_allowed++)

3. **场景 3: Egress 默认策略**
   - 无任何 egress 策略
   - 验证默认行为（应该放行或拒绝，取决于配置）

**验收标准**:
- [ ] 所有场景测试通过
- [ ] 测试可重复运行
- [ ] 测试隔离良好（不影响其他测试）

**状态**: ⏳ 待开始

---

### Task 4.2: 双向策略冲突测试

**描述**: 测试 Ingress 和 Egress 策略不一致的情况。

**文件**:
- `tests/integration/bidirectional_policy_test.go` (新建)

**测试场景**:
1. **场景 1: Ingress Allow, Egress Deny**
   - Ingress: 允许外部访问 Pod A:80
   - Egress: 拒绝 Pod A 主动访问外部
   - 测试外部访问 Pod A:80 → ✅ 成功
   - 测试 Pod A 访问外部 → ❌ 失败

2. **场景 2: Ingress Deny, Egress Allow**
   - Ingress: 拒绝外部访问 Pod A
   - Egress: 允许 Pod A 访问外部
   - 测试外部访问 Pod A → ❌ 失败
   - 测试 Pod A 访问外部 → ✅ 成功

3. **场景 3: 双向 Allow**
   - Ingress: 允许
   - Egress: 允许
   - 测试双向通信 → ✅ 都成功

**验收标准**:
- [ ] 所有场景测试通过
- [ ] 策略冲突不影响系统稳定性
- [ ] 日志清晰显示策略决策

**状态**: ⏳ 待开始

---

### Task 4.3: 会话状态一致性测试

**描述**: 验证 Ingress 和 Egress 共享同一个 session_map。

**文件**:
- `tests/integration/session_state_test.go` (新建)

**测试场景**:
1. **场景 1: Egress 创建会话**
   - Pod A 发起连接到 Pod B
   - 验证 session_map 中创建了 session
   - Pod B 返回响应
   - 验证使用同一个 session

2. **场景 2: Ingress 创建会话**
   - 外部发起连接到 Pod A
   - 验证 session_map 中创建了 session
   - Pod A 返回响应
   - 验证使用同一个 session

3. **场景 3: 会话超时清理**
   - 创建会话
   - 等待超时
   - 验证会话被清理

**验收标准**:
- [ ] 会话在 Ingress 和 Egress 之间共享
- [ ] 会话统计正确（bytes, packets）
- [ ] 会话生命周期管理正确

**状态**: ⏳ 待开始

---

### Task 4.4: 性能回归测试

**描述**: 验证添加 Egress Hook 后性能不会显著下降。

**文件**:
- `tests/performance/egress_performance_test.go` (新建)
- `scripts/benchmark_egress.sh` (新建)

**测试指标**:
1. **延迟测试**:
   - 测试 Egress Hook 增加的延迟
   - 目标: < 5μs

2. **吞吐量测试**:
   - 使用 iperf3 测试吞吐量
   - 目标: 吞吐量下降 < 5%

3. **CPU 使用率**:
   - 测试 eBPF 程序 CPU 占用
   - 目标: CPU 增加 < 10%

4. **内存使用**:
   - 测试内存增长
   - 目标: 内存增加 < 20%

**验收标准**:
- [ ] 所有性能指标满足目标
- [ ] 性能测试结果可复现
- [ ] 生成性能报告

**状态**: ⏳ 待开始

---

## Phase 5: 文档与示例 (预计 1 天)

### Task 5.1: 更新架构文档

**描述**: 更新系统架构文档，反映 Egress Hook 的加入。

**文件**:
- `docs/architecture.md`
- `docs/dataplane.md`

**内容**:
1. 更新数据平面架构图（包含 Ingress + Egress）。
2. 说明双向流量控制的实现原理。
3. 更新 TC 模式说明。

**验收标准**:
- [ ] 架构图清晰准确
- [ ] 文档更新完整
- [ ] 技术细节准确

**状态**: ⏳ 待开始

---

### Task 5.2: Egress 策略配置示例

**描述**: 提供清晰的 Egress 策略配置示例和最佳实践。

**文件**:
- `docs/policy-examples.md` (新建或更新)
- `examples/egress-policies.yaml` (新建)

**内容**:
1. **示例 1: Web 服务策略**
   ```yaml
   # 允许外部访问 Web 服务
   - rule_id: 1
     src_ip: "0.0.0.0/0"
     dst_ip: "web-pod/32"
     dst_port: 80
     protocol: tcp
     action: allow
     direction: ingress

   # 仅允许访问数据库
   - rule_id: 2
     src_ip: "web-pod/32"
     dst_ip: "db-pod/32"
     dst_port: 5432
     protocol: tcp
     action: allow
     direction: egress

   # 拒绝其他 egress
   - rule_id: 999
     src_ip: "web-pod/32"
     dst_ip: "0.0.0.0/0"
     dst_port: 0
     protocol: any
     action: deny
     direction: egress
   ```

2. **示例 2: 数据库策略**
3. **示例 3: 微服务间通信策略**

**验收标准**:
- [ ] 示例清晰易懂
- [ ] 覆盖常见场景
- [ ] 包含最佳实践说明

**状态**: ⏳ 待开始

---

### Task 5.3: 安全最佳实践文档

**描述**: 编写 Egress 策略的安全最佳实践文档。

**文件**:
- `docs/security-best-practices.md` (更新)

**内容**:
1. **零信任原则**: 默认拒绝所有 egress，显式允许需要的
2. **最小权限**: 每个 Pod 只允许访问必需的目标
3. **分层策略**: Ingress + Egress 双向控制
4. **审计日志**: 记录所有 egress 拒绝事件
5. **定期审查**: 定期检查 egress 策略的必要性

**验收标准**:
- [ ] 最佳实践文档完整
- [ ] 包含反模式说明
- [ ] 提供安全检查清单

**状态**: ⏳ 待开始

---

### Task 5.4: API 文档更新

**描述**: 更新 API 文档，添加 direction 参数说明。

**文件**:
- `docs/api.md`
- OpenAPI 规范文件（如果有）

**内容**:
1. 更新 `POST /api/v1/policies` 请求参数
2. 更新 `GET /api/v1/policies` 响应格式
3. 更新 `GET /api/v1/flows` 响应格式
4. 添加 direction 参数说明和示例

**验收标准**:
- [ ] API 文档完整准确
- [ ] 包含请求/响应示例
- [ ] OpenAPI 规范更新（如果有）

**状态**: ⏳ 待开始

---

## 任务统计

### 按 Phase

| Phase | 任务数 | 预计时间 | 状态 |
|-------|--------|---------|------|
| Phase 1: eBPF 程序扩展 | 4 | 2-3 天 | ⏳ 未开始 |
| Phase 2: TC Loader 扩展 | 4 | 2-3 天 | ⏳ 未开始 |
| Phase 3: 策略管理扩展 | 4 | 2 天 | ⏳ 未开始 |
| Phase 4: 集成测试 | 4 | 2-3 天 | ⏳ 未开始 |
| Phase 5: 文档与示例 | 4 | 1 天 | ⏳ 未开始 |
| **总计** | **20** | **10-14 天** | - |

### 按状态

- ⏳ 待开始: 20 个
- 🏗️ 进行中: 0 个
- ✅ 已完成: 0 个
- ⏭️ 已跳过: 0 个

### 按优先级

- 🔴 高优先级 (阻塞): Phase 1, Phase 2 (共 8 个任务)
- 🟡 中优先级: Phase 3, Phase 4 (共 8 个任务)
- 🟢 低优先级: Phase 5 (共 4 个任务)

## 里程碑

### Milestone 1: eBPF 基础完成 (Phase 1)
- **目标日期**: Day 3
- **交付物**: eBPF 程序支持方向感知策略匹配
- **验收**: eBPF 程序编译通过，策略匹配逻辑正确

### Milestone 2: TC Loader 完成 (Phase 2)
- **目标日期**: Day 6
- **交付物**: TCLoader 支持 Ingress + Egress 双向 attach
- **验收**: TCLoader 单元测试通过

### Milestone 3: 策略管理完成 (Phase 3)
- **目标日期**: Day 8
- **交付物**: Policy API 支持 direction 参数
- **验收**: API 测试通过

### Milestone 4: 集成测试完成 (Phase 4)
- **目标日期**: Day 11
- **交付物**: 端到端测试验证双向策略
- **验收**: 所有集成测试和性能测试通过

### Milestone 5: 文档完成 (Phase 5)
- **目标日期**: Day 12
- **交付物**: 完整的文档和示例
- **验收**: 文档评审通过

### Milestone 6: 变更就绪 (Ready for Merge)
- **目标日期**: Day 14
- **交付物**: 所有任务完成，评审通过
- **验收**: 代码评审通过，CI 通过，文档完整

## 风险与依赖

### 高风险任务

| 任务 | 风险 | 缓解措施 |
|------|------|---------|
| Task 1.2 | eBPF 程序复杂度增加 | 充分测试，代码审查 |
| Task 2.2 | Legacy TC netlink API 复杂 | 参考 Cilium 实现 |
| Task 4.4 | 性能可能不达标 | 提前进行性能测试，优化 eBPF 代码 |

### 关键依赖

- **内核版本**: TCX 需要 kernel >= 6.6（Legacy TC 可回退）
- **cilium/ebpf 库**: 确保支持 AttachTCXEgress
- **测试环境**: 需要 root 权限和真实网卡

## 测试清单

### 单元测试
- [ ] Task 2.4: TC Loader 单元测试
- [ ] Task 3.4: Policy 验证逻辑测试

### 集成测试
- [ ] Task 4.1: Egress 策略执行测试
- [ ] Task 4.2: 双向策略冲突测试
- [ ] Task 4.3: 会话状态一致性测试

### 性能测试
- [ ] Task 4.4: 延迟测试
- [ ] Task 4.4: 吞吐量测试
- [ ] Task 4.4: CPU/内存测试

### 手动测试
- [ ] TCX Egress Hook 在不同内核版本上的工作情况
- [ ] Legacy TC Egress Hook 在旧内核上的工作情况
- [ ] API 请求/响应格式验证

## 完成标准

### 功能完整性
- [ ] 所有 20 个任务完成
- [ ] 所有测试通过（单元 + 集成 + 性能）
- [ ] 代码覆盖率 >= 80%

### 质量标准
- [ ] 代码审查通过
- [ ] 无已知 bug
- [ ] 性能满足要求（延迟 < 5μs, 吞吐量下降 < 5%）
- [ ] 文档完整准确

### 向后兼容性
- [ ] 旧配置自动迁移
- [ ] API 向后兼容
- [ ] 无破坏性变更

### 安全性
- [ ] Egress 策略有效阻止未授权连接
- [ ] 无策略绕过漏洞
- [ ] 安全审计通过

---

**文档版本**: 1.0
**创建日期**: 2025-11-11
**最后更新**: 2025-11-11
**状态**: Draft
