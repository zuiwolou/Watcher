<template>
  <div>
    <h3>监管对接</h3>
    <el-tabs v-model="tab">
      <el-tab-pane label="对接配置" name="cfg">
        <el-form label-width="110px" style="max-width:720px">
          <el-form-item label="通道">
            <el-radio-group v-model="form.code">
              <el-radio-button value="openapi">OpenAPI</el-radio-button>
              <el-radio-button value="sftp">SFTP Drop</el-radio-button>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="名称"><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="启用"><el-switch v-model="form.enabled" /></el-form-item>
          <template v-if="form.code === 'openapi'">
            <el-form-item label="Endpoint"><el-input v-model="oa.endpoint" placeholder="https://regulator.example/api/events" /></el-form-item>
            <el-form-item label="Method">
              <el-select v-model="oa.method" style="width:140px">
                <el-option label="POST" value="POST" /><el-option label="PUT" value="PUT" />
              </el-select>
            </el-form-item>
            <el-form-item label="超时秒"><el-input-number v-model="oa.timeoutSec" :min="3" :max="120" /></el-form-item>
            <el-form-item label="Headers JSON">
              <el-input v-model="oa.headersText" type="textarea" :rows="3" placeholder='{"Authorization":"Bearer xxx"}' />
            </el-form-item>
          </template>
          <template v-else>
            <el-form-item label="落盘目录"><el-input v-model="sftp.path" placeholder="tmp/sftp-drop" /></el-form-item>
            <el-form-item label="Host（预留）"><el-input v-model="sftp.host" /></el-form-item>
            <el-form-item label="Username"><el-input v-model="sftp.username" /></el-form-item>
          </template>
          <el-button type="primary" @click="saveCfg">保存配置</el-button>
          <el-button @click="loadCfg">刷新</el-button>
        </el-form>
        <el-table :data="cfgs" border style="margin-top:16px">
          <el-table-column prop="Code" label="通道" width="100" />
          <el-table-column prop="Name" label="名称" />
          <el-table-column prop="Enabled" label="启用" width="80" />
          <el-table-column prop="UpdatedAt" label="更新时间" width="180" />
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button link type="primary" @click="editCfg(row)">编辑</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <el-tab-pane label="Outbox 投递" name="outbox">
        <el-form inline>
          <el-form-item label="状态">
            <el-select v-model="obStatus" clearable style="width:130px" @change="loadOutbox">
              <el-option label="pending" value="pending" />
              <el-option label="failed" value="failed" />
              <el-option label="success" value="success" />
              <el-option label="dead" value="dead" />
            </el-select>
          </el-form-item>
          <el-form-item label="通道">
            <el-select v-model="obChannel" clearable style="width:120px" @change="loadOutbox">
              <el-option label="openapi" value="openapi" />
              <el-option label="sftp" value="sftp" />
            </el-select>
          </el-form-item>
          <el-button @click="loadOutbox">刷新</el-button>
        </el-form>
        <el-table :data="outbox" border>
          <el-table-column prop="ID" label="ID" width="70" />
          <el-table-column prop="Channel" label="通道" width="90" />
          <el-table-column prop="EventType" label="事件" width="150" />
          <el-table-column prop="BizType" label="业务" width="100" />
          <el-table-column prop="BizID" label="业务ID" width="90" />
          <el-table-column prop="Status" label="状态" width="100" />
          <el-table-column prop="Attempts" label="次数" width="70" />
          <el-table-column prop="LastError" label="错误" show-overflow-tooltip />
          <el-table-column label="操作" width="100">
            <template #default="{ row }">
              <el-button
                v-if="row.Status === 'failed' || row.Status === 'dead' || row.Status === 'pending'"
                link type="primary" @click="retry(row.ID)"
              >重试</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>
<script setup>
import { onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api'

const tab = ref('cfg')
const cfgs = ref([])
const form = reactive({ id: 0, code: 'openapi', name: '监管 OpenAPI', enabled: false })
const oa = reactive({ endpoint: '', method: 'POST', timeoutSec: 10, headersText: '{}' })
const sftp = reactive({ path: 'tmp/sftp-drop', host: '', username: '' })
const outbox = ref([])
const obStatus = ref('')
const obChannel = ref('')

function parseHeaders() {
  try { return JSON.parse(oa.headersText || '{}') } catch { return {} }
}

async function loadCfg() {
  cfgs.value = (await http.get('/integrations')) || []
  const hit = cfgs.value.find((x) => x.Code === form.code)
  if (hit) editCfg(hit)
}

function editCfg(row) {
  form.id = row.ID
  form.code = row.Code
  form.name = row.Name
  form.enabled = !!row.Enabled
  let cfg = {}
  try { cfg = JSON.parse(row.ConfigJSON || '{}') } catch { cfg = {} }
  if (row.Code === 'sftp') {
    sftp.path = cfg.path || 'tmp/sftp-drop'
    sftp.host = cfg.host || ''
    sftp.username = cfg.username || ''
  } else {
    oa.endpoint = cfg.endpoint || ''
    oa.method = cfg.method || 'POST'
    oa.timeoutSec = cfg.timeoutSec || 10
    oa.headersText = JSON.stringify(cfg.headers || {}, null, 2)
  }
}

async function saveCfg() {
  let configJson = {}
  if (form.code === 'sftp') {
    configJson = { path: sftp.path, host: sftp.host, username: sftp.username }
  } else {
    configJson = {
      endpoint: oa.endpoint, method: oa.method, timeoutSec: oa.timeoutSec, headers: parseHeaders()
    }
  }
  await http.put('/integrations', {
    id: form.id || 0,
    code: form.code,
    name: form.name || form.code,
    enabled: form.enabled,
    configJson: JSON.stringify(configJson)
  })
  ElMessage.success('对接配置已保存')
  await loadCfg()
}

async function loadOutbox() {
  const params = { page: 1, pageSize: 50 }
  if (obStatus.value) params.status = obStatus.value
  if (obChannel.value) params.channel = obChannel.value
  const data = await http.get('/openapi/outbox', { params })
  outbox.value = data?.list || []
}

async function retry(id) {
  await http.post(`/openapi/outbox/${id}/retry`)
  ElMessage.success('已重新入队')
  loadOutbox()
}

watch(tab, (v) => { if (v === 'outbox') loadOutbox() })
onMounted(loadCfg)
</script>
