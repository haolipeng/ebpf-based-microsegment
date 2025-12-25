# NeuVector 源码分析文档

> NeuVector 是一个云原生容器安全平台，本目录包含对其核心功能的深度技术分析

---

## 📚 文档目录

### 1. [dp 组件编译指南](neuvector-dp-build-guide.md) (12KB)
**主题**: NeuVector dp (数据平面) 组件的编译和依赖

**内容**:
- dp 组件功能介绍（DPI、策略执行、会话跟踪）
- 完整的依赖安装脚本（8 个依赖库）
- 详细的编译步骤
- 常见编译问题及解决方案

**适用场景**:
- 需要编译和研究 NeuVector dp 组件
- 了解 dp 的技术栈和依赖关系

**关键依赖**:
- liburcu (RCU 无锁并发)
- libhyperscan (高性能正则匹配)
- libnetfilter-queue (内核包捕获)
- jansson (JSON 解析)

---

### 2. [Agent-dp 通信机制](neuvector-dp-agent-communication.md) (40KB) ⭐⭐⭐⭐⭐
**主题**: Agent (Go) 和 dp (C) 之间的通信协议

**内容**:
- Unix Domain Socket 通信架构
- JSON 命令协议（30+ 控制命令）
- Binary 数据上报协议
- Standalone Mode（独立测试模式）
- 完整的代码示例（Go + C）

**适用场景**:
- 理解用户态控制程序和内核态数据平面的交互
- 学习进程间通信的最佳实践
- 调试和测试 dp 组件

**关键技术**:
- Unix Socket (SOCK_DGRAM)
- JSON 序列化（jansson 库）
- 二进制消息格式（C 结构体）

**重要文件**:
- `agent/dp/ctrl.go` - Agent 侧通信代码
- `dp/ctrl.c` - dp 侧控制循环和消息处理

---

### 3. [Agent-dp 策略分发流程](neuvector-agent-dp-policy-flow.md) (46KB) ⭐⭐⭐⭐⭐
**主题**: 从 Controller 到 Kernel 的完整策略下发流程

**内容**:
- 6 步策略分发流程（Controller → Agent → dp → Kernel）
- 每一步的数据结构转换
- Go 语言侧的策略计算逻辑
- C 语言侧的策略编译和安装
- 4 个完整的数据包匹配场景

**适用场景**:
- 理解微隔离策略如何生效
- 学习分布式策略管理架构
- 优化策略下发性能

**关键数据结构**:
- Go 侧: `DPWorkloadIPPolicy`, `DPPolicyIPRule`
- C 侧: `dpi_policy_t`, `dpi_policy_rule_t`
- Protobuf: `CLUSWorkload`, `CLUSPolicyRule`

**核心流程**:
```
Controller 计算策略
    ↓ gRPC (Protobuf)
Agent 接收并转换
    ↓ Unix Socket (JSON)
dp 解析 JSON
    ↓ 编译为内部结构
Kernel 执行策略
```

---

### 4. [FQDN 域名过滤实现](neuvector-fqdn-implementation.md) (31KB) ⭐⭐⭐⭐
**主题**: 基于域名的网络访问控制实现

**内容**:
- FQDN 双向映射机制（FQDN ↔ IP）
- DNS 响应拦截和学习
- 通配符匹配（*.example.com）
- Code 机制（域名转整数，快速匹配）
- Virtual Host 模式（一个 IP 多个域名）

**适用场景**:
- 实现基于域名的访问控制
- 学习 DNS 拦截技术
- 处理动态 IP 场景（CDN）

**关键技术**:
- DNS 包解析（A/AAAA/CNAME 记录）
- RCU 无锁哈希表
- 通配符匹配算法

**核心数据结构**:
```c
typedef struct fqdn_record_ {
    char name[MAX_FQDN_LEN];   // 域名
    uint32_t code;             // 唯一 ID（快速匹配）
    uint32_t flag;             // WILDCARD / TO_DELETE
    uint32_t ip_cnt;           // 关联 IP 数量
    struct cds_list_head iplist;  // IP 列表
    bool vh;                   // Virtual Host 模式
} fqdn_record_t;
```

---

### 5. [网络拓扑图实现](neuvector-network-topology-implementation.md) (27KB) ⭐⭐⭐⭐⭐
**主题**: 网络流量拓扑可视化的完整技术方案

**内容**:
- 4 层架构（dp → Agent → Controller → Web UI）
- 数据采集层（会话跟踪和上报）
- 图数据存储（内存图数据库）
- 拓扑构建逻辑（节点识别和边聚合）
- REST API 设计
- 前端可视化方案（D3.js 示例）

**适用场景**:
- 实现网络流量拓扑可视化
- 微服务依赖关系梳理
- 东西向流量监控
- 零信任网络策略生成

**关键技术**:
- 内存图数据库（Go map 实现）
- 多重有向图（3 种边类型：policy / graph / attr）
- 多层数据聚合（详细条目 → 会话聚合 → 拓扑图）
- RESTful API + JSON
- D3.js 力导向图

**数据流**:
```
dp 会话跟踪 (DPMsgSession)
    ↓ Unix Socket
Agent 数据转换 (CLUSConnection)
    ↓ gRPC
Controller 图数据库 (Graph)
    ↓ REST API
Web UI 可视化 (D3.js)
```

---

## 🗂️ 文档分类

### 按功能分类

#### 数据平面 (Data Plane)
- [dp 组件编译指南](neuvector-dp-build-guide.md)
- [Agent-dp 通信机制](neuvector-dp-agent-communication.md)
- [Agent-dp 策略分发流程](neuvector-agent-dp-policy-flow.md)

#### 高级功能
- [FQDN 域名过滤实现](neuvector-fqdn-implementation.md)
- [网络拓扑图实现](neuvector-network-topology-implementation.md)

### 按重要性分类

#### ⭐⭐⭐⭐⭐ 必读（核心原理）
1. [Agent-dp 策略分发流程](neuvector-agent-dp-policy-flow.md) - 理解策略如何生效
2. [Agent-dp 通信机制](neuvector-dp-agent-communication.md) - 理解进程间通信
3. [网络拓扑图实现](neuvector-network-topology-implementation.md) - 理解流量可视化

#### ⭐⭐⭐⭐ 重要（高级功能）
4. [FQDN 域名过滤实现](neuvector-fqdn-implementation.md) - 域名访问控制

#### ⭐⭐⭐ 参考（工具和环境）
5. [dp 组件编译指南](neuvector-dp-build-guide.md) - 编译和调试

### 按学习路径

#### 路径 1: 快速理解架构（2-3 小时）
1. [Agent-dp 通信机制](neuvector-dp-agent-communication.md) (30 分钟)
2. [Agent-dp 策略分发流程](neuvector-agent-dp-policy-flow.md) (40 分钟)
3. [网络拓扑图实现](neuvector-network-topology-implementation.md) (30 分钟)

#### 路径 2: 深入技术细节（4-6 小时）
1. [dp 组件编译指南](neuvector-dp-build-guide.md) (20 分钟) - 环境搭建
2. [Agent-dp 通信机制](neuvector-dp-agent-communication.md) (1 小时) - 通信协议
3. [Agent-dp 策略分发流程](neuvector-agent-dp-policy-flow.md) (1.5 小时) - 策略流程
4. [FQDN 域名过滤实现](neuvector-fqdn-implementation.md) (1 小时) - FQDN 功能
5. [网络拓扑图实现](neuvector-network-topology-implementation.md) (1 小时) - 拓扑图

#### 路径 3: 特定功能研究
- **研究策略引擎**: 2 → 3
- **研究通信机制**: 2
- **研究 FQDN 功能**: 4
- **研究拓扑可视化**: 5

---

## 🔍 关键技术总结

### 通信和协议
| 技术 | 用途 | 相关文档 |
|------|------|---------|
| Unix Domain Socket | Agent ↔ dp 通信 | 文档 2 |
| gRPC + Protobuf | Controller ↔ Agent 通信 | 文档 3 |
| JSON | 控制命令序列化 | 文档 2 |
| Binary | 数据上报序列化 | 文档 2 |

### 数据结构
| 技术 | 用途 | 相关文档 |
|------|------|---------|
| RCU 哈希表 | 无锁并发数据结构 | 文档 4 |
| 内存图数据库 | 拓扑存储 | 文档 5 |
| LRU Map | 会话跟踪 | 文档 3 |

### 核心算法
| 技术 | 用途 | 相关文档 |
|------|------|---------|
| 5 元组匹配 | 策略匹配 | 文档 3 |
| DNS 解析 | FQDN 学习 | 文档 4 |
| 图遍历 | 拓扑构建 | 文档 5 |
| 通配符匹配 | 域名匹配 | 文档 4 |

### 性能优化
| 技术 | 用途 | 相关文档 |
|------|------|---------|
| RCU | 无锁读写 | 文档 2, 4 |
| 批量处理 | 减少通信开销 | 文档 2, 3 |
| Code 机制 | 快速 FQDN 匹配 | 文档 4 |
| 多层聚合 | 减少查询次数 | 文档 5 |

---

## 📊 统计信息

| 指标 | 数值 |
|------|------|
| 文档总数 | 5 个 |
| 总字数 | ~156KB |
| 总行数 | ~7,800 行 |
| 代码示例 | 100+ 个 |
| 数据结构 | 50+ 个 |
| 架构图 | 20+ 个 |

---

## 🎯 适用项目

这些文档分析的技术可用于：

- ✅ 容器网络安全
- ✅ 微隔离系统
- ✅ 零信任架构
- ✅ 网络流量监控
- ✅ 服务网格（Service Mesh）
- ✅ 云原生安全平台

---

## 📝 使用建议

### 对于新手
1. 先阅读 [Agent-dp 通信机制](neuvector-dp-agent-communication.md) 理解架构
2. 再阅读 [Agent-dp 策略分发流程](neuvector-agent-dp-policy-flow.md) 理解核心功能
3. 最后根据兴趣选择其他文档

### 对于开发者
1. 按照学习路径 2 深入学习
2. 结合源代码阅读文档
3. 使用 [dp-diagnostic-tool](../dp-diagnostic-tool.md) 进行调试

### 对于架构师
1. 重点阅读架构设计部分
2. 关注性能优化技巧
3. 参考实现建议部分

---

## 🔗 相关资源

### 项目文档
- [项目 README](../../README.md)
- [架构设计文档](../../design-docs/architecture/)
- [dp 诊断工具](../dp-diagnostic-tool.md)

### NeuVector 官方
- [NeuVector 官网](https://neuvector.com/)
- [NeuVector GitHub](https://github.com/neuvector/neuvector)
- [NeuVector 文档](https://open-docs.neuvector.com/)

### 相关技术
- [eBPF 学习指南](../weekly-guide/)
- [性能优化文档](../PERFORMANCE.md)

---

## 📮 反馈和改进

如果你在阅读这些文档时发现：
- 技术错误或理解偏差
- 需要补充的内容
- 更好的组织方式

请通过以下方式反馈：
- 提交 Issue
- 提交 Pull Request
- 联系项目维护者

---

**文档整理时间**: 2025-10-31
**NeuVector 版本**: v5.x
**分析者**: eBPF 微隔离项目组

---

## 📖 快速导航

| 我想了解... | 推荐阅读 |
|------------|---------|
| NeuVector 整体架构 | 文档 2 → 文档 3 |
| 策略如何生效 | 文档 3 |
| Agent 和 dp 如何通信 | 文档 2 |
| 域名过滤如何实现 | 文档 4 |
| 拓扑图如何实现 | 文档 5 |
| 如何编译和调试 | 文档 1 → 文档 2 |
| 性能优化技巧 | 所有文档的"优化"章节 |
| 数据结构设计 | 文档 3 → 文档 4 → 文档 5 |

---

**Happy Learning! 🚀**
