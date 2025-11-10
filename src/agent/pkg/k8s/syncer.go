package k8s

import (
	"fmt"

	"github.com/ebpf-microsegment/src/agent/pkg/workload"
	log "github.com/sirupsen/logrus"
)

// Syncer 负责同步 Kubernetes Pod 到 Workload 系统
type Syncer struct {
	client      *Client
	informer    *PodInformer
	handler     *PodEventHandler
	workloadMgr *workload.Manager
}

// NewSyncer 创建新的 Kubernetes 同步器
func NewSyncer(cfg *Config, workloadMgr *workload.Manager) (*Syncer, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, fmt.Errorf("kubernetes integration is not enabled")
	}

	if workloadMgr == nil {
		return nil, fmt.Errorf("workload manager is required")
	}

	// 创建 K8s 客户端
	client, err := NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// 创建 Pod Informer
	informer := NewPodInformer(client)

	// 创建 Pod 事件处理器
	handler := NewPodEventHandler(workloadMgr)

	// 注册事件处理器到 Informer
	informer.AddEventHandler(handler.ResourceEventHandlerFuncs())

	log.Info("Kubernetes Syncer created successfully")

	return &Syncer{
		client:      client,
		informer:    informer,
		handler:     handler,
		workloadMgr: workloadMgr,
	}, nil
}

// Start 启动同步器
func (s *Syncer) Start() error {
	log.Info("Starting Kubernetes Syncer...")

	// 启动 Informer
	s.informer.Start()

	log.Info("Kubernetes Syncer started successfully")
	return nil
}

// Stop 停止同步器
func (s *Syncer) Stop() error {
	log.Info("Stopping Kubernetes Syncer...")

	// 停止 Informer
	s.informer.Stop()

	// 关闭客户端
	if err := s.client.Close(); err != nil {
		log.WithError(err).Warn("Error closing Kubernetes client")
	}

	log.Info("Kubernetes Syncer stopped")
	return nil
}

// GetClient 返回底层的 Kubernetes 客户端
func (s *Syncer) GetClient() *Client {
	return s.client
}

// GetInformer 返回底层的 Pod Informer
func (s *Syncer) GetInformer() *PodInformer {
	return s.informer
}
