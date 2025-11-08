# Flow Reporter 重试机制

## 概述

为了提高 Agent 到 Server 的流量数据上报可靠性，我们为 GRPCReporter 实现了**指数退避重试机制（Exponential Backoff Retry）**。

## 问题场景

在原有实现中，如果流量数据发送失败，会直接丢弃：

```
网络中断 → gRPC 发送失败 → 记录错误日志 → 数据永久丢失 ❌
```

这导致在以下场景下会丢失数据：
- 网络暂时中断
- Server 短暂不可用
- 网络延迟导致超时
- 连接重置

## 解决方案：重试机制

### 核心特性

1. **指数退避**：重试延迟以指数方式增长（1s → 2s → 4s → 8s → 16s → 30s）
2. **最大重试次数**：默认最多重试 3 次
3. **延迟上限**：最大延迟不超过 30 秒
4. **可配置**：所有参数都可以通过配置文件调整
5. **指标追踪**：记录发送成功、失败、重试次数

### 工作流程

```
┌─────────────┐
│ 流量事件     │
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│ 批量缓冲队列    │ (200 条缓冲)
└──────┬──────────┘
       │
       ▼
┌─────────────────────────────────────────┐
│ 第 1 次尝试发送                         │
└──────┬──────────────────────────────────┘
       │
       ├─ 成功 ✅ → 记录成功指标
       │
       ├─ 失败 ❌
       │    │
       │    ▼ 等待 1 秒（base_delay）
       │    │
       ▼    ▼
┌─────────────────────────────────────────┐
│ 第 2 次尝试发送（重试 1/3）             │
└──────┬──────────────────────────────────┘
       │
       ├─ 成功 ✅ → 记录成功 + 重试次数
       │
       ├─ 失败 ❌
       │    │
       │    ▼ 等待 2 秒（base_delay * 2^1）
       │    │
       ▼    ▼
┌─────────────────────────────────────────┐
│ 第 3 次尝试发送（重试 2/3）             │
└──────┬──────────────────────────────────┘
       │
       ├─ 成功 ✅ → 记录成功 + 重试次数
       │
       ├─ 失败 ❌
       │    │
       │    ▼ 等待 4 秒（base_delay * 2^2）
       │    │
       ▼    ▼
┌─────────────────────────────────────────┐
│ 第 4 次尝试发送（重试 3/3，最后一次）  │
└──────┬──────────────────────────────────┘
       │
       ├─ 成功 ✅ → 记录成功 + 重试次数
       │
       └─ 失败 ❌ → 记录最终失败，数据丢失
```

## 配置说明

### 配置文件

在 `config/agent-server.yaml` 中配置：

```yaml
server:
  # gRPC 服务器地址
  server_addr: localhost:9090

  # 批量发送配置
  batch_size: 100
  batch_timeout: 5s

  # 重试配置
  max_retries: 3         # 最多重试 3 次（总共尝试 4 次）
  retry_base_delay: 1s   # 基础延迟 1 秒
  retry_max_delay: 30s   # 最大延迟上限 30 秒
```

### 配置参数说明

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `max_retries` | int | 3 | 最大重试次数，0 表示不重试 |
| `retry_base_delay` | duration | 1s | 基础延迟时间，用于指数退避计算 |
| `retry_max_delay` | duration | 30s | 最大延迟上限，避免无限增长 |

### 延迟计算公式

```
delay = min(retry_base_delay * 2^(attempt-1), retry_max_delay)
```

示例（base_delay=1s, max_delay=30s）：
- 第 1 次重试：1s * 2^0 = 1 秒
- 第 2 次重试：1s * 2^1 = 2 秒
- 第 3 次重试：1s * 2^2 = 4 秒
- 第 4 次重试：1s * 2^3 = 8 秒
- 第 5 次重试：1s * 2^4 = 16 秒
- 第 6 次重试：1s * 2^5 = 32 秒 → 限制为 30 秒
- 第 7+ 次重试：30 秒（达到上限）

## 代码实现

### 关键函数

#### 1. 重试逻辑

```go
func (r *GRPCReporter) sendBatchWithRetry(events []*flowpb.FlowEvent) error {
    var lastErr error

    for attempt := 0; attempt <= r.maxRetries; attempt++ {
        if attempt > 0 {
            // 指数退避
            delay := r.retryBaseDelay * time.Duration(1<<uint(attempt-1))
            if delay > r.retryMaxDelay {
                delay = r.retryMaxDelay
            }

            logrus.Warnf("Retry attempt %d/%d after %v delay", attempt, r.maxRetries, delay)
            time.Sleep(delay)
            r.totalRetried++
        }

        // 尝试发送
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        err := r.sendBatch(ctx, events)
        cancel()

        if err == nil {
            // 成功
            return nil
        }

        lastErr = err
    }

    return fmt.Errorf("all retry attempts exhausted: %w", lastErr)
}
```

#### 2. 指标收集

```go
// GetMetrics returns reporter metrics
func (r *GRPCReporter) GetMetrics() (sent, failed, retried uint64) {
    return r.totalSent, r.totalFailed, r.totalRetried
}
```

## 日志输出

### 成功场景

```
INFO  Sent 100 flow events to server
DEBUG Successfully sent 100 flow events to server
```

### 重试场景

```
WARN  Send attempt 1 failed: failed to create stream: connection refused
WARN  Retry attempt 1/3 after 1s delay
WARN  Send attempt 2 failed: failed to create stream: connection refused
WARN  Retry attempt 2/3 after 2s delay
INFO  Batch sent successfully after 2 retries
DEBUG Successfully sent 100 flow events to server
```

### 最终失败场景

```
WARN  Send attempt 1 failed: failed to create stream: connection refused
WARN  Retry attempt 1/3 after 1s delay
WARN  Send attempt 2 failed: failed to create stream: connection refused
WARN  Retry attempt 2/3 after 2s delay
WARN  Send attempt 3 failed: failed to create stream: connection refused
WARN  Retry attempt 3/3 after 4s delay
WARN  Send attempt 4 failed: failed to create stream: connection refused
ERROR Failed to send batch after 3 retries: all retry attempts exhausted: failed to create stream: connection refused
```

## 性能影响

### 内存

- 增加的内存开销：每个 Reporter 增加约 24 字节（3个uint64指标）
- 批次数据保留在内存中直到重试完成

### CPU

- 每次重试增加少量 CPU 开销（主要是 time.Sleep 和日志）
- 指数退避确保不会频繁重试，避免 CPU 浪费

### 网络

- 重试期间会增加网络流量
- 但指数退避减少了对 Server 的冲击

## 推荐配置

### 生产环境（高可靠性）

```yaml
server:
  max_retries: 5           # 更多重试次数
  retry_base_delay: 2s     # 稍长的基础延迟
  retry_max_delay: 60s     # 更长的最大延迟
```

**优点**：更高的成功率，适合网络不稳定环境
**缺点**：失败时延迟更长

### 低延迟环境（快速失败）

```yaml
server:
  max_retries: 1           # 仅重试一次
  retry_base_delay: 500ms  # 更短的延迟
  retry_max_delay: 5s      # 较短的上限
```

**优点**：快速失败，减少阻塞
**缺点**：成功率较低

### 测试环境（无重试）

```yaml
server:
  max_retries: 0           # 不重试
```

**优点**：快速发现问题
**缺点**：数据可能丢失

## 监控指标

可以通过 Reporter 的 `GetMetrics()` 方法获取指标：

```go
sent, failed, retried := reporter.GetMetrics()

log.Infof("Reporter Metrics:")
log.Infof("  Total Sent: %d batches", sent)
log.Infof("  Total Failed: %d batches", failed)
log.Infof("  Total Retries: %d attempts", retried)
log.Infof("  Success Rate: %.2f%%", float64(sent)/(float64(sent+failed))*100)
log.Infof("  Avg Retries per Batch: %.2f", float64(retried)/float64(sent))
```

建议监控的指标：
- **成功率**：`sent / (sent + failed)`，应该接近 100%
- **重试率**：`retried / sent`，应该较低（< 0.5）
- **失败率**：`failed / (sent + failed)`，应该接近 0

## 与持久化的配合

重试机制与 SQLite 持久化是**互补关系**：

| 场景 | 重试机制 | SQLite 持久化 |
|------|---------|---------------|
| 短暂网络中断（< 30秒） | ✅ 重试成功 | ⏸️ 不需要 |
| 长时间网络中断（> 1分钟） | ❌ 重试失败 | ✅ 本地保存 |
| 高流量导致队列溢出 | ❌ 队列满丢弃 | ✅ 持久化缓冲 |
| Agent 重启 | ❌ 内存数据丢失 | ✅ 数据恢复 |

**推荐配置**：
```yaml
# 启用重试机制（处理短暂故障）
server:
  max_retries: 3
  retry_base_delay: 1s
  retry_max_delay: 30s

# 同时启用持久化（防止长期故障）
flow:
  enable_persistence: true
  storage_path: ./data/flows.db
  retention_days: 7
```

## 未来改进

1. **基于 SQLite 的发送队列**（方案2）
   - 失败的批次保存到 SQLite
   - 后台 goroutine 定期重新发送
   - 真正的"零数据丢失"

2. **自适应重试**
   - 根据网络状态动态调整重试参数
   - 成功率高时减少延迟，低时增加延迟

3. **优先级队列**
   - 重要流量（DENY）优先重试
   - 普通流量（ALLOW）可以容忍丢失

4. **断路器模式**
   - Server 长期不可用时暂停发送
   - 避免浪费资源

## 总结

重试机制的核心价值：

✅ **提高可靠性**：短暂故障不再丢失数据
✅ **指数退避**：避免对 Server 造成雪崩效应
✅ **可配置**：适应不同环境需求
✅ **可观测**：提供指标用于监控

但仍需注意：
⚠️ 重试机制**不能替代**持久化
⚠️ 长期故障仍然会导致数据丢失
⚠️ 需要配合监控确保正常工作
