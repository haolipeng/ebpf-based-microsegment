// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"fmt"
	"net"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/asm"
	"github.com/cilium/ebpf/link"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// testNativeXDPSupport 测试网卡是否支持 Native XDP
// 通过实际尝试附加一个最小的 XDP 程序来验证驱动支持
func testNativeXDPSupport(ifaceName string) bool {
	// 1. 获取网卡索引
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		log.Debugf("Failed to get interface %s: %v", ifaceName, err)
		return false
	}

	// 2. 创建一个最小的测试 XDP 程序
	// 这个程序只返回 XDP_PASS,不做任何处理
	testProg, err := createMinimalXDPProgram()
	if err != nil {
		log.Debugf("Failed to create test XDP program: %v", err)
		return false
	}
	defer testProg.Close()

	// 3. 尝试以 Native XDP (DRV mode) 附加到网卡
	// 设置超时以防止长时间阻塞
	resultChan := make(chan bool, 1)
	go func() {
		supported := tryAttachNativeXDP(iface.Index, testProg)
		resultChan <- supported
	}()

	// 等待结果,最多等待 100ms
	select {
	case supported := <-resultChan:
		if supported {
			log.Debugf("Native XDP test: SUPPORTED on %s", ifaceName)
		} else {
			log.Debugf("Native XDP test: NOT SUPPORTED on %s (driver limitation)", ifaceName)
		}
		return supported
	case <-time.After(100 * time.Millisecond):
		log.Debugf("Native XDP test: TIMEOUT on %s", ifaceName)
		return false
	}
}

// createMinimalXDPProgram 创建一个最小的 XDP 测试程序
// 程序只包含一条指令: 返回 XDP_PASS (2)
func createMinimalXDPProgram() (*ebpf.Program, error) {
	// 创建一个最简单的 XDP 程序:
	// r0 = XDP_PASS (2)
	// exit
	instructions := asm.Instructions{
		// 加载 XDP_PASS (2) 到 r0
		asm.Mov.Imm(asm.R0, 2), // XDP_PASS = 2
		asm.Return(),
	}

	spec := &ebpf.ProgramSpec{
		Name:         "xdp_test",
		Type:         ebpf.XDP,
		Instructions: instructions,
		License:      "GPL",
	}

	prog, err := ebpf.NewProgram(spec)
	if err != nil {
		return nil, fmt.Errorf("creating test program: %w", err)
	}

	return prog, nil
}

// tryAttachNativeXDP 尝试以 Native XDP 模式附加程序
func tryAttachNativeXDP(ifaceIdx int, prog *ebpf.Program) bool {
	// 尝试使用 XDP_FLAGS_DRV_MODE (Native XDP)
	xdpLink, err := link.AttachXDP(link.XDPOptions{
		Program:   prog,
		Interface: ifaceIdx,
		Flags:     unix.XDP_FLAGS_DRV_MODE, // Native XDP (驱动模式)
	})

	if err != nil {
		// 附加失败,说明不支持 Native XDP
		log.Debugf("Native XDP attach failed: %v", err)
		return false
	}

	// 附加成功!立即清理
	defer func() {
		if err := xdpLink.Close(); err != nil {
			log.Debugf("Failed to detach test XDP program: %v", err)
		}
	}()

	return true
}

// testGenericXDPSupport 测试是否支持 Generic XDP
// Generic XDP 是内核级别的回退实现,几乎总是可用的 (如果内核支持 XDP)
func testGenericXDPSupport(ifaceName string) bool {
	// 获取网卡索引
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		log.Debugf("Failed to get interface %s: %v", ifaceName, err)
		return false
	}

	// 创建测试程序
	testProg, err := createMinimalXDPProgram()
	if err != nil {
		log.Debugf("Failed to create test XDP program: %v", err)
		return false
	}
	defer testProg.Close()

	// 尝试使用 XDP_FLAGS_SKB_MODE (Generic XDP)
	xdpLink, err := link.AttachXDP(link.XDPOptions{
		Program:   testProg,
		Interface: iface.Index,
		Flags:     unix.XDP_FLAGS_SKB_MODE, // Generic XDP (SKB 模式)
	})

	if err != nil {
		log.Debugf("Generic XDP attach failed: %v", err)
		return false
	}

	// 附加成功!立即清理
	defer func() {
		if err := xdpLink.Close(); err != nil {
			log.Debugf("Failed to detach test XDP program: %v", err)
		}
	}()

	log.Debugf("Generic XDP test: SUPPORTED on %s", ifaceName)
	return true
}
