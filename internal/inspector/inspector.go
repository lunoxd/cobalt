package inspector

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/lunoxd/cobalt/internal/database"
)

var identifierRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// Inspector provides safe administrative inspection and data browsing of PostgreSQL.
type Inspector struct {
	db *database.DB
}

// NewInspector creates a new Inspector instance.
func NewInspector(db *database.DB) *Inspector {
	return &Inspector{db: db}
}

// validateIdentifier ensures the string is a safe SQL identifier name.
func validateIdentifier(name string) error {
	if !identifierRegex.MatchString(name) {
		return fmt.Errorf("invalid identifier name: %q", name)
	}
	return nil
}

// ListTables returns all public tables with size and row estimates.
func (ins *Inspector) ListTables(ctx context.Context) ([]TableInfo, error) {
	query := `
		SELECT
			c.relname AS table_name,
			c.reltuples::bigint AS estimated_rows,
			pg_total_relation_size(c.oid) AS total_bytes,
			pg_size_pretty(pg_total_relation_size(c.oid)) AS pretty_size
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public'
		  AND c.relkind = 'r'
		ORDER BY c.relname ASC;
	`

	rows, err := ins.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var t TableInfo
		if err := rows.Scan(&t.Name, &t.EstimatedRows, &t.TotalBytes, &t.PrettySize); err != nil {
			return nil, fmt.Errorf("failed to scan table info: %w", err)
		}
		if t.EstimatedRows < 0 {
			t.EstimatedRows = 0
		}
		tables = append(tables, t)
	}

	return tables, nil
}

// ListColumns returns column details for a given table.
func (ins *Inspector) ListColumns(ctx context.Context, tableName string) ([]ColumnInfo, error) {
	if err := validateIdentifier(tableName); err != nil {
		return nil, err
	}

	query := `
		SELECT
			c.column_name,
			c.data_type,
			c.is_nullable = 'YES' AS is_nullable,
			c.column_default,
			COALESCE(pk.is_pk, false) AS is_primary_key
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT kcu.column_name, true AS is_pk
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
			  ON tc.constraint_name = kcu.constraint_name
			 AND tc.table_schema = kcu.table_schema
			WHERE tc.constraint_type = 'PRIMARY KEY'
			  AND tc.table_schema = 'public'
			  AND tc.table_name = $1
		) pk ON c.column_name = pk.column_name
		WHERE c.table_schema = 'public'
		  AND c.table_name = $1
		ORDER BY c.ordinal_position ASC;
	`

	rows, err := ins.db.Pool.Query(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to list columns: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		if err := rows.Scan(&col.Name, &col.DataType, &col.IsNullable, &col.ColumnDefault, &col.IsPrimaryKey); err != nil {
			return nil, fmt.Errorf("failed to scan column info: %w", err)
		}
		columns = append(columns, col)
	}

	return columns, nil
}

// ListIndexes returns index details for a given table.
func (ins *Inspector) ListIndexes(ctx context.Context, tableName string) ([]IndexInfo, error) {
	if err := validateIdentifier(tableName); err != nil {
		return nil, err
	}

	query := `
		SELECT
			i.relname AS index_name,
			pg_get_indexdef(i.oid) AS columns,
			idx.indisunique AS is_unique,
			idx.indisprimary AS is_primary
		FROM pg_index idx
		JOIN pg_class t ON t.oid = idx.indrelid
		JOIN pg_class i ON i.oid = idx.indexrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = 'public'
		  AND t.relname = $1
		ORDER BY i.relname ASC;
	`

	rows, err := ins.db.Pool.Query(ctx, query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to list indexes: %w", err)
	}
	defer rows.Close()

	var indexes []IndexInfo
	for rows.Next() {
		var idx IndexInfo
		if err := rows.Scan(&idx.Name, &idx.Columns, &idx.IsUnique, &idx.IsPrimary); err != nil {
			return nil, fmt.Errorf("failed to scan index info: %w", err)
		}
		indexes = append(indexes, idx)
	}

	return indexes, nil
}

// DescribeTable returns comprehensive schema metadata for a table.
func (ins *Inspector) DescribeTable(ctx context.Context, tableName string) (*TableDetail, error) {
	if err := validateIdentifier(tableName); err != nil {
		return nil, err
	}

	cols, err := ins.ListColumns(ctx, tableName)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("table %q not found or has no columns", tableName)
	}

	idxs, err := ins.ListIndexes(ctx, tableName)
	if err != nil {
		return nil, err
	}

	// Fetch table info
	query := `
		SELECT
			c.relname,
			c.reltuples::bigint,
			pg_total_relation_size(c.oid),
			pg_size_pretty(pg_total_relation_size(c.oid))
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public' AND c.relname = $1;
	`
	var info TableInfo
	_ = ins.db.Pool.QueryRow(ctx, query, tableName).Scan(&info.Name, &info.EstimatedRows, &info.TotalBytes, &info.PrettySize)
	if info.EstimatedRows < 0 {
		info.EstimatedRows = 0
	}

	return &TableDetail{
		Info:        info,
		Columns:     cols,
		Indexes:     idxs,
		Constraints: []ConstraintInfo{},
	}, nil
}

// GetRows paginates and retrieves records from a table.
func (ins *Inspector) GetRows(ctx context.Context, tableName string, limit, offset int, orderBy, sortOrder string) (*RowsResult, error) {
	if err := validateIdentifier(tableName); err != nil {
		return nil, err
	}

	cols, err := ins.ListColumns(ctx, tableName)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, errors.New("table not found")
	}

	validColMap := make(map[string]bool)
	defaultOrderCol := cols[0].Name
	for _, c := range cols {
		validColMap[c.Name] = true
		if c.IsPrimaryKey {
			defaultOrderCol = c.Name
		}
	}

	if orderBy == "" || !validColMap[orderBy] {
		orderBy = defaultOrderCol
	}

	sortOrder = strings.ToUpper(sortOrder)
	if sortOrder != "DESC" {
		sortOrder = "ASC"
	}

	if limit <= 0 || limit > 100 {
		limit = 25
	}
	if offset < 0 {
		offset = 0
	}

	// Safe quoting using pgx Identifier
	safeTable := pgx.Identifier{tableName}.Sanitize()
	safeOrderCol := pgx.Identifier{orderBy}.Sanitize()

	// 1. Get exact count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s;", safeTable)
	var total int64
	if err := ins.db.Pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed to count rows: %w", err)
	}

	// 2. Select rows
	dataQuery := fmt.Sprintf("SELECT * FROM %s ORDER BY %s %s LIMIT $1 OFFSET $2;", safeTable, safeOrderCol, sortOrder)
	rows, err := ins.db.Pool.Query(ctx, dataQuery, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query rows: %w", err)
	}
	defer rows.Close()

	fieldDescriptions := rows.FieldDescriptions()
	var records []map[string]any

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, fmt.Errorf("failed to scan row values: %w", err)
		}
		record := make(map[string]any)
		for i, fd := range fieldDescriptions {
			record[fd.Name] = values[i]
		}
		records = append(records, record)
	}

	return &RowsResult{
		Rows:   records,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// InsertRow safely inserts a single record with parameterized inputs.
func (ins *Inspector) InsertRow(ctx context.Context, tableName string, data map[string]any) (map[string]any, error) {
	if err := validateIdentifier(tableName); err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("cannot insert empty record")
	}

	cols, err := ins.ListColumns(ctx, tableName)
	if err != nil {
		return nil, err
	}
	validColMap := make(map[string]bool)
	for _, c := range cols {
		validColMap[c.Name] = true
	}

	var (
		colNames []string
		holders  []string
		args     []any
	)
	idx := 1
	for k, v := range data {
		if !validColMap[k] {
			return nil, fmt.Errorf("column %q does not exist on table %q", k, tableName)
		}
		colNames = append(colNames, pgx.Identifier{k}.Sanitize())
		holders = append(holders, fmt.Sprintf("$%d", idx))
		args = append(args, v)
		idx++
	}

	safeTable := pgx.Identifier{tableName}.Sanitize()
	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) RETURNING *;",
		safeTable,
		strings.Join(colNames, ", "),
		strings.Join(holders, ", "),
	)

	rows, err := ins.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to insert row: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		result := make(map[string]any)
		for i, fd := range rows.FieldDescriptions() {
			result[fd.Name] = values[i]
		}
		return result, nil
	}

	return nil, errors.New("no row returned after insert")
}

// UpdateRow safely updates a single record by primary key with parameterized inputs.
func (ins *Inspector) UpdateRow(ctx context.Context, tableName, pkCol string, pkVal any, data map[string]any) (map[string]any, error) {
	if err := validateIdentifier(tableName); err != nil {
		return nil, err
	}
	if err := validateIdentifier(pkCol); err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("no columns to update")
	}

	cols, err := ins.ListColumns(ctx, tableName)
	if err != nil {
		return nil, err
	}
	validColMap := make(map[string]bool)
	for _, c := range cols {
		validColMap[c.Name] = true
	}

	var sets []string
	var args []any
	idx := 1

	for k, v := range data {
		if !validColMap[k] {
			return nil, fmt.Errorf("column %q does not exist on table %q", k, tableName)
		}
		sets = append(sets, fmt.Sprintf("%s = $%d", pgx.Identifier{k}.Sanitize(), idx))
		args = append(args, v)
		idx++
	}

	args = append(args, pkVal)
	safeTable := pgx.Identifier{tableName}.Sanitize()
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = $%d RETURNING *;",
		safeTable,
		strings.Join(sets, ", "),
		pgx.Identifier{pkCol}.Sanitize(),
		idx,
	)

	rows, err := ins.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update row: %w", err)
	}
	defer rows.Close()

	if rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return nil, err
		}
		result := make(map[string]any)
		for i, fd := range rows.FieldDescriptions() {
			result[fd.Name] = values[i]
		}
		return result, nil
	}

	return nil, errors.New("row not found or not modified")
}

// DeleteRow safely deletes a single record by primary key.
func (ins *Inspector) DeleteRow(ctx context.Context, tableName, pkCol string, pkVal any) error {
	if err := validateIdentifier(tableName); err != nil {
		return err
	}
	if err := validateIdentifier(pkCol); err != nil {
		return err
	}

	safeTable := pgx.Identifier{tableName}.Sanitize()
	safePK := pgx.Identifier{pkCol}.Sanitize()

	query := fmt.Sprintf("DELETE FROM %s WHERE %s = $1;", safeTable, safePK)
	cmd, err := ins.db.Pool.Exec(ctx, query, pkVal)
	if err != nil {
		return fmt.Errorf("failed to delete row: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("row not found or already deleted")
	}

	return nil
}

// DatabaseStats returns high-level PostgreSQL metrics.
func (ins *Inspector) DatabaseStats(ctx context.Context) (*DatabaseStats, error) {
	if ins.db == nil || ins.db.Pool == nil {
		return nil, errors.New("database connection is not available")
	}
	stats := &DatabaseStats{}

	// Database name, version, and total size
	metaQuery := `
		SELECT
			current_database(),
			version(),
			pg_size_pretty(pg_database_size(current_database()))
	`
	if err := ins.db.Pool.QueryRow(ctx, metaQuery).Scan(&stats.DatabaseName, &stats.Version, &stats.TotalSize); err != nil {
		return nil, fmt.Errorf("failed to query database meta: %w", err)
	}

	// Table and index counts
	countsQuery := `
		SELECT
			COUNT(*) FILTER (WHERE relkind = 'r'),
			COUNT(*) FILTER (WHERE relkind = 'i')
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'public';
	`
	_ = ins.db.Pool.QueryRow(ctx, countsQuery).Scan(&stats.TotalTables, &stats.TotalIndexes)

	// Active connection count
	connsQuery := `SELECT COUNT(*) FROM pg_stat_activity WHERE datname = current_database();`
	_ = ins.db.Pool.QueryRow(ctx, connsQuery).Scan(&stats.ActiveConnections)

	// Cache hit ratio
	cacheQuery := `
		SELECT
			COALESCE(ROUND(sum(heap_blks_hit) / (sum(heap_blks_hit) + sum(heap_blks_read) + 0.00001) * 100, 2), 100.0)
		FROM pg_statio_user_tables;
	`
	_ = ins.db.Pool.QueryRow(ctx, cacheQuery).Scan(&stats.CacheHitRatio)

	return stats, nil
}
