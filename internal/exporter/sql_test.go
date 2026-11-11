package exporter

import (
	"strings"
	"testing"

	"project-manage/internal/model"
)

func TestBuildSQL(t *testing.T) {
	detail := model.FeatureDetail{
		Feature: model.Feature{Name: "测试功能", Version: "v1.0"},
		DBChanges: []model.DBChange{
			{Kind: "add_table", TableName: "t_new", Remark: "新表"},
			{Kind: "add_column", TableName: "t_new", ColumnName: "id", DataType: "bigint", Remark: "主键"},
			{Kind: "add_column", TableName: "t_new", ColumnName: "name", DataType: "varchar(64)", Remark: "名称"},
			{Kind: "add_column", TableName: "t_existing", ColumnName: "status", DataType: "tinyint", Remark: "状态"},
			{Kind: "modify_column", TableName: "t_existing", ColumnName: "remark", DataType: "varchar(255)", Remark: "备注"},
			{Kind: "drop_column", TableName: "t_existing", ColumnName: "old_col"},
			{Kind: "add_index", TableName: "t_existing", ColumnName: "status", Remark: "状态索引"},
			{Kind: "other", TableName: "t_existing", Remark: "手工调整"},
			{Kind: "add_table", TableName: "t_empty", Remark: "需要补字段"},
		},
	}

	sql := buildSQL(model.Project{Name: "测试项目"}, detail)

	checks := []string{
		"CREATE TABLE IF NOT EXISTS `t_new`",
		"`id` bigint",
		"COMMENT '主键'",
		"ALTER TABLE `t_existing` ADD COLUMN `status` tinyint",
		"ALTER TABLE `t_existing` MODIFY COLUMN `remark` varchar(255)",
		"ALTER TABLE `t_existing` DROP COLUMN `old_col`",
		"ALTER TABLE `t_existing` ADD INDEX (`status`)",
		"其他变更：手工调整",
		"TODO: 请补充表 t_empty 的字段定义",
	}
	for _, check := range checks {
		if !strings.Contains(sql, check) {
			t.Errorf("sql should contain %q:\n%s", check, sql)
		}
	}

	// 新增表下的字段应合并进 CREATE TABLE，不重复生成 ALTER
	if strings.Contains(sql, "ALTER TABLE `t_new` ADD COLUMN") {
		t.Errorf("new table columns should be merged into CREATE TABLE:\n%s", sql)
	}
}
