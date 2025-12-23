[上级索引](../CLAUDE.md) > **benchmark**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# benchmark

## 架构定位

提供性能基准测试工具，收集和分析策略操作、数据包处理的性能指标。
**输入**: 延迟样本切片、操作计数、时间戳
**输出**: 统计报告（Min/Max/Mean/P95/P99、吞吐量）

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| stats.go | 统计指标计算（百分位、标准差、吞吐量） | `CalculateStats()`, `Stats.String()` |
| report.go | 生成人类可读的性能报告 | `GenerateReport()`, `PrintReport()` |
| testdata.go | 生成测试数据和场景 | `GenerateTestData()` |

## 核心功能

- **百分位计算**: P50/P95/P99/P999
- **统计指标**: Min/Max/Mean/StdDev
- **吞吐量**: Operations per second
- **报告生成**: 格式化输出性能分析结果
