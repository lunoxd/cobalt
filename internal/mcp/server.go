package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/lunoxd/cobalt/internal/auth"
	"github.com/lunoxd/cobalt/internal/inspector"
	"github.com/lunoxd/cobalt/internal/projects"
)

// Server implements the Model Context Protocol (MCP) server over HTTP JSON-RPC 2.0.
type Server struct {
	inspector *inspector.Inspector
	projects  *projects.Service
	tools     map[string]ToolDefinition
}

// NewServer creates a new MCP server.
func NewServer(ins *inspector.Inspector, proj *projects.Service) *Server {
	toolMap := make(map[string]ToolDefinition)
	for _, t := range GetRegisteredTools() {
		toolMap[t.Name] = t
	}
	return &Server{
		inspector: ins,
		projects:  proj,
		tools:     toolMap,
	}
}

// Handler returns the HTTP handler for MCP requests.
func (s *Server) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// GET /mcp returns the server metadata and list of tools for easy inspection
		if r.Method == http.MethodGet {
			caller, _ := auth.GetIdentity(r.Context())
			var allowedTools []ToolDefinition
			for _, t := range GetRegisteredTools() {
				if caller == nil || caller.HasScope(t.RequiredScope) {
					allowedTools = append(allowedTools, t)
				}
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"mcp_server": "cobalt-mcp",
				"version":    "1.0.0",
				"protocol":   "2024-11-05",
				"tools":      allowedTools,
			})
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, `{"error":{"code":"METHOD_NOT_ALLOWED","message":"POST or GET required"}}`, http.StatusMethodNotAllowed)
			return
		}

		// MCP JSON-RPC 2.0 execution requires authentication
		caller, ok := auth.GetIdentity(r.Context())
		if !ok || caller == nil {
			s.writeRPCError(w, nil, CodeUnauthorized, "Authentication required to interact with MCP server")
			return
		}

		var req JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeRPCError(w, nil, CodeParseError, "Parse error: "+err.Error())
			return
		}

		if req.JSONRPC != "2.0" {
			s.writeRPCError(w, req.ID, CodeInvalidRequest, "Invalid JSON-RPC version; must be '2.0'")
			return
		}

		slog.Info("mcp request",
			"method", req.Method,
			"caller", caller.UserID,
			"role", caller.Role,
		)

		switch req.Method {
		case "initialize":
			s.handleInitialize(w, req)
		case "ping":
			s.writeRPCResult(w, req.ID, map[string]any{})
		case "tools/list":
			s.handleToolsList(w, req, caller)
		case "tools/call":
			s.handleToolCall(w, req, caller)
		default:
			s.writeRPCError(w, req.ID, CodeMethodNotFound, fmt.Sprintf("Method %q not found", req.Method))
		}
	}
}

func (s *Server) handleInitialize(w http.ResponseWriter, req JSONRPCRequest) {
	result := map[string]any{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]string{
			"name":    "cobalt-mcp",
			"version": "1.0.0",
		},
		"capabilities": map[string]any{
			"tools": map[string]any{
				"listChanged": false,
			},
		},
	}
	s.writeRPCResult(w, req.ID, result)
}

func (s *Server) handleToolsList(w http.ResponseWriter, req JSONRPCRequest, caller *auth.Identity) {
	var accessibleTools []ToolDefinition
	for _, tool := range GetRegisteredTools() {
		if caller.HasScope(tool.RequiredScope) {
			accessibleTools = append(accessibleTools, tool)
		}
	}
	s.writeRPCResult(w, req.ID, map[string]any{
		"tools": accessibleTools,
	})
}

func (s *Server) handleToolCall(w http.ResponseWriter, req JSONRPCRequest, caller *auth.Identity) {
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.writeRPCError(w, req.ID, CodeInvalidParams, "Invalid tool call params: "+err.Error())
		return
	}

	tool, exists := s.tools[params.Name]
	if !exists {
		s.writeRPCError(w, req.ID, CodeMethodNotFound, fmt.Sprintf("Tool %q not found", params.Name))
		return
	}

	// Scope verification
	if !caller.HasScope(tool.RequiredScope) {
		s.writeRPCError(w, req.ID, CodeForbidden,
			fmt.Sprintf("Forbidden: tool %q requires scope %q", tool.Name, tool.RequiredScope))
		return
	}

	// Dispatch tool
	resultData, err := s.executeTool(context.Background(), params.Name, params.Arguments, caller)
	if err != nil {
		s.writeRPCResult(w, req.ID, ToolCallResult{
			Content: []ToolCallContent{
				{Type: "text", Text: fmt.Sprintf("Error executing tool %s: %v", params.Name, err)},
			},
			IsError: true,
		})
		return
	}

	jsonBytes, _ := json.MarshalIndent(resultData, "", "  ")
	s.writeRPCResult(w, req.ID, ToolCallResult{
		Content: []ToolCallContent{
			{Type: "text", Text: string(jsonBytes)},
		},
	})
}

func (s *Server) executeTool(ctx context.Context, name string, args map[string]any, caller *auth.Identity) (any, error) {
	switch name {
	case "list_tables":
		return s.inspector.ListTables(ctx)

	case "describe_table":
		tbl, ok := args["table_name"].(string)
		if !ok || tbl == "" {
			return nil, fmt.Errorf("table_name string argument is required")
		}
		return s.inspector.DescribeTable(ctx, tbl)

	case "list_columns":
		tbl, ok := args["table_name"].(string)
		if !ok || tbl == "" {
			return nil, fmt.Errorf("table_name string argument is required")
		}
		return s.inspector.ListColumns(ctx, tbl)

	case "list_indexes":
		tbl, ok := args["table_name"].(string)
		if !ok || tbl == "" {
			return nil, fmt.Errorf("table_name string argument is required")
		}
		return s.inspector.ListIndexes(ctx, tbl)

	case "get_rows":
		tbl, ok := args["table_name"].(string)
		if !ok || tbl == "" {
			return nil, fmt.Errorf("table_name string argument is required")
		}
		limit := 25
		if l, ok := args["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}
		offset := 0
		if o, ok := args["offset"].(float64); ok && o >= 0 {
			offset = int(o)
		}
		orderBy, _ := args["order_by"].(string)
		sort, _ := args["sort"].(string)
		return s.inspector.GetRows(ctx, tbl, limit, offset, orderBy, sort)

	case "insert_row":
		tbl, ok := args["table_name"].(string)
		if !ok || tbl == "" {
			return nil, fmt.Errorf("table_name string argument is required")
		}
		data, ok := args["data"].(map[string]any)
		if !ok || len(data) == 0 {
			return nil, fmt.Errorf("data object argument is required")
		}
		return s.inspector.InsertRow(ctx, tbl, data)

	case "update_row":
		tbl, ok := args["table_name"].(string)
		pkCol, okPK := args["pk_column"].(string)
		pkVal, okVal := args["pk_value"]
		data, okData := args["data"].(map[string]any)
		if !ok || !okPK || !okVal || !okData {
			return nil, fmt.Errorf("table_name, pk_column, pk_value, and data are required")
		}
		return s.inspector.UpdateRow(ctx, tbl, pkCol, pkVal, data)

	case "delete_row":
		tbl, ok := args["table_name"].(string)
		pkCol, okPK := args["pk_column"].(string)
		pkVal, okVal := args["pk_value"]
		if !ok || !okPK || !okVal {
			return nil, fmt.Errorf("table_name, pk_column, and pk_value are required")
		}
		err := s.inspector.DeleteRow(ctx, tbl, pkCol, pkVal)
		if err != nil {
			return nil, err
		}
		return map[string]string{"message": "Row deleted successfully"}, nil

	case "database_stats":
		return s.inspector.DatabaseStats(ctx)

	case "list_projects":
		limit := 20
		if l, ok := args["limit"].(float64); ok && l > 0 {
			limit = int(l)
		}
		offset := 0
		if o, ok := args["offset"].(float64); ok && o >= 0 {
			offset = int(o)
		}
		items, total, err := s.projects.ListProjects(ctx, caller, limit, offset)
		if err != nil {
			return nil, err
		}
		return map[string]any{"data": items, "total": total}, nil

	case "get_project":
		projIDStr, ok := args["project_id"].(string)
		if !ok || projIDStr == "" {
			return nil, fmt.Errorf("project_id argument is required")
		}
		id, err := uuid.Parse(projIDStr)
		if err != nil {
			return nil, fmt.Errorf("invalid project_id uuid: %w", err)
		}
		return s.projects.GetProject(ctx, id, caller)

	default:
		return nil, fmt.Errorf("unsupported tool %q", name)
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) writeRPCResult(w http.ResponseWriter, id any, result any) {
	writeJSON(w, http.StatusOK, JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (s *Server) writeRPCError(w http.ResponseWriter, id any, code int, msg string) {
	writeJSON(w, http.StatusOK, JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: msg,
		},
	})
}
