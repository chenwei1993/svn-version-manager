package model

// Project 一个受管理的 SVN 项目（本地工作副本）
type Project struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Path      string `json:"path"`
	RepoURL   string `json:"repoUrl"`
	RepoRoot  string `json:"repoRoot"`
	Relative  string `json:"relative"`
	CreatedAt string `json:"createdAt"`

	DBName       string `json:"dbName"`
	DBConfigured bool   `json:"dbConfigured"`
}

// Feature 一次功能开发记录
type Feature struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"projectId"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`

	AutoSync         bool  `json:"autoSync"`
	LastSyncRevision int64 `json:"lastSyncRevision"`

	// 列表页统计
	DBChangeCount int `json:"dbChangeCount"`
	RevisionCount int `json:"revisionCount"`
	FileCount     int `json:"fileCount"`
}

// SyncResult 自动同步结果
type SyncResult struct {
	Added        int    `json:"added"`
	Scanned      int    `json:"scanned"`
	LastRevision int64  `json:"lastRevision"`
	Message      string `json:"message"`
}

// DBConfig 项目数据库连接配置
type DBConfig struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	User          string `json:"user"`
	Password      string `json:"password"`
	Database      string `json:"database"`
	HasPassword   bool   `json:"hasPassword"`
	ClearPassword bool   `json:"clearPassword"`
}

// DBColumn 数据库字段信息
type DBColumn struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable string `json:"nullable"`
	Default  string `json:"default"`
	Comment  string `json:"comment"`
	Key      string `json:"key"`
	Extra    string `json:"extra"`
}

// DBTable 数据库表信息
type DBTable struct {
	Name    string     `json:"name"`
	Comment string     `json:"comment"`
	Columns []DBColumn `json:"columns"`
}

// DBImportResult 从数据库导入变更的结果
type DBImportResult struct {
	Added   int    `json:"added"`
	Skipped int    `json:"skipped"`
	Message string `json:"message"`
}

// DBChange 数据库结构变更明细
type DBChange struct {
	ID         int64  `json:"id"`
	FeatureID  int64  `json:"featureId"`
	Kind       string `json:"kind"`
	TableName  string `json:"tableName"`
	ColumnName string `json:"columnName"`
	DataType   string `json:"dataType"`
	Remark     string `json:"remark"`
	CreatedAt  string `json:"createdAt"`
}

// FeatureRevision 功能关联的 SVN 提交
type FeatureRevision struct {
	ID        int64  `json:"id"`
	FeatureID int64  `json:"featureId"`
	Revision  int64  `json:"revision"`
	Author    string `json:"author"`
	Date      string `json:"date"`
	Message   string `json:"message"`
}

// FeatureFile 功能关联的变更文件
type FeatureFile struct {
	ID        int64  `json:"id"`
	FeatureID int64  `json:"featureId"`
	Revision  int64  `json:"revision"`
	Path      string `json:"path"`
	Action    string `json:"action"`
	Kind      string `json:"kind"`
}

// SvnChangedFile SVN 提交中的单个文件
type SvnChangedFile struct {
	Path   string `json:"path"`
	Action string `json:"action"`
	Kind   string `json:"kind"`
}

// SvnCommit 一次 SVN 提交
type SvnCommit struct {
	Revision int64            `json:"revision"`
	Author   string           `json:"author"`
	Date     string           `json:"date"`
	Message  string           `json:"message"`
	Files    []SvnChangedFile `json:"files"`
}

// SvnProjectInfo svn info 的关键信息
type SvnProjectInfo struct {
	URL         string `json:"url"`
	RepoRoot    string `json:"repoRoot"`
	RelativeURL string `json:"relativeUrl"`
	Revision    int64  `json:"revision"`
	UUID        string `json:"uuid"`
}

// SvnStatus SVN 命令行工具状态
type SvnStatus struct {
	Available bool   `json:"available"`
	Path      string `json:"path"`
	Version   string `json:"version"`
	Message   string `json:"message"`
}

// FeatureDetail 功能详情聚合
type FeatureDetail struct {
	Feature   Feature           `json:"feature"`
	DBChanges []DBChange        `json:"dbChanges"`
	Revisions []FeatureRevision `json:"revisions"`
	Files     []FeatureFile     `json:"files"`
}

// AppSettings 应用设置
type AppSettings struct {
	SvnPath          string `json:"svnPath"`
	SvnAuthor        string `json:"svnAuthor"`
	DefaultExportDir string `json:"defaultExportDir"`
	LogLimit         int    `json:"logLimit"`

	DataDir    string `json:"dataDir"`
	ConfigDir  string `json:"configDir"`
	ConfigFile string `json:"configFile"`
	DBPath     string `json:"dbPath"`
}

// ExportResult 导出结果
type ExportResult struct {
	Folder    string   `json:"folder"`
	ZipPath   string   `json:"zipPath"`
	FileCount int      `json:"fileCount"`
	Skipped   []string `json:"skipped"`
	Manifest  string   `json:"manifest"`
	SQLPath   string   `json:"sqlPath"`
}
