// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package dataplane

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cilium/ebpf"
	log "github.com/sirupsen/logrus"
)

const (
	// BPF 文件系统挂载点
	bpfFSPath = "/sys/fs/bpf"

	// Microsegment 项目的 pin 目录
	pinBasePath = "/sys/fs/bpf/microsegment"
)

// MapPinConfig 定义 Map Pinning 配置
type MapPinConfig struct {
	// Pin 目录路径 (默认: /sys/fs/bpf/microsegment)
	PinPath string

	// 是否在程序退出时取消 pin (默认: false,保留 map 以供共享)
	UnpinOnClose bool
}

// DefaultMapPinConfig 返回默认的 Map Pinning 配置
func DefaultMapPinConfig() *MapPinConfig {
	return &MapPinConfig{
		PinPath:      pinBasePath,
		UnpinOnClose: false, // 保留 map 以供 TC 和 XDP 共享
	}
}

// EnsureBPFFS 确保 BPF 文件系统已挂载
func EnsureBPFFS() error {
	// 检查 /sys/fs/bpf 目录是否存在
	info, err := os.Stat(bpfFSPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("BPF filesystem not mounted at %s: %w", bpfFSPath, err)
		}
		return fmt.Errorf("checking BPF filesystem: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", bpfFSPath)
	}

	log.Debugf("BPF filesystem available at %s", bpfFSPath)
	return nil
}

// EnsurePinPath 确保 pin 目录存在
func EnsurePinPath(pinPath string) error {
	// 首先确保 BPF FS 已挂载
	if err := EnsureBPFFS(); err != nil {
		return err
	}

	// 创建 pin 目录 (如果不存在)
	if err := os.MkdirAll(pinPath, 0755); err != nil {
		return fmt.Errorf("creating pin directory %s: %w", pinPath, err)
	}

	log.Debugf("Pin directory ready: %s", pinPath)
	return nil
}

// GetPinnedMapPath 返回指定 map 的 pin 路径
func GetPinnedMapPath(pinPath, mapName string) string {
	return filepath.Join(pinPath, mapName)
}

// IsPinned 检查指定的 map 是否已经 pinned
func IsPinned(pinPath, mapName string) bool {
	path := GetPinnedMapPath(pinPath, mapName)
	_, err := os.Stat(path)
	return err == nil
}

// LoadPinnedMap 从文件系统加载已 pinned 的 map
func LoadPinnedMap(pinPath, mapName string) (*ebpf.Map, error) {
	path := GetPinnedMapPath(pinPath, mapName)

	log.Debugf("Loading pinned map: %s", path)

	m, err := ebpf.LoadPinnedMap(path, &ebpf.LoadPinOptions{
		ReadOnly: false, // 需要写权限以更新策略
	})
	if err != nil {
		return nil, fmt.Errorf("loading pinned map %s: %w", path, err)
	}

	log.Infof("✓ Loaded pinned map: %s", mapName)
	return m, nil
}

// PinMap 将 map 固定到文件系统
func PinMap(m *ebpf.Map, pinPath, mapName string) error {
	// 确保 pin 目录存在
	if err := EnsurePinPath(pinPath); err != nil {
		return err
	}

	path := GetPinnedMapPath(pinPath, mapName)

	log.Debugf("Pinning map to: %s", path)

	// 如果已经 pinned,先删除旧的
	if IsPinned(pinPath, mapName) {
		log.Debugf("Map already pinned, removing old pin: %s", path)
		if err := os.Remove(path); err != nil {
			log.Warnf("Failed to remove old pin: %v", err)
		}
	}

	// Pin map
	if err := m.Pin(path); err != nil {
		return fmt.Errorf("pinning map %s: %w", mapName, err)
	}

	log.Infof("✓ Pinned map: %s -> %s", mapName, path)
	return nil
}

// UnpinMap 取消 map 的 pin
func UnpinMap(pinPath, mapName string) error {
	path := GetPinnedMapPath(pinPath, mapName)

	if !IsPinned(pinPath, mapName) {
		log.Debugf("Map not pinned, skipping unpin: %s", mapName)
		return nil
	}

	log.Debugf("Unpinning map: %s", path)

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("unpinning map %s: %w", mapName, err)
	}

	log.Infof("Unpinned map: %s", mapName)
	return nil
}

// CleanupPinnedMaps 清理指定目录下的所有 pinned maps
func CleanupPinnedMaps(pinPath string) error {
	// 检查目录是否存在
	if _, err := os.Stat(pinPath); os.IsNotExist(err) {
		log.Debugf("Pin directory does not exist, nothing to clean: %s", pinPath)
		return nil
	}

	log.Debugf("Cleaning up pinned maps in: %s", pinPath)

	// 读取目录内容
	entries, err := os.ReadDir(pinPath)
	if err != nil {
		return fmt.Errorf("reading pin directory: %w", err)
	}

	// 删除所有文件
	var errs []error
	for _, entry := range entries {
		if entry.IsDir() {
			continue // 跳过子目录
		}

		path := filepath.Join(pinPath, entry.Name())
		if err := os.Remove(path); err != nil {
			errs = append(errs, fmt.Errorf("removing %s: %w", path, err))
		} else {
			log.Debugf("Removed pinned map: %s", entry.Name())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %v", errs)
	}

	log.Infof("✓ Cleaned up pinned maps in: %s", pinPath)
	return nil
}

// ListPinnedMaps 列出指定目录下所有 pinned maps
func ListPinnedMaps(pinPath string) ([]string, error) {
	// 检查目录是否存在
	if _, err := os.Stat(pinPath); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(pinPath)
	if err != nil {
		return nil, fmt.Errorf("reading pin directory: %w", err)
	}

	var maps []string
	for _, entry := range entries {
		if !entry.IsDir() {
			maps = append(maps, entry.Name())
		}
	}

	return maps, nil
}
