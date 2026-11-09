<script setup>
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { ElMessageBox } from 'element-plus'
import {
  ArrowLeft,
  Clock,
  Delete,
  Download,
  Edit,
  FolderOpened,
  Plus,
  Refresh,
} from '@element-plus/icons-vue'
import { api } from '../api'
import {
  actionTagType,
  errText,
  kindLabel,
  statusList,
  statusTagType,
} from '../utils'

const props = defineProps({
  featureId: { type: Number, required: true },
  project: { type: Object, default: null },
  notify: { type: Function, required: true },
})
const emit = defineEmits(['back', 'changed', 'deleted', 'go-svn'])

const detail = ref(null)
const loading = ref(false)
const saving = ref(false)
const activeTab = ref('info')

const autoSync = ref(false)
const syncing = ref(false)
const syncMessage = ref('')
const myAuthor = ref('')
const authorLoaded = ref(false)
let autoSynced = false

const form = reactive({ name: '', version: '', status: '开发中', description: '' })

const emptyChange = () => ({ kind: 'add_column', tableName: '', columnName: '', dataType: '', remark: '' })
const change = reactive(emptyChange())
const editingChangeId = ref(0)
const changeModalOpen = ref(false)

const exportDir = ref('')
const exporting = ref(false)
const exportResult = ref(null)
const exportTableRef = ref(null)
const selectedPaths = ref([])

const exportFiles = computed(() => {
  const map = new Map()
  for (const f of detail.value?.files || []) {
    const old = map.get(f.path)
    if (!old || f.revision > old.revision) map.set(f.path, f)
  }
  return [...map.values()].sort((a, b) => a.path.localeCompare(b.path))
})

const revFileCounts = computed(() => {
  const map = {}
  for (const f of detail.value?.files || []) {
    map[f.revision] = (map[f.revision] || 0) + 1
  }
  return map
})

function filesByRevision(revision) {
  return (detail.value?.files || []).filter((f) => f.revision === revision)
}

function selectAllFiles() {
  const table = exportTableRef.value
  if (!table) return
  table.clearSelection()
  nextTick(() => {
    exportFiles.value.forEach((row) => table.toggleRowSelection(row, true))
  })
}

function clearFileSelection() {
  exportTableRef.value?.clearSelection()
}

function onSelectionChange(rows) {
  selectedPaths.value = rows.map((r) => r.path)
}

watch(activeTab, (tab) => {
  if (tab === 'export') nextTick(selectAllFiles)
})

async function load() {
  loading.value = true
  try {
    const d = await api.getFeatureDetail(props.featureId)
    detail.value = d
    form.name = d.feature.name
    form.version = d.feature.version
    form.status = d.feature.status
    form.description = d.feature.description
    autoSync.value = d.feature.autoSync
    if (d.feature.autoSync && !autoSynced) {
      autoSynced = true
      syncNow(true)
    }
    if (activeTab.value === 'export') nextTick(selectAllFiles)
  } catch (e) {
    props.notify('读取功能详情失败：' + errText(e), 'error')
  } finally {
    loading.value = false
  }
}

watch(() => props.featureId, () => {
  activeTab.value = 'info'
  changeModalOpen.value = false
  selectedPaths.value = []
  autoSync.value = false
  syncMessage.value = ''
  autoSynced = false
  load()
}, { immediate: true })

onMounted(async () => {
  try {
    const s = await api.getAppSettings()
    if (s.defaultExportDir && !exportDir.value) exportDir.value = s.defaultExportDir
    myAuthor.value = s.svnAuthor || ''
  } catch {
    // 忽略设置读取失败，用户仍可手动选择目录
  } finally {
    authorLoaded.value = true
  }
})

async function toggleAutoSync(enabled) {
  try {
    const f = await api.setFeatureAutoSync(props.featureId, enabled)
    detail.value.feature = f
    autoSync.value = f.autoSync
    props.notify(enabled ? '已开启自动同步' : '已关闭自动同步')
    if (enabled) {
      autoSynced = true
      await syncNow(false)
    }
  } catch (e) {
    autoSync.value = !enabled
    props.notify(errText(e), 'error')
  }
}

async function syncNow(silent = false) {
  if (syncing.value) return
  syncing.value = true
  syncMessage.value = ''
  try {
    const res = await api.syncFeature(props.featureId)
    if (detail.value?.feature) detail.value.feature.lastSyncRevision = res.lastRevision
    if (res.added > 0) {
      syncMessage.value = res.message
      props.notify(res.message)
      await load()
      emit('changed')
    } else if (!silent) {
      syncMessage.value = res.message
      props.notify(res.message || '没有新提交', 'info')
    }
  } catch (e) {
    syncMessage.value = errText(e)
    if (!silent) props.notify(errText(e), 'error')
  } finally {
    syncing.value = false
  }
}

async function saveInfo() {
  if (!form.name.trim()) return props.notify('功能名称不能为空', 'error')
  saving.value = true
  try {
    await api.updateFeature({
      ...detail.value.feature,
      name: form.name.trim(),
      version: form.version.trim(),
      status: form.status,
      description: form.description.trim(),
    })
    props.notify('功能信息已保存')
    await load()
    emit('changed')
  } catch (e) {
    props.notify(errText(e), 'error')
  } finally {
    saving.value = false
  }
}

async function removeFeature() {
  try {
    await ElMessageBox.confirm(
      `关联的提交、数据库变更记录也会一起删除。`,
      `确定删除功能「${detail.value.feature.name}」？`,
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  try {
    await api.deleteFeature(props.featureId)
    props.notify('功能已删除')
    emit('deleted')
  } catch (e) {
    props.notify(errText(e), 'error')
  }
}

function resetChange() {
  Object.assign(change, emptyChange())
  editingChangeId.value = 0
}

function openAddChange() {
  resetChange()
  changeModalOpen.value = true
}

function openEditChange(c) {
  editingChangeId.value = c.id
  change.kind = c.kind
  change.tableName = c.tableName
  change.columnName = c.columnName
  change.dataType = c.dataType
  change.remark = c.remark
  changeModalOpen.value = true
}

async function saveChange() {
  if (!change.tableName.trim()) return props.notify('请填写表名', 'error')
  try {
    const payload = {
      id: editingChangeId.value,
      featureId: props.featureId,
      kind: change.kind,
      tableName: change.tableName.trim(),
      columnName: change.columnName.trim(),
      dataType: change.dataType.trim(),
      remark: change.remark.trim(),
    }
    if (editingChangeId.value) {
      await api.updateDBChange(payload)
      props.notify('数据库变更已更新')
    } else {
      await api.addDBChange(payload)
      props.notify('数据库变更已添加')
    }
    changeModalOpen.value = false
    resetChange()
    await load()
    emit('changed')
  } catch (e) {
    props.notify(errText(e), 'error')
  }
}

async function removeChange(c) {
  try {
    await ElMessageBox.confirm(
      `${kindLabel(c.kind)} ${c.tableName}${c.columnName ? '.' + c.columnName : ''}`,
      '删除这条数据库变更记录？',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  try {
    await api.deleteDBChange(c.id)
    props.notify('已删除')
    await load()
    emit('changed')
  } catch (e) {
    props.notify(errText(e), 'error')
  }
}

async function unlinkRevision(r) {
  try {
    await ElMessageBox.confirm(
      `该提交涉及的文件会从本功能中移除，仓库不受影响。`,
      `取消关联 r${r.revision}？`,
      { type: 'warning', confirmButtonText: '取消关联', cancelButtonText: '返回' }
    )
  } catch {
    return
  }
  try {
    await api.unlinkRevision(props.featureId, r.revision)
    props.notify('已取消关联 r' + r.revision)
    await load()
    emit('changed')
  } catch (e) {
    props.notify(errText(e), 'error')
  }
}

async function chooseDir() {
  try {
    const dir = await api.pickDirectory('选择代码导出目录')
    if (dir) {
      exportDir.value = dir
      api.setDefaultExportDir(dir).catch(() => {})
    }
  } catch (e) {
    props.notify('选择目录失败：' + errText(e), 'error')
  }
}

async function doExport() {
  if (!exportDir.value) {
    await chooseDir()
    if (!exportDir.value) return
  }
  if (selectedPaths.value.length === 0) {
    return props.notify('请至少勾选一个要导出的文件', 'error')
  }
  exporting.value = true
  exportResult.value = null
  try {
    const res = await api.exportFeature(props.featureId, exportDir.value, [...selectedPaths.value])
    exportResult.value = res
    props.notify(`导出完成：${res.fileCount} 个文件`)
  } catch (e) {
    props.notify(errText(e), 'error')
  } finally {
    exporting.value = false
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
  <div v-if="loading && !detail" class="view">
    <el-card v-loading="true" shadow="never" class="panel" style="height: 220px" />
  </div>

  <div v-else-if="detail" class="view detail-page">
    <div class="detail-topbar">
      <el-button :icon="ArrowLeft" circle plain title="返回功能列表" @click="emit('back')" />
      <div class="detail-title">
        <h2>{{ detail.feature.name }}</h2>
        <div class="detail-sub">
          <el-tag :type="statusTagType(detail.feature.status)" size="small">{{ detail.feature.status }}</el-tag>
          <span>{{ project?.name }}</span>
          <span v-if="detail.feature.version">· {{ detail.feature.version }}</span>
          <span>· 更新于 {{ detail.feature.updatedAt }}</span>
        </div>
      </div>
      <div class="actions">
        <el-button type="danger" plain :icon="Delete" @click="removeFeature">删除</el-button>
      </div>
    </div>

    <el-card shadow="never" class="panel detail-card">
      <el-tabs v-model="activeTab" class="detail-tabs">
        <el-tab-pane label="基本信息" name="info">
          <div class="section-head">
            <div>
              <h4>基本信息</h4>
              <p class="hint">功能名称、版本和说明会写入导出的变更清单。</p>
            </div>
            <el-button type="primary" :loading="saving" @click="saveInfo">保存</el-button>
          </div>
          <el-form label-position="top">
            <el-row :gutter="16">
              <el-col :span="8">
                <el-form-item label="功能名称">
                  <el-input v-model="form.name" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="版本 / 迭代">
                  <el-input v-model="form.version" placeholder="v2.3.0" />
                </el-form-item>
              </el-col>
              <el-col :span="8">
                <el-form-item label="状态">
                  <el-select v-model="form.status" style="width: 100%">
                    <el-option v-for="s in statusList" :key="s" :label="s" :value="s" />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="24">
                <el-form-item label="功能说明">
                  <el-input v-model="form.description" type="textarea" :rows="4" placeholder="功能说明、影响范围、注意事项..." />
                </el-form-item>
              </el-col>
            </el-row>
          </el-form>
        </el-tab-pane>

        <el-tab-pane name="db">
          <template #label>
            <span>数据库变更</span>
            <el-tag v-if="detail.dbChanges.length" size="small" type="info" effect="plain" class="tab-tag">
              {{ detail.dbChanges.length }}
            </el-tag>
          </template>
          <div class="section-head">
            <div>
              <h4>数据库变更</h4>
              <p class="hint">登记新增表、字段或索引，导出时会自动写入变更清单。</p>
            </div>
            <el-button type="primary" :icon="Plus" @click="openAddChange">添加变更</el-button>
          </div>
          <el-table v-if="detail.dbChanges.length" :data="detail.dbChanges" style="width: 100%">
            <el-table-column label="类型" width="110">
              <template #default="{ row }">
                <el-tag :type="row.kind === 'add_table' ? 'success' : 'info'" size="small">
                  {{ kindLabel(row.kind) }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column label="表名" min-width="160">
              <template #default="{ row }"><span class="mono">{{ row.tableName }}</span></template>
            </el-table-column>
            <el-table-column label="字段名" min-width="140">
              <template #default="{ row }"><span class="mono">{{ row.columnName || '—' }}</span></template>
            </el-table-column>
            <el-table-column label="数据类型" width="130">
              <template #default="{ row }"><span class="mono">{{ row.dataType || '—' }}</span></template>
            </el-table-column>
            <el-table-column prop="remark" label="说明" min-width="180" show-overflow-tooltip />
            <el-table-column label="操作" width="100" align="right">
              <template #default="{ row }">
                <el-button link :icon="Edit" @click="openEditChange(row)" />
                <el-button link type="danger" :icon="Delete" @click="removeChange(row)" />
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-else description="还没有数据库变更">
            <el-button type="primary" :icon="Plus" @click="openAddChange">添加变更</el-button>
          </el-empty>
        </el-tab-pane>

        <el-tab-pane name="svn">
          <template #label>
            <span>SVN 提交</span>
            <el-tag v-if="detail.revisions.length" size="small" type="info" effect="plain" class="tab-tag">
              {{ detail.revisions.length }}
            </el-tag>
          </template>

          <div class="sync-bar">
            <div class="sync-main">
              <el-switch v-model="autoSync" :loading="syncing" @change="toggleAutoSync" />
              <span class="sync-label">自动同步新提交</span>
              <span class="hint">
                上次检查：{{ detail.feature.lastSyncRevision ? 'r' + detail.feature.lastSyncRevision : '—' }}
              </span>
              <el-tag v-if="authorLoaded && !myAuthor" type="warning" size="small">未配置 SVN 账号</el-tag>
            </div>
            <div class="sync-actions">
              <el-button :icon="Refresh" :loading="syncing" @click="syncNow(false)">同步新提交</el-button>
            </div>
          </div>
          <p class="hint" style="margin: -4px 0 14px">
            开启后，打开该功能时会自动把「我的 SVN 账号」在已关联版本之后的新提交加入本功能。建议只对当前开发的功能开启。
            <template v-if="authorLoaded && !myAuthor">请先在「设置 → SVN」中填写我的 SVN 账号。</template>
          </p>
          <el-alert
            v-if="syncMessage"
            type="info"
            :closable="false"
            show-icon
            :title="syncMessage"
            style="margin-bottom: 14px"
          />

          <div class="section-head">
            <div>
              <h4>关联提交</h4>
              <p class="hint">关联提交后会自动带出变更文件，导出时按对应版本取代码。</p>
            </div>
            <el-button :icon="Clock" @click="emit('go-svn', featureId)">去关联提交</el-button>
          </div>
          <el-table v-if="detail.revisions.length" :data="detail.revisions" row-key="revision" style="width: 100%">
            <el-table-column type="expand">
              <template #default="{ row }">
                <el-table :data="filesByRevision(row.revision)" size="small" style="width: 100%">
                  <el-table-column label="动作" width="80">
                    <template #default="{ row: file }">
                      <el-tag :type="actionTagType(file.action)" size="small">{{ file.action }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="文件" min-width="300">
                    <template #default="{ row: file }">
                      <span class="mono">{{ file.path.replace(/^\//, '') }}</span>
                    </template>
                  </el-table-column>
                </el-table>
              </template>
            </el-table-column>
            <el-table-column label="版本" width="100">
              <template #default="{ row }"><span class="mono">r{{ row.revision }}</span></template>
            </el-table-column>
            <el-table-column prop="author" label="作者" width="120" />
            <el-table-column prop="date" label="时间" width="170" />
            <el-table-column prop="message" label="提交说明" min-width="220" show-overflow-tooltip />
            <el-table-column label="文件" width="80" align="center">
              <template #default="{ row }">{{ revFileCounts[row.revision] || 0 }}</template>
            </el-table-column>
            <el-table-column label="操作" width="100" align="right">
              <template #default="{ row }">
                <el-button link type="danger" @click="unlinkRevision(row)">取消关联</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-else description="还没有关联提交">
            <el-button :icon="Clock" @click="emit('go-svn', featureId)">去关联提交</el-button>
          </el-empty>
        </el-tab-pane>

        <el-tab-pane name="files">
          <template #label>
            <span>变更文件</span>
            <el-tag v-if="detail.files.length" size="small" type="info" effect="plain" class="tab-tag">
              {{ detail.files.length }}
            </el-tag>
          </template>
          <div class="section-head">
            <div>
              <h4>变更文件</h4>
              <p class="hint">这些文件会按关联提交的版本导出。</p>
            </div>
          </div>
          <el-table v-if="detail.files.length" :data="detail.files" style="width: 100%">
            <el-table-column label="动作" width="80">
              <template #default="{ row }">
                <el-tag :type="actionTagType(row.action)" size="small">{{ row.action }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="文件" min-width="320">
              <template #default="{ row }">
                <span class="mono">{{ row.path.replace(/^\//, '') }}</span>
              </template>
            </el-table-column>
            <el-table-column label="版本" width="90">
              <template #default="{ row }"><span class="mono">r{{ row.revision }}</span></template>
            </el-table-column>
          </el-table>
          <el-empty v-else description="还没有变更文件">
            <span class="hint">在「SVN 提交」标签页关联提交后，文件会自动出现在这里。</span>
          </el-empty>
        </el-tab-pane>

        <el-tab-pane label="导出" name="export">
          <div class="section-head">
            <div>
              <h4>导出代码包</h4>
              <p class="hint">勾选要导出的文件，按关联提交的版本从 SVN 导出，并生成变更清单和 zip。</p>
            </div>
          </div>

          <template v-if="exportFiles.length">
            <el-table
              ref="exportTableRef"
              :data="exportFiles"
              row-key="path"
              style="width: 100%"
              @selection-change="onSelectionChange"
            >
              <el-table-column type="selection" width="46" />
              <el-table-column label="动作" width="80">
                <template #default="{ row }">
                  <el-tag :type="actionTagType(row.action)" size="small">{{ row.action }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="文件" min-width="320">
                <template #default="{ row }">
                  <span class="mono">{{ row.path.replace(/^\//, '') }}</span>
                </template>
              </el-table-column>
              <el-table-column label="版本" width="90">
                <template #default="{ row }"><span class="mono">r{{ row.revision }}</span></template>
              </el-table-column>
            </el-table>

            <div class="export-actions">
              <span class="hint">已选 {{ selectedPaths.length }} / {{ exportFiles.length }} 个文件</span>
              <el-button size="small" @click="selectAllFiles">全选</el-button>
              <el-button size="small" @click="clearFileSelection">清空</el-button>
            </div>

            <div class="export-bar">
              <el-input v-model="exportDir" placeholder="选择导出目录..." />
              <el-button :icon="FolderOpened" @click="chooseDir">选择目录</el-button>
              <el-button
                type="primary"
                :icon="Download"
                :loading="exporting"
                :disabled="!selectedPaths.length"
                @click="doExport"
              >
                导出选中文件
              </el-button>
            </div>
          </template>
          <el-empty v-else description="还没有可导出的文件">
            <span class="hint">先在「SVN 提交」标签页关联提交。</span>
          </el-empty>

          <el-card v-if="exportResult" shadow="never" class="result-card">
            <el-descriptions :column="1" border size="small">
              <el-descriptions-item label="导出目录">{{ exportResult.folder }}</el-descriptions-item>
              <el-descriptions-item label="压缩包">{{ exportResult.zipPath }}</el-descriptions-item>
              <el-descriptions-item label="文件数量">{{ exportResult.fileCount }} 个</el-descriptions-item>
            </el-descriptions>
            <div v-if="exportResult.skipped && exportResult.skipped.length" class="hint" style="margin-top: 8px">
              已跳过：{{ exportResult.skipped.join('；') }}
            </div>
            <div class="actions" style="margin-top: 12px">
              <el-button :icon="FolderOpened" @click="openPath(exportResult.folder)">打开导出目录</el-button>
              <el-button :icon="Download" @click="openPath(exportResult.zipPath)">定位 ZIP</el-button>
            </div>
          </el-card>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-dialog
      v-model="changeModalOpen"
      :title="editingChangeId ? '编辑数据库变更' : '添加数据库变更'"
      width="520px"
      :close-on-click-modal="false"
    >
      <el-form label-position="top">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="变更类型">
              <el-select v-model="change.kind" style="width: 100%">
                <el-option label="新增表" value="add_table" />
                <el-option label="新增字段" value="add_column" />
                <el-option label="修改字段" value="modify_column" />
                <el-option label="删除字段" value="drop_column" />
                <el-option label="新增索引" value="add_index" />
                <el-option label="其他" value="other" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="表名">
              <el-input v-model="change.tableName" placeholder="t_order" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="字段名">
              <el-input v-model="change.columnName" placeholder="batch_no" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="数据类型">
              <el-input v-model="change.dataType" placeholder="varchar(64)" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="说明">
              <el-input v-model="change.remark" placeholder="用途、默认值、是否可空..." />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="changeModalOpen = false">取消</el-button>
        <el-button type="primary" :icon="editingChangeId ? undefined : Plus" @click="saveChange">
          {{ editingChangeId ? '保存修改' : '添加' }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>
