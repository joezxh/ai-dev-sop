// .vitepress/theme/console/store/session.ts
//
// JWT-based session store (M2/M7 auth). Replaces the legacy cookie-based
// session store used by the v1 console.
//
// Auth flow:
//   1. login(username, password) → POST /api/auth/login → { access_token, refresh_token, user, must_change_password }
//   2. access_token stored in localStorage, sent as Bearer token on every request
//   3. On 401 response, automatic token refresh via POST /api/auth/refresh
//   4. logout() → POST /api/auth/logout → clear tokens
//
// Guarding: Components check session.isLoggedIn / session.role to show/hide UI.
// Route guards in Layout.vue redirect to /login when unauthenticated.
//

import { defineStore } from 'pinia'

export interface User {
  id: string
  username: string
  email?: string
  role: string   // admin | lead | developer | viewer
  must_change_password: boolean
  last_login_at?: string
}

export const useSessionStore = defineStore('console-session-v2', {
  state: () => ({
    accessToken: '' as string,
    refreshToken: '' as string,
    user: null as User | null,
    mustChangePassword: false,
  }),

  getters: {
    isLoggedIn: (s) => !!s.accessToken && !!s.user,
    role: (s) => s.user?.role ?? 'viewer',
    isAdmin: (s) => s.user?.role === 'admin',
    username: (s) => s.user?.username ?? '',
    userId: (s) => s.user?.id ?? '',
    requirePasswordChange: (s) => s.mustChangePassword,
  },

  actions: {
    /** Restore session from localStorage (call once on app init) */
    restore() {
      this.accessToken = localStorage.getItem('cb_access') ?? ''
      this.refreshToken = localStorage.getItem('cb_refresh') ?? ''
      const userJson = localStorage.getItem('cb_user')
      if (userJson) {
        try { this.user = JSON.parse(userJson) } catch { /* ignore */ }
      }
      this.mustChangePassword = localStorage.getItem('cb_must_chg_pwd') === '1'
    },

    /** Exchange username + password for JWT tokens */
    async login(username: string, password: string) {
      const r = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      const env = await r.json()
      if (env.code !== 0) throw new Error(env.msg || '登录失败')

      const data = env.data as {
        access_token: string
        refresh_token: string
        expires_in: number
        must_change_password: boolean
        user: User
      }

      this.accessToken = data.access_token
      this.refreshToken = data.refresh_token
      this.user = data.user
      this.mustChangePassword = data.must_change_password

      // Persist
      localStorage.setItem('cb_access', data.access_token)
      localStorage.setItem('cb_refresh', data.refresh_token)
      localStorage.setItem('cb_user', JSON.stringify(data.user))
      localStorage.setItem('cb_must_chg_pwd', data.must_change_password ? '1' : '0')
    },

    /** Refresh access token using refresh token */
    async refreshAccessToken(): Promise<boolean> {
      if (!this.refreshToken) return false
      try {
        const r = await fetch('/api/auth/refresh', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: this.refreshToken }),
        })
        const env = await r.json()
        if (env.code !== 0) {
          this.clear()
          return false
        }
        const data = env.data as { access_token: string; refresh_token: string }
        this.accessToken = data.access_token
        this.refreshToken = data.refresh_token
        localStorage.setItem('cb_access', data.access_token)
        localStorage.setItem('cb_refresh', data.refresh_token)
        return true
      } catch {
        this.clear()
        return false
      }
    },

    /** Change password and clear the must_change_password flag */
    async changePassword(oldPassword: string, newPassword: string) {
      const r = await fetch('/api/auth/change-password', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${this.accessToken}`,
        },
        body: JSON.stringify({ old_password: oldPassword, new_password: newPassword }),
      })
      const env = await r.json()
      if (env.code !== 0) throw new Error(env.msg || '修改密码失败')

      // Update must_change_password flag locally
      if (this.user) {
        this.user = { ...this.user, must_change_password: false }
        localStorage.setItem('cb_user', JSON.stringify(this.user))
      }
      this.mustChangePassword = false
      localStorage.setItem('cb_must_chg_pwd', '0')
    },

    /** Revoke tokens and clear session */
    async logout() {
      if (this.accessToken) {
        try {
          await fetch('/api/auth/logout', {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${this.accessToken}`,
            },
          })
        } catch {
          // best-effort
        }
      }
      this.clear()
    },

    clear() {
      this.accessToken = ''
      this.refreshToken = ''
      this.user = null
      this.mustChangePassword = false
      localStorage.removeItem('cb_access')
      localStorage.removeItem('cb_refresh')
      localStorage.removeItem('cb_user')
      localStorage.removeItem('cb_must_chg_pwd')
    },
  },
})
