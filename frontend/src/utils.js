export const statusList = ['开发中', '待测试', '已完成', '已发布', '已暂停']

export function statusClass(status) {
  switch (status) {
    case '已完成':
    case '已发布':
      return 'ok'
    case '待测试':
      return 'warn'
    case '已暂停':
      return 'danger'
    default:
      return 'info'
  }
}

export function statusTagType(status) {
  switch (status) {
    case '已完成':
    case '已发布':
      return 'success'
    case '待测试':
      return 'warning'
    case '已暂停':
      return 'info'
    default:
      return 'primary'
  }
}

export function actionTagType(action) {
  switch (action) {
    case 'A':
      return 'success'
    case 'D':
      return 'danger'
    case 'R':
      return 'info'
    default:
      return 'warning'
  }
}

export function actionLabel(action) {
  return { A: '新增', M: '修改', D: '删除', R: '替换' }[action] || action
}

export function actionClass(action) {
  return { A: 'add', M: 'mod', D: 'del', R: 'mod' }[action] || 'mod'
}

export function kindLabel(kind) {
  return {
    add_table: '新增表',
    add_column: '新增字段',
    modify_column: '修改字段',
    drop_column: '删除字段',
    add_index: '新增索引',
    other: '其他',
  }[kind] || kind
}

export function shortPath(path, max = 64) {
  if (!path || path.length <= max) return path
  return '…' + path.slice(path.length - max)
}

export function errText(err) {
  if (!err) return '未知错误'
  if (typeof err === 'string') return err
  return err.message || String(err)
}
