<template>
  <div>
    <div class="page-title">{{ t('dash.title') }}</div>
    <div class="page-desc">{{ t('dash.desc') }}</div>

    <el-row :gutter="14" style="margin-bottom: 16px">
      <el-col :span="4" v-for="s in stats" :key="s.label">
        <div class="card-block stat-card" :style="{ borderLeft: `4px solid ${s.color}` }">
          <div class="stat-v" :style="{ color: s.color }">{{ s.value }}</div>
          <div class="stat-l">{{ s.label }}</div>
        </div>
      </el-col>
    </el-row>

    <el-row :gutter="14">
      <el-col :span="8">
        <div class="card-block">
          <div class="block-title">{{ t('dash.statusPie') }}</div>
          <EChart :option="statusPieOption" height="280px" />
        </div>
      </el-col>
      <el-col :span="16">
        <div class="card-block">
          <div class="block-title">{{ t('dash.nodeBar') }}</div>
          <EChart :option="nodeBarOption" height="280px" />
        </div>
      </el-col>
    </el-row>

    <el-row :gutter="14">
      <el-col :span="16">
        <div class="card-block">
          <div class="block-title">{{ t('dash.recent') }}</div>
          <el-table :data="ov.recentOrders || []" size="small" border>
            <el-table-column prop="id" :label="t('dash.col.order')" width="100" />
            <el-table-column :label="t('dash.col.device')" min-width="190" show-overflow-tooltip>
              <template #default="{ row }">{{ localizeDeviceLabel(row.deviceLabel, categories) }}</template>
            </el-table-column>
            <el-table-column :label="t('common.status')" width="110">
              <template #default="{ row }">
                <el-tag size="small" :type="orderTag(row.status)" effect="dark">{{ statusLabel(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column :label="t('dash.col.progress')" width="200">
              <template #default="{ row }">
                <el-progress :percentage="pct(row)" :stroke-width="12"
                             :status="row.status === '整改中' ? 'exception' : row.status === '已通过' ? 'success' : undefined" />
              </template>
            </el-table-column>
            <el-table-column :label="t('dash.col.updated')" width="160">
              <template #default="{ row }">{{ fmtTime(row.updatedAt) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>
      <el-col :span="8">
        <div class="card-block">
          <div class="block-title">{{ t('dash.skillCover') }}</div>
          <EChart :option="skillOption" height="300px" />
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { get, fmtTime } from '../api'
import { t, statusLabel, skillName, catNameOf } from '../i18n'
import EChart from '../components/EChart.vue'

const ov = ref({})
const categories = ref([])

const stats = computed(() => [
  { label: t('dash.orders'), value: ov.value.orders?.total ?? 0, color: '#1f7cff' },
  { label: t('dash.passed'), value: ov.value.orders?.byStatus?.['已通过'] ?? 0, color: '#67c23a' },
  { label: t('dash.rework'), value: ov.value.orders?.byStatus?.['整改中'] ?? 0, color: '#f56c6c' },
  { label: t('dash.installers'), value: ov.value.installers?.total ?? 0, color: '#e6a23c' },
  { label: t('dash.deploying'), value: ov.value.tasks?.running ?? 0, color: '#909399' },
  { label: t('dash.records'), value: (ov.value.stress?.httpTotal ?? 0) + (ov.value.stress?.burnTotal ?? 0), color: '#9254de' },
])

const statusColors = { 待验收: '#909399', 进行中: '#e6a23c', 已通过: '#67c23a', 整改中: '#f56c6c' }

const statusPieOption = computed(() => {
  const by = ov.value.orders?.byStatus || {}
  return {
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [{
      type: 'pie', radius: ['42%', '68%'], center: ['50%', '45%'],
      label: { formatter: '{b}: {c}' },
      data: Object.keys(by)
        .filter((s) => by[s])
        .map((s) => ({ name: statusLabel(s), value: by[s], itemStyle: { color: statusColors[s] } })),
    }],
  }
})

const nodeBarOption = computed(() => {
  const np = ov.value.nodeProgress || []
  const catByName = (name) => categories.value.find((c) => c.name === name)
  return {
    tooltip: { trigger: 'axis' },
    legend: { data: [t('dash.legend.qty'), t('dash.legend.created'), t('dash.legend.passed')] },
    grid: { left: 45, right: 20, top: 40, bottom: 80 },
    xAxis: {
      type: 'category',
      data: np.map((n) => {
        const c = catByName(n.name)
        return c ? catNameOf(c) : n.name
      }),
      axisLabel: { interval: 0, rotate: 32, fontSize: 10 },
    },
    yAxis: { type: 'value', minInterval: 1 },
    series: [
      { name: t('dash.legend.qty'), type: 'bar', barMaxWidth: 22, itemStyle: { color: '#c6d3e8' }, data: np.map((n) => n.quantity) },
      { name: t('dash.legend.created'), type: 'bar', barMaxWidth: 22, itemStyle: { color: '#79bbff' }, data: np.map((n) => n.accepted) },
      { name: t('dash.legend.passed'), type: 'bar', barMaxWidth: 22, itemStyle: { color: '#67c23a' }, data: np.map((n) => n.passed) },
    ],
  }
})

const skillOption = computed(() => {
  const cover = ov.value.installers?.skillCover || {}
  const entries = Object.entries(cover)
  return {
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 20, top: 16, bottom: 80 },
    xAxis: { type: 'category', data: entries.map(([k]) => skillName(k)), axisLabel: { interval: 0, rotate: 40, fontSize: 10 } },
    yAxis: { type: 'value', minInterval: 1 },
    series: [{ type: 'bar', barMaxWidth: 18, data: entries.map(([, v]) => v), itemStyle: { color: '#e6a23c' } }],
  }
})

function orderTag(s) {
  return { 待验收: 'info', 进行中: 'warning', 已通过: 'success', 整改中: 'danger' }[s] || 'info'
}
function pct(o) {
  const done = o.items.filter((i) => ['合格', '不合格', '不适用'].includes(i.status)).length
  return o.items.length ? Math.round((done * 100) / o.items.length) : 0
}

onMounted(async () => {
  const [o, cs] = await Promise.all([get('/overview'), get('/categories')])
  ov.value = o
  categories.value = cs || []
})
</script>

<style scoped>
.stat-card { text-align: center; }
.stat-v { font-size: 26px; font-weight: 700; }
.stat-l { color: #8492a6; font-size: 13px; margin-top: 4px; }
</style>
