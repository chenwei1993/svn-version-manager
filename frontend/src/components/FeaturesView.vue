<script setup>
import { computed, reactive, ref, watch } from 'vue'
import AppIcon from './AppIcon.vue'
import {
  IconPlus as Plus,
  IconSearch as Search,
} from './icons'
import { api } from '../api'
import { errText, statusList } from '../utils'
import FeatureDetail from './FeatureDetail.vue'

const props = defineProps({
  project: { type: Object, default: null },
  notify: { type: Function, required: true },
})
const emit = defineEmits(['go-svn'])

const features = ref([])
const loading = ref(false)
const keyword = ref('')
const statusFilter = ref('')
const selectedId = ref(0)
const createOpen = ref(false)
const saving = ref(false)
const form = reactive({ name: '', version: '', description: '' })

const filtered = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  return features.value.filter((f) => {
    if (statusFilter.value && f.status !== statusFilter.value) return false
    if (!k) return true
    return (
      f.name.toLowerCase().includes(k) ||
      (f.version || '').toLowerCase().includes(k) ||
      (f.description || '').toLowerCase().includes(k)
    )
  })
})

const statusCounts = computed(() => {
  const map = {}
  for (const f of features.value) map[f.status] = (map[f.status] || 0) + 1
  return map
})

const statusStats = computed(() =>
  statusList
    .filter((s) => statusCounts.value[s])
    .map((s) => ({ label: s, count: statusCounts.value[s], cls: rowStatusClass(s) }))
)

const statusChips = computed(() => statusList.filter((s) => statusCounts.value[s]))

const doneCount = computed(
  () => (statusCounts.value['已完成'] || 0) + (statusCounts.value['已发布'] || 0)
)
const progress = computed(() =>
  features.value.length ? Math.round((doneCount.value / features.value.length) * 100) : 0
)
const currentVersion = computed(() => {
  const versions = [...new Set(features.value.map((f) => f.version).filter(Boolean))]
  versions.sort((a, b) => a.localeCompare(b, undefined, { numeric: true }))
  return versions[versions.length - 1] || '—'
})
const dbTotal = computed(() => features.value.reduce((n, f) => n + (f.dbChangeCount || 0), 0))
const fileTotal = computed(() => features.value.reduce((n, f) => n + (f.fileCount || 0), 0))

function rowStatusClass(status) {
  if (status === '已完成' || status === '已发布') return 'done'
  if (status === '已暂停') return 'archived'
  return 'dev'
}

function shortTime(value) {
  return (value || '').slice(5, 16) || '—'
}

async function load() {
  if (!props.project) {
    features.value = []
    selectedId.value = 0
    return
  }
  loading.value = true
  try {
    features.value = await api.listFeatures(props.project.id)
    if (selectedId.value && !features.value.some((f) => f.id === selectedId.value)) {
      selectedId.value = 0
    }
  } catch (e) {
    props.notify('读取功能记录失败：' + errText(e), 'error')
  } finally {
    loading.value = false
  }
}

watch(() => props.project?.id, () => {
  selectedId.value = 0
  statusFilter.value = ''
  keyword.value = ''
  load()
}, { immediate: true })

function openFeature(f) {
  selectedId.value = f.id
}

function openCreate() {
  form.name = ''
  form.version = ''
  form.description = ''
  createOpen.value = true
}

async function createFeature() {
  if (!form.name.trim()) return props.notify('请填写功能名称', 'error')
  saving.value = true
  try {
    const f = await api.createFeature(
      props.project.id,
      form.name.trim(),
      form.version.trim(),
      form.description.trim()
    )
    props.notify('功能「' + f.name + '」已创建')
    createOpen.value = false
    await load()
    selectedId.value = f.id
  } catch (e) {
    props.notify(errText(e), 'error')
  } finally {
    saving.value = false
  }
}

function onDeleted() {
  selectedId.value = 0
  load()
}

function onGoSvn(featureId) {
  emit('go-svn', featureId)
}
</script>

<template>
  <FeatureDetail
    v-if="project && selectedId"
    :key="selectedId"
    :feature-id="selectedId"
    :project="project"
    :notify="notify"
    @back="selectedId = 0"
    @changed="load"
    @deleted="onDeleted"
    @go-svn="onGoSvn"
  />

  <div v-else class="view">
    <div class="view-head">
      <div class="view-title">
        <div class="title-row">
          <h2>功能记录</h2>
          <span v-if="features.length" class="count-badge">{{ features.length }} 个功能</span>
        </div>
        <p>按功能组织 SVN 提交、数据库变更与导出内容</p>
      </div>
      <div class="actions">
        <el-button type="primary" :icon="Plus" :disabled="!project" @click="openCreate">新建功能</el-button>
      </div>
    </div>

    <div v-if="!project" class="panel empty-state">
      <div class="empty-icon"><AppIcon name="search" :size="20" /></div>
      <h4>请先选择项目</h4>
      <p>在顶部选择一个项目，或先到「项目管理」添加工作副本</p>
    </div>

    <template v-else>
      <div class="feature-layout">
        <div class="panel feature-list-panel">
          <div class="list-toolbar">
            <div class="chips">
              <button class="chip" :class="{ active: statusFilter === '' }" @click="statusFilter = ''">
                全部 {{ features.length }}
              </button>
              <button
                v-for="s in statusChips"
                :key="s"
                class="chip"
                :class="{ active: statusFilter === s }"
                @click="statusFilter = s"
              >
                {{ s }} {{ statusCounts[s] || 0 }}
              </button>
            </div>
            <div class="toolbar-right">
              <el-input
                v-model="keyword"
                :prefix-icon="Search"
                placeholder="搜索功能名称"
                clearable
                style="width: 188px"
              />
            </div>
          </div>

          <div v-if="loading" class="empty-state">
            <p>正在读取功能记录…</p>
          </div>

          <div v-else-if="!features.length" class="empty-state">
            <div class="empty-icon"><AppIcon name="plus" :size="20" /></div>
            <h4>还没有功能记录</h4>
            <p>新建一个功能，开始关联 SVN 提交与数据库变更</p>
            <el-button type="primary" :icon="Plus" @click="openCreate">新建功能</el-button>
          </div>

          <div v-else-if="!filtered.length" class="empty-state">
            <div class="empty-icon"><AppIcon name="search" :size="20" /></div>
            <h4>没有匹配的功能</h4>
            <p>换个关键词或状态筛选试试</p>
          </div>

          <div v-else class="feature-rows">
            <button
              v-for="f in filtered"
              :key="f.id"
              class="feature-row"
              @click="openFeature(f)"
            >
              <span class="feature-accent" :class="rowStatusClass(f.status)"></span>
              <span class="feature-main">
                <span class="feature-name">
                  {{ f.name }}
                  <el-tag v-if="f.autoSync" type="success" size="small">自动同步</el-tag>
                </span>
                <span class="feature-desc">{{ f.description || '暂无功能说明' }}</span>
              </span>
              <span class="feature-version">{{ f.version || '—' }}</span>
              <span class="status-tag" :class="rowStatusClass(f.status)">{{ f.status }}</span>
              <span class="feature-meta">
                数据库 {{ f.dbChangeCount || 0 }} · 提交 {{ f.revisionCount || 0 }}
              </span>
              <span class="feature-time">{{ shortTime(f.updatedAt) }}</span>
              <AppIcon class="feature-chevron" name="chevronRight" :size="14" />
            </button>
          </div>

          <div class="list-footer">
            <span>显示 {{ filtered.length }} / {{ features.length }} 个功能</span>
            <span>点击功能行查看数据库变更、提交记录与导出</span>
          </div>
        </div>

        <aside class="panel feature-aside">
          <div class="aside-head">
            <h3>本迭代概览</h3>
            <span class="mono">{{ currentVersion }}</span>
          </div>
          <div class="progress-row">
            <span class="big">{{ progress }}%</span>
            <span class="hint">完成度</span>
          </div>
          <div class="progress-track">
            <span :style="{ width: progress + '%' }"></span>
          </div>
          <div class="aside-stats">
            <div v-for="s in statusStats" :key="s.label" class="aside-stat">
              <span class="dot" :class="s.cls"></span>
              <span>{{ s.label }}</span>
              <strong>{{ s.count }}</strong>
            </div>
          </div>
          <div class="aside-divider"></div>
          <div class="aside-meta"><span>数据库变更</span><strong>{{ dbTotal }} 条</strong></div>
          <div class="aside-meta"><span>关联文件</span><strong>{{ fileTotal }} 个</strong></div>
          <div class="aside-tip">
            在功能详情中可直接从项目数据库导入表和字段，自动带出类型与注释。
          </div>
        </aside>
      </div>
    </template>

    <el-dialog v-model="createOpen" title="新建功能记录" width="560px" :close-on-click-modal="false">
      <el-form label-position="top">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="功能名称">
              <el-input v-model="form.name" placeholder="例如：订单批量导入" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="版本 / 迭代">
              <el-input v-model="form.version" placeholder="例如：v2.3.0" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="功能说明">
              <el-input
                v-model="form.description"
                type="textarea"
                :rows="4"
                placeholder="这次功能做了什么、影响范围、注意事项..."
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="createOpen = false">取消</el-button>
        <el-button type="primary" :loading="saving" :icon="Plus" @click="createFeature">创建</el-button>
      </template>
    </el-dialog>
  </div>
</template>
