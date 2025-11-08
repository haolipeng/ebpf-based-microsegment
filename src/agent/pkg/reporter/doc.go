// Package reporter 提供流量数据上报功能，将 Agent 收集的流量数据发送到远程 Server。
//
// 该包实现了灵活的上报机制，支持批量发送、重试、指标追踪等功能，
// 确保流量数据在网络不稳定情况下的可靠传输。
//
// # 核心接口
//
// Reporter 接口定义了流量上报的标准方法：
//
//	type Reporter interface {
//	    Report(ctx context.Context, flow *flow.Flow) error
//	    ReportBatch(ctx context.Context, flows []*flow.Flow) error
//	    Start() error
//	    Stop() error
//	}
//
// # 实现
//
// 当前包提供以下实现：
//
// 1. GRPCReporter - 基于 gRPC 的远程上报器
//   - 支持批量上报
//   - 指数退避重试机制
//   - 流式上报（gRPC Streaming）
//   - 上报指标追踪
//
// # 基本使用
//
// 创建并启动 GRPCReporter：
//
//	reporter := reporter.NewGRPCReporter(
//	    "localhost:9090",  // Server 地址
//	    "agent-001",       // Agent ID
//	    100,               // 批次大小
//	)
//
//	if err := reporter.Start(); err != nil {
//	    log.Fatal(err)
//	}
//	defer reporter.Stop()
//
// 上报单个流量：
//
//	flow := &flow.Flow{
//	    SourceIP:   "10.0.0.1",
//	    SourcePort: 12345,
//	    DestIP:     "10.0.0.2",
//	    DestPort:   80,
//	    Protocol:   "TCP",
//	}
//
//	err := reporter.Report(context.Background(), flow)
//	if err != nil {
//	    log.Errorf("Failed to report flow: %v", err)
//	}
//
// 批量上报流量：
//
//	flows := []*flow.Flow{flow1, flow2, flow3}
//	err := reporter.ReportBatch(context.Background(), flows)
//	if err != nil {
//	    log.Errorf("Failed to report batch: %v", err)
//	}
//
// # 重试机制
//
// GRPCReporter 实现了指数退避重试机制，提高网络不稳定时的数据传输可靠性。
//
// 使用自定义重试配置：
//
//	reporter := reporter.NewGRPCReporterWithRetry(
//	    "localhost:9090",
//	    "agent-001",
//	    100,                // 批次大小
//	    3,                  // 最大重试次数
//	    1*time.Second,      // 基础延迟
//	    30*time.Second,     // 最大延迟
//	)
//
// 重试延迟计算公式：
//
//	delay = min(retry_base_delay * 2^(attempt-1), retry_max_delay)
//
// 重试时间序列示例（base_delay=1s, max_delay=30s）：
//   - 第 1 次重试: 1s * 2^0 = 1 秒
//   - 第 2 次重试: 1s * 2^1 = 2 秒
//   - 第 3 次重试: 1s * 2^2 = 4 秒
//   - 第 4 次重试: 1s * 2^3 = 8 秒
//   - 第 5 次重试: 1s * 2^4 = 16 秒
//   - 第 6 次重试: 1s * 2^5 = 32 秒 → 限制为 30 秒
//
// # 批量上报机制
//
// GRPCReporter 内部维护一个批量队列，自动聚合流量事件：
//
//	┌──────────────────────────────────────┐
//	│ Report(flow)                         │
//	└──────────────┬───────────────────────┘
//	               ↓
//	┌──────────────────────────────────────┐
//	│ Batch Queue (200条缓冲)              │
//	└──────────────┬───────────────────────┘
//	               ↓
//	       触发条件：
//	       1. 队列满（达到批次大小）
//	       2. 定时器触发（5秒）
//	               ↓
//	┌──────────────────────────────────────┐
//	│ sendBatchWithRetry()                 │
//	└──────────────┬───────────────────────┘
//	               ↓
//	┌──────────────────────────────────────┐
//	│ gRPC Stream → Server                 │
//	└──────────────────────────────────────┘
//
// # 工作流程
//
// 1. 初始化连接
//
//	reporter.Start()
//	  → 建立 gRPC 连接
//	  → 创建 FlowService 客户端
//	  → 启动批量发送 goroutine
//
// 2. 流量上报
//
//	reporter.Report(flow)
//	  → 转换为 protobuf 格式
//	  → 加入批量队列
//	  → (异步批量发送)
//
// 3. 批量发送
//
//	batchSender goroutine:
//	  → 等待队列满或超时
//	  → 调用 sendBatchWithRetry()
//	  → 指数退避重试
//	  → 更新指标
//
// 4. 优雅关闭
//
//	reporter.Stop()
//	  → 停止接收新流量
//	  → 发送剩余批次
//	  → 关闭 gRPC 连接
//
// # gRPC 流式上报
//
// GRPCReporter 使用客户端流式 RPC 进行高效批量上报：
//
//	stream, err := client.ReportFlowEvents(ctx)
//	for _, event := range events {
//	    stream.Send(event)
//	}
//	resp, err := stream.CloseAndRecv()
//
// 流式上报的优势：
//   - 减少连接开销：单个连接发送多个事件
//   - 降低延迟：无需等待每个事件的响应
//   - 批量确认：接收端批量确认收到
//
// # 数据转换
//
// GRPCReporter 自动将内部 Flow 结构转换为 protobuf FlowEvent：
//
//	内部格式（flow.Flow）:
//	  - SourceIP: string ("10.0.0.1")
//	  - SourcePort: uint16 (12345)
//	  - Protocol: string ("TCP")
//	  - PolicyAction: string ("ALLOW")
//
//	↓ flowToProto()
//
//	Protobuf 格式（flowpb.FlowEvent）:
//	  - SrcIp: uint32 (0x0A000001)
//	  - SrcPort: uint32 (12345)
//	  - Protocol: enum (PROTOCOL_TCP)
//	  - PolicyAction: enum (ACTION_ALLOW)
//
// # 上报指标
//
// GRPCReporter 追踪以下指标：
//
//	sent, failed, retried := reporter.GetMetrics()
//
//	log.Printf("Sent: %d batches", sent)
//	log.Printf("Failed: %d batches", failed)
//	log.Printf("Retried: %d attempts", retried)
//
// 指标说明：
//   - totalSent: 成功发送的批次数
//   - totalFailed: 最终失败的批次数（重试耗尽后）
//   - totalRetried: 重试尝试总次数
//
// 计算成功率：
//
//	successRate := float64(sent) / float64(sent + failed) * 100
//	log.Printf("Success rate: %.2f%%", successRate)
//
// 计算平均重试次数：
//
//	avgRetries := float64(retried) / float64(sent)
//	log.Printf("Average retries per batch: %.2f", avgRetries)
//
// # 队列管理
//
// 批量队列配置：
//   - 队列容量: batchSize * 2（默认 200 条）
//   - 批次大小: batchSize（默认 100 条）
//   - 批次超时: 5 秒（固定）
//
// 队列行为：
//   - 队列未满: 非阻塞写入，立即返回
//   - 队列已满: 丢弃新流量，返回错误
//
// 队列满时的处理：
//
//	if err := reporter.Report(flow); err != nil {
//	    // 队列满，数据被丢弃
//	    log.Warnf("Flow dropped: %v", err)
//	}
//
// # 错误处理
//
// Reporter 的错误分类：
//
// 1. 连接错误
//
//	reporter.Start()
//	  → 无法连接到 Server
//	  → 返回错误，Agent 启动失败
//
// 2. 发送错误
//
//	reporter.Report(flow)
//	  → 队列已满
//	  → 返回错误，流量被丢弃
//
// 3. 网络错误（重试）
//
//	sendBatch()
//	  → 网络中断、超时
//	  → 自动重试（指数退避）
//	  → 重试耗尽后记录失败
//
// 4. Server 错误
//
//	stream.CloseAndRecv()
//	  → Server 返回失败响应
//	  → 视为发送失败，计入 totalFailed
//
// # 性能优化
//
// 批量发送优化：
//   - 默认批次大小 100 条，减少网络往返
//   - 5 秒超时，避免小批次等待过久
//   - 异步发送，不阻塞数据收集
//
// 内存优化：
//   - 队列容量限制，避免内存无限增长
//   - 队列满时丢弃，防止 OOM
//   - Protobuf 编码，减少网络传输
//
// 网络优化：
//   - gRPC 流式上报，减少连接开销
//   - 批量确认，降低延迟
//   - Keep-alive，避免连接重建
//
// # 配置建议
//
// 生产环境（高可靠性）：
//
//	reporter.NewGRPCReporterWithRetry(
//	    serverAddr,
//	    agentID,
//	    100,                // 批次大小
//	    5,                  // 更多重试次数
//	    2*time.Second,      // 稍长的基础延迟
//	    60*time.Second,     // 更长的最大延迟
//	)
//
// 低延迟环境（快速失败）：
//
//	reporter.NewGRPCReporterWithRetry(
//	    serverAddr,
//	    agentID,
//	    50,                 // 更小批次
//	    1,                  // 仅重试一次
//	    500*time.Millisecond, // 更短延迟
//	    5*time.Second,      // 较短上限
//	)
//
// 测试环境（无重试）：
//
//	reporter.NewGRPCReporterWithRetry(
//	    serverAddr,
//	    agentID,
//	    100,
//	    0,                  // 不重试
//	    0, 0,
//	)
//
// # 监控和告警
//
// 推荐监控的指标：
//
// 1. 成功率
//
//	successRate = sent / (sent + failed)
//	  应该接近 100%
//	  < 95% 时告警
//
// 2. 重试率
//
//	retryRate = retried / sent
//	  应该较低（< 0.5）
//	  > 1.0 时告警（网络不稳定）
//
// 3. 失败率
//
//	failureRate = failed / (sent + failed)
//	  应该接近 0
//	  > 1% 时告警
//
// 4. 队列使用率
//
//	queueUsage = len(batchQueue) / cap(batchQueue)
//	  应该保持较低（< 50%）
//	  > 80% 时告警（数据积压）
//
// # 日志输出
//
// 成功上报：
//
//	DEBUG Successfully sent 100 flow events to server
//
// 重试场景：
//
//	WARN  Send attempt 1 failed: connection refused
//	WARN  Retry attempt 1/3 after 1s delay
//	INFO  Batch sent successfully after 2 retries
//
// 最终失败：
//
//	ERROR Failed to send batch after 3 retries: connection refused
//
// 队列满：
//
//	WARN  Flow dropped: batch queue full, dropping flow
//
// # 线程安全
//
// GRPCReporter 的线程安全设计：
//   - Report() 方法并发安全（channel 通信）
//   - batchSender goroutine 独占队列读取
//   - 指标更新使用原子操作（atomic）
//
// 安全的并发使用：
//
//	// 多个 goroutine 并发上报
//	go reporter.Report(context.Background(), flow1)
//	go reporter.Report(context.Background(), flow2)
//	go reporter.Report(context.Background(), flow3)
//
// # 与持久化的配合
//
// Reporter 与 SQLite 持久化是互补关系：
//
// | 场景 | Reporter 重试 | SQLite 持久化 |
// |------|--------------|--------------|
// | 短暂网络中断（< 30秒） | ✅ 重试成功 | ⏸️ 不需要 |
// | 长时间网络中断（> 1分钟） | ❌ 重试失败 | ✅ 本地保存 |
// | 高流量导致队列溢出 | ❌ 队列满丢弃 | ✅ 持久化缓冲 |
// | Agent 重启 | ❌ 内存数据丢失 | ✅ 数据恢复 |
//
// 推荐配置：
//
//	# 启用重试机制（处理短暂故障）
//	max_retries: 3
//	retry_base_delay: 1s
//	retry_max_delay: 30s
//
//	# 同时启用持久化（防止长期故障）
//	enable_persistence: true
//	storage_path: ./data/flows.db
//
// # 协议定义
//
// gRPC 服务定义（proto/flow/flow_service.proto）：
//
//	service FlowService {
//	    rpc ReportFlowEvents(stream FlowEvent) returns (FlowReportResponse);
//	}
//
//	message FlowEvent {
//	    uint32 src_ip = 1;
//	    uint32 dst_ip = 2;
//	    uint32 src_port = 3;
//	    uint32 dst_port = 4;
//	    Protocol protocol = 5;
//	    FlowEventType event_type = 6;
//	    FlowDirection direction = 7;
//	    uint64 packet_count = 8;
//	    uint64 byte_count = 9;
//	    uint64 timestamp_ns = 10;
//	    uint32 policy_id = 11;
//	    PolicyAction policy_action = 12;
//	    FlowState state = 13;
//	    string agent_id = 14;
//	    map<string, string> source_labels = 15;
//	    map<string, string> dest_labels = 16;
//	}
//
//	message FlowReportResponse {
//	    bool success = 1;
//	    string message = 2;
//	    uint64 received_count = 3;
//	}
//
// # 未来改进
//
// 计划中的功能：
//   - 压缩传输：gzip/snappy 压缩减少带宽
//   - 优先级队列：重要流量（DENY）优先发送
//   - 持久化队列：失败批次保存到 SQLite，后台重传
//   - 自适应重试：根据网络状态动态调整重试参数
//   - 断路器模式：Server 长期不可用时暂停发送
//   - 多 Server 支持：负载均衡和高可用
//   - TLS 加密：支持安全连接
//   - 认证鉴权：支持 Token 认证
package reporter
