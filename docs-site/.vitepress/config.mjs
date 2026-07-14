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
            {
              text: 'SOP 入口',
              collapsible: true,
              items: [
                { text: '研发 SOP 主体', link: '/sop/' },
                { text: 'QA-Dev Skill', link: '/guide/intro' },
                { text: 'AI Skill 字典', link: '/guide/skills' },
                { text: 'Repo Wiki 技术参考', link: '/sop/repo-wiki-tech' },
                { text: '更新日志', link: '/guide/changelog' }
              ]
            },
            {
              text: '§4 Pipeline',
              collapsible: true,
              items: [
                { text: '一句话 Pipeline', link: '/sop/one-sentence-pipeline' },
                { text: '框架 Pipeline', link: '/sop/framework-pipeline' },
                { text: '文档 Pipeline', link: '/sop/docs-pipeline' },
                { text: '复制 Web', link: '/sop/copy-web-pipeline' },
                { text: '复制 App', link: '/sop/copy-app-pipeline' },
                { text: 'Java 升级迁移', link: '/sop/java-upgrade-pipeline' }
              ]
            },
            {
              text: '§5 范例开发文档',
              collapsible: true,
              items: [
                { text: 'System 服务场景', link: '/sop/system-scene' },
                { text: 'UAA 场景', link: '/sop/uaa-scene' }
              ]
            }
          ],
          '/reference/': [
            {
              text: '配套文档（指向 SOP 主体）',
              items: [
                { text: '场景模板', link: '/reference/scene-template' },
                { text: '一句话 Pipeline', link: '/reference/one-sentence-pipeline' },
                { text: '框架 Pipeline', link: '/reference/framework-pipeline' },
                { text: '文档 Pipeline', link: '/reference/docs-pipeline' },
                { text: '复制 Web Pipeline', link: '/reference/copy-web-pipeline' },
                { text: '复制 App Pipeline', link: '/reference/copy-app-pipeline' },
                { text: '升级 Pipeline', link: '/reference/java-upgrade-pipeline' }
              ]
            }
          ],'/guide/': [
            {
              text: '使用指南',
              items: [
                { text: '全局开发SOP', link: '/guide/index' },
                { text: '开发模板', link: '/guide/scene-template' },
                { text: 'QA-Dev Skill', link: '/guide/intro' },
                { text: 'AI Skill 字典', link: '/guide/skills' }
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
      },
    },
  }
}))
