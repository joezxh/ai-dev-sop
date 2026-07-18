---
layout: console
title: 控制台
---

# 双轨记忆控制台

这是一个客户端 SPA，请在浏览器中：

1. 输入用户名和密码登录（首次登录后需修改密码）
2. 登录后侧边栏会自动切换到主视图
3. 浏览器地址 URL hash 路由：
   - `#/teams` — 团队管理
   - `#/projects` — 项目管理
   - `#/modules` — 模块树
   - `#/memories` — 记忆库
   - `#/ai-tools` — AI 工具
   - `#/sessions` — 会话记录
   - `#/summarize` — 会话归纳
   - `#/distill` — 会话蒸馏
   - `#/status` — 运行状态
   - `#/logs` — 日志查看

> 控制台调用 cbmem-team 的 `/api/console/v2/*` 和 `/api/auth/*` 接口；
> 通过 JWT Bearer Token 进行认证；
> 所有数据写入 SQLite/MySQL。

## 文档站

回到文档浏览：[首页](/) | [SOP](/sop/) | [指南](/guide/intro)
