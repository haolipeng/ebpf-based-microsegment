# eBPF 微隔离功能快速开始指南

> 📌 **快速参考**: 优先级排序、实施步骤、关键代码位置

---

## 🎯 优先级概览

```
P0 (必须完成)  →  P1 (重要增强)  →  P2 (可选功能)
    10-12 天           16 天              12-15 天
  核心隔离功能      高级流量控制        增强和优化
```

---

## ✅ 已完成功能清单（60-65%）

| ✓ | 功能 | 实现位置 |
|---|------|----------|
| ✅ | **基础数据平面** | `src/bpf/tc_microsegment.bpf.c` |
| ✅ | **会话跟踪** | `session_map` (LRU_HASH, 100K) |
| ✅ | **5元组匹配** | `policy_map` + `wildcard_policy_map` |
| ✅ | **TCP 状态机** | `src/bpf/headers/tcp_state_machine.h` |
| ✅ | **IPv4/IPv6** | 统一 128 位地址格式 |
| ✅ | **VLAN 支持** | 802.1Q/802.1ad 标签识别 |
| ✅ | **协议索引优化** | `src/bpf/headers/indexed_policy_match_v2.h` |
| ✅ | **会话超时** | `src/agent/pkg/session/timeout_manager.go` |
| ✅ | **Flow 事件** | Ring Buffer (`flow_events`) |
| ✅ | **统计收集** | Per-CPU (`stats_map`) |

---

## 🔴 P0 优先级 - 核心隔离功能（10-12 天）

### 1. NAT 支持和检测 (3-4 天) 🚨 最高优先级

**为什么重要**:
- Docker/K8s 大量使用 NAT
- 当前版本在 NAT 环境下策略匹配失败
- 阻塞生产环境部署

**实施步骤**:
```
Day 1: Conntrack 集成方案设计
  ├─ 研究 bpf_ct_lookup_tcp/udp() helper (Kernel >= 5.18)
  ├─ 设计用户态同步方案（兼容低版本内核）
  └─ 定义 NAT 状态缓存 Map 结构

Day 2: 实现 NAT 检测逻辑
  ├─ 创建 src/bpf/headers/nat_support.h
  ├─ 实现 lookup_conntrack() 函数
  ├─ 在策略匹配前调用 NAT 检测
  └─ 更新 session_value 存储原始地址

Day 3: 用户态集成
  ├─ 添加 NAT 配置选项（MATCH_ORIGINAL/MATCH_TRANSLATED）
  ├─ 用户态 conntrack 同步模块（如需要）
  └─ API 配置接口

Day 4: 测试和调优
  ├─ Docker 容器间通信测试
  ├─ Kubernetes Service 访问测试
  └─ 性能测试（NAT 查询开销 < 5μs）
```

**关键代码**:
```c
// src/bpf/headers/nat_support.h
static __always_inline bool lookup_conntrack(
    struct flow_key *key,
    struct flow_key *original_key)
{
    #ifdef HAVE_BPF_CT_LOOKUP  // Kernel >= 5.18
        struct bpf_ct_opts opts = {...};
        struct bpf_sock_tuple tuple = {...};
        struct nf_conn *ct = bpf_ct_lookup_tcp(&tuple, sizeof(tuple), &opts, sizeof(opts));
        if (ct) {
            // Extract original addresses from conntrack
            original_key->src_ip = ct->tuplehash[IP_CT_DIR_ORIGINAL].tuple.src;
            // ...
            bpf_ct_release(ct);
            return true;
        }
    #else
        // Fallback: lookup user-space synced conntrack map
        struct conntrack_entry *entry = bpf_map_lookup_elem(&conntrack_cache_map, key);
        if (entry) {
            *original_key = entry->original_key;
            return true;
        }
    #endif
    return false;
}
```

---

### 2. 分片数据包处理 (3-4 天)

**为什么重要**:
- 分片可用于绕过策略
- 后续分片缺少端口信息
- 安全风险

**实施步骤**:
```
Day 1: 分片跟踪 Map 设计
  ├─ 定义 frag_key / frag_value 结构
  ├─ 创建 fragment_track_map (LRU_HASH, 10K)
  └─ 设计分片超时清理机制

Day 2: 分片检测和解析
  ├─ 创建 src/bpf/headers/fragment_handling.h
  ├─ IPv4 分片检测（MF 标志、Offset）
  ├─ IPv6 分片检测（Fragment Extension Header）
  └─ 区分首个分片和后续分片

Day 3: 分片策略匹配
  ├─ 首个分片：正常策略匹配 + 存储到 Map
  ├─ 后续分片：查找首个分片的策略动作
  └─ 集成到主数据平面逻辑

Day 4: 测试和防护
  ├─ 生成分片测试包
  ├─ 测试分片重组攻击防护
  └─ 性能测试
```

**关键数据结构**:
```c
struct frag_key {
    __u32 src_ip[4];
    __u32 dst_ip[4];
    __u32 frag_id;      // IPv4: identification, IPv6: fragment ID
    __u8  protocol;
    __u8  ip_version;
};

struct frag_value {
    struct flow_key complete_key;  // 完整的 5 元组
    __u64 timestamp;               // 分片到达时间
    __u8  policy_action;           // 首个分片匹配的策略动作
};
```

---

### 3. 默认拒绝策略 (1-2 天)

**为什么重要**:
- 零信任架构核心原则
- 防止策略遗漏导致的安全漏洞

**实施步骤**:
```
Day 1: 默认策略配置
  ├─ 创建 default_policy_map (ARRAY, 1 entry)
  ├─ 定义 enum default_policy (ALLOW/DENY/LOG)
  ├─ 用户态 API: SetDefaultPolicy(action)
  └─ 集成到策略匹配失败逻辑

Day 2: 测试和验证
  ├─ 测试三种默认模式
  ├─ E2E 测试（无策略时的行为）
  └─ 审计日志验证
```

**关键代码**:
```c
// In tc_microsegment.bpf.c
if (matched_rule_id == 0) {
    // No policy matched, use default policy
    __u8 default_action = POLICY_ACTION_DENY;  // Zero Trust
    __u32 key = 0;
    __u8 *config = bpf_map_lookup_elem(&default_policy_map, &key);
    if (config) {
        default_action = *config;
    }

    if (default_action == POLICY_ACTION_DENY) {
        push_flow_event(..., FLOW_EVENT_DENIED_DEFAULT);
        return TC_ACT_SHOT;  // DROP
    }
}
```

---

### 4. ICMP 协议增强 (2 天)

**实施步骤**:
```
Day 1: ICMP 类型/代码解析
  ├─ 创建 src/bpf/headers/icmp_support.h
  ├─ 解析 ICMP/ICMPv6 头
  └─ 扩展 wildcard_policy 支持 ICMP 过滤

Day 2: ICMP 错误消息关联
  ├─ 解析 ICMP 错误消息的内嵌 IP 头
  ├─ 关联到原始会话
  └─ 测试（Ping/Traceroute/Path MTU Discovery）
```

---

### 5. 连接跟踪增强 (2-3 天)

**实施步骤**:
```
Day 1: TCP 重传检测
  ├─ 实现 is_tcp_retransmission() 函数
  ├─ 更新 session->tcp_retrans_count
  └─ 设置 CONN_FLAG_RETRANS 标志

Day 2: TCP 窗口管理
  ├─ 解析 TCP Window Scale 选项
  ├─ 计算实际窗口大小
  └─ 检测 Zero Window

Day 3: 测试
  ├─ 模拟丢包和重传
  └─ 性能测试
```

---

## 🟠 P1 优先级 - 高级功能（16 天）

### 6. 应用层协议识别 (4-5 天)

**业务价值**: 基于协议的细粒度策略

**实施顺序**:
```
Day 1-2: HTTP/HTTPS 识别
  ├─ HTTP methods 特征检测（GET/POST/...）
  ├─ TLS Client Hello 解析（SNI 提取）
  └─ 协议结果存储到 session_value

Day 3: DNS 识别
  ├─ DNS 查询特征检测
  └─ 提取查询域名（可选）

Day 4: SSH/MySQL 识别
  ├─ SSH banner 检测
  └─ MySQL handshake 检测

Day 5: 测试和优化
  ├─ 协议识别准确率测试
  └─ 性能影响测试
```

**关键代码**:
```c
enum app_protocol {
    APP_PROTO_UNKNOWN = 0,
    APP_PROTO_HTTP = 1,
    APP_PROTO_HTTPS = 2,
    APP_PROTO_DNS = 3,
    APP_PROTO_SSH = 4,
};

struct session_value {
    // ... existing fields
    __u8  app_protocol;    // 应用层协议
    __u8  proto_confidence; // 识别置信度 (0-100)
};

static __always_inline enum app_protocol identify_protocol(
    void *data, void *data_end, __u16 dport, __u8 protocol)
{
    // HTTP detection
    if (protocol == IPPROTO_TCP && (dport == 80 || dport == 8080)) {
        if (data + 4 <= data_end) {
            char *payload = (char *)data;
            if (payload[0] == 'G' && payload[1] == 'E' &&
                payload[2] == 'T' && payload[3] == ' ') {
                return APP_PROTO_HTTP;
            }
        }
    }
    // ... more protocols
    return APP_PROTO_UNKNOWN;
}
```

---

### 7. 进程级别策略关联 (3-4 天)

**业务价值**: 容器/Pod 级别隔离

**实施顺序**:
```
Day 1-2: cgroup/sock_ops Hook
  ├─ 添加 cgroup/connect4 程序
  ├─ 提取 cgroup_id 和 PID
  └─ 与 TC hook 的 session 关联

Day 3: 策略扩展
  ├─ wildcard_policy 添加 cgroup_id 过滤
  ├─ 策略匹配时的 cgroup 检查
  └─ 用户态 API（按容器配置策略）

Day 4: 测试
  ├─ Docker 容器测试
  └─ Kubernetes Pod 测试
```

---

### 8. 带宽限流 (3 天)

**业务价值**: DoS/DDoS 防护

**实施顺序**:
```
Day 1: Token Bucket 算法实现
  ├─ 创建 rate_limit_map
  ├─ 实现 check_rate_limit() 函数
  └─ Per-Flow 限流

Day 2: 配置和集成
  ├─ 用户态 API 配置限流规则
  ├─ 集成到数据平面
  └─ 超限处理（DROP/MARK）

Day 3: 测试
  ├─ 限流精度测试
  └─ 性能影响测试
```

---

### 9. 连接数限制 (2 天)

**业务价值**: 防止连接洪水攻击

**实施顺序**:
```
Day 1: 连接计数 Map
  ├─ 创建 conn_limit_map
  ├─ Per-IP 连接计数
  └─ Per-Service 连接计数

Day 2: 限制逻辑和测试
  ├─ 超限拒绝
  └─ 扫描检测
```

---

### 10. 策略模拟模式 (2 天)

**业务价值**: 策略变更前的验证

**实施顺序**:
```
Day 1: 模拟模式实现
  ├─ policy_value 添加 simulate_mode 字段
  ├─ 模拟统计 Map（would_allow/would_deny）
  └─ 策略匹配逻辑修改

Day 2: 用户态 API 和测试
  ├─ 启用/禁用模拟模式 API
  ├─ 查询模拟结果 API
  └─ E2E 测试
```

---

## 🟢 P2 优先级 - 增强功能（可选，12-15 天）

### 11. 多租户隔离 (4-5 天)
### 12. DDoS 防护增强 (5-6 天)
### 13. 加密流量检测 (3-4 天)

*详细计划见完整路线图文档*

---

## 🗓️ 推荐实施顺序

### Week 1-2: P0 核心功能（必须完成）

| 天数 | 任务 | 关键产出 |
|------|------|----------|
| Day 1-4 | NAT 支持 | Docker/K8s 环境策略正常 |
| Day 5-8 | 分片处理 | 防止分片绕过 |
| Day 9-10 | 默认拒绝 | 零信任模式 |
| Day 11-12 | ICMP 增强 | 完整 ICMP 支持 |
| Day 13-15 | 连接跟踪 | TCP 重传检测 |

**里程碑 M1**: 核心微隔离功能完整，可部署生产环境

---

### Week 3-5: P1 高级功能（重要增强）

| 天数 | 任务 | 关键产出 |
|------|------|----------|
| Day 1-5 | 协议识别 | HTTP/DNS/SSH 识别 |
| Day 6-9 | 进程关联 | 容器级别隔离 |
| Day 10-12 | 带宽限流 | DoS 防护 |
| Day 13-14 | 连接限制 | 连接洪水防护 |
| Day 15-16 | 策略模拟 | Dry-run 模式 |

**里程碑 M2**: 高级流量控制功能完整，产品竞争力提升

---

## 🔧 关键文件清单

### 新增 eBPF 头文件

| 文件 | 用途 |
|------|------|
| `src/bpf/headers/nat_support.h` | NAT 检测逻辑 |
| `src/bpf/headers/fragment_handling.h` | 分片处理 |
| `src/bpf/headers/icmp_support.h` | ICMP 增强 |
| `src/bpf/headers/protocol_detect.h` | 应用层协议识别 |
| `src/bpf/headers/rate_limit.h` | 带宽限流 |

### 新增用户态模块

| 文件/目录 | 用途 |
|----------|------|
| `src/agent/pkg/conntrack/` | Conntrack 同步模块 |
| `src/agent/pkg/protocol/` | 协议识别管理 |
| `src/agent/pkg/ratelimit/` | 限流配置管理 |

### 修改现有文件

| 文件 | 修改内容 |
|------|----------|
| `src/bpf/tc_microsegment.bpf.c` | 集成所有新功能 |
| `src/bpf/headers/common_types.h` | 扩展数据结构 |
| `src/agent/pkg/dataplane/dataplane.go` | 新增配置 API |

---

## 📊 性能目标

| 指标 | 基线（当前）| 目标（P0 后）| 目标（P1 后）|
|------|-------------|--------------|--------------|
| 数据包延迟 | < 10μs | < 15μs | < 20μs |
| NAT 查询 | N/A | < 5μs | < 5μs |
| 协议识别 | N/A | N/A | < 10μs |
| 并发会话 | 100K+ | 100K+ | 100K+ |
| CPU 开销 | < 5% | < 8% | < 12% |

---

## ✅ 每日验收检查

### 开发阶段
- [ ] eBPF 程序编译成功（make bpf）
- [ ] 单元测试通过
- [ ] 性能测试未退化
- [ ] 代码符合规范

### 测试阶段
- [ ] 功能测试通过
- [ ] 兼容性测试通过（多内核版本）
- [ ] 压力测试通过
- [ ] 性能指标达标

### 发布前
- [ ] 完整 E2E 测试
- [ ] 文档更新
- [ ] 性能报告
- [ ] 风险评估

---

## 📞 快速命令参考

```bash
# 编译 eBPF 程序
make bpf

# 运行单元测试
make test-bpf

# 运行 E2E 测试
make test-e2e

# 性能基准测试
make benchmark

# 启动 Agent
sudo ./bin/microsegment-agent --interface eth0 --default-policy deny

# 配置 NAT 模式
curl -X POST http://localhost:8080/api/v1/config/nat \
  -d '{"match_mode": "original"}'

# 启用默认拒绝
curl -X PUT http://localhost:8080/api/v1/config/default-policy \
  -d '{"action": "deny"}'
```

---

## 🔗 相关文档

- 📘 **[完整开发路线图](./EBPF_MICROSEGMENTATION_ROADMAP.md)** - 详细任务分解
- 📗 **[MVP 实施计划](./microsegmentation-mvp-implementation-plan.md)** - 项目整体规划
- 📙 **[项目 README](../README.md)** - 快速开始
- 📕 **[架构设计](../PROJECT_STRUCTURE.md)** - 项目结构

---

**创建日期**: 2025-11-18
**维护者**: 开发团队
**状态**: ✅ 规划完成，可立即开始实施
