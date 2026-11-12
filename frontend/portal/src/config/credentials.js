/** 外部凭证。环境变量名与开放平台申请的密钥名对齐。 */
export const mapProvider = (import.meta.env.VITE_MAP_PROVIDER || 'leaflet').toLowerCase()

/** 百度 JavaScript API 的 AK（浏览器端） */
export const baiduJavascriptApiAk = (
  import.meta.env.VITE_BAIDU_JAVASCRIPT_API_AK ||
  import.meta.env.VITE_BAIDU_MAP_AK || // 兼容旧名
  ''
).trim()

export const useBaiduMap = mapProvider === 'baidu' && !!baiduJavascriptApiAk

/** @deprecated 使用 baiduJavascriptApiAk */
export const baiduMapAk = baiduJavascriptApiAk
