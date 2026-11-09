<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import appIcon from './assets/app-icon.png'
import { api } from './api'
import { errText } from './utils'
import ProjectsView from './components/ProjectsView.vue'
import FeaturesView from './components/FeaturesView.vue'
import SvnView from './components/SvnView.vue'
import ExportView from './components/ExportView.vue'
import SettingsModal from './components/SettingsModal.vue'

const navs = [
  { key: 'projects', label: '项目管理', icon: 'Folder' },
  { key: 'features', label: '功能记录', icon: 'Tickets' },
  { key: 'svn', label: 'SVN 提交', icon: 'Clock' },
  { key: 'export', label: '一键导出', icon: 'Download' },
]

const tab = ref('projects')
const projects = ref([])
const currentProjectId = ref(null)
const svnStatus = ref({ available: false, path: '', version: '', message: '正在检测...' })
const settingsOpen = ref(false)
const svnTargetFeature = ref(0)

const currentProject = computed(() => projects.value.find((p) => p.id === currentProjectId.value) || null)

function notify(text, type = 'success') {
  const typeMap = { error: 'error', warning: 'warning', info: 'info', success: 'success' }
  ElMessage({ message: text, type: typeMap[type] || 'success', duration: 3000, showClose: true })
}

async function loadProjects(selectId) {
  try {
    projects.value = await api.listProjects()
    const exists = projects.value.some((p) => p.id === currentProjectId.value)
    if (selectId) currentProjectId.value = selectId
    else if (!exists) currentProjectId.value = projects.value[0]?.id ?? null
  } catch (e) {
    notify('读取项目失败：' + errText(e), 'error')
  }
}

async function loadSvnStatus() {
  try {
    svnStatus.value = await api.getSvnStatus()
  } catch (e) {
    svnStatus.value = { available: false, path: '', version: '', message: errText(e) }
  }
}

function onProjectDeleted(id) {
  if (currentProjectId.value === id) currentProjectId.value = null
  loadProjects()
}

function goSvn(featureId) {
  svnTargetFeature.value = typeof featureId === 'number' ? featureId : 0
  tab.value = 'svn'
}

onMounted(() => {
  loadSvnStatus()
  loadProjects()
})
</script>

<template>
  <div class="app">
    <aside class="sidebar">
      <div class="brand">
        <img class="brand-logo" :src="appIcon" alt="SVN版本管理" />
        <div>
          <h1>SVN版本管理</h1>
          <p>SVN 变更记录与导出</p>
        </div>
      </div>

      <el-menu :default-active="tab" class="side-menu" @select="(key) => (tab = key)">
        <el-menu-item v-for="item in navs" :key="item.key" :index="item.key">
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </el-menu-item>
      </el-menu>
    </aside>

    <main class="main">
      <header class="topbar">
        <div class="project-picker">
          <span class="picker-icon"><el-icon><Folder /></el-icon></span>
          <el-select
            v-model="currentProjectId"
            placeholder="选择项目"
            no-data-text="还没有项目，先去「项目管理」添加"
            style="width: 260px"
          >
            <el-option v-for="p in projects" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </div>
        <div class="topbar-right">
          <el-button text @click="settingsOpen = true">
            <el-icon><Setting /></el-icon>
            <span>设置</span>
          </el-button>
        </div>
      </header>

      <section class="content">
        <ProjectsView
          v-if="tab === 'projects'"
          :projects="projects"
          :notify="notify"
          @refresh="loadProjects"
          @select="loadProjects($event)"
          @deleted="onProjectDeleted"
        />
        <FeaturesView
          v-else-if="tab === 'features'"
          :project="currentProject"
          :notify="notify"
          @go-svn="goSvn"
        />
        <SvnView
          v-else-if="tab === 'svn'"
          :project="currentProject"
          :notify="notify"
          :preselect-feature="svnTargetFeature"
          @go-features="tab = 'features'"
        />
        <ExportView v-else :project="currentProject" :notify="notify" />
      </section>

      <footer class="statusbar" title="打开设置" @click="settingsOpen = true">
        <span class="status-dot" :class="{ ok: svnStatus.available }"></span>
        <span>{{ svnStatus.available ? `SVN ${svnStatus.version} 已就绪` : 'SVN 未配置，点击设置' }}</span>
        <span class="status-sep"></span>
        <span class="status-path">{{ currentProject ? currentProject.path : '未选择项目' }}</span>
        <span class="status-meta">{{ projects.length }} 个项目</span>
      </footer>
    </main>

    <SettingsModal
      v-if="settingsOpen"
      :status="svnStatus"
      :notify="notify"
      @close="settingsOpen = false"
      @saved="svnStatus = $event"
    />
  </div>
</template>
