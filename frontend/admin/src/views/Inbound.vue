<template>
  <div>
    <h3>入库开单 / 发码</h3>
    <el-form inline>
      <el-form-item label="物资ID"><el-input v-model="materialId" style="width:120px" /></el-form-item>
      <el-form-item label="数量"><el-input-number v-model="qty" :min="1" /></el-form-item>
      <el-button type="primary" @click="createOrder">开单</el-button>
    </el-form>
    <el-table :data="orders" border>
      <el-table-column prop="ID" label="ID" width="80" />
      <el-table-column prop="OrderNo" label="单号" />
      <el-table-column prop="Status" label="状态" />
      <el-table-column label="操作" width="200">
        <template #default="{ row }">
          <el-button size="small" @click="issue(row.ID)">发码2个</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-alert v-if="codes.length" type="success" :title="'唯一码: '+codes.join(', ')" style="margin-top:12px" />
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
const orders = ref([])
const materialId = ref('')
const qty = ref(2)
const codes = ref([])
async function load() { orders.value = (await http.get('/inbound/orders')).list || [] }
async function createOrder() {
  await http.post('/inbound/orders', { warehouseId: 'WH01', materialId: Number(materialId.value), qty: qty.value })
  load()
}
async function issue(id) {
  const data = await http.post(`/inbound/orders/${id}/issue-codes`, { count: 2 })
  codes.value = (data || []).map((x) => x.Code)
}
onMounted(load)
</script>
