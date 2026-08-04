// .vitepress/theme/console/router.ts
import { ref, computed } from 'vue'

const route = ref(typeof window !== 'undefined' ? hashRoute() : '/users')

function hashRoute() {
  const h = window.location.hash || '#/users'
  return h.startsWith('#') ? h.slice(1) : h
}

if (typeof window !== 'undefined') {
  window.addEventListener('hashchange', () => { route.value = hashRoute() })
}

export function useRoute() {
  return computed(() => route.value)
}

export function navigate(path: string) {
  window.location.hash = `#${path}`
}
