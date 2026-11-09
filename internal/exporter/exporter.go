package exporter

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"project-manage/internal/model"
	"project-manage/internal/store"
)

// SvnClient 导出所需的 SVN 能力，便于测试替换
type SvnClient interface {
	Export(repoRoot, repoPath string, revision int64, dest string) error
	ExportWorkingCopy(wcPath string, revision int64, dest string) error
}

// Exporter 负责把功能涉及的变更文件导出为代码包
type Exporter struct {
	Store  *store.Store
	Client SvnClient
}

var kindLabels = map[string]string{
	"add_table":     "新增表",
	"add_column":    "新增字段",
	"modify_column": "修改字段",
	"drop_column":   "删除字段",
	"add_index":     "新增索引",
	"other":         "其他",
}

var actionLabels = map[string]string{
	"A": "新增",
	"M": "修改",
	"D": "删除",
	"R": "替换",
}

func KindLabel(k string) string {
	if v, ok := kindLabels[k]; ok {
		return v
	}
	return k
}

// ExportFeature 导出功能关联的文件并打包 zip；onlyPaths 为空时导出全部
func (e *Exporter) ExportFeature(featureID int64, outDir string, onlyPaths []string) (model.ExportResult, error) {
	result := model.ExportResult{Skipped: make([]string, 0)}

	detail, err := e.Store.GetFeatureDetail(featureID)
	if err != nil {
		return result, fmt.Errorf("读取功能记录失败: %w", err)
	}
	project, err := e.Store.GetProject(detail.Feature.ProjectID)
	if err != nil {
		return result, fmt.Errorf("读取项目失败: %w", err)
	}
	if outDir == "" {
		return result, fmt.Errorf("请选择导出目录")
	}
	if len(detail.Files) == 0 {
		return result, fmt.Errorf("该功能还没有关联变更文件，请先在 SVN 提交中关联提交")
	}

	selected := make(map[string]bool, len(onlyPaths))
	for _, p := range onlyPaths {
		selected[p] = true
	}

	folderName := safeName(detail.Feature.Name)
	if folderName == "" {
		folderName = fmt.Sprintf("feature-%d", detail.Feature.ID)
	}
	folderName = fmt.Sprintf("%s_%s", folderName, time.Now().Format("20060102_150405"))
	folder := filepath.Join(outDir, folderName)
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return result, fmt.Errorf("创建导出目录失败: %w", err)
	}

	// 同一个文件可能出现在多个提交里，取版本号最大的那次作为最终状态
	latest := make(map[string]model.FeatureFile)
	for _, f := range detail.Files {
		old, ok := latest[f.Path]
		if !ok || f.Revision > old.Revision {
			latest[f.Path] = f
		}
	}

	paths := make([]string, 0, len(latest))
	for path := range latest {
		if len(onlyPaths) > 0 && !selected[path] {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)

	exported := make(map[string]model.FeatureFile, len(paths))
	for _, path := range paths {
		f := latest[path]
		if strings.EqualFold(f.Kind, "dir") {
			result.Skipped = append(result.Skipped, path+" (目录)")
			continue
		}
		if strings.EqualFold(f.Action, "D") {
			result.Skipped = append(result.Skipped, path+" (该版本已删除)")
			continue
		}
		rel := projectRelativePath(project.Relative, path)
		rel = filepath.FromSlash(rel)
		dest := filepath.Join(folder, rel)
		if !within(folder, dest) {
			result.Skipped = append(result.Skipped, path+" (路径非法)")
			continue
		}
		if err := e.Client.Export(project.RepoRoot, path, f.Revision, dest); err != nil {
			// 回退：从工作副本按 peg 版本导出
			if wcErr := e.exportFromWorkingCopy(project, path, f.Revision, dest); wcErr != nil {
				return result, fmt.Errorf("导出 %s (r%d) 失败: %v", path, f.Revision, err)
			}
		}
		exported[path] = f
		result.FileCount++
	}

	if result.FileCount == 0 {
		return result, fmt.Errorf("没有可导出的文件，请检查选中的文件是否已被删除")
	}

	manifest := buildManifest(project, detail, exported)
	manifestPath := filepath.Join(folder, "变更清单.md")
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o644); err != nil {
		return result, fmt.Errorf("写入变更清单失败: %w", err)
	}
	result.Manifest = manifest
	result.Folder = folder

	zipPath := folder + ".zip"
	if err := zipFolder(folder, zipPath); err != nil {
		return result, fmt.Errorf("打包 zip 失败: %w", err)
	}
	result.ZipPath = zipPath
	return result, nil
}

func (e *Exporter) exportFromWorkingCopy(project model.Project, repoPath string, revision int64, dest string) error {
	wc := filepath.Join(project.Path, filepath.FromSlash(projectRelativePath(project.Relative, repoPath)))
	return e.Client.ExportWorkingCopy(wc, revision, dest)
}

// projectRelativePath 把仓库内路径转换成相对项目工作副本的路径
func projectRelativePath(relative, repoPath string) string {
	p := strings.TrimSpace(repoPath)
	rel := strings.TrimRight(strings.TrimSpace(relative), "/")
	if rel != "" && (p == rel || strings.HasPrefix(p, rel+"/")) {
		return strings.TrimPrefix(strings.TrimPrefix(p, rel), "/")
	}
	return strings.TrimPrefix(p, "/")
}

func safeName(s string) string {
	s = strings.TrimSpace(s)
	replacer := strings.NewReplacer(
		"\\", "_", "/", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_",
	)
	s = replacer.Replace(s)
	return strings.Trim(s, " .")
}

func within(base, target string) bool {
	absBase, err1 := filepath.Abs(base)
	absTarget, err2 := filepath.Abs(target)
	if err1 != nil || err2 != nil {
		return false
	}
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func buildManifest(project model.Project, d model.FeatureDetail, files map[string]model.FeatureFile) string {
	var b strings.Builder
	f := d.Feature
	b.WriteString("# 功能变更清单\n\n")
	fmt.Fprintf(&b, "## %s\n\n", f.Name)
	fmt.Fprintf(&b, "- 项目：%s\n", project.Name)
	if f.Version != "" {
		fmt.Fprintf(&b, "- 版本：%s\n", f.Version)
	}
	fmt.Fprintf(&b, "- 状态：%s\n", f.Status)
	fmt.Fprintf(&b, "- 创建时间：%s\n", f.CreatedAt)
	if project.RepoRoot != "" {
		fmt.Fprintf(&b, "- 仓库：%s\n", project.RepoRoot)
	}
	fmt.Fprintf(&b, "- 导出文件：%d 个\n", len(files))
	b.WriteString("\n")

	if strings.TrimSpace(f.Description) != "" {
		b.WriteString("## 功能说明\n\n")
		b.WriteString(strings.TrimSpace(f.Description))
		b.WriteString("\n\n")
	}

	b.WriteString("## 数据库变更\n\n")
	if len(d.DBChanges) == 0 {
		b.WriteString("无\n\n")
	} else {
		b.WriteString("| 类型 | 表名 | 字段名 | 数据类型 | 说明 |\n")
		b.WriteString("| --- | --- | --- | --- | --- |\n")
		for _, c := range d.DBChanges {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n",
				KindLabel(c.Kind), c.TableName, c.ColumnName, c.DataType, c.Remark)
		}
		b.WriteString("\n")
	}

	b.WriteString("## 关联提交\n\n")
	if len(d.Revisions) == 0 {
		b.WriteString("无\n\n")
	} else {
		b.WriteString("| 版本 | 作者 | 时间 | 说明 |\n")
		b.WriteString("| --- | --- | --- | --- |\n")
		for _, r := range d.Revisions {
			msg := strings.ReplaceAll(r.Message, "\n", " ")
			msg = strings.ReplaceAll(msg, "|", "\\|")
			fmt.Fprintf(&b, "| r%d | %s | %s | %s |\n", r.Revision, r.Author, r.Date, msg)
		}
		b.WriteString("\n")
	}

	b.WriteString("## 文件清单\n\n")
	if len(files) == 0 {
		b.WriteString("无\n")
	} else {
		paths := make([]string, 0, len(files))
		for p := range files {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		b.WriteString("| 文件 | 变更 | 版本 |\n")
		b.WriteString("| --- | --- | --- |\n")
		for _, p := range paths {
			file := files[p]
			action := actionLabels[file.Action]
			if action == "" {
				action = file.Action
			}
			fmt.Fprintf(&b, "| %s | %s | r%d |\n", projectRelativePath(project.Relative, file.Path), action, file.Revision)
		}
	}
	return b.String()
}

func zipFolder(src, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	base := filepath.Base(src)
	err = filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(filepath.Join(base, rel))
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
	if err != nil {
		zw.Close()
		return err
	}
	return zw.Close()
}
