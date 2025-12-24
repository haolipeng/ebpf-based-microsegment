# 参考文档

本章节提供 eBPF 微隔离系统的技术参考文档索引。

## 快速链接

| 类别 | 文档 | 位置 |
|------|------|------|
| **快速开始** | 构建指南 | [../build_guide.md](../build_guide.md) |
| **快速开始** | 部署指南 | [../DEPLOYMENT.md](../DEPLOYMENT.md) |
| **架构** | 架构概述 | [../architecture_overview.md](../architecture_overview.md) |
| **API** | Label-based Policy API | [../api-label-based-policies.md](../api-label-based-policies.md) |
| **学习** | 6周学习指南 | [../learning/weekly-guide/](../learning/weekly-guide/) |

## 文档目录

### 安装指南

- [源码编译](install/build-from-source.md) - 从源码编译和安装

### 架构设计

- [架构设计索引](architecture/index.md) - 架构文档导航

### 策略系统

- [基于标签的策略](policies/label-based.md) - Label-based Policy API 文档

## REST API 端点

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/v1/health` | GET | 健康检查 |
| `/api/v1/status` | GET | 系统状态 |
| `/api/v1/policies` | GET/POST | 策略管理 |
| `/api/v1/policies/:id` | GET/PUT/DELETE | 单个策略操作 |
| `/api/v1/stats` | GET | 统计数据 |
| `/api/v1/flows` | GET | 流数据查询 |
| `/api/v1/sessions` | GET | 会话数据查询 |

---

*注：更多文档正在持续更新中*
