<template>
  <div>
    <div class="bar"><h3>人员管理</h3><el-button type="primary" @click="openCreate">新建</el-button></div>
    <el-table :data="list" border>
      <el-table-column prop="Username" label="账号" />
      <el-table-column prop="DisplayName" label="姓名" />
      <el-table-column prop="Mobile" label="手机" width="120" />
      <el-table-column prop="Dept" label="部门" width="120" />
      <el-table-column prop="Status" label="状态" width="80" />
      <el-table-column label="角色"><template #default="{ row }">{{ (row.roleCodes||[]).join(',') }}</template></el-table-column>
      <el-table-column label="操作" width="100">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-dialog v-model="visible" :title="editing ? '编辑用户' : '新建用户'" width="520px">
      <el-form label-width="100px">
        <el-form-item v-if="!editing" label="账号"><el-input v-model="form.username" /></el-form-item>
        <el-form-item :label="editing ? '重置密码' : '密码'">
          <el-input v-model="form.password" type="password" :placeholder="editing ? '留空则不修改' : ''" />
        </el-form-item>
        <el-form-item label="姓名"><el-input v-model="form.displayName" /></el-form-item>
        <el-form-item label="手机"><el-input v-model="form.mobile" /></el-form-item>
        <el-form-item label="部门"><el-input v-model="form.dept" /></el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">停用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="强制改密"><el-switch v-model="form.mustChangePassword" /></el-form-item>
        <el-form-item label="角色"><el-select v-model="form.roleCodes" multiple style="width:100%"><el-option v-for="r in roles" :key="r.Code" :label="r.Name" :value="r.Code" /></el-select></el-form-item>
        <el-form-item label="端"><el-select v-model="form.clients" multiple style="width:100%"><el-option value="admin" /><el-option value="portal" /><el-option value="app" /></el-select></el-form-item>
        <el-form-item label="仓范围">
          <el-select v-model="form.warehouseIds" multiple style="width:100%">
            <el-option v-for="w in warehouses" :key="w.ID" :label="w.Name || w.ID" :value="w.ID" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer><el-button type="primary" @click="save">保存</el-button></template>
    </el-dialog>
  </div>
</template>
<script setup>
import { onMounted, reactive, ref } from 'vue'
import http from '../api'
import { useWarehouses } from '../composables/warehouses'
const { warehouses, loadWarehouses, defaultWarehouseId } = useWarehouses()
const list = ref([])
const roles = ref([])
const visible = ref(false)
const editing = ref(false)
const editId = ref(0)
const form = reactive({
  username: '', password: '', displayName: '', mobile: '', dept: '',
  status: 1, mustChangePassword: true, roleCodes: ['admin'], clients: ['admin', 'portal', 'app'], warehouseIds: []
})
async function load() {
  list.value = (await http.get('/admin/users')).list || []
  roles.value = (await http.get('/admin/roles')).list || []
}
function resetForm() {
  Object.assign(form, {
    username: '', password: '', displayName: '', mobile: '', dept: '',
    status: 1, mustChangePassword: true, roleCodes: ['admin'], clients: ['admin', 'portal', 'app'],
    warehouseIds: [defaultWarehouseId()].filter(Boolean)
  })
}
function openCreate() {
  editing.value = false
  resetForm()
  visible.value = true
}
function openEdit(row) {
  editing.value = true
  editId.value = row.ID
  Object.assign(form, {
    username: row.Username, password: '', displayName: row.DisplayName || '',
    mobile: row.Mobile || '', dept: row.Dept || '', status: row.Status ?? 1,
    mustChangePassword: !!row.MustChangePassword,
    roleCodes: [...(row.roleCodes || [])],
    clients: [...(row.clients || ['admin'])],
    warehouseIds: [...(row.warehouseIds || [])]
  })
  visible.value = true
}
async function save() {
  const body = {
    displayName: form.displayName,
    mobile: form.mobile,
    dept: form.dept,
    status: form.status,
    mustChangePassword: form.mustChangePassword,
    roleCodes: form.roleCodes,
    clients: form.clients,
    warehouseIds: form.warehouseIds
  }
  if (form.password) body.password = form.password
  if (editing.value) {
    await http.put(`/admin/users/${editId.value}`, body)
  } else {
    await http.post('/admin/users', { ...body, username: form.username, password: form.password })
  }
  visible.value = false
  load()
}
onMounted(async () => {
  await loadWarehouses()
  resetForm()
  load()
})
</script>
<style scoped>.bar{display:flex;justify-content:space-between;align-items:center}</style>
