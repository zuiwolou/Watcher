import { ref } from 'vue'
import http from '../api'

const list = ref([])
let loaded = false

export function useWarehouses() {
  async function loadWarehouses() {
    if (loaded && list.value.length) return list.value
    list.value = (await http.get('/warehouses')) || []
    loaded = true
    return list.value
  }
  function defaultWarehouseId() {
    return list.value[0]?.ID || list.value[0]?.id || ''
  }
  return { warehouses: list, loadWarehouses, defaultWarehouseId }
}
