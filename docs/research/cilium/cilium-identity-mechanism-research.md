# Cilium Identity 机制调研报告

> **调研目的**: 为 ebpf-based-microsegment 项目实现类似 Cilium 的基于身份（Identity）的访问控制和微隔离功能
> **调研日期**: 2025-12-18
> **参考源码**: `/home/work/ebpf-based-microsegment/source-references/cilium`

---

## 目录

1. [概述](#1-概述)
2. [Identity 核心概念](#2-identity-核心概念)
3. [Identity 数据结构](#3-identity-数据结构)
4. [Identity 分配机制](#4-identity-分配机制)
5. [IP-Identity 映射 (IPCache)](#5-ip-identity-映射-ipcache)
6. [eBPF 数据平面中的身份处理](#6-ebpf-数据平面中的身份处理)
7. [策略与身份的关联](#7-策略与身份的关联)
8. [与现有项目的对比分析](#8-与现有项目的对比分析)
9. [实现建议](#9-实现建议)
10. [关键源文件索引](#10-关键源文件索引)

---

## 1. 概述

### 1.1 什么是基于身份的访问控制

Cilium 的核心创新之一是 **基于身份（Identity-based）的网络策略**，而非传统的基于 IP 地址的策略。

**传统方式（基于 IP）**:
```
Rule: Allow 10.0.1.5 → 10.0.2.10:80
```
- 缺点：IP 地址不稳定（Pod 重启、扩缩容）
- 缺点：无法表达业务语义（"前端可以访问后端"）
- 缺点：规则数量爆炸式增长

**Cilium 方式（基于 Identity）**:
```
Rule: Allow Identity(app=frontend) → Identity(app=backend):80
```
- 优点：标签稳定，与 IP 解耦
- 优点：语义清晰，可读性强
- 优点：规则数量与服务数量线性相关

### 1.2 Identity 的核心作用

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Labels    │ ──▶ │  Identity   │ ──▶ │   Policy    │
│ (标签集合)   │     │ (数值 ID)   │     │  (策略匹配)  │
└─────────────┘     └─────────────┘     └─────────────┘
     │                    │                    │
     │                    │                    │
     ▼                    ▼                    ▼
  app=web             ID=12345           Allow/Deny
  env=prod            (32-bit)           决策
```

**Identity 是标签集合的数值化表示**，将复杂的标签匹配转换为高效的整数比较。

---

## 2. Identity 核心概念

### 2.1 Identity 的三个维度

| 维度 | 描述 | 示例 |
|------|------|------|
| **数值 ID** | 32-bit 无符号整数，用于快速匹配 | `12345`, `0x01000005` |
| **标签集合** | K8s/自定义标签的集合 | `{app:frontend, env:prod}` |
| **范围 (Scope)** | 身份的作用域 | Global, Local, RemoteNode |

### 2.2 Identity 范围 (Scope)

Cilium 使用 32-bit 身份标识符的**高 8 位**作为范围标记：

```
31        24 23                              0
┌──────────┬──────────────────────────────────┐
│  Scope   │         Identity Value           │
│ (8 bits) │           (24 bits)              │
└──────────┴──────────────────────────────────┘
```

| Scope 值 | 名称 | 范围描述 | 用途 |
|---------|------|---------|------|
| `0x00` | Global | 集群全局 | 工作负载身份（通过 KV-Store 分配） |
| `0x01` | Local | 节点本地 | CIDR、FQDN 身份 |
| `0x02` | RemoteNode | 远程节点 | 其他集群节点的身份 |

### 2.3 预留身份 (Reserved Identities)

Cilium 预留了 0-255 范围的身份用于特殊用途：

| ID | 名称 | 用途 |
|----|------|------|
| 0 | Unknown | 未知身份 |
| 1 | Host | 本地主机 |
| 2 | World | 集群外部流量（通用） |
| 3 | Unmanaged | 未托管的端点 |
| 4 | Health | cilium-health 端点 |
| 5 | Init | 初始化状态（未分配标签） |
| 6 | RemoteNode | 远程集群节点 |
| 7 | KubeAPIServer | Kubernetes API Server |
| 8 | Ingress | Ingress 代理源 IP |
| 9 | WorldIPv4 | 外部 IPv4 流量 |
| 10 | WorldIPv6 | 外部 IPv6 流量 |
| 102 | KubeDNS | kube-dns 服务 |
| 104 | CoreDNS | CoreDNS 服务 |
| 106 | CiliumOperator | Cilium Operator |

---

## 3. Identity 数据结构

### 3.1 Go 侧核心结构 (pkg/identity/identity.go)

```go
// Identity 是安全上下文的表示
type Identity struct {
    // 数值型身份标识 (32-bit)
    ID NumericIdentity `json:"id"`

    // 属于此身份的标签集合
    Labels labels.Labels `json:"labels"`

    // 标签的数组形式（优化查询性能）
    LabelArray labels.LabelArray `json:"-"`

    // 引用计数（用于垃圾回收）
    ReferenceCount int `json:"-"`
}

// NumericIdentity 是安全身份的数值表示
type NumericIdentity uint32

// 范围掩码定义
const (
    IdentityScopeMask       = NumericIdentity(0xFF_00_00_00)  // 高 8 位
    IdentityScopeGlobal     = NumericIdentity(0)              // 全局范围
    IdentityScopeLocal      = NumericIdentity(1 << 24)        // 本地范围
    IdentityScopeRemoteNode = NumericIdentity(2 << 24)        // 远程节点
)
```

### 3.2 IP-Identity 映射对

```go
// IP 地址与身份的映射关系
type IPIdentityPair struct {
    IP                net.IP          `json:"IP"`
    Mask              net.IPMask      `json:"Mask"`
    HostIP            net.IP          `json:"HostIP"`
    ID                NumericIdentity `json:"ID"`
    Key               uint8           `json:"Key"`           // 加密密钥索引
    Metadata          string          `json:"Metadata"`
    K8sNamespace      string          `json:"K8sNamespace"`
    K8sPodName        string          `json:"K8sPodName"`
    K8sServiceAccount string          `json:"K8sServiceAccount"`
    NamedPorts        []NamedPort     `json:"NamedPorts"`
}
```

### 3.3 eBPF 侧数据结构 (bpf/lib/eps.h)

```c
// 端点信息（本地端点）
struct endpoint_info {
    __u32 ifindex;        // 网卡索引
    __u16 unused;         // 已废弃
    __u16 lxc_id;         // 端点 ID
    __u32 flags;          // 端点标志
    mac_t mac;            // MAC 地址
    mac_t node_mac;       // 节点 MAC
    __u32 sec_id;         // 安全身份 ID ★
    __u32 parent_ifindex; // 父网卡索引
};

// 远程端点信息（通过 IPCache 查询）
struct remote_endpoint_info {
    __u32 sec_identity;   // 安全身份 ID ★
    union {
        __u32 ip4;
        union v6addr ip6;
    } tunnel_endpoint;    // 隧道端点
    __u8 flag_skip_tunnel:1;
    __u8 flag_has_tunnel_ep:1;
    __u8 flag_ipv6_tunnel_ep:1;
    __u8 flag_remote_cluster:1;
};

// IPCache 键结构（LPM Trie）
struct ipcache_key {
    struct bpf_lpm_trie_key lpm_key;  // LPM 前缀
    __u16 cluster_id;                  // 集群 ID
    __u8  family;                      // IPv4/IPv6
    union {
        __u32 ip4;
        union v6addr ip6;
    };
};
```

---

## 4. Identity 分配机制

### 4.1 三层身份体系

```
┌─────────────────────────────────────────────────┐
│         预留身份 (Reserved Identity)             │
│  • 固定分配：0-255                               │
│  • 硬编码标签集                                  │
│  • 无需 KV-Store                                │
│  • 例：Host(1), World(2), Health(4)             │
└─────────────────────────────────────────────────┘
                       ▲
                       │
┌─────────────────────────────────────────────────┐
│         本地身份 (Local Identity)                │
│  • 范围：0x01_00_00_01 ~ 0x01_FF_FF_FF          │
│  • 节点内分配，无需同步                          │
│  • 用途：CIDR 策略、FQDN 策略                   │
│  • 存储：localIdentityCache                     │
└─────────────────────────────────────────────────┘
                       ▲
                       │
┌─────────────────────────────────────────────────┐
│         全局身份 (Global Identity)               │
│  • 范围：256 ~ 0x00_FF_FF_FF                    │
│  • 通过 KV-Store (etcd) 分配                    │
│  • 集群级唯一，支持多集群同步                    │
│  • 存储：CachingIdentityAllocator               │
└─────────────────────────────────────────────────┘
```

### 4.2 身份分配流程

```
                    AllocateIdentity(labels)
                            │
                            ▼
            ┌───────────────────────────────┐
            │ 1. 检查预留身份                │
            │    LookupReservedIdentityByLabels()
            │    • 匹配 → 返回预留身份        │
            │    • 不匹配 → 继续              │
            └───────────────────────────────┘
                            │
                            ▼
            ┌───────────────────────────────┐
            │ 2. 判断身份范围                │
            │    ScopeForLabels(labels)     │
            │    • CIDR/FQDN → Local        │
            │    • RemoteNode → RemoteNode  │
            │    • 其他 → Global            │
            └───────────────────────────────┘
                            │
              ┌─────────────┼─────────────┐
              │             │             │
              ▼             ▼             ▼
         LOCAL        REMOTE_NODE     GLOBAL
         (本地)        (远程节点)      (全局)
              │             │             │
              ▼             ▼             ▼
    localIdentityCache  localNodeIdentities  KV-Store
              │             │             │
              └─────────────┴─────────────┘
                            │
                            ▼
                    返回 Identity
                    (ID + Labels)
```

### 4.3 标签来源与范围映射

```go
func ScopeForLabels(lbls labels.Labels) NumericIdentity {
    scope := IdentityScopeGlobal

    // 远程节点优先
    if lbls.HasRemoteNodeLabel() {
        return IdentityScopeRemoteNode
    }

    // 遍历所有标签
    for _, label := range lbls {
        switch label.Source {
        case labels.LabelSourceCIDR,       // CIDR 标签
             labels.LabelSourceFQDN,       // FQDN 标签
             labels.LabelSourceReserved,   // 预留标签
             labels.LabelSourceCIDRGroup:  // CIDR 组
            scope = IdentityScopeLocal
        default:
            // 包含其他来源标签 → 全局范围
            return IdentityScopeGlobal
        }
    }
    return scope
}
```

| 标签来源 | 身份范围 | 分配器 |
|---------|---------|--------|
| `LabelSourceK8s` | Global | KV-Store |
| `LabelSourceCIDR` | Local | localIdentityCache |
| `LabelSourceFQDN` | Local | localIdentityCache |
| `LabelSourceReserved` | Local/Reserved | 预留缓存 |

---

## 5. IP-Identity 映射 (IPCache)

### 5.1 IPCache 的作用

IPCache 是 IP 地址到身份的映射表，用于：
1. **出站流量**: 根据目的 IP 查询目的身份
2. **入站流量**: 根据源 IP 查询源身份
3. **策略执行**: 提供身份信息给策略匹配引擎

### 5.2 eBPF Map 定义

```c
// cilium_ipcache_v2: LPM Trie 类型的 BPF Map
struct {
    __uint(type, BPF_MAP_TYPE_LPM_TRIE);      // 最长前缀匹配
    __type(key, struct ipcache_key);
    __type(value, struct remote_endpoint_info);
    __uint(max_entries, IPCACHE_MAP_SIZE);    // 默认 512K
} cilium_ipcache_v2;
```

**为什么使用 LPM Trie？**
- 支持 CIDR 前缀匹配（如 `/24`, `/16`）
- 查询复杂度 O(前缀长度)
- 自动选择最精确的匹配项

### 5.3 查询函数

```c
// IPv4 查询
static __always_inline const struct remote_endpoint_info *
lookup_ip4_remote_endpoint(__be32 addr, __u32 cluster_id) {
    struct ipcache_key key = {
        .lpm_key = { IPCACHE_PREFIX_LEN(32), {} },
        .family = ENDPOINT_KEY_IPV4,
        .ip4 = addr,
        .cluster_id = cluster_id,
    };
    return map_lookup_elem(&cilium_ipcache_v2, &key);
}
```

### 5.4 IPCache 更新流程

```
       K8s Pod 创建/更新
              │
              ▼
    ┌─────────────────────┐
    │  Cilium Agent       │
    │  (Endpoint Manager) │
    └─────────────────────┘
              │
              ├─── 分配/查找 Identity
              │
              ▼
    ┌─────────────────────┐
    │  IPCache Manager    │
    │  (pkg/ipcache)      │
    └─────────────────────┘
              │
              ├─── 更新 IP → Identity 映射
              │
              ▼
    ┌─────────────────────┐
    │  BPF Map 更新        │
    │  (cilium_ipcache_v2)│
    └─────────────────────┘
```

---

## 6. eBPF 数据平面中的身份处理

### 6.1 身份常量定义 (bpf/lib/identity.h)

```c
#define HOST_ID          1    // 本地主机
#define WORLD_ID         2    // 外部流量
#define UNMANAGED_ID     3    // 未托管端点
#define HEALTH_ID        4    // 健康检查
#define REMOTE_NODE_ID   6    // 远程节点
#define WORLD_IPV4_ID    9    // IPv4 外部流量
#define WORLD_IPV6_ID    10   // IPv6 外部流量

#define IDENTITY_LOCAL_SCOPE_MASK        0xFF000000
#define IDENTITY_LOCAL_SCOPE_REMOTE_NODE 0x02000000

#define CIDR_IDENTITY_RANGE_START 0x010001
#define CIDR_IDENTITY_RANGE_END   0x01FFFF
```

### 6.2 身份分类函数

```c
// 检查是否为主机身份
static __always_inline bool identity_is_host(__u32 identity) {
    return identity == HOST_ID;
}

// 检查是否为远程节点
static __always_inline bool identity_is_remote_node(__u32 identity) {
    return identity == REMOTE_NODE_ID ||
           identity == KUBE_APISERVER_NODE_ID ||
           (identity & IDENTITY_LOCAL_SCOPE_MASK) == IDENTITY_LOCAL_SCOPE_REMOTE_NODE;
}

// 检查是否为预留身份
static __always_inline bool identity_is_reserved(__u32 identity) {
    return identity < UNMANAGED_ID ||
           identity_is_remote_node(identity) ||
           identity == WORLD_IPV4_ID ||
           identity == WORLD_IPV6_ID;
}

// 检查是否为 CIDR 范围身份
static __always_inline bool identity_is_cidr_range(__u32 identity) {
    return identity_in_range(identity,
                            CIDR_IDENTITY_RANGE_START,
                            CIDR_IDENTITY_RANGE_END);
}

// 检查是否为集群内部身份
static __always_inline bool identity_is_cluster(__u32 identity) {
    if (identity == WORLD_ID || identity == WORLD_IPV4_ID || identity == WORLD_IPV6_ID)
        return false;
    if (identity_is_cidr_range(identity))
        return false;
    return true;
}
```

### 6.3 数据包处理流程

#### 出站流程 (bpf_lxc.c)

```
    Pod 发出数据包
          │
          ▼
┌─────────────────────────┐
│ TC Egress Hook          │
│ handle_ipv4_from_lxc()  │
└─────────────────────────┘
          │
          ├─ 1. 从 endpoint_info 获取源身份
          │      src_identity = ep->sec_id
          │
          ├─ 2. 查询目的身份
          │      info = lookup_ip4_remote_endpoint(daddr, cluster_id)
          │      dst_identity = info ? info->sec_identity : WORLD_ID
          │
          ├─ 3. 策略检查
          │      policy_can_egress4(src_identity, dst_identity, proto, dport)
          │
          ▼
     ALLOW / DROP
```

#### 入站流程 (bpf_host.c)

```
    外部数据包到达
          │
          ▼
┌─────────────────────────┐
│ TC Ingress Hook         │
│ resolve_srcid_ipv4()    │
└─────────────────────────┘
          │
          ├─ 1. 检查 Mark 字段是否已有身份
          │      if (mark has identity) → use it
          │
          ├─ 2. 从 IPCache 查询源身份
          │      info = lookup_ip4_remote_endpoint(saddr, 0)
          │      src_identity = info ? info->sec_identity : WORLD_ID
          │
          ├─ 3. 策略检查
          │      policy_can_access(src_identity, dst_identity, ...)
          │
          ▼
     ALLOW / DROP
```

### 6.4 Mark 字段中的身份编码

为了在数据包处理链中传递身份信息，Cilium 使用 `skb->mark` 字段：

```c
/*
 * Mark 字段布局:
 * 31       24 23       16 15        8 7         0
 * ┌─────────┬───────────┬───────────┬───────────┐
 * │Identity │ Identity  │  Magic    │ ClusterID │
 * │ (upper) │ (lower)   │  Flags    │ (lower)   │
 * └─────────┴───────────┴───────────┴───────────┘
 */

// 设置身份到 mark 字段
static __always_inline void
set_identity_mark(struct __ctx_buff *ctx, __u32 identity, __u32 magic) {
    __u32 cluster_id = (identity >> IDENTITY_LEN) & get_cluster_id_max();
    __u32 cluster_id_lower = cluster_id & 0xFF;

    ctx->mark = magic & MARK_MAGIC_KEY_MASK;
    ctx->mark |= (identity & IDENTITY_MAX) << 16 | cluster_id_lower;
}

// 从 mark 字段获取身份
static __always_inline int get_identity(const struct __ctx_buff *ctx) {
    __u32 cluster_id_lower = ctx->mark & CLUSTER_ID_LOWER_MASK;
    __u32 identity = (ctx->mark >> 16) & IDENTITY_MAX;
    return cluster_id_lower << IDENTITY_LEN | identity;
}
```

---

## 7. 策略与身份的关联

### 7.1 SelectorCache 架构

```go
// pkg/policy/selectorcache.go

// 身份缓存：NumericIdentity → 标签
type scIdentityCache struct {
    ids         map[identity.NumericIdentity]*scIdentity
    byNamespace map[string]map[*scIdentity]struct{}
}

// 选择器缓存：选择器 → 匹配的身份集合
type selectorMap struct {
    selectors            map[string]*identitySelector
    selectorsByNamespace map[string]map[*identitySelector]struct{}
}
```

### 7.2 策略匹配流程

```
┌─────────────────────────────────────────┐
│ CiliumNetworkPolicy 定义                 │
│ spec:                                   │
│   endpointSelector:                     │
│     matchLabels:                        │
│       app: backend                      │
│   ingress:                              │
│   - fromEndpoints:                      │
│     - matchLabels:                      │
│         app: frontend                   │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│ SelectorCache.GetSelections()           │
│ • 将 matchLabels 转换为选择器           │
│ • 匹配所有身份，返回匹配的身份集合       │
│ • 例：{app:frontend} → [ID:1234, ID:5678]│
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│ PolicyRepository.resolvePolicyLocked()  │
│ • 为每个 Endpoint 计算允许的身份集合     │
│ • 生成 MapState 结构                    │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│ BPF Policy Map 更新                      │
│ Key: {src_identity, dst_identity,       │
│       protocol, port}                   │
│ Value: {action, proxy_port, ...}        │
└─────────────────────────────────────────┘
```

### 7.3 BPF Policy Map 结构

```c
// 策略键
struct policy_key {
    struct bpf_lpm_trie_key lpm_key;
    __u32 sec_label;     // 远程身份 ID
    __u8  egress:1;      // 出站标志
    __u8  protocol;      // 协议 (TCP/UDP)
    __be16 dport;        // 目的端口
};

// 策略值
struct policy_entry {
    __be16 proxy_port;   // 代理端口（L7 重定向）
    __u8   deny:1;       // 拒绝标志
    __u8   auth_type:7;  // 认证类型
    __u32  precedence;   // 优先级
};
```

### 7.4 策略查询算法（两阶段查询）

```c
// bpf/lib/policy.h

// 策略匹配采用两次查询：
// 1. L3 查询：精确身份 + L4 匹配
// 2. L4-only 查询：通配符身份 + L4 匹配

static __always_inline int
policy_can_access(struct __ctx_buff *ctx, __u32 local_id, __u32 remote_id,
                  __u16 dport, __u8 proto) {
    struct policy_key key = {
        .sec_label = remote_id,  // 精确身份
        .protocol = proto,
        .dport = dport,
    };

    // 第一次查询：精确身份
    struct policy_entry *policy = map_lookup_elem(&cilium_policy, &key);

    // 第二次查询：通配符身份 (sec_label = 0)
    key.sec_label = 0;
    struct policy_entry *l4policy = map_lookup_elem(&cilium_policy, &key);

    // 优先级选择
    // 1. precedence 值最高的优先
    // 2. LPM 前缀长度最长的优先
    // 3. 精确身份优于通配符身份

    if (l4policy && (!policy ||
        l4policy->precedence > policy->precedence)) {
        policy = l4policy;
    }

    if (!policy)
        return DROP_POLICY;

    return policy->deny ? DROP_POLICY : TC_ACT_OK;
}
```

---

## 8. 与现有项目的对比分析

### 8.1 当前 ebpf-based-microsegment 的策略模型

```go
// 当前：基于 5-tuple 的策略
type PolicyKey struct {
    SrcIP    uint32
    DstIP    uint32
    SrcPort  uint16
    DstPort  uint16
    Protocol uint8
}
```

**限制**:
- IP 地址硬编码，Pod 重启后失效
- 无法表达标签语义
- 规则数量与 Pod 数量平方相关

### 8.2 目标：基于身份的策略模型

```go
// 目标：基于身份的策略
type IdentityPolicyKey struct {
    SrcIdentity uint32    // 源身份 ID
    DstIdentity uint32    // 目的身份 ID
    DstPort     uint16    // 目的端口
    Protocol    uint8     // 协议
}
```

**优势**:
- 与 IP 解耦，Pod 重启不影响
- 标签语义清晰
- 规则数量与服务数量线性相关

### 8.3 实现差距分析

| 功能 | Cilium | ebpf-microsegment | 差距 |
|------|--------|-------------------|------|
| 身份数据结构 | 完整 | 无 | 需新建 |
| 身份分配器 | KV-Store + Local | 无 | 需新建 |
| IP-Identity 映射 | LPM Trie | 无 | 需新建 |
| 策略与身份关联 | SelectorCache | 无 | 需新建 |
| BPF 身份查询 | ipcache_lookup | 无 | 需新建 |
| 标签管理 | 完整 | 部分 | 需扩展 |

---

## 9. 实现建议

### 9.1 分阶段实现路线

#### Phase 1: 基础身份系统

1. **定义 Identity 数据结构**
   - NumericIdentity (uint32)
   - Identity (ID + Labels)
   - 身份范围常量

2. **实现身份分配器**
   - 本地身份缓存 (localIdentityCache)
   - 预留身份初始化
   - 引用计数管理

3. **创建 IPCache**
   - BPF LPM Trie Map
   - IP → Identity 映射
   - 更新接口

#### Phase 2: 策略集成

1. **扩展策略模型**
   - 支持身份选择器
   - 策略编译为身份对

2. **BPF 策略匹配**
   - 身份查询函数
   - 策略 Map 更新

3. **Endpoint 身份关联**
   - 工作负载 → 身份映射
   - 动态更新机制

#### Phase 3: 高级功能

1. **标签变更处理**
   - 身份重新分配
   - 策略增量更新

2. **性能优化**
   - 缓存策略
   - 批量更新

### 9.2 核心数据结构建议

```go
// pkg/identity/identity.go

package identity

type NumericIdentity uint32

const (
    IdentityScopeMask       = NumericIdentity(0xFF000000)
    IdentityScopeGlobal     = NumericIdentity(0)
    IdentityScopeLocal      = NumericIdentity(1 << 24)

    // 预留身份
    IdentityUnknown         = NumericIdentity(0)
    ReservedIdentityHost    = NumericIdentity(1)
    ReservedIdentityWorld   = NumericIdentity(2)
    ReservedIdentityHealth  = NumericIdentity(4)

    // 身份范围
    MinimalNumericIdentity  = NumericIdentity(256)
    MaxLocalIdentity        = NumericIdentity(0x01FFFFFF)
)

type Identity struct {
    ID             NumericIdentity
    Labels         map[string]string
    ReferenceCount int
}

func (id NumericIdentity) Scope() NumericIdentity {
    return id & IdentityScopeMask
}

func (id NumericIdentity) IsLocal() bool {
    return id.Scope() == IdentityScopeLocal
}
```

### 9.3 BPF Map 建议

```c
// src/bpf/headers/identity.h

#define HOST_ID     1
#define WORLD_ID    2
#define HEALTH_ID   4

struct identity_info {
    __u32 sec_identity;
    __u32 flags;
};

struct ipcache_key {
    struct bpf_lpm_trie_key lpm;
    __u32 ip4;
};

// IP → Identity 映射
struct {
    __uint(type, BPF_MAP_TYPE_LPM_TRIE);
    __type(key, struct ipcache_key);
    __type(value, struct identity_info);
    __uint(max_entries, 65536);
} ipcache SEC(".maps");

// 身份策略映射
struct policy_key {
    __u32 src_identity;
    __u32 dst_identity;
    __u16 dst_port;
    __u8  protocol;
};

struct {
    __uint(type, BPF_MAP_TYPE_HASH);
    __type(key, struct policy_key);
    __type(value, __u8);  // 0=deny, 1=allow
    __uint(max_entries, 100000);
} identity_policy SEC(".maps");
```

---

## 10. 关键源文件索引

### 10.1 Go 源文件

| 文件路径 | 核心职责 |
|---------|---------|
| `pkg/identity/identity.go` | Identity 数据结构定义 |
| `pkg/identity/numericidentity.go` | NumericIdentity 类型和预留身份 |
| `pkg/identity/reserved.go` | 预留身份缓存 |
| `pkg/identity/cache/allocator.go` | 全局身份分配器 |
| `pkg/identity/cache/local.go` | 本地身份缓存 |
| `pkg/ipcache/ipcache.go` | IPCache 管理器 |
| `pkg/policy/selectorcache.go` | 选择器与身份缓存 |
| `pkg/policy/repository.go` | 策略仓库 |
| `pkg/policy/mapstate.go` | BPF Map 状态管理 |
| `pkg/endpoint/policy.go` | Endpoint 策略计算 |

### 10.2 BPF 源文件

| 文件路径 | 核心职责 |
|---------|---------|
| `bpf/lib/identity.h` | 身份操作宏和工具函数 |
| `bpf/lib/eps.h` | IPCache Map 定义和查询函数 |
| `bpf/lib/policy.h` | 策略查询算法 |
| `bpf/bpf_lxc.c` | 容器出站处理 |
| `bpf/bpf_host.c` | 主机入站处理 |
| `bpf/lib/conntrack.h` | 连接跟踪（含身份存储） |

---

## 总结

Cilium 的基于身份的访问控制是一个精心设计的系统，核心思想是：

1. **将标签集合映射为数值身份**，实现高效的内核态匹配
2. **三层身份体系**（预留/本地/全局）平衡性能和一致性
3. **IPCache 作为桥梁**，将运行时 IP 地址映射到身份
4. **两阶段策略查询**，支持精确匹配和通配符

对于 ebpf-based-microsegment 项目，建议：
- 先实现本地身份系统（无需 KV-Store）
- 复用现有的标签管理基础设施
- 逐步扩展策略模型支持身份选择器
- 保持 eBPF 数据平面的高性能特性

---

*调研完成，下一步：进入 Phase 2 分析现有项目架构，评估集成方案*
