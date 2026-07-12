// .vitepress/theme/index.ts
import DefaultTheme from 'vitepress/theme'
import { createPinia } from 'pinia'
import ConsoleLayout from './console/Layout.vue'

export default {
  extends: DefaultTheme,
  Layout: ConsoleLayout,
  enhanceApp({ app }) {
    // pinia requires a per-app instance; VitePress SSR re-creates the
    // app on each request, so we attach a fresh pinia here rather
    // than at module scope (which would be shared across requests
    // and break isolation).
    app.use(createPinia())
  },
}
