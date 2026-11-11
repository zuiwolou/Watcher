<template>
  <div>
    <div class="bar">
      <h3>地图围栏</h3>
      <div>
        <el-select v-model="warehouseId" style="width:140px;margin-right:8px" @change="reload">
          <el-option v-for="w in warehouses" :key="w.ID" :label="w.Name || w.ID" :value="w.ID" />
        </el-select>
        <el-button @click="reload">刷新</el-button>
        <el-button type="primary" :disabled="!draftReady" @click="saveDraft">保存围栏</el-button>
      </div>
    </div>

    <el-row :gutter="12">
      <el-col :span="16">
        <div class="map-toolbar">
          <el-radio-group v-model="drawMode" size="small">
            <el-radio-button value="circle">圆形</el-radio-button>
            <el-radio-button value="polygon">多边形</el-radio-button>
          </el-radio-group>
          <span class="hint">{{ hint }}</span>
          <el-button size="small" @click="undoVertex" :disabled="drawMode!=='polygon' || vertices.length===0">撤销点</el-button>
          <el-button size="small" @click="clearDraft">清除草稿</el-button>
        </div>
        <div ref="mapEl" class="map" />
      </el-col>
      <el-col :span="8">
        <el-card shadow="never">
          <template #header>草稿属性</template>
          <el-form label-width="72px" size="small">
            <el-form-item label="名称"><el-input v-model="draft.name" /></el-form-item>
            <el-form-item v-if="drawMode==='circle'" label="半径m">
              <el-slider v-model="draft.radiusM" :min="50" :max="5000" :step="10" show-input />
            </el-form-item>
            <el-form-item label="滞回m">
              <el-input-number v-model="draft.hysteresisM" :min="0" :max="500" :step="5" />
            </el-form-item>
            <el-form-item label="时段">
              <el-input v-model="draft.scheduleJson" type="textarea" :rows="2" placeholder='空=全天；例 {"windows":[{"dow":[1,2,3,4,5],"start":"08:00","end":"18:00"}]}' />
            </el-form-item>
            <el-form-item label="中心/点">
              <pre class="mono">{{ draftPreview }}</pre>
            </el-form-item>
          </el-form>
        </el-card>
        <el-table :data="list" border size="small" style="margin-top:12px" height="360" @row-click="focusFence">
          <el-table-column prop="ID" label="ID" width="60" />
          <el-table-column prop="Name" label="名称" />
          <el-table-column prop="ShapeType" label="形状" width="80" />
        </el-table>
      </el-col>
    </el-row>
  </div>
</template>
<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api'
import { useWarehouses } from '../composables/warehouses'
import { loadLeaflet } from '../composables/mapLoader'

const { warehouses, loadWarehouses, defaultWarehouseId } = useWarehouses()
const warehouseId = ref('')
const list = ref([])
const mapEl = ref(null)
const drawMode = ref('circle')
const draft = reactive({ name: '主围栏', lat: 31.2304, lng: 121.4737, radiusM: 500, hysteresisM: 0, scheduleJson: '' })
const vertices = ref([])
let map, L, draftLayer, fenceLayers = []

const hint = computed(() => drawMode.value === 'circle'
  ? '单击地图设置圆心，右侧滑条调半径'
  : '单击依次加点，至少 3 个点；可撤销')

const draftReady = computed(() => {
  if (!draft.name.trim()) return false
  if (drawMode.value === 'circle') return !!draft.lat && !!draft.lng && draft.radiusM > 0
  return vertices.value.length >= 3
})

const draftPreview = computed(() => {
  if (drawMode.value === 'circle') {
    return JSON.stringify({ lat: draft.lat, lng: draft.lng, radiusM: draft.radiusM }, null, 2)
  }
  return JSON.stringify(vertices.value, null, 2)
})

async function initMap() {
  L = await loadLeaflet()
  await nextTick()
  if (map) { map.remove(); map = null }
  map = L.map(mapEl.value).setView([draft.lat, draft.lng], 14)
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; OpenStreetMap'
  }).addTo(map)
  map.on('click', onMapClick)
  redrawDraft()
}

function onMapClick(e) {
  if (drawMode.value === 'circle') {
    draft.lat = +e.latlng.lat.toFixed(6)
    draft.lng = +e.latlng.lng.toFixed(6)
  } else {
    vertices.value = [...vertices.value, [+e.latlng.lat.toFixed(6), +e.latlng.lng.toFixed(6)]]
  }
  redrawDraft()
}

function redrawDraft() {
  if (!map || !L) return
  if (draftLayer) { map.removeLayer(draftLayer); draftLayer = null }
  if (drawMode.value === 'circle') {
    draftLayer = L.circle([draft.lat, draft.lng], {
      radius: draft.radiusM, color: '#2bb0ed', fillOpacity: 0.2
    }).addTo(map)
  } else if (vertices.value.length) {
    const latlngs = vertices.value.map(([a, b]) => [a, b])
    if (latlngs.length >= 3) {
      draftLayer = L.polygon(latlngs, { color: '#f0a202', fillOpacity: 0.2 }).addTo(map)
    } else {
      draftLayer = L.polyline(latlngs, { color: '#f0a202' }).addTo(map)
      latlngs.forEach((ll) => L.circleMarker(ll, { radius: 4, color: '#f0a202' }).addTo(map))
    }
  }
}

function clearFenceLayers() {
  fenceLayers.forEach((ly) => map && map.removeLayer(ly))
  fenceLayers = []
}

function renderFences() {
  if (!map || !L) return
  clearFenceLayers()
  for (const f of list.value) {
    try {
      if (f.ShapeType === 'circle') {
        const g = JSON.parse(f.GeoJSON || '{}')
        const ly = L.circle([g.lat, g.lng], {
          radius: g.radiusM || 100, color: '#486581', dashArray: '4 4', fillOpacity: 0.08
        }).bindPopup(`${f.Name || f.ID}`)
        ly.addTo(map)
        fenceLayers.push(ly)
      } else {
        const g = JSON.parse(f.GeoJSON || '{}')
        const pts = (g.points || []).map(([a, b]) => [a, b])
        if (pts.length >= 3) {
          const ly = L.polygon(pts, { color: '#486581', dashArray: '4 4', fillOpacity: 0.08 })
            .bindPopup(`${f.Name || f.ID}`)
          ly.addTo(map)
          fenceLayers.push(ly)
        }
      }
    } catch (_) { /* ignore bad geo */ }
  }
}

function focusFence(row) {
  try {
    if (row.ShapeType === 'circle') {
      const g = JSON.parse(row.GeoJSON || '{}')
      map.setView([g.lat, g.lng], 15)
      drawMode.value = 'circle'
      draft.name = row.Name
      draft.lat = g.lat
      draft.lng = g.lng
      draft.radiusM = g.radiusM || 500
      vertices.value = []
    } else {
      const g = JSON.parse(row.GeoJSON || '{}')
      drawMode.value = 'polygon'
      draft.name = row.Name
      vertices.value = [...(g.points || [])]
      if (vertices.value[0]) map.setView(vertices.value[0], 15)
    }
    redrawDraft()
  } catch (_) {}
}

function undoVertex() { vertices.value = vertices.value.slice(0, -1); redrawDraft() }
function clearDraft() {
  vertices.value = []
  redrawDraft()
}

async function reload() {
  list.value = (await http.get('/geofences', { params: { warehouseId: warehouseId.value } })) || []
  renderFences()
}

async function saveDraft() {
  await ElMessageBox.confirm('确认保存当前围栏草稿？', '保存围栏', { type: 'warning' })
  const body = {
    warehouseId: warehouseId.value,
    name: draft.name,
    shapeType: drawMode.value,
    hysteresisM: draft.hysteresisM || 0,
    scheduleJson: (draft.scheduleJson || '').trim(),
    enabled: true
  }
  if (drawMode.value === 'circle') {
    body.circle = { lat: draft.lat, lng: draft.lng, radiusM: draft.radiusM }
  } else {
    body.geoJson = JSON.stringify({ type: 'polygon', points: vertices.value })
  }
  await http.put('/geofences', body)
  ElMessage.success('围栏已保存')
  await reload()
}

watch(() => draft.radiusM, () => { if (drawMode.value === 'circle') redrawDraft() })
watch(drawMode, () => { redrawDraft() })

onMounted(async () => {
  await loadWarehouses()
  warehouseId.value = defaultWarehouseId()
  await initMap()
  await reload()
})
onBeforeUnmount(() => { if (map) { map.remove(); map = null } })
</script>
<style scoped>
.bar { display:flex; justify-content:space-between; align-items:center; margin-bottom:8px; }
.map-toolbar { display:flex; align-items:center; gap:10px; margin-bottom:8px; flex-wrap:wrap; }
.hint { color:#627d98; font-size:12px; }
.map { height:560px; border:1px solid #d9e2ec; border-radius:4px; }
.mono { margin:0; font-size:12px; max-height:140px; overflow:auto; background:#f5f7fa; padding:8px; }
</style>
