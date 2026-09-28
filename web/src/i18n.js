// 轻量 i18n：界面中英文切换。
// 说明：存储与 API 仍使用中文枚举/原文（作为业务记录语言），
// 本模块仅在展示层做翻译；验收记录、压测摘要等留痕文本保留原文。
import { ref } from 'vue'

export const locale = ref(localStorage.getItem('mops-locale') || 'zh')

export function setLocale(l) {
  locale.value = l
  localStorage.setItem('mops-locale', l)
}

const dict = {
  zh: {
    'app.title': '部署管理压测平台',
    'app.sub': '安装人员要求 · 设备验收事宜',
    'menu.dashboard': '总览',
    'menu.acceptance': '设备验收',
    'menu.installers': '安装人员',
    'menu.deployment': '部署管理',
    'menu.stressHttp': '并发压测',
    'menu.stressBurnin': '满载烤机',
    'menu.spec': '验收规范',

    'common.actions': '操作',
    'common.detail': '详情',
    'common.cancel': '取消',
    'common.submit': '提交',
    'common.save': '保存',
    'common.confirm': '确认',
    'common.edit': '编辑',
    'common.delete': '删除',
    'common.create': '创建',
    'common.status': '状态',
    'common.time': '时间',
    'common.name': '任务名称',
    'common.id': '编号',
    'common.task': '任务',
    'common.verdict': '判定',
    'common.running': '运行中',
    'common.pass': '合格',
    'common.fail': '不合格',
    'common.lang': '语言',

    'dash.title': '总览',
    'dash.desc': '安装人员要求与设备验收事宜全景：验收进度、压测执行、人岗匹配覆盖。',
    'dash.orders': '验收单总数',
    'dash.passed': '已通过',
    'dash.rework': '整改中',
    'dash.installers': '安装人员',
    'dash.deploying': '进行中部署',
    'dash.records': '压测记录',
    'dash.statusPie': '验收单状态分布',
    'dash.nodeBar': '各节点类别验收进度（已生成验收单 / 设备数量）',
    'dash.recent': '最近验收动态',
    'dash.skillCover': '安装技能覆盖（具备人数）',
    'dash.col.order': '验收单',
    'dash.col.device': '设备',
    'dash.col.progress': '进度',
    'dash.col.updated': '更新时间',
    'dash.legend.qty': '设备数量',
    'dash.legend.created': '已生成验收单',
    'dash.legend.passed': '已通过',

    'acc.title': '设备验收',
    'acc.desc': '按验收规范逐台（组）生成验收单：开箱核对 → 配置核对 → 满载烤机 → 性能实测 → 并发压测 → 软件验收 → 文档归档；烤机与并发压测可一键引用压测中心结果自动判定。',
    'acc.filter.status': '验收单状态',
    'acc.filter.cat': '节点类别',
    'acc.filter.kw': '搜索 验收单号 / 设备',
    'acc.create': '生成验收单',
    'acc.col.no': '验收单号',
    'acc.col.device': '设备',
    'acc.col.cat': '节点类别',
    'acc.col.inspector': '验收人',
    'acc.col.progress': '进度',
    'acc.col.updated': '更新时间',
    'acc.btn.accept': '验收',
    'acc.drawer.order': '验收单',
    'acc.drawer.created': '创建时间',
    'acc.reworkAlert': '存在不合格条目，需整改后复检',
    'acc.mandatory': '强制项',
    'acc.record': '验收记录：',
    'acc.by': '验收人：',
    'acc.judge': '判定结论',
    'acc.startBurnin': '发起烤机',
    'acc.applyBurnin': '引用烤机结果',
    'acc.startHttp': '发起并发压测',
    'acc.applyHttp': '引用压测结果',
    'acc.conclusion': '验收结论归档',
    'acc.conclusionPh': '全部强制项合格后填写最终结论并归档',
    'acc.saveConclusion': '保存结论',
    'acc.judgeDialog': '判定',
    'acc.judgeResult': '验收记录',
    'acc.judgeResultFail': '记录不合格详情与整改要求',
    'acc.judgeResultPass': '记录核对/实测结果',
    'acc.evidence': '证据说明',
    'acc.evidencePh': '如：出厂测试报告编号 / 压测记录编号',
    'acc.applyDialog': '引用压测结果',
    'acc.applySummary': '摘要',
    'acc.applyBackfill': '回填到验收条目',
    'acc.createDialog': '生成验收单',
    'acc.createTask': '部署任务',
    'acc.createTaskPh': '选择有剩余设备的部署任务',
    'acc.createGen': '（已生成 {a}/{b} {u}）',
    'acc.inspectorPh': '可稍后在验收单中指定',
    'acc.msg.itemSaved': '条目已判定：',
    'acc.msg.backfilled': '压测结果已回填，条目自动判定',
    'acc.msg.orderCreated': '已生成验收单',
    'acc.msg.conclusionSaved': '结论已归档',
    'acc.msg.resultRequired': '判定合格/不合格必须填写验收记录',

    'ins.title': '安装人员',
    'ins.desc': '管理安装人员资质与技能标签，并对照各节点类别的「安装人员要求」做人岗匹配；分派部署任务时可参考匹配度。',
    'ins.reqTitle': '安装人员要求（岗位规范）',
    'ins.gpuNodes': 'GPU 类节点（科研微调 / 推理 / 教学 / 智能体 / 实训集群）',
    'ins.cpuNode': 'CPU 仿真节点',
    'ins.roster': '人员名册（{n} 人）',
    'ins.add': '新增人员',
    'ins.col.name': '姓名',
    'ins.col.title': '岗位',
    'ins.col.level': '等级',
    'ins.col.phone': '联系方式',
    'ins.col.skills': '技能标签',
    'ins.col.available': '在岗',
    'ins.available': '在岗',
    'ins.busy': '占用',
    'ins.matrix': '技能矩阵 · 人岗匹配',
    'ins.reqSkill': '要求技能',
    'ins.match': '匹配度',
    'ins.missing': '缺失技能',
    'ins.allMet': '全部满足',
    'ins.matchFor': '对「{c}」的匹配度',
    'ins.edit': '编辑人员',
    'ins.fName': '姓名',
    'ins.fTitle': '岗位',
    'ins.fTitlePh': '如：GPU集群部署专家',
    'ins.fLevel': '等级',
    'ins.fPhone': '联系方式',
    'ins.gpuSkills': 'GPU 技能',
    'ins.hpcSkills': 'HPC 技能',
    'ins.fAvailable': '在岗',
    'ins.msgSaved': '已保存',
    'ins.msgDeleted': '已删除',
    'ins.msgNameRequired': '姓名不能为空',
    'ins.deleteConfirm': '删除 {n}？',

    'dep.title': '部署管理',
    'dep.desc': '按节点类别跟踪设备部署全流程：到货清点 → 上架安装 → 组网接入 → 系统与驱动 → 压测烤机 → 设备验收 → 交付使用；安装人员分派参考「安装人员」页的匹配度。',
    'dep.tasks': '部署任务',
    'dep.new': '新建部署任务',
    'dep.col.no': '任务号',
    'dep.col.cat': '节点类别',
    'dep.col.qty': '数量',
    'dep.col.stage': '当前阶段',
    'dep.col.installers': '安装人员',
    'dep.col.orders': '验收单',
    'dep.col.started': '开始时间',
    'dep.unassigned': '未分派',
    'dep.current': '当前阶段',
    'dep.advance': '推进阶段',
    'dep.assign': '分派人员',
    'dep.assignDialog': '分派安装人员',
    'dep.reqColon': '岗位要求：',
    'dep.missing': '缺 ',
    'dep.confirmAssign': '确认分派',
    'dep.newDialog': '新建部署任务',
    'dep.fCat': '节点类别',
    'dep.fNote': '备注',
    'dep.msgAdvanced': '已推进至：',
    'dep.msgAssigned': '分派已更新',
    'dep.msgCreated': '已创建部署任务',

    'http.title': '校内网并发压测',
    'http.desc': '真实并发压测引擎：N 个并发模拟 N 名师生同时访问目标服务，统计 RPS、P50/P90/P95/P99 延迟、错误率与逐秒时序，对应验收条目「校内网性能与N人并发压测：模拟师生访问，验证响应时间与稳定性」（默认 10 人并发）。',
    'http.new': '发起压测',
    'http.namePh': '如：推理节点 10人并发验收压测',
    'http.target': '目标地址',
    'http.targetPh': 'http://127.0.0.1:8080/api/health（仅限本机/内网）',
    'http.concurrency': '并发数',
    'http.duration': '时长（秒）',
    'http.maxErr': '错误率阈值%',
    'http.maxP95': 'P95阈值(ms)',
    'http.linkOrder': '关联验收单',
    'http.linkPh': '可选：完成后回填验收条目',
    'http.item': '验收条目',
    'http.start': '开始压测',
    'http.live': '实时监控',
    'http.total': '总请求数',
    'http.rps': 'RPS',
    'http.avg': '平均延迟',
    'http.p95': 'P95',
    'http.errRate': '错误率（阈值 {v}%）',
    'http.failed': '失败请求数',
    'http.seriesRps': '每秒请求数',
    'http.seriesAvg': '平均延迟(ms)',
    'http.seriesErr': '每秒错误',
    'http.timeAxis': '时间',
    'http.history': '压测记录',
    'http.col.target': '目标',
    'http.col.conc': '并发',
    'http.col.duration': '时长',
    'http.col.p95': 'P95(ms)',
    'http.col.err': '错误率',
    'http.backfill': '回填验收',
    'http.detailTitle': '压测详情',
    'http.dTotal': '总请求',
    'http.dSuccess': '成功',
    'http.dFailed': '失败',
    'http.dAvg': '平均',
    'http.dMin': '最小',
    'http.dMax': '最大',
    'http.codes': '状态码分布：',
    'http.msgTargetRequired': '请填写目标地址',
    'http.msgStarted': '压测已启动：',
    'http.msgPass': '压测完成并判定合格：',
    'http.msgFail': '压测完成，判定不合格（阈值：错误率≤{e}%、P95≤{p}ms），详见记录',
    'http.msgBackfilled': '已回填验收单',
    'http.msgAutoBackfill': '验收单条目已按压测结果自动判定',

    'burn.title': '满载烤机监控',
    'burn.desc': '对应验收条目「③ 72h满载烤机：运行GPU压测监测温度/功耗/稳定性，ECC事件日志确认0错误」。判定标准与真实流程一致：满载时长内 ECC 错误累计为 0 且最高温度不超限 → 合格。',
    'burn.demoTitle': '演示模式说明',
    'burn.demoDesc': '平台宿主机无真实 GPU，遥测数据为模拟采集（温度爬升/按卡功耗/利用率/ECC计数）。接入真实环境时，将采样源替换为 DCGM-Exporter / nvidia-smi 即可，判定逻辑无需改动。',
    'burn.new': '发起烤机任务',
    'burn.namePh': '如：科研微调节点 72h满载烤机',
    'burn.device': '设备',
    'burn.devicePh': '设备标签',
    'burn.gpus': 'GPU数量',
    'burn.watt': '单卡功耗(W)',
    'burn.duration': '时长',
    'burn.demo2min': '演示2分钟',
    'burn.maxTemp': '温度限值(°C)',
    'burn.inject': '故障注入',
    'burn.injectLabel': '演示注入ECC错误',
    'burn.start': '开始烤机',
    'burn.live': '实时监控',
    'burn.curMaxTemp': '当前最高温度（限值 {v}°C）',
    'burn.curPower': '整机当前功耗（{n} 卡）',
    'burn.ecc': 'ECC 错误累计（须为 0）',
    'burn.samples': '已采样点数',
    'burn.verdictPass': '判定合格',
    'burn.verdictFail': '判定不合格',
    'burn.history': '烤机记录',
    'burn.col.device': '设备',
    'burn.col.maxTemp': '最高温度',
    'burn.col.duration': '时长',
    'burn.detailTitle': '烤机详情',
    'burn.totalPower': '整机功耗(W)',
    'burn.msgStarted': '烤机任务已启动：',
    'burn.msgBackfilled': '已回填验收单',
    'burn.msgAutoBackfill': '验收单条目已按烤机结果自动判定',

    'spec.title': '验收规范 · 安装人员要求',
    'spec.desc': '各节点类别的安装人员要求原文与设备验收事宜条款，是验收单条目模板与人岗匹配的唯一依据。',
    'spec.req': '安装人员要求',
    'spec.gpuCommon': 'GPU 类节点（通用规范）',
    'spec.cpuTitle': 'CPU 仿真节点',
    'spec.tmpl': '设备验收事宜（验收单条目模板）',
    'spec.col.item': '条目',
    'spec.col.content': '验收内容',
    'spec.col.method': '执行方式',
    'spec.applyTo': '适用节点类别',
    'spec.col.no': '编号',
    'spec.col.qty': '推荐量',
    'spec.reqSkills': '要求技能标签：',
  },

  en: {
    'app.title': 'Deployment & Stress-test Platform',
    'app.sub': 'Installer Requirements · Device Acceptance',
    'menu.dashboard': 'Overview',
    'menu.acceptance': 'Device Acceptance',
    'menu.installers': 'Installers',
    'menu.deployment': 'Deployment',
    'menu.stressHttp': 'Concurrency Test',
    'menu.stressBurnin': 'Burn-in',
    'menu.spec': 'Acceptance Spec',

    'common.actions': 'Actions',
    'common.detail': 'Detail',
    'common.cancel': 'Cancel',
    'common.submit': 'Submit',
    'common.save': 'Save',
    'common.confirm': 'Confirm',
    'common.edit': 'Edit',
    'common.delete': 'Delete',
    'common.create': 'Create',
    'common.status': 'Status',
    'common.time': 'Time',
    'common.name': 'Task Name',
    'common.id': 'ID',
    'common.task': 'Task',
    'common.verdict': 'Verdict',
    'common.running': 'Running',
    'common.pass': 'Pass',
    'common.fail': 'Fail',
    'common.lang': 'Language',

    'dash.title': 'Overview',
    'dash.desc': 'Full picture of installer requirements and device acceptance: acceptance progress, stress tests, and skill matching.',
    'dash.orders': 'Acceptance Orders',
    'dash.passed': 'Passed',
    'dash.rework': 'Rework',
    'dash.installers': 'Installers',
    'dash.deploying': 'Deploying',
    'dash.records': 'Stress Records',
    'dash.statusPie': 'Order Status Distribution',
    'dash.nodeBar': 'Acceptance Progress by Node Category (orders created / quantity)',
    'dash.recent': 'Recent Updates',
    'dash.skillCover': 'Installer Skill Coverage (headcount)',
    'dash.col.order': 'Order',
    'dash.col.device': 'Device',
    'dash.col.progress': 'Progress',
    'dash.col.updated': 'Updated',
    'dash.legend.qty': 'Quantity',
    'dash.legend.created': 'Orders Created',
    'dash.legend.passed': 'Passed',

    'acc.title': 'Device Acceptance',
    'acc.desc': 'Create acceptance orders per device from the spec template: unboxing check → configuration check → burn-in → performance test → concurrency test → software acceptance → documentation. Burn-in and concurrency results can be applied to items automatically.',
    'acc.filter.status': 'Order status',
    'acc.filter.cat': 'Node category',
    'acc.filter.kw': 'Search order / device',
    'acc.create': 'Create Order',
    'acc.col.no': 'Order No.',
    'acc.col.device': 'Device',
    'acc.col.cat': 'Category',
    'acc.col.inspector': 'Inspector',
    'acc.col.progress': 'Progress',
    'acc.col.updated': 'Updated',
    'acc.btn.accept': 'Accept',
    'acc.drawer.order': 'Order',
    'acc.drawer.created': 'Created',
    'acc.reworkAlert': 'Failed items found — rework and re-inspection required',
    'acc.mandatory': 'Mandatory',
    'acc.record': 'Record: ',
    'acc.by': 'By ',
    'acc.judge': 'Verdict',
    'acc.startBurnin': 'Start Burn-in',
    'acc.applyBurnin': 'Apply Burn-in Result',
    'acc.startHttp': 'Start HTTP Test',
    'acc.applyHttp': 'Apply Test Result',
    'acc.conclusion': 'Conclusion Archive',
    'acc.conclusionPh': 'Fill in the final conclusion after all mandatory items pass',
    'acc.saveConclusion': 'Save Conclusion',
    'acc.judgeDialog': 'Verdict',
    'acc.judgeResult': 'Record',
    'acc.judgeResultFail': 'Describe the failure and rework requirements',
    'acc.judgeResultPass': 'Describe the check / measurement result',
    'acc.evidence': 'Evidence',
    'acc.evidencePh': 'e.g. factory test report No. / stress record ID',
    'acc.applyDialog': 'Apply Stress Result',
    'acc.applySummary': 'Summary',
    'acc.applyBackfill': 'Backfill to Item',
    'acc.createDialog': 'Create Acceptance Order',
    'acc.createTask': 'Deployment task',
    'acc.createTaskPh': 'Select a task with remaining devices',
    'acc.createGen': ' (created {a}/{b} {u})',
    'acc.inspectorPh': 'Optional — can be set later in the order',
    'acc.msg.itemSaved': 'Item verdict saved: ',
    'acc.msg.backfilled': 'Stress result backfilled; item auto-judged',
    'acc.msg.orderCreated': 'Order created',
    'acc.msg.conclusionSaved': 'Conclusion archived',
    'acc.msg.resultRequired': 'A record is required for Pass/Fail verdicts',

    'ins.title': 'Installers',
    'ins.desc': 'Manage installer qualifications and skill tags, and match them against the installer requirements of each node category; assignment of deployment tasks can rely on the match rate.',
    'ins.reqTitle': 'Installer Requirements (Job Spec)',
    'ins.gpuNodes': 'GPU Nodes (research fine-tuning / inference / teaching / agent / training clusters)',
    'ins.cpuNode': 'CPU Simulation Node',
    'ins.roster': 'Roster ({n} installers)',
    'ins.add': 'Add Installer',
    'ins.col.name': 'Name',
    'ins.col.title': 'Title',
    'ins.col.level': 'Level',
    'ins.col.phone': 'Phone',
    'ins.col.skills': 'Skills',
    'ins.col.available': 'Available',
    'ins.available': 'Available',
    'ins.busy': 'Busy',
    'ins.matrix': 'Skill Matrix · Matching',
    'ins.reqSkill': 'Required Skill',
    'ins.match': 'Match',
    'ins.missing': 'Missing Skills',
    'ins.allMet': 'All met',
    'ins.matchFor': 'Match for "{c}"',
    'ins.edit': 'Edit Installer',
    'ins.fName': 'Name',
    'ins.fTitle': 'Title',
    'ins.fTitlePh': 'e.g. GPU cluster deployment expert',
    'ins.fLevel': 'Level',
    'ins.fPhone': 'Phone',
    'ins.gpuSkills': 'GPU Skills',
    'ins.hpcSkills': 'HPC Skills',
    'ins.fAvailable': 'Available',
    'ins.msgSaved': 'Saved',
    'ins.msgDeleted': 'Deleted',
    'ins.msgNameRequired': 'Name is required',
    'ins.deleteConfirm': 'Delete {n}?',

    'dep.title': 'Deployment',
    'dep.desc': 'Track device deployment by node category: arrival check → rack install → network setup → OS & drivers → stress & burn-in → acceptance → in service; installer assignment refers to the match rate on the Installers page.',
    'dep.tasks': 'Deployment Tasks',
    'dep.new': 'New Task',
    'dep.col.no': 'Task ID',
    'dep.col.cat': 'Category',
    'dep.col.qty': 'Qty',
    'dep.col.stage': 'Stage',
    'dep.col.installers': 'Installers',
    'dep.col.orders': 'Orders',
    'dep.col.started': 'Started',
    'dep.unassigned': 'Unassigned',
    'dep.current': 'current',
    'dep.advance': 'Advance Stage',
    'dep.assign': 'Assign',
    'dep.assignDialog': 'Assign Installers',
    'dep.reqColon': 'Requirements: ',
    'dep.missing': 'Missing ',
    'dep.confirmAssign': 'Confirm Assignment',
    'dep.newDialog': 'New Deployment Task',
    'dep.fCat': 'Node category',
    'dep.fNote': 'Note',
    'dep.msgAdvanced': 'Advanced to: ',
    'dep.msgAssigned': 'Assignment updated',
    'dep.msgCreated': 'Task created',

    'http.title': 'Campus Network Concurrency Test',
    'http.desc': 'Real load engine: N concurrent workers simulate N students/staff hitting the target service, measuring RPS, P50/P90/P95/P99 latency, error rate and per-second series — for the acceptance item "campus network performance with N concurrent users".',
    'http.new': 'New Test',
    'http.namePh': 'e.g. Inference node 10-user acceptance test',
    'http.target': 'Target URL',
    'http.targetPh': 'http://127.0.0.1:8080/api/health (localhost / intranet only)',
    'http.concurrency': 'Concurrency',
    'http.duration': 'Duration (s)',
    'http.maxErr': 'Max Err%',
    'http.maxP95': 'P95 Limit (ms)',
    'http.linkOrder': 'Link Order',
    'http.linkPh': 'Optional — backfill the acceptance item when finished',
    'http.item': 'Item',
    'http.start': 'Start Test',
    'http.live': 'Live Monitor',
    'http.total': 'Total Requests',
    'http.rps': 'RPS',
    'http.avg': 'Avg Latency',
    'http.p95': 'P95',
    'http.errRate': 'Error Rate (limit {v}%)',
    'http.failed': 'Failed',
    'http.seriesRps': 'RPS',
    'http.seriesAvg': 'Avg latency (ms)',
    'http.seriesErr': 'Errors/s',
    'http.timeAxis': 'Time',
    'http.history': 'Test Records',
    'http.col.target': 'Target',
    'http.col.conc': 'Conc',
    'http.col.duration': 'Duration',
    'http.col.p95': 'P95(ms)',
    'http.col.err': 'Err%',
    'http.backfill': 'Backfill',
    'http.detailTitle': 'Test Detail',
    'http.dTotal': 'Total',
    'http.dSuccess': 'Success',
    'http.dFailed': 'Failed',
    'http.dAvg': 'Avg',
    'http.dMin': 'Min',
    'http.dMax': 'Max',
    'http.codes': 'Status codes: ',
    'http.msgTargetRequired': 'Target URL is required',
    'http.msgStarted': 'Test started: ',
    'http.msgPass': 'Test finished and PASSED: ',
    'http.msgFail': 'Test finished, FAILED (limits: err≤{e}%, P95≤{p}ms) — see record',
    'http.msgBackfilled': 'Backfilled to order ',
    'http.msgAutoBackfill': 'Acceptance item auto-judged from the test result',

    'burn.title': 'Burn-in Monitoring',
    'burn.desc': 'For the acceptance item "72h full-load burn-in: monitor temperature/power/stability and confirm 0 ECC errors". Verdict rule matches the real process: ECC errors = 0 and max temperature within limit → Pass.',
    'burn.demoTitle': 'Demo mode note',
    'burn.demoDesc': 'The host has no real GPUs; telemetry is simulated (temperature ramp / per-GPU power / utilization / ECC counter). To run against real GPUs, replace the sample source with DCGM-Exporter / nvidia-smi — the verdict logic stays unchanged.',
    'burn.new': 'New Burn-in Task',
    'burn.namePh': 'e.g. Research fine-tuning node 72h burn-in',
    'burn.device': 'Device',
    'burn.devicePh': 'Device label',
    'burn.gpus': 'GPUs',
    'burn.watt': 'Watt per GPU (W)',
    'burn.duration': 'Duration',
    'burn.demo2min': 'Demo 2min',
    'burn.maxTemp': 'Temp Limit (°C)',
    'burn.inject': 'Fault Injection',
    'burn.injectLabel': 'Inject ECC errors (demo)',
    'burn.start': 'Start Burn-in',
    'burn.live': 'Live Monitor',
    'burn.curMaxTemp': 'Max Temp (limit {v}°C)',
    'burn.curPower': 'Total Power ({n} GPUs)',
    'burn.ecc': 'ECC Errors (must be 0)',
    'burn.samples': 'Samples',
    'burn.verdictPass': 'PASS',
    'burn.verdictFail': 'FAIL',
    'burn.history': 'Burn-in Records',
    'burn.col.device': 'Device',
    'burn.col.maxTemp': 'Max Temp',
    'burn.col.duration': 'Duration',
    'burn.detailTitle': 'Burn-in Detail',
    'burn.totalPower': 'Total Power (W)',
    'burn.msgStarted': 'Burn-in started: ',
    'burn.msgBackfilled': 'Backfilled to order ',
    'burn.msgAutoBackfill': 'Acceptance item auto-judged from the burn-in result',

    'spec.title': 'Acceptance Spec · Installer Requirements',
    'spec.desc': 'The installer requirements and acceptance clauses of each node category — the single source of the order item templates and skill matching.',
    'spec.req': 'Installer Requirements',
    'spec.gpuCommon': 'GPU Nodes (Common Spec)',
    'spec.cpuTitle': 'CPU Simulation Node',
    'spec.tmpl': 'Device Acceptance Items (Order Template)',
    'spec.col.item': 'Item',
    'spec.col.content': 'Content',
    'spec.col.method': 'Method',
    'spec.applyTo': 'Applicable Node Categories',
    'spec.col.no': 'No.',
    'spec.col.qty': 'Qty',
    'spec.reqSkills': 'Required skills: ',
  },
}

export function t(key, params) {
  const d = dict[locale.value] || dict.zh
  let s = d[key] ?? dict.zh[key] ?? key
  if (params) {
    for (const [k, v] of Object.entries(params)) s = s.replaceAll(`{${k}}`, String(v))
  }
  return s
}

// ---- 领域展示翻译（存储仍为中文枚举/原文） ----

const catNames = {
  '科研微调节点（满血版）': 'Research Fine-tuning Node (Full)',
  '科研微调节点（标准版）': 'Research Fine-tuning Node (Standard)',
  '科研推理节点（满血版）': 'Research Inference Node (Full)',
  '科研推理节点（标准版）': 'Research Inference Node (Standard)',
  '教学推理节点（满血版）': 'Teaching Inference Node (Full)',
  '教学推理节点（高级版）': 'Teaching Inference Node (Advanced)',
  '教学推理节点（标准版）': 'Teaching Inference Node (Standard)',
  'CPU仿真节点': 'CPU Simulation Node',
  '智能体集群': 'Agent Cluster',
  '实训集群': 'Training Cluster',
}

const skills = {
  'NVIDIA专业GPU服务器部署': 'NVIDIA professional GPU server deployment',
  'vLLM/Triton推理服务部署': 'vLLM/Triton inference deployment',
  'DCGM-Exporter监控配置': 'DCGM-Exporter monitoring',
  'CUDA/TensorRT环境配置': 'CUDA/TensorRT environment',
  '集群调度搭建': 'Cluster scheduling setup',
  'MIG多实例配置': 'MIG multi-instance config',
  'ECC日志读取': 'ECC log reading',
  '高性能计算集群部署': 'HPC cluster deployment',
  'oneAPI/MKL数学库配置': 'oneAPI/MKL math library',
  'CAE软件许可管理': 'CAE license management',
  '内存全通道均衡安装': 'Balanced memory channel population',
  'SPEC基准测试': 'SPEC benchmarking',
}

// 状态枚举：wire 值（中文，API/存储）↔ 展示
export const ORDER_STATUS = ['待验收', '进行中', '已通过', '整改中']
export const ITEM_STATUS_KEYS = [
  { key: 'pass', zh: '合格' },
  { key: 'fail', zh: '不合格' },
  { key: 'running', zh: '进行中' },
  { key: 'na', zh: '不适用' },
]
export const STATUS_LABELS = {
  待验收: { zh: '待验收', en: 'Pending' },
  进行中: { zh: '进行中', en: 'In Progress' },
  已通过: { zh: '已通过', en: 'Passed' },
  整改中: { zh: '整改中', en: 'Rework' },
  待检: { zh: '待检', en: 'Pending' },
  合格: { zh: '合格', en: 'Pass' },
  不合格: { zh: '不合格', en: 'Fail' },
  不适用: { zh: '不适用', en: 'N/A' },
  待启动: { zh: '待启动', en: 'Not Started' },
  已完成: { zh: '已完成', en: 'Done' },
  running: { zh: '运行中', en: 'Running' },
  finished: { zh: '已完成', en: 'Finished' },
}
export const LEVELS = ['专家级', '高级', '中级', '初级']
const levelNames = { 专家级: 'Expert', 高级: 'Senior', 中级: 'Mid-level', 初级: 'Junior' }

const stageNames = {
  zh: ['待启动', '到货清点', '上架安装', '组网接入', '系统与驱动', '压测烤机', '设备验收', '交付使用'],
  en: ['Not Started', 'Arrival Check', 'Rack Install', 'Network Setup', 'OS & Drivers', 'Stress & Burn-in', 'Acceptance', 'In Service'],
}

const unitNames = { 台: { zh: '台', en: 'unit' }, 组: { zh: '组', en: 'group' } }
const catMethodNames = { manual: { zh: '人工检查', en: 'Manual' }, burnin: { zh: '满载烤机', en: 'Burn-in' }, http: { zh: '并发压测', en: 'HTTP Test' }, performance: { zh: '性能实测', en: 'Performance' } }

// 安装人员要求原文（英文展示译文）
export const REQ_TEXT = {
  gpu: {
    zh: '具备NVIDIA专业GPU服务器部署经验，熟悉GPU推理服务部署（vLLM/Triton）与DCGM-Exporter监控配置，熟悉CUDA/TensorRT环境配置与集群调度搭建；掌握MIG多实例配置与ECC日志读取。',
    en: 'Experienced in NVIDIA professional GPU server deployment; familiar with GPU inference service deployment (vLLM/Triton) and DCGM-Exporter monitoring configuration; familiar with CUDA/TensorRT environment setup and cluster scheduler building; proficient in MIG multi-instance configuration and ECC log reading.',
  },
  cpu: {
    zh: '熟悉高性能计算集群部署，掌握oneAPI/MKL数学库配置与CAE软件许可管理；具备内存全通道均衡安装（32×32GB）与SPEC基准测试能力。',
    en: 'Familiar with HPC cluster deployment; proficient in oneAPI/MKL math library configuration and CAE software license management; capable of balanced memory channel population (32×32GB) and SPEC benchmarking.',
  },
}

// 验收条目模板（按节点类别 kind + 条目 key），title/content 提供英文展示译文
export const ITEM_TEXT = {
  gpu: {
    unpack: { title: { zh: '① 开箱核对', en: '① Unboxing Check' }, content: { zh: '显卡型号/序列号与合同逐项比对，外观无运输损伤', en: 'Compare GPU model / serial numbers against the contract item by item; no shipping damage on appearance' } },
    config: { title: { zh: '② 配置核对', en: '② Configuration Check' }, content: { zh: 'CPU核数、内存容量、SSD容量与配置单逐项比对', en: 'Compare CPU cores, memory capacity and SSD capacity against the configuration list item by item' } },
    burnin: { title: { zh: '③ 72h满载烤机', en: '③ 72h Full-load Burn-in' }, content: { zh: '运行GPU压测监测温度/功耗/稳定性，ECC事件日志确认0错误', en: 'Run GPU stress tests, monitor temperature/power/stability, and confirm 0 errors in ECC event logs' } },
    perf: { title: { zh: '④ 性能实测', en: '④ Performance Test' }, content: { zh: 'GPU P2P带宽（强制验收项）、显存带宽、FP32/FP8算力与标称值比对（偏差≤5%）', en: 'GPU P2P bandwidth (mandatory), memory bandwidth, FP32/FP8 compute vs. nominal values (deviation ≤5%)' } },
    conc: { title: { zh: '⑤ 校内网并发压测', en: '⑤ Campus Network Concurrency Test' }, content: { zh: '校内网性能与10人并发压测：模拟师生访问，验证响应时间与稳定性', en: 'Campus network performance with 10 concurrent users: simulate student/staff access and verify response time and stability' } },
    software: { title: { zh: '⑥ 软件验收', en: '⑥ Software Acceptance' }, content: { zh: 'CUDA驱动、PyTorch/TensorFlow框架安装运行验证；课程镜像部署、师生账号开通验证', en: 'Verify CUDA driver and PyTorch/TensorFlow installation; course image deployment and student/staff account provisioning' } },
    docs: { title: { zh: '⑦ 文档归档', en: '⑦ Documentation' }, content: { zh: '出厂测试报告、序列号注册、质保凭证归档', en: 'Archive factory test reports, serial number registration and warranty certificates' } },
  },
  cpu: {
    config: { title: { zh: '① 配置核对', en: '① Configuration Check' }, content: { zh: 'CPU核数/线程数、内存通道数与配置单比对', en: 'Compare CPU cores/threads and memory channel count against the configuration list' } },
    spec: { title: { zh: '② SPEC CPU 2017基准测试', en: '② SPEC CPU 2017 Benchmark' }, content: { zh: '整数/浮点得分与行业基准比对', en: 'Compare integer/floating-point scores against industry baselines' } },
    case: { title: { zh: '③ 真实工程算例计时', en: '③ Real Workload Timing' }, content: { zh: '使用Fluent/Abaqus标准算例验收实际求解性能', en: 'Validate actual solver performance with Fluent/Abaqus standard cases' } },
    burnin: { title: { zh: '④ 稳定性测试与并发压测', en: '④ Stability & Concurrency Test' }, content: { zh: '长时间满载运行监测；校内网性能与3人并发压测：模拟师生访问，验证响应时间与稳定性', en: 'Long-duration full-load monitoring; campus network performance with 3 concurrent users: simulate student/staff access and verify response time and stability' } },
    docs: { title: { zh: '⑤ 文档归档', en: '⑤ Documentation' }, content: { zh: '基准测试报告归档', en: 'Archive benchmark reports' } },
  },
}

// ---- 展示辅助 ----
export function statusLabel(zh) {
  const m = STATUS_LABELS[zh]
  return m ? m[locale.value] || m.zh : zh || '-'
}
export function itemStatusKeyOf(zh) {
  const f = ITEM_STATUS_KEYS.find((x) => x.zh === zh)
  return f ? f.key : 'pass'
}
export function itemStatusZhOf(key) {
  const f = ITEM_STATUS_KEYS.find((x) => x.key === key)
  return f ? f.zh : '合格'
}
export function stageName(i) {
  const arr = stageNames[locale.value] || stageNames.zh
  return arr[i] ?? i
}
export const STAGE_COUNT = stageNames.zh.length
export function levelName(l) {
  return locale.value === 'en' ? levelNames[l] || l : l
}
export function unitName(u) {
  const m = unitNames[u]
  return m ? m[locale.value] || m.zh : u
}
export function catNameOf(cat) {
  if (locale.value === 'en') return catNames[cat.name] || cat.name
  return cat.name
}
export function skillName(s) {
  return locale.value === 'en' ? skills[s] || s : s
}
export function methodName(c) {
  const m = catMethodNames[c]
  return m ? m[locale.value] || m.zh : c
}
export function reqText(kind) {
  const m = REQ_TEXT[kind]
  return m ? m[locale.value] || m.zh : ''
}
export function itemTitle(kind, key, fallback) {
  const m = ITEM_TEXT[kind]?.[key]
  return (m && (m.title[locale.value] || m.title.zh)) || fallback
}
export function itemContent(kind, key, fallback) {
  const m = ITEM_TEXT[kind]?.[key]
  return (m && (m.content[locale.value] || m.content.zh)) || fallback
}
// 将设备标签中的中文类别名替换为当前语言（如“教学推理节点（满血版） #2”）
export function localizeDeviceLabel(label, categories) {
  if (!label) return ''
  let out = label
  for (const c of categories || []) {
    if (out.includes(c.name)) out = out.replaceAll(c.name, catNameOf(c))
  }
  return out
}
export function fmtDur(sec) {
  if (sec == null) return '-'
  const en = locale.value === 'en'
  if (sec < 60) return en ? `${sec}s` : `${sec}秒`
  if (sec < 3600) {
    const m = Math.floor(sec / 60), r = sec % 60
    return en ? `${m}m${r ? r + 's' : ''}` : `${m}分${r ? r + '秒' : ''}`
  }
  const h = Math.floor(sec / 3600), m = Math.floor((sec % 3600) / 60)
  return en ? `${h}h${m ? m + 'm' : ''}` : `${h}时${m ? m + '分' : ''}`
}
