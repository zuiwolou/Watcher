<template>
  <div>
    <h3>审批中心</h3>
    <el-tabs v-model="tab">
      <el-tab-pane label="待办" name="todo">
        <el-form inline>
          <el-form-item label="仓库">
            <el-select v-model="warehouseId" style="width:140px">
              <el-option v-for="w in warehouses" :key="w.ID" :label="w.Name || w.ID" :value="w.ID" />
            </el-select>
          </el-form-item>
          <el-form-item label="物资ID"><el-input v-model="materialId" style="width:100px" /></el-form-item>
          <el-form-item label="调整量"><el-input-number v-model="delta" /></el-form-item>
          <el-form-item label="原因"><el-input v-model="reason" style="width:160px" /></el-form-item>
          <el-button type="primary" @click="adjust">提交调整</el-button>
          <el-button @click="loadTodo">刷新待办</el-button>
        </el-form>
        <el-table :data="todo" border>
          <el-table-column prop="ID" label="审批单" width="90" />
          <el-table-column prop="BizType" label="类型" />
          <el-table-column prop="BizID" label="业务ID" width="100" />
          <el-table-column prop="CurrentLevel" label="当前级" width="90" />
          <el-table-column prop="Status" label="状态" width="100" />
          <el-table-column label="操作" width="260">
            <template #default="{ row }">
              <el-button size="small" @click="openDetail(row.ID)">进度</el-button>
              <el-button size="small" type="success" @click="act(row.ID,'pass')">通过</el-button>
              <el-button size="small" type="danger" @click="act(row.ID,'reject')">驳回</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="审批模板" name="tpl">
        <el-form label-width="100px" style="max-width:760px">
          <el-form-item label="业务类型">
            <el-select v-model="tpl.bizType" style="width:220px">
              <el-option label="库存调整 stock_adjust" value="stock_adjust" />
            </el-select>
          </el-form-item>
          <el-form-item label="模板名称"><el-input v-model="tpl.name" /></el-form-item>
          <el-form-item label="启用"><el-switch v-model="tpl.enabled" /></el-form-item>
          <el-form-item label="审批层级">
            <div v-for="(lv, idx) in tpl.levels" :key="idx" class="level-row">
              <el-tag>L{{ lv.level }}</el-tag>
              <el-select v-model="lv.roleCodes" multiple filterable allow-create default-first-option
                placeholder="角色编码" style="width:280px;margin:0 8px">
                <el-option v-for="r in roleOptions" :key="r" :label="r" :value="r" />
              </el-select>
              <el-select v-model="lv.strategy" style="width:100px">
                <el-option label="任一" value="any" />
                <el-option label="全部" value="all" />
              </el-select>
              <el-button link type="danger" @click="tpl.levels.splice(idx,1)">删除</el-button>
            </div>
            <el-button size="small" @click="addLevel">加一级</el-button>
          </el-form-item>
          <el-button type="primary" @click="saveTpl">保存模板</el-button>
          <el-button @click="loadTpl">刷新</el-button>
        </el-form>
        <el-table :data="templates" border style="margin-top:12px" @row-click="editTpl">
          <el-table-column prop="BizType" label="业务" width="140" />
          <el-table-column prop="Name" label="名称" />
          <el-table-column prop="Enabled" label="启用" width="80" />
          <el-table-column label="层级摘要">
            <template #default="{ row }">{{ summarizeLevels(row.LevelsJSON) }}</template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="审批进度" name="progress">
        <el-form inline>
          <el-form-item label="状态">
            <el-select v-model="orderStatus" clearable style="width:130px" @change="loadOrders">
              <el-option label="pending" value="pending" />
              <el-option label="approved" value="approved" />
              <el-option label="rejected" value="rejected" />
            </el-select>
          </el-form-item>
          <el-button @click="loadOrders">刷新</el-button>
        </el-form>
        <el-table :data="orders" border>
          <el-table-column prop="ID" label="单号" width="80" />
          <el-table-column prop="BizType" label="类型" width="120" />
          <el-table-column prop="BizID" label="业务ID" width="100" />
          <el-table-column prop="CurrentLevel" label="当前级" width="90" />
          <el-table-column prop="Status" label="状态" width="110" />
          <el-table-column prop="ApplicantID" label="申请人" width="90" />
          <el-table-column prop="CreatedAt" label="创建时间" width="180" />
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button link type="primary" @click="openDetail(row.ID)">查看</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>

    <el-drawer v-model="detailVisible" title="审批进度" size="420px">
      <template v-if="detail">
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="单号">{{ detail.order?.ID }}</el-descriptions-item>
          <el-descriptions-item label="类型">{{ detail.order?.BizType }}</el-descriptions-item>
          <el-descriptions-item label="业务ID">{{ detail.order?.BizID }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ detail.order?.Status }}</el-descriptions-item>
          <el-descriptions-item label="当前级">{{ detail.order?.CurrentLevel }}</el-descriptions-item>
        </el-descriptions>
        <h4 style="margin:16px 0 8px">流程</h4>
        <el-steps direction="vertical" :active="stepActive" finish-status="success">
          <el-step
            v-for="lv in (detail.levels || [])"
            :key="lv.level"
            :title="`第 ${lv.level} 级`"
            :description="levelDesc(lv)"
            :status="stepStatus(lv.level)"
          />
        </el-steps>
        <h4 style="margin:16px 0 8px">表态记录</h4>
        <el-timeline>
          <el-timeline-item
            v-for="a in (detail.actions || [])"
            :key="a.ID"
            :type="a.Action === 'pass' ? 'success' : 'danger'"
            :timestamp="a.CreatedAt"
          >
            L{{ a.Level }} · 用户{{ a.ActorID }} · {{ a.Action }}
            <div v-if="a.Comment" class="cmt">{{ a.Comment }}</div>
          </el-timeline-item>
        </el-timeline>
      </template>
    </el-drawer>
  </div>
</template>
<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api'
import { useWarehouses } from '../composables/warehouses'

const { warehouses, loadWarehouses, defaultWarehouseId } = useWarehouses()
const tab = ref('todo')
const todo = ref([])
const warehouseId = ref('')
const materialId = ref('')
const delta = ref(1)
const reason = ref('')
const templates = ref([])
const roleOptions = ref(['admin'])
const tpl = reactive({
  bizType: 'stock_adjust',
  name: '库存调整默认两级',
  enabled: true,
  levels: [
    { level: 1, roleCodes: ['admin'], strategy: 'any' },
    { level: 2, roleCodes: ['admin'], strategy: 'any' }
  ]
})
const orders = ref([])
const orderStatus = ref('')
const detailVisible = ref(false)
const detail = ref(null)

const stepActive = computed(() => {
  if (!detail.value?.order) return 0
  const st = detail.value.order.Status
  const cur = detail.value.order.CurrentLevel || 1
  if (st === 'approved') return (detail.value.levels || []).length
  if (st === 'rejected') return Math.max(cur - 1, 0)
  return Math.max(cur - 1, 0)
})

function stepStatus(level) {
  const o = detail.value?.order
  if (!o) return 'wait'
  const acts = (detail.value.actions || []).filter((a) => a.Level === level)
  if (acts.some((a) => a.Action === 'reject')) return 'error'
  if (o.Status === 'approved' || (o.CurrentLevel > level) || acts.some((a) => a.Action === 'pass')) return 'success'
  if (o.Status === 'pending' && o.CurrentLevel === level) return 'process'
  return 'wait'
}

function levelDesc(lv) {
  const roles = (lv.roleCodes || []).join(',') || '-'
  return `角色 ${roles} · 策略 ${lv.strategy || 'any'}`
}

function summarizeLevels(json) {
  try {
    const arr = JSON.parse(json || '[]')
    return arr.map((x) => `L${x.level}:${(x.roleCodes || []).join('|')}/${x.strategy || 'any'}`).join(' → ')
  } catch { return json }
}

async function loadTodo() {
  const data = await http.get('/approvals/todo', { params: { page: 1, pageSize: 50 } })
  todo.value = data?.list || []
}

async function adjust() {
  if (!reason.value.trim()) { ElMessage.warning('请填写原因'); return }
  await http.post('/stocks/adjustments', {
    warehouseId: warehouseId.value, materialId: Number(materialId.value), delta: delta.value, reason: reason.value
  })
  ElMessage.success('已提交，进入审批')
  loadTodo()
}

async function act(id, action) {
  let comment = action
  if (action === 'reject') {
    const { value } = await ElMessageBox.prompt('请填写驳回原因', '驳回', { inputPattern: /.+/, inputErrorMessage: '必填' })
    comment = value
  }
  await http.post(`/approvals/${id}/act`, { action, comment })
  ElMessage.success(action === 'pass' ? '已通过' : '已驳回')
  loadTodo()
  if (tab.value === 'progress') loadOrders()
}

function addLevel() {
  const n = tpl.levels.length + 1
  tpl.levels.push({ level: n, roleCodes: ['admin'], strategy: 'any' })
}

async function loadTpl() {
  templates.value = (await http.get('/approvals/templates')) || []
  try {
    const roles = (await http.get('/admin/roles'))?.list || []
    roleOptions.value = roles.map((r) => r.Code).filter(Boolean)
    if (!roleOptions.value.includes('admin')) roleOptions.value.unshift('admin')
  } catch { /* role:view may miss */ }
  const hit = templates.value.find((t) => t.BizType === tpl.bizType) || templates.value[0]
  if (hit) editTpl(hit)
}

function editTpl(row) {
  tpl.bizType = row.BizType
  tpl.name = row.Name
  tpl.enabled = !!row.Enabled
  try {
    const levels = JSON.parse(row.LevelsJSON || '[]')
    tpl.levels = levels.length ? levels.map((x, i) => ({
      level: x.level || i + 1,
      roleCodes: x.roleCodes || [],
      strategy: x.strategy || 'any'
    })) : [{ level: 1, roleCodes: ['admin'], strategy: 'any' }]
  } catch {
    tpl.levels = [{ level: 1, roleCodes: ['admin'], strategy: 'any' }]
  }
}

async function saveTpl() {
  tpl.levels.forEach((lv, i) => { lv.level = i + 1 })
  await http.put('/approvals/templates', {
    bizType: tpl.bizType,
    name: tpl.name,
    enabled: tpl.enabled,
    levels: tpl.levels
  })
  ElMessage.success('模板已保存')
  loadTpl()
}

async function loadOrders() {
  const params = { page: 1, pageSize: 50 }
  if (orderStatus.value) params.status = orderStatus.value
  const data = await http.get('/approvals/orders', { params })
  orders.value = data?.list || []
}

async function openDetail(id) {
  detail.value = await http.get(`/approvals/orders/${id}`)
  detailVisible.value = true
}

watch(tab, (v) => {
  if (v === 'todo') loadTodo()
  if (v === 'tpl') loadTpl()
  if (v === 'progress') loadOrders()
})

onMounted(async () => {
  await loadWarehouses()
  warehouseId.value = defaultWarehouseId()
  loadTodo()
})
</script>
<style scoped>
.level-row { display:flex; align-items:center; margin-bottom:8px; flex-wrap:wrap; gap:4px; }
.cmt { color:#627d98; font-size:12px; margin-top:4px; }
</style>
