package k8s

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDefaultHealthCheckConfig(t *testing.T) {
	cfg := DefaultHealthCheckConfig()

	assert.Equal(t, 30*time.Second, cfg.Interval)
	assert.Equal(t, 5*time.Second, cfg.Timeout)
	assert.Equal(t, 1*time.Second, cfg.InitialBackoff)
	assert.Equal(t, 30*time.Second, cfg.MaxBackoff)
	assert.Equal(t, 2.0, cfg.BackoffMultiplier)
}

func TestHealthCheck(t *testing.T) {
	// 使用 fake clientset 创建测试客户端
	fakeClient := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClient,
	}

	// 健康检查应该成功(fake clientset 总是可用)
	err := client.HealthCheck()
	require.NoError(t, err)
}

func TestWaitForAPIServer(t *testing.T) {
	// 使用 fake clientset
	fakeClient := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClient,
	}

	// 应该立即成功
	err := client.WaitForAPIServer(5 * time.Second)
	require.NoError(t, err)
}

func TestPeriodicHealthCheck(t *testing.T) {
	// 使用 fake clientset
	fakeClient := fake.NewSimpleClientset()

	client := &Client{
		clientset: fakeClient,
	}

	cfg := &HealthCheckConfig{
		Interval:          100 * time.Millisecond, // 快速测试
		Timeout:           1 * time.Second,
		InitialBackoff:    10 * time.Millisecond,
		MaxBackoff:        100 * time.Millisecond,
		BackoffMultiplier: 2.0,
	}

	// 启动健康检查
	stop := client.StartPeriodicHealthCheck(cfg)

	// 运行一段时间
	time.Sleep(300 * time.Millisecond)

	// 停止健康检查
	stop()

	// 确保停止后不会 panic
	time.Sleep(100 * time.Millisecond)
}
