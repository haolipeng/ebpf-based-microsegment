
// input: K8s API server, resource types
// output: SharedInformerFactory
// pos: K8s informer factory - if file updated, must sync with this header comment and pkg/k8s/CLAUDE.md
package k8s

import (
	"time"

	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

const (
	// DefaultResyncPeriod 是 Informer 的默认重新同步周期
	DefaultResyncPeriod = 30 * time.Minute
)

// PodInformer 封装 Pod 资源的 Informer
type PodInformer struct {
	informer cache.SharedIndexInformer
	stopCh   chan struct{}
}

// NewPodInformer 创建新的 Pod Informer
// 监听所有 Namespace 的 Pod 资源变化
func NewPodInformer(client *Client) *PodInformer {
	// 创建 SharedInformerFactory（监听所有 Namespace）
	factory := informers.NewSharedInformerFactory(client.GetClientset(), DefaultResyncPeriod)

	// 获取 Pod Informer
	informer := factory.Core().V1().Pods().Informer()

	log.Info("Pod Informer created with resync period: 30 minutes")

	return &PodInformer{
		informer: informer,
		stopCh:   make(chan struct{}),
	}
}

// AddEventHandler 添加事件处理器
func (pi *PodInformer) AddEventHandler(handler cache.ResourceEventHandler) {
	pi.informer.AddEventHandler(handler)
}

// Start 启动 Informer
func (pi *PodInformer) Start() {
	log.Info("Starting Pod Informer...")
	go pi.informer.Run(pi.stopCh)

	// 等待缓存同步完成
	log.Info("Waiting for Pod Informer cache to sync...")
	if !cache.WaitForCacheSync(pi.stopCh, pi.informer.HasSynced) {
		log.Error("Failed to sync Pod Informer cache")
		return
	}

	log.Info("Pod Informer cache synced successfully")
}

// Stop 停止 Informer
func (pi *PodInformer) Stop() {
	log.Info("Stopping Pod Informer...")
	close(pi.stopCh)
}

// GetInformer 返回底层的 SharedIndexInformer
func (pi *PodInformer) GetInformer() cache.SharedIndexInformer {
	return pi.informer
}

// ListPods 从本地缓存列出所有 Pod
func (pi *PodInformer) ListPods() ([]*corev1.Pod, error) {
	var pods []*corev1.Pod

	for _, obj := range pi.informer.GetStore().List() {
		if pod, ok := obj.(*corev1.Pod); ok {
			pods = append(pods, pod)
		}
	}

	return pods, nil
}

// GetPod 从本地缓存获取指定的 Pod
func (pi *PodInformer) GetPod(namespace, name string) (*corev1.Pod, error) {
	key := namespace + "/" + name
	obj, exists, err := pi.informer.GetStore().GetByKey(key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, nil
	}

	if pod, ok := obj.(*corev1.Pod); ok {
		return pod, nil
	}

	return nil, nil
}

// SimpleListWatch 创建一个简单的 ListWatch（用于测试）
func SimpleListWatch(client *Client) cache.ListerWatcher {
	return &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return client.GetClientset().CoreV1().Pods(corev1.NamespaceAll).List(nil, options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return client.GetClientset().CoreV1().Pods(corev1.NamespaceAll).Watch(nil, options)
		},
	}
}
