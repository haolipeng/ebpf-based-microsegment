# NeuVector dp 与 agent 通信机制详解

## 目录

- [一、通信架构概览](#一通信架构概览)
- [二、Standalone 模式](#二standalone-模式)
- [三、核心通信流程](#三核心通信流程)
- [四、JSON 接口函数](#四json-接口函数)
- [五、典型消息示例](#五典型消息示例)
- [六、支持的命令类型](#六支持的命令类型)
- [七、关键数据结构](#七关键数据结构)
- [八、实战示例](#八实战示例)

---

## 一、通信架构概览

### 1.1 通信方式

**协议类型**：Unix Domain Socket (本地进程间通信)

**Socket 类型**：`SOCK_DGRAM` (数据报模式，无连接)

**数据格式**：JSON (基于 libjansson 库)

### 1.2 通信端点

| 端点 | Socket 路径 | 角色 | 定义位置 |
|------|-------------|------|----------|
| **dp 监听端点** | `/tmp/dp_listen.sock` | dp 接收 agent 命令 | ctrl.c:45 |
| **控制通知端点** | `/tmp/ctrl_listen.sock` | dp 主动通知 agent | ctrl.c:49 |

### 1.3 架构图

```
┌─────────────────┐                    ┌─────────────────┐
│     Agent       │                    │       dp        │
│   (控制平面)    │                    │   (数据平面)    │
├─────────────────┤                    ├─────────────────┤
│                 │                    │                 │
│  策略管理       │──── JSON over ────>│  策略执行       │
│  配置下发       │    Unix Socket     │  DPI 检测       │
│  统计查询       │<─── (dgram) ───────│  会话跟踪       │
│                 │                    │                 │
└─────────────────┘                    └─────────────────┘
        │                                      │
        │                                      │
        └──────── /tmp/dp_listen.sock ────────┘
        └──────── /tmp/ctrl_listen.sock ──────┘
```

### 1.4 通信特点

✅ **优势**：
- 本地通信，无网络开销
- 基于文件系统权限控制安全
- JSON 格式易于调试和扩展
- 支持双向通信（命令/响应、主动通知）

⚠️ **限制**：
- 仅限同主机进程通信
- SOCK_DGRAM 无连接，需应用层保证可靠性

---

## 二、Standalone 模式

### 2.1 什么是 Standalone 模式？

Standalone 模式允许 dp 组件独立运行，不依赖 agent 进程启动，但可以监听并响应来自控制通道的命令。

### 2.2 启动方式

```bash
# 基本启动
./dp -s

# 带调试信息启动
./dp -s -d all

# 指定数据处理线程数
./dp -s -n 4

# 完整示例：4 个数据线程 + 调试日志
./dp -s -n 4 -d "ctrl,packet,session"
```

### 2.3 命令行参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `-s` | 启用 standalone 模式 | `./dp -s` |
| `-d` | 调试类别 | `-d "ctrl,packet,session"` |
| `-n` | 数据处理线程数 | `-n 4` |
| `-p` | PCAP 文件分析模式 | `-p trace.pcap` |
| `-i` | 监听网络接口 | `-i eth0` |
| `-h` | 帮助信息 | `./dp -h` |

**调试类别**：
- `all` - 所有调试信息
- `init` - 初始化
- `error` - 错误信息
- `ctrl` - 控制消息
- `packet` - 数据包处理
- `session` - 会话跟踪
- `timer` - 定时器
- `tcp` - TCP 处理
- `parser` - 协议解析
- `log` - 日志
- `ddos` - DDoS 检测
- `policy` - 策略匹配
- `dlp` - 数据防泄漏

### 2.4 代码实现

**位置**：main.c:427-437

```c
else if (standalone) {
    // 设置回调函数
    g_callback.debug = debug_stdout;
    g_callback.send_packet = dp_send_packet;
    g_callback.send_ctrl_json = dp_ctrl_send_json;       // JSON 发送回调
    g_callback.send_ctrl_binary = dp_ctrl_send_binary;   // 二进制发送回调
    g_callback.threat_log = dp_ctrl_threat_log;
    g_callback.traffic_log = dp_ctrl_traffic_log;
    g_callback.connect_report = dp_ctrl_connect_report;

    dpi_setup(&g_callback, &g_config);
    // ... 启动线程并进入控制循环
}
```

---

## 三、核心通信流程

### 3.1 主控制循环 - `dp_ctrl_loop()`

**位置**：ctrl.c:3018-3079

**流程图**：

```
启动
 │
 ├─> 创建 Unix Socket (/tmp/dp_listen.sock)
 │
 ├─> 进入主循环
 │    │
 │    ├─> select() 等待消息 (2秒超时)
 │    │
 │    ├─> 收到消息？
 │    │    ├─ YES -> dp_ctrl_handler() 处理
 │    │    └─ NO  -> 继续
 │    │
 │    ├─> 每 2 秒执行：
 │    │    ├─ dp_ctrl_update_app()        # 更新应用识别
 │    │    ├─ dp_ctrl_update_fqdn_ip()    # 更新域名解析
 │    │    └─ dp_ctrl_consume_threat_log() # 消费威胁日志
 │    │
 │    └─> 每 6 秒执行：
 │         └─ dp_ctrl_update_connects()    # 更新连接报告
 │
 └─> 退出时清理资源
```

**关键代码**：

```c
void dp_ctrl_loop(void)
{
    // 1. 创建监听 socket
    unlink(DP_SERVER_SOCK);                          // 删除旧 socket 文件
    g_ctrl_fd = make_named_socket(DP_SERVER_SOCK);   // 创建 /tmp/dp_listen.sock
    g_ctrl_notify_fd = make_notify_client(CTRL_NOTIFY_SOCK);

    // 2. 主循环
    while (g_running) {
        timeout.tv_sec = 2;
        timeout.tv_usec = 0;

        // 3. 等待消息
        FD_ZERO(&read_fds);
        FD_SET(g_ctrl_fd, &read_fds);
        ret = select(g_ctrl_fd + 1, &read_fds, NULL, NULL, &timeout);

        // 4. 处理消息
        if (ret > 0 && FD_ISSET(g_ctrl_fd, &read_fds)) {
            dp_ctrl_handler(g_ctrl_fd);   // 核心处理函数
        }

        // 5. 定期任务（每 2 秒）
        if (now.tv_sec - last.tv_sec >= 2) {
            dp_ctrl_update_app(false);
            dp_ctrl_update_fqdn_ip();
            dp_ctrl_consume_threat_log();

            // 每 6 秒
            if ((round % 3) == 0) {
                dp_ctrl_update_connects();
            }
            round++;
        }
    }

    // 6. 清理资源
    close(g_ctrl_notify_fd);
    close(g_ctrl_fd);
    unlink(DP_SERVER_SOCK);
}
```

### 3.2 消息处理器 - `dp_ctrl_handler()`

**位置**：ctrl.c:2384-2464

**处理流程**：

```
接收消息
    │
    ├─> recvfrom() 接收 JSON 字符串
    │
    ├─> json_loads() 解析 JSON
    │    ├─ 成功 -> 继续
    │    └─ 失败 -> 返回错误
    │
    ├─> json_object_foreach() 遍历 JSON 对象
    │
    ├─> 根据 key 分发到对应处理函数
    │    ├─ "ctrl_add_mac" -> dp_ctrl_add_mac()
    │    ├─ "ctrl_cfg_policy" -> dp_ctrl_cfg_policy()
    │    ├─ "ctrl_stats_device" -> dp_ctrl_stats_device()
    │    └─ ... (30+ 种命令)
    │
    └─> 返回处理结果
```

**关键代码**：

```c
static int dp_ctrl_handler(int fd)
{
    socklen_t len;
    int size, ret = 0;

    // 1. 接收消息
    len = sizeof(struct sockaddr_un);
    size = recvfrom(fd, ctrl_msg_buf, BUF_SIZE - 1, 0,
                    (struct sockaddr *)&g_client_addr, &len);
    ctrl_msg_buf[size] = '\0';

    // 2. 解析 JSON
    json_t *root;
    json_error_t error;

    root = json_loads(ctrl_msg_buf, 0, &error);
    if (root == NULL) {
        DEBUG_ERROR(DBG_CTRL, "Invalid json format on line %d: %s\n",
                    error.line, error.text);
        return -1;
    }

    // 3. 遍历 JSON 对象并分发
    const char *key;
    json_t *msg;

    json_object_foreach(root, key, msg) {
        if (strcmp(key, "ctrl_keep_alive") == 0) {
            ret = dp_ctrl_keep_alive(msg);
            continue;
        }

        // 调试输出
        char *data = json_dumps(msg, JSON_ENSURE_ASCII);
        DEBUG_CTRL("\"%s\":%s\n", key, data);
        free(data);

        // 命令分发
        if (strcmp(key, "ctrl_add_mac") == 0) {
            ret = dp_ctrl_add_mac(msg);
        } else if (strcmp(key, "ctrl_cfg_policy") == 0) {
            ret = dp_ctrl_cfg_policy(msg);
        } else if (strcmp(key, "ctrl_stats_device") == 0) {
            ret = dp_ctrl_stats_device(msg);
        }
        // ... 30+ 种命令
    }

    json_decref(root);
    return ret;
}
```

---

## 四、JSON 接口函数

### 4.1 libjansson 核心 API

NeuVector dp 使用 **libjansson** 库进行 JSON 操作。

#### 解析类函数

| 函数 | 功能 | 返回值 | 示例 |
|------|------|--------|------|
| `json_loads()` | 解析 JSON 字符串 | `json_t*` | `root = json_loads(str, 0, &error)` |
| `json_object_get()` | 获取对象字段 | `json_t*` | `obj = json_object_get(root, "key")` |
| `json_string_value()` | 获取字符串值 | `const char*` | `str = json_string_value(obj)` |
| `json_string_length()` | 获取字符串长度 | `size_t` | `len = json_string_length(obj)` |
| `json_integer_value()` | 获取整数值 | `json_int_t` | `num = json_integer_value(obj)` |
| `json_boolean_value()` | 获取布尔值 | `int` | `flag = json_boolean_value(obj)` |
| `json_array_size()` | 获取数组大小 | `size_t` | `count = json_array_size(arr)` |
| `json_array_get()` | 获取数组元素 | `json_t*` | `item = json_array_get(arr, i)` |

#### 构造类函数

| 函数 | 功能 | 示例 |
|------|------|------|
| `json_object()` | 创建空对象 | `obj = json_object()` |
| `json_array()` | 创建空数组 | `arr = json_array()` |
| `json_string()` | 创建字符串 | `s = json_string("hello")` |
| `json_integer()` | 创建整数 | `i = json_integer(42)` |
| `json_pack()` | 格式化构造 | `obj = json_pack("{s:s, s:i}", "name", "dp", "port", 80)` |
| `json_dumps()` | 序列化为字符串 | `str = json_dumps(obj, JSON_ENSURE_ASCII)` |

#### 内存管理

| 函数 | 功能 | 说明 |
|------|------|------|
| `json_decref()` | 减少引用计数 | 引用计数归零时自动释放 |
| `json_incref()` | 增加引用计数 | 用于共享 JSON 对象 |

### 4.2 dp 封装的发送函数

#### `dp_ctrl_send_json()` - 发送 JSON 响应

**位置**：ctrl.c:102-125

**函数签名**：
```c
int dp_ctrl_send_json(json_t *root)
```

**功能**：将 JSON 对象序列化并通过 Unix Socket 发送给 agent。

**实现**：

```c
int dp_ctrl_send_json(json_t *root)
{
    if (root == NULL) {
        DEBUG_ERROR(DBG_CTRL, "Fail to create json object.\n");
        return -1;
    }

    // 1. JSON 对象序列化为字符串
    char *data = json_dumps(root, JSON_ENSURE_ASCII);
    if (data == NULL) {
        json_decref(root);
        return -1;
    }

    // 2. 通过 Unix Socket 发送
    socklen_t addr_len = sizeof(struct sockaddr_un);
    int sent = sendto(g_ctrl_fd, data, strlen(data), 0,
                      (struct sockaddr *)&g_client_addr, addr_len);

    // 3. 释放资源
    free(data);          // 释放 json_dumps() 分配的字符串
    json_decref(root);   // 释放 JSON 对象

    return sent;
}
```

**使用示例**：

```c
// 构造响应 JSON
json_t *root = json_pack("{s:{s:i, s:s}}",
                         "response",
                         "status", 0,
                         "message", "success");

// 发送
dp_ctrl_send_json(root);
// root 会被自动释放，无需手动 decref
```

#### `dp_ctrl_send_binary()` - 发送二进制响应

**位置**：ctrl.c:127-138

**函数签名**：
```c
int dp_ctrl_send_binary(void *data, int len)
```

**功能**：发送二进制数据（如统计信息、会话列表等结构化数据）。

---

## 五、典型消息示例

### 5.1 添加 MAC 地址

#### 命令：`ctrl_add_mac`

**功能**：注册容器或虚拟机的 MAC 地址到 dp。

**agent 发送的 JSON**：

```json
{
  "ctrl_add_mac": {
    "iface": "eth0",
    "mac": "aa:bb:cc:dd:ee:ff",
    "ucmac": "11:22:33:44:55:66",
    "bcmac": "ff:ff:ff:ff:ff:ff",
    "oldmac": "",
    "pmac": "00:11:22:33:44:55",
    "pips": [
      {"ip": "192.168.1.10"},
      {"ip": "192.168.1.11"}
    ]
  }
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `iface` | string | 网络接口名称 |
| `mac` | string | 原始 MAC 地址 |
| `ucmac` | string | 单播 MAC 地址 |
| `bcmac` | string | 广播 MAC 地址 |
| `oldmac` | string | 旧 MAC 地址（迁移时使用） |
| `pmac` | string | ProxyMesh 原始 MAC |
| `pips` | array | ProxyMesh 父节点 IP 列表 |

**dp 处理代码**（ctrl.c:408-588）：

```c
static int dp_ctrl_add_mac(json_t *msg)
{
    const char *iface, *mac_str, *ucmac_str, *bcmac_str, *pmac_str;
    io_internal_pip_t *pips = NULL;
    int count = 0;

    // 1. 解析基本字段
    iface = json_string_value(json_object_get(msg, "iface"));
    mac_str = json_string_value(json_object_get(msg, "mac"));
    ucmac_str = json_string_value(json_object_get(msg, "ucmac"));
    bcmac_str = json_string_value(json_object_get(msg, "bcmac"));
    pmac_str = json_string_value(json_object_get(msg, "pmac"));

    // 2. 解析 ProxyMesh IP 数组
    json_t *obj = json_object_get(msg, "pips");
    if (obj) {
        count = json_array_size(obj);
        pips = calloc(sizeof(io_internal_pip_t) + count * sizeof(io_pip_t), 1);

        pips->count = count;
        for (int i = 0; i < count; i++) {
            json_t *nw_obj = json_array_get(obj, i);
            pips->list[i].ip = inet_addr(
                json_string_value(json_object_get(nw_obj, "ip"))
            );
        }
    }

    // 3. 分配内存并填充结构体
    // ... (创建 io_ep_t 和 io_mac_t 结构)

    // 4. 插入到全局 MAP
    rcu_map_add(&g_ep_map, mac, &mac->node);

    return 0;
}
```

### 5.2 配置策略规则

#### 命令：`ctrl_cfg_policy`

**功能**：下发网络访问策略到 dp。

**agent 发送的 JSON**：

```json
{
  "ctrl_cfg_policy": {
    "cmd": 1,
    "flag": 0,
    "defact": 1,
    "dir": 3,
    "mac": ["aa:bb:cc:dd:ee:ff"],
    "rules": [
      {
        "id": 1001,
        "sip": "192.168.1.0",
        "sipr": "192.168.1.255",
        "dip": "10.0.0.0",
        "dipr": "10.0.0.255",
        "port": 80,
        "portr": 443,
        "proto": 6,
        "action": 2,
        "ingress": true,
        "vhost": false,
        "fqdn": "example.com",
        "apps": [
          {"rid": 1, "app": 100, "action": 1},
          {"rid": 2, "app": 200, "action": 2}
        ]
      }
    ]
  }
}
```

**字段说明**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `cmd` | int | 命令类型（1=添加, 2=删除, 3=修改） |
| `flag` | int | 标志位 |
| `defact` | int | 默认动作（1=允许, 2=拒绝） |
| `dir` | int | 应用方向（1=入站, 2=出站, 3=双向） |
| `mac` | array | 应用此策略的 MAC 地址列表 |
| `rules` | array | 策略规则列表 |

**规则字段**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | int | 规则 ID |
| `sip` | string | 源 IP 起始 |
| `sipr` | string | 源 IP 结束（范围） |
| `dip` | string | 目标 IP 起始 |
| `dipr` | string | 目标 IP 结束 |
| `port` | int | 目标端口起始 |
| `portr` | int | 目标端口结束 |
| `proto` | int | 协议（6=TCP, 17=UDP） |
| `action` | int | 动作（1=允许, 2=拒绝, 3=学习） |
| `ingress` | bool | 是否入站规则 |
| `vhost` | bool | 是否虚拟主机模式 |
| `fqdn` | string | 域名匹配（可选） |
| `apps` | array | 应用层规则（可选） |

**dp 处理代码**（ctrl.c:1358-1463）：

```c
static int dp_ctrl_cfg_policy(json_t *msg)
{
    int cmd, flag;
    dpi_policy_t policy;

    // 1. 解析策略头部
    cmd = json_integer_value(json_object_get(msg, "cmd"));
    flag = json_integer_value(json_object_get(msg, "flag"));
    policy.def_action = json_integer_value(json_object_get(msg, "defact"));
    policy.apply_dir = json_integer_value(json_object_get(msg, "dir"));

    // 2. 解析 MAC 列表
    json_t *obj = json_object_get(msg, "mac");
    policy.num_macs = json_array_size(obj);
    policy.mac_list = calloc(policy.num_macs, sizeof(struct ether_addr));

    for (int i = 0; i < policy.num_macs; i++) {
        const char *mac_str = json_string_value(json_array_get(obj, i));
        ether_aton_r(mac_str, &policy.mac_list[i]);
    }

    // 3. 解析规则列表
    obj = json_object_get(msg, "rules");
    policy.num_rules = json_array_size(obj);
    policy.rule_list = calloc(policy.num_rules, sizeof(dpi_policy_rule_t));

    for (int i = 0; i < policy.num_rules; i++) {
        json_t *rule_obj = json_array_get(obj, i);

        policy.rule_list[i].id = json_integer_value(
            json_object_get(rule_obj, "id")
        );
        policy.rule_list[i].sip = inet_addr(
            json_string_value(json_object_get(rule_obj, "sip"))
        );
        policy.rule_list[i].dip = inet_addr(
            json_string_value(json_object_get(rule_obj, "dip"))
        );
        policy.rule_list[i].dport = json_integer_value(
            json_object_get(rule_obj, "port")
        );
        policy.rule_list[i].proto = json_integer_value(
            json_object_get(rule_obj, "proto")
        );
        policy.rule_list[i].action = json_integer_value(
            json_object_get(rule_obj, "action")
        );
        policy.rule_list[i].ingress = json_boolean_value(
            json_object_get(rule_obj, "ingress")
        );

        // 4. 解析 FQDN（可选）
        json_t *fqdn_obj = json_object_get(rule_obj, "fqdn");
        if (fqdn_obj != NULL) {
            strlcpy(policy.rule_list[i].fqdn,
                    json_string_value(fqdn_obj),
                    MAX_FQDN_LEN);
        }

        // 5. 解析应用层规则（可选）
        json_t *app_obj = json_object_get(rule_obj, "apps");
        if (app_obj != NULL) {
            int num_apps = json_array_size(app_obj);
            policy.rule_list[i].num_apps = num_apps;
            policy.rule_list[i].app_rules = calloc(num_apps,
                sizeof(dpi_policy_app_rule_t));

            for (int j = 0; j < num_apps; j++) {
                json_t *app_rule_obj = json_array_get(app_obj, j);
                policy.rule_list[i].app_rules[j].rule_id =
                    json_integer_value(json_object_get(app_rule_obj, "rid"));
                policy.rule_list[i].app_rules[j].app =
                    json_integer_value(json_object_get(app_rule_obj, "app"));
                policy.rule_list[i].app_rules[j].action =
                    json_integer_value(json_object_get(app_rule_obj, "action"));
            }
        }
    }

    // 6. 调用策略配置函数
    int ret = dpi_policy_cfg(cmd, &policy, flag);

    // 7. 释放内存
    free(policy.mac_list);
    for (int i = 0; i < policy.num_rules; i++) {
        if (policy.rule_list[i].app_rules) {
            free(policy.rule_list[i].app_rules);
        }
    }
    free(policy.rule_list);

    return ret;
}
```

### 5.3 查询设备统计

#### 命令：`ctrl_stats_device`

**功能**：查询 dp 的统计信息（会话数、数据包、字节数等）。

**agent 发送的 JSON**：

```json
{
  "ctrl_stats_device": {}
}
```

**dp 响应**（二进制格式）：

dp 不返回 JSON，而是返回二进制结构体：

```c
typedef struct {
    DPMsgHdr hdr;           // 消息头
    DPMsgStats stats;       // 统计数据
} StatsResponse;
```

**dp 处理代码**（ctrl.c:1037-1127）：

```c
static int dp_ctrl_stats_device(json_t *msg)
{
    uint8_t buf[sizeof(DPMsgHdr) + sizeof(DPMsgStats)];
    io_stats_t stats;

    // 1. 收集统计数据
    memset(&stats, 0, sizeof(stats));
    dpi_get_stats(&stats, NULL);

    // 2. 构造消息头
    DPMsgHdr *hdr = (DPMsgHdr *)buf;
    hdr->Kind = DP_KIND_DEVICE_STATS;
    hdr->Length = htons(sizeof(DPMsgStats));
    hdr->More = 0;

    // 3. 填充统计数据
    DPMsgStats *s = (DPMsgStats *)(hdr + 1);
    s->CurSess = htonl(stats.in.cur_session);
    s->TotalSess = htonll(stats.in.session);
    s->TotalPkts = htonll(stats.in.packet);
    s->TotalBytes = htonll(stats.in.byte);

    // 填充环形缓冲区统计（最近 60 个采样点）
    for (int i = 0; i < STATS_SLOTS; i++) {
        s->SessRing[i] = htonl(stats.in.sess_ring[i]);
        s->PktRing[i] = htonl(stats.in.pkt_ring[i]);
        s->ByteRing[i] = htonl(stats.in.byte_ring[i]);
    }

    // 4. 发送二进制响应
    return dp_ctrl_send_binary(buf, sizeof(buf));
}
```

### 5.4 设置 FQDN 映射

#### 命令：`ctrl_cfg_set_fqdn`

**功能**：配置域名到 IP 的映射，用于基于域名的策略匹配。

**agent 发送的 JSON**：

```json
{
  "ctrl_cfg_set_fqdn": {
    "fqdn_name": "api.example.com",
    "vhost": false,
    "fqdn_ips": [
      "52.1.2.3",
      "52.1.2.4",
      "52.1.2.5"
    ]
  }
}
```

**dp 处理代码**（ctrl.c:1465-1492）：

```c
static int dp_ctrl_set_fqdn(json_t *msg)
{
    char fqdname[MAX_FQDN_LEN];
    json_t *obj, *vhost_obj;
    bool vhost = false;
    uint32_t fqdnip;
    int count;

    // 1. 解析虚拟主机标志
    vhost_obj = json_object_get(msg, "vhost");
    if (vhost_obj != NULL) {
        vhost = json_boolean_value(vhost_obj);
    }

    // 2. 解析域名
    strlcpy(fqdname,
            json_string_value(json_object_get(msg, "fqdn_name")),
            MAX_FQDN_LEN);

    // 3. 解析 IP 列表
    obj = json_object_get(msg, "fqdn_ips");
    count = json_array_size(obj);

    rcu_read_lock();
    for (int i = 0; i < count; i++) {
        fqdnip = inet_addr(json_string_value(json_array_get(obj, i)));
        config_fqdn_ipv4_mapping(g_fqdn_hdl, fqdname, fqdnip, vhost);
    }
    rcu_read_unlock();

    return 0;
}
```

**使用场景**：

```
策略规则：允许访问 "api.example.com"
    ↓
agent 查询 DNS: api.example.com -> [52.1.2.3, 52.1.2.4, 52.1.2.5]
    ↓
agent 下发 FQDN 映射到 dp
    ↓
dp 匹配数据包时：
  - 如果 dst_ip == 52.1.2.3/4/5 -> 匹配 "api.example.com" 规则
```

### 5.5 心跳保活

#### 命令：`ctrl_keep_alive`

**功能**：维持 agent 和 dp 之间的连接，检测对方是否存活。

**agent 发送的 JSON**：

```json
{
  "ctrl_keep_alive": {
    "seq_num": 12345
  }
}
```

**dp 响应**（二进制格式）：

```c
// 简单的确认消息
DPMsgHdr {
    Kind: DP_KIND_KEEP_ALIVE,
    SeqNum: 12345  // 回显序列号
}
```

**dp 处理代码**（ctrl.c:141-156）：

```c
static int dp_ctrl_keep_alive(json_t *msg)
{
    uint32_t seq_num = json_integer_value(json_object_get(msg, "seq_num"));
    uint8_t buf[sizeof(DPMsgHdr) + sizeof(uint32_t)];

    DPMsgHdr *hdr = (DPMsgHdr *)buf;
    hdr->Kind = DP_KIND_KEEP_ALIVE;
    hdr->Length = htons(sizeof(uint32_t));
    hdr->SeqNum = htonl(seq_num);  // 回显序列号

    uint32_t *data = (uint32_t *)(hdr + 1);
    *data = htonl(seq_num);

    return dp_ctrl_send_binary(buf, sizeof(buf));
}
```

---

## 六、支持的命令类型

dp 支持 **30+ 种**控制命令，涵盖端口管理、MAC 管理、策略配置、统计查询等。

### 6.1 端口管理

| 命令 | JSON Key | 功能 | 参数 |
|------|----------|------|------|
| 添加服务端口 | `ctrl_add_srvc_port` | 添加网络接口到数据处理 | `iface`, `jumboframe` |
| 删除服务端口 | `ctrl_del_srvc_port` | 移除网络接口 | `iface` |
| 添加 TAP 端口 | `ctrl_add_tap_port` | 添加 TAP 设备 | `netns`, `iface`, `epmac` |
| 删除 TAP 端口 | `ctrl_del_tap_port` | 移除 TAP 设备 | `netns`, `iface` |
| 添加 NFQ 端口 | `ctrl_add_nfq_port` | 添加 Netfilter Queue | `netns`, `iface`, `qnum`, `epmac` |
| 删除 NFQ 端口 | `ctrl_del_nfq_port` | 移除 NFQ | `netns`, `iface` |
| 添加端口对 | `ctrl_add_port_pair` | 添加进/出端口对 | `vin_iface`, `vex_iface`, `epmac`, `quar` |
| 删除端口对 | `ctrl_del_port_pair` | 删除端口对 | `vin_iface`, `vex_iface` |

**示例 - 添加 Netfilter Queue**：

```json
{
  "ctrl_add_nfq_port": {
    "netns": "/proc/12345/ns/net",
    "iface": "eth0",
    "qnum": 0,
    "epmac": "aa:bb:cc:dd:ee:ff",
    "jumboframe": false
  }
}
```

### 6.2 MAC 地址管理

| 命令 | JSON Key | 功能 | 参数 |
|------|----------|------|------|
| 添加 MAC | `ctrl_add_mac` | 注册容器/VM MAC | `iface`, `mac`, `ucmac`, `bcmac`, `pips` |
| 删除 MAC | `ctrl_del_mac` | 注销 MAC 地址 | `mac` |
| 配置 MAC | `ctrl_cfg_mac` | 更新 MAC 配置 | `macs`, `tap`, `apps` |
| 配置 NBE | `ctrl_cfg_nbe` | 配置网络行为引擎 | `macs`, `nbe` |
| 刷新应用 | `ctrl_refresh_app` | 刷新应用识别数据 | 无 |

**示例 - 配置 MAC 应用端口**：

```json
{
  "ctrl_cfg_mac": {
    "macs": ["aa:bb:cc:dd:ee:ff"],
    "tap": false,
    "apps": [
      {"port": 80, "ip_proto": 6, "app": 100, "server": 1},
      {"port": 443, "ip_proto": 6, "app": 101, "server": 1},
      {"port": 3306, "ip_proto": 6, "app": 200, "server": 1}
    ]
  }
}
```

### 6.3 策略管理

| 命令 | JSON Key | 功能 | 参数 |
|------|----------|------|------|
| 配置策略 | `ctrl_cfg_policy` | 下发访问控制策略 | `cmd`, `mac`, `defact`, `rules` |
| 设置 FQDN | `ctrl_cfg_set_fqdn` | 配置域名映射 | `fqdn_name`, `fqdn_ips`, `vhost` |
| 删除 FQDN | `ctrl_cfg_del_fqdn` | 删除域名映射 | `names` |
| 配置内网 | `ctrl_cfg_internal_subnet` | 配置内网子网 | `subnet_addr`, `flag` |
| 配置特殊 IP | `ctrl_cfg_specialip_net` | 配置特殊 IP 类型 | `subnet_addr`, `flag` |
| 配置地址映射 | `ctrl_cfg_pol_addr` | 配置策略地址映射 | `subnet_addr`, `flag` |

**示例 - 配置内网子网**：

```json
{
  "ctrl_cfg_internal_subnet": {
    "flag": 1,
    "subnet_addr": [
      {"ip": "10.0.0.0", "mask": "255.0.0.0"},
      {"ip": "172.16.0.0", "mask": "255.240.0.0"},
      {"ip": "192.168.0.0", "mask": "255.255.0.0"}
    ]
  }
}
```

### 6.4 统计与监控

| 命令 | JSON Key | 功能 | 返回 |
|------|----------|------|------|
| 设备统计 | `ctrl_stats_device` | 查询设备级统计 | `DPMsgStats` (binary) |
| MAC 统计 | `ctrl_stats_macs` | 查询指定 MAC 统计 | `DPMsgStats` (binary) |
| 设备计数器 | `ctrl_counter_device` | 查询设备计数器 | `DPMsgDeviceCounter` (binary) |
| 会话计数 | `ctrl_count_session` | 查询会话总数 | `DPMsgSessionCount` (binary) |
| 会话列表 | `ctrl_list_session` | 列出所有活动会话 | `DPMsgSession[]` (binary) |
| 清除会话 | `ctrl_clear_session` | 删除指定会话 | 无 |
| 计量列表 | `ctrl_list_meter` | 列出计量信息 | `DPMsgMeter[]` (binary) |

**示例 - 查询 MAC 统计**：

```json
{
  "ctrl_stats_macs": {
    "macs": [
      "aa:bb:cc:dd:ee:ff",
      "11:22:33:44:55:66"
    ]
  }
}
```

### 6.5 DLP/WAF 管理

| 命令 | JSON Key | 功能 | 参数 |
|------|----------|------|------|
| DLP 构建 | `ctrl_dlp_bld` | 构建 DLP 检测规则 | `mac`, `dir`, `dlp_rules` |
| DLP 删除 | `ctrl_dlp_del` | 删除 DLP 规则 | `del_mac_list` |
| WAF 构建 | `ctrl_waf_bld` | 构建 WAF 检测规则 | `mac`, `dir`, `waf_rules` |
| WAF 删除 | `ctrl_waf_del` | 删除 WAF 规则 | `del_mac_list` |

### 6.6 调试与控制

| 命令 | JSON Key | 功能 | 参数 |
|------|----------|------|------|
| 设置调试级别 | `ctrl_set_debug` | 动态调整日志级别 | `categories` |
| 心跳保活 | `ctrl_keep_alive` | 维持连接 | `seq_num` |

**示例 - 设置调试级别**：

```json
{
  "ctrl_set_debug": {
    "categories": ["ctrl", "packet", "session", "policy"]
  }
}
```

---

## 七、关键数据结构

### 7.1 回调接口 - `io_callback_t`

**位置**：apis.h:210-218

**定义**：

```c
typedef struct io_callback_ {
    int (*debug) (bool print_ts, const char *fmt, va_list args);
    int (*send_packet) (io_ctx_t *ctx, uint8_t *data, int len);
    int (*send_ctrl_json) (json_t *root);        // JSON 发送接口 ⭐
    int (*send_ctrl_binary) (void *buf, int len); // 二进制发送接口 ⭐
    int (*threat_log) (DPMsgThreatLog *log);
    int (*traffic_log) (DPMsgSession *log);
    int (*connect_report) (DPMsgSession *log, DPMonitorMetric *metric,
                           int count_session, int count_violate);
} io_callback_t;
```

**用途**：

- dp 核心引擎通过回调函数与外部模块通信
- Standalone 模式下，回调函数指向 `dp_ctrl_send_json()` 等 Unix Socket 发送函数
- PCAP 分析模式下，回调函数指向文件输出函数

### 7.2 端点信息 - `io_ep_t`

**位置**：apis.h:115-150

**定义**：

```c
typedef struct io_ep_ {
    char iface[IFACE_NAME_LEN];     // 网络接口名
    struct io_mac_ *mac;            // 原始 MAC
    struct io_mac_ *ucmac;          // 单播 MAC
    struct io_mac_ *bcmac;          // 广播 MAC
    struct ether_addr pmac;         // ProxyMesh MAC
    io_internal_pip_t *pips;        // ProxyMesh 父节点 IP

    io_stats_t stats;               // 统计信息

    rcu_map_t app_map;              // 应用端口映射
    uint16_t app_ports;             // 应用端口数量

    bool tap;                       // 是否 TAP 模式
    uint8_t cassandra_svr: 1;       // Cassandra 服务器标志
    uint8_t kafka_svr:     1;       // Kafka 服务器标志
    uint8_t zookeeper_svr: 1;       // ZooKeeper 服务器标志
    // ...

    void *policy_hdl;               // 策略句柄
    uint16_t policy_ver;            // 策略版本

    rcu_map_t dlp_cfg_map;          // DLP 配置
    rcu_map_t waf_cfg_map;          // WAF 配置
    void *dlp_detector;             // DLP 检测器
} io_ep_t;
```

### 7.3 策略规则 - `dpi_policy_rule_t`

**位置**：apis.h:255-270

**定义**：

```c
typedef struct dpi_policy_rule_ {
    uint32_t id;                    // 规则 ID
    uint32_t sip;                   // 源 IP 起始
    uint32_t sip_r;                 // 源 IP 结束
    uint32_t dip;                   // 目标 IP 起始
    uint32_t dip_r;                 // 目标 IP 结束
    uint16_t dport;                 // 目标端口起始
    uint16_t dport_r;               // 目标端口结束
    uint16_t proto;                 // 协议（TCP/UDP）
    uint8_t action;                 // 动作（允许/拒绝）
    bool ingress;                   // 入站/出站
    bool vh;                        // 虚拟主机模式
    char fqdn[MAX_FQDN_LEN];        // 域名匹配
    uint32_t num_apps;              // 应用规则数量
    dpi_policy_app_rule_t *app_rules; // 应用规则列表
} dpi_policy_rule_t;
```

### 7.4 统计信息 - `io_stats_t`

**位置**：apis.h:78-83

**定义**：

```c
typedef struct io_stats_ {
    uint32_t cur_slot;              // 当前时间槽
    io_metry_t in;                  // 入站统计
    io_metry_t out;                 // 出站统计
} io_stats_t;

typedef struct io_metry_ {
    uint64_t session;               // 总会话数
    uint64_t packet;                // 总数据包数
    uint64_t byte;                  // 总字节数
    uint32_t sess_ring[STATS_SLOTS]; // 会话数环形缓冲（60 个槽）
    uint32_t pkt_ring[STATS_SLOTS];  // 数据包环形缓冲
    uint32_t byte_ring[STATS_SLOTS]; // 字节数环形缓冲
    uint32_t cur_session;            // 当前活动会话数
} io_metry_t;
```

**环形缓冲区说明**：

- 每个槽代表 5 秒的统计数据
- 60 个槽 × 5 秒 = 5 分钟的历史数据
- 用于生成时间序列图表

---

## 八、实战示例

### 8.1 完整的策略下发流程

**场景**：agent 需要为容器 `container-1` 下发访问策略，允许访问 MySQL (3306) 但拒绝访问 Redis (6379)。

#### Step 1: agent 构造 JSON 消息

```json
{
  "ctrl_cfg_policy": {
    "cmd": 1,
    "flag": 0,
    "defact": 2,
    "dir": 3,
    "mac": ["aa:bb:cc:dd:ee:ff"],
    "rules": [
      {
        "id": 1001,
        "sip": "0.0.0.0",
        "sipr": "255.255.255.255",
        "dip": "10.20.30.40",
        "dipr": "10.20.30.40",
        "port": 3306,
        "portr": 3306,
        "proto": 6,
        "action": 1,
        "ingress": false,
        "vhost": false,
        "fqdn": "",
        "apps": []
      },
      {
        "id": 1002,
        "sip": "0.0.0.0",
        "sipr": "255.255.255.255",
        "dip": "10.20.30.50",
        "dipr": "10.20.30.50",
        "port": 6379,
        "portr": 6379,
        "proto": 6,
        "action": 2,
        "ingress": false,
        "vhost": false,
        "fqdn": "",
        "apps": []
      }
    ]
  }
}
```

#### Step 2: agent 发送到 dp

```python
import socket
import json

# 1. 创建 Unix Socket
sock = socket.socket(socket.AF_UNIX, socket.SOCK_DGRAM)

# 2. 构造消息
policy_msg = {
    "ctrl_cfg_policy": {
        "cmd": 1,
        "flag": 0,
        "defact": 2,
        "dir": 3,
        "mac": ["aa:bb:cc:dd:ee:ff"],
        "rules": [
            {
                "id": 1001,
                "sip": "0.0.0.0",
                "sipr": "255.255.255.255",
                "dip": "10.20.30.40",
                "dipr": "10.20.30.40",
                "port": 3306,
                "portr": 3306,
                "proto": 6,
                "action": 1,
                "ingress": False,
                "vhost": False,
                "fqdn": "",
                "apps": []
            },
            {
                "id": 1002,
                "sip": "0.0.0.0",
                "sipr": "255.255.255.255",
                "dip": "10.20.30.50",
                "dipr": "10.20.30.50",
                "port": 6379,
                "portr": 6379,
                "proto": 6,
                "action": 2,
                "ingress": False,
                "vhost": False,
                "fqdn": "",
                "apps": []
            }
        ]
    }
}

# 3. 序列化为 JSON 字符串
msg_str = json.dumps(policy_msg)

# 4. 发送到 dp
sock.sendto(msg_str.encode('utf-8'), '/tmp/dp_listen.sock')

print(f"Sent policy: {len(msg_str)} bytes")
sock.close()
```

#### Step 3: dp 接收并处理

```
dp_ctrl_loop()
    ↓ select() 检测到消息
    ↓
dp_ctrl_handler()
    ↓ recvfrom() 接收 JSON
    ↓ json_loads() 解析
    ↓ 检测到 key = "ctrl_cfg_policy"
    ↓
dp_ctrl_cfg_policy()
    ↓ 解析 mac, rules
    ↓ 构造 dpi_policy_t 结构
    ↓
dpi_policy_cfg()
    ↓ 编译策略规则
    ↓ 插入到策略树
    ↓ 返回成功
```

#### Step 4: 策略生效

```
数据包到达（dst=10.20.30.40:3306）
    ↓
dpi_recv_packet()
    ↓
策略匹配：规则 1001 (MySQL)
    ↓ action = 1 (允许)
    ↓
TC_ACT_OK (放行)

---

数据包到达（dst=10.20.30.50:6379）
    ↓
dpi_recv_packet()
    ↓
策略匹配：规则 1002 (Redis)
    ↓ action = 2 (拒绝)
    ↓
TC_ACT_SHOT (丢弃)
```

### 8.2 查询统计信息并绘图

**场景**：agent 需要查询最近 5 分钟的流量趋势。

#### Step 1: 发送查询请求

```python
import socket
import json

sock = socket.socket(socket.AF_UNIX, socket.SOCK_DGRAM)

query_msg = json.dumps({"ctrl_stats_device": {}})
sock.sendto(query_msg.encode('utf-8'), '/tmp/dp_listen.sock')

# 接收二进制响应
data, _ = sock.recvfrom(8192)
sock.close()
```

#### Step 2: 解析二进制响应

```python
import struct

# 解析消息头
hdr_format = '!HHI'  # Kind(2) + Length(2) + More(4)
hdr_size = struct.calcsize(hdr_format)
kind, length, more = struct.unpack(hdr_format, data[:hdr_size])

# 解析统计数据
stats_data = data[hdr_size:]

# 假设 DPMsgStats 结构（简化版）
cur_sess, total_sess, total_pkts, total_bytes = struct.unpack(
    '!IQqq', stats_data[:28]
)

# 解析环形缓冲区（60 个槽）
sess_ring = struct.unpack('!60I', stats_data[28:28+240])
pkt_ring = struct.unpack('!60I', stats_data[268:268+240])
byte_ring = struct.unpack('!60I', stats_data[508:508+240])

print(f"当前会话数: {cur_sess}")
print(f"总会话数: {total_sess}")
print(f"总数据包: {total_pkts}")
print(f"总字节数: {total_bytes}")
```

#### Step 3: 绘制流量趋势图

```python
import matplotlib.pyplot as plt
import numpy as np

# 时间轴（每个槽 5 秒，共 300 秒）
time_slots = np.arange(60) * 5  # 0, 5, 10, ..., 295

# 绘制会话数趋势
plt.figure(figsize=(12, 4))
plt.subplot(1, 3, 1)
plt.plot(time_slots, sess_ring)
plt.title('Sessions per 5s')
plt.xlabel('Time (s)')
plt.ylabel('Sessions')

# 绘制数据包趋势
plt.subplot(1, 3, 2)
plt.plot(time_slots, pkt_ring)
plt.title('Packets per 5s')
plt.xlabel('Time (s)')
plt.ylabel('Packets')

# 绘制字节数趋势
plt.subplot(1, 3, 3)
plt.plot(time_slots, byte_ring / 1024 / 1024)  # 转换为 MB
plt.title('Traffic per 5s')
plt.xlabel('Time (s)')
plt.ylabel('MB')

plt.tight_layout()
plt.show()
```

### 8.3 动态调整日志级别

**场景**：dp 运行时遇到问题，需要临时开启 packet 和 session 调试日志。

```python
import socket
import json

sock = socket.socket(socket.AF_UNIX, socket.SOCK_DGRAM)

# 开启调试日志
debug_msg = json.dumps({
    "ctrl_set_debug": {
        "categories": ["ctrl", "packet", "session", "policy"]
    }
})

sock.sendto(debug_msg.encode('utf-8'), '/tmp/dp_listen.sock')
print("Debug logging enabled")

# ... 复现问题 ...

# 关闭调试日志（只保留 error）
debug_msg = json.dumps({
    "ctrl_set_debug": {
        "categories": ["error"]
    }
})

sock.sendto(debug_msg.encode('utf-8'), '/tmp/dp_listen.sock')
print("Debug logging disabled")

sock.close()
```

### 8.4 FQDN 策略实战

**场景**：允许访问 `api.github.com`，但 GitHub 使用多个 IP 地址。

#### Step 1: DNS 解析

```python
import socket

# 解析域名
ips = socket.getaddrinfo('api.github.com', None)
github_ips = list(set([ip[4][0] for ip in ips]))

print(f"api.github.com -> {github_ips}")
# 输出: ['140.82.113.5', '140.82.114.5', ...]
```

#### Step 2: 下发 FQDN 映射

```python
import socket
import json

sock = socket.socket(socket.AF_UNIX, socket.SOCK_DGRAM)

# 1. 下发 FQDN 映射
fqdn_msg = json.dumps({
    "ctrl_cfg_set_fqdn": {
        "fqdn_name": "api.github.com",
        "vhost": False,
        "fqdn_ips": github_ips
    }
})

sock.sendto(fqdn_msg.encode('utf-8'), '/tmp/dp_listen.sock')

# 2. 下发策略规则（允许访问 api.github.com）
policy_msg = json.dumps({
    "ctrl_cfg_policy": {
        "cmd": 1,
        "flag": 0,
        "defact": 2,  # 默认拒绝
        "dir": 3,
        "mac": ["aa:bb:cc:dd:ee:ff"],
        "rules": [
            {
                "id": 2001,
                "sip": "0.0.0.0",
                "sipr": "255.255.255.255",
                "dip": "0.0.0.0",
                "dipr": "0.0.0.0",
                "port": 443,
                "portr": 443,
                "proto": 6,
                "action": 1,  # 允许
                "ingress": False,
                "vhost": False,
                "fqdn": "api.github.com",  # FQDN 匹配
                "apps": []
            }
        ]
    }
})

sock.sendto(policy_msg.encode('utf-8'), '/tmp/dp_listen.sock')
print("FQDN policy configured")

sock.close()
```

#### Step 3: 数据包匹配流程

```
数据包到达（dst=140.82.113.5:443）
    ↓
dpi_recv_packet()
    ↓
查找 FQDN 映射：140.82.113.5 -> "api.github.com"
    ↓
策略匹配：规则 2001
    ├─ fqdn = "api.github.com" ✅
    ├─ dport = 443 ✅
    └─ proto = TCP ✅
    ↓
action = 1 (允许)
    ↓
TC_ACT_OK
```

---

## 附录

### A. JSON 消息格式速查表

```json
{
  "命令名称": {
    "参数1": "值",
    "参数2": 123,
    "参数3": true,
    "参数4": ["数组元素1", "数组元素2"],
    "参数5": {
      "嵌套对象": "值"
    }
  }
}
```

### B. 常见数据类型转换

| C 类型 | JSON 类型 | libjansson 函数 |
|--------|-----------|-----------------|
| `const char*` | string | `json_string_value()` |
| `int` / `uint32_t` | number | `json_integer_value()` |
| `bool` | boolean | `json_boolean_value()` |
| `array` | array | `json_array_size()` + `json_array_get()` |
| `struct` | object | `json_object_get()` |

### C. 调试技巧

#### 1. 监听 Unix Socket 通信

```bash
# 使用 socat 监听
socat -v UNIX-LISTEN:/tmp/dp_listen_debug.sock,fork \
         UNIX-CONNECT:/tmp/dp_listen.sock
```

#### 2. 查看 JSON 格式

```bash
# 格式化 JSON 输出
echo '{"ctrl_stats_device":{}}' | python -m json.tool
```

#### 3. 测试发送消息

```bash
# 使用 nc (netcat) 发送
echo '{"ctrl_keep_alive":{"seq_num":1}}' | \
  nc -U /tmp/dp_listen.sock
```

### D. 参考资料

- [libjansson 官方文档](https://jansson.readthedocs.io/)
- [Unix Domain Socket 编程指南](https://man7.org/linux/man-pages/man7/unix.7.html)
- [NeuVector 开源仓库](https://github.com/neuvector/neuvector)

---

**文档版本**：1.0
**最后更新**：2025-10-29
**适用项目**：eBPF 微隔离项目参考
