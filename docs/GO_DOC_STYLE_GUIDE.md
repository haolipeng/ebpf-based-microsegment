# Go 文档注释规范

## 核心原则

1. **每个导出的标识符都应该有注释**（大写开头的）
2. **注释以被注释对象的名称开头**
3. **使用完整的句子**
4. **首字母大写，以句点结尾**
5. **包注释放在 package 声明前**

## 示例

### 包注释

```go
// Package api provides RESTful HTTP API endpoints for managing
// the eBPF-based microsegmentation system.
//
// The API server exposes the following functionality:
//   - Policy management (CRUD operations)
//   - Real-time statistics queries
//   - Health checks and system status
//   - Configuration management
//
// Example usage:
//
//   cfg := api.DefaultConfig()
//   server, err := api.NewAPIServer(cfg, dataPlane, policyMgr)
//   if err != nil {
//       log.Fatal(err)
//   }
//   server.Start()
//
package api
```

### 函数注释

```go
// NewAPIServer creates and initializes a new API server instance.
// It sets up the Gin router, configures middleware, and registers all routes.
//
// If cfg is nil, default configuration is used. The data plane (dp) and
// policy manager (pm) must not be nil.
//
// Returns an error if server initialization fails.
func NewAPIServer(cfg *Config, dp *dataplane.DataPlane, pm *policy.PolicyManager) (*Server, error)
```

### 类型注释

```go
// Server represents the HTTP API server that provides RESTful endpoints
// for the microsegmentation system. It integrates with the eBPF data plane
// and policy manager to expose control plane functionality.
type Server struct {
    config        *Config               // API server configuration
    dataPlane     *dataplane.DataPlane  // eBPF data plane interface
    policyManager *policy.PolicyManager // Policy manager
    httpServer    *http.Server          // HTTP server
    router        *gin.Engine           // Gin router
}
```

### 常量/变量注释

```go
// DefaultPort is the default HTTP port for the API server.
const DefaultPort = 8080

// ErrServerNotRunning is returned when operations are attempted
// on a server that hasn't been started.
var ErrServerNotRunning = errors.New("server not running")
```

## 格式化技巧

### 段落

空行分隔段落：

```go
// Start starts the HTTP server in a background goroutine.
//
// The server listens on the configured host and port.
// This method returns immediately; the server runs asynchronously.
```

### 列表

使用缩进表示列表：

```go
// Supported operations:
//   - Create policy
//   - Update policy
//   - Delete policy
//   - List policies
```

### 代码示例

缩进 4 个空格或 1 个 tab：

```go
// Example:
//
//   server, _ := NewAPIServer(cfg, dp, pm)
//   server.Start()
//   defer server.Stop()
```

### 链接

使用 URL 或 Go 标识符：

```go
// See also: Server.Stop, Config.Port
// Documentation: https://pkg.go.dev/github.com/gin-gonic/gin
```

## 生成文档

```bash
# 查看包文档
go doc github.com/ebpf-microsegment/src/agent/pkg/api

# 查看特定类型
go doc api.Server

# 查看特定方法
go doc api.Server.Start

# 启动本地文档服务器
godoc -http=:6060
# 访问 http://localhost:6060/pkg/
```

## 避免的错误

❌ **不要这样写：**

```go
// this function starts the server
func (s *Server) Start() error

// param: cfg - configuration
// return: server instance
func NewAPIServer(cfg *Config) *Server
```

✅ **应该这样写：**

```go
// Start starts the HTTP server and begins accepting requests.
func (s *Server) Start() error

// NewAPIServer creates a new API server with the given configuration.
func NewAPIServer(cfg *Config) *Server
```
