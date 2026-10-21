import axios from 'axios'
import { ElMessage } from 'element-plus'

const http = axios.create({ baseURL: '/api/v1', timeout: 20000 })
http.interceptors.request.use((cfg) => {
  const t = localStorage.getItem('accessToken')
  if (t) cfg.headers.Authorization = `Bearer ${t}`
  cfg.headers['X-Client'] = 'portal'
  return cfg
})
http.interceptors.response.use(
  (res) => {
    const body = res.data
    if (body?.code && body.code !== 'OK') {
      ElMessage.error(body.message || body.code)
      return Promise.reject(body)
    }
    return body?.data !== undefined ? body.data : body
  },
  (err) => {
    ElMessage.error(err.response?.data?.message || err.message)
    return Promise.reject(err)
  }
)
export default http
