<template>
  <div>
    <!-- 未配置提示 -->
    <div v-if="!configured" class="error" style="margin-bottom:16px;">
      ⚠️ 系统尚未配置完成，请填写数据库和 17track API 信息后保存，然后重启应用。
    </div>

    <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:16px;">
      <h2 style="font-size:18px;">系统配置</h2>
      <div style="display:flex; gap:8px;">
        <button class="btn btn-primary" @click="loadConfig">刷新</button>
        <button class="btn btn-primary" :disabled="saving" @click="saveConfig">
          {{ saving ? '保存中...' : '保存配置' }}
        </button>
      </div>
    </div>

    <div v-if="msg" :class="['badge', msgOk ? 'badge-success' : 'badge-warning']" style="margin-bottom:16px; padding:8px 12px;">
      {{ msg }}
    </div>

    <div v-if="loading" class="loading">加载中...</div>
    <template v-else>
      <!-- 数据库配置 -->
      <fieldset style="border:1px solid #dfe6e9; border-radius:8px; padding:16px; margin-bottom:16px;">
        <legend style="color:#0984e3; font-weight:600; padding:0 8px;">数据库</legend>
        <div class="config-grid">
          <label class="config-key">地址</label>
          <input v-model="form.database.host" class="config-input" placeholder="例如: 192.168.1.100" />
          <label class="config-key">端口</label>
          <input v-model.number="form.database.port" type="number" class="config-input" />
          <label class="config-key">数据库名</label>
          <input v-model="form.database.name" class="config-input" />
          <label class="config-key">用户名</label>
          <input v-model="form.database.username" class="config-input" />
          <label class="config-key">密码</label>
          <input v-model="form.database.password" type="password" class="config-input" placeholder="输入新密码" />
        </div>
      </fieldset>

      <!-- 17track API -->
      <fieldset style="border:1px solid #dfe6e9; border-radius:8px; padding:16px; margin-bottom:16px;">
        <legend style="color:#0984e3; font-weight:600; padding:0 8px;">17track API</legend>
        <div class="config-grid">
          <label class="config-key">API 密钥</label>
          <input v-model="form.track17.api_key" class="config-input" placeholder="17track API 密钥" />
          <label class="config-key">接口地址</label>
          <input v-model="form.track17.base_url" class="config-input" />
          <label class="config-key">每批数量</label>
          <input v-model.number="form.track17.batch_size" type="number" class="config-input" />
        </div>
      </fieldset>

      <!-- 同步计划 -->
      <fieldset style="border:1px solid #dfe6e9; border-radius:8px; padding:16px; margin-bottom:16px;">
        <legend style="color:#0984e3; font-weight:600; padding:0 8px;">同步计划</legend>
        <div style="margin-bottom:12px; display:flex; align-items:center; gap:12px;">
          <label class="config-key" style="margin:0;">启用定时同步</label>
          <input v-model="form.scheduler.enabled" type="checkbox" style="width:18px; height:18px; cursor:pointer;" />
        </div>
        <div v-if="form.scheduler.enabled" style="margin-top:12px;">
          <label class="config-key" style="margin-bottom:8px; display:block;">同步频率</label>
          <select v-model="scheduleMode" class="config-input" style="max-width:400px; cursor:pointer;">
            <option value="every2h">每 2 小时</option>
            <option value="every3h">每 3 小时</option>
            <option value="every4h">每 4 小时</option>
            <option value="every6h">每 6 小时</option>
            <option value="every8h">每 8 小时</option>
            <option value="every12h">每 12 小时</option>
            <option value="daily">每天一次</option>
            <option value="custom">自定义时间</option>
          </select>

          <div v-if="scheduleMode === 'daily'" style="margin-top:12px;">
            <label class="config-key" style="margin-bottom:8px; display:block;">每天同步时间</label>
            <input v-model="dailyHour" type="time" class="config-input" style="max-width:200px;" />
          </div>

          <div v-if="scheduleMode === 'custom'" style="margin-top:12px;">
            <label class="config-key" style="margin-bottom:8px; display:block;">选择每天同步的时间点（可多选）</label>
            <div class="hour-grid">
              <label v-for="h in 24" :key="h-1" class="hour-chip"
                :class="{ active: selectedHours.includes(h-1) }">
                <input type="checkbox" :value="h-1"
                  :checked="selectedHours.includes(h-1)"
                  @change="toggleHour(h-1)" />
                {{ String(h-1).padStart(2, '0') }}:00
              </label>
            </div>
          </div>

          <div v-if="scheduleMode !== 'custom'" style="margin-top:12px; padding:8px 12px; background:#f0f7ff; border-radius:6px; font-size:13px; color:#636e72;">
            💡 将在每天 {{ scheduleDescription }} 自动同步物流轨迹
          </div>
        </div>
      </fieldset>

      <!-- 查询配置 -->
      <fieldset style="border:1px solid #dfe6e9; border-radius:8px; padding:16px; margin-bottom:16px;">
        <legend style="color:#0984e3; font-weight:600; padding:0 8px;">查询配置</legend>
        <div class="config-grid">
          <label class="config-key">只同步此日期之后的订单</label>
          <input v-model="form.query.order_date_filter" type="date" class="config-input" />
        </div>
      </fieldset>
    </template>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { GetConfig, SaveConfig, IsConfigured } from '../composables/useApi.js'

const loading = ref(true)
const saving = ref(false)
const configured = ref(true)
const msg = ref('')
const msgOk = ref(true)

const scheduleMode = ref('every6h')
const selectedHours = ref([])
const dailyHour = ref('03:00')

const form = reactive({
  database: { host: '', port: 3366, name: 'FumaCRM8', username: 'sa', password: '' },
  track17: { api_key: '', base_url: 'https://api.17track.net/track/v2.4', batch_size: 40 },
  scheduler: { cron: '0 0 3,9,15,21 * * *', enabled: true },
  query: { order_date_filter: '2026-05-01' },
})

// 预设模式对应的小时
const presetHours = {
  every2h:  [0,2,4,6,8,10,12,14,16,18,20,22],
  every3h:  [0,3,6,9,12,15,18,21],
  every4h:  [0,4,8,12,16,20],
  every6h:  [3,9,15,21],
  every8h:  [1,9,17],
  every12h: [3,15],
}

const scheduleDescription = computed(() => {
  const hours = presetHours[scheduleMode.value]
  if (hours) return hours.map(h => `${String(h).padStart(2,'0')}:00`).join('、')
  return ''
})

// 从 cron 解析出小时列表（6字段格式：秒 分 时 日 月 周）
function parseCronHours(cron) {
  const parts = cron.trim().split(/\s+/)
  // parts[2] 是小时字段（0=秒, 1=分, 2=时）
  if (parts.length >= 3 && parts[2] !== '*') {
    return parts[2].split(',').map(Number).filter(n => !isNaN(n))
  }
  return [3, 9, 15, 21]
}

// 根据 cron 推断 scheduleMode
function detectScheduleMode(hours) {
  for (const [mode, preset] of Object.entries(presetHours)) {
    if (hours.length === preset.length && hours.every(h => preset.includes(h))) {
      return mode
    }
  }
  if (hours.length === 1) return 'daily'
  return 'custom'
}

// 将选择转换为 cron 表达式
function buildCron() {
  let hours
  if (scheduleMode.value === 'daily') {
    const h = parseInt(dailyHour.value.split(':')[0]) || 0
    hours = [h]
  } else if (scheduleMode.value === 'custom') {
    hours = [...selectedHours.value].sort((a, b) => a - b)
  } else {
    hours = presetHours[scheduleMode.value] || [3, 9, 15, 21]
  }
  return `0 0 ${hours.join(',')} * * *`
}

function toggleHour(h) {
  const idx = selectedHours.value.indexOf(h)
  if (idx >= 0) selectedHours.value.splice(idx, 1)
  else selectedHours.value.push(h)
}

// scheduleMode / selectedHours / dailyHour 变化时自动更新 cron
watch([scheduleMode, selectedHours, dailyHour], () => {
  form.scheduler.cron = buildCron()
}, { deep: true })

async function loadConfig() {
  loading.value = true
  try {
    const c = await GetConfig()
    if (c) {
      Object.assign(form.database, c.database || {})
      Object.assign(form.track17, c.track17 || {})
      Object.assign(form.scheduler, c.scheduler || {})
      Object.assign(form.query, c.query || {})

      // 解析已有 cron → UI 状态
      const hours = parseCronHours(form.scheduler.cron)
      scheduleMode.value = detectScheduleMode(hours)
      if (scheduleMode.value === 'daily' && hours.length === 1) {
        dailyHour.value = `${String(hours[0]).padStart(2, '0')}:00`
      } else if (scheduleMode.value === 'custom') {
        selectedHours.value = hours
      }
    }
    configured.value = await IsConfigured()
  } catch (e) { console.error(e) }
  finally { loading.value = false }
}

async function saveConfig() {
  saving.value = true
  msg.value = ''
  try {
    await SaveConfig(JSON.parse(JSON.stringify(form)))
    msg.value = '✅ 配置已保存！请重启应用以使配置生效。'
    msgOk.value = true
    configured.value = await IsConfigured()
  } catch (e) {
    msg.value = '❌ 保存失败：' + (e.message || e)
    msgOk.value = false
  } finally { saving.value = false }
}

onMounted(loadConfig)
</script>

<style scoped>
.config-input {
  padding: 6px 10px;
  border: 1px solid #dfe6e9;
  border-radius: 4px;
  font-size: 14px;
  width: 100%;
  max-width: 400px;
}
.config-input:focus {
  outline: none;
  border-color: #0984e3;
}
fieldset { border: 1px solid #dfe6e9; }

.hour-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  max-width: 600px;
}
.hour-chip {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: 1px solid #dfe6e9;
  border-radius: 16px;
  font-size: 13px;
  cursor: pointer;
  user-select: none;
  transition: all 0.15s;
  background: #fff;
}
.hour-chip:hover {
  background: #dfe6e9;
}
.hour-chip.active {
  background: #0984e3;
  color: #fff;
  border-color: #0984e3;
}
.hour-chip input {
  display: none;
}
</style>
