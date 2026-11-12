<template>
  <div>
    <div class="bar">
      <h2>围栏状态</h2>
      <div>
        <el-input v-model="warehouseId" style="width:120px;margin-right:8px" placeholder="仓ID" />
        <el-button @click="load">刷新</el-button>
      </div>
    </div>
    <el-row :gutter="12">
      <el-col :span="14"><div ref="mapEl" class="map" /></el-col>
      <el-col :span="10">
        <el-table :data="list" border size="small" height="280" @row-click="focus">
          <el-table-column prop="Name" label="名称" />
          <el-table-column prop="ShapeType" label="形状" width="90" />
          <el-table-column prop="WarehouseID" label="仓" width="90" />
          <el-table-column prop="Enabled" label="启用" width="70" />
        </el-table>
        <h3 style="margin:12px 0 8px">围栏事件</h3>
        <el-table :data="events" border size="small" height="200">
          <el-table-column prop="Title" label="事件" />
          <el-table-column prop="DeviceID" label="设备" width="100" />
          <el-table-column prop="Status" label="状态" width="80" />
          <el-table-column prop="RaisedAt" label="时间" width="160" />
        </el-table>
      </el-col>
    </el-row>
  </div>
</template>
<script setup>
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import http from '../api'
import { loadLeaflet } from '../composables/mapLoader'

const warehouseId = ref('WH01')
const list = ref([])
const events = ref([])
const mapEl = ref(null)
let map, L, layers = []

async function initMap() {
  L = await loadLeaflet()
  await nextTick()
  map = L.map(mapEl.value).setView([31.2304, 121.4737], 13)
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; OpenStreetMap'
  }).addTo(map)
}

function clearLayers() {
  layers.forEach((ly) => map && map.removeLayer(ly))
  layers = []
}

function render() {
  if (!map || !L) return
  clearLayers()
  for (const f of list.value) {
    try {
      if (f.ShapeType === 'circle') {
        const g = JSON.parse(f.GeoJSON || '{}')
        const ly = L.circle([g.lat, g.lng], {
          radius: g.radiusM || 100, color: '#486581', fillOpacity: 0.12
        }).bindPopup(f.Name || String(f.ID))
        ly.addTo(map)
        layers.push(ly)
      } else {
        const g = JSON.parse(f.GeoJSON || '{}')
        const pts = (g.points || []).map(([a, b]) => [a, b])
        if (pts.length >= 3) {
          const ly = L.polygon(pts, { color: '#486581', fillOpacity: 0.12 }).bindPopup(f.Name || String(f.ID))
          ly.addTo(map)
          layers.push(ly)
        }
      }
    } catch (_) {}
  }
  if (layers.length) {
    const g = L.featureGroup(layers)
    map.fitBounds(g.getBounds(), { padding: [20, 20] })
  }
}

function focus(row) {
  try {
    if (row.ShapeType === 'circle') {
      const g = JSON.parse(row.GeoJSON || '{}')
      map.setView([g.lat, g.lng], 15)
    } else {
      const g = JSON.parse(row.GeoJSON || '{}')
      if (g.points?.[0]) map.setView(g.points[0], 15)
    }
  } catch (_) {}
}

async function load() {
  list.value = (await http.get('/geofences', { params: { warehouseId: warehouseId.value } })) || []
  const alarms = await http.get('/alarms', { params: { status: 'raised', page: 1, pageSize: 50 } })
  events.value = (alarms.list || []).filter((a) => String(a.RuleCode || '').startsWith('geofence_'))
  render()
}

onMounted(async () => {
  await initMap()
  await load()
})
onBeforeUnmount(() => { if (map) { map.remove(); map = null } })
</script>
<style scoped>
.bar{display:flex;justify-content:space-between;align-items:center;margin-bottom:8px}
.map{height:520px;border:1px solid #d9e2ec;border-radius:6px}
h2{margin:0}
</style>
