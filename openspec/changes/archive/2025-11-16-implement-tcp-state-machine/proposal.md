# 提案: 实现完整的 TCP 状态机追踪

## 元数据

- **变更 ID**: `implement-tcp-state-machine`
- **标题**: 在 eBPF 层实现完整的 TCP 状态机追踪
- **状态**: 提案中
- **创建日期**: 2025-01-14
- **作者**: System
- **相关能力**: `session-tracking`, `policy-enforcement`

## 概述

当前系统在 `session_value` 结构中包含 `tcp_state` 字段，并在 `common_types.h` 中定义了完整的 TCP 状态枚举，但实际的状态转换逻辑尚未完全实现。本提案旨在在 eBPF 数据平面层实现完整、符合 RFC 793 的 TCP 状态机，为以下功能提供基础：

1. **精确的连接生命周期追踪**: 准确识别 TCP 连接的每个阶段
2. **更智能的会话管理**: 基于 TCP 状态的会话超时和清理
3. **安全策略增强**: 支持基于连接状态的策略（例如，仅允许已建立的连接）
4. **异常检测**: 识别异常的 TCP 行为（例如，无 SYN 的数据传输）
5. **准确的连接关闭检测**: 区分正常关闭（FIN）和异常重置（RST）

## 动机

### 当前限制

当前实现存在以下限制：

1. **简单的状态跟踪**: 仅区分 NEW、ESTABLISHED、CLOSING、CLOSED 四种会话状态
2. **不完整的 TCP 感知**: `tcp_state` 字段已定义但未充分利用
4. **缺少状态验证**: 无法检测无效的状态转换（例如，跳过握手的数据包）

### 为什么现在需要

1. **生产就绪性**: 完整的 TCP 状态机是生产级网络安全系统的基本要求
2. **安全增强**: 能够实施更细粒度的安全策略（例如，阻止半开连接）
3. **可观测性**: 提供更详细的连接状态信息用于监控和故障排除
4. **与现有规范对齐**: `session-tracking` 规范已定义了 TCP 状态机要求，但实现不完整

## 目标

### 主要目标

1. **实现标准 TCP 状态机**: 实现 RFC 793 定义的 11 种状态转换
2. **TCP 标志处理**: 正确解析和处理 SYN、ACK、FIN、RST 等标志
3. **双向状态追踪**: 分别追踪客户端和服务器方向的状态转换
4. **状态验证**: 检测并处理无效的状态转换
5. **会话生命周期集成**: 根据 TCP 状态更新会话状态（SESSION_STATE_*）

### 次要目标

1. **性能优化**: 确保状态机逻辑不增加显著的处理延迟（< 0.5 μs）
2. **调试支持**: 添加可选的调试日志以追踪状态转换
3. **统计信息**: 添加 TCP 状态相关的统计计数器
4. **文档和测试**: 提供完整的状态转换图和测试用例

## 非目标

1. **TCP 序列号追踪**: 不实现完整的序列号验证（复杂度高，性能影响大）
2. **TCP 窗口管理**: 不处理滑动窗口和流量控制
3. **TCP 重传检测**: 不追踪重传数据包
4. **应用层协议识别**: 仅关注 TCP 状态，不解析应用数据

## 提案细节

### TCP 状态定义

实现 RFC 793 定义的 11 种 TCP 状态：

```c
enum tcp_state {
    TCP_STATE_CLOSED = 0,      // 无连接
    TCP_STATE_LISTEN,          // 被动打开，等待连接请求
    TCP_STATE_SYN_SENT,        // 主动打开，已发送 SYN
    TCP_STATE_SYN_RECV,        // 收到 SYN，已发送 SYN+ACK
    TCP_STATE_ESTABLISHED,     // 连接已建立，可以传输数据
    TCP_STATE_FIN_WAIT_1,      // 主动关闭，已发送 FIN
    TCP_STATE_FIN_WAIT_2,      // 主动关闭，收到对方 ACK
    TCP_STATE_CLOSE_WAIT,      // 被动关闭，收到 FIN
    TCP_STATE_CLOSING,         // 同时关闭，双方都发送 FIN
    TCP_STATE_LAST_ACK,        // 被动关闭，发送 FIN 等待最后 ACK
    TCP_STATE_TIME_WAIT,       // 主动关闭方等待 2MSL
};
```

### 状态转换逻辑

**客户端（主动打开）流程**:
```
CLOSED -> SYN_SENT (发送 SYN)
         -> ESTABLISHED (收到 SYN+ACK，发送 ACK)
         -> FIN_WAIT_1 (发送 FIN)
         -> FIN_WAIT_2 (收到 ACK)
         -> TIME_WAIT (收到 FIN，发送 ACK)
         -> CLOSED (等待 2MSL 后)
```

**服务器（被动打开）流程**:
```
CLOSED -> LISTEN (被动打开)
       -> SYN_RECV (收到 SYN，发送 SYN+ACK)
       -> ESTABLISHED (收到 ACK)
       -> CLOSE_WAIT (收到 FIN，发送 ACK)
       -> LAST_ACK (发送 FIN)
       -> CLOSED (收到 ACK)
```

**异常情况处理**:
- **收到 RST**: 任何状态 -> CLOSED
- **超时**: 某些状态（如 SYN_SENT）-> CLOSED
- **无效转换**: 保持当前状态，可选记录日志

### 实现方式

1. **核心函数**: `update_tcp_state(struct session_value *session, struct tcphdr *tcp, bool is_client_to_server)`
2. **调用位置**: 在会话查找/更新逻辑中，每次处理 TCP 数据包时调用
3. **优化**: 使用内联函数和快速路径优化，最小化分支预测失败

### 与会话状态的映射

| TCP 状态 | 会话状态 (SESSION_STATE_*) |
|---------|---------------------------|
| CLOSED, LISTEN | N/A (无会话) |
| SYN_SENT, SYN_RECV | NEW |
| ESTABLISHED | ESTABLISHED |
| FIN_WAIT_1, FIN_WAIT_2, CLOSE_WAIT, CLOSING, LAST_ACK | CLOSING |
| TIME_WAIT, CLOSED (连接结束后) | CLOSED |

## 影响的能力

### 修改的能力

1. **session-tracking**: 扩展 TCP 状态追踪要求
   - 添加完整状态转换场景
   - 增强双向状态追踪要求
   - 添加异常状态处理要求

2. **policy-enforcement**: 增强基于状态的策略执行
   - 支持基于 TCP 状态的策略匹配（未来扩展）
   - 改进连接关闭检测的准确性

### 新增的能力

**tcp-state-machine**: 新增独立的 TCP 状态机能力
   - 定义完整的状态转换要求
   - 定义 TCP 标志处理要求
   - 定义双向状态追踪要求
   - 定义异常处理要求

## 实施策略

### 阶段 1: 核心状态机实现 (Week 1)

1. 创建 `tcp_state_machine.h` 头文件
2. 实现 `update_tcp_state()` 核心函数
3. 实现 TCP 标志提取辅助函数
4. 添加基本的状态转换测试

### 阶段 2: 集成和优化 (Week 1-2)

1. 集成到 TC 和 XDP 程序
2. 性能基准测试和优化
3. 添加调试日志（可选编译）
4. 更新统计计数器

### 阶段 3: 测试和文档 (Week 2)

1. 单元测试（覆盖所有状态转换）
2. 集成测试（真实 TCP 连接）
3. 压力测试（高并发场景）
4. 更新文档和示例

## 风险和缓解

### 风险 1: 性能影响

**风险**: TCP 状态机逻辑可能增加数据包处理延迟

**缓解**:
- 使用内联函数避免函数调用开销
- 实施快速路径优化（ESTABLISHED 状态跳过大部分检查）
- 基准测试确保延迟增加 < 0.5 μs

### 风险 2: eBPF 验证器复杂性

**风险**: 复杂的状态机逻辑可能导致 eBPF 验证器失败

**缓解**:
- 限制嵌套深度和循环
- 使用明确的边界检查
- 测试多个内核版本（5.10, 5.15, 6.1, 6.6）

### 风险 3: 边缘情况处理

**风险**: 真实网络中的异常 TCP 行为可能导致状态不一致

**缓解**:
- 保守的状态转换（遇到无效转换时保持当前状态）
- 添加异常计数器用于监控
- 超时机制清理卡住的会话

## 成功标准

### 功能标准

- [ ] 通过所有状态转换单元测试（覆盖 11 种状态）
- [ ] 通过真实 TCP 连接的集成测试（建立、传输、关闭）
- [ ] 正确处理异常情况（RST、超时、无效转换）
- [ ] 准确映射到会话状态

### 性能标准

- [ ] 热路径延迟增加 < 0.5 μs
- [ ] 无数据包丢弃（100K pps 压力测试）
- [ ] eBPF 验证器通过（内核 5.10+）

### 文档标准

- [ ] 完整的状态转换图
- [ ] TCP 标志处理说明
- [ ] 示例代码和使用说明
- [ ] 故障排除指南

## 时间线

- **Week 1**: 核心实现和基本测试
- **Week 2**: 集成、优化和完整测试
- **Week 3**: 文档和代码审查（缓冲）

总计: ~2-3 周

## 依赖关系

### 前置条件

- 现有的会话追踪基础设施
- 现有的 TCP 数据包解析逻辑

### 阻塞的能力

- 基于状态的策略匹配（未来功能）
- 高级异常检测（未来功能）

## 参考

- [RFC 793 - Transmission Control Protocol](https://www.rfc-editor.org/rfc/rfc793)
- [Linux Kernel TCP 实现](https://github.com/torvalds/linux/blob/master/include/net/tcp_states.h)
- [ZFW TCP State Tracking](https://github.com/netfoundry/zfw)
- [Cilium Connection Tracking](https://docs.cilium.io/en/stable/concepts/ebpf/maps/#connection-tracking)

## 开放问题

1. **是否需要追踪 LISTEN 状态**?
   - eBPF 在数据平面运行，通常看不到被动打开的 LISTEN 状态
   - 建议: 跳过 LISTEN 状态，从 SYN_RECV 开始追踪服务器端

2. **TIME_WAIT 状态的超时处理**?
   - 标准 2MSL (Maximum Segment Lifetime) 是 2 分钟
   - 建议: 在用户空间处理超时清理，eBPF 仅负责状态转换

3. **如何处理乱序数据包**?
   - eBPF 看到的是原始数据包，可能乱序
   - 建议: 基于标志位的状态转换，容忍一定程度的乱序

## 下一步

1. 审查提案并获得反馈
2. 创建详细的设计文档 (`design.md`)
3. 实现核心状态机逻辑
4. 编写单元测试和集成测试
5. 性能基准测试和优化
