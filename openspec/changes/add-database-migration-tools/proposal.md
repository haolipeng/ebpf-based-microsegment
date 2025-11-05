# 提案: 添加数据库迁移工具

**变更 ID**: `add-database-migration-tools`
**提案日期**: 2025-11-04
**状态**: 提案中
**优先级**: P1
**预估工作量**: 2-3 天

---

## 📋 概述

提供从单体架构（SQLite）迁移到 Agent-Server 架构（PostgreSQL）的数据迁移工具,包括策略、工作负载和历史流量数据。

## 🎯 目标

1. **实现迁移工具** - 从 SQLite 读取数据并写入 PostgreSQL
2. **支持批量迁移** - 高效处理大量流量数据
3. **数据验证** - 迁移前后数据一致性检查
4. **回滚支持** - 迁移失败时能够回滚
5. **进度显示** - 实时显示迁移进度

## 🏗️ 核心设计

### 目录结构

```
src/tools/migrate/
├── main.go                 # 迁移工具入口
├── sqlite_reader.go        # SQLite 数据读取
├── postgres_writer.go      # PostgreSQL 数据写入
├── validator.go            # 数据验证
├── progress.go             # 进度显示
└── README.md
```

### 使用方式

```bash
# 基本用法
./migrate \
  --source sqlite:///var/lib/agent/flows.db \
  --target postgres://user:pass@localhost:5432/microsegment \
  --batch-size 1000

# 只迁移策略
./migrate --source ... --target ... --tables policies

# 验证模式（不写入）
./migrate --source ... --target ... --dry-run
```

### 迁移流程

```
1. 连接源数据库 (SQLite)
2. 连接目标数据库 (PostgreSQL)
3. 验证目标数据库schema
4. 读取并转换数据
   - policies 表
   - workloads 表
   - flows 表 (批量处理)
5. 写入目标数据库
6. 验证数据一致性
7. 生成迁移报告
```

## ✅ 验收标准

- [ ] 成功迁移策略数据
- [ ] 成功迁移工作负载数据
- [ ] 成功迁移历史流量数据
- [ ] 数据一致性验证通过
- [ ] 提供详细的迁移日志
- [ ] 支持断点续传

## 🔗 依赖

**前置依赖**: add-server-component

---

**提案人**: Claude Code
