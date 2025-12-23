[上级索引](../CLAUDE.md) > **policy**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# policy

## 架构定位

策略管理核心模块 | 输入: 高层策略规则（CIDR、标签、端口范围、协议） | 输出: eBPF Map 条目（精确匹配、通配符匹配）、策略匹配结果

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| storage.go | 策略持久化接口（SQLite） | `SavePolicy()`, `LoadPolicies()`, `DeletePolicy()` |
| policy.go | 策略定义和基础操作 | `Policy`, `PolicyRule`, `Validate()` |
| rule.go | 策略规则详细定义 | `Rule`, `RuleSpec`, `AppliesTo()` |
| compiler.go | 策略编译器，生成 eBPF Map 条目 | `CompilePolicy()`, `ExpandCIDR()`, `ExpandPortRange()` |
| compiled.go | 编译后策略表示 | `CompiledPolicy`, `MapEntry` |
| indexed_policy_manager.go | 策略管理器，支持多维度索引 | `AddPolicy()`, `RemovePolicy()`, `LookupPolicy()` |
| interface.go | 策略管理接口定义 | `PolicyManager` interface |
| errors.go | 策略相关错误定义 | `ErrPolicyNotFound`, `ErrDuplicatePolicy` |
| debug_wildcard.go | 通配符策略调试工具 | `DumpWildcardPolicies()` |

## 核心功能

- **策略编译**: CIDR/端口范围展开为精确匹配条目或通配符条目
- **优先级排序**: 按优先级（数字越小越高）匹配策略
- **方向支持**: Ingress、Egress、Bidirectional
- **通配符**: 支持 ANY IP、ANY Port
- **标签策略**: 基于工作负载标签的策略（编译为 IP 策略）
- **动态更新**: 支持策略热更新，无需重启 Agent

## 索引结构

- **By RuleID**: 快速查找特定策略
- **By SrcIP**: 查询源 IP 相关策略
- **By DstIP**: 查询目标 IP 相关策略
- **By Workload**: 查询工作负载相关策略
