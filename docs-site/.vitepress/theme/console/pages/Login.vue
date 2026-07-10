<!-- .vitepress/theme/console/pages/Login.vue -->
<template>
  <div style="max-width: 360px; margin: 64px auto;">
    <el-card>
      <h2>登录</h2>
      <el-input v-model="token" placeholder="管理员 Token" type="password" />
      <el-button type="primary" :loading="loading" @click="onLogin" style="margin-top: 16px; width: 100%;">登录</el-button>
      <p v-if="error" style="color:red;">{{ error }}</p>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useSessionStore } from '../store/session'

const token = ref('')
const loading = ref(false)
const error = ref('')
const session = useSessionStore()

async function onLogin() {
  loading.value = true
  error.value = ''
  try {
    await session.login(token.value)
    window.location.hash = '#/users'
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>
