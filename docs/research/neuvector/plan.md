已分析的内容 ✅

  | 文档            | 覆盖内容                          |
  |---------------|-------------------------------|
  | Agent-dp 通信机制 | Unix Socket 通信、JSON/Binary 协议 |
  | Agent-dp 策略分发 | 策略下发完整流程、数据结构转换               |
  | FQDN 实现       | 域名过滤、DNS 拦截                   |
  | 网络拓扑图         | 流量可视化、图数据库                    |
  | dp 编译指南       | 数据平面依赖和编译                     |

  建议补充分析的组件 📋

  根据你的项目当前实现（已有 policy、flow、workload、groups 等模块）和微隔离目标，以下是建议优先级：

  🔴 高优先级（核心功能缺失）

  | 组件        | 位置                                 | 原因                                                               |
  |-----------|------------------------------------|------------------------------------------------------------------|
  | 容器/工作负载发现 | agent/container/, agent/workerlet/ | 你已有 workload 模块，但 NeuVector 的容器运行时集成更完整（Docker/containerd/CRI-O） |
  | 会话状态机     | dp/dpi/ 中的连接跟踪                     | TCP 状态追踪、会话超时处理，你的 session/ 模块可能需要参考                             |
  | DPI 深度包检测 | dp/dpi/dpi_packet.c                | 如果需要 L7 协议识别（HTTP/DNS/MySQL等）                                    |

  🟡 中优先级（增强功能）

  | 组件                | 位置                              | 原因                          |
  |-------------------|---------------------------------|-----------------------------|
  | 进程白名单             | agent/policy/process.go         | 进程级别的访问控制，你的 process/ 模块可参考 |
  | 学习模式 (Learn Mode) | agent/engine.go, controller/    | 自动学习流量模式并生成策略建议             |
  | 告警和事件             | controller/rest/notification.go | 安全事件生成和上报机制                 |

  🟢 低优先级（高级功能）

  | 组件      | 位置                   | 原因                              |
  |---------|----------------------|---------------------------------|
  | CRD 策略  | controller/resource/ | Kubernetes NetworkPolicy CRD 支持 |
  | 合规扫描    | agent/nvbench/       | CIS 基准检查（与微隔离关系不大）              |
  | WAF/DLP | dp/dpi/dpi_dlp.c     | Web 应用防火墙和数据泄露防护                |

  ---
  具体建议

  1. 如果你的重点是 eBPF 数据平面

  建议深入研究：
  source-references/neuvector/dp/
  ├── dpi/dpi_packet.c      # 包解析和协议识别
  ├── dpi/dpi_session.c     # 会话管理
  └── dpi/dpi_policy.c      # 策略匹配（已在文档3中分析）

  2. 如果你的重点是 控制平面策略管理

  建议研究：
  source-references/neuvector/
  ├── agent/policy/network.go    # 网络策略计算
  ├── controller/ruleid.go       # 规则 ID 分配
  └── controller/graph.go        # 拓扑和策略关系

  3. 如果你想完善 工作负载发现

  建议研究：
  source-references/neuvector/agent/
  ├── container/             # 容器运行时抽象
  ├── workerlet/             # 工作负载管理
  └── pipe/                  # 网络命名空间处理