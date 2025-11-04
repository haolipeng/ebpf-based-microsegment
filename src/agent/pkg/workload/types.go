// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package workload

import (
	"encoding/json"
	"net"
	"time"
)

// WorkloadState 表示工作负载的运行状态
type WorkloadState string

const (
	// WorkloadRunning 表示工作负载正在运行
	WorkloadRunning WorkloadState = "running"
	// WorkloadStopped 表示工作负载已停止
	WorkloadStopped WorkloadState = "stopped"
	// WorkloadPaused 表示工作负载已暂停
	WorkloadPaused WorkloadState = "paused"
)

// Workload 表示一个工作负载（容器、进程或虚拟机）
// 工作负载通过唯一 ID 标识，并可以有多个 IP 地址和标签
type Workload struct {
	// 身份信息
	ID     string `json:"id" db:"id"`           // 唯一标识符
	Name   string `json:"name" db:"name"`       // 工作负载名称
	HostID string `json:"host_id" db:"host_id"` // 主机标识符

	// 网络信息（在数据库中以 JSON 序列化）
	IPs   []net.IP `json:"ips" db:"ips"`             // IP 地址列表
	MACs  []string `json:"macs" db:"macs"`           // MAC 地址列表
	Ports []uint16 `json:"ports,omitempty" db:"ports"` // 监听端口列表

	// 标签（系统核心 - 用于分组和策略匹配）
	Labels map[string]string `json:"labels" db:"labels"`

	// 用于自动标记的元数据
	Image       string `json:"image,omitempty" db:"image"`               // 容器镜像
	Namespace   string `json:"namespace,omitempty" db:"namespace"`       // Kubernetes 命名空间
	ServiceName string `json:"service_name,omitempty" db:"service_name"` // 服务名称
	PodName     string `json:"pod_name,omitempty" db:"pod_name"`         // Pod 名称
	ContainerID string `json:"container_id,omitempty" db:"container_id"` // 容器 ID（用于从运行时获取标签）
	ProcessName string `json:"process_name,omitempty" db:"process_name"` // 进程名称（用于自动标记）

	// 状态和时间戳
	State     WorkloadState `json:"state" db:"state"`           // 当前状态
	CreatedAt time.Time     `json:"created_at" db:"created_at"` // 创建时间
	UpdatedAt time.Time     `json:"updated_at" db:"updated_at"` // 最后更新时间
}

// NewWorkload 创建一个新的工作负载实例
// 如果未提供标签，则初始化为空 map
func NewWorkload(id, name, hostID string) *Workload {
	now := time.Now()
	return &Workload{
		ID:        id,
		Name:      name,
		HostID:    hostID,
		IPs:       make([]net.IP, 0),
		MACs:      make([]string, 0),
		Ports:     make([]uint16, 0),
		Labels:    make(map[string]string),
		State:     WorkloadRunning,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddLabel 添加或更新一个标签
func (w *Workload) AddLabel(key, value string) {
	if w.Labels == nil {
		w.Labels = make(map[string]string)
	}
	w.Labels[key] = value
	w.UpdatedAt = time.Now()
}

// RemoveLabel 删除一个标签
func (w *Workload) RemoveLabel(key string) {
	if w.Labels != nil {
		delete(w.Labels, key)
		w.UpdatedAt = time.Now()
	}
}

// GetLabel 获取标签值，如果不存在返回空字符串和 false
func (w *Workload) GetLabel(key string) (string, bool) {
	if w.Labels == nil {
		return "", false
	}
	value, exists := w.Labels[key]
	return value, exists
}

// HasLabel 检查是否存在指定的标签键
func (w *Workload) HasLabel(key string) bool {
	if w.Labels == nil {
		return false
	}
	_, exists := w.Labels[key]
	return exists
}

// AddIP 添加一个 IP 地址
func (w *Workload) AddIP(ip net.IP) {
	if w.IPs == nil {
		w.IPs = make([]net.IP, 0)
	}
	w.IPs = append(w.IPs, ip)
	w.UpdatedAt = time.Now()
}

// AddMAC 添加一个 MAC 地址
func (w *Workload) AddMAC(mac string) {
	if w.MACs == nil {
		w.MACs = make([]string, 0)
	}
	w.MACs = append(w.MACs, mac)
	w.UpdatedAt = time.Now()
}

// AddPort 添加一个监听端口
func (w *Workload) AddPort(port uint16) {
	if w.Ports == nil {
		w.Ports = make([]uint16, 0)
	}
	w.Ports = append(w.Ports, port)
	w.UpdatedAt = time.Now()
}

// IsRunning 检查工作负载是否处于运行状态
func (w *Workload) IsRunning() bool {
	return w.State == WorkloadRunning
}

// MarshalJSON 自定义 JSON 序列化
// 将 net.IP 转换为字符串以便正确序列化
func (w *Workload) MarshalJSON() ([]byte, error) {
	type Alias Workload

	// 将 net.IP 转换为字符串
	ips := make([]string, len(w.IPs))
	for i, ip := range w.IPs {
		ips[i] = ip.String()
	}

	return json.Marshal(&struct {
		IPs []string `json:"ips"`
		*Alias
	}{
		IPs:   ips,
		Alias: (*Alias)(w),
	})
}

// UnmarshalJSON 自定义 JSON 反序列化
// 将字符串转换回 net.IP
func (w *Workload) UnmarshalJSON(data []byte) error {
	type Alias Workload

	aux := &struct {
		IPs []string `json:"ips"`
		*Alias
	}{
		Alias: (*Alias)(w),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 将字符串转换为 net.IP
	w.IPs = make([]net.IP, len(aux.IPs))
	for i, ipStr := range aux.IPs {
		w.IPs[i] = net.ParseIP(ipStr)
	}

	return nil
}
