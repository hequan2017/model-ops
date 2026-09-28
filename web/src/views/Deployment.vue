<template>
  <div>
    <div class="page-title">{{ t('dep.title') }}</div>
    <div class="page-desc">{{ t('dep.desc') }}</div>

    <div class="card-block">
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px">
        <div class="block-title" style="margin: 0">{{ t('dep.tasks') }}</div>
        <el-button type="primary" @click="createVisible = true"><el-icon><Plus /></el-icon>&nbsp;{{ t('dep.new') }}</el-button>
      </div>

      <el-table :data="tasks" border :row-key="(r) => r.id">
        <el-table-column type="expand">
          <template #default="{ row }">
            <div style="padding: 12px 24px">
              <el-steps :active="row.stage" align-center finish-status="success">
                <el-step v-for="(s, i) in stageNames" :key="s" :title="s" :description="i === row.stage ? t('dep.current') : ''" />
              </el-steps>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="id" :label="t('dep.col.no')" width="100" />
        <el-table-column :label="t('dep.col.cat')" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">{{ catName(row.categoryId) }}</template>
        </el-table-column>
        <el-table-column :label="t('dep.col.qty')" width="90">
          <template #default="{ row }">{{ row.quantity }} {{ unitName(row.unit) }}</template>
        </el-table-column>
        <el-table-column :label="t('dep.col.stage')" width="130">
          <template #default="{ row }">
            <el-tag :type="row.status === '已完成' ? 'success' : row.status === '待启动' ? 'info' : 'warning'" effect="dark">
              {{ stageName(row.stage) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('dep.col.installers')" min-width="170">
          <template #default="{ row }">
            <template v-if="row.installerIds.length">
              <el-tag v-for="n in installerNames(row)" :key="n" size="small" style="margin: 2px 4px 2px 0">{{ n }}</el-tag>
            </template>
            <span v-else style="color: #c0c4cc">{{ t('dep.unassigned') }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('dep.col.orders')" width="90">
          <template #default="{ row }">
            <el-link type="primary" @click="goAcceptance(row)">{{ row.accepted }}/{{ row.quantity }}</el-link>
          </template>
        </el-table-column>
        <el-table-column :label="t('dep.col.started')" width="165">
          <template #default="{ row }">{{ fmtTime(row.startAt) }}</template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="250" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" plain :disabled="row.stage >= stageNames.length - 1"
                       @click="advance(row)">{{ t('dep.advance') }}</el-button>
            <el-button size="small" plain @click="openAssign(row)">{{ t('dep.assign') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 分派人员 -->
    <el-dialog v-model="assignVisible" :title="t('dep.assignDialog') + ' · ' + (assignTask ? catName(assignTask.categoryId) : '')" width="680px">
      <el-alert type="info" :closable="false" style="margin-bottom: 12px"
                :title="t('dep.reqColon') + (assignTask ? reqTextOf(assignTask.categoryId) : '')" />
      <el-table :data="assignRows" border size="small" @selection-change="(s) => (assignPick = s)"
                :row-key="(r) => r.id" ref="assignTable">
        <el-table-column type="selection" width="46" />
        <el-table-column prop="name" :label="t('ins.col.name')" width="100" />
        <el-table-column :label="t('ins.col.level')" width="110">
          <template #default="{ row }">{{ levelName(row.level) }}</template>
        </el-table-column>
        <el-table-column :label="t('ins.match')" min-width="160">
          <template #default="{ row }">
            <el-progress :percentage="row.pct" :stroke-width="12"
                         :status="row.pct === 100 ? 'success' : row.pct >= 60 ? undefined : 'exception'" />
          </template>
        </el-table-column>
        <el-table-column :label="t('ins.missing')" min-width="200">
          <template #default="{ row }">
            <span v-if="!row.missing.length" style="color: #67c23c">{{ t('ins.allMet') }}</span>
            <span v-else style="color: #f56c6c; font-size: 12px">
              {{ t('dep.missing') }}{{ row.missing.map(skillName).join(' / ') }}
            </span>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="assignVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitAssign">{{ t('dep.confirmAssign') }}</el-button>
      </template>
    </el-dialog>

    <!-- 新建任务 -->
    <el-dialog v-model="createVisible" :title="t('dep.newDialog')" width="520px">
      <el-form label-width="110px">
        <el-form-item :label="t('dep.fCat')">
          <el-select v-model="createForm.categoryId" style="width: 100%" :placeholder="t('dep.fCat')">
            <el-option v-for="c in categories" :key="c.id"
                       :label="`${catNameOf(c)}（${c.quantity} ${unitName(c.unit)}）`" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('dep.fNote')">
          <el-input v-model="createForm.note" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :disabled="!createForm.categoryId" @click="submitCreate">{{ t('common.create') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { get, post } from '../api'
import { fmtTime } from '../api'
import { t, catNameOf, levelName, unitName, stageName, STAGE_COUNT, skillName, reqText } from '../i18n'

const router = useRouter()
const tasks = ref([])
const installers = ref([])
const categories = ref([])

const assignVisible = ref(false)
const assignTask = ref(null)
const assignPick = ref([])
const assignTable = ref(null)

const createVisible = ref(false)
const createForm = reactive({ categoryId: '', note: '' })

const stageNames = computed(() => Array.from({ length: STAGE_COUNT }, (_, i) => stageName(i)))

const catById = (id) => categories.value.find((c) => c.id === id)
const catName = (id) => {
  const c = catById(id)
  return c ? catNameOf(c) : ''
}
const reqTextOf = (id) => {
  const c = catById(id)
  return c ? reqText(c.kind) : ''
}

const insMap = computed(() => Object.fromEntries(installers.value.map((i) => [i.id, i])))

const assignRows = computed(() => {
  if (!assignTask.value) return []
  const cat = catById(assignTask.value.categoryId)
  if (!cat) return []
  return installers.value
    .map((i) => {
      const missing = cat.requiredSkills.filter((s) => !i.skills.includes(s))
      const pct = Math.round(((cat.requiredSkills.length - missing.length) * 100) / cat.requiredSkills.length)
      return { ...i, pct, missing }
    })
    .sort((a, b) => b.pct - a.pct)
})

function installerNames(row) {
  return row.installerIds.map((id) => insMap.value[id]?.name).filter(Boolean)
}

async function load() {
  const [taskResp, is, cs] = await Promise.all([get('/tasks'), get('/installers'), get('/categories')])
  tasks.value = (taskResp.tasks || []).map((tk) => ({ ...tk, installerIds: tk.installerIds || [] }))
  installers.value = is || []
  categories.value = cs || []
}

async function advance(row) {
  const next = row.stage + 1
  await post(`/tasks/${row.id}/stage`, { stage: next })
  ElMessage.success(`${t('dep.msgAdvanced')}${stageName(next)}`)
  load()
}

async function openAssign(row) {
  assignTask.value = row
  assignVisible.value = true
  await nextTick()
  assignTable.value?.clearSelection()
  for (const r of assignRows.value) {
    if (row.installerIds.includes(r.id)) assignTable.value?.toggleRowSelection(r, true)
  }
}

async function submitAssign() {
  await post(`/tasks/${assignTask.value.id}/assign`, { installerIds: assignPick.value.map((r) => r.id) })
  assignVisible.value = false
  ElMessage.success(t('dep.msgAssigned'))
  load()
}

async function submitCreate() {
  const tk = await post('/tasks', createForm)
  createVisible.value = false
  ElMessage.success(`${t('dep.msgCreated')} ${tk.id}`)
  load()
}

function goAcceptance(row) {
  router.push({ path: '/acceptance', query: { cat: row.categoryId } })
}

onMounted(load)
</script>
