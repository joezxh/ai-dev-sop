<!-- .vitepress/theme/console/Layout.vue -->
<template>
  <div class="console-shell">
    <header class="topbar">
      <h1>双轨记忆控制台</h1>
      <div class="user">
        <span>admin@local</span>
        <el-button size="small" @click="onLogout">退出</el-button>
      </div>
    </header>
    <div class="body">
      <aside>
        <ul>
          <li :class="{active: route === '/users'}"><a href="#/users">用户</a></li>
          <li :class="{active: route === '/projects'}"><a href="#/projects">项目</a></li>
          <li :class="{active: route === '/sessions'}"><a href="#/sessions">会话</a></li>
          <li :class="{active: route === '/summarize'}"><a href="#/summarize">归纳</a></li>
          <li :class="{active: route === '/distill'}"><a href="#/distill">蒸馏</a></li>
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
import { useRoute, navigate } from './router'
import { useSessionStore } from './store/session'
import Login from './pages/Login.vue'
import Users from './pages/Users.vue'
import Projects from './pages/Projects.vue'
import Sessions from './pages/Sessions.vue'
import Summarize from './pages/Summarize.vue'
import Distill from './pages/Distill.vue'

const route = useRoute()
const session = useSessionStore()

const current = computed(() => {
  const r = route.value
  if (!session.sessionId) return Login
  if (r.startsWith('/projects')) return Projects
  if (r.startsWith('/sessions')) return Sessions
  if (r.startsWith('/summarize')) return Summarize
  if (r.startsWith('/distill')) return Distill
  return Users
})

function onLogout() {
  session.logout()
  navigate('/users')
}
</script>

<style scoped>
.console-shell { font-family: -apple-system, BlinkMacSystemFont, sans-serif; height: 100vh; }
.topbar { display: flex; justify-content: space-between; align-items: center; padding: 8px 16px; background: #001529; color: white; }
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
