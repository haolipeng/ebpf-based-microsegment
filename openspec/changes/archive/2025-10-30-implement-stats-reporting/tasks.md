# 实施任务

## 1. 统计基础设施
- [x] 1.1 定义 stats_key 枚举（8 个计数器）
- [x] 1.2 为统计创建 PERCPU_ARRAY 映射
- [x] 1.3 实现 update_stats() 辅助函数
- [x] 1.4 在整个数据路径中添加统计调用

## 2. 统计类型
- [x] 2.1 STATS_TOTAL_PACKETS 计数器
- [x] 2.2 STATS_ALLOWED_PACKETS 计数器
- [x] 2.3 STATS_DENIED_PACKETS 计数器
- [x] 2.4 STATS_NEW_SESSIONS 计数器
- [x] 2.5 STATS_CLOSED_SESSIONS 计数器
- [x] 2.6 STATS_ACTIVE_SESSIONS 计数器
- [x] 2.7 STATS_POLICY_HITS 计数器
- [x] 2.8 STATS_POLICY_MISSES 计数器

## 3. 流事件报告
- [x] 3.1 定义 flow_event 结构
- [x] 3.2 创建 RINGBUF 映射（256KB）
- [x] 3.3 为新会话实现事件生成
- [x] 3.4 选择性报告（仅 DENY/LOG）
- [x] 3.5 包含 5 元组、时间戳、操作

## 4. 用户空间统计
- [x] 4.1 在 DataPlane 中实现 GetStatistics()
- [x] 4.2 读取每 CPU 数组
- [x] 4.3 聚合跨 CPU 的值
- [x] 4.4 返回 Statistics 结构

## 5. 用户空间事件监控
- [x] 5.1 实现 MonitorFlowEvents() goroutine
- [x] 5.2 从环形缓冲区读取
- [x] 5.3 解析 flow_event 结构
- [x] 5.4 使用结构化格式记录事件
- [x] 5.5 处理优雅关闭

## 6. 性能优化
- [x] 6.1 对每 CPU 统计使用直接递增（无原子）
- [x] 6.2 最小化环形缓冲区使用（减少 99%）
- [x] 6.3 非阻塞事件提交

## 7. 测试和验证
- [x] 7.1 验证统计准确性
- [x] 7.2 测试事件生成
- [x] 7.3 验证每 CPU 聚合
- [x] 7.4 性能影响测试

**状态**：所有任务于 2025-10-30 完成

