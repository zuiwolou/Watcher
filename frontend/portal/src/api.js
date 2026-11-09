import axios from 'axios'
import { ElMessage } from 'element-plus'
import { clearSession } from './auth'

const http = axios.create({ baseURL: '/api/v1', timeout: 20000 })

http.interceptors.request.use((cfg) => {
  const t = localStorage.getItem('accessToken')
  if (t) cfg.headers.Authorization = `Bearer ${t}`
  cfg.headers['X-Client'] = 'portal'
  return cfg
})

let refreshing = null

async function refreshTokens() {
  const rt = localStorage.getItem('refreshToken')
  if (!rt) throw new Error('no refresh token')
  const body = await axios.post('/api/v1/auth/refresh', { refreshToken: rt }).then((r) => r.data)
  if (body?.code && body.code !== 'OK') throw new Error(body.message || body.code)
  const data = body.data
  localStorage.setItem('accessToken', data.accessToken)
  localStorage.setItem('refreshToken', data.refreshToken)
  return data.accessToken
}

http.interceptors.response.use(
  (res) => {
    const body = res.data
    if (body?.code && body.code !== 'OK') {
      ElMessage.error(body.message || body.code)
      return Promise.reject(body)
    }
    return body?.data !== undefined ? body.data : body
  },
  async (err) => {
    const cfg = err.config
    if (err.response?.status === 401 && cfg && !cfg.__retried) {
      try {
        if (!refreshing) refreshing = refreshTokens().finally(() => { refreshing = null })
        const access = await refreshing
        cfg.__retried = true
        cfg.headers = cfg.headers || {}
        cfg.headers.Authorization = `Bearer ${access}`
        return http(cfg)
      } catch (_) {
        clearSession()
        if (!location.pathname.includes('/login')) location.href = '/login'
      }
    }
    ElMessage.error(err.response?.data?.message || err.message)
    return Promise.reject(err)
  }
)

export default http
