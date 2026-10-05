package inspector

// TableInfo represents basic metadata for a database table.
type TableInfo struct {
	Name          string `json:"name"`
	EstimatedRows int64  `json:"estimated_rows"`
	TotalBytes    int64  `json:"total_bytes"`
	PrettySize    string `json:"pretty_size"`
}

// ColumnInfo describes a single table column.
type ColumnInfo struct {
	Name          string  `json:"name"`
	DataType      string  `json:"data_type"`
	IsNullable    bool    `json:"is_nullable"`
	ColumnDefault *string `json:"column_default,omitempty"`
	IsPrimaryKey  bool    `json:"is_primary_key"`
}

// IndexInfo describes an index on a table.
type IndexInfo struct {
	Name      string `json:"name"`
	Columns   string `json:"columns"`
	IsUnique  bool   `json:"is_unique"`
	IsPrimary bool   `json:"is_primary"`
}

// ConstraintInfo describes a table constraint.
type ConstraintInfo struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Definition string `json:"definition"`
}

// TableDetail contains full schema details for a table.
type TableDetail struct {
	Info        TableInfo        `json:"info"`
	Columns     []ColumnInfo     `json:"columns"`
	Indexes     []IndexInfo      `json:"indexes"`
	Constraints []ConstraintInfo `json:"constraints"`
}

// DatabaseStats represents high-level metrics for the database.
type DatabaseStats struct {
	DatabaseName      string  `json:"database_name"`
	Version           string  `json:"version"`
	TotalSize         string  `json:"total_size"`
	TotalTables       int     `json:"total_tables"`
	TotalIndexes      int     `json:"total_indexes"`
	ActiveConnections int     `json:"active_connections"`
	CacheHitRatio     float64 `json:"cache_hit_ratio"`
}

// RowsResult contains paginated table row records.
type RowsResult struct {
	Rows   []map[string]any `json:"rows"`
	Total  int64            `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}
