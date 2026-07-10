<!-- .vitepress/theme/console/pages/Users.vue -->
<template>
  <div>
    <h2>用户管理</h2>
    <el-button type="primary" @click="openCreate">新增用户</el-button>
    <el-table :data="users" v-loading="loading" style="margin-top: 16px;">
      <el-table-column prop="id" label="ID" />
      <el-table-column prop="display_name" label="显示名" />
      <el-table-column label="Token 状态" width="120">
        <template #default="{ row }">
          <el-tag :type="row.disabled ? 'danger' : 'success'">
            {{ row.disabled ? '停用' : '有效' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作">
        <template #default="{ row }">
          <el-button size="small" @click="onEdit(row)">编辑</el-button>
          <el-button size="small" type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialog" :title="editing ? '编辑用户' : '新增用户'">
      <el-form :model="form" label-width="100px">
        <el-form-item label="用户 ID"><el-input v-model="form.id" :disabled="!!editing" /></el-form-item>
        <el-form-item label="显示名"><el-input v-model="form.display_name" /></el-form-item>
        <el-form-item label="项目路径">
          <el-input v-model="pathsText" placeholder="逗号分隔" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="onSubmit">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { UsersAPI } from '../store/api'

const users = ref<any[]>([])
const loading = ref(false)
const dialog = ref(false)
const editing = ref<any>(null)
const form = ref<any>({ id: '', display_name: '', project_paths: [] })
const pathsText = computed({
  get: () => (form.value.project_paths || []).join(','),
  set: (v) => { form.value.project_paths = v.split(',').map((x: string) => x.trim()).filter(Boolean) },
})

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
  form.value = { id: '', display_name: '', project_paths: [] }
  dialog.value = true
}
function onEdit(row: any) {
  editing.value = row
  form.value = { ...row }
  dialog.value = true
}
async function onSubmit() {
  if (editing.value) {
    await UsersAPI.update(editing.value.id, form.value)
  } else {
    await UsersAPI.create(form.value)
  }
  ElMessage.success('保存成功')
  dialog.value = false
  await load()
}
async function onDelete(row: any) {
  await ElMessageBox.confirm(`删除 ${row.id}?`, '确认')
  await UsersAPI.remove(row.id)
  ElMessage.success('已删除')
  await load()
}
onMounted(load)
</script>
