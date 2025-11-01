// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package testutil

import (
	"context"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/vishvananda/netns"
)

// TestServer represents a test TCP/UDP server running in a namespace.
type TestServer struct {
	Protocol  string
	Port      int
	Namespace netns.NsHandle
	listener  net.Listener
	conn      net.PacketConn
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// StartTCPServer starts a TCP server in the specified namespace.
func StartTCPServer(ns netns.NsHandle, port int) (*TestServer, error) {
	server := &TestServer{
		Protocol:  "tcp",
		Port:      port,
		Namespace: ns,
	}

	ctx, cancel := context.WithCancel(context.Background())
	server.cancel = cancel

	// Start server in namespace
	errCh := make(chan error, 1)
	server.wg.Add(1)

	go func() {
		defer server.wg.Done()

		// Create listener in namespace
		var listener net.Listener
		err := RunInNamespace(ns, func() error {
			var listenErr error
			listener, listenErr = net.Listen("tcp", fmt.Sprintf(":%d", port))
			return listenErr
		})

		if err != nil {
			errCh <- fmt.Errorf("failed to create listener: %w", err)
			return
		}

		server.listener = listener
		errCh <- nil

		// Accept connections
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			conn, err := listener.Accept()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			// Handle connection in goroutine
			go func(c net.Conn) {
				defer c.Close()
				// Echo server - read and write back
				io.Copy(c, c)
			}(conn)
		}
	}()

	// Wait for server to start
	if err := <-errCh; err != nil {
		cancel()
		return nil, err
	}

	return server, nil
}

// StartUDPServer starts a UDP server in the specified namespace.
func StartUDPServer(ns netns.NsHandle, port int) (*TestServer, error) {
	server := &TestServer{
		Protocol:  "udp",
		Port:      port,
		Namespace: ns,
	}

	ctx, cancel := context.WithCancel(context.Background())
	server.cancel = cancel

	// Start server in namespace
	errCh := make(chan error, 1)
	server.wg.Add(1)

	go func() {
		defer server.wg.Done()

		// Create connection in namespace
		var conn net.PacketConn
		err := RunInNamespace(ns, func() error {
			var listenErr error
			conn, listenErr = net.ListenPacket("udp", fmt.Sprintf(":%d", port))
			return listenErr
		})

		if err != nil {
			errCh <- fmt.Errorf("failed to create UDP listener: %w", err)
			return
		}

		server.conn = conn
		errCh <- nil

		// Read packets
		buf := make([]byte, 65536)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, addr, err := conn.ReadFrom(buf)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			// Echo back
			conn.WriteTo(buf[:n], addr)
		}
	}()

	// Wait for server to start
	if err := <-errCh; err != nil {
		cancel()
		return nil, err
	}

	return server, nil
}

// Stop stops the test server.
func (ts *TestServer) Stop() {
	if ts.cancel != nil {
		ts.cancel()
	}

	if ts.listener != nil {
		ts.listener.Close()
	}

	if ts.conn != nil {
		ts.conn.Close()
	}

	ts.wg.Wait()
}

// SendTCPPacket sends a TCP packet to the specified destination.
// It establishes a connection, sends data, and verifies the echo response.
func SendTCPPacket(ns netns.NsHandle, dst string, port int, data []byte) error {
	return RunInNamespace(ns, func() error {
		// Connect to server
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", dst, port), 2*time.Second)
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		defer conn.Close()

		// Set deadline
		conn.SetDeadline(time.Now().Add(2 * time.Second))

		// Send data
		if _, err := conn.Write(data); err != nil {
			return fmt.Errorf("failed to send data: %w", err)
		}

		// Read response
		buf := make([]byte, len(data))
		if _, err := io.ReadFull(conn, buf); err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		// Verify echo
		if string(buf) != string(data) {
			return fmt.Errorf("echo mismatch: expected %s, got %s", data, buf)
		}

		return nil
	})
}

// SendUDPPacket sends a UDP packet to the specified destination.
func SendUDPPacket(ns netns.NsHandle, dst string, port int, data []byte) error {
	return RunInNamespace(ns, func() error {
		// Connect to server
		conn, err := net.Dial("udp", fmt.Sprintf("%s:%d", dst, port))
		if err != nil {
			return fmt.Errorf("failed to create UDP connection: %w", err)
		}
		defer conn.Close()

		// Set deadline
		conn.SetDeadline(time.Now().Add(2 * time.Second))

		// Send data
		if _, err := conn.Write(data); err != nil {
			return fmt.Errorf("failed to send UDP data: %w", err)
		}

		// Read response
		buf := make([]byte, 65536)
		n, err := conn.Read(buf)
		if err != nil {
			return fmt.Errorf("failed to read UDP response: %w", err)
		}

		// Verify echo
		if string(buf[:n]) != string(data) {
			return fmt.Errorf("UDP echo mismatch: expected %s, got %s", data, buf[:n])
		}

		return nil
	})
}

// PingHost sends an ICMP ping to the specified host from the namespace.
// Returns true if the ping succeeds, false otherwise.
func PingHost(ns netns.NsHandle, host string) (bool, error) {
	return PingHostWithCount(ns, host, 1)
}

// PingHostWithCount sends multiple ICMP pings and returns success status.
func PingHostWithCount(ns netns.NsHandle, host string, count int) (bool, error) {
	var success bool

	err := RunInNamespace(ns, func() error {
		cmd := exec.Command("ping", "-c", fmt.Sprintf("%d", count), "-W", "2", host)
		output, err := cmd.CombinedOutput()
		if err != nil {
			success = false
			return fmt.Errorf("ping failed: %w, output: %s", err, output)
		}

		success = true
		return nil
	})

	if err != nil {
		return false, err
	}

	return success, nil
}

// TryConnect attempts to establish a TCP connection to test reachability.
// Returns true if connection succeeds, false otherwise.
func TryConnect(ns netns.NsHandle, host string, port int) bool {
	var connected bool

	err := RunInNamespace(ns, func() error {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 1*time.Second)
		if err != nil {
			connected = false
			return err
		}
		defer conn.Close()

		connected = true
		return nil
	})

	return err == nil && connected
}

// TryConnectUDP attempts to send a UDP packet to test reachability.
// Returns true if packet can be sent, false otherwise.
func TryConnectUDP(ns netns.NsHandle, host string, port int) bool {
	var connected bool

	err := RunInNamespace(ns, func() error {
		conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:%d", host, port), 1*time.Second)
		if err != nil {
			connected = false
			return err
		}
		defer conn.Close()

		// Send a test packet
		if _, err := conn.Write([]byte("test")); err != nil {
			connected = false
			return err
		}

		connected = true
		return nil
	})

	return err == nil && connected
}

// WaitForServer waits for a TCP server to be ready.
// It tries to connect multiple times with exponential backoff.
func WaitForServer(ns netns.NsHandle, host string, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	attempt := 0

	for time.Now().Before(deadline) {
		if TryConnect(ns, host, port) {
			return nil
		}

		// Exponential backoff
		attempt++
		wait := time.Duration(1<<uint(attempt)) * 10 * time.Millisecond
		if wait > 500*time.Millisecond {
			wait = 500 * time.Millisecond
		}
		time.Sleep(wait)
	}

	return fmt.Errorf("timeout waiting for server at %s:%d", host, port)
}

// CapturePackets captures packets on an interface in a namespace.
// Returns the captured output as a string.
func CapturePackets(ns netns.NsHandle, iface string, filter string, duration time.Duration) (string, error) {
	var output string

	err := RunInNamespace(ns, func() error {
		// Use tcpdump to capture packets
		args := []string{"-i", iface, "-c", "10", "-n"}
		if filter != "" {
			args = append(args, filter)
		}

		ctx, cancel := context.WithTimeout(context.Background(), duration)
		defer cancel()

		cmd := exec.CommandContext(ctx, "tcpdump", args...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			// tcpdump returns error if no packets captured, check output
			if !strings.Contains(string(out), "packets captured") {
				return fmt.Errorf("tcpdump failed: %w, output: %s", err, out)
			}
		}

		output = string(out)
		return nil
	})

	return output, err
}

// RunInNamespace executes a function in the specified namespace.
// This is a helper function for internal use.
func RunInNamespace(ns netns.NsHandle, fn func() error) error {
	// Get original namespace
	originalNS, err := netns.Get()
	if err != nil {
		return fmt.Errorf("failed to get original namespace: %w", err)
	}
	defer originalNS.Close()

	// Enter the namespace
	if err := netns.Set(ns); err != nil {
		return fmt.Errorf("failed to enter namespace: %w", err)
	}

	// Execute the function
	execErr := fn()

	// Return to original namespace
	if err := netns.Set(originalNS); err != nil {
		if execErr != nil {
			return fmt.Errorf("function error: %v, namespace restore error: %w", execErr, err)
		}
		return fmt.Errorf("failed to restore namespace: %w", err)
	}

	return execErr
}
