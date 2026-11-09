<template>
  <div class="chart-wrap">
    <svg :viewBox="`0 0 ${width} ${height}`" class="chart" preserveAspectRatio="none">
      <polyline
        v-if="points.length > 1"
        fill="none"
        stroke="#2bb0ed"
        stroke-width="2"
        :points="poly"
      />
      <circle
        v-for="(p, i) in points"
        :key="i"
        :cx="p.x"
        :cy="p.y"
        r="2.5"
        fill="#0b3d4a"
      />
      <text x="8" y="16" class="label">{{ title }}  max {{ maxV.toFixed(1) }} / min {{ minV.toFixed(1) }}</text>
    </svg>
    <div v-if="!points.length" class="empty">暂无曲线数据</div>
  </div>
</template>
<script setup>
import { computed } from 'vue'

const props = defineProps({
  series: { type: Array, default: () => [] }, // [{t, v}]
  title: { type: String, default: '' },
  width: { type: Number, default: 640 },
  height: { type: Number, default: 220 }
})

const minV = computed(() => {
  if (!props.series.length) return 0
  return Math.min(...props.series.map((x) => Number(x.v)))
})
const maxV = computed(() => {
  if (!props.series.length) return 1
  const m = Math.max(...props.series.map((x) => Number(x.v)))
  return m === minV.value ? m + 1 : m
})

const points = computed(() => {
  const arr = props.series || []
  if (!arr.length) return []
  const pad = 24
  const w = props.width - pad * 2
  const h = props.height - pad * 2
  const n = arr.length
  return arr.map((item, i) => {
    const x = pad + (n === 1 ? w / 2 : (i / (n - 1)) * w)
    const ratio = (Number(item.v) - minV.value) / (maxV.value - minV.value || 1)
    const y = pad + (1 - ratio) * h
    return { x, y }
  })
})

const poly = computed(() => points.value.map((p) => `${p.x},${p.y}`).join(' '))
</script>
<style scoped>
.chart-wrap { position:relative; background:#f5f7fa; border:1px solid #d9e2ec; border-radius:6px; }
.chart { width:100%; height:220px; display:block; }
.label { fill:#486581; font-size:12px; }
.empty { position:absolute; inset:0; display:flex; align-items:center; justify-content:center; color:#829ab1; }
</style>
