<!--
  .vitepress/theme/console/Layout.vue

  Console layout — VitePress picks this Layout for any markdown page
  that declares `layout: console` in its frontmatter. All other
  pages (the bulk of the docs site) keep the default VitePress
  Layout, so markdown browsing is unaffected.

  Auth: JWT-based via session store. On mount, session.restore() is
  called to rehydrate tokens from localStorage. If not logged in,
  the Login page is shown unconditionally.
-->
<template>
  <div class="console-shell">
    <header class="topbar">
      <h1>控制台</h1>
      <div class="user">
        <template v-if="session.isLoggedIn">
          <span class="username">{{ session.username }}</span>
          <a-tag :color="roleColor">{{ session.role }}</a-tag>
          <a-button size="small" @click="onLogout">退出</a-button>
        </template>
        <template v-else>
          <span class="not-logged-in">未登录</span>
        </template>
      </div>
    </header>

    <div class="body">
      <aside v-if="session.isLoggedIn">
        <ul>
          <!-- Teams (admin/lead/developer) -->
          <li v-if="canViewTeams" :class="{active: route.startsWith('/teams')}" @click="navigate('/teams')">
            团队
          </li>
          <!-- Projects (all members) -->
          <li :class="{active: route.startsWith('/projects')}" @click="navigate('/projects')">
            项目
          </li>
          <!-- Modules (all members) -->
          <li :class="{active: route.startsWith('/modules')}" @click="navigate('/modules')">
            模块
          </li>
          <!-- Memories (all members) -->
          <li :class="{active: route.startsWith('/memories')}" @click="navigate('/memories')">
            记忆
          </li>
          <!-- AI Tools (developer+) -->
          <li v-if="canViewAITools" :class="{active: route.startsWith('/ai-tools')}" @click="navigate('/ai-tools')">
            AI 工具
          </li>
          <li class="divider" />
          <!-- Sessions (all members) -->
          <li :class="{active: route.startsWith('/sessions')}" @click="navigate('/sessions')">
            会话
          </li>
          <!-- Summarize (all members) -->
          <li :class="{active: route.startsWith('/summarize')}" @click="navigate('/summarize')">
            归纳
          </li>
          <!-- Distill (all members) -->
          <li :class="{active: route.startsWith('/distill')}" @click="navigate('/distill')">
            蒸馏
          </li>
          <li class="divider" />
          <!-- Status (all members) -->
          <li :class="{active: route.startsWith('/status')}" @click="navigate('/status')">
            状态
          </li>
          <!-- Logs (all members) -->
          <li :class="{active: route.startsWith('/logs')}" @click="navigate('/logs')">
            日志
          </li>
        </ul>
      </aside>

      <main>
        <!-- Login page is always available for unauthenticated users -->
        <Login v-if="!session.isLoggedIn" />
        <component v-else :is="currentPage" />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useSessionStore } from './store/session'
import { useRoute } from './router'
import Login from './pages/Login.vue'
import Teams from './pages/Teams.vue'
import Projects from './pages/Projects.vue'
import Modules from './pages/Modules.vue'
import Memories from './pages/Memories.vue'
import Sessions from './pages/Sessions.vue'
import Summarize from './pages/Summarize.vue'
import Distill from './pages/Distill.vue'
import AITools from './pages/AITools.vue'
import Status from './pages/Status.vue'
import Logs from './pages/Logs.vue'

const session = useSessionStore()
const route = useRoute()

// Rehydrate JWT session from localStorage on mount
onMounted(() => {
  session.restore()
})

const roleColor = computed(() => {
  const r = session.role
  if (r === 'admin') return 'red'
  if (r === 'lead') return 'orange'
  if (r === 'developer') return 'blue'
  return 'default'
})

const canViewTeams = computed(() => ['admin', 'lead', 'developer'].includes(session.role))
const canViewAITools = computed(() => ['admin', 'lead', 'developer'].includes(session.role))

const currentPage = computed(() => {
  const r = route.value
  if (r.startsWith('/teams')) return Teams
  if (r.startsWith('/projects')) return Projects
  if (r.startsWith('/modules')) return Modules
  if (r.startsWith('/memories')) return Memories
  if (r.startsWith('/sessions')) return Sessions
  if (r.startsWith('/summarize')) return Summarize
  if (r.startsWith('/distill')) return Distill
  if (r.startsWith('/ai-tools')) return AITools
  if (r.startsWith('/status')) return Status
  if (r.startsWith('/logs')) return Logs
  return Teams
})

function navigate(path: string) {
  window.location.hash = '#' + path
}

function onLogout() {
  session.logout()
  window.location.hash = '#/login'
}
</script>

<style scoped>
.console-shell {
  font-family: -apple-system, BlinkMacSystemFont, 'PingFang SC', sans-serif;
  height: 100vh;
}
.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 16px;
  background: #001529;
  color: white;
  height: 48px;
  box-sizing: border-box;
}
.topbar h1 { margin: 0; font-size: 16px; }
.user { display: flex; gap: 8px; align-items: center; }
.username { color: #fff; font-size: 14px; }
.not-logged-in { color: #aaa; font-size: 14px; }
.body { display: flex; height: calc(100vh - 48px); }
aside { width: 180px; background: #f0f2f5; padding: 12px 0; flex-shrink: 0; }
aside ul { list-style: none; padding: 0; margin: 0; }
aside li { padding: 9px 20px; cursor: pointer; color: #333; font-size: 14px; }
aside li:hover { background: #e6f7ff; color: #1890ff; }
aside li.active { background: #1890ff; color: white; }
aside li.divider { height: 1px; background: #d9d9d9; margin: 6px 12px; padding: 0; cursor: default; }
aside li.divider:hover { background: #d9d9d9; color: inherit; }
main { flex: 1; overflow: auto; padding: 24px; background: #fafafa; }
</style>
