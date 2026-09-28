<template>
  <div>
    <div class="page-title">{{ t('http.title') }}</div>
    <div class="page-desc">{{ t('http.desc') }}</div>

    <div class="card-block">
      <div class="block-title">{{ t('http.new') }}</div>
      <el-form :inline="true" label-width="130px">
        <el-form-item :label="t('common.name')">
          <el-input v-model="form.name" style="width: 260px" :placeholder="t('http.namePh')" />
        </el-form-item>
        <el-form-item :label="t('http.target')">
          <el-input v-model="form.target" style="width: 300px" :placeholder="t('http.targetPh')" />
        </el-form-item>
        <el-form-item :label="t('http.concurrency')">
          <el-input-number v-model="form.concurrency" :min="1" :max="2000" />
        </el-form-item>
        <el-form-item :label="t('http.duration')">
          <el-input-number v-model="form.durationSec" :min="1" :max="86400" />
        </el-form-item>
        <el-form-item :label="t('http.maxErr')">
          <el-input-number v-model="form.maxErrRatePct" :min="0.1" :step="0.5" />
        </el-form-item>
        <el-form-item :label="t('http.maxP95')">
          <el-input-number v-model="form.maxP95Ms" :min="10" :step="50" />
        </el-form-item>
        <el-form-item :label="t('http.linkOrder')">
          <el-select v-model="form.acceptanceId" clearable style="width: 300px" :placeholder="t('http.linkPh')">
            <el-option v-for="o in orders" :key="o.id" :label="`${o.id} · ${localizeDeviceLabel(o.deviceLabel, categories)}`" :value="o.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.acceptanceId" :label="t('http.item')">
          <el-select v-model="form.itemKey" style="width: 300px">
            <el-option v-for="it in linkableItems" :key="it.key" :label="itemTitleFor(it)" :value="it.key" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="runningTask !== null" @click="start">
            <el-icon><VideoPlay /></el-icon>&nbsp;{{ t('http.start') }}
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 实时面板 -->
    <div v-if="live" class="card-block">
      <div class="block-title">
        {{ t('http.live') }} · {{ live.id }}
        <el-tag style="margin-left: 8px" type="warning" effect="dark">
          {{ t('common.running') }} {{ live.result.durationSec }}s / {{ live.durationSec }}s
        </el-tag>
      </div>
      <el-row :gutter="12" style="margin-bottom: 12px">
        <el-col :span="4"><div class="stat"><div class="stat-v">{{ live.result.total ?? 0 }}</div><div class="stat-l">{{ t('http.total') }}</div></div></el-col>
        <el-col :span="4"><div class="stat"><div class="stat-v">{{ live.result.rps ?? 0 }}</div><div class="stat-l">{{ t('http.rps') }}</div></div></el-col>
        <el-col :span="4"><div class="stat"><div class="stat-v">{{ live.result.avgMs ?? 0 }}ms</div><div class="stat-l">{{ t('http.avg') }}</div></div></el-col>
        <el-col :span="4"><div class="stat"><div class="stat-v">{{ live.result.p95 ?? '-' }}ms</div><div class="stat-l">{{ t('http.p95') }}</div></div></el-col>
        <el-col :span="4"><div class="stat"><div class="stat-v" :style="{ color: errColor }">{{ live.result.errRatePct ?? 0 }}%</div><div class="stat-l">{{ t('http.errRate', { v: live.maxErrRatePct }) }}</div></div></el-col>
        <el-col :span="4"><div class="stat"><div class="stat-v">{{ live.result.failed ?? 0 }}</div><div class="stat-l">{{ t('http.failed') }}</div></div></el-col>
      </el-row>
      <EChart :option="seriesOption" height="280px" />
    </div>

    <!-- 历史 -->
    <div class="card-block">
      <div class="block-title">{{ t('http.history') }}</div>
      <el-table :data="history" border size="small">
        <el-table-column prop="id" :label="t('common.id')" width="90" />
        <el-table-column prop="name" :label="t('common.task')" min-width="200" show-overflow-tooltip />
        <el-table-column prop="target" :label="t('http.col.target')" min-width="200" show-overflow-tooltip />
        <el-table-column :label="t('http.col.conc')" width="70" align="center">
          <template #default="{ row }">{{ row.concurrency }}</template>
        </el-table-column>
        <el-table-column :label="t('http.col.duration')" width="100" align="center">
          <template #default="{ row }">{{ fmtDur(row.result?.durationSec ?? row.durationSec) }}</template>
        </el-table-column>
        <el-table-column :label="t('http.rps')" width="90" align="center">
          <template #default="{ row }">{{ row.result?.rps ?? '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('http.col.p95')" width="95" align="center">
          <template #default="{ row }">{{ row.result?.p95 ?? '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('http.col.err')" width="90" align="center">
          <template #default="{ row }">{{ row.result ? row.result.errRatePct + '%' : '-' }}</template>
        </el-table-column>
        <el-table-column :label="t('common.verdict')" width="100" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.status === 'running'" type="warning">{{ t('common.running') }}</el-tag>
            <el-tag v-else :type="row.result?.pass ? 'success' : 'danger'" effect="dark">
              {{ row.result?.pass ? t('common.pass') : t('common.fail') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('common.time')" width="160">
          <template #default="{ row }">{{ fmtTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="showDetail(row)">{{ t('common.detail') }}</el-button>
            <el-button v-if="row.acceptanceId && row.result" link type="success" @click="backfill(row)">{{ t('http.backfill') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 详情 -->
    <el-dialog v-model="detailVisible" :title="t('http.detailTitle') + ' · ' + (detail?.id || '')" width="860px">
      <template v-if="detail?.result">
        <el-descriptions :column="4" border size="small" style="margin-bottom: 12px">
          <el-descriptions-item :label="t('http.dTotal')">{{ detail.result.total }}</el-descriptions-item>
          <el-descriptions-item :label="t('http.dSuccess')">{{ detail.result.success }}</el-descriptions-item>
          <el-descriptions-item :label="t('http.dFailed')">{{ detail.result.failed }}</el-descriptions-item>
          <el-descriptions-item :label="t('http.rps')">{{ detail.result.rps }}</el-descriptions-item>
          <el-descriptions-item :label="t('http.dAvg')">{{ detail.result.avgMs }}ms</el-descriptions-item>
          <el-descriptions-item :label="t('http.dMin')">{{ detail.result.minMs }}ms</el-descriptions-item>
          <el-descriptions-item :label="t('http.dMax')">{{ detail.result.maxMs }}ms</el-descriptions-item>
          <el-descriptions-item :label="t('http.col.err')">{{ detail.result.errRatePct }}%</el-descriptions-item>
          <el-descriptions-item label="P50">{{ detail.result.p50 }}ms</el-descriptions-item>
          <el-descriptions-item label="P90">{{ detail.result.p90 }}ms</el-descriptions-item>
          <el-descriptions-item label="P95">{{ detail.result.p95 }}ms</el-descriptions-item>
          <el-descriptions-item label="P99">{{ detail.result.p99 }}ms</el-descriptions-item>
        </el-descriptions>
        <EChart :option="detailSeriesOption" height="300px" />
        <div style="margin-top: 10px">
          <span style="color: #8492a6; font-size: 13px">{{ t('http.codes') }}</span>
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
import { t, itemTitle, localizeDeviceLabel } from '../i18n'
import EChart from '../components/EChart.vue'

const route = useRoute()
const form = reactive({
  name: '', target: 'http://127.0.0.1:8080/api/health', concurrency: 10, durationSec: 60,
  maxErrRatePct: 1, maxP95Ms: 500, acceptanceId: '', itemKey: '',
})
const history = ref([])
const orders = ref([])
const categories = ref([])
const live = ref(null)
const runningTask = ref(null)
const detailVisible = ref(false)
const detail = ref(null)
let timer = null

const itemTitleFor = (it) => itemTitle('gpu', it.key, it.title)

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
    legend: { data: [t('http.seriesRps'), t('http.seriesAvg'), t('http.seriesErr')] },
    grid: { left: 50, right: 55, top: 40, bottom: 30 },
    xAxis: { type: 'category', data: pts.map((p) => p.t + 's'), name: t('http.timeAxis') },
    yAxis: [
      { type: 'value', name: 'RPS' },
      { type: 'value', name: 'ms', position: 'right' },
    ],
    series: [
      { name: t('http.seriesRps'), type: 'line', smooth: true, showSymbol: false, areaStyle: { opacity: 0.15 }, data: pts.map((p) => p.count) },
      { name: t('http.seriesAvg'), type: 'line', smooth: true, showSymbol: false, yAxisIndex: 1, data: pts.map((p) => Math.round(p.avgMs * 10) / 10) },
      { name: t('http.seriesErr'), type: 'line', showSymbol: false, data: pts.map((p) => p.errs), itemStyle: { color: '#f56c6c' } },
    ],
  }
}

async function loadHistory() {
  history.value = (await get('/stress/http')) || []
}

async function start() {
  if (!form.target) {
    ElMessage.warning(t('http.msgTargetRequired'))
    return
  }
  const tk = await post('/stress/http', { ...form })
  runningTask.value = tk
  live.value = tk
  ElMessage.success(`${t('http.msgStarted')}${tk.id}`)
  startPolling(tk.id)
  loadHistory()
}

function startPolling(id) {
  stopPolling()
  timer = setInterval(async () => {
    const tk = await get(`/stress/http/${id}/live`)
    live.value = tk
    if (tk.status !== 'running') {
      stopPolling()
      runningTask.value = null
      loadHistory()
      ElMessage({
        type: tk.result?.pass ? 'success' : 'error',
        message: tk.result?.pass
          ? `${t('http.msgPass')}RPS ${tk.result.rps}, P95 ${tk.result.p95}ms, Err ${tk.result.errRatePct}%`
          : t('http.msgFail', { e: tk.maxErrRatePct, p: tk.maxP95Ms }),
        duration: 6000,
      })
      if (tk.acceptanceId && tk.itemKey) backfillSilent(tk)
    }
  }, 1000)
}

function stopPolling() {
  if (timer) clearInterval(timer)
  timer = null
}

async function backfill(tk) {
  const o = await post(`/orders/${tk.acceptanceId}/items/${tk.itemKey}/apply-stress`, { stressId: tk.id, type: 'http' })
  ElMessage.success(`${t('http.msgBackfilled')}${o.id}`)
}

async function backfillSilent(tk) {
  try {
    await post(`/orders/${tk.acceptanceId}/items/${tk.itemKey}/apply-stress`, { stressId: tk.id, type: 'http' })
    ElMessage.info(t('http.msgAutoBackfill'))
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
    form.name = `${route.query.device || ''} 10-user acceptance test`
  }
  const [os, cs] = await Promise.all([get('/orders'), get('/categories')])
  orders.value = os || []
  categories.value = cs || []
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
