<template>
  <div class="app">
    <header class="header">
      <h1>M8 物流轨迹同步</h1>
    </header>
    <nav class="tabs">
      <button v-for="tab in tabs" :key="tab.key"
        :class="['tab', { active: activeTab === tab.key }]"
        @click="activeTab = tab.key">
        {{ tab.label }}
      </button>
    </nav>
    <main class="content">
      <Dashboard v-if="activeTab === 'dashboard'" />
      <Orders v-else-if="activeTab === 'orders'" />
      <Logs v-else-if="activeTab === 'logs'" />
      <ConfigView v-else-if="activeTab === 'config'" />
    </main>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import Dashboard from './views/Dashboard.vue'
import Orders from './views/Orders.vue'
import Logs from './views/Logs.vue'
import ConfigView from './views/Config.vue'

const activeTab = ref('dashboard')
const tabs = [
  { key: 'dashboard', label: '仪表盘' },
  { key: 'orders', label: '订单列表' },
  { key: 'logs', label: '同步日志' },
  { key: 'config', label: '配置' },
]
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f5f6fa; color: #2d3436; }
.app { max-width: 1200px; margin: 0 auto; padding: 20px; }
.header { margin-bottom: 20px; }
.header h1 { font-size: 24px; color: #2d3436; }
.tabs { display: flex; gap: 8px; margin-bottom: 20px; border-bottom: 2px solid #dfe6e9; padding-bottom: 8px; }
.tab { padding: 8px 20px; border: 1px solid #dfe6e9; border-radius: 6px 6px 0 0; background: #fff; cursor: pointer; font-size: 14px; color: #636e72; transition: all 0.2s; }
.tab:hover { background: #dfe6e9; }
.tab.active { background: #0984e3; color: #fff; border-color: #0984e3; }
.content { background: #fff; border-radius: 8px; padding: 24px; box-shadow: 0 2px 8px rgba(0,0,0,0.06); }

.card { background: #fff; border: 1px solid #dfe6e9; border-radius: 8px; padding: 16px; text-align: center; }
.card .value { font-size: 32px; font-weight: 700; color: #2d3436; }
.card .label { font-size: 13px; color: #636e72; margin-top: 4px; }

.stats-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(160px, 1fr)); gap: 16px; margin-bottom: 24px; }

.btn { padding: 8px 20px; border: none; border-radius: 6px; cursor: pointer; font-size: 14px; transition: all 0.2s; }
.btn-primary { background: #0984e3; color: #fff; }
.btn-primary:hover { background: #0652DD; }
.btn-primary:disabled { background: #b2bec3; cursor: not-allowed; }

table { width: 100%; border-collapse: collapse; }
th, td { padding: 10px 12px; text-align: left; border-bottom: 1px solid #dfe6e9; font-size: 13px; }
th { background: #f5f6fa; font-weight: 600; color: #636e72; }
tr:hover { background: #f8f9fa; }

.badge { display: inline-block; padding: 2px 8px; border-radius: 10px; font-size: 12px; }
.badge-success { background: #d4edda; color: #155724; }
.badge-info { background: #d1ecf1; color: #0c5460; }
.badge-warning { background: #fff3cd; color: #856404; }

.loading { text-align: center; padding: 40px; color: #636e72; }
.error { color: #d63031; padding: 12px; background: #ffeaea; border-radius: 6px; margin-bottom: 16px; }

.config-grid { display: grid; grid-template-columns: 140px 1fr; gap: 8px 16px; font-size: 14px; }
.config-key { color: #636e72; font-weight: 500; }
.config-val { color: #2d3436; word-break: break-all; }
</style>
