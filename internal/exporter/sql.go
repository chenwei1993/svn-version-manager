package exporter

import (
	"fmt"
	"strings"
	"time"

	"project-manage/internal/model"
)

// buildSQL 根据数据库变更记录生成一份可执行的 SQL 草稿
func buildSQL(project model.Project, d model.FeatureDetail) string {
	var b strings.Builder
	b.WriteString("-- ============================================================\n")
	fmt.Fprintf(&b, "-- 功能：%s\n", d.Feature.Name)
	fmt.Fprintf(&b, "-- 项目：%s\n", project.Name)
	if d.Feature.Version != "" {
		fmt.Fprintf(&b, "-- 版本：%s\n", d.Feature.Version)
	}
	fmt.Fprintf(&b, "-- 生成时间：%s\n", time.Now().Format("2006-01-02 15:04:05"))
	b.WriteString("-- 说明：以下 SQL 根据「数据库变更」记录生成，执行前请核对表名、字段类型、默认值和索引名。\n")
	b.WriteString("-- ============================================================\n\n")

	if len(d.DBChanges) == 0 {
		b.WriteString("-- 没有数据库变更记录\n")
		return b.String()
	}

	newTables := make(map[string]bool)
	for _, c := range d.DBChanges {
		if c.Kind == "add_table" && strings.TrimSpace(c.TableName) != "" {
			newTables[c.TableName] = true
		}
	}

	// 新增表下的新增字段合并进 CREATE TABLE，不再单独生成 ALTER
	newTableColumns := make(map[string][]model.DBChange)
	handled := make(map[int]bool)
	for i, c := range d.DBChanges {
		if c.Kind == "add_column" && newTables[c.TableName] {
			newTableColumns[c.TableName] = append(newTableColumns[c.TableName], c)
			handled[i] = true
		}
	}

	created := make(map[string]bool)
	for _, c := range d.DBChanges {
		if c.Kind != "add_table" {
			continue
		}
		table := strings.TrimSpace(c.TableName)
		if table == "" {
			b.WriteString("-- TODO: 新增表记录缺少表名\n\n")
			continue
		}
		if created[table] {
			continue
		}
		created[table] = true
		writeCreateTable(&b, c, newTableColumns[table])
	}

	for i, c := range d.DBChanges {
		if c.Kind == "add_table" || handled[i] {
			continue
		}
		writeAlter(&b, c)
	}

	return b.String()
}

func writeCreateTable(b *strings.Builder, c model.DBChange, columns []model.DBChange) {
	table := strings.TrimSpace(c.TableName)
	writeChangeComment(b, c)

	valid := make([]model.DBChange, 0, len(columns))
	for _, col := range columns {
		name := strings.TrimSpace(col.ColumnName)
		typ := strings.TrimSpace(col.DataType)
		if name == "" || typ == "" {
			fmt.Fprintf(b, "-- TODO: 表 %s 的字段 %s 缺少字段名或数据类型\n", table, name)
			continue
		}
		valid = append(valid, col)
	}

	if len(valid) == 0 {
		fmt.Fprintf(b, "-- TODO: 请补充表 %s 的字段定义\n", table)
		fmt.Fprintf(b, "-- CREATE TABLE IF NOT EXISTS %s (\n", quoteIdent(table))
		b.WriteString("--   ...\n")
		b.WriteString("-- ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\n\n")
		return
	}

	fmt.Fprintf(b, "CREATE TABLE IF NOT EXISTS %s (\n", quoteIdent(table))
	for i, col := range valid {
		line := fmt.Sprintf("  %s %s NULL", quoteIdent(strings.TrimSpace(col.ColumnName)), strings.TrimSpace(col.DataType))
		if remark := strings.TrimSpace(col.Remark); remark != "" {
			line += fmt.Sprintf(" COMMENT '%s'", escapeSQLString(remark))
		}
		if i < len(valid)-1 {
			line += ","
		}
		b.WriteString(line + "\n")
	}
	b.WriteString(") ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;\n\n")
}

func writeAlter(b *strings.Builder, c model.DBChange) {
	table := strings.TrimSpace(c.TableName)
	column := strings.TrimSpace(c.ColumnName)
	remark := strings.TrimSpace(c.Remark)

	writeChangeComment(b, c)

	if table == "" {
		b.WriteString("-- TODO: 缺少表名\n\n")
		return
	}

	switch c.Kind {
	case "add_column":
		if column == "" || strings.TrimSpace(c.DataType) == "" {
			b.WriteString("-- TODO: 缺少字段名或数据类型\n\n")
			return
		}
		fmt.Fprintf(b, "ALTER TABLE %s ADD COLUMN %s %s NULL", quoteIdent(table), quoteIdent(column), strings.TrimSpace(c.DataType))
		if remark != "" {
			fmt.Fprintf(b, " COMMENT '%s'", escapeSQLString(remark))
		}
		b.WriteString(";\n\n")

	case "modify_column":
		if column == "" || strings.TrimSpace(c.DataType) == "" {
			b.WriteString("-- TODO: 缺少字段名或数据类型\n\n")
			return
		}
		fmt.Fprintf(b, "ALTER TABLE %s MODIFY COLUMN %s %s NULL", quoteIdent(table), quoteIdent(column), strings.TrimSpace(c.DataType))
		if remark != "" {
			fmt.Fprintf(b, " COMMENT '%s'", escapeSQLString(remark))
		}
		b.WriteString(";\n\n")

	case "drop_column":
		if column == "" {
			b.WriteString("-- TODO: 缺少字段名\n\n")
			return
		}
		fmt.Fprintf(b, "ALTER TABLE %s DROP COLUMN %s;\n\n", quoteIdent(table), quoteIdent(column))

	case "add_index":
		if column == "" {
			b.WriteString("-- TODO: 缺少索引字段\n\n")
			return
		}
		fmt.Fprintf(b, "ALTER TABLE %s ADD INDEX (%s);\n\n", quoteIdent(table), quoteIdent(column))

	default:
		if remark != "" {
			fmt.Fprintf(b, "-- 其他变更：%s\n\n", remark)
		} else {
			b.WriteString("-- 其他变更\n\n")
		}
	}
}

func writeChangeComment(b *strings.Builder, c model.DBChange) {
	label := KindLabel(c.Kind)
	table := strings.TrimSpace(c.TableName)
	column := strings.TrimSpace(c.ColumnName)
	remark := strings.TrimSpace(c.Remark)
	fmt.Fprintf(b, "-- %s", label)
	if table != "" {
		fmt.Fprintf(b, " %s", table)
	}
	if column != "" {
		fmt.Fprintf(b, ".%s", column)
	}
	if remark != "" {
		fmt.Fprintf(b, " - %s", remark)
	}
	b.WriteString("\n")
}

func quoteIdent(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

func escapeSQLString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "''")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
