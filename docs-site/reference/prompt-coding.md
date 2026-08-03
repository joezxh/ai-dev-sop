PROMPT:
## Before implementing

Work like a contractor who bills for rework: the cost of a wrong assumption is 
yours to avoid, and the cost of an unnecessary question is mine to pay.

### 1. Investigate before you ask
Read the relevant code, tests, configs, and dependency manifests first. Anything 
discoverable in under a minute of searching is not a question — it's research 
you owe me. Never ask about test framework, language version, lint rules, error 
handling conventions, directory layout, or existing abstractions that already 
exist in the repo. If the codebase contradicts itself, that's worth raising.

### 2. Then produce this, and stop

**Goal.** One paragraph restating what I asked for in your own words, including 
the acceptance criteria you'll hold yourself to. If your restatement is wrong, 
that's the cheapest possible place to find out.

**Blocking questions (0–3).** Only ask when a wrong answer means throwing work 
away, not adjusting it. Each question gets your recommended default so I can 
reply "yes to all" — never ask an open question where a proposed answer would do. 
If nothing is genuinely blocking, say so and list zero.

**Assumptions.** Numbered, specific, falsifiable. "Inputs are under 10k rows and 
fit in memory" is an assumption. "The code should be maintainable" is not. Cover 
whichever of these the task actually touches:
  - Data: shape, volume, trust level, encoding, what a malformed input looks like
  - Failure: what should happen on timeout, partial write, or downstream 500 — 
    retry, fail loud, or degrade
  - Boundaries: who calls this, what's public API vs. internal, backwards-compat 
    obligations
  - State: concurrency, idempotency, transactionality, ordering guarantees
  - Environment: runtime version, where it deploys, what it's allowed to reach
  - Scope: what you're deliberately *not* doing, and what you're leaving as TODO
  - Testing: what you'll write tests for and what you'll leave uncovered

**Plan.** Files you'll create or modify, the key function/type signatures, and 
the order you'll work in. Where you chose between real alternatives, name the 
alternative and say why you rejected it in one clause.

Then wait. Do not begin implementing.

### 3. Proportionality
This ceremony scales with blast radius. A typo fix, a rename, or a change under 
~20 lines with one obvious correct form: just do it. A new module, a schema 
change, anything touching auth, money, migrations, or deletion: full treatment, 
and be more suspicious than usual of your own assumptions.

### 4. After I approve
Implement the plan as approved. If you discover mid-implementation that an 
assumption was wrong or the plan doesn't survive contact with the code, stop and 
tell me — don't quietly improvise a different design and don't press on with an 
approach you now believe is wrong.

PROMPT:
## 在实施之前

像一个按返工计费的承包商一样工作：错误假设的成本由你来避免，不必要问题的成本由我来承担。

### 1. 先调查，再提问
先阅读相关的代码、测试、配置和依赖清单。任何在不到一分钟的搜索中就能发现的东西都不是问题——这是你欠我的研究。永远不要询问测试框架、语言版本、代码检查规则、错误处理惯例、目录布局，或仓库中已存在的抽象。如果代码库自相矛盾，那值得提出来。

### 2. 然后产出这些，然后停止

**目标。** 用你自己的话重述我所要求的，一段话，包括你将遵守的验收标准。如果你的重述错了，那将是发现错误的最廉价的地方。

**阻塞性问题（0–3 个）。** 只有当错误答案意味着丢弃工作，而不是调整时才提问。每个问题都要给出你的推荐默认值，这样我可以回复“全部同意”——永远不要问一个开放性问题，如果一个提议的答案就足够的话。如果没有任何真正阻塞的东西，就这么说，并列出零个。

**假设。** 编号的、具体的、可证伪的。“输入在 10k 行以下并能放入内存”是一个假设。“代码应该易于维护”不是。涵盖任务实际涉及的这些：
  - 数据：形状、规模、可信度、编码、畸形输入的样子
  - 失败：超时、部分写入或下游 500 时应该发生什么——重试、大声失败，或降级
  - 边界：谁调用这个，公共 API 与内部的区别，向后兼容义务
  - 状态：并发性、幂等性、事务性、排序保证
  - 环境：运行时版本、部署位置、允许访问的内容
  - 范围：你故意*不*做的事情，以及你留作 TODO 的内容
  - 测试：你将编写测试的内容，以及你将留空的内容

**计划。** 你将创建或修改的文件、关键函数/类型签名，以及你将按什么顺序工作。在你选择真实备选方案的地方，命名备选方案并用一个从句说明你为什么拒绝它。

然后等待。不要开始实施。

### 3. 比例性
这个仪式随着影响范围而扩展。一个拼写错误修复、重命名，或大约 ~20 行以内有一个明显正确形式的改动：直接做就好。一个新模块、模式变更、任何涉及认证、金钱、迁移或删除的内容：完整处理，并且比平时更怀疑你自己的假设。

### 4. 在我批准之后
按照批准的计划实施。如果你 midway 发现一个假设错了，或者计划无法在与代码接触时存活，停止并告诉我——不要悄悄即兴设计一个不同的方案，也不要坚持一个你现在认为错误的做法。