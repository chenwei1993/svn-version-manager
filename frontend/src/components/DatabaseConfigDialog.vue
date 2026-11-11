<script setup>
import { onMounted, reactive, ref, watch } from 'vue'
import { Check, Connection } from '@element-plus/icons-vue'
import { api } from '../api'
import { errText } from '../utils'

const props = defineProps({
  project: { type: Object, required: true },
  notify: { type: Function, required: true },
})
const emit = defineEmits(['close', 'saved'])

const loading = ref(false)
const testing = ref(false)
const saving = ref(false)
const form = reactive({
  host: '127.0.0.1',
  port: 3306,
  user: 'root',
  password: '',
  database: '',
  hasPassword: false,
  clearPassword: false,
})

onMounted(async () => {
  loading.value = true
  try {
    const cfg = await api.getProjectDBConfig(props.project.id)
    Object.assign(form, cfg)
    form.password = ''
    form.clearPassword = false
  } catch (e) {
    props.notify('读取数据库配置失败：' + errText(e), 'error')
  } finally {
    loading.value = false
  }
})

watch(() => form.password, (v) => {
  if (v) form.clearPassword = false
})

async function testConnection() {
  if (!form.database.trim()) return props.notify('请填写数据库名', 'error')
  testing.value = true
  try {
    const version = await api.testProjectDB(props.project.id, { ...form })
    props.notify('连接成功，MySQL ' + version, 'success')
  } catch (e) {
    props.notify(errText(e), 'error')
  } finally {
    testing.value = false
  }
}

async function save() {
  if (!form.database.trim()) return props.notify('请填写数据库名', 'error')
  saving.value = true
  try {
    const cfg = await api.saveProjectDBConfig(props.project.id, { ...form })
    Object.assign(form, cfg)
    form.password = ''
    form.clearPassword = false
    emit('saved', cfg)
    props.notify('数据库配置已保存')
  } catch (e) {
    props.notify(errText(e), 'error')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="true"
    :title="`数据库连接 - ${project.name}`"
    width="560px"
    :close-on-click-modal="false"
    @close="emit('close')"
  >
    <div v-loading="loading">
      <el-form label-position="top">
        <el-row :gutter="16">
          <el-col :span="16">
            <el-form-item label="主机">
              <el-input v-model="form.host" placeholder="127.0.0.1" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="端口">
              <el-input-number v-model="form.port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="用户名">
              <el-input v-model="form.user" placeholder="root" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="数据库名">
              <el-input v-model="form.database" placeholder="例如：demo_db" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="密码">
              <el-input
                v-model="form.password"
                type="password"
                show-password
                :placeholder="form.hasPassword ? '已保存，留空表示不修改' : '数据库密码（没有密码可留空）'"
              />
              <el-checkbox v-if="form.hasPassword" v-model="form.clearPassword" style="margin-top: 4px">
                清除已保存的密码
              </el-checkbox>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="密码使用 Windows DPAPI 加密保存在本机，不会以明文返回前端。"
      />
    </div>

    <template #footer>
      <el-button :icon="Connection" :loading="testing" :disabled="loading" @click="testConnection">
        测试连接
      </el-button>
      <el-button @click="emit('close')">关闭</el-button>
      <el-button type="primary" :icon="Check" :loading="saving" :disabled="loading" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>
