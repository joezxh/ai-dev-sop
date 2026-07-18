<!-- .vitepress/theme/console/pages/Sessions.vue -->
<template>
  <div>
    <!-- Filters -->
    <div style="margin-bottom: 12px; display: flex; gap: 8px; align-items: center; flex-wrap: wrap;">
      <a-select v-model:value="filterTeamId" placeholder="筛选团队" style="width:160px" allowClear @change="onFilterChange">
        <a-select-option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</a-select-option>
      </a-select>
      <a-select v-model:value="filterProjectId" placeholder="筛选项目" style="width:200px" allowClear @change="onFilterChange">
        <a-select-option v-for="p in projects" :key="p.id" :value="p.id">{{ p.name }}</a-select-option>
      </a-select>
      <a-input v-model:value="filterQ" placeholder="搜索会话ID/用户" style="width:180px" allowClear @change="onFilterChange" />
      <a-button @click="onFilterChange">刷新</a-button>
      <a-button @click="loadStats" :loading="statsLoading">刷新统计</a-button>
    </div>

    <!-- Stats -->
    <a-row :gutter="12" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-statistic title="总会话数" :value="stats.total" />
      </a-col>
      <a-col :span="6">
        <a-statistic title="今日会话" :value="stats.today" />
      </a-col>
      <a-col :span="6">
        <a-statistic title="总工具调用" :value="stats.total_tool_calls" />
      </a-col>
      <a-col :span="6">
        <a-statistic title="总对话轮次" :value="stats.total_turns" />
      </a-col>
    </a-row>

    <!-- Session list -->
    <a-table
      :dataSource="items"
      :columns="columns"
      :loading="loading"
      rowKey="id"
      :pagination="{ pageSize: 20, showSizeChanger: true }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'actions'">
          <a-button size="small" @click="openDetail(record)">查看详情</a-button>
        </template>
      </template>
    </a-table>

    <!-- Detail Drawer -->
    <a-drawer v-model:open="detailOpen" :title="'会话详情 ' + (detail?.session?.id || '')" size="large">
      <template v-if="detail">
        <a-descriptions :column="1" bordered size="small">
          <a-descriptions-item label="会话ID">{{ detail.session?.id }}</a-descriptions-item>
          <a-descriptions-item label="用户">{{ detail.session?.user_id }}</a-descriptions-item>
          <a-descriptions-item label="团队">{{ detail.session?.team_id }}</a-descriptions-item>
          <a-descriptions-item label="项目">{{ detail.session?.project_id }}</a-descriptions-item>
          <a-descriptions-item label="模块">{{ detail.session?.module_id }}</a-descriptions-item>
          <a-descriptions-item label="开始时间">{{ detail.session?.started_at }}</a-descriptions-item>
          <a-descriptions-item label="结束时间">{{ detail.session?.ended_at }}</a-descriptions-item>
          <a-descriptions-item label="工具调用">{{ detail.session?.tool_count }}</a-descriptions-item>
          <a-descriptions-item label="对话轮次">{{ detail.session?.turn_count }}</a-descriptions-item>
        </a-descriptions>

        <h3 style="margin-top: 16px">对话轮次 ({{ detail.turns?.length || 0 }})</h3>
        <div v-for="t in (detail.turns || [])" :key="t.turn_no" class="turn-card">
          <div class="turn-header">
            <a-tag :color="t.role === 'user' ? 'blue' : (t.role === 'assistant' ? 'green' : 'orange')">
              {{ t.role }}
            </a-tag>
            #{{ t.turn_no }}
            <span style="margin-left:auto; color:#999; font-size:12px">{{ t.ts }}</span>
          </div>
          <pre class="turn-content">{{ t.content }}</pre>
        </div>
      </template>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { TeamsAPI, SessionsAPI } from '../store/api'

const items = ref<any[]>([])
const loading = ref(false)
const teams = ref<any[]>([])
const projects = ref<any[]>([])
const filterTeamId = ref('')
const filterProjectId = ref('')
const filterQ = ref('')

// Stats
const stats = ref({ total: 0, today: 0, total_tool_calls: 0, total_turns: 0 })
const statsLoading = ref(false)

// Detail
const detailOpen = ref(false)
const detail = ref<any>(null)

const columns = [
  { title: '会话ID', dataIndex: 'id', key: 'id', width: 200, ellipsis: true },
  { title: '用户', dataIndex: 'user_id', key: 'user_id', width: 120 },
  { title: '团队', dataIndex: 'team_id', key: 'team_id', width: 120, ellipsis: true },
  { title: '项目', dataIndex: 'project_id', key: 'project_id', width: 160, ellipsis: true },
  { title: '模块', dataIndex: 'module_id', key: 'module_id', width: 160, ellipsis: true },
  { title: '工具调用', dataIndex: 'tool_count', key: 'tool_count', width: 90 },
  { title: '轮次', dataIndex: 'turn_count', key: 'turn_count', width: 60 },
  { title: '开始时间', dataIndex: 'started_at', key: 'started_at', width: 170 },
  { title: '操作', key: 'actions', width: 100 },
]

async function loadTeams() {
  try { teams.value = await TeamsAPI.list() } catch {}
}

async function onTeamChange() {
  filterProjectId.value = ''
  projects.value = []
  if (!filterTeamId.value) return
  try {
    const resp = await TeamsAPI.listProjects(filterTeamId.value)
    projects.value = resp.list || resp || []
  } catch {}
}

async function load() {
  loading.value = true
  try {
    const params: Record<string, string> = {}
    if (filterTeamId.value) params.team_id = filterTeamId.value
    if (filterProjectId.value) params.project_id = filterProjectId.value
    if (filterQ.value) params.q = filterQ.value
    const resp = await SessionsAPI.list(params)
    items.value = resp.list || resp || []
  } catch (e: any) { message.error(e.message) }
  finally { loading.value = false }
}

async function loadStats() {
  statsLoading.value = true
  try {
    const resp = await SessionsAPI.stats()
    stats.value = resp
  } catch {}
  finally { statsLoading.value = false }
}

function onFilterChange() { load() }

async function openDetail(record: any) {
  try {
    detail.value = await SessionsAPI.get(record.id)
    detailOpen.value = true
  } catch (e: any) { message.error(e.message) }
}

onMounted(async () => {
  await loadTeams()
  await load()
  await loadStats()
})
</script>

<style scoped>
.turn-card {
  border: 1px solid #f0f0f0;
  border-radius: 6px;
  padding: 12px;
  margin-bottom: 12px;
}
.turn-header {
  font-weight: 600;
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.turn-content {
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 13px;
  color: #262626;
  margin: 0;
  max-height: 400px;
  overflow: auto;
}
</style>
