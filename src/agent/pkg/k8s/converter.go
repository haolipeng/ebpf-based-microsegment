package k8s

import (
	"fmt"
	"net"

	"github.com/ebpf-microsegment/src/agent/pkg/workload"
	corev1 "k8s.io/api/core/v1"
)

// PodToWorkload 将 Kubernetes Pod 转换为 Workload
// 返回 nil 表示该 Pod 应该被跳过（例如：没有 IP 地址）
func PodToWorkload(pod *corev1.Pod) (*workload.Workload, error) {
	if pod == nil {
		return nil, fmt.Errorf("pod is nil")
	}

	// 跳过没有 IP 的 Pod（Pending 状态或刚创建）
	if pod.Status.PodIP == "" {
		return nil, nil // 返回 nil 表示跳过，不是错误
	}

	// 生成 Workload ID: k8s:<namespace>:<uid>
	workloadID := fmt.Sprintf("k8s:%s:%s", pod.Namespace, pod.UID)

	// 解析 Pod IP
	podIP := net.ParseIP(pod.Status.PodIP)
	if podIP == nil {
		return nil, fmt.Errorf("invalid pod IP: %s", pod.Status.PodIP)
	}

	// 提取和映射标签
	labels := mapPodLabels(pod)

	// 添加 Kubernetes 元数据标签
	labels["k8s.namespace"] = pod.Namespace
	labels["k8s.pod.name"] = pod.Name
	if pod.Spec.NodeName != "" {
		labels["k8s.node.name"] = pod.Spec.NodeName
	}

	// 创建 Workload
	wl := &workload.Workload{
		ID:     workloadID,
		Name:   pod.Name,
		IPs:    []net.IP{podIP},
		Labels: labels,
	}

	return wl, nil
}

// mapPodLabels 映射 Pod 标签到 Workload 标签
// 应用标准 Kubernetes 标签映射规则
func mapPodLabels(pod *corev1.Pod) map[string]string {
	result := make(map[string]string)

	if pod.Labels == nil {
		return result
	}

	// 标准 Kubernetes 标签映射
	labelMappings := map[string]string{
		"app.kubernetes.io/name":      "app",
		"app.kubernetes.io/component": "role",
	}

	for k8sLabel, workloadLabel := range labelMappings {
		if value, ok := pod.Labels[k8sLabel]; ok {
			result[workloadLabel] = value
		}
	}

	// 保留其他所有标签（使用原始键名）
	for key, value := range pod.Labels {
		if _, isMapped := labelMappings[key]; !isMapped {
			result[key] = value
		}
	}

	return result
}
