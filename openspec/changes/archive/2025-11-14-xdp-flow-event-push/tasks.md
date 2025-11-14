# Tasks: XDP 流事件推送实现

## 实现任务 (Implementation)

### Task 1: 添加 XDP 数据包长度计算
- [ ] 在 `xdp_microsegment.bpf.c` 的新会话创建前计算数据包长度
- [ ] 添加代码: `__u32 packet_len = (void *)(long)ctx->data_end - (void *)(long)ctx->data;`
- [ ] 修改 `new_session.bytes_to_server = packet_len` (替换原来的 0)
- [ ] 在现有会话更新路径添加字节累加逻辑
- **验证**: 编译通过,无 eBPF 验证器错误

### Task 2: 实现 push_flow_event_xdp() 辅助函数
- [ ] 在 `xdp_microsegment.bpf.c` 添加函数定义 (参考 TC 实现)
- [ ] 实现 Ring Buffer reserve 逻辑
- [ ] 实现事件字段填充 (5-tuple, 统计, 策略上下文)
- [ ] 实现 Ring Buffer submit 逻辑
- [ ] 添加错误处理 (reserve 失败返回 -1)
- [ ] 添加 DEBUG_MODE 下的调试日志
- **验证**: 编译通过,函数签名与规范一致

### Task 3: 在会话创建后调用事件推送
- [ ] 找到 `bpf_map_update_elem(&session_map, ...)` 成功的位置
- [ ] 在 `ret == 0` 分支中添加 `push_flow_event_xdp()` 调用
- [ ] 传递正确的参数: key, timestamp, packet_count=1, byte_count, event_type=FLOW_EVENT_NEW
- [ ] 设置 direction=POLICY_DIR_INGRESS (XDP 固定值)
- [ ] 删除原有的 TODO 注释
- **验证**: 编译通过,逻辑正确

### Task 4: 代码审查和优化
- [ ] 检查所有指针边界检查是否充分
- [ ] 确认 `__always_inline` 标记存在
- [ ] 检查 padding 和 reserved 字段是否清零
- [ ] 对比 TC 实现,确保字段顺序和类型一致
- [ ] 添加代码注释说明 XDP 特定行为
- **验证**: 代码审查通过

## 测试任务 (Testing)

### Task 5: 编译验证
- [ ] 运行 `make bpf` 编译 eBPF 代码
- [ ] 检查无编译错误
- [ ] 检查无 eBPF 验证器错误
- [ ] 检查生成的 .o 文件大小合理 (约 200KB)
- **验证**: 编译成功

### Task 6: 功能测试 - 事件推送验证
- [ ] 配置 Agent 使用 XDP 模式
- [ ] 启动 Agent: `sudo ./bin/microsegment-agent -c config/agent-xdp.yaml`
- [ ] 生成测试流量: `curl http://test-server`
- [ ] 检查 Agent 日志中是否有 "Flow event received" 消息
- [ ] 验证日志中的 5-tuple 与测试流量一致
- [ ] 验证 `byte_count` 不为 0
- [ ] 验证 `event_type` 为 FLOW_EVENT_NEW
- **验证**: 事件成功接收,字段正确

### Task 7: 功能测试 - 多连接场景
- [ ] 生成多个并发连接 (如 10 个 HTTP 请求)
- [ ] 验证接收到对应数量的流事件
- [ ] 检查每个事件的 5-tuple 唯一
- [ ] 验证无事件丢失 (Ring Buffer 无满)
- **验证**: 多连接场景正常

### Task 8: 性能测试
- [ ] 使用 `iperf3` 生成高流量 (1Gbps+)
- [ ] 测量 XDP 程序的延迟 (通过 bpf_printk 时间戳)
- [ ] 对比修改前后的延迟差异
- [ ] 确认延迟增加 <5%
- [ ] 检查 Ring Buffer 丢包率 (通过统计)
- **验证**: 性能无明显回归

### Task 9: 对比测试 - TC vs XDP
- [ ] 运行 TC 模式 Agent,收集 100 个流事件
- [ ] 运行 XDP 模式 Agent,收集 100 个流事件
- [ ] 对比两组事件的字段完整性
- [ ] 验证 XDP 事件的 direction 始终为 INGRESS
- [ ] 验证其他字段格式一致
- **验证**: XDP 和 TC 事件格式一致

## 文档任务 (Documentation)

### Task 10: 更新代码注释
- [ ] 在 `push_flow_event_xdp()` 添加函数注释
- [ ] 说明 XDP 特定行为 (仅 ingress, 字节统计计算方式)
- [ ] 更新 `xdp_microsegment.bpf.c` 文件头注释
- [ ] 删除完成的 TODO 标记
- **验证**: 注释完整,清晰

### Task 11: 更新 CONSTANTS_SYNC.md
- [ ] 在 `src/agent/pkg/dataplane/CONSTANTS_SYNC.md` 中添加流事件相关说明
- [ ] 记录 XDP 和 TC 事件推送的区别
- [ ] 更新验证清单包含流事件检查
- **验证**: 文档更新完整

## 集成任务 (Integration)

### Task 12: 验证 Go Flow Collector 集成
- [ ] 检查 `src/agent/pkg/flow/collector.go` 是否需要修改
- [ ] 验证 Collector 能正确解析 XDP 事件
- [ ] 测试 Collector 的错误处理 (无效事件)
- [ ] 验证事件持久化到 SQLite
- **验证**: Flow Collector 无需修改,正常工作

### Task 13: 端到端验证
- [ ] 启动完整系统 (PostgreSQL, Server, Agent with XDP)
- [ ] 生成测试流量
- [ ] 验证流事件在 Web UI 中可见
- [ ] 验证流统计数据正确 (packet_count, byte_count)
- [ ] 验证策略上下文 (policy_id, action) 正确
- **验证**: 端到端流程正常

## 清理任务 (Cleanup)

### Task 14: 代码清理和格式化
- [ ] 运行 `make fmt` 格式化代码
- [ ] 删除调试用的 bpf_printk (如果有)
- [ ] 检查无未使用的变量或函数
- [ ] 统一代码风格 (与 TC 程序一致)
- **验证**: 代码整洁

### Task 15: 最终验证
- [ ] 重新编译整个项目: `make clean && make all`
- [ ] 运行所有单元测试: `make test`
- [ ] 运行集成测试 (如果有)
- [ ] 验证所有测试通过
- **验证**: 全部测试通过

## 任务依赖关系

```
Task 1 (字节计算) ─→ Task 2 (函数实现) ─→ Task 3 (调用推送)
                           ↓
                        Task 4 (审查) ─→ Task 5 (编译)
                                             ↓
                   Task 6-9 (功能和性能测试)
                                             ↓
                   Task 10-11 (文档更新)
                                             ↓
                   Task 12-13 (集成验证)
                                             ↓
                   Task 14-15 (清理和最终验证)
```

## 并行执行机会

- Task 10-11 (文档) 可与 Task 6-9 (测试) 并行
- Task 4 (审查) 可与 Task 1-3 (实现) 交叉进行

## 估算工作量

- **实现任务 (Task 1-4)**: 2-3 小时
- **测试任务 (Task 5-9)**: 3-4 小时
- **文档任务 (Task 10-11)**: 1 小时
- **集成任务 (Task 12-13)**: 1-2 小时
- **清理任务 (Task 14-15)**: 1 小时

**总计**: 8-11 小时 (约 1-1.5 个工作日)
