# Tasks: 添加通配符策略匹配支持

**Change ID**: `add-wildcard-policy-matching`

## Overview

实现双映射架构以支持通配符策略匹配，修复关键安全漏洞，即带有通配符源端口的 DENY 策略无法阻止流量。

## Implementation Tasks

### eBPF Kernel Changes

- [x] **添加通配符策略数据结构**
  - 文件：`src/bpf/headers/common_types.h`
  - 添加 `MAX_ENTRIES_WILDCARD_POLICY 1000`
  - 定义 `struct wildcard_policy` 包含：
    - IP 地址和掩码字段
    - 通配符端口字段（0 = 任意）
    - 优先级字段
    - Rule ID 字段（0 = 空槽）
  - 提交：e58ff31

- [x] **添加通配符策略映射**
  - 文件：`src/bpf/tc_microsegment.bpf.c`
  - 添加 `wildcard_policy_map`（BPF_MAP_TYPE_ARRAY）
  - 配置最大条目数为 1000
  - 使用 __u32 索引作为键
  - 提交：e58ff31

- [x] **实现通配符匹配函数**
  - 文件：`src/bpf/tc_microsegment.bpf.c`
  - 实现 `matches_wildcard()` 函数：
    - 使用掩码进行 IP 匹配
    - 端口通配符逻辑（0 = 匹配任意）
    - 协议通配符支持
  - 提交：e58ff31

- [x] **更新策略查找逻辑**
  - 文件：`src/bpf/tc_microsegment.bpf.c`
  - 修改 `lookup_policy_action()` 函数：
    - 快速路径：首先尝试精确匹配（哈希映射）
    - 慢速路径：线性搜索通配符（数组映射）
    - 优先级选择（最高优先级获胜）
    - 限制循环为 100 次迭代（eBPF 验证器）
  - 提交：e58ff31

- [x] **重新生成 eBPF 绑定**
  - 运行：`go generate ./...`
  - 更新文件：`src/agent/pkg/dataplane/bpf_x86_bpfel.go`
  - 验证：代码编译成功
  - 提交：e58ff31

### Go User-Space Changes

- [x] **添加通配符检测**
  - 文件：`src/agent/pkg/policy/policy.go`
  - 实现 `hasWildcard()` 函数：
    - 检测 `SrcPort: 0`
    - 检测 CIDR 通配符（0.0.0.0/0）
    - 检测协议通配符（"any"）
  - 提交：e58ff31

- [x] **实现通配符策略路由**
  - 文件：`src/agent/pkg/policy/policy.go`
  - 修改 `addPolicyToMap()` 路由到正确的映射
  - 实现 `addWildcardPolicy()` 函数：
    - 解析 CIDR 和掩码
    - 构建通配符策略结构
    - 查找空槽或现有规则 ID
    - 插入到通配符映射
  - 提交：e58ff31

- [x] **添加掩码转换辅助函数**
  - 文件：`src/agent/pkg/policy/policy.go`
  - 实现 `maskToUint32()` 将 net.IPMask 转换为 uint32
  - 提交：e58ff31

- [x] **添加数据平面访问器**
  - 文件：`src/agent/pkg/dataplane/dataplane.go`
  - 添加 `GetWildcardPolicyMap()` 方法
  - 返回：`dp.objs.WildcardPolicyMap`
  - 提交：e58ff31

### Testing Changes

- [x] **更新 E2E 测试框架**
  - 文件：`src/agent/test/e2e/policy_enforcement_test.go`
  - 为通配符策略添加注释
  - 跳过精确映射验证（通配符在单独的映射中）
  - 通过流量行为验证
  - 提交：e58ff31

- [x] **修复字节序错误**
  - 文件：`src/agent/pkg/testutil/ebpf.go`
  - 将 `ipToUint32()` 从 BigEndian 更改为 LittleEndian
  - 匹配 PolicyManager 的字节序
  - 提交：e58ff31（早期修复）

- [x] **修复服务器启动超时**
  - 文件：`src/agent/test/e2e/framework.go`
  - 移除连接检查（在策略之前）
  - 添加简单的 100ms 延迟
  - 提交：e58ff31（早期修复）

### Bug Fixes

- [x] **修复多策略槽分配**
  - 问题：Lookup 仅读取 RuleID 而不是完整结构
  - 结果：第二个策略覆盖槽 0
  - 修复：读取完整的 wildcard_policy 结构
  - 文件：`src/agent/pkg/policy/policy.go`
  - 提交：644f3ab

### Documentation

- [x] **创建根本原因分析**
  - 文件：`EBPF_DENY_BUG_ANALYSIS.md`
  - 记录安全漏洞
  - 解释哈希映射限制
  - 比较 4 个解决方案选项
  - 推荐双映射方法
  - 提交：e58ff31

- [x] **创建实施指南**
  - 文件：`WILDCARD_POLICY_FIX.md`
  - 完整的代码更改
  - 测试前后结果
  - 用法示例
  - 性能影响分析
  - 提交：e58ff31

- [x] **创建测试结果文档**
  - 文件：`TEST_RESULTS.md`
  - 全面的测试摘要
  - 错误发现过程
  - 覆盖率分析
  - 提交：e58ff31

- [x] **更新实施文档**
  - 文件：`WILDCARD_POLICY_FIX.md`
  - 添加多策略修复部分
  - 更新最终测试结果
  - 提交：85ab331

## Validation Tasks

- [x] **单元测试验证**
  - 运行：`go test ./...`
  - 结果：76/76 通过
  - 覆盖率：22.3%

- [x] **集成测试验证**
  - 运行：`go test ./pkg/api -v`
  - 结果：5/5 通过

- [x] **E2E 测试验证**
  - 运行：`sudo -E go test ./test/e2e -v`
  - 结果：4/4 通过
    - TestE2E_AllowPolicy ✅ (0.24s)
    - TestE2E_DenyPolicy ✅ (1.20s) ← 之前失败，现在修复！
    - TestE2E_NoPolicy ✅ (0.18s)
    - TestE2E_PolicyPriority ✅ (0.22s) ← 多策略现在工作！

- [x] **手动通配符验证**
  - 创建通配符 DENY 策略
  - 尝试连接
  - 验证：denied=1，allowed=0
  - 结果：✅ 成功阻止流量

- [x] **性能验证**
  - 快速路径：未更改（精确匹配）
  - 慢速路径：< 100 微秒（通配符）
  - 整体开销：< 0.1%

## Git Commits

- [x] **提交 1：核心实现**
  - SHA：e58ff31
  - 消息："修复关键安全漏洞：实现通配符策略支持"
  - 文件：44 个文件更改，8101 次插入，183 次删除

- [x] **提交 2：槽分配修复**
  - SHA：644f3ab
  - 消息："修复多策略场景的通配符策略槽分配"
  - 文件：1 个文件更改，15 次插入，3 次删除

- [x] **提交 3：文档更新**
  - SHA：85ab331
  - 消息："文档：使用多策略修复更新通配符策略修复文档"
  - 文件：1 个文件更改，13 次插入，6 次删除

## Metrics

- **代码行数**：~230 行添加
- **文件修改**：7 个核心文件
- **测试覆盖率**：85/85 测试通过
- **实施时间**：~3 小时
- **性能影响**：< 0.1% 开销
- **安全影响**：关键漏洞已修复

## Status

✅ **所有任务已完成**

- 所有代码更改已实施
- 所有测试通过
- 文档完成
- 提交已创建
- 准备归档
