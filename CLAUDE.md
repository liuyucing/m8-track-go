# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

M8 物流轨迹同步服务 — a Wails v3 desktop app that syncs logistics tracking data from SQL Server via the 17track API. Migrated from a Java Spring Boot project.

## Commands

```bash
# Development (launches desktop window with hot-reload)
wails3 dev

# Production build (output in bin/)
wails3 build

# Frontend only (from frontend/)
cd frontend && npm run dev

# Go build (no GUI, for checking compilation)
go build ./...
```

No test framework is configured yet. There are no `*_test.go` files.

## Architecture

### Backend (Go)

`main.go` bootstraps everything: loads config, opens DB (if configured), wires repos/services, creates the Wails app with `AppService` as the sole exposed service.

**Data flow (two-phase sync):**
1. `TrackSyncService.RegisterPendingOrders` — queries `scbn`/`scBNDtl` tables for untracked shipments, registers tracking numbers with 17track `/register` endpoint, inserts into `track_sync_record`
2. `TrackSyncService.SyncTrackingInfo` — fetches tracking updates from 17track `/gettrackinfo`, updates `track_sync_record`, inserts new events into `track_sync_detail`, and writes status back to `scBNDtl.FCtrack` / `scBNDtl.TrackDelivered`

**Package responsibilities:**
- `config/` — YAML config loading with `config.yaml`, defaults via `DefaultConfig()`, `IsConfigured()` guards
- `internal/model/` — plain structs with `db` and `json` tags; `track17.go` holds the 17track API request/response types
- `internal/repository/` — raw `database/sql` queries against SQL Server. Uses `@p1, @p2...` parameter placeholders (go-mssqldb convention). `ShipOrderRepo` reads/writes the ERP tables `scbn`/`scBNDtl`; `TrackRecordRepo` and `TrackDetailRepo` manage the local sync tables
- `internal/trackapi/` — HTTP client for 17track v2.4 API (`/register`, `/gettrackinfo`). Auth via `17token` header. Generic `Partition[T]` for batching
- `internal/service/` — `TrackSyncService` (core sync logic) and `Scheduler` (cron via `robfig/cron/v3` with 6-field format including seconds). Scheduler has mutex-guarded `isRunning` to prevent re-entrant sync
- `internal/app/` — `AppService` is the Wails service layer. All methods exposed to frontend are on this single struct. Handles dashboard stats, manual sync trigger, order listing, log viewing, config read/save with password masking

**Graceful degradation:** If `config.yaml` is missing or `IsConfigured()` returns false, the app starts without DB/services. Users configure via the GUI Config tab, then restart.

### Frontend (Vue 3 + Vite)

- Single-page app with tab navigation: Dashboard / Orders / Logs / Config
- Wails auto-generates JS bindings in `frontend/bindings/` — the composable `frontend/src/composables/useApi.js` re-exports them
- `frontend/dist/` is embedded into the Go binary via `//go:embed`
- No router, no state management library — tabs controlled by a `ref` in `App.vue`

### Database

SQL Server (go-mssqldb driver). Two "domains" of tables:
- **ERP tables** (`scbn`, `scBNDtl`) — read-only except `FCtrack` and `TrackDelivered` columns written back by the sync service
- **Local sync tables** (`track_sync_record`, `track_sync_detail`) — created and managed by this application

Config default DB: `FumaCRM8` on port `3366`.

## Key Conventions

- SQL Server uses `@p1, @p2...` parameter placeholders (not `?` or `$1`)
- Cron expressions are 6-field with seconds: `0 0 3,9,15,21 * * *`
- All log messages and error messages are in Chinese
- Nullable model fields use `*string` / `*int` / `*time.Time` pointers
- `SCOPE_IDENTITY()` is used after INSERT to retrieve auto-increment IDs
- Module name is `m8-track-go` (not a full URL path)
