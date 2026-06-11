// Wails bindings - 直接调用 Go 后端方法
import { GetDashboardStats, TriggerSync, GetOrders, GetOrderDetails, GetRecentLogs, GetConfig, SaveConfig, IsConfigured } from '../../bindings/m8-track-go/internal/app/appservice.js'

export { GetDashboardStats, TriggerSync, GetOrders, GetOrderDetails, GetRecentLogs, GetConfig, SaveConfig, IsConfigured }

export function useApi() {
  return { GetDashboardStats, TriggerSync, GetOrders, GetOrderDetails, GetRecentLogs, GetConfig, SaveConfig, IsConfigured }
}
