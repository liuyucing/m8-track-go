<template>
  <div>
    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px;">
      <h2 style="font-size:18px;">订单列表</h2>
      <button class="btn btn-primary" @click="loadOrders">刷新</button>
    </div>
    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="orders.length === 0" class="loading">暂无订单数据</div>
    <table v-else>
      <thead>
        <tr><th>运单号</th><th>订单ID</th><th>状态</th><th>最新事件</th><th>签收</th><th>最后同步</th><th>操作</th></tr>
      </thead>
      <tbody>
        <tr v-for="order in orders" :key="order.mdNo">
          <td>{{ order.mdNo }}</td>
          <td>{{ order.fid }}</td>
          <td><span :class="['badge', order.trackStatus === 'Delivered' ? 'badge-success' : 'badge-info']">{{ order.trackStatus || '-' }}</span></td>
          <td>{{ order.lastEvent || '-' }}</td>
          <td><span :class="['badge', order.isDelivered ? 'badge-success' : 'badge-warning']">{{ order.isDelivered ? '已签收' : '未签收' }}</span></td>
          <td>{{ order.lastSyncTime || '-' }}</td>
          <td><button class="btn btn-primary" style="padding:4px 12px; font-size:12px;" @click="showDetails(order.mdNo)">详情</button></td>
        </tr>
      </tbody>
    </table>
    <div v-if="selectedOrder" style="margin-top:20px; border-top:1px solid #dfe6e9; padding-top:16px;">
      <h3 style="font-size:16px; margin-bottom:12px;">运单 {{ selectedOrder }} 轨迹详情</h3>
      <div v-if="details.length === 0" style="color:#636e72; font-size:13px;">暂无轨迹记录</div>
      <table v-else>
        <thead><tr><th>时间</th><th>状态</th><th>事件</th><th>记录时间</th></tr></thead>
        <tbody>
          <tr v-for="d in details" :key="d.id">
            <td>{{ d.eventTime || '-' }}</td>
            <td>{{ d.trackStatus || '-' }}</td>
            <td>{{ d.eventDesc || '-' }}</td>
            <td>{{ d.createTime || '-' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { GetOrders, GetOrderDetails } from '../composables/useApi.js'

const orders = ref([])
const loading = ref(true)
const selectedOrder = ref('')
const details = ref([])

async function loadOrders() {
  loading.value = true
  try { orders.value = await GetOrders(100) || [] } catch (e) { console.error(e) }
  finally { loading.value = false }
}

async function showDetails(mdNo) {
  selectedOrder.value = mdNo
  try { details.value = await GetOrderDetails(mdNo) || [] } catch (e) { console.error(e) }
}

onMounted(loadOrders)
</script>
