package k8s

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Config 包含 Kubernetes 客户端配置
type Config struct {
	// Enabled 是否启用 Kubernetes 集成
	Enabled bool
}

// Client 封装 Kubernetes 客户端
type Client struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

// NewClient 创建新的 Kubernetes 客户端（仅支持 in-cluster 模式）
// 该函数假设 Agent 运行在 Kubernetes 集群内，使用 ServiceAccount 进行认证
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, fmt.Errorf("kubernetes integration is disabled")
	}

	log.Info("Initializing Kubernetes client in in-cluster mode")

	// 使用 in-cluster 配置（从 ServiceAccount 读取 token 和证书）
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	// 创建 clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	log.Info("Kubernetes client initialized successfully in in-cluster mode")

	return &Client{
		clientset: clientset,
		config:    config,
	}, nil
}

// GetClientset 返回底层的 Kubernetes clientset
func (c *Client) GetClientset() *kubernetes.Clientset {
	return c.clientset
}

// GetConfig 返回底层的 REST config
func (c *Client) GetConfig() *rest.Config {
	return c.config
}

// Close 关闭客户端连接（当前为空实现，client-go 不需要显式关闭）
func (c *Client) Close() error {
	log.Info("Kubernetes client closed")
	return nil
}
