<template>
  <div>
    <div class="bar">
      <h2>唯一码台账</h2>
      <div>
        <el-button v-if="canExport" type="primary" @click="exportCsv">导出 CSV</el-button>
        <el-button @click="printSelected" :disabled="!selected.length">打印选中</el-button>
      </div>
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
    <el-table :data="list" border @selection-change="onSel">
      <el-table-column type="selection" width="48" />
      <el-table-column prop="Code" label="唯一码" min-width="160" />
      <el-table-column prop="MaterialID" label="物资ID" width="100" />
      <el-table-column prop="WarehouseID" label="仓" width="100" />
      <el-table-column prop="Status" label="状态" width="110" />
      <el-table-column prop="UpdatedAt" label="更新时间" width="180" />
      <el-table-column label="操作" width="180">
        <template #default="{ row }">
          <el-button size="small" @click="copyCode(row.Code)">复制</el-button>
          <el-button v-if="canVoid" size="small" type="danger" :disabled="row.Status==='signed'||row.Status==='voided'" @click="voidCode(row)">作废</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api'
import { hasPerm } from '../auth'

const list = ref([])
const selected = ref([])
const warehouseId = ref('')
const status = ref('')
const q = ref('')
const canExport = hasPerm('material:export')
const canVoid = hasPerm('code:void')

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

function onSel(rows) { selected.value = rows || [] }

async function copyCode(code) {
  try {
    await navigator.clipboard.writeText(code)
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
  }
}

async function voidCode(row) {
  await ElMessageBox.confirm(`确认作废唯一码 ${row.Code}？`, '作废', { type: 'warning' })
  await http.post(`/unique-codes/${encodeURIComponent(row.Code)}/void`, { reason: 'admin void' })
  ElMessage.success('已作废')
  load()
}

function printSelected() {
  const codes = selected.value.map((r) => r.Code).filter(Boolean)
  if (!codes.length) return
  const w = window.open('', '_blank')
  if (!w) return
  w.document.write(`<html><head><title>唯一码打印</title><style>
    body{font-family:monospace;padding:16px} .item{margin:12px 0;padding:8px;border:1px solid #333;font-size:18px}
  </style></head><body>`)
  codes.forEach((c) => w.document.write(`<div class="item">${c}</div>`))
  w.document.write('</body></html>')
  w.document.close()
  w.focus()
  w.print()
}

function exportCsv() {
  const token = localStorage.getItem('accessToken')
  const params = new URLSearchParams()
  if (warehouseId.value) params.set('warehouseId', warehouseId.value)
  if (status.value) params.set('status', status.value)
  if (q.value) params.set('q', q.value)
  const url = `/api/v1/unique-codes/export?${params}`
  fetch(url, { headers: { Authorization: `Bearer ${token}`, 'X-Client': 'admin' } })
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
