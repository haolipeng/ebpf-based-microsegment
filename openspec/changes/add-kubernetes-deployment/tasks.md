# 任务列表: Kubernetes 部署配置

## 阶段 1: 基础资源创建 (1 天)

### 1.1 创建 Namespace 和 RBAC
- [ ] 创建 `deploy/kubernetes/namespace.yaml`
- [ ] 定义 microsegment namespace
- [ ] 创建 `deploy/kubernetes/rbac.yaml`
- [ ] 创建 ServiceAccount `microsegment-agent`
- [ ] 创建 ClusterRole (访问 nodes, pods 信息)
- [ ] 创建 ClusterRoleBinding
- [ ] 验证: RBAC 配置正确

### 1.2 创建 ConfigMap 和 Secret
- [ ] 创建 `deploy/kubernetes/configmap.yaml`
- [ ] 添加 server.yaml 配置
- [ ] 添加 agent.yaml 配置
- [ ] 创建 `deploy/kubernetes/secret.yaml`
- [ ] 添加数据库密码 (base64 编码)
- [ ] 添加说明如何生成 base64
- [ ] 验证: ConfigMap 和 Secret 可被 Pod 挂载

## 阶段 2: PostgreSQL 部署 (0.5 天)

### 2.1 创建 PostgreSQL StatefulSet
- [ ] 创建 `deploy/kubernetes/postgres/statefulset.yaml`
- [ ] 配置 1 个副本
- [ ] 配置 volumeClaimTemplates (PVC)
- [ ] 配置环境变量 (POSTGRES_DB, USER, PASSWORD)
- [ ] 配置资源请求和限制
- [ ] 配置 livenessProbe 和 readinessProbe
- [ ] 验证: StatefulSet 正常运行

### 2.2 创建 PostgreSQL Service
- [ ] 创建 `deploy/kubernetes/postgres/service.yaml`
- [ ] 配置 ClusterIP Service
- [ ] 配置端口 5432
- [ ] 验证: Service 可解析和访问

## 阶段 3: Server Deployment (1 天)

### 3.1 创建 Server Deployment
- [ ] 创建 `deploy/kubernetes/server/deployment.yaml`
- [ ] 配置 2 个副本 (高可用)
- [ ] 配置镜像和版本标签
- [ ] 配置环境变量 (数据库连接信息)
- [ ] 配置 ConfigMap 挂载 (/etc/microsegment/)
- [ ] 配置 Secret 挂载 (环境变量方式)
- [ ] 配置 livenessProbe (httpGet /health)
- [ ] 配置 readinessProbe (httpGet /ready)
- [ ] 配置资源请求 (cpu: 500m, memory: 512Mi)
- [ ] 配置资源限制 (cpu: 1000m, memory: 1Gi)
- [ ] 配置 PodAntiAffinity (preferredDuringScheduling)
- [ ] 配置 RollingUpdate 策略
- [ ] 验证: Deployment 成功创建
- [ ] 验证: 2 个 Pod 运行在不同节点
- [ ] 验证: 健康检查正常工作

### 3.2 创建 Server Service
- [ ] 创建 `deploy/kubernetes/server/service.yaml`
- [ ] 配置 LoadBalancer 或 ClusterIP Service
- [ ] 配置端口 8080 (HTTP) 和 9090 (gRPC)
- [ ] 添加 selector 匹配 Server Pods
- [ ] 验证: Service 可访问 Server Pods

### 3.3 创建 Ingress (可选)
- [ ] 创建 `deploy/kubernetes/server/ingress.yaml`
- [ ] 配置域名和路径规则
- [ ] 配置后端 Service
- [ ] 配置 TLS (可选)
- [ ] 添加 annotations (根据 Ingress Controller)
- [ ] 验证: 通过 Ingress 可访问 Server

## 阶段 4: Agent DaemonSet (1 天)

### 4.1 创建 Agent DaemonSet
- [ ] 创建 `deploy/kubernetes/agent/daemonset.yaml`
- [ ] 配置 DaemonSet 基本信息
- [ ] 配置 serviceAccountName: microsegment-agent
- [ ] 配置 hostNetwork: true
- [ ] 配置 hostPID: true
- [ ] 配置容器镜像
- [ ] 配置 securityContext (privileged: true)
- [ ] 配置 capabilities (SYS_ADMIN, NET_ADMIN, BPF, PERFMON)
- [ ] 配置 /sys/fs/bpf hostPath 挂载
- [ ] 配置 mountPropagation: Bidirectional
- [ ] 配置 ConfigMap 挂载
- [ ] 配置环境变量 (SERVER_URL)
- [ ] 配置资源请求 (cpu: 200m, memory: 256Mi)
- [ ] 配置资源限制 (cpu: 500m, memory: 512Mi)
- [ ] 配置 nodeSelector (可选,选择特定节点)
- [ ] 配置 tolerations (容忍污点,如 master 节点)
- [ ] 验证: 每个节点运行一个 Agent Pod
- [ ] 验证: Agent 可正常加载 eBPF 程序
- [ ] 验证: Agent 可连接 Server

## 阶段 5: Kustomize 配置 (0.5 天)

### 5.1 创建 Kustomization
- [ ] 创建 `deploy/kubernetes/kustomization.yaml`
- [ ] 配置资源列表 (所有 yaml 文件)
- [ ] 配置 namespace
- [ ] 配置 commonLabels
- [ ] 配置 commonAnnotations
- [ ] 配置 namePrefix (可选)
- [ ] 配置 images (镜像替换)
- [ ] 验证: `kubectl kustomize` 生成正确的清单
- [ ] 验证: `kubectl apply -k` 成功部署

### 5.2 创建环境特定的 Overlay (可选)
- [ ] 创建 `deploy/kubernetes/overlays/dev/`
- [ ] 创建 dev kustomization.yaml
- [ ] 覆盖副本数、资源限制等
- [ ] 创建 `deploy/kubernetes/overlays/prod/`
- [ ] 创建 prod kustomization.yaml
- [ ] 验证: 不同环境配置正确

## 阶段 6: 部署脚本 (1 天)

### 6.1 创建 Kubernetes 部署脚本
- [ ] 创建 `deploy/scripts/deploy-k8s.sh`
- [ ] 实现检查 kubectl 环境
- [ ] 实现检查集群连接
- [ ] 实现创建 Namespace 步骤
- [ ] 实现应用 ConfigMap/Secret 步骤
- [ ] 实现部署 PostgreSQL 步骤
- [ ] 实现等待 PostgreSQL 就绪步骤 (kubectl wait)
- [ ] 实现部署 Server 步骤
- [ ] 实现等待 Server 就绪步骤
- [ ] 实现创建 Server Service 步骤
- [ ] 实现部署 Agent 步骤
- [ ] 实现验证所有资源步骤 (kubectl get all)
- [ ] 支持 --namespace 参数
- [ ] 支持 --kubeconfig 参数
- [ ] 添加错误处理和退出码
- [ ] 添加彩色输出和进度提示
- [ ] 验证: 脚本可一键部署到 K8s

### 6.2 创建健康检查脚本
- [ ] 创建 `deploy/scripts/health-check-k8s.sh`
- [ ] 检查 Namespace 存在
- [ ] 检查所有 Pods 状态 (Running/Ready)
- [ ] 检查 Server Deployment 副本数
- [ ] 检查 Agent DaemonSet 节点覆盖
- [ ] 检查 Services 端点
- [ ] 检查 Server API 可用性 (通过 Service)
- [ ] 返回正确的退出码
- [ ] 输出详细的检查结果
- [ ] 验证: 可准确检测系统健康状态

### 6.3 创建卸载脚本
- [ ] 创建 `deploy/scripts/undeploy-k8s.sh`
- [ ] 实现删除 Agent DaemonSet
- [ ] 实现删除 Server Deployment 和 Service
- [ ] 实现删除 PostgreSQL StatefulSet
- [ ] 实现删除 PVC (可选,带确认)
- [ ] 实现删除 ConfigMap 和 Secret
- [ ] 实现删除 RBAC 资源
- [ ] 实现删除 Namespace (可选,带确认)
- [ ] 验证: 脚本可完全清理部署

## 阶段 7: 文档编写 (1 天)

### 7.1 创建 Kubernetes 部署文档
- [ ] 创建 `deploy/kubernetes/README.md`
- [ ] 编写前置要求 (K8s 版本、kubectl、集群权限)
- [ ] 编写快速开始步骤
- [ ] 使用部署脚本部署
- [ ] 使用 kubectl 手工部署
- [ ] 编写配置说明
- [ ] ConfigMap 和 Secret 使用
- [ ] 镜像版本配置
- [ ] 资源限制调整
- [ ] 编写 RBAC 权限说明
- [ ] Agent 需要的权限及原因
- [ ] 如何查看权限
- [ ] 编写 Agent 特权说明
- [ ] privileged, hostNetwork, hostPID 原因
- [ ] 安全风险和缓解措施
- [ ] 编写服务访问方法
- [ ] 通过 Service 访问
- [ ] 通过 Ingress 访问
- [ ] 通过 Port Forward 访问
- [ ] 编写升级和回滚步骤
- [ ] 滚动更新命令
- [ ] 回滚命令
- [ ] 零停机更新策略
- [ ] 编写监控和日志
- [ ] 查看 Pod 日志
- [ ] Prometheus 集成
- [ ] 编写故障排查指南
- [ ] 常见问题及解决方法
- [ ] 调试技巧

### 7.2 创建配置定制指南
- [ ] 在 README 中添加定制章节
- [ ] 说明如何修改副本数
- [ ] 说明如何修改资源限制
- [ ] 说明如何使用不同的镜像
- [ ] 说明如何配置 Ingress
- [ ] 提供 Kustomize 使用示例

### 7.3 更新主 README
- [ ] 在主 README.md 中添加 K8s 部署章节
- [ ] 链接到详细的 K8s 部署文档
- [ ] 提供快速开始命令

## 阶段 8: 测试验证 (1 天)

### 8.1 资源清单测试
- [ ] 使用 kubectl --dry-run 验证清单语法
- [ ] 使用 kubeval 或 kube-score 验证最佳实践
- [ ] 在测试集群测试部署
- [ ] 验证所有资源创建成功

### 8.2 Server Deployment 测试
- [ ] 验证 2 个副本正常运行
- [ ] 验证副本分散到不同节点 (Anti-affinity)
- [ ] 验证健康检查正常工作
- [ ] 验证 Service 负载均衡
- [ ] 测试滚动更新
- [ ] 修改镜像版本
- [ ] 验证无中断更新
- [ ] 测试回滚
- [ ] 验证快速回滚

### 8.3 Agent DaemonSet 测试
- [ ] 验证每个节点运行一个 Agent
- [ ] 验证 Agent 具有正确的权限
- [ ] 验证 eBPF 程序加载成功
- [ ] 验证 Agent 可访问 Server
- [ ] 测试节点添加/删除
- [ ] 添加新节点,验证 Agent 自动部署
- [ ] 删除节点,验证 Agent 自动清理

### 8.4 配置管理测试
- [ ] 测试 ConfigMap 更新
- [ ] 修改配置
- [ ] 重启 Pods 应用配置
- [ ] 测试 Secret 更新
- [ ] 修改密钥
- [ ] 验证新密钥生效

### 8.5 高可用测试
- [ ] 测试 Server Pod 故障恢复
- [ ] 手动删除一个 Pod
- [ ] 验证自动重建
- [ ] 测试节点故障
- [ ] 模拟节点下线
- [ ] 验证 Pod 重新调度

### 8.6 性能和资源测试
- [ ] 测试资源限制
- [ ] 监控 CPU 和内存使用
- [ ] 测试资源不足情况
- [ ] 验证 OOMKill 保护

### 8.7 文档验证
- [ ] 让 K8s 管理员按文档执行部署
- [ ] 收集反馈并改进文档
- [ ] 确保所有命令可执行
- [ ] 确保所有步骤准确

## 阶段 9: 安全审查和优化 (0.5 天)

### 9.1 安全审查
- [ ] 审查 RBAC 权限最小化
- [ ] 确认 Agent 权限必要性
- [ ] 审查 Agent 特权容器配置
- [ ] 考虑使用 capabilities 替代 privileged
- [ ] 审查 Secret 管理
- [ ] 确保密钥加密存储
- [ ] 审查网络策略 (可选)
- [ ] 限制 Pod 间网络访问

### 9.2 性能优化
- [ ] 优化资源请求和限制
- [ ] 测试不同配置的性能
- [ ] 优化健康检查频率
- [ ] 平衡响应速度和开销
- [ ] 优化镜像拉取策略
- [ ] 使用 ImagePullPolicy: IfNotPresent

### 9.3 监控集成
- [ ] 添加 Prometheus annotations
- [ ] 验证 /metrics 端点可访问
- [ ] 创建 ServiceMonitor (Prometheus Operator)
- [ ] 验证指标收集正常

## 总计

- **总任务数**: 131 任务
- **预估工时**: 6 天
- **阶段数**: 9 个阶段

## 里程碑

1. **M1 (第 1.5 天)**: 基础资源和 PostgreSQL 完成
2. **M2 (第 2.5 天)**: Server Deployment 完成
3. **M3 (第 3.5 天)**: Agent DaemonSet 完成
4. **M4 (第 4.5 天)**: Kustomize 和脚本完成
5. **M5 (第 5.5 天)**: 文档完成
6. **M6 (第 6 天)**: 测试和安全审查完成,提案可归档
