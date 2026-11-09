<template>
  <div>
    <h3>MQTT Schema</h3>
    <el-form label-width="100px" style="max-width:640px">
      <el-form-item label="编码"><el-input v-model="form.code" /></el-form-item>
      <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
      <el-form-item label="Topic"><el-input v-model="form.topicPattern" /></el-form-item>
      <el-form-item label="Mapping"><el-input v-model="form.mappingJson" type="textarea" :rows="4" /></el-form-item>
      <el-button type="primary" @click="save">保存</el-button>
      <el-button @click="dryRun">Dry-run</el-button>
    </el-form>
    <pre v-if="dry" style="background:#f5f5f5;padding:12px;margin-top:12px">{{ dry }}</pre>
    <el-table :data="list" border style="margin-top:12px">
      <el-table-column prop="Code" label="编码" />
      <el-table-column prop="Name" label="名称" />
      <el-table-column prop="TopicPattern" label="Topic" />
      <el-table-column prop="Enabled" label="启用" width="80" />
    </el-table>
  </div>
</template>
<script setup>
import { onMounted, reactive, ref } from 'vue'
import http from '../api'
const list = ref([])
const dry = ref('')
const form = reactive({
  code: 'default', name: '默认遥测', topicPattern: 'watcher/+/telemetry',
  mappingJson: '{"temperature":"temperature","humidity":"humidity"}'
})
async function load() { list.value = (await http.get('/mqtt/schemas')) || [] }
async function save() {
  await http.put('/mqtt/schemas', { ...form, enabled: true })
  load()
}
async function dryRun() {
  dry.value = JSON.stringify(await http.post('/mqtt/schemas/dry-run', {
    mappingJson: form.mappingJson,
    samplePayload: '{"warehouseId":"WH01","deviceId":"DEV01","temperature":25}'
  }), null, 2)
}
onMounted(load)
</script>