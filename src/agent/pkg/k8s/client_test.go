package k8s

import (
	"testing"
)

func TestNewClient_Disabled(t *testing.T) {
	// 测试禁用 K8s 集成
	cfg := &Config{
		Enabled: false,
	}

	client, err := NewClient(cfg)
	if err == nil {
		t.Error("Expected error when kubernetes integration is disabled")
	}
	if client != nil {
		t.Error("Expected nil client when kubernetes integration is disabled")
	}
}

func TestNewClient_NilConfig(t *testing.T) {
	// 测试 nil 配置
	client, err := NewClient(nil)
	if err == nil {
		t.Error("Expected error for nil config")
	}
	if client != nil {
		t.Error("Expected nil client for nil config")
	}
}

// 注意：测试 in-cluster 配置需要在实际的 Kubernetes 集群中运行
// 或者使用 fake clientset 进行 mock，这里我们只测试配置验证逻辑
