# AI编码技能插件包完整查询字典

> 本文档提供5个主流AI编码技能插件包的完整参考指南,包括技能列表、使用说明、最佳实践和对比分析。

---

## 目录

- [1. 插件包概览对比](#1-插件包概览对比)
- [2. Superpowers 详细指南](#2-superpowers-详细指南)
- [3. Get Shit Done (GSD) 详细指南](#3-get-shit-done-gsd-详细指南)
- [4. OpenSpec 详细指南](#4-openspec-详细指南)
- [5. gstack 详细指南](#5-gstack-详细指南)
- [6. awesome-design-md 详细指南](#6-awesome-design-md-详细指南)
- [7. 场景化使用范例](#7-场景化使用范例)
- [8. 协作与集成策略](#8-协作与集成策略)
- [9. 统一调用语法参考](#9-统一调用语法参考)

---

## 1. 插件包概览对比

### 基本信息对比表

| 属性 | Superpowers | Get Shit Done (GSD) | OpenSpec | gstack | awesome-design-md |
|------|-------------|---------------------|----------|--------|-------------------|
| **版本** | 最新(持续更新) | 最新(持续更新) | 最新(持续更新) | v0.19+ | 最新(71个设计) |
| **作者** | Jesse Vincent | TÂCHES | Fission AI | Garry Tan (YC CEO) | VoltAgent |
| **许可** | MIT | MIT | MIT | MIT | MIT |
| **安装路径** | `~/.claude/skills/superpowers/` | `~/.claude/skills/gsd-*/` | 全局npm包 | `~/.claude/skills/gstack/` | 项目根目录 |
| **核心定位** | 完整软件开发方法论 | 规范驱动开发+上下文工程 | 轻量级规范框架 | 虚拟工程团队模拟 | 设计系统集合 |
| **Skill数量** | 14个 | 60+命令 | 工作流驱动 | 30+技能 | 71个DESIGN.md |
| **适用场景** | TDD、系统化调试、代码质量 | 大型项目管理、多agent编排 | 需求管理、规范驱动 | 全流程开发、QA、安全 | UI设计一致性 |
| **学习曲线** | 中等 | 中高 | 低中 | 高 | 低 |
| **最佳搭配** | 小型项目、代码质量要求高 | 大型项目、复杂需求 | 需求频繁变更 | 全栈开发、生产环境 | 前端UI开发 |

### 核心哲学对比

| 插件包 | 核心理念 | 工作方式 |
|--------|---------|---------|
| **Superpowers** | "系统化胜于临时"、"证据胜于声明" | 自动触发式,强制性工作流 |
| **GSD** | "解决上下文腐化"、"复杂度在系统而非工作流" | 命令驱动循环,6步工作流 |
| **OpenSpec** | "灵活胜于僵化"、"迭代胜于瀑布" | 规范 artifact 管理 |
| **gstack** | "一个人像20人团队一样交付" | 角色模拟,冲刺流程 |
| **awesome-design-md** | "AI agent可读的设计系统" | 设计文件复制使用 |

---

## 2. Superpowers 详细指南

### 2.1 基本信息

- **安装路径**: `~/.claude/skills/superpowers/skills/`
- **安装命令**: `/plugin install superpowers@claude-plugins-official`
- **核心特性**: 自动触发、强制性工作流、TDD优先
- **支持平台**: Claude Code, Codex CLI, Cursor, Gemini CLI, OpenCode等

### 2.2 完整Skills列表

| Skill名称 | 触发条件 | 描述 | 允许工具 | 关键文件 |
|----------|---------|------|---------|---------|
| **brainstorming** | 任何创造性工作前 | 苏格拉底式设计细化,探索用户意图 | 所有读写工具 | `SKILL.md`, `visual-companion.md` |
| **using-git-worktrees** | 设计批准后 | 创建隔离工作区,新项目设置 | git命令 | `SKILL.md` |
| **writing-plans** | 设计批准后 | 将工作分解为2-5分钟任务 | 文件读写 | `SKILL.md` |
| **subagent-driven-development** | 有计划后 | 每个任务调度独立subagent | 所有工具 | `SKILL.md` |
| **executing-plans** | 有计划后 | 批量执行,人工检查点 | 所有工具 | `SKILL.md` |
| **test-driven-development** | 实现任何功能前 | RED-GREEN-REFACTOR循环 | 测试框架,文件读写 | `SKILL.md`, `testing-anti-patterns.md` |
| **systematic-debugging** | 遇到bug/测试失败 | 4阶段根因分析 | 调试工具,日志 | `SKILL.md`, `root-cause-tracing.md` |
| **verification-before-completion** | 声称完成前 | 验证修复真正生效 | 测试运行 | `SKILL.md` |
| **requesting-code-review** | 任务间 | 预审查清单 | 代码审查工具 | `SKILL.md` |
| **receiving-code-review** | 收到反馈时 | 响应审查反馈 | 所有工具 | `SKILL.md` |
| **finishing-a-development-branch** | 任务完成时 | 合并/PR决策工作流 | git命令 | `SKILL.md` |
| **dispatching-parallel-agents** | 独立任务≥2个 | 并发subagent工作流 | 所有工具 | `SKILL.md` |
| **writing-skills** | 创建新skill | 遵循最佳实践创建skill | 文件读写 | `SKILL.md` |
| **using-superpowers** | 初次使用 | skills系统介绍 | - | `SKILL.md` |

### 2.3 工作流程

```
brainstorming → using-git-worktrees → writing-plans → subagent-driven-development 
→ test-driven-development → requesting-code-review → verification-before-completion 
→ finishing-a-development-branch
```

### 2.4 使用说明

#### 安装步骤

```bash
# Claude Code官方市场
/plugin install superpowers@claude-plugins-official

# 或Superpowers市场
/plugin marketplace add obra/superpowers-marketplace
/plugin install superpowers@superpowers-marketplace
```

#### 最佳实践

1. **不要跳过brainstorming**: 即使是"简单"功能也要走设计流程
2. **严格遵守TDD**: 先写测试,看着失败,再写代码
3. **系统化调试**: 不要猜,按4阶段流程走
4. **证据胜于声明**: 声称完成前必须验证
5. **让skills自动触发**: 不需要手动调用,agent会自动识别

#### 关键原则

- **测试驱动开发**: 永远先写测试
- **系统化胜于临时**: 流程胜于猜测
- **复杂度降低**: 简单性是首要目标
- **证据胜于声明**: 验证后才宣布成功

### 2.5 使用范例

#### 范例1: 新功能开发

```
用户: 我想添加用户登录功能

AI自动触发: brainstorming
→ 探索项目上下文
→ 询问澄清问题(一次一个)
→ 提出2-3个方案(JWT vs Session vs OAuth)
→ 呈现设计并获得批准
→ 保存设计文档到 docs/superpowers/specs/

AI自动触发: writing-plans
→ 分解为小任务(每个2-5分钟)
→ 明确文件路径和验证步骤

AI自动触发: subagent-driven-development
→ 为每个任务调度subagent
→ 两阶段审查(规范合规+代码质量)
→ 独立提交

AI自动触发: test-driven-development
→ RED: 写失败测试
→ GREEN: 写最小代码通过测试
→ REFACTOR: 清理代码

AI自动触发: verification-before-completion
→ 运行所有测试
→ 验证功能真正工作
```

#### 范例2: 修复Bug

```
用户: 测试失败了,帮我修复

AI自动触发: systematic-debugging

阶段1: 根因调查
→ 仔细阅读错误消息
→ 重现问题
→ 检查最近变更
→ 追踪数据流

阶段2: 模式分析
→ 找到工作示例
→ 比较差异
→ 理解依赖

阶段3: 假设和测试
→ 形成单一假设
→ 最小化测试
→ 验证结果

阶段4: 实施
→ 创建失败测试用例(TDD)
→ 实施单一修复
→ 验证修复
→ 如果不工作,回到阶段1(最多3次)

AI自动触发: verification-before-completion
→ 确认测试通过
→ 确认没有引入新bug
```

#### 范例3: 代码审查

```
用户: 我完成了任务,帮我审查

AI自动触发: requesting-code-review
→ 检查清单:
  - 代码符合规范?
  - 测试覆盖?
  - 边界情况处理?
  - 性能考虑?

AI自动触发: receiving-code-review (如果收到反馈)
→ 理解反馈
→ 系统化响应
→ 不要防御性

AI自动触发: verification-before-completion
→ 验证所有问题已解决
→ 重新运行测试
```

---

## 3. Get Shit Done (GSD) 详细指南

### 3.1 基本信息

- **安装路径**: `~/.claude/skills/gsd-*/`
- **安装命令**: `npx get-shit-done-cc@latest`
- **核心特性**: 上下文工程、规范驱动、多agent编排
- **支持平台**: Claude Code, OpenCode, Gemini CLI, Kilo, Codex, Copilot, Cursor等15+运行时

### 3.2 核心命令列表

#### 主循环命令

| 命令 | 功能 | 触发场景 | 输出 |
|------|------|---------|------|
| `/gsd-new-project` | 问题→研究→需求→路线图 | 项目初始化 | PROJECT.md, REQUIREMENTS.md, ROADMAP.md |
| `/gsd-discuss-phase [N]` | 捕获实现决策 | 规划前 | CONTEXT.md |
| `/gsd-plan-phase [N]` | 研究+规划+验证循环 | 讨论后 | 可执行计划 |
| `/gsd-execute-phase <N>` | 并行波次执行计划 | 计划批准后 | 原子提交 |
| `/gsd-verify-work [N]` | 手动验收测试 | 执行后 | 验证报告+修复计划 |
| `/gsd-ship [N]` | 从验证工作创建PR | 验证通过后 | PR链接 |
| `/gsd-progress --next` | 自动检测并运行下一步 | 任何时间 | 自动流转 |
| `/gsd-complete-milestone` | 归档里程碑并标记发布 | 阶段完成 | 归档+tag |
| `/gsd-new-milestone` | 启动下一版本 | 里程碑完成后 | 新里程碑 |

#### 高级命令(部分)

| 命令 | 功能 | 使用场景 |
|------|------|---------|
| `/gsd-map-codebase` | 分析代码库架构和约定 | 已有代码库初始化 |
| `/gsd-code-review` | 代码审查 | 执行后审查 |
| `/gsd-debug` | 系统化调试 | 问题诊断 |
| `/gsd-forensics` | 深度问题调查 | 复杂bug |
| `/gsd-autonomous` | 自主模式 | 无人值守执行 |
| `/gsd-capture` | 捕获想法和需求 | 随时记录 |
| `/gsd-workspace` | 工作区管理 | 上下文切换 |
| `/gsd-config` | 配置管理 | 设置调整 |
| `/gsd-settings` | 交互式设置 | 配置修改 |
| `/gsd-fast` | 快速模式 | 跳过质量检查 |
| `/gsd-quick` | 快速任务 | 小型变更 |
| `/gsd-spike` | 技术探针 | 技术验证 |
| `/gsd-explore` | 探索代码库 | 理解现有代码 |

### 3.3 工作循环

```
/gsd-new-project (初始化)
  ↓
/gsd-discuss-phase 1 (讨论决策)
  ↓
/gsd-plan-phase 1 (研究+规划+验证)
  ↓
/gsd-execute-phase 1 (并行执行)
  ↓
/gsd-verify-work 1 (验收测试)
  ↓
/gsd-ship 1 (创建PR)
  ↓
/gsd-complete-milestone (归档)
  ↓
/gsd-new-milestone (下一版本)
```

### 3.4 使用说明

#### 安装步骤

```bash
# 交互式安装
npx get-shit-done-cc@latest

# 非交互安装(Claude Code)
npx get-shit-done-cc@latest --runtime claude --global

# 最小安装
npx get-shit-done-cc@latest --minimal
```

#### 配置

配置文件: `.planning/config.json`

关键配置项:

```json
{
  "mode": "interactive",  // interactive 或 yolo
  "workflow": {
    "research": true,     // 质量agent
    "plan_check": true,
    "verifier": true
  },
  "parallelization": {
    "enabled": true       // 并行执行
  },
  "model_profiles": {
    "quality": "opus",    // 高质量模型
    "balanced": "sonnet", // 平衡
    "budget": "haiku"     // 预算
  }
}
```

#### 最佳实践

1. **先映射代码库**: 如果已有代码,先运行`/gsd-map-codebase`
2. **保持上下文清洁**: 主上下文保持30-40%,重工作在subagent
3. **使用结构化artifacts**: PROJECT.md, ROADMAP.md, STATE.md等
4. **验证胜于信任**: 每个阶段都要验证
5. **利用并行化**: 独立计划同时执行
6. **配置质量agent**: 根据项目调整research/plan_check/verifier

#### 解决上下文腐化

GSD通过以下方式解决:
- **新鲜上下文**: 每个researcher/planner/executor启动新鲜200k token上下文
- **结构化持久化**: artifacts跨会话保存
- **主上下文保护**: 主会话保持轻量

### 3.5 使用范例

#### 范例1: 新项目启动

```bash
# 步骤1: 安装并初始化
npx get-shit-done-cc@latest
/gsd-new-project

# AI会问:
# - 你要构建什么?
# - 目标用户是谁?
# - 成功标准是什么?
# - 技术栈偏好?
# - 时间约束?

# 输出:
# ✓ PROJECT.md (愿景)
# ✓ REQUIREMENTS.md (范围)
# ✓ ROADMAP.md (路线图)
# ✓ STATE.md (当前状态)

# 步骤2: 讨论第1阶段
/gsd-discuss-phase 1

# AI会问:
# - 布局偏好?
# - API形状?
# - 错误处理策略?
# - 数据结构设计?

# 输出:
# ✓ CONTEXT.md (实现决策)

# 步骤3: 规划第1阶段
/gsd-plan-phase 1

# AI会:
# - 研究技术方案
# - 制定详细计划
# - 验证计划可行性
# - 循环直到计划通过

# 输出:
# ✓ 可执行计划(每个任务小到一个上下文窗口)

# 步骤4: 执行第1阶段
/gsd-execute-phase 1

# AI会:
# - 并行波次执行
# - 每个executor新鲜上下文
# - 每个任务原子提交
# - 主上下文保持30-40%

# 步骤5: 验证工作
/gsd-verify-work 1

# AI会:
# -  walkthrough构建内容
# - 诊断失败
# - 生成修复计划
# - 准备重新执行

# 步骤6: 发布
/gsd-ship 1
/gsd-complete-milestone
```

#### 范例2: 已有代码库添加功能

```bash
# 步骤1: 映射现有代码
/gsd-map-codebase

# AI分析:
# - 技术栈
# - 架构模式
# - 代码约定
# - 依赖关系

# 步骤2: 初始化项目上下文
/gsd-new-project

# AI会基于映射结果提出更精准的问题

# 步骤3: 捕获新需求
/gsd-capture "添加用户通知系统"

# 步骤4: 进入主循环
/gsd-progress --next

# 自动流转到下一步(discuss/plan/execute/verify)
```

#### 范例3: 调试复杂问题

```bash
# 使用forensics深度调查
/gsd-forensics "用户登录后偶尔看到空白页面"

# AI会:
# 1. 收集证据(日志、错误报告、用户反馈)
# 2. 分析模式(何时发生、频率、共同因素)
# 3. 形成假设
# 4. 最小化测试
# 5. 诊断根本原因
# 6. 生成修复计划

# 或者使用debug命令
/gsd-debug "API响应超时"

# AI会系统化调试:
# - 检查网络请求
# - 分析数据库查询
# - 检查缓存策略
# - 定位瓶颈
```

---

## 4. OpenSpec 详细指南

### 4.1 基本信息

- **安装路径**: 全局npm包 + 项目内`openspec/`目录
- **安装命令**: `npm install -g @fission-ai/openspec@latest`
- **初始化**: `openspec init`
- **核心特性**: artifact驱动、灵活迭代、棕色field友好
- **支持工具**: 25+ AI助手

### 4.2 核心概念

| 概念 | 描述 | 位置 |
|------|------|------|
| **Proposal** | 为什么做这个、什么在变更 | `openspec/changes/<name>/proposal.md` |
| **Specs** | 需求和场景 | `openspec/changes/<name>/specs/` |
| **Design** | 技术方法 | `openspec/changes/<name>/design.md` |
| **Tasks** | 实现清单 | `openspec/changes/<name>/tasks.md` |
| **Archive** | 已完成的变更 | `openspec/changes/archive/` |

### 4.3 命令列表

#### Slash Commands

| 命令 | 功能 | 使用场景 |
|------|------|---------|
| `/opsx:propose <idea>` | 创建变更提案 | 开始新功能 |
| `/opsx:apply` | 实现tasks | 执行变更 |
| `/opsx:archive` | 归档已完成变更 | 功能完成后 |
| `/opsx:new` | 扩展工作流-新建 | 完整工作流 |
| `/opsx:continue` | 扩展工作流-继续 | 继续工作 |
| `/opsx:ff` | 扩展工作流-快速前进 | 跳过某些步骤 |
| `/opsx:verify` | 验证实现 | 检查完成度 |
| `/opsx:bulk-archive` | 批量归档 | 清理多个变更 |
| `/opsx:onboard` | 新项目引导 | 初次使用 |

#### CLI Commands

| 命令 | 功能 |
|------|------|
| `openspec init` | 初始化项目 |
| `openspec update` | 更新AI指导 |
| `openspec config profile` | 配置工作流 |

### 4.4 工作流

#### 标准工作流

```
/opsx:propose "add-dark-mode"
  ↓
Created openspec/changes/add-dark-mode/
  ✓ proposal.md
  ✓ specs/
  ✓ design.md
  ✓ tasks.md
  ↓
/opsx:apply
  ↓
Implementing tasks...
  ✓ 1.1 Add theme context provider
  ✓ 1.2 Create toggle component
  ✓ 2.1 Add CSS variables
  ✓ 2.2 Wire up localStorage
  ↓
/opsx:archive
  ↓
Archived to openspec/changes/archive/2025-01-23-add-dark-mode/
```

#### 扩展工作流

```
/opsx:new → 创建变更(完整流程)
  ↓
/opsx:continue → 继续工作
  ↓
/opsx:ff → 快速前进(跳过某些验证)
  ↓
/opsx:verify → 验证实现
  ↓
/opsx:archive → 归档
```

### 4.5 使用说明

#### 安装步骤

```bash
# 全局安装
npm install -g @fission-ai/openspec@latest

# 项目初始化
cd your-project
openspec init

# 选择工作流
openspec config profile

# 应用配置
openspec update
```

#### 最佳实践

1. **先提案后实现**: 使用`/opsx:propose`明确要构建什么
2. **灵活更新artifact**: 任何时候都可以更新proposal/specs/design/tasks
3. **保持变更隔离**: 每个变更独立文件夹
4. **及时归档**: 完成后立即归档保持整洁
5. **使用高推理模型**: 推荐Opus 4.5和GPT 5.2
6. **保持上下文清洁**: 开始实现前清理上下文

#### 哲学

```
→ fluid not rigid (灵活胜于僵化)
→ iterative not waterfall (迭代胜于瀑布)
→ easy not complex (简单胜于复杂)
→ built for brownfield not just greenfield (棕色field优先)
→ scalable from personal projects to enterprises (可扩展)
```

### 4.6 使用范例

#### 范例1: 添加暗色模式

```bash
# 步骤1: 提案
/opsx:propose "add-dark-mode"

# AI创建:
# openspec/changes/add-dark-mode/
# ├── proposal.md (为什么做、什么变更)
# ├── specs/ (需求和场景)
# ├── design.md (技术方法)
# └── tasks.md (实现清单)

# 步骤2: 审查和批准
# 用户可以审查所有artifacts
# 可以修改proposal、specs、design、tasks

# 步骤3: 实现
/opsx:apply

# AI按tasks.md逐项实现
# ✓ 1.1 Add theme context provider
# ✓ 1.2 Create toggle component
# ✓ 2.1 Add CSS variables
# ✓ 2.2 Wire up localStorage

# 步骤4: 归档
/opsx:archive

# 移动到 openspec/changes/archive/2025-01-23-add-dark-mode/
# 更新specs到主目录
```

#### 范例2: 更新现有功能

```bash
# 步骤1: 提案
/opsx:propose "update-user-profile-api"

# AI创建变更提案
# 描述为什么更新、什么变更

# 步骤2: 修改设计
# 发现design.md需要调整
# 直接编辑 design.md

# 步骤3: 继续实现
/opsx:continue

# 步骤4: 验证
/opsx:verify

# AI验证:
# - 所有tasks完成?
# - specs满足?
# - 没有回归?

# 步骤5: 归档
/opsx:archive
```

#### 范例3: 大型功能分解

```bash
# 步骤1: 提案大型功能
/opsx:propose "build-notification-system"

# AI识别需要分解:
# - Email通知
# - Push通知
# - In-app通知
# - 通知偏好设置

# 步骤2: 创建子变更
/opsx:new "email-notifications"
/opsx:new "push-notifications"
/opsx:new "in-app-notifications"
/opsx:new "notification-preferences"

# 步骤3: 分别实现
/opsx:apply "email-notifications"
/opsx:archive "email-notifications"

/opsx:apply "push-notifications"
/opsx:archive "push-notifications"

# ... 依此类推

# 步骤4: 批量归档
/opsx:bulk-archive
```

---

## 5. gstack 详细指南

### 5.1 基本信息

- **安装路径**: `~/.claude/skills/gstack/`
- **安装命令**: 通过Claude Code paste命令
- **核心特性**: 23个专家角色、8个功率工具、冲刺流程
- **创建者**: Garry Tan (Y Combinator CEO)
- **支持平台**: 10个AI编码agent

### 5.2 完整Skills列表

#### 思考阶段(Think)

| Skill | 角色 | 功能 | 触发方式 |
|-------|------|------|---------|
| `/office-hours` | YC Office Hours | 6个强制问题重新定义产品 | 描述想法后 |
| `/plan-ceo-review` | CEO/Founder | 重新思考问题,找到10星产品 | 设计文档后 |
| `/plan-eng-review` | Eng Manager | 锁定架构、数据流、边界情况 | CEO review后 |
| `/plan-design-review` | Senior Designer | 设计维度评分0-10,AI Slop检测 | 需要设计审查 |
| `/plan-devex-review` | DX Lead | 交互式DX审查,TTHW基准 | 开发者工具 |
| `/design-consultation` | Design Partner | 从零构建完整设计系统 | 需要设计系统 |
| `/autoplan` | Review Pipeline | CEO→design→eng→DX自动流水线 | 一键完整规划 |

#### 构建阶段(Build)

| Skill | 角色 | 功能 | 触发方式 |
|-------|------|------|---------|
| `/review` | Staff Engineer | 查找CI通过但生产会炸的bug | 代码写完后 |
| `/investigate` | Debugger | 系统化根因调试 | 遇到bug |
| `/design-review` | Designer Who Codes | 设计审查+修复 | 需要设计修复 |
| `/devex-review` | DX Tester | 实时DX审计 | 开发者工具完成后 |
| `/design-shotgun` | Design Explorer | 生成4-6个AI mockup变体 | 探索设计选项 |
| `/design-html` | Design Engineer | mockup转生产HTML/CSS | 批准mockup后 |
| `/careful` | Safety Guard | 破坏性命令警告 | 说"be careful" |
| `/freeze` | Edit Lock | 限制编辑到一个目录 | 调试时 |
| `/guard` | Full Safety | careful+freeze组合 | 生产工作 |

#### 测试阶段(Test)

| Skill | 角色 | 功能 | 触发方式 |
|-------|------|------|---------|
| `/qa` | QA Lead | 测试应用、发现bug、修复、回归测试 | 提供staging URL |
| `/qa-only` | QA Reporter | 仅报告bug,不修改代码 | 需要纯报告 |
| `/browse` | QA Engineer | 真实浏览器操作 | 需要浏览器访问 |
| `/open-gstack-browser` | GStack Browser | 启动GStack Browser | 需要完整浏览器 |
| `/setup-browser-cookies` | Session Manager | 导入真实浏览器cookies | 测试认证页面 |
| `/benchmark` | Performance Engineer | 页面加载时间、Core Web Vitals | 性能测试 |
| `/canary` | SRE | 部署后监控 | 部署后 |

#### 发布阶段(Ship)

| Skill | 角色 | 功能 | 触发方式 |
|-------|------|------|---------|
| `/ship` | Release Engineer | 同步main、运行测试、审计覆盖、创建PR | 准备发布 |
| `/land-and-deploy` | Release Engineer | 合并PR、等待CI、部署、验证生产 | PR批准后 |
| `/document-release` | Technical Writer | 更新所有文档匹配发布 | 发布后 |
| `/setup-deploy` | Deploy Configurator | 一次性部署配置 | 首次部署 |

#### 反思阶段(Reflect)

| Skill | 角色 | 功能 | 触发方式 |
|-------|------|------|---------|
| `/retro` | Eng Manager | 团队感知每周回顾 | 周末/里程碑后 |
| `/learn` | Memory | 管理跨会话学习 | 查看/搜索/修剪记忆 |

#### 功率工具(Power Tools)

| Skill | 功能 | 使用场景 |
|-------|------|---------|
| `/codex` | 独立代码审查(Codex CLI) | 第二意见 |
| `/cso` | OWASP Top 10 + STRIDE威胁模型 | 安全审计 |
| `/pair-agent` | 多agent浏览器协调 | 共享浏览器 |
| `/gstack-upgrade` | 自更新 | 升级gstack |
| `/setup-gbrain` | GBrain设置 | 持久知识 |
| `/sync-gbrain` | GBrain同步 | 保持最新 |
| `/context-restore` | 上下文恢复 | 崩溃恢复 |
| `/unfreeze` | 解锁编辑 | 解除freeze |

### 5.3 冲刺流程

```
Think (思考)
  /office-hours → /plan-ceo-review → /plan-eng-review → /plan-design-review
  ↓
Build (构建)
  /review / /investigate / /design-review / /devex-review
  ↓
Test (测试)
  /qa → /qa-only → /browse → /benchmark
  ↓
Ship (发布)
  /ship → /land-and-deploy → /document-release
  ↓
Reflect (反思)
  /retro → /learn
```

### 5.4 使用说明

#### 安装步骤

```bash
# 在Claude Code中粘贴:
Install gstack: run git clone --single-branch --depth 1 https://github.com/garrytan/gstack.git ~/.claude/skills/gstack && cd ~/.claude/skills/gstack && ./setup

# 团队模式(推荐)
(cd ~/.claude/skills/gstack && ./setup --team) && ~/.claude/skills/gstack/bin/gstack-team-init required

# 其他agent
./setup --host codex    # OpenAI Codex
./setup --host opencode # OpenCode
./setup --host cursor   # Cursor
```

#### 最佳实践

1. **从/office-hours开始**: 重新定义产品再写代码
2. **使用/autoplan**: 一键完整规划(CEO→design→eng→DX)
3. **每个阶段用对应review**:
   - 终端用户: `/plan-design-review` + `/design-review`
   - 开发者: `/plan-devex-review` + `/devex-review`
   - 架构: `/plan-eng-review` + `/review`
4. **测试一切**: `/ship`自动引导测试框架
5. **使用真实浏览器**: `/open-gstack-browser`绕过反bot
6. **安全第一**: `/cso`做安全审计
7. **并行冲刺**: 10-15个并行sprints是可行的
8. **持续检查点**: `gstack-config set checkpoint_mode continuous`

#### 配置

```bash
# 设置检查点模式
gstack-config set checkpoint_mode continuous

# 启用遥测(可选)
gstack-config set telemetry on

# 切换命令前缀
cd ~/.claude/skills/gstack && ./setup --no-prefix  # /qa
cd ~/.claude/skills/gstack && ./setup --prefix     # /gstack-qa
```

### 5.5 使用范例

#### 范例1: 端到端功能开发

```bash
# 步骤1: 产品 interrogation
/office-hours

# AI问6个强制问题:
# - 痛点是什么?(具体例子,不是假设)
# - 谁有这个问题?
# - 现在怎么解决?
# - 为什么现有方案不够?
# - 成功看起来像什么?
# - 最小可行方案是什么?

# 输出: 设计文档

# 步骤2: CEO审查
/plan-ceo-review

# AI重新思考问题:
# - 找到请求中的10星产品
# - 4种模式: Expansion/Selective/Hold/Reduction
# - 挑战范围

# 步骤3: 工程审查
/plan-eng-review

# AI锁定:
# - 架构
# - 数据流(ASCII图)
# - 状态机
# - 错误路径
# - 测试矩阵
# - 失败模式
# - 安全考虑

# 步骤4: 批准计划并实现
# 退出计划模式,开始编码

# 步骤5: 代码审查
/review

# AI查找:
# - CI通过但生产会炸的bug
# - 自动修复明显的
# - 标记完整性差距

# 步骤6: QA测试
/qa https://staging.myapp.com

# AI:
# - 打开真实浏览器
# - 点击流程
# - 发现并修复bug
# - 生成回归测试

# 步骤7: 发布
/ship

# AI:
# - 同步main
# - 运行测试
# - 审计覆盖率
# - 推送
# - 创建PR

# 步骤8: 部署
/land-and-deploy

# AI:
# - 合并PR
# - 等待CI
# - 部署
# - 验证生产健康
```

#### 范例2: 设计探索到实现

```bash
# 步骤1: 设计系统构建
/design-consultation

# AI:
# - 研究景观
# - 提出创意风险
# - 生成真实产品mockup
# - 写入DESIGN.md

# 步骤2: 设计探索
/design-shotgun

# AI:
# - 生成4-6个AI mockup变体
# - 在浏览器中打开比较板
# - 收集反馈
# - 迭代直到满意
# - Taste memory学习偏好

# 步骤3: 转生产HTML
/design-html

# AI:
# - mockup转生产HTML/CSS
# - 使用Pretext计算文本布局
# - 检测框架(React/Svelte/Vue)
# - 智能API路由
# - 输出可发布的

# 步骤4: 设计审查
/design-review

# AI:
# - 运行设计审计
# - 修复发现的问题
# - 原子提交
# - 前后截图
```

#### 范例3: 安全审计

```bash
# 运行首席安全官审计
/cso

# AI执行:
# 1. OWASP Top 10检查
# 2. STRIDE威胁建模
# 3. 17个误报排除
# 4. 8/10+置信度门控
# 5. 独立发现验证
# 6. 每个发现包含具体利用场景

# 输出:
# - 安全报告
# - 威胁模型
# - 修复建议
# - 优先级排序
```

---

## 6. awesome-design-md 详细指南

### 6.1 基本信息

- **安装路径**: 项目根目录
- **获取方式**: 复制到项目
- **核心特性**: 71个真实网站DESIGN.md,AI可读格式
- **创建者**: VoltAgent
- **格式**: Google Stitch DESIGN.md格式扩展

### 6.2 设计系统集合

#### AI & LLM平台 (12个)

| 品牌 | 设计风格 | 适用场景 |
|------|---------|---------|
| **Claude** | 暖赤土色点缀,清洁编辑布局 | AI助手、内容平台 |
| **Cohere** | 活力渐变,数据丰富仪表板 | 企业AI、数据可视化 |
| **ElevenLabs** | 暗色电影UI,音频波形美学 | 音频平台、媒体应用 |
| **Ollama** | 终端优先,单色简洁 | 开发者工具、CLI |
| **Vercel** | 黑白精度,Geist字体 | 开发者平台、部署 |
| ... | ... | ... |

#### 开发者工具 & IDE (7个)

| 品牌 | 设计风格 | 适用场景 |
|------|---------|---------|
| **Cursor** | 流畅暗色界面,渐变点缀 | AI代码编辑器 |
| **Raycast** | 流畅暗色铬,活力渐变 | 生产力工具 |
| **Warp** | 暗色IDE界面,块状命令UI | 现代终端 |
| ... | ... | ... |

#### 后端、数据库 & DevOps (8个)

| 品牌 | 设计风格 | 适用场景 |
|------|---------|---------|
| **Supabase** | 暗色翡翠主题,代码优先 | 后端平台、数据库 |
| **MongoDB** | 绿叶品牌,开发者文档 | 数据库、开发者工具 |
| **Sentry** | 暗色仪表板,数据密集 | 监控、错误追踪 |
| ... | ... | ... |

####  productivity & SaaS (7个)

| 品牌 | 设计风格 | 适用场景 |
|------|---------|---------|
| **Linear** | 超极简,精确,紫色点缀 | 项目管理、工具 |
| **Notion** | 暖极简,serif标题,柔和表面 | 工作区、文档 |
| **Stripe** | 签名紫色渐变,weight-300优雅 | 支付、SaaS |
| ... | ... | ... |

#### 设计 & 创意工具 (6个)

| 品牌 | 设计风格 | 适用场景 |
|------|---------|---------|
| **Figma** | 活力多色,有趣而专业 | 设计工具 |
| **Framer** | 大胆黑蓝,动效优先 | 网站构建器 |
| **Airtable** | 彩色,友好,结构化数据 | 数据管理 |
| ... | ... | ... |

#### 金融科技 & 加密货币 (7个)

| 品牌 | 设计风格 | 适用场景 |
|------|---------|---------|
| **Stripe** | 紫色渐变,优雅 | 支付基础设施 |
| **Coinbase** | 清洁蓝色,信任感 | 加密货币交易所 |
| **Revolut** | 流畅暗色,渐变卡片 | 数字银行 |
| ... | ... | ... |

#### 电商 & 零售 (5个)

| 品牌 | 设计风格 | 适用场景 |
|------|---------|---------|
| **Airbnb** | 暖珊瑚点缀,摄影驱动 | 旅游市场 |
| **Shopify** | 暗色电影,霓虹绿点缀 | 电商平台 |
| **Nike** | 单色UI,巨大Futura | 运动零售 |
| ... | ... | ... |

#### 媒体 & 消费科技 (12个)

| 品牌 | 设计风格 | 适用场景 |
|------|---------|---------|
| **Apple** | 高级留白,SF Pro,电影图像 | 消费电子 |
| **Spotify** | 活力绿暗色,粗体字体 | 音乐流媒体 |
| **SpaceX** |  stark黑白,全幅图像 | 航天科技 |
| ... | ... | ... |

#### 汽车 (7个)

| 品牌 | 设计风格 | 适用场景 |
|------|---------|---------|
| **Tesla** | 激进减法,电影全幅摄影 | 电动车 |
| **BMW** | 暗色高级表面,精确德国工程 | 豪华汽车 |
| **Ferrari** | 明暗对比黑白编辑,法拉利红 | 豪华超跑 |
| ... | ... | ... |

### 6.3 DESIGN.md结构

每个DESIGN.md包含9个核心部分:

| # | 部分 | 捕获内容 |
|---|------|---------|
| 1 | Visual Theme & Atmosphere | 情绪、密度、设计哲学 |
| 2 | Color Palette & Roles | 语义名称+十六进制+功能角色 |
| 3 | Typography Rules | 字体系列,完整层级表 |
| 4 | Component Stylings | 按钮、卡片、输入、导航及状态 |
| 5 | Layout Principles | 间距尺度、网格、留白哲学 |
| 6 | Depth & Elevation | 阴影系统、表面层级 |
| 7 | Do's and Don'ts | 设计护栏和反模式 |
| 8 | Responsive Behavior | 断点、触摸目标、折叠策略 |
| 9 | Agent Prompt Guide | 快速颜色参考,即用提示 |

### 6.4 使用说明

#### 使用步骤

```bash
# 步骤1: 选择设计风格
# 浏览 https://getdesign.md/ 查看完整列表

# 步骤2: 复制DESIGN.md到项目
cp ~/awesome-design-md/design-md/stripe/DESIGN.md ./my-project/

# 步骤3: 告诉AI agent使用它
# "使用DESIGN.md构建登录页面"
```

#### 最佳实践

1. **选择匹配品牌的风格**: 支付应用选Stripe,社交选Linear
2. **复制后审查**: 检查颜色、字体是否符合需求
3. **自定义调整**: 修改DESIGN.md中的颜色和字体
4. **保持一致性**: 所有UI组件遵循同一DESIGN.md
5. **预览参考**: 使用preview.html查看颜色、类型、按钮、卡片

#### 配套文件

每个设计包含:

| 文件 | 用途 |
|------|------|
| `DESIGN.md` | 设计系统(AI读取) |
| `preview.html` | 视觉目录(亮色) |
| `preview-dark.html` | 视觉目录(暗色) |

### 6.5 使用范例

#### 范例1: 构建支付页面

```bash
# 步骤1: 复制Stripe设计系统
cp ~/awesome-design-md/design-md/stripe/DESIGN.md ./payment-app/
cp ~/awesome-design-md/design-md/stripe/preview.html ./payment-app/

# 步骤2: 告诉AI
# "使用DESIGN.md构建支付页面,遵循Stripe设计系统"

# AI会:
# - 使用紫色渐变主题
# - 应用weight-300字体
# - 遵循组件样式
# - 保持布局原则

# 步骤3: 预览参考
# 打开preview.html查看颜色样本、类型比例、按钮、卡片
```

#### 范例2: 构建项目管理工具

```bash
# 步骤1: 复制Linear设计系统
cp ~/awesome-design-md/design-md/linear.app/DESIGN.md ./project-mgmt/

# 步骤2: 告诉AI
# "使用DESIGN.md构建项目列表页面,遵循Linear设计系统"

# AI会:
# - 超极简风格
# - 精确间距
# - 紫色点缀
# - 清洁表面
```

#### 范例3: 构建音乐应用

```bash
# 步骤1: 复制Spotify设计系统
cp ~/awesome-design-md/design-md/spotify/DESIGN.md ./music-app/

# 步骤2: 告诉AI
# "使用DESIGN.md构建播放器界面,遵循Spotify设计系统"

# AI会:
# - 活力绿暗色主题
# - 粗体字体
# - 专辑封面驱动布局
```

---

## 7. 场景化使用范例

### 7.1 项目启动场景

#### 场景: 从零开始构建SaaS产品

**推荐插件**: GSD + gstack + awesome-design-md

```bash
# 步骤1: 使用GSD初始化项目
npx get-shit-done-cc@latest
/gsd-new-project
# → PROJECT.md, REQUIREMENTS.md, ROADMAP.md

# 步骤2: 使用gstack进行产品思考
/office-hours
# → 重新定义产品,6个强制问题

# 步骤3: 选择设计系统
cp ~/awesome-design-md/design-md/stripe/DESIGN.md ./
# → 一致的设计语言

# 步骤4: 使用gstack规划
/autoplan
# → CEO→design→eng→DX完整规划

# 步骤5: 使用GSD执行
/gsd-execute-phase 1
# → 并行波次执行

# 步骤6: QA测试
/qa https://staging.myapp.com
# → 真实浏览器测试

# 步骤7: 发布
/gsd-ship 1
# → 创建PR
```

#### 场景: 企业级API开发

**推荐插件**: OpenSpec + Superpowers

```bash
# 步骤1: 使用OpenSpec提案
/opsx:propose "build-user-api"
# → proposal.md, specs/, design.md, tasks.md

# 步骤2: 使用Superpowers进行TDD实现
# (自动触发test-driven-development)
# → RED-GREEN-REFACTOR

# 步骤3: 系统化调试(如遇问题)
# (自动触发systematic-debugging)
# → 4阶段根因分析

# 步骤4: OpenSpec验证
/opsx:verify
# → 验证specs满足度

# 步骤5: 归档
/opsx:archive
```

### 7.2 代码审查场景

#### 场景: PR审查和质量保证

**推荐插件**: gstack + Superpowers

```bash
# 步骤1: gstack代码审查
/review
# → 查找CI通过但生产会炸的bug
# → 自动修复明显的

# 步骤2: 第二意见(可选)
/codex
# → OpenAI Codex独立审查
# → 跨模型分析

# 步骤3: 安全审计
/cso
# → OWASP Top 10 + STRIDE

# 步骤4: Superpowers审查
# (自动触发requesting-code-review)
# → 预审查清单

# 步骤5: QA测试
/qa https://staging.myapp.com
# → 真实浏览器测试流程
```

### 7.3 设计咨询场景

#### 场景: 构建用户界面

**推荐插件**: gstack + awesome-design-md

```bash
# 步骤1: 选择设计系统
cp ~/awesome-design-md/design-md/linear.app/DESIGN.md ./

# 步骤2: 设计咨询
/design-consultation
# → 从零构建完整设计系统

# 步骤3: 设计探索
/design-shotgun
# → 生成4-6个AI mockup变体
# → 浏览器比较板
# → 迭代直到满意

# 步骤4: 转生产HTML
/design-html
# → mockup转生产HTML/CSS
# → Pretext计算布局

# 步骤5: 设计审查
/design-review
# → 设计审计+修复
```

### 7.4 问题调试场景

#### 场景: 生产环境Bug修复

**推荐插件**: Superpowers + gstack

```bash
# 步骤1: Superpowers系统化调试
# (自动触发systematic-debugging)

# 阶段1: 根因调查
# → 阅读错误消息
# → 重现问题
# → 检查最近变更
# → 追踪数据流

# 阶段2: 模式分析
# → 找到工作示例
# → 比较差异

# 阶段3: 假设和测试
# → 形成单一假设
# → 最小化测试

# 阶段4: 实施(TDD)
# → 创建失败测试
# → 修复
# → 验证

# 步骤2: 如果不工作,使用gstack深入
/investigate
# → 系统化根因调试
# → 最多3次修复后质疑架构

# 步骤3: QA验证
/qa https://staging.myapp.com
# → 真实浏览器验证修复
# → 生成回归测试
```

### 7.5 大型项目协作场景

#### 场景: 10人团队协作开发

**推荐插件**: GSD + OpenSpec + gstack

```bash
# 步骤1: OpenSpec管理需求
/opsx:propose "feature-a"
/opsx:propose "feature-b"
/opsx:propose "feature-c"

# 步骤2: GSD项目管理
/gsd-new-project
/gsd-discuss-phase 1
/gsd-plan-phase 1

# 步骤3: gstack并行sprints
# Sprint 1: /office-hours → /autoplan → implement → /ship
# Sprint 2: /office-hours → /autoplan → implement → /ship
# Sprint 3: /office-hours → /autoplan → implement → /ship

# 步骤4: GSD验证
/gsd-verify-work 1
/gsd-verify-work 2
/gsd-verify-work 3

# 步骤5: GSD发布
/gsd-ship 1
/gsd-ship 2
/gsd-ship 3
/gsd-complete-milestone
```

---

## 8. 协作与集成策略

### 8.1 插件包对比总结

| 维度 | Superpowers | GSD | OpenSpec | gstack | awesome-design-md |
|------|-------------|-----|----------|--------|-------------------|
| **优势** | TDD、代码质量、系统化调试 | 大型项目管理、上下文工程 | 需求管理、灵活迭代 | 全流程、QA、安全 | 设计一致性 |
| **劣势** | 不适合大型项目 | 学习曲线陡 | 不直接管代码质量 | 复杂度高 | 仅设计,无开发流程 |
| **最佳用于** | 代码质量要求高的项目 | 复杂大型项目 | 需求频繁变更 | 全栈生产开发 | 前端UI开发 |
| **自动化程度** | 高(自动触发) | 中(命令驱动) | 中(artifact驱动) | 中高(角色模拟) | 低(手动复制) |
| **团队协作** | 适合小团队 | 适合大团队 | 适合需求管理 | 适合全流程团队 | 适合设计团队 |

### 8.2 协作模式

#### 模式1: Superpowers + awesome-design-md (前端质量优先)

```
适用场景: 构建高质量前端应用
工作流:
1. 复制DESIGN.md (awesome-design-md)
2. brainstorming (Superpowers) - 设计UI实现方案
3. TDD实现 (Superpowers) - 严格测试驱动
4. systematic-debugging (Superpowers) - 问题修复
5. verification (Superpowers) - 验证设计一致性
```

#### 模式2: GSD + OpenSpec (大型项目管理)

```
适用场景: 企业级应用开发
工作流:
1. OpenSpec提案需求 (/opsx:propose)
2. GSD项目初始化 (/gsd-new-project)
3. GSD阶段规划 (/gsd-plan-phase)
4. OpenSpec跟踪变更 (/opsx:apply, /opsx:archive)
5. GSD执行和验证 (/gsd-execute, /gsd-verify)
```

#### 模式3: gstack + Superpowers (全流程质量保证)

```
适用场景: 生产级全栈开发
工作流:
1. gstack产品思考 (/office-hours)
2. Superpowers设计细化 (brainstorming)
3. gstack工程规划 (/plan-eng-review)
4. Superpowers TDD实现 (test-driven-development)
5. gstack QA和发布 (/qa, /ship)
```

#### 模式4: GSD + gstack + awesome-design-md (完整产品交付)

```
适用场景: 独立开发者构建完整SaaS
工作流:
1. GSD项目初始化 (/gsd-new-project)
2. gstack产品思考 (/office-hours)
3. awesome-design-md选择设计系统
4. gstack设计探索 (/design-shotgun, /design-html)
5. GSD阶段执行 (/gsd-execute-phase)
6. gstack QA (/qa)
7. GSD发布 (/gsd-ship)
```

### 8.3 集成建议

#### 不要同时使用

- **Superpowers + GSD主循环**: 两者都有完整工作流,会冲突
- **OpenSpec + GSD需求管理**: 功能重叠,选一个
- **多个设计系统**: 一个项目只用一个DESIGN.md

#### 可以组合使用

- **gstack + awesome-design-md**: gstack调用DESIGN.md指导UI开发
- **OpenSpec + Superpowers**: OpenSpec管需求,Superpowers管实现质量
- **GSD + awesome-design-md**: GSD项目管理,awesome-design-md管设计一致性

### 8.4 选择决策树

```
你要构建什么?
├─ 前端UI应用
│  ├─ 需要设计系统? → awesome-design-md
│  ├─ 需要严格TDD? → Superpowers
│  └─ 需要全流程? → gstack
│
├─ 大型复杂项目
│  ├─ 需要上下文管理? → GSD
│  ├─ 需要需求跟踪? → OpenSpec
│  └─ 需要多agent? → GSD
│
├─ 全栈生产应用
│  ├─ 需要QA和浏览器测试? → gstack
│  ├─ 需要安全审计? → gstack (/cso)
│  └─ 需要完整冲刺? → gstack
│
└─ API/后端服务
   ├─ 需要规范驱动? → OpenSpec
   ├─ 需要TDD? → Superpowers
   └─ 需要项目管理? → GSD
```

---

## 9. 统一调用语法参考

### 9.1 安装命令汇总

```bash
# Superpowers
/plugin install superpowers@claude-plugins-official

# Get Shit Done (GSD)
npx get-shit-done-cc@latest

# OpenSpec
npm install -g @fission-ai/openspec@latest
openspec init

# gstack
git clone --single-branch --depth 1 https://github.com/garrytan/gstack.git ~/.claude/skills/gstack && cd ~/.claude/skills/gstack && ./setup

# awesome-design-md
# 无需安装,复制DESIGN.md到项目即可
cp ~/awesome-design-md/design-md/<brand>/DESIGN.md ./my-project/
```

### 9.2 触发方式分类

#### 自动触发(Superpowers)

Superpowers的skills根据上下文自动触发,无需手动调用:

```
创造性工作 → brainstorming (自动)
设计批准 → using-git-worktrees (自动)
有计划 → writing-plans (自动)
实现 → test-driven-development (自动)
遇bug → systematic-debugging (自动)
任务间 → requesting-code-review (自动)
完成 → finishing-a-development-branch (自动)
```

#### 手动命令(GSD, OpenSpec, gstack)

需要用户显式调用:

```bash
# GSD
/gsd-new-project
/gsd-plan-phase 1
/gsd-execute-phase 1

# OpenSpec
/opsx:propose "idea"
/opsx:apply
/opsx:archive

# gstack
/office-hours
/plan-ceo-review
/review
/qa https://url
```

#### 文件驱动(awesome-design-md)

AI agent自动读取项目中的DESIGN.md:

```
项目根目录有DESIGN.md → AI自动遵循设计规范
```

### 9.3 参数规范

#### GSD命令参数

```bash
/gsd-discuss-phase [N]          # N=阶段号
/gsd-plan-phase [N]             # N=阶段号
/gsd-execute-phase <N>          # N=阶段号(必需)
/gsd-verify-work [N]            # N=阶段号
/gsd-ship [N]                   # N=阶段号
/gsd-progress --next            # 自动下一步
```

#### OpenSpec命令参数

```bash
/opsx:propose <idea>            # idea=功能描述(必需)
/opsx:apply [change-name]       # change-name=可选
/opsx:archive [change-name]     # change-name=可选
```

#### gstack命令参数

```bash
/qa <URL>                       # URL=staging URL(推荐)
/browse                         # 无参数
/design-shotgun                 # 无参数
/design-html [description]      # description=可选
/cso                            # 无参数
/review                         # 无参数
/ship                           # 无参数
```

### 9.4 配置文件位置

| 插件包 | 配置文件 | 关键配置 |
|--------|---------|---------|
| **Superpowers** | 无(自动) | 项目偏好覆盖 |
| **GSD** | `.planning/config.json` | mode, workflow, parallelization, models |
| **OpenSpec** | `openspec/config.yaml` | schema, context, rules |
| **gstack** | `~/.gstack/config.yaml` | checkpoint_mode, telemetry, prefix |
| **awesome-design-md** | `DESIGN.md` (项目根目录) | 颜色、字体、组件样式 |

### 9.5 退出和恢复

| 场景 | Superpowers | GSD | OpenSpec | gstack |
|------|-------------|-----|----------|--------|
| **暂停工作** | 保存设计文档 | `/gsd-pause-work` | 保持artifacts | `/context-save` |
| **恢复工作** | 读取设计文档 | `/gsd-resume-work` | 读取artifacts | `/context-restore` |
| **切换任务** | 新brainstorming | `/gsd-workspace` | 新proposal | 新/office-hours |
| **完全重置** | 删除specs | `/gsd-new-project` | 新init | 重新clone |

---

## 附录

### A. 快速参考卡片

#### Superpowers (14 skills)

```
核心: TDD + 系统化调试 + 自动触发
工作流: brainstorming → plans → TDD → review → verify
最佳: 代码质量要求高的小中型项目
安装: /plugin install superpowers@claude-plugins-official
```

#### GSD (60+ commands)

```
核心: 上下文工程 + 多agent编排 + 规范驱动
工作流: new-project → discuss → plan → execute → verify → ship
最佳: 大型复杂项目,需要项目管理
安装: npx get-shit-done-cc@latest
```

#### OpenSpec (artifact-driven)

```
核心: 灵活规范 + 迭代 + artifact管理
工作流: propose → apply → archive
最佳: 需求频繁变更,需要需求跟踪
安装: npm install -g @fission-ai/openspec@latest
```

#### gstack (30+ skills)

```
核心: 23个专家角色 + 冲刺流程 + QA/安全
工作流: think → plan → build → review → test → ship → reflect
最佳: 全栈生产开发,需要QA和安全
安装: git clone + ./setup
```

#### awesome-design-md (71 designs)

```
核心: 真实网站设计系统 + AI可读
工作流: 复制DESIGN.md → AI遵循
最佳: 前端UI开发,需要设计一致性
安装: 复制到项目根目录
```

### B. 学习路径推荐

#### 初学者路径

```
1. awesome-design-md (低门槛,立即见效)
2. OpenSpec (学习规范驱动)
3. Superpowers (学习TDD和系统化调试)
4. gstack (学习全流程开发)
5. GSD (学习大型项目管理)
```

#### 专业路径

```
1. Superpowers (建立代码质量基础)
2. awesome-design-md (建立设计系统)
3. gstack (建立全流程)
4. OpenSpec (建立需求管理)
5. GSD (建立大型项目管理)
```

#### 团队路径

```
1. OpenSpec (统一需求语言)
2. Superpowers (统一代码质量标准)
3. awesome-design-md (统一设计语言)
4. GSD (统一项目管理)
5. gstack (统一QA和安全标准)
```

### C. 常见问题FAQ

**Q: 我可以同时使用多个插件包吗?**
A: 可以,但要注意工作流冲突。推荐:OpenSpec(需求) + Superpowers(代码质量) + awesome-design-md(设计)

**Q: 哪个最适合初学者?**
A: awesome-design-md最简单,复制即用。其次是OpenSpec,工作流清晰。

**Q: 哪个最适合大型团队?**
A: GSD最强大,支持多agent、并行执行、完整项目管理。

**Q: TDD真的必需吗?**
A: Superpowers强制TDD,实践证明TDD减少bug、提高质量。如果不需要,可选GSD或OpenSpec。

**Q: 如何选择合适的插件包?**
A: 参考第8.4节的决策树,根据项目类型和需求选择。

**Q: 这些插件包免费吗?**
A: 全部MIT许可,完全免费开源。

---

## 文档版本

- **版本**: 1.0.0
- **创建日期**: 2026-05-09
- **最后更新**: 2026-05-09
- **维护者**: AI Assistant
- **反馈**: 请提交issue或PR

---

**声明**: 本文档基于5个开源项目的README和源代码编写,所有项目均为MIT许可。文档中的示例和说明基于项目官方文档,旨在帮助用户理解和使用这些工具。
