import axios from 'axios'
import { ElMessage } from 'element-plus'

const http = axios.create({ baseURL: '/api/v1', timeout: 20000 })

http.interceptors.request.use((cfg) => {
  const t = localStorage.getItem('accessToken')
  if (t) cfg.headers.Authorization = `Bearer ${t}`
  cfg.headers['X-Client'] = 'admin'
  return cfg
})

http.interceptors.response.use(
  (res) => {
    const body = res.data
    if (body && body.code && body.code !== 'OK') {
      ElMessage.error(body.message || body.code)
      return Promise.reject(body)
    }
    return body?.data !== undefined ? body.data : body
  },
  async (err) => {
    if (err.response?.status === 401) {
      const rt = localStorage.getItem('refreshToken')
      if (rt && !err.config.__retried) {
        try {
          const data = await axios.post('/api/v1/auth/refresh', { refreshToken: rt }).then((r) => r.data.data)
          localStorage.setItem('accessToken', data.accessToken)
          localStorage.setItem('refreshToken', data.refreshToken)
          err.config.__retried = true
          err.config.headers.Authorization = `Bearer ${data.accessToken}`
          return http(err.config)
        } catch (_) {
          localStorage.clear()
          location.href = '/login'
        }
      }
    }
    ElMessage.error(err.response?.data?.message || err.message)
    return Promise.reject(err)
  }
)

export default http
