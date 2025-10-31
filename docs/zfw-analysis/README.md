# ZFW (Zero Trust Firewall) 技术分析文档

本目录包含 ZFW 零信任防火墙的完整技术分析文档，涵盖架构设计、实现细节和快速参考。

## 📚 文档目录

### 1. [zfw-architecture-analysis.md](zfw-architecture-analysis.md) (33KB, 1330 行) ⭐⭐⭐⭐⭐
**ZFW eBPF 架构深度分析**

完整的 ZFW 架构技术分析，包含所有实现细节：

**内容概览**:
- **eBPF 挂载点**: 3 个关键挂载点 (XDP, TC Ingress, TC Egress) 的详细分析
- **BPF Maps**: 34 个 Maps 的完整说明（类型、大小、用途）
- **核心数据结构**: 16 个关键结构体的字段解析
- **TCP 状态机**: 11 种 TCP 状态的完整状态转换图
- **工作流程**: 4 条数据路径的详细流程图
- **架构图表**: 多层架构、数据流、状态机的可视化

**重点技术**:
- XDP 隧道重定向机制
- TCP 连接跟踪与状态管理
- TPROXY 透明代理实现
- DDoS 防护 (SYN Flood 检测)
- NAT/Masquerade 地址转换
- 工控协议 (DNP3, Modbus) 深度检测

**适合场景**:
- 深入理解 ZFW 完整架构
- eBPF 防火墙技术学习
- 实现类似系统的参考设计
- 调试和性能优化

**阅读时间**: 60-90 分钟

---

### 2. [zfw-technical-diagrams.md](zfw-technical-diagrams.md) (34KB, 1031 行) ⭐⭐⭐⭐⭐
**ZFW 关键技术图表集** 🆕

通过详细的 Mermaid 技术图表深入理解 ZFW 核心实现机制：

**包含图表** (6 个完整流程图):
1. **完整数据包处理流程图** - 出站/入站连接的完整生命周期
2. **策略匹配和缓存流程图** - matched_map 缓存机制和前缀匹配
3. **TPROXY 决策树和 action/6 调用时机** - Socket 查找和程序分发逻辑
4. **Masquerade 完整流程** - NAT/SNAT/DNAT 和端口随机化
5. **隧道快速路径优化** - XDP 加速和 tun_map 状态缓存
6. **Map 操作和数据流关系图** - 所有 Map 的读写关系矩阵

**核心价值**:
- ✅ 可视化理解复杂的数据包流转路径
- ✅ 准确理解双 Map 架构 (tcp_map + tcp_ingress_map)
- ✅ 掌握 action vs action/6 的协作机制
- ✅ 理解性能优化的关键技术点

**适合场景**:
- 深入技术调研和学习
- 架构设计参考
- 技术分享和文档编写
- 调试复杂问题

**阅读时间**: 45-60 分钟

---

### 3. [zfw-quick-reference.md](zfw-quick-reference.md) (7.4KB, 303 行) ⭐⭐⭐⭐
**ZFW 快速参考手册**

快速查阅的技术手册，适合日常开发和调试：

**内容概览**:
- **一分钟了解 ZFW**: 核心功能和特性概览
- **核心 Map 速查表**: 34 个 Maps 按功能分类
  - 策略 Maps (5 个)
  - 连接追踪 Maps (6 个)
  - NAT Maps (3 个)
  - DDoS 防护 Maps (6 个)
  - 工控协议 Maps (4 个)
  - 其他 Maps (10 个)
- **数据结构速查**: 关键结构体的快速参考
- **TCP 状态速查**: 11 种状态的简明表格
- **常用操作**: Map 查询、调试命令

**适合场景**:
- 快速查找 Map 名称和用途
- 回忆数据结构字段
- 调试时查看状态含义
- 日常开发参考

**阅读时间**: 10-15 分钟

---

## 📖 按功能分类

### 架构设计类
- [zfw-architecture-analysis.md](zfw-architecture-analysis.md) - 完整架构分析 ⭐⭐⭐⭐⭐
- [zfw-technical-diagrams.md](zfw-technical-diagrams.md) - 关键技术图表集 ⭐⭐⭐⭐⭐ 🆕

### 参考手册类
- [zfw-quick-reference.md](zfw-quick-reference.md) - 快速参考手册 ⭐⭐⭐⭐

---

## 🎯 推荐学习路径

### 路径 1: 快速上手 (10-15 分钟)
适合已经了解 eBPF 基础，需要快速了解 ZFW 的开发者

1. ✅ [zfw-quick-reference.md](zfw-quick-reference.md) - 快速浏览核心概念
   - 重点：Map 速查表、TCP 状态表

### 路径 2: 深度学习 (1-1.5 小时)
适合需要深入理解 ZFW 架构和实现细节的开发者

1. ✅ [zfw-quick-reference.md](zfw-quick-reference.md) (15 分钟) - 先建立整体印象
2. ✅ [zfw-architecture-analysis.md](zfw-architecture-analysis.md) (60-90 分钟) - 系统学习
   - 重点章节：
     - eBPF 挂载点 (理解数据包处理流程)
     - BPF Maps 映射表 (理解数据存储)
     - TCP 状态机 (理解连接跟踪)
     - 工作流程 (理解完整数据路径)

### 路径 3: 特定功能研究
根据具体需求选择性阅读

**研究 TCP 连接跟踪**:
- [zfw-architecture-analysis.md](zfw-architecture-analysis.md) → "TCP 状态机" 章节
- [zfw-quick-reference.md](zfw-quick-reference.md) → "TCP 状态速查" 部分

**研究 DDoS 防护**:
- [zfw-architecture-analysis.md](zfw-architecture-analysis.md) → "DDoS 防护 Maps" 章节

**研究工控协议过滤**:
- [zfw-architecture-analysis.md](zfw-architecture-analysis.md) → "工控协议 Maps" 章节

**研究 NAT 实现**:
- [zfw-architecture-analysis.md](zfw-architecture-analysis.md) → "NAT Maps" 章节

---

## 🗺️ 快速导航表

| 需求 | 推荐文档 | 章节 |
|------|---------|------|
| 了解 ZFW 基本功能 | zfw-quick-reference.md | 一分钟了解 ZFW |
| 查看 Map 列表 | zfw-quick-reference.md | 核心 Map 速查表 |
| 理解 eBPF 挂载点 | zfw-architecture-analysis.md | eBPF 挂载点 |
| 查看所有 Maps | zfw-architecture-analysis.md | BPF Maps 映射表 |
| 理解 TCP 状态机 | zfw-architecture-analysis.md | TCP 状态机 |
| 理解数据流 | zfw-architecture-analysis.md | 工作流程 |
| 查看架构图 | zfw-architecture-analysis.md | 架构图表 |
| 快速查找数据结构 | zfw-quick-reference.md | 数据结构速查 |

---

## 🔑 关键技术摘要

### eBPF 挂载点
- **XDP** (`zfw_xdp_tun_ingress.c`) - 隧道入口流量重定向，最早期包处理
- **TC Ingress** (`zfw_tc_ingress.c`, `zfw_tc_ingress_object.c`) - 入向流量过滤和策略应用
- **TC Egress** (`zfw_tc_outbound_track.c`) - 出向流量状态跟踪

### 核心 Maps 统计
- **总数**: 34 个 BPF Maps
- **策略相关**: 5 个 (TPROXY 策略、端口范围、缓存)
- **连接追踪**: 6 个 (TCP、UDP、ICMP 状态)
- **NAT 相关**: 3 个 (SNAT/DNAT 映射)
- **DDoS 防护**: 6 个 (SYN Flood 检测和封禁)
- **工控协议**: 4 个 (DNP3、Modbus 过滤规则)

### TCP 状态机
支持 11 种 TCP 状态:
- `CLOSED` (0) → `SYN_SENT` (1) → `SYN_RECV` (2) → `ESTABLISHED` (3)
- `FIN_WAIT_1` (4) → `FIN_WAIT_2` (5) → `CLOSING` (6) → `TIME_WAIT` (7)
- `CLOSE_WAIT` (8) → `LAST_ACK` (9)
- `RST_RECV` (10) - 特殊状态

### 性能参数
- **最大并发连接**: 65,535 (TCP/UDP 各)
- **策略条目**: 250,000 (端口范围)
- **TPROXY 策略**: 100 (IPv4/IPv6 各)
- **隧道连接**: 10,000
- **LRU 自动淘汰**: 连接超限时自动清理

### 安全功能
1. **DDoS 防护**:
   - SYN Flood 检测 (每秒 SYN 包计数)
   - 自动封禁机制 (超过阈值)
   - 封禁列表管理 (IPv4/IPv6)

2. **工控协议过滤**:
   - DNP3 功能码过滤
   - Modbus 功能码过滤
   - 深度数据包检测 (DPI)

3. **透明代理**:
   - TPROXY 重定向
   - NAT/Masquerade
   - 连接状态保持

---

## 📊 文档统计

| 项目 | 数量 |
|------|------|
| 文档总数 | 3 个 |
| 总大小 | 102.4KB |
| 总行数 | 3,400+ 行 |
| 核心文档 | 2 个 (⭐⭐⭐⭐⭐) |
| 参考文档 | 1 个 (⭐⭐⭐⭐) |
| Mermaid 图表 | 20+ 个 |

---

## 💡 使用建议

### 首次阅读（推荐顺序）
1. 先阅读 [zfw-quick-reference.md](zfw-quick-reference.md) 建立整体概念 (15 分钟)
2. 查看 [zfw-technical-diagrams.md](zfw-technical-diagrams.md) 理解核心流程 (45-60 分钟) 🆕
   - 重点：完整数据包处理流程图、TPROXY 决策树
3. 再深入阅读 [zfw-architecture-analysis.md](zfw-architecture-analysis.md) (60-90 分钟)
4. 结合源码 `/source-references/zfw/` 验证理解

### 日常开发
- 查询 Map: 使用 [zfw-quick-reference.md](zfw-quick-reference.md)
- 调试问题: 查看 [zfw-architecture-analysis.md](zfw-architecture-analysis.md) 对应章节
- 性能优化: 参考架构分析中的性能参数

### 实现参考
- 借鉴 eBPF 挂载点设计
- 参考 TCP 状态机实现
- 学习 DDoS 防护机制
- 了解工控协议 DPI 实现

---

## 🔗 相关资源

### 项目内文档
- [ARCHITECTURE_OVERVIEW.md](../ARCHITECTURE_OVERVIEW.md) - 本项目整体架构
- [neuvector-analysis/](../neuvector-analysis/) - NeuVector 源码分析
- [READING_GUIDE.md](../READING_GUIDE.md) - 完整文档阅读指南

### 源码位置
- ZFW 源码: `/source-references/zfw/`
- eBPF 程序: `/source-references/zfw/` (`.c` 文件)
- 配置工具: `/source-references/zfw/` (用户态程序)

---

## 📝 文档版本

- **创建日期**: 2025-10-31
- **最后更新**: 2025-10-31
- **文档数量**: 2 个
- **维护状态**: ✅ 活跃维护

---

**建议**: 从 [zfw-quick-reference.md](zfw-quick-reference.md) 开始，快速建立对 ZFW 的整体认识！
