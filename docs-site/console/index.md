---
layout: console
title: 控制台
---

# 双轨记忆控制台

这是一个客户端 SPA，请在浏览器中：

1. 先在登录框输入管理员 Token（与 cbmem-team 的 `-admin-token` 一致）
2. 登录后侧边栏会自动切换到主视图
3. 浏览器地址 URL hash 路由：
   - `#/users` — 用户管理
   - `#/projects` — 项目管理
   - `#/sessions` — 会话记录
   - `#/summarize` — 会话归纳
   - `#/distill` — 会话蒸馏

> 控制台直接调用 cbmem-team 的 `/api/console/*` 接口；
> 通过 `Content-Security-Policy: frame-ancestors` 限制嵌入位置；
> 所有数据写入 SQLite/MySQL。

## 文档站

回到文档浏览：[首页](/) | [SOP](/sop/) | [指南](/guide/intro)
