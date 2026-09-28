# model-ops · Deployment & Stress-test Platform

[中文](README.md) | [English](README.en.md)

An open-source deployment management & stress-test platform (Vue 3 + Go) centered on
**installer requirements** and **device acceptance**.

Built for batch device delivery scenarios such as AI edge computing centers: from deployment
task assignment and installer qualification matching, to per-device acceptance order workflows
(unboxing check → configuration check → burn-in → performance test → concurrency test →
software acceptance → documentation), to burn-in/concurrency test execution with one-click
result backfill into acceptance orders — a complete closed loop.

> Data boundary: the platform carries only the "installer requirements" and "device acceptance"
> clauses plus the node identifiers necessary to support acceptance (category name / quantity).
> Procurement-sensitive information such as device configurations, prices and performance
> parameters is never stored in the platform or the code.

## Features

| Module | Description |
| --- | --- |
| **Device Acceptance** (core) | Generate acceptance orders per device from node-category templates; item-level verdicts (Pass / Fail / In Progress / N/A — Pass/Fail requires a record); one-click application of burn-in / concurrency results with automatic verdict; any mandatory failure → Rework, all mandatory pass → Passed with archived conclusion |
| **Installers** (core) | Roster (level / skill tags / certifications); verbatim job requirements; skill matrix; match rate and missing-skill hints per node category for task assignment |
| Deployment | Create tasks per node category; stage pipeline (arrival check → rack install → network setup → OS & drivers → stress & burn-in → acceptance → in service); installer assignment with match rate; one-click acceptance order generation |
| Concurrency Test | **Real load engine**: N concurrent workers simulate N students/staff hitting the target service, measuring RPS, P50/P90/P95/P99, error rate, status-code distribution and per-second series; automatic verdict against thresholds (default: error ≤1%, P95 ≤500ms); links to acceptance items for automatic backfill |
| Burn-in | For the acceptance item "72h full-load burn-in": per-GPU temperature/power/utilization and ECC error counters; verdict identical to the real process (ECC = 0 and max temperature within limit). **Demo mode uses simulated telemetry** — swap the sample source for DCGM-Exporter to run against real GPUs |
| Acceptance Spec | Verbatim installer requirements and acceptance clauses per node category (the single source of order templates) |
| Overview | Acceptance status distribution, per-category acceptance progress, skill coverage, recent updates |
| **Bilingual UI (中文 / EN)** | One-click language switch in the sidebar covering the whole UI, status enums and display translations of node categories / skills / acceptance items / job requirements; persisted in localStorage |

> i18n boundary: storage and APIs keep Chinese as the language of record (enums and verbatim
> evidence); English is display-layer only — recorded evidence such as acceptance records and
> stress summaries keeps its original language, as expected for audit trails.

## Tech Stack

- Backend: Go 1.24+ (standard-library `net/http`, zero third-party dependencies), JSON file storage with atomic writes
- Frontend: Vue 3 + Vite 5 + Element Plus + ECharts
- Deployment: a single binary serves the frontend bundle — ready to run out of the box

## Quick Start

```bash
# 1) Build and start the backend (default :8080, data at server/data/store.json,
#    seed data is injected automatically on first start)
cd server
go run .
# or: go build -o model-ops . && ./model-ops

# 2) Frontend dev mode (hot reload, /api proxied to 127.0.0.1:8080)
cd web
npm install
npm run dev          # http://localhost:5173
```

Production (single process):

```bash
cd web && npm run build        # outputs to web/dist
cd ../server && go run .       # open http://127.0.0.1:8080
```

Common flags: `-addr :9000` listen address · `-data ./data/prod.json` data file ·
`-web ../web/dist` frontend directory.
Additional CORS origins can be allowed via the `MOPS_CORS_ORIGIN` environment variable
(comma-separated; by default only the local Vite dev origin is allowed).

## Project Layout

```
server/                 Go backend
├── main.go             entry: routing, static assets (SPA fallback + path-traversal guard), timeouts
├── internal/model      domain models (orders/items/installers/tasks/stress) and status recalculation
├── internal/seed       seed data: node categories, verbatim installer requirements, acceptance templates
├── internal/store      JSON file storage (RW lock + atomic replace via temp file)
├── internal/engine     stress engines: HTTP concurrency (real) / burn-in (demo simulated telemetry)
├── internal/api        REST handlers and middleware (CORS allowlist, logging)
└── internal/api/*_test.go  unit/integration tests (go test ./...)

web/                    Vue 3 frontend
├── src/i18n.js         zh/en dictionaries and domain translations (statuses/categories/skills/items/requirements)
└── src/views           Dashboard / Acceptance / Installers / Deployment /
                        StressHttp / StressBurnin / Spec
```

## Acceptance Loop Example

1. Deployment: task `TASK-05` reaches the "Acceptance" stage → click "Create Order" → `ACC-100x`
2. Device Acceptance: open the order → judge items ①② manually as Pass
3. Item ③ "72h burn-in" → "Start Burn-in" jumps to the burn-in page (auto-linked to the item)
   → demo duration finishes → verdict Pass → item ③ is **backfilled automatically**
4. Item ⑤ "campus network concurrency test" → 10-user test against the inference service →
   threshold verdict → item ⑤ backfilled automatically
5. All mandatory items pass → order "Passed" → archive the conclusion; any failure → "Rework" for re-inspection

## Key APIs

```
GET  /api/overview                      dashboard stats
GET  /api/categories                    node categories (verbatim requirements + acceptance templates)
GET  /api/installers      POST /api/installers      PUT/DELETE /api/installers/{id}
GET  /api/tasks           POST /api/tasks           POST /api/tasks/{id}/stage|assign|orders
GET  /api/orders          GET  /api/orders/{id}
PUT  /api/orders/{id}/items/{key}                   item verdict (Pass/Fail requires a record)
POST /api/orders/{id}/items/{key}/apply-stress      apply stress/burn-in result with automatic verdict
GET/POST /api/stress/http    GET /api/stress/http/{id}/live
GET/POST /api/stress/burnin  GET /api/stress/burnin/{id}/live
```

## Security & Boundaries

- Concurrency test targets are restricted to **localhost / intranet addresses** (127.0.0.1, 10.x, 172.16-31.x, 192.168.x) to prevent abuse against external sites
- Allowlist-based CORS: only the local frontend dev origin by default, extendable explicitly via `MOPS_CORS_ORIGIN`
- Static file serving is guarded against path traversal (resolved paths must stay inside the root)
- HTTP server configured with Read/Write/Idle timeouts
- The burn-in page clearly labels "demo mode: simulated telemetry"; connecting real GPUs only requires swapping the sample source (DCGM-Exporter / nvidia-smi) — the verdict logic stays unchanged

## Tests

```bash
cd server && go test ./...     # seed data, acceptance state machine, real load engine, burn-in verdicts
```

## License

[MIT](LICENSE)
