/** Load Leaflet from CDN (no npm). Baidu tiles optional via VITE_BAIDU_MAP_AK later. */
let loading = null

export function loadLeaflet() {
  if (typeof window !== 'undefined' && window.L) return Promise.resolve(window.L)
  if (loading) return loading
  loading = new Promise((resolve, reject) => {
    const css = document.createElement('link')
    css.rel = 'stylesheet'
    css.href = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.css'
    document.head.appendChild(css)
    const s = document.createElement('script')
    s.src = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.js'
    s.onload = () => resolve(window.L)
    s.onerror = () => reject(new Error('leaflet load failed'))
    document.head.appendChild(s)
  })
  return loading
}

/** Approximate meters → degrees at mid latitude (good enough for warehouse fence preview). */
export function metersToLatDeg(m) { return m / 111320 }
export function metersToLngDeg(m, lat) { return m / (111320 * Math.cos((lat * Math.PI) / 180)) }
