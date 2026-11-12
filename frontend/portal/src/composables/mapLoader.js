import { baiduJavascriptApiAk, baiduMapAk, useBaiduMap } from '../config/credentials'

export { baiduJavascriptApiAk, baiduMapAk, useBaiduMap }

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
