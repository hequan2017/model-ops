<template>
  <div>
    <div class="page-title">安装人员</div>
    <div class="page-desc">
      管理安装人员资质与技能标签，并对照各节点类别的「安装人员要求」做人岗匹配；分派部署任务时可参考匹配度。
    </div>

    <!-- 岗位要求原文 -->
    <div class="card-block">
      <div class="block-title">安装人员要求（岗位规范）</div>
      <div v-for="kind in ['gpu', 'cpu']" :key="kind" style="margin-bottom: 10px">
        <el-alert type="info" :closable="false">
          <template #title>
            <b>{{ kind === 'gpu' ? 'GPU 类节点（科研微调 / 推理 / 教学 / 智能体 / 实训集群）' : 'CPU 仿真节点' }}</b>
          </template>
          {{ reqText(kind) }}
        </el-alert>
      </div>
    </div>

    <!-- 人员名册 -->
    <div class="card-block">
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px">
        <div class="block-title" style="margin: 0">人员名册（{{ installers.length }} 人）</div>
        <el-button type="primary" @click="openEdit()"><el-icon><Plus /></el-icon>&nbsp;新增人员</el-button>
      </div>
      <el-table :data="installers" border>
        <el-table-column prop="name" label="姓名" width="90" />
        <el-table-column prop="title" label="岗位" min-width="150" show-overflow-tooltip />
        <el-table-column label="等级" width="90">
          <template #default="{ row }">
            <el-tag :type="levelTag(row.level)" effect="dark">{{ row.level }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="phone" label="联系方式" width="140" />
        <el-table-column label="技能标签" min-width="320">
          <template #default="{ row }">
            <el-tag v-for="s in row.skills" :key="s" size="small" style="margin: 2px 4px 2px 0"
                    :type="isGpuSkill(s) ? 'primary' : 'warning'" effect="plain">{{ s }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="在岗" width="70">
          <template #default="{ row }">
            <el-tag :type="row.available ? 'success' : 'info'" size="small">{{ row.available ? '在岗' : '占用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="130" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-popconfirm :title="`删除 ${row.name}？`" @confirm="removeInstaller(row)">
              <template #reference><el-button link type="danger">删除</el-button></template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 技能矩阵与人岗匹配 -->
    <div class="card-block">
      <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px">
        <div class="block-title" style="margin: 0">技能矩阵 · 人岗匹配</div>
        <el-select v-model="matchCat" style="width: 260px" @change="onMatchCatChange">
          <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
        </el-select>
      </div>

      <el-table v-if="curCat" :data="curCat.requiredSkills.map((s) => ({ skill: s }))" border size="small">
        <el-table-column prop="skill" label="要求技能" min-width="180" />
        <el-table-column v-for="i in installers" :key="i.id" :label="i.name" :width="76" align="center">
          <template #default="{ row }">
            <el-icon v-if="i.skills.includes(row.skill)" color="#67c23a"><CircleCheckFilled /></el-icon>
            <el-icon v-else color="#dcdfe6"><CircleClose /></el-icon>
          </template>
        </el-table-column>
      </el-table>

      <div v-if="curCat" style="margin-top: 14px">
        <div class="block-title">对「{{ curCat.name }}」的匹配度</div>
        <el-table :data="matchRows" border size="small">
          <el-table-column prop="name" label="人员" width="100" />
          <el-table-column label="匹配度" min-width="240">
            <template #default="{ row }">
              <el-progress :percentage="row.pct" :stroke-width="14"
                           :status="row.pct === 100 ? 'success' : row.pct >= 60 ? undefined : 'exception'" />
            </template>
          </el-table-column>
          <el-table-column label="缺失技能" min-width="260">
            <template #default="{ row }">
              <template v-if="row.missing.length">
                <el-tag v-for="m in row.missing" :key="m" size="small" type="danger" effect="plain"
                        style="margin: 2px 4px 2px 0">{{ m }}</el-tag>
              </template>
              <el-tag v-else size="small" type="success">全部满足</el-tag>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>

    <!-- 编辑人员 -->
    <el-dialog v-model="editVisible" :title="form.id ? '编辑人员' : '新增人员'" width="560px">
      <el-form label-width="90px">
        <el-form-item label="姓名" required><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="岗位"><el-input v-model="form.title" placeholder="如：GPU集群部署专家" /></el-form-item>
        <el-form-item label="等级" required>
          <el-select v-model="form.level" style="width: 160px">
            <el-option v-for="l in ['专家级', '高级', '中级', '初级']" :key="l" :label="l" :value="l" />
          </el-select>
        </el-form-item>
        <el-form-item label="联系方式"><el-input v-model="form.phone" /></el-form-item>
        <el-form-item label="GPU 技能">
          <el-checkbox-group v-model="form.skills">
            <el-checkbox v-for="s in skills.gpu" :key="s" :value="s" style="width: 100%">{{ s }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="HPC 技能">
          <el-checkbox-group v-model="form.skills">
            <el-checkbox v-for="s in skills.cpu" :key="s" :value="s" style="width: 100%">{{ s }}</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="在岗">
          <el-switch v-model="form.available" active-text="在岗" inactive-text="占用" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { get, post, put, del } from '../api'

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
function reqText(kind) {
  const c = categories.value.find((x) => x.kind === kind)
  return c ? c.installerReq : ''
}

async function load() {
  const [is, cs, ss] = await Promise.all([get('/installers'), get('/categories'), get('/skills')])
  installers.value = is || []
  categories.value = cs || []
  skills.value = ss || { gpu: [], cpu: [] }
}

function onMatchCatChange() { /* 匹配表为响应式计算，无需处理 */ }

function openEdit(row) {
  Object.assign(form, row ? { ...row } : { id: '', name: '', title: '', level: '中级', phone: '', skills: [], available: true })
  editVisible.value = true
}

async function save() {
  if (!form.name.trim()) {
    ElMessage.warning('姓名不能为空')
    return
  }
  if (form.id) await put(`/installers/${form.id}`, { ...form })
  else await post('/installers', { ...form })
  editVisible.value = false
  ElMessage.success('已保存')
  load()
}

async function removeInstaller(row) {
  await del(`/installers/${row.id}`)
  ElMessage.success('已删除')
  load()
}

onMounted(load)
</script>
