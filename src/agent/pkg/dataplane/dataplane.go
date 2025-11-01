// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

//go:generate go run github.com/cilium/ebpf/cmd/bpf2go -cc clang -cflags "-O2 -g -Wall" -target amd64 bpf ../../../bpf/tc_microsegment.bpf.c -- -I../../../bpf -I../../../../vmlinux/x86

// DataPlane manages the eBPF data plane
type DataPlane struct {
	objs      *bpfObjects
	iface     string
	ifaceIdx  int
	tcLink    link.Link
	tcFilter  *netlink.BpfFilter // For legacy TC cleanup
	rbReader  *ringbuf.Reader
	useLegacy bool // Track if using legacy TC attachment
}

// Statistics holds packet processing statistics
type Statistics struct {
	TotalPackets   uint64
	AllowedPackets uint64
	DeniedPackets  uint64
	NewSessions    uint64
	ClosedSessions uint64
	ActiveSessions uint64
	PolicyHits     uint64
	PolicyMisses   uint64
}

// New creates a new data plane instance
func New(iface string) (*DataPlane, error) {
	// Get interface index
	ifaceObj, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, fmt.Errorf("interface %s not found: %w", iface, err)
	}

	// Load eBPF objects
	objs := &bpfObjects{}
	if err := loadBpfObjects(objs, nil); err != nil {
		return nil, fmt.Errorf("loading eBPF objects: %w", err)
	}

	log.Debugf("eBPF objects loaded successfully")

	// Attach TC program to interface
	// Try TCX first (kernel >= 6.6), fallback to legacy TC hook if not supported
	var tcLink link.Link
	tcLink, err = link.AttachTCX(link.TCXOptions{
		Interface: ifaceObj.Index,
		Program:   objs.TcMicrosegmentFilter,
		Attach:    ebpf.AttachTCXIngress,
	})
	if err != nil {
		// TCX not supported (kernel < 6.6), fallback to legacy netlink-based TC hook
		log.Warnf("TCX attach failed (requires kernel >= 6.6), falling back to legacy TC hook: %v", err)

		// Attach using netlink (compatible with kernel >= 4.18)
		nlLink, err := netlink.LinkByIndex(ifaceObj.Index)
		if err != nil {
			objs.Close()
			return nil, fmt.Errorf("getting netlink interface: %w", err)
		}

		// Create clsact qdisc if not exists
		qdisc := &netlink.GenericQdisc{
			QdiscAttrs: netlink.QdiscAttrs{
				LinkIndex: ifaceObj.Index,
				Handle:    netlink.MakeHandle(0xffff, 0),
				Parent:    netlink.HANDLE_CLSACT,
			},
			QdiscType: "clsact",
		}

		// Try to add qdisc, ignore "file exists" error
		if err := netlink.QdiscAdd(qdisc); err != nil {
			// Check if it's "file exists" error (qdisc already present)
			if !isFileExistsError(err) {
				objs.Close()
				return nil, fmt.Errorf("adding clsact qdisc: %w", err)
			}
			log.Debugf("clsact qdisc already exists on %s", iface)
		} else {
			log.Debugf("Added clsact qdisc to %s", iface)
		}

		// Clean up any existing filters first (from previous runs)
		existingFilters, err := netlink.FilterList(nlLink, netlink.HANDLE_MIN_INGRESS)
		if err == nil {
			for _, f := range existingFilters {
				if bpfFilter, ok := f.(*netlink.BpfFilter); ok {
					if bpfFilter.Name == "tc_microsegment_filter" {
						netlink.FilterDel(bpfFilter)
						log.Debugf("Removed old BPF filter from %s", iface)
					}
				}
			}
		}

		// Attach BPF filter
		filter := &netlink.BpfFilter{
			FilterAttrs: netlink.FilterAttrs{
				LinkIndex: ifaceObj.Index,
				Parent:    netlink.HANDLE_MIN_INGRESS,
				Handle:    1,
				Protocol:  unix.ETH_P_ALL,
				Priority:  1,
			},
			Fd:           objs.TcMicrosegmentFilter.FD(),
			Name:         "tc_microsegment_filter",
			DirectAction: true,
		}

		if err := netlink.FilterAdd(filter); err != nil {
			objs.Close()
			return nil, fmt.Errorf("attaching TC filter: %w", err)
		}

		log.Infof("✓ TC program attached to %s ingress (legacy netlink mode, kernel < 6.6)", iface)

		// Setup ring buffer reader for flow events
		rbReader, err := ringbuf.NewReader(objs.FlowEvents)
		if err != nil {
			netlink.FilterDel(filter)
			objs.Close()
			return nil, fmt.Errorf("creating ring buffer reader: %w", err)
		}

		dp := &DataPlane{
			objs:      objs,
			iface:     iface,
			ifaceIdx:  ifaceObj.Index,
			tcLink:    nil,
			tcFilter:  filter,
			rbReader:  rbReader,
			useLegacy: true,
		}

		return dp, nil
	} else {
		log.Infof("✓ TC program attached to %s ingress (TCX mode, kernel >= 6.6)", iface)
	}

	// Setup ring buffer reader for flow events
	rbReader, err := ringbuf.NewReader(objs.FlowEvents)
	if err != nil {
		tcLink.Close()
		objs.Close()
		return nil, fmt.Errorf("creating ring buffer reader: %w", err)
	}

	dp := &DataPlane{
		objs:      objs,
		iface:     iface,
		ifaceIdx:  ifaceObj.Index,
		tcLink:    tcLink,
		tcFilter:  nil,
		rbReader:  rbReader,
		useLegacy: false,
	}

	return dp, nil
}

// Close cleans up the data plane resources
func (dp *DataPlane) Close() error {
	var errs []error

	if dp.rbReader != nil {
		if err := dp.rbReader.Close(); err != nil {
			errs = append(errs, fmt.Errorf("closing ring buffer reader: %w", err))
		}
	}

	// Clean up TC attachment (TCX or legacy)
	if dp.useLegacy && dp.tcFilter != nil {
		// Legacy netlink-based TC cleanup
		if err := netlink.FilterDel(dp.tcFilter); err != nil {
			errs = append(errs, fmt.Errorf("removing TC filter: %w", err))
		} else {
			log.Debugf("TC filter removed from %s", dp.iface)
		}
	} else if dp.tcLink != nil {
		// TCX cleanup
		if err := dp.tcLink.Close(); err != nil {
			errs = append(errs, fmt.Errorf("detaching TC program: %w", err))
		}
	}

	if dp.objs != nil {
		dp.objs.Close()
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	log.Info("Data plane closed successfully")
	return nil
}

// GetStatistics retrieves current packet processing statistics
func (dp *DataPlane) GetStatistics() Statistics {
	stats := Statistics{}

	// Helper function to read per-CPU array and sum values
	readStat := func(key uint32) uint64 {
		var values []uint64
		if err := dp.objs.StatsMap.Lookup(&key, &values); err != nil {
			log.Debugf("Failed to lookup stat key %d: %v", key, err)
			return 0
		}

		var total uint64
		for _, v := range values {
			total += v
		}
		return total
	}

	stats.TotalPackets = readStat(0)
	stats.AllowedPackets = readStat(1)
	stats.DeniedPackets = readStat(2)
	stats.NewSessions = readStat(3)
	stats.ClosedSessions = readStat(4)
	stats.ActiveSessions = readStat(5)
	stats.PolicyHits = readStat(6)
	stats.PolicyMisses = readStat(7)

	return stats
}

// MonitorFlowEvents continuously reads and processes flow events from ring buffer
func (dp *DataPlane) MonitorFlowEvents() {
	log.Info("Starting flow event monitoring")

	for {
		record, err := dp.rbReader.Read()
		if err != nil {
			if errors.Is(err, ringbuf.ErrClosed) {
				log.Info("Ring buffer closed")
				return
			}
			log.Errorf("Reading from ring buffer: %v", err)
			continue
		}

		// Parse flow event - simple struct parsing
		if len(record.RawSample) < 32 {
			log.Warn("Received incomplete flow event")
			continue
		}

		// Manual parsing of flow event structure
		// Parse flow key (5-tuple)
		srcIP := binary.LittleEndian.Uint32(record.RawSample[0:4])
		dstIP := binary.LittleEndian.Uint32(record.RawSample[4:8])
		srcPort := binary.LittleEndian.Uint16(record.RawSample[8:10])
		dstPort := binary.LittleEndian.Uint16(record.RawSample[10:12])
		protocol := record.RawSample[12]

		srcIPStr := intToIP(srcIP)
		dstIPStr := intToIP(dstIP)

		log.Infof("[FLOW EVENT] %s:%d -> %s:%d proto=%d",
			srcIPStr, srcPort,
			dstIPStr, dstPort,
			protocol)
	}
}

// Helper: Convert uint32 IP to net.IP
func intToIP(ip uint32) net.IP {
	return net.IPv4(byte(ip), byte(ip>>8), byte(ip>>16), byte(ip>>24))
}

// GetSessionMap returns the session map for external access
func (dp *DataPlane) GetSessionMap() *ebpf.Map {
	return dp.objs.SessionMap
}

// GetPolicyMap returns the policy map for external access
func (dp *DataPlane) GetPolicyMap() *ebpf.Map {
	return dp.objs.PolicyMap
}

// GetWildcardPolicyMap returns the wildcard policy map for external access
func (dp *DataPlane) GetWildcardPolicyMap() *ebpf.Map {
	return dp.objs.WildcardPolicyMap
}

// isFileExistsError checks if an error is due to "file exists"
func isFileExistsError(err error) bool {
	if err == nil {
		return false
	}
	// Check for EEXIST error (file exists)
	return err.Error() == "file exists" ||
		err.Error() == unix.EEXIST.Error() ||
		errors.Is(err, unix.EEXIST)
}
