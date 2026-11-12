<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import appIcon from './assets/app-icon.png'
import { api } from './api'
import { errText } from './utils'
import AppIcon from './components/AppIcon.vue'
import ProjectsView from './components/ProjectsView.vue'
import FeaturesView from './components/FeaturesView.vue'
import SvnView from './components/SvnView.vue'
import ExportView from './components/ExportView.vue'
import SettingsModal from './components/SettingsModal.vue'

const navs = [
  { key: 'projects', label: '项目管理', icon: 'grid' },
  { key: 'features', label: '功能记录', icon: 'list' },
  { key: 'svn', label: 'SVN 提交', icon: 'branch' },
  { key: 'export', label: '代码导出', icon: 'download' },
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
        <img class="brand-logo" :src="appIcon" alt="SVN 版本管理" />
        <div class="brand-text">
          <h1>SVN 版本管理</h1>
          <p>v1.2 · 本地工作台</p>
        </div>
      </div>

      <div class="nav-label">工作区</div>
      <nav class="nav">
        <button
          v-for="item in navs"
          :key="item.key"
          class="nav-item"
          :class="{ active: tab === item.key }"
          @click="tab = item.key"
        >
          <AppIcon :name="item.icon" :size="17" />
          <span>{{ item.label }}</span>
        </button>
      </nav>

      <div class="nav-label secondary">配置</div>
      <nav class="nav">
        <button class="nav-item" @click="settingsOpen = true">
          <AppIcon name="settings" :size="17" />
          <span>设置</span>
        </button>
      </nav>

      <div class="sidebar-spacer"></div>

      <div class="sidebar-status" title="打开设置" @click="settingsOpen = true">
        <div class="row">
          <span class="dot" :class="{ warn: !svnStatus.available }"></span>
          <span class="status-text" :class="{ warn: !svnStatus.available }">
            {{ svnStatus.available ? 'SVN 连接正常' : 'SVN 未配置' }}
          </span>
        </div>
        <div class="repo">{{ currentProject?.repoRoot || currentProject?.path || '尚未选择项目' }}</div>
        <div class="divider"></div>
        <div class="meta">
          {{ currentProject ? currentProject.relative || '/' : '—' }} · {{ projects.length }} 个项目
        </div>
      </div>
    </aside>

    <main class="main">
      <header class="topbar">
        <div class="project-picker">
          <span class="picker-icon"><AppIcon name="building" :size="16" /></span>
          <el-select
            v-model="currentProjectId"
            placeholder="选择项目"
            no-data-text="还没有项目，先去「项目管理」添加"
            style="width: 190px"
          >
            <el-option v-for="p in projects" :key="p.id" :label="p.name" :value="p.id" />
          </el-select>
        </div>
        <div class="topbar-right">
          <button
            class="status-pill"
            :class="{ warn: !svnStatus.available }"
            :title="svnStatus.available ? `SVN ${svnStatus.version}` : svnStatus.message"
            @click="settingsOpen = true"
          >
            <span class="status-dot" :class="{ warn: !svnStatus.available }"></span>
            {{ svnStatus.available ? 'SVN 已就绪' : 'SVN 未配置' }}
          </button>
          <button class="icon-button" title="设置" @click="settingsOpen = true">
            <AppIcon name="settings" :size="16" />
          </button>
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
