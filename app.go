package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"project-manage/internal/exporter"
	"project-manage/internal/model"
	"project-manage/internal/store"
	"project-manage/internal/svnx"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 应用主体，所有方法都会被绑定到前端 window.go.main.App
type App struct {
	ctx context.Context

	mu        sync.Mutex
	store     *store.Store
	svnExe    string
	initErr   error
	configDir string
	dataDir   string
	dbPath    string
}

type appConfig struct {
	DataDir string `json:"dataDir"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	base, err := os.UserConfigDir()
	if err != nil {
		a.initErr = fmt.Errorf("获取配置目录失败: %w", err)
		return
	}
	a.configDir = filepath.Join(base, "project-manage")
	if err := os.MkdirAll(a.configDir, 0o755); err != nil {
		a.initErr = fmt.Errorf("创建配置目录失败: %w", err)
		return
	}

	a.dataDir = a.configDir
	cfgPath := filepath.Join(a.configDir, "config.json")
	if data, err := os.ReadFile(cfgPath); err == nil {
		var cfg appConfig
		if json.Unmarshal(data, &cfg) == nil {
			if dir := strings.TrimSpace(cfg.DataDir); dir != "" {
				if err := os.MkdirAll(dir, 0o755); err == nil {
					a.dataDir = dir
				}
			}
		}
	}

	a.dbPath = filepath.Join(a.dataDir, "project-manage.db")
	_ = a.writeConfig()
	st, err := store.Open(a.dbPath)
	if err != nil {
		a.initErr = err
		return
	}
	a.store = st

	a.svnExe = st.GetSetting("svn_exe")
	if a.svnExe == "" || !fileExists(a.svnExe) {
		if p, ok := svnx.Detect(); ok {
			a.svnExe = p
			_ = st.SetSetting("svn_exe", p)
		}
	}
}

func (a *App) shutdown(ctx context.Context) {
	if a.store != nil {
		_ = a.store.Close()
	}
}

func (a *App) db() (*store.Store, error) {
	if a.initErr != nil {
		return nil, a.initErr
	}
	if a.store == nil {
		return nil, fmt.Errorf("数据库尚未初始化")
	}
	return a.store, nil
}

func (a *App) svn() *svnx.Client {
	a.mu.Lock()
	defer a.mu.Unlock()
	return &svnx.Client{Exe: a.svnExe}
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

// ---------- SVN 工具 ----------

// GetSvnStatus 获取当前 svn 命令行状态
func (a *App) GetSvnStatus() model.SvnStatus {
	c := a.svn()
	st := model.SvnStatus{Path: c.Exe}
	if c.Exe == "" {
		st.Message = "未找到 svn 命令行工具。安装 TortoiseSVN 时请勾选 command line client tools，或在下方手动指定 svn.exe。"
		return st
	}
	if !fileExists(c.Exe) {
		st.Message = "配置的 svn.exe 不存在: " + c.Exe
		return st
	}
	v, err := c.Version()
	if err != nil {
		st.Message = err.Error()
		return st
	}
	st.Available = true
	st.Version = v
	st.Message = "SVN 命令行可用"
	return st
}

// SetSvnPath 手动指定 svn.exe 路径
func (a *App) SetSvnPath(path string) (model.SvnStatus, error) {
	path = strings.TrimSpace(strings.Trim(path, `"`))
	if path != "" {
		if !fileExists(path) {
			return model.SvnStatus{}, fmt.Errorf("文件不存在: %s", path)
		}
		client := &svnx.Client{Exe: path}
		if _, err := client.Version(); err != nil {
			return model.SvnStatus{}, fmt.Errorf("该路径不是可用的 svn.exe: %w", err)
		}
	}
	a.mu.Lock()
	a.svnExe = path
	a.mu.Unlock()
	st, err := a.db()
	if err == nil {
		_ = st.SetSetting("svn_exe", path)
	}
	return a.GetSvnStatus(), nil
}

// AutoDetectSvn 自动探测 svn.exe
func (a *App) AutoDetectSvn() (model.SvnStatus, error) {
	p, ok := svnx.Detect()
	if !ok {
		return a.GetSvnStatus(), fmt.Errorf("未自动找到 svn.exe，请手动指定")
	}
	return a.SetSvnPath(p)
}

// GetSvnAuthor 获取「我的 SVN 账号」，用于默认只看自己的提交
func (a *App) GetSvnAuthor() (string, error) {
	st, err := a.db()
	if err != nil {
		return "", err
	}
	return st.GetSetting("svn_author"), nil
}

// SetSvnAuthor 保存「我的 SVN 账号」
func (a *App) SetSvnAuthor(name string) error {
	st, err := a.db()
	if err != nil {
		return err
	}
	return st.SetSetting("svn_author", strings.TrimSpace(name))
}

// ---------- 应用设置 ----------

const defaultLogLimit = 200

// GetAppSettings 获取全部设置和路径信息
func (a *App) GetAppSettings() (model.AppSettings, error) {
	st, err := a.db()
	if err != nil {
		return model.AppSettings{}, err
	}
	a.mu.Lock()
	svnPath := a.svnExe
	a.mu.Unlock()

	limit := defaultLogLimit
	if v := st.GetSetting("log_limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	return model.AppSettings{
		SvnPath:          svnPath,
		SvnAuthor:        st.GetSetting("svn_author"),
		DefaultExportDir: st.GetSetting("default_export_dir"),
		LogLimit:         limit,
		DataDir:          a.dataDir,
		ConfigDir:        a.configDir,
		ConfigFile:       filepath.Join(a.configDir, "config.json"),
		DBPath:           a.dbPath,
	}, nil
}

// SaveAppSettings 保存常规设置（数据目录请使用 ChangeDataDir）
func (a *App) SaveAppSettings(s model.AppSettings) (model.SvnStatus, error) {
	st, err := a.db()
	if err != nil {
		return model.SvnStatus{}, err
	}
	if _, err := a.SetSvnPath(s.SvnPath); err != nil {
		return model.SvnStatus{}, err
	}
	if err := st.SetSetting("svn_author", strings.TrimSpace(s.SvnAuthor)); err != nil {
		return model.SvnStatus{}, err
	}
	if err := st.SetSetting("default_export_dir", strings.TrimSpace(s.DefaultExportDir)); err != nil {
		return model.SvnStatus{}, err
	}
	if err := st.SetSetting("log_limit", strconv.Itoa(clampLogLimit(s.LogLimit))); err != nil {
		return model.SvnStatus{}, err
	}
	return a.GetSvnStatus(), nil
}

func clampLogLimit(limit int) int {
	if limit < 50 {
		return 50
	}
	if limit > 1000 {
		return 1000
	}
	return limit
}

// SetDefaultExportDir 保存默认导出目录
func (a *App) SetDefaultExportDir(dir string) error {
	st, err := a.db()
	if err != nil {
		return err
	}
	return st.SetSetting("default_export_dir", strings.TrimSpace(dir))
}

// SetLogLimit 保存默认读取提交数
func (a *App) SetLogLimit(limit int) error {
	st, err := a.db()
	if err != nil {
		return err
	}
	return st.SetSetting("log_limit", strconv.Itoa(clampLogLimit(limit)))
}

// ChangeDataDir 把数据库复制到新目录并切换过去
func (a *App) ChangeDataDir(newDir string) (model.AppSettings, error) {
	newDir = strings.TrimSpace(strings.Trim(newDir, `"`))
	if newDir == "" {
		return model.AppSettings{}, fmt.Errorf("请选择数据目录")
	}
	abs, err := filepath.Abs(newDir)
	if err != nil {
		return model.AppSettings{}, fmt.Errorf("解析目录失败: %w", err)
	}
	if !dirExists(abs) {
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return model.AppSettings{}, fmt.Errorf("创建目录失败: %w", err)
		}
	}

	oldDir, oldDB := a.dataDir, a.dbPath
	if filepath.Clean(abs) == filepath.Clean(oldDir) {
		return a.GetAppSettings()
	}
	destDB := filepath.Join(abs, "project-manage.db")
	if fileExists(destDB) {
		return model.AppSettings{}, fmt.Errorf("目标目录已经存在 project-manage.db，请选择空目录")
	}

	st, err := a.db()
	if err != nil {
		return model.AppSettings{}, err
	}
	if err := st.Checkpoint(); err != nil {
		return model.AppSettings{}, fmt.Errorf("合并数据库失败: %w", err)
	}
	if err := st.Close(); err != nil {
		return model.AppSettings{}, fmt.Errorf("关闭数据库失败: %w", err)
	}
	a.store = nil

	if err := copyFileIfExists(oldDB, destDB); err != nil {
		a.restoreStore(oldDir, oldDB)
		return model.AppSettings{}, fmt.Errorf("复制数据库失败: %w", err)
	}

	a.dataDir = abs
	a.dbPath = destDB
	if err := a.writeConfig(); err != nil {
		a.dataDir, a.dbPath = oldDir, oldDB
		a.restoreStore(oldDir, oldDB)
		return model.AppSettings{}, fmt.Errorf("写入配置失败: %w", err)
	}

	newStore, err := store.Open(destDB)
	if err != nil {
		a.dataDir, a.dbPath = oldDir, oldDB
		_ = a.writeConfig()
		a.restoreStore(oldDir, oldDB)
		return model.AppSettings{}, fmt.Errorf("打开新数据库失败: %w", err)
	}
	a.store = newStore
	return a.GetAppSettings()
}

func (a *App) restoreStore(dir, dbPath string) {
	a.dataDir = dir
	a.dbPath = dbPath
	if st, err := store.Open(dbPath); err == nil {
		a.store = st
	}
}

func (a *App) writeConfig() error {
	data, err := json.MarshalIndent(appConfig{DataDir: a.dataDir}, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(a.configDir, "config.json")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	_ = os.Remove(path)
	return os.Rename(tmp, path)
}

func copyFileIfExists(src, dst string) error {
	in, err := os.Open(src)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// PickDirectory 打开系统目录选择框
func (a *App) PickDirectory(title string) (string, error) {
	if title == "" {
		title = "选择目录"
	}
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{Title: title})
}

// PickSvnExe 选择 svn.exe 文件
func (a *App) PickSvnExe() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "选择 svn.exe",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "svn 命令行工具 (svn.exe)", Pattern: "svn.exe"},
			{DisplayName: "可执行文件 (*.exe)", Pattern: "*.exe"},
		},
	})
}

// OpenFolder 在文件管理器中打开目录或定位文件
func (a *App) OpenFolder(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("路径为空")
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("路径不存在: %s", path)
	}
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("explorer.exe", path)
		return cmd.Start()
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

// ---------- 项目 ----------

func (a *App) ListProjects() ([]model.Project, error) {
	st, err := a.db()
	if err != nil {
		return nil, err
	}
	return st.ListProjects()
}

// CreateProject 新增项目，自动读取工作副本的 SVN 信息
func (a *App) CreateProject(name, path string) (model.Project, error) {
	var p model.Project
	st, err := a.db()
	if err != nil {
		return p, err
	}
	path = strings.TrimSpace(strings.Trim(path, `"`))
	if path == "" {
		return p, fmt.Errorf("请选择项目目录")
	}
	info, err := a.svn().Info(path)
	if err != nil {
		return p, fmt.Errorf("读取 SVN 信息失败，请确认该目录是 SVN 工作副本: %w", err)
	}
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(path)
	}
	p = model.Project{
		Name:     strings.TrimSpace(name),
		Path:     path,
		RepoURL:  info.URL,
		RepoRoot: info.RepoRoot,
		Relative: info.RelativeURL,
	}
	return st.CreateProject(p)
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// UpdateProject 更新项目名称
func (a *App) UpdateProject(p model.Project) error {
	st, err := a.db()
	if err != nil {
		return err
	}
	return st.UpdateProject(p)
}

// RefreshProject 重新读取工作副本信息
func (a *App) RefreshProject(id int64) (model.Project, error) {
	st, err := a.db()
	if err != nil {
		return model.Project{}, err
	}
	p, err := st.GetProject(id)
	if err != nil {
		return p, err
	}
	info, err := a.svn().Info(p.Path)
	if err != nil {
		return p, err
	}
	p.RepoURL = info.URL
	p.RepoRoot = info.RepoRoot
	p.Relative = info.RelativeURL
	return p, st.UpdateProject(p)
}

func (a *App) DeleteProject(id int64) error {
	st, err := a.db()
	if err != nil {
		return err
	}
	return st.DeleteProject(id)
}

// ---------- 功能记录 ----------

func (a *App) ListFeatures(projectID int64) ([]model.Feature, error) {
	st, err := a.db()
	if err != nil {
		return nil, err
	}
	return st.ListFeatures(projectID)
}

func (a *App) CreateFeature(projectID int64, name, version, description string) (model.Feature, error) {
	var f model.Feature
	st, err := a.db()
	if err != nil {
		return f, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return f, fmt.Errorf("请填写功能名称")
	}
	return st.CreateFeature(model.Feature{
		ProjectID:   projectID,
		Name:        name,
		Version:     strings.TrimSpace(version),
		Description: strings.TrimSpace(description),
		Status:      "开发中",
	})
}

func (a *App) UpdateFeature(f model.Feature) error {
	st, err := a.db()
	if err != nil {
		return err
	}
	return st.UpdateFeature(f)
}

func (a *App) DeleteFeature(id int64) error {
	st, err := a.db()
	if err != nil {
		return err
	}
	return st.DeleteFeature(id)
}

func (a *App) GetFeatureDetail(id int64) (model.FeatureDetail, error) {
	st, err := a.db()
	if err != nil {
		return model.FeatureDetail{}, err
	}
	return st.GetFeatureDetail(id)
}

// ---------- 数据库变更 ----------

func (a *App) AddDBChange(c model.DBChange) (model.DBChange, error) {
	st, err := a.db()
	if err != nil {
		return c, err
	}
	if strings.TrimSpace(c.TableName) == "" {
		return c, fmt.Errorf("请填写表名")
	}
	return st.AddDBChange(c)
}

func (a *App) UpdateDBChange(c model.DBChange) error {
	st, err := a.db()
	if err != nil {
		return err
	}
	return st.UpdateDBChange(c)
}

func (a *App) DeleteDBChange(id int64) error {
	st, err := a.db()
	if err != nil {
		return err
	}
	return st.DeleteDBChange(id)
}

// ---------- SVN 提交 ----------

// LoadSvnLog 读取项目最近的提交记录
func (a *App) LoadSvnLog(projectID int64, limit int) ([]model.SvnCommit, error) {
	st, err := a.db()
	if err != nil {
		return nil, err
	}
	p, err := st.GetProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("读取项目失败: %w", err)
	}
	if !dirExists(p.Path) {
		return nil, fmt.Errorf("项目目录不存在: %s", p.Path)
	}
	return a.svn().Log(p.Path, limit)
}

// LinkCommits 把选中的提交关联到功能
func (a *App) LinkCommits(featureID int64, commits []model.SvnCommit) error {
	st, err := a.db()
	if err != nil {
		return err
	}
	if len(commits) == 0 {
		return fmt.Errorf("请先选择提交")
	}
	for _, c := range commits {
		if err := st.LinkCommit(featureID, c); err != nil {
			return fmt.Errorf("关联 r%d 失败: %w", c.Revision, err)
		}
	}
	return nil
}

// UnlinkRevision 取消功能与某个提交的关联
func (a *App) UnlinkRevision(featureID, revision int64) error {
	st, err := a.db()
	if err != nil {
		return err
	}
	return st.UnlinkRevision(featureID, revision)
}

// ---------- 导出 ----------

// ExportFeature 一键导出功能涉及的变更文件；paths 为空时导出全部
func (a *App) ExportFeature(featureID int64, outDir string, paths []string) (model.ExportResult, error) {
	st, err := a.db()
	if err != nil {
		return model.ExportResult{}, err
	}
	e := &exporter.Exporter{Store: st, Client: a.svn()}
	return e.ExportFeature(featureID, outDir, paths)
}
