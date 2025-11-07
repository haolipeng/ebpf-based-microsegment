# flow-management 规范 (ADDED)

## 目的
本规范定义了网络流（Flow）数据收集、存储和查询系统的需求。Flow 系统用于支持应用依赖地图（ADM）可视化、流量分析和审计功能。

## 变更类型
**ADDED** - 新增 Flow 管理功能规范

## ADDED Requirements

### Requirement: Flow 数据收集（Flow Data Collection）
系统 SHALL 从 eBPF 数据平面收集网络流事件。

#### Scenario: Ring Buffer 事件推送
- **WHEN** eBPF 程序检测到新连接建立（NEW）
- **THEN** 必须通过 Ring Buffer 推送 FLOW_NEW 事件到用户空间
- **AND** 事件必须包含 5-tuple（src_ip, dst_ip, src_port, dst_port, protocol）
- **AND** 事件必须包含初始统计信息（packet_count, byte_count）

#### Scenario: 连接关闭事件
- **WHEN** eBPF 程序检测到连接关闭（TCP FIN/RST）
- **THEN** 必须推送 FLOW_CLOSED 事件
- **AND** 事件必须包含最终统计信息

#### Scenario: Flow 事件结构验证
- **THEN** Flow 事件必须是固定大小的 48 字节 packed 结构体
- **AND** 必须包含时间戳（timestamp_ns）
- **AND** 必须包含策略上下文（policy_id, policy_action）

#### Scenario: Ring Buffer 容量
- **THEN** Ring Buffer 默认大小必须为 256KB
- **AND** 必须支持通过编译参数配置大小
- **AND** 当 Ring Buffer 满时必须增加丢弃计数器

---

### Requirement: Flow Collector（Flow 收集器）
系统 SHALL 实现 Go 用户空间 Flow Collector 读取和处理流事件。

#### Scenario: Ring Buffer 读取
- **WHEN** Flow Collector 启动时
- **THEN** 必须打开 eBPF Ring Buffer reader
- **AND** 必须持续读取事件直到停止信号

#### Scenario: 事件解析
- **WHEN** 从 Ring Buffer 读取原始事件数据时
- **THEN** 必须解析为 FlowEvent 结构体
- **AND** 必须验证事件大小（48 字节）
- **AND** 必须将网络字节序 IP 地址转换为字符串

#### Scenario: 标签增强
- **WHEN** 解析 Flow 事件后
- **THEN** 必须查询 WorkloadManager 获取源 IP 的标签
- **AND** 必须查询目标 IP 的标签
- **AND** 必须将标签附加到 Flow 对象

#### Scenario: Collector 错误处理
- **WHEN** Ring Buffer 读取失败时
- **THEN** 必须记录错误日志
- **AND** 必须继续处理后续事件（不崩溃）
- **WHEN** Ring Buffer 关闭时
- **THEN** 必须优雅退出收集循环

---

### Requirement: Flow 持久化（Flow Persistence）
系统 SHALL 将收集的 Flow 数据持久化到存储层。

#### Scenario: SQLite 存储
- **THEN** 必须实现 SQLite 作为默认存储后端
- **AND** 必须创建 `flows` 表
- **AND** 表必须包含所有 Flow 字段

#### Scenario: Flow 表模式
- **THEN** flows 表必须包含以下列：
  - id (TEXT PRIMARY KEY, UUID)
  - source_ip, source_port, dest_ip, dest_port, protocol
  - packet_count, byte_count, duration
  - start_time, end_time, last_seen (DATETIME)
  - source_labels, dest_labels (JSON TEXT)
  - policy_id, policy_action, state, direction

#### Scenario: 数据库索引
- **THEN** 必须创建索引 idx_flows_start_time (start_time)
- **AND** 必须创建索引 idx_flows_source_ip (source_ip)
- **AND** 必须创建索引 idx_flows_dest_ip (dest_ip)
- **AND** 必须创建索引 idx_flows_protocol (protocol)

#### Scenario: Flow 插入
- **WHEN** 收集到新 Flow 时
- **THEN** 必须插入到 flows 表
- **AND** 插入必须在 <10ms 内完成

---

### Requirement: Flow 查询 API（Flow Query API）
系统 SHALL 提供 REST API 用于查询 Flow 数据。

#### Scenario: 列出 Flows
- **WHEN** GET /api/v1/flows
- **THEN** 必须返回 200 OK
- **AND** 必须返回 JSON 响应包含 flows 数组
- **AND** 必须支持分页（limit, offset）
- **AND** 默认 limit 为 100，最大 1000

#### Scenario: 时间范围过滤
- **WHEN** GET /api/v1/flows?start_time=2025-11-04T00:00:00Z&end_time=2025-11-04T23:59:59Z
- **THEN** 必须仅返回此时间范围内的 flows
- **AND** 默认 start_time 为 1 小时前
- **AND** 默认 end_time 为当前时间

#### Scenario: IP 过滤
- **WHEN** GET /api/v1/flows?source_ip=10.0.1.10
- **THEN** 必须仅返回 source_ip=10.0.1.10 的 flows
- **WHEN** GET /api/v1/flows?dest_ip=10.0.2.20
- **THEN** 必须仅返回 dest_ip=10.0.2.20 的 flows

#### Scenario: 协议过滤
- **WHEN** GET /api/v1/flows?protocol=TCP
- **THEN** 必须仅返回 TCP flows
- **AND** 支持的协议：TCP, UDP, ICMP

#### Scenario: 状态过滤
- **WHEN** GET /api/v1/flows?state=ACTIVE
- **THEN** 必须仅返回状态为 ACTIVE 的 flows
- **AND** 支持的状态：ACTIVE, CLOSED, TIMEOUT

#### Scenario: 排序
- **WHEN** GET /api/v1/flows?order_by=byte_count&order=desc
- **THEN** 必须按 byte_count 降序返回
- **AND** 支持的排序字段：start_time, byte_count, packet_count
- **AND** 默认排序：start_time DESC

#### Scenario: 查询性能
- **GIVEN** 数据库中有 10,000 条 flow 记录
- **WHEN** 查询 1000 条记录时
- **THEN** API 响应时间必须 <100ms

---

### Requirement: Flow Summary API（Flow 统计摘要 API）
系统 SHALL 提供聚合统计信息 API。

#### Scenario: Flow 统计摘要
- **WHEN** GET /api/v1/flows/summary
- **THEN** 必须返回 200 OK
- **AND** 必须包含以下字段：
  - total_flows (总流数量)
  - total_packets (总包数)
  - total_bytes (总字节数)
  - active_flows (活跃流数量)
  - closed_flows (已关闭流数量)
  - flows_by_protocol (按协议分组)
  - flows_by_action (按动作分组)

#### Scenario: Summary 时间范围
- **WHEN** GET /api/v1/flows/summary?start_time=...&end_time=...
- **THEN** 统计必须仅包含时间范围内的 flows
- **AND** 默认时间范围为过去 1 小时

#### Scenario: Summary 性能
- **GIVEN** 数据库中有 50,000 条 flow 记录
- **WHEN** 计算 summary 时
- **THEN** 响应时间必须 <50ms

---

### Requirement: 应用依赖关系 API（Application Dependencies API）
系统 SHALL 提供工作负载间依赖关系数据。

#### Scenario: 依赖关系查询
- **WHEN** GET /api/v1/flows/dependencies
- **THEN** 必须返回 200 OK
- **AND** 必须包含 nodes 数组（工作负载节点）
- **AND** 必须包含 edges 数组（通信关系）

#### Scenario: 依赖关系节点格式
- **THEN** 每个 node 必须包含：
  - id (唯一标识符)
  - labels (工作负载标签 map)

#### Scenario: 依赖关系边格式
- **THEN** 每个 edge 必须包含：
  - source (源节点 ID)
  - target (目标节点 ID)
  - flow_count (流数量)
  - total_bytes (总字节数)
  - total_packets (总包数)

#### Scenario: 按标签分组
- **WHEN** GET /api/v1/flows/dependencies?group_by=app,role
- **THEN** 必须按 app 和 role 标签分组工作负载
- **AND** 默认 group_by 为 "app,role"

#### Scenario: 依赖关系聚合
- **GIVEN** 多个 flows 在同一工作负载对之间
- **WHEN** 计算依赖关系时
- **THEN** 必须聚合为单个 edge
- **AND** 必须累加 flow_count, total_bytes, total_packets

---

### Requirement: Top Talkers API
系统 SHALL 提供流量 Top N 分析 API。

#### Scenario: Top Talkers 查询
- **WHEN** GET /api/v1/flows/top-talkers
- **THEN** 必须返回 200 OK
- **AND** 必须返回按字节数排序的 top 10 IP
- **AND** 默认 n=10，最大 n=100

#### Scenario: Top 源 IP
- **WHEN** GET /api/v1/flows/top-talkers?by=source
- **THEN** 必须返回按 total_bytes 排序的 top N 源 IP

#### Scenario: Top 目标 IP
- **WHEN** GET /api/v1/flows/top-talkers?by=destination
- **THEN** 必须返回按 total_bytes 排序的 top N 目标 IP

#### Scenario: Top Talker 格式
- **THEN** 每个 top talker 必须包含：
  - ip (IP 地址)
  - labels (工作负载标签)
  - total_bytes (总字节数)
  - total_flows (总流数量)

---

### Requirement: 实时 Flow 推送（Real-time Flow Streaming）
系统 SHALL 支持通过 WebSocket 实时推送 Flow 事件。

#### Scenario: WebSocket 连接
- **WHEN** 客户端连接 ws://host/api/v1/flows/stream
- **THEN** 必须成功建立 WebSocket 连接
- **AND** 连接必须注册到 WebSocket Hub

#### Scenario: 实时 Flow 广播
- **WHEN** Collector 收集到新 Flow 时
- **THEN** 必须广播到所有连接的 WebSocket 客户端
- **AND** 广播延迟必须 <500ms

#### Scenario: Flow 消息格式
- **THEN** WebSocket 消息必须为 JSON 格式
- **AND** 必须包含 type="flow"
- **AND** 必须包含 data 字段（完整 Flow 对象）

#### Scenario: 客户端断开
- **WHEN** WebSocket 客户端断开时
- **THEN** 必须从 Hub 注销
- **AND** 必须清理连接资源
- **AND** 不得影响其他客户端

#### Scenario: 心跳机制
- **THEN** 必须每 30 秒发送 ping 消息
- **AND** 必须期望 pong 响应
- **WHEN** 连续 3 次 ping 无响应时
- **THEN** 必须关闭连接

---

### Requirement: Flow 过滤订阅（Flow Filter Subscription）
系统 SHALL 支持客户端订阅特定 Flow 类型。

#### Scenario: 订阅协议过滤
- **WHEN** 客户端发送 JSON 消息 {"action":"subscribe", "filters":{"protocol":"TCP"}}
- **THEN** 客户端必须仅接收 TCP flows
- **AND** UDP/ICMP flows 必须被过滤掉

#### Scenario: 订阅标签过滤
- **WHEN** 客户端订阅 {"filters":{"source_labels":{"app":"nginx"}}}
- **THEN** 客户端必须仅接收源标签包含 app=nginx 的 flows

#### Scenario: 取消订阅
- **WHEN** 客户端发送 {"action":"unsubscribe"}
- **THEN** 必须接收所有 flows（无过滤）

---

### Requirement: Flow 数据生命周期管理（Flow Data Lifecycle）
系统 SHALL 管理 Flow 数据的生命周期。

#### Scenario: 自动清理旧 Flows
- **THEN** 必须每小时运行清理任务
- **AND** 必须删除超过 7 天的 flows
- **AND** 保留期必须可配置

#### Scenario: 清理任务日志
- **WHEN** 清理任务执行时
- **THEN** 必须记录删除的 flow 数量
- **AND** 必须记录清理耗时

#### Scenario: 磁盘空间监控
- **WHEN** 数据库文件大小超过阈值（如 10GB）时
- **THEN** 必须记录警告日志
- **AND** 应当触发提前清理

---

### Requirement: Flow API 错误处理（Flow API Error Handling）
系统 SHALL 提供清晰的错误响应。

#### Scenario: 无效查询参数
- **WHEN** GET /api/v1/flows?limit=999999
- **THEN** 必须返回 400 Bad Request
- **AND** 错误消息必须说明 limit 最大为 1000

#### Scenario: 无效时间格式
- **WHEN** GET /api/v1/flows?start_time=invalid
- **THEN** 必须返回 400 Bad Request
- **AND** 错误消息必须说明正确的时间格式（ISO8601）

#### Scenario: 存储错误
- **WHEN** 数据库查询失败时
- **THEN** 必须返回 500 Internal Server Error
- **AND** 必须记录详细错误日志
- **AND** 不得暴露敏感信息给客户端

---

### Requirement: Flow 性能监控（Flow Performance Monitoring）
系统 SHALL 提供性能指标。

#### Scenario: Prometheus 指标
- **THEN** 必须暴露以下 Prometheus 指标：
  - flow_events_total (按 event_type, protocol 分组)
  - flow_collector_errors_total (按 error_type 分组)
  - flow_storage_latency_seconds (按 operation 分组)
  - websocket_clients_connected (gauge)
  - ring_buffer_events_dropped_total

#### Scenario: Ring Buffer 丢弃监控
- **WHEN** Ring Buffer 满导致事件丢弃时
- **THEN** 必须增加 ring_buffer_events_dropped_total 计数器
- **AND** 必须记录警告日志

#### Scenario: Collector 性能
- **THEN** Collector 必须能够处理 ≥10,000 flows/s
- **AND** CPU 使用率应当 <20%（单核）

---

### Requirement: Flow 标签完整性（Flow Label Integrity）
系统 SHALL 确保 Flow 与工作负载标签的一致性。

#### Scenario: 标签查询失败处理
- **WHEN** WorkloadManager.GetLabelsByIP() 返回错误时
- **THEN** Flow 的 source_labels 或 dest_labels 必须为空 map
- **AND** 必须继续处理 Flow（不丢弃）

#### Scenario: 标签 JSON 序列化
- **WHEN** 保存 Flow 到数据库时
- **THEN** 标签 map 必须序列化为 JSON 字符串
- **WHEN** 从数据库读取时
- **THEN** 必须反序列化为 map[string]string

#### Scenario: 空标签处理
- **WHEN** 工作负载没有标签时
- **THEN** source_labels/dest_labels 必须为空 map（不是 null）
- **AND** JSON 响应中必须为 {}

---

### Requirement: Flow 查询分页（Flow Query Pagination）
系统 SHALL 支持高效分页查询。

#### Scenario: Offset 分页
- **WHEN** GET /api/v1/flows?limit=100&offset=0
- **THEN** 必须返回第 1-100 条记录
- **WHEN** GET /api/v1/flows?limit=100&offset=100
- **THEN** 必须返回第 101-200 条记录

#### Scenario: 总数返回
- **THEN** API 响应必须包含 total 字段（总记录数）
- **AND** total 必须反映过滤后的记录数

#### Scenario: Page 参数
- **THEN** API 响应应当包含 page 字段（当前页码）
- **AND** page = offset / limit + 1

---

### Requirement: Flow 并发控制（Flow Concurrency Control）
系统 SHALL 安全处理并发操作。

#### Scenario: 并发 Flow 插入
- **WHEN** Collector 并发插入多个 Flows 时
- **THEN** 所有插入必须成功
- **AND** 不得出现数据竞争

#### Scenario: 并发查询和插入
- **WHEN** API 查询和 Collector 插入同时进行时
- **THEN** 查询必须返回一致的数据
- **AND** 不得出现死锁

#### Scenario: WebSocket 并发广播
- **WHEN** 多个 Flows 快速到达时
- **THEN** WebSocket Hub 必须正确广播所有 Flows
- **AND** 不得丢失消息（除非客户端缓冲区满）

---

### Requirement: Flow 溯源（Flow Provenance）
系统 SHALL 记录 Flow 的来源和上下文。

#### Scenario: 策略 ID 记录
- **WHEN** Flow 匹配策略时
- **THEN** 必须记录 policy_id
- **AND** 必须记录 policy_action (ALLOW/DENY)

#### Scenario: 方向记录
- **THEN** 必须记录 direction (INGRESS/EGRESS)
- **AND** INGRESS 表示入站流量
- **AND** EGRESS 表示出站流量

#### Scenario: 持续时间计算
- **WHEN** Flow 关闭时
- **THEN** 必须计算 duration = end_time - start_time
- **AND** duration 必须以毫秒为单位

---

### Requirement: Flow API 安全性（Flow API Security）
系统 SHALL 实施 API 访问控制（未来增强）。

#### Scenario: API 认证（未来）
- **THEN** Flow API 应当支持 Bearer Token 认证
- **AND** 未认证请求应当返回 401 Unauthorized

#### Scenario: 查询限制（当前）
- **THEN** 单次查询必须限制最大 1000 条记录
- **AND** 时间范围查询必须限制最大 24 小时

#### Scenario: 敏感数据保护
- **THEN** Flow 数据不得包含数据包内容
- **AND** 仅包含元数据（5-tuple, 统计信息）
