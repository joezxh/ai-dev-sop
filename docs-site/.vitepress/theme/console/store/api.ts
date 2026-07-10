// .vitepress/theme/console/store/api.ts
import { useSessionStore } from './session'

const BASE = '/api/console'

export async function api<T = any>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const s = useSessionStore()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(init.headers as Record<string, string> | undefined),
  }
  if (s.csrfToken) headers['X-CSRF-Token'] = s.csrfToken
  const r = await fetch(`${BASE}${path}`, { ...init, headers, credentials: 'same-origin' })
  const env = await r.json()
  if (env.code !== 0) throw new Error(env.msg || 'request failed')
  return env.data
}

export const UsersAPI = {
  list: (q = '') => api(`/users?q=${encodeURIComponent(q)}`),
  create: (body: any) => api(`/users`, { method: 'POST', body: JSON.stringify(body) }),
  update: (id: string, body: any) => api(`/users/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  remove: (id: string) => api(`/users/${id}`, { method: 'DELETE' }),
}

export const ProjectsAPI = {
  list: (q = '') => api(`/projects?q=${encodeURIComponent(q)}`),
  create: (body: any) => api(`/projects`, { method: 'POST', body: JSON.stringify(body) }),
  remove: (id: string) => api(`/projects/${id}`, { method: 'DELETE' }),
  reindex: (id: string) => api(`/projects/${id}/reindex`, { method: 'POST' }),
}

export const SessionsAPI = {
  list: (q = '') => api(`/sessions?${q}`),
  detail: (id: string) => api(`/sessions/${id}`),
  stats: () => api(`/sessions-stats`),
}

export const SummarizeAPI = {
  trigger: (body: any) => api(`/summarize`, { method: 'POST', body: JSON.stringify(body) }),
  get: (id: string) => api(`/summarize/${id}`),
}

export const DistillAPI = {
  trigger: (body: any) => api(`/distill`, { method: 'POST', body: JSON.stringify(body) }),
  get: (id: string) => api(`/distill/${id}`),
  commit: (id: string, target_wing: string) =>
    api(`/distill/${id}/commit`, { method: 'POST', body: JSON.stringify({ target_wing }) }),
}
