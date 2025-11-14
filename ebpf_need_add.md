🎯 需要完善的功能点(按优先级排序)
P0 - 关键缺陷(必须修复)
XDP 流事件推送缺失 ⚠️
位置: xdp_microsegment.bpf.c:194
问题: XDP 创建新会话后未推送到 Ring Buffer
影响: XDP 模式下所有新连接事件丢失,控制平面无法感知
修复难度: 低(复制 TC 的 push_flow_event() 即可)
XDP 字节统计缺失 ⚠️
位置: xdp_microsegment.bpf.c:181-182
问题: bytes_to_server/client 硬编码为 0
影响: 带宽监控不准确
修复: 添加 packet_len = ctx->data_end - ctx->data 计算
P1 - 重要优化
通配符策略性能优化 🔥
问题: 线性扫描最多 100 次,策略多时性能差
建议:
短期:限制通配符数量 < 50
长期:使用 LPM Trie Map 优化 CIDR 匹配
TCP 状态机未完整实现 📊
问题: 定义了 10 个 TCP 状态,但只检测 FIN/RST
影响: 无法精确跟踪连接状态(半关闭、异常等)
需要: 实现完整的 SYN/ACK/FIN 状态转换
会话超时机制缺失 ⏱️
问题: 仅依赖 LRU 自动淘汰,无精确控制
建议: 用户态周期性扫描删除过期会话
P2 - 功能增强
连接关闭事件未上报 📤
问题: 已检测 TCP 关闭,但未推送 FLOW_EVENT_CLOSED 事件
建议: 在 is_tcp_closing() 成功后推送关闭事件
协议支持受限 🌐
❌ IPv6 不支持
❌ VLAN 标签不支持
⚠️ 分片包仅处理首片
TC/XDP 会话表分离 🔄
问题: 同一流量可能创建两个会话(浪费内存)
建议: 考虑 Pinning 共享 session_map(需处理并发)
P3 - 代码质量
代码重复: is_tcp_closing() 在 TC/XDP 中重复
魔法数字: for (i < 100) 应使用 MAX_WILDCARD_LOOP 常量