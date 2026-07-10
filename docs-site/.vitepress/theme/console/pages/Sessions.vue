<!-- .vitepress/theme/console/pages/Sessions.vue -->
<template>
  <div>
    <h2>会话记录</h2>
    <el-table :data="items" style="margin-top: 16px;">
      <el-table-column prop="id" label="ID" />
      <el-table-column prop="user_id" label="用户" />
      <el-table-column prop="project_path" label="项目" />
      <el-table-column prop="turn_count" label="轮次" width="80" />
      <el-table-column prop="started_at" label="开始时间" />
      <el-table-column label="操作">
        <template #default="{ row }">
          <el-button size="small" @click="openDetail(row)">查看</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-drawer v-model="detailOpen" :title="detail?.session.id" size="60%">
      <div v-if="detail">
        <p>用户：{{ detail.session.user_id }}</p>
        <p>项目：{{ detail.session.project_path }}</p>
        <h3>对话轮次（{{ detail.turns.length }}）</h3>
        <el-card v-for="t in detail.turns" :key="t.turn_no" style="margin-bottom: 8px;">
          <strong>{{ t.role }} #{{ t.turn_no }}</strong>
          <pre style="white-space: pre-wrap;">{{ t.content }}</pre>
        </el-card>
      </div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { SessionsAPI } from '../store/api'

const items = ref<any[]>([])
const detailOpen = ref(false)
const detail = ref<any>(null)

async function load() { items.value = (await SessionsAPI.list()).list || [] }
async function openDetail(row: any) {
  detail.value = await SessionsAPI.detail(row.id)
  detailOpen.value = true
}
onMounted(load)
</script>
