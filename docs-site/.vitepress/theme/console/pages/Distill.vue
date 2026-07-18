<!-- .vitepress/theme/console/pages/Distill.vue -->
<template>
  <div>
    <h2>会话蒸馏</h2>
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
    <a-card title="新建蒸馏任务" style="margin-top: 16px">
      <a-form layout="vertical">
        <a-form-item label="会话 ID（逗号分隔）">
          <a-input v-model:value="form.sourceText" placeholder="session-1-id, session-2-id" />
        </a-form-item>
        <a-form-item label="最小价值分数">
          <a-input-number v-model:value="form.min_value_score" :min="0" :max="1" :step="0.05" />
        </a-form-item>
        <a-form-item label="目标 Wing">
          <a-input v-model:value="form.target_wing" placeholder="wing_name（可选）" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" :loading="loading" @click="onSubmit">开始蒸馏</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- Result display -->
    <div v-if="currentResult">
      <h3>蒸馏结果</h3>
      <a-tabs v-model:activeKey="activeTab">
        <a-tab-pane key="fragments" tab="知识片段">
          <div v-for="(f, i) in (currentResult.knowledge_fragments || currentResult.fragments || [])" :key="i" class="frag-card">
            <div class="frag-header">
              <a-tag>{{ f.type }}</a-tag>
              <span>score: {{ f.value_score }}</span>
              <span class="ref">{{ f.source_ref }}</span>
            </div>
            <p class="frag-body">{{ f.content }}</p>
          </div>
          <a-empty v-if="!(currentResult.knowledge_fragments?.length) && !(currentResult.fragments?.length)" description="暂无知识片段" />
        </a-tab-pane>

        <a-tab-pane key="decisions" tab="决策点">
          <div v-for="(d, i) in (currentResult.decisions || [])" :key="i" class="dec-card">
            <h4>{{ d.title }}</h4>
            <p>{{ d.decision }}</p>
          </div>
          <a-empty v-if="!(currentResult.decisions?.length)" description="暂无决策点" />
        </a-tab-pane>

        <a-tab-pane key="techdebt" tab="技术债务">
          <div v-for="(t, i) in (currentResult.tech_debt || [])" :key="i" class="td-row">
            <a-tag :color="severityColor(t.severity)">{{ t.severity }}</a-tag>
            <span>{{ t.description }}</span>
          </div>
          <a-empty v-if="!(currentResult.tech_debt?.length)" description="无技术债务" />
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
const form = ref({ sourceText: '', min_value_score: 0.7, target_wing: '' })
const loading = ref(false)
const currentResult = ref<any>(null)
const activeTab = ref('fragments')

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
  try { tasks.value = await SummarizeDistillAPI.listDistill() }
  catch {}
}

async function viewTask(record: any) {
  try {
    const detail = await SummarizeDistillAPI.getDistill(record.id)
    currentResult.value = detail.result || detail
  } catch (e: any) { message.error(e.message) }
}

async function onSubmit() {
  loading.value = true
  try {
    const data = await SummarizeDistillAPI.createDistill({
      source_ids: form.value.sourceText.split(',').map((s: string) => s.trim()).filter(Boolean),
      rules: { min_value_score: form.value.min_value_score, dimensions: ['fact', 'decision', 'discovery'] },
    })
    message.success('蒸馏任务已创建')
    await loadTasks()
    if (data.result) currentResult.value = data.result
  } catch (e: any) {
    message.error(e?.message || '蒸馏失败')
  } finally {
    loading.value = false
  }
}

function severityColor(s: string): string {
  if (s === 'high') return 'red'
  if (s === 'medium') return 'orange'
  return 'blue'
}

onMounted(loadTasks)
</script>

<style scoped>
.frag-card {
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 12px;
}
.frag-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 13px;
  color: #595959;
}
.frag-body {
  margin: 0;
  white-space: pre-wrap;
  color: #262626;
}
.ref {
  color: #8c8c8c;
  font-size: 12px;
}
.dec-card {
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 12px;
}
.dec-card h4 { margin: 0 0 8px; }
.dec-card p { margin: 0; }
.td-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 0;
  border-bottom: 1px solid #f5f5f5;
}
</style>
