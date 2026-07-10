<!-- .vitepress/theme/console/pages/Projects.vue -->
<template>
  <div>
    <h2>项目管理</h2>
    <el-button type="primary" @click="openCreate">新增项目</el-button>
    <el-table :data="items" style="margin-top: 16px;">
      <el-table-column prop="id" label="ID" />
      <el-table-column prop="name" label="名称" />
      <el-table-column prop="path" label="路径" />
      <el-table-column prop="wing" label="Wing" />
      <el-table-column label="操作">
        <template #default="{ row }">
          <el-button size="small" @click="onReindex(row)">触发索引</el-button>
          <el-button size="small" type="danger" @click="onDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialog" title="新增项目">
      <el-form :model="form" label-width="100px">
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="路径"><el-input v-model="form.path" /></el-form-item>
        <el-form-item label="Wing"><el-input v-model="form.wing" /></el-form-item>
        <el-form-item label="mcp_bin"><el-input v-model="form.mcp_bin" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog = false">取消</el-button>
        <el-button type="primary" @click="onSubmit">提交</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ProjectsAPI } from '../store/api'

const items = ref<any[]>([])
const dialog = ref(false)
const form = ref({ name: '', path: '', wing: '', mcp_bin: '' })

async function load() { items.value = (await ProjectsAPI.list()).list || [] }
function openCreate() {
  form.value = { name: '', path: '', wing: '', mcp_bin: '' }
  dialog.value = true
}
async function onSubmit() {
  await ProjectsAPI.create(form.value)
  dialog.value = false
  ElMessage.success('创建成功')
  await load()
}
async function onDelete(row: any) {
  await ElMessageBox.confirm(`删除项目 ${row.name}?`, '确认')
  await ProjectsAPI.remove(row.id)
  await load()
}
async function onReindex(row: any) {
  ElMessage.info('触发索引...')
  await ProjectsAPI.reindex(row.id)
  ElMessage.success('完成')
}
onMounted(load)
</script>
