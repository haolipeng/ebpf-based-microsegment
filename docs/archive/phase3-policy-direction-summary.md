# Phase 3: 策略管理扩展 - Direction 支持完整总结

**完成日期**: 2025-11-12
**OpenSpec Change ID**: add-tc-egress-support
**状态**: ✅ 已完成

---

## 📋 概述

Phase 3 为策略管理系统添加了完整的方向感知能力,使得系统能够区分 ingress 和 egress 流量的策略控制。这是实现双向流量控制(TC Egress Hook)的关键组件。

## ✅ 完成的任务

### Task 3.1: Policy 结构添加 Direction 字段

**文件修改**: `src/agent/pkg/policy/policy.go`

#### 1. 添加 Direction 常量
```go
const (
    DirectionAny     = "any"      // 匹配双向流量
    DirectionIngress = "ingress"  // 只匹配入向流量
    DirectionEgress  = "egress"   // 只匹配出向流量
)
```

#### 2. 扩展 Policy 结构
```go
type Policy struct {
    RuleID    uint32
    SrcIP     string
    DstIP     string
    SrcPort   uint16
    DstPort   uint16
    Protocol  string
    Action    string
    Direction string   // ✅ 新增: 方向字段
    Priority  uint16
}
```

#### 3. Direction 辅助方法

**GetDirectionValue()** - 字符串到 eBPF 数值转换:
```go
func (p *Policy) GetDirectionValue() uint8 {
    switch strings.ToLower(p.Direction) {
    case DirectionIngress:
        return 1 // POLICY_DIR_INGRESS
    case DirectionEgress:
        return 2 // POLICY_DIR_EGRESS
    default:
        return 0 // POLICY_DIR_ANY
    }
}
```

**NormalizeDirection()** - 规范化 direction 字段:
```go
func (p *Policy) NormalizeDirection() {
    p.Direction = strings.ToLower(p.Direction)
    if p.Direction != DirectionIngress && p.Direction != DirectionEgress {
        p.Direction = DirectionAny
    }
}
```

**Validate()** - 完整的策略验证:
```go
func (p *Policy) Validate() error {
    // 验证 direction
    if p.Direction != "" &&
       p.Direction != DirectionAny &&
       p.Direction != DirectionIngress &&
       p.Direction != DirectionEgress {
        return fmt.Errorf("invalid direction: %s", p.Direction)
    }
    // 验证 protocol, action, IPs...
    return nil
}
```

**directionToString()** - eBPF 数值到字符串转换:
```go
func directionToString(direction uint8) string {
    switch direction {
    case 1:
        return DirectionIngress
    case 2:
        return DirectionEgress
    default:
        return DirectionAny
    }
}
```

---

### Task 3.2: PolicyManager 支持方向感知策略

**文件修改**: `src/agent/pkg/policy/policy.go`

#### 1. AddPolicy - 自动验证和规范化
```go
func (pm *PolicyManager) AddPolicy(p *Policy) error {
    // ✅ 新增: 验证并规范化 direction
    if err := p.Validate(); err != nil {
        return fmt.Errorf("policy validation failed: %w", err)
    }
    p.NormalizeDirection()

    // 添加到 eBPF map (包含 direction)
    if err := pm.addPolicyToMap(p); err != nil {
        return err
    }

    // 持久化存储...
    return nil
}
```

#### 2. addExactPolicy - 精确匹配策略 (6-tuple)
```go
func (pm *PolicyManager) addExactPolicy(p *Policy) error {
    // eBPF policy key (6-tuple: 5-tuple + direction)
    key := struct {
        SrcIp     uint32
        DstIp     uint32
        SrcPort   uint16
        DstPort   uint16
        Protocol  uint8
        Direction uint8  // ✅ 新增
        Pad       uint16
    }{
        SrcIp:     ipToUint32(srcIP),
        DstIp:     ipToUint32(dstIP),
        SrcPort:   htons(p.SrcPort),
        DstPort:   htons(p.DstPort),
        Protocol:  proto,
        Direction: p.GetDirectionValue(), // ✅ 新增
        Pad:       0,
    }

    // 插入 eBPF policy_map
    if err := pm.policyMap.Put(&key, &value); err != nil {
        return fmt.Errorf("failed to add policy to map: %w", err)
    }

    log.Infof("Policy added: rule_id=%d %s:%d -> %s:%d proto=%s action=%s dir=%s",
        p.RuleID, p.SrcIP, p.SrcPort, p.DstIP, p.DstPort, p.Protocol, p.Action, p.Direction)

    return nil
}
```

#### 3. addWildcardPolicy - 通配符策略
```go
func (pm *PolicyManager) addWildcardPolicy(p *Policy) error {
    wildcard := struct {
        SrcIP      uint32
        SrcIPMask  uint32
        DstIP      uint32
        DstIPMask  uint32
        SrcPort    uint16
        DstPort    uint16
        Protocol   uint8
        Direction  uint8  // ✅ 新增
        Action     uint8
        LogEnabled uint8
        Priority   uint16
        Pad        uint16
        RuleID     uint32
    }{
        // ... 其他字段
        Direction: p.GetDirectionValue(), // ✅ 新增
        // ...
    }

    // 插入 wildcard_policy_map
    if err := pm.wildcardPolicyMap.Put(&i, &wildcard); err != nil {
        return fmt.Errorf("failed to add wildcard policy: %w", err)
    }

    log.Infof("Wildcard policy added: rule_id=%d ... dir=%s (priority=%d)",
        p.RuleID, ..., p.Direction, p.Priority)

    return nil
}
```

#### 4. DeletePolicy - 删除时需要完整的 6-tuple
```go
func (pm *PolicyManager) DeletePolicy(p *Policy) error {
    // 构造 policy key (包含 direction)
    key := struct {
        SrcIp     uint32
        DstIp     uint32
        SrcPort   uint16
        DstPort   uint16
        Protocol  uint8
        Direction uint8  // ✅ 必需: 唯一确定策略
        Pad       uint16
    }{
        // ... 字段赋值
        Direction: p.GetDirectionValue(),
        // ...
    }

    // 从 eBPF map 删除
    if err := pm.policyMap.Delete(&key); err != nil {
        return fmt.Errorf("failed to delete policy: %w", err)
    }

    log.Infof("Policy deleted: rule_id=%d ... dir=%s", p.RuleID, p.Direction)
    return nil
}
```

#### 5. ListPolicies - 返回 direction
```go
func (pm *PolicyManager) ListPolicies() ([]Policy, error) {
    var key struct {
        SrcIp     uint32
        DstIp     uint32
        SrcPort   uint16
        DstPort   uint16
        Protocol  uint8
        Direction uint8  // ✅ 读取 direction
        Pad       uint16
    }

    // 遍历 eBPF map
    iter := pm.policyMap.Iterate()
    for iter.Next(&key, &value) {
        policy := Policy{
            // ... 其他字段
            Direction: directionToString(key.Direction), // ✅ 转换回字符串
            // ...
        }
        policies = append(policies, policy)
    }

    return policies, nil
}
```

---

### Task 3.3: API 扩展 (Policy CRUD 支持 Direction)

#### 1. 更新 API 模型

**文件修改**: `src/agent/pkg/api/models/policy.go`

**PolicyRequest** - 请求模型:
```go
type PolicyRequest struct {
    RuleID    uint32 `json:"rule_id" binding:"required"`
    SrcIP     string `json:"src_ip" binding:"required"`
    DstIP     string `json:"dst_ip" binding:"required"`
    SrcPort   uint16 `json:"src_port"`
    DstPort   uint16 `json:"dst_port"`
    Protocol  string `json:"protocol" binding:"required,oneof=tcp udp icmp any"`
    Action    string `json:"action" binding:"required,oneof=allow deny log"`
    Direction string `json:"direction" binding:"omitempty,oneof=any ingress egress"` // ✅ 新增
    Priority  uint16 `json:"priority"`
}
```

**PolicyResponse** - 响应模型:
```go
type PolicyResponse struct {
    RuleID    uint32 `json:"rule_id"`
    SrcIP     string `json:"src_ip"`
    DstIP     string `json:"dst_ip"`
    SrcPort   uint16 `json:"src_port"`
    DstPort   uint16 `json:"dst_port"`
    Protocol  string `json:"protocol"`
    Action    string `json:"action"`
    Direction string `json:"direction"` // ✅ 新增
    Priority  uint16 `json:"priority"`
}
```

#### 2. 更新 API 处理函数

**文件修改**: `src/agent/pkg/api/handlers/policy.go`

**CreatePolicy** - POST /api/v1/policies:
```go
func (h *PolicyHandler) CreatePolicy(c *gin.Context) {
    var req models.PolicyRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ...)
        return
    }

    // 转换为内部 Policy 格式
    p := &policy.Policy{
        RuleID:    req.RuleID,
        SrcIP:     req.SrcIP,
        DstIP:     req.DstIP,
        SrcPort:   req.SrcPort,
        DstPort:   req.DstPort,
        Protocol:  req.Protocol,
        Action:    req.Action,
        Direction: req.Direction, // ✅ 新增
        Priority:  req.Priority,
    }

    // 添加策略 (自动验证和规范化)
    if err := h.policyManager.AddPolicy(p); err != nil {
        c.JSON(http.StatusInternalServerError, ...)
        return
    }

    // 返回创建的策略
    response := models.PolicyResponse{
        // ... 所有字段
        Direction: p.Direction, // ✅ 包含 direction
        // ...
    }

    c.JSON(http.StatusCreated, response)
}
```

**ListPolicies** - GET /api/v1/policies:
```go
func (h *PolicyHandler) ListPolicies(c *gin.Context) {
    policies, err := h.policyManager.ListPolicies()
    if err != nil {
        c.JSON(http.StatusInternalServerError, ...)
        return
    }

    var policyResponses []models.PolicyResponse
    for _, p := range policies {
        policyResponses = append(policyResponses, models.PolicyResponse{
            // ... 所有字段
            Direction: p.Direction, // ✅ 包含 direction
            // ...
        })
    }

    c.JSON(http.StatusOK, models.PolicyListResponse{
        Policies: policyResponses,
        Count:    len(policyResponses),
    })
}
```

**GetPolicy** - GET /api/v1/policies/:id:
```go
func (h *PolicyHandler) GetPolicy(c *gin.Context) {
    // ... 获取 rule ID

    // 查找匹配的策略
    for _, p := range policies {
        if p.RuleID == uint32(ruleID) {
            response := models.PolicyResponse{
                // ... 所有字段
                Direction: p.Direction, // ✅ 包含 direction
                // ...
            }
            c.JSON(http.StatusOK, response)
            return
        }
    }

    c.JSON(http.StatusNotFound, ...)
}
```

**UpdatePolicy** - PUT /api/v1/policies/:id:
```go
func (h *PolicyHandler) UpdatePolicy(c *gin.Context) {
    // ... 验证 rule ID

    var req models.PolicyRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, ...)
        return
    }

    // 转换为内部格式
    p := &policy.Policy{
        // ... 所有字段
        Direction: req.Direction, // ✅ 新增
        // ...
    }

    // 删除旧策略 (包含 direction 的完整 key)
    if err := h.policyManager.DeletePolicy(p); err != nil {
        log.Warnf("Failed to delete old policy: %v", err)
    }

    // 添加更新后的策略
    if err := h.policyManager.AddPolicy(p); err != nil {
        c.JSON(http.StatusInternalServerError, ...)
        return
    }

    // 返回更新后的策略
    response := models.PolicyResponse{
        // ... 所有字段
        Direction: p.Direction, // ✅ 包含 direction
        // ...
    }

    c.JSON(http.StatusOK, response)
}
```

**DeletePolicy** - DELETE /api/v1/policies/:id:
```go
func (h *PolicyHandler) DeletePolicy(c *gin.Context) {
    // ... 获取 rule ID

    // 查找要删除的策略
    var policyToDelete *policy.Policy
    for i := range policies {
        if policies[i].RuleID == uint32(ruleID) {
            policyToDelete = &policies[i] // 包含完整的 direction
            break
        }
    }

    if policyToDelete == nil {
        c.JSON(http.StatusNotFound, ...)
        return
    }

    // 删除策略 (使用完整的 6-tuple)
    if err := h.policyManager.DeletePolicy(policyToDelete); err != nil {
        c.JSON(http.StatusInternalServerError, ...)
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": fmt.Sprintf("Policy with rule ID %d deleted successfully", ruleID),
    })
}
```

---

### Task 3.4: 策略验证逻辑更新

**实现在 Task 3.1 中完成**

#### Validate() 方法验证所有字段
```go
func (p *Policy) Validate() error {
    // 1. 验证 direction
    if p.Direction != "" &&
       p.Direction != DirectionAny &&
       p.Direction != DirectionIngress &&
       p.Direction != DirectionEgress {
        return fmt.Errorf("invalid direction: %s (must be 'any', 'ingress', or 'egress')", p.Direction)
    }

    // 2. 验证 protocol
    validProtocols := map[string]bool{
        "tcp": true, "udp": true, "icmp": true, "any": true, "": true,
    }
    if !validProtocols[strings.ToLower(p.Protocol)] {
        return fmt.Errorf("invalid protocol: %s", p.Protocol)
    }

    // 3. 验证 action
    validActions := map[string]bool{
        "allow": true, "deny": true, "log": true,
    }
    if !validActions[strings.ToLower(p.Action)] {
        return fmt.Errorf("invalid action: %s", p.Action)
    }

    // 4. 验证 IPs
    if p.SrcIP != "" {
        if _, _, err := parseCIDR(p.SrcIP); err != nil {
            return fmt.Errorf("invalid source IP: %w", err)
        }
    }
    if p.DstIP != "" {
        if _, _, err := parseCIDR(p.DstIP); err != nil {
            return fmt.Errorf("invalid destination IP: %w", err)
        }
    }

    return nil
}
```

#### 集成到 AddPolicy
```go
func (pm *PolicyManager) AddPolicy(p *Policy) error {
    // 自动调用验证
    if err := p.Validate(); err != nil {
        return fmt.Errorf("policy validation failed: %w", err)
    }

    // 自动规范化 direction
    p.NormalizeDirection()

    // ... 添加到 eBPF map
}
```

---

## 🔄 向后兼容性

### 1. API 兼容性
- ✅ Direction 字段为**可选** (`binding:"omitempty"`)
- ✅ 不传 direction 时默认为 "any"
- ✅ 旧的 API 请求仍然有效

### 2. 策略兼容性
- ✅ 空 direction 自动规范化为 "any"
- ✅ direction="any" 匹配所有方向 (ingress + egress)
- ✅ 旧的 5-tuple 策略继续工作

### 3. 存储兼容性
- ✅ 旧策略从存储加载时,direction 为空字符串
- ✅ 自动规范化为 "any"
- ✅ 不需要数据迁移

---

## 📊 API 使用示例

### 1. 创建 Ingress 策略
```bash
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "rule_id": 100,
    "src_ip": "192.168.1.0/24",
    "dst_ip": "10.0.0.5/32",
    "src_port": 0,
    "dst_port": 22,
    "protocol": "tcp",
    "action": "allow",
    "direction": "ingress",
    "priority": 100
  }'
```

**响应**:
```json
{
  "rule_id": 100,
  "src_ip": "192.168.1.0/24",
  "dst_ip": "10.0.0.5/32",
  "src_port": 0,
  "dst_port": 22,
  "protocol": "tcp",
  "action": "allow",
  "direction": "ingress",
  "priority": 100
}
```

### 2. 创建 Egress 策略
```bash
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "rule_id": 101,
    "src_ip": "10.0.0.10/32",
    "dst_ip": "8.8.8.8/32",
    "src_port": 0,
    "dst_port": 53,
    "protocol": "udp",
    "action": "allow",
    "direction": "egress",
    "priority": 100
  }'
```

### 3. 创建双向策略 (默认)
```bash
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "rule_id": 102,
    "src_ip": "0.0.0.0/0",
    "dst_ip": "10.0.0.0/8",
    "src_port": 0,
    "dst_port": 80,
    "protocol": "tcp",
    "action": "allow"
  }'
```
**注意**: 不传 direction 时默认为 "any" (双向)

### 4. 列出所有策略
```bash
curl http://localhost:8080/api/v1/policies
```

**响应**:
```json
{
  "policies": [
    {
      "rule_id": 100,
      "src_ip": "192.168.1.0/24",
      "dst_ip": "10.0.0.5/32",
      "src_port": 0,
      "dst_port": 22,
      "protocol": "tcp",
      "action": "allow",
      "direction": "ingress",
      "priority": 100
    },
    {
      "rule_id": 101,
      "src_ip": "10.0.0.10/32",
      "dst_ip": "8.8.8.8/32",
      "src_port": 0,
      "dst_port": 53,
      "protocol": "udp",
      "action": "allow",
      "direction": "egress",
      "priority": 100
    },
    {
      "rule_id": 102,
      "src_ip": "0.0.0.0/0",
      "dst_ip": "10.0.0.0/8",
      "src_port": 0,
      "dst_port": 80,
      "protocol": "tcp",
      "action": "allow",
      "direction": "any",
      "priority": 0
    }
  ],
  "count": 3
}
```

---

## 🧪 测试验证

### 单元测试

**测试文件**: `src/agent/pkg/policy/policy_direction_test.go`

```bash
go test -v ./src/agent/pkg/policy -run TestPolicy.*Direction
```

**测试覆盖**:
- ✅ Direction 常量定义
- ✅ GetDirectionValue 转换 (支持大小写)
- ✅ NormalizeDirection 规范化
- ✅ Validate 验证 (有效/无效 direction)
- ✅ directionToString 反向转换
- ✅ 往返转换 (string → uint8 → string)

### 独立验证程序

**测试文件**: `src/agent/pkg/policy/test_direction_standalone.go`

```bash
go run ./src/agent/pkg/policy/test_direction_standalone.go
```

**输出**:
```
=== Policy Direction 字段测试 ===

测试 1: Direction 常量定义
  DirectionAny     = 'any'
  DirectionIngress = 'ingress'
  DirectionEgress  = 'egress'

测试 2: GetDirectionValue 转换
  ✓ 'any' -> 0 (expected 0)
  ✓ 'ingress' -> 1 (expected 1)
  ✓ 'egress' -> 2 (expected 2)
  ✓ 'INGRESS' -> 1 (expected 1)
  ✓ 'Egress' -> 2 (expected 2)
  ✓ '' -> 0 (expected 0)
  ✓ 'invalid' -> 0 (expected 0)

测试 3: NormalizeDirection 规范化
  ✓ 'ANY' -> 'any' (expected 'any')
  ✓ 'INGRESS' -> 'ingress' (expected 'ingress')
  ✓ 'Egress' -> 'egress' (expected 'egress')
  ✓ '' -> 'any' (expected 'any')
  ✓ 'invalid' -> 'any' (expected 'any')

测试 4: Validate 验证
  ✓ Valid ingress policy passed validation
  ✓ Invalid direction rejected: invalid direction: invalid_direction

测试 5: 完整的策略结构
  RuleID:    100
  SrcIP:     192.168.1.100/32
  DstIP:     10.0.0.10/32
  SrcPort:   0
  DstPort:   80
  Protocol:  tcp
  Action:    deny
  Direction: egress (value=2)
  Priority:  200

=== 测试结果 ===
✓ 所有测试通过!
```

### 编译验证
```bash
# 编译 policy 包
go build -v ./src/agent/pkg/policy/...
# 输出: github.com/ebpf-microsegment/src/agent/pkg/policy

# 编译 API 包
go build -v ./src/agent/pkg/api/...
# 输出: github.com/ebpf-microsegment/src/agent/pkg/api/...
```

---

## 📁 修改的文件清单

### 核心策略管理
1. **src/agent/pkg/policy/policy.go** (核心修改)
   - 添加 Direction 常量
   - 扩展 Policy 结构
   - 添加 Direction 辅助方法
   - 更新 AddPolicy, DeletePolicy, ListPolicies
   - 更新 addExactPolicy, addWildcardPolicy

### API 层
2. **src/agent/pkg/api/models/policy.go** (API 模型)
   - 更新 PolicyRequest (添加 direction 验证)
   - 更新 PolicyResponse (添加 direction 字段)

3. **src/agent/pkg/api/handlers/policy.go** (API 处理函数)
   - 更新 CreatePolicy (支持 direction)
   - 更新 ListPolicies (返回 direction)
   - 更新 GetPolicy (返回 direction)
   - 更新 UpdatePolicy (支持 direction)
   - 更新 DeletePolicy (使用完整 6-tuple)

### 测试文件
4. **src/agent/pkg/policy/policy_direction_test.go** (新建)
   - Direction 相关单元测试

5. **src/agent/pkg/policy/test_direction_standalone.go** (新建)
   - 独立验证程序

### 文档
6. **docs/phase3-policy-direction-summary.md** (新建)
   - 本文档

---

## 🎯 验收标准

### Task 3.1
- ✅ Policy 结构包含 Direction 字段
- ✅ Direction 验证逻辑正确
- ✅ 转换函数返回正确的 eBPF 值
- ✅ 向后兼容(Direction 为空时默认 "any")

### Task 3.2
- ✅ AddPolicy 正确处理 direction
- ✅ DeletePolicy 需要 direction 参数
- ✅ ListPolicies 返回 direction
- ✅ 向后兼容旧配置

### Task 3.3
- ✅ API 请求/响应包含 direction 字段
- ✅ Direction 验证正确 (any/ingress/egress)
- ✅ 所有 CRUD 操作支持 direction
- ✅ API 文档更新

### Task 3.4
- ✅ Validate() 方法完整验证所有字段
- ✅ AddPolicy 自动调用验证
- ✅ 验证错误返回清晰消息
- ✅ Direction 为空时自动规范化

---

## 🔗 关联 Phase

### 已完成的 Phase
- ✅ **Phase 1**: eBPF 程序扩展 (方向检测, 策略匹配, 统计)
- ✅ **Phase 2**: TC Loader 扩展 (双向 Hook 支持)
- ✅ **Phase 3**: 策略管理扩展 (Policy + API 支持 Direction)

### 待完成的 Phase
- ⏳ **Phase 4**: 集成测试 (Egress 策略执行, 双向策略冲突, 性能测试)
- ⏳ **Phase 5**: 文档与示例 (架构文档, 配置示例, 最佳实践)

---

## 📈 技术亮点

### 1. 完整的方向感知
- 从 eBPF 内核到用户空间 API 的完整支持
- 6-tuple 匹配 (5-tuple + direction)
- 两阶段策略查找 (方向特定 → ANY 回退)

### 2. 优秀的向后兼容性
- 可选字段,不破坏现有 API
- 自动规范化和默认值处理
- 无需数据迁移

### 3. 健壮的验证
- 多层验证(API 层 + PolicyManager 层)
- 清晰的错误消息
- 自动规范化(大小写,空值)

### 4. 完整的测试覆盖
- 单元测试
- 独立验证程序
- 编译验证

---

## 💡 使用建议

### 1. Ingress 策略
用于控制外部流量进入:
```json
{
  "direction": "ingress",
  "src_ip": "0.0.0.0/0",
  "dst_ip": "10.0.0.5/32",
  "dst_port": 22,
  "protocol": "tcp",
  "action": "allow"
}
```

### 2. Egress 策略
用于控制内部流量外出:
```json
{
  "direction": "egress",
  "src_ip": "10.0.0.10/32",
  "dst_ip": "8.8.8.8/32",
  "dst_port": 53,
  "protocol": "udp",
  "action": "allow"
}
```

### 3. 双向策略
用于简化配置:
```json
{
  "direction": "any",
  "src_ip": "10.0.0.0/8",
  "dst_ip": "10.0.0.0/8",
  "protocol": "tcp",
  "action": "allow"
}
```

### 4. 策略优先级
- 方向特定策略优先于 ANY 策略
- 相同 5-tuple 可以有不同的 ingress/egress 策略
- 独立控制双向流量

---

**完成标识**: ✅ Phase 3 完成,所有任务验收标准已满足
**下一步**: Phase 4 - 集成测试
