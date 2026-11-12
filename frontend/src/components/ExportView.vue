<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import AppIcon from './AppIcon.vue'
import {
  IconDownload as Download,
  IconFile as Document,
  IconFolder as FolderOpened,
  IconSearch as Search,
} from './icons'
import { api } from '../api'
import { errText, statusTagType } from '../utils'

const props = defineProps({
  project: { type: Object, default: null },
  notify: { type: Function, required: true },
})

const features = ref([])
const loading = ref(false)
const keyword = ref('')
const outDir = ref('')
const exportingId = ref(0)
const result = ref(null)
const resultFeature = ref('')
const viewportWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1280)
const compact = computed(() => viewportWidth.value >= 1180 && viewportWidth.value < 1320)

function onResize() {
  viewportWidth.value = window.innerWidth
}

const filtered = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  if (!k) return features.value
  return features.value.filter(
    (f) => f.name.toLowerCase().includes(k) || (f.version || '').toLowerCase().includes(k)
  )
})

function shortTime(value) {
  return (value || '').slice(5, 16) || '—'
}

async function load() {
  if (!props.project) {
    features.value = []
    return
  }
  loading.value = true
  try {
    features.value = await api.listFeatures(props.project.id)
  } catch (e) {
    props.notify('读取功能列表失败：' + errText(e), 'error')
  } finally {
    loading.value = false
  }
}

watch(
  () => props.project?.id,
  () => {
    result.value = null
    load()
  },
  { immediate: true }
)

onMounted(async () => {
  window.addEventListener('resize', onResize)
  try {
    const s = await api.getAppSettings()
    if (s.defaultExportDir && !outDir.value) outDir.value = s.defaultExportDir
  } catch {
    // 用户仍可手动选择目录
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', onResize)
})

async function chooseDir() {
  try {
    const dir = await api.pickDirectory('选择代码导出目录')
    if (dir) {
      outDir.value = dir
      api.setDefaultExportDir(dir).catch(() => {})
    }
  } catch (e) {
    props.notify('选择目录失败：' + errText(e), 'error')
  }
}

async function exportOne(f) {
  if (!outDir.value) {
    await chooseDir()
    if (!outDir.value) return
  }
  exportingId.value = f.id
  result.value = null
  try {
    const res = await api.exportFeature(f.id, outDir.value)
    result.value = res
    resultFeature.value = f.name
    props.notify(`「${f.name}」导出完成：${res.fileCount} 个文件`)
  } catch (e) {
    props.notify(errText(e), 'error')
  } finally {
    exportingId.value = 0
  }
}

async function openPath(p) {
  try {
    await api.openFolder(p)
  } catch (e) {
    props.notify(errText(e), 'error')
  }
}
</script>

<template>
  <div class="view">
    <div class="view-head">
      <div class="view-title">
        <div class="title-row">
          <h2>代码导出</h2>
          <span v-if="project && features.length" class="count-badge">{{ features.length }} 个功能</span>
        </div>
        <p>按关联提交从 SVN 导出代码，并生成变更清单与数据库变更 SQL</p>
      </div>
      <div v-if="outDir" class="actions">
        <el-button :icon="FolderOpened" @click="openPath(outDir)">打开导出目录</el-button>
      </div>
    </div>

    <div v-if="!project" class="panel empty-state">
      <div class="empty-icon"><AppIcon name="download" :size="20" /></div>
      <h4>请先选择项目</h4>
      <p>在顶部选择项目后，这里会列出可导出的功能</p>
    </div>

    <template v-else>
      <div class="feature-layout export-layout">
        <div class="panel feature-list-panel">
          <div class="list-toolbar">
            <div class="title-row">
              <h3 style="margin: 0; font-size: 13.5px; font-weight: 600">可导出功能</h3>
              <span class="count-badge">{{ filtered.length }}</span>
            </div>
            <el-input
              v-model="keyword"
              :prefix-icon="Search"
              placeholder="搜索功能"
              clearable
              style="width: 200px"
            />
          </div>

          <el-table v-if="filtered.length" :data="filtered" style="width: 100%">
            <el-table-column label="功能" min-width="220">
              <template #default="{ row }">
                <div class="cell-title">{{ row.name }}</div>
                <div v-if="row.description" class="cell-sub">{{ row.description }}</div>
              </template>
            </el-table-column>
            <el-table-column label="版本" width="90">
              <template #default="{ row }">
                <span class="mono" style="color: var(--pm-text-2)">{{ row.version || '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="statusTagType(row.status)" size="small">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column v-if="!compact" label="文件 / 提交" width="110">
              <template #default="{ row }">
                <span class="hint">{{ row.fileCount || 0 }} / {{ row.revisionCount || 0 }}</span>
              </template>
            </el-table-column>
            <el-table-column v-if="!compact" label="更新时间" width="130">
              <template #default="{ row }"><span class="hint">{{ shortTime(row.updatedAt) }}</span></template>
            </el-table-column>
            <el-table-column label="操作" width="100" align="right">
              <template #default="{ row }">
                <el-button
                  type="primary"
                  size="small"
                  :icon="Download"
                  :loading="exportingId === row.id"
                  :disabled="!!exportingId"
                  @click="exportOne(row)"
                >
                  导出
                </el-button>
              </template>
            </el-table-column>
            <template #empty>
              <div class="empty-state" style="padding: 36px 20px">
                <div class="empty-icon"><AppIcon name="search" :size="20" /></div>
                <h4>没有匹配的功能</h4>
                <p>换个关键词试试</p>
              </div>
            </template>
          </el-table>

          <div v-else-if="loading" class="empty-state" style="padding: 46px 20px">
            <p>正在读取功能列表…</p>
          </div>

          <div v-else class="empty-state">
            <div class="empty-icon"><AppIcon name="download" :size="20" /></div>
            <h4>还没有可导出的功能</h4>
            <p>先到「功能记录」新建功能，再到「SVN 提交」关联提交</p>
          </div>
        </div>

        <aside class="panel feature-aside" style="width: 340px; flex: 0 0 340px">
          <div class="aside-head">
            <h3>导出设置</h3>
            <span class="mono">ZIP</span>
          </div>
          <div class="field-label">输出目录</div>
          <el-input v-model="outDir" placeholder="选择导出目录...">
            <template #append>
              <el-button :icon="FolderOpened" @click="chooseDir">选择</el-button>
            </template>
          </el-input>
          <div class="aside-divider"></div>
          <div class="aside-meta"><span>可导出功能</span><strong>{{ features.length }} 个</strong></div>
          <div class="aside-meta"><span>导出方式</span><strong>按功能整体导出</strong></div>
          <div class="aside-tip">
            导出的是每个文件在关联提交中的版本快照，不受当前工作副本改动影响。如登记了数据库变更，会自动生成「数据库变更.sql」。
          </div>
          <el-button :icon="FolderOpened" :disabled="!outDir" @click="openPath(outDir)">
            打开导出目录
          </el-button>
        </aside>
      </div>

      <div v-if="result" class="result-card">
        <div class="cell-title" style="margin-bottom: 10px">「{{ resultFeature }}」导出完成</div>
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="导出目录">{{ result.folder }}</el-descriptions-item>
          <el-descriptions-item label="压缩包">{{ result.zipPath }}</el-descriptions-item>
          <el-descriptions-item label="代码文件">{{ result.fileCount }} 个</el-descriptions-item>
          <el-descriptions-item v-if="result.sqlPath" label="SQL 脚本">{{ result.sqlPath }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="result.skipped && result.skipped.length" class="hint" style="margin-top: 8px">
          已跳过：{{ result.skipped.join('；') }}
        </div>
        <div class="actions" style="margin-top: 12px">
          <el-button :icon="FolderOpened" @click="openPath(result.folder)">打开导出目录</el-button>
          <el-button v-if="result.sqlPath" :icon="Document" @click="openPath(result.sqlPath)">打开 SQL</el-button>
          <el-button :icon="Download" @click="openPath(result.zipPath)">定位 ZIP</el-button>
        </div>
      </div>
    </template>
  </div>
</template>
