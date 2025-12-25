# NeuVector Agent 与 dp 通信与策略下发详解

## 目录

- [一、Agent 与 dp 通信架构](#一agent-与-dp-通信架构)
- [二、通信建立流程](#二通信建立流程)
- [三、策略下发完整流程](#三策略下发完整流程)
- [四、策略数据结构详解](#四策略数据结构详解)
- [五、策略类型与命令](#五策略类型与命令)
- [六、双向通信机制](#六双向通信机制)
- [七、完整实战示例](#七完整实战示例)
- [八、故障处理与重连机制](#八故障处理与重连机制)

---

## 一、Agent 与 dp 通信架构

### 1.1 整体架构图

```
┌──────────────────────────────────────────────────────────────┐
│                     NeuVector 架构                            │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐          ┌─────────────┐                  │
│  │  Controller │          │   Agent     │                  │
│  │  (集中管理)  │          │  (Go 语言)   │                  │
│  └─────────────┘          └─────────────┘                  │
│         │                        │                          │
│         │ gRPC/REST API          │ Unix Socket             │
│         │                        │ (JSON)                  │
│         ▼                        ▼                          │
│  ┌─────────────┐          ┌─────────────┐                  │
│  │   etcd/     │          │      dp     │                  │
│  │  Consul     │          │   (C 语言)   │                  │
│  │ (配置存储)   │          │ (数据平面)   │                  │
│  └─────────────┘          └─────────────┘                  │
│                                  │                          │
│                                  │ eBPF/Netfilter          │
│                                  ▼                          │
│                           ┌─────────────┐                  │
│                           │    Kernel    │                  │
│                           │  (数据包处理) │                  │
│                           └─────────────┘                  │
└──────────────────────────────────────────────────────────────┘
```

### 1.2 Agent 与 dp 通信详图

```
┌────────────────────────────────┐
│         Agent (Go)             │
│                                │
│  ┌──────────────────────────┐ │
│  │  Policy Module           │ │
│  │  - 策略计算              │ │
│  │  - 规则生成              │ │
│  └─────────┬────────────────┘ │
│            │                   │
│  ┌─────────▼────────────────┐ │
│  │  dp Package (agent/dp/)  │ │
│  │  - JSON 序列化           │ │
│  │  - Unix Socket 通信      │ │
│  └─────────┬────────────────┘ │
│            │                   │
└────────────┼────────────────────┘
             │
             │ /tmp/dp_listen.sock
             │ (JSON over UnixGram)
             ▼
┌────────────────────────────────┐
│         dp (C)                 │
│                                │
│  ┌──────────────────────────┐ │
│  │  dp_ctrl_loop()          │ │
│  │  - select() 监听          │ │
│  │  - JSON 解析             │ │
│  └─────────┬────────────────┘ │
│            │                   │
│  ┌─────────▼────────────────┐ │
│  │  dp_ctrl_handler()       │ │
│  │  - 消息分发              │ │
│  │  - 调用处理函数          │ │
│  └─────────┬────────────────┘ │
│            │                   │
│  ┌─────────▼────────────────┐ │
│  │  dpi_policy_cfg()        │ │
│  │  - 策略编译              │ │
│  │  - 策略执行引擎          │ │
│  └──────────────────────────┘ │
└────────────────────────────────┘
             │
             ▼
      eBPF/Netfilter
      (数据包过滤)
```

### 1.3 通信特点

| 特性 | 说明 |
|------|------|
| **通信协议** | Unix Domain Socket (SOCK_DGRAM) |
| **数据格式** | JSON (agent → dp 下发命令) <br> Binary (dp → agent 上报数据) |
| **Socket 路径** | `/tmp/dp_listen.sock` (dp 监听) <br> `/tmp/ctrl_listen.sock` (agent 监听) |
| **编程语言** | Agent: Go <br> dp: C |
| **序列化库** | Agent: encoding/json <br> dp: jansson |
| **双向通信** | 支持 (命令/响应 + 主动上报) |
| **心跳保活** | 2 秒间隔 |

---

## 二、通信建立流程

### 2.1 dp 启动监听

**代码位置**：dp/ctrl.c:3018-3079

```c
void dp_ctrl_loop(void)
{
    // 1. 删除旧 socket 文件
    unlink(DP_SERVER_SOCK);  // /tmp/dp_listen.sock

    // 2. 创建 Unix Domain Socket
    g_ctrl_fd = make_named_socket(DP_SERVER_SOCK);
    g_ctrl_notify_fd = make_notify_client(CTRL_NOTIFY_SOCK);

    // 3. 初始化同步原语
    pthread_mutex_init(&g_ctrl_req_lock, NULL);
    pthread_cond_init(&g_ctrl_req_cond, NULL);

    // 4. 主循环
    while (g_running) {
        FD_ZERO(&read_fds);
        FD_SET(g_ctrl_fd, &read_fds);

        // 5. 等待消息（2秒超时）
        ret = select(g_ctrl_fd + 1, &read_fds, NULL, NULL, &timeout);

        // 6. 处理消息
        if (ret > 0 && FD_ISSET(g_ctrl_fd, &read_fds)) {
            dp_ctrl_handler(g_ctrl_fd);
        }

        // 7. 定期任务（每 2 秒）
        dp_ctrl_update_app(false);
        dp_ctrl_update_fqdn_ip();
        dp_ctrl_consume_threat_log();
    }

    // 8. 清理
    close(g_ctrl_fd);
    unlink(DP_SERVER_SOCK);
}
```

### 2.2 Agent 连接 dp

**代码位置**：agent/dp/ctrl.go:24-26, dp.go:382-404

#### Agent 监听 dp 上报

```go
func listenDP() {
    log.Debug("Listening to CTRL socket ...")

    // 1. 删除旧 socket 文件
    os.Remove(ctrlServer)  // /tmp/ctrl_listen.sock

    // 2. 创建监听 socket
    kind := "unixgram"
    addr := net.UnixAddr{Name: ctrlServer, Net: kind}
    defer os.Remove(ctrlServer)

    conn, _ = net.ListenUnixgram(kind, &addr)
    defer conn.Close()

    // 3. 循环接收消息
    for {
        var buf [C.DP_MSG_SIZE]byte
        n, err := conn.Read(buf[:])
        if err != nil {
            log.WithFields(log.Fields{"error": err}).Error("Read message error.")
        } else {
            dpAliveMsgCnt++
            dpMessenger(buf[:n])  // 处理 dp 上报的消息
        }
    }
}
```

#### Agent 发送消息到 dp

**代码位置**：agent/dp/ctrl.go:42-119

```go
// 全局变量
const DPServer string = "/tmp/dp_listen.sock"
var dpConn *net.UnixConn
var dpClientMutex sync.Mutex

// 发送消息函数
func dpSendMsgEx(msg []byte, timeout int, cb DPCallback, param interface{}) int {
    dpClientLock()
    defer dpClientUnlock()

    // 1. 检查连接
    if dpConn == nil {
        log.Error("Data path not connected")
        return -1
    }

    // 2. 设置写超时
    dpConn.SetWriteDeadline(time.Now().Add(time.Second * 2))

    // 3. 发送 JSON 消息
    _, err := dpConn.Write(msg)
    if err != nil {
        log.WithFields(log.Fields{"error": err}).Error("Send error")
        return -1
    }

    // 4. 等待响应（如果有回调）
    if cb != nil && param != nil {
        var buf []byte = make([]byte, C.DP_MSG_SIZE)

        dpConn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(timeout)))
        n, err := dpConn.Read(buf)
        if err != nil {
            log.WithFields(log.Fields{"error": err}).Error("Read error")
            cb(nil, param)
            return -1
        } else {
            cb(buf[:n], param)
        }
    }

    dpAliveMsgCnt++
    return 0
}

// 简化版发送（无回调）
func dpSendMsg(msg []byte) int {
    return dpSendMsgEx(msg, 0, nil, nil)
}
```

### 2.3 连接建立流程图

```
Agent 启动
    │
    ├─> 创建 dp 连接
    │    ├─ net.DialUnix("unixgram", ..., "/tmp/dp_listen.sock")
    │    └─ 保存到 dpConn
    │
    ├─> 启动监听线程 listenDP()
    │    └─ 监听 /tmp/ctrl_listen.sock
    │
    └─> 启动心跳线程
         └─ 每 2 秒发送 ctrl_keep_alive

dp 启动
    │
    ├─> 创建监听 socket
    │    └─ bind("/tmp/dp_listen.sock")
    │
    ├─> 进入主循环 dp_ctrl_loop()
    │    ├─ select() 等待消息
    │    └─ 处理 agent 命令
    │
    └─> 定期任务
         ├─ 更新应用识别
         ├─ 更新 FQDN 映射
         └─ 上报威胁日志
```

---

## 三、策略下发完整流程

### 3.1 策略下发全流程图

```
┌────────────────────────────────────────────────────────────┐
│                    Controller                              │
│  (用户通过 UI/API 配置策略)                                  │
└──────────────┬─────────────────────────────────────────────┘
               │ gRPC/REST API
               │
               ▼
┌────────────────────────────────────────────────────────────┐
│                     Agent                                  │
│                                                            │
│  1. 接收 Controller 策略                                    │
│     ├─ gRPC watch etcd 变更                                │
│     └─ 解析 share.CLUSPolicyIPRule                         │
│                                                            │
│  2. 策略计算与转换 (policy/network.go)                     │
│     ├─ calculateIPPolicyFunc()                             │
│     ├─ 处理 from/to 地址 (IP/FQDN/Workload)               │
│     ├─ 处理端口范围                                        │
│     ├─ 处理应用层规则                                      │
│     └─ 生成 DPWorkloadIPPolicy                            │
│                                                            │
│  3. JSON 序列化 (dp/ctrl.go)                               │
│     ├─ DPCtrlConfigPolicy()                                │
│     ├─ 分批处理（每批 40 条规则）                           │
│     ├─ json.Marshal(DPPolicyCfgReq)                       │
│     └─ 检查消息大小（最大 256KB）                          │
│                                                            │
│  4. 发送到 dp (dp/ctrl.go)                                 │
│     ├─ dpSendMsg(msg)                                      │
│     └─ Unix Socket Write                                  │
└──────────────┬─────────────────────────────────────────────┘
               │ JSON over Unix Socket
               │ /tmp/dp_listen.sock
               ▼
┌────────────────────────────────────────────────────────────┐
│                      dp (C)                                │
│                                                            │
│  5. 接收 JSON (ctrl.c)                                     │
│     ├─ select() 检测到消息                                 │
│     ├─ recvfrom(g_ctrl_fd, ...)                           │
│     └─ dp_ctrl_handler()                                   │
│                                                            │
│  6. 解析 JSON (ctrl.c)                                     │
│     ├─ json_loads(ctrl_msg_buf, ...)                      │
│     ├─ json_object_foreach(root, key, msg)                │
│     └─ 识别 "ctrl_cfg_policy"                             │
│                                                            │
│  7. 提取策略数据 (ctrl.c:dp_ctrl_cfg_policy)               │
│     ├─ 解析 cmd, flag, defact, dir                        │
│     ├─ 解析 MAC 列表                                       │
│     ├─ 解析规则数组                                        │
│     │   ├─ id, sip, dip, port, proto                     │
│     │   ├─ action, ingress                               │
│     │   ├─ FQDN (可选)                                   │
│     │   └─ apps[] (应用层规则)                           │
│     └─ 构造 dpi_policy_t 结构                              │
│                                                            │
│  8. 策略编译与安装 (dpi/dpi_policy.c)                      │
│     ├─ dpi_policy_cfg(cmd, &policy, flag)                 │
│     ├─ 编译规则树                                          │
│     ├─ 插入策略引擎                                        │
│     └─ 关联到 Endpoint (MAC)                              │
└──────────────┬─────────────────────────────────────────────┘
               │
               ▼
┌────────────────────────────────────────────────────────────┐
│                  策略执行引擎                               │
│                                                            │
│  9. 数据包匹配 (dpi/dpi_packet.c)                          │
│     ├─ dpi_recv_packet()                                   │
│     ├─ 提取五元组                                          │
│     ├─ 查找策略规则                                        │
│     ├─ 匹配规则条件                                        │
│     │   ├─ IP 范围匹配                                    │
│     │   ├─ 端口范围匹配                                   │
│     │   ├─ FQDN 匹配 (如果有)                            │
│     │   └─ 应用层匹配 (DPI)                              │
│     └─ 执行动作                                            │
│         ├─ DP_POLICY_ACTION_OPEN -> TC_ACT_OK             │
│         ├─ DP_POLICY_ACTION_DENY -> TC_ACT_SHOT           │
│         └─ DP_POLICY_ACTION_LEARN -> 学习模式             │
└────────────────────────────────────────────────────────────┘
```

### 3.2 策略下发核心函数调用链

#### Agent 侧

```
policy.calculateIPPolicyFunc()
    │
    ├─> 遍历所有策略规则
    │    └─> 处理 from/to workload
    │         ├─ isWorkloadFqdn() -> 解析 FQDN
    │         ├─ isWorkloadIP() -> 提取 IP
    │         └─ 查找 workload MAC
    │
    ├─> createIPRule()
    │    ├─ 解析端口和协议
    │    ├─ 解析应用层规则
    │    └─> 填充 DPPolicyIPRule 结构
    │
    └─> dp.DPCtrlConfigPolicy(&policy, cmd)
         │
         ├─> 分批处理（40 条/批）
         │    ├─ 第一批：flag = MSG_START
         │    ├─ 中间批：flag = 0
         │    └─ 最后批：flag = MSG_END
         │
         ├─> json.Marshal(DPPolicyCfgReq{...})
         │
         └─> dpSendMsg(msg)
```

#### dp 侧

```
dp_ctrl_loop()
    │
    ├─> select() 等待消息
    │
    └─> dp_ctrl_handler(fd)
         │
         ├─> recvfrom() 接收 JSON
         │
         ├─> json_loads() 解析
         │
         ├─> json_object_foreach()
         │    └─ key == "ctrl_cfg_policy"
         │
         └─> dp_ctrl_cfg_policy(msg)
              │
              ├─> 解析 JSON 字段
              │    ├─ cmd = json_integer_value(json_object_get(msg, "cmd"))
              │    ├─ defact = json_integer_value(...)
              │    ├─ mac[] = json_array_get(...)
              │    └─ rules[] = json_array_get(...)
              │
              ├─> 构造 dpi_policy_t
              │    ├─ policy.num_macs
              │    ├─ policy.mac_list[]
              │    ├─ policy.def_action
              │    ├─ policy.apply_dir
              │    ├─ policy.num_rules
              │    └─ policy.rule_list[]
              │         ├─ rule.id
              │         ├─ rule.sip/dip (inet_addr)
              │         ├─ rule.dport/proto
              │         ├─ rule.action
              │         ├─ rule.fqdn
              │         └─ rule.app_rules[]
              │
              └─> dpi_policy_cfg(cmd, &policy, flag)
                   │
                   ├─> 编译策略规则树
                   ├─> 插入到全局策略引擎
                   └─> 关联到 io_ep_t (Endpoint)
```

---

## 四、策略数据结构详解

### 4.1 Agent 侧数据结构 (Go)

#### DPWorkloadIPPolicy - 工作负载策略

**文件位置**：agent/dp/dp_apis.go:254-261

```go
type DPWorkloadIPPolicy struct {
    WlID        string            `json:"wl_id"`      // Workload ID
    Mode        string            `json:"mode"`       // 策略模式（protect/monitor/discover）
    DefAction   uint8             `json:"defact"`     // 默认动作（1=允许, 2=拒绝）
    ApplyDir    int               `json:"apply_dir"`  // 应用方向（1=入站, 2=出站, 3=双向）
    WorkloadMac []string          `json:"mac"`        // Workload MAC 地址列表
    IPRules     []*DPPolicyIPRule `json:"policy_rules"` // IP 策略规则列表
}
```

#### DPPolicyIPRule - IP 策略规则

**文件位置**：agent/dp/dp_apis.go:238-252

```go
type DPPolicyIPRule struct {
    ID      uint32         `json:"id"`      // 规则 ID
    SrcIP   net.IP         `json:"sip"`     // 源 IP 起始
    DstIP   net.IP         `json:"dip"`     // 目标 IP 起始
    SrcIPR  net.IP         `json:"sipr,omitempty"`  // 源 IP 结束（范围）
    DstIPR  net.IP         `json:"dipr,omitempty"`  // 目标 IP 结束（范围）
    Port    uint16         `json:"port"`    // 目标端口起始
    PortR   uint16         `json:"portr"`   // 目标端口结束
    IPProto uint8          `json:"proto"`   // 协议（6=TCP, 17=UDP, 1=ICMP）
    Action  uint8          `json:"action"`  // 动作（1=允许, 2=拒绝, 3=学习）
    Ingress bool           `json:"ingress"` // 是否入站规则
    Fqdn    string         `json:"fqdn,omitempty"` // 域名匹配（可选）
    Vhost   bool           `json:"vhost,omitempty"` // 虚拟主机模式
    Apps    []*DPPolicyApp `json:"apps,omitempty"` // 应用层规则（可选）
}
```

#### DPPolicyApp - 应用层策略

**文件位置**：agent/dp/dp_apis.go:232-236

```go
type DPPolicyApp struct {
    App    uint32 `json:"app"`    // 应用协议 ID（HTTP=100, MySQL=200...）
    Action uint8  `json:"action"` // 动作（1=允许, 2=拒绝）
    RuleID uint32 `json:"rid"`    // 关联的规则 ID
}
```

#### DPPolicyCfgReq - JSON 请求封装

**文件位置**：agent/dp/dp_apis.go:272-274

```go
type DPPolicyCfgReq struct {
    DPPolicyCfg *DPPolicyCfg `json:"ctrl_cfg_policy"`
}

type DPPolicyCfg struct {
    Cmd         uint              `json:"cmd"`    // 命令类型（1=添加, 2=删除, 3=修改）
    Flag        uint              `json:"flag"`   // 标志（MSG_START/MSG_END）
    DefAction   uint8             `json:"defact"` // 默认动作
    ApplyDir    int               `json:"dir"`    // 应用方向
    WorkloadMac []string          `json:"mac"`    // MAC 列表
    IPRules     []*DPPolicyIPRule `json:"rules"`  // 规则列表
}
```

### 4.2 dp 侧数据结构 (C)

#### dpi_policy_t - dp 策略结构

**文件位置**：dp/apis.h:272-279

```c
typedef struct dpi_policy_ {
    int num_macs;                    // MAC 地址数量
    struct ether_addr *mac_list;     // MAC 地址列表
    int def_action;                  // 默认动作
    int apply_dir;                   // 应用方向（1=入站, 2=出站, 3=双向）
    int num_rules;                   // 规则数量
    dpi_policy_rule_t *rule_list;    // 规则列表
} dpi_policy_t;
```

#### dpi_policy_rule_t - dp 策略规则

**文件位置**：dp/apis.h:255-270

```c
typedef struct dpi_policy_rule_ {
    uint32_t id;                     // 规则 ID
    uint32_t sip;                    // 源 IP 起始
    uint32_t sip_r;                  // 源 IP 结束
    uint32_t dip;                    // 目标 IP 起始
    uint32_t dip_r;                  // 目标 IP 结束
    uint16_t dport;                  // 目标端口起始
    uint16_t dport_r;                // 目标端口结束
    uint16_t proto;                  // 协议
    uint8_t action;                  // 动作
    bool ingress;                    // 入站/出站
    bool vh;                         // 虚拟主机模式
    char fqdn[MAX_FQDN_LEN];         // FQDN（域名）
    uint32_t num_apps;               // 应用规则数量
    dpi_policy_app_rule_t *app_rules; // 应用规则列表
} dpi_policy_rule_t;
```

#### dpi_policy_app_rule_t - 应用层规则

**文件位置**：dp/apis.h:248-252

```c
typedef struct dpi_policy_app_rule_ {
    uint32_t rule_id;                // 关联的规则 ID
    uint32_t app;                    // 应用协议 ID
    uint8_t action;                  // 动作
} dpi_policy_app_rule_t;
```

### 4.3 数据结构转换流程

```
Controller 策略 (share.CLUSPolicyIPRule)
    │
    ├─ from: "nv.ip.192.168.1.10"
    ├─ to: "nv.fqdn.api.github.com"
    ├─ ports: "tcp/443"
    ├─ action: "allow"
    └─ applications: ["HTTP"]
    │
    ▼ Agent 策略计算
    │
Agent 策略 (DPWorkloadIPPolicy)
    │
    ├─ WorkloadMac: ["aa:bb:cc:dd:ee:ff"]
    ├─ DefAction: 2 (拒绝)
    ├─ ApplyDir: 3 (双向)
    └─ IPRules: [
         {
           ID: 1001,
           SrcIP: "192.168.1.10",
           DstIP: "0.0.0.0",  // FQDN 匹配不限制 IP
           Port: 443,
           PortR: 443,
           IPProto: 6,  // TCP
           Action: 1,   // 允许
           Ingress: false,
           Fqdn: "api.github.com",
           Apps: [{App: 100, Action: 1, RuleID: 1001}]  // HTTP
         }
       ]
    │
    ▼ JSON 序列化
    │
JSON 消息
{
  "ctrl_cfg_policy": {
    "cmd": 1,
    "flag": 3,  // MSG_START | MSG_END
    "defact": 2,
    "dir": 3,
    "mac": ["aa:bb:cc:dd:ee:ff"],
    "rules": [
      {
        "id": 1001,
        "sip": "192.168.1.10",
        "dip": "0.0.0.0",
        "sipr": "192.168.1.10",
        "dipr": "0.0.0.0",
        "port": 443,
        "portr": 443,
        "proto": 6,
        "action": 1,
        "ingress": false,
        "fqdn": "api.github.com",
        "vhost": false,
        "apps": [
          {"rid": 1001, "app": 100, "action": 1}
        ]
      }
    ]
  }
}
    │
    ▼ dp JSON 解析
    │
dp 策略 (dpi_policy_t)
    │
    ├─ num_macs: 1
    ├─ mac_list: [0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff]
    ├─ def_action: 2
    ├─ apply_dir: 3
    ├─ num_rules: 1
    └─ rule_list: [
         {
           id: 1001,
           sip: 0xC0A8010A,     // 192.168.1.10 (网络字节序)
           dip: 0x00000000,
           sip_r: 0xC0A8010A,
           dip_r: 0x00000000,
           dport: 443,
           dport_r: 443,
           proto: 6,
           action: 1,
           ingress: false,
           vh: false,
           fqdn: "api.github.com",
           num_apps: 1,
           app_rules: [
             {rule_id: 1001, app: 100, action: 1}
           ]
         }
       ]
```

---

## 五、策略类型与命令

### 5.1 策略命令类型

| cmd 值 | 命令 | 说明 |
|--------|------|------|
| 1 | `DP_POLICY_CMD_ADD` | 添加策略 |
| 2 | `DP_POLICY_CMD_DEL` | 删除策略 |
| 3 | `DP_POLICY_CMD_MOD` | 修改策略 |

### 5.2 策略标志 (Flag)

| flag 值 | 标志 | 说明 |
|---------|------|------|
| 0x01 | `MSG_START` | 消息起始（第一批规则） |
| 0x02 | `MSG_END` | 消息结束（最后一批规则） |
| 0x03 | `MSG_START \| MSG_END` | 单批消息（所有规则） |

**分批发送原因**：
- 单次消息最大 256KB
- 每批默认 40 条规则
- 如果超过大小限制，动态调整每批数量

### 5.3 策略动作 (Action)

| action 值 | 动作 | 说明 |
|-----------|------|------|
| 1 | `DP_POLICY_ACTION_OPEN` | 允许通过 |
| 2 | `DP_POLICY_ACTION_DENY` | 拒绝（丢弃） |
| 3 | `DP_POLICY_ACTION_LEARN` | 学习模式（记录但放行） |
| 4 | `DP_POLICY_ACTION_VIOLATE` | 违规（特殊标记） |

### 5.4 应用方向 (ApplyDir)

| dir 值 | 方向 | 说明 |
|--------|------|------|
| 1 | `DP_POLICY_APPLY_INGRESS` | 仅入站 |
| 2 | `DP_POLICY_APPLY_EGRESS` | 仅出站 |
| 3 | `DP_POLICY_APPLY_BOTH` | 双向（默认） |

### 5.5 协议类型 (IPProto)

| proto 值 | 协议 | 说明 |
|----------|------|------|
| 0 | `ANY` | 任意协议 |
| 1 | `IPPROTO_ICMP` | ICMP |
| 6 | `IPPROTO_TCP` | TCP |
| 17 | `IPPROTO_UDP` | UDP |
| 58 | `IPPROTO_ICMPV6` | ICMPv6 |

### 5.6 应用协议 ID (App)

| app 值 | 应用 | 说明 |
|--------|------|------|
| 0 | `DP_POLICY_APP_UNKNOWN` | 未知应用 |
| 100 | `HTTP` | HTTP 协议 |
| 101 | `HTTPS/SSL` | HTTPS/SSL |
| 200 | `MySQL` | MySQL 数据库 |
| 201 | `PostgreSQL` | PostgreSQL 数据库 |
| 202 | `MongoDB` | MongoDB 数据库 |
| 300 | `Redis` | Redis |
| 301 | `Kafka` | Kafka |
| 302 | `Zookeeper` | Zookeeper |
| 400 | `DNS` | DNS |
| 500 | `SSH` | SSH |
| 501 | `Telnet` | Telnet |

**注意**：具体 ID 定义在 `defs.h` 中的 `DPI_APP_*` 常量。

---

## 六、双向通信机制

### 6.1 Agent → dp (命令下发)

#### 支持的命令类型

| 命令 | JSON Key | 功能 | 数据结构 |
|------|----------|------|----------|
| 配置策略 | `ctrl_cfg_policy` | 下发访问控制策略 | `DPPolicyCfgReq` |
| 添加 MAC | `ctrl_add_mac` | 注册容器/VM MAC | `DPAddMACReq` |
| 删除 MAC | `ctrl_del_mac` | 注销 MAC | `DPDelMACReq` |
| 配置 MAC | `ctrl_cfg_mac` | 更新 MAC 配置（应用端口） | `DPConfigMACReq` |
| 设置 FQDN | `ctrl_cfg_set_fqdn` | 配置域名映射 | `DPFqdnIpSetReq` |
| 删除 FQDN | `ctrl_cfg_del_fqdn` | 删除域名映射 | `DPFqdnDeleteReq` |
| 添加 TAP 端口 | `ctrl_add_tap_port` | 添加 TAP 设备 | `DPAddTapPortReq` |
| 添加 NFQ 端口 | `ctrl_add_nfq_port` | 添加 Netfilter Queue | `DPAddNfqPortReq` |
| 配置内网 | `ctrl_cfg_internal_net` | 配置内网子网 | `DPInternalSubnetCfgReq` |
| DLP 规则 | `ctrl_bld_dlp` | 构建 DLP 规则 | `DPDlpBldReq` |
| WAF 规则 | `ctrl_cfg_dlp` | 配置 WAF 规则 | `DPDlpCfgReq` |
| 系统配置 | `ctrl_sys_conf` | 系统级配置 | `DPSysConfReq` |
| 心跳保活 | `ctrl_keep_alive` | 维持连接 | `DPKeepAliveReq` |

#### 命令发送示例

**配置策略**：

```go
func DPCtrlConfigPolicy(policy *DPWorkloadIPPolicy, cmd uint) int {
    data := DPPolicyCfgReq{
        DPPolicyCfg: &DPPolicyCfg{
            Cmd:         cmd,
            Flag:        flag,
            DefAction:   policy.DefAction,
            ApplyDir:    policy.ApplyDir,
            WorkloadMac: policy.WorkloadMac,
            IPRules:     policy.IPRules[start:end],
        },
    }
    msg, _ := json.Marshal(data)
    return dpSendMsg(msg)
}
```

**添加 MAC**：

```go
func DPCtrlAddMAC(iface, mac, ucmac, bcmac string, pmac net.HardwareAddr,
                  pips []net.IP) {
    pipList := make([]DPMacPip, len(pips))
    for i, pip := range pips {
        pipList[i] = DPMacPip{IP: pip}
    }

    data := DPAddMACReq{
        AddMAC: &DPAddMAC{
            Iface:  iface,
            MAC:    mac,
            UCMAC:  ucmac,
            BCMAC:  bcmac,
            PMAC:   pmac.String(),
            PIPS:   pipList,
        },
    }
    msg, _ := json.Marshal(data)
    dpSendMsg(msg)
}
```

### 6.2 dp → Agent (数据上报)

#### 上报的消息类型

| 消息类型 | Kind 值 | 功能 | 处理函数 |
|----------|---------|------|----------|
| 应用更新 | `DP_KIND_APP_UPDATE` | 上报识别的应用端口 | `dpMsgAppUpdate()` |
| 威胁日志 | `DP_KIND_THREAT_LOG` | 上报安全威胁事件 | `dpMsgThreatLog()` |
| 连接信息 | `DP_KIND_CONNECTION` | 上报网络连接统计 | `dpMsgConnection()` |
| FQDN 更新 | `DP_KIND_FQDN_UPDATE` | 上报 FQDN 到 IP 映射 | `dpMsgFqdnIpUpdate()` |
| IP-FQDN 存储 | `DP_KIND_IP_FQDN_STORAGE_UPDATE` | IP 到 FQDN 反向映射 | `dpMsgIpFqdnStorageUpdate()` |

#### 消息格式（二进制）

**消息头结构** (C):

```c
typedef struct {
    uint16_t Kind;       // 消息类型
    uint16_t Length;     // 消息长度（包括头部）
    uint32_t More;       // 是否有后续消息
    uint32_t SeqNum;     // 序列号（可选）
} DPMsgHdr;
```

#### 威胁日志上报示例

**dp 侧发送** (C):

```c
void dp_send_threat_log(DPMsgThreatLog *log) {
    uint8_t buf[sizeof(DPMsgHdr) + sizeof(DPMsgThreatLog)];

    // 填充消息头
    DPMsgHdr *hdr = (DPMsgHdr *)buf;
    hdr->Kind = htons(DP_KIND_THREAT_LOG);
    hdr->Length = htons(sizeof(buf));

    // 填充威胁日志
    DPMsgThreatLog *tlog = (DPMsgThreatLog *)(hdr + 1);
    memcpy(tlog, log, sizeof(DPMsgThreatLog));

    // 发送到 agent
    dp_ctrl_notify_ctrl(buf, sizeof(buf));
}
```

**Agent 侧接收** (Go):

**文件位置**：agent/dp/dp.go:71-128

```go
func dpMsgThreatLog(msg []byte) {
    var tlog C.DPMsgThreatLog

    r := bytes.NewReader(msg)
    binary.Read(r, binary.BigEndian, &tlog)

    jlog := share.CLUSThreatLog{
        ID:          utils.GetTimeUUID(time.Now().UTC()),
        ThreatID:    uint32(tlog.ThreatID),
        Count:       uint32(tlog.Count),
        Action:      uint8(tlog.Action),
        Severity:    uint8(tlog.Severity),
        EtherType:   uint16(tlog.EtherType),
        IPProto:     uint8(tlog.IPProto),
        Application: uint32(tlog.Application),
        CapLen:      uint16(tlog.CapLen),
    }

    // 提取 IP 地址
    switch jlog.EtherType {
    case syscall.ETH_P_IP:
        jlog.SrcIP = net.IP(C.GoBytes(unsafe.Pointer(&tlog.SrcIP[0]), 4))
        jlog.DstIP = net.IP(C.GoBytes(unsafe.Pointer(&tlog.DstIP[0]), 4))
    case syscall.ETH_P_IPV6:
        jlog.SrcIP = net.IP(C.GoBytes(unsafe.Pointer(&tlog.SrcIP[0]), 16))
        jlog.DstIP = net.IP(C.GoBytes(unsafe.Pointer(&tlog.DstIP[0]), 16))
    }

    // 回调到 agent 主逻辑
    task := DPTask{Task: DP_TASK_THREAT_LOG, SecLog: &jlog, MAC: EPMAC}
    taskCallback(&task)
}
```

---

## 七、完整实战示例

### 7.1 示例场景

**需求**：为容器 `container-1` 配置以下策略：
1. 允许访问 MySQL 服务器 (10.20.30.40:3306)
2. 允许访问 api.github.com (443/HTTPS)
3. 拒绝访问 Redis (10.20.30.50:6379)
4. 默认拒绝所有其他流量

### 7.2 Agent 侧代码

#### Step 1: 构造策略数据

```go
package main

import (
    "net"
    "github.com/neuvector/neuvector/agent/dp"
)

func configureContainerPolicy() {
    // 1. 定义策略
    policy := &dp.DPWorkloadIPPolicy{
        WlID:      "container-1",
        Mode:      "protect",
        DefAction: 2,  // 默认拒绝
        ApplyDir:  3,  // 双向
        WorkloadMac: []string{"aa:bb:cc:dd:ee:ff"},
        IPRules: []*dp.DPPolicyIPRule{
            // 规则 1: 允许访问 MySQL
            {
                ID:      1001,
                SrcIP:   net.ParseIP("0.0.0.0"),
                DstIP:   net.ParseIP("10.20.30.40"),
                SrcIPR:  net.ParseIP("255.255.255.255"),
                DstIPR:  net.ParseIP("10.20.30.40"),
                Port:    3306,
                PortR:   3306,
                IPProto: 6,  // TCP
                Action:  1,  // 允许
                Ingress: false,
            },
            // 规则 2: 允许访问 api.github.com
            {
                ID:      1002,
                SrcIP:   net.ParseIP("0.0.0.0"),
                DstIP:   net.ParseIP("0.0.0.0"),
                SrcIPR:  net.ParseIP("255.255.255.255"),
                DstIPR:  net.ParseIP("0.0.0.0"),
                Port:    443,
                PortR:   443,
                IPProto: 6,
                Action:  1,
                Ingress: false,
                Fqdn:    "api.github.com",
                Apps: []*dp.DPPolicyApp{
                    {App: 101, Action: 1, RuleID: 1002},  // HTTPS
                },
            },
            // 规则 3: 拒绝访问 Redis
            {
                ID:      1003,
                SrcIP:   net.ParseIP("0.0.0.0"),
                DstIP:   net.ParseIP("10.20.30.50"),
                SrcIPR:  net.ParseIP("255.255.255.255"),
                DstIPR:  net.ParseIP("10.20.30.50"),
                Port:    6379,
                PortR:   6379,
                IPProto: 6,
                Action:  2,  // 拒绝
                Ingress: false,
            },
        },
    }

    // 2. 下发策略
    if err := dp.DPCtrlConfigPolicy(policy, 1); err != 0 {
        log.Error("Failed to configure policy")
    } else {
        log.Info("Policy configured successfully")
    }
}
```

#### Step 2: 配置 FQDN 映射

```go
func configureFqdn() {
    // 解析 api.github.com
    ips, _ := net.LookupIP("api.github.com")

    fqdnIp := &share.CLUSFqdnIp{
        FqdnName: "api.github.com",
        FqdnIP:   ips,
        Vhost:    false,
    }

    // 下发 FQDN 映射
    dp.DPCtrlSetFqdnIp(fqdnIp)
}
```

### 7.3 生成的 JSON 消息

#### 策略配置消息

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
        "dip": "10.20.30.40",
        "sipr": "255.255.255.255",
        "dipr": "10.20.30.40",
        "port": 3306,
        "portr": 3306,
        "proto": 6,
        "action": 1,
        "ingress": false
      },
      {
        "id": 1002,
        "sip": "0.0.0.0",
        "dip": "0.0.0.0",
        "sipr": "255.255.255.255",
        "dipr": "0.0.0.0",
        "port": 443,
        "portr": 443,
        "proto": 6,
        "action": 1,
        "ingress": false,
        "fqdn": "api.github.com",
        "apps": [
          {"rid": 1002, "app": 101, "action": 1}
        ]
      },
      {
        "id": 1003,
        "sip": "0.0.0.0",
        "dip": "10.20.30.50",
        "sipr": "255.255.255.255",
        "dipr": "10.20.30.50",
        "port": 6379,
        "portr": 6379,
        "proto": 6,
        "action": 2,
        "ingress": false
      }
    ]
  }
}
```

#### FQDN 配置消息

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

### 7.4 dp 侧处理流程

```
1. 接收策略配置
   ├─> 解析 JSON: ctrl_cfg_policy
   ├─> 提取 3 条规则
   └─> 构造 dpi_policy_t

2. 编译策略规则树
   ├─> 规则 1001: 10.20.30.40:3306/TCP -> ALLOW
   ├─> 规则 1002: *:443/TCP (FQDN: api.github.com, App: HTTPS) -> ALLOW
   └─> 规则 1003: 10.20.30.50:6379/TCP -> DENY

3. 安装到策略引擎
   └─> 关联到 MAC: aa:bb:cc:dd:ee:ff

4. 接收 FQDN 配置
   ├─> api.github.com -> [140.82.113.5, 140.82.114.5]
   └─> 更新 FQDN 映射表
```

### 7.5 数据包匹配流程

#### 场景 1: 访问 MySQL

```
数据包: 192.168.1.100:52345 -> 10.20.30.40:3306 (TCP)
    ↓
1. 提取五元组
   ├─ src_ip: 192.168.1.100
   ├─ dst_ip: 10.20.30.40
   ├─ src_port: 52345
   ├─ dst_port: 3306
   └─ proto: TCP

2. 查找策略规则
   ├─ 匹配规则 1001
   │   ├─ dst_ip: 10.20.30.40 ✅
   │   ├─ dst_port: 3306 ✅
   │   └─ proto: TCP ✅
   └─ action: ALLOW (1)

3. 执行动作
   └─> TC_ACT_OK (放行)
```

#### 场景 2: 访问 api.github.com

```
数据包: 192.168.1.100:52346 -> 140.82.113.5:443 (TCP/HTTPS)
    ↓
1. 提取五元组
   ├─ dst_ip: 140.82.113.5
   ├─ dst_port: 443
   └─ proto: TCP

2. 查找 FQDN 映射
   └─> 140.82.113.5 -> "api.github.com"

3. 匹配规则 1002
   ├─ fqdn: "api.github.com" ✅
   ├─ dst_port: 443 ✅
   └─ proto: TCP ✅

4. DPI 检测
   └─> 识别为 HTTPS (app: 101)

5. 匹配应用层规则
   ├─ app: 101 (HTTPS) ✅
   └─ action: ALLOW (1)

6. 执行动作
   └─> TC_ACT_OK (放行)
```

#### 场景 3: 访问 Redis

```
数据包: 192.168.1.100:52347 -> 10.20.30.50:6379 (TCP)
    ↓
1. 提取五元组
   ├─ dst_ip: 10.20.30.50
   ├─ dst_port: 6379
   └─ proto: TCP

2. 匹配规则 1003
   ├─ dst_ip: 10.20.30.50 ✅
   ├─ dst_port: 6379 ✅
   └─ proto: TCP ✅

3. 执行动作
   └─> TC_ACT_SHOT (丢弃)

4. 记录威胁日志
   └─> 上报到 agent
```

#### 场景 4: 访问其他服务（默认拒绝）

```
数据包: 192.168.1.100:52348 -> 8.8.8.8:53 (UDP/DNS)
    ↓
1. 提取五元组
   └─> dst_ip: 8.8.8.8, dst_port: 53, proto: UDP

2. 遍历所有规则
   └─> 无匹配规则

3. 应用默认动作
   └─> DefAction: DENY (2)

4. 执行动作
   └─> TC_ACT_SHOT (丢弃)
```

---

## 八、故障处理与重连机制

### 8.1 心跳保活机制

#### Agent 侧心跳发送

**代码位置**：agent/dp/ctrl.go

```go
const dpKeepAliveInterval = time.Second * 2
var keepAliveSeq uint32

func sendKeepAlive() {
    ticker := time.NewTicker(dpKeepAliveInterval)
    defer ticker.Stop()

    for range ticker.C {
        keepAliveSeq++

        data := DPKeepAliveReq{
            Alive: &DPKeepAlive{
                SeqNum: keepAliveSeq,
            },
        }

        msg, _ := json.Marshal(data)
        dpSendMsg(msg)
    }
}
```

#### dp 侧心跳响应

**代码位置**：dp/ctrl.c:141-156

```c
static int dp_ctrl_keep_alive(json_t *msg)
{
    uint32_t seq_num = json_integer_value(json_object_get(msg, "seq_num"));
    uint8_t buf[sizeof(DPMsgHdr) + sizeof(uint32_t)];

    // 构造响应
    DPMsgHdr *hdr = (DPMsgHdr *)buf;
    hdr->Kind = DP_KIND_KEEP_ALIVE;
    hdr->Length = htons(sizeof(uint32_t));
    hdr->SeqNum = htonl(seq_num);  // 回显序列号

    uint32_t *data = (uint32_t *)(hdr + 1);
    *data = htonl(seq_num);

    // 发送响应
    return dp_ctrl_send_binary(buf, sizeof(buf));
}
```

### 8.2 连接检测与重连

#### 连接失败检测

```go
var dpAliveMsgCnt uint = 0
var lastDpAliveMsgCnt uint = 0

func monitorDPConnection() {
    ticker := time.NewTicker(time.Second * 10)
    defer ticker.Stop()

    for range ticker.C {
        // 检查消息计数是否增长
        if dpAliveMsgCnt == lastDpAliveMsgCnt {
            log.Warn("DP connection seems dead, reconnecting...")
            reconnectDP()
        }
        lastDpAliveMsgCnt = dpAliveMsgCnt
    }
}
```

#### 重连逻辑

```go
func reconnectDP() {
    dpClientLock()
    defer dpClientUnlock()

    // 1. 关闭旧连接
    if dpConn != nil {
        dpConn.Close()
        dpConn = nil
    }

    // 2. 重新连接
    for retry := 0; retry < 5; retry++ {
        addr := net.UnixAddr{
            Name: DPServer,  // /tmp/dp_listen.sock
            Net:  "unixgram",
        }

        conn, err := net.DialUnix("unixgram", nil, &addr)
        if err != nil {
            log.WithFields(log.Fields{
                "retry": retry, "error": err,
            }).Warn("Failed to connect to DP")

            time.Sleep(time.Second * 2)
            continue
        }

        dpConn = conn
        log.Info("Reconnected to DP")

        // 3. 重新下发策略
        resyncAllPolicies()
        return
    }

    log.Error("Failed to reconnect to DP after retries")
}
```

### 8.3 策略重新同步

```go
func resyncAllPolicies() {
    log.Info("Resyncing all policies to DP...")

    // 1. 遍历所有 workload
    for _, wl := range getAllWorkloads() {
        // 2. 重新计算策略
        policy := calculateWorkloadPolicy(wl)

        // 3. 下发策略
        dp.DPCtrlConfigPolicy(policy, 1)  // cmd=1 (ADD)
    }

    // 4. 重新下发 FQDN 映射
    for _, fqdn := range getAllFqdns() {
        dp.DPCtrlSetFqdnIp(fqdn)
    }

    // 5. 重新下发 MAC 配置
    for _, mac := range getAllMacs() {
        dp.DPCtrlAddMAC(mac.Iface, mac.MAC, ...)
    }

    log.Info("Policy resync completed")
}
```

### 8.4 错误处理最佳实践

#### Agent 侧

```go
func sendPolicyWithRetry(policy *dp.DPWorkloadIPPolicy, maxRetries int) error {
    for retry := 0; retry < maxRetries; retry++ {
        err := dp.DPCtrlConfigPolicy(policy, 1)
        if err == 0 {
            return nil  // 成功
        }

        log.WithFields(log.Fields{
            "retry": retry, "wl_id": policy.WlID,
        }).Warn("Failed to send policy, retrying...")

        time.Sleep(time.Second * 1)
    }

    return fmt.Errorf("failed to send policy after %d retries", maxRetries)
}
```

#### dp 侧

```c
static int dp_ctrl_handler(int fd)
{
    json_t *root;
    json_error_t error;

    // 1. 解析 JSON
    root = json_loads(ctrl_msg_buf, 0, &error);
    if (root == NULL) {
        DEBUG_ERROR(DBG_CTRL,
                    "Invalid json format on line %d: %s\n",
                    error.line, error.text);
        return -1;  // 解析失败，忽略消息
    }

    // 2. 遍历处理
    json_object_foreach(root, key, msg) {
        // 处理各种命令
        if (strcmp(key, "ctrl_cfg_policy") == 0) {
            ret = dp_ctrl_cfg_policy(msg);
            if (ret < 0) {
                DEBUG_ERROR(DBG_CTRL, "Policy config failed\n");
                // 继续处理其他消息，不中断
            }
        }
        // ...
    }

    json_decref(root);
    return ret;
}
```

---

## 附录

### A. 常见问题

#### Q1: 策略下发后多久生效？

**A**: 立即生效。dp 接收到策略后会立即编译并插入到策略引擎，下一个数据包就会使用新策略匹配。

#### Q2: 单个 workload 可以配置多少条规则？

**A**: 理论上无限制，但受限于：
- JSON 消息大小限制（256KB/批）
- 每批默认 40 条规则
- 内存限制

实际生产环境建议不超过 10000 条规则/workload。

#### Q3: FQDN 规则如何更新 IP？

**A**: Agent 定期（默认每 30 秒）重新解析 FQDN，如果 IP 变化，会自动下发新的映射到 dp。

#### Q4: dp 崩溃后如何恢复策略？

**A**: Agent 通过心跳检测到 dp 断线后，会自动重连并重新下发所有策略（resyncAllPolicies）。

#### Q5: 应用层规则匹配失败怎么办？

**A**: 如果 DPI 无法识别应用协议，规则不会匹配。建议同时配置：
- 基于端口的规则（保底）
- 基于应用的规则（精确）

### B. 性能优化建议

1. **批量下发策略**
   - 使用 MSG_START/MSG_END 标志
   - 合理设置批大小（40-100 条/批）

2. **减少 FQDN 数量**
   - 使用通配符 FQDN（`*.github.com`）
   - 定期清理未使用的 FQDN

3. **优化规则顺序**
   - 高频规则放前面
   - 使用 IP 范围而非多条单 IP 规则

4. **避免规则冲突**
   - 确保规则不重叠
   - 使用优先级（rule ID 越小优先级越高）

### C. 调试技巧

#### 启用 dp 调试日志

```bash
# 启用所有日志
./dp -s -d all

# 启用特定类别
./dp -s -d "ctrl,packet,policy"
```

#### 监控 Unix Socket 通信

```bash
# 使用 socat 拦截
socat -v UNIX-LISTEN:/tmp/dp_debug.sock,fork \
         UNIX-CONNECT:/tmp/dp_listen.sock

# agent 连接到 /tmp/dp_debug.sock 而非 /tmp/dp_listen.sock
```

#### 查看策略配置

```bash
# 发送信号触发策略 dump
kill -USR1 $(pidof dp)

# 查看日志
tail -f /var/log/agent/dp.log
```

---

**文档版本**：1.0
**最后更新**：2025-10-29
**相关文档**：
- [NeuVector dp 与 agent 通信机制详解](./neuvector-dp-agent-communication.md)
- [NeuVector dp 组件编译指南](./neuvector-dp-build-guide.md)
