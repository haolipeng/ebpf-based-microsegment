# AI/LLM 集成技术实现指南

## 概述

本文档提供 AI/LLM 功能集成到微隔离产品的详细技术实现方案，包括架构设计、技术选型、代码示例和最佳实践。

---

## 系统架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                     用户界面层                                │
│  ┌────────────┐  ┌────────────┐  ┌──────────────────────┐  │
│  │ Web UI     │  │ CLI Tool   │  │ VS Code Extension    │  │
│  │ (React)    │  │            │  │                      │  │
│  └──────┬─────┘  └──────┬─────┘  └──────────┬───────────┘  │
└─────────┼────────────────┼────────────────────┼──────────────┘
          │                │                    │
          └────────────────┴────────────────────┘
                           │
          ┌────────────────▼───────────────────────┐
          │        API Gateway (Nginx/Kong)        │
          │     - Rate Limiting                    │
          │     - Authentication                   │
          │     - Load Balancing                   │
          └────────────────┬───────────────────────┘
                           │
     ┌─────────────────────┼─────────────────────┐
     │                     │                     │
┌────▼────────┐  ┌─────────▼─────────┐  ┌───────▼──────────┐
│ Agent API   │  │  Server API       │  │  AI Service      │
│ (Existing)  │  │  (Existing)       │  │  (New)           │
│             │  │                   │  │                  │
│ - Policies  │  │ - Multi-Agent     │  │ - LLM Gateway   │
│ - Flows     │  │ - Aggregation     │  │ - RAG Engine    │
│ - Stats     │  │ - WebSocket       │  │ - Prompt Mgmt   │
└─────┬───────┘  └─────────┬─────────┘  └────────┬─────────┘
      │                    │                      │
      │                    │                      │
┌─────┴────────────────────┴──────────────────────┴─────────┐
│                    数据层                                   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────┐ │
│  │ SQLite/  │  │ Vector DB│  │ Time     │  │ Cache     │ │
│  │ Postgres │  │ (Chroma) │  │ Series DB│  │ (Redis)   │ │
│  │          │  │          │  │(Optional)│  │           │ │
│  │- Flows   │  │- Docs    │  │- Metrics │  │- Sessions │ │
│  │- Policies│  │- Embeds  │  │- Trends  │  │- Results  │ │
│  └──────────┘  └──────────┘  └──────────┘  └───────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### AI Service 详细架构

```
┌─────────────────────────────────────────────────────────────┐
│                      AI Service                             │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐ │
│  │              HTTP/gRPC API Server                     │ │
│  │  - /ai/ask                  (问答)                    │ │
│  │  - /ai/policies/generate    (策略生成)                │ │
│  │  - /ai/traffic/analyze      (流量分析)                │ │
│  │  - /ai/diagnose             (故障诊断)                │ │
│  └───────────────────────┬───────────────────────────────┘ │
│                          │                                 │
│  ┌───────────────────────▼───────────────────────────────┐ │
│  │               Request Router                          │ │
│  │  - 路由到对应的 Handler                                │ │
│  │  - 请求验证和预处理                                    │ │
│  └───────────────────────┬───────────────────────────────┘ │
│                          │                                 │
│       ┌──────────────────┼──────────────────┐             │
│       │                  │                  │             │
│  ┌────▼─────┐  ┌─────────▼────────┐  ┌──────▼──────┐    │
│  │ RAG      │  │ Policy Generator │  │ Traffic     │    │
│  │ Handler  │  │                  │  │ Analyzer    │    │
│  └────┬─────┘  └─────────┬────────┘  └──────┬──────┘    │
│       │                  │                   │            │
│  ┌────▼──────────────────▼───────────────────▼──────┐   │
│  │              LLM Gateway                          │   │
│  │  - Provider 抽象 (OpenAI/Claude/Local)            │   │
│  │  - Retry & Fallback                               │   │
│  │  - Cost Tracking                                  │   │
│  │  - Response Caching                               │   │
│  └────┬──────────────────────────────────────────────┘   │
│       │                                                   │
│  ┌────▼──────────────────────────────────────────────┐   │
│  │         Supporting Services                        │   │
│  │                                                    │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────┐ │   │
│  │  │ Vector Store │  │ Prompt Mgmt  │  │ Context │ │   │
│  │  │ (RAG)        │  │ (Templates)  │  │ Builder │ │   │
│  │  └──────────────┘  └──────────────┘  └─────────┘ │   │
│  └────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## 技术选型

### 1. LLM 提供商

#### 选项 A: OpenAI (推荐用于 MVP)

**优势**:
- API 稳定可靠
- 文档完善
- GPT-4 质量优秀
- 生态成熟

**劣势**:
- 成本较高
- 数据隐私(需发送到外部)
- 有地区限制

**价格** (2025-01):
- GPT-4 Turbo: $0.01/1K input tokens, $0.03/1K output tokens
- GPT-3.5 Turbo: $0.0005/1K input tokens, $0.0015/1K output tokens
- Embedding: $0.00002/1K tokens

**使用场景**:
- 策略生成
- 流量分析
- 合规报告

#### 选项 B: Anthropic Claude

**优势**:
- 长上下文 (200K tokens)
- 更安全(Constitutional AI)
- 代码理解能力强

**劣势**:
- API 可用性略低于 OpenAI
- 生态较小

**价格**:
- Claude 3 Opus: $0.015/1K input, $0.075/1K output
- Claude 3 Sonnet: $0.003/1K input, $0.015/1K output

**使用场景**:
- 复杂策略分析
- 架构评审
- 长文档问答

#### 选项 C: 本地部署模型

**推荐模型**:
- LLaMA 3 70B (高质量)
- Mixtral 8x7B (性价比)
- Qwen 72B (中文友好)

**优势**:
- 数据隐私
- 无API调用成本
- 无地区限制

**劣势**:
- 需要 GPU 资源
- 部署和维护成本高
- 质量略低于商业模型

**硬件需求**:
- LLaMA 3 70B: 2x A100 (80GB) 或 4x A6000
- Mixtral 8x7B: 1x A100 (80GB)
- 量化版本: 1x RTX 4090 (24GB) 可运行 13B 模型

**使用场景**:
- 私有云/本地部署
- 高安全性要求
- 高并发场景(成本考虑)

### 2. 向量数据库

#### 选项 A: Chroma (推荐用于 MVP)

**优势**:
- 轻量级,易部署
- Python/Go SDK 完善
- 支持本地文件存储
- 开源免费

**安装**:
```bash
pip install chromadb
```

**使用场景**:
- 小规模(<100K文档)
- 快速 POC
- 单机部署

#### 选项 B: Milvus

**优势**:
- 高性能
- 支持分布式
- 功能丰富

**劣势**:
- 部署复杂
- 资源占用多

**使用场景**:
- 大规模(>100K文档)
- 分布式部署
- 高并发查询

#### 选项 C: Pinecone

**优势**:
- 全托管 SaaS
- 零运维
- 性能优秀

**劣势**:
- 有成本
- 数据在外部

**使用场景**:
- SaaS 版本产品
- 快速上线

### 3. 开发语言和框架

#### AI Service

**推荐**: Go (与现有代码库一致)

**优势**:
- 与 Agent/Server 代码共享
- 高性能
- 部署简单(单二进制文件)

**框架选择**:
```go
// HTTP 框架
github.com/gin-gonic/gin

// LLM 客户端
github.com/sashabaranov/go-openai

// 向量数据库
github.com/amikos-tech/chroma-go

// 配置管理
github.com/spf13/viper
```

**替代方案**: Python (如果需要更丰富的 AI 生态)

优势:
- LangChain / LlamaIndex 等框架成熟
- AI 库丰富
- 快速开发

劣势:
- 部署复杂(需要虚拟环境)
- 性能略低

### 4. 存储方案

#### 向量存储

- **开发/测试**: Chroma (本地文件)
- **生产**: Chroma (PostgreSQL 后端) 或 Milvus

#### 会话存储

- **短期**: Redis (对话上下文)
- **长期**: SQLite/Postgres (历史记录)

#### Prompt 模板

- **版本管理**: Git 仓库
- **运行时加载**: 文件系统或数据库

---

## 核心组件实现

### 1. LLM Gateway

```go
package llm

import (
    "context"
    "fmt"
    "time"

    "github.com/sashabaranov/go-openai"
)

// Provider 接口,支持多种 LLM
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
    Embed(ctx context.Context, text string) ([]float64, error)
}

type CompletionRequest struct {
    Model       string
    Messages    []Message
    Temperature float64
    MaxTokens   int
}

type Message struct {
    Role    string // system/user/assistant
    Content string
}

type CompletionResponse struct {
    Content      string
    TokensUsed   int
    Cost         float64
    FinishReason string
}

// OpenAI Provider 实现
type OpenAIProvider struct {
    client *openai.Client
    config OpenAIConfig
}

type OpenAIConfig struct {
    APIKey      string
    Model       string
    BaseURL     string // 可选,用于代理
    Timeout     time.Duration
}

func NewOpenAIProvider(config OpenAIConfig) *OpenAIProvider {
    client := openai.NewClient(config.APIKey)
    if config.BaseURL != "" {
        clientConfig := openai.DefaultConfig(config.APIKey)
        clientConfig.BaseURL = config.BaseURL
        client = openai.NewClientWithConfig(clientConfig)
    }

    return &OpenAIProvider{
        client: client,
        config: config,
    }
}

func (p *OpenAIProvider) Complete(
    ctx context.Context,
    req CompletionRequest,
) (*CompletionResponse, error) {
    // 构建 OpenAI 请求
    messages := make([]openai.ChatCompletionMessage, len(req.Messages))
    for i, msg := range req.Messages {
        messages[i] = openai.ChatCompletionMessage{
            Role:    msg.Role,
            Content: msg.Content,
        }
    }

    // 调用 OpenAI API
    resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model:       req.Model,
        Messages:    messages,
        Temperature: float32(req.Temperature),
        MaxTokens:   req.MaxTokens,
    })
    if err != nil {
        return nil, fmt.Errorf("openai api error: %w", err)
    }

    // 计算成本
    cost := p.calculateCost(req.Model, resp.Usage)

    return &CompletionResponse{
        Content:      resp.Choices[0].Message.Content,
        TokensUsed:   resp.Usage.TotalTokens,
        Cost:         cost,
        FinishReason: string(resp.Choices[0].FinishReason),
    }, nil
}

func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float64, error) {
    resp, err := p.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
        Model: openai.AdaEmbeddingV2,
        Input: []string{text},
    })
    if err != nil {
        return nil, err
    }

    embedding := make([]float64, len(resp.Data[0].Embedding))
    for i, v := range resp.Data[0].Embedding {
        embedding[i] = float64(v)
    }

    return embedding, nil
}

func (p *OpenAIProvider) calculateCost(model string, usage openai.Usage) float64 {
    // 价格表 (2025-01)
    prices := map[string]struct{ input, output float64 }{
        "gpt-4-turbo":    {0.01, 0.03},
        "gpt-3.5-turbo":  {0.0005, 0.0015},
    }

    price, ok := prices[model]
    if !ok {
        return 0 // 未知模型
    }

    inputCost := float64(usage.PromptTokens) / 1000 * price.input
    outputCost := float64(usage.CompletionTokens) / 1000 * price.output

    return inputCost + outputCost
}

// Gateway 包装多个 Provider,提供重试、回退等功能
type Gateway struct {
    providers []Provider
    cache     Cache
    metrics   Metrics
}

func (g *Gateway) Complete(
    ctx context.Context,
    req CompletionRequest,
) (*CompletionResponse, error) {
    // 1. 检查缓存
    if cached := g.cache.Get(req); cached != nil {
        return cached, nil
    }

    // 2. 尝试所有 Provider (带重试)
    var lastErr error
    for _, provider := range g.providers {
        resp, err := g.completewithRetry(ctx, provider, req)
        if err == nil {
            // 成功,缓存结果
            g.cache.Set(req, resp)
            g.metrics.RecordSuccess(provider, resp)
            return resp, nil
        }
        lastErr = err
    }

    return nil, fmt.Errorf("all providers failed: %w", lastErr)
}

func (g *Gateway) CompleteWithRetry(
    ctx context.Context,
    provider Provider,
    req CompletionRequest,
) (*CompletionResponse, error) {
    maxRetries := 3
    backoff := time.Second

    for i := 0; i < maxRetries; i++ {
        resp, err := provider.Complete(ctx, req)
        if err == nil {
            return resp, nil
        }

        // 指数退避
        if i < maxRetries-1 {
            time.Sleep(backoff)
            backoff *= 2
        }
    }

    return nil, fmt.Errorf("max retries exceeded")
}
```

### 2. RAG Engine

```go
package rag

import (
    "context"
    "fmt"

    chroma "github.com/amikos-tech/chroma-go"
)

type RAGEngine struct {
    vectorDB  *chroma.Client
    collection *chroma.Collection
    llm       llm.Provider
    embedder  llm.Provider
}

type RAGConfig struct {
    ChromaURL      string
    CollectionName string
    TopK           int
}

func NewRAGEngine(config RAGConfig, llmProvider llm.Provider) (*RAGEngine, error) {
    // 连接 Chroma
    client := chroma.NewClient(config.ChromaURL)

    // 获取或创建 collection
    collection, err := client.GetOrCreateCollection(config.CollectionName, nil)
    if err != nil {
        return nil, err
    }

    return &RAGEngine{
        vectorDB:   client,
        collection: collection,
        llm:        llmProvider,
        embedder:   llmProvider,
    }, nil
}

// 索引文档
func (e *RAGEngine) IndexDocuments(docs []Document) error {
    // 批量生成 embedding
    embeddings := make([][]float64, len(docs))
    for i, doc := range docs {
        embedding, err := e.embedder.Embed(context.Background(), doc.Content)
        if err != nil {
            return err
        }
        embeddings[i] = embedding
    }

    // 存储到 Chroma
    ids := make([]string, len(docs))
    documents := make([]string, len(docs))
    metadatas := make([]map[string]interface{}, len(docs))

    for i, doc := range docs {
        ids[i] = doc.ID
        documents[i] = doc.Content
        metadatas[i] = doc.Metadata
    }

    _, err := e.collection.Add(
        embeddings,
        metadatas,
        documents,
        ids,
    )

    return err
}

// 查询
func (e *RAGEngine) Query(ctx context.Context, question string, topK int) ([]Document, error) {
    // 1. 问题向量化
    queryEmbedding, err := e.embedder.Embed(ctx, question)
    if err != nil {
        return nil, err
    }

    // 2. 相似度搜索
    results, err := e.collection.Query(
        [][]float64{queryEmbedding},
        topK,
        nil, // where 过滤条件
        nil, // where_document
    )
    if err != nil {
        return nil, err
    }

    // 3. 转换结果
    docs := make([]Document, len(results.Documents[0]))
    for i := range docs {
        docs[i] = Document{
            ID:       results.IDs[0][i],
            Content:  results.Documents[0][i],
            Metadata: results.Metadatas[0][i],
            Distance: results.Distances[0][i],
        }
    }

    return docs, nil
}

// 回答问题
func (e *RAGEngine) Answer(ctx context.Context, question string) (*Answer, error) {
    // 1. 检索相关文档
    docs, err := e.Query(ctx, question, 5)
    if err != nil {
        return nil, err
    }

    // 2. 构建 Prompt
    prompt := e.buildPrompt(question, docs)

    // 3. LLM 生成回答
    resp, err := e.llm.Complete(ctx, llm.CompletionRequest{
        Model: "gpt-4-turbo",
        Messages: []llm.Message{
            {Role: "system", Content: systemPrompt},
            {Role: "user", Content: prompt},
        },
        Temperature: 0.7,
        MaxTokens:   1000,
    })
    if err != nil {
        return nil, err
    }

    return &Answer{
        Content:    resp.Content,
        Sources:    docs,
        Confidence: e.calculateConfidence(docs),
        Cost:       resp.Cost,
    }, nil
}

func (e *RAGEngine) buildPrompt(question string, docs []Document) string {
    context := ""
    for i, doc := range docs {
        context += fmt.Sprintf("\n[文档 %d]\n%s\n", i+1, doc.Content)
    }

    return fmt.Sprintf(`
基于以下文档回答问题。

文档:
%s

问题: %s

要求:
1. 优先使用文档中的信息
2. 如果文档中没有答案,明确告知
3. 提供具体示例和命令
4. 引用文档编号

回答:
`, context, question)
}

func (e *RAGEngine) calculateConfidence(docs []Document) float64 {
    if len(docs) == 0 {
        return 0.0
    }

    // 基于相似度距离计算置信度
    // 距离越小,置信度越高
    avgDistance := 0.0
    for _, doc := range docs {
        avgDistance += doc.Distance
    }
    avgDistance /= float64(len(docs))

    // 转换为置信度 (0-1)
    confidence := 1.0 - avgDistance
    if confidence < 0 {
        confidence = 0
    }

    return confidence
}

type Document struct {
    ID       string
    Content  string
    Metadata map[string]interface{}
    Distance float64 // 相似度距离
}

type Answer struct {
    Content    string
    Sources    []Document
    Confidence float64
    Cost       float64
}

const systemPrompt = `
你是一个微隔离产品的技术支持专家。你的职责是:
1. 基于文档回答用户的技术问题
2. 提供清晰、准确、可执行的解决方案
3. 使用中文回答
4. 友好、专业的语气

回答格式:
- 先直接回答问题
- 然后提供详细步骤或解释
- 如果适用,给出命令示例
- 最后引用文档来源
`
```

### 3. Prompt 管理

```go
package prompts

import (
    "bytes"
    "text/template"
)

type PromptManager struct {
    templates map[string]*template.Template
}

func NewPromptManager() *PromptManager {
    return &PromptManager{
        templates: make(map[string]*template.Template),
    }
}

// 加载 Prompt 模板
func (m *PromptManager) LoadTemplate(name string, content string) error {
    tmpl, err := template.New(name).Parse(content)
    if err != nil {
        return err
    }
    m.templates[name] = tmpl
    return nil
}

// 渲染 Prompt
func (m *PromptManager) Render(name string, data interface{}) (string, error) {
    tmpl, ok := m.templates[name]
    if !ok {
        return "", fmt.Errorf("template not found: %s", name)
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", err
    }

    return buf.String(), nil
}

// Prompt 模板示例
const policyGenerationPrompt = `
你是一个微隔离策略配置专家。用户会用自然语言描述安全需求，你需要生成策略配置。

用户需求: {{.UserInput}}

当前环境:
- 工作负载: {{range .Workloads}}
  * {{.Name}} ({{.Labels}})
{{end}}

- 现有策略数量: {{.PolicyCount}}

请生成策略配置 (JSON 格式)，确保:
1. 符合最小权限原则
2. 使用标签选择器而非硬编码 IP
3. 提供清晰的策略说明

输出格式:
{
  "policies": [...],
  "explanation": "为什么需要这些策略"
}
`

const trafficAnalysisPrompt = `
你是一个网络安全专家。请分析以下异常流量并提供诊断。

流量信息:
- 源: {{.SourceIP}} ({{.SourceWorkload}})
- 目标: {{.DestIP}}:{{.DestPort}}
- 协议: {{.Protocol}}
- 数据量: {{.Bytes}} bytes
- 时间: {{.Timestamp}}

历史基线:
- 平均连接数: {{.Baseline.AvgConnections}}/天
- 平均数据量: {{.Baseline.AvgBytes}}/天

上下文:
- 工作负载标签: {{.SourceLabels}}
- 策略: {{.AppliedPolicy}}
- 关联事件: {{.RelatedEvents}}

请分析:
1. 是否异常? 异常类型?
2. 可能的威胁场景
3. 推荐的应对措施

输出格式:
{
  "is_anomaly": true/false,
  "anomaly_types": [...],
  "threat_scenarios": [...],
  "recommendations": [...]
}
`
```

---

## API 实现

### HTTP API Server

```go
package api

import (
    "github.com/gin-gonic/gin"
)

type AIService struct {
    llm      llm.Gateway
    rag      *rag.RAGEngine
    prompts  *prompts.PromptManager
}

func NewAIService(
    llm llm.Gateway,
    rag *rag.RAGEngine,
    prompts *prompts.PromptManager,
) *AIService {
    return &AIService{
        llm:     llm,
        rag:     rag,
        prompts: prompts,
    }
}

func (s *AIService) SetupRoutes(r *gin.Engine) {
    ai := r.Group("/api/v1/ai")
    {
        // 问答
        ai.POST("/ask", s.handleAsk)

        // 策略生成
        ai.POST("/policies/generate", s.handlePolicyGenerate)
        ai.POST("/policies/analyze-conflicts", s.handleAnalyzeConflicts)

        // 流量分析
        ai.POST("/traffic/analyze-anomaly", s.handleAnalyzeAnomaly)
        ai.POST("/top-talkers/analyze", s.handleAnalyzeTopTalkers)

        // 诊断
        ai.POST("/diagnose-connectivity", s.handleDiagnoseConnectivity)

        // 学习模式
        ai.POST("/learning-sessions", s.handleCreateLearningSession)
        ai.GET("/learning-sessions/:id", s.handleGetLearningSession)
        ai.POST("/learning-sessions/:id/generate-policies", s.handleGeneratePoliciesFromLearning)

        // 合规报告
        ai.POST("/compliance-report", s.handleGenerateComplianceReport)
    }
}

// 问答 Handler
func (s *AIService) handleAsk(c *gin.Context) {
    var req AskRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // 调用 RAG
    answer, err := s.rag.Answer(c.Request.Context(), req.Question)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // 返回结果
    c.JSON(200, AskResponse{
        Answer:     answer.Content,
        Sources:    convertSources(answer.Sources),
        Confidence: answer.Confidence,
    })
}

// 策略生成 Handler
func (s *AIService) handlePolicyGenerate(c *gin.Context) {
    var req GeneratePolicyRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    // 1. 收集上下文
    context := s.buildPolicyContext(c, req)

    // 2. 渲染 Prompt
    prompt, err := s.prompts.Render("policy_generation", context)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // 3. 调用 LLM
    resp, err := s.llm.Complete(c.Request.Context(), llm.CompletionRequest{
        Model: "gpt-4-turbo",
        Messages: []llm.Message{
            {Role: "system", Content: systemPrompt},
            {Role: "user", Content: prompt},
        },
        Temperature: 0.7,
        MaxTokens:   2000,
    })
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    // 4. 解析 LLM 输出
    var result PolicyGenerationResult
    if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
        c.JSON(500, gin.H{"error": "failed to parse LLM output"})
        return
    }

    // 5. 验证策略
    warnings := []string{}
    for _, policy := range result.Policies {
        if err := s.validatePolicy(policy); err != nil {
            warnings = append(warnings, fmt.Sprintf("Policy %s: %s", policy.ID, err))
        }
    }

    // 6. 返回结果
    c.JSON(200, GeneratePolicyResponse{
        Policies:    result.Policies,
        Explanation: result.Explanation,
        Warnings:    warnings,
    })
}

func (s *AIService) buildPolicyContext(c *gin.Context, req GeneratePolicyRequest) map[string]interface{} {
    // 从 Agent API 获取工作负载信息
    workloads := s.fetchWorkloads()
    policies := s.fetchPolicies()

    return map[string]interface{}{
        "UserInput":   req.UserInput,
        "Workloads":   workloads,
        "PolicyCount": len(policies),
    }
}

func (s *AIService) validatePolicy(policy Policy) error {
    // 调用 Agent API 验证策略
    // 这里简化处理
    return nil
}

type AskRequest struct {
    Question  string `json:"question" binding:"required"`
    SessionID string `json:"session_id"`
}

type AskResponse struct {
    Answer     string   `json:"answer"`
    Sources    []Source `json:"sources"`
    Confidence float64  `json:"confidence"`
}

type GeneratePolicyRequest struct {
    UserInput string `json:"user_input" binding:"required"`
    DryRun    bool   `json:"dry_run"`
}

type GeneratePolicyResponse struct {
    Policies    []Policy `json:"policies"`
    Explanation string   `json:"explanation"`
    Warnings    []string `json:"warnings"`
}

type PolicyGenerationResult struct {
    Policies    []Policy `json:"policies"`
    Explanation string   `json:"explanation"`
}
```

---

## 部署方案

### 1. Docker Compose (开发/测试)

```yaml
# docker-compose-ai.yml
version: '3.8'

services:
  # AI Service
  ai-service:
    build: ./ai-service
    ports:
      - "8090:8090"
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - CHROMA_URL=http://chroma:8000
      - AGENT_API_URL=http://agent:8080
    depends_on:
      - chroma
      - redis

  # Chroma Vector DB
  chroma:
    image: chromadb/chroma:latest
    ports:
      - "8000:8000"
    volumes:
      - chroma-data:/chroma/data

  # Redis (缓存和会话)
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis-data:/data

  # (可选) Qdrant (Chroma 替代品)
  # qdrant:
  #   image: qdrant/qdrant:latest
  #   ports:
  #     - "6333:6333"

volumes:
  chroma-data:
  redis-data:
```

启动:
```bash
export OPENAI_API_KEY=sk-xxx
docker-compose -f docker-compose-ai.yml up
```

### 2. Kubernetes 部署

```yaml
# k8s/ai-service-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ai-service
  namespace: microsegment
spec:
  replicas: 2
  selector:
    matchLabels:
      app: ai-service
  template:
    metadata:
      labels:
        app: ai-service
    spec:
      containers:
      - name: ai-service
        image: microsegment/ai-service:v1.0.0
        ports:
        - containerPort: 8090
        env:
        - name: OPENAI_API_KEY
          valueFrom:
            secretKeyRef:
              name: ai-secrets
              key: openai-api-key
        - name: CHROMA_URL
          value: "http://chroma-service:8000"
        - name: REDIS_URL
          value: "redis://redis-service:6379"
        resources:
          requests:
            cpu: "500m"
            memory: "512Mi"
          limits:
            cpu: "2000m"
            memory: "2Gi"
---
apiVersion: v1
kind: Service
metadata:
  name: ai-service
  namespace: microsegment
spec:
  selector:
    app: ai-service
  ports:
  - port: 8090
    targetPort: 8090
---
apiVersion: v1
kind: Secret
metadata:
  name: ai-secrets
  namespace: microsegment
type: Opaque
stringData:
  openai-api-key: sk-xxx
```

部署:
```bash
kubectl apply -f k8s/ai-service-deployment.yaml
```

### 3. 本地 LLM 部署 (可选)

```yaml
# docker-compose-local-llm.yml
version: '3.8'

services:
  # Ollama (本地 LLM)
  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama-data:/root/.ollama
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]

  # AI Service (使用本地 LLM)
  ai-service:
    build: ./ai-service
    ports:
      - "8090:8090"
    environment:
      - LLM_PROVIDER=ollama
      - OLLAMA_URL=http://ollama:11434
      - LLM_MODEL=llama3:70b
    depends_on:
      - ollama

volumes:
  ollama-data:
```

下载模型:
```bash
docker exec -it ollama ollama pull llama3:70b
```

---

## 成本优化

### 1. 响应缓存

```go
type ResponseCache struct {
    redis *redis.Client
    ttl   time.Duration
}

func (c *ResponseCache) Get(req llm.CompletionRequest) *llm.CompletionResponse {
    key := c.cacheKey(req)
    val, err := c.redis.Get(context.Background(), key).Result()
    if err != nil {
        return nil
    }

    var resp llm.CompletionResponse
    json.Unmarshal([]byte(val), &resp)
    return &resp
}

func (c *ResponseCache) Set(req llm.CompletionRequest, resp *llm.CompletionResponse) {
    key := c.cacheKey(req)
    val, _ := json.Marshal(resp)
    c.redis.Set(context.Background(), key, val, c.ttl)
}

func (c *ResponseCache) cacheKey(req llm.CompletionRequest) string {
    // 基于请求内容生成唯一 key
    hash := sha256.Sum256([]byte(fmt.Sprintf("%+v", req)))
    return fmt.Sprintf("llm:cache:%x", hash)
}
```

**预期效果**:
- 缓存命中率: 30-50%
- 成本节约: 30-50%
- 响应时间: 从秒级降至毫秒级

### 2. 模型分级使用

```go
func (s *AIService) selectModel(taskComplexity string) string {
    switch taskComplexity {
    case "simple":
        return "gpt-3.5-turbo"  // 便宜
    case "medium":
        return "gpt-4-turbo"
    case "complex":
        return "gpt-4"          // 贵但质量高
    default:
        return "gpt-3.5-turbo"
    }
}
```

**任务分类**:
- Simple: 基础问答、语法检查
- Medium: 策略生成、日志分析
- Complex: 架构分析、合规报告

**成本对比**:
- GPT-3.5: $0.002/1K tokens
- GPT-4 Turbo: $0.04/1K tokens (20倍)

### 3. 输出限制

```go
func (s *AIService) Complete(req llm.CompletionRequest) {
    // 限制最大 token 数
    if req.MaxTokens == 0 {
        req.MaxTokens = 1000  // 默认限制
    }

    // 对于简单任务,进一步降低
    if req.TaskType == "simple" {
        req.MaxTokens = min(req.MaxTokens, 500)
    }
}
```

### 4. 成本监控

```go
type CostTracker struct {
    db *sql.DB
}

func (t *CostTracker) Track(userID string, cost float64, model string) {
    // 记录每次调用的成本
    t.db.Exec(`
        INSERT INTO llm_costs (user_id, model, cost, timestamp)
        VALUES (?, ?, ?, ?)
    `, userID, model, cost, time.Now())
}

func (t *CostTracker) GetMonthlyCost(userID string) float64 {
    var total float64
    t.db.QueryRow(`
        SELECT SUM(cost)
        FROM llm_costs
        WHERE user_id = ?
          AND timestamp >= datetime('now', '-30 days')
    `, userID).Scan(&total)
    return total
}

// 设置预算告警
func (t *CostTracker) CheckBudget(userID string, budget float64) error {
    current := t.GetMonthlyCost(userID)
    if current > budget {
        return fmt.Errorf("budget exceeded: $%.2f / $%.2f", current, budget)
    }
    return nil
}
```

---

## 监控和可观测性

### 1. 指标收集

```go
// Prometheus metrics
var (
    llmRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "llm_requests_total",
            Help: "Total number of LLM requests",
        },
        []string{"model", "provider", "status"},
    )

    llmRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "llm_request_duration_seconds",
            Help:    "LLM request duration in seconds",
            Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
        },
        []string{"model", "provider"},
    )

    llmCostTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "llm_cost_dollars_total",
            Help: "Total LLM cost in dollars",
        },
        []string{"model", "provider"},
    )

    llmTokensTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "llm_tokens_total",
            Help: "Total tokens used",
        },
        []string{"model", "type"}, // type: input/output
    )
)

func init() {
    prometheus.MustRegister(llmRequestsTotal)
    prometheus.MustRegister(llmRequestDuration)
    prometheus.MustRegister(llmCostTotal)
    prometheus.MustRegister(llmTokensTotal)
}

// 在每次调用后记录指标
func (g *Gateway) recordMetrics(provider string, req llm.CompletionRequest, resp *llm.CompletionResponse, duration time.Duration, err error) {
    status := "success"
    if err != nil {
        status = "error"
    }

    llmRequestsTotal.WithLabelValues(req.Model, provider, status).Inc()
    llmRequestDuration.WithLabelValues(req.Model, provider).Observe(duration.Seconds())

    if resp != nil {
        llmCostTotal.WithLabelValues(req.Model, provider).Add(resp.Cost)
        llmTokensTotal.WithLabelValues(req.Model, "input").Add(float64(resp.TokensUsed))
    }
}
```

### 2. Grafana Dashboard

```json
{
  "dashboard": {
    "title": "AI Service Monitoring",
    "panels": [
      {
        "title": "LLM Requests per Minute",
        "targets": [
          {
            "expr": "rate(llm_requests_total[1m])"
          }
        ]
      },
      {
        "title": "LLM Cost per Day",
        "targets": [
          {
            "expr": "sum(increase(llm_cost_dollars_total[1d]))"
          }
        ]
      },
      {
        "title": "P95 Latency",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, llm_request_duration_seconds)"
          }
        ]
      }
    ]
  }
}
```

---

## 安全和隐私

### 1. 数据脱敏

```go
func sanitizeForLLM(flow Flow) Flow {
    // 移除敏感信息
    sanitized := flow
    sanitized.SourceIP = maskIP(flow.SourceIP)
    sanitized.DestIP = maskIP(flow.DestIP)

    // 移除自定义标签中的敏感值
    if labels, ok := sanitized.Labels["secret"]; ok {
        delete(sanitized.Labels, "secret")
    }

    return sanitized
}

func maskIP(ip string) string {
    // 保留网段,隐藏主机部分
    // 10.0.1.5 -> 10.0.1.xxx
    parts := strings.Split(ip, ".")
    if len(parts) == 4 {
        return fmt.Sprintf("%s.%s.%s.xxx", parts[0], parts[1], parts[2])
    }
    return "xxx.xxx.xxx.xxx"
}
```

### 2. 审计日志

```go
type AuditLog struct {
    Timestamp   time.Time
    UserID      string
    Action      string  // "ask", "generate_policy", etc.
    Input       string  // 脱敏后的输入
    Output      string  // 脱敏后的输出
    Model       string
    TokensUsed  int
    Cost        float64
}

func (s *AIService) logAudit(log AuditLog) {
    // 写入审计数据库
    s.auditDB.Insert(log)
}
```

### 3. 访问控制

```go
func (s *AIService) checkPermission(userID string, action string) error {
    // 检查用户是否有权限使用 AI 功能
    if !s.rbac.HasPermission(userID, "ai."+action) {
        return fmt.Errorf("permission denied")
    }

    // 检查是否超出预算
    if err := s.costTracker.CheckBudget(userID, s.config.MonthlyBudget); err != nil {
        return err
    }

    return nil
}
```

---

## 测试策略

### 1. 单元测试

```go
func TestPolicyGenerator(t *testing.T) {
    // Mock LLM
    mockLLM := &MockLLMProvider{
        response: `{
            "policies": [{
                "from": {"selector": "role=web"},
                "to": {"selector": "role=api"},
                "action": "allow"
            }],
            "explanation": "Allow web to access API"
        }`,
    }

    generator := NewPolicyGenerator(mockLLM)

    result, err := generator.Generate(context.Background(), "允许 Web 访问 API")
    assert.NoError(t, err)
    assert.Len(t, result.Policies, 1)
    assert.Equal(t, "allow", result.Policies[0].Action)
}
```

### 2. 集成测试

```go
func TestRAGEndToEnd(t *testing.T) {
    // 启动测试环境
    chromaContainer := testcontainers.StartChroma(t)
    defer chromaContainer.Stop()

    // 创建 RAG Engine
    rag := NewRAGEngine(RAGConfig{
        ChromaURL: chromaContainer.URL(),
    }, openaiProvider)

    // 索引测试文档
    rag.IndexDocuments([]Document{
        {ID: "doc1", Content: "eBPF programs require kernel >= 5.10"},
    })

    // 测试查询
    answer, err := rag.Answer(context.Background(), "What kernel version is required?")
    assert.NoError(t, err)
    assert.Contains(t, answer.Content, "5.10")
}
```

### 3. LLM 输出验证

```go
func TestLLMOutputValidation(t *testing.T) {
    // 使用真实 LLM 生成策略
    resp, err := llm.Complete(context.Background(), llm.CompletionRequest{
        Model: "gpt-3.5-turbo",
        Messages: []llm.Message{
            {Role: "user", Content: "生成一条允许 Web 访问 API 的策略"},
        },
    })
    require.NoError(t, err)

    // 验证输出格式
    var result map[string]interface{}
    err = json.Unmarshal([]byte(resp.Content), &result)
    assert.NoError(t, err, "LLM should return valid JSON")

    // 验证必需字段
    policies := result["policies"].([]interface{})
    assert.NotEmpty(t, policies, "Should contain at least one policy")
}
```

---

## 总结

### 实施路线图

**第 1 个月**: MVP
- ✅ LLM Gateway
- ✅ 基础 RAG (文档问答)
- ✅ 简单策略生成
- ✅ 部署到测试环境

**第 2-3 个月**: 核心功能
- ✅ 流量异常检测
- ✅ 学习模式
- ✅ 故障诊断
- ✅ 成本优化

**第 4-6 个月**: 高级功能
- ✅ 合规报告
- ✅ What-If 分析
- ✅ 威胁情报集成
- ✅ 性能优化

### 技术债务管理

- 定期评审 Prompt 质量
- 收集用户反馈并迭代
- 监控 LLM 成本趋势
- 保持与最新 LLM 的兼容性

---

**文档版本**: v1.0
**最后更新**: 2025-01-13
**维护者**: AI/LLM 集成工作组
