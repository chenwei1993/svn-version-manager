<script setup>
import { reactive, ref } from 'vue'
import { ElMessageBox } from 'element-plus'
import { Plus, FolderOpened, Refresh, Edit, Delete, Coin } from '@element-plus/icons-vue'
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
const form = reactive({ name: '', path: '' })

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
      `功能记录会一并删除，但不会修改磁盘上的任何文件。`,
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
        <p>添加需要跟踪的 SVN 工作副本，后续的功能记录和提交都有项目归属。</p>
      </div>
      <div class="actions">
        <el-button type="primary" :icon="Plus" @click="openCreate">添加项目</el-button>
      </div>
    </div>

    <el-card v-if="!projects.length" shadow="never" class="panel">
      <el-empty description="还没有项目">
        <el-button type="primary" :icon="Plus" @click="openCreate">添加项目</el-button>
      </el-empty>
    </el-card>

    <el-card v-else shadow="never" class="panel table-panel">
      <el-table :data="projects" style="width: 100%">
        <el-table-column label="项目" min-width="160">
          <template #default="{ row }">
            <div class="cell-title">{{ row.name }}</div>
            <div class="cell-sub">{{ row.relative || '—' }}</div>
            <el-tag v-if="row.dbConfigured" size="small" type="info" effect="plain" style="margin-top: 3px; align-self: flex-start">
              DB: {{ row.dbName }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="工作副本路径" min-width="280" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="mono">{{ row.path }}</span>
          </template>
        </el-table-column>
        <el-table-column label="SVN 仓库" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            <span class="mono">{{ row.repoRoot || '未读取' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" align="right">
          <template #default="{ row }">
            <el-tooltip content="数据库连接" placement="top">
              <el-button link :icon="Coin" @click="dbProject = row" />
            </el-tooltip>
            <el-tooltip content="打开目录" placement="top">
              <el-button link :icon="FolderOpened" @click="openFolder(row)" />
            </el-tooltip>
            <el-tooltip content="刷新 SVN 信息" placement="top">
              <el-button link :icon="Refresh" @click="refreshProject(row)" />
            </el-tooltip>
            <el-tooltip content="重命名" placement="top">
              <el-button link :icon="Edit" @click="openEdit(row)" />
            </el-tooltip>
            <el-tooltip content="删除" placement="top">
              <el-button link type="danger" :icon="Delete" @click="remove(row)" />
            </el-tooltip>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

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
