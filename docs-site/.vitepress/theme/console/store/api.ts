// .vitepress/theme/console/store/api.ts
//
// Typed API client for the cbmem-team console API (v2 JWT-gated).
// All requests send the JWT Bearer token and auto-refresh on 401.
//
// V2 base path: /api/console/v2
// Auth base path: /api/auth
//

import { useSessionStore } from './session'

const V2 = '/api/console/v2'
const AUTH = '/api/auth'

// ---------------------------------------------------------------------------
// Core fetch wrapper with auto-refresh
// ---------------------------------------------------------------------------

let refreshing = false
let refreshQueue: (() => void)[] = []

async function apiCore<T = any>(
  path: string,
  init: RequestInit = {},
  base = V2,
): Promise<T> {
  const s = useSessionStore()

  const doFetch = (token: string) => {
    const url = base + path
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer ' + token,
    }
    const extra = init.headers as Record<string, string> | undefined
    if (extra) Object.assign(headers, extra)
    return fetch(url, { ...init, headers })
  }

  let r = await doFetch(s.accessToken)
  const env = await r.json()

  // Auto-refresh on 401
  if (r.status === 401 && s.refreshToken && !refreshing) {
    refreshing = true
    const ok = await s.refreshAccessToken()
    refreshing = false
    if (ok) {
      r = await doFetch(s.accessToken)
    } else {
      s.clear()
      window.location.hash = '#/login'
      throw new Error('会话已过期，请重新登录')
    }
    // Drain queue
    refreshQueue.forEach(fn => fn())
    refreshQueue = []
  } else if (r.status === 401) {
    // Queue while already refreshing
    return new Promise((resolve) => {
      refreshQueue.push(async () => {
        const r2 = await doFetch(s.accessToken)
        const env2 = await r2.json()
        resolve(env2 as T)
      })
    })
  }

  if (env.code !== 0) throw new Error(env.msg || '请求失败')
  return env.data as T
}

// ---------------------------------------------------------------------------
// Auth API (no JWT required for login; JWT required for me/change-password)
// ---------------------------------------------------------------------------

export const AuthAPI = {
  login: (username: string, password: string) => {
    return apiCore<{
      access_token: string
      refresh_token: string
      expires_in: number
      must_change_password: boolean
      user: any
    }>('/login', {
      method: 'POST',
      body: JSON.stringify({ username: username, password: password }),
    }, AUTH)
  },

  me: () => apiCore<any>('/me', {}, AUTH),

  changePassword: (oldPassword: string, newPassword: string) =>
    apiCore<any>('/change-password', {
      method: 'POST',
      body: JSON.stringify({ old_password: oldPassword, new_password: newPassword }),
    }, AUTH),
}

// ---------------------------------------------------------------------------
// Teams API
// ---------------------------------------------------------------------------

export const TeamsAPI = {
  list: () => apiCore<any[]>('/teams'),
  get: (id: string) => apiCore<any>('/teams/' + id),
  create: (body: any) => apiCore<any>('/teams', { method: 'POST', body: JSON.stringify(body) }),
  update: (id: string, body: any) =>
    apiCore<any>('/teams/' + id, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id: string) => apiCore<any>('/teams/' + id, { method: 'DELETE' }),

  // Members
  listMembers: (teamId: string) => apiCore<any[]>('/teams/' + teamId + '/members'),
  addMember: (teamId: string, body: any) =>
    apiCore<any>('/teams/' + teamId + '/members', { method: 'POST', body: JSON.stringify(body) }),
  updateMember: (teamId: string, uid: string, body: any) =>
    apiCore<any>('/teams/' + teamId + '/members/' + uid, { method: 'PUT', body: JSON.stringify(body) }),
  removeMember: (teamId: string, uid: string) =>
    apiCore<any>('/teams/' + teamId + '/members/' + uid, { method: 'DELETE' }),

  // Projects under a team
  listProjects: (teamId: string) => apiCore<any>('/teams/' + teamId + '/projects'),
  createProject: (teamId: string, body: any) =>
    apiCore<any>('/teams/' + teamId + '/projects', { method: 'POST', body: JSON.stringify(body) }),
}

// ---------------------------------------------------------------------------
// Projects API
// ---------------------------------------------------------------------------

export const ProjectsAPI = {
  get: (pid: string) => apiCore<any>('/projects/' + pid),
  update: (pid: string, body: any) =>
    apiCore<any>('/projects/' + pid, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (pid: string) => apiCore<any>('/projects/' + pid, { method: 'DELETE' }),
  clone: (pid: string) => apiCore<any>('/projects/' + pid + '/clone', { method: 'POST' }),

  // Indexing
  indexStatus: (pid: string) => apiCore<any>('/projects/' + pid + '/index-status'),
  reindex: (pid: string) => apiCore<any>('/projects/' + pid + '/reindex', { method: 'POST' }),

  // Sessions under a project
  projectSessions: (pid: string, query = '') =>
    apiCore<any>('/projects/' + pid + '/sessions' + (query ? '?' + query : '')),

  // Modules under a project
  projectModules: (pid: string) => apiCore<any[]>('/projects/' + pid + '/modules'),
  projectModulesTree: (pid: string) => apiCore<any[]>('/projects/' + pid + '/modules/tree'),
}

// ---------------------------------------------------------------------------
// Modules API
// ---------------------------------------------------------------------------

export const ModulesAPI = {
  get: (id: string) => apiCore<any>('/modules/' + id),
  create: (pid: string, body: any) =>
    apiCore<any>('/projects/' + pid + '/modules', { method: 'POST', body: JSON.stringify(body) }),
  update: (id: string, body: any) =>
    apiCore<any>('/modules/' + id, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id: string) => apiCore<any>('/modules/' + id, { method: 'DELETE' }),
  move: (id: string, body: any) =>
    apiCore<any>('/modules/' + id + '/move', { method: 'POST', body: JSON.stringify(body) }),
  memories: (id: string) => apiCore<any[]>('/modules/' + id + '/memories'),
}

// ---------------------------------------------------------------------------
// Sessions API (v2)
// ---------------------------------------------------------------------------

export const SessionsAPI = {
  list: (params: Record<string, string> = {}) => {
    const q = new URLSearchParams(params).toString()
    return apiCore<any>('/sessions' + (q ? '?' + q : ''))
  },
  stats: () => apiCore<any>('/sessions/stats'),
  get: (id: string) => apiCore<any>('/sessions/' + id),
}

// ---------------------------------------------------------------------------
// Memory Templates API
// ---------------------------------------------------------------------------

export const MemoryTemplatesAPI = {
  list: () => apiCore<any[]>('/memory-templates'),
  get: (id: string) => apiCore<any>('/memory-templates/' + id),
  render: (id: string, body: any) =>
    apiCore<any>('/memory-templates/' + id + '/render', { method: 'POST', body: JSON.stringify(body) }),
}

// ---------------------------------------------------------------------------
// Memories API
// ---------------------------------------------------------------------------

export const MemoriesAPI = {
  list: (params: Record<string, string> = {}) => {
    const q = new URLSearchParams(params).toString()
    return apiCore<any>('/memories' + (q ? '?' + q : ''))
  },
  create: (body: any) => apiCore<any>('/memories', { method: 'POST', body: JSON.stringify(body) }),
  get: (id: string) => apiCore<any>('/memories/' + id),
  update: (id: string, body: any) =>
    apiCore<any>('/memories/' + id, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id: string) => apiCore<any>('/memories/' + id, { method: 'DELETE' }),
  addTag: (id: string, tag: string) =>
    apiCore<any>('/memories/' + id + '/tags', { method: 'POST', body: JSON.stringify({ tag: tag }) }),
}

// ---------------------------------------------------------------------------
// Summarize / Distill Tasks API (v2)
// ---------------------------------------------------------------------------

export const SummarizeDistillAPI = {
  // Summarize tasks
  listSummarize: () => apiCore<any[]>('/summarize-tasks'),
  createSummarize: (body: any) =>
    apiCore<any>('/summarize-tasks', { method: 'POST', body: JSON.stringify(body) }),
  getSummarize: (id: string) => apiCore<any>('/summarize-tasks/' + id),

  // Distill tasks
  listDistill: () => apiCore<any[]>('/distill-tasks'),
  createDistill: (body: any) =>
    apiCore<any>('/distill-tasks', { method: 'POST', body: JSON.stringify(body) }),
  getDistill: (id: string) => apiCore<any>('/distill-tasks/' + id),
}

// ---------------------------------------------------------------------------
// AI Tools API
// ---------------------------------------------------------------------------

export const AIToolsAPI = {
  list: () => apiCore<any[]>('/ai-tools'),
  create: (body: any) => apiCore<any>('/ai-tools', { method: 'POST', body: JSON.stringify(body) }),
  get: (id: string) => apiCore<any>('/ai-tools/' + id),
  update: (id: string, body: any) =>
    apiCore<any>('/ai-tools/' + id, { method: 'PUT', body: JSON.stringify(body) }),
  delete: (id: string) => apiCore<any>('/ai-tools/' + id, { method: 'DELETE' }),
  invoke: (id: string, body: any) =>
    apiCore<any>('/ai-tools/' + id + '/invoke', { method: 'POST', body: JSON.stringify(body) }),
  listInvocations: (id: string) => apiCore<any[]>('/ai-tools/' + id + '/invocations'),
  getInvocation: (invId: string) => apiCore<any>('/ai-tools/invocations/' + invId),
}

// ---------------------------------------------------------------------------
// Dashboard / Status API
// ---------------------------------------------------------------------------

export const DashboardAPI = {
  summary: () => apiCore<any>('/dashboard/summary'),
  runtime: () => apiCore<any>('/status/runtime'),
  restart: () => apiCore<any>('/status/restart', { method: 'POST' }),
}
