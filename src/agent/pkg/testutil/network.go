// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause

// Package testutil provides utilities for testing eBPF microsegmentation.
// It includes network namespace management, virtual interface creation,
// and traffic generation tools for end-to-end testing.
package testutil

import (
	"fmt"
	"net"
	"os"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
	"golang.org/x/sys/unix"
)

// TestNetwork represents an isolated network environment for testing.
// It creates two network namespaces connected via a veth pair.
type TestNetwork struct {
	// Network namespaces
	ClientNS netns.NsHandle
	ServerNS netns.NsHandle

	// Virtual ethernet interfaces
	ClientVeth string
	ServerVeth string

	// IP addresses
	ClientIP string
	ServerIP string

	// Original namespace (for cleanup)
	OriginalNS netns.NsHandle
}

// NetworkConfig contains configuration for test network creation.
type NetworkConfig struct {
	ClientVethName string
	ServerVethName string
	ClientIP       string
	ServerIP       string
	Subnet         string
}

// DefaultNetworkConfig returns default configuration for test network.
func DefaultNetworkConfig() *NetworkConfig {
	return &NetworkConfig{
		ClientVethName: "veth-client",
		ServerVethName: "veth-server",
		ClientIP:       "10.100.0.1/24",
		ServerIP:       "10.100.0.2/24",
		Subnet:         "10.100.0.0/24",
	}
}

// NewTestNetwork creates an isolated network environment with two namespaces.
// It creates veth pairs, assigns IP addresses, and sets up routing.
//
// The network topology:
//
//	[Client NS]                [Server NS]
//	    |                          |
//	veth-client  <-------->  veth-server
//	10.100.0.1              10.100.0.2
//
// Returns:
//   - *TestNetwork: The created test network
//   - error: Error if creation fails
func NewTestNetwork() (*TestNetwork, error) {
	return NewTestNetworkWithConfig(DefaultNetworkConfig())
}

// NewTestNetworkWithConfig creates a test network with custom configuration.
func NewTestNetworkWithConfig(cfg *NetworkConfig) (*TestNetwork, error) {
	// Lock the OS thread to ensure network namespace operations work correctly
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Save original namespace
	originalNS, err := netns.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get original namespace: %w", err)
	}

	tn := &TestNetwork{
		ClientVeth: cfg.ClientVethName,
		ServerVeth: cfg.ServerVethName,
		ClientIP:   cfg.ClientIP,
		ServerIP:   cfg.ServerIP,
		OriginalNS: originalNS,
	}

	// Create client namespace
	clientNS, err := netns.New()
	if err != nil {
		tn.Cleanup()
		return nil, fmt.Errorf("failed to create client namespace: %w", err)
	}
	tn.ClientNS = clientNS

	// Return to original namespace
	if err := netns.Set(originalNS); err != nil {
		tn.Cleanup()
		return nil, fmt.Errorf("failed to return to original namespace: %w", err)
	}

	// Create server namespace
	serverNS, err := netns.New()
	if err != nil {
		tn.Cleanup()
		return nil, fmt.Errorf("failed to create server namespace: %w", err)
	}
	tn.ServerNS = serverNS

	// Return to original namespace
	if err := netns.Set(originalNS); err != nil {
		tn.Cleanup()
		return nil, fmt.Errorf("failed to return to original namespace: %w", err)
	}

	// Create veth pair in original namespace
	veth := &netlink.Veth{
		LinkAttrs: netlink.LinkAttrs{
			Name: cfg.ClientVethName,
		},
		PeerName: cfg.ServerVethName,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		tn.Cleanup()
		return nil, fmt.Errorf("failed to create veth pair: %w", err)
	}

	// Get the veth interfaces
	clientVethLink, err := netlink.LinkByName(cfg.ClientVethName)
	if err != nil {
		tn.Cleanup()
		return nil, fmt.Errorf("failed to get client veth: %w", err)
	}

	serverVethLink, err := netlink.LinkByName(cfg.ServerVethName)
	if err != nil {
		tn.Cleanup()
		return nil, fmt.Errorf("failed to get server veth: %w", err)
	}

	// Move client veth to client namespace
	if err := netlink.LinkSetNsFd(clientVethLink, int(clientNS)); err != nil {
		tn.Cleanup()
		return nil, fmt.Errorf("failed to move client veth to namespace: %w", err)
	}

	// Move server veth to server namespace
	if err := netlink.LinkSetNsFd(serverVethLink, int(serverNS)); err != nil {
		tn.Cleanup()
		return nil, fmt.Errorf("failed to move server veth to namespace: %w", err)
	}

	// Configure client namespace
	if err := tn.configureNamespace(clientNS, cfg.ClientVethName, cfg.ClientIP); err != nil {
		tn.Cleanup()
		return nil, fmt.Errorf("failed to configure client namespace: %w", err)
	}

	// Configure server namespace
	if err := tn.configureNamespace(serverNS, cfg.ServerVethName, cfg.ServerIP); err != nil {
		tn.Cleanup()
		return nil, fmt.Errorf("failed to configure server namespace: %w", err)
	}

	// Return to original namespace
	if err := netns.Set(originalNS); err != nil {
		tn.Cleanup()
		return nil, fmt.Errorf("failed to return to original namespace: %w", err)
	}

	return tn, nil
}

// configureNamespace sets up network interface in a namespace.
func (tn *TestNetwork) configureNamespace(ns netns.NsHandle, vethName, ipAddr string) error {
	// Enter the namespace
	if err := netns.Set(ns); err != nil {
		return fmt.Errorf("failed to enter namespace: %w", err)
	}

	// Get the veth interface
	link, err := netlink.LinkByName(vethName)
	if err != nil {
		return fmt.Errorf("failed to get veth %s: %w", vethName, err)
	}

	// Parse IP address
	addr, err := netlink.ParseAddr(ipAddr)
	if err != nil {
		return fmt.Errorf("failed to parse IP %s: %w", ipAddr, err)
	}

	// Add IP address to interface
	if err := netlink.AddrAdd(link, addr); err != nil {
		return fmt.Errorf("failed to add IP address: %w", err)
	}

	// Bring up loopback
	lo, err := netlink.LinkByName("lo")
	if err != nil {
		return fmt.Errorf("failed to get loopback: %w", err)
	}
	if err := netlink.LinkSetUp(lo); err != nil {
		return fmt.Errorf("failed to bring up loopback: %w", err)
	}

	// Bring up the veth interface
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("failed to bring up veth: %w", err)
	}

	return nil
}

// RunInClientNS executes a function in the client network namespace.
func (tn *TestNetwork) RunInClientNS(fn func() error) error {
	return tn.runInNS(tn.ClientNS, fn)
}

// RunInServerNS executes a function in the server network namespace.
func (tn *TestNetwork) RunInServerNS(fn func() error) error {
	return tn.runInNS(tn.ServerNS, fn)
}

// runInNS executes a function in the specified namespace.
func (tn *TestNetwork) runInNS(ns netns.NsHandle, fn func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Enter the namespace
	if err := netns.Set(ns); err != nil {
		return fmt.Errorf("failed to enter namespace: %w", err)
	}

	// Execute the function
	err := fn()

	// Return to original namespace
	if setErr := netns.Set(tn.OriginalNS); setErr != nil {
		if err != nil {
			return fmt.Errorf("function error: %v, namespace restore error: %w", err, setErr)
		}
		return fmt.Errorf("failed to restore namespace: %w", setErr)
	}

	return err
}

// GetClientIP returns the client IP address without CIDR suffix.
func (tn *TestNetwork) GetClientIP() string {
	ip, _, _ := net.ParseCIDR(tn.ClientIP)
	return ip.String()
}

// GetServerIP returns the server IP address without CIDR suffix.
func (tn *TestNetwork) GetServerIP() string {
	ip, _, _ := net.ParseCIDR(tn.ServerIP)
	return ip.String()
}

// Cleanup removes all created network resources.
// It should be called with defer after creating the test network.
func (tn *TestNetwork) Cleanup() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Return to original namespace
	if tn.OriginalNS != 0 {
		_ = netns.Set(tn.OriginalNS)
	}

	// Close client namespace
	if tn.ClientNS != 0 {
		// Enter and delete the namespace by deleting its file
		_ = tn.ClientNS.Close()
	}

	// Close server namespace
	if tn.ServerNS != 0 {
		_ = tn.ServerNS.Close()
	}

	// Close original namespace handle
	if tn.OriginalNS != 0 {
		_ = tn.OriginalNS.Close()
	}
}

// IsRoot checks if the current process has root privileges.
// E2E tests require root to create network namespaces and load eBPF programs.
func IsRoot() bool {
	return os.Geteuid() == 0
}

// HasCapability checks if the process has a specific capability.
func HasCapability(cap int) bool {
	var header unix.CapUserHeader
	var data [2]unix.CapUserData

	header.Version = unix.LINUX_CAPABILITY_VERSION_3
	header.Pid = 0 // Current process

	if err := unix.Capget(&header, &data[0]); err != nil {
		return false
	}

	// Check if capability is in effective set
	capMask := uint32(1 << uint(cap))
	return (data[cap/32].Effective & capMask) != 0
}

// CheckE2ERequirements checks if the environment supports E2E testing.
// Returns an error message if requirements are not met, empty string otherwise.
func CheckE2ERequirements() string {
	// Check for root or capabilities
	if !IsRoot() {
		if !HasCapability(unix.CAP_NET_ADMIN) {
			return "E2E tests require root privileges or CAP_NET_ADMIN capability"
		}
		if !HasCapability(unix.CAP_BPF) && !HasCapability(unix.CAP_SYS_ADMIN) {
			return "E2E tests require CAP_BPF or CAP_SYS_ADMIN capability for eBPF operations"
		}
	}

	// Check if network namespaces are supported
	testNS, err := netns.New()
	if err != nil {
		return fmt.Sprintf("Network namespaces not supported: %v", err)
	}
	_ = testNS.Close()

	return ""
}
