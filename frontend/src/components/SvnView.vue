<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { Link, Refresh, Search } from '@element-plus/icons-vue'
import { api } from '../api'
import { actionTagType, errText } from '../utils'

const props = defineProps({
  project: { type: Object, default: null },
  notify: { type: Function, required: true },
  preselectFeature: { type: Number, default: 0 },
})
const emit = defineEmits(['go-features'])

const commits = ref([])
const features = ref([])
const featuresLoaded = ref(false)
const loading = ref(false)
const error = ref('')
const keyword = ref('')
const limit = ref(200)
const targetFeature = ref(null)
const linking = ref(false)
const selectedList = ref([])
const commitTableRef = ref(null)
const settingsReady = ref(false)

const myAuthor = ref('')
const authorFilter = ref('')
let authorTouched = false

const authors = computed(() => {
  const set = new Set(commits.value.map((c) => c.author).filter(Boolean))
  if (myAuthor.value) set.add(myAuthor.value)
  return [...set].sort((a, b) => a.localeCompare(b))
})

const filtered = computed(() => {
  const k = keyword.value.trim().toLowerCase()
  return commits.value.filter((c) => {
    if (authorFilter.value && c.author !== authorFilter.value) return false
    if (!k) return true
    if (String(c.revision).includes(k)) return true
    if ((c.author || '').toLowerCase().includes(k)) return true
    if ((c.message || '').toLowerCase().includes(k)) return true
    return (c.files || []).some((f) => f.path.toLowerCase().includes(k))
  })
})

const selectedCount = computed(() => selectedList.value.length)

const limitOptions = computed(() => {
  const base = [50, 100, 200, 500, 1000]
  const current = Number(limit.value)
  return current && !base.includes(current) ? [...base, current].sort((a, b) => a - b) : base
})

function onSelectionChange(rows) {
  selectedList.value = rows
}

function clearCommitSelection() {
  commitTableRef.value?.clearSelection()
  selectedList.value = []
}

watch([keyword, authorFilter], () => {
  nextTick(clearCommitSelection)
})

async function loadMyAuthor() {
  try {
    myAuthor.value = (await api.getSvnAuthor()) || ''
  } catch {
    myAuthor.value = ''
  }
  if (!authorTouched) authorFilter.value = myAuthor.value
}

async function setAsMine() {
  if (!authorFilter.value) return
  try {
    await api.setSvnAuthor(authorFilter.value)
    myAuthor.value = authorFilter.value
    props.notify(`已把「${authorFilter.value}」设为我的账号，以后默认只看自己的提交`)
  } catch (e) {
    props.notify(errText(e), 'error')
  }
}

function onAuthorChange() {
  authorTouched = true
}

async function loadFeatures() {
  featuresLoaded.value = false
  if (!props.project) {
    features.value = []
    targetFeature.value = null
    featuresLoaded.value = true
    return
  }
  try {
    features.value = await api.listFeatures(props.project.id)
    if (!features.value.length) {
      targetFeature.value = null
      return
    }
    if (props.preselectFeature && features.value.some((f) => f.id === props.preselectFeature)) {
      targetFeature.value = props.preselectFeature
    } else if (!features.value.some((f) => f.id === targetFeature.value)) {
      targetFeature.value = features.value[0]?.id || null
    }
  } catch (e) {
    features.value = []
    targetFeature.value = null
    props.notify('读取功能列表失败：' + errText(e), 'error')
  } finally {
    featuresLoaded.value = true
  }
}

async function loadLog() {
  if (!props.project) {
    commits.value = []
    return
  }
  loading.value = true
  error.value = ''
  commitTableRef.value?.clearSelection()
  selectedList.value = []
  try {
    commits.value = await api.loadSvnLog(props.project.id, limit.value)
    if (commits.value.length === 0) props.notify('该工作副本没有读取到提交记录')
  } catch (e) {
    error.value = errText(e)
    commits.value = []
  } finally {
    loading.value = false
  }
}

watch(
  () => props.project?.id,
  () => {
    if (!settingsReady.value) return
    loadFeatures()
    loadLog()
  }
)

watch(() => props.preselectFeature, (id) => {
  if (id && features.value.some((f) => f.id === id)) targetFeature.value = id
})

onMounted(async () => {
  await loadMyAuthor()
  try {
    const s = await api.getAppSettings()
    if (s.logLimit) limit.value = s.logLimit
  } catch {
    // 使用默认值
  }
  settingsReady.value = true
  loadFeatures()
  loadLog()
})

async function onLimitChange() {
  try {
    await api.setLogLimit(limit.value)
  } catch {
    // 忽略保存失败，不影响本次读取
  }
  loadLog()
}

async function link() {
  if (!targetFeature.value) return props.notify('请先在「功能记录」里创建功能', 'error')
  if (selectedCount.value === 0) return props.notify('请先勾选要关联的提交', 'error')
  linking.value = true
  try {
    await api.linkCommits(targetFeature.value, selectedList.value)
    const name = features.value.find((f) => f.id === targetFeature.value)?.name || ''
    props.notify(`已把 ${selectedCount.value} 个提交关联到「${name}」`)
    clearCommitSelection()
  } catch (e) {
    props.notify(errText(e), 'error')
  } finally {
    linking.value = false
  }
}
</script>

<template>
  <div class="view">
    <div class="view-head">
      <div class="view-title">
        <h2>SVN 提交</h2>
        <p>读取工作副本的提交记录，勾选提交后关联到功能，变更文件会自动带出。</p>
      </div>
      <div class="actions">
        <el-select v-model="limit" style="width: 130px" @change="onLimitChange">
          <el-option v-for="n in limitOptions" :key="n" :value="n" :label="`最近 ${n} 条`" />
        </el-select>
        <el-button type="primary" :icon="Refresh" :loading="loading" :disabled="!project" @click="loadLog">
          刷新提交
        </el-button>
      </div>
    </div>

    <el-card v-if="!project" shadow="never" class="panel">
      <el-empty description="请先选择项目" />
    </el-card>

    <template v-else>
      <el-alert
        v-if="error"
        type="error"
        :closable="false"
        show-icon
        title="读取 SVN 提交失败"
        :description="error"
        style="margin-bottom: 12px"
      />
      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="提交记录来自 svn log，未提交的本地修改不会出现。可按作者过滤，勾选提交后在底部关联到功能。"
        style="margin-bottom: 12px"
      />

      <el-alert
        v-if="featuresLoaded && !features.length"
        type="warning"
        :closable="false"
        show-icon
        title="当前项目还没有功能记录，暂时无法关联提交"
        style="margin-bottom: 12px"
      >
        <div class="empty-assoc">
          <span>功能记录属于当前项目。请先为「{{ project?.name }}」新建功能，或在顶部切换到有功能记录的项目。</span>
          <el-button type="primary" size="small" @click="emit('go-features')">去功能记录</el-button>
        </div>
      </el-alert>

      <el-card shadow="never" class="panel toolbar-card">
        <div class="toolbar">
          <div class="toolbar-left">
            <el-input
              v-model="keyword"
              :prefix-icon="Search"
              placeholder="按版本号 / 说明 / 文件路径过滤"
              clearable
              style="width: 280px"
            />
            <el-select
              v-model="authorFilter"
              placeholder="全部作者"
              clearable
              filterable
              style="width: 150px"
              @change="onAuthorChange"
            >
              <el-option label="全部作者" value="" />
              <el-option v-for="a in authors" :key="a" :label="a" :value="a" />
            </el-select>
            <el-button v-if="authorFilter && authorFilter !== myAuthor" @click="setAsMine">
              设为我的账号
            </el-button>
          </div>
          <span class="hint">
            显示 {{ filtered.length }} / {{ commits.length }} 条
            <template v-if="myAuthor"> · 我的账号：{{ myAuthor }}</template>
          </span>
        </div>
      </el-card>

      <el-card v-if="loading" v-loading="true" shadow="never" class="panel" style="height: 180px; margin-top: 14px" />

      <el-card v-else-if="filtered.length" shadow="never" class="panel table-panel" style="margin-top: 14px">
        <el-table
          ref="commitTableRef"
          :data="filtered"
          row-key="revision"
          style="width: 100%"
          @selection-change="onSelectionChange"
        >
          <el-table-column type="selection" width="46" />
          <el-table-column type="expand">
            <template #default="{ row }">
              <el-table :data="row.files" size="small" style="width: 100%">
                <el-table-column label="动作" width="80">
                  <template #default="{ row: file }">
                    <el-tag :type="actionTagType(file.action)" size="small">{{ file.action }}</el-tag>
                  </template>
                </el-table-column>
                <el-table-column label="文件" min-width="320">
                  <template #default="{ row: file }">
                    <span class="mono">{{ file.path }}</span>
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
          <el-table-column prop="message" label="提交说明" min-width="240" show-overflow-tooltip />
          <el-table-column label="文件" width="80" align="center">
            <template #default="{ row }">{{ (row.files || []).length }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-card v-else shadow="never" class="panel" style="margin-top: 14px">
        <el-empty description="没有提交记录">
          <span class="hint">换个过滤条件，或点击右上角刷新试试。</span>
        </el-empty>
      </el-card>

      <div v-if="features.length" class="selectbar">
        <span class="count">已选择 <strong>{{ selectedCount }}</strong> 个提交</span>
        <el-select v-model="targetFeature" placeholder="选择目标功能" style="width: 260px">
          <el-option
            v-for="f in features"
            :key="f.id"
            :label="f.name + (f.version ? ' / ' + f.version : '')"
            :value="f.id"
          />
        </el-select>
        <el-button
          type="primary"
          :icon="Link"
          :loading="linking"
          :disabled="!selectedCount || !targetFeature"
          @click="link"
        >
          关联到功能
        </el-button>
        <span class="hint">关联后到功能详情里即可选择文件导出</span>
      </div>
    </template>
  </div>
</template>
