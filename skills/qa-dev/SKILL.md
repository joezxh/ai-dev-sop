---
name: qa-dev
description: 集成QA完整测试与自动开发的二合一Skill。具备/qa的全部严格测试功能，支持三级测试级别、健康分数评分、自动修复与原子提交。
triggers:
  - qa-dev
  - 测试开发一体化
  - test and develop
  - 自动测试开发
  - qa-dev batch
allowed-tools:
  - Read
  - Glob
  - Grep
  - Write
  - Edit
  - Bash
  - AskUserQuestion
  - Task
  - Shell
  - CallMcpTool
---

# /qa-dev Skill

**集成QA完整测试、自动修复与批量开发的二合一智能Skill**

**继承 `/qa` 的全部严格测试能力 + 测试开发一体化**

---

## 核心能力

```
┌────────────────────────────────────────────────────────────────────────────┐
│                              /qa-dev                                       │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  ✓ 继承 /qa 全部测试能力                                                  │
│  ├── 三级测试级别 (Quick/Standard/Exhaustive)                            │
│  ├── 健康分数评分 (0-100)                                                │
│  ├── Diff-aware 智能测试                                                 │
│  ├── 响应时间监控                                                        │
│  ├── 原子提交修复                                                        │
│  └── 自动回归测试生成                                                     │
│                                                                            │
│  Phase 1: 完整场景测试                                                   │
│  ├── 读取测试场景文件中所有测试场景                                        │
│  ├── MCP Playwright 端到端测试                                            │
│  ├── 前后端错误信息收集                                                   │
│  └── 控制台+网络请求监控                                                  │
│                              ↓                                             │
│  Phase 2: 问题分析与自动修复                                              │
│  ├── 分析前端错误 (控制台错误)                                            │
│  ├── 分析后端错误 (API响应/日志)                                         │
│  ├── 自动修复能力 (原子提交)                                             │
│  └── 触发缺失功能开发                                                    │
│                              ↓                                             │
│  Phase 3: 回归验证                                                       │
│  ├── 自动化重新测试                                                      │
│  └── 回归测试生成                                                        │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 使用方式

```bash
# ========== 批量执行 (推荐) ==========

# 完整场景测试 + 自动修复 + 开发 (Standard级别)
/qa-dev --batch --file ai-scene.md

# 快速测试 (关键路径)
/qa-dev --batch --tier quick --file ai-scene.md

# 穷尽测试 (包含低优先级问题)
/qa-dev --batch --tier exhaustive --file ai-scene.md

# 仅测试 + 生成报告
/qa-dev --batch --test-only --file ai-scene.md

# ========== 单模块执行 ==========

# 完整流程: 测试 → 分析 → 修复 → 开发 → 验证
/qa-dev --file ai-scene.md --module AI-01

# 指定测试级别
/qa-dev --file ai-scene.md --module AI-01 --tier exhaustive

# 仅测试
/qa-dev --test-only --file ai-scene.md --module AI-01

# 仅开发
/qa-dev --dev-only --module AI-01
```

---

## 三级测试级别

### 级别说明

| 级别 | 说明 | 修复范围 | 适用场景 |
|------|------|---------|---------|
| `--tier quick` | 关键路径测试 | Critical + High | 快速验证 |
| `--tier standard` | 标准测试 | + Medium | **默认** |
| `--tier exhaustive` | 穷尽测试 | + Low + 样式 | 全面审查 |

### 级别对应问题修复

```
Quick (关键路径):
├── Critical: ✅ 自动修复
└── High: ✅ 自动修复

Standard (标准):
├── Critical: ✅ 自动修复
├── High: ✅ 自动修复
└── Medium: ✅ 自动修复

Exhaustive (穷尽):
├── Critical: ✅ 自动修复
├── High: ✅ 自动修复
├── Medium: ✅ 自动修复
└── Low/Cosmetic: ✅ 自动修复
```

---

## 健康分数评分

### 评分体系

| 维度 | 权重 | 评分标准 |
|------|------|---------|
| Console | 15% | 0错误=100, 1-3错误=70, 4-10错误=40, 10+=10 |
| Links | 10% | 0断链=100, 每断一个-15 |
| Visual | 10% | Critical=-25, High=-15, Medium=-8, Low=-3 |
| Functional | 20% | Critical=-25, High=-15, Medium=-8, Low=-3 |
| UX | 15% | Critical=-25, High=-15, Medium=-8, Low=-3 |
| Performance | 10% | 响应时间超标扣分 |
| Content | 5% | Critical=-25, High=-15, Medium=-8, Low=-3 |
| Accessibility | 15% | Critical=-25, High=-15, Medium=-8, Low=-3 |

### 最终分数计算

```
score = Σ (category_score × weight)
```

### 健康分数标准

| 分数范围 | 等级 | 说明 |
|---------|------|------|
| 90-100 | 优秀 | 应用质量良好 |
| 70-89 | 良好 | 有少量问题 |
| 50-69 | 一般 | 需要修复 |
| 30-49 | 较差 | 存在严重问题 |
| 0-29 | 危险 | 核心功能不可用 |

---

## Diff-aware 智能测试

### 自动检测分支变更

当没有指定URL且在特性分支上时，自动进入Diff-aware模式：

```bash
# 1. 分析分支差异
git diff main...HEAD --name-only
git log main..HEAD --oneline

# 2. 识别受影响的页面
# Controller/Router → 确定URL路径
# View/Component → 确定渲染页面
# Model/Service → 确定数据依赖
# CSS/Style → 确定影响范围

# 3. 智能测试策略
# 只测试变更相关的功能
# 不测试未变更的功能
```

### Diff-aware 测试流程

```
┌────────────────────────────────────────────────────────────────────────────┐
│                         Diff-aware 模式                                   │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  Step 1: 分析分支差异                                                   │
│  ├── git diff main...HEAD --name-only                                   │
│  ├── 识别变更文件类型                                                   │
│  └── 确定影响范围                                                       │
│                                                                            │
│  Step 2: 映射到页面                                                    │
│  ├── 前端变更 → 对应页面                                               │
│  ├── 后端变更 → 对应API                                                │
│  └── 配置变更 → 影响页面                                               │
│                                                                            │
│  Step 3: 智能测试                                                      │
│  ├── 只测试变更相关的页面                                               │
│  ├── 验证变更是否生效                                                  │
│  └── 检查是否有回归问题                                                 │
│                                                                            │
│  Step 4: 回归检查                                                      │
│  └── 验证相邻功能不受影响                                              │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 响应时间监控

### 性能基准

| 操作类型 | 基准时间 | 超时时间 |
|---------|---------|---------|
| 页面加载 | < 2s | 5s |
| API请求 | < 500ms | 2s |
| 表单提交 | < 1s | 3s |
| 搜索响应 | < 1s | 3s |
| 分页切换 | < 500ms | 2s |

### 性能测试脚本

```bash
#!/bin/bash
# 响应时间监控脚本

MODULE_ID="AI-01"
PAGE_PATH="/ai/model"

echo "=== 性能测试: $MODULE_ID ==="

# 1. 页面加载时间
echo "[1/5] 页面加载时间"
START=$(date +%s%3N)
$B goto "http://localhost:5173#$PAGE_PATH"
$B wait-for-load-state "networkidle"
END=$(date +%s%3N)
LOAD_TIME=$((END - START))
echo "加载时间: ${LOAD_TIME}ms"

if [ "$LOAD_TIME" -gt 5000 ]; then
    echo "⚠️ 页面加载超时 (>5s)"
fi

# 2. API响应时间
echo "[2/5] API响应时间"
START=$(date +%s%3N)
curl -s "http://localhost:8080/admin-api/ai/model/page" > /dev/null
END=$(date +%s%3N)
API_TIME=$((END - START))
echo "API响应: ${API_TIME}ms"

if [ "$API_TIME" -gt 2000 ]; then
    echo "⚠️ API响应超时 (>2s)"
fi

# 3. 表格渲染时间
echo "[3/5] 表格渲染时间"
START=$(date +%s%3N)
$B wait-for-selector ".ant-table"
END=$(date +%s%3N)
RENDER_TIME=$((END - START))
echo "渲染时间: ${RENDER_TIME}ms"

# 4. 搜索响应时间
echo "[4/5] 搜索响应时间"
$B fill "input[placeholder*='搜索']" "test"
START=$(date +%s%3N)
$B click "button:has-text('搜索')"
$B wait-for-load-state "networkidle"
END=$(date +%s%3N)
SEARCH_TIME=$((END - START))
echo "搜索响应: ${SEARCH_TIME}ms"

# 5. 分页响应时间
echo "[5/5] 分页响应时间"
START=$(date +%s%3N)
$B click ".ant-pagination-item-2"
$B wait-for-load-state "networkidle"
END=$(date +%s%3N)
PAGE_TIME=$((END - START))
echo "分页响应: ${PAGE_TIME}ms"

# 汇总
echo ""
echo "=== 性能汇总 ==="
echo "页面加载: ${LOAD_TIME}ms"
echo "API响应: ${API_TIME}ms"
echo "表格渲染: ${RENDER_TIME}ms"
echo "搜索响应: ${SEARCH_TIME}ms"
echo "分页响应: ${PAGE_TIME}ms"
```

---

## Phase 1: 完整场景测试

### 1.1 测试场景解析

读取测试场景文件，提取所有测试场景:

```bash
# 1. 读取测试场景文件
CONTENT=$(cat "docs/scene/ai-scene.md")

# 2. 提取模块列表
MODULES=$(echo "$CONTENT" | grep -E "^### [A-Z]+-[0-9]+" | sed 's/^### //')

# 3. 提取每个模块的测试场景
SCENARIOS=$(echo "$CONTENT" | grep -E "^#### 测试场景[0-9]+:" | sed 's/^#### //')

# 4. 提取测试步骤
STEPS=$(echo "$CONTENT" | grep -E "^[0-9]+\. " | sed 's/^[0-9]*\. //')
```

### 1.2 完整测试执行

对每个测试场景执行完整测试:

```bash
#!/bin/bash
# 完整场景测试脚本

MODULE_ID="AI-01"
MODULE_NAME="模型配置"
PAGE_PATH="/ai/model"
TIER="${TIER:-standard}"

echo "=== 测试模块: $MODULE_ID $MODULE_NAME (Tier: $TIER) ==="

# 1. 导航到页面
echo "[1/X] 导航到页面"
$B goto "http://localhost:5173#$PAGE_PATH"
$B wait-for-load-state "networkidle"
sleep 2

# 2. 性能测试
echo ""
echo "[性能] 页面加载时间测试"
START=$(date +%s%3N)
$B wait-for-selector ".ant-table" --timeout 10000
END=$(date +%s%3N)
LOAD_TIME=$((END - START))
echo "加载时间: ${LOAD_TIME}ms"

# 3. 控制台错误检查
echo ""
echo "[2/X] 控制台错误检查"
$B console --errors > "logs/${MODULE_ID}-console.log"
if grep -q "Error" "logs/${MODULE_ID}-console.log"; then
    echo "⚠️ 发现控制台错误"
    # 提取错误信息
    ERROR_MSG=$(grep "Error" "logs/${MODULE_ID}-console.log" | head -1)
    echo "错误: $ERROR_MSG"
fi

# 4. 测试场景执行
echo ""
echo "[3/X] 执行测试场景"

# 场景1: 列表加载
echo "=== 场景1: 列表加载 ==="
$B snapshot -i -a -o "screenshots/${MODULE_ID}-scene1.png"
$B wait-for-selector ".ant-table"

# 场景2: 新增模型
echo "=== 场景2: 新增模型 ==="
$B click "button:has-text('新增')"
$B wait-for-selector ".ant-modal"
$B snapshot -i -a -o "screenshots/${MODULE_ID}-scene2-form.png"

# 填写表单
$B fill "input[placeholder*='名称']" "Test Model"
$B click "button:has-text('确定')"
sleep 2

# 验证结果
$B console --errors > "logs/${MODULE_ID}-scene2-console.log"

# 场景3: 编辑模型
echo "=== 场景3: 编辑模型 ==="
$B click ".ant-table-row:first-child button:has-text('编辑')"
$B wait-for-selector ".ant-modal"
$B snapshot -i -a -o "screenshots/${MODULE_ID}-scene3-form.png"

# 场景4: 删除模型
echo "=== 场景4: 删除模型 ==="
$B click ".ant-table-row:first-child button:has-text('删除')"
$B wait-for-selector ".ant-popover"
$B click "button:has-text('确定')"
sleep 2

echo ""
echo "[完成] 测试完成"
```

### 1.3 错误信息收集

#### 前端控制台错误

```bash
# 获取控制台错误
CONSOLE_ERRORS=$($B console --errors)

# 解析错误类型
if echo "$CONSOLE_ERRORS" | grep -q "Error"; then
    echo "发现前端错误:"
    echo "$CONSOLE_ERRORS"

    # 提取错误信息和堆栈
    ERROR_MSG=$(echo "$CONSOLE_ERRORS" | grep -oP '(?<=Error:).*' | head -1)
    ERROR_STACK=$(echo "$CONSOLE_ERRORS" | grep -oP '(?<=at ).*' | head -3)
fi
```

#### 后端API错误

```bash
# 测试API端点
API_URL="http://localhost:8080/admin-api/ai/model/page"

# 获取HTTP状态码
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL")

# 获取响应内容
RESPONSE=$(curl -s "$API_URL")

echo "HTTP状态码: $HTTP_CODE"
echo "响应内容: $RESPONSE"
```

---

## Phase 2: 问题分析与自动修复

### 2.1 问题分类与严重级别

| 分类 | 级别 | 修复策略 |
|------|------|---------|
| 页面404 | Critical | 立即修复 |
| API 500错误 | Critical | 立即修复 |
| 核心功能失效 | High | 立即修复 |
| 组件加载失败 | High | 立即修复 |
| 表单验证错误 | Medium | Standard+修复 |
| 样式显示问题 | Medium | Standard+修复 |
| 性能问题 | Medium | Standard+修复 |
| 样式细节 | Low | Exhaustive修复 |

### 2.2 原子提交修复

每个问题独立修复，单独提交：

```bash
#!/bin/bash
# 原子提交修复脚本

ISSUE_ID="ISSUE-001"
ISSUE_TITLE="API接口404错误"
ISSUE_FILE="src/api/ai/model.ts"
FIX_DESCRIPTION="添加缺失的create接口"

echo "=== 修复问题: $ISSUE_ID ==="

# 1. 修复代码
# ... 执行修复 ...

# 2. 验证修复
$B goto "http://localhost:5173/#/ai/model"
$B click "button:has-text('新增')"
$B wait-for-selector ".ant-modal"

# 3. 截图验证
$B screenshot "screenshots/${ISSUE_ID}-fixed.png"

# 4. 原子提交
git add "$ISSUE_FILE"
git commit -m "fix(qa): $ISSUE_ID — $ISSUE_TITLE

修复内容:
- $FIX_DESCRIPTION

Issue: $ISSUE_ID"

echo "✓ 原子提交完成"
```

### 2.3 自动修复能力

#### 前端错误修复

```bash
fix_frontend_error() {
    local ERROR_MSG="$1"
    local ERROR_TYPE="$2"

    case "$ERROR_TYPE" in
        "组件未导入")
            COMPONENT=$(echo "$ERROR_MSG" | grep -oP '(?<=Cannot find component ).*' | tr -d "'")
            echo "修复: 导入组件 $COMPONENT"
            fix_import "$COMPONENT"
            ;;

        "API路径错误")
            API_PATH=$(echo "$ERROR_MSG" | grep -oP '"/[^"]+"' | head -1)
            echo "修复: 修正API路径 $API_PATH"
            fix_api_path "$API_PATH"
            ;;

        "v-model绑定问题")
            echo "修复: 检查v-model绑定"
            fix_vmodel
            ;;
    esac
}
```

#### 后端错误修复

```bash
fix_backend_error() {
    local HTTP_CODE="$1"
    local RESPONSE="$2"

    ERROR_CODE=$(echo "$RESPONSE" | grep -oP '"code"\s*:\s*\K[0-9-]+')
    ERROR_MSG=$(echo "$RESPONSE" | grep -oP '"msg"\s*:\s*"\K[^"]+')

    case "$HTTP_CODE" in
        "404")
            echo "修复: 后端接口不存在"
            trigger_backend_dev
            ;;

        "500")
            echo "修复: 后端服务器错误"
            check_backend_logs
            trigger_backend_fix
            ;;
    esac
}
```

---

## Phase 3: 回归验证

### 3.1 验证流程

```bash
# 开发完成后重新测试所有场景
echo "=== 回归验证 ==="

for SCENE in 1 2 3 4; do
    echo "验证测试场景$SCENE..."
    execute_scene "$SCENE"

    if check_result "$SCENE"; then
        echo "✓ 场景$SCENE 通过"
    else
        echo "✗ 场景$SCENE 失败"
    fi
done
```

### 3.2 回归测试生成

为每个修复生成单元测试：

```bash
#!/bin/bash
# 生成回归测试脚本

ISSUE_ID="ISSUE-001"
ISSUE_FILE="src/api/ai/model.ts"
TEST_FILE="src/api/ai/__tests__/model.regression.test.ts"

echo "=== 生成回归测试: $ISSUE_ID ==="

# 1. 读取修复的源文件
# 2. 分析修复的代码路径
# 3. 生成测试用例

cat > "$TEST_FILE" << 'EOF'
// Regression: ISSUE-001 — API接口404错误
// Found by /qa-dev on {date}
// Report: .gstack/qa-reports/

import { describe, it, expect } from 'vitest';
import { modelApi } from '../model';

describe('Model API - ISSUE-001 Regression', () => {
    it('should create model successfully', async () => {
        const result = await modelApi.create({
            name: 'Test Model',
            type: 'GPT-4'
        });
        expect(result.code).toBe(0);
    });
});
EOF

# 4. 运行测试
npm test "$TEST_FILE"

# 5. 提交测试
git add "$TEST_FILE"
git commit -m "test(qa): regression test for ISSUE-001"
```

---

## 批量执行流程

### 执行流程图

```
┌────────────────────────────────────────────────────────────────────────────┐
│                        /qa-dev --batch --file ai-scene.md                │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                            │
│  Step 1: 解析测试场景文件                                                 │
│  └── 提取所有模块、所有场景、所有步骤                                       │
│                                                                            │
│  Step 2: 逐个模块测试 (继承/qa全部能力)                                  │
│  ├── 三级测试级别 (Quick/Standard/Exhaustive)                            │
│  ├── 健康分数评分                                                         │
│  ├── 响应时间监控                                                         │
│  └── 前后端错误收集                                                       │
│                                                                            │
│  Step 3: 问题分析                                                         │
│  ├── 前端错误分析                                                         │
│  ├── 后端错误分析                                                         │
│  └── 严重级别分类                                                         │
│                                                                            │
│  Step 4: 自动修复 (原子提交)                                             │
│  ├── 前端修复                                                             │
│  ├── 后端修复                                                             │
│  └── 回归测试生成                                                          │
│                                                                            │
│  Step 5: 回归验证                                                         │
│  └── 所有模块重新测试                                                     │
│                                                                            │
│  Step 6: 健康分数汇总                                                     │
│  └── docs/scene/batch-report-{date}.md                                   │
│                                                                            │
└────────────────────────────────────────────────────────────────────────────┘
```

### 批量执行参数

| 参数 | 说明 | 示例 |
|------|------|------|
| --batch | 批量模式 | - |
| --file | 测试场景文件 | ai-scene.md |
| --tier | 测试级别 | quick/standard/exhaustive |
| --module | 指定模块 | AI-01 |
| --from | 起始模块 | AI-01 |
| --to | 结束模块 | AI-10 |
| --test-only | 仅测试 | - |
| --dev-only | 仅开发 | - |
| --failed | 仅失败模块 | - |

---

## 测试报告模板

```markdown
# /qa-dev 测试报告

**执行时间**: {timestamp}
**文件**: {scene-file}
**模块**: {MODULE_ID}
**测试级别**: {TIER}

## 健康分数

| 维度 | 分数 | 权重 | 加权分数 |
|------|------|------|---------|
| Console | {N} | 15% | {N} |
| Links | {N} | 10% | {N} |
| Visual | {N} | 10% | {N} |
| Functional | {N} | 20% | {N} |
| UX | {N} | 15% | {N} |
| Performance | {N} | 10% | {N} |
| Content | {N} | 5% | {N} |
| Accessibility | {N} | 15% | {N} |
| **总分** | | 100% | **{SCORE}** |

**健康等级**: {优秀/良好/一般/较差/危险}

## 测试场景执行情况

| # | 场景名称 | 状态 | 性能 | 错误类型 |
|---|---------|------|------|---------|
| 1 | 列表加载 | ✅ 通过 | 1200ms | - |
| 2 | 新增模型 | ❌ 失败 | - | API 404 |
| 3 | 编辑模型 | ⚠️ 跳过 | - | 依赖场景2 |
| 4 | 删除模型 | ⚠️ 跳过 | - | 依赖场景2 |

## 问题详情

### ISSUE-001: API接口不存在
- **严重级别**: Critical
- **类型**: 后端API错误
- **HTTP状态码**: 404
- **修复状态**: ✅ 已修复
- **提交SHA**: {sha}
- **修复文件**: src/api/ai/model.ts

### ISSUE-002: 页面加载超时
- **严重级别**: High
- **类型**: 性能问题
- **响应时间**: 8500ms (基准: 2000ms)
- **修复状态**: ⏳ 待修复

## 原子提交记录

| # | Commit SHA | 问题 | 状态 |
|---|------------|------|------|
| 1 | abc1234 | ISSUE-001 API接口404 | ✅ |
| 2 | def5678 | ISSUE-002 性能问题 | ⏳ |

## 回归测试

| 测试文件 | 状态 |
|---------|------|
| model.regression.test.ts | ✅ 通过 |

## 性能报告

| 操作 | 响应时间 | 基准 | 状态 |
|------|---------|------|------|
| 页面加载 | 1200ms | <2000ms | ✅ |
| API响应 | 450ms | <500ms | ✅ |
| 搜索响应 | 1200ms | <1000ms | ⚠️ |
| 分页响应 | 300ms | <500ms | ✅ |
```

---

## 与 /qa 能力对比

| 能力 | /qa | /qa-dev | 说明 |
|------|-----|---------|------|
| 三级测试级别 | ✅ | ✅ | 相同 |
| 健康分数评分 | ✅ | ✅ | 相同 |
| Diff-aware模式 | ✅ | ✅ | 相同 |
| 响应时间监控 | ✅ | ✅ | 相同 |
| 原子提交修复 | ✅ | ✅ | 相同 |
| 自动回归测试 | ✅ | ✅ | 相同 |
| 批量无人值守 | ❌ | ✅ | /qa-dev独有 |
| 测试+开发一体 | ❌ | ✅ | /qa-dev独有 |
| 场景文件驱动 | ❌ | ✅ | /qa-dev独有 |

---

## 输出文件

```
docs/scene/batch-report-{date}.md    # 批量执行报告
.gstack/qa-reports/                   # QA测试报告
screenshots/                          # 截图
logs/                                 # 日志
src/**/__tests__/*.test.ts          # 回归测试
```

---

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0.0 | 2026-06-29 | 初始版本 |
| 1.1.0 | 2026-06-29 | 添加批量执行 |
| 2.0.0 | 2026-06-29 | 集成浏览器测试、报告生成 |
| 2.1.0 | 2026-06-29 | 完整场景测试、错误收集 |
| 3.0.0 | 2026-06-29 | **集成/qa全部严格测试功能** |
