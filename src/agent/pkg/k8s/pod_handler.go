package k8s

import (
	"github.com/ebpf-microsegment/src/agent/pkg/workload"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

// PodEventHandler 处理 Pod 资源的 Add/Update/Delete 事件
type PodEventHandler struct {
	workloadMgr *workload.Manager
}

// NewPodEventHandler 创建新的 Pod 事件处理器
func NewPodEventHandler(workloadMgr *workload.Manager) *PodEventHandler {
	return &PodEventHandler{
		workloadMgr: workloadMgr,
	}
}

// OnAdd 处理 Pod 添加事件
func (h *PodEventHandler) OnAdd(obj interface{}, isInInitialList bool) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		log.Errorf("Expected Pod object but got %T", obj)
		return
	}

	h.handlePodAdd(pod)
}

// OnUpdate 处理 Pod 更新事件
func (h *PodEventHandler) OnUpdate(oldObj, newObj interface{}) {
	oldPod, ok1 := oldObj.(*corev1.Pod)
	newPod, ok2 := newObj.(*corev1.Pod)

	if !ok1 || !ok2 {
		log.Errorf("Expected Pod objects but got %T and %T", oldObj, newObj)
		return
	}

	h.handlePodUpdate(oldPod, newPod)
}

// OnDelete 处理 Pod 删除事件
func (h *PodEventHandler) OnDelete(obj interface{}) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		// 处理 DeletedFinalStateUnknown 情况
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Errorf("Expected Pod or DeletedFinalStateUnknown but got %T", obj)
			return
		}
		pod, ok = tombstone.Obj.(*corev1.Pod)
		if !ok {
			log.Errorf("DeletedFinalStateUnknown contained non-Pod object: %T", tombstone.Obj)
			return
		}
	}

	h.handlePodDelete(pod)
}

// handlePodAdd 处理 Pod 添加逻辑
func (h *PodEventHandler) handlePodAdd(pod *corev1.Pod) {
	log.WithFields(log.Fields{
		"namespace": pod.Namespace,
		"pod":       pod.Name,
		"uid":       pod.UID,
		"ip":        pod.Status.PodIP,
	}).Debug("Pod Add event received")

	// 转换 Pod 到 Workload
	wl, err := PodToWorkload(pod)
	if err != nil {
		log.WithError(err).Errorf("Failed to convert Pod to Workload: %s/%s", pod.Namespace, pod.Name)
		return
	}

	// 如果返回 nil，表示应该跳过这个 Pod（例如：没有 IP）
	if wl == nil {
		log.WithFields(log.Fields{
			"namespace": pod.Namespace,
			"pod":       pod.Name,
			"phase":     pod.Status.Phase,
		}).Debug("Skipping Pod without IP address")
		return
	}

	// 创建 Workload
	if err := h.workloadMgr.CreateWorkload(wl); err != nil {
		log.WithError(err).Errorf("Failed to create Workload for Pod %s/%s", pod.Namespace, pod.Name)
		return
	}

	log.WithFields(log.Fields{
		"namespace":   pod.Namespace,
		"pod":         pod.Name,
		"workload_id": wl.ID,
		"ip":          pod.Status.PodIP,
		"labels":      wl.Labels,
	}).Info("Pod synchronized to Workload system")
}

// handlePodUpdate 处理 Pod 更新逻辑
func (h *PodEventHandler) handlePodUpdate(oldPod, newPod *corev1.Pod) {
	log.WithFields(log.Fields{
		"namespace": newPod.Namespace,
		"pod":       newPod.Name,
		"uid":       newPod.UID,
		"old_ip":    oldPod.Status.PodIP,
		"new_ip":    newPod.Status.PodIP,
	}).Debug("Pod Update event received")

	// 转换新 Pod 到 Workload
	wl, err := PodToWorkload(newPod)
	if err != nil {
		log.WithError(err).Errorf("Failed to convert updated Pod to Workload: %s/%s", newPod.Namespace, newPod.Name)
		return
	}

	// 如果新 Pod 没有 IP，删除对应的 Workload
	if wl == nil {
		workloadID := string(newPod.UID)
		if err := h.workloadMgr.DeleteWorkload(workloadID); err != nil {
			log.WithError(err).Warnf("Failed to delete Workload for Pod without IP: %s/%s", newPod.Namespace, newPod.Name)
		}
		return
	}

	// 更新 Workload
	if err := h.workloadMgr.UpdateWorkload(wl); err != nil {
		log.WithError(err).Errorf("Failed to update Workload for Pod %s/%s", newPod.Namespace, newPod.Name)
		return
	}

	log.WithFields(log.Fields{
		"namespace":   newPod.Namespace,
		"pod":         newPod.Name,
		"workload_id": wl.ID,
		"ip":          newPod.Status.PodIP,
		"labels":      wl.Labels,
	}).Info("Pod updated in Workload system")
}

// handlePodDelete 处理 Pod 删除逻辑
func (h *PodEventHandler) handlePodDelete(pod *corev1.Pod) {
	log.WithFields(log.Fields{
		"namespace": pod.Namespace,
		"pod":       pod.Name,
		"uid":       pod.UID,
	}).Debug("Pod Delete event received")

	// 生成 Workload ID
	workloadID := string(pod.UID)

	// 删除 Workload
	if err := h.workloadMgr.DeleteWorkload(workloadID); err != nil {
		log.WithError(err).Errorf("Failed to delete Workload for Pod %s/%s", pod.Namespace, pod.Name)
		return
	}

	log.WithFields(log.Fields{
		"namespace":   pod.Namespace,
		"pod":         pod.Name,
		"workload_id": workloadID,
	}).Info("Pod deleted from Workload system")
}

// ResourceEventHandlerFuncs 返回符合 cache.ResourceEventHandler 接口的函数集
func (h *PodEventHandler) ResourceEventHandlerFuncs() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			h.OnAdd(obj, false)
		},
		UpdateFunc: h.OnUpdate,
		DeleteFunc: h.OnDelete,
	}
}
