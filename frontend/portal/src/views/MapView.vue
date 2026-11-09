<template>
  <div>
    <div class="bar">
      <h2>地图轨迹 / 实时点位</h2>
      <div>
        <el-input v-model="warehouseId" style="width:120px;margin-right:8px" placeholder="仓ID" />
        <el-input v-model="deviceId" style="width:140px;margin-right:8px" placeholder="设备ID(可选)" />
        <el-button type="primary" @click="loadTrack">加载轨迹</el-button>
        <el-button @click="fitAll">适配视野</el-button>
      </div>
    </div>
    <div ref="mapEl" class="map" />
    <div class="legend">蓝线=历史轨迹 · 橙点=实时 LOCATION（Protobuf）</div>
  </div>
</template>
<script setup>
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import http from '../api'
import { loadLeaflet } from '../composables/mapLoader'
import { realtime } from '../realtime'

const warehouseId = ref('WH01')
const deviceId = ref('')
const mapEl = ref(null)
let map, L, trackLayer, liveMarkers = {}, off

async function initMap() {
  L = await loadLeaflet()
  await nextTick()
  map = L.map(mapEl.value).setView([31.2304, 121.4737], 13)
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    maxZoom: 19,
    attribution: '&copy; OpenStreetMap'
  }).addTo(map)
}

async function loadTrack() {
  const list = (await http.get('/history/tracks', {
    params: { warehouseId: warehouseId.value, deviceId: deviceId.value }
  })) || []
  const rows = Array.isArray(list) ? list : (list.list || [])
  if (trackLayer) { map.removeLayer(trackLayer); trackLayer = null }
  const latlngs = rows
    .map((r) => [Number(r.Lat), Number(r.Lng)])
    .filter((p) => Number.isFinite(p[0]) && Number.isFinite(p[1]))
  if (latlngs.length) {
    trackLayer = L.polyline(latlngs, { color: '#2bb0ed', weight: 3 }).addTo(map)
    map.fitBounds(trackLayer.getBounds(), { padding: [24, 24] })
  }
}

function upsertLive(env) {
  if (!map || !L || !env.location) return
  if (warehouseId.value && env.warehouseId && env.warehouseId !== warehouseId.value) return
  if (deviceId.value && env.deviceId && env.deviceId !== deviceId.value) return
  const id = env.deviceId || 'unknown'
  const ll = [env.location.lat, env.location.lng]
  if (liveMarkers[id]) {
    liveMarkers[id].setLatLng(ll)
  } else {
    liveMarkers[id] = L.circleMarker(ll, {
      radius: 7, color: '#f0a202', fillColor: '#f0a202', fillOpacity: 0.85
    }).bindPopup(`${id}`).addTo(map)
  }
  liveMarkers[id].setPopupContent(`${id}<br/>${new Date(env.tsUnixMs).toLocaleTimeString()}`)
}

function fitAll() {
  const layers = []
  if (trackLayer) layers.push(trackLayer)
  Object.values(liveMarkers).forEach((m) => layers.push(m))
  if (!layers.length) return
  const group = L.featureGroup(layers)
  map.fitBounds(group.getBounds(), { padding: [24, 24] })
}

onMounted(async () => {
  await initMap()
  await loadTrack()
  realtime.start()
  off = realtime.on((evt) => {
    if (evt.type === 'message' && evt.envelope?.msgType === 'LOCATION') {
      upsertLive(evt.envelope)
    }
  })
})
onBeforeUnmount(() => {
  off?.()
  if (map) { map.remove(); map = null }
})
</script>
<style scoped>
.bar{display:flex;justify-content:space-between;align-items:center;margin-bottom:8px;gap:12px;flex-wrap:wrap}
.map{height:560px;border:1px solid #d9e2ec;border-radius:6px}
.legend{margin-top:8px;color:#627d98;font-size:12px}
h2{margin:0}
</style>
