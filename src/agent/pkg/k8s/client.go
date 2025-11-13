package k8s

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ConfigMode 定义客户端配置模式
type ConfigMode string

const (
	// ConfigModeAuto 自动检测模式（先 in-cluster，后 kubeconfig）
	ConfigModeAuto ConfigMode = "auto"
	// ConfigModeInCluster 集群内模式（使用 ServiceAccount）
	ConfigModeInCluster ConfigMode = "in-cluster"
	// ConfigModeKubeconfig 集群外模式（使用 kubeconfig）
	ConfigModeKubeconfig ConfigMode = "kubeconfig"
)

// Config 包含 Kubernetes 客户端配置
type Config struct {
	// Enabled 是否启用 Kubernetes 集成
	Enabled bool

	// ConfigMode 配置模式 (auto/in-cluster/kubeconfig)
	ConfigMode ConfigMode

	// KubeconfigPath kubeconfig 文件路径（仅 kubeconfig 模式）
	// 如果为空，使用默认路径 ~/.kube/config
	KubeconfigPath string

	// APIServer API Server 地址（可选，覆盖 kubeconfig 中的地址）
	APIServer string

	// QPS 客户端 QPS 限制（默认 5）
	QPS float32

	// Burst 客户端突发请求数（默认 10）
	Burst int

	// Timeout 请求超时时间（秒，默认 30）
	Timeout int
}

// Client 封装 Kubernetes 客户端
type Client struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

// NewClient 创建新的 Kubernetes 客户端
// 支持三种模式：auto（自动检测）、in-cluster（集群内）、kubeconfig（集群外）
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, fmt.Errorf("kubernetes integration is disabled")
	}

	// 设置默认值
	if cfg.ConfigMode == "" {
		cfg.ConfigMode = ConfigModeAuto
	}
	if cfg.QPS == 0 {
		cfg.QPS = 5
	}
	if cfg.Burst == 0 {
		cfg.Burst = 10
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30
	}

	var config *rest.Config
	var err error

	switch cfg.ConfigMode {
	case ConfigModeAuto:
		config, err = buildAutoConfig(cfg)
	case ConfigModeInCluster:
		config, err = buildInClusterConfig()
	case ConfigModeKubeconfig:
		config, err = buildKubeconfigConfig(cfg)
	default:
		return nil, fmt.Errorf("invalid config mode: %s (must be auto/in-cluster/kubeconfig)", cfg.ConfigMode)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	// 应用配置参数
	config.QPS = cfg.QPS
	config.Burst = cfg.Burst
	// Timeout 是持续时间(秒)
	if cfg.Timeout > 0 {
		config.Timeout = (time.Duration(cfg.Timeout) * time.Second).Round(time.Second)
	}

	// 创建 clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	log.WithFields(log.Fields{
		"mode":    cfg.ConfigMode,
		"qps":     cfg.QPS,
		"burst":   cfg.Burst,
		"timeout": cfg.Timeout,
	}).Info("Kubernetes client initialized successfully")

	return &Client{
		clientset: clientset,
		config:    config,
	}, nil
}

// buildAutoConfig 自动检测配置（先 in-cluster，后 kubeconfig）
func buildAutoConfig(cfg *Config) (*rest.Config, error) {
	// 1. 尝试 in-cluster 配置
	config, err := rest.InClusterConfig()
	if err == nil {
		log.Info("Using in-cluster configuration (detected ServiceAccount)")
		return config, nil
	}

	log.WithError(err).Debug("In-cluster config not available, trying kubeconfig")

	// 2. 回退到 kubeconfig
	config, err = buildKubeconfigConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("auto-detection failed: neither in-cluster nor kubeconfig available: %w", err)
	}

	return config, nil
}

// buildInClusterConfig 构建集群内配置
func buildInClusterConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load in-cluster config (is this running inside a Kubernetes pod?): %w", err)
	}

	log.Info("Using in-cluster configuration")
	return config, nil
}

// buildKubeconfigConfig 构建 kubeconfig 配置
func buildKubeconfigConfig(cfg *Config) (*rest.Config, error) {
	kubeconfigPath := cfg.KubeconfigPath

	// 如果未指定路径，使用默认路径
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	// 检查文件是否存在
	if _, err := os.Stat(kubeconfigPath); err != nil {
		return nil, fmt.Errorf("kubeconfig file not found at %s: %w", kubeconfigPath, err)
	}

	// 加载 kubeconfig
	config, err := clientcmd.BuildConfigFromFlags(cfg.APIServer, kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfigPath, err)
	}

	log.WithFields(log.Fields{
		"kubeconfig": kubeconfigPath,
		"api_server": config.Host,
	}).Info("Using kubeconfig configuration")

	return config, nil
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
