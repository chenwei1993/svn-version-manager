package main

import (
	"testing"

	"project-manage/internal/model"
)

func TestBuildDBImportChanges(t *testing.T) {
	table := model.DBTable{
		Name:    "t_user",
		Comment: "用户表",
		Columns: []model.DBColumn{
			{Name: "id", Type: "bigint", Comment: "主键"},
			{Name: "name", Type: "varchar(64)", Comment: "名称"},
		},
	}
	existing := []model.DBChange{
		{Kind: "add_table", TableName: "t_user"},
		{Kind: "add_column", TableName: "t_user", ColumnName: "id"},
	}

	changes, skipped := buildDBImportChanges(9, existing, table, "table", table.Columns)
	if len(changes) != 1 {
		t.Fatalf("changes = %d, want 1: %+v", len(changes), changes)
	}
	if skipped != 2 {
		t.Fatalf("skipped = %d, want 2", skipped)
	}
	got := changes[0]
	if got.FeatureID != 9 || got.Kind != "add_column" || got.TableName != "t_user" || got.ColumnName != "name" {
		t.Fatalf("unexpected change: %+v", got)
	}
	if got.DataType != "varchar(64)" || got.Remark != "名称" {
		t.Fatalf("unexpected column meta: %+v", got)
	}
}

func TestBuildDBImportChangesColumnsOnly(t *testing.T) {
	table := model.DBTable{
		Name: "t_order",
		Columns: []model.DBColumn{
			{Name: "source", Type: "tinyint", Comment: "来源"},
		},
	}
	changes, skipped := buildDBImportChanges(1, nil, table, "columns", table.Columns)
	if len(changes) != 1 || skipped != 0 {
		t.Fatalf("changes=%d skipped=%d", len(changes), skipped)
	}
	if changes[0].Kind != "add_column" || changes[0].ColumnName != "source" {
		t.Fatalf("unexpected change: %+v", changes[0])
	}
}

func TestBuildDBImportChangesTableMode(t *testing.T) {
	table := model.DBTable{
		Name:    "t_new",
		Comment: "新表",
		Columns: []model.DBColumn{
			{Name: "id", Type: "bigint"},
		},
	}
	changes, skipped := buildDBImportChanges(1, nil, table, "table", table.Columns)
	if len(changes) != 2 || skipped != 0 {
		t.Fatalf("changes=%d skipped=%d", len(changes), skipped)
	}
	if changes[0].Kind != "add_table" || changes[0].Remark != "新表" {
		t.Fatalf("unexpected table change: %+v", changes[0])
	}
	if changes[1].Kind != "add_column" || changes[1].ColumnName != "id" {
		t.Fatalf("unexpected column change: %+v", changes[1])
	}
}
