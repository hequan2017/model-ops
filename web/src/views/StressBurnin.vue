<template>
  <div>
    <div class="page-title">{{ t('burn.title') }}</div>
    <div class="page-desc">{{ t('burn.desc') }}</div>

    <el-alert type="warning" :closable="false" style="margin-bottom: 16px"
              :title="t('burn.demoTitle')" :description="t('burn.demoDesc')" />

    <div class="card-block">
      <div class="block-title">{{ t('burn.new') }}</div>
      <el-form :inline="true" label-width="140px">
        <el-form-item :label="t('common.name')">
          <el-input v-model="form.name" style="width: 240px" :placeholder="t('burn.namePh')" />
        </el-form-item>
        <el-form-item :label="t('burn.device')">
          <el-input v-model="form.deviceLabel" style="width: 220px" :placeholder="t('burn.devicePh')" />
        </el-form-item>
        <el-form-item :label="t('burn.gpus')">
          <el-input-number v-model="form.gpus" :min="1" :max="16" />
        </el-form-item>
        <el-form-item :label="t('burn.watt')">
          <el-input-number v-model="form.wattPerGpu" :min="50" :max="800" :step="50" />
        </el-form-item>
        <el-form-item :label="t('burn.duration')">
          <el-input-number v-model="form.durationSec" :min="5" :max="259200" :step="60" />
          <el-button size="small" style="margin-left: 8px" @click="form.durationSec = 259200">72h</el-button>
          <el-button size="small" @click="form.durationSec = 120">{{ t('burn.demo2min') }}</el-button>
        </el-form-item>
        <el-form-item :label="t('burn.maxTemp')">
          <el-input-number v-model="form.maxTempC" :min="60" :max="95" />
        </el-form-item>
        <el-form-item :label="t('http.linkOrder')">
          <el-select v-model="form.acceptanceId" clearable style="width: 280px" :placeholder="t('http.linkPh')">
            <el-option v-for="o in orders" :key="o.id" :label="`${o.id} · ${localizeDeviceLabel(o.deviceLabel, categories)}`" :value="o.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.acceptanceId" :label="t('http.item')">
          <el-select v-model="form.itemKey" style="width: 280px">
            <el-option v-for="it in linkableItems" :key="it.key" :label="itemTitleFor(it)" :value="it.key" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('burn.inject')">
          <el-switch v-model="form.injectFault" :active-text="t('burn.injectLabel')" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="runningTask !== null" @click="start">
            <el-icon><VideoPlay /></el-icon>&nbsp;{{ t('burn.start') }}
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 实时监控 -->
    <div v-if="live" class="card-block">
      <div class="block-title">
        {{ t('burn.live') }} · {{ live.id }}（{{ localizeDeviceLabel(live.deviceLabel, categories) }}）
        <el-tag v-if="live.status === 'running'" style="margin-left: 8px" type="warning" effect="dark">
          {{ t('common.running') }} {{ elapsed }}s / {{ live.durationSec }}s
        </el-tag>
        <el-tag v-else style="margin-left: 8px" :type="live.result?.pass ? 'success' : 'danger'" effect="dark">
          {{ live.result?.pass ? t('burn.verdictPass') : t('burn.verdictFail') }}
        </el-tag>
      </div>
      <el-row :gutter="12" style="margin-bottom: 12px">
        <el-col :span="6"><div class="stat"><div class="stat-v">{{ curMaxTemp }}°C</div><div class="stat-l">{{ t('burn.curMaxTemp', { v: live.maxTempC }) }}</div></div></el-col>
        <el-col :span="6"><div class="stat"><div class="stat-v">{{ curPower }}W</div><div class="stat-l">{{ t('burn.curPower', { n: live.gpus }) }}</div></div></el-col>
        <el-col :span="6"><div class="stat"><div class="stat-v" :style="{ color: eccColor }">{{ curEcc }}</div><div class="stat-l">{{ t('burn.ecc') }}</div></div></el-col>
        <el-col :span="6"><div class="stat"><div class="stat-v">{{ samples.length }}</div><div class="stat-l">{{ t('burn.samples') }}</div></div></el-col>
      </el-row>
      <EChart :option="tempOption" height="300px" />
      <EChart :option="powerOption" height="240px" style="margin-top: 8px" />
    </div>

    <!-- 历史 -->
    <div class="card-block">
      <div class="block-title">{{ t('burn.history') }}</div>
      <el-table :data="history" border size="small">
        <el-table-column prop="id" :label="t('common.id')" width="90" />
        <el-table-column prop="name" :label="t('common.task')" min-width="190" show-overflow-tooltip />
        <el-table-column :label="t('burn.col.device')" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">{{ localizeDeviceLabel(row.deviceLabel, categories) }}</template>
        </el-table-column>
        <el-table-column :label="t('burn.gpus')" width="70" align="center">
          <template #default="{ row }">{{ row.gpus }}</template>
        </el-table-column>
        <el-table-column :label="t('burn.col.duration')" width="110" align="center">
          <template #default="{ row }">{{ fmtDur(row.durationSec) }}</template>
        </el-table-column>
        <el-table-column :label="t('burn.col.maxTemp')" width="100" align="center">
          <template #default="{ row }">{{ row.result ? row.result.maxTempC + '°C' : '-' }}</template>
        </el-table-column>
        <el-table-column label="ECC" width="70" align="center">
          <template #default="{ row }">{{ row.result?.eccTotal ?? '-' }}</template>
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
    <el-dialog v-model="detailVisible" :title="t('burn.detailTitle') + ' · ' + (detail?.id || '')" width="860px">
      <template v-if="detail">
        <el-alert :type="detail.result?.pass ? 'success' : 'error'" :closable="false" style="margin-bottom: 12px"
                  :title="detail.result?.summary || t('common.running')" />
        <EChart :option="detailTempOption" height="300px" />
        <EChart :option="detailPowerOption" height="220px" style="margin-top: 8px" />
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
  name: '', deviceLabel: '', gpus: 8, wattPerGpu: 600, durationSec: 259200,
  maxTempC: 85, acceptanceId: '', itemKey: '', injectFault: false,
})
const history = ref([])
const orders = ref([])
const categories = ref([])
const live = ref(null)
const runningTask = ref(null)
const detailVisible = ref(false)
const detail = ref(null)
let timer = null

const samples = computed(() => live.value?.samples || [])
const elapsed = computed(() => samples.value.length ? samples.value[samples.value.length - 1].t : 0)
const curMaxTemp = computed(() => {
  const last = samples.value[samples.value.length - 1]
  return last ? Math.max(...last.temps).toFixed(1) : '-'
})
const curPower = computed(() => {
  const last = samples.value[samples.value.length - 1]
  return last ? Math.round(last.powers.reduce((a, b) => a + b, 0)) : '-'
})
const curEcc = computed(() => samples.value.length ? samples.value[samples.value.length - 1].eccErrs : 0)
const eccColor = computed(() => (curEcc.value > 0 ? '#f56c6c' : '#1f2d3d'))

const itemTitleFor = (it) => itemTitle('gpu', it.key, it.title)

const linkableItems = computed(() => {
  const o = orders.value.find((x) => x.id === form.acceptanceId)
  if (!o) return []
  return o.items.filter((i) => i.category === 'burnin' || i.category === 'http')
})

function tempSeries(list) {
  const gpuCount = list.length ? list[0].temps.length : 0
  const series = []
  for (let g = 0; g < gpuCount; g++) {
    series.push({
      name: `GPU${g}`, type: 'line', smooth: true, showSymbol: false,
      data: list.map((s) => s.temps[g]),
    })
  }
  return {
    tooltip: { trigger: 'axis' },
    legend: { data: series.map((s) => s.name) },
    grid: { left: 50, right: 30, top: 40, bottom: 30 },
    xAxis: { type: 'category', data: list.map((s) => fmtDur(s.t)), name: t('http.timeAxis') },
    yAxis: { type: 'value', name: '°C', max: 95, min: 20 },
    series,
  }
}

function powerSeries(list) {
  return {
    tooltip: { trigger: 'axis' },
    legend: { data: [t('burn.totalPower')] },
    grid: { left: 60, right: 30, top: 36, bottom: 30 },
    xAxis: { type: 'category', data: list.map((s) => fmtDur(s.t)) },
    yAxis: { type: 'value', name: 'W' },
    series: [{
      name: t('burn.totalPower'), type: 'line', smooth: true, showSymbol: false, areaStyle: { opacity: 0.15 },
      data: list.map((s) => Math.round(s.powers.reduce((a, b) => a + b, 0))),
    }],
  }
}

const tempOption = computed(() => tempSeries(samples.value))
const powerOption = computed(() => powerSeries(samples.value))
const detailTempOption = computed(() => tempSeries(detail.value?.samples || []))
const detailPowerOption = computed(() => powerSeries(detail.value?.samples || []))

async function loadHistory() {
  history.value = (await get('/stress/burnin')) || []
}

async function start() {
  const tk = await post('/stress/burnin', { ...form })
  runningTask.value = tk
  live.value = tk
  ElMessage.success(`${t('burn.msgStarted')}${tk.id}（${fmtDur(tk.durationSec)}）`)
  startPolling(tk.id)
  loadHistory()
}

function startPolling(id) {
  stopPolling()
  timer = setInterval(async () => {
    const tk = await get(`/stress/burnin/${id}/live`)
    live.value = tk
    if (tk.status !== 'running') {
      stopPolling()
      runningTask.value = null
      loadHistory()
      ElMessage({
        type: tk.result?.pass ? 'success' : 'error',
        message: tk.result?.summary || t('common.running'),
        duration: 8000,
      })
      if (tk.acceptanceId && tk.itemKey) backfillSilent(tk)
    }
  }, Math.min(5000, (form.intervalSec || 5) * 1000))
}

function stopPolling() {
  if (timer) clearInterval(timer)
  timer = null
}

async function backfill(tk) {
  const o = await post(`/orders/${tk.acceptanceId}/items/${tk.itemKey}/apply-stress`, { stressId: tk.id, type: 'burnin' })
  ElMessage.success(`${t('burn.msgBackfilled')}${o.id}`)
}

async function backfillSilent(tk) {
  try {
    await post(`/orders/${tk.acceptanceId}/items/${tk.itemKey}/apply-stress`, { stressId: tk.id, type: 'burnin' })
    ElMessage.info(t('burn.msgAutoBackfill'))
  } catch { /* 未关联时忽略 */ }
}

function showDetail(row) {
  detail.value = row
  detailVisible.value = true
}

onMounted(async () => {
  if (route.query.acceptanceId) {
    form.acceptanceId = route.query.acceptanceId
    form.itemKey = route.query.itemKey || ''
    form.deviceLabel = route.query.device || ''
    form.name = `${route.query.device || ''} burn-in`
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
