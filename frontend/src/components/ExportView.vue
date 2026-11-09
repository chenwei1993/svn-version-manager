<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { Download, FolderOpened, Search } from '@element-plus/icons-vue'
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

const filtered = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  if (!k) return features.value
  return features.value.filter(
    (f) => f.name.toLowerCase().includes(k) || (f.version || '').toLowerCase().includes(k)
  )
})

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
  try {
    const s = await api.getAppSettings()
    if (s.defaultExportDir && !outDir.value) outDir.value = s.defaultExportDir
  } catch {
    // 用户仍可手动选择目录
  }
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
        <h2>一键导出</h2>
        <p>选择功能和导出目录，按关联的 SVN 版本导出代码快照，自动附带变更清单。</p>
      </div>
    </div>

    <el-card v-if="!project" shadow="never" class="panel">
      <el-empty description="请先选择项目" />
    </el-card>

    <template v-else>
      <el-card shadow="never" class="panel toolbar-card">
        <div class="toolbar">
          <div class="toolbar-left" style="flex: 1">
            <el-input v-model="outDir" placeholder="选择导出目录..." style="flex: 1; min-width: 260px">
              <template #append>
                <el-button :icon="FolderOpened" @click="chooseDir">选择目录</el-button>
              </template>
            </el-input>
            <el-input
              v-model="keyword"
              :prefix-icon="Search"
              placeholder="搜索功能"
              clearable
              style="width: 220px"
            />
          </div>
        </div>
        <p class="hint" style="margin: 8px 0 0">
          导出的是每个文件在关联提交中的版本快照，不受当前工作副本改动影响。需要挑选文件时，请进入功能详情的「导出」标签页。
        </p>
      </el-card>

      <el-card v-if="loading" v-loading="true" shadow="never" class="panel" style="height: 180px; margin-top: 14px" />

      <el-card v-else-if="!features.length" shadow="never" class="panel" style="margin-top: 14px">
        <el-empty description="还没有可导出的功能">
          <span class="hint">先到「功能记录」新建功能，再到「SVN 提交」关联提交。</span>
        </el-empty>
      </el-card>

      <el-card v-else shadow="never" class="panel table-panel" style="margin-top: 14px">
        <el-table :data="filtered" style="width: 100%">
          <el-table-column label="功能" min-width="240">
            <template #default="{ row }">
              <div class="cell-title">{{ row.name }}</div>
              <div v-if="row.description" class="cell-sub">{{ row.description }}</div>
            </template>
          </el-table-column>
          <el-table-column label="版本 / 迭代" width="140">
            <template #default="{ row }">
              <span class="muted">{{ row.version || '—' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="statusTagType(row.status)" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="170">
            <template #default="{ row }"><span class="hint">{{ row.updatedAt }}</span></template>
          </el-table-column>
          <el-table-column label="操作" width="130" align="right">
            <template #default="{ row }">
              <el-button
                type="primary"
                size="small"
                :icon="Download"
                :loading="exportingId === row.id"
                :disabled="!!exportingId"
                @click="exportOne(row)"
              >
                导出全部
              </el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="!filtered.length" description="没有匹配的功能" />
      </el-card>

      <el-card v-if="result" shadow="never" class="result-card">
        <div class="cell-title" style="margin-bottom: 10px">「{{ resultFeature }}」导出完成</div>
        <el-descriptions :column="1" border size="small">
          <el-descriptions-item label="导出目录">{{ result.folder }}</el-descriptions-item>
          <el-descriptions-item label="压缩包">{{ result.zipPath }}</el-descriptions-item>
          <el-descriptions-item label="文件数量">{{ result.fileCount }} 个</el-descriptions-item>
        </el-descriptions>
        <div v-if="result.skipped && result.skipped.length" class="hint" style="margin-top: 8px">
          已跳过：{{ result.skipped.join('；') }}
        </div>
        <div class="actions" style="margin-top: 12px">
          <el-button :icon="FolderOpened" @click="openPath(result.folder)">打开导出目录</el-button>
          <el-button :icon="Download" @click="openPath(result.zipPath)">定位 ZIP</el-button>
        </div>
      </el-card>
    </template>
  </div>
</template>
