# M8 物流轨迹同步服务 (Go + Wails)

从 Java Spring Boot 项目重构为 Golang + Wails v3 桌面应用。

## 功能

- 🚚 自动从 SQL Server 查询未签收的物流订单
- 📡 向 17track API 注册运单号并获取物流轨迹
- ⏰ 定时同步（默认每天 3:00/9:00/15:00/21:00）
- 🖥️ Wails 桌面窗口界面（仪表盘/订单列表/同步日志/配置）
- ▶️ 支持手动触发同步

## 快速开始

### 1. 安装 Wails CLI

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
```

### 2. 配置

```bash
cp config.yaml.example config.yaml
# 编辑 config.yaml 填入真实数据库和 API 配置
```

### 3. 开发模式

```bash
wails3 dev
```

启动后会自动打开桌面窗口。

### 4. 生产构建

```bash
wails3 build
```

生成的可执行文件在 `bin/` 目录。

> **注意：** 如果 `wails3 build` 未能正确生成 exe，可直接使用：
> ```bash
> go build -ldflags="-s -w" -o bin/m8-track-go.exe .
> ```
记得先 cd frontend && npm run build 再重新构建 exe，否则嵌入的还是旧的前端。

## 配置说明 (config.yaml)

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `database.host` | SQL Server 地址 | - |
| `database.port` | SQL Server 端口 | 3366 |
| `database.name` | 数据库名 | FumaCRM8 |
| `database.username` | 数据库用户名 | sa |
| `database.password` | 数据库密码 | - |
| `database.encrypt` | 是否加密连接 | false |
| `database.trust_cert` | 是否信任服务器证书 | true |
| `track17.api_key` | 17track API 密钥 | - |
| `track17.base_url` | 17track API 地址 | `https://api.17track.net/track/v2.4` |
| `track17.batch_size` | 每批处理数量 | 40 |
| `track17.http_timeout_ms` | HTTP 超时（毫秒） | 30000 |
| `scheduler.cron` | 定时同步 Cron（6字段含秒） | `0 0 3,9,15,21 * * *` |
| `scheduler.enabled` | 是否启用定时同步 | true |
| `query.order_date_filter` | 只同步此日期之后的订单 | `2026-05-01` |
| `app.log_level` | 日志级别 | debug |

> **首次启动：** 如果 `config.yaml` 不存在或未填写必要配置，应用会以"未配置"模式启动，可在 GUI 配置页面填写后保存并重启。

## 项目结构

```
├── main.go                         # Wails 应用入口
├── config.yaml                     # 配置文件（不入版本控制）
├── config.yaml.example             # 配置文件模板
├── Taskfile.yml                    # Wails 构建任务定义
├── build/                          # 构建配置
│   ├── config.yml                  # Wails 开发模式配置
│   └── windows/Taskfile.yml        # Windows 构建任务
├── bin/                            # 构建输出目录
├── config/                         # 配置加载
│   └── config.go                   # YAML 解析、默认值、校验
├── internal/
│   ├── model/                      # 数据模型
│   │   ├── ship_order.go           # 物流订单（映射 scbn/scBNDtl）
│   │   ├── track_sync_record.go    # 同步记录（映射 track_sync_record）
│   │   ├── track_sync_detail.go    # 轨迹事件（映射 track_sync_detail）
│   │   └── track17.go              # 17track API 请求/响应类型
│   ├── repository/                 # 数据库访问 (SQL Server)
│   │   ├── db.go                   # 连接池初始化
│   │   ├── ship_order_repo.go      # ERP 表 scbn/scBNDtl 读写
│   │   ├── track_record_repo.go    # track_sync_record CRUD
│   │   └── track_detail_repo.go    # track_sync_detail CRUD
│   ├── trackapi/                   # 17track API 客户端
│   │   ├── client.go               # HTTP 客户端（17token 认证）
│   │   └── track17.go              # /register、/gettrackinfo、分批工具
│   ├── service/                    # 业务逻辑
│   │   ├── track_sync.go           # 两阶段同步（注册→查询轨迹）
│   │   └── scheduler.go            # cron 定时调度（防重入）
│   └── app/                        # Wails 服务层（暴露给前端）
│       └── app_service.go          # 仪表盘/订单/日志/配置/手动同步
└── frontend/                       # Vue 3 + Vite 前端
    ├── package.json
    ├── bindings/                   # Wails 自动生成的 JS 绑定
    └── src/
        ├── App.vue                 # 主布局（Tab 导航）
        ├── composables/useApi.js   # 统一导出 Wails 绑定
        └── views/                  # Dashboard / Orders / Logs / Config
```

## 核心流程

```
┌─────────────────────────────────────────────────────┐
│                   SyncAll (定时/手动)                 │
├──────────────────────┬──────────────────────────────┤
│  1. RegisterPending  │  2. SyncTrackingInfo          │
│                      │                               │
│  SQL Server          │  17track /gettrackinfo        │
│  scbn+scBNDtl        │         ↓                     │
│     ↓                │  解析轨迹状态/事件              │
│  过滤未注册运单       │         ↓                     │
│     ↓                │  更新 track_sync_record        │
│  17track /register   │  插入 track_sync_detail        │
│     ↓                │         ↓                     │
│  写入 track_sync_    │  回写 scBNDtl.FCtrack          │
│  record              │  签收则标记 TrackDelivered=1    │
└──────────────────────┴──────────────────────────────┘
```

## 数据库

使用 SQL Server（go-mssqldb 驱动），涉及两组表：

- **ERP 表**（`scbn`、`scBNDtl`）— 系统只读，仅回写 `FCtrack` 和 `TrackDelivered` 字段
- **本地同步表**（`track_sync_record`、`track_sync_detail`）— 由本系统创建和维护

## 技术栈

- **后端**: Go 1.26 + Wails v3 + go-mssqldb + robfig/cron
- **前端**: Vue 3 + Vite + @wailsio/runtime
- **数据库**: Microsoft SQL Server
- **外部API**: 17track v2.4

## 与原 Java 项目对比

| 特性 | Java (原) | Go (新) |
|------|-----------|---------|
| 框架 | Spring Boot 3.2 | Wails v3 |
| 界面 | 无（命令行） | 桌面窗口 GUI |
| 数据库 | MyBatis Plus + Druid | database/sql + go-mssqldb |
| HTTP | Java HttpClient | net/http |
| 定时 | @Scheduled | robfig/cron v3 |
| 配置 | application-local.yaml | config.yaml |
| 构建 | Maven → JAR | go build → EXE |
