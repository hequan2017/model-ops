<template>
  <div>
    <div class="page-title">{{ t('spec.title') }}</div>
    <div class="page-desc">{{ t('spec.desc') }}</div>

    <!-- GPU 类 -->
    <div class="card-block" v-if="gpuCat">
      <div class="block-title">
        <el-icon color="#1f7cff"><Cpu /></el-icon>&nbsp;{{ t('spec.gpuCommon') }}
      </div>
      <el-alert type="info" :closable="false" style="margin-bottom: 14px">
        <template #title><b>{{ t('spec.req') }}</b></template>
        {{ reqText('gpu') }}
      </el-alert>

      <div class="skill-line">
        <span class="skill-label">{{ t('spec.reqSkills') }}</span>
        <el-tag v-for="s in gpuCat.requiredSkills" :key="s" size="small" effect="plain" style="margin: 2px 6px 2px 0">{{ skillName(s) }}</el-tag>
      </div>

      <div class="block-title" style="margin-top: 18px">{{ t('spec.tmpl') }}</div>
      <el-table :data="gpuCat.items || []" border>
        <el-table-column :label="t('spec.col.item')" width="220">
          <template #default="{ row }">
            <b>{{ itemTitle('gpu', row.key, row.title) }}</b>
            <el-tag v-if="row.mandatory" size="small" type="danger" effect="plain" style="margin-left: 6px">{{ t('acc.mandatory') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('spec.col.content')" min-width="460">
          <template #default="{ row }">{{ itemContent('gpu', row.key, row.content) }}</template>
        </el-table-column>
        <el-table-column :label="t('spec.col.method')" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="catTag(row.category)">{{ methodName(row.category) }}</el-tag>
          </template>
        </el-table-column>
      </el-table>

      <div class="block-title" style="margin-top: 18px">{{ t('spec.applyTo') }}</div>
      <el-table :data="gpuCats" border size="small">
        <el-table-column :label="t('spec.col.no')" width="80" align="center">
          <template #default="{ row }">{{ row.code }}</template>
        </el-table-column>
        <el-table-column :label="t('dep.col.cat')" min-width="220">
          <template #default="{ row }">{{ catNameOf(row) }}</template>
        </el-table-column>
        <el-table-column :label="t('spec.col.qty')" width="120" align="center">
          <template #default="{ row }">{{ row.quantity }} {{ unitName(row.unit) }}</template>
        </el-table-column>
      </el-table>
    </div>

    <!-- CPU 类 -->
    <div class="card-block" v-if="cpuCat">
      <div class="block-title">
        <el-icon color="#e6a23c"><Cpu /></el-icon>&nbsp;{{ t('spec.cpuTitle') }}
      </div>
      <el-alert type="info" :closable="false" style="margin-bottom: 14px">
        <template #title><b>{{ t('spec.req') }}</b></template>
        {{ reqText('cpu') }}
      </el-alert>

      <div class="skill-line">
        <span class="skill-label">{{ t('spec.reqSkills') }}</span>
        <el-tag v-for="s in cpuCat.requiredSkills" :key="s" size="small" type="warning" effect="plain" style="margin: 2px 6px 2px 0">{{ skillName(s) }}</el-tag>
      </div>

      <div class="block-title" style="margin-top: 18px">{{ t('spec.tmpl') }}</div>
      <el-table :data="cpuCat.items || []" border>
        <el-table-column :label="t('spec.col.item')" width="220">
          <template #default="{ row }">
            <b>{{ itemTitle('cpu', row.key, row.title) }}</b>
            <el-tag v-if="row.mandatory" size="small" type="danger" effect="plain" style="margin-left: 6px">{{ t('acc.mandatory') }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('spec.col.content')" min-width="460">
          <template #default="{ row }">{{ itemContent('cpu', row.key, row.content) }}</template>
        </el-table-column>
        <el-table-column :label="t('spec.col.method')" width="120">
          <template #default="{ row }">
            <el-tag size="small" :type="catTag(row.category)">{{ methodName(row.category) }}</el-tag>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { get } from '../api'
import { t, catNameOf, skillName, reqText, itemTitle, itemContent, methodName, unitName } from '../i18n'

const categories = ref([])
const gpuCat = computed(() => categories.value.find((c) => c.kind === 'gpu'))
const gpuCats = computed(() => categories.value.filter((c) => c.kind === 'gpu'))
const cpuCat = computed(() => categories.value.find((c) => c.kind === 'cpu'))

function catTag(c) {
  return { manual: '', burnin: 'warning', http: 'warning', performance: 'danger' }[c] || 'info'
}

onMounted(async () => {
  categories.value = (await get('/categories')) || []
})
</script>

<style scoped>
.skill-line { line-height: 2; }
.skill-label { color: #8492a6; font-size: 13px; }
</style>
