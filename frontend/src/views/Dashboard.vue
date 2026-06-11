<template>
  <div>
    <div v-if="error" class="error">{{ error }}</div>
    <div v-if="loading" class="loading">加载中...</div>
    <template v-else>
      <div class="stats-grid">
        <div class="card">
          <div class="value">{{ stats.totalOrders }}</div>
          <div class="label">已注册运单</div>
        </div>
        <div class="card">
          <div class="value" style="color:#0984e3">{{ stats.inTransit }}</div>
          <div class="label">在途</div>
        </div>
        <div class="card">
          <div class="value" style="color:#00b894">{{ stats.delivered }}</div>
          <div class="label">已签收</div>
        </div>
        <div class="card">
          <div class="value" :style="{color: stats.isSyncRunning ? '#e17055' : '#636e72'}">{{ stats.isSyncRunning ? '同步中' : '空闲' }}</div>
          <div class="label">同步状态</div>
        </div>
      </div>
      <div style="display:flex; gap:16px; align-items:center; margin-bottom:16px;">
        <button class="btn btn-primary" @click="triggerSync" :disabled="syncing">
          {{ syncing ? '同步中...' : '立即同步' }}
        </button>
        <span style="font-size:13px; color:#636e72;">
          上次同步: {{ stats.lastSyncTime || '从未' }}
          <span v-if="stats.lastSyncError" style="color:#d63031"> ({{ stats.lastSyncError }})</span>
        </span>
      </div>
      <div style="font-size:13px; color:#636e72;">
        调度器: {{ stats.schedulerEnabled ? '已启用' : '已禁用' }} | Cron: {{ stats.cronSpec }}
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { GetDashboardStats, TriggerSync } from '../composables/useApi.js'

const stats = ref({ totalOrders: 0, delivered: 0, inTransit: 0, isSyncRunning: false, lastSyncTime: '', lastSyncError: '', schedulerEnabled: false, cronSpec: '' })
const loading = ref(true)
const error = ref('')
const syncing = ref(false)
let timer = null

async function loadStats() {
  try {
    const result = await GetDashboardStats()
    stats.value = {
      totalOrders: result.totalOrders || 0,
      delivered: result.delivered || 0,
      inTransit: result.inTransit || 0,
      isSyncRunning: result.isSyncRunning || false,
      lastSyncTime: result.lastSyncTime || '',
      lastSyncError: result.lastSyncError || '',
      schedulerEnabled: result.schedulerEnabled || false,
      cronSpec: result.cronSpec || '',
    }
    error.value = ''
  } catch (e) {
    error.value = e.message || String(e)
  } finally {
    loading.value = false
  }
}

async function triggerSync() {
  syncing.value = true
  try {
    await TriggerSync()
    setTimeout(loadStats, 2000)
  } catch (e) {
    error.value = e.message || String(e)
  } finally {
    syncing.value = false
  }
}

onMounted(() => { loadStats(); timer = setInterval(loadStats, 30000) })
onUnmounted(() => { if (timer) clearInterval(timer) })
</script>
