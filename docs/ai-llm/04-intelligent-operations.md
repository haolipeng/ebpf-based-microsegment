# 智能运维与故障排查

## 概述

运维和故障排查是微隔离产品使用过程中的常见挑战:
- **学习曲线陡峭**: 需要理解 eBPF、网络、策略等多个领域知识
- **故障诊断复杂**: 连通性问题涉及策略、网络、应用多个层面
- **文档分散**: 信息散落在各处,难以快速找到答案
- **响应时间长**: 需要专家介入才能解决问题

通过集成 AI/LLM 技术,可以提供 24/7 的智能运维助手,大幅提升问题解决效率。

---

## 场景 1: 智能问答助手 (RAG)

### 功能描述

基于 RAG (Retrieval-Augmented Generation) 技术,构建微隔离产品的智能知识库问答系统。

### 知识库内容

#### 1. 产品文档
- 安装部署指南
- API 参考文档
- 架构设计文档
- 最佳实践指南
- 故障排查手册

#### 2. 常见问题 (FAQ)
- 策略配置问题
- 性能优化建议
- 网络连通性问题
- 权限和安全问题

#### 3. 历史工单
- 已解决的技术支持工单
- 常见错误及解决方案
- 用户反馈和 Bug 报告

#### 4. 代码示例
- 策略配置示例
- API 调用示例
- 集成代码示例

### 对话示例

#### 示例 1: 基础问答

```
用户: "如何查看 eBPF 程序是否成功加载?"

AI 助手:
您可以通过以下几种方式检查 eBPF 程序是否成功加载:

**方法 1: 使用 bpftool (推荐)**
```bash
# 查看所有已加载的 eBPF 程序
sudo bpftool prog show

# 查看 TC eBPF 程序
sudo bpftool prog show type sched_cls

# 查看 XDP eBPF 程序
sudo bpftool prog show type xdp
```

**方法 2: 检查 TC filter**
```bash
# 查看网卡的 TC 配置
sudo tc filter show dev eth0 ingress
sudo tc filter show dev eth0 egress
```

**方法 3: 查看应用日志**
```bash
# 如果使用 systemd
sudo journalctl -u microsegment-agent -f

# 查看最近的加载日志
sudo journalctl -u microsegment-agent | grep "eBPF program loaded"
```

**方法 4: 使用 API 查询**
```bash
curl http://localhost:8080/api/v1/status
```

**预期输出示例**:
```json
{
  "status": "running",
  "dataplane": {
    "type": "tc",
    "loaded": true,
    "interface": "eth0",
    "programs": {
      "ingress": "attached",
      "egress": "attached"
    }
  }
}
```

**常见问题**:
- 如果看不到程序,检查是否有足够权限(需要 root 或 CAP_BPF)
- 如果加载失败,查看日志中的具体错误信息
- 如果使用容器,确保容器有 --privileged 或适当的 capabilities

相关文档:
- [eBPF 故障排查指南](docs/troubleshooting/ebpf.md)
- [数据平面架构](docs/architecture/dataplane.md)

还有其他问题吗? 🤔
```

#### 示例 2: 故障排查

```
用户: "我的策略配置了,但是流量还是不通,怎么办?"

AI 助手:
我来帮您系统地排查连通性问题。请按照以下步骤操作:

**🔍 步骤 1: 验证策略是否生效**

首先检查策略是否成功写入 eBPF Map:

```bash
# 查看策略数量
curl http://localhost:8080/api/v1/policies | jq 'length'

# 查看具体策略
curl http://localhost:8080/api/v1/policies/<policy-id> | jq
```

请提供:
- [ ] 您配置的策略 ID
- [ ] 上述命令的输出

---

**🔍 步骤 2: 检查流量是否到达 eBPF 程序**

```bash
# 查看统计数据
curl http://localhost:8080/api/v1/stats | jq

# 关注这些字段:
# - total_packets: 应该持续增长
# - policy_hits: 策略命中次数
# - policy_misses: 策略未命中次数
```

如果 total_packets 是 0,说明:
- eBPF 程序可能未正确加载
- 或流量没有经过该网卡

---

**🔍 步骤 3: 使用 Flow Collection 追踪**

```bash
# 查看实时流量(过去 5 分钟)
curl "http://localhost:8080/api/v1/flows?time_range=5m&source_ip=<源IP>&dest_ip=<目标IP>" | jq

# 检查是否有相关 Flow 记录
```

请提供:
- [ ] 源 IP 和端口
- [ ] 目标 IP 和端口
- [ ] 协议 (TCP/UDP/ICMP)

---

**🔍 步骤 4: 检查策略方向**

常见错误: 策略方向配置错误

问题场景:
- 您配置了 `direction: ingress`,但流量是主动发起的(应该用 `egress`)
- 或反之

**判断方法**:
- Ingress: 流量从外部进入本工作负载(被动接收)
- Egress: 流量从本工作负载主动发起
- Any: 双向(推荐新手使用)

**修复**:
```bash
# 修改策略方向
curl -X PUT http://localhost:8080/api/v1/policies/<policy-id> \
  -H "Content-Type: application/json" \
  -d '{"direction": "any"}'
```

---

**🔍 步骤 5: 使用连通性诊断工具**

我可以帮您自动诊断,请运行:

```bash
# 使用 AI 诊断工具
curl -X POST http://localhost:8080/api/v1/ai/diagnose-connectivity \
  -H "Content-Type: application/json" \
  -d '{
    "source_workload": "<源工作负载 ID>",
    "dest_ip": "<目标 IP>",
    "dest_port": <端口>,
    "protocol": "tcp"
  }' | jq
```

这个工具会自动:
- ✅ 检查策略配置
- ✅ 检查流量日志
- ✅ 检查 eBPF 统计
- ✅ 给出具体的修复建议

---

**🆘 如果以上步骤都无法解决**

请收集诊断信息并提交工单:

```bash
# 生成诊断报告
curl http://localhost:8080/api/v1/debug/report > diagnostic-report.json

# 或使用 CLI
microsegment-agent diagnose --output diagnostic-report.json
```

然后:
1. 在 GitHub Issues 创建问题: https://github.com/xxx/issues
2. 附上诊断报告
3. 我会在 24 小时内回复

---

请先尝试步骤 1-2,告诉我结果,我会继续帮您诊断 🔧
```

#### 示例 3: 最佳实践咨询

```
用户: "我有 100 个微服务,怎么高效地配置策略?"

AI 助手:
对于大规模微服务环境,我推荐以下策略管理最佳实践:

**🎯 策略 1: 使用标签驱动的策略 (Label-based)**

不要为每个服务单独写策略,而是使用标签分组:

```yaml
# ❌ 不推荐: 100 个微服务 = 100+ 条策略
from: {selector: "name=service-001"}
to: {selector: "name=database-001"}

# ✅ 推荐: 按角色分组
from: {selector: "role=api, tier=backend"}
to: {selector: "role=db"}
```

**优势**:
- 100 个微服务可能只需要 10-20 条策略
- 新服务自动继承策略(只要打了正确标签)
- 易于理解和维护

---

**🎯 策略 2: 分层架构模型**

定义清晰的分层:

```
Internet → DMZ Layer → Frontend Layer → API Layer → Data Layer
```

**标签体系**:
```yaml
workloads:
  - name: web-service
    labels:
      role: web
      tier: frontend
      layer: L3
      env: prod

  - name: api-service
    labels:
      role: api
      tier: backend
      layer: L4
      env: prod

  - name: db-service
    labels:
      role: db
      tier: data
      layer: L5
      env: prod
```

**策略示例**:
```yaml
# 允许跨层访问(仅限相邻层)
- from: {selector: "layer=L3"}
  to: {selector: "layer=L4"}
  action: allow

# 禁止跳层访问
- from: {selector: "layer=L3"}
  to: {selector: "layer=L5"}
  action: deny
```

---

**🎯 策略 3: 使用学习模式自动生成**

对于复杂的微服务依赖:

```bash
# 启动学习模式 14 天
curl -X POST http://localhost:8080/api/v1/ai/learning-sessions \
  -d '{
    "name": "microservices-prod",
    "workload_scope": "env=prod",
    "duration": 14
  }'
```

系统会:
- 自动观察所有服务间通信
- 识别稳定的依赖关系
- 生成最小权限策略

详见: [自动化策略生成文档](03-automated-policy-generation.md)

---

**🎯 策略 4: 渐进式部署**

不要一次性部署所有策略:

**第 1 周: 观察模式**
```yaml
action: log  # 只记录,不阻断
```

**第 2 周: 金丝雀部署**
```yaml
# 先在 10% 的服务上启用 deny
workload_scope: "env=prod, canary=true"
```

**第 3 周: 全量部署**
```yaml
# 所有生产服务
workload_scope: "env=prod"
```

---

**🎯 策略 5: 策略模板化**

为常见服务类型创建模板:

```yaml
# templates/web-service.yaml
metadata:
  name: web-service-template
  applies_to:
    - role=web
policies:
  - allow_inbound_http_https
  - allow_access_to_api
  - deny_direct_db_access
  - log_outbound_internet
```

使用时:
```bash
microsegment apply-template \
  --template web-service \
  --workload web-service-001
```

---

**🎯 策略 6: 自动化和 GitOps**

将策略代码化,使用 Git 管理:

```
repo/
├── policies/
│   ├── base/
│   │   └── default-deny.yaml
│   ├── frontend/
│   │   └── web-policies.yaml
│   ├── backend/
│   │   └── api-policies.yaml
│   └── data/
│       └── db-policies.yaml
├── .github/
│   └── workflows/
│       └── apply-policies.yml
```

**CI/CD 流程**:
1. 开发人员提交策略变更
2. GitHub Actions 自动验证语法
3. 在测试环境自动部署
4. 人工 Review 后部署到生产

---

**📊 预期效果**

采用以上最佳实践:
- 策略数量: 100 服务 → 20-30 条策略(减少 70%)
- 配置时间: 从 2 周 → 2 天(提升 80%)
- 维护成本: 降低 60%
- 错误率: 降低 90%

---

**🔧 实施建议**

1. **第 1 周**: 设计标签体系和分层架构
2. **第 2-3 周**: 运行学习模式
3. **第 4 周**: 部署自动生成的策略(观察模式)
4. **第 5-6 周**: 逐步切换到强制模式
5. **长期**: 建立 GitOps 流程

需要帮您生成具体的标签体系设计吗? 或者有其他问题? 🚀
```

### RAG 技术架构

```
┌─────────────────────────────────────┐
│         用户提问                     │
└───────────┬─────────────────────────┘
            │
┌───────────▼─────────────────────────┐
│    Query Embedding (向量化)          │
│    OpenAI text-embedding-3-small    │
└───────────┬─────────────────────────┘
            │
┌───────────▼─────────────────────────┐
│  Vector DB Search (相似度搜索)       │
│  Chroma / Milvus / Pinecone         │
│  - 检索 Top-K 相关文档 (K=5)         │
└───────────┬─────────────────────────┘
            │
┌───────────▼─────────────────────────┐
│  Rerank (重排序,可选)                │
│  Cohere Rerank / BGE Reranker       │
└───────────┬─────────────────────────┘
            │
┌───────────▼─────────────────────────┐
│  Prompt 构建                         │
│  System Prompt + Context + Query    │
└───────────┬─────────────────────────┘
            │
┌───────────▼─────────────────────────┐
│  LLM 生成回答                        │
│  OpenAI GPT-4 / Claude Sonnet       │
└───────────┬─────────────────────────┘
            │
┌───────────▼─────────────────────────┐
│  回答 + 引用来源                     │
└─────────────────────────────────────┘
```

### 实现代码示例

```go
// RAG Service
type RAGService struct {
    vectorDB  VectorDB
    llmClient LLMClient
    embedder  Embedder
}

func (s *RAGService) Answer(query string) (*Answer, error) {
    // 1. 查询向量化
    queryVector, err := s.embedder.Embed(query)
    if err != nil {
        return nil, err
    }

    // 2. 检索相关文档
    docs, err := s.vectorDB.Search(queryVector, TopK(5))
    if err != nil {
        return nil, err
    }

    // 3. 构建 Prompt
    prompt := s.buildPrompt(query, docs)

    // 4. LLM 生成回答
    answer, err := s.llmClient.Complete(prompt)
    if err != nil {
        return nil, err
    }

    return &Answer{
        Content:    answer,
        Sources:    docs,
        Confidence: s.calculateConfidence(docs),
    }, nil
}

func (s *RAGService) buildPrompt(query string, docs []Document) string {
    return fmt.Sprintf(`
你是一个微隔离产品的技术支持专家。基于以下文档回答用户问题。

文档:
%s

用户问题: %s

要求:
1. 优先使用文档中的信息回答
2. 如果文档中没有相关信息,明确告知用户
3. 提供具体的命令、代码示例或配置
4. 引用文档来源
5. 语气友好、专业

回答:
`, formatDocs(docs), query)
}
```

### API 接口

```go
// POST /api/v1/ai/ask
type AskRequest struct {
    Question    string `json:"question" binding:"required"`
    Context     string `json:"context"`     // 可选,当前页面上下文
    SessionID   string `json:"session_id"`  // 对话 ID
}

type AskResponse struct {
    Answer      string     `json:"answer"`
    Sources     []Source   `json:"sources"`
    RelatedDocs []Document `json:"related_docs"`
    Confidence  float64    `json:"confidence"`
}

type Source struct {
    Title    string `json:"title"`
    URL      string `json:"url"`
    Excerpt  string `json:"excerpt"`
}
```

---

## 场景 2: 日志智能分析

### 功能描述

自动分析错误日志、告警信息,提供故障排查建议和解决方案。

### 分析示例

#### 示例 1: eBPF 加载失败

```
用户: 点击"分析日志"按钮

系统检测到错误日志:
2025-01-13 10:15:23 ERROR Failed to load eBPF program: invalid BTF

AI 分析中... 🔍

---

📋 错误诊断报告

**错误类型**: eBPF 程序加载失败
**错误码**: BTF_INVALID
**严重级别**: 🔴 高 (数据平面无法工作)

---

**🔍 根因分析**

错误信息 "invalid BTF" 表示内核缺少 BTF (BPF Type Format) 信息。

BTF 是什么?
- eBPF CO-RE (Compile Once, Run Everywhere) 的基础
- 包含内核数据结构的类型信息
- 现代 Linux 内核(>= 5.10)通常自带

**为什么会出现这个错误?**

可能原因 1: 内核版本过低 (概率 70%) 🔴
- BTF 在内核 5.2+ 引入,但很多发行版默认不开启
- 在内核 5.10+ 才广泛可用

可能原因 2: BTF 未启用 (概率 25%) 🟡
- 某些精简版或定制内核禁用了 BTF
- 或缺少 linux-headers 包

可能原因 3: 文件损坏 (概率 5%) ⚪
- /sys/kernel/btf/vmlinux 文件损坏或缺失

---

**✅ 解决方案**

**方案 A: 检查并安装 BTF (推荐)**

步骤 1: 检查内核版本
```bash
uname -r
# 需要 >= 5.10
```

步骤 2: 检查 BTF 是否存在
```bash
ls -lh /sys/kernel/btf/vmlinux
# 应该看到一个文件(通常 4-6 MB)
```

如果不存在,安装 linux-headers:
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install linux-headers-$(uname -r)

# CentOS/RHEL
sudo yum install kernel-devel-$(uname -r)

# 重启后 BTF 会自动生成
sudo reboot
```

步骤 3: 重新启动 Agent
```bash
sudo systemctl restart microsegment-agent
```

---

**方案 B: 升级内核 (如果版本 < 5.10)**

```bash
# Ubuntu 20.04+
sudo apt install linux-generic-hwe-20.04

# 或使用 mainline 内核
sudo apt install linux-image-5.15.0-xxx-generic
sudo reboot
```

---

**方案 C: 使用非 CO-RE 模式 (临时方案)**

如果无法升级内核,可以使用兼容模式:

```bash
# 编辑配置
sudo vi /etc/microsegment/agent.conf

# 添加:
dataplane:
  mode: legacy  # 使用非 CO-RE 模式

# 重启
sudo systemctl restart microsegment-agent
```

**注意**: 非 CO-RE 模式:
- ✅ 可以在老内核上运行
- ❌ 性能略低(~5%)
- ❌ 需要为每个内核版本单独编译

---

**🔧 自动修复**

我可以帮您自动执行方案 A:

[ ] 自动检测并安装 BTF
[ ] 重启 Agent 服务
[ ] 验证 eBPF 程序加载成功

是否执行自动修复? [执行/手动修复/查看更多信息]

---

**📚 相关文档**
- [eBPF 环境要求](docs/requirements.md#ebpf)
- [内核升级指南](docs/installation/kernel-upgrade.md)
- [BTF 故障排查](docs/troubleshooting/btf.md)

---

**💬 还需要帮助?**
- 在工单中提供 `microsegment-agent diagnose` 输出
- 或加入 Slack 社区: https://slack.example.com
```

#### 示例 2: 策略语法错误

```
日志:
2025-01-13 14:30:15 ERROR Failed to create policy: invalid selector syntax

AI 分析:

📋 错误诊断报告

**错误类型**: 策略配置语法错误
**影响范围**: 策略未生效
**严重级别**: 🟡 中等

---

**🔍 根因分析**

检查最近的策略创建请求:
```json
{
  "from": {"selector": "role=web and env=prod"},  // ❌ 错误
  "to": {"selector": "role=db"},
  "action": "allow"
}
```

**问题**: 选择器语法错误

错误: `role=web and env=prod`
- 使用了 "and",但正确的语法是 ","(逗号)

正确: `role=web,env=prod`

---

**✅ 修复方案**

方案 1: 手动修复
```json
{
  "from": {"selector": "role=web,env=prod"},  // ✅ 正确
  "to": {"selector": "role=db"},
  "action": "allow"
}
```

方案 2: 使用 AI 自动修复
```
我已经为您修复了语法:
- 将 "and" 替换为 ","
- 验证选择器语法正确

是否应用修复后的策略? [应用/查看详情/取消]
```

---

**📖 选择器语法参考**

正确语法:
```yaml
role=web                      # 精确匹配
role=web,env=prod            # 多个条件(AND)
env in (prod, staging)       # IN 操作符
role!=test                   # 不等于
location exists              # 键存在
```

错误语法:
```yaml
role=web and env=prod        # ❌ 不支持 "and" 关键字
role=web or role=api         # ❌ 不支持 "or" (用 "in" 代替)
role="web"                   # ❌ 不需要引号(除非值包含特殊字符)
```

更多示例: [标签选择器文档](docs/concepts/labels.md)
```

---

## 场景 3: 连通性问题自动诊断

### 功能描述

用户报告"服务 A 无法访问服务 B",系统自动执行全面诊断并提供修复建议。

### 诊断示例

```
用户: "Web 服务无法连接到 Redis,怎么回事?"

AI: 我来帮您诊断。请提供以下信息:
- Web 服务的工作负载 ID 或 IP: _______
- Redis 的 IP 和端口: _______

---

用户输入:
- Web: web-frontend-1 (10.0.1.5)
- Redis: 10.0.2.10:6379

---

🔍 正在执行自动诊断...

═══════════════════════════════════════
📊 连通性诊断报告
═══════════════════════════════════════

**源**: web-frontend-1 (10.0.1.5)
**目标**: Redis (10.0.2.10:6379)
**协议**: TCP
**诊断时间**: 2025-01-13 15:45:00

---

**阶段 1: 基础信息检查** ✅

✅ 源工作负载存在
   - ID: web-frontend-1
   - Labels: role=web, app=myapp, env=prod
   - Status: Running
   - Node: node-1

✅ 目标 IP 可达
   - Ping: 0.5ms (网络层正常)
   - 目标工作负载: redis-primary
   - Labels: role=cache, app=myapp, env=prod

---

**阶段 2: 策略检查** ❌ 发现问题!

检查是否有允许该流量的策略...

❌ **未找到匹配的 ALLOW 策略**

搜索范围:
- From: role=web, app=myapp, env=prod
- To: role=cache, app=myapp, env=prod
- Port: 6379
- Protocol: TCP

现有策略:
1. policy-001: role=web → role=api (port 8080) ✅
2. policy-002: role=api → role=cache (port 6379) ✅
3. policy-003: * → * (DENY, default) ❌ **这条策略阻止了您的流量!**

**问题**:
- 您配置了 API → Redis 的策略
- 但缺少 Web → Redis 的策略
- Default Deny 策略阻止了未明确允许的流量

---

**阶段 3: Flow 日志检查** ✅ 确认

查询过去 1 小时的 Flow 日志:

找到 23 条相关 Flow:
- Source: 10.0.1.5:随机端口
- Dest: 10.0.2.10:6379
- Action: **DENY** ❌
- Policy: policy-003 (Default Deny)
- Time: 最近一次 2 分钟前

**确认**: 流量确实被策略阻断

---

**阶段 4: 架构分析** ⚠️ 发现架构问题

🤔 LLM 分析:

**潜在问题**: Web 服务直接访问 Redis 可能不是最佳架构

标准的三层架构应该是:
```
Web → API → Redis
```

而您当前的架构是:
```
Web → Redis (直接访问)
```

**风险**:
1. 缓存密钥管理分散(Web 和 API 都访问 Redis)
2. 增加 Redis 连接数压力
3. 缓存一致性难以保证

**建议**:
- 如果这是临时需求,可以添加策略允许
- 如果是长期方案,建议重构为通过 API 访问

---

**阶段 5: 解决方案** ✅

**方案 A: 添加策略允许 (快速修复)**

我可以为您自动生成策略:
```yaml
id: policy-auto-web-to-redis
name: Allow Web to Redis
from:
  selector: role=web,app=myapp
to:
  selector: role=cache,app=myapp
ports:
  - protocol: tcp
    port: 6379
action: allow
priority: 100
```

预计影响:
- ✅ 立即恢复 Web → Redis 连通性
- ⚠️  但仍存在架构问题

**方案 B: 架构重构 (推荐)**

重构应用代码:
1. Web 服务不再直接访问 Redis
2. 在 API 层实现缓存读写接口
3. Web 通过 API 间接使用缓存

预计工作量: 2-3 个开发日
预计收益:
- 更清晰的架构边界
- 更易维护
- 更好的可扩展性

**方案 C: 混合方案 (渐进式)**

第 1 周: 应用方案 A,恢复服务
第 2-3 周: 实施方案 B,逐步迁移

---

**您的选择**: [方案 A / 方案 B / 方案 C]

如果选择方案 A,我可以立即部署策略 ⚡
```

**用户选择**: 方案 A

```
AI: 正在部署策略...

✅ 策略部署成功!

策略 ID: policy-auto-web-to-redis
部署时间: 2025-01-13 15:46:15

---

🔄 验证连通性...

测试: web-frontend-1 → 10.0.2.10:6379

[ ████████████ ] 测试中...

✅ 连通性恢复!

- TCP 连接: 成功 (延迟 1.2ms)
- Redis PING: PONG ✅
- 策略命中: policy-auto-web-to-redis

---

📊 后续监控

我会持续监控该连接:
- 如果 24 小时内无问题,此工单将自动关闭
- 如果检测到新问题,会立即通知您

您可以查看实时流量:
https://console.example.com/flows?filter=web-to-redis

---

💡 别忘了:
考虑实施方案 B (架构重构),以获得长期收益。

需要帮助制定重构计划吗? [需要/不需要]
```

### 诊断 API

```go
// POST /api/v1/ai/diagnose-connectivity
type DiagnoseConnectivityRequest struct {
    SourceWorkloadID string `json:"source_workload_id"`
    DestIP           string `json:"dest_ip"`
    DestPort         int    `json:"dest_port"`
    Protocol         string `json:"protocol"` // tcp/udp
}

type DiagnoseConnectivityResponse struct {
    Summary        string                `json:"summary"`
    Stages         []DiagnosticStage     `json:"stages"`
    RootCause      string                `json:"root_cause"`
    Solutions      []Solution            `json:"solutions"`
    AutoFixAvailable bool                `json:"auto_fix_available"`
}

type DiagnosticStage struct {
    Name       string   `json:"name"`
    Status     string   `json:"status"` // pass/fail/warning
    Details    string   `json:"details"`
    Evidence   []string `json:"evidence"`
}

type Solution struct {
    ID             string   `json:"id"`
    Name           string   `json:"name"`
    Description    string   `json:"description"`
    Steps          []string `json:"steps"`
    EstimatedTime  string   `json:"estimated_time"`
    Pros           []string `json:"pros"`
    Cons           []string `json:"cons"`
    AutoApplicable bool     `json:"auto_applicable"`
}
```

---

## 场景 4: 合规报告自动生成

### 功能描述

根据流量数据、策略配置和合规标准,自动生成符合 PCI-DSS、HIPAA 等标准的合规报告。

### 报告示例

```
用户: "生成本月的 PCI-DSS 合规报告"

AI: 正在生成报告...

═══════════════════════════════════════
📋 PCI-DSS 合规检查报告
═══════════════════════════════════════

**报告期间**: 2025-01-01 至 2025-01-31
**组织**: Example Corp
**评估范围**: 生产环境 (env=prod)
**生成时间**: 2025-01-31 23:59:00

---

## 执行摘要

**总体合规状态**: 🟡 基本合规 (需改进)

**评分**: 78/100

**分类结果**:
- ✅ 完全合规: 8 项 (67%)
- ⚠️  部分合规: 3 项 (25%)
- ❌ 不合规: 1 项 (8%)

**关键发现**:
- 网络隔离: ✅ 已实施
- 访问控制: ✅ 已实施
- 日志记录: ⚠️  部分覆盖(需加强)
- 数据保护: ❌ 发现违规访问

---

## 详细检查结果

### 要求 1.2.1: 网络隔离

**状态**: ✅ 合规

**检查项**:
1. ✅ 生产环境与开发环境隔离
   - 策略: policy-isolation-prod-dev (DENY)
   - 过去 30 天阻断尝试: 45 次
   - 无误报

2. ✅ 数据库网络隔离
   - 仅允许应用层访问数据库
   - 禁止直接 SSH 访问
   - 策略: policy-db-protection

3. ✅ DMZ 区域隔离
   - 前端服务与后端服务隔离
   - 策略: policy-dmz-isolation

**证据**:
- 策略配置文件: [查看](reports/policies-2025-01.yaml)
- Flow 日志: [查看](reports/flows-2025-01.csv)
- 阻断记录: [查看](reports/denials-2025-01.log)

**LLM 评估**:
网络隔离措施完善,符合 PCI-DSS 1.2.1 要求。
建议: 保持现状,继续监控。

---

### 要求 2.2.2: 最小权限原则

**状态**: ✅ 合规

**检查项**:
1. ✅ 默认拒绝策略已实施
   - Default Deny 策略生效
   - 仅明确允许的流量被放行

2. ✅ 服务账户权限最小化
   - Web 服务不可访问数据库
   - API 服务仅访问必要的后端

3. ✅ 端口限制
   - 数据库仅开放必要端口 (5432)
   - 管理端口 (22, 3389) 已禁用

**统计**:
- 总策略数: 45
- ALLOW 策略: 32 (71%)
- DENY 策略: 13 (29%)
- 平均每个工作负载被 2.3 条策略保护

**LLM 评估**:
最小权限原则得到良好执行。
策略粒度适中,既保证安全又不影响业务。

---

### 要求 10.2.5: 访问日志记录

**状态**: ⚠️  部分合规 (需改进)

**检查项**:
1. ✅ 数据库访问日志完整
   - 所有数据库访问都有 Flow 记录
   - 记录包含: 源 IP, 用户, 时间, 操作

2. ⚠️  外部 API 调用日志不完整
   - 仅 65% 的外部调用被记录
   - 原因: 部分服务未配置 LOG 策略

3. ✅ 日志保留期限符合要求
   - 保留期: 90 天
   - 要求: >= 90 天 ✅

**需要改进**:
为所有外部 API 调用添加日志记录策略:
```yaml
- from: {selector: "env=prod"}
  to: {ip: "0.0.0.0/0"}
  action: log
  priority: 100
```

**LLM 评估**:
日志记录基本满足要求,但需要加强外部调用的审计。
建议: 2 周内完成改进。

---

### 要求 4.1: 敏感数据加密传输

**状态**: ❌ 不合规 (严重)

**检查项**:
1. ✅ 外部 HTTPS 连接已加密
   - 所有互联网流量使用 TLS

2. ❌ **内网敏感数据未加密** 🔴
   - 发现未加密的支付数据传输
   - payment-service ←→ order-service (HTTP, 端口 8080)
   - 涉及工作负载: 5 个
   - 过去 30 天流量: 12,450 次

**详细发现**:
```
Source: payment-service (10.0.1.15)
Dest: order-service (10.0.1.20:8080)
Protocol: HTTP (未加密) ❌
数据类型: 支付交易数据 (推测)
风险级别: 🔴 高
```

**影响**:
- 违反 PCI-DSS 4.1 要求
- 支付数据可能被中间人窃取
- 可能导致信用卡信息泄露

**修复方案**:
1. **立即措施** (本周内):
   - 为 payment-service 配置 TLS 证书
   - 修改 order-service 调用为 HTTPS
   - 预计工作量: 1-2 个开发日

2. **长期方案** (1 个月内):
   - 部署 Service Mesh (Istio/Linkerd)
   - 自动为所有服务间通信启用 mTLS
   - 预计工作量: 1 周

**紧急程度**: 🔴 高
**建议**: 立即启动修复流程

---

## 风险评估

### 高风险项 (需立即处理)

1. **未加密的支付数据传输** 🔴
   - 影响: 数据泄露风险
   - 处理: 2 周内修复

### 中风险项 (需关注)

1. **外部 API 调用日志不完整** 🟡
   - 影响: 审计追踪不完整
   - 处理: 1 个月内改进

### 低风险项 (持续改进)

1. **策略数量较多** ⚪
   - 影响: 维护成本略高
   - 建议: 考虑合并相似策略

---

## 改进建议

### 短期 (1 个月)
1. ✅ 修复支付数据加密问题
2. ✅ 完善外部 API 调用日志
3. ✅ 增加自动化合规检查

### 中期 (3 个月)
1. 部署 Service Mesh 实现全面 mTLS
2. 实施自动化策略生成和优化
3. 建立持续合规监控

### 长期 (6 个月)
1. 通过 PCI-DSS 正式审计
2. 获得 SOC 2 认证
3. 实施零信任架构

---

## 附录

### A. 策略清单
[完整策略列表 (45 条)](reports/appendix-a-policies.pdf)

### B. Flow 日志样本
[日志样本 (100 条)](reports/appendix-b-flow-samples.csv)

### C. 阻断事件详情
[阻断事件列表 (234 条)](reports/appendix-c-denials.csv)

### D. 架构拓扑图
[依赖关系图](reports/appendix-d-topology.png)

---

**报告生成**: 由 AI 引擎自动生成
**审核**: 需要人工审核并签字
**下次报告**: 2025-02-28

---

是否导出 PDF 版本? [导出 PDF/导出 Word/查看在线版]
```

### API 接口

```go
// POST /api/v1/ai/compliance-report
type GenerateComplianceReportRequest struct {
    Standard    string    `json:"standard"`  // pci-dss/hipaa/gdpr/soc2
    StartTime   time.Time `json:"start_time"`
    EndTime     time.Time `json:"end_time"`
    Scope       string    `json:"scope"`     // 标签选择器
    Format      string    `json:"format"`    // html/pdf/json
}

type ComplianceReport struct {
    Summary        ComplianceSummary     `json:"summary"`
    Requirements   []RequirementCheck    `json:"requirements"`
    Risks          []Risk                `json:"risks"`
    Recommendations []Recommendation     `json:"recommendations"`
    Appendices     []Appendix            `json:"appendices"`
}
```

---

## 实施建议

### 1. 知识库建设

**阶段 1**: 基础知识库 (1 个月)
- 产品文档
- API 参考
- 常见问题 FAQ
- 故障排查手册

**阶段 2**: 扩展知识库 (2-3 个月)
- 历史工单
- 社区讨论
- Blog 文章
- 视频教程

**阶段 3**: 持续更新 (长期)
- 新功能文档
- 用户反馈
- 最佳实践

### 2. 向量化策略

- 使用 OpenAI text-embedding-3-small (性价比高)
- 或使用开源模型 BGE-M3 (私有化部署)
- Chunk 大小: 512-1024 tokens
- 添加元数据: 文档类型、版本、标签

### 3. Prompt 工程

- System Prompt: 定义角色和行为
- Few-Shot Examples: 提供高质量示例
- Chain-of-Thought: 要求分步推理
- 输出格式: 使用 JSON Schema 约束

---

**下一篇**: [技术实现指南 →](05-technical-implementation.md)
