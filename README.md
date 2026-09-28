# model-ops · 部署管理压测平台

[中文](README.md) | [English](README.en.md)

围绕 **安装人员要求** 与 **设备验收事宜** 的开源部署管理压测平台（Vue 3 + Go）。

面向 AI 边缘算力中心等批量设备交付场景：从部署任务分派、安装人员资质匹配，到逐台设备的
验收单流转（开箱核对 → 配置核对 → 满载烤机 → 性能实测 → 并发压测 → 软件验收 → 文档归档），
再到烤机/并发压测的执行与结果一键回填验收单，形成完整闭环。

> 数据边界：平台仅承载「安装人员要求」「设备验收事宜」及为支撑验收所必需的节点标识
> （类别名称/数量）。设备配置、价格、性能参数等采购敏感信息一律不入库、不入库代码。

## 功能总览

| 模块 | 说明 |
| --- | --- |
| **设备验收**（核心） | 按节点类别模板逐台（组）生成验收单；条目级判定（合格/不合格/进行中/不适用，合格必须附验收记录）；烤机、并发压测结果一键引用自动判定；任一强制项不合格 → 整改中，全部合格 → 已通过并归档结论 |
| **安装人员**（核心） | 人员名册（等级/技能标签/认证）；岗位要求原文展示；技能矩阵；对任意节点类别的匹配度与缺失技能提示，供任务分派参考 |
| 部署管理 | 按节点类别创建部署任务；阶段推进（到货清点 → 上架安装 → 组网接入 → 系统与驱动 → 压测烤机 → 设备验收 → 交付使用）；安装人员分派（带匹配度）；一键为到货设备生成验收单 |
| 并发压测 | **真实压测引擎**：N 并发模拟 N 名师生访问目标服务，统计 RPS、P50/P90/P95/P99、错误率、状态码分布与逐秒时序；按阈值（默认错误率≤1%、P95≤500ms）自动判定；可关联验收条目完成后自动回填 |
| 满载烤机 | 对应验收条目「72h满载烤机」：逐秒采集每卡温度/功耗/利用率与 ECC 错误计数；判定规则与真实流程一致（ECC=0 且最高温度不超限）。**演示模式为模拟遥测**，接入真实 GPU 时替换采样源为 DCGM-Exporter 即可 |
| 验收规范 | 各节点类别安装人员要求原文与验收事宜条款展示（验收单模板唯一依据） |
| 总览 | 验收状态分布、各节点验收进度、技能覆盖、最近验收动态 |
| **中英文界面** | 侧栏「中文 / EN」一键切换，覆盖全部界面、状态枚举、节点类别/技能/验收条目/岗位要求的展示翻译；选择持久化到 localStorage |

> i18n 边界：存储与 API 仍以中文为业务记录语言（枚举与原文留痕）；英文仅为展示层翻译——验收记录、压测摘要等留痕文本保留原文，符合审计留痕要求。

## 技术栈

- 后端：Go 1.24+（标准库 `net/http`，零第三方依赖）、JSON 文件存储（原子落盘）
- 前端：Vue 3 + Vite 5 + Element Plus + ECharts
- 部署：单二进制托管前端产物，开箱即用

## 快速开始

```bash
# 1) 构建并启动后端（默认 :8080，数据存 server/data/store.json，首次启动自动注入种子数据）
cd server
go run .
# 或: go build -o model-ops . && ./model-ops

# 2) 前端开发模式（热更新，/api 代理到 127.0.0.1:8080）
cd web
npm install
npm run dev          # http://localhost:5173
```

生产部署（单进程）：

```bash
cd web && npm run build        # 产物输出 web/dist
cd ../server && go run .       # 打开 http://127.0.0.1:8080
```

常用参数：`-addr :9000` 监听地址 · `-data ./data/prod.json` 数据文件 · `-web ../web/dist` 前端目录。
跨域白名单可用环境变量 `MOPS_CORS_ORIGIN` 追加（逗号分隔，默认仅放行本地 Vite 开发源）。

## 目录结构

```
server/                 Go 后端
├── main.go             入口：路由、静态资源（含 SPA 回退与路径穿越防护）、超时配置
├── internal/model      领域模型（验收单/条目/人员/任务/压测）与状态机重算
├── internal/seed       种子数据：节点类别、安装人员要求原文、验收模板、示例记录
├── internal/store      JSON 文件存储（读写锁 + 临时文件原子替换）
├── internal/engine     压测引擎：HTTP 并发压测（真实）/ 满载烤机（演示模拟采集）
├── internal/api        REST 处理器与中间件（CORS 白名单、日志）
└── internal/api/*_test.go  单元/集成测试（go test ./...）

web/                    Vue 3 前端
├── src/i18n.js         中英文词典与领域内容翻译（状态/类别/技能/条目/岗位要求）
└── src/views           Dashboard / Acceptance / Installers / Deployment /
                        StressHttp / StressBurnin / Spec
```

## 验收闭环示例

1. 部署管理：任务 `TASK-05` 到达「设备验收」阶段 → 点击「生成验收单」→ `ACC-100x`
2. 设备验收：打开验收单 → 条目①②人工判定合格
3. 条目③「72h满载烤机」→「发起烤机」跳转烤机页（自动关联该条目）→ 演示时长跑完 →
   判定合格 → **自动回填**条目③
4. 条目⑤「校内网并发压测」→ 10 并发压测推理服务 → 阈值判定 → 自动回填条目⑤
5. 全部强制项合格 → 验收单「已通过」→ 填写结论归档；任一项不合格 → 「整改中」复检

## 主要 API

```
GET  /api/overview                      总览统计
GET  /api/categories                    节点类别（含安装人员要求原文与验收模板）
GET  /api/installers      POST /api/installers      PUT/DELETE /api/installers/{id}
GET  /api/tasks           POST /api/tasks           POST /api/tasks/{id}/stage|assign|orders
GET  /api/orders          GET  /api/orders/{id}
PUT  /api/orders/{id}/items/{key}                   条目判定（合格/不合格须附记录）
POST /api/orders/{id}/items/{key}/apply-stress      引用压测/烤机结果自动判定
GET/POST /api/stress/http    GET /api/stress/http/{id}/live
GET/POST /api/stress/burnin  GET /api/stress/burnin/{id}/live
```

## 安全与边界

- 并发压测目标仅允许 **本机/内网地址**（127.0.0.1、10.x、172.16-31.x、192.168.x），防止平台被滥用于攻击外部站点
- CORS 白名单制，默认只放行本地前端开发源，可用 `MOPS_CORS_ORIGIN` 显式扩展
- 静态资源服务做了路径穿越防护（解析路径必须仍在根目录内）
- HTTP 服务配置了 Read/Write/Idle 超时
- 烤机页明示「演示模式：模拟遥测」；接入真实 GPU 只需替换采样源（DCGM-Exporter / nvidia-smi），判定逻辑不变

## 测试

```bash
cd server && go test ./...     # 种子数据、验收状态机、真实压测引擎、烤机判定
```

## License

[MIT](LICENSE)
