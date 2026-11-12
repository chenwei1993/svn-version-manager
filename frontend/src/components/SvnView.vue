<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import AppIcon from './AppIcon.vue'
import {
  IconLink as Link,
  IconRefresh as Refresh,
  IconSearch as Search,
} from './icons'
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
        <div class="title-row">
          <h2>SVN 提交</h2>
          <span v-if="project" class="count-badge">{{ filtered.length }} 条</span>
        </div>
        <p>查看仓库提交记录，按作者筛选，并关联到对应功能</p>
      </div>
      <div class="actions">
        <el-select v-model="limit" style="width: 132px" @change="onLimitChange">
          <el-option v-for="n in limitOptions" :key="n" :value="n" :label="`最近 ${n} 条`" />
        </el-select>
        <el-button type="primary" :icon="Refresh" :loading="loading" :disabled="!project" @click="loadLog">
          刷新提交
        </el-button>
      </div>
    </div>

    <div v-if="!project" class="panel empty-state">
      <div class="empty-icon"><AppIcon name="search" :size="20" /></div>
      <h4>请先选择项目</h4>
      <p>在顶部选择工作副本后，这里会读取它的 SVN 提交记录</p>
    </div>

    <template v-else>
      <el-alert
        v-if="error"
        type="error"
        :closable="false"
        show-icon
        title="读取 SVN 提交失败"
        :description="error"
      />

      <el-alert
        v-if="featuresLoaded && !features.length"
        type="warning"
        :closable="false"
        show-icon
        title="当前项目还没有功能记录，暂时无法关联提交"
      >
        <div class="empty-assoc">
          <span>功能记录属于当前项目。请先为「{{ project?.name }}」新建功能，或在顶部切换到有功能记录的项目。</span>
          <el-button type="primary" size="small" @click="emit('go-features')">去功能记录</el-button>
        </div>
      </el-alert>

      <div class="panel filter-bar">
        <el-input
          v-model="keyword"
          :prefix-icon="Search"
          placeholder="搜索版本号或提交说明"
          clearable
          style="width: 236px"
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
        <span class="spacer"></span>
        <span class="filter-info">
          显示 {{ filtered.length }} / {{ commits.length }} 条
          <template v-if="myAuthor"> · 我的账号：{{ myAuthor }}</template>
        </span>
      </div>

      <div v-if="loading" v-loading="true" class="panel" style="height: 180px"></div>

      <div v-else-if="filtered.length" class="panel table-panel">
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
      </div>

      <div v-else class="panel empty-state">
        <div class="empty-icon"><AppIcon name="search" :size="20" /></div>
        <h4>没有提交记录</h4>
        <p>换个过滤条件，或点击右上角刷新提交</p>
      </div>

      <div v-if="features.length" class="selectbar">
        <div class="select-icon"><AppIcon name="check" :size="15" /></div>
        <div class="select-info">
          <span class="count">已选 <strong>{{ selectedCount }}</strong> 条提交</span>
          <span class="select-sub">将关联到同一个功能</span>
        </div>
        <div class="v-divider"></div>
        <span class="select-label">关联到功能</span>
        <el-select v-model="targetFeature" placeholder="选择功能" style="width: 240px">
          <el-option
            v-for="f in features"
            :key="f.id"
            :label="f.name + (f.version ? ' / ' + f.version : '')"
            :value="f.id"
          />
        </el-select>
        <span class="spacer"></span>
        <span class="hint">关联后到功能详情里即可选择文件导出</span>
        <el-button
          type="primary"
          :icon="Link"
          :loading="linking"
          :disabled="!selectedCount || !targetFeature"
          @click="link"
        >
          关联提交
        </el-button>
      </div>
    </template>
  </div>
</template>
