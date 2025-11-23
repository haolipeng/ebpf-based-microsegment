# eBPF 微隔离系统

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org/)
[![Linux Kernel](https://img.shields.io/badge/Linux-6.x+-FCC624?logo=linux&logoColor=black)](https://kernel.org/)
[![eBPF](https://img.shields.io/badge/eBPF-TC%20Hook-orange)](https://ebpf.io/)

**[English](README.md) | [中文](README_CN.md)**

基于 eBPF 技术的高性能内核级微隔离解决方案，为云原生环境提供细粒度的网络流量控制能力。

## 概述

eBPF 微隔离系统在内核层面实现网络隔离和访问控制，数据包处理延迟可达亚微秒级。系统由以下组件构成：

- **数据平面**：挂载在 TC（流量控制）钩子上的 eBPF 程序，实现线速数据包过滤
- **控制平面**：基于 Go 语言的 Agent 和 Server 组件，负责策略管理和监控
- **Web UI**：基于 React 的可视化管理界面

## 特性

- **极致性能**：热路径延迟 < 1μs，冷路径延迟 < 20μs
- **会话追踪**：基于 LRU 的连接跟踪，支持 10 万并发会话
- **多级策略匹配**：精确匹配 + 通配符匹配（CIDR/端口范围）+ 默认策略
- **Per-CPU 统计**：无锁计数器，零 CPU 竞争
- **实时事件**：Ring Buffer 实现流事件上报（新连接、拒绝事件）
- **RESTful API**：完整的策略 CRUD 操作接口
- **gRPC 通信**：Agent-Server 间基于 Protocol Buffers 的通信
- **TCP 状态机**：连接状态跟踪，支持有状态过滤

## 架构

```
┌────────────────────────────────────────────────────────────────┐
│                      用户 / 外部系统                            │
│                  (Web UI / API / 编排系统)                      │
└─────────────────────────────┬──────────────────────────────────┘
                              │ HTTP/gRPC
┌─────────────────────────────▼──────────────────────────────────┐
│                     控制平面 (用户空间)                          │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────────┐ │
│  │    Server    │  │    Agent     │  │   策略管理器          │ │
│  │  (gRPC API)  │◄─┤ (eBPF 管理)  │──┤   + 数据平面管理器    │ │
│  └──────────────┘  └──────────────┘  └───────────────────────┘ │
└─────────────────────────────┬──────────────────────────────────┘
                              │ Cilium eBPF Library
┌─────────────────────────────▼──────────────────────────────────┐
│                      数据平面 (内核空间)                         │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  TC eBPF 程序                                            │  │
│  │  • 数据包解析 (5元组)      • 会话跟踪 (LRU)              │  │
│  │  • 策略匹配 (Hash)         • 统计收集 (Per-CPU)          │  │
│  └──────────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  eBPF Maps: session_map | policy_map | stats_map | ...   │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
```

## 快速开始

### 环境要求

- Linux 内核 6.x+（支持 eBPF）
- Go 1.21+
- Clang/LLVM（用于 eBPF 编译）
- PostgreSQL 14+（Server 组件依赖）
- Node.js 18+（Web UI 依赖）

### 安装

```bash
# 克隆仓库
git clone https://github.com/your-org/ebpf-based-microsegment.git
cd ebpf-based-microsegment

# 安装依赖
make deps

# 构建所有组件
make all

# 或单独构建特定组件
make agent    # 仅构建 Agent
make server   # 仅构建 Server
```

### 运行

**启动 Server：**
```bash
# 初始化数据库
./src/server/scripts/migrate.sh

# 启动服务
./bin/microsegment-server --config config/server.yaml
```

**启动 Agent：**
```bash
# 需要 root 权限加载 eBPF 程序
sudo ./bin/microsegment-agent --interface eth0 --server localhost:50051
```

**启动 Web UI：**
```bash
cd web
npm install
npm run dev
```

### 快速演示

```bash
# 一键启动所有组件（Server + Agent + Web）
./start-all.sh

# Web UI 访问地址：http://localhost:5173
# API 访问地址：http://localhost:8080
```

## 配置说明

### Agent 配置

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--interface` | 挂载 eBPF 的网络接口 | `eth0` |
| `--server` | Server gRPC 地址 | `localhost:50051` |
| `--api-addr` | 本地 API 监听地址 | `127.0.0.1:8080` |
| `--log-level` | 日志级别（debug/info/warn/error） | `info` |

### Server 配置

通过 `config/server.yaml` 配置：

```yaml
server:
  grpc_port: 50051
  http_port: 8081

database:
  host: localhost
  port: 5432
  user: microsegment_user
  password: secret
  name: microsegment
```

## API 参考

### 策略管理

| 方法 | 端点 | 说明 |
|------|------|------|
| POST | `/api/v1/policies` | 创建策略 |
| GET | `/api/v1/policies` | 获取所有策略 |
| GET | `/api/v1/policies/:id` | 获取指定策略 |
| PUT | `/api/v1/policies/:id` | 更新策略 |
| DELETE | `/api/v1/policies/:id` | 删除策略 |

### 统计信息

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/v1/stats` | 获取所有统计 |
| GET | `/api/v1/stats/packets` | 获取数据包统计 |
| GET | `/api/v1/stats/policies` | 获取策略命中统计 |

### 健康检查

| 方法 | 端点 | 说明 |
|------|------|------|
| GET | `/api/v1/health` | 简单健康检查 |
| GET | `/api/v1/status` | 详细系统状态 |

### 示例：创建策略

```bash
curl -X POST http://localhost:8080/api/v1/policies \
  -H "Content-Type: application/json" \
  -d '{
    "rule_id": 1001,
    "src_ip": "10.0.0.0/24",
    "dst_ip": "192.168.1.100",
    "dst_port": 443,
    "protocol": "tcp",
    "action": "allow"
  }'
```

## 构建选项

```bash
# 生产版本构建（优化，全部特性）
make build-production

# 调试版本构建（包含调试日志）
make build-debug

# 最小版本构建（无 NAT/分片处理）
make build-minimal

# 显示当前配置
make show-config
```

### eBPF 特性开关

| 开关 | 说明 | 默认值 |
|------|------|--------|
| `DEBUG_MODE` | 启用 eBPF 调试日志 | 0 |
| `ENABLE_IP_FRAGMENT_HANDLING` | 处理 IP 分片 | 1 |
| `ENABLE_NAT_SUPPORT` | NAT 检测支持 | 1 |

## 性能指标

| 指标 | 数值 | 说明 |
|------|------|------|
| 热路径延迟 | < 1μs | 99%+ 数据包（已有会话） |
| 冷路径延迟 | 5-20μs | 新会话含策略查询 |
| 精确策略匹配 | ~0.1μs | O(1) 哈希查找 |
| 通配符策略匹配 | 2-20μs | 索引扫描 + CIDR 匹配 |
| 最大并发会话 | 10 万 | LRU 自动淘汰 |
| 最大策略数 | 1 万精确 + 1 千通配符 | 可配置 |

## 项目结构

```
.
├── src/
│   ├── agent/          # Agent 组件（eBPF 管理）
│   │   ├── cmd/        # 入口
│   │   └── pkg/        # 包（api, dataplane, policy）
│   ├── bpf/            # eBPF C 程序
│   └── server/         # Server 组件（策略服务器）
├── api/
│   └── proto/          # Protocol Buffer 定义
├── web/                # React Web UI
├── config/             # 配置文件
├── deploy/             # 部署脚本（systemd, docker）
├── docs/               # 文档
└── tests/              # 集成测试
```

## 测试

```bash
# 运行单元测试
make test

# 运行集成测试（需要 root 权限）
sudo make test-integration

# 运行特定测试
cd src/agent && go test -v ./pkg/dataplane/...
```

## 部署

### Systemd 部署

```bash
# 安装 systemd 服务
sudo ./deploy/scripts/install-systemd.sh

# 启动服务
sudo systemctl start microsegment-server
sudo systemctl start microsegment-agent
```

### Docker 部署

```bash
# 使用 docker-compose 构建并运行
docker-compose up -d
```

## 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 提交 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件。

## 致谢

- [Cilium eBPF Library](https://github.com/cilium/ebpf) - Go eBPF 库
- [libbpf](https://github.com/libbpf/libbpf) - eBPF 库
- [NeuVector](https://github.com/neuvector/neuvector) - 网络安全概念参考
