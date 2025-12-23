[上级索引](../CLAUDE.md) > **labels**

> ⚠️ 若所属目录结构变化，请同步更新本文档

---

# labels

## 架构定位

标签验证和管理器 | 输入: 工作负载标签（来自 K8s/Docker/手动配置） | 输出: 验证后的标签、合并后的标签、自动标记的标签

## 文件清单

| 文件 | 职责 | 核心接口 |
|------|------|----------|
| validator.go | 标签键值验证器，检查格式和长度 | `ValidateLabelKey()`, `ValidateLabelValue()`, `ValidateLabels()` |
| merger.go | 标签合并器，处理多来源标签冲突 | `MergeLabels()`, `ResolvePriority()` |
| autotagger.go | 自动标记器，基于规则自动添加标签 | `AutoTag()`, `ApplyRules()` |
| types.go | 标签约束和规则定义 | `LabelConstraints`, `AutoTagRule` |

## 核心功能

- **格式验证**: 遵循 K8s 标签规范（字母数字、`-`、`_`、`.`）
- **长度检查**: Key ≤ 253 字符，Value ≤ 63 字符
- **冲突解决**: 多来源标签冲突时按优先级合并（K8s > Runtime > Manual）
- **自动标记**: 基于规则自动添加标签（如根据 IP 段打环境标签）

## 验证规则

- **标签 Key 格式**: `[prefix/]name`，prefix 为可选 DNS 子域（如 example.com），name 必须以字母数字开头和结尾
- **标签 Value 格式**: 可为空，或字母数字开头结尾，允许 `-`、`_`、`.`

## 应用场景

- **标签规范**: 确保所有工作负载标签符合规范
- **多来源合并**: 合并 K8s、Docker、手动配置的标签
- **自动打标**: 根据 IP 段、命名空间等自动添加环境标签
