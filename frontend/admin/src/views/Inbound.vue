<template>
  <div>
    <h3>入库开单 / 发码</h3>
    <el-form inline>
      <el-form-item label="仓库">
        <el-select v-model="warehouseId" style="width:140px">
          <el-option v-for="w in warehouses" :key="w.ID" :label="w.Name || w.ID" :value="w.ID" />
        </el-select>
      </el-form-item>
      <el-form-item label="物资ID"><el-input v-model="materialId" style="width:120px" /></el-form-item>
      <el-form-item label="数量"><el-input-number v-model="qty" :min="1" /></el-form-item>
      <el-form-item label="发码数"><el-input-number v-model="issueCount" :min="1" :max="100" /></el-form-item>
      <el-button type="primary" @click="createOrder">开单</el-button>
    </el-form>
    <el-table :data="orders" border>
      <el-table-column prop="ID" label="ID" width="80" />
      <el-table-column prop="OrderNo" label="单号" />
      <el-table-column prop="Status" label="状态" />
      <el-table-column label="操作" width="200">
        <template #default="{ row }">
          <el-button size="small" @click="issue(row.ID)">发码</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-alert v-if="codes.length" type="success" :title="'唯一码: '+codes.join(', ')" style="margin-top:12px" />
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
import { useWarehouses } from '../composables/warehouses'
const { warehouses, loadWarehouses, defaultWarehouseId } = useWarehouses()
const orders = ref([])
const warehouseId = ref('')
const materialId = ref('')
const qty = ref(2)
const issueCount = ref(2)
const codes = ref([])
async function load() { orders.value = (await http.get('/inbound/orders')).list || [] }
async function createOrder() {
  await http.post('/inbound/orders', { warehouseId: warehouseId.value, materialId: Number(materialId.value), qty: qty.value })
  load()
}
async function issue(id) {
  const data = await http.post(`/inbound/orders/${id}/issue-codes`, { count: issueCount.value })
  codes.value = (data || []).map((x) => x.Code)
}
onMounted(async () => {
  await loadWarehouses()
  warehouseId.value = defaultWarehouseId()
  load()
})
</script>
