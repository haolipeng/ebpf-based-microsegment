# ZFW 快速参考手册

## 🎯 一分钟了解 ZFW

**ZFW (Zero Trust Firewall)** 是基于 eBPF 的高性能零信任防火墙：

- ⚡ **3个eBPF挂载点**: XDP (隧道重定向) + TC Ingress (入向过滤) + TC Egress (出向追踪)
- 🗺️ **34个BPF Maps**: 策略、状态、缓存、NAT、DDoS防护
- 🔄 **完整TCP状态机**: 支持11种状态跟踪
- 🛡️ **DDoS防护**: SYN Flood检测与封禁
- 🏭 **工控协议支持**: DNP3, Modbus 功能码过滤
- 📊 **65K并发连接**: LRU自动淘汰

---

## 📊 核心Map速查表

### 策略 Maps

| Map名称 | 类型 | 最大条目 | 用途 |
|---------|------|----------|------|
| `zt_tproxy_map` | HASH | 100 | IPv4 TPROXY策略 (前缀匹配) |
| `zt_tproxy6_map` | HASH | 100 | IPv6 TPROXY策略 |
| `range_map` | HASH | 250,000 | 端口范围→TPROXY端口映射 |
| `matched_map` | LRU_HASH | 65,536 | IPv4 策略匹配缓存 |
| `matched6_map` | LRU_HASH | 65,536 | IPv6 策略匹配缓存 |

### 连接追踪 Maps

| Map名称 | 类型 | 最大条目 | 用途 |
|---------|------|----------|------|
| `tcp_map` | LRU_HASH | 65,535 | TCP连接状态 (Egress) |
| `tcp_ingress_map` | LRU_HASH | 65,535 | TCP连接状态 (Ingress) |
| `udp_map` | LRU_HASH | 65,535 | UDP会话 (Egress) |
| `udp_ingress_map` | LRU_HASH | 65,535 | UDP会话 (Ingress) |
| `icmp_echo_map` | LRU_HASH | 65,536 | ICMP Echo追踪 |
| `tun_map` | LRU_HASH | 10,000 | 隧道连接状态 |

### NAT Maps

| Map名称 | 类型 | 最大条目 | 用途 |
|---------|------|----------|------|
| `masquerade_map` | HASH | 65,536 | SNAT 地址映射 |
| `masquerade_reverse_map` | HASH | 65,536 | DNAT 反向映射 |
| `icmp_masquerade_map` | HASH | 100 | ICMP Masquerade |

### DDoS 防护 Maps

| Map名称 | 类型 | 最大条目 | 用途 |
|---------|------|----------|------|
| `syn_count_map` | HASH | 256 | 每接口SYN包计数 |
| `ddos_saddr_map` | LRU_HASH | 100 | 源地址黑名单 |
| `ddos_dport_map` | HASH | 100 | 目标端口黑名单 |

### 接口管理 Maps

| Map名称 | 类型 | 最大条目 | 用途 |
|---------|------|----------|------|
| `ifindex_ip_map` | HASH | 256 | 接口→IPv4地址列表 |
| `ifindex_ip6_map` | HASH | 256 | 接口→IPv6地址列表 |
| `ifindex_tun_map` | ARRAY | 1 | 隧道接口配置 |
| `diag_map` | ARRAY | 1 | 全局诊断配置 |

### 事件日志 Map

| Map名称 | 类型 | 大小 | 用途 |
|---------|------|------|------|
| `rb_map` | RINGBUF | 256KB | 事件日志 Ring Buffer |

---

## 🔑 关键数据结构

### tproxy_key (策略键)
```c
{
    __u32 dst_ip;       // 目标IP
    __u32 src_ip;       // 源IP
    __u8 dprefix_len;   // 目标前缀长度
    __u8 sprefix_len;   // 源前缀长度
    __u8 protocol;      // TCP/UDP/ICMP
}
```
**示例**: `{192.168.1.0, 10.0.0.0, 24, 16, 6}` = 10.0.0.0/16 → 192.168.1.0/24 (TCP)

### tuple_key (5元组键)
```c
{
    union __in46_u_dst; // 目标IP (IPv4/IPv6)
    union __in46_u_src; // 源IP
    __u16 sport;        // 源端口
    __u16 dport;        // 目标端口
    __u32 ifindex;      // 接口索引
    __u8 type;          // 4=IPv4, 6=IPv6
}
```

### tcp_state (TCP状态)
```c
{
    unsigned long long tstamp;  // 时间戳
    __u32 sfseq, cfseq;         // FIN序列号
    __u8 syn;                   // SYN标志
    __u8 sfin, cfin;            // FIN标志
    __u8 sfack, cfack, ack;     // ACK标志
    __u8 rst;                   // RST标志
    __u8 est;                   // ESTABLISHED标志
}
```

---

## 🔄 数据流快速理解

### Ingress (入向) 流程
```
数据包到达 → XDP检查
             ↓
        隧道? → 是 → XDP_REDIRECT重定向
             ↓ 否
        TC Ingress
             ↓
   1. 查matched_map缓存
   2. 查zt_tproxy_map策略
   3. 查range_map端口范围
   4. 查tcp_ingress_map状态
   5. DDoS检查 (ddos_saddr_map)
   6. NAT (masquerade_reverse_map)
   7. 工控协议检查 (dnp3_fcode_map)
             ↓
   ALLOW → TC_ACT_OK
   DENY  → TC_ACT_SHOT
```

### Egress (出向) 流程
```
应用发包 → 内核协议栈
             ↓
        TC Egress
             ↓
   1. 查egress_matched_map缓存
   2. 查zt_egress_map策略
   3. 查tcp_map状态 (创建/更新)
   4. SNAT (masquerade_map)
   5. 同步到tcp_ingress_map
             ↓
   ALLOW → TC_ACT_OK
   DENY  → TC_ACT_SHOT
```

---

## 🎭 eBPF 程序位置

### XDP 程序
- **文件**: `zfw_xdp_tun_ingress.c`
- **挂载**: `SEC("xdp_redirect")`
- **函数**: `xdp_tun_ingress(struct xdp_md *ctx)`

### TC Ingress 程序
- **文件**: `zfw_tc_ingress.c`
- **挂载**:
  - `SEC("action")` - 主程序
  - `SEC("action/1")` ~ `SEC("action/6")` - Tail Call 子程序
- **主函数**: `ingress_filter(struct __sk_buff *skb)`

### TC Egress 程序
- **文件**: `zfw_tc_outbound_track.c`
- **挂载**:
  - `SEC("action")` - 主程序
  - `SEC("action/1")` ~ `SEC("action/6")` - Tail Call 子程序
- **主函数**: `egress_filter(struct __sk_buff *skb)`

---

## 🛠️ 常用操作

### 查看Map内容
```bash
# 查看TPROXY策略
sudo bpftool map dump name zt_tproxy_map

# 查看TCP连接
sudo bpftool map dump name tcp_map

# 查看策略缓存
sudo bpftool map dump name matched_map

# 查看NAT映射
sudo bpftool map dump name masquerade_map

# 查看DDoS黑名单
sudo bpftool map dump name ddos_saddr_map
```

### 查看eBPF程序
```bash
# 列出所有加载的程序
sudo bpftool prog show

# 查看XDP程序
sudo bpftool prog show type xdp

# 查看TC程序
sudo bpftool prog show type tc

# 查看程序详情
sudo bpftool prog dump xlated id <PROG_ID>
```

### 监控事件日志
```bash
# 使用zfw_monitor消费Ring Buffer
sudo ./zfw_monitor

# 或使用bpftool
sudo bpftool map event rb_map
```

---

## 📈 性能优化点

### 1. 缓存策略
- ✅ `matched_map`: 避免重复策略查找
- ✅ LRU Maps: 自动淘汰旧连接
- ✅ XDP卸载: 隧道流量绕过TC层

### 2. Map 大小调优
```c
// 根据实际需求调整
#define BPF_MAX_SESSIONS 65535      // TCP/UDP连接数
#define BPF_MAX_TUN_SESSIONS 10000  // 隧道连接数
#define BPF_MAX_RANGES 250000       // 端口范围数
#define MAX_TABLE_SIZE 65536        // 匹配缓存大小
```

### 3. Tail Call 分解
- 主程序 + 6个子程序
- 绕过eBPF 1M指令限制
- 每个子程序独立优化

### 4. Per-Interface 优化
```c
// diag_map中启用
bool per_interface = true;  // 接口级策略
```

---

## 🔍 故障排查

### 策略不生效
```bash
# 1. 检查策略是否加载
sudo bpftool map dump name zt_tproxy_map

# 2. 检查缓存状态
sudo bpftool map dump name matched_map | grep <IP>

# 3. 查看事件日志
sudo ./zfw_monitor | grep <IP>

# 4. 检查TC程序是否加载
sudo tc filter show dev eth0 ingress
```

### 连接被误拦截
```bash
# 1. 检查TCP状态
sudo bpftool map dump name tcp_ingress_map | grep <IP>:<PORT>

# 2. 检查DDoS黑名单
sudo bpftool map dump name ddos_saddr_map | grep <IP>

# 3. 查看丢包事件
sudo ./zfw_monitor | grep SHOT
```

### 性能问题
```bash
# 1. 检查Map压力
sudo bpftool map show

# 2. 查看连接数
sudo bpftool map dump name tcp_map | wc -l

# 3. 检查CPU使用
sudo bpftool prog profile
```

---

## 📚 相关文档

- **完整分析**: [zfw-architecture-analysis.md](./zfw-architecture-analysis.md)
- **6周学习指南**: [weekly-guide/](./weekly-guide/)
- **源代码**: `source-references/zfw/src/`

---

**版本**: 1.0
**更新日期**: 2025-10-24
