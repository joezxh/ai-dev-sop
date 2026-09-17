# 更新日志

## v2.0.0（2026-09-18）

### 记忆系统切换为 mem0

- 移除旧双轨记忆系统（`tools/mempalace`、`tools/codebase-memory-mcp`、`tools/cbmem-team`）
- 记忆功能统一替换为自托管 **mem0**（仓库 `deploy/mem0/`：API :8888 / MCP :8080 / Dashboard :3001）
- SOP-M1~M5 全部更新为 mem0 工作流；新增《mem0 AI 工具配置手册》（`docs/quick-ref/`）
- 控制台页面下线（管理功能由 mem0 Dashboard 提供），导航与开发代理同步移除
- 历史设计文档（superpowers specs/plans、benchmark）标注废弃归档

## v1.1.0（2026-07-12）

### 顶栏文档聚合菜单

- 顶栏新增「文档 / Docs / ドキュメント」下拉菜单，聚合 指南 / SOP / 参考 / 控制台 四个入口
- 三语 nav 同步改造（zh-CN / en / ja）
- 新增 `en/console/index.md` 与 `ja/console/index.md` 占位页（`layout: console`），避免 en/ja 下拉 404
- 现有 markdown 文档、侧边栏、控制台 SPA（`theme/console/Layout.vue` + `theme/index.ts`）保持不变

## v1.0.0（2026-07-08）

### 首次发布

- VitePress 1.3 + Vue 3 文档站点
- 中英日三语支持（zh-CN / en / ja）
- 完整 7 阶段 SOP（§0 → §6）拆分
- 双轨记忆框架（§5）配套【v2.0 起替换为 mem0】
- 场景 pipeline 文档（§4 / reference/）
- GitHub Pages 部署工作流
- Giscus 评论 / Mermaid 图表
- 进度条 / 书签 / 贡献者卡片组件

### 特性

- 三语切换 / 全文搜索 / 暗色模式
- 移动端响应式 / 代码块行号
- 上次更新时间 / 自定义 CSS 主题

### 待办（v1.1）

- 完整 EN/JA 子页面翻译
- MemPalace / codebase-mem-mcp 可视化演示【已随旧记忆系统废弃】
- SOP 视频教程
