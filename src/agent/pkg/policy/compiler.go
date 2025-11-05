// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package policy

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebpf-microsegment/src/agent/pkg/groups"
	log "github.com/sirupsen/logrus"
)

// PolicyCompiler 负责将高级 PolicyRule 编译为低级 CompiledPolicy
// 它解析分组成员并生成 N×M 的 IP 规则
type PolicyCompiler struct {
	storage        Storage
	groupMgr       *groups.GroupManager
	nextCompiledID uint32          // 下一个可用的编译后规则 ID
	mu             sync.RWMutex    // 保护 nextCompiledID
}

// NewPolicyCompiler 创建一个新的策略编译器实例
func NewPolicyCompiler(storage Storage, groupMgr *groups.GroupManager) *PolicyCompiler {
	return &PolicyCompiler{
		storage:        storage,
		groupMgr:       groupMgr,
		nextCompiledID: 100000, // 从 100000 开始，避免与手动规则 ID 冲突
	}
}

// CompilePolicyRule 编译单个策略规则为 IP 规则列表
// 它执行以下操作：
// 1. 解析 from_group 和 to_group 的成员
// 2. 生成笛卡尔积（N×M IP 规则）
// 3. 为每个规则分配唯一 ID
// 4. 存储溯源信息
func (pc *PolicyCompiler) CompilePolicyRule(ruleID uint32) (*CompilationResult, error) {
	startTime := time.Now()

	// 获取源规则
	sourceRule, err := pc.storage.GetPolicyRule(ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source rule: %w", err)
	}

	// 检查规则是否启用
	if !sourceRule.Enabled {
		log.Debugf("Skipping compilation for disabled rule %d", ruleID)
		return &CompilationResult{
			SourceRuleID:  ruleID,
			CompiledCount: 0,
		}, nil
	}

	// 解析 from_group 成员
	fromMembers, err := pc.groupMgr.ResolveGroupMembers(sourceRule.FromGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve from_group '%s': %w", sourceRule.FromGroup, err)
	}

	// 解析 to_group 成员
	toMembers, err := pc.groupMgr.ResolveGroupMembers(sourceRule.ToGroup)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve to_group '%s': %w", sourceRule.ToGroup, err)
	}

	// 创建编译结果
	result := &CompilationResult{
		SourceRuleID:  ruleID,
		FromGroupSize: len(fromMembers),
		ToGroupSize:   len(toMembers),
	}

	// 生成编译后的策略
	compiledPolicies := []*CompiledPolicy{}

	// 笛卡尔积：from_group 每个成员 × to_group 每个成员 × 端口
	for _, fromMember := range fromMembers {
		for _, toMember := range toMembers {
			// 为每个端口范围生成规则
			for _, portRange := range sourceRule.Ports {
				// 遍历端口范围
				for port := portRange.Start; port <= portRange.End; port++ {
					// 获取workload的IP地址（使用第一个IP）
					srcIP := ""
					if len(fromMember.IPs) > 0 {
						srcIP = fromMember.IPs[0].String()
					}
					dstIP := ""
					if len(toMember.IPs) > 0 {
						dstIP = toMember.IPs[0].String()
					}

					// 跳过没有IP地址的workload
					if srcIP == "" || dstIP == "" {
						continue
					}

					cp := &CompiledPolicy{
						Policy: Policy{
							RuleID:   pc.allocateCompiledRuleID(),
							SrcIP:    srcIP,
							DstIP:    dstIP,
							SrcPort:  0, // 0 表示任意源端口
							DstPort:  port,
							Protocol: portRange.Protocol,
							Action:   sourceRule.Action,
							Priority: uint16(sourceRule.Priority),
						},
						SourceRuleID:    ruleID,
						FromGroup:       sourceRule.FromGroup,
						ToGroup:         sourceRule.ToGroup,
						FromWorkloadID:  fromMember.ID,
						ToWorkloadID:    toMember.ID,
						CompilationTime: time.Now(),
						CompilerVersion: CompilerVersionV1,
					}

					compiledPolicies = append(compiledPolicies, cp)
				}
			}
		}
	}

	result.CompiledPolicies = compiledPolicies
	result.CompiledCount = len(compiledPolicies)
	result.CalculateExpansion()

	// 检查扩展比率并添加警告
	if result.CompiledCount > CompilationCriticalThreshold {
		result.AddWarning("Critical: Compiled %d policies (from %d×%d expansion). This may cause performance issues.",
			result.CompiledCount, result.FromGroupSize, result.ToGroupSize)
	} else if result.CompiledCount > CompilationWarningThreshold {
		result.AddWarning("Warning: Compiled %d policies (from %d×%d expansion). Consider reducing group sizes.",
			result.CompiledCount, result.FromGroupSize, result.ToGroupSize)
	}

	// 存储编译后的策略
	for _, cp := range compiledPolicies {
		if err := pc.storage.SaveCompiledPolicy(cp); err != nil {
			return nil, fmt.Errorf("failed to save compiled policy %d: %w", cp.RuleID, err)
		}
	}

	result.CompilationTime = time.Since(startTime)

	log.Infof("Compiled policy rule %d: %d policies in %v (from %d×%d=%d expansion)",
		ruleID, result.CompiledCount, result.CompilationTime,
		result.FromGroupSize, result.ToGroupSize, result.FromGroupSize*result.ToGroupSize)

	if result.HasWarnings() {
		for _, warning := range result.Warnings {
			log.Warn(warning)
		}
	}

	return result, nil
}

// CompileAllPolicies 编译所有启用的策略规则
func (pc *PolicyCompiler) CompileAllPolicies() error {
	rules, err := pc.storage.ListEnabledPolicyRules()
	if err != nil {
		return fmt.Errorf("failed to list policy rules: %w", err)
	}

	log.Infof("Compiling %d policy rules...", len(rules))

	totalCompiled := 0
	totalTime := time.Duration(0)

	for _, rule := range rules {
		result, err := pc.CompilePolicyRule(rule.ID)
		if err != nil {
			log.Errorf("Failed to compile rule %d (%s): %v", rule.ID, rule.Name, err)
			continue
		}

		totalCompiled += result.CompiledCount
		totalTime += result.CompilationTime
	}

	log.Infof("Compilation complete: %d rules → %d compiled policies in %v",
		len(rules), totalCompiled, totalTime)

	return nil
}

// InvalidateCompiledPolicies 删除某个规则的所有编译后策略
// 这在规则被删除或修改时调用
func (pc *PolicyCompiler) InvalidateCompiledPolicies(ruleID uint32) error {
	log.Debugf("Invalidating compiled policies for rule %d", ruleID)

	err := pc.storage.DeleteCompiledPoliciesForRule(ruleID)
	if err != nil {
		return fmt.Errorf("failed to delete compiled policies for rule %d: %w", ruleID, err)
	}

	return nil
}

// GetCompilationSummary 返回编译统计概览
func (pc *PolicyCompiler) GetCompilationSummary() (*CompilationSummary, error) {
	// 获取所有规则
	rules, err := pc.storage.ListPolicyRules()
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}

	// 获取所有编译后策略
	compiledPolicies, err := pc.storage.ListCompiledPolicies()
	if err != nil {
		return nil, fmt.Errorf("failed to list compiled policies: %w", err)
	}

	// 计算统计信息
	summary := &CompilationSummary{
		TotalRules:    len(rules),
		TotalCompiled: len(compiledPolicies),
	}

	// 统计已编译规则数
	compiledRuleIDs := make(map[uint32]bool)
	for _, cp := range compiledPolicies {
		compiledRuleIDs[cp.SourceRuleID] = true
		if cp.CompilationTime.After(summary.LastCompilation) {
			summary.LastCompilation = cp.CompilationTime
		}
	}
	summary.CompiledRules = len(compiledRuleIDs)

	// 计算平均扩展比率
	if summary.CompiledRules > 0 {
		summary.AverageExpansion = float64(summary.TotalCompiled) / float64(summary.CompiledRules)
	}

	return summary, nil
}

// allocateCompiledRuleID 分配一个新的编译后规则 ID
// 使用原子操作确保线程安全
func (pc *PolicyCompiler) allocateCompiledRuleID() uint32 {
	return atomic.AddUint32(&pc.nextCompiledID, 1) - 1
}

// SetNextCompiledID 设置下一个可用的编译后规则 ID
// 这在从存储恢复时使用，以避免 ID 冲突
func (pc *PolicyCompiler) SetNextCompiledID(id uint32) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if id > pc.nextCompiledID {
		pc.nextCompiledID = id
	}
}

// GetCompiledPoliciesForRule 获取某个源规则编译生成的所有策略
func (pc *PolicyCompiler) GetCompiledPoliciesForRule(ruleID uint32) ([]*CompiledPolicy, error) {
	return pc.storage.ListCompiledPoliciesForRule(ruleID)
}
