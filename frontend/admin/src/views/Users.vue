<template>
  <div>
    <div class="bar"><h3>人员管理</h3><el-button type="primary" @click="openCreate">新建</el-button></div>
    <el-table :data="list" border>
      <el-table-column prop="Username" label="账号" />
      <el-table-column prop="DisplayName" label="姓名" />
      <el-table-column prop="Status" label="状态" />
      <el-table-column label="角色"><template #default="{ row }">{{ (row.roleCodes||[]).join(',') }}</template></el-table-column>
    </el-table>
    <el-dialog v-model="visible" title="新建用户">
      <el-form label-width="90px">
        <el-form-item label="账号"><el-input v-model="form.username" /></el-form-item>
        <el-form-item label="密码"><el-input v-model="form.password" type="password" /></el-form-item>
        <el-form-item label="姓名"><el-input v-model="form.displayName" /></el-form-item>
        <el-form-item label="角色"><el-select v-model="form.roleCodes" multiple style="width:100%"><el-option v-for="r in roles" :key="r.Code" :label="r.Name" :value="r.Code" /></el-select></el-form-item>
        <el-form-item label="端"><el-select v-model="form.clients" multiple style="width:100%"><el-option value="admin" /><el-option value="portal" /><el-option value="app" /></el-select></el-form-item>
        <el-form-item label="仓范围">
          <el-select v-model="form.warehouseIds" multiple style="width:100%">
            <el-option v-for="w in warehouses" :key="w.ID" :label="w.Name || w.ID" :value="w.ID" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer><el-button type="primary" @click="create">保存</el-button></template>
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
const form = reactive({ username: '', password: '', displayName: '', roleCodes: ['admin'], clients: ['admin', 'portal', 'app'], warehouseIds: [] })
async function load() {
  list.value = (await http.get('/admin/users')).list || []
  roles.value = (await http.get('/admin/roles')).list || []
}
function openCreate() { visible.value = true }
async function create() {
  await http.post('/admin/users', form)
  visible.value = false
  load()
}
onMounted(async () => {
  await loadWarehouses()
  form.warehouseIds = [defaultWarehouseId()].filter(Boolean)
  load()
})
</script>
<style scoped>.bar{display:flex;justify-content:space-between;align-items:center}</style>
