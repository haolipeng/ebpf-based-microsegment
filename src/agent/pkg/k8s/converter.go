package k8s

import (
	"fmt"
	"net"

	"github.com/haolipeng/ebpf-based-microsegment/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// 标准 Kubernetes 标签映射规则
	// 优先级: 如果同一个目标标签有多个源标签,使用第一个找到的值
	labelMappings := []struct {
		source string // Kubernetes 标签
		target string // Workload 标签
	}{
		// 应用名称
		{"app.kubernetes.io/name", "app"},
		{"app", "app"}, // 兼容简化标签
		// 组件/角色
		{"app.kubernetes.io/component", "role"},
		{"component", "role"}, // 兼容简化标签
		// 环境
		{"environment", "env"},
		{"env", "env"},
		// 位置/区域
		{"topology.kubernetes.io/zone", "loc"},
		{"topology.kubernetes.io/region", "region"},
		// 版本
		{"app.kubernetes.io/version", "version"},
		{"version", "version"},
		// 实例标识
		{"app.kubernetes.io/instance", "instance"},
	}

	// 跟踪已映射的目标标签,避免重复映射
	mappedTargets := make(map[string]bool)
	// 跟踪已处理的源标签,避免重复保留
	processedSources := make(map[string]bool)

	// 应用映射规则
	for _, mapping := range labelMappings {
		if value, ok := pod.Labels[mapping.source]; ok {
			// 如果目标标签还没有被映射过,则映射
			if !mappedTargets[mapping.target] {
				result[mapping.target] = value
				mappedTargets[mapping.target] = true
			}
			// 标记源标签已处理
			processedSources[mapping.source] = true
		}
	}

	// 保留其他所有未处理的标签（使用原始键名）
	for key, value := range pod.Labels {
		if !processedSources[key] {
			result[key] = value
		}
	}

	return result
}

// MapPodLabels 导出版本的 mapPodLabels,供外部使用
func MapPodLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return make(map[string]string)
	}

	// 创建临时 Pod 对象用于复用映射逻辑
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
	}

	return mapPodLabels(pod)
}
