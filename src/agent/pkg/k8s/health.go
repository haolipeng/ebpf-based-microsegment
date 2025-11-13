package k8s

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	// Interval 健康检查间隔（默认 30 秒）
	Interval time.Duration
	// Timeout 单次检查超时（默认 5 秒）
	Timeout time.Duration
	// InitialBackoff 初始退避时间（默认 1 秒）
	InitialBackoff time.Duration
	// MaxBackoff 最大退避时间（默认 30 秒）
	MaxBackoff time.Duration
	// BackoffMultiplier 退避倍数（默认 2.0）
	BackoffMultiplier float64
}

// DefaultHealthCheckConfig 返回默认健康检查配置
func DefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Interval:          30 * time.Second,
		Timeout:           5 * time.Second,
		InitialBackoff:    1 * time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// HealthCheck 执行单次健康检查
// 通过调用 /api/v1 端点验证 API Server 连通性
func (c *Client) HealthCheck() error {
	// 发送 GET /api/v1 请求
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("health check failed: unable to reach API server: %w", err)
	}

	return nil
}

// HealthCheckWithContext 执行带上下文的健康检查
func (c *Client) HealthCheckWithContext(ctx context.Context) error {
	// 尝试获取 server version
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// 验证 API 资源访问
	_, err = c.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		return fmt.Errorf("health check failed: unable to list namespaces: %w", err)
	}

	return nil
}

// StartPeriodicHealthCheck 启动定期健康检查
// 返回停止函数,调用者应在不再需要时调用停止函数
func (c *Client) StartPeriodicHealthCheck(cfg *HealthCheckConfig) (stopFunc func()) {
	if cfg == nil {
		cfg = DefaultHealthCheckConfig()
	}

	stopCh := make(chan struct{})
	stopped := make(chan struct{})

	go func() {
		defer close(stopped)

		ticker := time.NewTicker(cfg.Interval)
		defer ticker.Stop()

		currentBackoff := cfg.InitialBackoff
		consecutiveFailures := 0

		// 初始健康检查
		if err := c.HealthCheck(); err != nil {
			log.WithError(err).Warn("Initial health check failed, will retry")
			consecutiveFailures++
		} else {
			log.Info("Initial health check succeeded")
		}

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
				err := c.HealthCheckWithContext(ctx)
				cancel()

				if err != nil {
					consecutiveFailures++
					log.WithFields(log.Fields{
						"error":               err,
						"consecutive_failures": consecutiveFailures,
						"next_retry_in":       currentBackoff,
					}).Warn("Health check failed, will retry with backoff")

					// 指数退避
					time.Sleep(currentBackoff)
					currentBackoff = time.Duration(float64(currentBackoff) * cfg.BackoffMultiplier)
					if currentBackoff > cfg.MaxBackoff {
						currentBackoff = cfg.MaxBackoff
					}
				} else {
					if consecutiveFailures > 0 {
						log.WithFields(log.Fields{
							"previous_failures": consecutiveFailures,
						}).Info("Health check recovered")
					}
					// 重置退避
					currentBackoff = cfg.InitialBackoff
					consecutiveFailures = 0
				}

			case <-stopCh:
				log.Info("Stopping periodic health check")
				return
			}
		}
	}()

	return func() {
		close(stopCh)
		<-stopped // 等待 goroutine 退出
	}
}

// WaitForAPIServer 等待 API Server 可用
// 使用指数退避重试,直到成功或超时
func (c *Client) WaitForAPIServer(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cfg := DefaultHealthCheckConfig()
	currentBackoff := cfg.InitialBackoff

	log.WithField("timeout", timeout).Info("Waiting for API server to become available")

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for API server: %w", ctx.Err())
		default:
			err := c.HealthCheck()
			if err == nil {
				log.Info("API server is now available")
				return nil
			}

			log.WithFields(log.Fields{
				"error":       err,
				"retry_in":    currentBackoff,
			}).Debug("API server not ready, retrying with backoff")

			time.Sleep(currentBackoff)
			currentBackoff = time.Duration(float64(currentBackoff) * cfg.BackoffMultiplier)
			if currentBackoff > cfg.MaxBackoff {
				currentBackoff = cfg.MaxBackoff
			}
		}
	}
}
