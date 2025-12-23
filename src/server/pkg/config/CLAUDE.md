[上级索引](../CLAUDE.md) > **config**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# config

## 架构定位

配置管理器 | 输入: YAML 配置文件（server.yaml）、环境变量 | 输出: 配置结构体（HTTP/gRPC 端口、数据库连接、日志级别）

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| config.go | 配置定义和加载逻辑 | `LoadConfig()`, `Config`, `ServerConfig`, `DatabaseConfig` |

## 配置结构

主要配置项：
- **Server**: HTTP 服务器主机和端口
- **GRPC**: gRPC 服务器主机和端口
- **Database**: PostgreSQL 连接参数、连接池配置
- **Log**: 日志级别和格式

## 核心功能

- **Viper 集成**: YAML 文件解析
- **环境变量覆盖**: `MICROSEGMENT_` 前缀的环境变量
- **默认值**: 合理的生产默认配置
- **连接池配置**: MaxOpenConns、MaxIdleConns、ConnMaxLifetime

