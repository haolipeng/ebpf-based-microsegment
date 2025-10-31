# NeuVector FQDN 匹配功能详解

## 目录

- [一、FQDN 功能概述](#一fqdn-功能概述)
- [二、核心数据结构](#二核心数据结构)
- [三、工作流程](#三工作流程)
- [四、DNS 解析拦截](#四dns-解析拦截)
- [五、双向映射机制](#五双向映射机制)
- [六、通配符 FQDN 支持](#六通配符-fqdn-支持)
- [七、完整实战示例](#七完整实战示例)
- [八、性能优化](#八性能优化)

---

## 一、FQDN 功能概述

### 1.1 什么是 FQDN 匹配？

**FQDN (Fully Qualified Domain Name)** 匹配允许用户使用域名而非 IP 地址来定义安全策略。

**为什么需要 FQDN 匹配？**

1. **动态 IP 地址**
   - 云服务 IP 地址频繁变化（如 AWS S3、Google APIs）
   - CDN 服务器地址不固定
   - 容器/微服务动态扩缩容

2. **易用性**
   - 用户记忆域名比 IP 地址容易
   - 策略定义更直观（如 `api.github.com` vs `140.82.113.5`）

3. **可维护性**
   - IP 变化时无需修改策略
   - 一次配置，自动适应 IP 变化

### 1.2 NeuVector FQDN 功能特性

✅ **已实现功能**：

| 功能 | 状态 | 说明 |
|------|------|------|
| **基本 FQDN 匹配** | ✅ 支持 | 策略中使用域名 |
| **双向映射** | ✅ 支持 | FQDN → IP 和 IP → FQDN |
| **通配符 FQDN** | ✅ 支持 | `*.github.com` 匹配所有子域名 |
| **DNS 拦截** | ✅ 支持 | 自动学习 DNS 响应中的 IP |
| **动态更新** | ✅ 支持 | IP 变化时自动更新映射 |
| **虚拟主机模式** | ✅ 支持 | 同一 IP 托管多个域名 |
| **IPv4 支持** | ✅ 支持 | 完整的 IPv4 FQDN 映射 |
| **IPv6 支持** | ⚠️ 部分 | 代码中有框架，未完全实现 |

### 1.3 架构图

```
┌─────────────────────────────────────────────────────────┐
│                     FQDN 匹配流程                        │
└─────────────────────────────────────────────────────────┘

Step 1: 用户配置策略
    ↓
┌─────────────────────────────────────────────────────────┐
│  Controller/Agent                                       │
│  策略规则: Allow traffic to "api.github.com"            │
└─────────────────┬───────────────────────────────────────┘
                  │ 下发策略
                  ▼
┌─────────────────────────────────────────────────────────┐
│  dp 接收策略                                             │
│  config_fqdn_ipv4_mapping("api.github.com", ...)        │
│    ├─> 创建 FQDN 记录                                   │
│    ├─> 分配唯一 Code (ID)                               │
│    └─> 建立映射表                                       │
└─────────────────┬───────────────────────────────────────┘
                  │
                  ▼
┌─────────────────────────────────────────────────────────┐
│  双向映射表                                              │
│                                                         │
│  ┌─────────────────────────────────────────────────┐  │
│  │  fqdn_name_map (FQDN → Code)                    │  │
│  │  ┌──────────────────┬────┬──────────────┐       │  │
│  │  │ api.github.com   │ 10 │ [IP list]    │       │  │
│  │  │ *.example.com    │ 11 │ [IP list]    │       │  │
│  │  └──────────────────┴────┴──────────────┘       │  │
│  └─────────────────────────────────────────────────┘  │
│                                                         │
│  ┌─────────────────────────────────────────────────┐  │
│  │  fqdn_ipv4_map (IP → FQDN list)                 │  │
│  │  ┌──────────────────┬─────────────────────────┐ │  │
│  │  │ 140.82.113.5     │ [api.github.com]        │ │  │
│  │  │ 140.82.114.5     │ [api.github.com]        │ │  │
│  │  │ 192.168.1.100    │ [www.example.com,       │ │  │
│  │  │                  │  mail.example.com]       │ │  │
│  │  └──────────────────┴─────────────────────────┘ │  │
│  └─────────────────────────────────────────────────┘  │
└─────────────────┬───────────────────────────────────────┘
                  │
                  ▼
Step 2: 容器发起 DNS 查询
    ↓
┌─────────────────────────────────────────────────────────┐
│  DNS Query: api.github.com                              │
│    ↓                                                    │
│  dp 拦截 DNS 响应 (dpi_dns.c)                            │
│    ├─> 解析 DNS Answer (A 记录)                         │
│    ├─> 提取 IP: 140.82.113.5, 140.82.114.5             │
│    └─> 更新 FQDN 映射表                                 │
│         ├─ api.github.com → [140.82.113.5, ...]        │
│         └─ 140.82.113.5 → [api.github.com]             │
└─────────────────┬───────────────────────────────────────┘
                  │
                  ▼
Step 3: 数据包匹配
    ↓
┌─────────────────────────────────────────────────────────┐
│  数据包: 192.168.1.10 → 140.82.113.5:443                │
│    ↓                                                    │
│  策略匹配 (dpi_policy.c)                                 │
│    ├─> 查找 IP 对应的 FQDN                              │
│    │   └─ 140.82.113.5 → api.github.com                │
│    │                                                    │
│    ├─> 匹配策略规则                                     │
│    │   └─ 规则: Allow api.github.com ✅                │
│    │                                                    │
│    └─> 返回动作: ALLOW                                  │
└─────────────────────────────────────────────────────────┘
```

---

## 二、核心数据结构

### 2.1 FQDN 记录 (fqdn_record_t)

**文件位置**：apis.h:291-302

```c
typedef struct fqdn_record_ {
    char name[MAX_FQDN_LEN];        // 域名（最大 256 字符）
    uint32_t code;                  // 唯一标识符（用于策略匹配）
    uint32_t flag;                  // 标志位
    #define FQDN_RECORD_TO_DELETE      0x00000001  // 标记删除
    #define FQDN_RECORD_DELETED        0x00000002  // 已删除
    #define FQDN_RECORD_WILDCARD       0x00000004  // 通配符 FQDN
    uint32_t ip_cnt;                // 关联的 IP 数量
    uint32_t record_updated;        // 更新时间戳（用于通配符）
    struct cds_list_head iplist;    // IP 列表（链表）
    bool vh;                        // 虚拟主机模式
} fqdn_record_t;
```

**字段说明**：
- `name`: 存储完整域名，如 `"api.github.com"` 或 `"*.example.com"`
- `code`: 内部唯一 ID，用于快速匹配（避免字符串比较）
- `flag`: 标志位
  - `FQDN_RECORD_WILDCARD`: 通配符 FQDN（如 `*.github.com`）
  - `TO_DELETE`/`DELETED`: 延迟删除机制（RCU 安全）
- `iplist`: 关联的 IP 地址列表（一对多映射）
- `vh`: 虚拟主机模式（同一 IP 托管多个域名时使用）

### 2.2 FQDN 名称条目 (fqdn_name_entry_t)

**文件位置**：apis.h:309-312

```c
typedef struct fqdn_name_entry_ {
    struct cds_lfht_node node;      // RCU 哈希表节点
    fqdn_record_t *r;               // 指向 FQDN 记录
} fqdn_name_entry_t;
```

**用途**：FQDN 名称到记录的映射（`fqdn_name_map`）

### 2.3 FQDN IPv4 条目 (fqdn_ipv4_entry_t)

**文件位置**：apis.h:314-318

```c
typedef struct fqdn_ipv4_entry_ {
    struct cds_lfht_node node;      // RCU 哈希表节点
    uint32_t ip;                    // IPv4 地址
    struct cds_list_head rlist;     // 关联的 FQDN 记录列表（反向映射）
} fqdn_ipv4_entry_t;
```

**用途**：IP 到 FQDN 列表的映射（`fqdn_ipv4_map`）

**为什么需要反向映射？**
- 数据包处理时，只有目标 IP（140.82.113.5）
- 需要查找该 IP 对应的域名（api.github.com）
- 然后匹配策略规则

### 2.4 FQDN 句柄 (dpi_fqdn_hdl_t)

**文件位置**：apis.h:327-337

```c
typedef struct dpi_fqdn_hdl_ {
    rcu_map_t fqdn_name_map;        // 域名 → 记录（哈希表）
    rcu_map_t fqdn_ipv4_map;        // IP → FQDN 列表（哈希表）
    bitmap *bm;                     // Code 分配位图
    int code_cnt;                   // 已分配的 Code 数量
    int del_name_cnt;               // 待删除的名称数量
    int del_ipv4_cnt;               // 待删除的 IP 数量
    fqdn_name_entry_t *del_name_list[DPI_FQDN_DELETE_QLEN];  // 删除队列
    fqdn_ipv4_entry_t *del_ipv4_list[DPI_FQDN_DELETE_QLEN];
    struct cds_list_head del_rlist; // 待删除记录列表
} dpi_fqdn_hdl_t;
```

**全局变量**：
```c
dpi_fqdn_hdl_t *g_fqdn_hdl = NULL;  // 全局 FQDN 句柄
```

**关键点**：
- **双向映射**：`fqdn_name_map` + `fqdn_ipv4_map`
- **RCU 保护**：无锁并发读取
- **延迟删除**：删除队列保证 RCU 安全

---

## 三、工作流程

### 3.1 FQDN 策略配置流程

#### Step 1: Agent 下发策略

```json
{
  "ctrl_cfg_policy": {
    "cmd": 1,
    "mac": ["aa:bb:cc:dd:ee:ff"],
    "rules": [
      {
        "id": 1001,
        "sip": "0.0.0.0",
        "dip": "0.0.0.0",
        "port": 443,
        "proto": 6,
        "action": 1,
        "ingress": false,
        "fqdn": "api.github.com",  // ← FQDN 字段
        "vhost": false
      }
    ]
  }
}
```

#### Step 2: dp 处理策略

**文件位置**：dpi/dpi_policy.c:1445-1473

```c
// dp_ctrl_cfg_policy() 解析 JSON 后调用 dpi_policy_cfg()
int dpi_policy_cfg(int cmd, dpi_policy_t *p, int flag) {
    // ...
    for (int i = 0; i < p->num_rules; i++) {
        // 检查规则是否有 FQDN
        if (p->rule_list[i].fqdn[0] != '\0') {
            uint32_t code;

            // 判断 FQDN 是源还是目标
            if (p->rule_list[i].ingress) {
                // 入站规则：FQDN 是源
                rcu_read_lock();
                code = config_fqdn_ipv4_mapping(g_fqdn_hdl,
                            p->rule_list[i].fqdn,
                            p->rule_list[i].sip,
                            p->rule_list[i].vh);
                rcu_read_unlock();

                if (code == -1) {
                    continue;  // 分配失败，跳过
                }

                // 用 code 替换 IP（优化匹配速度）
                p->rule_list[i].sip = code;
                p->rule_list[i].sip_r = code;
            } else {
                // 出站规则：FQDN 是目标
                rcu_read_lock();
                code = config_fqdn_ipv4_mapping(g_fqdn_hdl,
                            p->rule_list[i].fqdn,
                            p->rule_list[i].dip,
                            p->rule_list[i].vh);
                rcu_read_unlock();

                if (code == -1) {
                    continue;
                }

                p->rule_list[i].dip = code;
                p->rule_list[i].dip_r = code;
            }

            // 标记策略句柄包含 FQDN 规则
            hdl->flag |= POLICY_HDL_FLAG_FQDN;
        }

        // 继续添加规则到策略树...
    }
}
```

**关键点**：
- FQDN 被转换为唯一的 `code`（整数 ID）
- `code` 替换规则中的 IP 地址
- 策略匹配时使用 `code` 而非字符串比较（性能优化）

#### Step 3: 创建 FQDN 映射

**文件位置**：dpi/dpi_policy.c:1745-1845

```c
uint32_t config_fqdn_ipv4_mapping(dpi_fqdn_hdl_t *hdl, char *name,
                                   uint32_t ip, bool vh)
{
    fqdn_name_entry_t *name_entry;
    fqdn_ipv4_entry_t *ipv4_entry;
    fqdn_record_t *r = NULL;

    // 1. 查找 FQDN 是否已存在
    name_entry = rcu_map_lookup(&hdl->fqdn_name_map, name);

    if (!name_entry) {
        // 2. FQDN 不存在，创建新记录
        name_entry = (fqdn_name_entry_t *)calloc(1, sizeof(fqdn_name_entry_t));
        r = (fqdn_record_t *)calloc(1, sizeof(fqdn_record_t));

        strlcpy(r->name, name, MAX_FQDN_LEN);

        // 3. 分配唯一 Code
        r->code = alloc_fqdn_code(hdl);  // 从位图分配
        if (r->code == -1) {
            // Code 耗尽（最大 DPI_FQDN_MAX_ENTRIES）
            free(name_entry);
            free(r);
            return -1;
        }

        r->vh = vh;

        // 4. 检查是否为通配符 FQDN
        if (is_fqdn_name_wildcard(r->name)) {
            r->flag = FQDN_RECORD_WILDCARD;
        }

        CDS_INIT_LIST_HEAD(&r->iplist);  // 初始化 IP 列表

        // 5. 插入到 fqdn_name_map
        name_entry->r = r;
        rcu_map_add(&hdl->fqdn_name_map, name_entry, name);
    } else {
        r = name_entry->r;
    }

    // 6. 添加 IP 到 FQDN 的映射
    ipv4_entry = rcu_map_lookup(&hdl->fqdn_ipv4_map, &ip);

    if (!ipv4_entry) {
        // IP 不存在，创建新条目
        ipv4_entry = (fqdn_ipv4_entry_t *)calloc(1, sizeof(fqdn_ipv4_entry_t));
        ipv4_entry->ip = ip;
        CDS_INIT_LIST_HEAD(&ipv4_entry->rlist);

        // 插入到 fqdn_ipv4_map
        rcu_map_add(&hdl->fqdn_ipv4_map, ipv4_entry, &ip);
    }

    // 7. 建立双向关联
    // IP → FQDN
    fqdn_record_item_t *record_item = calloc(1, sizeof(fqdn_record_item_t));
    record_item->r = r;
    cds_list_add_tail(&record_item->node, &ipv4_entry->rlist);

    // FQDN → IP
    fqdn_ipv4_item_t *ip_item = calloc(1, sizeof(fqdn_ipv4_item_t));
    ip_item->ip = ip;
    cds_list_add_tail(&ip_item->node, &r->iplist);

    r->ip_cnt++;

    // 8. 返回 Code
    return r->code;
}
```

**数据结构示例**：

配置 `api.github.com` → `[140.82.113.5, 140.82.114.5]` 后：

```
fqdn_name_map:
  "api.github.com" → {
    code: 10,
    flag: 0,
    iplist: [140.82.113.5, 140.82.114.5],
    ip_cnt: 2
  }

fqdn_ipv4_map:
  140.82.113.5 → {
    rlist: [api.github.com]
  }
  140.82.114.5 → {
    rlist: [api.github.com]
  }
```

### 3.2 数据包匹配流程

#### 场景：容器访问 api.github.com

```
1. 容器发起 DNS 查询：api.github.com
   ↓
2. dp 拦截 DNS 响应，学习 IP: 140.82.113.5
   ↓
3. 容器建立连接：192.168.1.10 → 140.82.113.5:443
   ↓
4. dp 匹配策略：
   ├─ 提取目标 IP: 140.82.113.5
   ├─ 查找 fqdn_ipv4_map: 140.82.113.5 → api.github.com
   ├─ 获取 code: 10
   ├─ 匹配策略规则: dip == 10 (api.github.com)
   └─ 返回动作: ALLOW
```

**代码实现**：

**文件位置**：dpi/dpi_policy.c

```c
// 策略匹配时调用
static int policy_match_ipv4_fqdn_code(dpi_fqdn_hdl_t *fqdn_hdl,
                                       uint32_t ip,
                                       dpi_policy_hdl_t *hdl,
                                       dpi_rule_key_t *key,
                                       int is_ingress,
                                       dpi_policy_desc_t *desc2,
                                       dpi_packet_t *p)
{
    fqdn_ipv4_entry_t *ipv4_entry;

    // 1. 根据 IP 查找关联的 FQDN
    ipv4_entry = rcu_map_lookup(&fqdn_hdl->fqdn_ipv4_map, &ip);
    if (!ipv4_entry) {
        return -1;  // IP 没有关联的 FQDN
    }

    // 2. 遍历 IP 关联的所有 FQDN
    fqdn_record_item_t *record_item;
    cds_list_for_each_entry(record_item, &ipv4_entry->rlist, node) {
        fqdn_record_t *r = record_item->r;

        // 3. 用 FQDN 的 code 替换 key 中的 IP
        if (is_ingress) {
            key->sip = r->code;  // 入站：替换源 IP
        } else {
            key->dip = r->code;  // 出站：替换目标 IP
        }

        // 4. 用 code 进行策略匹配
        if (policy_lookup(hdl, key, desc2) == 0) {
            // 匹配成功
            return 0;
        }
    }

    return -1;  // 没有匹配的策略
}
```

---

## 四、DNS 解析拦截

### 4.1 为什么需要拦截 DNS？

当用户配置 FQDN 策略时，dp 需要知道域名对应的 IP 地址。有两种方式：

1. **Agent 主动查询 DNS**（已实现）
   - Agent 定期解析 FQDN
   - 将 IP 列表下发给 dp

2. **dp 拦截 DNS 响应**（已实现）✅
   - 容器发起 DNS 查询
   - dp 解析 DNS 响应数据包
   - 自动学习域名 → IP 映射

**优势**：
- 无需额外 DNS 查询（节省网络流量）
- IP 变化时自动更新（实时性好）
- 学习用户实际访问的 IP（准确性高）

### 4.2 DNS 解析器实现

**文件位置**：dpi/parsers/dpi_dns.c

#### DNS 数据包结构

```c
typedef struct dns_hdr_ {
    uint16_t id;
    uint8_t qr     :1,  // 查询(0) 或 响应(1)
            opcode :4,  // 操作码
            aa     :1,  // 权威应答
            tc     :1,  // 截断
            rd     :1;  // 期望递归
    uint8_t ra     :1,  // 可递归
            z      :1,  // 保留
            ad     :1,  // 已验证数据
            cd     :1,  // 禁用检查
            rcode  :4;  // 响应码
    uint16_t qd_count;  // 问题数量
    uint16_t an_count;  // 应答数量
    uint16_t ns_count;  // 授权记录数量
    uint16_t ar_count;  // 附加记录数量
} dns_hdr_t;
```

#### DNS 解析流程

```c
void dpi_dns_parser(dpi_packet_t *p) {
    dns_hdr_t *hdr = (dns_hdr_t *)p->ptr;

    // 1. 只处理 DNS 响应（qr == 1）
    if (hdr->qr == 0) {
        return;  // 查询包，忽略
    }

    // 2. 解析 Answer 记录
    int answer_count = ntohs(hdr->an_count);
    uint8_t *ptr = p->ptr + sizeof(dns_hdr_t);

    for (int i = 0; i < answer_count; i++) {
        // 3. 提取域名
        char domain[MAX_LABEL_LEN];
        ptr = get_dns_name(p, ptr, domain);

        // 4. 解析 RR (Resource Record)
        uint16_t type = ntohs(*(uint16_t *)ptr);
        ptr += 2;  // type
        ptr += 2;  // class
        ptr += 4;  // TTL
        uint16_t rdlength = ntohs(*(uint16_t *)ptr);
        ptr += 2;

        // 5. 只处理 A 记录（IPv4）
        if (type == DNS_TYPE_A && rdlength == 4) {
            uint32_t ip = *(uint32_t *)ptr;

            // 6. 上报 FQDN → IP 映射到 agent
            report_fqdn_ip_mapping(domain, ip);
        }

        ptr += rdlength;
    }
}
```

**上报流程**：

```
dp (DNS 解析器)
    ↓
dp_ctrl_notify_ctrl()
    ↓ Unix Socket (Binary)
agent listenDP()
    ↓
dpMsgFqdnIpUpdate()
    ↓
更新本地 FQDN 映射
    ↓
下发到 dp: DPCtrlSetFqdnIp()
```

### 4.3 Agent 侧 FQDN 更新

**文件位置**：agent/dp/ctrl.go:557-578

```go
func DPCtrlSetFqdnIp(fqdnip *share.CLUSFqdnIp) int {
    // 1. 过滤 IPv4 地址
    fips := make([]net.IP, 0, len(fqdnip.FqdnIP))
    for _, fip := range fqdnip.FqdnIP {
        if !utils.IsIPv4(fip) {
            continue  // 跳过 IPv6
        }
        fips = append(fips, fip)
    }

    // 2. 构造 JSON 消息
    Vhost := fqdnip.Vhost
    data := DPFqdnIpSetReq{
        Fqdns: &DPFqdnIps{
            FqdnName: fqdnip.FqdnName,
            FqdnIps:  fips,
            Vhost:    &Vhost,
        },
    }

    // 3. 发送到 dp
    msg, _ := json.Marshal(data)
    if dpSendMsg(msg) < 0 {
        return -1
    }

    return 0
}
```

**JSON 消息示例**：

```json
{
  "ctrl_cfg_set_fqdn": {
    "fqdn_name": "api.github.com",
    "vhost": false,
    "fqdn_ips": [
      "140.82.113.5",
      "140.82.114.5"
    ]
  }
}
```

---

## 五、双向映射机制

### 5.1 为什么需要双向映射？

#### 正向映射：FQDN → IP 列表

**用途**：查询域名对应的所有 IP

```c
// fqdn_name_map
"api.github.com" → {
    code: 10,
    iplist: [140.82.113.5, 140.82.114.5, ...]
}
```

**应用场景**：
- Agent 查询 FQDN 对应的 IP
- 显示策略详情时展示 IP 列表

#### 反向映射：IP → FQDN 列表

**用途**：查询 IP 对应的所有域名（一对多）

```c
// fqdn_ipv4_map
140.82.113.5 → {
    rlist: [api.github.com, github.com]
}
```

**应用场景**：
- 数据包匹配时，只有目标 IP
- 需要反查 IP 对应的 FQDN
- 然后用 FQDN 的 code 匹配策略

**为什么一对多？**
- 一个 IP 可以托管多个网站（虚拟主机）
- 例如：CDN 共享 IP，多个域名指向同一 IP

### 5.2 虚拟主机模式 (vhost)

**场景**：同一 IP 托管多个域名

```
IP: 192.168.1.100
  ├─ www.example.com
  ├─ blog.example.com
  └─ mail.example.com
```

**数据结构**：

```c
fqdn_ipv4_map:
  192.168.1.100 → {
    rlist: [
      www.example.com (code: 10),
      blog.example.com (code: 11),
      mail.example.com (code: 12)
    ]
  }
```

**匹配逻辑**：

```c
// 访问 192.168.1.100:443
ipv4_entry = lookup(fqdn_ipv4_map, 192.168.1.100);

// 遍历所有关联的 FQDN，逐一尝试匹配
for each fqdn in ipv4_entry->rlist:
    key.dip = fqdn.code;
    if (policy_lookup(hdl, key, desc) == 0):
        return ALLOW;  // 匹配成功

return DENY;  // 无匹配规则
```

---

## 六、通配符 FQDN 支持

### 6.1 通配符语法

NeuVector 支持前缀通配符：

| 语法 | 匹配范围 | 示例 |
|------|----------|------|
| `*.example.com` | 所有子域名 | `www.example.com`, `api.example.com` |
| `*.github.com` | GitHub 所有子域名 | `api.github.com`, `raw.githubusercontent.com` |
| `*` | 所有域名 | （不推荐使用） |

**不支持**：
- 中间通配符：`api.*.com` ❌
- 后缀通配符：`example.*` ❌

### 6.2 通配符检测

**文件位置**：dpi/dpi_policy.c

```c
static bool is_fqdn_name_wildcard(const char *name) {
    return (name[0] == '*');  // 检查首字符
}
```

**标记为通配符**：

```c
if (is_fqdn_name_wildcard(r->name)) {
    r->flag |= FQDN_RECORD_WILDCARD;
}
```

### 6.3 通配符匹配逻辑

**场景**：配置策略允许 `*.github.com`

#### Step 1: DNS 响应学习

```
DNS Response:
  api.github.com → 140.82.113.5
```

#### Step 2: 匹配通配符

```c
// 检查 "api.github.com" 是否匹配 "*.github.com"
bool match_wildcard_fqdn(const char *name, const char *pattern) {
    if (pattern[0] != '*') {
        return false;  // 非通配符
    }

    const char *suffix = pattern + 1;  // 跳过 '*'
    size_t name_len = strlen(name);
    size_t suffix_len = strlen(suffix);

    if (name_len < suffix_len) {
        return false;  // 名称太短
    }

    // 比较后缀
    return strcmp(name + name_len - suffix_len, suffix) == 0;
}

// match_wildcard_fqdn("api.github.com", "*.github.com")
//   → suffix = ".github.com"
//   → name[3:] = ".github.com"
//   → 匹配成功 ✅
```

#### Step 3: 关联 IP 到通配符 FQDN

```c
fqdn_name_map:
  "*.github.com" → {
    code: 10,
    flag: FQDN_RECORD_WILDCARD,
    iplist: [
      140.82.113.5,  // api.github.com
      185.199.108.153,  // pages.github.com
      ...
    ]
  }

fqdn_ipv4_map:
  140.82.113.5 → {
    rlist: [*.github.com (code: 10)]
  }
```

**匹配**：

访问 `api.github.com` (140.82.113.5) 时：
```
1. 查找 IP: 140.82.113.5
2. 反查 FQDN: *.github.com (code: 10)
3. 匹配策略: dip == 10 → ALLOW ✅
```

---

## 七、完整实战示例

### 7.1 场景：允许访问 GitHub API

**需求**：
- 允许容器访问 `api.github.com` 和 `*.githubusercontent.com`
- 拒绝其他 GitHub 子域名
- 默认拒绝所有流量

### 7.2 Step 1: Controller 配置策略

```yaml
# 策略规则
rules:
  - id: 1001
    from: "container-app"
    to: "nv.fqdn.api.github.com"
    ports: "tcp/443"
    applications: ["HTTPS"]
    action: "allow"

  - id: 1002
    from: "container-app"
    to: "nv.fqdn.*.githubusercontent.com"
    ports: "tcp/443"
    applications: ["HTTPS"]
    action: "allow"

  - id: 9999
    from: "container-app"
    to: "external"
    action: "deny"  # 默认拒绝
```

### 7.3 Step 2: Agent 下发到 dp

**JSON 消息**：

```json
{
  "ctrl_cfg_policy": {
    "cmd": 1,
    "flag": 3,
    "defact": 2,
    "dir": 3,
    "mac": ["aa:bb:cc:dd:ee:ff"],
    "rules": [
      {
        "id": 1001,
        "sip": "0.0.0.0",
        "dip": "0.0.0.0",
        "port": 443,
        "proto": 6,
        "action": 1,
        "fqdn": "api.github.com",
        "vhost": false,
        "apps": [{"rid": 1001, "app": 101, "action": 1}]
      },
      {
        "id": 1002,
        "sip": "0.0.0.0",
        "dip": "0.0.0.0",
        "port": 443,
        "proto": 6,
        "action": 1,
        "fqdn": "*.githubusercontent.com",
        "vhost": false,
        "apps": [{"rid": 1002, "app": 101, "action": 1}]
      }
    ]
  }
}
```

### 7.4 Step 3: dp 创建 FQDN 映射

```
fqdn_name_map:
  "api.github.com" → {code: 10, flag: 0, iplist: []}
  "*.githubusercontent.com" → {code: 11, flag: WILDCARD, iplist: []}

策略规则树:
  rule 1001: dip=10, dport=443, proto=6 → ALLOW
  rule 1002: dip=11, dport=443, proto=6 → ALLOW
```

### 7.5 Step 4: 容器发起 DNS 查询

```
Container → DNS Query: api.github.com
           ↓
DNS Server → DNS Response: api.github.com = 140.82.113.5
           ↓
dp 拦截 DNS 响应
           ↓
dp 更新映射:
  fqdn_name_map["api.github.com"].iplist.add(140.82.113.5)
  fqdn_ipv4_map[140.82.113.5].rlist.add(api.github.com, code=10)
```

### 7.6 Step 5: 数据包匹配

#### 场景 1: 访问 api.github.com

```
数据包: 192.168.1.10 → 140.82.113.5:443 (HTTPS)

匹配流程:
  1. 查找 IP: 140.82.113.5
  2. 反查 FQDN: api.github.com (code: 10)
  3. 匹配规则: dip=10, dport=443, proto=6 → rule 1001
  4. DPI 检测: app=HTTPS (101)
  5. 匹配应用规则: app=101 → ALLOW
  6. 返回: TC_ACT_OK ✅
```

#### 场景 2: 访问 raw.githubusercontent.com

```
DNS 查询学习:
  raw.githubusercontent.com → 185.199.108.133
  匹配通配符: *.githubusercontent.com → code 11
  更新映射: fqdn_ipv4_map[185.199.108.133] = [code: 11]

数据包: 192.168.1.10 → 185.199.108.133:443

匹配流程:
  1. 查找 IP: 185.199.108.133
  2. 反查 FQDN: *.githubusercontent.com (code: 11)
  3. 匹配规则: dip=11, dport=443 → rule 1002
  4. 返回: ALLOW ✅
```

#### 场景 3: 访问 github.com（主站）

```
DNS 查询: github.com → 140.82.114.4

数据包: 192.168.1.10 → 140.82.114.4:443

匹配流程:
  1. 查找 IP: 140.82.114.4
  2. 反查 FQDN: 无匹配（github.com 不在策略中）
  3. 应用默认动作: DENY
  4. 返回: TC_ACT_SHOT ❌
```

---

## 八、性能优化

### 8.1 Code 机制优化

**问题**：字符串比较慢

**解决方案**：将 FQDN 转换为整数 `code`

```c
// 慢速版本（字符串比较）
if (strcmp(policy_rule.fqdn, "api.github.com") == 0) {
    // 匹配成功
}

// 快速版本（整数比较）
if (policy_rule.dip == 10) {  // 10 是 api.github.com 的 code
    // 匹配成功
}
```

**性能提升**：
- 字符串比较：O(n)，n = 字符串长度
- 整数比较：O(1)
- **提升 10-100 倍**

### 8.2 RCU 无锁读取

**问题**：多线程并发访问 FQDN 映射表

**解决方案**：RCU (Read-Copy-Update)

```c
// 读取（无锁，超快）
rcu_read_lock();
fqdn_name_entry_t *entry = rcu_map_lookup(&hdl->fqdn_name_map, "api.github.com");
// ... 使用 entry
rcu_read_unlock();

// 写入（少量，加锁）
synchronize_rcu();  // 等待所有读者
rcu_map_add(&hdl->fqdn_name_map, entry, "api.github.com");
```

**性能特点**：
- 读操作：零锁开销
- 写操作：罕见（策略更新）
- 适合读多写少场景

### 8.3 内存池优化

**问题**：频繁分配/释放小对象导致碎片

**解决方案**：内存池

```c
// 预分配 FQDN 记录池
#define FQDN_POOL_SIZE 1024
fqdn_record_t *fqdn_pool[FQDN_POOL_SIZE];

fqdn_record_t* alloc_fqdn_record() {
    // 从池中分配
    if (pool_index < FQDN_POOL_SIZE) {
        return fqdn_pool[pool_index++];
    }
    return calloc(1, sizeof(fqdn_record_t));
}
```

### 8.4 容量限制

```c
#define DPI_FQDN_MAX_ENTRIES  DP_POLICY_FQDN_MAX_ENTRIES  // 通常 1024-4096

// 检查是否超过限制
if (hdl->code_cnt >= DPI_FQDN_MAX_ENTRIES) {
    DEBUG_ERROR(DBG_POLICY, "FQDN entries exceeded limit\n");
    return -1;
}
```

**建议**：
- 小型部署：1024 个 FQDN
- 中型部署：2048 个 FQDN
- 大型部署：4096 个 FQDN

---

## 总结

### ✅ NeuVector FQDN 功能完整实现

| 功能 | 实现状态 | 代码位置 |
|------|----------|----------|
| 基本 FQDN 匹配 | ✅ 完整 | dpi/dpi_policy.c |
| DNS 拦截学习 | ✅ 完整 | dpi/parsers/dpi_dns.c |
| 双向映射 | ✅ 完整 | fqdn_name_map + fqdn_ipv4_map |
| 通配符 FQDN | ✅ 完整 | `*.example.com` |
| 虚拟主机 | ✅ 完整 | vhost 模式 |
| 动态更新 | ✅ 完整 | Agent 定期更新 |
| 性能优化 | ✅ 完整 | Code 机制 + RCU |

### 核心优势

1. **零配置 DNS 学习**：自动拦截 DNS 响应
2. **高性能匹配**：Code 机制 + RCU 无锁
3. **通配符支持**：灵活的策略配置
4. **双向映射**：快速 IP ↔ FQDN 查找
5. **动态适应**：IP 变化自动更新

### 适用场景

- ✅ 云服务访问控制（AWS、GCP、Azure）
- ✅ API 网关策略（api.example.com）
- ✅ CDN 服务访问（*.cloudfront.net）
- ✅ 微服务间通信（service.namespace.svc.cluster.local）

NeuVector 的 FQDN 功能是一个**生产级、完整实现**的功能，适合在你的 eBPF 微隔离项目中参考！

---

**文档版本**：1.0
**最后更新**：2025-10-29
**相关文档**：
- [NeuVector Agent 与 dp 策略下发详解](./neuvector-agent-dp-policy-flow.md)
- [NeuVector dp 与 agent 通信机制详解](./neuvector-dp-agent-communication.md)
