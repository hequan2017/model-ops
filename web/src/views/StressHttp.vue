<template>
  <div>
    <div class="page-title">校内网并发压测</div>
    <div class="page-desc">
      真实并发压测引擎：N 个并发模拟 N 名师生同时访问目标服务，统计 RPS、P50/P90/P95/P99 延迟、错误率与逐秒时序，
      对应验收条目「校内网性能与N人并发压测：模拟师生访问，验证响应时间与稳定性」（默认 10 人并发）。
    </div>

    <div class="card-block">
      <div class="block-title">发起压测</div>
      <el-form :inline="true" label-width="110px">
        <el-form-item label="任务名称">
          <el-input v-model="form.name" style="width: 260px" placeholder="如：推理节点 10人并发验收压测" />
        </el-form-item>
        <el-form-item label="目标地址">
          <el-input v-model="form.target" style="width: 300px" placeholder="http://127.0.0.1:8080/api/health（仅限本机/内网）" />
        </el-form-item>
        <el-form-item label="并发数">
          <el-input-number v-model="form.concurrency" :min="1" :max="2000" />
        </el-form-item>
        <el-form-item label="时长（秒）">
          <el-input-number v-model="form.durationSec" :min="1" :max="86400" />
        </el-form-item>
        <el-form-item label="错误率阈值%">
          <el-input-number v-model="form.maxErrRatePct" :min="0.1" :step="0.5" />
        </el-form-item>
        <el-form-item label="P95阈值(ms)">
          <el-input-number v-model="form.maxP95Ms" :min="10" :step="50" />
        </el-form-item>
        <el-form-item label="关联验收单">
          <el-select v-model="form.acceptanceId" clearable style="width: 300px" placeholder="可选：完成后回填验收条目">
            <el-option v-for="o in orders" :key="o.id" :label="`${o.id} · ${o.deviceLabel}`" :value="o.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.acceptanceId" label="验收条目">
          <el-select v-model="form.itemKey" style="width: 300px">
            <el-option v-for="it in linkableItems" :key="it.key" :label="it.title" :value="it.key" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="runningTask !== null" @click="start">
            <el-icon><VideoPlay /></el-icon>&nbsp;开始压测
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 实时面板 -->
    <div v-if="live" class="card-block">
      <div class="block-title">
        实时监控 · {{ live.id }}
        <el-tag style="margin-left: 8px" type="warning" effect="dark">运行中 {{ live.result.durationSec }}s / {{ live.durationSec }}s</el-tag>
      </div>
      <el-row :gutter="12" style="margin-bottom: 12px">
        <el-col :span="4"><div class="stat"><div class="stat-v">{{ live.result.total ?? 0 }}</div><div class="stat-l">总请求数</div></div></el-col>
        <el-col :span="4"><div class="stat"><div class="stat-v">{{ live.result.rps ?? 0 }}</div><div class="stat-l">RPS</div></div></el-col>
        <el-col :span="4"><div class="stat"><div class="stat-v">{{ live.result.avgMs ?? 0 }}ms</div><div class="stat-l">平均延迟</div></div></el-col>
        <el-col :span="4"><div class="stat"><div class="stat-v">{{ live.result.p95 ?? '-' }}ms</div><div class="stat-l">P95</div></div></el-col>
        <el-col :span="4"><div class="stat"><div class="stat-v" :style="{ color: errColor }">{{ live.result.errRatePct ?? 0 }}%</div><div class="stat-l">错误率（阈值 {{ live.maxErrRatePct }}%）</div></div></el-col>
        <el-col :span="4"><div class="stat"><div class="stat-v">{{ live.result.failed ?? 0 }}</div><div class="stat-l">失败请求数</div></div></el-col>
      </el-row>
      <EChart :option="seriesOption" height="280px" />
    </div>

    <!-- 历史 -->
    <div class="card-block">
      <div class="block-title">压测记录</div>
      <el-table :data="history" border size="small">
        <el-table-column prop="id" label="编号" width="90" />
        <el-table-column prop="name" label="任务" min-width="200" show-overflow-tooltip />
        <el-table-column prop="target" label="目标" min-width="200" show-overflow-tooltip />
        <el-table-column label="并发" width="70" align="center">
          <template #default="{ row }">{{ row.concurrency }}</template>
        </el-table-column>
        <el-table-column label="时长" width="90" align="center">
          <template #default="{ row }">{{ fmtDur(row.result?.durationSec ?? row.durationSec) }}</template>
        </el-table-column>
        <el-table-column label="RPS" width="90" align="center">
          <template #default="{ row }">{{ row.result?.rps ?? '-' }}</template>
        </el-table-column>
        <el-table-column label="P95(ms)" width="90" align="center">
          <template #default="{ row }">{{ row.result?.p95 ?? '-' }}</template>
        </el-table-column>
        <el-table-column label="错误率" width="90" align="center">
          <template #default="{ row }">{{ row.result ? row.result.errRatePct + '%' : '-' }}</template>
        </el-table-column>
        <el-table-column label="判定" width="90" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.status === 'running'" type="warning">运行中</el-tag>
            <el-tag v-else :type="row.result?.pass ? 'success' : 'danger'" effect="dark">
              {{ row.result?.pass ? '合格' : '不合格' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="时间" width="160">
          <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="showDetail(row)">详情</el-button>
            <el-button v-if="row.acceptanceId && row.result" link type="success" @click="backfill(row)">回填验收</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 详情 -->
    <el-dialog v-model="detailVisible" :title="'压测详情 · ' + (detail?.id || '')" width="860px">
      <template v-if="detail?.result">
        <el-descriptions :column="4" border size="small" style="margin-bottom: 12px">
          <el-descriptions-item label="总请求">{{ detail.result.total }}</el-descriptions-item>
          <el-descriptions-item label="成功">{{ detail.result.success }}</el-descriptions-item>
          <el-descriptions-item label="失败">{{ detail.result.failed }}</el-descriptions-item>
          <el-descriptions-item label="RPS">{{ detail.result.rps }}</el-descriptions-item>
          <el-descriptions-item label="平均">{{ detail.result.avgMs }}ms</el-descriptions-item>
          <el-descriptions-item label="最小">{{ detail.result.minMs }}ms</el-descriptions-item>
          <el-descriptions-item label="最大">{{ detail.result.maxMs }}ms</el-descriptions-item>
          <el-descriptions-item label="错误率">{{ detail.result.errRatePct }}%</el-descriptions-item>
          <el-descriptions-item label="P50">{{ detail.result.p50 }}ms</el-descriptions-item>
          <el-descriptions-item label="P90">{{ detail.result.p90 }}ms</el-descriptions-item>
          <el-descriptions-item label="P95">{{ detail.result.p95 }}ms</el-descriptions-item>
          <el-descriptions-item label="P99">{{ detail.result.p99 }}ms</el-descriptions-item>
        </el-descriptions>
        <EChart :option="detailSeriesOption" height="300px" />
        <div style="margin-top: 10px">
          <span style="color: #8492a6; font-size: 13px">状态码分布：</span>
          <el-tag v-for="(v, k) in detail.result.statusCodes" :key="k" size="small" style="margin: 2px"
                  :type="k === '200' || k === 'OK' ? 'success' : 'danger'">{{ k }} × {{ v }}</el-tag>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { get, post } from '../api'
import { fmtTime, fmtDur } from '../api'
import EChart from '../components/EChart.vue'

const route = useRoute()
const form = reactive({
  name: '', target: 'http://127.0.0.1:8080/api/health', concurrency: 10, durationSec: 60,
  maxErrRatePct: 1, maxP95Ms: 500, acceptanceId: '', itemKey: '',
})
const history = ref([])
const orders = ref([])
const live = ref(null)
const runningTask = ref(null)
const detailVisible = ref(false)
const detail = ref(null)
let timer = null

const linkableItems = computed(() => {
  const o = orders.value.find((x) => x.id === form.acceptanceId)
  if (!o) return []
  return o.items.filter((i) => i.category === 'http' || i.category === 'burnin')
})

const errColor = computed(() =>
  (live.value?.result?.errRatePct ?? 0) > (live.value?.maxErrRatePct ?? 1) ? '#f56c6c' : '#1f2d3d',
)

const seriesOption = computed(() => buildSeries(live.value?.result?.series))
const detailSeriesOption = computed(() => buildSeries(detail.value?.result?.series))

function buildSeries(series) {
  const pts = series || []
  return {
    tooltip: { trigger: 'axis' },
    legend: { data: ['每秒请求数', '平均延迟(ms)', '每秒错误'] },
    grid: { left: 50, right: 55, top: 40, bottom: 30 },
    xAxis: { type: 'category', data: pts.map((p) => p.t + 's'), name: '时间' },
    yAxis: [
      { type: 'value', name: 'RPS' },
      { type: 'value', name: 'ms', position: 'right' },
    ],
    series: [
      { name: '每秒请求数', type: 'line', smooth: true, showSymbol: false, areaStyle: { opacity: 0.15 }, data: pts.map((p) => p.count) },
      { name: '平均延迟(ms)', type: 'line', smooth: true, showSymbol: false, yAxisIndex: 1, data: pts.map((p) => Math.round(p.avgMs * 10) / 10) },
      { name: '每秒错误', type: 'line', showSymbol: false, data: pts.map((p) => p.errs), itemStyle: { color: '#f56c6c' } },
    ],
  }
}

async function loadHistory() {
  history.value = (await get('/stress/http')) || []
}

async function start() {
  if (!form.target) {
    ElMessage.warning('请填写目标地址')
    return
  }
  const t = await post('/stress/http', { ...form })
  runningTask.value = t
  live.value = t
  ElMessage.success(`压测已启动：${t.id}（并发 ${form.concurrency}，${form.durationSec}s）`)
  startPolling(t.id)
  loadHistory()
}

function startPolling(id) {
  stopPolling()
  timer = setInterval(async () => {
    const t = await get(`/stress/http/${id}/live`)
    live.value = t
    if (t.status !== 'running') {
      stopPolling()
      runningTask.value = null
      loadHistory()
      ElMessage({
        type: t.result?.pass ? 'success' : 'error',
        message: t.result?.pass
          ? `压测完成并判定合格：RPS ${t.result.rps}，P95 ${t.result.p95}ms，错误率 ${t.result.errRatePct}%`
          : `压测完成，判定不合格（阈值：错误率≤${t.maxErrRatePct}%、P95≤${t.maxP95Ms}ms），详见记录`,
        duration: 6000,
      })
      if (t.acceptanceId && t.itemKey) backfillSilent(t)
    }
  }, 1000)
}

function stopPolling() {
  if (timer) clearInterval(timer)
  timer = null
}

async function backfill(t) {
  const o = await post(`/orders/${t.acceptanceId}/items/${t.itemKey}/apply-stress`, { stressId: t.id, type: 'http' })
  ElMessage.success(`已回填验收单 ${o.id}，条目判定：${o.status}`)
}

async function backfillSilent(t) {
  try {
    const o = await post(`/orders/${t.acceptanceId}/items/${t.itemKey}/apply-stress`, { stressId: t.id, type: 'http' })
    ElMessage.info(`验收单 ${o.id} 条目已按压测结果自动判定`)
  } catch { /* 未关联或条目不匹配时忽略 */ }
}

function showDetail(row) {
  detail.value = row
  detailVisible.value = true
}

onMounted(async () => {
  // 从验收页跳转预填
  if (route.query.acceptanceId) {
    form.acceptanceId = route.query.acceptanceId
    form.itemKey = route.query.itemKey || ''
    form.name = `${route.query.device || '设备'} 10人并发验收压测`
  }
  orders.value = (await get('/orders')) || []
  await loadHistory()
  const run = history.value.find((h) => h.status === 'running')
  if (run) {
    runningTask.value = run
    live.value = run
    startPolling(run.id)
  }
})

onBeforeUnmount(stopPolling)
</script>

<style scoped>
.stat { background: #f5f7fa; border-radius: 6px; padding: 10px; text-align: center; }
.stat-v { font-size: 20px; font-weight: 700; color: #1f2d3d; }
.stat-l { font-size: 12px; color: #8492a6; margin-top: 4px; }
</style>
