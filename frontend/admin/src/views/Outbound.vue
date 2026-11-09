<template>
  <div>
    <h3>出库开单 / 锁定·签收</h3>
    <el-form inline>
      <el-form-item label="仓库">
        <el-select v-model="warehouseId" style="width:140px">
          <el-option v-for="w in warehouses" :key="w.ID" :label="w.Name || w.ID" :value="w.ID" />
        </el-select>
      </el-form-item>
      <el-form-item label="物资ID"><el-input v-model="materialId" style="width:100px" /></el-form-item>
      <el-form-item label="数量"><el-input-number v-model="qty" :min="1" /></el-form-item>
      <el-button type="primary" @click="createOrder">开单</el-button>
      <el-button @click="load">刷新</el-button>
    </el-form>
    <el-table :data="orders" border style="margin-top:12px">
      <el-table-column prop="ID" label="ID" width="80" />
      <el-table-column prop="OrderNo" label="单号" />
      <el-table-column prop="Status" label="状态" width="120" />
      <el-table-column label="操作" width="280">
        <template #default="{ row }">
          <el-button size="small" @click="confirm(row.ID, 'lock')">锁定码</el-button>
          <el-button size="small" type="success" @click="confirm(row.ID, 'receive')">签收</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import { ElMessageBox } from 'element-plus'
import http from '../api'
import { useWarehouses } from '../composables/warehouses'
const { warehouses, loadWarehouses, defaultWarehouseId } = useWarehouses()
const orders = ref([])
const warehouseId = ref('')
const materialId = ref('')
const qty = ref(1)
async function load() { orders.value = (await http.get('/outbound/orders')).list || [] }
async function createOrder() {
  await http.post('/outbound/orders', { warehouseId: warehouseId.value, materialId: Number(materialId.value), qty: qty.value })
  load()
}
async function confirm(id, action) {
  const { value } = await ElMessageBox.prompt('唯一码', action === 'lock' ? '出库锁定' : '签收')
  await http.post(`/outbound/orders/${id}/confirm`, { uniqueCode: value, action, clientRequestId: crypto.randomUUID() })
  load()
}
onMounted(async () => {
  await loadWarehouses()
  warehouseId.value = defaultWarehouseId()
  load()
})
</script>
