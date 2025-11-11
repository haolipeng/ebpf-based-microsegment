// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"testing"
)

func TestCreateMinimalXDPProgram(t *testing.T) {
	// 测试创建最小的 XDP 程序
	prog, err := createMinimalXDPProgram()
	if err != nil {
		t.Fatalf("Failed to create minimal XDP program: %v", err)
	}
	defer prog.Close()

	// 验证程序不为 nil
	if prog == nil {
		t.Fatal("Expected program but got nil")
	}

	// 验证程序类型为 XDP
	if prog.Type() != 6 { // XDP = 6
		t.Errorf("Expected program type XDP (6) but got %d", prog.Type())
	}
}

func TestTestNativeXDPSupport_InvalidInterface(t *testing.T) {
	// 测试不存在的网卡
	supported := testNativeXDPSupport("nonexistent999")
	if supported {
		t.Error("Expected false for nonexistent interface but got true")
	}
}

func TestTestGenericXDPSupport_InvalidInterface(t *testing.T) {
	// 测试不存在的网卡
	supported := testGenericXDPSupport("nonexistent999")
	if supported {
		t.Error("Expected false for nonexistent interface but got true")
	}
}

// 注意: 以下测试需要实际的网卡和 root 权限,因此被跳过
// 在集成测试环境中应该启用这些测试

func TestTestNativeXDPSupport_RealInterface(t *testing.T) {
	t.Skip("Skipping test that requires real network interface and root privileges")

	// 这个测试需要实际的网卡,例如 lo, eth0 等
	// 结果取决于网卡驱动是否支持 Native XDP
	// supported := testNativeXDPSupport("lo")
	// t.Logf("Native XDP support on lo: %v", supported)
}

func TestTestGenericXDPSupport_RealInterface(t *testing.T) {
	t.Skip("Skipping test that requires real network interface and root privileges")

	// Generic XDP 应该在大多数网卡上可用 (如果内核支持 XDP)
	// supported := testGenericXDPSupport("lo")
	// t.Logf("Generic XDP support on lo: %v", supported)
}

// 测试 tryAttachNativeXDP 函数的基本逻辑
func TestTryAttachNativeXDP_InvalidProgram(t *testing.T) {
	// 使用无效的程序指针应该失败
	// 注意: 这个测试实际上无法直接运行,因为需要有效的程序和网卡索引
	// 这里只是验证函数存在且可以调用
	t.Skip("Skipping test that requires real setup")
}
