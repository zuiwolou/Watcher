<template>
  <div>
    <h3>阈值配置</h3>
    <el-form inline>
      <el-form-item label="仓库">
        <el-select v-model="warehouseId" style="width:140px" @change="load">
          <el-option v-for="w in warehouses" :key="w.ID" :label="w.Name || w.ID" :value="w.ID" />
        </el-select>
      </el-form-item>
      <el-form-item label="指标"><el-input v-model="metric" style="width:120px" /></el-form-item>
      <el-form-item label="上限"><el-input-number v-model="upper" /></el-form-item>
      <el-form-item label="下限"><el-input-number v-model="lower" /></el-form-item>
      <el-button type="primary" @click="save">保存</el-button>
      <el-button @click="evalOnce">评估一次</el-button>
    </el-form>
    <el-table :data="list" border style="margin-top:12px">
      <el-table-column prop="Metric" label="指标" />
      <el-table-column prop="Upper" label="上限" />
      <el-table-column prop="Lower" label="下限" />
      <el-table-column prop="Version" label="版本" width="80" />
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import { ElMessageBox } from 'element-plus'
import http from '../api'
import { useWarehouses } from '../composables/warehouses'
const { warehouses, loadWarehouses, defaultWarehouseId } = useWarehouses()
const list = ref([])
const warehouseId = ref('')
const metric = ref('temperature')
const upper = ref(30)
const lower = ref(0)
async function load() {
  if (!warehouseId.value) return
  list.value = (await http.get('/thresholds', { params: { warehouseId: warehouseId.value } })) || []
}
async function save() {
  await http.put(`/thresholds/${warehouseId.value}`, { metric: metric.value, upper: upper.value, lower: lower.value })
  load()
}
async function evalOnce() {
  const { value } = await ElMessageBox.prompt('设备ID与指标值，格式 deviceId,metricValue', '评估遥测', { inputValue: 'DEV01,45' })
  const [deviceId, v] = value.split(',')
  await http.post('/monitor/evaluate', {
    warehouseId: warehouseId.value, deviceId: deviceId.trim(),
    metrics: { [metric.value]: Number(v) }
  })
}
onMounted(async () => {
  await loadWarehouses()
  warehouseId.value = defaultWarehouseId()
  load()
})
</script>
