[上级索引](../CLAUDE.md) > **runtime**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# runtime

## 架构定位

容器运行时适配器 | 输入: 容器 ID、运行时 API 连接 | 输出: 容器标签、名称、镜像、网络信息

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| interface.go | 运行时检测器接口定义 | `RuntimeDetector` interface |
| docker.go | Docker 运行时适配器 | `NewDockerDetector()`, `GetContainerLabels()`, `WatchContainerEvents()` |
| containerd.go | Containerd 运行时适配器 | `NewContainerdDetector()`, `ListContainersWithLabels()` |

## 核心功能

- **多运行时支持**: Docker、Containerd（可扩展至 CRI-O）
- **标签提取**: 从容器元数据提取用户定义标签
- **事件监听**: 监听容器生命周期事件（create、start、stop、remove）
- **自动发现**: 新容器启动时自动注册为 Workload

## 事件类型

- **created**: 容器创建时触发
- **started**: 容器启动时触发
- **stopped**: 容器停止时触发
- **removed**: 容器删除时触发

## 应用场景

- **非 K8s 环境**: Docker Compose、Standalone 容器
- **混合环境**: K8s + Docker 共存
- **标签继承**: 容器标签映射到工作负载标签
