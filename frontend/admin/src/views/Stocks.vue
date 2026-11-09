<template>
  <div>
    <h3>库存账</h3>
    <el-form inline>
      <el-form-item label="仓库">
        <el-select v-model="warehouseId" clearable style="width:160px" @change="load">
          <el-option v-for="w in warehouses" :key="w.ID" :label="w.Name || w.ID" :value="w.ID" />
        </el-select>
      </el-form-item>
      <el-button @click="load">刷新</el-button>
    </el-form>
    <el-table :data="list" border style="margin-top:12px">
      <el-table-column prop="WarehouseID" label="仓" />
      <el-table-column prop="MaterialID" label="物资ID" />
      <el-table-column prop="Qty" label="数量" />
      <el-table-column prop="LockedQty" label="锁定" />
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
import { useWarehouses } from '../composables/warehouses'
const { warehouses, loadWarehouses } = useWarehouses()
const list = ref([])
const warehouseId = ref('')
async function load() {
  list.value = (await http.get('/stocks', { params: { warehouseId: warehouseId.value || undefined } })) || []
}
onMounted(async () => { await loadWarehouses(); load() })
</script>
