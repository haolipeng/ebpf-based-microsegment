# 参考文档

本章节提供 eBPF 微隔离系统的完整技术参考文档。

## 文档目录

### [安装指南](install/index.md)

系统安装和部署的详细说明。

- [系统要求](install/requirements.md)
- [Docker 安装](install/docker.md)
- [Kubernetes 安装](install/kubernetes.md)
- [二进制安装](install/binary.md)
- [源码编译](install/build-from-source.md)

### [架构设计](architecture/index.md)

系统架构和设计原理。

- [架构概述](architecture/overview.md)
- [数据流](architecture/data-flow.md)
- [数据平面](architecture/data-plane.md)
- [控制平面](architecture/control-plane.md)
- [eBPF Maps](architecture/ebpf-maps.md)

### [事件系统](events/index.md)

系统产生的各类事件。

- [事件概述](events/overview.md)
- [流事件](events/flow-events.md)
- [会话事件](events/session-events.md)
- [策略事件](events/policy-events.md)

### [策略系统](policies/index.md)

策略定义和管理。

- [策略概述](policies/overview.md)
- [策略语法](policies/syntax.md)
- [精确匹配策略](policies/exact-match.md)
- [通配符策略](policies/wildcard.md)
- [基于标签的策略](policies/label-based.md)

### [CLI 参数](cli-flags/index.md)

命令行参数参考。

- [Agent 参数](cli-flags/agent.md)
- [Server 参数](cli-flags/server.md)

### [数据源](data-sources/index.md)

内置和自定义数据源。

- [概述](data-sources/overview.md)
- [会话数据](data-sources/sessions.md)
- [流数据](data-sources/flows.md)
- [统计数据](data-sources/statistics.md)

### [集成](integrations/index.md)

与第三方系统集成。

- [Prometheus](integrations/prometheus.md)
- [Grafana](integrations/grafana.md)
- [日志系统](integrations/logging.md)

## API 参考

### REST API

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/v1/health` | GET | 健康检查 |
| `/api/v1/status` | GET | 系统状态 |
| `/api/v1/policies` | GET/POST | 策略管理 |
| `/api/v1/policies/:id` | GET/PUT/DELETE | 单个策略操作 |
| `/api/v1/stats` | GET | 统计数据 |
| `/api/v1/flows` | GET | 流数据查询 |
| `/api/v1/sessions` | GET | 会话数据查询 |

### gRPC API

用于 Agent 和 Server 之间的通信，详见 [gRPC API 文档](api/grpc.md)。
