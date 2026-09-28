<template>
  <div>
    <div class="page-title">满载烤机监控</div>
    <div class="page-desc">
      对应验收条目「③ 72h满载烤机：运行GPU压测监测温度/功耗/稳定性，ECC事件日志确认0错误」。
      判定标准与真实流程一致：满载时长内 ECC 错误累计为 0 且最高温度不超限 → 合格。
    </div>

    <el-alert type="warning" :closable="false" style="margin-bottom: 16px"
              title="演示模式说明"
              description="平台宿主机无真实 GPU，遥测数据为模拟采集（温度爬升/按卡功耗/利用率/ECC计数）。接入真实环境时，将采样源替换为 DCGM-Exporter / nvidia-smi 即可，判定逻辑无需改动。" />

    <div class="card-block">
      <div class="block-title">发起烤机任务</div>
      <el-form :inline="true" label-width="110px">
        <el-form-item label="任务名称">
          <el-input v-model="form.name" style="width: 240px" placeholder="如：科研微调节点 72h满载烤机" />
        </el-form-item>
        <el-form-item label="设备">
          <el-input v-model="form.deviceLabel" style="width: 220px" placeholder="设备标签" />
        </el-form-item>
        <el-form-item label="GPU数量">
          <el-input-number v-model="form.gpus" :min="1" :max="16" />
        </el-form-item>
        <el-form-item label="单卡功耗(W)">
          <el-input-number v-model="form.wattPerGpu" :min="50" :max="800" :step="50" />
        </el-form-item>
        <el-form-item label="时长">
          <el-input-number v-model="form.durationSec" :min="5" :max="259200" :step="60" />
          <el-button size="small" style="margin-left: 8px" @click="form.durationSec = 259200">72h</el-button>
          <el-button size="small" @click="form.durationSec = 120">演示2分钟</el-button>
        </el-form-item>
        <el-form-item label="温度限值(°C)">
          <el-input-number v-model="form.maxTempC" :min="60" :max="95" />
        </el-form-item>
        <el-form-item label="关联验收单">
          <el-select v-model="form.acceptanceId" clearable style="width: 280px" placeholder="可选：完成后回填验收条目">
            <el-option v-for="o in orders" :key="o.id" :label="`${o.id} · ${o.deviceLabel}`" :value="o.id" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.acceptanceId" label="验收条目">
          <el-select v-model="form.itemKey" style="width: 280px">
            <el-option v-for="it in linkableItems" :key="it.key" :label="it.title" :value="it.key" />
          </el-select>
        </el-form-item>
        <el-form-item label="故障注入">
          <el-switch v-model="form.injectFault" active-text="演示注入ECC错误" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :loading="runningTask !== null" @click="start">
            <el-icon><VideoPlay /></el-icon>&nbsp;开始烤机
          </el-button>
        </el-form-item>
      </el-form>
    </div>

    <!-- 实时监控 -->
    <div v-if="live" class="card-block">
      <div class="block-title">
        实时监控 · {{ live.id }}（{{ live.deviceLabel }}）
        <el-tag v-if="live.status === 'running'" style="margin-left: 8px" type="warning" effect="dark">
          运行中 {{ elapsed }}s / {{ live.durationSec }}s
        </el-tag>
        <el-tag v-else style="margin-left: 8px" :type="live.result?.pass ? 'success' : 'danger'" effect="dark">
          {{ live.result?.pass ? '判定合格' : '判定不合格' }}
        </el-tag>
      </div>
      <el-row :gutter="12" style="margin-bottom: 12px">
        <el-col :span="6"><div class="stat"><div class="stat-v">{{ curMaxTemp }}°C</div><div class="stat-l">当前最高温度（限值 {{ live.maxTempC }}°C）</div></div></el-col>
        <el-col :span="6"><div class="stat"><div class="stat-v">{{ curPower }}W</div><div class="stat-l">整机当前功耗（{{ live.gpus }} 卡）</div></div></el-col>
        <el-col :span="6"><div class="stat"><div class="stat-v" :style="{ color: eccColor }">{{ curEcc }}</div><div class="stat-l">ECC 错误累计（须为 0）</div></div></el-col>
        <el-col :span="6"><div class="stat"><div class="stat-v">{{ samples.length }}</div><div class="stat-l">已采样点数</div></div></el-col>
      </el-row>
      <EChart :option="tempOption" height="300px" />
      <EChart :option="powerOption" height="240px" style="margin-top: 8px" />
    </div>

    <!-- 历史 -->
    <div class="card-block">
      <div class="block-title">烤机记录</div>
      <el-table :data="history" border size="small">
        <el-table-column prop="id" label="编号" width="90" />
        <el-table-column prop="name" label="任务" min-width="190" show-overflow-tooltip />
        <el-table-column prop="deviceLabel" label="设备" min-width="180" show-overflow-tooltip />
        <el-table-column label="GPU" width="60" align="center">
          <template #default="{ row }">{{ row.gpus }}</template>
        </el-table-column>
        <el-table-column label="时长" width="100" align="center">
          <template #default="{ row }">{{ fmtDur(row.durationSec) }}</template>
        </el-table-column>
        <el-table-column label="最高温度" width="95" align="center">
          <template #default="{ row }">{{ row.result ? row.result.maxTempC + '°C' : '-' }}</template>
        </el-table-column>
        <el-table-column label="ECC" width="70" align="center">
          <template #default="{ row }">{{ row.result?.eccTotal ?? '-' }}</template>
        </el-table-column>
        <el-table-column label="判定" width="95" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.status === 'running'" type="warning">运行中</el-tag>
            <el-tag v-else :type="row.result?.pass ? 'success' : 'danger'" effect="dark">
              {{ row.result?.pass ? '合格' : '不合格' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="开始时间" width="160">
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
    <el-dialog v-model="detailVisible" :title="'烤机详情 · ' + (detail?.id || '')" width="860px">
      <template v-if="detail">
        <el-alert :type="detail.result?.pass ? 'success' : 'error'" :closable="false" style="margin-bottom: 12px"
                  :title="detail.result?.summary || '任务进行中'" />
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
import EChart from '../components/EChart.vue'

const route = useRoute()
const form = reactive({
  name: '', deviceLabel: '', gpus: 8, wattPerGpu: 600, durationSec: 259200,
  maxTempC: 85, acceptanceId: '', itemKey: '', injectFault: false,
})
const history = ref([])
const orders = ref([])
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
    xAxis: { type: 'category', data: list.map((s) => fmtDur(s.t)), name: '时间' },
    yAxis: { type: 'value', name: '°C', max: 95, min: 20 },
    series,
  }
}

function powerSeries(list) {
  return {
    tooltip: { trigger: 'axis' },
    legend: { data: ['整机功耗(W)'] },
    grid: { left: 60, right: 30, top: 36, bottom: 30 },
    xAxis: { type: 'category', data: list.map((s) => fmtDur(s.t)) },
    yAxis: { type: 'value', name: 'W' },
    series: [{
      name: '整机功耗(W)', type: 'line', smooth: true, showSymbol: false, areaStyle: { opacity: 0.15 },
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
  if (form.durationSec >= 259200 && !form.name.includes('72h')) form.name = form.name || '满载烤机'
  const t = await post('/stress/burnin', { ...form })
  runningTask.value = t
  live.value = t
  ElMessage.success(`烤机任务已启动：${t.id}（${fmtDur(t.durationSec)}，采样间隔 ${t.intervalSec}s）`)
  startPolling(t.id)
  loadHistory()
}

function startPolling(id) {
  stopPolling()
  timer = setInterval(async () => {
    const t = await get(`/stress/burnin/${id}/live`)
    live.value = t
    if (t.status !== 'running') {
      stopPolling()
      runningTask.value = null
      loadHistory()
      ElMessage({
        type: t.result?.pass ? 'success' : 'error',
        message: t.result?.summary || '烤机结束',
        duration: 8000,
      })
      if (t.acceptanceId && t.itemKey) backfillSilent(t)
    }
  }, Math.min(5000, (form.intervalSec || 5) * 1000))
}

function stopPolling() {
  if (timer) clearInterval(timer)
  timer = null
}

async function backfill(t) {
  const o = await post(`/orders/${t.acceptanceId}/items/${t.itemKey}/apply-stress`, { stressId: t.id, type: 'burnin' })
  ElMessage.success(`已回填验收单 ${o.id}，条目判定：${o.status}`)
}

async function backfillSilent(t) {
  try {
    const o = await post(`/orders/${t.acceptanceId}/items/${t.itemKey}/apply-stress`, { stressId: t.id, type: 'burnin' })
    ElMessage.info(`验收单 ${o.id} 条目已按烤机结果自动判定`)
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
    form.name = `${route.query.device || '设备'} 满载烤机`
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
