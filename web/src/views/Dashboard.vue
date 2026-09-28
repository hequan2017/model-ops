<template>
  <div>
    <div class="page-title">总览</div>
    <div class="page-desc">安装人员要求与设备验收事宜全景：验收进度、压测执行、人岗匹配覆盖。</div>

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
          <div class="block-title">验收单状态分布</div>
          <EChart :option="statusPieOption" height="280px" />
        </div>
      </el-col>
      <el-col :span="16">
        <div class="card-block">
          <div class="block-title">各节点类别验收进度（已生成验收单 / 设备数量）</div>
          <EChart :option="nodeBarOption" height="280px" />
        </div>
      </el-col>
    </el-row>

    <el-row :gutter="14">
      <el-col :span="16">
        <div class="card-block">
          <div class="block-title">最近验收动态</div>
          <el-table :data="ov.recentOrders || []" size="small" border>
            <el-table-column prop="id" label="验收单" width="100" />
            <el-table-column prop="deviceLabel" label="设备" min-width="190" show-overflow-tooltip />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="orderTag(row.status)" effect="dark">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="进度" width="200">
              <template #default="{ row }">
                <el-progress :percentage="pct(row)" :stroke-width="12"
                             :status="row.status === '整改中' ? 'exception' : row.status === '已通过' ? 'success' : undefined" />
              </template>
            </el-table-column>
            <el-table-column label="更新时间" width="160">
              <template #default="{ row }">{{ fmtTime(row.updatedAt) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </el-col>
      <el-col :span="8">
        <div class="card-block">
          <div class="block-title">安装技能覆盖（具备人数）</div>
          <EChart :option="skillOption" height="300px" />
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { get, fmtTime } from '../api'
import EChart from '../components/EChart.vue'

const ov = ref({})

const stats = computed(() => [
  { label: '验收单总数', value: ov.value.orders?.total ?? 0, color: '#1f7cff' },
  { label: '已通过', value: ov.value.orders?.byStatus?.['已通过'] ?? 0, color: '#67c23a' },
  { label: '整改中', value: ov.value.orders?.byStatus?.['整改中'] ?? 0, color: '#f56c6c' },
  { label: '安装人员', value: ov.value.installers?.total ?? 0, color: '#e6a23c' },
  { label: '进行中部署', value: ov.value.tasks?.running ?? 0, color: '#909399' },
  { label: '压测记录', value: (ov.value.stress?.httpTotal ?? 0) + (ov.value.stress?.burnTotal ?? 0), color: '#9254de' },
])

const orderStatusNames = ['待验收', '进行中', '已通过', '整改中']
const statusColors = { 待验收: '#909399', 进行中: '#e6a23c', 已通过: '#67c23a', 整改中: '#f56c6c' }

const statusPieOption = computed(() => {
  const by = ov.value.orders?.byStatus || {}
  return {
    tooltip: { trigger: 'item' },
    legend: { bottom: 0 },
    series: [{
      type: 'pie', radius: ['42%', '68%'], center: ['50%', '45%'],
      label: { formatter: '{b}: {c}' },
      data: orderStatusNames.filter((s) => by[s]).map((s) => ({ name: s, value: by[s], itemStyle: { color: statusColors[s] } })),
    }],
  }
})

const nodeBarOption = computed(() => {
  const np = ov.value.nodeProgress || []
  return {
    tooltip: { trigger: 'axis' },
    legend: { data: ['设备数量', '已生成验收单', '已通过'] },
    grid: { left: 45, right: 20, top: 40, bottom: 70 },
    xAxis: { type: 'category', data: np.map((n) => n.name), axisLabel: { interval: 0, rotate: 32, fontSize: 10 } },
    yAxis: { type: 'value', minInterval: 1 },
    series: [
      { name: '设备数量', type: 'bar', barMaxWidth: 22, itemStyle: { color: '#c6d3e8' }, data: np.map((n) => n.quantity) },
      { name: '已生成验收单', type: 'bar', barMaxWidth: 22, itemStyle: { color: '#79bbff' }, data: np.map((n) => n.accepted) },
      { name: '已通过', type: 'bar', barMaxWidth: 22, itemStyle: { color: '#67c23a' }, data: np.map((n) => n.passed) },
    ],
  }
})

const skillOption = computed(() => {
  const cover = ov.value.installers?.skillCover || {}
  const entries = Object.entries(cover)
  return {
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 20, top: 16, bottom: 60 },
    xAxis: { type: 'category', data: entries.map(([k]) => k), axisLabel: { interval: 0, rotate: 40, fontSize: 10 } },
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
  ov.value = await get('/overview')
})
</script>

<style scoped>
.stat-card { text-align: center; }
.stat-v { font-size: 26px; font-weight: 700; }
.stat-l { color: #8492a6; font-size: 13px; margin-top: 4px; }
</style>
