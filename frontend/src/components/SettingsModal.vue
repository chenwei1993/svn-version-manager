<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessageBox } from 'element-plus'
import { Check, FolderOpened, Refresh } from '@element-plus/icons-vue'
import { api } from '../api'
import { errText } from '../utils'

const props = defineProps({
  status: { type: Object, required: true },
  notify: { type: Function, required: true },
})
const emit = defineEmits(['close', 'saved'])

const activeTab = ref('general')
const svnStatus = ref({ ...props.status })
const busy = ref(false)
const changingData = ref(false)
const settings = reactive({
  svnPath: '',
  svnAuthor: '',
  defaultExportDir: '',
  logLimit: 200,
  dataDir: '',
  configDir: '',
  configFile: '',
  dbPath: '',
})

const logLimitOptions = computed(() => {
  const base = [50, 100, 200, 500, 1000]
  const current = Number(settings.logLimit)
  if (current && !base.includes(current)) {
    return [...base, current].sort((a, b) => a - b)
  }
  return base
})

onMounted(async () => {
  try {
    const s = await api.getAppSettings()
    Object.assign(settings, s)
  } catch (e) {
    props.notify('读取设置失败：' + errText(e), 'error')
  }
})

async function pickSvnExe() {
  try {
    const p = await api.pickSvnExe()
    if (p) settings.svnPath = p
  } catch (e) {
    props.notify('选择文件失败：' + errText(e), 'error')
  }
}

async function autoDetect() {
  busy.value = true
  try {
    const st = await api.autoDetectSvn()
    settings.svnPath = st.path || ''
    svnStatus.value = st
    emit('saved', st)
    props.notify(st.available ? '已找到 svn.exe：' + st.path : '未自动找到 svn.exe', st.available ? 'success' : 'error')
  } catch (e) {
    props.notify(errText(e), 'error')
  } finally {
    busy.value = false
  }
}

async function chooseExportDir() {
  try {
    const dir = await api.pickDirectory('选择默认导出目录')
    if (dir) settings.defaultExportDir = dir
  } catch (e) {
    props.notify('选择目录失败：' + errText(e), 'error')
  }
}

async function openPath(path) {
  if (!path) return
  try {
    await api.openFolder(path)
  } catch (e) {
    props.notify(errText(e), 'error')
  }
}

async function save() {
  busy.value = true
  try {
    const st = await api.saveAppSettings({
      svnPath: settings.svnPath,
      svnAuthor: settings.svnAuthor,
      defaultExportDir: settings.defaultExportDir,
      logLimit: settings.logLimit,
    })
    svnStatus.value = st
    emit('saved', st)
    props.notify('设置已保存')
  } catch (e) {
    props.notify(errText(e), 'error')
  } finally {
    busy.value = false
  }
}

async function changeDataDir() {
  let dir = ''
  try {
    dir = await api.pickDirectory('选择数据存放目录')
  } catch (e) {
    props.notify('选择目录失败：' + errText(e), 'error')
    return
  }
  if (!dir || dir === settings.dataDir) return

  try {
    await ElMessageBox.confirm(
      `当前数据库会复制到：\n${dir}\n\n原目录的数据文件会保留，不会自动删除。切换后软件会自动重新加载。`,
      '更改数据目录',
      { type: 'warning', confirmButtonText: '切换', cancelButtonText: '取消' }
    )
  } catch {
    return
  }

  changingData.value = true
  try {
    const s = await api.changeDataDir(dir)
    Object.assign(settings, s)
    props.notify('数据目录已切换，正在重新加载...')
    setTimeout(() => window.location.reload(), 800)
  } catch (e) {
    props.notify(errText(e), 'error')
  } finally {
    changingData.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="true"
    title="设置"
    width="680px"
    :close-on-click-modal="false"
    @close="emit('close')"
  >
    <el-tabs v-model="activeTab">
      <el-tab-pane label="通用" name="general">
        <el-form label-position="top">
          <el-form-item label="默认导出目录">
            <el-input v-model="settings.defaultExportDir" placeholder="选择每次导出时默认使用的目录">
              <template #append>
                <el-button :icon="FolderOpened" @click="chooseExportDir">选择目录</el-button>
              </template>
            </el-input>
            <el-button
              text
              type="primary"
              :disabled="!settings.defaultExportDir"
              style="margin-top: 4px"
              @click="openPath(settings.defaultExportDir)"
            >
              打开目录
            </el-button>
          </el-form-item>

          <el-form-item label="默认读取提交数">
            <el-select v-model="settings.logLimit" style="width: 200px">
              <el-option v-for="n in logLimitOptions" :key="n" :value="n" :label="`最近 ${n} 条`" />
            </el-select>
          </el-form-item>
        </el-form>

        <el-alert
          type="info"
          :closable="false"
          show-icon
          title="默认导出目录会填入功能详情和一键导出页；在导出时重新选择目录会自动记住。"
        />
      </el-tab-pane>

      <el-tab-pane label="SVN" name="svn">
        <el-alert
          :type="svnStatus.available ? 'success' : 'warning'"
          :closable="false"
          show-icon
          :title="svnStatus.available ? '已检测到 SVN 命令行' : '未找到可用的 svn 命令行'"
          :description="svnStatus.available ? `版本 ${svnStatus.version}` : svnStatus.message"
        />

        <el-form label-position="top" style="margin-top: 16px">
          <el-form-item label="svn.exe 路径">
            <el-input v-model="settings.svnPath" placeholder="例如 C:\Program Files\TortoiseSVN\bin\svn.exe">
              <template #append>
                <el-button :icon="FolderOpened" @click="pickSvnExe">选择文件</el-button>
              </template>
            </el-input>
            <el-button text type="primary" :icon="Refresh" :loading="busy" style="margin-top: 4px" @click="autoDetect">
              自动检测
            </el-button>
          </el-form-item>

          <el-form-item label="我的 SVN 账号">
            <el-input v-model="settings.svnAuthor" placeholder="提交作者名，例如 joe" />
            <div class="hint" style="margin-top: 4px">
              填写后，「SVN 提交」页面默认只显示这个账号的提交，仍可切换为全部作者。
            </div>
          </el-form-item>

          <el-alert
            v-if="!svnStatus.available"
            type="warning"
            :closable="false"
            show-icon
            title="补装方法"
            description="重新运行 TortoiseSVN 安装包选择 Modify，勾选 command line client tools，完成后重启本软件并点击自动检测。"
          />

          <div class="hint" style="margin-top: 12px">
            本软件调用系统 SVN 凭据缓存，不会保存你的 SVN 密码。私有仓库请先用 TortoiseSVN 正常访问一次。
          </div>
        </el-form>
      </el-tab-pane>

      <el-tab-pane label="数据与配置" name="storage">
        <el-descriptions :column="1" border>
          <el-descriptions-item label="配置目录">{{ settings.configDir }}</el-descriptions-item>
          <el-descriptions-item label="配置文件">{{ settings.configFile }}</el-descriptions-item>
          <el-descriptions-item label="数据目录">{{ settings.dataDir }}</el-descriptions-item>
          <el-descriptions-item label="数据库">{{ settings.dbPath }}</el-descriptions-item>
        </el-descriptions>

        <div class="actions" style="margin-top: 12px">
          <el-button :icon="FolderOpened" @click="openPath(settings.configDir)">打开配置目录</el-button>
          <el-button :icon="FolderOpened" @click="openPath(settings.dataDir)">打开数据目录</el-button>
          <el-button type="warning" plain :loading="changingData" @click="changeDataDir">更改数据目录</el-button>
        </div>

        <el-alert
          type="info"
          :closable="false"
          show-icon
          title="关于更改数据目录"
          description="会把当前 project-manage.db 复制到新目录并切换过去。原目录的数据文件会保留，可手动删除。"
          style="margin-top: 12px"
        />
      </el-tab-pane>
    </el-tabs>

    <template #footer>
      <el-button @click="emit('close')">关闭</el-button>
      <el-button type="primary" :icon="Check" :loading="busy" @click="save">保存</el-button>
    </template>
  </el-dialog>
</template>
