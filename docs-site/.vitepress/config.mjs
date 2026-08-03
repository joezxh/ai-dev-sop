import { defineConfig } from 'vitepress'
import { createRequire } from 'node:module'
import { existsSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { withMermaid } from 'vitepress-plugin-mermaid'

const require = createRequire(import.meta.url)
const dayjsEsmPath = require.resolve('dayjs/esm/index.js')
const dayjsRoot = dirname(require.resolve('dayjs'))

// VitePress + element-plus need dayjs *plugins* (e.g. `dayjs/plugin/
// customParseFormat.js`, imported by `ElTimePicker`). Those plugin files are
// published as UMD (`module.exports = ...`) and the dayjs package is ESM-first,
// so when served raw they expose **no `default` export** — surfacing in the
// browser as:
//   The requested module '…/dayjs/plugin/customParseFormat.js' does not
//   provide an export named 'default'
// Redirect every `dayjs/plugin/<name>` (with or without `.js`) to the proper
// ESM build at `dayjs/esm/plugin/<name>/index.js`, which uses `export default`
// and resolves correctly in the browser. (The bare `dayjs` import keeps using
// the `dayjsEsmPath` alias declared below.)
const dayjsEsmPluginResolver = () => ({
  name: 'dayjs-esm-plugin-resolver',
  enforce: 'pre',
  resolveId(source) {
    if (typeof source !== 'string' || !source.startsWith('dayjs/plugin/')) {
      return null
    }
    const name = source.slice('dayjs/plugin/'.length).replace(/\.js$/, '')
    const esmPath = join(dayjsRoot, 'esm/plugin', name, 'index.js')
    if (existsSync(esmPath)) return esmPath
    return null
  },
})

// `vitepress-plugin-mermaid`'s `withMermaid` injects resolve aliases that map
// dayjs plugin imports (e.g. `dayjs/plugin/advancedFormat.js`) to their
// `esm/plugin/<name>` *directories* (dayjs/esm/plugin/advancedFormat). Vite
// cannot read a directory, which surfaces as `EISDIR: illegal operation on a
// directory`. We strip those `dayjs/plugin/*` aliases so the imports fall back
// to the real CJS wrapper files at `dayjs/plugin/<name>.js` (which exist and
// load fine in dev).
// `withMermaid` also injects bare specifiers into `optimizeDeps.include`:
//   "@braintree/sanitize-url", "dayjs", "debug",
//   "cytoscape-cose-bilkent", "cytoscape"
// These are now declared as direct dependencies in package.json, so pnpm
// hoists them to the project root and esbuild can resolve them natively
// (pre-bundling the CJS ones as ESM). That removes the previous
// "Failed to resolve dependency" startup warnings and the
// "does not provide an export named 'sanitizeUrl'" browser SyntaxError.

const withMermaidWithoutDayjs = (config) => {
  const result = withMermaid(config)
  // `dayjs` is a direct dependency (resolvable from root) and is also covered
  // by the bare `dayjs` alias below, so keep it OUT of optimizeDeps.include —
  // otherwise esbuild marks the aliased entry as external and errors with
  // "The entry point 'dayjs' cannot be marked as external".
  if (Array.isArray(result.vite?.optimizeDeps?.include)) {
    result.vite.optimizeDeps.include = result.vite.optimizeDeps.include.filter(
      (dep) => String(dep) !== 'dayjs'
    )
  }
  const alias = result.vite?.resolve?.alias
  const entries = alias
    ? (Array.isArray(alias)
        ? alias
        : Object.entries(alias).map(([find, replacement]) => ({ find, replacement })))
    : []
    const kept = entries.filter((a) => !String(a.find).startsWith('dayjs/plugin/'))
    result.vite = result.vite || {}
    result.vite.resolve = result.vite.resolve || {}
    result.vite.resolve.alias = kept
    // Redirect dayjs plugin subpath imports to their ESM builds (see
    // dayjsEsmPluginResolver above) so element-plus dayjs plugins resolve with
    // a proper default export in the browser.
    result.vite.plugins = result.vite.plugins || []
    if (!Array.isArray(result.vite.plugins)) {
      result.vite.plugins = [result.vite.plugins]
    }
    result.vite.plugins.push(dayjsEsmPluginResolver())
  return result
}

const i18n = {
  root: {
    lang: 'zh-CN',
    title: 'AI 开发 SOP',
    description: 'adsop-platform 全栈开发者 SOP 文档'
  },
  en: {
    lang: 'en-US',
    title: 'AI Development SOP',
    description: 'adsop-platform Full-stack Developer SOP Documentation'
  },
  ja: {
    lang: 'ja-JP',
    title: 'AI 開発 SOP',
    description: 'adsop-platform フルスタック開発者 SOP ドキュメント'
  }
}

export default withMermaidWithoutDayjs(defineConfig({
  title: 'AI 开发 SOP',
  description: 'adsop-platform 全栈开发者 SOP 文档',

  rewrites: {},

  locales: {
    root: {
      label: '简体中文',
      lang: 'zh-CN',
      link: '/',
      themeConfig: {
        nav: [
          { text: '首页', link: '/' },
          {
            text: '文档',
            items: [
              { text: '指南', link: '/guide/' },
              { text: 'SOP', link: '/sop/' },
              { text: '参考', link: '/reference/' },
              { text: '控制台', link: '/console/' }
            ]
          },
          {
            text: '语言',
            items: [
              { text: '简体中文', link: '/' },
              { text: 'English', link: '/en/' },
              { text: '日本語', link: '/ja/' }
            ]
          }
        ],
        sidebar: {
          '/sop/': [
            {text: '文档模板', link: '/sop/dev-template'},
            {text: '常规流程', link: '/sop/normal'},
            {
              text: '新工程制作',
              collapsible: true,
              items: [
                { text: '产品研究 <span style="font-size:0.75em">✅</span>', link: '/sop/prod-design' },
                { text: '需求到PRD <span style="font-size:0.75em">❌</span>', link: '/sop/requirement-prd' },
                { text: '需求到原型 <span style="font-size:0.75em">❌</span>', link: '/sop/requirement-prototype' },
                { text: 'UI设计 <span style="font-size:0.75em">❌</span>', link: '/sop/ui-design' },
                { text: '前端框架制作 <span style="font-size:0.75em">❓</span>', link: '/sop/frontend-fw' },
                { text: '后端框架制作 <span style="font-size:0.75em">❓</span>', link: '/guide/backend-fw' },
                { text: '基于框架开发 <span style="font-size:0.75em">❓</span>', link: '/sop/fw-dev' }
              ]
            },
            {
              text: '已有工程改造',
              collapsible: true,
              items: [
                { text: '已有工程文档制作 <span style="font-size:0.75em">❓</span>', link: '/sop/write-docs' },
                { text: '已有工程转提示词 <span style="font-size:0.75em">✅</span>', link: '/sop/exists-scene' },
                { text: '复制已有Web应用 <span style="font-size:0.75em">❓</span>', link: '/sop/copy-webapp' },
                { text: '复制已有移动应用 <span style="font-size:0.75em">❌</span>', link: '/sop/copy-uniapp' },
                { text: '前端升级 <span style="font-size:0.75em">❓</span>', link: '/sop/frontend-upgrade' },
                { text: '后端升级 <span style="font-size:0.75em">❓</span>', link: '/sop/backend-upgrade' }
              ]
            },
            {
              text: '范例开发文档',
              collapsible: true,
              items: [
                { text: 'System 服务场景', link: '/sop/system-scene' },
                { text: 'UAA 场景', link: '/sop/uaa-scene' }
              ]
            }
          ],
          '/reference/': [
            { text: 'AI-Coding 16个实战技巧', link: '/reference/16-coding-skill/16-coding-skill'},
            { text: 'Graph Engineering', link: '/reference/graph-engineering/graph-engineering'},
            { text: 'Codex 进阶指南', link: '/reference/codex-multi-agent-guide/codex-multi-agent-guide'},
            { text: 'FDE 到底是干嘛的', link: '/reference/fde-ai-position/fde-position'},
            {
              text: 'Agentic Design Patterns',
              collapsible: true,
              items: [
                { text: '概览', link: '/reference/agentic-design-patterns/README' },
                { text: '简介', link: '/reference/agentic-design-patterns/00-intro' },
                { text: '前言', link: '/reference/agentic-design-patterns/00-preface' },
                { text: '译者序', link: '/reference/agentic-design-patterns/00-translator' },
                { text: '智能体的特征', link: '/reference/agentic-design-patterns/01-intro-agent' },
                { text: '智能体未来：五大假设', link: '/reference/agentic-design-patterns/02-agent-hypotheses' },
                { text: '第1章：提示链', link: '/reference/agentic-design-patterns/ch01-prompt-chaining' },
                { text: '第2章：路由', link: '/reference/agentic-design-patterns/ch02-routing' },
                { text: '第3章：并行化', link: '/reference/agentic-design-patterns/ch03-parallelization' },
                { text: '第4章：反思', link: '/reference/agentic-design-patterns/ch04-reflection' },
                { text: '第5章：工具使用', link: '/reference/agentic-design-patterns/ch05-tool-use' },
                { text: '第6章：规划', link: '/reference/agentic-design-patterns/ch06-planning' },
                { text: '第7章：多智能体协作', link: '/reference/agentic-design-patterns/ch07-multi-agent' },
                { text: '第8章：记忆管理', link: '/reference/agentic-design-patterns/ch08-memory' },
                { text: '第9章：学习与适应', link: '/reference/agentic-design-patterns/ch09-learning' },
                { text: '第10章：模型上下文协议（MCP）', link: '/reference/agentic-design-patterns/ch10-mcp' },
                { text: '第11章：目标设定与监控', link: '/reference/agentic-design-patterns/ch11-goal-setting' },
                { text: '第12章：异常处理与恢复', link: '/reference/agentic-design-patterns/ch12-error-handling' },
                { text: '第13章：人类参与环节', link: '/reference/agentic-design-patterns/ch13-hitl' },
                { text: '第14章：知识检索（RAG）', link: '/reference/agentic-design-patterns/ch14-rag' },
                { text: '第15章：智能体间通信（A2A）', link: '/reference/agentic-design-patterns/ch15-a2a' },
                { text: '第16章：资源感知优化', link: '/reference/agentic-design-patterns/ch16-resource-awareness' },
                { text: '第17章：推理技术', link: '/reference/agentic-design-patterns/ch17-reasoning' },
                { text: '第18章：护栏与安全模式', link: '/reference/agentic-design-patterns/ch18-guardrails' },
                { text: '第19章：评估与监控', link: '/reference/agentic-design-patterns/ch19-evaluation' },
                { text: '第20章：优先级排序', link: '/reference/agentic-design-patterns/ch20-prioritization' },
                { text: '第21章：探索与发现', link: '/reference/agentic-design-patterns/ch21-exploration' }
              ]
            }
          ],'/guide/': [
            {
              text: '使用指南',
              items: [
                { text: '概述', link: '/guide/overview' },
                { text: '准备工作', link: '/guide/ready' },
                { text: '记忆tools字典', collapsible: true,
                  items: [
                    { text: '安装', link: '/guide/ready.md#1-记忆体安装' },
                    { text: '速查表', link: '/guide/mem-tools' },
                    { text: '项目理解', link: '/guide/SOP-M2-understanding' },
                    { text: '开发调试', link: '/guide/SOP-M3-development' },
                    { text: '知识沉淀', link: '/guide/SOP-M4-knowledge' },
                    { text: '团队协作', link: '/guide/SOP-M5-collaboration' }
                  ]},
                { text: 'AI Skill 字典', link: '/guide/skills' },
                { text: 'QA-Dev Skill', link: '/guide/qa-dev-intro' },
                { text: '市场规划', link: '/guide/market' }
              ]
            }
          ]
        },
        search: {
          provider: 'local',
          options: {
            detailedView: true
          }
        },
        socialLinks: [
          { icon: 'github', link: 'https://github.com/your-org/ai-dev-sop' }
        ],
        footer: {
          message: '基于 MIT 许可证发布',
          copyright: 'Copyright © 2024-present AI Dev SOP Team'
        },
        editLink: {
          pattern: 'https://github.com/your-org/ai-dev-sop/edit/main/docs-site/:path',
          text: '在 GitHub 上编辑此页面'
        },
        lastUpdated: {
          text: '最后更新',
          formatOptions: {
            dateStyle: 'short',
            timeStyle: 'short'
          }
        },
        docFooter: {
          prev: '上一页',
          next: '下一页'
        },
        returnToTopLabel: '返回顶部',
        sidebarMenuLabel: '菜单'
      }
    },
    en: {
      label: 'English',
      lang: 'en-US',
      link: '/en/',
      themeConfig: {
        nav: [
          { text: 'Home', link: '/en/' },
          {
            text: 'Docs',
            items: [
              { text: 'Guide', link: '/en/guide/intro' },
              { text: 'SOP', link: '/en/sop/' },
              { text: 'Reference', link: '/en/reference/' },
              { text: 'Console', link: '/en/console/' }
            ]
          },
          {
            text: 'Language',
            items: [
              { text: '简体中文', link: '/' },
              { text: 'English', link: '/en/' },
              { text: '日本語', link: '/ja/' }
            ]
          }
        ],
        sidebar: {
          '/en/sop/': [
            {
              text: 'Overview',
              collapsible: true,
              items: [
                { text: 'Index', link: '/en/sop/' },
                { text: '0.1 Goals', link: '/en/sop/overview/goals' },
                { text: '0.2 Roles', link: '/en/sop/overview/roles' },
                { text: '0.3-0.4 Flow', link: '/en/sop/overview/flow' },
                { text: '0.5 Deliverables', link: '/en/sop/overview/deliverables' },
                { text: '0.6 Benefits', link: '/en/sop/overview/benefits' },
                { text: '0.7 Operations', link: '/en/sop/overview/operations' },
                { text: '0.8 Status', link: '/en/sop/overview/status' },
                { text: '0.9 Architecture', link: '/en/sop/overview/architecture' }
              ]
            }
          ]
        },
        search: {
          provider: 'local'
        },
        socialLinks: [
          { icon: 'github', link: 'https://github.com/your-org/ai-dev-sop' }
        ]
      }
    },
    ja: {
      label: '日本語',
      lang: 'ja-JP',
      link: '/ja/',
      themeConfig: {
        nav: [
          { text: 'ホーム', link: '/ja/' },
          {
            text: 'ドキュメント',
            items: [
              { text: 'ガイド', link: '/ja/guide/intro' },
              { text: 'SOP', link: '/ja/sop/' },
              { text: 'リファレンス', link: '/ja/reference/' },
              { text: 'コンソール', link: '/ja/console/' }
            ]
          },
          {
            text: '言語',
            items: [
              { text: '简体中文', link: '/' },
              { text: 'English', link: '/en/' },
              { text: '日本語', link: '/ja/' }
            ]
          }
        ],
        sidebar: {
          '/ja/sop/': [
            {
              text: '概要',
              collapsible: true,
              items: [
                { text: 'インデックス', link: '/ja/sop/' },
                { text: '0.1 目標', link: '/ja/sop/overview/goals' },
                { text: '0.2 役割', link: '/ja/sop/overview/roles' }
              ]
            }
          ]
        },
        search: {
          provider: 'local'
        },
        socialLinks: [
          { icon: 'github', link: 'https://github.com/your-org/ai-dev-sop' }
        ]
      }
    }
  },

  themeConfig: {
    outline: {
      level: [2, 3],
      label: '目录'
    },
    langMenuLabel: '多语言'
  },

  markdown: {
    theme: {
      light: 'github-light',
      dark: 'github-dark'
    },
    lineNumbers: true
  },

  head: [
    ['link', { rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' }],
    ['meta', { name: 'theme-color', content: '#ffffff' }],
    ['meta', { name: 'og:type', content: 'website' }],
    ['meta', { name: 'og:site_name', content: 'AI Development SOP' }]
  ],

  cleanUrls: true,
  ignoreDeadLinks: true,
  lastUpdated: true,
  contributors: true,

  vite: {
    resolve: {
      // Only alias the bare `dayjs` import to its ESM entry so consumers get a
      // proper `dayjs` function. `dayjs/plugin/*` subpaths are NOT aliased here
      // — those are handled by real files (see withMermaidWithoutDayjs, which
      // removes the broken directory aliases that `withMermaid` adds).
      alias: [
        { find: /^dayjs$/, replacement: dayjsEsmPath }
      ]
    },
    optimizeDeps: {
      exclude: ['dayjs', 'element-plus']
    },
    // Local dev proxy: forward console API calls to the cbmem-team backend.
    // In production the vitepress `dist/` is served by gin under /console,
    // so /api/console/* hits the same process and no proxy is needed.
    server: {
      proxy: {
        '/api/console': {
          target: 'http://127.0.0.1:8787',
          changeOrigin: false,
        },
        '/api/auth': {
          target: 'http://127.0.0.1:8787',
          changeOrigin: false,
        },
      },
    },
  }
}))
