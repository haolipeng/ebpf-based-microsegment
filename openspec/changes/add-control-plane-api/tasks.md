# 实施任务：控制平面 API 服务

## 1. 项目设置和基础

- [ ] 1.1 将 Gin 框架添加到 `go.mod`
- [ ] 1.2 创建 `src/agent/pkg/api/` 目录结构
- [ ] 1.3 创建 `src/agent/pkg/api/server.go`（API 服务器设置）
- [ ] 1.4 创建 `src/agent/pkg/api/router.go`（路由定义）
- [ ] 1.5 为 API 设置添加配置结构
- [ ] 1.6 更新 `cmd/main.go` 以启动 API 服务器

## 2. 核心 API 服务器

- [ ] 2.1 实现 `NewAPIServer()` 构造函数
- [ ] 2.2 实现 `Start()` 方法（HTTP 服务器生命周期）
- [ ] 2.3 实现 `Stop()` 方法（优雅关闭）
- [ ] 2.4 添加中间件：Recovery（panic 处理）
- [ ] 2.5 添加中间件：Logger（请求日志）
- [ ] 2.6 添加中间件：CORS（跨源支持）
- [ ] 2.7 配置 HTTP 超时（read/write/idle）

## 3. 请求/响应模型

- [ ] 3.1 创建 `api/models/policy.go`（策略 DTOs）
- [ ] 3.2 创建 `api/models/statistics.go`（统计 DTOs）
- [ ] 3.3 创建 `api/models/config.go`（配置 DTOs）
- [ ] 3.4 创建 `api/models/error.go`（错误响应结构）
- [ ] 3.5 为所有结构添加 JSON 标签
- [ ] 3.6 添加验证标签（使用 `go-playground/validator`）

## 4. 健康和状态端点

- [ ] 4.1 创建 `api/handlers/health.go`
- [ ] 4.2 实现 `GET /api/v1/health`（简单健康检查）
- [ ] 4.3 实现 `GET /api/v1/status`（详细状态）
- [ ] 4.4 在响应中包含数据平面状态
- [ ] 4.5 为健康处理器添加单元测试

## 5. 策略管理端点

- [ ] 5.1 创建 `api/handlers/policy.go`
- [ ] 5.2 实现 `POST /api/v1/policies`（创建策略）
- [ ] 5.3 实现 `GET /api/v1/policies`（列出所有策略）
- [ ] 5.4 实现 `GET /api/v1/policies/:id`（获取特定策略）
- [ ] 5.5 实现 `PUT /api/v1/policies/:id`（更新策略）
- [ ] 5.6 实现 `DELETE /api/v1/policies/:id`（删除策略）
- [ ] 5.7 为所有端点添加输入验证
- [ ] 5.8 添加错误处理（400、404、409、500）

## 6. 线程安全策略访问

- [ ] 6.1 使用 `sync.RWMutex` 创建 `SafePolicyManager` 包装器
- [ ] 6.2 为查询操作实现读锁
- [ ] 6.3 为更新操作实现写锁
- [ ] 6.4 添加死锁检测（基于超时）
- [ ] 6.5 添加并发访问测试

## 7. 统计端点

- [ ] 7.1 创建 `api/handlers/statistics.go`
- [ ] 7.2 实现 `GET /api/v1/stats`（所有统计信息）
- [ ] 7.3 实现 `GET /api/v1/stats/packets`（数据包统计）
- [ ] 7.4 实现 `GET /api/v1/stats/sessions`（会话统计）
- [ ] 7.5 实现 `GET /api/v1/stats/policies`（策略统计）
- [ ] 7.6 添加响应缓存（1 秒 TTL）
- [ ] 7.7 为统计处理器添加单元测试

## 8. 配置端点

- [ ] 8.1 创建 `api/handlers/config.go`
- [ ] 8.2 实现 `GET /api/v1/config`（获取当前配置）
- [ ] 8.3 实现 `PUT /api/v1/config`（更新配置）
- [ ] 8.4 添加配置验证
- [ ] 8.5 持久化配置更改（可选，基于文件）

## 9. 错误处理和验证

- [ ] 9.1 创建统一的错误响应格式
- [ ] 9.2 添加验证中间件
- [ ] 9.3 将内部错误映射到 HTTP 状态码
- [ ] 9.4 为调试添加详细的错误消息
- [ ] 9.5 添加错误日志记录

## 10. 文档

- [ ] 10.1 安装 swaggo/swag 用于 OpenAPI 生成
- [ ] 10.2 为处理器添加 Swagger 注释
- [ ] 10.3 生成 OpenAPI 规范（`swagger.json`）
- [ ] 10.4 在 `/api/docs` 提供 Swagger UI
- [ ] 10.5 在 `docs/API.md` 中编写 API 使用示例
- [ ] 10.6 在 README 中记录身份验证（未来）

## 11. 与现有组件集成

- [ ] 11.1 更新 `DataPlane` 以公开线程安全方法
- [ ] 11.2 更新 `PolicyManager` 以进行并发访问
- [ ] 11.3 确保没有数据平面性能下降
- [ ] 11.4 添加指标收集（可选）

## 12. 测试

### 单元测试
- [ ] 12.1 使用模拟依赖项测试所有处理器函数
- [ ] 12.2 测试输入验证
- [ ] 12.3 测试错误处理路径
- [ ] 12.4 测试并发策略访问
- [ ] 12.5 实现 >80% 的代码覆盖率

### 集成测试
- [ ] 12.6 创建 `tests/api_integration_test.go`
- [ ] 12.7 测试完整的 HTTP 请求/响应周期
- [ ] 12.8 测试真实 eBPF maps 上的 CRUD 操作
- [ ] 12.9 测试并发 API 请求
- [ ] 12.10 测试优雅关闭

### 性能测试
- [ ] 12.11 策略 CRUD 延迟基准测试（< 10ms）
- [ ] 12.12 统计查询延迟基准测试（< 50ms）
- [ ] 12.13 使用 `hey` 进行负载测试（目标 1000 req/s）
- [ ] 12.14 验证无数据平面影响（< 1% CPU）

## 13. 配置管理

- [ ] 13.1 创建默认配置
- [ ] 13.2 支持环境变量
- [ ] 13.3 支持 YAML 配置文件
- [ ] 13.4 添加配置验证
- [ ] 13.5 记录所有配置选项

## 14. 部署和运维

- [ ] 14.1 使用 API 构建目标更新 `Makefile`
- [ ] 14.2 使用 API 用法更新 `README.md`
- [ ] 14.3 添加示例 `curl` 命令
- [ ] 14.4 创建 Postman/Insomnia 集合
- [ ] 14.5 更新 systemd 单元文件（如果适用）

## 15. 代码质量

- [ ] 15.1 对所有新代码运行 `go fmt`
- [ ] 15.2 运行 `golangci-lint`
- [ ] 15.3 添加代码注释
- [ ] 15.4 审查错误处理
- [ ] 15.5 检查资源泄漏

## Dependencies

**必需：**
- `github.com/gin-gonic/gin` - HTTP 框架
- `github.com/go-playground/validator/v10` - 输入验证
- `github.com/swaggo/swag` - OpenAPI 生成（可选）

**测试：**
- `github.com/stretchr/testify` - 测试断言
- `net/http/httptest` - HTTP 测试实用程序

## Validation Criteria

在将此变更标记为完成之前：

- ✅ 所有端点已实现并测试
- ✅ API 对所有操作的响应时间在 100ms 内
- ✅ 没有数据平面性能下降
- ✅ 优雅关闭正常工作
- ✅ 生成 OpenAPI 文档
- ✅ 集成测试通过
- ✅ 代码覆盖率 > 80%
- ✅ API 与数据平面一起运行无问题

## Estimated Time

- **设置和基础**：2 天
- **核心 API 和模型**：3 天
- **策略端点**：3 天
- **统计端点**：2 天
- **测试**：3 天
- **文档**：2 天

**总计**：~15 天（3 周）

