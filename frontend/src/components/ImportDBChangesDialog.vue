<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'
import { api } from '../api'
import { errText } from '../utils'

const props = defineProps({
  featureId: { type: Number, required: true },
  projectId: { type: Number, required: true },
  notify: { type: Function, required: true },
})
const emit = defineEmits(['close', 'imported'])

const loading = ref(false)
const importing = ref(false)
const error = ref('')
const tables = ref([])
const tableName = ref('')
const mode = ref('table')
const selectedColumns = ref([])
const colTableRef = ref(null)

const currentTable = computed(() => tables.value.find((t) => t.name === tableName.value) || null)
const columns = computed(() => currentTable.value?.columns || [])

onMounted(loadSchema)

async function loadSchema() {
  loading.value = true
  error.value = ''
  try {
    tables.value = await api.getDatabaseSchema(props.projectId)
    if (tables.value.length) {
      tableName.value = tables.value[0].name
    }
  } catch (e) {
    error.value = errText(e)
  } finally {
    loading.value = false
  }
  if (!error.value && tableName.value) {
    await nextTick()
    selectAllColumns()
  }
}

function selectAllColumns() {
  const table = colTableRef.value
  if (!table) return
  table.clearSelection()
  nextTick(() => {
    columns.value.forEach((row) => table.toggleRowSelection(row, true))
  })
}

function clearColumns() {
  colTableRef.value?.clearSelection()
}

function onSelectionChange(rows) {
  selectedColumns.value = rows.map((r) => r.name)
}

async function onTableChange() {
  selectedColumns.value = []
  await nextTick()
  selectAllColumns()
}

async function importChanges() {
  if (!tableName.value) return props.notify('请选择表', 'error')
  if (!selectedColumns.value.length) return props.notify('请至少选择一个字段', 'error')
  importing.value = true
  try {
    const res = await api.importDatabaseChanges(
      props.featureId,
      tableName.value,
      mode.value,
      selectedColumns.value
    )
    props.notify(res.message)
    emit('imported', res)
    emit('close')
  } catch (e) {
    props.notify(errText(e), 'error')
  } finally {
    importing.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="true"
    title="从数据库导入变更"
    width="760px"
    :close-on-click-modal="false"
    @close="emit('close')"
  >
    <div v-loading="loading" style="min-height: 180px">
      <el-alert
        v-if="error"
        type="error"
        :closable="false"
        show-icon
        :title="error"
        description="请先在「项目管理」中配置该项目的数据库连接，再回来导入。"
      />

      <el-empty v-else-if="!tables.length && !loading" description="数据库里没有可导入的表" />

      <template v-else>
        <el-form label-position="top">
          <el-row :gutter="16">
            <el-col :span="14">
              <el-form-item label="选择表">
                <el-select v-model="tableName" filterable style="width: 100%" @change="onTableChange">
                  <el-option
                    v-for="t in tables"
                    :key="t.name"
                    :label="t.name + (t.comment ? ' - ' + t.comment : '')"
                    :value="t.name"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="10">
              <el-form-item label="导入方式">
                <el-radio-group v-model="mode">
                  <el-radio value="table">新增表 + 字段</el-radio>
                  <el-radio value="columns">只新增字段</el-radio>
                </el-radio-group>
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>

        <div class="section-head" style="margin-top: 4px">
          <div>
            <h4>选择字段</h4>
            <p class="hint">
              已选 {{ selectedColumns.length }} / {{ columns.length }} 个字段
              <template v-if="currentTable?.comment"> · {{ currentTable.comment }}</template>
            </p>
          </div>
          <div class="actions">
            <el-button size="small" @click="selectAllColumns">全选</el-button>
            <el-button size="small" @click="clearColumns">清空</el-button>
          </div>
        </div>

        <el-table
          ref="colTableRef"
          :data="columns"
          size="small"
          max-height="320"
          style="width: 100%"
          @selection-change="onSelectionChange"
        >
          <el-table-column type="selection" width="46" />
          <el-table-column prop="name" label="字段" min-width="150" />
          <el-table-column prop="type" label="类型" min-width="140" />
          <el-table-column prop="comment" label="注释" min-width="180" show-overflow-tooltip />
          <el-table-column prop="key" label="键" width="70" />
        </el-table>

        <p class="hint" style="margin-top: 8px">
          「新增表 + 字段」会同时生成新增表记录和选中字段记录；「只新增字段」只生成字段记录。重复导入会自动跳过。
        </p>
      </template>
    </div>

    <template #footer>
      <el-button @click="emit('close')">取消</el-button>
      <el-button
        type="primary"
        :loading="importing"
        :disabled="!!error || !tableName || !selectedColumns.length"
        @click="importChanges"
      >
        导入 {{ selectedColumns.length }} 个字段
      </el-button>
    </template>
  </el-dialog>
</template>
