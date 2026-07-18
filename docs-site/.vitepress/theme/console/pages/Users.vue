<!-- .vitepress/theme/console/pages/Users.vue -->
<template>
  <div>
    <div style="margin-bottom: 12px">
      <a-button type="primary" @click="openCreate">新增用户</a-button>
    </div>
    <a-table :dataSource="users" :columns="columns" :loading="loading" rowKey="id" pagination>
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'token_status'">
          <a-tag :color="record.disabled ? 'red' : 'green'">
            {{ record.disabled ? '停用' : '有效' }}
          </a-tag>
        </template>
        <template v-else-if="column.key === 'actions'">
          <a-space>
            <a-button size="small" @click="onEdit(record)">编辑</a-button>
            <a-button size="small" type="success" @click="openToken(record)">签发 JWT</a-button>
            <a-popconfirm title="确认删除？" @confirm="onDelete(record)">
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
          </a-space>
        </template>
      </template>
    </a-table>

    <!-- 新增/编辑用户对话框 -->
    <a-modal v-model:open="dialog" :title="editing ? '编辑用户' : '新增用户'" @ok="onSubmit" :confirmLoading="submitting">
      <a-form :model="formData" layout="vertical">
        <a-form-item label="用户 ID" name="id">
          <a-input v-model:value="formData.id" :disabled="!!editing" />
        </a-form-item>
        <a-form-item label="显示名" name="display_name">
          <a-input v-model:value="formData.display_name" />
        </a-form-item>
        <a-form-item label="项目路径" name="project_paths">
          <a-input v-model:value="pathsText" placeholder="逗号分隔" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 签发 JWT 对话框 -->
    <a-modal v-model:open="tokenDialog" title="签发 JWT" @ok="onMintToken" :confirmLoading="tokenLoading">
      <a-form layout="vertical">
        <a-form-item label="用户">
          <a-input :model-value="tokenUser?.id" disabled />
        </a-form-item>
        <a-form-item label="有效期 (TTL)">
          <a-select v-model:value="tokenTtl" style="width: 100%">
            <a-select-option value="1h">1 小时</a-select-option>
            <a-select-option value="12h">12 小时</a-select-option>
            <a-select-option value="24h">1 天</a-select-option>
            <a-select-option value="168h">7 天</a-select-option>
            <a-select-option value="720h">30 天（默认）</a-select-option>
            <a-select-option value="2160h">90 天</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- JWT 显示/复制对话框 -->
    <a-modal v-model:open="tokenResultDialog" title="JWT 签发成功" :footer="null" width="640px">
      <a-alert type="success" :message="`有效期至：${tokenResult?.expires}`" show-icon style="margin-bottom: 16px" />
      <div class="token-block">
        <pre class="token-text">{{ tokenResult?.token }}</pre>
        <a-button type="primary" class="copy-btn" :type="copied ? 'default' : 'primary'" @click="copyToken">
          {{ copied ? '已复制' : '复制 Token' }}
        </a-button>
      </div>
      <p class="token-tip">
        粘贴到 IDE MCP 配置的 <code>Authorization: Bearer &lt;token&gt;</code> 中使用。
      </p>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { UsersAPI } from '../store/api'

const users = ref<any[]>([])
const loading = ref(false)
const dialog = ref(false)
const submitting = ref(false)
const editing = ref<any>(null)
const formData = ref({ id: '', display_name: '', project_paths: [] })
const pathsText = computed({
  get: () => (formData.value.project_paths || []).join(','),
  set: (v: string) => { formData.value.project_paths = v.split(',').map((x: string) => x.trim()).filter(Boolean) },
})

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id' },
  { title: '显示名', dataIndex: 'display_name', key: 'display_name' },
  { title: 'Token 状态', key: 'token_status', width: 120 },
  { title: '操作', key: 'actions', width: 260 },
]

async function load() {
  loading.value = true
  try {
    const d = await UsersAPI.list()
    users.value = d.list || []
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editing.value = null
  formData.value = { id: '', display_name: '', project_paths: [] }
  pathsText.value = ''
  dialog.value = true
}

function onEdit(row: any) {
  editing.value = row
  formData.value = { ...row }
  pathsText.value = (row.project_paths || []).join(',')
  dialog.value = true
}

async function onSubmit() {
  submitting.value = true
  try {
    if (editing.value) {
      await UsersAPI.update(editing.value.id, formData.value)
      message.success('保存成功')
    } else {
      await UsersAPI.create(formData.value)
      message.success('创建成功')
    }
    dialog.value = false
    await load()
  } catch (e: any) {
    message.error(e?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

async function onDelete(row: any) {
  await UsersAPI.remove(row.id)
  message.success('已删除')
  await load()
}

// ---- JWT 签发 ----
const tokenDialog = ref(false)
const tokenLoading = ref(false)
const tokenUser = ref<any>(null)
const tokenTtl = ref('720h')
const tokenResultDialog = ref(false)
const tokenResult = ref<{ token: string; expires: string } | null>(null)
const copied = ref(false)

function openToken(row: any) {
  tokenUser.value = row
  tokenTtl.value = '720h'
  tokenResult.value = null
  copied.value = false
  tokenDialog.value = true
}

async function onMintToken() {
  tokenLoading.value = true
  try {
    const result = await UsersAPI.mintToken(tokenUser.value.id, tokenTtl.value)
    tokenResult.value = result
    tokenDialog.value = false
    tokenResultDialog.value = true
  } catch (e: any) {
    message.error(e?.message || '签发失败')
  } finally {
    tokenLoading.value = false
  }
}

async function copyToken() {
  if (!tokenResult.value?.token) return
  try {
    await navigator.clipboard.writeText(tokenResult.value.token)
    copied.value = true
    message.success('Token 已复制到剪贴板')
    setTimeout(() => { copied.value = false }, 2000)
  } catch {
    message.error('复制失败，请手动选中复制')
  }
}

onMounted(load)
</script>

<style scoped>
.token-block {
  display: flex;
  gap: 8px;
  background: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 12px;
}
.token-text {
  flex: 1;
  word-break: break-all;
  font-size: 12px;
  font-family: 'Courier New', monospace;
  color: #303133;
  line-height: 1.6;
  margin: 0;
  user-select: all;
  white-space: pre-wrap;
}
.copy-btn { flex-shrink: 0; align-self: flex-start; }
.token-tip {
  margin-top: 12px;
  font-size: 13px;
  color: #909399;
  line-height: 1.5;
}
.token-tip code {
  background: #f5f7fa;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 12px;
  color: #409eff;
}
</style>
