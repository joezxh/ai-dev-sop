// .vitepress/theme/console/store/session.ts
import { defineStore } from 'pinia'

export const useSessionStore = defineStore('console-session', {
  state: () => ({
    sessionId: '' as string,
    csrfToken: '' as string,
  }),
  actions: {
    async login(adminToken: string) {
      const r = await fetch('/api/console/login', {
        method: 'POST',
        headers: { 'X-Admin-Token': adminToken },
      })
      const env = await r.json()
      if (env.code !== 0) throw new Error(env.msg)
      this.sessionId = env.data.session_id
      this.csrfToken = env.data.csrf_token
    },
    logout() {
      fetch('/api/console/logout', {
        method: 'POST',
        headers: { 'X-CSRF-Token': this.csrfToken },
      })
      this.sessionId = ''
      this.csrfToken = ''
    },
  },
})
