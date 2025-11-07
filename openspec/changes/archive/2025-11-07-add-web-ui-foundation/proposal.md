# 提案：Web UI 基础架构

**Change ID**: `add-web-ui-foundation`
**创建时间**: 2025-11-07
**状态**: 提案
**优先级**: P1 (高优先级 - 用户界面基础)
**预计工作量**: 2-3 天

---

## 为什么

微隔离系统当前仅提供后端 API 和 CLI 工具,缺少直观的 Web 管理界面。运维人员需要:

**当前痛点**:
- 无法可视化查看 Agent 状态和系统概览
- 无法通过图形界面管理策略
- 无法实时监控网络流量
- 学习成本高(需要记忆 API 或 CLI 命令)

**业务需求**:
- 提供现代化的 Web 管理界面
- 降低系统使用门槛
- 支持实时数据可视化
- 提升运维效率

## 变更内容

建立前端 Web UI 的基础架构,为后续功能模块开发奠定基础。

### 1. 项目初始化
- **技术栈**: React 18 + TypeScript + Vite
- **UI 框架**: Ant Design (antd)
- **构建工具**: Vite 5.x (快速开发体验)
- **包管理器**: npm

### 2. 核心依赖配置
- **路由**: React Router v6
- **状态管理**: Zustand (轻量级)
- **数据获取**: TanStack Query (React Query)
- **HTTP 客户端**: Axios
- **代码规范**: ESLint + Prettier

### 3. 基础架构
- **目录结构**: 模块化组织(api/components/pages/hooks/types/utils)
- **布局系统**: 响应式主布局(Header + Sidebar + Content)
- **路由系统**: 页面级路由配置
- **API 层**: 统一的 API 客户端封装

### 4. 类型系统
- **TypeScript 类型定义**: Agent、Flow、Policy 数据模型
- **API 响应类型**: 标准化接口定义
- **组件 Props 类型**: 类型安全的组件接口

### 5. 开发工具链
- **热重载**: Vite HMR
- **代码检查**: ESLint 规则配置
- **代码格式化**: Prettier 配置
- **类型检查**: TypeScript strict mode

---

## 成功标准

- [ ] 项目可正常启动(`npm run dev`)
- [ ] 基础布局渲染正常(Header、Sidebar、Content)
- [ ] 路由系统工作正常(可切换页面)
- [ ] API 客户端可连接后端(健康检查端点测试)
- [ ] TypeScript 类型定义完整且无错误
- [ ] 代码符合 ESLint 规范
- [ ] 构建成功(`npm run build`)

---

## 依赖

- 需要 Server MVP 提供 HTTP API (已完成)
- 需要后端配置 CORS 允许前端访问
- 无新的外部服务依赖

---

## 范围

**包含范围**:
- 前端项目初始化
- 核心依赖安装和配置
- 基础布局组件
- 路由系统
- API 客户端封装
- TypeScript 类型定义
- 开发工具配置

**不包含范围**:
- 具体业务功能(Agent 管理、Flow 分析等)
- 数据可视化图表
- WebSocket 实时推送
- 用户认证
- 国际化
- 暗色模式
