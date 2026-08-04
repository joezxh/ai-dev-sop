<script setup lang="ts">
import { useData } from 'vitepress'
import DefaultTheme from 'vitepress/theme'
import ConsoleLayout from './console/Layout.vue'

// Only use the console SPA shell for pages that explicitly opt in via
// `layout: console` in their frontmatter. Every other page (the docs, the
// homepage, 404, etc.) falls back to VitePress's default Layout.
const { frontmatter } = useData()
</script>

<template>
  <ConsoleLayout v-if="frontmatter.layout === 'console'" />
  <DefaultTheme.Layout v-else />
</template>
