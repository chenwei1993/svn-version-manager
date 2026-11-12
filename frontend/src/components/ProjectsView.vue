<script setup>
import { computed, reactive, ref } from 'vue'
import { ElMessageBox } from 'element-plus'
import AppIcon from './AppIcon.vue'
import {
  IconDatabase as Coin,
  IconEdit as Edit,
  IconFolder as FolderOpened,
  IconPlus as Plus,
  IconRefresh as Refresh,
  IconSearch as Search,
  IconTrash as Delete,
} from './icons'
import { api } from '../api'
import { errText } from '../utils'
import DatabaseConfigDialog from './DatabaseConfigDialog.vue'

const props = defineProps({
  projects: { type: Array, default: () => [] },
  notify: { type: Function, required: true },
})
const emit = defineEmits(['refresh', 'select', 'deleted'])

const modalOpen = ref(false)
const editing = ref(null)
const saving = ref(false)
const dbProject = ref(null)
const filter = ref('all')
const keyword = ref('')
const form = reactive({ name: '', path: '' })

const dbCount = computed(() => props.projects.filter((p) => p.dbConfigured).length)

const filteredProjects = computed(() => {
  let list = props.projects
  if (filter.value === 'db') list = list.filter((p) => p.dbConfigured)
  if (filter.value === 'none') list = list.filter((p) => !p.dbConfigured)
  const k = keyword.value.trim().toLowerCase()
  if (k) {
    list = list.filter((p) =>
      [p.name, p.path, p.repoRoot, p.dbName].some((v) => (v || '').toLowerCase().includes(k))
    )
  }
  return list
})

function initials(name) {
  return (name || '?').trim().slice(0, 1)
}

function dateOnly(value) {
  return (value || '').slice(0, 10) || '—'
}

function openCreate() {
  editing.value = null
  form.name = ''
  form.path = ''
  modalOpen.value = true
}

function openEdit(p) {
  editing.value = p
  form.name = p.name
  form.path = p.path
  modalOpen.value = true
}

async function pickPath() {
  try {
    const dir = await api.pickDirectory('选择 SVN 工作副本目录')
    if (!dir) return
    form.path = dir
    if (!form.name.trim()) {
      const parts = dir.split(/[\\/]/).filter(Boolean)
      form.name = parts[parts.length - 1] || ''
    }
  } catch (e) {
    props.notify('选择目录失败：' + errText(e), 'error')
  }
}

async function save() {
  if (!form.name.trim()) return props.notify('请填写项目名称', 'error')
  if (!form.path.trim()) return props.notify('请选择项目目录', 'error')
  saving.value = true
  try {
    if (editing.value) {
      await api.updateProject({ ...editing.value, name: form.name.trim() })
      props.notify('项目已更新')
      emit('refresh')
    } else {
      const created = await api.createProject(form.name.trim(), form.path.trim())
      props.notify('项目已添加，仓库：' + (created.relative || created.repoRoot || '-'))
      emit('select', created.id)
    }
    modalOpen.value = false
  } catch (e) {
    props.notify(errText(e), 'error')
  } finally {
    saving.value = false
  }
}

async function refreshProject(p) {
  try {
    await api.refreshProject(p.id)
    props.notify('已重新读取 SVN 信息')
    emit('refresh')
  } catch (e) {
    props.notify(errText(e), 'error')
  }
}

async function remove(p) {
  try {
    await ElMessageBox.confirm(
      '功能记录会一并删除，但不会修改磁盘上的任何文件。',
      `删除项目「${p.name}」？`,
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  try {
    await api.deleteProject(p.id)
    props.notify('项目已删除')
    emit('deleted', p.id)
  } catch (e) {
    props.notify(errText(e), 'error')
  }
}

async function openFolder(p) {
  try {
    await api.openFolder(p.path)
  } catch (e) {
    props.notify(errText(e), 'error')
  }
}

function onDBSaved() {
  emit('refresh')
}
</script>

<template>
  <div class="view">
    <div class="view-head">
      <div class="view-title">
        <h2>项目管理</h2>
        <p>管理 SVN 工作副本，并为每个项目配置数据库连接</p>
      </div>
      <div class="actions">
        <el-button :icon="Refresh" @click="emit('refresh')">刷新列表</el-button>
        <el-button type="primary" :icon="Plus" @click="openCreate">添加项目</el-button>
      </div>
    </div>

    <div class="panel stats-strip">
      <div class="stat">
        <div class="value">{{ projects.length }}</div>
        <div class="label">个项目</div>
      </div>
      <div class="stat">
        <div class="value">{{ dbCount }}<span class="sub"> / {{ projects.length }}</span></div>
        <div class="label">已连接数据库</div>
      </div>
      <div class="stat">
        <div class="value">{{ projects.length - dbCount }}</div>
        <div class="label">待配置数据库</div>
      </div>
    </div>

    <div v-if="!projects.length" class="panel empty-state">
      <div class="empty-icon"><AppIcon name="folder" :size="20" /></div>
      <h4>还没有项目</h4>
      <p>添加一个包含 .svn 目录的工作副本，开始记录功能与提交</p>
      <el-button type="primary" :icon="Plus" @click="openCreate">添加项目</el-button>
    </div>

    <template v-else>
      <div class="panel toolbar-panel">
        <div class="chips">
          <button class="chip" :class="{ active: filter === 'all' }" @click="filter = 'all'">
            全部 {{ projects.length }}
          </button>
          <button class="chip" :class="{ active: filter === 'db' }" @click="filter = 'db'">
            已连接数据库 {{ dbCount }}
          </button>
          <button class="chip" :class="{ active: filter === 'none' }" @click="filter = 'none'">
            未配置 {{ projects.length - dbCount }}
          </button>
        </div>
        <div class="toolbar-right">
          <el-input
            v-model="keyword"
            :prefix-icon="Search"
            placeholder="搜索项目 / 仓库"
            clearable
            style="width: 210px"
          />
        </div>
      </div>

      <div class="panel table-panel">
        <el-table :data="filteredProjects" style="width: 100%">
          <el-table-column label="项目" min-width="250">
            <template #default="{ row, $index }">
              <div class="project-cell">
                <div class="project-avatar" :class="`tone-${$index % 4}`">{{ initials(row.name) }}</div>
                <div class="text">
                  <div class="name">{{ row.name }}</div>
                  <div class="path">{{ row.path }}</div>
                </div>
              </div>
            </template>
          </el-table-column>
          <el-table-column label="SVN 仓库" min-width="230" show-overflow-tooltip>
            <template #default="{ row }">
              <span class="mono" style="color: var(--pm-text-2)">{{ row.repoRoot || '未读取' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="数据库" width="170">
            <template #default="{ row }">
              <span v-if="row.dbConfigured" class="db-tag">已连接 · {{ row.dbName }}</span>
              <span v-else class="db-tag empty">未配置数据库</span>
            </template>
          </el-table-column>
          <el-table-column label="创建时间" width="120">
            <template #default="{ row }">
              <span class="hint">{{ dateOnly(row.createdAt) }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="180" align="right">
            <template #default="{ row }">
              <div class="row-actions">
                <button class="icon-button sm" title="数据库连接" @click="dbProject = row">
                  <el-icon><Coin /></el-icon>
                </button>
                <button class="icon-button sm" title="打开目录" @click="openFolder(row)">
                  <el-icon><FolderOpened /></el-icon>
                </button>
                <button class="icon-button sm" title="刷新 SVN 信息" @click="refreshProject(row)">
                  <el-icon><Refresh /></el-icon>
                </button>
                <button class="icon-button sm" title="重命名" @click="openEdit(row)">
                  <el-icon><Edit /></el-icon>
                </button>
                <button class="icon-button sm danger" title="删除" @click="remove(row)">
                  <el-icon><Delete /></el-icon>
                </button>
              </div>
            </template>
          </el-table-column>
          <template #empty>
            <div class="empty-state" style="padding: 36px 20px">
              <div class="empty-icon"><AppIcon name="search" :size="20" /></div>
              <h4>没有匹配的项目</h4>
              <p>换个关键词或筛选条件试试</p>
            </div>
          </template>
        </el-table>
      </div>
    </template>

    <el-dialog
      v-model="modalOpen"
      :title="editing ? '编辑项目' : '添加项目'"
      width="520px"
      :close-on-click-modal="false"
    >
      <el-form label-position="top">
        <el-form-item label="项目名称">
          <el-input v-model="form.name" placeholder="例如：订单中心" />
        </el-form-item>
        <el-form-item label="本地工作副本目录">
          <el-input v-model="form.path" placeholder="包含 .svn 目录的项目根目录">
            <template #append>
              <el-button @click="pickPath">选择目录</el-button>
            </template>
          </el-input>
        </el-form-item>
        <el-alert
          type="info"
          :closable="false"
          show-icon
          title="添加时会自动读取该目录的 SVN 仓库地址。目录必须是 SVN 工作副本。"
        />
      </el-form>
      <template #footer>
        <el-button @click="modalOpen = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">
          {{ editing ? '保存' : '添加' }}
        </el-button>
      </template>
    </el-dialog>

    <DatabaseConfigDialog
      v-if="dbProject"
      :project="dbProject"
      :notify="notify"
      @close="dbProject = null"
      @saved="onDBSaved"
    />
  </div>
</template>
