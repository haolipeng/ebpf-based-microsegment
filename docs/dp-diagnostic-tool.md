# dp 诊断工具 (diag.py)

## 概述

NeuVector 提供了一个 Python 诊断工具 `diag.py`，可以直接向 dp 组件发送 JSON 命令，用于：
- **调试和诊断** dp 组件的运行状态
- **模拟 agent 行为**，手动发送策略和控制命令
- **测试 dp 功能**，验证 dp 是否正确响应各种控制命令

## 工具位置

```
source-references/neuvector/dp/diagnose/diag.py
```

## 工作原理

### 通信机制

```
┌─────────────┐                                ┌─────────────┐
│   diag.py   │                                │  dp process │
│  (Python)   │                                │     (C)     │
└──────┬──────┘                                └──────┬──────┘
       │                                              │
       │  Unix Domain Socket (SOCK_DGRAM)            │
       │  /tmp/dp_ctrl_client.<PID>                  │
       │                                              │
       │  JSON Commands                               │
       │  ──────────────────────────────────────────> │
       │                          /tmp/dp_ctrl.sock   │
       │                                              │
       │  JSON Responses                              │
       │  <────────────────────────────────────────── │
       │                                              │
```

**关键特点：**
- 使用 Unix Domain Socket (DATAGRAM 模式)
- Client socket: `/tmp/dp_ctrl_client.<PID>`
- Server socket: `/tmp/dp_ctrl.sock`
- 通信协议: JSON (双向)
- 独立于 agent，可直接与 dp 交互

### 与 agent 通信的区别

| 特性 | agent → dp | diag.py → dp |
|------|-----------|-------------|
| Socket 路径 | `/tmp/ctrl_listen.sock` → `/tmp/dp_listen.sock` | `/tmp/dp_ctrl_client.<PID>` → `/tmp/dp_ctrl.sock` |
| 用途 | 生产环境策略下发和数据上报 | 调试和诊断 |
| 消息类型 | JSON (Agent→dp) + Binary (dp→Agent) | 仅 JSON |
| 支持命令 | 30+ 全功能控制命令 | 诊断相关子集 |

## 当前实现的功能

### 1. 会话管理 (session)

#### 列出所有会话
```bash
./diag.py session list
```

**发送的 JSON:**
```json
{
  "ctrl_list_session": {}
}
```

**返回示例:**
```json
{
  "sessions": [
    {
      "client_ip": "192.168.1.100",
      "server_ip": "10.0.0.5",
      "client_port": 54321,
      "server_port": 443,
      "protocol": 6,
      "application": "TLS",
      "policy_action": 1,
      "age": 120
    }
  ],
  "more": false
}
```

#### 统计会话数量
```bash
./diag.py session count
```

**发送的 JSON:**
```json
{
  "ctrl_count_session": {}
}
```

**返回示例:**
```json
{
  "dp_count_session": {
    "cur_sessions": 1234,
    "cur_tcp_sessions": 890,
    "cur_udp_sessions": 244,
    "cur_icmp_sessions": 100
  }
}
```

### 2. 调试控制 (debug)

#### 启用调试类别
```bash
./diag.py debug enable packet
./diag.py debug enable all
```

**支持的类别:**
- `all` - 所有调试信息
- `init` - 初始化阶段
- `error` - 错误信息
- `ctrl` - 控制消息
- `packet` - 数据包处理
- `session` - 会话跟踪
- `timer` - 定时器事件
- `tcp` - TCP 协议处理
- `parser` - 协议解析器

**发送的 JSON:**
```json
{
  "ctrl_set_debug": {
    "categories": ["+packet"]
  }
}
```

#### 禁用调试类别
```bash
./diag.py debug disable packet
```

**发送的 JSON:**
```json
{
  "ctrl_set_debug": {
    "categories": ["-packet"]
  }
}
```

#### 查看调试设置
```bash
./diag.py debug show
```

**发送的 JSON:**
```json
{
  "ctrl_get_debug": {}
}
```

**返回示例:**
```json
{
  "dp_debug": {
    "categories": ["error", "ctrl", "packet"],
    "level": 3
  }
}
```

## dp 支持的完整 JSON 命令列表

通过分析 `source-references/neuvector/dp/ctrl.c:2384-2496`，dp 支持以下所有 JSON 命令：

### 端口和网络接口管理
```json
{"ctrl_add_srvc_port": {...}}        // 添加服务端口
{"ctrl_del_srvc_port": {...}}        // 删除服务端口
{"ctrl_add_port_pair": {...}}        // 添加端口对
{"ctrl_del_port_pair": {...}}        // 删除端口对
{"ctrl_add_tap_port": {...}}         // 添加 TAP 端口
{"ctrl_del_tap_port": {...}}         // 删除 TAP 端口
{"ctrl_add_nfq_port": {...}}         // 添加 Netfilter Queue 端口
{"ctrl_del_nfq_port": {...}}         // 删除 Netfilter Queue 端口
```

### MAC 地址管理
```json
{"ctrl_add_mac": {...}}              // 添加 MAC 地址
{"ctrl_del_mac": {...}}              // 删除 MAC 地址
{"ctrl_cfg_mac": {...}}              // 配置 MAC 地址（完整配置）
{"ctrl_cfg_nbe": {...}}              // 配置 Network Behavior Engine
```

### 应用和统计
```json
{"ctrl_refresh_app": {...}}          // 刷新应用识别
{"ctrl_stats_macs": {...}}           // MAC 地址统计
{"ctrl_stats_device": {...}}         // 设备统计
{"ctrl_counter_device": {...}}       // 设备计数器
```

### 会话管理
```json
{"ctrl_count_session": {}}           // 统计会话数量 ✅ diag.py 已实现
{"ctrl_list_session": {}}            // 列出所有会话 ✅ diag.py 已实现
{"ctrl_clear_session": {...}}        // 清除会话
{"ctrl_list_meter": {...}}           // 列出流量计量
```

### 调试
```json
{"ctrl_set_debug": {...}}            // 设置调试级别 ✅ diag.py 已实现
{"ctrl_get_debug": {}}               // 获取调试设置 ✅ diag.py 已实现（已注释）
{"ctrl_keep_alive": {}}              // 保活消息
```

### 策略配置（核心功能）
```json
{"ctrl_cfg_policy": {...}}           // 配置网络策略（最重要）
{"ctrl_cfg_del_fqdn": {...}}         // 删除 FQDN 映射
{"ctrl_cfg_set_fqdn": {...}}         // 设置 FQDN 映射
{"ctrl_cfg_internal_net": {...}}     // 配置内部网络
{"ctrl_cfg_specip_net": {...}}       // 配置特殊 IP 网络
{"ctrl_cfg_policy_addr": {...}}      // 配置策略地址
```

### DLP（数据泄露防护）
```json
{"ctrl_cfg_dlp": {...}}              // 配置 DLP 规则
{"ctrl_cfg_dlpmac": {...}}           // 删除 DLP MAC
{"ctrl_bld_dlp": {...}}              // 构建 DLP 规则
{"ctrl_bld_dlpmac": {...}}           // 更新 DLP 端点
```

### 系统配置
```json
{"ctrl_sys_conf": {...}}                  // 系统配置
{"ctrl_disable_net_policy": {...}}        // 禁用网络策略
{"ctrl_detect_unmanaged_wl": {...}}       // 检测未管理的工作负载
{"ctrl_enable_icmp_policy": {...}}        // 启用 ICMP 策略
{"ctrl_strict_group_mode": {...}}         // 严格分组模式
```

## 扩展 diag.py 来模拟策略下发

### 添加策略配置命令

在 `diag.py` 中可以添加以下功能来模拟 agent 下发策略：

```python
@cli.group()
@click.pass_obj
def policy(data):
    """Policy operation."""

@policy.command()
@click.argument('wl_id')
@click.argument('json_file', type=click.File('r'))
@click.pass_obj
def configure(data, wl_id, json_file):
    """Configure workload policy from JSON file."""
    import json
    policy_data = json.load(json_file)

    body = {
        "ctrl_cfg_policy": {
            "workload_id": wl_id,
            "policy_mode": policy_data.get("policy_mode", "protect"),
            "default_action": policy_data.get("default_action", 2),  # 2=VIOLATE
            "apply_dir": policy_data.get("apply_dir", 3),  # 3=BOTH
            "mac_addresses": policy_data.get("mac_addresses", []),
            "rules": policy_data.get("rules", [])
        }
    }

    data.sock.sendall(json.dumps(body))
    click.echo("Policy configured for workload: %s" % wl_id)

@policy.command()
@click.argument('fqdn')
@click.argument('ip_addresses', nargs=-1)
@click.pass_obj
def add_fqdn(data, fqdn, ip_addresses):
    """Add FQDN to IP mapping."""
    body = {
        "ctrl_cfg_set_fqdn": {
            "fqdn_name": fqdn,
            "fqdn_ip": list(ip_addresses)
        }
    }

    data.sock.sendall(json.dumps(body))
    click.echo("FQDN mapping added: %s -> %s" % (fqdn, ", ".join(ip_addresses)))

@policy.command()
@click.argument('fqdn')
@click.pass_obj
def del_fqdn(data, fqdn):
    """Delete FQDN mapping."""
    body = {
        "ctrl_cfg_del_fqdn": {
            "fqdn_name": fqdn
        }
    }

    data.sock.sendall(json.dumps(body))
    click.echo("FQDN mapping deleted: %s" % fqdn)
```

### 策略 JSON 文件示例

创建 `test-policy.json`:

```json
{
  "policy_mode": "protect",
  "default_action": 2,
  "apply_dir": 3,
  "mac_addresses": ["02:42:ac:11:00:02"],
  "rules": [
    {
      "rule_id": 1,
      "src_ip": "192.168.1.0/24",
      "dst_ip": "0.0.0.0/0",
      "port": 443,
      "ip_proto": 6,
      "action": 1,
      "ingress": false,
      "applications": [
        {
          "app": 2100,
          "action": 1
        }
      ]
    },
    {
      "rule_id": 2,
      "fqdn": "*.example.com",
      "port": 443,
      "ip_proto": 6,
      "action": 1,
      "ingress": false
    }
  ]
}
```

### 使用方法

```bash
# 1. 启动 dp standalone 模式
./dp -s

# 2. 使用扩展的 diag.py 下发策略
./diag.py policy configure workload-abc test-policy.json

# 3. 添加 FQDN 映射
./diag.py policy add-fqdn www.example.com 93.184.216.34 2001:500:88:200::10

# 4. 查看会话
./diag.py session list

# 5. 启用调试来观察策略执行
./diag.py debug enable packet
./diag.py debug enable session

# 6. 删除 FQDN 映射
./diag.py policy del-fqdn www.example.com
```

## 源代码参考

### diag.py 实现
- **文件**: `source-references/neuvector/dp/diagnose/diag.py`
- **行数**: 1-106
- **语言**: Python 2.7
- **依赖**: `click` (命令行框架), `json`, `socket`

### dp 控制消息处理
- **文件**: `source-references/neuvector/dp/ctrl.c`
- **核心函数**:
  - `dp_ctrl_handler()` (2384-2496行) - JSON 消息分发
  - `dp_ctrl_cfg_policy()` (740-1050行) - 策略配置处理
  - `dp_ctrl_set_fqdn()` - FQDN 映射设置
  - `dp_ctrl_list_session()` (1226-1250行) - 会话列表
  - `dp_ctrl_set_debug()` (1286-1320行) - 调试控制

### Socket 路径定义
```c
// 文件: source-references/neuvector/dp/apis.h
#define DP_SERVER_SOCK      "/tmp/dp_ctrl.sock"      // diag.py 连接的 socket
#define DP_LISTEN_SOCK      "/tmp/dp_listen.sock"    // agent 连接的 socket
#define CTRL_NOTIFY_SOCK    "/tmp/ctrl_listen.sock"  // dp → agent 通知
```

## 测试场景示例

### 场景 1：验证 FQDN 匹配功能

```bash
# 1. 启动 dp
./dp -s

# 2. 配置 FQDN 策略（模拟 agent 下发）
./diag.py policy configure wl-test fqdn-policy.json

# fqdn-policy.json 内容：
# {
#   "rules": [{
#     "rule_id": 100,
#     "fqdn": "*.malicious.com",
#     "action": 2,  // VIOLATE/DENY
#     "ip_proto": 6,
#     "port": 443
#   }]
# }

# 3. 模拟 DNS 响应（触发 FQDN 映射学习）
# dp 会自动从流量中学习 DNS 响应，或者手动添加：
./diag.py policy add-fqdn evil.malicious.com 1.2.3.4

# 4. 启用调试观察匹配结果
./diag.py debug enable packet
./diag.py debug enable session

# 5. 发送测试流量到 1.2.3.4:443
# dp 会匹配到 FQDN 规则并阻止连接

# 6. 查看会话验证阻止结果
./diag.py session list
# 预期看到 policy_action=2 (VIOLATE)
```

### 场景 2：调试策略不生效问题

```bash
# 1. 启用所有调试
./diag.py debug enable all

# 2. 下发策略
./diag.py policy configure wl-001 policy.json

# 3. 观察 dp 日志输出（dp 会打印详细的策略安装过程）
# 日志示例：
# [CTRL] "ctrl_cfg_policy":{"workload_id":"wl-001",...}
# [POLICY] Compiling 15 rules for MAC 02:42:ac:11:00:02
# [POLICY] Rule 1: 192.168.1.0/24 -> 0.0.0.0/0:443 TCP action=ALLOW
# [POLICY] Policy installed successfully

# 4. 生成测试流量

# 5. 查看会话和统计
./diag.py session list
./diag.py session count

# 6. 如果策略不生效，检查：
#    - MAC 地址是否正确
#    - 规则优先级和匹配条件
#    - dp 是否正确捕获流量
```

### 场景 3：性能测试

```bash
# 1. 清空现有会话
./diag.py session clear

# 2. 配置测试策略
./diag.py policy configure wl-perf perf-policy.json

# 3. 禁用不必要的调试（提高性能）
./diag.py debug disable all
./diag.py debug enable error  # 只保留错误日志

# 4. 生成大量测试流量

# 5. 定期采样会话数和统计
while true; do
  ./diag.py session count
  ./diag.py stats device
  sleep 5
done

# 6. 观察 dp CPU 和内存使用
top -p $(pgrep dp)
```

## 与 agent 的集成测试

可以同时运行 agent 和 diag.py 来测试两者的交互：

```bash
# Terminal 1: 启动 dp standalone 模式
./dp -s

# Terminal 2: 启动 agent（或模拟 agent）
# agent 会连接到 /tmp/dp_listen.sock

# Terminal 3: 使用 diag.py 观察和调试
./diag.py debug enable ctrl
./diag.py session list
./diag.py session count

# Terminal 4: 触发 agent 下发策略
# 通过 REST API 或 gRPC 触发 agent 更新策略

# Terminal 3: 实时观察策略生效
watch -n 1 './diag.py session list'
```

## 优势总结

使用 `diag.py` 的好处：

1. **独立测试**：无需完整的 controller + agent 环境
2. **快速验证**：直接发送 JSON 命令，立即看到结果
3. **调试友好**：可以精确控制每个参数，逐步测试
4. **教育价值**：清晰展示 agent-dp 通信协议
5. **自动化测试**：可以编写脚本批量测试各种场景

## 注意事项

1. **Python 版本**：原始代码为 Python 2.7，可能需要修改为 Python 3
2. **Socket 权限**：需要有权限访问 `/tmp/dp_ctrl.sock`
3. **JSON 格式**：必须严格遵守 dp 期望的 JSON schema
4. **并发使用**：`diag.py` 和 agent 可以同时使用，但要注意配置冲突
5. **生产环境**：此工具仅用于开发和调试，不应在生产环境使用

## 进一步扩展建议

1. **添加更多命令**：实现上述列出的所有 30+ JSON 命令
2. **JSON Schema 验证**：在发送前验证 JSON 格式
3. **交互式模式**：提供 REPL 界面实时测试
4. **日志记录**：记录所有发送的命令和响应
5. **批处理模式**：从文件读取多个命令批量执行
6. **性能测试套件**：内置性能测试场景和报告
7. **升级到 Python 3**：使用现代 Python 特性和类型提示
