package store

import (
	"database/sql"
	"fmt"
	"time"

	"project-manage/internal/model"

	_ "modernc.org/sqlite"
)

// Store 本地数据存储
type Store struct {
	db *sql.DB
}

const timeLayout = "2006-01-02 15:04:05"

func now() string { return time.Now().Format(timeLayout) }

// Open 打开（不存在则创建）SQLite 数据库
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.init(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Checkpoint 把 WAL 内容合并回主数据库，便于复制数据文件
func (s *Store) Checkpoint() error {
	_, err := s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE);`)
	return err
}

func (s *Store) init() error {
	stmts := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			repo_url TEXT NOT NULL DEFAULT '',
			repo_root TEXT NOT NULL DEFAULT '',
			relative TEXT NOT NULL DEFAULT '',
			db_host TEXT NOT NULL DEFAULT '',
			db_port INTEGER NOT NULL DEFAULT 3306,
			db_user TEXT NOT NULL DEFAULT '',
			db_password TEXT NOT NULL DEFAULT '',
			db_name TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS features (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			version TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '开发中',
			auto_sync INTEGER NOT NULL DEFAULT 0,
			last_sync_revision INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS db_changes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feature_id INTEGER NOT NULL REFERENCES features(id) ON DELETE CASCADE,
			kind TEXT NOT NULL,
			table_name TEXT NOT NULL DEFAULT '',
			column_name TEXT NOT NULL DEFAULT '',
			data_type TEXT NOT NULL DEFAULT '',
			remark TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS feature_revisions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feature_id INTEGER NOT NULL REFERENCES features(id) ON DELETE CASCADE,
			revision INTEGER NOT NULL,
			author TEXT NOT NULL DEFAULT '',
			date TEXT NOT NULL DEFAULT '',
			message TEXT NOT NULL DEFAULT '',
			UNIQUE(feature_id, revision)
		);`,
		`CREATE TABLE IF NOT EXISTS feature_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			feature_id INTEGER NOT NULL REFERENCES features(id) ON DELETE CASCADE,
			revision INTEGER NOT NULL,
			path TEXT NOT NULL,
			action TEXT NOT NULL DEFAULT 'M',
			kind TEXT NOT NULL DEFAULT 'file',
			UNIQUE(feature_id, revision, path)
		);`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_features_project ON features(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_files_feature ON feature_files(feature_id);`,
	}
	for _, q := range stmts {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("初始化数据库失败: %w", err)
		}
	}
	// 兼容旧版本数据库
	migrations := []struct{ column, ddl string }{
		{"auto_sync", "auto_sync INTEGER NOT NULL DEFAULT 0"},
		{"last_sync_revision", "last_sync_revision INTEGER NOT NULL DEFAULT 0"},
	}
	for _, m := range migrations {
		if err := s.addColumnIfMissing("features", m.column, m.ddl); err != nil {
			return fmt.Errorf("迁移数据库失败: %w", err)
		}
	}
	projectMigrations := []struct{ column, ddl string }{
		{"db_host", "db_host TEXT NOT NULL DEFAULT ''"},
		{"db_port", "db_port INTEGER NOT NULL DEFAULT 3306"},
		{"db_user", "db_user TEXT NOT NULL DEFAULT ''"},
		{"db_password", "db_password TEXT NOT NULL DEFAULT ''"},
		{"db_name", "db_name TEXT NOT NULL DEFAULT ''"},
	}
	for _, m := range projectMigrations {
		if err := s.addColumnIfMissing("projects", m.column, m.ddl); err != nil {
			return fmt.Errorf("迁移项目数据库失败: %w", err)
		}
	}
	return nil
}

func (s *Store) addColumnIfMissing(table, column, ddl string) error {
	rows, err := s.db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec("ALTER TABLE " + table + " ADD COLUMN " + ddl)
	return err
}

// ---------- settings ----------

func (s *Store) GetSetting(key string) string {
	var v string
	_ = s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	return v
}

func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO settings(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// ---------- projects ----------

func (s *Store) ListProjects() ([]model.Project, error) {
	rows, err := s.db.Query(`SELECT id, name, path, repo_url, repo_root, relative, created_at, db_name,
		CASE WHEN db_host != '' AND db_name != '' THEN 1 ELSE 0 END
		FROM projects ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]model.Project, 0)
	for rows.Next() {
		var p model.Project
		var dbConfigured int
		if err := rows.Scan(&p.ID, &p.Name, &p.Path, &p.RepoURL, &p.RepoRoot, &p.Relative, &p.CreatedAt, &p.DBName, &dbConfigured); err != nil {
			return nil, err
		}
		p.DBConfigured = dbConfigured != 0
		list = append(list, p)
	}
	return list, rows.Err()
}

func (s *Store) GetProject(id int64) (model.Project, error) {
	var p model.Project
	var dbConfigured int
	err := s.db.QueryRow(`SELECT id, name, path, repo_url, repo_root, relative, created_at, db_name,
		CASE WHEN db_host != '' AND db_name != '' THEN 1 ELSE 0 END
		FROM projects WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Path, &p.RepoURL, &p.RepoRoot, &p.Relative, &p.CreatedAt, &p.DBName, &dbConfigured)
	p.DBConfigured = dbConfigured != 0
	return p, err
}

func (s *Store) CreateProject(p model.Project) (model.Project, error) {
	p.CreatedAt = now()
	res, err := s.db.Exec(`INSERT INTO projects(name, path, repo_url, repo_root, relative, created_at) VALUES(?,?,?,?,?,?)`,
		p.Name, p.Path, p.RepoURL, p.RepoRoot, p.Relative, p.CreatedAt)
	if err != nil {
		return p, err
	}
	p.ID, err = res.LastInsertId()
	return p, err
}

func (s *Store) UpdateProject(p model.Project) error {
	_, err := s.db.Exec(`UPDATE projects SET name=?, path=?, repo_url=?, repo_root=?, relative=? WHERE id=?`,
		p.Name, p.Path, p.RepoURL, p.RepoRoot, p.Relative, p.ID)
	return err
}

func (s *Store) DeleteProject(id int64) error {
	_, err := s.db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	return err
}

// GetProjectDBRaw 读取项目的数据库连接（密码为加密后的密文）
func (s *Store) GetProjectDBRaw(id int64) (host string, port int, user, password, database string, err error) {
	err = s.db.QueryRow(`SELECT db_host, db_port, db_user, db_password, db_name FROM projects WHERE id = ?`, id).
		Scan(&host, &port, &user, &password, &database)
	return
}

// SetProjectDB 保存项目的数据库连接
func (s *Store) SetProjectDB(id int64, host string, port int, user, password, database string) error {
	_, err := s.db.Exec(`UPDATE projects SET db_host = ?, db_port = ?, db_user = ?, db_password = ?, db_name = ? WHERE id = ?`,
		host, port, user, password, database, id)
	return err
}

// ---------- features ----------

func (s *Store) ListFeatures(projectID int64) ([]model.Feature, error) {
	rows, err := s.db.Query(`SELECT f.id, f.project_id, f.name, f.version, f.description, f.status, f.created_at, f.updated_at,
		f.auto_sync, f.last_sync_revision,
		(SELECT COUNT(*) FROM db_changes c WHERE c.feature_id = f.id),
		(SELECT COUNT(*) FROM feature_revisions r WHERE r.feature_id = f.id),
		(SELECT COUNT(*) FROM feature_files ff WHERE ff.feature_id = f.id)
		FROM features f WHERE f.project_id = ? ORDER BY f.id DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]model.Feature, 0)
	for rows.Next() {
		var f model.Feature
		var autoSync int
		if err := rows.Scan(&f.ID, &f.ProjectID, &f.Name, &f.Version, &f.Description, &f.Status, &f.CreatedAt, &f.UpdatedAt,
			&autoSync, &f.LastSyncRevision, &f.DBChangeCount, &f.RevisionCount, &f.FileCount); err != nil {
			return nil, err
		}
		f.AutoSync = autoSync != 0
		list = append(list, f)
	}
	return list, rows.Err()
}

func (s *Store) GetFeature(id int64) (model.Feature, error) {
	var f model.Feature
	var autoSync int
	err := s.db.QueryRow(`SELECT id, project_id, name, version, description, status, created_at, updated_at, auto_sync, last_sync_revision
		FROM features WHERE id = ?`, id).
		Scan(&f.ID, &f.ProjectID, &f.Name, &f.Version, &f.Description, &f.Status, &f.CreatedAt, &f.UpdatedAt, &autoSync, &f.LastSyncRevision)
	f.AutoSync = autoSync != 0
	return f, err
}

func (s *Store) CreateFeature(f model.Feature) (model.Feature, error) {
	f.CreatedAt = now()
	f.UpdatedAt = f.CreatedAt
	if f.Status == "" {
		f.Status = "开发中"
	}
	res, err := s.db.Exec(`INSERT INTO features(project_id, name, version, description, status, created_at, updated_at)
		VALUES(?,?,?,?,?,?,?)`, f.ProjectID, f.Name, f.Version, f.Description, f.Status, f.CreatedAt, f.UpdatedAt)
	if err != nil {
		return f, err
	}
	f.ID, err = res.LastInsertId()
	return f, err
}

func (s *Store) UpdateFeature(f model.Feature) error {
	f.UpdatedAt = now()
	_, err := s.db.Exec(`UPDATE features SET name=?, version=?, description=?, status=?, updated_at=? WHERE id=?`,
		f.Name, f.Version, f.Description, f.Status, f.UpdatedAt, f.ID)
	return err
}

func (s *Store) DeleteFeature(id int64) error {
	_, err := s.db.Exec(`DELETE FROM features WHERE id = ?`, id)
	return err
}

// SetFeatureAutoSync 开启/关闭功能的自动同步
func (s *Store) SetFeatureAutoSync(id int64, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	_, err := s.db.Exec(`UPDATE features SET auto_sync = ? WHERE id = ?`, v, id)
	return err
}

// SetFeatureLastSyncRevision 记录已经同步到的版本
func (s *Store) SetFeatureLastSyncRevision(id, revision int64) error {
	_, err := s.db.Exec(`UPDATE features SET last_sync_revision = ? WHERE id = ?`, revision, id)
	return err
}

// MaxFeatureRevision 返回功能已关联的最大版本号
func (s *Store) MaxFeatureRevision(id int64) (int64, error) {
	var rev int64
	err := s.db.QueryRow(`SELECT COALESCE(MAX(revision), 0) FROM feature_revisions WHERE feature_id = ?`, id).Scan(&rev)
	return rev, err
}

// ---------- db changes ----------

func (s *Store) ListDBChanges(featureID int64) ([]model.DBChange, error) {
	rows, err := s.db.Query(`SELECT id, feature_id, kind, table_name, column_name, data_type, remark, created_at
		FROM db_changes WHERE feature_id = ? ORDER BY id`, featureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]model.DBChange, 0)
	for rows.Next() {
		var c model.DBChange
		if err := rows.Scan(&c.ID, &c.FeatureID, &c.Kind, &c.TableName, &c.ColumnName, &c.DataType, &c.Remark, &c.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

func (s *Store) AddDBChange(c model.DBChange) (model.DBChange, error) {
	c.CreatedAt = now()
	res, err := s.db.Exec(`INSERT INTO db_changes(feature_id, kind, table_name, column_name, data_type, remark, created_at)
		VALUES(?,?,?,?,?,?,?)`, c.FeatureID, c.Kind, c.TableName, c.ColumnName, c.DataType, c.Remark, c.CreatedAt)
	if err != nil {
		return c, err
	}
	c.ID, err = res.LastInsertId()
	return c, err
}

func (s *Store) UpdateDBChange(c model.DBChange) error {
	_, err := s.db.Exec(`UPDATE db_changes SET kind=?, table_name=?, column_name=?, data_type=?, remark=? WHERE id=?`,
		c.Kind, c.TableName, c.ColumnName, c.DataType, c.Remark, c.ID)
	return err
}

func (s *Store) DeleteDBChange(id int64) error {
	_, err := s.db.Exec(`DELETE FROM db_changes WHERE id = ?`, id)
	return err
}

// ---------- revisions & files ----------

func (s *Store) ListFeatureRevisions(featureID int64) ([]model.FeatureRevision, error) {
	rows, err := s.db.Query(`SELECT id, feature_id, revision, author, date, message
		FROM feature_revisions WHERE feature_id = ? ORDER BY revision DESC`, featureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]model.FeatureRevision, 0)
	for rows.Next() {
		var r model.FeatureRevision
		if err := rows.Scan(&r.ID, &r.FeatureID, &r.Revision, &r.Author, &r.Date, &r.Message); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	return list, rows.Err()
}

func (s *Store) ListFeatureFiles(featureID int64) ([]model.FeatureFile, error) {
	rows, err := s.db.Query(`SELECT id, feature_id, revision, path, action, kind
		FROM feature_files WHERE feature_id = ? ORDER BY path, revision`, featureID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := make([]model.FeatureFile, 0)
	for rows.Next() {
		var f model.FeatureFile
		if err := rows.Scan(&f.ID, &f.FeatureID, &f.Revision, &f.Path, &f.Action, &f.Kind); err != nil {
			return nil, err
		}
		list = append(list, f)
	}
	return list, rows.Err()
}

// LinkCommit 把一个提交及其文件关联到功能
func (s *Store) LinkCommit(featureID int64, c model.SvnCommit) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`INSERT INTO feature_revisions(feature_id, revision, author, date, message)
		VALUES(?,?,?,?,?) ON CONFLICT(feature_id, revision) DO UPDATE SET
		author=excluded.author, date=excluded.date, message=excluded.message`,
		featureID, c.Revision, c.Author, c.Date, c.Message); err != nil {
		return err
	}
	for _, f := range c.Files {
		if _, err := tx.Exec(`INSERT INTO feature_files(feature_id, revision, path, action, kind)
			VALUES(?,?,?,?,?) ON CONFLICT(feature_id, revision, path) DO UPDATE SET
			action=excluded.action, kind=excluded.kind`,
			featureID, c.Revision, f.Path, f.Action, f.Kind); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) UnlinkRevision(featureID, revision int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM feature_revisions WHERE feature_id=? AND revision=?`, featureID, revision); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM feature_files WHERE feature_id=? AND revision=?`, featureID, revision); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetFeatureDetail(id int64) (model.FeatureDetail, error) {
	var d model.FeatureDetail
	f, err := s.GetFeature(id)
	if err != nil {
		return d, err
	}
	d.Feature = f
	if d.DBChanges, err = s.ListDBChanges(id); err != nil {
		return d, err
	}
	if d.Revisions, err = s.ListFeatureRevisions(id); err != nil {
		return d, err
	}
	if d.Files, err = s.ListFeatureFiles(id); err != nil {
		return d, err
	}
	return d, nil
}

// DeleteFeatureFiles 清空功能关联的文件（保留提交记录）
func (s *Store) DeleteFeatureFiles(featureID int64) error {
	_, err := s.db.Exec(`DELETE FROM feature_files WHERE feature_id = ?`, featureID)
	return err
}

// AddFeatureFiles 追加文件关联（按 revision+path 去重）
func (s *Store) AddFeatureFiles(featureID int64, files []model.FeatureFile) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, f := range files {
		if _, err := tx.Exec(`INSERT INTO feature_files(feature_id, revision, path, action, kind)
			VALUES(?,?,?,?,?) ON CONFLICT(feature_id, revision, path) DO UPDATE SET
			action=excluded.action, kind=excluded.kind`,
			featureID, f.Revision, f.Path, f.Action, f.Kind); err != nil {
			return err
		}
	}
	return tx.Commit()
}
