<!-- .vitepress/theme/console/pages/Summarize.vue -->
<template>
  <div>
    <h2>会话归纳</h2>
    <el-form :inline="true" :model="form">
      <el-form-item label="会话"><el-input v-model="form.sourceText" placeholder="逗号分隔 ID" /></el-form-item>
      <el-form-item label="深度">
        <el-select v-model="form.depth">
          <el-option label="浅层" value="shallow" />
          <el-option label="深层" value="deep" />
          <el-option label="专家" value="expert" />
        </el-select>
      </el-form-item>
      <el-form-item label="目标 Wing"><el-input v-model="form.target_wing" /></el-form-item>
      <el-button type="primary" @click="onSubmit" :loading="loading">开始归纳</el-button>
    </el-form>

    <div v-if="result">
      <h3>5 Hall 结果</h3>
      <el-tabs>
        <el-tab-pane label="facts"><pre>{{ JSON.stringify(result.hall_facts, null, 2) }}</pre></el-tab-pane>
        <el-tab-pane label="events"><pre>{{ JSON.stringify(result.hall_events, null, 2) }}</pre></el-tab-pane>
        <el-tab-pane label="discoveries"><pre>{{ JSON.stringify(result.hall_discoveries, null, 2) }}</pre></el-tab-pane>
        <el-tab-pane label="preferences"><pre>{{ JSON.stringify(result.hall_preferences, null, 2) }}</pre></el-tab-pane>
        <el-tab-pane label="advice"><pre>{{ JSON.stringify(result.hall_advice, null, 2) }}</pre></el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { SummarizeAPI } from '../store/api'

const form = ref({ sourceText: '', depth: 'deep', target_wing: '' })
const loading = ref(false)
const result = ref<any>(null)

async function onSubmit() {
  loading.value = true
  try {
    const data = await SummarizeAPI.trigger({
      source_ids: form.value.sourceText.split(',').map((s) => s.trim()).filter(Boolean),
      depth: form.value.depth,
      target_wing: form.value.target_wing,
    })
    result.value = data.result
  } finally {
    loading.value = false
  }
}
</script>
