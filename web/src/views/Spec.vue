<template>
  <div>
    <div class="page-title">验收规范 · 安装人员要求</div>
    <div class="page-desc">
      各节点类别的安装人员要求原文与设备验收事宜条款，是验收单条目模板与人岗匹配的唯一依据。
    </div>

    <!-- GPU 类 -->
    <div class="card-block">
      <div class="block-title">
        <el-icon color="#1f7cff"><Cpu /></el-icon>&nbsp;GPU 类节点（通用规范）
      </div>
      <el-alert type="info" :closable="false" style="margin-bottom: 14px">
        <template #title><b>安装人员要求</b></template>
        {{ gpuReq.installerReq }}
      </el-alert>

      <div class="skill-line">
        <span class="skill-label">要求技能标签：</span>
        <el-tag v-for="s in gpuReq.requiredSkills" :key="s" size="small" effect="plain" style="margin: 2px 6px 2px 0">{{ s }}</el-tag>
      </div>

      <div class="block-title" style="margin-top: 18px">设备验收事宜（验收单条目模板）</div>
      <el-table :data="gpuReq.items || []" border>
        <el-table-column label="条目" width="190">
          <template #default="{ row }">
            <b>{{ row.title }}</b>
            <el-tag v-if="row.mandatory" size="small" type="danger" effect="plain" style="margin-left: 6px">强制项</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="content" label="验收内容" min-width="440" />
        <el-table-column label="执行方式" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="catTag(row.category)">{{ catName(row.category) }}</el-tag>
          </template>
        </el-table-column>
      </el-table>

      <div class="block-title" style="margin-top: 18px">适用节点类别</div>
      <el-table :data="gpuCats" border size="small">
        <el-table-column prop="code" label="编号" width="70" align="center" />
        <el-table-column prop="name" label="节点类别" min-width="200" />
        <el-table-column label="推荐量" width="110" align="center">
          <template #default="{ row }">{{ row.quantity }} {{ row.unit }}</template>
        </el-table-column>
      </el-table>
    </div>

    <!-- CPU 类 -->
    <div class="card-block" v-if="cpuReq">
      <div class="block-title">
        <el-icon color="#e6a23c"><Cpu /></el-icon>&nbsp;CPU 仿真节点
      </div>
      <el-alert type="info" :closable="false" style="margin-bottom: 14px">
        <template #title><b>安装人员要求</b></template>
        {{ cpuReq.installerReq }}
      </el-alert>

      <div class="skill-line">
        <span class="skill-label">要求技能标签：</span>
        <el-tag v-for="s in cpuReq.requiredSkills" :key="s" size="small" type="warning" effect="plain" style="margin: 2px 6px 2px 0">{{ s }}</el-tag>
      </div>

      <div class="block-title" style="margin-top: 18px">设备验收事宜（验收单条目模板）</div>
      <el-table :data="cpuReq.items || []" border>
        <el-table-column label="条目" width="190">
          <template #default="{ row }">
            <b>{{ row.title }}</b>
            <el-tag v-if="row.mandatory" size="small" type="danger" effect="plain" style="margin-left: 6px">强制项</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="content" label="验收内容" min-width="440" />
        <el-table-column label="执行方式" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="catTag(row.category)">{{ catName(row.category) }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { get } from '../api'

const categories = ref([])
const gpuReq = computed(() => categories.value.find((c) => c.kind === 'gpu') || {})
const gpuCats = computed(() => categories.value.filter((c) => c.kind === 'gpu'))
const cpuReq = computed(() => categories.value.find((c) => c.kind === 'cpu'))

function catTag(c) {
  return { manual: '', burnin: 'warning', http: 'warning', performance: 'danger' }[c] || 'info'
}
function catName(c) {
  return { manual: '人工检查', burnin: '满载烤机', http: '并发压测', performance: '性能实测' }[c] || c
}

onMounted(async () => {
  categories.value = (await get('/categories')) || []
})
</script>

<style scoped>
.skill-line { line-height: 2; }
.skill-label { color: #8492a6; font-size: 13px; }
</style>
