<template>
  <div>
    <div class="page-title">{{ t('acc.title') }}</div>
    <div class="page-desc">{{ t('acc.desc') }}</div>

    <div class="card-block">
      <div style="display: flex; gap: 12px; flex-wrap: wrap; align-items: center">
        <el-select v-model="filter.status" :placeholder="t('acc.filter.status')" clearable style="width: 150px" @change="load">
          <el-option v-for="s in ORDER_STATUS" :key="s" :label="statusLabel(s)" :value="s" />
        </el-select>
        <el-select v-model="filter.cat" :placeholder="t('acc.filter.cat')" clearable filterable style="width: 220px" @change="load">
          <el-option v-for="c in categories" :key="c.id" :label="catNameOf(c)" :value="c.id" />
        </el-select>
        <el-input v-model="filter.kw" :placeholder="t('acc.filter.kw')" clearable style="width: 220px" @input="load" />
        <el-button type="primary" @click="openCreate"><el-icon><Plus /></el-icon>&nbsp;{{ t('acc.create') }}</el-button>
      </div>

      <el-table :data="filtered" style="width: 100%; margin-top: 14px" border>
        <el-table-column prop="id" :label="t('acc.col.no')" width="110" />
        <el-table-column :label="t('acc.col.device')" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">{{ deviceLabelOf(row) }}</template>
        </el-table-column>
        <el-table-column :label="t('acc.col.cat')" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">{{ catName(row.categoryId) }}</template>
        </el-table-column>
        <el-table-column :label="t('acc.col.inspector')" width="100">
          <template #default="{ row }">{{ insName(row.inspectorId) }}</template>
        </el-table-column>
        <el-table-column :label="t('acc.col.progress')" width="140">
          <template #default="{ row }">
            <el-progress :percentage="progressPct(row)" :stroke-width="14"
                         :status="row.status === '整改中' ? 'exception' : row.status === '已通过' ? 'success' : undefined" />
          </template>
        </el-table-column>
        <el-table-column :label="t('common.status')" width="110">
          <template #default="{ row }">
            <el-tag :type="orderTag(row.status)" effect="dark">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('acc.col.updated')" width="165">
          <template #default="{ row }">{{ fmtTime(row.updatedAt) }}</template>
        </el-table-column>
        <el-table-column :label="t('common.actions')" width="100" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">{{ t('acc.btn.accept') }}</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 验收单详情 -->
    <el-drawer v-model="detailVisible" size="720px" :title="deviceLabelOf(detail || {}) + ' · ' + t('acc.drawer.order') + ' ' + (detail?.id || '')">
      <template v-if="detail">
        <el-descriptions :column="2" border size="small" style="margin-bottom: 14px">
          <el-descriptions-item :label="t('acc.col.cat')">{{ catName(detail.categoryId) }}</el-descriptions-item>
          <el-descriptions-item :label="t('common.status')">
            <el-tag :type="orderTag(detail.status)" effect="dark">{{ statusLabel(detail.status) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="t('acc.col.inspector')">
            <el-select v-model="detail.inspectorId" size="small" style="width: 160px" @change="saveInspector">
              <el-option v-for="i in installers" :key="i.id" :label="i.name + '（' + levelName(i.level) + '）'" :value="i.id" />
            </el-select>
          </el-descriptions-item>
          <el-descriptions-item :label="t('acc.drawer.created')">{{ fmtTime(detail.createdAt) }}</el-descriptions-item>
        </el-descriptions>

        <el-alert v-if="detail.status === '整改中'" type="error" :closable="false" style="margin-bottom: 12px"
                  :title="t('acc.reworkAlert')" />

        <div v-for="it in detail.items" :key="it.key" class="item-card">
          <div class="item-head">
            <span class="item-title">{{ itemTitleOf(it) }}</span>
            <el-tag size="small" :type="catTag(it.category)">{{ methodName(it.category) }}</el-tag>
            <el-tag v-if="it.mandatory" size="small" type="danger" effect="plain">{{ t('acc.mandatory') }}</el-tag>
            <el-tag size="small" :type="itemTag(it.status)" effect="dark">{{ statusLabel(it.status) }}</el-tag>
          </div>
          <div class="item-content">{{ itemContentOf(it) }}</div>
          <div v-if="it.result" class="item-result">
            <span class="result-label">{{ t('acc.record') }}</span>{{ it.result }}
            <span v-if="it.evidence" class="evidence">（{{ it.evidence }}）</span>
          </div>
          <div v-if="it.checkedBy || it.checkedAt" class="item-meta">
            {{ t('acc.by') }}{{ insName(it.checkedBy) }} · {{ fmtTime(it.checkedAt) }}
          </div>
          <div class="item-actions">
            <el-button size="small" type="primary" plain @click="openJudge(it)">{{ t('acc.judge') }}</el-button>
            <template v-if="it.category === 'burnin'">
              <el-button size="small" type="warning" plain @click="launchBurnin(it)">
                <el-icon><Monitor /></el-icon>&nbsp;{{ t('acc.startBurnin') }}
              </el-button>
              <el-button size="small" plain @click="openApply(it, 'burnin')">{{ t('acc.applyBurnin') }}</el-button>
            </template>
            <template v-if="it.category === 'http'">
              <el-button size="small" type="warning" plain @click="launchHttp(it)">
                <el-icon><DataLine /></el-icon>&nbsp;{{ t('acc.startHttp') }}
              </el-button>
              <el-button size="small" plain @click="openApply(it, 'http')">{{ t('acc.applyHttp') }}</el-button>
            </template>
          </div>
        </div>

        <div class="card-block" style="margin-top: 6px">
          <div class="block-title">{{ t('acc.conclusion') }}</div>
          <el-input v-model="detail.conclusion" type="textarea" :rows="2" :placeholder="t('acc.conclusionPh')" />
          <el-button type="success" style="margin-top: 10px" :disabled="detail.status !== '已通过'" @click="saveConclusion">
            {{ t('acc.saveConclusion') }}
          </el-button>
        </div>
      </template>
    </el-drawer>

    <!-- 判定对话框 -->
    <el-dialog v-model="judgeVisible" :title="t('acc.judgeDialog') + ' · ' + (judgeItem ? itemTitleOf(judgeItem) : '')" width="520px">
      <el-form label-width="110px">
        <el-form-item :label="t('common.verdict')">
          <el-radio-group v-model="judgeForm.status">
            <el-radio-button v-for="s in ITEM_STATUS_KEYS" :key="s.key" :value="s.key">{{ statusLabel(s.zh) }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('acc.judgeResult')" required>
          <el-input v-model="judgeForm.result" type="textarea" :rows="3"
                    :placeholder="judgeForm.status === 'fail' ? t('acc.judgeResultFail') : t('acc.judgeResultPass')" />
        </el-form-item>
        <el-form-item :label="t('acc.evidence')">
          <el-input v-model="judgeForm.evidence" :placeholder="t('acc.evidencePh')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="judgeVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="submitJudge">{{ t('common.submit') }}</el-button>
      </template>
    </el-dialog>

    <!-- 引用压测结果 -->
    <el-dialog v-model="applyVisible" :title="t('acc.applyDialog') + ' · ' + (applyItem ? itemTitleOf(applyItem) : '')" width="640px">
      <el-table :data="applyList" border size="small" max-height="360" highlight-current-row
                @current-change="(r) => (applyPick = r)">
        <el-table-column prop="id" :label="t('common.id')" width="90" />
        <el-table-column prop="name" :label="t('common.task')" min-width="180" show-overflow-tooltip />
        <el-table-column :label="t('common.verdict')" width="90">
          <template #default="{ row }">
            <el-tag :type="row.result?.pass ? 'success' : 'danger'" effect="dark">
              {{ row.result?.pass ? t('common.pass') : t('common.fail') }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('acc.applySummary')" min-width="200" show-overflow-tooltip>
          <template #default="{ row }">{{ applySummary(row) }}</template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="applyVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :disabled="!applyPick" @click="submitApply">{{ t('acc.applyBackfill') }}</el-button>
      </template>
    </el-dialog>

    <!-- 生成验收单 -->
    <el-dialog v-model="createVisible" :title="t('acc.createDialog')" width="520px">
      <el-form label-width="110px">
        <el-form-item :label="t('acc.createTask')">
          <el-select v-model="createForm.taskId" style="width: 100%" :placeholder="t('acc.createTaskPh')">
            <el-option v-for="tk in remainingTasks" :key="tk.id"
                       :label="`${tk.id} · ${catName(tk.categoryId)}${t('acc.createGen', { a: tk.accepted, b: tk.quantity, u: unitName(tk.unit) })}`"
                       :value="tk.id" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('acc.col.inspector')">
          <el-select v-model="createForm.inspectorId" style="width: 100%" clearable :placeholder="t('acc.inspectorPh')">
            <el-option v-for="i in installers" :key="i.id" :label="i.name + '（' + levelName(i.level) + '）· ' + i.title" :value="i.id" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :disabled="!createForm.taskId" @click="submitCreate">{{ t('common.create') }}</el-button>
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
import {
  t, ORDER_STATUS, ITEM_STATUS_KEYS, statusLabel, itemStatusKeyOf, itemStatusZhOf,
  catNameOf, levelName, unitName, methodName, itemTitle, itemContent, localizeDeviceLabel,
} from '../i18n'

const router = useRouter()
const orders = ref([])
const categories = ref([])
const installers = ref([])
const tasks = ref([])
const filter = reactive({ status: '', cat: '', kw: '' })

const detailVisible = ref(false)
const detail = ref(null)

const judgeVisible = ref(false)
const judgeItem = ref(null)
const judgeForm = reactive({ status: 'pass', result: '', evidence: '' })

const applyVisible = ref(false)
const applyItem = ref(null)
const applyType = ref('http')
const applyList = ref([])
const applyPick = ref(null)

const createVisible = ref(false)
const createForm = reactive({ taskId: '', inspectorId: '' })

const insMap = computed(() => Object.fromEntries(installers.value.map((i) => [i.id, i.name])))
const insName = (id) => (id && insMap.value[id]) || '-'
const catById = (id) => categories.value.find((c) => c.id === id)
const catName = (id) => {
  const c = catById(id)
  return c ? catNameOf(c) : ''
}
const kindOf = (o) => catById(o.categoryId)?.kind || 'gpu'
const deviceLabelOf = (o) => localizeDeviceLabel(o.deviceLabel, categories.value)
const itemTitleOf = (it) => itemTitle(kindOf(detail.value || { categoryId: '' }), it.key, it.title)
const itemContentOf = (it) => itemContent(kindOf(detail.value || { categoryId: '' }), it.key, it.content)

const filtered = computed(() =>
  orders.value.filter(
    (o) =>
      (!filter.status || o.status === filter.status) &&
      (!filter.cat || o.categoryId === filter.cat) &&
      (!filter.kw || o.id.includes(filter.kw) || o.deviceLabel.includes(filter.kw)),
  ),
)

const remainingTasks = computed(() => tasks.value.filter((tk) => tk.accepted < tk.quantity))

function orderTag(s) {
  return { 待验收: 'info', 进行中: 'warning', 已通过: 'success', 整改中: 'danger' }[s] || 'info'
}
function itemTag(s) {
  return { 待检: 'info', 进行中: 'warning', 合格: 'success', 不合格: 'danger', 不适用: '' }[s] || 'info'
}
function catTag(c) {
  return { manual: '', burnin: 'warning', http: 'warning', performance: 'danger' }[c] || 'info'
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
  judgeForm.status = itemStatusKeyOf(it.status)
  judgeForm.result = it.result || ''
  judgeForm.evidence = it.evidence || ''
  judgeVisible.value = true
}

async function submitJudge() {
  const zh = itemStatusZhOf(judgeForm.status)
  if (['合格', '不合格'].includes(zh) && !judgeForm.result.trim()) {
    ElMessage.warning(t('acc.msg.resultRequired'))
    return
  }
  const o = await put(`/orders/${detail.value.id}/items/${judgeItem.value.key}`, {
    status: zh,
    result: judgeForm.result,
    evidence: judgeForm.evidence,
    checkedBy: detail.value.inspectorId || '',
  })
  detail.value = o
  judgeVisible.value = false
  ElMessage.success(`${t('acc.msg.itemSaved')}${statusLabel(zh)}`)
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
    .filter((tk) => tk.status === 'finished' && tk.result)
  applyVisible.value = true
}

function applySummary(row) {
  if (applyType.value === 'http') {
    const r = row.result
    return `Conc ${row.concurrency} · RPS ${r.rps} · P95 ${r.p95}ms · Err ${r.errRatePct}%`
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
  ElMessage.success(t('acc.msg.backfilled'))
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
  ElMessage.success(`${t('acc.msg.orderCreated')} ${o.id}：${localizeDeviceLabel(o.deviceLabel, categories.value)}`)
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
  ElMessage.success(t('acc.msg.conclusionSaved'))
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
