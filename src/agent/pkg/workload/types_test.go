// SPDX-License-Identifier: GPL-2.0 OR BSD-3-Clause
package workload

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

// TestNewWorkload 测试 NewWorkload 构造函数
func TestNewWorkload(t *testing.T) {
	id := "workload-123"
	name := "test-workload"
	hostID := "host-456"

	wl := NewWorkload(id, name, hostID)

	if wl.ID != id {
		t.Errorf("expected ID %s, got %s", id, wl.ID)
	}
	if wl.Name != name {
		t.Errorf("expected Name %s, got %s", name, wl.Name)
	}
	if wl.HostID != hostID {
		t.Errorf("expected HostID %s, got %s", hostID, wl.HostID)
	}
	if wl.State != WorkloadRunning {
		t.Errorf("expected State %s, got %s", WorkloadRunning, wl.State)
	}
	if wl.Labels == nil {
		t.Error("expected Labels to be initialized, got nil")
	}
	if wl.IPs == nil {
		t.Error("expected IPs to be initialized, got nil")
	}
	if wl.MACs == nil {
		t.Error("expected MACs to be initialized, got nil")
	}
	if wl.Ports == nil {
		t.Error("expected Ports to be initialized, got nil")
	}
	if wl.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if wl.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
}

// TestAddLabel 测试添加标签功能
func TestAddLabel(t *testing.T) {
	wl := NewWorkload("wl-1", "test", "host-1")
	initialTime := wl.UpdatedAt

	// 等待一小段时间以确保时间戳会改变
	time.Sleep(1 * time.Millisecond)

	wl.AddLabel("role", "web")

	value, exists := wl.Labels["role"]
	if !exists {
		t.Error("expected label 'role' to exist")
	}
	if value != "web" {
		t.Errorf("expected label value 'web', got %s", value)
	}
	if !wl.UpdatedAt.After(initialTime) {
		t.Error("expected UpdatedAt to be updated")
	}
}

// TestAddLabelUpdate 测试更新现有标签
func TestAddLabelUpdate(t *testing.T) {
	wl := NewWorkload("wl-1", "test", "host-1")
	wl.AddLabel("role", "web")
	wl.AddLabel("role", "db")

	value, exists := wl.Labels["role"]
	if !exists {
		t.Error("expected label 'role' to exist")
	}
	if value != "db" {
		t.Errorf("expected label value 'db', got %s", value)
	}
}

// TestAddLabelNilMap 测试在 Labels 为 nil 时添加标签
func TestAddLabelNilMap(t *testing.T) {
	wl := &Workload{
		ID:     "wl-1",
		Labels: nil,
	}

	wl.AddLabel("role", "web")

	if wl.Labels == nil {
		t.Error("expected Labels to be initialized")
	}
	value, exists := wl.Labels["role"]
	if !exists {
		t.Error("expected label 'role' to exist")
	}
	if value != "web" {
		t.Errorf("expected label value 'web', got %s", value)
	}
}

// TestRemoveLabel 测试删除标签功能
func TestRemoveLabel(t *testing.T) {
	wl := NewWorkload("wl-1", "test", "host-1")
	wl.AddLabel("role", "web")
	wl.AddLabel("app", "frontend")
	initialTime := wl.UpdatedAt

	time.Sleep(1 * time.Millisecond)

	wl.RemoveLabel("role")

	if _, exists := wl.Labels["role"]; exists {
		t.Error("expected label 'role' to be removed")
	}
	if _, exists := wl.Labels["app"]; !exists {
		t.Error("expected label 'app' to still exist")
	}
	if !wl.UpdatedAt.After(initialTime) {
		t.Error("expected UpdatedAt to be updated")
	}
}

// TestRemoveLabelNonExistent 测试删除不存在的标签
func TestRemoveLabelNonExistent(t *testing.T) {
	wl := NewWorkload("wl-1", "test", "host-1")
	wl.AddLabel("role", "web")

	// 删除不存在的标签不应导致错误
	wl.RemoveLabel("nonexistent")

	if len(wl.Labels) != 1 {
		t.Errorf("expected 1 label, got %d", len(wl.Labels))
	}
}

// TestRemoveLabelNilMap 测试在 Labels 为 nil 时删除标签
func TestRemoveLabelNilMap(t *testing.T) {
	wl := &Workload{
		ID:     "wl-1",
		Labels: nil,
	}

	// 删除标签时 Labels 为 nil 不应导致 panic
	wl.RemoveLabel("role")
}

// TestGetLabel 测试获取标签功能
func TestGetLabel(t *testing.T) {
	wl := NewWorkload("wl-1", "test", "host-1")
	wl.AddLabel("role", "web")

	value, exists := wl.GetLabel("role")
	if !exists {
		t.Error("expected label 'role' to exist")
	}
	if value != "web" {
		t.Errorf("expected label value 'web', got %s", value)
	}

	value, exists = wl.GetLabel("nonexistent")
	if exists {
		t.Error("expected label 'nonexistent' to not exist")
	}
	if value != "" {
		t.Errorf("expected empty string for nonexistent label, got %s", value)
	}
}

// TestGetLabelNilMap 测试在 Labels 为 nil 时获取标签
func TestGetLabelNilMap(t *testing.T) {
	wl := &Workload{
		ID:     "wl-1",
		Labels: nil,
	}

	value, exists := wl.GetLabel("role")
	if exists {
		t.Error("expected label to not exist when Labels is nil")
	}
	if value != "" {
		t.Errorf("expected empty string, got %s", value)
	}
}

// TestHasLabel 测试检查标签是否存在
func TestHasLabel(t *testing.T) {
	wl := NewWorkload("wl-1", "test", "host-1")
	wl.AddLabel("role", "web")

	if !wl.HasLabel("role") {
		t.Error("expected HasLabel('role') to return true")
	}
	if wl.HasLabel("nonexistent") {
		t.Error("expected HasLabel('nonexistent') to return false")
	}
}

// TestHasLabelNilMap 测试在 Labels 为 nil 时检查标签
func TestHasLabelNilMap(t *testing.T) {
	wl := &Workload{
		ID:     "wl-1",
		Labels: nil,
	}

	if wl.HasLabel("role") {
		t.Error("expected HasLabel to return false when Labels is nil")
	}
}

// TestAddIP 测试添加 IP 地址功能
func TestAddIP(t *testing.T) {
	wl := NewWorkload("wl-1", "test", "host-1")
	initialTime := wl.UpdatedAt

	time.Sleep(1 * time.Millisecond)

	ip := net.ParseIP("10.0.1.10")
	wl.AddIP(ip)

	if len(wl.IPs) != 1 {
		t.Errorf("expected 1 IP, got %d", len(wl.IPs))
	}
	if !wl.IPs[0].Equal(ip) {
		t.Errorf("expected IP %s, got %s", ip, wl.IPs[0])
	}
	if !wl.UpdatedAt.After(initialTime) {
		t.Error("expected UpdatedAt to be updated")
	}
}

// TestAddIPNilSlice 测试在 IPs 为 nil 时添加 IP
func TestAddIPNilSlice(t *testing.T) {
	wl := &Workload{
		ID:  "wl-1",
		IPs: nil,
	}

	ip := net.ParseIP("10.0.1.10")
	wl.AddIP(ip)

	if wl.IPs == nil {
		t.Error("expected IPs to be initialized")
	}
	if len(wl.IPs) != 1 {
		t.Errorf("expected 1 IP, got %d", len(wl.IPs))
	}
}

// TestAddMAC 测试添加 MAC 地址功能
func TestAddMAC(t *testing.T) {
	wl := NewWorkload("wl-1", "test", "host-1")
	initialTime := wl.UpdatedAt

	time.Sleep(1 * time.Millisecond)

	mac := "00:11:22:33:44:55"
	wl.AddMAC(mac)

	if len(wl.MACs) != 1 {
		t.Errorf("expected 1 MAC, got %d", len(wl.MACs))
	}
	if wl.MACs[0] != mac {
		t.Errorf("expected MAC %s, got %s", mac, wl.MACs[0])
	}
	if !wl.UpdatedAt.After(initialTime) {
		t.Error("expected UpdatedAt to be updated")
	}
}

// TestAddMACNilSlice 测试在 MACs 为 nil 时添加 MAC
func TestAddMACNilSlice(t *testing.T) {
	wl := &Workload{
		ID:   "wl-1",
		MACs: nil,
	}

	mac := "00:11:22:33:44:55"
	wl.AddMAC(mac)

	if wl.MACs == nil {
		t.Error("expected MACs to be initialized")
	}
	if len(wl.MACs) != 1 {
		t.Errorf("expected 1 MAC, got %d", len(wl.MACs))
	}
}

// TestAddPort 测试添加端口功能
func TestAddPort(t *testing.T) {
	wl := NewWorkload("wl-1", "test", "host-1")
	initialTime := wl.UpdatedAt

	time.Sleep(1 * time.Millisecond)

	port := uint16(8080)
	wl.AddPort(port)

	if len(wl.Ports) != 1 {
		t.Errorf("expected 1 port, got %d", len(wl.Ports))
	}
	if wl.Ports[0] != port {
		t.Errorf("expected port %d, got %d", port, wl.Ports[0])
	}
	if !wl.UpdatedAt.After(initialTime) {
		t.Error("expected UpdatedAt to be updated")
	}
}

// TestAddPortNilSlice 测试在 Ports 为 nil 时添加端口
func TestAddPortNilSlice(t *testing.T) {
	wl := &Workload{
		ID:    "wl-1",
		Ports: nil,
	}

	port := uint16(8080)
	wl.AddPort(port)

	if wl.Ports == nil {
		t.Error("expected Ports to be initialized")
	}
	if len(wl.Ports) != 1 {
		t.Errorf("expected 1 port, got %d", len(wl.Ports))
	}
}

// TestIsRunning 测试 IsRunning 状态检查
func TestIsRunning(t *testing.T) {
	wl := NewWorkload("wl-1", "test", "host-1")

	if !wl.IsRunning() {
		t.Error("expected new workload to be running")
	}

	wl.State = WorkloadStopped
	if wl.IsRunning() {
		t.Error("expected stopped workload to not be running")
	}

	wl.State = WorkloadPaused
	if wl.IsRunning() {
		t.Error("expected paused workload to not be running")
	}

	wl.State = WorkloadRunning
	if !wl.IsRunning() {
		t.Error("expected running workload to be running")
	}
}

// TestMarshalJSON 测试 JSON 序列化
func TestMarshalJSON(t *testing.T) {
	wl := NewWorkload("wl-1", "test-workload", "host-1")
	wl.AddLabel("role", "web")
	wl.AddLabel("app", "frontend")
	wl.AddIP(net.ParseIP("10.0.1.10"))
	wl.AddIP(net.ParseIP("192.168.1.100"))
	wl.AddMAC("00:11:22:33:44:55")
	wl.AddPort(8080)

	data, err := json.Marshal(wl)
	if err != nil {
		t.Fatalf("failed to marshal workload: %v", err)
	}

	// 验证 JSON 包含预期的字段
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if result["id"] != "wl-1" {
		t.Errorf("expected id 'wl-1', got %v", result["id"])
	}
	if result["name"] != "test-workload" {
		t.Errorf("expected name 'test-workload', got %v", result["name"])
	}

	// 检查 IPs 是否被序列化为字符串数组
	ips, ok := result["ips"].([]interface{})
	if !ok {
		t.Error("expected ips to be an array")
	}
	if len(ips) != 2 {
		t.Errorf("expected 2 IPs, got %d", len(ips))
	}
	if ips[0] != "10.0.1.10" {
		t.Errorf("expected IP '10.0.1.10', got %v", ips[0])
	}

	// 检查 Labels
	labels, ok := result["labels"].(map[string]interface{})
	if !ok {
		t.Error("expected labels to be a map")
	}
	if labels["role"] != "web" {
		t.Errorf("expected label role='web', got %v", labels["role"])
	}
}

// TestUnmarshalJSON 测试 JSON 反序列化
func TestUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"id": "wl-1",
		"name": "test-workload",
		"host_id": "host-1",
		"ips": ["10.0.1.10", "192.168.1.100"],
		"macs": ["00:11:22:33:44:55"],
		"ports": [8080, 443],
		"labels": {
			"role": "web",
			"app": "frontend"
		},
		"state": "running",
		"created_at": "2024-01-01T00:00:00Z",
		"updated_at": "2024-01-01T00:00:00Z"
	}`

	var wl Workload
	if err := json.Unmarshal([]byte(jsonData), &wl); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if wl.ID != "wl-1" {
		t.Errorf("expected ID 'wl-1', got %s", wl.ID)
	}
	if wl.Name != "test-workload" {
		t.Errorf("expected Name 'test-workload', got %s", wl.Name)
	}
	if len(wl.IPs) != 2 {
		t.Errorf("expected 2 IPs, got %d", len(wl.IPs))
	}
	if wl.IPs[0].String() != "10.0.1.10" {
		t.Errorf("expected IP '10.0.1.10', got %s", wl.IPs[0])
	}
	if wl.IPs[1].String() != "192.168.1.100" {
		t.Errorf("expected IP '192.168.1.100', got %s", wl.IPs[1])
	}
	if len(wl.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(wl.Labels))
	}
	if wl.Labels["role"] != "web" {
		t.Errorf("expected label role='web', got %s", wl.Labels["role"])
	}
	if len(wl.MACs) != 1 {
		t.Errorf("expected 1 MAC, got %d", len(wl.MACs))
	}
	if len(wl.Ports) != 2 {
		t.Errorf("expected 2 ports, got %d", len(wl.Ports))
	}
}

// TestJSONRoundTrip 测试 JSON 序列化和反序列化的往返转换
func TestJSONRoundTrip(t *testing.T) {
	original := NewWorkload("wl-1", "test-workload", "host-1")
	original.AddLabel("role", "web")
	original.AddLabel("app", "frontend")
	original.AddLabel("env", "production")
	original.AddIP(net.ParseIP("10.0.1.10"))
	original.AddIP(net.ParseIP("192.168.1.100"))
	original.AddMAC("00:11:22:33:44:55")
	original.AddMAC("aa:bb:cc:dd:ee:ff")
	original.AddPort(8080)
	original.AddPort(443)
	original.Image = "nginx:latest"
	original.Namespace = "default"
	original.ServiceName = "web-service"
	original.PodName = "web-pod-123"

	// 序列化
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	// 反序列化
	var restored Workload
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// 验证所有字段
	if restored.ID != original.ID {
		t.Errorf("ID mismatch: expected %s, got %s", original.ID, restored.ID)
	}
	if restored.Name != original.Name {
		t.Errorf("Name mismatch: expected %s, got %s", original.Name, restored.Name)
	}
	if restored.HostID != original.HostID {
		t.Errorf("HostID mismatch: expected %s, got %s", original.HostID, restored.HostID)
	}
	if len(restored.IPs) != len(original.IPs) {
		t.Errorf("IPs length mismatch: expected %d, got %d", len(original.IPs), len(restored.IPs))
	}
	for i := range original.IPs {
		if !restored.IPs[i].Equal(original.IPs[i]) {
			t.Errorf("IP mismatch at index %d: expected %s, got %s", i, original.IPs[i], restored.IPs[i])
		}
	}
	if len(restored.Labels) != len(original.Labels) {
		t.Errorf("Labels length mismatch: expected %d, got %d", len(original.Labels), len(restored.Labels))
	}
	for key, value := range original.Labels {
		if restored.Labels[key] != value {
			t.Errorf("Label mismatch for key %s: expected %s, got %s", key, value, restored.Labels[key])
		}
	}
	if restored.Image != original.Image {
		t.Errorf("Image mismatch: expected %s, got %s", original.Image, restored.Image)
	}
	if restored.State != original.State {
		t.Errorf("State mismatch: expected %s, got %s", original.State, restored.State)
	}
}
