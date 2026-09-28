<template>
  <div>
    <div class="page-title">设备验收</div>
    <div class="page-desc">
      按验收规范逐台（组）生成验收单：开箱核对 → 配置核对 → 满载烤机 → 性能实测 → 并发压测 → 软件验收 → 文档归档；
      烤机与并发压测可一键引用压测中心结果自动判定。
    </div>

    <div class="card-block">
      <div style="display: flex; gap: 12px; flex-wrap: wrap; align-items: center">
        <el-select v-model="filter.status" placeholder="验收单状态" clearable style="width: 150px" @change="load">
          <el-option v-for="s in orderStatuses" :key="s" :label="s" :value="s" />
        </el-select>
        <el-select v-model="filter.cat" placeholder="节点类别" clearable filterable style="width: 220px" @change="load">
          <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
        </el-select>
        <el-input v-model="filter.kw" placeholder="搜索 验收单号 / 设备" clearable style="width: 220px" @input="load" />
        <el-button type="primary" @click="openCreate"><el-icon><Plus /></el-icon>&nbsp;生成验收单</el-button>
      </div>

      <el-table :data="filtered" style="width: 100%; margin-top: 14px" border>
        <el-table-column prop="id" label="验收单号" width="110" />
        <el-table-column prop="deviceLabel" label="设备" min-width="200" show-overflow-tooltip />
        <el-table-column prop="categoryName" label="节点类别" min-width="180" show-overflow-tooltip />
        <el-table-column label="验收人" width="100">
          <template #default="{ row }">{{ insName(row.inspectorId) }}</template>
        </el-table-column>
        <el-table-column label="进度" width="140">
          <template #default="{ row }">
            <el-progress :percentage="progressPct(row)" :stroke-width="14"
                         :status="row.status === '整改中' ? 'exception' : row.status === '已通过' ? 'success' : undefined" />
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="orderTag(row.status)" effect="dark">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="165">
          <template #default="{ row }">{{ fmtTime(row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">验收</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 验收单详情 -->
    <el-drawer v-model="detailVisible" size="720px" :title="detail?.deviceLabel + ' · 验收单 ' + (detail?.id || '')">
      <template v-if="detail">
        <el-descriptions :column="2" border size="small" style="margin-bottom: 14px">
          <el-descriptions-item label="节点类别">{{ detail.categoryName }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="orderTag(detail.status)" effect="dark">{{ detail.status }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="验收人">
            <el-select v-model="detail.inspectorId" size="small" style="width: 150px" @change="saveInspector">
              <el-option v-for="i in installers" :key="i.id" :label="i.name + '（' + i.level + '）'" :value="i.id" />
            </el-select>
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ fmtTime(detail.createdAt) }}</el-descriptions-item>
        </el-descriptions>

        <el-alert v-if="detail.status === '整改中'" type="error" :closable="false" style="margin-bottom: 12px"
                  title="存在不合格条目，需整改后复检" />

        <div v-for="it in detail.items" :key="it.key" class="item-card">
          <div class="item-head">
            <span class="item-title">{{ it.title }}</span>
            <el-tag size="small" :type="catTag(it.category)">{{ catName(it.category) }}</el-tag>
            <el-tag v-if="it.mandatory" size="small" type="danger" effect="plain">强制项</el-tag>
            <el-tag size="small" :type="itemTag(it.status)" effect="dark">{{ it.status }}</el-tag>
          </div>
          <div class="item-content">{{ it.content }}</div>
          <div v-if="it.result" class="item-result">
            <span class="result-label">验收记录：</span>{{ it.result }}
            <span v-if="it.evidence" class="evidence">（{{ it.evidence }}）</span>
          </div>
          <div v-if="it.checkedBy || it.checkedAt" class="item-meta">
            验收人：{{ insName(it.checkedBy) }} · {{ fmtTime(it.checkedAt) }}
          </div>
          <div class="item-actions">
            <el-button size="small" type="primary" plain @click="openJudge(it)">判定结论</el-button>
            <template v-if="it.category === 'burnin'">
              <el-button size="small" type="warning" plain @click="launchBurnin(it)">
                <el-icon><Monitor /></el-icon>&nbsp;发起烤机
              </el-button>
              <el-button size="small" plain @click="openApply(it, 'burnin')">引用烤机结果</el-button>
            </template>
            <template v-if="it.category === 'http'">
              <el-button size="small" type="warning" plain @click="launchHttp(it)">
                <el-icon><DataLine /></el-icon>&nbsp;发起并发压测
              </el-button>
              <el-button size="small" plain @click="openApply(it, 'http')">引用压测结果</el-button>
            </template>
          </div>
        </div>

        <div class="card-block" style="margin-top: 6px">
          <div class="block-title">验收结论归档</div>
          <el-input v-model="detail.conclusion" type="textarea" :rows="2" placeholder="全部强制项合格后填写最终结论并归档" />
          <el-button type="success" style="margin-top: 10px" :disabled="detail.status !== '已通过'" @click="saveConclusion">
            保存结论
          </el-button>
        </div>
      </template>
    </el-drawer>

    <!-- 判定对话框 -->
    <el-dialog v-model="judgeVisible" :title="'判定 · ' + (judgeItem?.title || '')" width="520px">
      <el-form label-width="90px">
        <el-form-item label="结论">
          <el-radio-group v-model="judgeForm.status">
            <el-radio-button value="合格">合格</el-radio-button>
            <el-radio-button value="不合格">不合格</el-radio-button>
            <el-radio-button value="进行中">进行中</el-radio-button>
            <el-radio-button value="不适用">不适用</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="验收记录" required>
          <el-input v-model="judgeForm.result" type="textarea" :rows="3"
                    :placeholder="judgeForm.status === '不合格' ? '记录不合格详情与整改要求' : '记录核对/实测结果'" />
        </el-form-item>
        <el-form-item label="证据说明">
          <el-input v-model="judgeForm.evidence" placeholder="如：出厂测试报告编号 / 压测记录编号" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="judgeVisible = false">取消</el-button>
        <el-button type="primary" @click="submitJudge">提交</el-button>
      </template>
    </el-dialog>

    <!-- 引用压测结果 -->
    <el-dialog v-model="applyVisible" :title="'引用压测结果 · ' + (applyItem?.title || '')" width="640px">
      <el-table :data="applyList" border size="small" max-height="360" highlight-current-row
                @current-change="(r) => (applyPick = r)">
        <el-table-column prop="id" label="编号" width="90" />
        <el-table-column prop="name" label="任务" min-width="180" show-overflow-tooltip />
        <el-table-column label="判定" width="90">
          <template #default="{ row }">
            <el-tag :type="row.result?.pass ? 'success' : 'danger'" effect="dark">
              {{ row.result?.pass ? '合格' : '不合格' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="result.summary" label="摘要" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">{{ applySummary(row) }}</template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="applyVisible = false">取消</el-button>
        <el-button type="primary" :disabled="!applyPick" @click="submitApply">回填到验收条目</el-button>
      </template>
    </el-dialog>

    <!-- 生成验收单 -->
    <el-dialog v-model="createVisible" title="生成验收单" width="520px">
      <el-form label-width="90px">
        <el-form-item label="部署任务">
          <el-select v-model="createForm.taskId" style="width: 100%" placeholder="选择有剩余设备的部署任务">
            <el-option v-for="t in remainingTasks" :key="t.id"
                       :label="`${t.id} · ${t.categoryName}（已生成 ${t.accepted}/${t.quantity} ${t.unit}）`" :value="t.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="验收人">
          <el-select v-model="createForm.inspectorId" style="width: 100%" clearable placeholder="可稍后在验收单中指定">
            <el-option v-for="i in installers" :key="i.id" :label="i.name + '（' + i.level + '）· ' + i.title" :value="i.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :disabled="!createForm.taskId" @click="submitCreate">生成</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { get, post, put } from '../api'
import { fmtTime } from '../api'

const router = useRouter()
const orders = ref([])
const categories = ref([])
const installers = ref([])
const tasks = ref([])
const filter = reactive({ status: '', cat: '', kw: '' })
const orderStatuses = ['待验收', '进行中', '已通过', '整改中']

const detailVisible = ref(false)
const detail = ref(null)

const judgeVisible = ref(false)
const judgeItem = ref(null)
const judgeForm = reactive({ status: '合格', result: '', evidence: '' })

const applyVisible = ref(false)
const applyItem = ref(null)
const applyType = ref('http')
const applyList = ref([])
const applyPick = ref(null)

const createVisible = ref(false)
const createForm = reactive({ taskId: '', inspectorId: '' })

const insMap = computed(() => Object.fromEntries(installers.value.map((i) => [i.id, i.name])))
const insName = (id) => (id && insMap.value[id]) || '-'

const filtered = computed(() =>
  orders.value.filter(
    (o) =>
      (!filter.status || o.status === filter.status) &&
      (!filter.cat || o.categoryId === filter.cat) &&
      (!filter.kw || o.id.includes(filter.kw) || o.deviceLabel.includes(filter.kw)),
  ),
)

const remainingTasks = computed(() => tasks.value.filter((t) => t.accepted < t.quantity))

function orderTag(s) {
  return { 待验收: 'info', 进行中: 'warning', 已通过: 'success', 整改中: 'danger' }[s] || 'info'
}
function itemTag(s) {
  return { 待检: 'info', 进行中: 'warning', 合格: 'success', 不合格: 'danger', 不适用: '' }[s] || 'info'
}
function catTag(c) {
  return { manual: '', burnin: 'warning', http: 'warning', performance: 'danger' }[c] || 'info'
}
function catName(c) {
  return { manual: '人工检查', burnin: '满载烤机', http: '并发压测', performance: '性能实测' }[c] || c
}

function progressPct(o) {
  const done = o.items.filter((i) => ['合格', '不合格', '不适用'].includes(i.status)).length
  return o.items.length ? Math.round((done * 100) / o.items.length) : 0
}

async function load() {
  const [os, cs, is, ts] = await Promise.all([
    get('/orders'),
    get('/categories'),
    get('/installers'),
    get('/tasks').then((r) => r.tasks),
  ])
  orders.value = os || []
  categories.value = cs || []
  installers.value = is || []
  tasks.value = ts || []
}

function openDetail(row) {
  detail.value = row
  detailVisible.value = true
}

function openJudge(it) {
  judgeItem.value = it
  judgeForm.status = ['合格', '不合格'].includes(it.status) ? it.status : '合格'
  judgeForm.result = it.result || ''
  judgeForm.evidence = it.evidence || ''
  judgeVisible.value = true
}

async function submitJudge() {
  if (['合格', '不合格'].includes(judgeForm.status) && !judgeForm.result.trim()) {
    ElMessage.warning('判定合格/不合格必须填写验收记录')
    return
  }
  const o = await put(`/orders/${detail.value.id}/items/${judgeItem.value.key}`, {
    status: judgeForm.status,
    result: judgeForm.result,
    evidence: judgeForm.evidence,
    checkedBy: detail.value.inspectorId || '',
  })
  detail.value = o
  judgeVisible.value = false
  ElMessage.success(`条目已判定：${judgeForm.status}`)
  load()
}

function launchBurnin(it) {
  router.push({
    path: '/stress/burnin',
    query: { acceptanceId: detail.value.id, itemKey: it.key, device: detail.value.deviceLabel },
  })
}
function launchHttp(it) {
  router.push({
    path: '/stress/http',
    query: { acceptanceId: detail.value.id, itemKey: it.key, device: detail.value.deviceLabel },
  })
}

async function openApply(it, type) {
  applyItem.value = it
  applyType.value = type
  applyPick.value = null
  applyList.value = ((type === 'http' ? await get('/stress/http') : await get('/stress/burnin')) || [])
    .filter((t) => t.status === 'finished' && t.result)
  applyVisible.value = true
}

function applySummary(row) {
  if (applyType.value === 'http') {
    const r = row.result
    return `并发${row.concurrency} · RPS ${r.rps} · P95 ${r.p95}ms · 错误率 ${r.errRatePct}%`
  }
  return row.result.summary
}

async function submitApply() {
  const o = await post(`/orders/${detail.value.id}/items/${applyItem.value.key}/apply-stress`, {
    stressId: applyPick.value.id,
    type: applyType.value,
  })
  detail.value = o
  applyVisible.value = false
  ElMessage.success('压测结果已回填，条目自动判定')
  load()
}

function openCreate() {
  createForm.taskId = ''
  createForm.inspectorId = ''
  createVisible.value = true
}

async function submitCreate() {
  const o = await post(`/tasks/${createForm.taskId}/orders`, { inspectorId: createForm.inspectorId })
  createVisible.value = false
  ElMessage.success(`已生成验收单 ${o.id}：${o.deviceLabel}`)
  load()
  openDetail(o)
}

async function saveInspector() {
  const o = await put(`/orders/${detail.value.id}`, { inspectorId: detail.value.inspectorId })
  detail.value = o
  load()
}

async function saveConclusion() {
  const o = await put(`/orders/${detail.value.id}`, { conclusion: detail.value.conclusion })
  detail.value = o
  ElMessage.success('结论已归档')
  load()
}

onMounted(load)
</script>

<style scoped>
.item-card { border: 1px solid #e4e7ed; border-radius: 8px; padding: 12px 14px; margin-bottom: 10px; background: #fafbfc; }
.item-card:hover { border-color: #c0c4cc; }
.item-head { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.item-title { font-weight: 600; color: #1f2d3d; margin-right: 4px; }
.item-content { color: #5e6d82; font-size: 13px; margin: 8px 0; line-height: 1.6; }
.item-result { background: #f0f9eb; border-radius: 4px; padding: 6px 10px; font-size: 13px; color: #33691e; line-height: 1.6; }
.item-result .evidence { color: #8492a6; }
.item-meta { color: #8492a6; font-size: 12px; margin-top: 6px; }
.item-actions { margin-top: 10px; display: flex; gap: 8px; flex-wrap: wrap; }
</style>
