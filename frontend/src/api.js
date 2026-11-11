function call(name, ...args) {
  const app = window.go && window.go.main && window.go.main.App
  if (!app || typeof app[name] !== 'function') {
    return Promise.reject(new Error('后端方法未就绪：' + name))
  }
  return app[name](...args)
}

export const api = {
  // 环境
  getSvnStatus: () => call('GetSvnStatus'),
  setSvnPath: (path) => call('SetSvnPath', path),
  autoDetectSvn: () => call('AutoDetectSvn'),
  pickDirectory: (title) => call('PickDirectory', title || '选择目录'),
  pickSvnExe: () => call('PickSvnExe'),
  getSvnAuthor: () => call('GetSvnAuthor'),
  setSvnAuthor: (name) => call('SetSvnAuthor', name),
  openFolder: (path) => call('OpenFolder', path),

  // 应用设置
  getAppSettings: () => call('GetAppSettings'),
  saveAppSettings: (settings) => call('SaveAppSettings', settings),
  setDefaultExportDir: (dir) => call('SetDefaultExportDir', dir),
  setLogLimit: (limit) => call('SetLogLimit', limit),
  changeDataDir: (dir) => call('ChangeDataDir', dir),

  // 项目
  listProjects: () => call('ListProjects'),
  createProject: (name, path) => call('CreateProject', name, path),
  updateProject: (project) => call('UpdateProject', project),
  refreshProject: (id) => call('RefreshProject', id),
  deleteProject: (id) => call('DeleteProject', id),
  getProjectDBConfig: (projectId) => call('GetProjectDBConfig', projectId),
  saveProjectDBConfig: (projectId, config) => call('SaveProjectDBConfig', projectId, config),
  testProjectDB: (projectId, config) => call('TestProjectDB', projectId, config),

  // 数据库结构
  getDatabaseSchema: (projectId) => call('GetDatabaseSchema', projectId),
  importDatabaseChanges: (featureId, tableName, mode, columns) =>
    call('ImportDatabaseChanges', featureId, tableName, mode, columns),

  // 功能
  listFeatures: (projectId) => call('ListFeatures', projectId),
  createFeature: (projectId, name, version, description) =>
    call('CreateFeature', projectId, name, version, description),
  updateFeature: (feature) => call('UpdateFeature', feature),
  deleteFeature: (id) => call('DeleteFeature', id),
  getFeatureDetail: (id) => call('GetFeatureDetail', id),
  setFeatureAutoSync: (id, enabled) => call('SetFeatureAutoSync', id, enabled),
  syncFeature: (id) => call('SyncFeature', id),

  // 数据库变更
  addDBChange: (change) => call('AddDBChange', change),
  updateDBChange: (change) => call('UpdateDBChange', change),
  deleteDBChange: (id) => call('DeleteDBChange', id),

  // SVN
  loadSvnLog: (projectId, limit) => call('LoadSvnLog', projectId, limit),
  linkCommits: (featureId, commits) => call('LinkCommits', featureId, commits),
  unlinkRevision: (featureId, revision) => call('UnlinkRevision', featureId, revision),

  // 导出
  exportFeature: (featureId, outDir, paths) =>
    call('ExportFeature', featureId, outDir, paths || []),
}
