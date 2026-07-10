<!-- .vitepress/theme/console/pages/Distill.vue -->
<template>
  <div>
    <h2>会话蒸馏</h2>
    <el-form :inline="true" :model="form">
      <el-form-item label="会话"><el-input v-model="form.sourceText" placeholder="逗号分隔 ID" /></el-form-item>
      <el-form-item label="最小价值">
        <el-input-number v-model="form.min_value_score" :min="0" :max="1" :step="0.05" />
      </el-form-item>
      <el-form-item label="目标 Wing"><el-input v-model="form.target_wing" /></el-form-item>
      <el-button type="primary" @click="onSubmit" :loading="loading">开始蒸馏</el-button>
    </el-form>

    <div v-if="taskId">
      <p>Task ID：<code>{{ taskId }}</code></p>
      <el-tabs>
        <el-tab-pane label="知识片段">
          <el-card v-for="(f, i) in (result?.knowledge_fragments || [])" :key="i">
            <strong>{{ f.type }}</strong> · score: {{ f.value_score }} · {{ f.source_ref }}
            <p>{{ f.content }}</p>
          </el-card>
        </el-tab-pane>
        <el-tab-pane label="决策点">
          <el-card v-for="(d, i) in (result?.decisions || [])" :key="i">
            <h4>{{ d.title }}</h4>
            <p>{{ d.decision }}</p>
          </el-card>
        </el-tab-pane>
        <el-tab-pane label="技术债务">
          <el-card v-for="(t, i) in (result?.tech_debt || [])" :key="i">
            <el-tag :type="severityColor(t.severity)">{{ t.severity }}</el-tag>
            {{ t.description }}
          </el-card>
        </el-tab-pane>
      </el-tabs>
      <el-button type="success" @click="onCommit">写入 MemPalace</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { DistillAPI } from '../store/api'

const form = ref({ sourceText: '', min_value_score: 0.7, target_wing: '' })
const loading = ref(false)
const taskId = ref('')
const result = ref<any>(null)

async function onSubmit() {
  loading.value = true
  try {
    const data = await DistillAPI.trigger({
      source_ids: form.value.sourceText.split(',').map((s) => s.trim()).filter(Boolean),
      rules: { min_value_score: form.value.min_value_score, dimensions: ['fact','decision','discovery'] },
    })
    taskId.value = data.task_id
    result.value = data.result
  } finally {
    loading.value = false
  }
}

async function onCommit() {
  await DistillAPI.commit(taskId.value, form.value.target_wing)
  ElMessage.success('已写入 MemPalace')
}

function severityColor(s: string) {
  return s === 'high' ? 'danger' : s === 'medium' ? 'warning' : 'info'
}
</script>
