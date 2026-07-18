<!-- .vitepress/theme/console/pages/Projects.vue -->
<template>
  <div>
    <div style="margin-bottom: 12px; display: flex; gap: 8px; align-items: center; flex-wrap: wrap;">
      <a-select v-model:value="filterTeamId" placeholder="筛选团队" style="width:180px" allowClear @change="onFilterChange">
        <a-select-option v-for="t in teams" :key="t.id" :value="t.id">{{ t.name }}</a-select-option>
      </a-select>
      <a-input v-model:value="filterQ" placeholder="搜索项目名称" style="width: 200px" allowClear @change="onFilterChange" />
      <a-button @click="onFilterChange">刷新</a-button>
    </div>

    <a-table
      :dataSource="items"
      :columns="columns"
      :loading="loading"
      rowKey="id"
      :pagination="{ pageSize: 20, showSizeChanger: true, showTotal: (t: number) => '共 ' + t + ' 条' }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <a-tag :color="statusColor(record.status)">{{ record.status }}</a-tag>
        </template>
        <template v-if="column.key === 'indexed'">
          <a-tag :color="record.indexed ? 'green' : 'default'">{{ record.indexed ? '已索引' : '未索引' }}</a-tag>
        </template>
        <template v-if="column.key === 'actions'">
          <a-space>
            <a-button size="small" @click="openDetail(record)">详情</a-button>
            <a-button size="small" @click="onIndexStatus(record)" :loading="record._indexLoading">索引状态</a-button>
            <a-button size="small" @click="onReindex(record)" :loading="record._reindexLoading">触发索引</a-button>
          </a-space>
        </template>
      </template>
    </a-table>

    <!-- Detail drawer -->
    <a-drawer v-model:open="detailOpen" :title="detail?.name || '项目详情'" width="560">
      <template v-if="detail">
        <a-descriptions :column="1" bordered size="small">
          <a-descriptions-item label="ID">{{ detail.id }}</a-descriptions-item>
          <a-descriptions-item label="团队">{{ detail.team_id }}</a-descriptions-item>
          <a-descriptions-item label="Slug">{{ detail.slug }}</a-descriptions-item>
          <a-descriptions-item label="路径">{{ detail.path }}</a-descriptions-item>
          <a-descriptions-item label="Git URL">{{ detail.git_url }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="statusColor(detail.status)">{{ detail.status }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="Git Commit">{{ detail.git_commit_sha }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ detail.created_at }}</a-descriptions-item>
        </a-descriptions>
        <div style="margin-top: 16px">
          <a-space direction="vertical">
            <a-button @click="onReindexDetail" :loading="reindexLoading">触发重新索引</a-button>
            <a-button @click="onIndexStatusDetail" :loading="indexLoading">查看索引状态</a-button>
          </a-space>
        </div>
        <a-card v-if="idxResult" size="small" style="margin-top: 12px">
          <p><strong>已索引:</strong> {{ idxResult.indexed ? '是' : '否' }}</p>
          <p><strong>索引目录:</strong> {{ idxResult.index_dir }}</p>
          <p v-if="idxResult.last_update"><strong>最后更新:</strong> {{ idxResult.last_update }}</p>
        </a-card>
      </template>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { TeamsAPI, ProjectsAPI } from '../store/api'

const items = ref<any[]>([])
const loading = ref(false)
const teams = ref<any[]>([])
const filterTeamId = ref('')
const filterQ = ref('')
const detailOpen = ref(false)
const detail = ref<any>(null)
const idxResult = ref<any>(null)
const reindexLoading = ref(false)
const indexLoading = ref(false)

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 220, ellipsis: true },
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '团队', dataIndex: 'team_id', key: 'team_id', width: 160, ellipsis: true },
  { title: '状态', key: 'status', width: 100 },
  { title: '已索引', key: 'indexed', width: 80 },
  { title: '操作', key: 'actions', width: 240 },
]

function statusColor(s: string) {
  if (s === 'ready') return 'green'
  if (s === 'error') return 'red'
  if (s === 'indexing' || s === 'cloning') return 'orange'
  return 'default'
}

async function loadTeams() {
  try { const r = await TeamsAPI.list(); teams.value = r?.list || r || [] } catch {}
}

async function load() {
  loading.value = true
  try {
    if (filterTeamId.value) {
      const resp = await TeamsAPI.listProjects(filterTeamId.value)
      items.value = resp.list || resp || []
    } else {
      // Load from all teams
      const all: any[] = []
      for (const t of teams.value) {
        try {
          const resp = await TeamsAPI.listProjects(t.id)
          const list = resp.list || resp || []
          list.forEach((p: any) => {
            p._teamName = t.name
            all.push(p)
          })
        } catch {}
      }
      items.value = all.filter((p: any) => !filterQ.value || p.name.includes(filterQ.value))
    }
  } catch (e: any) { message.error(e.message) }
  finally { loading.value = false }
}

function onFilterChange() { load() }

function openDetail(record: any) {
  detail.value = record
  idxResult.value = null
  detailOpen.value = true
}

async function onIndexStatus(record: any) {
  record._indexLoading = true
  try {
    const result = await ProjectsAPI.indexStatus(record.id)
    record.indexed = result.indexed
    record._idxResult = result
    message.success('已索引: ' + (result.indexed ? '是' : '否'))
  } catch (e: any) { message.error(e.message) }
  finally { record._indexLoading = false }
}

async function onReindex(record: any) {
  record._reindexLoading = true
  try {
    await ProjectsAPI.reindex(record.id)
    record.status = 'indexing'
    message.success('索引已触发')
    await load()
  } catch (e: any) { message.error(e.message) }
  finally { record._reindexLoading = false }
}

async function onReindexDetail() {
  if (!detail.value) return
  reindexLoading.value = true
  try {
    await ProjectsAPI.reindex(detail.value.id)
    detail.value.status = 'indexing'
    idxResult.value = null
    message.success('索引已触发')
  } catch (e: any) { message.error(e.message) }
  finally { reindexLoading.value = false }
}

async function onIndexStatusDetail() {
  if (!detail.value) return
  indexLoading.value = true
  idxResult.value = null
  try {
    idxResult.value = await ProjectsAPI.indexStatus(detail.value.id)
  } catch (e: any) { message.error(e.message) }
  finally { indexLoading.value = false }
}

onMounted(async () => {
  await loadTeams()
  await load()
})
</script>
