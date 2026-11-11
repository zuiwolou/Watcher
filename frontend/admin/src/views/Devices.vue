<template>
  <div>
    <div class="bar">
      <h3>设备台账</h3>
      <el-button type="primary" @click="openCreate">新建设备</el-button>
    </div>
    <el-table :data="list" border>
      <el-table-column prop="ID" label="ID" width="70" />
      <el-table-column prop="DeviceCode" label="设备编码" />
      <el-table-column prop="Name" label="名称" />
      <el-table-column prop="Type" label="类型" width="100" />
      <el-table-column prop="WarehouseID" label="仓" width="100" />
      <el-table-column prop="SchemaProfileID" label="Schema" width="90" />
      <el-table-column prop="Status" label="状态" width="90" />
      <el-table-column label="操作" width="160">
        <template #default="{ row }">
          <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
          <el-button link @click="openBind(row)">绑定Schema</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="visible" :title="editing ? '编辑设备' : '新建设备'" width="480px">
      <el-form label-width="110px">
        <el-form-item v-if="!editing" label="设备编码"><el-input v-model="form.deviceCode" /></el-form-item>
        <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" style="width:100%">
            <el-option value="warehouse" label="仓内" />
            <el-option value="vehicle" label="车载" />
            <el-option value="handheld" label="手持" />
          </el-select>
        </el-form-item>
        <el-form-item label="仓">
          <el-select v-model="form.warehouseId" style="width:100%">
            <el-option v-for="w in warehouses" :key="w.ID" :label="w.Name || w.ID" :value="w.ID" />
          </el-select>
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="form.status" style="width:100%">
            <el-option value="active" /><el-option value="disabled" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="bindVisible" title="绑定 MQTT Schema" width="420px">
      <el-form label-width="110px">
        <el-form-item label="Schema">
          <el-select v-model="bindSchemaId" style="width:100%" clearable>
            <el-option v-for="s in schemas" :key="s.ID" :label="`${s.Code} / ${s.Name}`" :value="s.ID" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button type="primary" @click="saveBind">绑定</el-button>
      </template>
    </el-dialog>
  </div>
</template>
<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api'
import { useWarehouses } from '../composables/warehouses'

const { warehouses, loadWarehouses, defaultWarehouseId } = useWarehouses()
const list = ref([])
const schemas = ref([])
const visible = ref(false)
const bindVisible = ref(false)
const editing = ref(false)
const editId = ref(0)
const bindId = ref(0)
const bindSchemaId = ref(null)
const form = reactive({
  deviceCode: '', name: '', type: 'warehouse', warehouseId: '', status: 'active'
})

async function load() {
  list.value = (await http.get('/devices')) || []
  if (!Array.isArray(list.value)) list.value = list.value.list || []
  schemas.value = (await http.get('/mqtt/schemas')) || []
  if (!Array.isArray(schemas.value)) schemas.value = schemas.value.list || schemas.value || []
}
function openCreate() {
  editing.value = false
  Object.assign(form, { deviceCode: '', name: '', type: 'warehouse', warehouseId: defaultWarehouseId(), status: 'active' })
  visible.value = true
}
function openEdit(row) {
  editing.value = true
  editId.value = row.ID
  Object.assign(form, {
    deviceCode: row.DeviceCode, name: row.Name, type: row.Type || 'warehouse',
    warehouseId: row.WarehouseID, status: row.Status || 'active'
  })
  visible.value = true
}
async function save() {
  if (editing.value) {
    await http.put(`/devices/${editId.value}`, {
      name: form.name, type: form.type, warehouseId: form.warehouseId, status: form.status
    })
  } else {
    if (!form.deviceCode) { ElMessage.warning('请填写设备编码'); return }
    await http.post('/devices', { ...form })
  }
  visible.value = false
  load()
}
function openBind(row) {
  bindId.value = row.ID
  bindSchemaId.value = row.SchemaProfileID || null
  bindVisible.value = true
}
async function saveBind() {
  await http.put(`/devices/${bindId.value}/mqtt-binding`, { schemaProfileId: bindSchemaId.value || 0 })
  bindVisible.value = false
  load()
}
onMounted(async () => {
  await loadWarehouses()
  form.warehouseId = defaultWarehouseId()
  load()
})
</script>
<style scoped>.bar{display:flex;justify-content:space-between;align-items:center;margin-bottom:12px}</style>
