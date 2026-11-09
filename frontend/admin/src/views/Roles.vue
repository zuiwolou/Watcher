<template>
  <div>
    <div class="bar">
      <h3>角色权限矩阵</h3>
      <div>
        <el-button @click="load">刷新</el-button>
        <el-button type="primary" :disabled="!dirty" :loading="saving" @click="saveAll">保存变更</el-button>
        <el-button @click="showCreate = true">新建角色</el-button>
      </div>
    </div>
    <el-alert type="info" :closable="false" style="margin-bottom:12px"
      title="行=权限点，列=角色；勾选后点「保存变更」。admin 角色请谨慎调整。" />
    <div class="matrix-wrap">
      <el-table :data="rows" border height="620" size="small" fixed>
        <el-table-column prop="perm" label="权限点" width="180" fixed />
        <el-table-column prop="group" label="模块" width="100" fixed />
        <el-table-column v-for="r in roles" :key="r.ID" :label="r.Name || r.Code" min-width="110" align="center">
          <template #header>
            <div class="col-h">
              <div>{{ r.Name || r.Code }}</div>
              <div class="code">{{ r.Code }}</div>
            </div>
          </template>
          <template #default="{ row }">
            <el-checkbox
              :model-value="!!matrix[r.Code]?.[row.perm]"
              @change="(v) => toggle(r.Code, row.perm, v)"
            />
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="showCreate" title="新建角色" width="420px">
      <el-form label-width="80px">
        <el-form-item label="编码"><el-input v-model="createForm.code" placeholder="如 warehouse_mgr" /></el-form-item>
        <el-form-item label="名称"><el-input v-model="createForm.name" /></el-form-item>
        <el-form-item label="备注"><el-input v-model="createForm.remark" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showCreate = false">取消</el-button>
        <el-button type="primary" @click="createRole">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>
<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api'

const roles = ref([])
const catalog = ref([])
const matrix = reactive({})
const baseline = reactive({})
const dirty = ref(false)
const saving = ref(false)
const showCreate = ref(false)
const createForm = reactive({ code: '', name: '', remark: '' })

const rows = computed(() => catalog.value.map((p) => ({
  perm: p,
  group: (p.split(':')[0] || '')
})))

function snapshot() {
  Object.keys(baseline).forEach((k) => delete baseline[k])
  for (const code of Object.keys(matrix)) {
    baseline[code] = { ...matrix[code] }
  }
  dirty.value = false
}

function toggle(roleCode, perm, checked) {
  if (!matrix[roleCode]) matrix[roleCode] = {}
  matrix[roleCode][perm] = !!checked
  dirty.value = true
}

async function load() {
  const [permBody, roleBody] = await Promise.all([
    http.get('/admin/permissions'),
    http.get('/admin/roles')
  ])
  catalog.value = permBody?.list || []
  roles.value = roleBody?.list || []
  Object.keys(matrix).forEach((k) => delete matrix[k])
  for (const r of roles.value) {
    matrix[r.Code] = {}
    for (const p of (r.permissions || [])) matrix[r.Code][p] = true
  }
  snapshot()
}

async function saveAll() {
  saving.value = true
  try {
    for (const r of roles.value) {
      const perms = Object.keys(matrix[r.Code] || {}).filter((p) => matrix[r.Code][p])
      await http.put(`/admin/roles/${r.ID}`, {
        name: r.Name,
        remark: r.Remark,
        permissions: perms
      })
    }
    ElMessage.success('权限矩阵已保存')
    await load()
  } finally {
    saving.value = false
  }
}

async function createRole() {
  if (!createForm.code.trim()) {
    ElMessage.warning('请填写角色编码')
    return
  }
  await http.post('/admin/roles', {
    code: createForm.code.trim(),
    name: createForm.name || createForm.code,
    remark: createForm.remark,
    permissions: []
  })
  showCreate.value = false
  createForm.code = ''
  createForm.name = ''
  createForm.remark = ''
  ElMessage.success('角色已创建')
  await load()
}

onMounted(load)
</script>
<style scoped>
.bar { display:flex; justify-content:space-between; align-items:center; margin-bottom:8px; }
.matrix-wrap { overflow:auto; }
.col-h { line-height:1.2; }
.col-h .code { font-size:11px; color:#888; font-weight:400; }
</style>
