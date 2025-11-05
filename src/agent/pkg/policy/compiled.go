// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"fmt"
	"time"
)

// CompiledPolicy 代表从高级 PolicyRule 编译后的低级策略
// 它扩展了基础 Policy 结构,添加了溯源信息用于追踪编译后规则回到源规则
type CompiledPolicy struct {
	// 基础策略字段 (从 Policy 嵌入)
	Policy

	// 溯源信息
	SourceRuleID     uint32    `db:"source_rule_id" json:"source_rule_id"`           // 源 PolicyRule 的 ID
	FromGroup        string    `db:"from_group" json:"from_group"`                   // 源分组名称
	ToGroup          string    `db:"to_group" json:"to_group"`                       // 目标分组名称
	FromWorkloadID   string    `db:"from_workload_id" json:"from_workload_id"`       // 源工作负载 ID
	ToWorkloadID     string    `db:"to_workload_id" json:"to_workload_id"`           // 目标工作负载 ID
	CompilationTime  time.Time `db:"compilation_time" json:"compilation_time"`       // 编译时间
	CompilerVersion  string    `db:"compiler_version" json:"compiler_version"`       // 编译器版本 (用于调试)
}

// CompiledPolicyWithRule 包含编译后策略及其源规则的完整信息
// 用于溯源查询的响应
type CompiledPolicyWithRule struct {
	CompiledPolicy *CompiledPolicy `json:"compiled_policy"`
	SourceRule     *PolicyRule     `json:"source_rule"`
}

// CompilationResult 表示策略编译操作的结果
type CompilationResult struct {
	SourceRuleID     uint32            `json:"source_rule_id"`
	CompiledPolicies []*CompiledPolicy `json:"compiled_policies"`
	CompiledCount    int               `json:"compiled_count"`
	FromGroupSize    int               `json:"from_group_size"`    // 源分组大小
	ToGroupSize      int               `json:"to_group_size"`      // 目标分组大小
	ExpansionRatio   float64           `json:"expansion_ratio"`    // 扩展比率 (编译数 / 1)
	CompilationTime  time.Duration     `json:"compilation_time"`   // 编译耗时
	Warnings         []string          `json:"warnings,omitempty"` // 警告信息
}

// CompilationSummary 提供编译统计的概览
type CompilationSummary struct {
	TotalRules       int       `json:"total_rules"`        // 总规则数
	CompiledRules    int       `json:"compiled_rules"`     // 已编译规则数
	TotalCompiled    int       `json:"total_compiled"`     // 总编译后策略数
	LastCompilation  time.Time `json:"last_compilation"`   // 最后编译时间
	AverageExpansion float64   `json:"average_expansion"`  // 平均扩展比率
}

// String 返回 CompiledPolicy 的人类可读字符串表示
func (cp *CompiledPolicy) String() string {
	return fmt.Sprintf("CompiledPolicy{RuleID=%d, Source=%d, %s:%s->%s:%d, Action=%s, FromGroup=%s->ToGroup=%s, WorkloadIDs=%s->%s}",
		cp.RuleID,
		cp.SourceRuleID,
		cp.SrcIP,
		cp.Protocol,
		cp.DstIP,
		cp.DstPort,
		cp.Action,
		cp.FromGroup,
		cp.ToGroup,
		cp.FromWorkloadID,
		cp.ToWorkloadID,
	)
}

// Validate 验证 CompiledPolicy 的有效性
func (cp *CompiledPolicy) Validate() error {
	// 验证基础 Policy 字段
	if cp.RuleID == 0 {
		return fmt.Errorf("rule_id is required")
	}
	if cp.SrcIP == "" {
		return fmt.Errorf("src_ip is required")
	}
	if cp.DstIP == "" {
		return fmt.Errorf("dst_ip is required")
	}
	if cp.Protocol == "" {
		return fmt.Errorf("protocol is required")
	}
	if cp.Action == "" {
		return fmt.Errorf("action is required")
	}

	// 验证溯源字段
	if cp.SourceRuleID == 0 {
		return fmt.Errorf("source_rule_id is required")
	}

	if cp.FromGroup == "" {
		return fmt.Errorf("from_group is required")
	}

	if cp.ToGroup == "" {
		return fmt.Errorf("to_group is required")
	}

	if cp.FromWorkloadID == "" {
		return fmt.Errorf("from_workload_id is required")
	}

	if cp.ToWorkloadID == "" {
		return fmt.Errorf("to_workload_id is required")
	}

	if cp.CompilationTime.IsZero() {
		return fmt.Errorf("compilation_time is required")
	}

	return nil
}

// AddWarning 添加警告信息到编译结果
func (cr *CompilationResult) AddWarning(format string, args ...interface{}) {
	warning := fmt.Sprintf(format, args...)
	cr.Warnings = append(cr.Warnings, warning)
}

// HasWarnings 检查是否有警告
func (cr *CompilationResult) HasWarnings() bool {
	return len(cr.Warnings) > 0
}

// CalculateExpansion 计算扩展比率
func (cr *CompilationResult) CalculateExpansion() {
	if cr.CompiledCount > 0 {
		cr.ExpansionRatio = float64(cr.CompiledCount)
	} else {
		cr.ExpansionRatio = 0
	}
}

// String 返回 CompilationResult 的摘要字符串
func (cr *CompilationResult) String() string {
	return fmt.Sprintf("CompilationResult{SourceRule=%d, Compiled=%d, Expansion=%dx%d=%d (%.1fx), Time=%v, Warnings=%d}",
		cr.SourceRuleID,
		cr.CompiledCount,
		cr.FromGroupSize,
		cr.ToGroupSize,
		cr.CompiledCount,
		cr.ExpansionRatio,
		cr.CompilationTime,
		len(cr.Warnings),
	)
}

// 编译器版本常量
const (
	CompilerVersionV1 = "v1.0.0"
)

// 编译警告阈值
const (
	// 警告阈值: 编译后策略数量超过此值时发出警告
	CompilationWarningThreshold = 1000

	// 严重警告阈值: 超过此值时强烈警告性能问题
	CompilationCriticalThreshold = 10000
)
