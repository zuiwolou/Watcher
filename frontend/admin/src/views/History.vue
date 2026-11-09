<template>
  <div>
    <h3>监测 / 轨迹历史</h3>
    <el-form inline>
      <el-form-item label="仓库">
        <el-select v-model="warehouseId" style="width:140px">
          <el-option v-for="w in warehouses" :key="w.ID" :label="w.Name || w.ID" :value="w.ID" />
        </el-select>
      </el-form-item>
      <el-form-item label="指标"><el-input v-model="metric" style="width:120px" /></el-form-item>
      <el-form-item label="设备"><el-input v-model="deviceId" style="width:120px" /></el-form-item>
      <el-button type="primary" @click="loadTelemetry">查遥测</el-button>
      <el-button @click="loadTracks">查轨迹</el-button>
    </el-form>
    <el-table :data="rows" border style="margin-top:12px" max-height="480">
      <el-table-column v-for="c in cols" :key="c" :prop="c" :label="c" />
    </el-table>
  </div>
</template>
<script setup>
import { computed, onMounted, ref } from 'vue'
import http from '../api'
import { useWarehouses } from '../composables/warehouses'
const { warehouses, loadWarehouses, defaultWarehouseId } = useWarehouses()
const warehouseId = ref('')
const metric = ref('temperature')
const deviceId = ref('')
const rows = ref([])
const cols = computed(() => (rows.value[0] ? Object.keys(rows.value[0]) : []))
async function loadTelemetry() {
  rows.value = (await http.get('/history/telemetry', { params: { warehouseId: warehouseId.value, metric: metric.value } })) || []
}
async function loadTracks() {
  rows.value = (await http.get('/history/tracks', { params: { warehouseId: warehouseId.value, deviceId: deviceId.value } })) || []
}
onMounted(async () => {
  await loadWarehouses()
  warehouseId.value = defaultWarehouseId()
})
</script>
