// .vitepress/theme/index.ts
import DefaultTheme from 'vitepress/theme'
import { createPinia } from 'pinia'
import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'
import AppLayout from './Layout.vue'

export default {
  extends: DefaultTheme,
  Layout: AppLayout,
  enhanceApp({ app }) {
    app.use(createPinia())
    app.use(Antd)
  },
}
