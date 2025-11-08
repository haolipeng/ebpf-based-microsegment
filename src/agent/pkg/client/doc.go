// Package client 提供 Agent 与 Server 的通信客户端功能。
//
// 该包实现了 Agent 向 Server 注册、心跳、策略同步等核心通信功能，
// 确保 Agent 与 Server 之间的持续连接和数据交换。
//
// # 核心组件
//
// AgentClient - Agent 与 Server 通信的客户端：
//   - Agent 注册和注销
//   - 定期心跳保活
//   - 策略同步
//   - 指标上报
//
// # 基本使用
//
// 创建并连接 AgentClient：
//
//	client := client.NewAgentClient(
//	    "localhost:9090",  // Server 地址
//	    "agent-001",       // Agent ID
//	    "node-1",          // 主机名
//	    "1.0.0",           // 版本号
//	)
//
//	if err := client.Connect(); err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
// 启动心跳：
//
//	go client.StartHeartbeat()
//
// 同步策略：
//
//	currentVersion := uint64(0)
//	policies, newVersion, err := client.SyncPolicies(currentVersion)
//	if err != nil {
//	    log.Errorf("Failed to sync policies: %v", err)
//	}
//	log.Printf("Synced %d policies (version: %d)", len(policies), newVersion)
//
// 更新指标：
//
//	client.UpdateMetrics(flowCount, policyCount)
//
// # 工作流程
//
// 1. Agent 启动时注册
//
//	client.Connect()
//	  → 建立 gRPC 连接
//	  → 调用 RegisterAgent RPC
//	  → 接收 Server 配置（心跳间隔等）
//
// 2. 定期发送心跳
//
//	client.StartHeartbeat()
//	  → 创建定时器（默认 30 秒）
//	  → 发送 Heartbeat RPC
//	  → 上报 Agent 指标（CPU、内存、流量数）
//
// 3. 同步策略
//
//	client.SyncPolicies(currentVersion)
//	  → 发送当前策略版本
//	  → 接收增量或全量策略
//	  → 应用到本地 PolicyManager
//
// 4. Agent 关闭时注销
//
//	client.Close()
//	  → 调用 UnregisterAgent RPC
//	  → 关闭 gRPC 连接
//
// # Agent 注册
//
// 注册时发送的信息：
//
//	RegisterRequest:
//	  - agent_id: Agent 唯一标识
//	  - hostname: 主机名
//	  - version: Agent 版本号
//	  - ip_addresses: 本地 IP 地址列表
//	  - os: 操作系统信息
//	  - start_time: Agent 启动时间
//
// Server 返回的信息：
//
//	RegisterResponse:
//	  - success: 注册是否成功
//	  - message: 注册结果消息
//	  - server_version: Server 版本号
//	  - config:
//	      - heartbeat_interval: 心跳间隔（秒）
//	      - stats_interval: 统计间隔（秒）
//
// # 心跳机制
//
// 心跳数据包含：
//
//	HeartbeatRequest:
//	  - agent_id: Agent 标识
//	  - timestamp: 当前时间戳
//	  - metrics:
//	      - cpu_usage: CPU 使用率（百分比）
//	      - memory_usage: 内存使用量（字节）
//	      - flows_reported: 已上报流量数
//	      - active_policies: 活跃策略数
//
// Server 响应：
//
//	HeartbeatResponse:
//	  - healthy: Agent 健康状态
//	  - message: 健康检查消息
//	  - commands: Server 下发的命令（未来扩展）
//
// 心跳失败处理：
//   - 记录错误日志
//   - 继续发送下一次心跳
//   - 不中断 Agent 运行
//
// # 策略同步
//
// 策略同步请求：
//
//	SyncRequest:
//	  - agent_id: Agent 标识
//	  - policy_version: 当前策略版本
//	  - last_sync_time: 上次同步时间
//
// 策略同步响应：
//
//	SyncResponse:
//	  - policy_count: 策略数量
//	  - policy_version: 最新策略版本
//	  - policies: 策略列表
//	  - full_sync: 是否全量同步
//
// 同步策略：
//   - 增量同步：仅下发版本号更新的策略
//   - 全量同步：下发所有策略（版本不匹配时）
//
// # gRPC 服务接口
//
// AgentService 接口：
//
//	service AgentService {
//	    rpc RegisterAgent(RegisterRequest) returns (RegisterResponse);
//	    rpc UnregisterAgent(UnregisterRequest) returns (UnregisterResponse);
//	    rpc Heartbeat(HeartbeatRequest) returns (HeartbeatResponse);
//	}
//
// PolicyService 接口：
//
//	service PolicyService {
//	    rpc SyncPolicies(SyncRequest) returns (SyncResponse);
//	}
//
// # 连接管理
//
// 连接建立：
//   - 使用 gRPC 客户端
//   - 支持不安全连接（测试用）
//   - 支持 TLS 加密连接（生产环境）
//
// 连接保活：
//   - gRPC Keep-Alive
//   - 定期心跳检测
//   - 自动重连（未来实现）
//
// 优雅关闭：
//   - 停止心跳 goroutine
//   - 发送 UnregisterAgent
//   - 关闭 gRPC 连接
//
// # 指标收集
//
// AgentClient 收集并上报以下指标：
//
// 系统指标：
//   - CPU 使用率（cpu_usage）
//   - 内存使用量（memory_usage）
//
// 业务指标：
//   - 已上报流量数（flows_reported）
//   - 活跃策略数（active_policies）
//
// 更新业务指标：
//
//	// 在统计循环中定期更新
//	flowCount := stats.NewSessions
//	policyCount := uint32(pm.GetPolicyCount())
//	agentClient.UpdateMetrics(flowCount, policyCount)
//
// # 配置选项
//
// 心跳间隔：
//   - 默认: 30 秒
//   - Server 可通过注册响应配置
//   - 建议范围: 10-60 秒
//
// 统计间隔：
//   - 默认: 30 秒
//   - 用于指标上报频率
//   - 建议与心跳间隔一致
//
// RPC 超时：
//   - 注册: 10 秒
//   - 心跳: 5 秒
//   - 策略同步: 10 秒
//   - 注销: 5 秒
//
// # 错误处理
//
// 连接错误：
//
//	client.Connect()
//	  → 无法连接到 Server
//	  → 返回错误，Agent 启动失败
//	  → 建议：重试或使用 standalone 模式
//
// 注册错误：
//
//	RegisterAgent RPC
//	  → Server 拒绝注册
//	  → 返回错误，Agent 启动失败
//	  → 原因：重复 agent_id、版本不兼容等
//
// 心跳错误：
//
//	Heartbeat RPC
//	  → 网络超时或 Server 不可用
//	  → 记录错误日志
//	  → 继续发送下一次心跳
//	  → 不影响 Agent 运行
//
// 策略同步错误：
//
//	SyncPolicies RPC
//	  → 网络错误或 Server 异常
//	  → 返回错误
//	  → 保留当前策略继续运行
//	  → 下次同步时重试
//
// 注销错误：
//
//	UnregisterAgent RPC
//	  → 注销失败（网络中断等）
//	  → 记录警告日志
//	  → 继续关闭连接
//	  → Server 会通过心跳超时检测到 Agent 离线
//
// # 重连机制（未来实现）
//
// 计划中的重连功能：
//
//	连接断开检测:
//	  → 心跳连续失败 N 次
//	  → 触发重连逻辑
//
//	重连策略:
//	  → 指数退避重连
//	  → 最大重连次数
//	  → 重连成功后重新注册
//
//	降级策略:
//	  → 重连失败后切换到 standalone 模式
//	  → 保留本地策略继续运行
//	  → 本地存储流量数据
//
// # 安全性
//
// 当前实现（测试环境）：
//   - 不安全连接（insecure credentials）
//   - 无身份认证
//   - 明文传输
//
// 生产环境建议：
//
//	// TLS 加密连接
//	creds, err := credentials.NewClientTLSFromFile("server.crt", "")
//	conn, err := grpc.NewClient(
//	    serverAddr,
//	    grpc.WithTransportCredentials(creds),
//	)
//
//	// Token 认证
//	type tokenAuth struct {
//	    token string
//	}
//
//	func (t tokenAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
//	    return map[string]string{
//	        "authorization": "Bearer " + t.token,
//	    }, nil
//	}
//
//	conn, err := grpc.NewClient(
//	    serverAddr,
//	    grpc.WithPerRPCCredentials(tokenAuth{token: "secret-token"}),
//	)
//
// # Agent 生命周期
//
// 完整的 Agent 生命周期管理：
//
//	1. 启动阶段
//	   → 加载配置
//	   → 创建 AgentClient
//	   → 连接并注册到 Server
//	   → 启动心跳 goroutine
//
//	2. 运行阶段
//	   → 定期发送心跳
//	   → 响应 Server 命令（未来）
//	   → 同步策略更新
//	   → 上报指标数据
//
//	3. 关闭阶段
//	   → 收到 SIGINT/SIGTERM
//	   → 停止心跳
//	   → 发送注销请求
//	   → 关闭 gRPC 连接
//
// # 线程安全
//
// AgentClient 的并发安全：
//   - Connect()/Close() 方法不线程安全，仅在启动/关闭时调用
//   - StartHeartbeat() 在独立 goroutine 运行，通过 stopCh 优雅退出
//   - UpdateMetrics() 简单赋值，无并发保护（建议从单一 goroutine 调用）
//   - SyncPolicies() 可并发调用（RPC 本身线程安全）
//
// 推荐使用模式：
//
//	// 主 goroutine
//	client.Connect()
//	defer client.Close()
//
//	// 心跳 goroutine
//	go client.StartHeartbeat()
//
//	// 统计 goroutine
//	go func() {
//	    for range ticker.C {
//	        stats := getStats()
//	        client.UpdateMetrics(stats.FlowCount, stats.PolicyCount)
//	    }
//	}()
//
// # 监控和调试
//
// 关键日志：
//
//	INFO  AgentClient connected to localhost:9090
//	INFO  Agent registered successfully (agent_id=agent-001, heartbeat_interval=30)
//	DEBUG Heartbeat successful
//	INFO  Policies synchronized (policy_count=10, policy_version=5)
//	WARN  Heartbeat failed: rpc error: code = Unavailable desc = connection refused
//	INFO  Agent unregistered successfully
//
// 推荐监控指标：
//   - 连接状态：是否成功连接到 Server
//   - 心跳成功率：heartbeat_success / heartbeat_total
//   - 策略同步延迟：sync_duration_seconds
//   - RPC 错误率：rpc_errors / rpc_total
//
// # 与其他组件的交互
//
// AgentClient 在 Agent 架构中的位置：
//
//	┌─────────────────────────────────────┐
//	│ Agent Main                          │
//	└──────────────┬──────────────────────┘
//	               │
//	               ├─→ DataPlane (eBPF)
//	               ├─→ PolicyManager
//	               ├─→ FlowCollector
//	               ├─→ Reporter (流量上报)
//	               └─→ AgentClient (控制面通信)
//	                       │
//	                       ↓
//	                   gRPC → Server
//
// 交互流程：
//   1. AgentClient 注册到 Server
//   2. AgentClient 同步策略到 PolicyManager
//   3. AgentClient 定期上报指标（从 FlowCollector、PolicyManager 获取）
//   4. Reporter 独立上报流量数据（不通过 AgentClient）
//
// # 配置示例
//
// 配置文件（agent-server.yaml）：
//
//	server:
//	  server_addr: localhost:9090
//	  agent_id: ""  # 自动生成
//	  heartbeat_interval: 30s
//	  reconnect_interval: 30s
//
// 环境变量覆盖：
//
//	export MICROSEGMENT_SERVER_SERVER_ADDR=server.example.com:9090
//	export MICROSEGMENT_SERVER_AGENT_ID=agent-prod-001
//
// # 未来改进
//
// 计划中的功能：
//   - 自动重连机制
//   - TLS 加密和 Token 认证
//   - 压缩传输（gzip）
//   - 多 Server 支持（高可用）
//   - 策略推送（Server 主动推送而非轮询）
//   - 命令执行（Server 远程控制 Agent）
//   - 增强的健康检查
//   - 更详细的指标上报（网络带宽、磁盘使用等）
package client
