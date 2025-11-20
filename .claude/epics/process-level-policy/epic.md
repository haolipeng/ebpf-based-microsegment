---
name: process-level-policy
status: in_progress
created: 2025-11-19T11:43:06Z
updated: 2025-11-20T00:00:00Z
progress: 10%
prd: .claude/prds/process-level-policy.md
github: https://github.com/haolipeng/ebpf-based-microsegment/issues/45
---

# Epic: 进程级别策略关联

## Overview

为 eBPF 微隔离系统添加进程级别的流量可见性和策略控制能力。核心技术方案采用 **eBPF tracepoint 主动监控进程生命周期**，在进程创建时立即捕获并缓存进程信息（comm、路径、容器ID），解决短生命周期进程的信息获取问题。支持基于进程名称和路径的混合策略匹配，并从 cgroup 提取容器 ID 实现容器级流量隔离。

**技术亮点**:
- eBPF tracepoint (sched_process_exec) 主动监控，路径解析成功率 > 95%
- 双层缓存架构（eBPF LRU map + 用户态 LRU cache）
- 容器环境原生支持（cgroup → 容器 ID）
- 混合策略匹配（process_name + process_path）
- 安全加固（路径合法性检查 + 可疑行为审计）

## Architecture Decisions

### AD-1: 进程监控方案 - Tracepoint 主动捕获

**决策**: 使用 eBPF tracepoint `tp/sched/sched_process_exec` 主动监控进程执行

**理由**:
- ✅ 在进程创建时立即捕获信息（此时进程一定存在）
- ✅ 解决短生命周期进程（如 curl）路径查询失败问题
- ✅ 性能开销可控（仅监控 exec 事件，比 fork 少很多）
- ✅ 能同时获取 PID、comm、exec_time、cgroup（容器ID）

**技术实现**:
```c
// eBPF tracepoint
SEC("tp/sched/sched_process_exec")
int trace_sched_process_exec(struct trace_event_raw_sched_process_exec *ctx)
{
    // 1. 获取进程信息
    struct process_event event = {
        .pid = bpf_get_current_pid_tgid() >> 32,
        .exec_time = bpf_ktime_get_ns(),
    };
    bpf_get_current_comm(&event.comm, sizeof(event.comm));

    // 2. 提取容器 ID（从 cgroup 路径）
    extract_container_id_from_cgroup(&event.container_id);

    // 3. 缓存到 eBPF map（供 TC/XDP 快速查询）
    bpf_map_update_elem(&process_info_map, &event.pid, &event, BPF_ANY);

    // 4. 发送到用户态（用于完整路径查询）
    bpf_ringbuf_output(&process_events, &event, sizeof(event), 0);

    return 0;
}
```

### AD-2: 双层缓存架构

**决策**: eBPF 内核缓存 + 用户态缓存

**架构**:
```
eBPF 层:
- process_info_map (LRU Hash, 10000 条目)
- 存储: PID → {comm, exec_time, container_id}
- 用途: TC/XDP 快速查询（< 2μs）

用户态 Agent:
- ProcessCache (map + LRU, 20000 条目)
- 存储: PID → {comm, path, container_id, cached_time}
- 用途: 完整进程信息（含路径）
- TTL: 5 分钟
```

**理由**:
- eBPF map 无法存储长路径（256 字节），仅存关键信息
- 用户态缓存提供完整信息，避免重复查询 /proc
- LRU 策略自动淘汰，内存可控（eBPF ~320KB, 用户态 ~2MB）

### AD-3: 策略匹配优先级

**决策**: 多级优先级 + AND 逻辑

**匹配优先级**（从高到低）:
1. **精确匹配**: 进程信息 + 网络 5 元组
2. **进程策略**: 包含 process_name/process_path
3. **网络策略**: 传统 5 元组
4. **默认策略**: DENY (Fail-closed)

**字段组合**: 所有字段必须同时匹配（AND 逻辑）

**示例**:
```yaml
# 策略 1 (高优先级)
- process_name: "nginx"
  dest_port: 3306
  action: "ALLOW"

# 测试:
- nginx → db:3306 → ALLOW (匹配策略 1)
- nginx → redis:6379 → DENY (无匹配，默认拒绝)
- curl → db:3306 → DENY (进程不匹配)
```

### AD-4: 容器 ID 提取

**决策**: Phase 1 从 cgroup 路径提取容器 ID，Phase 2 查询 K8s API

**实现**:
```c
// eBPF 层提取容器 ID
static __always_inline void extract_container_id_from_cgroup(char *container_id)
{
    struct task_struct *task = (struct task_struct *)bpf_get_current_task();

    // 读取 cgroup 路径
    // 示例: /kubepods/besteffort/pod-uuid/container-id
    // 或: /docker/container-id

    // 解析提取最后一段（容器ID）
    // 实现细节：通过 BPF_CORE_READ 读取 task->cgroups
}
```

**理由**:
- Phase 1 实现简单，不依赖外部 API
- 容器 ID 足以提供基础的容器隔离可见性
- Phase 2 再扩展 Pod 名称、命名空间（需要 K8s API）

### AD-5: 复用现有组件

**决策**: 最大化复用现有微隔离系统组件

**复用清单**:
- ✅ **Ring Buffer 机制**: 复用现有 flow_events，仅扩展数据结构
- ✅ **gRPC 框架**: 复用 FlowEvent 消息，增加进程字段
- ✅ **策略管理框架**: 复用 Policy API，扩展支持 process_name/path
- ✅ **Dashboard 流量列表**: 复用现有页面，增加进程列
- ✅ **eBPF 加载框架**: 复用 cilium/ebpf 库和 bpf2go 工具链

**避免重新发明轮子**:
- 不创建新的事件传输机制
- 不创建新的策略下发通道
- 不创建新的 UI 页面（扩展现有页面）

## Technical Approach

### eBPF 数据平面

**核心组件**:

1. **Tracepoint 程序** (新增)
```c
// 文件: src/bpf/process_monitor.bpf.c
SEC("tp/sched/sched_process_exec")
int trace_sched_process_exec(...)

Maps:
- process_info_map: LRU Hash, 10000 entries
- process_events: Ring Buffer, 256KB
```

2. **TC/XDP 程序扩展** (修改)
```c
// 文件: src/bpf/tc_microsegment.bpf.c, xdp_microsegment.bpf.c

新增功能:
- 获取当前进程 PID/comm
- 查询 process_info_map 获取容器 ID
- 进程策略匹配逻辑
- 流量事件增加进程字段
```

3. **数据结构扩展**
```c
// flow_event 扩展
struct flow_event {
    // ... existing fields ...
    char process_name[16];
    __u32 pid;
    char container_id[64];
    __u64 process_exec_time;
};

// process_info_map value
struct process_cache_entry {
    char comm[16];
    __u64 exec_time;
    char container_id[64];
};
```

### Agent 用户态

**核心组件**:

1. **ProcessMonitor** (新增)
```go
// 文件: src/agent/pkg/process/monitor.go

功能:
- 监听 process_events ring buffer
- 收到事件后立即查询 /proc/<pid>/exe
- 缓存到 ProcessCache (map[uint32]ProcessInfo)
- LRU 淘汰 + TTL 清理
```

2. **FlowCollector 扩展** (修改)
```go
// 文件: src/agent/pkg/flow/collector.go

新增:
- 从 flow_event 解析进程字段
- 查询 ProcessCache 获取完整路径
- 如果缓存未命中，fallback 查询 /proc
- 路径合法性检查（安全加固）
```

3. **PolicyManager 扩展** (修改)
```go
// 文件: src/agent/pkg/policy/manager.go

新增:
- 接收进程策略（process_name/process_path）
- 更新 eBPF process_policy_map
```

### Server 后端

**核心组件**:

1. **gRPC API 扩展**
```protobuf
// api/proto/flow/flow.proto

message FlowEvent {
    // ... existing fields ...
    ProcessInfo process_info = 10;
}

message ProcessInfo {
    string name = 1;
    uint32 pid = 2;
    string path = 3;
    string container_id = 4;
    bool path_resolved = 5;  // 路径是否成功解析
}
```

2. **Policy API 扩展**
```protobuf
// api/proto/policy/policy.proto

message Policy {
    // ... existing fields ...
    string process_name = 10;   // 可选
    string process_path = 11;   // 可选
}
```

3. **数据库 Schema 更新**
```sql
-- flow_events 表扩展
ALTER TABLE flow_events ADD COLUMN process_name VARCHAR(16);
ALTER TABLE flow_events ADD COLUMN process_pid INT;
ALTER TABLE flow_events ADD COLUMN process_path VARCHAR(256);
ALTER TABLE flow_events ADD COLUMN container_id VARCHAR(64);

-- policies 表扩展
ALTER TABLE policies ADD COLUMN process_name VARCHAR(16);
ALTER TABLE policies ADD COLUMN process_path VARCHAR(256);

-- 索引优化
CREATE INDEX idx_flow_events_process ON flow_events(process_name, process_path);
```

4. **Security Alert API** (新增)
```go
// API endpoints:
GET /api/v1/security/alerts - 获取安全告警列表
GET /api/v1/security/suspicious-processes - 获取可疑进程列表
```

### Web Dashboard

**核心组件**:

1. **流量列表扩展** (修改)
```typescript
// 文件: web/src/components/FlowList.tsx

新增列:
- Process Name
- Process Path
- Container ID

新增过滤:
- 按进程名称过滤
- 按容器 ID 过滤
```

2. **进程统计图表** (新增)
```typescript
// 文件: web/src/components/ProcessStats.tsx

功能:
- 按进程分组的流量统计
- Top N 进程排行
- 进程流量趋势图
```

3. **安全告警面板** (新增)
```typescript
// 文件: web/src/components/SecurityAlerts.tsx

功能:
- 展示可疑进程列表（/tmp 路径等）
- 告警计数
- 一键阻断功能
```

## Implementation Strategy

### 开发阶段

**Phase 1: 核心功能** (Week 1-9)

遵循 PRD 中的 11 周计划，聚焦核心功能交付。

**Phase 2: 高级特性** (Future)

根据用户反馈决定优先级：
- 进程 Profile 管理（学习模式）
- 进程哈希校验（防篡改）
- K8s Pod 元数据关联

### 风险缓解

1. **性能风险**: Tracepoint 可能影响性能
   - 缓解: 简化版性能测试，确保 CPU < 10%
   - 降级: 提供开关允许禁用进程监控

2. **准确性风险**: PID 重用导致缓存错误
   - 缓解: 记录 exec_time 时间戳，验证一致性
   - 降级: 缓存失效时 fallback 查询 /proc

3. **兼容性风险**: 旧内核不支持 tracepoint
   - 缓解: 检测内核版本 >= 4.18
   - 降级: 旧内核回退到被动查询模式

### 测试策略

**简化版测试**（3 个基本场景）:

1. **功能测试**:
   - 启动 nginx 容器，验证进程信息捕获
   - 配置进程策略，验证策略匹配
   - 测试短生命周期进程（curl）

2. **性能测试**:
   - 对比启用/禁用进程监控的 CPU 使用率
   - 验证 CPU 增加 < 10%

3. **安全测试**:
   - /tmp 路径进程触发告警
   - 进程名称伪装不绕过 path 匹配

## Task Breakdown Preview

以下是将要创建的高层次任务类别（总计 10 个任务，符合简化要求）:

### Core eBPF Development

- [ ] **Task 1: eBPF Tracepoint 和进程监控**
  - 实现 `tp/sched/sched_process_exec` tracepoint 程序
  - 创建 `process_info_map` (LRU Hash) 和 `process_events` (Ring Buffer)
  - 实现容器 ID 提取（从 cgroup 路径）
  - 扩展 `flow_event` 数据结构
  - **估算**: 2 周

- [ ] **Task 2: TC/XDP 程序进程策略匹配**
  - 获取当前进程 PID/comm
  - 查询 `process_info_map` 获取缓存信息
  - 实现进程策略匹配逻辑（process_name + process_path）
  - 实现策略优先级（进程策略 > 网络策略）
  - **估算**: 1.5 周

### Agent Development

- [ ] **Task 3: ProcessMonitor 组件实现**
  - 监听 `process_events` ring buffer
  - 实现进程路径查询（/proc/<pid>/exe）
  - 实现用户态进程缓存（LRU + TTL）
  - **估算**: 1.5 周

- [ ] **Task 4: FlowCollector 和 PolicyManager 扩展**
  - FlowCollector 解析进程字段
  - 查询 ProcessCache 获取完整信息
  - PolicyManager 支持进程策略下发
  - **估算**: 1.5 周

- [ ] **Task 5: 安全加固实现**
  - 进程路径合法性检查（系统目录 vs 可疑目录）
  - 可疑进程识别和标记
  - 安全告警生成
  - **估算**: 1 周

### Backend Development

- [ ] **Task 6: gRPC API 和数据模型扩展**
  - 扩展 FlowEvent 消息（ProcessInfo）
  - 扩展 Policy 消息（process_name/process_path）
  - 数据库 schema 更新
  - **估算**: 1 周

- [ ] **Task 7: 进程策略 API 实现**
  - 进程策略 CRUD API
  - 策略验证（字段长度、合法性）
  - 安全告警 API
  - **估算**: 1 周

### Frontend Development

- [ ] **Task 8: Dashboard 进程可见性**
  - 流量列表增加进程列（Name, Path, Container ID）
  - 进程过滤和搜索
  - 进程统计图表
  - **估算**: 1 周

- [ ] **Task 9: 安全告警面板**
  - 可疑进程列表展示
  - 告警计数和通知
  - 一键阻断功能
  - **估算**: 0.5 周

### Testing & Documentation

- [ ] **Task 10: 集成测试、性能测试和文档**
  - 端到端功能测试（进程监控、策略匹配、容器ID）
  - 简化版性能测试（3 个基本场景）
  - 安全测试（路径伪装、进程欺骗）
  - 用户手册和 API 文档
  - **估算**: 2 周

**总估算**: 11 周（与 PRD 一致）

## Dependencies

### External Dependencies

| Dependency | Version | Purpose | Risk |
|-----------|---------|---------|------|
| Linux Kernel | >= 4.18 | Tracepoint 支持 | Medium - 旧内核需降级 |
| BTF | Enabled | eBPF CO-RE | Medium - 部分发行版默认关闭 |
| cilium/ebpf | Latest | eBPF 加载库 | Low |
| Docker/containerd | >= 19.03/1.4 | 容器运行时 | Low |

### Internal Dependencies

| Component | Dependency | Owner | Status |
|-----------|-----------|-------|--------|
| eBPF Dataplane | Ring Buffer 支持 | eBPF Team | ✅ 已完成 |
| Agent | Flow Collector | Agent Team | ✅ 已完成 |
| Server | gRPC 框架 | Backend Team | ✅ 已完成 |
| Dashboard | 流量列表页面 | Frontend Team | ✅ 已完成 |

### Prerequisite Work

无前置依赖，可立即开始实施。

## Success Criteria (Technical)

### 功能验收

- ✅ Tracepoint 能成功捕获所有进程执行事件
- ✅ 进程路径解析成功率 > 90%
- ✅ 容器进程能正确提取容器 ID
- ✅ 进程策略匹配准确率 100%
- ✅ Dashboard 实时显示进程信息

### 性能指标

- ✅ CPU 额外开销 < 10%
- ✅ 网络延迟无明显增加（肉眼无感知）
- ✅ eBPF map lookup < 2μs
- ✅ 内存增长 < 5MB

### 质量门禁

- ✅ 所有单元测试通过
- ✅ 端到端集成测试通过
- ✅ 性能测试通过（3 个基本场景）
- ✅ 安全测试通过（无绕过漏洞）
- ✅ 代码评审完成

## Estimated Effort

### 总体时间线

- **Phase 1 (MVP)**: 11 weeks
- **关键路径**: eBPF 开发 → Agent 开发 → 集成测试

### 资源需求

- **eBPF 工程师**: 1 人，全职 5 周（Task 1-2）
- **后端工程师**: 1 人，全职 5 周（Task 3-7）
- **前端工程师**: 0.5 人，全职 1.5 周（Task 8-9）
- **QA 工程师**: 0.5 人，全职 2 周（Task 10）

### 风险缓冲

- 预留 1 周缓冲时间应对意外问题
- 关键路径监控，及时调整资源分配

## Tasks Created

- [x] #46 - eBPF Tracepoint 和进程监控 (parallel: true) ✅ **已完成 (68h/80h)**
- [ ] #47 - TC/XDP 程序进程策略匹配 (parallel: false, depends: [46])
- [ ] #48 - ProcessMonitor 组件实现 (parallel: false, depends: [46])
- [ ] #49 - FlowCollector 和 PolicyManager 扩展 (parallel: false, depends: [48])
- [ ] #50 - 安全加固实现 (parallel: true, depends: [48])
- [ ] #51 - gRPC API 和数据模型扩展 (parallel: true, depends: [46])
- [ ] #52 - 进程策略 API 实现 (parallel: false, depends: [51])
- [ ] #53 - Dashboard 进程可见性 (parallel: true, depends: [52])
- [ ] #54 - 安全告警面板 (parallel: true, depends: [52])
- [ ] #55 - 集成测试、性能测试和文档 (parallel: false, depends: [46, 47, 48, 49, 50, 51, 52, 53, 54])

**任务统计**:
- 总任务数: 10
- 可并行任务: 5
- 顺序任务: 5
- GitHub Issues: #46 - #55
---

**创建时间**: 2025-11-19T11:43:06Z
**任务分解时间**: 2025-11-19T12:30:26Z
**PRD 来源**: .claude/prds/process-level-policy.md
**下一步**: 运行 `/pm:epic-sync process-level-policy` 同步到 GitHub Issues
