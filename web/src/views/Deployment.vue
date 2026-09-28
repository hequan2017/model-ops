<template>
  <div>
    <div class="page-title">部署管理</div>
    <div class="page-desc">
      按节点类别跟踪设备部署全流程：到货清点 → 上架安装 → 组网接入 → 系统与驱动 → 压测烤机 → 设备验收 → 交付使用；
      安装人员分派参考「安装人员」页的匹配度。
    </div>

    <div class="card-block">
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px">
        <div class="block-title" style="margin: 0">部署任务</div>
        <el-button type="primary" @click="createVisible = true"><el-icon><Plus /></el-icon>&nbsp;新建部署任务</el-button>
      </div>

      <el-table :data="tasks" border :row-key="(r) => r.id">
        <el-table-column type="expand">
          <template #default="{ row }">
            <div style="padding: 12px 24px">
              <el-steps :active="row.stage" align-center finish-status="success">
                <el-step v-for="(s, i) in stages" :key="s" :title="s" :description="i === row.stage ? '当前阶段' : ''" />
              </el-steps>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="id" label="任务号" width="100" />
        <el-table-column prop="categoryName" label="节点类别" min-width="170" show-overflow-tooltip />
        <el-table-column label="数量" width="90">
          <template #default="{ row }">{{ row.quantity }} {{ row.unit }}</template>
        </el-table-column>
        <el-table-column label="当前阶段" width="110">
          <template #default="{ row }">
            <el-tag :type="row.status === '已完成' ? 'success' : row.status === '待启动' ? 'info' : 'warning'" effect="dark">
              {{ stages[row.stage] }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="安装人员" min-width="170">
          <template #default="{ row }">
            <template v-if="row.installerIds.length">
              <el-tag v-for="n in installerNames(row)" :key="n" size="small" style="margin: 2px 4px 2px 0">{{ n }}</el-tag>
            </template>
            <span v-else style="color: #c0c4cc">未分派</span>
          </template>
        </el-table-column>
        <el-table-column label="验收单" width="100">
          <template #default="{ row }">
            <el-link type="primary" @click="goAcceptance(row)">{{ row.accepted }}/{{ row.quantity }}</el-link>
          </template>
        </el-table-column>
        <el-table-column label="开始时间" width="165">
          <template #default="{ row }">{{ fmtTime(row.startAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="250" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" plain :disabled="row.stage >= stages.length - 1"
                       @click="advance(row)">推进阶段</el-button>
            <el-button size="small" plain @click="openAssign(row)">分派人员</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 分派人员 -->
    <el-dialog v-model="assignVisible" :title="`分派安装人员 · ${assignTask?.categoryName || ''}`" width="640px">
      <el-alert type="info" :closable="false" style="margin-bottom: 12px"
                :title="`岗位要求：${categoryReq(assignTask?.categoryId)}`" />
      <el-table :data="assignRows" border size="small" @selection-change="(s) => (assignPick = s)"
                :row-key="(r) => r.id" ref="assignTable">
        <el-table-column type="selection" width="46" :selectable="(r) => true" />
        <el-table-column prop="name" label="姓名" width="90" />
        <el-table-column prop="level" label="等级" width="90" />
        <el-table-column label="匹配度" min-width="160">
          <template #default="{ row }">
            <el-progress :percentage="row.pct" :stroke-width="12"
                         :status="row.pct === 100 ? 'success' : row.pct >= 60 ? undefined : 'exception'" />
          </template>
        </el-table-column>
        <el-table-column label="缺失" min-width="180">
          <template #default="{ row }">
            <span v-if="!row.missing.length" style="color: #67c23a">全部满足</span>
            <span v-else style="color: #f56c6c; font-size: 12px">缺 {{ row.missing.join('、') }}</span>
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="assignVisible = false">取消</el-button>
        <el-button type="primary" @click="submitAssign">确认分派</el-button>
      </template>
    </el-dialog>

    <!-- 新建任务 -->
    <el-dialog v-model="createVisible" title="新建部署任务" width="520px">
      <el-form label-width="90px">
        <el-form-item label="节点类别">
          <el-select v-model="createForm.categoryId" style="width: 100%" placeholder="选择节点类别">
            <el-option v-for="c in categories" :key="c.id"
                       :label="`${c.name}（${c.quantity} ${c.unit}）`" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="createForm.note" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :disabled="!createForm.categoryId" @click="submitCreate">创建</el-button>
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

const router = useRouter()
const tasks = ref([])
const stages = ref([])
const installers = ref([])
const categories = ref([])

const assignVisible = ref(false)
const assignTask = ref(null)
const assignPick = ref([])
const assignTable = ref(null)

const createVisible = ref(false)
const createForm = reactive({ categoryId: '', note: '' })

const catMap = computed(() => Object.fromEntries(categories.value.map((c) => [c.id, c])))
const insMap = computed(() => Object.fromEntries(installers.value.map((i) => [i.id, i])))

const assignRows = computed(() => {
  if (!assignTask.value) return []
  const cat = catMap.value[assignTask.value.categoryId]
  if (!cat) return []
  return installers.value
    .map((i) => {
      const missing = cat.requiredSkills.filter((s) => !i.skills.includes(s))
      const pct = Math.round(((cat.requiredSkills.length - missing.length) * 100) / cat.requiredSkills.length)
      return { ...i, pct, missing }
    })
    .sort((a, b) => b.pct - a.pct)
})

function installerNames(t) {
  return t.installerIds.map((id) => insMap.value[id]?.name).filter(Boolean)
}
function categoryReq(catId) {
  return catMap.value[catId]?.installerReq || '-'
}

async function load() {
  const [taskResp, is, cs] = await Promise.all([get('/tasks'), get('/installers'), get('/categories')])
  stages.value = taskResp.stages || []
  tasks.value = (taskResp.tasks || []).map((t) => ({ ...t, installerIds: t.installerIds || [] }))
  installers.value = is || []
  categories.value = cs || []
}

async function advance(row) {
  const next = row.stage + 1
  await post(`/tasks/${row.id}/stage`, { stage: next })
  ElMessage.success(`已推进至：${stages.value[next]}`)
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
  ElMessage.success('分派已更新')
  load()
}

async function submitCreate() {
  const t = await post('/tasks', createForm)
  createVisible.value = false
  ElMessage.success(`已创建部署任务 ${t.id}`)
  load()
}

function goAcceptance(row) {
  router.push({ path: '/acceptance', query: { cat: row.categoryId } })
}

onMounted(load)
</script>
