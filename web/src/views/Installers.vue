<template>
  <div>
    <div class="page-title">{{ t('ins.title') }}</div>
    <div class="page-desc">{{ t('ins.desc') }}</div>

    <!-- 岗位要求原文 -->
    <div class="card-block">
      <div class="block-title">{{ t('ins.reqTitle') }}</div>
      <div v-for="kind in ['gpu', 'cpu']" :key="kind" style="margin-bottom: 10px">
        <el-alert type="info" :closable="false">
          <template #title>
            <b>{{ kind === 'gpu' ? t('ins.gpuNodes') : t('ins.cpuNode') }}</b>
          </template>
          {{ reqText(kind) }}
        </el-alert>
      </div>
    </div>

    <!-- 人员名册 -->
    <div class="card-block">
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px">
        <div class="block-title" style="margin: 0">{{ t('ins.roster', { n: installers.length }) }}</div>
        <el-button type="primary" @click="openEdit()"><el-icon><Plus /></el-icon>&nbsp;{{ t('ins.add') }}</el-button>
      </div>
      <el-table :data="installers" border>
        <el-table-column prop="name" :label="t('ins.col.name')" width="100" />
        <el-table-column prop="title" :label="t('ins.col.title')" min-width="150" show-overflow-tooltip />
        <el-table-column :label="t('ins.col.level')" width="110">
          <template #default="{ row }">
            <el-tag :type="levelTag(row.level)" effect="dark">{{ levelName(row.level) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="phone" :label="t('ins.col.phone')" width="140" />
        <el-table-column :label="t('ins.col.skills')" min-width="320">
          <template #default="{ row }">
            <el-tag v-for="s in row.skills" :key="s" size="small" style="margin: 2px 4px 2px 0"
                    :type="isGpuSkill(s) ? 'primary' : 'warning'" effect="plain">{{ skillName(s) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('ins.col.available')" width="90">
          <template #default="{ row }">
            <el-tag :type="row.available ? 'success' : 'info'" size="small">
              {{ row.available ? t('ins.available') : t('ins.busy') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="130" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">{{ t('common.edit') }}</el-button>
            <el-popconfirm :title="t('ins.deleteConfirm', { n: row.name })" @confirm="removeInstaller(row)">
              <template #reference><el-button link type="danger">{{ t('common.delete') }}</el-button></template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 技能矩阵与人岗匹配 -->
    <div class="card-block">
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px">
        <div class="block-title" style="margin: 0">{{ t('ins.matrix') }}</div>
        <el-select v-model="matchCat" style="width: 280px">
          <el-option v-for="c in categories" :key="c.id" :label="catNameOf(c)" :value="c.id" />
        </el-select>
      </div>

      <el-table v-if="curCat" :data="curCat.requiredSkills.map((s) => ({ skill: s }))" border size="small">
        <el-table-column :label="t('ins.reqSkill')" min-width="200">
          <template #default="{ row }">{{ skillName(row.skill) }}</template>
        </el-table-column>
        <el-table-column v-for="i in installers" :key="i.id" :label="i.name" :width="76" align="center">
          <template #default="{ row }">
            <el-icon v-if="i.skills.includes(row.skill)" color="#67c23a"><CircleCheckFilled /></el-icon>
            <el-icon v-else color="#dcdfe6"><CircleClose /></el-icon>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="curCat" style="margin-top: 14px">
        <div class="block-title">{{ t('ins.matchFor', { c: catNameOf(curCat) }) }}</div>
        <el-table :data="matchRows" border size="small">
          <el-table-column prop="name" :label="t('ins.col.name')" width="110" />
          <el-table-column :label="t('ins.match')" min-width="240">
            <template #default="{ row }">
              <el-progress :percentage="row.pct" :stroke-width="14"
                           :status="row.pct === 100 ? 'success' : row.pct >= 60 ? undefined : 'exception'" />
            </template>
          </el-table-column>
          <el-table-column :label="t('ins.missing')" min-width="280">
            <template #default="{ row }">
              <template v-if="row.missing.length">
                <el-tag v-for="m in row.missing" :key="m" size="small" type="danger" effect="plain"
                        style="margin: 2px 4px 2px 0">{{ skillName(m) }}</el-tag>
              </template>
              <el-tag v-else size="small" type="success">{{ t('ins.allMet') }}</el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 编辑人员 -->
    <el-dialog v-model="editVisible" :title="form.id ? t('ins.edit') : t('ins.add')" width="580px">
      <el-form label-width="110px">
        <el-form-item :label="t('ins.fName')" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item :label="t('ins.fTitle')"><el-input v-model="form.title" :placeholder="t('ins.fTitlePh')" /></el-form-item>
        <el-form-item :label="t('ins.fLevel')" required>
          <el-select v-model="form.level" style="width: 160px">
            <el-option v-for="l in LEVELS" :key="l" :label="levelName(l)" :value="l" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('ins.fPhone')"><el-input v-model="form.phone" /></el-form-item>
        <el-form-item :label="t('ins.gpuSkills')">
          <el-checkbox-group v-model="form.skills">
            <el-checkbox v-for="s in skills.gpu" :key="s" :value="s" style="width: 100%">{{ skillName(s) }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item :label="t('ins.hpcSkills')">
          <el-checkbox-group v-model="form.skills">
            <el-checkbox v-for="s in skills.cpu" :key="s" :value="s" style="width: 100%">{{ skillName(s) }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item :label="t('ins.fAvailable')">
          <el-switch v-model="form.available" :active-text="t('ins.available')" :inactive-text="t('ins.busy')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="save">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { get, post, put, del } from '../api'
import { t, LEVELS, catNameOf, levelName, skillName, reqText } from '../i18n'

const installers = ref([])
const categories = ref([])
const skills = ref({ gpu: [], cpu: [] })
const matchCat = ref('NODE-01')
const editVisible = ref(false)
const form = reactive({ id: '', name: '', title: '', level: '中级', phone: '', skills: [], available: true })

const curCat = computed(() => categories.value.find((c) => c.id === matchCat.value))

const matchRows = computed(() => {
  if (!curCat.value) return []
  return installers.value
    .map((i) => {
      const missing = curCat.value.requiredSkills.filter((s) => !i.skills.includes(s))
      const pct = Math.round(((curCat.value.requiredSkills.length - missing.length) * 100) / curCat.value.requiredSkills.length)
      return { name: i.name, pct, missing }
    })
    .sort((a, b) => b.pct - a.pct)
})

const gpuSkillSet = computed(() => new Set(skills.value.gpu))
const isGpuSkill = (s) => gpuSkillSet.value.has(s)

function levelTag(l) {
  return { 专家级: 'danger', 高级: 'warning', 中级: 'primary', 初级: 'info' }[l] || 'info'
}

async function load() {
  const [is, cs, ss] = await Promise.all([get('/installers'), get('/categories'), get('/skills')])
  installers.value = is || []
  categories.value = cs || []
  skills.value = ss || { gpu: [], cpu: [] }
}

function openEdit(row) {
  Object.assign(form, row ? { ...row } : { id: '', name: '', title: '', level: '中级', phone: '', skills: [], available: true })
  editVisible.value = true
}

async function save() {
  if (!form.name.trim()) {
    ElMessage.warning(t('ins.msgNameRequired'))
    return
  }
  if (form.id) await put(`/installers/${form.id}`, { ...form })
  else await post('/installers', { ...form })
  editVisible.value = false
  ElMessage.success(t('ins.msgSaved'))
  load()
}

async function removeInstaller(row) {
  await del(`/installers/${row.id}`)
  ElMessage.success(t('ins.msgDeleted'))
  load()
}

onMounted(load)
</script>
