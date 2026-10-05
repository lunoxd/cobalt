package mcp

import (
	"github.com/lunoxd/cobalt/internal/auth"
)

// GetRegisteredTools returns the list of all available MCP tools with their schemas and required scopes.
func GetRegisteredTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:          "list_tables",
			Description:   "List all public database tables with row counts and storage sizes.",
			RequiredScope: auth.ScopeDatabaseRead,
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:          "describe_table",
			Description:   "Get full schema definition for a table including columns, data types, indexes, and primary keys.",
			RequiredScope: auth.ScopeDatabaseRead,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"table_name": map[string]any{
						"type":        "string",
						"description": "Name of the table to describe",
					},
				},
				"required": []string{"table_name"},
			},
		},
		{
			Name:          "list_columns",
			Description:   "List columns, data types, and nullability for a table.",
			RequiredScope: auth.ScopeDatabaseRead,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"table_name": map[string]any{
						"type":        "string",
						"description": "Name of the table",
					},
				},
				"required": []string{"table_name"},
			},
		},
		{
			Name:          "list_indexes",
			Description:   "List indexes, columns, and uniqueness for a table.",
			RequiredScope: auth.ScopeDatabaseRead,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"table_name": map[string]any{
						"type":        "string",
						"description": "Name of the table",
					},
				},
				"required": []string{"table_name"},
			},
		},
		{
			Name:          "get_rows",
			Description:   "Safely browse and paginate table rows with optional ordering.",
			RequiredScope: auth.ScopeDatabaseRead,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"table_name": map[string]any{
						"type":        "string",
						"description": "Table to read rows from",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum number of rows to return (default: 25, max: 100)",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Number of rows to skip",
					},
					"order_by": map[string]any{
						"type":        "string",
						"description": "Column name to sort by",
					},
					"sort": map[string]any{
						"type":        "string",
						"enum":        []string{"ASC", "DESC", "asc", "desc"},
						"description": "Sort direction (ASC or DESC)",
					},
				},
				"required": []string{"table_name"},
			},
		},
		{
			Name:          "insert_row",
			Description:   "Safely insert a single row into a table using parameterized values.",
			RequiredScope: auth.ScopeDatabaseWrite,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"table_name": map[string]any{
						"type":        "string",
						"description": "Target table name",
					},
					"data": map[string]any{
						"type":        "object",
						"description": "Key-value map of column data to insert",
					},
				},
				"required": []string{"table_name", "data"},
			},
		},
		{
			Name:          "update_row",
			Description:   "Safely update a single row by primary key using parameterized values.",
			RequiredScope: auth.ScopeDatabaseWrite,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"table_name": map[string]any{
						"type":        "string",
						"description": "Target table name",
					},
					"pk_column": map[string]any{
						"type":        "string",
						"description": "Primary key column name",
					},
					"pk_value": map[string]any{
						"description": "Primary key value of the row to update",
					},
					"data": map[string]any{
						"type":        "object",
						"description": "Columns to update and their new values",
					},
				},
				"required": []string{"table_name", "pk_column", "pk_value", "data"},
			},
		},
		{
			Name:          "delete_row",
			Description:   "Safely delete a single row from a table by primary key.",
			RequiredScope: auth.ScopeDatabaseWrite,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"table_name": map[string]any{
						"type":        "string",
						"description": "Target table name",
					},
					"pk_column": map[string]any{
						"type":        "string",
						"description": "Primary key column name",
					},
					"pk_value": map[string]any{
						"description": "Primary key value of the row to delete",
					},
				},
				"required": []string{"table_name", "pk_column", "pk_value"},
			},
		},
		{
			Name:          "database_stats",
			Description:   "Get high-level database metrics: size, active connections, total tables, and cache ratio.",
			RequiredScope: auth.ScopeDatabaseRead,
			InputSchema: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			Name:          "list_projects",
			Description:   "List user projects with pagination.",
			RequiredScope: auth.ScopeProjectsRead,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{
						"type":        "integer",
						"description": "Number of projects to return (default: 20)",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Offset for pagination",
					},
				},
			},
		},
		{
			Name:          "get_project",
			Description:   "Fetch a single project by UUID.",
			RequiredScope: auth.ScopeProjectsRead,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project_id": map[string]any{
						"type":        "string",
						"description": "UUID of the project",
					},
				},
				"required": []string{"project_id"},
			},
		},
	}
}
