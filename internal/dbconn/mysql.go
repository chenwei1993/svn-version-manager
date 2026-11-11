package dbconn

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strconv"
	"time"

	"project-manage/internal/model"

	"github.com/go-sql-driver/mysql"
)

// Config 数据库连接参数
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
}

func (c Config) normalize() Config {
	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	if c.Port == 0 {
		c.Port = 3306
	}
	return c
}

// Open 打开并测试 MySQL 连接
func Open(c Config) (*sql.DB, error) {
	c = c.normalize()
	if c.Database == "" {
		return nil, fmt.Errorf("请填写数据库名")
	}
	cfg := mysql.NewConfig()
	cfg.User = c.User
	cfg.Passwd = c.Password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	cfg.DBName = c.Database
	cfg.ParseTime = true
	cfg.Timeout = 6 * time.Second
	cfg.ReadTimeout = 20 * time.Second
	cfg.WriteTimeout = 20 * time.Second
	cfg.Params = map[string]string{"charset": "utf8mb4"}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	return db, nil
}

// Test 测试连接并返回数据库版本
func Test(c Config) (string, error) {
	db, err := Open(c)
	if err != nil {
		return "", err
	}
	defer db.Close()
	var version string
	if err := db.QueryRow("SELECT VERSION()").Scan(&version); err != nil {
		return "", err
	}
	return version, nil
}

// Schema 读取数据库全部表及字段
func Schema(c Config) ([]model.DBTable, error) {
	db, err := Open(c)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return readSchema(db, c.Database, "")
}

// TableSchema 读取单张表及字段
func TableSchema(c Config, table string) (model.DBTable, error) {
	db, err := Open(c)
	if err != nil {
		return model.DBTable{}, err
	}
	defer db.Close()
	tables, err := readSchema(db, c.Database, table)
	if err != nil {
		return model.DBTable{}, err
	}
	if len(tables) == 0 {
		return model.DBTable{}, fmt.Errorf("表不存在: %s", table)
	}
	return tables[0], nil
}

func readSchema(db *sql.DB, database, table string) ([]model.DBTable, error) {
	query := `SELECT t.TABLE_NAME, t.TABLE_COMMENT,
		c.COLUMN_NAME, c.COLUMN_TYPE, c.IS_NULLABLE, c.COLUMN_DEFAULT,
		c.COLUMN_COMMENT, c.COLUMN_KEY, c.EXTRA
		FROM information_schema.TABLES t
		JOIN information_schema.COLUMNS c
			ON c.TABLE_SCHEMA = t.TABLE_SCHEMA AND c.TABLE_NAME = t.TABLE_NAME
		WHERE t.TABLE_SCHEMA = ? AND t.TABLE_TYPE = 'BASE TABLE'`
	args := []interface{}{database}
	if table != "" {
		query += " AND t.TABLE_NAME = ?"
		args = append(args, table)
	}
	query += " ORDER BY t.TABLE_NAME, c.ORDINAL_POSITION"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type tableRef struct {
		table *model.DBTable
	}
	byName := make(map[string]*tableRef)
	order := make([]string, 0)
	for rows.Next() {
		var tableName, tableComment string
		var columnName, columnType, nullable, columnComment, columnKey, extra string
		var defaultValue sql.NullString
		if err := rows.Scan(&tableName, &tableComment, &columnName, &columnType, &nullable,
			&defaultValue, &columnComment, &columnKey, &extra); err != nil {
			return nil, err
		}
		ref := byName[tableName]
		if ref == nil {
			ref = &tableRef{table: &model.DBTable{Name: tableName, Comment: tableComment}}
			byName[tableName] = ref
			order = append(order, tableName)
		}
		def := ""
		if defaultValue.Valid {
			def = defaultValue.String
		}
		ref.table.Columns = append(ref.table.Columns, model.DBColumn{
			Name:     columnName,
			Type:     columnType,
			Nullable: nullable,
			Default:  def,
			Comment:  columnComment,
			Key:      columnKey,
			Extra:    extra,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tables := make([]model.DBTable, 0, len(order))
	for _, name := range order {
		tables = append(tables, *byName[name].table)
	}
	return tables, nil
}
