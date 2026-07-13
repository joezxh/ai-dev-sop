// .vitepress/theme/index.ts
import DefaultTheme from 'vitepress/theme'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import AppLayout from './Layout.vue'

export default {
  extends: DefaultTheme,
  Layout: AppLayout,
  enhanceApp({ app }) {
    // pinia requires a per-app instance; VitePress SSR re-creates the
    // app on each request, so we attach a fresh pinia here rather
    // than at module scope (which would be shared across requests
    // and break isolation).
    app.use(createPinia())
    // The console SPA (layout: console) relies on element-plus
    // components (el-button, el-table, el-dialog, ...) and the
    // ElMessage / ElMessageBox helpers, so register the library
    // globally. Without this, those components are unresolved and the
    // console renders empty.
    app.use(ElementPlus)
  },
}
