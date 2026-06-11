<template>
  <div>
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px;">
      <h2 style="font-size:18px;">同步日志</h2>
      <button class="btn btn-primary" @click="loadLogs">刷新</button>
    </div>
    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="logs.length === 0" class="loading">暂无日志记录</div>
    <table v-else>
      <thead><tr><th>时间</th><th>运单号</th><th>状态</th><th>事件描述</th><th>记录时间</th></tr></thead>
      <tbody>
        <tr v-for="log in logs" :key="log.id">
          <td>{{ log.eventTime || '-' }}</td>
          <td>{{ log.mdNo }}</td>
          <td><span :class="['badge', log.status === 'Delivered' ? 'badge-success' : 'badge-info']">{{ log.status || '-' }}</span></td>
          <td>{{ log.eventDesc || '-' }}</td>
          <td>{{ log.createTime }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { GetRecentLogs } from '../composables/useApi.js'

const logs = ref([])
const loading = ref(true)

async function loadLogs() {
  loading.value = true
  try { logs.value = await GetRecentLogs(100) || [] } catch (e) { console.error(e) }
  finally { loading.value = false }
}

onMounted(loadLogs)
</script>
