# Docker镜像仓库使用指南

本文档介绍如何拉取和使用 eBPF 微分段系统的 Docker 镜像。

## 镜像仓库

所有 Docker 镜像都发布到 GitHub Container Registry (ghcr.io)。

### 仓库地址

- **Server 镜像**: `ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server`
- **Agent 镜像**: `ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-agent`

## 镜像标签策略

### 语义化版本标签

发布的版本会自动打上语义化版本标签:

```bash
# 完整版本号
ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:v1.2.3

# 主版本.次版本
ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:v1.2

# 主版本
ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:v1
```

### 分支标签

每个分支都会有对应的标签:

```bash
# master 分支 (同时也标记为 latest)
ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:master
ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:latest

# develop 分支
ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:develop
```

### Git SHA 标签

每次提交都会生成 SHA 标签:

```bash
# 格式: {branch}-{short-sha}
ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:master-a1b2c3d
```

## 拉取镜像

### 公开镜像拉取

如果镜像是公开的,可以直接拉取:

```bash
# 拉取最新版本
docker pull ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:latest

# 拉取特定版本
docker pull ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:v1.0.0
```

### 私有镜像拉取

如果镜像是私有的,需要先登录:

#### 1. 创建 Personal Access Token (PAT)

1. 访问 GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. 点击 "Generate new token" → "Generate new token (classic)"
3. 选择以下权限:
   - `read:packages` (下载容器镜像)
   - `write:packages` (上传容器镜像,可选)
4. 生成并保存 token

#### 2. 登录到 GitHub Container Registry

```bash
export CR_PAT=YOUR_GITHUB_TOKEN
echo $CR_PAT | docker login ghcr.io -u USERNAME --password-stdin
```

#### 3. 拉取镜像

```bash
docker pull ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:latest
```

## 运行容器

### Server 容器

#### 基础运行

```bash
docker run -d \
  --name microsegment-server \
  -p 8080:8080 \
  -p 9090:9090 \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_NAME=microsegment \
  -e DB_USER=microsegment_user \
  -e DB_PASSWORD=secret \
  ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:latest
```

#### 使用自定义配置

```bash
docker run -d \
  --name microsegment-server \
  -p 8080:8080 \
  -p 9090:9090 \
  -v /path/to/config.yaml:/etc/microsegment/config.yaml:ro \
  ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:latest
```

#### 查看日志

```bash
docker logs -f microsegment-server
```

#### 健康检查

```bash
# 使用 Docker 自带的健康检查
docker inspect --format='{{.State.Health.Status}}' microsegment-server

# 手动检查
curl http://localhost:8080/health
```

### Agent 容器

Agent 需要特权模式和主机网络访问来加载 eBPF 程序:

```bash
docker run -d \
  --name microsegment-agent \
  --privileged \
  --network host \
  --pid host \
  -v /sys/fs/bpf:/sys/fs/bpf:rw \
  -e SERVER_URL=http://server-host:8080 \
  ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-agent:latest
```

## Docker Compose 部署

推荐使用 Docker Compose 部署完整技术栈:

```bash
# 拉取 docker-compose.yml
curl -O https://raw.githubusercontent.com/haolipeng/ebpf-based-microsegment/master/deploy/docker-compose.yml

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

## Kubernetes 部署

在 Kubernetes 中使用镜像:

### 创建 Secret 用于私有镜像

```bash
kubectl create secret docker-registry ghcr-secret \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USERNAME \
  --docker-password=YOUR_GITHUB_TOKEN \
  --docker-email=YOUR_EMAIL
```

### 在 Pod 中引用镜像

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: microsegment-server
spec:
  containers:
  - name: server
    image: ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:latest
    imagePullPolicy: Always
  imagePullSecrets:
  - name: ghcr-secret
```

## 镜像信息

### 查看镜像元数据

```bash
# 查看镜像标签
docker inspect ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:latest | jq '.[].Config.Labels'

# 查看构建信息
docker inspect ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:latest | \
  jq '.[].Config.Labels | {version, commit, build_date}'
```

### 镜像大小

- **Server**: ~30-50 MB (Alpine 基础镜像)
- **Agent**: ~50-80 MB (包含 eBPF 程序和依赖)

## 常见问题

### 1. 拉取镜像失败: "unauthorized"

**原因**: 镜像是私有的,需要认证

**解决方法**:
```bash
# 使用 GitHub token 登录
echo $CR_PAT | docker login ghcr.io -u USERNAME --password-stdin
```

### 2. 容器启动失败: "permission denied"

**原因**: Server 容器以非 root 用户运行,可能是挂载卷的权限问题

**解决方法**:
```bash
# 确保挂载的目录有正确的权限
chown -R 1000:1000 /path/to/volume
```

### 3. 健康检查失败

**原因**: 容器内的健康检查端点不可访问

**解决方法**:
```bash
# 检查容器日志
docker logs microsegment-server

# 进入容器检查
docker exec -it microsegment-server sh
curl http://localhost:8080/health
```

### 4. Agent 无法加载 eBPF 程序

**原因**: 缺少必要的权限或内核版本过低

**解决方法**:
```bash
# 确保使用特权模式
docker run --privileged ...

# 检查内核版本 (需要 >= 5.10)
uname -r
```

## 镜像更新

### 自动更新 (不推荐生产环境)

```bash
# 使用 Watchtower 自动更新容器
docker run -d \
  --name watchtower \
  -v /var/run/docker.sock:/var/run/docker.sock \
  containrrr/watchtower \
  --interval 300 \
  microsegment-server
```

### 手动更新

```bash
# 1. 拉取最新镜像
docker pull ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:latest

# 2. 停止并删除旧容器
docker stop microsegment-server
docker rm microsegment-server

# 3. 启动新容器
docker run -d ... (使用新镜像)
```

## CI/CD 集成

### GitHub Actions 示例

```yaml
- name: Pull and run server
  run: |
    echo ${{ secrets.GITHUB_TOKEN }} | docker login ghcr.io -u ${{ github.actor }} --password-stdin
    docker pull ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:latest
    docker run -d ... ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:latest
```

### GitLab CI 示例

```yaml
deploy:
  script:
    - echo $CI_JOB_TOKEN | docker login ghcr.io -u $CI_REGISTRY_USER --password-stdin
    - docker pull ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:latest
    - docker run -d ... ghcr.io/haolipeng/ebpf-based-microsegment/microsegment-server:latest
```

## 支持

如有问题,请访问:
- GitHub Issues: https://github.com/haolipeng/ebpf-based-microsegment/issues
- 文档: https://github.com/haolipeng/ebpf-based-microsegment/tree/master/docs
