<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { Plus, Search } from '@element-plus/icons-vue'
import { api } from '../api'
import { errText, statusList, statusTagType } from '../utils'
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
        <h2>功能记录</h2>
        <p>按功能管理开发内容、数据库变更和关联的 SVN 提交。</p>
      </div>
      <div class="actions">
        <el-button type="primary" :icon="Plus" :disabled="!project" @click="openCreate">新建功能</el-button>
      </div>
    </div>

    <el-card v-if="!project" shadow="never" class="panel">
      <el-empty description="请先选择项目">
        <span class="hint">在顶部下拉框选择一个项目，或先到「项目管理」添加。</span>
      </el-empty>
    </el-card>

    <template v-else>
      <el-card shadow="never" class="panel toolbar-card">
        <div class="toolbar">
          <div class="toolbar-left">
            <el-input
              v-model="keyword"
              :prefix-icon="Search"
              placeholder="搜索功能名称 / 版本 / 说明"
              clearable
              style="width: 280px"
            />
            <el-select v-model="statusFilter" placeholder="全部状态" clearable style="width: 130px">
              <el-option label="全部状态" value="" />
              <el-option v-for="s in statusList" :key="s" :label="s" :value="s" />
            </el-select>
          </div>
          <span class="hint">共 {{ filtered.length }} 个功能</span>
        </div>
      </el-card>

      <el-card v-if="!loading && features.length === 0" shadow="never" class="panel" style="margin-top: 14px">
        <el-empty description="还没有功能记录">
          <el-button type="primary" :icon="Plus" @click="openCreate">新建功能</el-button>
        </el-empty>
      </el-card>

      <el-card v-else-if="!loading && filtered.length === 0" shadow="never" class="panel" style="margin-top: 14px">
        <el-empty description="没有匹配的功能" />
      </el-card>

      <el-card v-else shadow="never" class="panel table-panel" style="margin-top: 14px">
        <el-table
          v-loading="loading"
          :data="filtered"
          class="clickable-table"
          row-key="id"
          style="width: 100%"
          @row-click="openFeature"
        >
          <el-table-column label="功能" min-width="240">
            <template #default="{ row }">
              <div class="cell-title">{{ row.name }}</div>
              <div v-if="row.description" class="cell-sub">{{ row.description }}</div>
            </template>
          </el-table-column>
          <el-table-column label="版本 / 迭代" width="120">
            <template #default="{ row }">
              <span class="muted">{{ row.version || '—' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="statusTagType(row.status)" size="small">{{ row.status }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="数据库" width="90" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.dbChangeCount" type="info" size="small" effect="plain">{{ row.dbChangeCount }}</el-tag>
              <span v-else class="muted">—</span>
            </template>
          </el-table-column>
          <el-table-column label="提交" width="80" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.revisionCount" type="info" size="small" effect="plain">{{ row.revisionCount }}</el-tag>
              <span v-else class="muted">—</span>
            </template>
          </el-table-column>
          <el-table-column label="文件" width="80" align="center">
            <template #default="{ row }">
              <el-tag v-if="row.fileCount" type="info" size="small" effect="plain">{{ row.fileCount }}</el-tag>
              <span v-else class="muted">—</span>
            </template>
          </el-table-column>
          <el-table-column label="更新时间" width="170">
            <template #default="{ row }">
              <span class="hint">{{ row.updatedAt }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="90" align="right">
            <template #default="{ row }">
              <el-button link type="primary" @click.stop="openFeature(row)">打开</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-card>
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
