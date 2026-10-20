<template>
  <div>
    <div class="orders-header">
      <h2 class="section-title">订单列表</h2>
      <button class="btn btn-primary" @click="loadOrders">刷新</button>
    </div>
    <div v-if="loading" class="loading">加载中...</div>
    <div v-else-if="orders.length === 0" class="loading">暂无订单数据</div>
    <div v-else class="table-wrap">
      <table>
        <thead>
          <tr>
            <th>运单号</th><th>订单ID</th><th>状态</th><th>最新事件</th>
            <th>发货日期</th><th>签收日期</th><th>签收</th><th>最后同步</th><th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="order in orders" :key="order.mdNo">
            <td>{{ order.mdNo }}</td>
            <td>{{ order.fid }}</td>
            <td><span :class="['badge', order.trackStatus === 'Delivered' ? 'badge-success' : 'badge-info']">{{ order.trackStatus || '-' }}</span></td>
            <td class="event-cell">{{ order.lastEvent || '-' }}</td>
            <td>{{ order.shipDate || '-' }}</td>
            <td>{{ order.deliveredDate || '-' }}</td>
            <td><span :class="['badge', order.isDelivered ? 'badge-success' : 'badge-warning']">{{ order.isDelivered ? '已签收' : '未签收' }}</span></td>
            <td>{{ order.lastSyncTime || '-' }}</td>
            <td><button class="btn btn-primary btn-sm" @click="showDetails(order.mdNo)">详情</button></td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="selectedOrder" class="details-panel">
      <div class="details-header">
        <h3 class="section-title sub">运单 {{ selectedOrder }} 轨迹时间线</h3>
        <span v-if="!detailsLoading && timeline.length" class="muted count">共 {{ timeline.length }} 条事件</span>
      </div>
      <div v-if="detailsLoading" class="loading">加载中...</div>
      <div v-else-if="timeline.length === 0" class="muted">暂无轨迹记录</div>
      <div v-else class="timeline">
        <div
          v-for="(d, i) in timeline"
          :key="d.id != null ? d.id : 'e' + i"
          :class="['tl-item', stageItemClass(d.stage, d.trackStatus)]"
        >
          <span class="tl-dot"></span>
          <div class="tl-time">{{ d.eventTime || '时间未知' }}</div>
          <div class="tl-line">
            <span
              v-if="d.stage || d.trackStatus"
              :class="['badge', stageBadgeClass(d.stage, d.trackStatus)]"
            >{{ d.stage || d.trackStatus }}</span>
            <span class="tl-desc">{{ d.eventDesc || '-' }}</span>
          </div>
          <div v-if="d.location" class="tl-loc">📍 {{ d.location }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { GetOrders, GetOrderDetails } from '../composables/useApi.js'

const orders = ref([])
const loading = ref(true)
const selectedOrder = ref('')
const details = ref([])
const detailsLoading = ref(false)

async function loadOrders() {
  loading.value = true
  try { orders.value = await GetOrders(100) || [] } catch (e) { console.error(e) }
  finally { loading.value = false }
}

async function showDetails(mdNo) {
  selectedOrder.value = mdNo
  detailsLoading.value = true
  try { details.value = await GetOrderDetails(mdNo) || [] } catch (e) { console.error(e); details.value = [] }
  finally { detailsLoading.value = false }
}

// 时间线按时间升序展示（最早在上，形成寄件→签收的旅程）；后端按时间 DESC 返回
const timeline = computed(() => [...details.value].slice().reverse())

// 阶段 → 徽章颜色
function stageBadgeClass(stage, trackStatus) {
  const s = stage || trackStatus
  if (s === 'Delivered') return 'badge-success'
  if (s === 'PickedUp' || s === 'InfoReceived') return 'badge-info'
  if (s === 'Exception' || s === 'Undeliverable' || s === 'DeliveryFailure' || s === 'Expired') return 'badge-danger'
  return 'badge-muted'
}
// 阶段 → 时间线圆点高亮
function stageItemClass(stage, trackStatus) {
  const c = stageBadgeClass(stage, trackStatus)
  return {
    'is-delivered': c === 'badge-success',
    'is-picked': c === 'badge-info',
    'is-danger': c === 'badge-danger',
  }
}

onMounted(loadOrders)
</script>

<style scoped>
.orders-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.section-title { font-size: 18px; margin: 0; }
.section-title.sub { font-size: 16px; }
.btn-sm { padding: 4px 12px; font-size: 12px; }
.event-cell { max-width: 240px; color: #2d3436; }
.muted { color: #636e72; font-size: 13px; }
.count { margin-left: 8px; }

.table-wrap { overflow-x: auto; }

.details-panel {
  margin-top: 20px;
  border-top: 1px solid #dfe6e9;
  padding-top: 16px;
}
.details-header {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 14px;
}

/* 垂直时间线 */
.timeline { position: relative; }
.tl-item {
  position: relative;
  padding: 0 0 18px 26px;
  border-left: 2px solid #dfe6e9;
  margin-left: 4px;
}
.tl-item:last-child { border-left-color: transparent; padding-bottom: 0; }
.tl-dot {
  position: absolute;
  left: -7px;
  top: 2px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #b2bec3;
  border: 2px solid #fff;
  box-shadow: 0 0 0 1px #dfe6e9;
}
.tl-item.is-delivered .tl-dot { background: #00b894; box-shadow: 0 0 0 1px #00b894; }
.tl-item.is-picked .tl-dot { background: #0984e3; box-shadow: 0 0 0 1px #0984e3; }
.tl-item.is-danger .tl-dot { background: #d63031; box-shadow: 0 0 0 1px #d63031; }

.tl-time { font-size: 12px; color: #636e72; margin-bottom: 3px; }
.tl-line { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.tl-desc { font-size: 13px; color: #2d3436; }
.tl-loc { font-size: 12px; color: #636e72; margin-top: 3px; }

/* 徽章配色补充（复用全局 .badge 基础样式） */
.badge-muted { background: #e9ecef; color: #636e72; }
.badge-danger { background: #fadbd8; color: #c0392b; }
</style>
