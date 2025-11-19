# Kubernetes Deployment Guide

This guide describes how to deploy the eBPF Microsegment system to a Kubernetes cluster for testing purposes.

## Prerequisites

### Required Software
- **Kubernetes cluster** (version >= 1.20)
- **kubectl** CLI tool configured to access your cluster
- **Docker images** built for server and agent components

### Cluster Requirements
- At least 1 worker node (more nodes recommended for DaemonSet testing)
- Nodes must support eBPF (Linux kernel >= 5.10 recommended)
- Sufficient resources:
  - PostgreSQL: 100m CPU, 256Mi memory
  - Server: 200m CPU, 256Mi memory
  - Agent (per node): 200m CPU, 256Mi memory

### Build Docker Images

Before deploying, ensure Docker images are built and available to your cluster:

```bash
# Build images
cd /path/to/ebpf-based-microsegment
make docker-build

# Tag images (adjust registry as needed)
docker tag microsegment-server:latest your-registry/microsegment-server:latest
docker tag microsegment-agent:latest your-registry/microsegment-agent:latest

# Push to registry (if using a registry)
docker push your-registry/microsegment-server:latest
docker push your-registry/microsegment-agent:latest

# Or load images directly to cluster nodes (for local testing)
# Example for kind cluster:
kind load docker-image microsegment-server:latest
kind load docker-image microsegment-agent:latest
```

## Quick Start

### One-Command Deployment

The easiest way to deploy is using the deployment script:

```bash
cd /path/to/ebpf-based-microsegment
./deploy/scripts/deploy-k8s.sh
```

This script will:
1. Create the `microsegment` namespace
2. Apply RBAC configurations
3. Deploy PostgreSQL
4. Deploy Server
5. Deploy Agent DaemonSet
6. Wait for all components to be ready

### Manual Deployment

If you prefer to deploy manually:

```bash
# 1. Create namespace
kubectl apply -f deploy/kubernetes/namespace.yaml

# 2. Create RBAC resources
kubectl apply -f deploy/kubernetes/rbac.yaml

# 3. Create ConfigMap
kubectl apply -f deploy/kubernetes/configmap.yaml

# 4. Deploy PostgreSQL
kubectl apply -f deploy/kubernetes/postgres.yaml

# Wait for PostgreSQL to be ready
kubectl wait --for=condition=available --timeout=120s \
  deployment/microsegment-postgres -n microsegment

# 5. Deploy Server
kubectl apply -f deploy/kubernetes/server.yaml

# Wait for Server to be ready
kubectl wait --for=condition=available --timeout=120s \
  deployment/microsegment-server -n microsegment

# 6. Deploy Agent
kubectl apply -f deploy/kubernetes/agent.yaml

# Wait for Agent to be ready (adjust node count as needed)
kubectl rollout status daemonset/microsegment-agent -n microsegment
```

## Verify Deployment

### Check Resource Status

```bash
# Check all resources
kubectl get all -n microsegment

# Check pods
kubectl get pods -n microsegment

# Check services
kubectl get svc -n microsegment

# Check daemonset
kubectl get ds -n microsegment
```

### Check Logs

```bash
# Server logs
kubectl logs -f deployment/microsegment-server -n microsegment

# Agent logs (from a specific pod)
kubectl logs -f daemonset/microsegment-agent -n microsegment

# Agent logs (select specific pod)
POD_NAME=$(kubectl get pods -n microsegment -l app=microsegment-agent -o jsonpath='{.items[0].metadata.name}')
kubectl logs -f $POD_NAME -n microsegment

# PostgreSQL logs
kubectl logs -f deployment/microsegment-postgres -n microsegment
```

### Verify eBPF Programs

Check if the Agent successfully loaded eBPF programs:

```bash
# Get a shell into an agent pod
POD_NAME=$(kubectl get pods -n microsegment -l app=microsegment-agent -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it $POD_NAME -n microsegment -- /bin/sh

# Inside the pod, check loaded eBPF programs (if bpftool is available)
bpftool prog list
bpftool map list
```

### Test Connectivity

```bash
# Port forward to access the server
kubectl port-forward -n microsegment svc/microsegment-server 8080:8080 9090:9090

# In another terminal, test the API
curl http://localhost:8080/api/v1/agents
```

## Configuration

### Modify Image Names

If you're using a custom registry, edit the image names in:
- `deploy/kubernetes/server.yaml` - line with `image: microsegment-server:latest`
- `deploy/kubernetes/agent.yaml` - line with `image: microsegment-agent:latest`

Example:
```yaml
image: your-registry.com/microsegment-server:v1.0.0
```

### Modify Resource Limits

Resource requests and limits can be adjusted in the deployment files:

```yaml
resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 512Mi
```

### Modify Database Configuration

Database credentials can be changed in:
- `deploy/kubernetes/configmap.yaml` - server configuration
- `deploy/kubernetes/postgres.yaml` - PostgreSQL environment variables

**Note**: For production use, consider using Kubernetes Secrets instead of ConfigMap for sensitive data.

### Node Selection

To run Agent only on specific nodes, uncomment and modify the nodeSelector in `deploy/kubernetes/agent.yaml`:

```yaml
nodeSelector:
  kubernetes.io/os: linux
  node-type: worker
```

### Tolerations

To allow Agent to run on control plane nodes, uncomment the tolerations in `deploy/kubernetes/agent.yaml`:

```yaml
tolerations:
- key: node-role.kubernetes.io/control-plane
  operator: Exists
  effect: NoSchedule
```

## Undeployment

### One-Command Cleanup

```bash
./deploy/scripts/undeploy-k8s.sh
```

This script will remove all deployed resources and optionally delete the namespace.

### Manual Cleanup

```bash
# Delete in reverse order
kubectl delete -f deploy/kubernetes/agent.yaml
kubectl delete -f deploy/kubernetes/server.yaml
kubectl delete -f deploy/kubernetes/postgres.yaml
kubectl delete -f deploy/kubernetes/configmap.yaml
kubectl delete -f deploy/kubernetes/rbac.yaml
kubectl delete -f deploy/kubernetes/namespace.yaml
```

## Troubleshooting

### Pods Not Starting

**Check pod status:**
```bash
kubectl describe pod <pod-name> -n microsegment
```

**Common issues:**
- Image pull errors: Verify images are available to the cluster
- Insufficient resources: Check node capacity
- Init container failures: Check if dependent services are ready

### Agent Cannot Load eBPF Programs

**Symptoms:**
- Agent pods crash or restart frequently
- Logs show eBPF loading errors

**Solutions:**
1. Verify kernel version supports eBPF:
   ```bash
   kubectl exec -it <agent-pod> -n microsegment -- uname -r
   ```
   Minimum kernel version: 5.10

2. Check if privileged mode is enabled (should be `true` in agent.yaml)

3. Verify BPF filesystem is mounted:
   ```bash
   kubectl exec -it <agent-pod> -n microsegment -- mount | grep bpf
   ```

### Agent Cannot Connect to Server

**Check service DNS:**
```bash
kubectl exec -it <agent-pod> -n microsegment -- \
  nslookup microsegment-server.microsegment.svc.cluster.local
```

**Check service endpoints:**
```bash
kubectl get endpoints -n microsegment
```

### PostgreSQL Connection Issues

**Verify PostgreSQL is running:**
```bash
kubectl get pods -n microsegment -l app=microsegment-postgres
```

**Test connection from server pod:**
```bash
kubectl exec -it <server-pod> -n microsegment -- \
  nc -zv microsegment-postgres 5432
```

### Check RBAC Permissions

**Verify ServiceAccount:**
```bash
kubectl get serviceaccount microsegment-agent -n microsegment
```

**Check ClusterRole and Binding:**
```bash
kubectl get clusterrole microsegment-agent
kubectl get clusterrolebinding microsegment-agent
```

## Limitations (Test Environment)

This deployment is designed for **testing purposes only** and has the following limitations:

1. **No Data Persistence**: PostgreSQL uses `emptyDir` volume, data will be lost on pod restart
2. **Single Replica**: All services run with 1 replica, no high availability
3. **No Ingress**: Services are only accessible within the cluster or via port-forward
4. **Privileged Containers**: Agent requires privileged mode for eBPF
5. **No Monitoring**: No Prometheus/Grafana integration
6. **No Log Aggregation**: Logs are only accessible via kubectl

## Next Steps

For production deployment, consider:
- Use StatefulSet for PostgreSQL with persistent volumes
- Implement multi-replica deployments for high availability
- Add Ingress for external access
- Integrate with monitoring systems (Prometheus, Grafana)
- Use centralized logging (ELK, Loki)
- Implement proper secret management
- Add network policies for security
- Use Helm charts for easier configuration management

## Support

For issues or questions, please refer to the main project documentation or open an issue in the project repository.
