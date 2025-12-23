[上级索引](../CLAUDE.md) > **testutil**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# testutil

## 架构定位

测试工具包 | 输入: 测试用例需求 | 输出: 测试数据库容器、固定测试数据、断言助手

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| database.go | 测试数据库容器管理 | `SetupTestDB()`, `TestDB`, `Cleanup()` |
| fixtures.go | 测试数据工厂 | `NewTestPolicy()`, `NewTestFlow()`, `NewTestAgent()` |
| assertions.go | 自定义断言助手 | `AssertPolicyEqual()`, `AssertFlowEqual()` |

## 核心功能

- **Testcontainers**: 自动启动 PostgreSQL 测试容器
- **数据隔离**: 每个测试使用独立数据库
- **测试夹具**: 预定义的测试数据生成
- **清理机制**: 测试结束后自动清理容器

## 使用示例

```go
func TestSomething(t *testing.T) {
    db := testutil.SetupTestDB(t)
    defer db.Cleanup()

    // 使用 db.DB 执行测试...
}
```

## 依赖

- `testcontainers-go`: Docker 容器管理
- `timescale/timescaledb:latest-pg16`: 测试数据库镜像

