# eBPF 微隔离项目 - 代码架构与学习指南

> **目标读者**: 需要快速理解代码逻辑和执行流程的开发者
> **阅读时间**: 主线 1 小时，深入学习 3-4 小时
> **最后更新**: 2025-11-02

---

## 目录

1. [项目架构概览](#1-项目架构概览)
2. [快速入门 - 5 分钟理解核心](#2-快速入门---5-分钟理解核心)
3. [Agent 启动流程](#3-agent-启动流程)
4. [策略管理完整流程](#4-策略管理完整流程)
5. [eBPF 数据包处理流程](#5-ebpf-数据包处理流程)
6. [模块深度解析](#6-模块深度解析)
7. [关键数据结构](#7-关键数据结构)
8. [学习路径建议](#8-学习路径建议)
9. [常见问题解答](#9-常见问题解答)

---

## 1. 项目架构概览

### 1.1 整体架构（三层设计）

```
┌─────────────────────────────────────────────────────────────┐
│                    REST API 层 (Gin)                         │
│  POST /api/v1/policies   GET /health   GET /stats           │
│                                                               │
│  文件: src/agent/pkg/api/                                    │
│    ├─ server.go       - HTTP 服务器                          │
│    ├─ router.go       - 路由配置                             │
│    └─ handlers/       - API 处理器                           │
└──────────────────────────┬──────────────────────────────────┘
                           │ 调用
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                 控制平面 (Go User Space)                     │
│  ┌──────────────────────┐   ┌──────────────────────┐        │
│  │  PolicyManager       │   │  DataPlane           │        │
│  │  (策略管理)          │◄──│  (eBPF 管理)         │        │
│  │  - AddPolicy()       │   │  - New()             │        │
│  │  - DeletePolicy()    │   │  - GetStatistics()   │        │
│  │  - ListPolicies()    │   │  - MonitorFlowEvents()│       │
│  └──────┬───────────────┘   └──────┬───────────────┘        │
│         │                           │                         │
│  文件: pkg/policy/          文件: pkg/dataplane/            │
│    ├─ policy.go                ├─ dataplane.go               │
│    └─ storage.go               └─ interface.go               │
└─────────┼───────────────────────────┼─────────────────────────┘
          │ Update eBPF Maps         │ Load eBPF Program
          ▼                           ▼
┌─────────────────────────────────────────────────────────────┐
│               数据平面 (eBPF Kernel Space)                   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  eBPF Maps (Kernel Memory)                           │   │
│  │  ┌────────────────────────────────────────────────┐  │   │
│  │  │ session_map: 会话跟踪 (LRU_HASH, 100k)        │  │   │
│  │  │   缓存策略决策，热路径优化                    │  │   │
│  │  ├────────────────────────────────────────────────┤  │   │
│  │  │ policy_map: 精确匹配策略 (HASH, 10k)         │  │   │
│  │  │   5-tuple → action/priority                    │  │   │
│  │  ├────────────────────────────────────────────────┤  │   │
│  │  │ wildcard_policy_map: 通配符策略 (ARRAY, 1k)  │  │   │
│  │  │   支持 IP/端口/协议通配符                     │  │   │
│  │  ├────────────────────────────────────────────────┤  │   │
│  │  │ stats_map: 统计计数器 (PERCPU_ARRAY)         │  │   │
│  │  ├────────────────────────────────────────────────┤  │   │
│  │  │ flow_events: 流事件缓冲 (RINGBUF, 256KB)     │  │   │
│  │  └────────────────────────────────────────────────┘  │   │
│  └──────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  TC eBPF Program: tc_microsegment_filter()           │   │
│  │    - 提取 5-tuple (src_ip, dst_ip, ports, proto)    │   │
│  │    - 查找会话缓存 (热路径 >99%)                     │   │
│  │    - 查找策略 (精确 + 通配符)                        │   │
│  │    - 执行决策 (TC_ACT_OK / TC_ACT_SHOT)             │   │
│  │                                                       │   │
│  │  文件: src/bpf/tc_microsegment.bpf.c                 │   │
│  └──────────────────────────────────────────────────────┘   │
│                              ▲                                │
│                              │ TC Ingress Hook               │
└──────────────────────────────┼───────────────────────────────┘
                               │
                          网络数据包流
```

### 1.2 目录结构详解

```
src/
├── agent/                          # Go 用户态程序
│   ├── cmd/
│   │   └── main.go                 # 程序入口 ⭐ 从这里开始
│   │       - 解析命令行参数
│   │       - 初始化 DataPlane
│   │       - 初始化 PolicyManager
│   │       - 启动 API Server
│   │       - 启动监控 goroutines
│   │
│   ├── pkg/
│   │   ├── dataplane/              # eBPF 数据平面管理 ⭐ 核心模块
│   │   │   ├── dataplane.go        # [350 行] 主要逻辑
│   │   │   │   - New(): 加载 eBPF 程序
│   │   │   │   - GetStatistics(): 读取计数器
│   │   │   │   - MonitorFlowEvents(): Ring Buffer 监控
│   │   │   ├── interface.go        # [50 行] 接口定义
│   │   │   └── bpf_x86_bpfel.go   # [自动生成] eBPF 对象绑定
│   │   │
│   │   ├── policy/                 # 策略管理 ⭐ 核心模块
│   │   │   ├── policy.go           # [500 行] 策略 CRUD
│   │   │   │   - AddPolicy(): 精确/通配符策略
│   │   │   │   - DeletePolicy(): 策略删除
│   │   │   │   - ListPolicies(): 策略列表
│   │   │   ├── storage.go          # [200 行] SQLite 持久化
│   │   │   └── interface.go        # [40 行] 接口定义
│   │   │
│   │   ├── api/                    # REST API 服务器
│   │   │   ├── server.go           # [100 行] HTTP 服务器
│   │   │   ├── router.go           # [60 行] 路由配置
│   │   │   ├── middleware.go       # [80 行] 日志/CORS/错误处理
│   │   │   ├── handlers/           # API 处理器
│   │   │   │   ├── policy.go       # [250 行] 策略 CRUD API
│   │   │   │   ├── health.go       # [60 行] 健康检查
│   │   │   │   ├── statistics.go   # [120 行] 统计 API
│   │   │   │   └── config.go       # [80 行] 配置 API
│   │   │   └── models/             # API 数据模型
│   │   │       ├── policy.go       # 策略请求/响应
│   │   │       ├── statistics.go   # 统计响应
│   │   │       └── error.go        # 错误响应
│   │   │
│   │   ├── workload/               # 工作负载标签系统 (新增)
│   │   │   ├── types.go            # [179 行] Workload 数据模型
│   │   │   └── types_test.go       # [400 行] 单元测试
│   │   │
│   │   └── testutil/               # 测试工具
│   │       └── fixtures.go         # 测试数据生成
│   │
│   └── test/
│       ├── e2e/                    # 端到端测试
│       │   └── microsegment_test.go
│       └── benchmark/              # 性能基准测试
│           └── policy_bench_test.go
│
└── bpf/                            # eBPF 内核程序 (C)
    ├── tc_microsegment.bpf.c       # [800 行] TC eBPF 主程序 ⭐ 核心
    │   - tc_microsegment_filter(): 主入口
    │   - extract_flow_key(): 提取 5-tuple
    │   - lookup_policy_action(): 策略查找
    │   - matches_wildcard(): 通配符匹配
    │   - create_session(): 会话创建
    │
    └── headers/
        └── common_types.h          # [300 行] 共享数据结构
            - struct flow_key
            - struct session_value
            - struct policy_value
            - struct wildcard_policy
            - struct flow_event
```

### 1.3 模块间调用关系

```
main.go
  │
  ├─> dataplane.New(iface)
  │     └─> 加载 eBPF 程序 (tc_microsegment.bpf.c)
  │     └─> 创建 Ring Buffer Reader
  │
  ├─> policy.NewManager(dataPlane)
  │     └─> 获取 eBPF map 引用 (policy_map, wildcard_policy_map)
  │
  ├─> api.NewAPIServer(config, dataPlane, policyManager)
  │     ├─> setupRoutes()
  │     │     ├─> handlers.NewPolicyHandler(policyManager)
  │     │     ├─> handlers.NewHealthHandler(dataPlane)
  │     │     └─> handlers.NewStatisticsHandler(dataPlane)
  │     │
  │     └─> apiServer.Start()
  │           └─> router.Run(":8080")
  │
  ├─> go dataPlane.MonitorFlowEvents()
  │     └─> 无限循环读取 Ring Buffer
  │
  └─> go func() { 定期打印统计 }
        └─> dataPlane.GetStatistics()
```

---

## 2. 快速入门 - 5 分钟理解核心

### 2.1 这是什么？

**eBPF 微隔离系统**：基于 eBPF TC (Traffic Control) 的高性能网络访问控制系统

- **数据平面**: eBPF 程序在内核空间拦截网络包，执行策略决策（<10μs 延迟）
- **控制平面**: Go 程序在用户空间管理策略，提供 REST API

### 2.2 关键概念（5 个核心点）

#### 1. **TC Hook（流量拦截点）**
```
网络包到达 → TC Ingress Hook → eBPF 程序执行 → 允许/拒绝
```

#### 2. **策略类型（两种）**
- **精确匹配**: `10.0.1.1:80 → 10.0.2.2:3306 allow` (O(1) 哈希查找)
- **通配符匹配**: `10.0.1.0/24:* → *:3306 deny` (O(n) 线性搜索)

#### 3. **会话跟踪（性能关键）**
- 首包：查找策略（慢）→ 创建会话 → 缓存决策
- 后续包：直接从会话读取决策（快，>99% 的包）

#### 4. **数据流向**
```
API 请求
  → PolicyManager.AddPolicy()
  → eBPF Map Update (policy_map / wildcard_policy_map)
  → eBPF 程序实时生效
```

#### 5. **事件上报**
```
eBPF 检测到 DENY/新会话
  → Ring Buffer 写入事件
  → 用户空间读取
  → 日志记录
```

### 2.3 三大核心文件（必读）

| 文件 | 作用 | 关键函数 |
|------|------|----------|
| **src/agent/cmd/main.go** | 程序入口 | `runAgent()` - 启动流程 |
| **src/agent/pkg/dataplane/dataplane.go** | eBPF 管理 | `New()` - 加载 eBPF<br>`GetStatistics()` - 读取统计<br>`MonitorFlowEvents()` - 事件监控 |
| **src/bpf/tc_microsegment.bpf.c** | 数据包过滤 | `tc_microsegment_filter()` - 主入口<br>`lookup_policy_action()` - 策略查找 |

---

## 3. Agent 启动流程

### 3.1 启动序列（带代码引用）

```go
// src/agent/cmd/main.go

func main() {
    rootCmd.Execute()  // Cobra CLI 框架
}

func runAgent(cmd *cobra.Command, args []string) {
    // ============ 阶段 1: 配置解析 ============
    level, _ := log.ParseLevel(logLevel)  // 日志级别
    log.SetLevel(level)
    log.Infof("Starting microsegmentation agent on interface %s", iface)

    // ============ 阶段 2: 数据平面初始化 ============
    // 文件: src/agent/pkg/dataplane/dataplane.go:44
    dp, err := dataplane.New(iface)  // ⭐ 核心：加载 eBPF 程序
    if err != nil {
        log.Fatalf("Failed to initialize dataplane: %v", err)
    }
    defer dp.Close()

    // ============ 阶段 3: 策略管理器初始化 ============
    // 文件: src/agent/pkg/policy/policy.go:37
    pm := policy.NewManager(dp)  // 创建策略管理器

    // ============ 阶段 4: 加载持久化策略 (可选) ============
    if err := pm.LoadPersisted(); err != nil {
        log.Warnf("Failed to load persisted policies: %v", err)
    }

    // ============ 阶段 5: 添加默认策略 ============
    defaultPolicy := &policy.Policy{
        RuleID:   1,
        SrcIP:    "0.0.0.0/0",  // 任意源
        DstIP:    "0.0.0.0/0",  // 任意目标
        SrcPort:  0,            // 任意源端口
        DstPort:  0,            // 任意目标端口
        Protocol: "any",        // 任意协议
        Action:   "allow",      // 默认允许
        Priority: 1,            // 最低优先级
    }
    pm.AddPolicy(defaultPolicy)  // 添加到 eBPF map

    // ============ 阶段 6: 启动 API 服务器 (可选) ============
    var apiServer *api.Server
    if enableAPI {
        apiConfig := &api.Config{
            Host: apiHost,  // "127.0.0.1"
            Port: apiPort,  // 8080
        }

        // 文件: src/agent/pkg/api/server.go:40
        apiServer, err = api.NewAPIServer(apiConfig, dp, pm)
        if err != nil {
            log.Fatalf("Failed to create API server: %v", err)
        }

        // 启动 HTTP 服务器（后台 goroutine）
        if err := apiServer.Start(); err != nil {
            log.Fatalf("Failed to start API server: %v", err)
        }
        log.Infof("API server listening on %s:%d", apiHost, apiPort)
    }

    // ============ 阶段 7: 启动流事件监控 ============
    go dp.MonitorFlowEvents()  // 后台监控 Ring Buffer

    // ============ 阶段 8: 启动统计打印 ============
    ticker := time.NewTicker(time.Duration(statsInterval) * time.Second)
    defer ticker.Stop()

    go func() {
        for range ticker.C {
            stats := dp.GetStatistics()  // 读取 eBPF 统计
            log.Infof("Statistics: %+v", stats)
        }
    }()

    // ============ 阶段 9: 等待退出信号 ============
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    <-sigCh  // 阻塞等待 Ctrl+C 或 kill

    // ============ 阶段 10: 优雅关闭 ============
    log.Info("Shutting down...")
    if apiServer != nil {
        apiServer.Shutdown(context.Background())  // 关闭 HTTP 服务器
    }
    dp.Close()  // 关闭 eBPF 程序
}
```

### 3.2 dataplane.New() 详解

```go
// src/agent/pkg/dataplane/dataplane.go:44-176

func New(iface string) (*DataPlane, error) {
    // ============ 步骤 1: 获取网卡索引 ============
    ifaceObj, err := net.InterfaceByName(iface)  // 例如: "lo", "eth0"
    if err != nil {
        return nil, fmt.Errorf("failed to get interface %s: %w", iface, err)
    }

    // ============ 步骤 2: 加载 eBPF 对象 ============
    objs := &bpfObjects{}
    // 由 bpf2go 自动生成的函数，加载编译好的 eBPF 字节码
    if err := loadBpfObjects(objs, nil); err != nil {
        return nil, fmt.Errorf("failed to load eBPF objects: %w", err)
    }

    // ============ 步骤 3: 附加 eBPF 程序到 TC ============
    var tcLink link.Link

    // 尝试使用新的 TCX 接口 (kernel >= 6.6)
    tcLink, err = link.AttachTCX(link.TCXOptions{
        Interface: ifaceObj.Index,
        Program:   objs.TcMicrosegmentFilter,  // eBPF 程序
        Attach:    ebpf.AttachTCXIngress,      // Ingress 方向
    })

    if err != nil {
        // 回退到传统 netlink TC 接口 (kernel < 6.6)
        log.Warnf("TCX attach failed, falling back to legacy netlink TC: %v", err)

        // 创建 clsact qdisc（TC 队列规则）
        qdisc := &netlink.GenericQdisc{
            QdiscAttrs: netlink.QdiscAttrs{
                LinkIndex: ifaceObj.Index,
                Handle:    netlink.MakeHandle(0xffff, 0),
                Parent:    netlink.HANDLE_CLSACT,
            },
            QdiscType: "clsact",
        }
        netlink.QdiscAdd(qdisc)

        // 附加 BPF filter
        filter := &netlink.BpfFilter{
            FilterAttrs: netlink.FilterAttrs{
                LinkIndex: ifaceObj.Index,
                Parent:    netlink.HANDLE_MIN_INGRESS,
                Handle:    1,
                Protocol:  unix.ETH_P_ALL,
                Priority:  1,
            },
            Fd:           objs.TcMicrosegmentFilter.FD(),
            Name:         "tc_microsegment_filter",
            DirectAction: true,
        }
        netlink.FilterAdd(filter)
    }

    // ============ 步骤 4: 创建 Ring Buffer Reader ============
    rbReader, err := ringbuf.NewReader(objs.FlowEvents)  // 读取流事件
    if err != nil {
        return nil, fmt.Errorf("failed to create ringbuf reader: %w", err)
    }

    // ============ 步骤 5: 返回 DataPlane 实例 ============
    return &DataPlane{
        objs:      objs,        // eBPF maps
        iface:     iface,       // 网卡名称
        tcLink:    tcLink,      // TC 链接（用于清理）
        rbReader:  rbReader,    // Ring Buffer 读取器
    }, nil
}
```

### 3.3 启动流程时序图

```
main()
  │
  ├─> [1] 解析命令行参数
  │     - interface = "lo"
  │     - log-level = "info"
  │     - enable-api = true
  │
  ├─> [2] dataplane.New("lo")
  │     ├─> 加载 eBPF 字节码 (loadBpfObjects)
  │     ├─> 附加到 TC Ingress Hook
  │     └─> 创建 Ring Buffer Reader
  │     ⏱ 耗时: ~100ms
  │
  ├─> [3] policy.NewManager(dp)
  │     └─> 获取 policy_map 和 wildcard_policy_map 引用
  │     ⏱ 耗时: <1ms
  │
  ├─> [4] pm.AddPolicy(defaultPolicy)
  │     └─> 添加 "allow all" 默认策略
  │     ⏱ 耗时: <1ms
  │
  ├─> [5] api.NewAPIServer(...)
  │     ├─> 创建 Gin Router
  │     ├─> 注册路由和处理器
  │     └─> apiServer.Start()
  │           └─> go router.Run(":8080")  # 后台 HTTP 服务器
  │     ⏱ 耗时: ~10ms
  │
  ├─> [6] go dp.MonitorFlowEvents()  # 后台监控 goroutine
  │     └─> 无限循环读取 Ring Buffer
  │
  ├─> [7] go func() { 定期打印统计 }  # 后台统计 goroutine
  │     └─> 每 5 秒调用 dp.GetStatistics()
  │
  └─> [8] 等待退出信号 (Ctrl+C)
        └─> 清理资源
              ├─> apiServer.Shutdown()
              └─> dp.Close()
                    ├─> 关闭 Ring Buffer Reader
                    ├─> 分离 TC 链接
                    └─> 关闭 eBPF maps
```

---

## 4. 策略管理完整流程

### 4.1 策略添加流程（从 API 到 eBPF）

#### 完整调用链

```
[HTTP 请求]
POST /api/v1/policies
Body: {
  "rule_id": 100,
  "src_ip": "10.0.1.0/24",
  "dst_ip": "10.0.2.2/32",
  "dst_port": 3306,
  "protocol": "tcp",
  "action": "allow",
  "priority": 100
}

  ↓ (1) Gin 路由匹配

[API Handler]
policyHandler.CreatePolicy(c *gin.Context)
  ├─> c.ShouldBindJSON(&req)           // 解析 JSON
  ├─> 验证参数 (IP/端口/协议)
  └─> policyManager.AddPolicy(policy)  // 调用策略管理器

  ↓ (2) 策略管理器处理

[PolicyManager]
pm.AddPolicy(p *Policy)
  ├─> addPolicyToMap(p)                // 添加到 eBPF map
  │     ├─> 检查是否包含通配符 (hasWildcard)
  │     │     - 检查 src_ip 是否为 /32
  │     │     - 检查 dst_port 是否为 0
  │     │
  │     ├─> [精确匹配分支] addExactPolicy(p)
  │     │     ├─> 解析 IP/端口/协议
  │     │     ├─> 构建 policy_key (5-tuple)
  │     │     ├─> 构建 policy_value (action, priority)
  │     │     └─> policyMap.Put(&key, &value)
  │     │           └─> eBPF 系统调用 bpf(BPF_MAP_UPDATE_ELEM)
  │     │
  │     └─> [通配符分支] addWildcardPolicy(p)
  │           ├─> 计算 IP 掩码 (例如: /24 → 0xFFFFFF00)
  │           ├─> 在 wildcard_policy_map 中查找空槽位
  │           └─> wildcardPolicyMap.Put(&index, &wildcard)
  │                 └─> eBPF 系统调用 bpf(BPF_MAP_UPDATE_ELEM)
  │
  └─> storage.SavePolicy(p)            // 持久化到 SQLite (可选)

  ↓ (3) eBPF Map 更新完成

[Kernel Space]
policy_map / wildcard_policy_map 已更新
  └─> eBPF 程序实时生效，无需重启

  ↓ (4) HTTP 响应

[API Response]
HTTP 201 Created
Body: {
  "rule_id": 100,
  "message": "Policy added successfully"
}
```

### 4.2 精确匹配策略添加（代码详解）

```go
// src/agent/pkg/policy/policy.go:131-201

func (pm *PolicyManager) addExactPolicy(p *Policy) error {
    // ============ 步骤 1: 解析 IP 地址 ============
    srcIP, srcMask, err := parseCIDR(p.SrcIP)  // 例如: "10.0.1.10/32"
    if err != nil {
        return fmt.Errorf("invalid src_ip: %w", err)
    }

    dstIP, dstMask, err := parseCIDR(p.DstIP)  // 例如: "10.0.2.2/32"
    if err != nil {
        return fmt.Errorf("invalid dst_ip: %w", err)
    }

    // 精确匹配要求 /32 掩码
    if srcMask != 32 || dstMask != 32 {
        return fmt.Errorf("exact policy requires /32 CIDR")
    }

    // ============ 步骤 2: 解析协议 ============
    proto, err := parseProtocol(p.Protocol)  // "tcp" → 6, "udp" → 17
    if err != nil {
        return fmt.Errorf("invalid protocol: %w", err)
    }

    // ============ 步骤 3: 解析动作 ============
    action, err := parseAction(p.Action)  // "allow" → 0, "deny" → 1
    if err != nil {
        return fmt.Errorf("invalid action: %w", err)
    }

    // ============ 步骤 4: 构建 policy_key（5-tuple）============
    // 必须与 eBPF 中的 struct policy_key 布局完全一致
    key := struct {
        SrcIp    uint32   // 网络字节序 (大端)
        DstIp    uint32   // 网络字节序 (大端)
        SrcPort  uint16   // 网络字节序 (大端)
        DstPort  uint16   // 网络字节序 (大端)
        Protocol uint8
        Pad      [3]uint8 // 对齐到 8 字节
    }{
        SrcIp:    srcIP,                        // 10.0.1.10
        DstIp:    dstIP,                        // 10.0.2.2
        SrcPort:  htons(uint16(p.SrcPort)),    // 0 (任意)
        DstPort:  htons(uint16(p.DstPort)),    // 3306
        Protocol: proto,                        // 6 (TCP)
    }

    // ============ 步骤 5: 构建 policy_value ============
    value := struct {
        Action     uint8
        LogEnabled uint8
        Priority   uint16
        RuleID     uint32
        HitCount   uint64
    }{
        Action:     action,       // 0 (allow)
        LogEnabled: 0,            // 暂未实现
        Priority:   uint16(p.Priority),  // 100
        RuleID:     p.RuleID,     // 100
        HitCount:   0,            // 初始命中数
    }

    // ============ 步骤 6: 插入 eBPF HASH map ============
    // 使用 cilium/ebpf 库的 Put 方法
    if err := pm.policyMap.Put(&key, &value); err != nil {
        return fmt.Errorf("failed to update policy_map: %w", err)
    }

    log.Infof("Exact policy added: %s:%d -> %s:%d proto=%s action=%s priority=%d",
        p.SrcIP, p.SrcPort, p.DstIP, p.DstPort, p.Protocol, p.Action, p.Priority)

    return nil
}
```

### 4.3 通配符策略添加（代码详解）

```go
// src/agent/pkg/policy/policy.go:374-476

func (pm *PolicyManager) addWildcardPolicy(p *Policy) error {
    // ============ 步骤 1: 解析 IP 和掩码 ============
    srcIP, srcMask, _ := parseCIDR(p.SrcIP)  // "10.0.1.0/24"
    dstIP, dstMask, _ := parseCIDR(p.DstIP)  // "10.0.2.0/24"

    // 计算 IP 掩码（用于 eBPF 匹配）
    srcIPMask := cidrToMask(srcMask)  // /24 → 0xFFFFFF00
    dstIPMask := cidrToMask(dstMask)  // /24 → 0xFFFFFF00

    // ============ 步骤 2: 构建 wildcard_policy 结构 ============
    wildcard := struct {
        SrcIP      uint32  // 10.0.1.0
        SrcIPMask  uint32  // 0xFFFFFF00 (匹配前 24 位)
        DstIP      uint32  // 10.0.2.0
        DstIPMask  uint32  // 0xFFFFFF00 (匹配前 24 位)
        SrcPort    uint16  // 0 = 任意端口
        DstPort    uint16  // 3306 = 指定端口
        Protocol   uint8   // 6 = TCP (0 = 任意协议)
        Action     uint8   // 0 = allow
        Priority   uint16  // 100
        RuleID     uint32  // 100
        Pad        [2]uint8
    }{
        SrcIP:     srcIP & srcIPMask,  // 归一化 IP (清除主机位)
        SrcIPMask: srcIPMask,
        DstIP:     dstIP & dstIPMask,
        DstIPMask: dstIPMask,
        SrcPort:   htons(uint16(p.SrcPort)),
        DstPort:   htons(uint16(p.DstPort)),
        Protocol:  parseProtocol(p.Protocol),
        Action:    parseAction(p.Action),
        Priority:  uint16(p.Priority),
        RuleID:    p.RuleID,
    }

    // ============ 步骤 3: 在 ARRAY map 中查找槽位 ============
    // wildcard_policy_map 是固定大小的数组（1000 个槽位）
    for i := uint32(0); i < 1000; i++ {
        var existing struct {...}
        err := pm.wildcardPolicyMap.Lookup(&i, &existing)

        // 找到空槽位（RuleID == 0）
        if err != nil || existing.RuleID == 0 {
            // 插入新策略
            if err := pm.wildcardPolicyMap.Put(&i, &wildcard); err != nil {
                return fmt.Errorf("failed to add wildcard policy: %w", err)
            }
            log.Infof("Wildcard policy added to slot %d: %s -> %s",
                i, p.SrcIP, p.DstIP)
            return nil
        }

        // 找到已有相同 RuleID 的策略（更新）
        if existing.RuleID == p.RuleID {
            if err := pm.wildcardPolicyMap.Put(&i, &wildcard); err != nil {
                return fmt.Errorf("failed to update wildcard policy: %w", err)
            }
            log.Infof("Wildcard policy updated at slot %d", i)
            return nil
        }
    }

    return fmt.Errorf("wildcard policy map is full (1000 slots)")
}
```

### 4.4 策略查询流程

```go
// src/agent/pkg/policy/policy.go:256-310

func (pm *PolicyManager) ListPolicies() ([]*Policy, error) {
    policies := make([]*Policy, 0)

    // ============ 步骤 1: 遍历精确匹配策略 ============
    var key struct {
        SrcIp    uint32
        DstIp    uint32
        SrcPort  uint16
        DstPort  uint16
        Protocol uint8
        Pad      [3]uint8
    }
    var value struct {
        Action     uint8
        LogEnabled uint8
        Priority   uint16
        RuleID     uint32
        HitCount   uint64
    }

    // 创建 map 迭代器
    iter := pm.policyMap.Iterate()

    for iter.Next(&key, &value) {
        // 转换为 Policy 结构
        p := &Policy{
            RuleID:   value.RuleID,
            SrcIP:    ipToString(key.SrcIp),      // 转换为 "10.0.1.10"
            DstIP:    ipToString(key.DstIp),
            SrcPort:  ntohs(key.SrcPort),         // 转换为主机字节序
            DstPort:  ntohs(key.DstPort),
            Protocol: protoToString(key.Protocol), // 6 → "tcp"
            Action:   actionToString(value.Action), // 0 → "allow"
            Priority: int(value.Priority),
        }
        policies = append(policies, p)
    }

    // ============ 步骤 2: 遍历通配符策略 ============
    for i := uint32(0); i < 1000; i++ {
        var wildcard struct {...}
        err := pm.wildcardPolicyMap.Lookup(&i, &wildcard)

        if err != nil || wildcard.RuleID == 0 {
            continue  // 空槽位，跳过
        }

        // 转换为 Policy 结构
        p := &Policy{
            RuleID:   wildcard.RuleID,
            SrcIP:    maskToC IDR(wildcard.SrcIP, wildcard.SrcIPMask),  // "10.0.1.0/24"
            DstIP:    maskToCIDR(wildcard.DstIP, wildcard.DstIPMask),
            SrcPort:  ntohs(wildcard.SrcPort),
            DstPort:  ntohs(wildcard.DstPort),
            Protocol: protoToString(wildcard.Protocol),
            Action:   actionToString(wildcard.Action),
            Priority: int(wildcard.Priority),
        }
        policies = append(policies, p)
    }

    return policies, nil
}
```

---

## 5. eBPF 数据包处理流程

### 5.1 数据包处理主函数

```c
// src/bpf/tc_microsegment.bpf.c:246-320

SEC("tc")
int tc_microsegment_filter(struct __sk_buff *skb) {
    // ============ 步骤 1: 提取流密钥（5-tuple）============
    struct flow_key key = {0};

    if (extract_flow_key(skb, &key) < 0) {
        // 非 IP 包（例如 ARP），直接放行
        return TC_ACT_OK;
    }

    // ============ 步骤 2: 更新统计：总包数 ============
    update_stats(STATS_TOTAL_PACKETS);

    // ============ 步骤 3: 快速路径 - 查找现有会话 ============
    // 这是性能关键：>99% 的数据包走这条路径
    struct session_value *session = bpf_map_lookup_elem(&session_map, &key);

    if (session) {
        // ========== 热路径：已存在的会话 ==========

        // 直接使用缓存的策略决策（无需重新查找策略）
        __u8 action = session->policy_action;

        // 更新会话统计
        session->last_seen_ts = get_timestamp_ns();
        session->packets_to_server += 1;
        session->bytes_to_server += skb->len;

        // 执行策略
        if (action == POLICY_ACTION_DENY) {
            update_stats(STATS_DENIED_PACKETS);
            return TC_ACT_SHOT;  // 丢弃数据包
        }

        update_stats(STATS_ALLOWED_PACKETS);
        return TC_ACT_OK;  // 允许数据包通过
    }

    // ============ 步骤 4: 慢速路径 - 新会话策略查找 ============
    // 只有首包会走这条路径（<1% 的数据包）

    __u64 now = get_timestamp_ns();
    __u32 matched_rule_id = 0;

    // 查找策略（精确匹配 + 通配符匹配）
    __u8 action = lookup_policy_action(&key, &matched_rule_id);

    // ============ 步骤 5: 创建新会话 ============
    // 缓存策略决策，后续包直接使用
    create_session(&key, action, now, skb->len);

    // ============ 步骤 6: 执行策略 ============
    if (action == POLICY_ACTION_DENY) {
        update_stats(STATS_DENIED_PACKETS);
        return TC_ACT_SHOT;  // 丢弃数据包
    }

    update_stats(STATS_ALLOWED_PACKETS);
    return TC_ACT_OK;  // 允许数据包通过
}
```

### 5.2 流密钥提取（extract_flow_key）

```c
// src/bpf/tc_microsegment.bpf.c:59-120

static __always_inline int extract_flow_key(struct __sk_buff *skb, struct flow_key *key) {
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;

    // ============ 步骤 1: 解析 Ethernet 头 ============
    struct ethhdr *eth = data;
    if ((void *)(eth + 1) > data_end)
        return -1;  // 数据包太短

    // 检查是否为 IP 包
    if (eth->h_proto != bpf_htons(ETH_P_IP))
        return -1;  // 非 IP 包（例如 IPv6, ARP）

    // ============ 步骤 2: 解析 IP 头 ============
    struct iphdr *ip = (void *)(eth + 1);
    if ((void *)(ip + 1) > data_end)
        return -1;

    // 提取 IP 地址和协议
    key->src_ip = ip->saddr;  // 源 IP (网络字节序)
    key->dst_ip = ip->daddr;  // 目标 IP (网络字节序)
    key->protocol = ip->protocol;  // 6=TCP, 17=UDP, 1=ICMP

    // ============ 步骤 3: 根据协议解析传输层头 ============
    void *l4_header = (void *)ip + (ip->ihl * 4);  // L4 头位置

    if (ip->protocol == IPPROTO_TCP) {
        // TCP 包
        struct tcphdr *tcp = l4_header;
        if ((void *)(tcp + 1) > data_end)
            return -1;

        key->src_port = tcp->source;  // 源端口 (网络字节序)
        key->dst_port = tcp->dest;    // 目标端口 (网络字节序)

    } else if (ip->protocol == IPPROTO_UDP) {
        // UDP 包
        struct udphdr *udp = l4_header;
        if ((void *)(udp + 1) > data_end)
            return -1;

        key->src_port = udp->source;
        key->dst_port = udp->dest;

    } else {
        // ICMP 或其他协议，端口设为 0
        key->src_port = 0;
        key->dst_port = 0;
    }

    return 0;  // 成功提取
}
```

### 5.3 策略查找（lookup_policy_action）

```c
// src/bpf/tc_microsegment.bpf.c:154-206

static __always_inline __u8 lookup_policy_action(struct flow_key *key, __u32 *rule_id) {
    // ============ 快速路径：精确匹配（O(1) 哈希查找）============
    struct policy_value *policy = bpf_map_lookup_elem(&policy_map, key);

    if (policy) {
        // 找到精确匹配策略
        policy->hit_count += 1;  // 更新命中计数
        update_stats(STATS_POLICY_HITS);
        *rule_id = policy->rule_id;
        return policy->action;  // 返回 allow/deny
    }

    // ============ 慢速路径：通配符线性搜索 ============
    struct wildcard_policy *wildcard = NULL;
    struct wildcard_policy *best_match = NULL;
    __u16 best_priority = 0;

    // 遍历通配符策略数组（最多 100 次迭代，Verifier 限制）
    #pragma unroll
    for (__u32 i = 0; i < 100; i++) {
        __u32 idx = i;
        if (idx >= MAX_ENTRIES_WILDCARD_POLICY)
            break;

        // 读取通配符策略
        wildcard = bpf_map_lookup_elem(&wildcard_policy_map, &idx);
        if (!wildcard)
            continue;

        // 跳过空槽位（RuleID == 0 表示未使用）
        if (wildcard->rule_id == 0)
            continue;

        // 检查是否匹配
        if (matches_wildcard(key, wildcard)) {
            // 选择优先级最高的匹配（priority 值越大越优先）
            if (!best_match || wildcard->priority > best_priority) {
                best_match = wildcard;
                best_priority = wildcard->priority;
            }
        }
    }

    if (best_match) {
        update_stats(STATS_POLICY_HITS);
        *rule_id = best_match->rule_id;
        return best_match->action;
    }

    // ============ 默认策略：允许 ============
    update_stats(STATS_POLICY_MISSES);
    *rule_id = 0;
    return POLICY_ACTION_ALLOW;
}
```

### 5.4 通配符匹配（matches_wildcard）

```c
// src/bpf/tc_microsegment.bpf.c:124-149

static __always_inline bool matches_wildcard(
    struct flow_key *key,
    struct wildcard_policy *wildcard)
{
    // ============ 步骤 1: 源 IP 匹配（使用掩码）============
    // 例如: key->src_ip = 10.0.1.50 (0x0A000132)
    //       wildcard->src_ip = 10.0.1.0 (0x0A000100)
    //       wildcard->src_ip_mask = 0xFFFFFF00 (/24)
    // 匹配逻辑: (0x0A000132 & 0xFFFFFF00) == (0x0A000100 & 0xFFFFFF00)
    //            0x0A000100 == 0x0A000100 → 匹配！

    if ((key->src_ip & wildcard->src_ip_mask) !=
        (wildcard->src_ip & wildcard->src_ip_mask))
        return false;

    // ============ 步骤 2: 目标 IP 匹配 ============
    if ((key->dst_ip & wildcard->dst_ip_mask) !=
        (wildcard->dst_ip & wildcard->dst_ip_mask))
        return false;

    // ============ 步骤 3: 源端口匹配 ============
    // 0 = 通配符（匹配任意端口）
    if (wildcard->src_port != 0 && key->src_port != wildcard->src_port)
        return false;

    // ============ 步骤 4: 目标端口匹配 ============
    if (wildcard->dst_port != 0 && key->dst_port != wildcard->dst_port)
        return false;

    // ============ 步骤 5: 协议匹配 ============
    // 0 = 通配符（匹配任意协议）
    if (wildcard->protocol != 0 && key->protocol != wildcard->protocol)
        return false;

    // 所有条件满足，匹配成功！
    return true;
}
```

### 5.5 会话创建（create_session）

```c
// src/bpf/tc_microsegment.bpf.c:209-243

static __always_inline int create_session(
    struct flow_key *key,
    __u8 action,
    __u64 ts,
    __u32 packet_len)
{
    // ============ 步骤 1: 初始化会话值 ============
    struct session_value new_session = {
        .created_ts = ts,              // 创建时间戳
        .last_seen_ts = ts,            // 最后活跃时间
        .packets_to_server = 1,        // 首包计数
        .packets_to_client = 0,
        .bytes_to_server = packet_len, // 首包字节数
        .bytes_to_client = 0,
        .state = SESSION_STATE_NEW,
        .tcp_state = TCP_STATE_CLOSED,
        .policy_action = action,        // ⭐ 缓存策略决策（关键！）
        .flags = 0,
    };

    // ============ 步骤 2: 插入 session_map（LRU_HASH）============
    // BPF_NOEXIST: 仅当键不存在时插入（避免竞态条件）
    int ret = bpf_map_update_elem(&session_map, key, &new_session, BPF_NOEXIST);

    if (ret == 0) {
        // 插入成功
        update_stats(STATS_NEW_SESSIONS);

        // ============ 步骤 3: 发送流事件到用户空间 ============
        // 仅对 DENY 或 LOG 动作发送事件（减少开销）
        if (action == POLICY_ACTION_DENY || action == POLICY_ACTION_LOG) {
            struct flow_event *event = bpf_ringbuf_reserve(&flow_events, sizeof(*event), 0);
            if (event) {
                // 填充事件数据
                event->key = *key;
                event->timestamp = ts;
                event->packets = 1;
                event->bytes = packet_len;
                event->action = action;
                event->event_type = 0;  // 0 = new session

                // 提交到 Ring Buffer
                bpf_ringbuf_submit(event, 0);
            }
        }
    }

    return ret;
}
```

### 5.6 数据包处理流程图

```
┌─────────────────────────────────────────────────────────────┐
│                    网络数据包到达                            │
│                  10.0.1.10:45678 → 10.0.2.2:3306            │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│        TC Ingress Hook 触发 eBPF 程序                        │
│        tc_microsegment_filter(skb)                          │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
         ┌───────────────────────┐
         │ extract_flow_key()    │
         │ 提取 5-tuple         │
         └───────────┬───────────┘
                     │
                     ▼
         key = {
           src_ip: 10.0.1.10,
           dst_ip: 10.0.2.2,
           src_port: 45678,
           dst_port: 3306,
           protocol: 6 (TCP)
         }
                     │
                     ▼
         ┌───────────────────────┐
         │ update_stats()        │
         │ TOTAL_PACKETS++       │
         └───────────┬───────────┘
                     │
                     ▼
    ┌────────────────────────────────────┐
    │ bpf_map_lookup_elem(&session_map)  │
    │ 查找现有会话                       │
    └────────────┬───────────────────────┘
                 │
        ┌────────┴────────┐
        │                 │
   找到会话          未找到会话
   (热路径 >99%)     (冷路径 <1%)
        │                 │
        ▼                 ▼
┌────────────────┐   ┌──────────────────────┐
│ 热路径处理     │   │ 冷路径处理            │
│                │   │                       │
│ 1. 读取缓存的  │   │ 1. lookup_policy()   │
│    policy_action│   │    ├─ 精确匹配 (O(1))│
│                │   │    └─ 通配符 (O(n))   │
│ 2. 更新统计    │   │                       │
│    - last_seen │   │ 2. create_session()  │
│    - packets++ │   │    - 缓存 action      │
│    - bytes++   │   │    - 初始化统计       │
│                │   │    - 发送事件(可选)   │
└────────┬───────┘   └──────────┬───────────┘
         │                      │
         └──────────┬───────────┘
                    │
                    ▼
         ┌───────────────────────┐
         │ if (action == DENY)   │
         └───────────┬───────────┘
                     │
        ┌────────────┴────────────┐
        │                         │
   action=DENY              action=ALLOW
        │                         │
        ▼                         ▼
┌──────────────────┐   ┌──────────────────┐
│ return TC_ACT_SHOT│   │ return TC_ACT_OK │
│ 丢弃数据包        │   │ 允许数据包通过    │
└───────────────────┘   └──────────────────┘
```

---

## 6. 模块深度解析

### 6.1 DataPlane 模块（pkg/dataplane）

#### 核心职责
- 加载和管理 eBPF 程序
- 读取统计数据
- 监控流事件

#### 关键方法

**1. New() - 初始化数据平面**

```go
// 文件: src/agent/pkg/dataplane/dataplane.go:44
func New(iface string) (*DataPlane, error)
```
- **作用**: 加载 eBPF 程序并附加到网卡
- **流程**:
  1. 获取网卡索引
  2. 加载 eBPF 对象（loadBpfObjects）
  3. 尝试 TCX 附加（新接口）
  4. 失败则回退到 netlink TC（旧接口）
  5. 创建 Ring Buffer Reader
- **耗时**: ~100ms
- **调用时机**: Agent 启动时

**2. GetStatistics() - 读取统计**

```go
// 文件: src/agent/pkg/dataplane/dataplane.go:216
func (dp *DataPlane) GetStatistics() Statistics
```
- **作用**: 从 Per-CPU 数组读取计数器并求和
- **流程**:
  1. 对每个统计类型（8 个）
  2. 调用 `statsMap.Lookup()` 读取 Per-CPU 值
  3. 求和所有 CPU 的计数
- **返回**: Statistics 结构（总包数、允许数、拒绝数等）
- **耗时**: <1ms
- **调用时机**: API 查询或定期打印

**3. MonitorFlowEvents() - 事件监控**

```go
// 文件: src/agent/pkg/dataplane/dataplane.go:246
func (dp *DataPlane) MonitorFlowEvents()
```
- **作用**: 从 Ring Buffer 读取流事件并记录日志
- **流程**:
  1. 无限循环 `rbReader.Read()`
  2. 解析二进制数据（flow_event 结构）
  3. 打印日志
- **运行方式**: 后台 goroutine
- **调用时机**: Agent 启动后立即启动

#### 数据结构

```go
// src/agent/pkg/dataplane/dataplane.go:22-29

type DataPlane struct {
    objs     *bpfObjects       // eBPF 对象（maps + programs）
    iface    string            // 网卡名称（例如 "lo"）
    tcLink   link.Link         // TC 链接（用于清理）
    rbReader *ringbuf.Reader   // Ring Buffer 读取器
}

type Statistics struct {
    TotalPackets    uint64  // 总数据包数
    AllowedPackets  uint64  // 允许的数据包数
    DeniedPackets   uint64  // 拒绝的数据包数
    NewSessions     uint64  // 新会话数
    ClosedSessions  uint64  // 关闭的会话数
    ActiveSessions  uint64  // 活跃会话数
    PolicyHits      uint64  // 策略命中数
    PolicyMisses    uint64  // 策略未命中数
}
```

### 6.2 Policy 模块（pkg/policy）

#### 核心职责
- 策略 CRUD（增删改查）
- 精确匹配和通配符策略管理
- SQLite 持久化

#### 关键方法

**1. AddPolicy() - 添加策略**

```go
// 文件: src/agent/pkg/policy/policy.go:82
func (pm *PolicyManager) AddPolicy(p *Policy) error
```
- **作用**: 添加策略到 eBPF map
- **流程**:
  1. 调用 `addPolicyToMap()`
  2. 根据是否有通配符分支处理
  3. 持久化到 SQLite（可选）
- **支持**: 精确匹配 + 通配符
- **耗时**: <1ms（精确）, <10ms（通配符）

**2. DeletePolicy() - 删除策略**

```go
// 文件: src/agent/pkg/policy/policy.go:203
func (pm *PolicyManager) DeletePolicy(p *Policy) error
```
- **作用**: 从 eBPF map 删除策略
- **流程**:
  1. 构建 policy_key
  2. 调用 `policyMap.Delete()`
  3. 从 SQLite 删除（可选）

**3. ListPolicies() - 列出所有策略**

```go
// 文件: src/agent/pkg/policy/policy.go:256
func (pm *PolicyManager) ListPolicies() ([]*Policy, error)
```
- **作用**: 遍历 eBPF map 并转换为 Policy 列表
- **流程**:
  1. 遍历 policy_map（精确匹配）
  2. 遍历 wildcard_policy_map（通配符）
  3. 合并结果
- **耗时**: <10ms（10,000 条策略）

#### 数据结构

```go
// src/agent/pkg/policy/policy.go:13-27

type Policy struct {
    RuleID   uint32  // 规则 ID（唯一）
    SrcIP    string  // 源 IP (CIDR 格式，例如 "10.0.1.0/24")
    DstIP    string  // 目标 IP
    SrcPort  int     // 源端口 (0 = 任意)
    DstPort  int     // 目标端口 (0 = 任意)
    Protocol string  // 协议 ("tcp", "udp", "icmp", "any")
    Action   string  // 动作 ("allow", "deny", "log")
    Priority int     // 优先级 (值越大越优先)
}

type PolicyManager struct {
    policyMap         *ebpf.Map  // 精确匹配 map
    wildcardPolicyMap *ebpf.Map  // 通配符 map
    storage           Storage    // SQLite 持久化（可选）
}
```

### 6.3 API 模块（pkg/api）

#### 核心职责
- 提供 REST API 接口
- 路由和中间件管理
- 请求验证和错误处理

#### 路由列表

| 方法 | 路径 | 处理器 | 作用 |
|------|------|--------|------|
| GET | /health | healthHandler.GetHealth | 健康检查 |
| GET | /api/v1/status | healthHandler.GetStatus | 状态查询 |
| POST | /api/v1/policies | policyHandler.CreatePolicy | 创建策略 |
| GET | /api/v1/policies | policyHandler.ListPolicies | 列出策略 |
| GET | /api/v1/policies/:id | policyHandler.GetPolicy | 查询策略 |
| PUT | /api/v1/policies/:id | policyHandler.UpdatePolicy | 更新策略 |
| DELETE | /api/v1/policies/:id | policyHandler.DeletePolicy | 删除策略 |
| GET | /api/v1/stats | statsHandler.GetAllStats | 获取统计 |
| GET | /api/v1/stats/packets | statsHandler.GetPacketStats | 包统计 |
| GET | /api/v1/stats/sessions | statsHandler.GetSessionStats | 会话统计 |
| GET | /api/v1/stats/policies | statsHandler.GetPolicyStats | 策略统计 |

#### 中间件

```go
// src/agent/pkg/api/middleware.go

1. LoggingMiddleware()   - 请求日志（记录 IP、方法、路径、耗时）
2. CORSMiddleware()      - 跨域支持
3. RecoveryMiddleware()  - Panic 恢复
```

#### API 示例

**创建策略**
```bash
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "rule_id": 100,
    "src_ip": "10.0.1.0/24",
    "dst_ip": "10.0.2.2/32",
    "dst_port": 3306,
    "protocol": "tcp",
    "action": "allow",
    "priority": 100
  }'

# 响应
{
  "rule_id": 100,
  "src_ip": "10.0.1.0/24",
  "dst_ip": "10.0.2.2/32",
  "dst_port": 3306,
  "protocol": "tcp",
  "action": "allow",
  "priority": 100,
  "created_at": "2025-11-02T10:00:00Z"
}
```

**查询统计**
```bash
curl http://localhost:8080/api/v1/stats

# 响应
{
  "total_packets": 1234567,
  "allowed_packets": 1234500,
  "denied_packets": 67,
  "new_sessions": 1000,
  "active_sessions": 500,
  "policy_hits": 1234567,
  "policy_misses": 0
}
```

### 6.4 Workload 模块（pkg/workload）

#### 核心职责
- 工作负载数据模型（新增功能）
- 标签系统基础
- 为未来的基于标签的策略做准备

#### 数据结构

```go
// src/agent/pkg/workload/types.go:24-48

type Workload struct {
    // 身份信息
    ID     string  `json:"id"`       // 唯一标识符
    Name   string  `json:"name"`     // 工作负载名称
    HostID string  `json:"host_id"`  // 主机标识符

    // 网络信息
    IPs   []net.IP `json:"ips"`   // IP 地址列表
    MACs  []string `json:"macs"`  // MAC 地址列表
    Ports []uint16 `json:"ports"` // 监听端口列表

    // 标签（核心）
    Labels map[string]string `json:"labels"`  // 键值对标签

    // 元数据
    Image       string `json:"image"`        // 容器镜像
    Namespace   string `json:"namespace"`    // K8s 命名空间
    ServiceName string `json:"service_name"` // 服务名称
    PodName     string `json:"pod_name"`     // Pod 名称

    // 状态
    State     WorkloadState `json:"state"`      // running/stopped/paused
    CreatedAt time.Time     `json:"created_at"`
    UpdatedAt time.Time     `json:"updated_at"`
}
```

#### 标签操作

```go
// 添加标签
wl.AddLabel("role", "web")
wl.AddLabel("env", "prod")

// 删除标签
wl.RemoveLabel("role")

// 查询标签
value, exists := wl.GetLabel("role")
if wl.HasLabel("env") {
    // ...
}
```

---

## 7. 关键数据结构

### 7.1 eBPF Maps 详解

#### session_map（会话跟踪）

```c
// 类型: LRU_HASH（自动淘汰最久未使用）
// 最大条目: 100,000
// Key: flow_key (20 bytes)
// Value: session_value (64 bytes)

struct flow_key {
    __u32 src_ip;     // 源 IP (网络字节序)
    __u32 dst_ip;     // 目标 IP (网络字节序)
    __u16 src_port;   // 源端口 (网络字节序)
    __u16 dst_port;   // 目标端口 (网络字节序)
    __u8  protocol;   // 6=TCP, 17=UDP, 1=ICMP
    __u8  pad[3];     // 填充到 8 字节对齐
};

struct session_value {
    __u64 created_ts;          // 创建时间戳 (纳秒)
    __u64 last_seen_ts;        // 最后活跃时间 (纳秒)
    __u64 packets_to_server;   // 客户端→服务器包数
    __u64 packets_to_client;   // 服务器→客户端包数
    __u64 bytes_to_server;     // 客户端→服务器字节数
    __u64 bytes_to_client;     // 服务器→客户端字节数
    __u8  state;               // 会话状态 (NEW, ESTABLISHED, CLOSED)
    __u8  tcp_state;           // TCP 状态机 (SYN_SENT, ESTABLISHED, ...)
    __u8  policy_action;       // ⭐ 缓存的策略决策 (0=allow, 1=deny)
    __u8  flags;               // 标志位
    __u32 pad;                 // 填充
};

// 作用：性能关键！
// - 首包：查策略（慢）→ 创建会话 → 缓存决策
// - 后续包：直接读 policy_action（快，>99% 的包）
```

#### policy_map（精确匹配策略）

```c
// 类型: HASH
// 最大条目: 10,000
// Key: policy_key (20 bytes，同 flow_key)
// Value: policy_value (16 bytes)

struct policy_key {
    __u32 src_ip;     // 必须精确匹配（/32）
    __u32 dst_ip;     // 必须精确匹配（/32）
    __u16 src_port;   // 0 = 任意
    __u16 dst_port;   // 指定端口
    __u8  protocol;   // 6=TCP, 17=UDP, 0=任意
    __u8  pad[3];
};

struct policy_value {
    __u8  action;       // 0=allow, 1=deny, 2=log
    __u8  log_enabled;  // 是否记录日志
    __u16 priority;     // 优先级（值越大越优先）
    __u32 rule_id;      // 规则 ID（用于溯源）
    __u64 hit_count;    // 命中计数（统计）
};

// 查找复杂度: O(1) - 哈希表
// 使用场景: 精确 5-tuple 匹配
// 示例: 10.0.1.10:80 → 10.0.2.2:3306 TCP allow
```

#### wildcard_policy_map（通配符策略）

```c
// 类型: ARRAY（固定大小）
// 最大条目: 1,000
// Key: __u32 (索引 0-999)
// Value: wildcard_policy (32 bytes)

struct wildcard_policy {
    __u32 src_ip;       // 源 IP（归一化）
    __u32 src_ip_mask;  // 源 IP 掩码 (/24 → 0xFFFFFF00, /0 → 0x00000000)
    __u32 dst_ip;       // 目标 IP（归一化）
    __u32 dst_ip_mask;  // 目标 IP 掩码
    __u16 src_port;     // 源端口 (0 = 任意)
    __u16 dst_port;     // 目标端口 (0 = 任意)
    __u8  protocol;     // 协议 (0 = 任意)
    __u8  action;       // 0=allow, 1=deny, 2=log
    __u8  log_enabled;  // 是否记录日志
    __u8  pad1;
    __u16 priority;     // 优先级（值越大越优先）
    __u16 pad2;
    __u32 rule_id;      // 规则 ID（0 = 空槽位）
};

// 查找复杂度: O(n) - 线性搜索（n ≤ 1000）
// 优化: 使用 #pragma unroll 展开循环，提前退出
// 使用场景: CIDR、端口范围、协议通配
// 示例: 10.0.1.0/24:* → *:3306 TCP deny
```

#### stats_map（统计计数器）

```c
// 类型: PERCPU_ARRAY（每个 CPU 独立计数）
// 最大条目: 8
// Key: __u32 (统计类型枚举)
// Value: __u64 (计数器)

enum stats_key {
    STATS_TOTAL_PACKETS = 0,    // 总数据包数
    STATS_ALLOWED_PACKETS,      // 允许的数据包数
    STATS_DENIED_PACKETS,       // 拒绝的数据包数
    STATS_NEW_SESSIONS,         // 新会话数
    STATS_CLOSED_SESSIONS,      // 关闭的会话数
    STATS_ACTIVE_SESSIONS,      // 活跃会话数
    STATS_POLICY_HITS,          // 策略命中数
    STATS_POLICY_MISSES,        // 策略未命中数（使用默认策略）
    STATS_MAX,
};

// 为什么使用 PERCPU_ARRAY？
// - 避免原子操作开销（每个 CPU 独立写入）
// - 用户空间读取时求和所有 CPU 的值
```

#### flow_events（流事件 Ring Buffer）

```c
// 类型: RINGBUF
// 大小: 256 KB
// Entry: flow_event (48 bytes)

struct flow_event {
    struct flow_key key;  // 20 bytes - 5-tuple
    __u64 timestamp;      // 8 bytes - 时间戳
    __u64 packets;        // 8 bytes - 包数
    __u64 bytes;          // 8 bytes - 字节数
    __u8  action;         // 1 byte - 策略动作
    __u8  event_type;     // 1 byte - 事件类型 (0=new, 1=close, 2=deny)
    __u16 pad;            // 2 bytes - 填充
};

// 使用场景:
// - 新会话创建（DENY 或 LOG 动作）
// - 会话关闭
// - 异常事件
//
// 读取方式:
// - 用户空间 goroutine 阻塞读取
// - Ring Buffer 满时丢弃旧事件
```

### 7.2 Go 结构体

```go
// Statistics - 统计信息
type Statistics struct {
    TotalPackets    uint64
    AllowedPackets  uint64
    DeniedPackets   uint64
    NewSessions     uint64
    ClosedSessions  uint64
    ActiveSessions  uint64
    PolicyHits      uint64
    PolicyMisses    uint64
}

// Policy - 策略规则
type Policy struct {
    RuleID   uint32
    SrcIP    string  // CIDR 格式
    DstIP    string
    SrcPort  int
    DstPort  int
    Protocol string  // "tcp", "udp", "icmp", "any"
    Action   string  // "allow", "deny", "log"
    Priority int
}

// Workload - 工作负载（新增）
type Workload struct {
    ID          string
    Name        string
    HostID      string
    IPs         []net.IP
    MACs        []string
    Ports       []uint16
    Labels      map[string]string  // 标签系统
    Image       string
    Namespace   string
    ServiceName string
    PodName     string
    State       WorkloadState
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

---

## 8. 学习路径建议

### 8.1 新手入门路径（3 个阶段）

#### 阶段 1: 建立整体认知（1-2 小时）

**目标**: 理解架构和执行流程

1. **阅读本文档**:
   - 第 1-2 章：架构概览和快速入门
   - 第 3 章：启动流程
   - 跳过细节，先建立全局观

2. **动手实验**:
   ```bash
   # 编译项目
   cd src/agent
   go build -o microsegment-agent cmd/main.go

   # 启动 Agent（本地回环）
   sudo ./microsegment-agent -i lo --log-level=info

   # 观察日志输出
   # - 看到 "DataPlane initialized"
   # - 看到 "API server listening on 127.0.0.1:8080"
   ```

3. **调用 API**:
   ```bash
   # 查看状态
   curl http://localhost:8080/health

   # 查看统计
   curl http://localhost:8080/api/v1/stats
   ```

**输出**: 对系统有感性认识，知道 Agent 做了什么

---

#### 阶段 2: 理解核心流程（2-3 小时）

**目标**: 掌握策略管理和数据包处理

1. **代码阅读顺序**:
   ```
   1. src/agent/cmd/main.go (runAgent)
      ├─> 理解启动流程
      └─> 关注 dataplane.New() 和 api.NewAPIServer()

   2. src/agent/pkg/dataplane/dataplane.go (New, GetStatistics)
      ├─> 理解 eBPF 加载
      └─> 理解统计读取

   3. src/agent/pkg/policy/policy.go (AddPolicy, addExactPolicy)
      ├─> 理解策略添加
      └─> 理解 eBPF map 操作

   4. src/bpf/tc_microsegment.bpf.c (tc_microsegment_filter)
      ├─> 理解数据包处理流程
      └─> 关注会话查找和策略查找
   ```

2. **动手实验**:
   ```bash
   # 添加策略
   curl -X POST http://localhost:8080/api/v1/policies \
     -H "Content-Type: application/json" \
     -d '{
       "rule_id": 100,
       "src_ip": "127.0.0.1/32",
       "dst_ip": "127.0.0.1/32",
       "dst_port": 8080,
       "protocol": "tcp",
       "action": "allow",
       "priority": 100
     }'

   # 查看策略
   curl http://localhost:8080/api/v1/policies

   # 使用 bpftool 查看 eBPF map
   sudo bpftool map dump name policy_map
   ```

3. **调试技巧**:
   ```bash
   # 查看 eBPF 程序统计
   sudo bpftool prog show

   # 查看 map 详情
   sudo bpftool map list

   # 监控流事件（观察 Agent 日志）
   # 生成流量：curl http://localhost:8080/health
   ```

**输出**: 能独立添加策略，理解策略如何生效

---

#### 阶段 3: 深入细节（3-4 小时）

**目标**: 理解通配符策略、性能优化、边界情况

1. **深入阅读**:
   - 第 4.3 节：通配符策略实现
   - 第 5.3-5.4 节：策略查找和通配符匹配
   - 第 7.1 节：eBPF Maps 详解

2. **实验通配符策略**:
   ```bash
   # 添加通配符策略
   curl -X POST http://localhost:8080/api/v1/policies \
     -d '{
       "rule_id": 200,
       "src_ip": "10.0.1.0/24",
       "dst_ip": "0.0.0.0/0",
       "dst_port": 0,
       "protocol": "any",
       "action": "deny",
       "priority": 50
     }'

   # 查看通配符 map
   sudo bpftool map dump name wildcard_policy_map
   ```

3. **性能测试**:
   ```bash
   # 运行基准测试
   cd src/agent/test/benchmark
   go test -bench=. -benchmem
   ```

**输出**: 理解系统实现细节，能处理复杂场景

---

### 8.2 代码阅读顺序推荐

#### 快速路径（2 小时）

```
1. src/agent/cmd/main.go (150 行)
   - 关注 runAgent() 函数
   - 理解模块初始化顺序

2. src/bpf/tc_microsegment.bpf.c (800 行)
   - 关注 tc_microsegment_filter() 主函数
   - 关注 lookup_policy_action() 策略查找
   - 跳过辅助函数细节

3. src/agent/pkg/api/router.go (60 行)
   - 理解 API 路由
   - 了解有哪些接口可用
```

#### 完整路径（6 小时）

```
【启动流程】
1. src/agent/cmd/main.go

【数据平面】
2. src/agent/pkg/dataplane/dataplane.go (New, GetStatistics, MonitorFlowEvents)
3. src/bpf/tc_microsegment.bpf.c (tc_microsegment_filter, lookup_policy_action)
4. src/bpf/headers/common_types.h (数据结构定义)

【策略管理】
5. src/agent/pkg/policy/policy.go (AddPolicy, DeletePolicy, ListPolicies)
6. src/agent/pkg/policy/storage.go (SQLite 持久化)

【API 层】
7. src/agent/pkg/api/server.go (NewAPIServer, Start)
8. src/agent/pkg/api/router.go (setupRoutes)
9. src/agent/pkg/api/handlers/policy.go (API 处理器)

【标签系统】
10. src/agent/pkg/workload/types.go (Workload 数据模型)
11. src/agent/pkg/workload/types_test.go (单元测试，理解用法)
```

#### 深入路径（10+ 小时）

```
在完整路径基础上，额外阅读：

【eBPF 细节】
- extract_flow_key() - 数据包解析
- matches_wildcard() - 通配符匹配算法
- create_session() - 会话创建和事件上报
- update_stats() - 统计更新

【API 细节】
- src/agent/pkg/api/middleware.go (中间件)
- src/agent/pkg/api/models/* (数据模型)
- src/agent/pkg/api/handlers/* (所有处理器)

【测试代码】
- src/agent/test/e2e/* (端到端测试)
- src/agent/test/benchmark/* (性能测试)
- src/agent/pkg/*/\*_test.go (单元测试)
```

### 8.3 调试技巧

#### eBPF 调试

```bash
# 1. 查看加载的 eBPF 程序
sudo bpftool prog list

# 2. 查看程序统计
sudo bpftool prog show id <PROG_ID>

# 3. 查看 map 内容
sudo bpftool map list
sudo bpftool map dump name session_map
sudo bpftool map dump name policy_map

# 4. 查看 map 详细信息
sudo bpftool map show name policy_map

# 5. 实时跟踪 eBPF 输出（如果使用 bpf_printk）
sudo cat /sys/kernel/debug/tracing/trace_pipe
```

#### Go 程序调试

```bash
# 1. 使用 delve 调试器
go install github.com/go-delve/delve/cmd/dlv@latest
sudo dlv exec ./microsegment-agent -- -i lo

# 在 delve 中设置断点
(dlv) break dataplane.New
(dlv) break policy.AddPolicy
(dlv) continue

# 2. 使用 pprof 性能分析
# 在 main.go 中添加:
import _ "net/http/pprof"
go func() {
    http.ListenAndServe(":6060", nil)
}()

# 查看 CPU profile
go tool pprof http://localhost:6060/debug/pprof/profile
```

#### 日志技巧

```bash
# 1. 启用详细日志
sudo ./microsegment-agent -i lo --log-level=debug

# 2. 过滤特定日志
sudo ./microsegment-agent -i lo 2>&1 | grep "FLOW EVENT"

# 3. 保存日志到文件
sudo ./microsegment-agent -i lo 2>&1 | tee agent.log
```

---

## 9. 常见问题解答

### Q1: 为什么使用两层策略 Map（policy_map + wildcard_policy_map）？

**答**: 性能优化

- **精确匹配（policy_map）**: O(1) 哈希查找，适合固定 5-tuple
- **通配符匹配（wildcard_policy_map）**: O(n) 线性搜索，支持 CIDR/端口范围

**执行顺序**: 先查精确 map（快），未命中再查通配符 map（慢）

**类比**: 精确匹配是字典查词，通配符是正则匹配

---

### Q2: 会话如何自动淘汰？为什么不需要手动清理？

**答**: session_map 使用 **LRU_HASH** 类型

- **LRU (Least Recently Used)**: 最近最少使用淘汰策略
- **自动淘汰**: Map 满时（100,000 条目），自动删除最久未访问的会话
- **无需定时器**: 内核自动管理，无用户空间开销

**性能影响**: 淘汰后的首包需要重新查策略（可接受的代价）

---

### Q3: 通配符策略如何避免冲突？优先级如何工作？

**答**: 使用 **priority 字段 + 线性搜索**

**匹配逻辑**:
1. 遍历所有通配符策略（最多 1000 个）
2. 检查每个策略是否匹配
3. 选择 **优先级最高**（priority 值最大）的匹配
4. 如果优先级相同，选择 **最先匹配** 的

**示例**:
```
策略 1: 10.0.1.0/24 → * deny (priority=100)
策略 2: 10.0.1.10/32 → * allow (priority=200)

数据包: 10.0.1.10 → 10.0.2.2
匹配: 两个策略都匹配
决策: 策略 2 胜出 (priority=200 > 100) → allow
```

---

### Q4: eBPF 程序如何限制循环次数？为什么是 100 次？

**答**: **eBPF Verifier 限制**

- eBPF 程序必须可验证（不能有无限循环）
- Verifier 要求循环次数有上界
- 使用 `#pragma unroll` 提示编译器展开循环

**为什么是 100?**
- wildcard_policy_map 有 1000 个槽位
- 使用 `#pragma unroll` 限制每次最多遍历 100 个
- 需要 10 次循环才能遍历完（实际会提前退出）

**优化空间**:
- 使用二分查找（需要排序）
- 使用 Trie 数据结构（需要更复杂的 eBPF 代码）

---

### Q5: Per-CPU 数组如何避免竞态条件？

**答**: **每个 CPU 独立计数**

**原理**:
```
CPU 0:  stats_map[STATS_TOTAL_PACKETS] = 1000
CPU 1:  stats_map[STATS_TOTAL_PACKETS] = 2000
CPU 2:  stats_map[STATS_TOTAL_PACKETS] = 1500
...

用户空间读取: 1000 + 2000 + 1500 + ... = 总数
```

**优势**:
- **无锁**: 每个 CPU 写自己的副本，无竞争
- **高性能**: 避免原子操作开销（可节省 10-20% CPU）
- **精确**: 读取时求和所有 CPU 的值

---

### Q6: 如何处理 IP 分片？

**答**: **当前版本不支持**

**原因**:
- TC Ingress Hook 在 IP 重组之前触发
- 分片后的数据包没有完整的传输层头（端口信息）

**解决方案**（未来）:
1. 使用 **XDP** 或 **TC Egress** Hook（在 IP 重组之后）
2. 维护分片重组状态表
3. 仅匹配第一个分片（不推荐，可能被绕过）

---

### Q7: 为什么默认策略是 allow-all？如何修改？

**答**: **安全考虑**

**默认 allow-all 的理由**:
- 避免误配置导致网络中断
- MVP 阶段优先保证可用性
- 明确的 deny 策略优先级更高

**修改方法**:
```go
// src/agent/cmd/main.go:80

// 修改默认策略为 deny-all
defaultPolicy := &policy.Policy{
    RuleID:   1,
    SrcIP:    "0.0.0.0/0",
    DstIP:    "0.0.0.0/0",
    Protocol: "any",
    Action:   "deny",  // 改为 deny
    Priority: 1,       // 最低优先级
}
```

**生产环境建议**:
1. 默认 deny-all
2. 逐步添加 allow 规则
3. 监控 denied_packets 统计

---

### Q8: 如何区分入站和出站流量？

**答**: **当前版本仅处理 Ingress（入站）**

**原因**:
- TC Ingress Hook 仅拦截入站流量
- 需要附加到 TC Egress Hook 才能处理出站

**扩展方法**:
```c
// 添加 Egress 程序
SEC("tc")
int tc_microsegment_filter_egress(struct __sk_buff *skb) {
    // 类似的处理逻辑
    // 但方向相反（src/dst 互换）
}
```

**实际应用**:
- 入站策略：防止未授权访问
- 出站策略：防止数据泄露

---

### Q9: eBPF 程序如何更新？需要重启吗？

**答**: **无需重启，实时更新**

**Map 更新**:
- 用户空间调用 `bpf(BPF_MAP_UPDATE_ELEM)` 系统调用
- eBPF 程序立即看到新数据
- 无需重载程序

**程序更新**:
- 需要重新编译 eBPF 程序（.c → .o）
- 需要重启 Agent（重新加载）
- 会话状态会丢失

**最佳实践**:
- 策略变更：仅更新 Map（无影响）
- 程序逻辑变更：需要重启（影响流量）

---

### Q10: 性能瓶颈在哪里？如何优化？

**答**: **三个关键路径**

**1. 热路径（>99% 数据包）**:
- 会话查找：O(1) 哈希查找
- **瓶颈**: session_map 大小（100k 条目）
- **优化**: 增加 session_map 大小，调整 LRU 策略

**2. 冷路径（<1% 数据包）**:
- 策略查找：O(1) 精确 + O(n) 通配符
- **瓶颈**: 通配符线性搜索（1000 个策略）
- **优化**:
  - 减少通配符策略数量
  - 提高优先级，尽早匹配
  - 使用 LPM Trie（未来）

**3. 用户空间**:
- API 响应：<10ms
- **瓶颈**: SQLite 写入（持久化）
- **优化**: 批量写入，异步持久化

**基准测试**:
```bash
cd src/agent/test/benchmark
go test -bench=. -benchmem

# 预期性能:
# - 会话查找: ~100ns
# - 精确策略查找: ~200ns
# - 通配符策略查找: ~5μs (100 个策略)
```

---

## 附录 A: 术语表

| 术语 | 英文 | 解释 |
|------|------|------|
| TC | Traffic Control | Linux 流量控制子系统 |
| eBPF | extended Berkeley Packet Filter | 内核可编程技术 |
| 5-tuple | 5-tuple | 源 IP、目标 IP、源端口、目标端口、协议 |
| CIDR | Classless Inter-Domain Routing | 无类别域间路由（例如 10.0.0.0/24） |
| LRU | Least Recently Used | 最近最少使用淘汰策略 |
| Ring Buffer | Ring Buffer | 环形缓冲区，用于内核向用户空间传递事件 |
| Ingress | Ingress | 入站流量（进入网卡） |
| Egress | Egress | 出站流量（离开网卡） |
| Map | eBPF Map | eBPF 程序和用户空间共享数据的机制 |
| Verifier | eBPF Verifier | 内核 eBPF 程序验证器，确保程序安全 |
| Per-CPU | Per-CPU | 每个 CPU 独立一份数据，避免竞态 |
| Wildcard | Wildcard | 通配符，例如 `*` 表示任意 |
| Hot Path | Hot Path | 热路径，频繁执行的代码路径 |
| Cold Path | Cold Path | 冷路径，偶尔执行的代码路径 |

---

## 附录 B: 推荐阅读

### eBPF 学习资源

1. **BPF Performance Tools** by Brendan Gregg
   - 第 2 章：eBPF 基础
   - 第 8 章：网络过滤

2. **Linux Observability with BPF**
   - 第 3 章：eBPF Maps
   - 第 7 章：TC 和 XDP

3. **Cilium eBPF 文档**
   - https://ebpf-go.dev/
   - Go eBPF 库官方文档

### Linux 网络

1. **Understanding Linux Network Internals**
   - 第 10 章：Traffic Control

2. **Linux Network Stack**
   - https://wiki.linuxfoundation.org/networking/kernel_flow

### 微隔离技术

1. **Illumio Adaptive Security Platform**
   - 工作负载可见性
   - 基于标签的策略

2. **NeuVector 文档**
   - 容器网络安全
   - 自动策略生成

---

## 结语

这份文档旨在帮助你快速建立对 eBPF 微隔离项目的全面理解。建议按照以下顺序学习：

1. **通读全文**（1-2 小时）- 建立整体认知
2. **动手实验**（1 小时）- 启动 Agent，调用 API
3. **阅读代码**（4-6 小时）- 按推荐顺序深入源码
4. **修改测试**（2-3 小时）- 添加新策略，观察效果

完成后，你将能够：
- ✅ 理解系统整体架构
- ✅ 独立添加和调试策略
- ✅ 修改 eBPF 程序
- ✅ 扩展 API 功能
- ✅ 排查性能问题

如有疑问，请参考源码注释或查阅 eBPF 官方文档。

祝学习愉快！🚀
