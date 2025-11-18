# Web UI 开发快速参考卡

> 📌 **快速查阅**: 核心任务、优先级、时间估算一目了然

---

## 🎯 核心目标

| 指标 | 目标值 |
|------|--------|
| **总工期** | 18-23 天（1 人全职）|
| **总任务数** | 80+ 个 |
| **当前完成度** | 70-75% |
| **代码覆盖率目标** | ≥ 70% |
| **性能目标** | LCP < 2.5s, FCP < 1.5s |

---

## 📋 四阶段概览

```
Phase 1 (P0)  →  Phase 2 (P1)  →  Phase 3 (P1)  →  Phase 4 (P2)
核心功能完善     用户体验增强     高级功能         长期优化
4-5 天           7-8 天           4-5 天           3-5 天
```

---

## ✅ Phase 1: 核心功能完善 (4-5 天)

### 🔐 用户认证系统 (3-4 天)
| Day | 任务 | 关键产出 |
|-----|------|----------|
| 1 | 认证基础架构 | Auth API 客户端、类型定义 |
| 2 | 认证状态管理 | Auth Store、Hooks、Token 刷新 |
| 3 | 登录页面和守卫 | 登录 UI、路由守卫、用户菜单 |
| 4 | 权限控制和测试 | 权限系统、403/401 页面 |

**快速检查**:
- [ ] 登录/登出正常 ✅
- [ ] Token 自动刷新 ✅
- [ ] 路由守卫拦截 ✅
- [ ] 权限控制生效 ✅

### 📊 策略统计 API (1 天)
- [ ] 确认后端 API: `GET /api/v1/policies/:id/stats`
- [ ] 更新 `policiesApi.getStats()`
- [ ] 集成到 `useVisualization.ts`
- [ ] Policy 页面显示统计

---

## ✅ Phase 2: 用户体验增强 (7-8 天)

### 🌙 暗黑模式 (2 天)
**Day 1**: 主题 Store + Ant Design ConfigProvider
**Day 2**: ECharts 主题 + CSS 变量 + 测试

**文件清单**:
```
src/store/themeStore.ts
src/themes/echarts-dark.ts
src/themes/echarts-light.ts
src/styles/variables.css
src/components/common/ThemeToggle.tsx
```

### 📄 报表导出 (3 天)
**Day 1**: PDF 导出 (jsPDF + jspdf-autotable)
**Day 2**: CSV 导出 + 图表导出
**Day 3**: 批量导出 + 模板定制

**依赖安装**:
```bash
npm install jspdf jspdf-autotable
npm install --save-dev @types/jspdf
```

### 🧪 测试覆盖 (3-4 天)
**Day 1**: 组件单元测试 (MetricCard, TopTalkersList, SessionDetail)
**Day 2**: Hook 和工具函数测试
**Day 3-4**: 集成测试 + E2E (可选)

**目标覆盖率**: Statements 70%, Branches 65%, Functions 70%

---

## ✅ Phase 3: 高级功能 (4-5 天)

### 📈 高级可视化 (2-3 天)
**新图表类型**:
- 流量热力图 (Heatmap)
- Top Talkers 气泡图 (Bubble)
- 策略命中漏斗图 (Funnel)
- 网络流量桑基图 (Sankey)
- 协议雷达图 (Radar)

**可选增强**:
- 可拖拽仪表板 (react-grid-layout)
- 图表联动
- 数据钻取

### ⚡ 性能优化 (2 天)
**Day 1**: 代码分割
- 路由级懒加载 (`React.lazy`)
- 组件级懒加载
- Bundle 分析 (rollup-plugin-visualizer)

**Day 2**: 缓存和渲染
- TanStack Query 优化
- 虚拟滚动 (react-window)
- React.memo / useMemo / useCallback
- 性能监控 (web-vitals)

---

## ✅ Phase 4: 长期优化 (3-5 天)

### 🌍 国际化 (2-3 天)
**依赖**:
```bash
npm install react-i18next i18next i18next-browser-languagedetector
```

**文件结构**:
```
src/i18n/
├── config.ts
└── locales/
    ├── en.json
    └── zh.json
```

**翻译范围**: 所有 UI 文本、表单、验证消息、Ant Design 组件

### 📱 移动端优化 (2 天)
**Day 1**: 响应式布局 (Sidebar 抽屉、Header 精简、表格卡片视图)
**Day 2**: 触摸交互 + 性能 + PWA (可选)

**测试设备**: iOS Safari, Android Chrome, 平板

---

## 🚨 精简版路线（10 天）

如果时间紧张，按此顺序执行：

| 天数 | 任务 | 工时 |
|------|------|------|
| Day 1-3 | 用户认证系统 | 3 天 |
| Day 4 | 策略统计 API | 1 天 |
| Day 5-6 | 暗黑模式 | 2 天 |
| Day 7-8 | 核心测试（70% 覆盖）| 2 天 |
| Day 9 | 性能优化（代码分割）| 1 天 |
| Day 10 | 报表导出基础（仅 PDF）| 1 天 |

**延后**: 国际化、高级可视化、移动端深度优化

---

## 📦 依赖安装清单

### 必须安装 (Phase 1-2)
```bash
# 报表导出
npm install jspdf jspdf-autotable
npm install --save-dev @types/jspdf

# 性能监控
npm install web-vitals
```

### 可选安装 (Phase 3-4)
```bash
# 虚拟滚动
npm install react-window

# 可拖拽布局
npm install react-grid-layout

# 国际化
npm install react-i18next i18next i18next-browser-languagedetector

# E2E 测试
npm install --save-dev @playwright/test

# Bundle 分析
npm install --save-dev rollup-plugin-visualizer
```

---

## ✅ 每日验收检查

### 代码质量
- [ ] ESLint 无警告
- [ ] TypeScript 无类型错误
- [ ] Prettier 格式化正确
- [ ] Git commit 信息规范

### 功能完整性
- [ ] 功能正常工作
- [ ] Loading 状态友好
- [ ] 错误处理完善
- [ ] 响应式布局正常

### 性能
- [ ] 首屏加载 < 2s
- [ ] 交互流畅 (60fps)
- [ ] 无内存泄漏
- [ ] Console 无错误

### 测试
- [ ] 单元测试通过
- [ ] 手动测试关键流程
- [ ] 跨浏览器测试 (Chrome, Firefox, Edge)

---

## 🎯 关键里程碑

| 里程碑 | 时间 | 验收标准 |
|--------|------|----------|
| M1: 认证上线 | Day 4 | 登录、权限控制、路由守卫 |
| M2: 主题导出 | Day 10 | 暗黑模式、PDF/CSV 导出 |
| M3: 测试完成 | Day 14 | 70% 覆盖率、关键流程测试 |
| M4: 性能达标 | Day 19 | LCP < 2.5s, Bundle 优化 |
| M5: 国际化发布 | Day 23 | 中英文完整支持 |

---

## 📊 进度跟踪

### 按优先级
- **P0 (必须)**: ⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜ 27% (5 天)
- **P1 (重要)**: ⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜ 55% (11-13 天)
- **P2 (优化)**: ⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜ 18% (3-5 天)

### 按类型
- **功能开发**: 60% (12 天)
- **测试**: 18% (3-4 天)
- **优化**: 22% (4-5 天)

---

## 🔗 相关文档

- 📘 **[详细开发计划](./DEVELOPMENT_ROADMAP.md)** - 完整任务分解
- 📗 **[可视化路线图](./ROADMAP_VISUAL.md)** - 时间线和里程碑
- 📙 **[项目 README](./README.md)** - 快速开始和技术栈
- 📕 **[图数据库文档](./GRAPH_DATABASE_IMPLEMENTATION.md)** - 拓扑实现细节

---

## 💡 快速命令

```bash
# 开发
npm run dev              # 启动开发服务器
npm run build            # 生产构建
npm run preview          # 预览生产版本

# 代码质量
npm run lint             # ESLint 检查
npm run format           # Prettier 格式化
npx tsc --noEmit         # TypeScript 类型检查

# 测试
npm run test             # 运行测试
npm run test:ui          # 测试 UI 界面
npm run test:coverage    # 覆盖率报告

# 分析
npm run build -- --mode analyze  # Bundle 分析
```

---

## 📞 问题反馈

遇到问题？参考以下资源：
- 技术问题：查看详细开发计划文档
- API 问题：联系后端团队确认接口规范
- 设计问题：与 UI/UX 团队沟通
- 性能问题：参考性能优化章节

---

**最后更新**: 2025-11-18
**维护者**: 开发团队
**版本**: v1.0
