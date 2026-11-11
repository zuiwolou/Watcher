<template>
  <div>
    <div class="bar">
      <h2>唯一码台账</h2>
      <el-button v-if="canExport" type="primary" @click="exportCsv">导出 CSV</el-button>
    </div>
    <el-form inline>
      <el-form-item label="仓"><el-input v-model="warehouseId" placeholder="WH01" clearable /></el-form-item>
      <el-form-item label="状态">
        <el-select v-model="status" clearable style="width:140px">
          <el-option value="issued" /><el-option value="in_stock" /><el-option value="locked" />
          <el-option value="in_transit" /><el-option value="signed" /><el-option value="voided" />
        </el-select>
      </el-form-item>
      <el-form-item label="码"><el-input v-model="q" clearable placeholder="搜索唯一码" /></el-form-item>
      <el-button type="primary" @click="load">查询</el-button>
    </el-form>
    <el-table :data="list" border>
      <el-table-column prop="Code" label="唯一码" min-width="160" />
      <el-table-column prop="MaterialID" label="物资ID" width="90" />
      <el-table-column prop="WarehouseID" label="仓" width="100" />
      <el-table-column prop="Status" label="状态" width="110" />
      <el-table-column prop="UpdatedAt" label="更新时间" width="180" />
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import http from '../api'
import { hasPerm } from '../auth'

const list = ref([])
const warehouseId = ref('')
const status = ref('')
const q = ref('')
const canExport = hasPerm('material:export')

async function load() {
  const params = new URLSearchParams()
  if (warehouseId.value) params.set('warehouseId', warehouseId.value)
  if (status.value) params.set('status', status.value)
  if (q.value) params.set('q', q.value)
  params.set('page', '1')
  params.set('pageSize', '100')
  const data = await http.get(`/unique-codes?${params}`)
  list.value = data.list || []
}

function exportCsv() {
  const token = localStorage.getItem('accessToken')
  const params = new URLSearchParams()
  if (warehouseId.value) params.set('warehouseId', warehouseId.value)
  if (status.value) params.set('status', status.value)
  if (q.value) params.set('q', q.value)
  const url = `/api/v1/unique-codes/export?${params}`
  fetch(url, { headers: { Authorization: `Bearer ${token}`, 'X-Client': 'portal' } })
    .then((r) => r.blob())
    .then((blob) => {
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = 'unique-codes.csv'
      a.click()
    })
}

onMounted(load)
</script>
<style scoped>.bar{display:flex;justify-content:space-between;align-items:center}</style>
