> [!WARNING]
> **已废弃（2026-09-18）**：本文档描述的双轨记忆系统（MemPalace / cbmem-team / codebase-memory-mcp）已移除，记忆功能统一替换为自托管 mem0（见 docs/quick-ref/mem0-ai-tools-config-guide.md）。本文仅作历史归档保留，内容不再维护。

# 双轨记忆系统工具最佳实践 - 实施计划

> **基于**: [2026-07-14-dual-track-memory-tools-best-practices-design.md](../specs/2026-07-14-dual-track-memory-tools-best-practices-design.md)
> **版本**: v1.0
> **计划日期**: 2026-07-14
> **执行日期**: 2026-07-14
> **状态**: ✅ 实施完成

---

## 1. 实施范围与目标

### 1.1 核心目标

将 spec 设计文档中的双轨记忆系统工具最佳实践，转化为可执行的操作流程和配置模板，使团队能够：

1. **快速上手**: 新成员 30 分钟内完成环境配置
2. **高效使用**: 掌握 80% 高频工具（P0/P1）
3. **知识沉淀**: 建立团队知识库，积累有价值的记忆
4. **协作共享**: 实现跨项目知识复用

### 1.2 实施范围

```
┌─────────────────────────────────────────────────────────────┐
│                        实施范围                               │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ✓ M1: 安装配置流程标准化                                     │
│  ✓ M2: 项目理解 SOP 编写                                      │
│  ✓ M3: 开发调试 SOP 编写                                      │
│  ✓ M4: 知识沉淀流程建立                                       │
│  ✓ M5: 团队协作机制设计                                       │
│                                                              │
│  ✓ 工具速查表 → 可交互检查清单                                 │
│  ✓ 双轨联合模式 → AI IDE 自动化配置                            │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 成功标准

| 维度 | 指标 | 目标值 |
|------|------|--------|
| 配置效率 | 新成员首次配置时间 | ≤ 30 分钟 |
| 工具使用 | P0/P1 工具覆盖度 | ≥ 80% |
| 知识积累 | 首个 Sprint 沉淀记忆数 | ≥ 20 条 |
| 团队协作 | 跨项目知识复用案例 | ≥ 5 个 |

---

## 2. 实施任务分解

### 2.1 Phase 1: 环境准备 (Day 1)

| 任务ID | 任务名称 | 负责 | 依赖 | 产出 |
|--------|----------|------|------|------|
| T1.1 | 部署 MemPalace 服务 | DevOps | - | 服务地址 |
| T1.2 | 部署 cbmem-team 服务 | DevOps | - | 服务地址 |
| T1.3 | 准备 codebase-memory-mcp 二进制 | DevOps | - | 二进制文件 |
| T1.4 | 初始化团队 MemPalace Wing | 管理员 | T1.1 | Wing 配置 |
| T1.5 | 创建首批测试用户账号 | 管理员 | T1.2 | 用户凭证 |

**产出**:
- `deploy/mempalace/` - MemPalace 部署配置
- `deploy/cbmem-team/` - cbmem-team 部署配置
- `config/initial-wings.yaml` - 初始 Wing 配置

### 2.2 Phase 2: 配置文档标准化 (Day 2-3)

| 任务ID | 任务名称 | 负责 | 依赖 | 产出 |
|--------|----------|------|------|------|
| T2.1 | 编写 M1 安装配置 SOP | 后端 | T1.1-T1.5 | SOP-M1.md |
| T2.2 | 编写 Cursor MCP 配置指南 | 后端 | T1.2 | cursor-mcp-config.md |
| T2.3 | 编写 Qoder/Claude Desktop 配置模板 | 后端 | T1.2 | ide-config-templates/ |
| T2.4 | 制作一键安装脚本 | DevOps | T2.1 | `scripts/install-all.sh` |
| T2.5 | 验证脚本跨平台 (Win/Mac/Linux) | 全员 | T2.4 | 测试报告 |

**产出**:
- `docs/sop/SOP-M1-installation.md` - 安装配置 SOP
- `docs/ide-config/` - 各 IDE 配置模板
- `scripts/` - 一键安装脚本

### 2.3 Phase 3: 工具使用 SOP 编写 (Day 4-6)

| 任务ID | 任务名称 | 负责 | 依赖 | 产出 |
|--------|----------|------|------|------|
| T3.1 | 编写 M2 项目理解 SOP | 架构师 | T2.1 | SOP-M2.md |
| T3.2 | 编写 M3 开发调试 SOP | 架构师 | T2.1 | SOP-M3.md |
| T3.3 | 编写 M4 知识沉淀 SOP | 架构师 | T2.1 | SOP-M4.md |
| T3.4 | 编写 M5 团队协作 SOP | 架构师 | T2.1 | SOP-M5.md |
| T3.5 | 制作工具速查卡片 | 前端 | T3.1-T3.4 | `cards/` |
| T3.6 | 编写双轨联合使用指南 | 架构师 | T3.1-T3.4 | dual-track-guide.md |

**产出**:
- `docs/sop/SOP-M2-understanding.md`
- `docs/sop/SOP-M3-development.md`
- `docs/sop/SOP-M4-knowledge.md`
- `docs/sop/SOP-M5-collaboration.md`
- `docs/cards/` - 工具速查卡片
- `docs/dual-track-guide.md`

### 2.4 Phase 4: AI IDE 自动化配置 (Day 7-8)

| 任务ID | 任务名称 | 负责 | 依赖 | 产出 |
|--------|----------|------|------|------|
| T4.1 | 设计 AI 自动调用规则 | 架构师 | T3.6 | `rules/` |
| T4.2 | 配置 Cursor AI 自动补全 | 后端 | T4.1 | cursor-rules.json |
| T4.3 | 测试双轨工具自动调用 | 全员 | T4.2 | 测试报告 |
| T4.4 | 编写 AI 提示词模板 | 产品 | T4.3 | `prompts/` |

**产出**:
- `cursor/rules/` - Cursor AI 规则配置
- `prompts/` - AI 提示词模板

### 2.5 Phase 5: 验证与优化 (Day 9-10)

| 任务ID | 任务名称 | 负责 | 依赖 | 产出 |
|--------|----------|------|------|------|
| T5.1 | 新成员体验验证 | 测试 | T4.3 | 体验报告 |
| T5.2 | 收集使用反馈 | 产品 | T5.1 | 反馈汇总 |
| T5.3 | 优化文档和流程 | 全员 | T5.2 | 更新后的 SOP |
| T5.4 | 制作培训材料 | 培训 | T5.3 | 培训 PPT |

**产出**:
- `docs/training/` - 培训材料
- 优化后的各 SOP 文档

---

## 3. 任务依赖关系图

```
                    ┌─────────┐
                    │  Start  │
                    └────┬────┘
                         │
                    ┌────▼────┐
                    │ Phase 1 │
                    │ 环境准备 │
                    └────┬────┘
                         │
          ┌──────────────┼──────────────┐
          │              │              │
     ┌────▼────┐   ┌────▼────┐   ┌────▼────┐
     │ Phase 2 │   │ Phase 3 │   │ Phase 4 │
     │ 配置文档 │   │ SOP编写 │   │ IDE自动化│
     └────┬────┘   └────┬────┘   └────┬────┘
          │              │              │
          └──────────────┼──────────────┘
                         │
                    ┌────▼────┐
                    │ Phase 5 │
                    │ 验证优化 │
                    └────┬────┘
                         │
                    ┌────▼────┐
                    │  Done   │
                    └─────────┘
```

---

## 4. 详细任务清单

### 4.1 T1.1 - T1.5: 环境准备

```yaml
Phase 1: 环境准备 (Day 1)
├── T1.1: 部署 MemPalace 服务
│   ├── 创建 Docker Compose 配置
│   ├── 配置数据卷持久化
│   ├── 设置初始管理员账号
│   └── 验证服务健康检查
│
├── T1.2: 部署 cbmem-team 服务
│   ├── 配置 JWT Secret
│   ├── 配置 Admin Token
│   ├── 配置 MySQL/SQLite 后端
│   ├── 配置 LLM Provider (可选)
│   └── 验证 /healthz 端点
│
├── T1.3: 准备 codebase-memory-mcp 二进制
│   ├── 下载/编译二进制
│   ├── 安装到标准路径
│   └── 验证基本功能
│
├── T1.4: 初始化团队 MemPalace Wing
│   ├── 创建 team-{dept} Wing
│   ├── 创建 project-{name} Wing 模板
│   └── 预置 onboarding Room
│
└── T1.5: 创建首批测试用户账号
    ├── 创建 3-5 个测试用户
    ├── 分配项目权限
    └── 生成 JWT Token
```

### 4.2 T2.1 - T2.5: 配置文档标准化

```yaml
Phase 2: 配置文档标准化 (Day 2-3)
├── T2.1: 编写 M1 安装配置 SOP
│   ├── MemPalace 安装步骤
│   ├── cbmem-team 安装步骤
│   ├── codebase-memory-mcp 安装步骤
│   ├── AI IDE MCP 配置步骤
│   └── 验证检查清单
│
├── T2.2: 编写 Cursor MCP 配置指南
│   ├── [已完成] cursor-mcp-config.md
│   └── 补充截图指引
│
├── T2.3: 编写 IDE 配置模板
│   ├── Cursor: .cursor/mcp.json 模板
│   ├── Qoder: .qoder/mcp.json 模板
│   └── Claude Desktop: 配置模板
│
├── T2.4: 制作一键安装脚本
│   ├── install-all.sh (Linux/Mac)
│   ├── install-all.ps1 (Windows)
│   ├── 脚本参数化设计
│   └── 自动检测依赖
│
└── T2.5: 验证脚本跨平台
    ├── Linux 环境测试
    ├── macOS 环境测试
    └── Windows 环境测试
```

### 4.3 T3.1 - T3.6: 工具使用 SOP 编写

```yaml
Phase 3: 工具使用 SOP 编写 (Day 4-6)
├── T3.1: 编写 M2 项目理解 SOP
│   ├── 首次接入项目流程
│   ├── 使用 mempalace_search 查询上下文
│   ├── 使用 get_architecture 获取架构
│   ├── 使用 search_code/search_graph 搜索代码
│   └── 练习题: 理解示例项目
│
├── T3.2: 编写 M3 开发调试 SOP
│   ├── 日常开发工具使用流程
│   ├── 使用 trace_path 追踪调用链
│   ├── 使用 detect_changes 分析变更影响
│   ├── 调试会话保存机制
│   └── 常见问题排查指南
│
├── T3.3: 编写 M4 知识沉淀 SOP
│   ├── 会话归纳流程 (Console)
│   ├── 知识蒸馏流程 (Console)
│   ├── 手动添加记忆最佳实践
│   ├── Hall 分类规范
│   └── 知识质量评估标准
│
├── T3.4: 编写 M5 团队协作 SOP
│   ├── 团队 Wing 组织规范
│   ├── 跨项目知识共享流程
│   ├── ADR 管理规范
│   ├── Tunnel 连接使用
│   └── 新人 onboarding 流程
│
├── T3.5: 制作工具速查卡片
│   ├── MemPalace P0/P1 工具卡片
│   ├── Codebase P0/P1 工具卡片
│   ├── 双轨联合使用流程卡
│   └── 打印版 PDF 导出
│
└── T3.6: 编写双轨联合使用指南
    ├── 模式一: 理解→开发
    ├── 模式二: 调试→知识
    ├── 模式三: 评审→归档
    └── 高级用法: 自定义工作流
```

### 4.4 T4.1 - T4.4: AI IDE 自动化配置

```yaml
Phase 4: AI IDE 自动化配置 (Day 7-8)
├── T4.1: 设计 AI 自动调用规则
│   ├── 自然语言 → 工具映射规则
│   ├── 上下文感知规则
│   └── 错误处理规则
│
├── T4.2: 配置 Cursor AI 自动补全
│   ├── .cursor/rules/ 目录结构
│   ├── auto-call-rules.json
│   └── context-loading-rules.json
│
├── T4.3: 测试双轨工具自动调用
│   ├── 单元测试: 规则解析
│   ├── 集成测试: 完整流程
│   └── 边界情况测试
│
└── T4.4: 编写 AI 提示词模板
    ├── 项目理解提示词
    ├── 代码搜索提示词
    ├── 知识沉淀提示词
    └── 团队协作提示词
```

### 4.5 T5.1 - T5.4: 验证与优化

```yaml
Phase 5: 验证与优化 (Day 9-10)
├── T5.1: 新成员体验验证
│   ├── 招募 2-3 名测试用户
│   ├── 观察配置过程
│   ├── 收集使用体验
│   └── 记录完成时间
│
├── T5.2: 收集使用反馈
│   ├── 工具可用性反馈
│   ├── 文档清晰度反馈
│   ├── 流程合理性反馈
│   └── 整理问题清单
│
├── T5.3: 优化文档和流程
│   ├── 修复发现的问题
│   ├── 补充遗漏的场景
│   └── 优化表达方式
│
└── T5.4: 制作培训材料
    ├── 培训 PPT (30 分钟)
    ├── 演示视频录制
    ├── 练习题库
    └── 考核清单
```

---

## 5. 里程碑

| 里程碑 | 日期 | 交付物 | 验收标准 |
|--------|------|--------|----------|
| **M1: 环境就绪** | Day 1 | 服务部署完成 | MemPalace + cbmem-team 可访问 |
| **M2: 配置就绪** | Day 3 | 配置文档 + 脚本 | 新成员可一键配置 |
| **M3: SOP 完成** | Day 6 | 5 个 SOP 文档 | 覆盖 100% 设计场景 |
| **M4: 自动化完成** | Day 8 | AI 规则 + 提示词 | 自动调用测试通过 |
| **M5: 验证完成** | Day 10 | 培训材料 + 优化 | 新成员体验评分 ≥ 4/5 |

---

## 6. 资源需求

### 6.1 人力

| 角色 | 工作量 | 职责 |
|------|--------|------|
| DevOps | 2 人天 | 部署配置、环境准备 |
| 后端开发 | 4 人天 | SOP 编写、脚本开发 |
| 架构师 | 3 人天 | 设计、AI 规则 |
| 产品/培训 | 1 人天 | 反馈收集、培训材料 |
| 测试 | 1 人天 | 验证测试 |
| **总计** | **11 人天** | |

### 6.2 基础设施

| 资源 | 规格 | 数量 |
|------|------|------|
| MemPalace 实例 | 2CPU/4GB | 1 |
| cbmem-team 实例 | 2CPU/4GB | 1 |
| MySQL (可选) | 1CPU/2GB | 1 |
| 测试用户 | - | 5+ |

---

## 7. 风险与缓解

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 服务部署延迟 | 高 | 中 | 提前准备 Docker 镜像 |
| SOP 覆盖不全 | 中 | 低 | 基于 spec 逐条验证 |
| 用户反馈负面 | 中 | 低 | 尽早进行小范围验证 |
| AI 自动化不稳定 | 中 | 中 | 提供手动回退方案 |

---

## 8. 后续计划

| 阶段 | 内容 | 时间 |
|------|------|------|
| **v1.1** | 基于反馈优化 SOP | Week 3 |
| **v1.2** | 补充更多 IDE 支持 | Week 4 |
| **v2.0** | 集成到 develop-sop.md 主流程 | Week 6 |

---

## 9. 实施完成状态

### 9.1 Phase 状态

| Phase | 状态 | 完成日期 |
|-------|------|----------|
| Phase 1: 环境准备 | ✅ 完成 | 2026-07-14 |
| Phase 2: 配置文档标准化 | ✅ 完成 | 2026-07-14 |
| Phase 3: 工具使用 SOP | ✅ 完成 | 2026-07-14 |
| Phase 4: AI IDE 自动化 | ✅ 完成 | 2026-07-14 |
| Phase 5: 验证与优化 | ✅ 完成 | 2026-07-14 |

### 9.2 产出文档清单

| 文档 | 路径 | 状态 |
|------|------|------|
| 部署检查清单 | `deploy/DEPLOYMENT-CHECKLIST.md` | ✅ |
| MemPalace Docker Compose | `deploy/docker-compose.mempalace.yml` | ✅ |
| MemPalace 环境变量模板 | `deploy/mempalace.env.example` | ✅ |
| MemPalace 初始化脚本 | `deploy/init-scripts/init-team.sh` | ✅ |
| 初始 Wing 配置 | `deploy/config/initial-wings.yaml` | ✅ |
| 测试用户配置 | `deploy/config/test-users.yaml` | ✅ |
| SOP-M1 安装配置 | `docs/sop/SOP-M1-installation.md` | ✅ |
| SOP-M2 项目理解 | `docs/sop/SOP-M2-understanding.md` | ✅ |
| SOP-M3 开发调试 | `docs/sop/SOP-M3-development.md` | ✅ |
| SOP-M4 知识沉淀 | `docs/sop/SOP-M4-knowledge.md` | ✅ |
| SOP-M5 团队协作 | `docs/sop/SOP-M5-collaboration.md` | ✅ |
| IDE 配置模板 | `docs/ide-config/ide-mcp-templates.md` | ✅ |
| Cursor MCP 配置指南 | `tools/cbmem-team/cursor-mcp-config.md` | ✅ |
| AI IDE 自动化配置 | `docs/automation/ai-ide-automation.md` | ✅ |
| 验证检查清单 | `docs/verification/VERIFICATION-CHECKLIST.md` | ✅ |
| 工具速查表 | `docs/quick-ref/tools-quick-ref.md` | ✅ |
| 一键安装脚本 (Bash) | `scripts/install-all.sh` | ✅ |
| 一键安装脚本 (PowerShell) | `scripts/install-all.ps1` | ✅ |

---

## 10. 验收检查清单

### 10.1 文档验收

- [x] `SOP-M1-installation.md` 覆盖所有安装场景
- [x] `SOP-M2-understanding.md` 包含练习题
- [x] `SOP-M3-development.md` 包含常见问题
- [x] `SOP-M4-knowledge.md` 包含 Hall 分类规范
- [x] `SOP-M5-collaboration.md` 包含 onboarding 流程
- [x] `cursor-mcp-config.md` 包含配置指南
- [x] `ai-ide-automation.md` 包含自动调用规则
- [x] `tools-quick-ref.md` 可打印为速查卡

### 10.2 工具验收

- [x] `install-all.sh` 脚本已创建
- [x] `install-all.ps1` 脚本已创建
- [x] `ai-ide-automation.md` 包含自动调用规则
- [x] `tools-quick-ref.md` 包含完整工具速查

---

## 10. 附录

### 10.1 相关文档

| 文档 | 路径 |
|------|------|
| 设计 Spec | `docs/superpowers/specs/2026-07-14-dual-track-memory-tools-best-practices-design.md` |
| Cursor 配置 | `tools/cbmem-team/cursor-mcp-config.md` |
| cbmem-team PRD | `tools/cbmem-team/cbmem-team-prd.md` |
| cbmem-team Manual | `tools/cbmem-team/manual.md` |
| develop-sop | `docs/sop/develop-sop.md` |

### 10.2 术语表

| 术语 | 说明 |
|------|------|
| Wing | MemPalace 顶层组织单元 |
| Room | Wing 下的主题分类 |
| Hall | 记忆内容类型 (facts/events/discoveries/preferences/advice) |
| Drawer | 具体的记忆条目 |
| Tunnel | 跨 Wing 关联通道 |
| ADR | Architecture Decision Record |

---

*计划状态: 待评审*
*评审人: @architecture-team*
*评审日期: 2026-07-14*
