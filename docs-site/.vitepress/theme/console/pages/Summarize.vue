<!-- .vitepress/theme/console/pages/Summarize.vue -->
<template>
  <div>
    <h2>会话归纳</h2>
    <!-- Task list -->
    <div style="margin-bottom: 16px">
      <a-button @click="loadTasks">刷新任务</a-button>
    </div>
    <a-table
      v-if="tasks.length"
      :dataSource="tasks"
      :columns="taskCols"
      rowKey="id"
      :pagination="{ pageSize: 10 }"
      size="small"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <a-tag :color="taskStatusColor(record.status)">{{ record.status }}</a-tag>
        </template>
        <template v-if="column.key === 'actions'">
          <a-button size="small" @click="viewTask(record)">查看</a-button>
        </template>
      </template>
    </a-table>

    <!-- Trigger form -->
    <a-card title="新建归纳任务" style="margin-top: 16px">
      <a-form layout="vertical">
        <a-form-item label="会话 ID（逗号分隔）">
          <a-input v-model:value="form.sourceText" placeholder="session-1-id, session-2-id" />
        </a-form-item>
        <a-form-item label="深度">
          <a-select v-model:value="form.depth" style="width: 200px">
            <a-select-option value="shallow">浅层</a-select-option>
            <a-select-option value="deep">深层</a-select-option>
            <a-select-option value="expert">专家</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="目标 Wing">
          <a-input v-model:value="form.target_wing" placeholder="wing_name（可选）" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" :loading="loading" @click="onSubmit">开始归纳</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- Result display -->
    <div v-if="currentResult">
      <h3>归纳结果</h3>
      <a-tabs v-model:activeKey="activeTab">
        <a-tab-pane key="facts" tab="事实">
          <pre class="result-pre">{{ JSON.stringify(currentResult.hall_facts || currentResult.facts || currentResult, null, 2) }}</pre>
        </a-tab-pane>
        <a-tab-pane key="events" tab="事件">
          <pre class="result-pre">{{ JSON.stringify(currentResult.hall_events || currentResult.events || [], null, 2) }}</pre>
        </a-tab-pane>
        <a-tab-pane key="discoveries" tab="发现">
          <pre class="result-pre">{{ JSON.stringify(currentResult.hall_discoveries || currentResult.discoveries || [], null, 2) }}</pre>
        </a-tab-pane>
        <a-tab-pane key="preferences" tab="偏好">
          <pre class="result-pre">{{ JSON.stringify(currentResult.hall_preferences || currentResult.preferences || [], null, 2) }}</pre>
        </a-tab-pane>
        <a-tab-pane key="advice" tab="建议">
          <pre class="result-pre">{{ JSON.stringify(currentResult.hall_advice || currentResult.advice || [], null, 2) }}</pre>
        </a-tab-pane>
      </a-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { SummarizeDistillAPI } from '../store/api'

const tasks = ref<any[]>([])
const form = ref({ sourceText: '', depth: 'deep', target_wing: '' })
const loading = ref(false)
const currentResult = ref<any>(null)
const activeTab = ref('facts')

const taskCols = [
  { title: 'ID', dataIndex: 'id', key: 'id', ellipsis: true },
  { title: '状态', key: 'status', width: 100 },
  { title: '会话数', dataIndex: 'source_ids', key: 'source_ids', width: 80 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '操作', key: 'actions', width: 80 },
]

function taskStatusColor(s: string) {
  if (s === 'done' || s === 'completed') return 'green'
  if (s === 'failed') return 'red'
  if (s === 'running' || s === 'pending') return 'orange'
  return 'default'
}

async function loadTasks() {
  try { const r = await SummarizeDistillAPI.listSummarize(); tasks.value = r?.list || r || [] }
  catch {}
}

async function viewTask(record: any) {
  try {
    const detail = await SummarizeDistillAPI.getSummarize(record.id)
    currentResult.value = detail.result || detail
  } catch (e: any) { message.error(e.message) }
}

async function onSubmit() {
  if (!form.value.sourceText.trim()) {
    message.warning('请输入会话 ID')
    return
  }
  loading.value = true
  try {
    const data = await SummarizeDistillAPI.createSummarize({
      source_ids: form.value.sourceText.split(',').map((s: string) => s.trim()).filter(Boolean),
      depth: form.value.depth,
      target_wing: form.value.target_wing,
    })
    message.success('归纳任务已创建')
    await loadTasks()
    if (data.result) currentResult.value = data.result
  } catch (e: any) {
    message.error(e?.message || '归纳失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadTasks)
</script>

<style scoped>
.result-pre { white-space: pre-wrap; word-break: break-word; background: #f5f5f5; padding: 12px; border-radius: 4px; max-height: 500px; overflow: auto; }
</style>
