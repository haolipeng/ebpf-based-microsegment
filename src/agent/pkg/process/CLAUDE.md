[上级索引](../CLAUDE.md) > **process**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# process

## 架构定位

进程监控器 | 输入: eBPF Ring Buffer 进程事件（exec、exit） | 输出: 进程信息缓存（PID -> 进程名/路径/命令行）

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| monitor.go | 进程监控器，从 Ring Buffer 读取进程事件 | `NewProcessMonitor()`, `Start()`, `Stop()`, `ReadEvents()` |
| cache.go | 进程缓存，维护 PID -> ProcessInfo 映射 | `NewProcessCache()`, `Add()`, `Get()`, `Delete()`, `Cleanup()` |
| types.go | 进程数据类型定义 | `ProcessInfo`, `ProcessEvent` |

## 核心功能

- **实时监控**: 监听 eBPF 上报的进程 exec/exit 事件
- **进程缓存**: 维护活跃进程信息（LRU 缓存，默认 20000 条）
- **TTL 清理**: 定期清理超时的进程缓存条目
- **路径解析**: 记录进程可执行文件路径和命令行参数
- **审计支持**: 为流事件关联进程信息

## 数据结构

进程信息包含：PID、PPID、进程名（16 字符）、可执行文件路径、命令行参数、启动时间戳、用户 ID

## 应用场景

- **流事件关联**: 将网络连接关联到发起进程
- **审计日志**: 记录哪个进程发起了被拒绝的连接
- **异常检测**: 识别非预期进程的网络行为
