<!-- .vitepress/theme/console/pages/Login.vue -->
<template>
  <div class="login-wrap">
    <!-- Change password dialog (forced on first login) -->
    <a-modal
      :open="showChangePwd"
      title="首次登录：修改密码"
      :maskClosable="false"
      :closable="false"
      :footer="null"
    >
      <a-form layout="vertical">
        <a-form-item label="用户名" :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
          <a-input :value="session.user?.username" disabled />
        </a-form-item>
        <a-form-item label="旧密码" required :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
          <a-input-password v-model:value="pwdForm.old" placeholder="请输入当前密码" />
        </a-form-item>
        <a-form-item label="新密码" required :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
          <a-input-password v-model:value="pwdForm.new_" placeholder="请输入新密码（至少8位）" />
        </a-form-item>
        <a-form-item label="确认密码" required :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
          <a-input-password v-model:value="pwdForm.confirm" placeholder="再次输入新密码" />
        </a-form-item>
        <a-alert v-if="pwdError" type="error" :message="pwdError" show-icon style="margin-bottom:12px" />
        <a-button type="primary" :loading="pwdLoading" block @click="onChangePassword">
          确认修改
        </a-button>
      </a-form>
    </a-modal>

    <!-- Login card -->
    <a-card class="login-card">
      <h2>控制台登录</h2>
      <a-form layout="vertical">
        <a-form-item label="用户名">
          <a-input v-model:value="form.username" placeholder="用户名" size="large" />
        </a-form-item>
        <a-form-item label="密码">
          <a-input-password v-model:value="form.password" placeholder="密码" size="large" @keyup.enter="onLogin" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" :loading="loading" block size="large" @click="onLogin">
            登录
          </a-button>
        </a-form-item>
      </a-form>
      <a-alert v-if="error" type="error" :message="error" show-icon style="margin-top: 8px" />
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useSessionStore } from '../store/session'

const session = useSessionStore()

const form = ref({ username: '', password: '' })
const loading = ref(false)
const error = ref('')

// Change password state
const showChangePwd = ref(false)
const pwdForm = ref({ old: '', new_: '', confirm: '' })
const pwdError = ref('')
const pwdLoading = ref(false)

// Show change-password dialog if the user is logged in but needs to change password
onMounted(() => {
  session.restore()
  if (session.isLoggedIn && session.requirePasswordChange) {
    showChangePwd.value = true
  }
})

async function onLogin() {
  if (!form.value.username || !form.value.password) {
    error.value = '请输入用户名和密码'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await session.login(form.value.username, form.value.password)
    if (session.requirePasswordChange) {
      showChangePwd.value = true
    } else {
      window.location.hash = '#/teams'
    }
  } catch (e: any) {
    error.value = e.message || '登录失败'
  } finally {
    loading.value = false
  }
}

async function onChangePassword() {
  pwdError.value = ''
  if (!pwdForm.value.old) { pwdError.value = '请输入旧密码'; return }
  if (!pwdForm.value.new_) { pwdError.value = '请输入新密码'; return }
  if (pwdForm.value.new_.length < 8) { pwdError.value = '新密码至少8位'; return }
  if (pwdForm.value.new_ !== pwdForm.value.confirm) { pwdError.value = '两次输入的密码不一致'; return }

  pwdLoading.value = true
  try {
    await session.changePassword(pwdForm.value.old, pwdForm.value.new_)
    showChangePwd.value = false
    window.location.hash = '#/teams'
  } catch (e: any) {
    pwdError.value = e.message || '修改密码失败'
  } finally {
    pwdLoading.value = false
  }
}
</script>

<style scoped>
.login-wrap {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 80vh;
}
.login-card {
  width: 100%;
  max-width: 400px;
}
</style>
