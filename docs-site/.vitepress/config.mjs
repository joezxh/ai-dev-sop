import { defineConfig } from 'vitepress'
import { createRequire } from 'node:module'
import { withMermaid } from 'vitepress-plugin-mermaid'

const require = createRequire(import.meta.url)

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

export default withMermaid(defineConfig({
  title: 'AI 开发 SOP',
  description: 'adsop-platform 全栈开发者 SOP 文档',

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
              { text: '指南', link: '/guide/intro' },
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
              text: '概述',
              collapsible: true,
              items: [
                { text: '总览', link: '/sop/' },
                { text: '0.1 目标与用途', link: '/sop/overview/goals' },
                { text: '0.2 角色与阶段', link: '/sop/overview/roles' },
                { text: '0.3-0.4 流程图', link: '/sop/overview/flow' },
                { text: '0.5 产出物', link: '/sop/overview/deliverables' },
                { text: '0.6 核心收益', link: '/sop/overview/benefits' },
                { text: '0.7 运营自动化', link: '/sop/overview/operations' },
                { text: '0.8 实施状态', link: '/sop/overview/status' },
                { text: '0.9 AI 架构', link: '/sop/overview/architecture' }
              ]
            },
            {
              text: '指南',
              collapsible: true,
              items: [
                { text: '索引', link: '/guide/' },
                { text: '快速入门', link: '/guide/intro' },
                { text: 'Skill 安装', link: '/guide/skills' },
                { text: '常见问题', link: '/guide/faq' },
                { text: '更新日志', link: '/guide/changelog' }
              ]
            },
            {
              text: '§1 准备',
              collapsible: true,
              items: [
                { text: '1.1 读取工程', link: '/sop/prepare/map-codebase' },
                { text: '1.2 Skill 安装', link: '/sop/prepare/skill-install' },
                { text: '1.3 Project Rules', link: '/sop/prepare/project-rules' },
                { text: '1.4 IDE 配置', link: '/sop/prepare/ide-config' }
              ]
            },
            {
              text: '§2 提示词生成',
              collapsible: true,
              items: [
                { text: '2.1 已有工程提示词', link: '/sop/prompts/existing-project' },
                { text: '2.2 需求转提示词', link: '/sop/prompts/demand-to-prompt' },
                { text: '2.3 子模块补齐', link: '/sop/prompts/submodule' },
                { text: '2.4 PRD 文档', link: '/sop/prompts/prd' },
                { text: '2.5 一句话开发', link: '/sop/prompts/one-sentence' }
              ]
            },
            {
              text: '§3 测试开发',
              collapsible: true,
              items: [
                { text: '3.1 全新开发', link: '/sop/testing/new-development' },
                { text: '3.2 批量执行', link: '/sop/testing/batch-execution' },
                { text: '3.3 测试执行', link: '/sop/testing/execution' }
              ]
            },
            {
              text: '§4 场景 SOP',
              collapsible: true,
              items: [
                { text: '4.1 基线脚手架', link: '/sop/scenes/framework' },
                { text: '4.2 一句话原型', link: '/sop/scenes/one-sentence' },
                { text: '4.3 文档自动化', link: '/sop/scenes/docs-automation' },
                { text: '4.4 复制 Web', link: '/sop/scenes/copy-web' },
                { text: '4.5 复制 App', link: '/sop/scenes/copy-app' },
                { text: '4.6 升级迁移', link: '/sop/scenes/upgrade' },
                { text: '4.7 部署', link: '/sop/scenes/deploy' }
              ]
            },
            {
              text: '§5 双轨记忆',
              collapsible: true,
              items: [
                { text: '5.1 架构总览', link: '/sop/memory/overview' },
                { text: '5.2 服务端部署', link: '/sop/memory/deploy' },
                { text: '5.3 多租户 Bridge', link: '/sop/memory/bridge' },
                { text: '5.4 客户端配置', link: '/sop/memory/client' },
                { text: '5.5 开发手册', link: '/sop/memory/dev-guide' },
                { text: '5.6 管理手册', link: '/sop/memory/admin-guide' },
                { text: '5.7 组织管理', link: '/sop/memory/org-guide' },
                { text: '5.8 MCP 工具', link: '/sop/memory/tools' },
                { text: '5.8.x 工具最佳实践', link: '/sop/memory/tools-best-practices' },
                { text: '5.9 故障排查', link: '/sop/memory/troubleshooting' },
                { text: '5.10 最佳实践', link: '/sop/memory/best-practices' },
                { text: '5.11 总览索引', link: '/sop/memory/summary' },
                { text: '5.12 Self-check', link: '/sop/memory/selfcheck' }
              ]
            },
            {
              text: '§6 文档编写',
              collapsible: true,
              items: [
                { text: '6.1 产品设计', link: '/sop/writing/product-design' },
                { text: '6.2 市场调研', link: '/sop/writing/market-research' },
                { text: '6.3 模板使用', link: '/sop/writing/template-guide' }
              ]
            }
          ],
          '/reference/': [
            {
              text: '配套文档',
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
    optimizeDeps: {
      include: ['dayjs'],
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
