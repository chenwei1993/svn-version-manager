<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessageBox } from 'element-plus'
import AppIcon from './AppIcon.vue'
import {
  IconCheck as Check,
  IconFolder as FolderOpened,
} from './icons'
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

const tabs = [
  { key: 'general', label: '通用', icon: 'sliders' },
  { key: 'svn', label: 'SVN', icon: 'branch' },
  { key: 'storage', label: '数据与配置', icon: 'database' },
]

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
    width="720px"
    top="9vh"
    class="settings-dialog"
    :show-close="false"
    :close-on-click-modal="false"
    @close="emit('close')"
  >
    <template #header>
      <div class="settings-head">
        <div>
          <h3>设置</h3>
          <p>偏好设置保存在本机</p>
        </div>
        <button class="icon-button" title="关闭" @click="emit('close')">
          <AppIcon name="close" :size="14" />
        </button>
      </div>
    </template>

    <div class="settings-body">
      <aside class="settings-rail">
        <div class="rail-label">设置项</div>
        <button
          v-for="t in tabs"
          :key="t.key"
          class="rail-item"
          :class="{ active: activeTab === t.key }"
          @click="activeTab = t.key"
        >
          <AppIcon :name="t.icon" :size="16" />
          <span>{{ t.label }}</span>
        </button>
      </aside>

      <section class="settings-content">
        <template v-if="activeTab === 'general'">
          <div class="settings-section">
            <div class="settings-section-title">通用</div>
            <div class="settings-section-desc">设置导出默认位置和 SVN 提交读取数量</div>

            <div class="field-label">默认导出目录</div>
            <el-input v-model="settings.defaultExportDir" placeholder="选择每次导出时默认使用的目录">
              <template #append>
                <el-button :icon="FolderOpened" @click="chooseExportDir">选择目录</el-button>
              </template>
            </el-input>
            <div class="field-help">
              <span>导出时自动填入，仍可在导出页临时更改</span>
              <span
                class="link"
                :class="{ disabled: !settings.defaultExportDir }"
                @click="openPath(settings.defaultExportDir)"
              >
                打开目录
              </span>
            </div>

            <div class="field-label">默认读取提交数</div>
            <el-select v-model="settings.logLimit" style="width: 220px">
              <el-option v-for="n in logLimitOptions" :key="n" :value="n" :label="`最近 ${n} 条`" />
            </el-select>
            <div class="field-help"><span>读取 SVN 日志时默认拉取的提交条数</span></div>

            <el-alert
              type="info"
              :closable="false"
              show-icon
              title="默认导出目录会填入功能详情和导出页；在导出时重新选择目录会自动记住。"
            />

            <div class="aside-divider"></div>

            <div class="field-label">当前数据目录</div>
            <div class="data-box">
              <el-icon><FolderOpened /></el-icon>
              <span class="path">{{ settings.dataDir || '—' }}</span>
              <span class="link" @click="openPath(settings.dataDir)">打开</span>
            </div>
            <div class="field-help"><span>包含本地 SQLite 数据库与配置文件，不会上传</span></div>
          </div>
        </template>

        <template v-else-if="activeTab === 'svn'">
          <div class="settings-section">
            <div class="settings-section-title">SVN</div>
            <div class="settings-section-desc">配置 svn.exe 与默认提交作者</div>

            <el-alert
              :type="svnStatus.available ? 'success' : 'warning'"
              :closable="false"
              show-icon
              :title="svnStatus.available ? '已检测到 SVN 命令行' : '未找到可用的 svn 命令行'"
              :description="svnStatus.available ? `版本 ${svnStatus.version}` : svnStatus.message"
            />

            <div class="field-label">svn.exe 路径</div>
            <el-input v-model="settings.svnPath" placeholder="例如 C:\Program Files\TortoiseSVN\bin\svn.exe">
              <template #append>
                <el-button :icon="FolderOpened" @click="pickSvnExe">选择文件</el-button>
              </template>
            </el-input>
            <div class="field-help">
              <span>填写后优先使用该路径，留空则使用系统 PATH</span>
              <span class="link" @click="autoDetect">自动检测</span>
            </div>

            <div class="field-label">我的 SVN 账号</div>
            <el-input v-model="settings.svnAuthor" placeholder="提交作者名，例如 joe" />
            <div class="field-help">
              <span>填写后，「SVN 提交」页面默认只显示这个账号的提交，仍可切换为全部作者。</span>
            </div>

            <el-alert
              v-if="!svnStatus.available"
              type="warning"
              :closable="false"
              show-icon
              title="补装方法"
              description="重新运行 TortoiseSVN 安装包选择 Modify，勾选 command line client tools，完成后重启本软件并点击自动检测。"
            />

            <div class="aside-divider"></div>
            <div class="field-help">
              <span>本软件调用系统 SVN 凭据缓存，不会保存你的 SVN 密码。私有仓库请先用 TortoiseSVN 正常访问一次。</span>
            </div>
          </div>
        </template>

        <template v-else>
          <div class="settings-section">
            <div class="settings-section-title">数据与配置</div>
            <div class="settings-section-desc">查看文件位置，或将数据迁移到其他目录</div>

            <el-descriptions :column="1" border size="small">
              <el-descriptions-item label="配置目录">{{ settings.configDir }}</el-descriptions-item>
              <el-descriptions-item label="配置文件">{{ settings.configFile }}</el-descriptions-item>
              <el-descriptions-item label="数据目录">{{ settings.dataDir }}</el-descriptions-item>
              <el-descriptions-item label="数据库">{{ settings.dbPath }}</el-descriptions-item>
            </el-descriptions>

            <div class="actions">
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
            />
          </div>
        </template>
      </section>
    </div>

    <template #footer>
      <div class="settings-footer">
        <span class="note">设置保存在本机，不会上传</span>
        <div class="actions">
          <el-button @click="emit('close')">关闭</el-button>
          <el-button type="primary" :icon="Check" :loading="busy" @click="save">保存设置</el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>
