---
name: 进程级别策略关联
description: 为微隔离系统添加进程级别的流量可见性和策略控制能力，优先支持容器环境
status: backlog
created: 2025-11-19T10:47:49Z
---

# PRD: 进程级别策略关联

## Executive Summary

为 eBPF 微隔离系统增加进程级别的流量关联能力，使系统能够识别每个网络连接是由哪个进程发起的，并支持基于进程的策略控制。该功能采用分阶段实现策略：

- **Phase 1 (MVP)**: 进程信息可见性 + 基于进程名称/路径的策略 + 容器 ID 关联
- **Phase 2 (Future)**: 完整的进程 Profile 管理（学习模式、强制模式、哈希校验）+ K8s Pod 元数据

**核心技术方案**（经评审确认）：

1. **进程生命周期监控**: 使用 eBPF tracepoint 主动监控进程执行（sched_process_exec），在进程创建时立即捕获并缓存进程信息，解决短生命周期进程的路径解析问题
2. **混合匹配策略**: 支持基于进程名称（comm）和完整路径（path）两种匹配模式
3. **容器环境集成**: 从 cgroup 提取容器 ID，提供容器级别的流量隔离和可见性
4. **安全加固**: 自动检查进程路径合法性 + 可疑行为审计告警

**实施周期**: 11 周

## Problem Statement

### 当前限制

当前微隔离系统仅基于网络 5 元组（源 IP、目标 IP、源端口、目标端口、协议）进行策略匹配，存在以下问题：

1. **缺乏应用层可见性**
   - 无法识别产生流量的具体进程/应用
   - 容器环境中多个进程共享同一 IP，难以区分
   - 无法实现基于应用的细粒度访问控制

2. **安全盲区**
   - 恶意进程可以伪装成合法 IP 发起连接
   - 容器逃逸后的异常进程行为难以发现
   - 缺少进程级别的审计和追溯能力

3. **运维困难**
   - 故障排查时无法快速定位问题进程
   - 无法实现"只允许 nginx 访问数据库"这类策略
   - 缺少进程级别的流量统计和分析

### 为什么现在做

- **容器化趋势**: Kubernetes 环境中，应用级别的访问控制需求强烈
- **零信任架构**: 需要基于进程的细粒度身份识别
- **合规要求**: 审计日志需要记录具体的进程信息
- **技术可行**: eBPF 技术成熟，可以低开销获取进程信息

## User Stories

### 主要用户画像

#### 1. 安全管理员 - 王工
- **目标**: 监控和审计所有网络流量的进程来源
- **痛点**: 只看到 IP 和端口，无法判断是否为正常应用
- **期望**: 在 Dashboard 中看到每个连接对应的进程名称和路径

#### 2. 运维工程师 - 李工
- **目标**: 实现应用级别的网络访问控制
- **痛点**: 无法配置"只允许 API 服务访问数据库"的策略
- **期望**: 能够基于进程名称配置策略规则

#### 3. 开发人员 - 张工
- **目标**: 快速排查网络连接问题
- **痛点**: 流量异常时不知道是哪个进程在捣乱
- **期望**: 实时看到进程级别的流量监控

### 详细用户旅程

#### Journey 1: 流量可见性
```
场景: 王工发现某个 Pod 有异常外联流量
1. 打开 Dashboard，查看该 Pod 的流量列表
2. 看到流量事件中包含进程信息：
   - 进程名称: curl
   - 进程路径: /usr/bin/curl
   - PID: 12345
   - 目标: malicious-domain.com:443
3. 识别出这是一个异常进程发起的连接
4. 立即配置策略阻断该进程
```

**Acceptance Criteria**:
- ✅ 流量事件包含进程名称、路径、PID
- ✅ Dashboard 实时显示进程信息
- ✅ 能够快速定位异常进程

#### Journey 2: 进程级别策略
```
场景: 李工需要配置"只允许 nginx 访问 Redis"
1. 打开策略配置页面
2. 创建新策略：
   - 源进程名称: nginx
   - 目标服务: redis:6379
   - 协议: TCP
   - 动作: ALLOW
3. 保存策略，策略下发到 Agent
4. Agent 在 eBPF 层匹配进程名称
5. 其他进程（如 curl）访问 Redis 被拒绝
```

**Acceptance Criteria**:
- ✅ 支持基于进程名称的策略配置
- ✅ eBPF 层正确匹配进程信息
- ✅ 非匹配进程被正确拒绝

#### Journey 3: 故障排查
```
场景: 张工发现服务响应慢，怀疑有异常连接
1. 查看流量监控，按进程名称分组
2. 发现某个不熟悉的进程占用大量连接：
   - 进程: suspicious-miner
   - 连接数: 5000+
   - 目标: mining-pool.com
3. 立即识别为挖矿程序
4. 配置策略阻断该进程的所有外联
```

**Acceptance Criteria**:
- ✅ 支持按进程分组的流量统计
- ✅ 能够快速识别异常进程
- ✅ 一键阻断恶意进程

## Requirements

### Functional Requirements - Phase 1 (MVP)

#### FR-1: 进程生命周期监控与信息捕获
- **Priority**: P0
- **Description**: 使用 eBPF tracepoint 主动监控进程执行，建立进程信息缓存
- **Details**:
  - **Tracepoint 监控**: 挂载 `tp/sched/sched_process_exec`，监听进程执行事件
  - **立即捕获**: 进程执行时立即获取 PID、comm、执行时间戳
  - **eBPF 缓存**: 存储到 `process_info_map` (LRU Hash, 10000 条目)
  - **用户态缓存**: Agent 收到事件后立即查询 `/proc/<pid>/exe` 获取完整路径并缓存
  - **网络事件关联**: TC/XDP 处理网络包时，通过 PID 查询 eBPF map 和用户态缓存
  - **容器 ID 提取**: 从 task_struct 的 cgroup 路径提取容器 ID（如果是容器进程）
- **Acceptance Criteria**:
  - Tracepoint 能捕获所有进程执行事件
  - 进程路径解析成功率 > 95%（含短生命周期进程）
  - eBPF map 性能开销 < 2μs per lookup
  - 用户态缓存命中率 > 90%

#### FR-2: 流量事件扩展
- **Priority**: P0
- **Description**: 扩展 flow_event 数据结构，包含进程和容器信息
- **Details**:
  ```c
  struct flow_event {
      // ... existing fields ...
      char process_name[16];     // 进程名称（comm）
      __u32 pid;                 // 进程 ID
      char process_path[256];    // 进程路径（用户态填充）
      char container_id[64];     // 容器 ID（从 cgroup 提取）
      __u64 process_exec_time;   // 进程启动时间戳（用于验证 PID 重用）
  };
  ```
- **Acceptance Criteria**:
  - Ring buffer 能正确传递进程和容器信息
  - Agent 能解析所有新增字段
  - 数据结构向后兼容
  - 容器进程能正确提取容器 ID

#### FR-3: 进程信息上报
- **Priority**: P0
- **Description**: Agent 将进程信息上报到 Server
- **Details**:
  - 扩展 gRPC FlowEvent 消息包含进程字段
  - Agent 通过 `/proc/<pid>/exe` 补充完整路径
  - 处理进程已退出的情况（路径为空）
- **Acceptance Criteria**:
  - Server 能接收到进程信息
  - 路径解析成功率 > 95%
  - 处理边界情况（PID 重用、进程退出）

#### FR-4: Dashboard 进程可见性
- **Priority**: P0
- **Description**: Web UI 显示进程级别的流量信息
- **Details**:
  - 流量列表增加"进程"列
  - 显示进程名称和路径
  - 支持按进程名称过滤
  - 进程级别流量统计图表
- **Acceptance Criteria**:
  - 流量列表实时显示进程信息
  - 过滤和搜索功能正常
  - 图表准确展示进程流量分布

#### FR-5: 基于进程名称的策略
- **Priority**: P0
- **Description**: 支持配置基于进程名称的简单策略
- **Details**:
  - 策略配置增加 `process_name` 字段（可选）
  - 支持精确匹配进程名称
  - 策略匹配逻辑：先匹配进程，再匹配网络 5 元组
  - 策略示例：
    ```yaml
    - process_name: "nginx"
      dest_ip: "10.0.0.5"
      dest_port: 3306
      protocol: "TCP"
      action: "ALLOW"
    ```
- **Acceptance Criteria**:
  - 策略配置 API 支持 process_name 字段
  - eBPF 层正确匹配进程名称
  - 匹配失败时正确拒绝连接
  - 策略优先级：进程策略 > 网络策略

#### FR-6: 进程策略管理 API
- **Priority**: P1
- **Description**: Server 提供进程策略的 CRUD API
- **Details**:
  - `POST /api/v1/policies` - 创建进程策略
  - `GET /api/v1/policies` - 查询策略（支持进程过滤）
  - `PUT /api/v1/policies/:id` - 更新策略
  - `DELETE /api/v1/policies/:id` - 删除策略
- **Acceptance Criteria**:
  - API 正确处理 process_name/process_path 字段
  - 策略验证（进程名称长度限制）
  - 策略下发到 Agent 正常工作

#### FR-7: 进程路径策略匹配（混合模式）
- **Priority**: P0
- **Description**: 支持基于进程名称和完整路径的混合匹配
- **Details**:
  - 策略可以指定 `process_name`（简单匹配，精确到 15 字符）
  - 策略可以指定 `process_path`（精确匹配完整路径）
  - 两种模式可以单独使用或组合使用
  - 策略示例：
    ```yaml
    # 模式 1: 仅进程名称
    - process_name: "nginx"
      action: "ALLOW"

    # 模式 2: 完整路径（更精确）
    - process_path: "/usr/sbin/nginx"
      action: "ALLOW"

    # 模式 3: 组合（最严格）
    - process_name: "nginx"
      process_path: "/usr/sbin/nginx"
      dest_port: 3306
      action: "ALLOW"
    ```
- **Acceptance Criteria**:
  - 两种匹配模式都能正确工作
  - 路径匹配能区分同名进程（如 /usr/bin/python vs /tmp/python）

#### FR-8: 策略匹配优先级与组合规则
- **Priority**: P0
- **Description**: 明确定义多策略场景下的匹配逻辑
- **Details**:
  - **匹配优先级**（从高到低）:
    1. 精确匹配：进程信息 + 网络 5 元组
    2. 进程策略：包含 process_name/process_path 的策略
    3. 网络策略：传统 5 元组策略
    4. 默认策略：DENY (Fail-closed)
  - **字段组合逻辑**: 策略中所有字段必须**同时匹配**（AND 逻辑）
  - **具体度优先**: 字段越多的策略优先级越高
- **Acceptance Criteria**:
  - 策略匹配逻辑文档化
  - 集成测试覆盖所有匹配场景
  - 冲突策略有明确的决策结果

#### FR-9: 安全加固措施
- **Priority**: P1
- **Description**: 自动检查进程路径合法性并审计可疑行为
- **Details**:
  - **路径合法性检查**（用户态）:
    - 信任系统目录：/usr/bin, /usr/sbin, /bin, /sbin, /usr/local/bin
    - 可疑目录标记：/tmp, /var/tmp, /dev/shm
    - 记录路径检查结果到流量事件
  - **可疑行为审计**:
    - 自动告警可疑路径的进程（如 /tmp/suspicious）
    - Dashboard 展示可疑进程列表
    - 提供告警通知机制
  - **不强制阻断**: Phase 1 仅审计，不自动拒绝
- **Acceptance Criteria**:
  - 能识别并标记可疑路径的进程
  - Dashboard 显示安全告警
  - 不会误判合法应用

### Functional Requirements - Phase 2 (Future Scope)

#### FR-10: 进程 Profile 管理
- **Priority**: P2
- **Description**: 支持进程白名单/黑名单管理
- **Details**:
  - 为每个服务组配置进程 Profile
  - 支持路径通配符（`/usr/bin/*`, `/opt/app/*`）
  - 学习模式：自动记录容器内运行的进程
  - 强制模式：阻断不在白名单中的进程

#### FR-11: 进程哈希校验
- **Priority**: P2
- **Description**: 支持基于二进制哈希的防篡改
- **Details**:
  - 记录进程二进制文件的 SHA256 哈希
  - 运行时校验进程哈希是否匹配
  - 检测进程替换攻击

#### FR-12: 完整容器元数据关联
- **Priority**: P2
- **Description**: 关联 K8s Pod 信息（名称、命名空间、标签）
- **Details**:
  - 基于容器 ID 查询容器运行时 API
  - 或者查询 Kubernetes API 获取 Pod 信息
  - 在流量事件中记录 Pod 名称、命名空间、标签
  - 支持基于 Pod/Namespace 的策略

### Non-Functional Requirements

#### NFR-1: 性能（简化版）
- **Metric**: 性能下降可接受，无明显延迟
- **Rationale**: MVP 阶段聚焦功能验证
- **Measurement**:
  - 对比启用/禁用进程关联的性能
  - CPU 增加 < 10%
  - 网络延迟肉眼无感知

#### NFR-2: 准确性
- **Metric**: 进程路径解析成功率 > 90%（通过进程监控提升）
- **Rationale**: 进程生命周期监控能捕获短生命周期进程
- **Measurement**:
  - 对比 eBPF 进程事件和流量事件
  - 统计路径解析成功/失败比例

#### NFR-3: 资源开销（放宽要求）
- **Metric**: 内存增长 < 5MB, CPU 可接受
- **Rationale**: MVP 阶段优先保证功能，后续优化
- **Measurement**:
  - 监控 Agent 内存占用
  - eBPF map 大小（process_info_map ~320KB）

#### NFR-4: 兼容性
- **Metric**: 支持 Linux kernel >= 4.18（需要 tracepoint 支持）
- **Rationale**: 进程生命周期监控需要 sched_process_exec tracepoint
- **Measurement**: 多内核版本测试

#### NFR-5: 可扩展性
- **Metric**: 支持 10000+ 并发进程
- **Rationale**: LRU map 自动淘汰，内存可控
- **Measurement**: 容器环境压力测试

## Technical Architecture

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        User Space                            │
├─────────────────────────────────────────────────────────────┤
│  Web Dashboard                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Flow List: [Process Name] [Path] [Src] [Dst] [Action]│  │
│  │ Process Stats: nginx(1000), curl(50), ssh(10)         │  │
│  └──────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│  Server (gRPC API)                                           │
│  - 接收 Agent 上报的进程关联流量                              │
│  - 存储进程策略配置                                           │
│  - 下发进程策略到 Agent                                       │
├─────────────────────────────────────────────────────────────┤
│  Agent                                                       │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Flow Collector                                        │  │
│  │ - 从 ring buffer 读取流量事件                         │  │
│  │ - 解析进程信息（process_name, pid）                   │  │
│  │ - 通过 /proc/<pid>/exe 获取进程路径                   │  │
│  │ - 上报到 Server                                       │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ Policy Manager                                        │  │
│  │ - 接收 Server 下发的进程策略                          │  │
│  │ - 更新 eBPF process_policy_map                        │  │
│  └──────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│                      Kernel Space (eBPF)                     │
├─────────────────────────────────────────────────────────────┤
│  TC/XDP Program                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 1. Extract 5-tuple from packet                        │  │
│  │ 2. bpf_get_current_comm() -> process_name             │  │
│  │ 3. bpf_get_current_task() -> pid                      │  │
│  │ 4. Lookup process_policy_map(process_name)            │  │
│  │ 5. If match: apply process policy                     │  │
│  │ 6. Else: fallback to network policy                   │  │
│  │ 7. Emit flow event with process info to ring buffer   │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ eBPF Maps                                             │  │
│  │ - process_policy_map: (process_name) -> policy        │  │
│  │ - flow_events (ring buffer): enhanced with process    │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Data Structures

#### eBPF Data Structures

```c
// Process policy key (simple version - MVP)
struct process_policy_key {
    char process_name[16];  // From bpf_get_current_comm()
};

// Process policy value
struct process_policy_value {
    __u8 action;           // ALLOW or DENY
    __u32 policy_id;       // Policy ID for tracking
    __u64 timestamp;       // Last update time
};

// Enhanced flow event
struct flow_event {
    // Existing fields
    __u32 src_ip;
    __u32 dst_ip;
    __u16 src_port;
    __u16 dst_port;
    __u8 protocol;
    __u8 event_type;
    __u8 policy_action;
    __u64 packet_count;
    __u64 byte_count;

    // New process fields
    char process_name[16];  // Process comm name
    __u32 pid;              // Process ID
    __u32 tid;              // Thread ID
    __u32 uid;              // User ID (future)
    __u32 gid;              // Group ID (future)
};
```

#### Server/Agent Data Structures

```protobuf
// gRPC message
message FlowEvent {
    // Existing fields
    string source_ip = 1;
    string dest_ip = 2;
    uint32 source_port = 3;
    uint32 dest_port = 4;
    string protocol = 5;
    string event_type = 6;
    string policy_action = 7;

    // New process fields
    ProcessInfo process_info = 8;
}

message ProcessInfo {
    string name = 1;        // Process comm name (max 15 chars)
    uint32 pid = 2;         // Process ID
    string path = 3;        // Full executable path (from /proc/<pid>/exe)
    uint32 uid = 4;         // User ID
    uint32 gid = 5;         // Group ID
}

message ProcessPolicy {
    string process_name = 1;   // Process name to match
    string dest_ip = 2;        // Destination IP (optional)
    uint32 dest_port = 3;      // Destination port (optional)
    string protocol = 4;       // Protocol (optional)
    string action = 5;         // ALLOW or DENY
    uint32 policy_id = 6;      // Unique policy ID
}
```

### Implementation Details

#### eBPF Helper Functions

```c
// Get current process name
static __always_inline void get_process_info(
    struct flow_event *event)
{
    // Get process comm (max 16 bytes, null-terminated)
    bpf_get_current_comm(&event->process_name, sizeof(event->process_name));

    // Get current task struct
    struct task_struct *task = (struct task_struct *)bpf_get_current_task();
    if (!task)
        return;

    // Extract PID and TID
    event->pid = BPF_CORE_READ(task, tgid);  // Thread group ID (process ID)
    event->tid = BPF_CORE_READ(task, pid);   // Thread ID

    // Optional: Get UID/GID (Phase 2)
    // event->uid = BPF_CORE_READ(task, real_cred, uid.val);
    // event->gid = BPF_CORE_READ(task, real_cred, gid.val);
}
```

#### Process Policy Matching

```c
static __always_inline __u8 match_process_policy(
    struct flow_event *event,
    struct flow_key *key)
{
    // 1. Build process policy key
    struct process_policy_key proc_key = {};
    __builtin_memcpy(proc_key.process_name, event->process_name, 16);

    // 2. Lookup process policy
    struct process_policy_value *policy =
        bpf_map_lookup_elem(&process_policy_map, &proc_key);

    if (policy) {
        // 3. Process policy exists, apply it
        event->policy_action = policy->action;
        return policy->action;
    }

    // 4. No process policy, fallback to network policy
    return match_network_policy(key);
}
```

#### Agent: Process Path Resolution

```go
// Resolve process path from PID
func (c *Collector) resolveProcessPath(pid uint32) string {
    exePath := fmt.Sprintf("/proc/%d/exe", pid)

    // Read symbolic link
    path, err := os.Readlink(exePath)
    if err != nil {
        // Process may have exited
        log.Debugf("Failed to resolve path for PID %d: %v", pid, err)
        return ""
    }

    return path
}

// Enhance flow event with process path
func (c *Collector) processFlowEvent(event *flow.Event) {
    if event.ProcessInfo.Pid > 0 {
        path := c.resolveProcessPath(event.ProcessInfo.Pid)
        event.ProcessInfo.Path = path
    }
}
```

### Container Support

#### Docker/Kubernetes Integration

```c
// Get container ID from cgroup (Phase 2)
static __always_inline void get_container_id(
    struct flow_event *event)
{
    struct task_struct *task = (struct task_struct *)bpf_get_current_task();
    if (!task)
        return;

    // Read cgroup path
    // /kubepods/besteffort/pod<uuid>/<container-id>
    // Extract container ID from cgroup path
    // This will be implemented in Phase 2
}
```

## Success Criteria

### Quantitative Metrics

| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| 进程信息捕获率 | > 95% | 对比 eBPF 事件和 /proc 扫描 |
| 性能开销 | < 5% CPU | 性能基准测试 |
| 延迟增加 | < 10μs | bpf_ktime_get_ns() 测量 |
| 策略匹配准确率 | > 99% | 功能测试 |
| 内存增长 | < 10MB | 资源监控 |

### Qualitative Criteria

- ✅ Dashboard 能直观显示进程级别流量
- ✅ 进程策略配置简单易用
- ✅ 日志和审计包含完整进程信息
- ✅ 用户反馈满意度 > 8/10

### MVP Definition of Done

Phase 1 完成标准：
- [x] eBPF 程序能捕获进程名称和 PID
- [x] 流量事件包含进程信息
- [x] Agent 能解析进程路径
- [x] Server 能接收和存储进程信息
- [x] Dashboard 显示进程列表和统计
- [x] 支持基于进程名称的简单策略
- [x] 策略能正确匹配和执行
- [x] 通过集成测试
- [x] 文档完成（用户手册、API 文档）

## Constraints & Assumptions

### Technical Constraints

1. **Kernel Version**
   - 需要 Linux kernel >= 4.18
   - 需要 BTF (BPF Type Format) 支持
   - 旧内核需要降级方案（仅记录 PID，不获取路径）

2. **eBPF Limitations**
   - `bpf_get_current_comm()` 返回最多 16 字节
   - 无法在 eBPF 中直接读取 `/proc` 文件系统
   - 进程路径需要在用户态补充

3. **Container Runtime**
   - Docker: 通过 cgroup 提取容器 ID
   - containerd: 支持
   - CRI-O: 支持
   - 其他运行时可能需要适配

### Performance Constraints

1. **高频场景**
   - 每秒 100k+ 新连接时，性能开销需控制在 5% 以内
   - 进程信息获取不能成为瓶颈

2. **内存限制**
   - Agent 内存增长 < 10MB
   - eBPF map 大小需合理设置

### Business Constraints

1. **优先级**
   - Phase 1 需在 Q1 2026 完成
   - Phase 2 根据用户反馈决定优先级

2. **资源**
   - 需要 1 名 eBPF 工程师
   - 需要 1 名后端工程师
   - 需要 0.5 名前端工程师

### Assumptions

1. **用户环境**
   - 假设用户主要在容器环境部署
   - 假设用户有基本的 eBPF 知识

2. **进程行为**
   - 假设大部分进程生命周期 > 1 秒（足够查询路径）
   - 假设进程名称有一定辨识度

3. **策略需求**
   - 假设用户主要需要基于进程名称的简单策略
   - 复杂的路径匹配需求较少（Phase 2）

## Out of Scope

明确 **不在** 本 PRD 范围内的功能：

### Phase 1 不包含

1. **进程行为分析**
   - 不追踪进程的文件访问
   - 不监控进程的系统调用
   - 不分析进程的内存行为

2. **复杂匹配规则**
   - 不支持进程路径通配符（`/usr/bin/*`）
   - 不支持正则表达式匹配
   - 不支持进程参数匹配

3. **高级安全特性**
   - 不做进程哈希校验
   - 不做进程签名验证
   - 不检测进程注入攻击

4. **跨主机追踪**
   - 不追踪进程跨主机的通信
   - 不关联分布式进程

5. **进程家族树**
   - 不记录父进程信息
   - 不分析进程调用链
   - 不展示进程树状图

### Phase 2 可能包含

以下功能在 Phase 2 考虑：

1. 进程 Profile 管理（学习模式、强制模式）
2. 进程路径通配符匹配
3. 进程哈希校验（防篡改）
4. 容器元数据关联（Pod、Namespace）
5. 父进程信息记录

## Dependencies

### External Dependencies

| Dependency | Version | Purpose | Risk |
|-----------|---------|---------|------|
| Linux Kernel | >= 4.18 | eBPF helper functions | 旧内核不支持 |
| BTF | Enabled | CO-RE support | 部分发行版默认关闭 |
| Docker | >= 19.03 | 容器运行时 | 低 |
| Kubernetes | >= 1.18 | 容器编排 | 低 |
| containerd | >= 1.4 | 容器运行时 | 低 |

### Internal Dependencies

| Component | Required Feature | Owner | Status |
|-----------|-----------------|-------|--------|
| eBPF Dataplane | Ring buffer 支持 | eBPF Team | ✅ 已完成 |
| Agent | Flow collector | Agent Team | ✅ 已完成 |
| Server | gRPC API | Backend Team | ✅ 已完成 |
| Web UI | 流量列表页面 | Frontend Team | ✅ 已完成 |
| Database | FlowEvent 表 | Backend Team | 🔄 需扩展 |

### Team Dependencies

- **eBPF Team**: 实现 eBPF 层进程信息捕获
- **Agent Team**: 实现进程路径解析和上报
- **Backend Team**: 实现进程策略 API
- **Frontend Team**: 实现进程可见性 UI
- **QA Team**: 编写测试用例，进行集成测试

## Implementation Phases

### Phase 1: MVP (Q1 2026)

**Timeline**: 11 weeks（评审后更新）

**Week 1-3: eBPF 层实现**
- 实现 tracepoint 监控（sched_process_exec）
- 实现进程信息捕获（comm, PID, exec_time）
- 实现容器 ID 提取（从 cgroup 路径）
- 创建 process_info_map (LRU Hash)
- 扩展 flow_event 数据结构
- 实现基于进程名称/路径的策略匹配
- 实现策略优先级逻辑
- 单元测试

**Week 4-6: Agent 层实现**
- 实现 ProcessMonitor 组件（监听进程事件）
- 实现用户态进程缓存（LRU + TTL）
- 扩展 Flow Collector 解析进程信息
- 实现进程路径解析（/proc/pid/exe）
- 实现安全检查（路径合法性验证）
- 实现进程策略管理
- 单元测试

**Week 7-8: Server 和 API**
- 扩展 gRPC API 支持进程和容器字段
- 实现进程策略 CRUD API（支持 process_name/process_path）
- 数据库 schema 更新
- 实现安全告警 API
- API 测试

**Week 9: Web UI**
- 流量列表增加进程和容器列
- 进程/容器过滤和搜索
- 进程流量统计图表
- 安全告警面板（可疑进程）
- UI 测试

**Week 10-11: 集成测试和文档**
- 端到端测试（进程监控、策略匹配、容器ID）
- 性能测试（简化版：3 个基本场景）
- 安全测试（路径伪装、进程欺骗）
- 用户手册
- API 文档

**Deliverables**:
- ✅ eBPF 进程生命周期监控
- ✅ 进程信息缓存（eBPF + 用户态）
- ✅ 容器 ID 关联
- ✅ Dashboard 显示进程和容器流量
- ✅ 基于进程名称/路径的混合策略
- ✅ 安全加固（路径检查 + 审计）
- ✅ 测试报告
- ✅ 用户文档

### Phase 2: 高级特性 (Q2 2026)

**Timeline**: TBD (根据用户反馈)

**Features**:
- 进程 Profile 管理
- 学习模式和强制模式
- 进程路径通配符
- 进程哈希校验
- 容器元数据关联

## Risks & Mitigation

### Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| 旧内核不支持 BPF helpers | Medium | High | 提供降级方案（仅 PID） |
| 进程路径查询失败率高 | Medium | Medium | 缓存进程信息，容忍部分失败 |
| 性能开销超预期 | Low | High | 优化 eBPF 代码，减少不必要的查询 |
| 容器 ID 提取失败 | Medium | Low | 提供手动配置选项 |

### Product Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| 用户不需要进程策略 | Low | Medium | 先做可见性，验证需求 |
| 策略配置过于复杂 | Medium | Medium | 提供模板和向导 |
| 性能影响用户接受度 | Low | High | 严格的性能测试和优化 |

### Operational Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| 升级兼容性问题 | Low | High | 向后兼容的数据结构 |
| 文档不足影响使用 | Medium | Medium | 提前准备详细文档 |
| 支持成本高 | Medium | Medium | 提供故障排查指南 |

## Open Questions

需要进一步讨论和决策的问题：

1. **进程名称冲突**
   - 如何处理同名进程（如多个 python 进程）？
   - 是否需要支持 PID 维度的策略？
   - **建议**: Phase 1 仅支持进程名称，Phase 2 考虑更细粒度

2. **短生命周期进程**
   - 如何处理秒级甚至毫秒级的进程？
   - 路径查询失败时如何处理？
   - **建议**: 尽力而为（best effort），容忍部分失败

3. **策略优先级**
   - 进程策略和网络策略如何组合？
   - 冲突时如何决策？
   - **建议**: 进程策略优先，无匹配时回退到网络策略

4. **容器内进程**
   - 是否需要区分宿主机进程和容器进程？
   - 如何关联 Pod/Container 信息？
   - **建议**: Phase 1 不区分，Phase 2 添加容器元数据

5. **性能优化**
   - 是否需要进程信息缓存？
   - 缓存失效策略是什么？
   - **建议**: 暂不缓存，通过测试确定是否需要

## Appendix

### Reference Materials

- [NeuVector Process Profile Documentation](https://open-docs.neuvector.com/policy/processrules)
- [eBPF Helper bpf_get_current_comm()](https://man7.org/linux/man-pages/man7/bpf-helpers.7.html)
- [Linux /proc filesystem](https://www.kernel.org/doc/html/latest/filesystems/proc.html)
- [Container Runtime Interface (CRI)](https://kubernetes.io/docs/concepts/architecture/cri/)

### Glossary

- **Comm**: Command name，进程名称（最多 16 字节）
- **Task Struct**: Linux 内核中的进程描述符
- **PID**: Process ID，进程标识符
- **TID**: Thread ID，线程标识符
- **TGID**: Thread Group ID，进程 ID（等同于 PID）
- **cgroup**: Control Groups，Linux 资源控制机制
- **BTF**: BPF Type Format，eBPF 类型信息格式
- **CO-RE**: Compile Once Run Everywhere，一次编译到处运行

### Example Scenarios

#### Scenario 1: 阻断异常进程访问数据库

**背景**: 应用容器被入侵，攻击者植入了反弹 shell

**策略配置**:
```yaml
# 只允许应用进程访问数据库
- process_name: "app-server"
  dest_ip: "10.0.0.5"
  dest_port: 5432
  protocol: "TCP"
  action: "ALLOW"

# 拒绝其他进程
- process_name: "*"
  dest_ip: "10.0.0.5"
  dest_port: 5432
  protocol: "TCP"
  action: "DENY"
```

**效果**:
- `app-server` 进程可以访问数据库
- `bash`, `nc`, `curl` 等进程被阻断
- Dashboard 显示被拒绝的进程尝试

#### Scenario 2: 发现挖矿程序

**背景**: 容器内运行了未授权的挖矿程序

**检测**:
- Dashboard 显示异常进程 `xmrig` 产生大量外联流量
- 进程路径: `/tmp/xmrig`
- 目标: `mining-pool.com:3333`

**响应**:
```yaml
# 阻断挖矿程序
- process_name: "xmrig"
  action: "DENY"
```

**效果**:
- 立即阻断挖矿程序的所有网络访问
- 告警通知安全团队
- 审计日志记录完整进程信息

---

## Review Decisions

本 PRD 已完成评审，以下是关键技术决策：

### 决策 1: 进程退出处理方案
**选择**: 方案 C - 进程生命周期监控（立即全面实施）

**理由**:
- ✅ 主动监控进程执行，解决短生命周期进程路径解析失败问题
- ✅ 通过 tracepoint 监听 sched_process_exec，在进程创建时立即缓存信息
- ✅ 预期成功率从 70-80% 提升到 95%+
- ❌ 实施复杂度增加，时间从 8 周延长到 11 周（可接受）

### 决策 2: 进程名称匹配策略
**选择**: Comm + Path 混合模式

**理由**:
- ✅ 兼顾简单性和精确性
- ✅ process_name 适合简单场景（nginx, curl）
- ✅ process_path 解决同名进程问题（/usr/bin/python vs /tmp/python）
- ✅ 用户可根据需求选择合适的匹配模式

### 决策 3: 策略匹配逻辑
**选择**: 优先级 + AND 逻辑

**匹配优先级**（从高到低）:
1. 精确匹配：进程信息 + 网络 5 元组
2. 进程策略：包含 process_name/process_path
3. 网络策略：传统 5 元组
4. 默认策略：DENY

**字段组合**: 所有字段必须同时匹配（AND 逻辑）

### 决策 4: 性能测试要求
**选择**: 简化版（3 个基本场景）

**测试场景**:
1. 功能验证：正确捕获和匹配
2. 性能对比：启用/禁用对比
3. 短连接测试：验证路径解析成功率

**目标**: 实用主义，聚焦核心验证

### 决策 5: 安全措施
**选择**: 组合方案（路径检查 + 审计告警）

**措施**:
- 自动检查进程路径合法性（信任系统目录，标记可疑目录）
- 审计可疑进程行为，Dashboard 展示告警
- Phase 1 不强制阻断，避免误伤

### 决策 6: 容器环境支持
**选择**: 选项 B - 包含容器 ID

**实施范围**:
- Phase 1: 从 cgroup 提取容器 ID（+1 周）
- Phase 2: 查询 K8s API 获取 Pod 名称、命名空间

**理由**: 平衡功能和实施成本

---

**Last Updated**: 2025-11-19 (Reviewed)
**Author**: Claude (Product Manager AI)
**Reviewers**: User (Approved)
**Status**: Reviewed - Ready for Epic Creation
