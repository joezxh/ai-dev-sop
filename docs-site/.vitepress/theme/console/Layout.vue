<!--
  .vitepress/theme/console/Layout.vue

  Console layout — VitePress picks this Layout for any markdown page
  that declares `layout: console` in its frontmatter. All other
  pages (the bulk of the docs site) keep the default VitePress
  Layout, so markdown browsing is unaffected.

  The router shell is a hash-mode SPA: /console routes to a vue-router-
  style internal page (Users / Projects / Sessions / Summarize /
  Distill / Login). External routes /sop/* and /reference/* still
  render normally because they're regular markdown files that don't
  set layout: console.
-->
<template>
  <div class="console-shell">
    <header class="topbar">
      <h1>双轨记忆控制台</h1>
      <div class="user">
        <span v-if="session.sessionId">{{ session.sessionId }}</span>
        <span v-else>未登录</span>
        <el-button size="small" @click="onLogout">退出</el-button>
      </div>
    </header>
    <div class="body">
      <aside>
        <ul>
          <li :class="{active: consoleHash === '/users'}">
            <a href="#/users">用户</a>
          </li>
          <li :class="{active: consoleHash === '/projects'}">
            <a href="#/projects">项目</a>
          </li>
          <li :class="{active: consoleHash === '/sessions'}">
            <a href="#/sessions">会话</a>
          </li>
          <li :class="{active: consoleHash === '/summarize'}">
            <a href="#/summarize">归纳</a>
          </li>
          <li :class="{active: consoleHash === '/distill'}">
            <a href="#/distill">蒸馏</a>
          </li>
        </ul>
      </aside>
      <main>
        <component :is="current" />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useSessionStore } from './store/session'
import Login from './pages/Login.vue'
import Users from './pages/Users.vue'
import Projects from './pages/Projects.vue'
import Sessions from './pages/Sessions.vue'
import Summarize from './pages/Summarize.vue'
import Distill from './pages/Distill.vue'

const session = useSessionStore()

// Hash route: read from window.location.hash, default to /users.
// SSR-safe — server has no window, so return default.
const consoleHash = computed(() => {
  if (typeof window === 'undefined') return '/users'
  const h = window.location.hash || '#/users'
  return h.startsWith('#') ? h.slice(1) : h
})

const current = computed(() => {
  if (!session.sessionId) return Login
  const r = consoleHash.value
  if (r.startsWith('/projects')) return Projects
  if (r.startsWith('/sessions')) return Sessions
  if (r.startsWith('/summarize')) return Summarize
  if (r.startsWith('/distill')) return Distill
  return Users
})

function onLogout() {
  session.logout()
  window.location.hash = '#/users'
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
}
.topbar h1 { margin: 0; font-size: 16px; }
.user { display: flex; gap: 8px; align-items: center; }
.body { display: flex; height: calc(100vh - 40px); }
aside { width: 200px; background: #f0f2f5; padding: 16px 0; }
aside ul { list-style: none; padding: 0; margin: 0; }
aside li { padding: 10px 20px; }
aside li.active { background: #1890ff; color: white; }
aside a { color: inherit; text-decoration: none; }
main { flex: 1; overflow: auto; padding: 24px; }
</style>
