package main

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type ConnectionConfiguration struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSLMode  string `json:"sslMode"`
}

type ConnectionInfo struct {
	Database string `json:"database"`
	Version  string `json:"version"`
}

type QueryRequest struct {
	SQL     string `json:"sql"`
	MaxRows int    `json:"maxRows"`
	Timeout int    `json:"timeoutSeconds"`
}

type QueryResult struct {
	Columns      []string `json:"columns"`
	Rows         [][]any  `json:"rows"`
	AffectedRows int64    `json:"affectedRows"`
	DurationMs   int64    `json:"durationMs"`
	ReadOnly     bool     `json:"readOnly"`
}

type TableInfo struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
	Kind   string `json:"kind"`
}

type DatabaseInfo struct {
	Name       string `json:"name"`
	CanConnect bool   `json:"canConnect"`
}

type TableData struct {
	Schema     string     `json:"schema"`
	Name       string     `json:"name"`
	Columns    []string   `json:"columns"`
	Rows       [][]string `json:"rows"`
	PrimaryKey string     `json:"primaryKey"`
	DurationMs int64      `json:"durationMs"`
}

const nullValueToken = "\x00DBACCESS_NULL"

type DatabaseService struct {
	ctx context.Context
	db  *sql.DB
	mu  sync.Mutex
}

func NewDatabaseService() *DatabaseService { return &DatabaseService{} }

func (s *DatabaseService) SetContext(ctx context.Context) { s.ctx = ctx }

func (s *DatabaseService) Connect(config ConnectionConfiguration) (ConnectionInfo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if config.Host == "" || config.Username == "" {
		return ConnectionInfo{}, fmt.Errorf("host、usernameは必須です")
	}
	if config.Database == "" {
		config.Database = "postgres"
	}
	if config.Port == 0 {
		config.Port = 5432
	}
	if config.SSLMode == "" {
		config.SSLMode = "prefer"
	}
	dsnURL := url.URL{
		Scheme: "postgresql",
		User:   url.UserPassword(config.Username, config.Password),
		Host:   net.JoinHostPort(config.Host, strconv.Itoa(config.Port)),
		Path:   "/" + config.Database,
	}
	query := url.Values{}
	query.Set("sslmode", config.SSLMode)
	dsnURL.RawQuery = query.Encode()
	dsn := dsnURL.String()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return ConnectionInfo{}, fmt.Errorf("接続の準備に失敗しました: %w", err)
	}
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(2)
	db.SetConnMaxIdleTime(5 * time.Minute)
	ctx, cancel := context.WithTimeout(s.context(), 8*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return ConnectionInfo{}, fmt.Errorf("PostgreSQLへ接続できません: %w", err)
	}
	var version string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
		version = ""
	}
	previous := s.db
	s.db = db
	if previous != nil {
		_ = previous.Close()
	}
	return ConnectionInfo{Database: config.Database, Version: version}, nil
}

func (s *DatabaseService) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

func (s *DatabaseService) Close() { _ = s.Disconnect() }

func (s *DatabaseService) ListTables() ([]TableInfo, error) {
	db, err := s.connection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(s.context(), 10*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, `
		SELECT table_schema, table_name, table_type
		FROM information_schema.tables
		WHERE table_schema NOT IN ('pg_catalog', 'information_schema')
		ORDER BY table_schema, table_name`)
	if err != nil {
		return nil, fmt.Errorf("テーブル一覧の取得に失敗しました: %w", err)
	}
	defer rows.Close()
	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		if err := rows.Scan(&table.Schema, &table.Name, &table.Kind); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, rows.Err()
}

func (s *DatabaseService) ListDatabases() ([]DatabaseInfo, error) {
	db, err := s.connection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(s.context(), 10*time.Second)
	defer cancel()
	rows, err := db.QueryContext(ctx, `
		SELECT datname, datallowconn
		FROM pg_database
		WHERE datistemplate = false
		ORDER BY datname`)
	if err != nil {
		return nil, fmt.Errorf("データベース一覧の取得に失敗しました: %w", err)
	}
	defer rows.Close()
	var databases []DatabaseInfo
	for rows.Next() {
		var database DatabaseInfo
		if err := rows.Scan(&database.Name, &database.CanConnect); err != nil {
			return nil, err
		}
		databases = append(databases, database)
	}
	return databases, rows.Err()
}

func (s *DatabaseService) LoadTable(schema, table string, maxRows int) (TableData, error) {
	db, err := s.connection()
	if err != nil {
		return TableData{}, err
	}
	if maxRows <= 0 || maxRows > 500 {
		maxRows = 100
	}
	if !validIdentifier(schema) || !validIdentifier(table) {
		return TableData{}, fmt.Errorf("不正なテーブル名です")
	}
	ctx, cancel := context.WithTimeout(s.context(), 10*time.Second)
	defer cancel()
	started := time.Now()
	query := fmt.Sprintf("SELECT * FROM %s.%s LIMIT $1", quoteIdentifier(schema), quoteIdentifier(table))
	rows, err := db.QueryContext(ctx, query, maxRows)
	if err != nil {
		return TableData{}, fmt.Errorf("テーブルデータの取得に失敗しました: %w", err)
	}
	defer rows.Close()
	columns, err := rows.Columns()
	if err != nil {
		return TableData{}, err
	}
	data := TableData{Schema: schema, Name: table, Columns: columns, Rows: make([][]string, 0), DurationMs: time.Since(started).Milliseconds()}
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return TableData{}, fmt.Errorf("テーブルデータの読み取りに失敗しました: %w", err)
		}
		data.Rows = append(data.Rows, normalizeRow(values))
	}
	if err := rows.Err(); err != nil {
		return TableData{}, err
	}
	data.PrimaryKey = s.primaryKey(ctx, schema, table)
	data.DurationMs = time.Since(started).Milliseconds()
	return data, nil
}

func (s *DatabaseService) UpdateCell(schema, table, column, primaryKey, primaryKeyValue, value string) error {
	db, err := s.connection()
	if err != nil {
		return err
	}
	if !validIdentifier(schema) || !validIdentifier(table) || !validIdentifier(column) || !validIdentifier(primaryKey) {
		return fmt.Errorf("不正なテーブル名またはカラム名です")
	}
	if primaryKey == "" {
		return fmt.Errorf("主キーがないテーブルはセル編集に対応していません")
	}
	ctx, cancel := context.WithTimeout(s.context(), 10*time.Second)
	defer cancel()
	query := fmt.Sprintf("UPDATE %s.%s SET %s = $1 WHERE %s = $2", quoteIdentifier(schema), quoteIdentifier(table), quoteIdentifier(column), quoteIdentifier(primaryKey))
	var updateValue any = value
	if value == nullValueToken {
		updateValue = nil
	}
	if _, err := db.ExecContext(ctx, query, updateValue, primaryKeyValue); err != nil {
		return fmt.Errorf("変更の保存に失敗しました: %w", err)
	}
	return nil
}

func (s *DatabaseService) connection() (*sql.DB, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil, fmt.Errorf("データベースに接続されていません")
	}
	return s.db, nil
}

func (s *DatabaseService) primaryKey(ctx context.Context, schema, table string) string {
	var column string
	err := s.db.QueryRowContext(ctx, `
		SELECT kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema = kcu.table_schema
		 AND tc.table_name = kcu.table_name
		WHERE tc.constraint_type = 'PRIMARY KEY'
		  AND tc.table_schema = $1 AND tc.table_name = $2
		ORDER BY kcu.ordinal_position LIMIT 1`, schema, table).Scan(&column)
	if err != nil {
		return ""
	}
	return column
}

func validIdentifier(value string) bool {
	return value != "" && !strings.ContainsRune(value, 0)
}

func quoteIdentifier(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func (s *DatabaseService) execute(request QueryRequest) (QueryResult, error) {
	s.mu.Lock()
	db := s.db
	s.mu.Unlock()
	if db == nil {
		return QueryResult{}, fmt.Errorf("データベースに接続されていません")
	}
	statement := strings.TrimSpace(request.SQL)
	if statement == "" {
		return QueryResult{}, fmt.Errorf("SQLを入力してください")
	}
	maxRows := request.MaxRows
	if maxRows <= 0 || maxRows > 10000 {
		maxRows = 100
	}
	timeout := request.Timeout
	if timeout <= 0 || timeout > 600 {
		timeout = 60
	}
	ctx, cancel := context.WithTimeout(s.context(), time.Duration(timeout)*time.Second)
	defer cancel()
	started := time.Now()
	if isReadQuery(statement) {
		rows, err := db.QueryContext(ctx, statement)
		if err != nil {
			return QueryResult{}, fmt.Errorf("SQL実行エラー: %w", err)
		}
		defer rows.Close()
		columns, err := rows.Columns()
		if err != nil {
			return QueryResult{}, err
		}
		result := QueryResult{Columns: columns, ReadOnly: true}
		for len(result.Rows) < maxRows && rows.Next() {
			values := make([]any, len(columns))
			pointers := make([]any, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				return QueryResult{}, fmt.Errorf("結果の読み取りに失敗しました: %w", err)
			}
			result.Rows = append(result.Rows, normalizeAnyRow(values))
		}
		if err := rows.Err(); err != nil {
			return QueryResult{}, err
		}
		result.DurationMs = time.Since(started).Milliseconds()
		return result, nil
	}
	res, err := db.ExecContext(ctx, statement)
	if err != nil {
		return QueryResult{}, fmt.Errorf("SQL実行エラー: %w", err)
	}
	affected, _ := res.RowsAffected()
	return QueryResult{AffectedRows: affected, DurationMs: time.Since(started).Milliseconds()}, nil
}

func (s *DatabaseService) context() context.Context {
	if s.ctx != nil {
		return s.ctx
	}
	return context.Background()
}

func isReadQuery(sqlText string) bool {
	upper := strings.ToUpper(strings.TrimSpace(sqlText))
	return strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "WITH") || strings.HasPrefix(upper, "SHOW") || strings.HasPrefix(upper, "EXPLAIN") || strings.HasPrefix(upper, "TABLE")
}

func normalizeRow(values []any) []string {
	result := make([]string, len(values))
	for i, value := range values {
		switch typed := value.(type) {
		case []byte:
			result[i] = string(typed)
		case time.Time:
			result[i] = typed.Format(time.RFC3339Nano)
		case nil:
			result[i] = nullValueToken
		default:
			result[i] = fmt.Sprint(value)
		}
	}
	return result
}

func normalizeAnyRow(values []any) []any {
	result := make([]any, len(values))
	for i, value := range values {
		switch typed := value.(type) {
		case []byte:
			result[i] = string(typed)
		case time.Time:
			result[i] = typed.Format(time.RFC3339Nano)
		default:
			result[i] = value
		}
	}
	return result
}
