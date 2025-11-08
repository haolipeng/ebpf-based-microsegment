// Package flow 提供网络流量数据的收集、聚合和持久化功能。
//
// 该包负责从 eBPF Ring Buffer 接收流量事件，进行聚合处理，添加工作负载标签，
// 并提供存储、查询和实时流式传输功能。
//
// # 核心组件
//
// 该包包含以下主要组件：
//
// 1. Collector - 流量收集器，从 eBPF Ring Buffer 读取事件
// 2. Storage - 流量持久化存储（SQLite 实现）
// 3. Hub - WebSocket 实时流式传输中心
// 4. LifecycleManager - 数据生命周期管理（清理过期数据）
// 5. Aggregator - 全局流量聚合分析器
//
// # 数据流
//
//	eBPF Kernel
//	    ↓ (Ring Buffer)
//	Collector
//	    ↓
//	├─→ 内存聚合（5元组）
//	├─→ 工作负载标签富化
//	├─→ SQLite 持久化
//	└─→ WebSocket 实时推送
//
// # 基本使用
//
// 创建流量收集器：
//
//	storage, _ := flow.NewSQLiteStorage("./data/flows.db")
//	ringBuf := dataPlane.GetFlowRingBuffer()
//
//	config := flow.CollectorConfig{
//	    FlowTimeout:       5 * time.Minute,
//	    BatchSize:         100,
//	    EnableEnrichment:  true,
//	    EnablePersistence: true,
//	    CleanupInterval:   1 * time.Minute,
//	}
//
//	collector := flow.NewCollector(ringBuf, storage, nil, config)
//	collector.Start()
//	defer collector.Stop()
//
// 查询流量数据：
//
//	query := &flow.FlowQuery{
//	    StartTime:    &startTime,
//	    EndTime:      &endTime,
//	    Protocol:     stringPtr("TCP"),
//	    PolicyAction: stringPtr("DENY"),
//	    Limit:        100,
//	    SortBy:       "start_time",
//	    SortOrder:    "desc",
//	}
//
//	flows, err := storage.QueryFlows(query)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// 获取流量统计摘要：
//
//	summary, err := storage.GetFlowSummary(startTime, endTime)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("Total flows: %d, Total bytes: %d", summary.TotalFlows, summary.TotalBytes)
//
// # WebSocket 实时流式传输
//
// 创建和使用 WebSocket Hub：
//
//	hub := flow.NewHub()
//	go hub.Run()
//	collector.SetWebSocketHub(hub)
//
//	// 在 HTTP 服务器中注册路由
//	http.HandleFunc("/ws/flows", func(w http.ResponseWriter, r *http.Request) {
//	    hub.ServeWS(w, r)
//	})
//
// 客户端连接后将实时收到流量事件：
//
//	const ws = new WebSocket('ws://localhost:8080/ws/flows');
//	ws.onmessage = (event) => {
//	    const flow = JSON.parse(event.data);
//	    console.log('New flow:', flow);
//	};
//
// # 数据生命周期管理
//
// 自动清理过期数据：
//
//	lifecycleConfig := flow.LifecycleConfig{
//	    CleanupInterval:            24 * time.Hour,
//	    RetentionDuration:          7 * 24 * time.Hour,  // 保留7天
//	    StoragePath:                "./data/flows.db",
//	    DiskSpaceThresholdPercent:  80,
//	    EnableDiskMonitoring:       true,
//	}
//
//	lifecycleManager := flow.NewLifecycleManager(storage, lifecycleConfig)
//	lifecycleManager.Start()
//	defer lifecycleManager.Stop()
//
//	// 获取清理统计信息
//	stats := lifecycleManager.GetStats()
//	log.Printf("Cleanup runs: %d, Total deleted: %d",
//	    stats.TotalCleanupRuns, stats.TotalFlowsDeleted)
//
// # 流量事件类型
//
// 流量事件类型包括：
//   - NEW: 新连接建立
//   - UPDATE: 连接活跃/更新
//   - CLOSED: 连接正常关闭
//   - TIMEOUT: 连接超时
//
// # 流量状态
//
// 流量状态包括：
//   - ACTIVE: 活跃流量
//   - CLOSED: 已关闭流量
//   - TIMEOUT: 超时流量
//
// # 策略动作
//
// 策略动作包括：
//   - ALLOW: 允许流量
//   - DENY: 拒绝流量
//   - LOG: 记录流量
//
// # 数据模型
//
// Flow 结构体包含：
//   - 5元组标识（源IP、源端口、目标IP、目标端口、协议）
//   - 流量统计（数据包数、字节数、持续时间）
//   - 时间戳（开始时间、结束时间、最后可见时间）
//   - 工作负载标签（源标签、目标标签）
//   - 策略上下文（策略ID、策略动作）
//   - 状态信息（状态、方向、事件类型）
//
// # SQLite 存储实现
//
// 数据库表结构：
//
//	CREATE TABLE flows (
//	    id TEXT PRIMARY KEY,
//	    source_ip TEXT NOT NULL,
//	    source_port INTEGER NOT NULL,
//	    dest_ip TEXT NOT NULL,
//	    dest_port INTEGER NOT NULL,
//	    protocol TEXT NOT NULL,
//	    packet_count INTEGER,
//	    byte_count INTEGER,
//	    duration_ms INTEGER,
//	    start_time INTEGER,
//	    end_time INTEGER,
//	    last_seen INTEGER,
//	    source_labels TEXT,
//	    dest_labels TEXT,
//	    policy_id INTEGER,
//	    policy_action TEXT,
//	    state TEXT,
//	    direction TEXT,
//	    event_type TEXT
//	);
//
// 支持的索引：
//   - idx_flows_time: (start_time, end_time)
//   - idx_flows_source: (source_ip)
//   - idx_flows_dest: (dest_ip)
//   - idx_flows_protocol: (protocol)
//   - idx_flows_state: (state)
//   - idx_flows_policy_action: (policy_action)
//
// # 性能特性
//
// 流量收集性能：
//   - Ring Buffer 读取：非阻塞模式，超时控制
//   - 批量持久化：默认批次大小 100 条
//   - 内存聚合：基于 5元组的 hashmap
//   - 超时清理：定期扫描（默认 1 分钟）
//
// 存储性能：
//   - WAL 模式：支持并发读写
//   - 批量写入：事务优化
//   - 索引查询：多字段索引加速
//   - 数据清理：定期删除过期数据（默认 7 天）
//
// WebSocket 性能：
//   - 广播模式：支持多客户端
//   - 非阻塞发送：避免慢客户端阻塞
//   - 心跳保活：定期 ping/pong
//   - 自动重连：客户端断线重连
//
// # 线程安全
//
// Collector 使用互斥锁保护：
//   - activeFlows map: RWMutex
//   - metrics: RWMutex
//
// SQLiteStorage 使用：
//   - 数据库连接池（single writer, multiple readers）
//   - 事务保护批量操作
//
// Hub 使用通道通信：
//   - register/unregister channels
//   - broadcast channel
//
// # 工作负载标签富化
//
// 如果配置了 WorkloadManager，Collector 会自动：
//   - 根据源 IP 查找源工作负载标签
//   - 根据目标 IP 查找目标工作负载标签
//   - 将标签信息附加到流量记录
//
// 示例标签：
//
//	{
//	    "app": "nginx",
//	    "env": "prod",
//	    "tier": "frontend"
//	}
//
// # 依赖分析
//
// 支持基于标签的依赖关系分析：
//
//	dependencies, err := storage.GetDependencies(startTime, endTime)
//	for _, dep := range dependencies {
//	    log.Printf("%v -> %v: %d flows, %d bytes",
//	        dep.SourceLabels, dep.DestLabels, dep.FlowCount, dep.ByteCount)
//	}
//
// # 全局聚合分析
//
// Aggregator 提供全局级别的流量分析：
//
//	aggregator := flow.NewAggregator(storage)
//	aggregator.Start()
//	defer aggregator.Stop()
//
//	// 获取聚合统计
//	stats := aggregator.GetAggregatedStats()
//	log.Printf("Global flows: %d, Top protocol: %s",
//	    stats.TotalFlows, stats.TopProtocols[0].Protocol)
//
// # 注意事项
//
// 1. Ring Buffer 大小有限，需要及时读取避免事件丢失
// 2. SQLite 数据库需要定期清理，避免磁盘空间耗尽
// 3. WebSocket 客户端数量会影响广播性能
// 4. 工作负载标签查询需要 WorkloadManager 支持
// 5. 大规模查询建议添加时间范围和分页限制
//
// # 错误处理
//
// 包中的错误分类：
//   - 解析错误: Ring Buffer 数据格式错误
//   - 存储错误: SQLite 数据库操作失败
//   - 网络错误: WebSocket 连接失败
//   - 超时错误: 流量清理超时
//
// 错误处理策略：
//   - 解析错误: 记录日志，跳过该事件，增加 dropped 计数
//   - 存储错误: 记录警告，继续处理（内存数据仍可用）
//   - 网络错误: 自动重连，移除失败客户端
//   - 超时错误: 继续清理，记录错误统计
//
// # 监控指标
//
// Collector 提供的指标：
//   - eventsProcessed: 成功处理的事件数
//   - eventsDropped: 丢弃的事件数
//   - activeFlows: 当前活跃流量数
//
// LifecycleManager 提供的指标：
//   - TotalCleanupRuns: 清理运行次数
//   - TotalFlowsDeleted: 删除的流量总数
//   - CleanupErrors: 清理错误次数
//   - DiskUsagePercent: 磁盘使用率
//
// # 扩展性
//
// 支持自定义存储实现：
//
//	type CustomStorage struct {}
//
//	func (s *CustomStorage) SaveFlow(flow *Flow) error { ... }
//	func (s *CustomStorage) QueryFlows(query *FlowQuery) ([]*Flow, error) { ... }
//	// 实现 Storage 接口的其他方法
//
//	collector := flow.NewCollector(ringBuf, customStorage, nil, config)
//
// 支持自定义工作负载管理器：
//
//	type CustomWorkloadManager struct {}
//
//	func (m *CustomWorkloadManager) GetLabelsByIP(ip string) (map[string]string, bool) {
//	    // 自定义实现
//	}
//
//	collector := flow.NewCollector(ringBuf, storage, customWorkloadMgr, config)
//
// # 未来改进
//
// 计划中的功能：
//   - 支持 IPv6 流量
//   - 流量采样和抽样
//   - 更多聚合维度（时间窗口、标签组合）
//   - 流量异常检测
//   - 导出到外部系统（Prometheus、InfluxDB）
package flow
