# add-kubernetes-integration

## Summary
实现 Kubernetes 深度集成功能，使系统能够自动发现 Kubernetes 集群中的 Pod 和 Service，实时同步标签和元数据到工作负载系统，并支持基于 Kubernetes 原生标签的策略管理。

## Background
当前系统已支持 Docker 和 Containerd 容器运行时的标签提取，但缺乏对 Kubernetes 的深度集成。在 Kubernetes 环境中，Pod 和 Service 包含丰富的元数据（标签、注解、Namespace 等），这些信息对于实现细粒度的网络策略至关重要。

**现状**：
- ✅ 支持 Docker/Containerd 标签提取 ([runtime/docker.go](../../../src/agent/pkg/runtime/docker.go), [runtime/containerd.go](../../../src/agent/pkg/runtime/containerd.go))
- ✅ 基础的 K8s 元数据提取（PodName, PodNS）
- ✅ 标签合并器支持 K8s 标签映射 ([labels/merger.go](../../../src/agent/pkg/labels/merger.go))
- ❌ 缺少直接的 Kubernetes API 集成
- ❌ 无法实时监听 Pod/Service 变化
- ❌ 无法自动发现集群中的所有工作负载

**问题**：
1. 依赖容器运行时间接获取 K8s 元数据，信息不完整
2. 无法感知 Pod 生命周期变化（调度、重启、删除）
3. 无法利用 Kubernetes 原生的 Service 概念
4. 缺少对 NetworkPolicy 的原生支持

## Goals
1. **Kubernetes API 客户端**：实现可靠的 K8s API 客户端，支持 in-cluster 和 out-of-cluster 配置
2. **Pod/Service 实时监听**：使用 Watch API 实时监听 Pod 和 Service 变化
3. **标签自动同步**：将 K8s 标签和注解自动映射到工作负载标签
4. **Namespace 管理**：支持按 Namespace 过滤和管理工作负载
5. **Service 发现**：自动发现和同步 Kubernetes Service

## Non-Goals
- 不实现 Kubernetes NetworkPolicy 控制器（未来可扩展）
- 不替换现有的容器运行时集成（Docker/Containerd）
- 不实现 Kubernetes Operator 模式（保持为 DaemonSet Agent）
- 不支持多集群管理（限定单集群）

## Proposed Changes

### 1. Kubernetes API 客户端（新增 spec: `k8s-api-client`）
- 使用 `client-go` 库创建 Kubernetes 客户端
- 支持两种配置模式：
  - In-cluster：自动使用 ServiceAccount token 和证书
  - Out-of-cluster：从 `~/.kube/config` 或环境变量读取
- 实现健康检查和自动重连机制
- 提供统一的错误处理和日志记录

### 2. Pod/Service 监听和同步（新增 spec: `k8s-workload-sync`）
- 使用 Informer 模式监听 Pod 和 Service 资源
- 实现事件处理器：
  - Pod Add/Update/Delete → 同步到 Workload Manager
  - Service Add/Update/Delete → 同步到 Group Manager（可选）
- 标签映射规则：
  - `app.kubernetes.io/name` → `app`
  - `app.kubernetes.io/component` → `role`
  - `environment`/`env` → `env`
  - `topology.kubernetes.io/zone` → `loc`
- 支持 Namespace 过滤（配置白名单/黑名单）

### 3. Workload Manager 集成（修改现有 spec: `label-management`）
- 扩展 Workload 数据模型以存储 K8s 元数据
- 添加 `k8s` 字段：Namespace, PodName, ServiceName, UID
- 实现幂等的 Upsert 操作（基于 K8s UID）
- 优雅处理 Pod 重建（UID 变化但名称相同）

## Design Decisions

### 为什么使用 Informer 而非原生 Watch？
**选择**：使用 client-go 的 Informer 机制

**原因**：
- ✅ 内置本地缓存，减少 API Server 负载
- ✅ 自动处理 Watch 重连和 Resync
- ✅ 提供 ResourceVersion 管理
- ✅ 支持事件去重和排序

**替代方案**：原生 Watch API
- ❌ 需要手动处理重连和缓存
- ❌ 容易丢失事件
- ❌ 增加实现复杂度

### 如何处理 Pod IP 变化？
**选择**：基于 Kubernetes UID 作为工作负载的唯一标识

**原因**：
- ✅ UID 在 Pod 生命周期内不变
- ✅ Pod 重建后 UID 会变化，避免陈旧数据
- ✅ 可以追踪 Pod 的完整生命周期

**实现**：
```go
workloadID := fmt.Sprintf("k8s:%s:%s", pod.Namespace, pod.UID)
```

### Namespace 隔离策略
**选择**：支持可配置的 Namespace 过滤

**配置示例**：
```yaml
kubernetes:
  namespaces:
    include:
      - production
      - staging
    exclude:
      - kube-system
      - kube-public
```

## Dependencies
- `k8s.io/client-go` v0.31.0+
- `k8s.io/api` v0.31.0+
- `k8s.io/apimachinery` v0.31.0+
- 现有的 Workload Manager
- 现有的 Label Merger

## Risks and Mitigations

### 风险 1：API Server 负载
**影响**：大规模集群中频繁的 Watch 事件可能影响性能

**缓解措施**：
- 使用 Informer 的本地缓存
- 配置合理的 Resync 周期（默认 30 分钟）
- 支持 Namespace 过滤减少监听范围
- 实现 Rate Limiting

### 风险 2：RBAC 权限不足
**影响**：Agent 可能无法访问必要的 K8s 资源

**缓解措施**：
- 提供清晰的 RBAC 配置文档
- 实现权限检查和友好的错误提示
- 支持降级运行（无 K8s 集成）

### 风险 3：网络故障导致同步中断
**影响**：可能错过 Pod 变化事件

**缓解措施**：
- Informer 自动重连和 Resync
- 启动时执行全量同步
- 记录同步状态到日志

## Testing Strategy
1. **单元测试**：
   - K8s Client 初始化和配置
   - Informer 事件处理逻辑
   - 标签映射规则

2. **集成测试**：
   - 使用 fake Clientset 模拟 K8s API
   - 验证 Pod Add/Update/Delete 流程
   - 验证 Namespace 过滤

3. **E2E 测试**（可选）：
   - 在本地 KinD/Minikube 集群测试
   - 验证完整的 Pod 生命周期同步

## Success Criteria
- [ ] K8s API Client 成功连接到集群（in-cluster 和 out-of-cluster）
- [ ] 能够监听和同步 Pod 的创建、更新、删除事件
- [ ] K8s 标签正确映射到工作负载标签
- [ ] 支持 Namespace 过滤
- [ ] 测试覆盖率 ≥ 70%
- [ ] 文档完整（配置说明、RBAC 示例、故障排查）

## Timeline
- **Week 1**: K8s API Client 实现和测试
- **Week 2**: Pod Informer 和事件处理
- **Week 3**: Service 监听和 Namespace 管理
- **Week 4**: 集成测试和文档

## Related Changes
- Depends on: `add-label-based-policy` (标签系统基础)
- Blocks: 未来的 NetworkPolicy 控制器
- Related: `add-kubernetes-deployment` (K8s 部署配置)

## References
- [Kubernetes client-go Documentation](https://github.com/kubernetes/client-go)
- [Informer Design Pattern](https://github.com/kubernetes/sample-controller)
- [RBAC Best Practices](https://kubernetes.io/docs/reference/access-authn-authz/rbac/)
