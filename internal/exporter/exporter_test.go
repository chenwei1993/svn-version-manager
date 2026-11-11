package exporter

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"project-manage/internal/model"
	"project-manage/internal/store"
)

type fakeSvn struct{}

func (fakeSvn) Export(_ string, repoPath string, revision int64, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, []byte(fmt.Sprintf("%s@%d", repoPath, revision)), 0o644)
}

func (fakeSvn) ExportWorkingCopy(repoPath string, revision int64, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, []byte(fmt.Sprintf("%s@%d", repoPath, revision)), 0o644)
}

func TestExportSelectedFiles(t *testing.T) {
	root := t.TempDir()
	st, err := store.Open(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	project, err := st.CreateProject(model.Project{
		Name:     "演示项目",
		Path:     filepath.Join(root, "wc"),
		RepoRoot: "file:///repo",
		Relative: "/trunk",
	})
	if err != nil {
		t.Fatal(err)
	}
	feature, err := st.CreateFeature(model.Feature{
		ProjectID: project.ID,
		Name:      "导出测试",
		Version:   "v1.0",
		Status:    "开发中",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := st.AddDBChange(model.DBChange{
		FeatureID: feature.ID, Kind: "add_table", TableName: "t_demo", Remark: "演示表",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddDBChange(model.DBChange{
		FeatureID: feature.ID, Kind: "add_column", TableName: "t_demo", ColumnName: "id", DataType: "bigint", Remark: "主键",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AddDBChange(model.DBChange{
		FeatureID: feature.ID, Kind: "add_column", TableName: "t_order", ColumnName: "source", DataType: "tinyint", Remark: "订单来源",
	}); err != nil {
		t.Fatal(err)
	}

	if err := st.LinkCommit(feature.ID, model.SvnCommit{
		Revision: 10, Author: "tester", Date: "2026-01-01 10:00:00", Message: "init",
		Files: []model.SvnChangedFile{
			{Path: "/trunk/a.go", Action: "M", Kind: "file"},
			{Path: "/trunk/b.go", Action: "A", Kind: "file"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.LinkCommit(feature.ID, model.SvnCommit{
		Revision: 11, Author: "tester", Date: "2026-01-02 10:00:00", Message: "update a",
		Files: []model.SvnChangedFile{
			{Path: "/trunk/a.go", Action: "M", Kind: "file"},
		},
	}); err != nil {
		t.Fatal(err)
	}

	e := &Exporter{Store: st, Client: fakeSvn{}}

	// 只导出 a.go：应取它最新的 r11，b.go 不应出现
	res, err := e.ExportFeature(feature.ID, filepath.Join(root, "out1"), []string{"/trunk/a.go"})
	if err != nil {
		t.Fatal(err)
	}
	if res.FileCount != 1 {
		t.Fatalf("file count = %d, want 1", res.FileCount)
	}
	data, err := os.ReadFile(filepath.Join(res.Folder, "a.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "/trunk/a.go@11" {
		t.Fatalf("a.go content = %q, want latest revision", data)
	}
	if _, err := os.Stat(filepath.Join(res.Folder, "b.go")); !os.IsNotExist(err) {
		t.Fatalf("b.go should not be exported, stat err = %v", err)
	}
	manifest, err := os.ReadFile(filepath.Join(res.Folder, "变更清单.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(manifest)
	if !strings.Contains(text, "a.go") {
		t.Fatalf("manifest should list a.go:\n%s", text)
	}
	if strings.Contains(text, "b.go") {
		t.Fatalf("manifest should not list b.go:\n%s", text)
	}
	if !strings.Contains(text, "数据库变更.sql") {
		t.Fatalf("manifest should reference the SQL file:\n%s", text)
	}
	if res.SQLPath == "" {
		t.Fatal("sql path should not be empty")
	}
	sqlContent, err := os.ReadFile(res.SQLPath)
	if err != nil {
		t.Fatal(err)
	}
	sqlText := string(sqlContent)
	if !strings.Contains(sqlText, "CREATE TABLE IF NOT EXISTS `t_demo`") {
		t.Fatalf("sql should create t_demo:\n%s", sqlText)
	}
	if !strings.Contains(sqlText, "`id` bigint") {
		t.Fatalf("sql should contain the t_demo column:\n%s", sqlText)
	}
	if !strings.Contains(sqlText, "ALTER TABLE `t_order` ADD COLUMN `source` tinyint") {
		t.Fatalf("sql should alter t_order:\n%s", sqlText)
	}
	zr, err := zip.OpenReader(res.ZipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	if len(zr.File) != 3 {
		t.Fatalf("zip entries = %d, want 3 (a.go + manifest + sql)", len(zr.File))
	}

	// 不传路径：导出全部
	resAll, err := e.ExportFeature(feature.ID, filepath.Join(root, "out2"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if resAll.FileCount != 2 {
		t.Fatalf("all file count = %d, want 2", resAll.FileCount)
	}
}
