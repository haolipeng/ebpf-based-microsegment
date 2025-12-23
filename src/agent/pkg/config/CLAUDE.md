[上级索引](../CLAUDE.md) > **config**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# config

## 架构定位

配置管理器 | 输入: YAML 配置文件（agent-server.yaml / standalone.yaml）、环境变量 | 输出: 配置结构体（运行模式、网卡、API 地址、K8s 设置等）

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| config.go | 配置定义和加载逻辑 | `LoadConfig()`, `DefaultConfig()`, `Validate()` |

## 配置结构

主要配置项：Mode（agent-server/standalone）、Interface（网卡名称）、LogLevel（debug/info/warn/error）、StatsInterval（统计打印间隔）、API（API 服务器配置）、AgentServer（Server 连接配置）、Flow（流收集配置）、DataPlane（数据平面配置）、Kubernetes（K8s 集成配置）

## 核心功能

- **YAML 解析**: 使用 Viper 解析配置文件
- **默认值**: 提供合理的默认配置
- **验证**: 检查必填字段和配置合法性
- **环境变量**: 支持通过环境变量覆盖配置

## 应用场景

- **多环境部署**: 不同环境使用不同配置文件
- **动态配置**: 通过环境变量覆盖配置
- **配置验证**: 启动时检查配置合法性
