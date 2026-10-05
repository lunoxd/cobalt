package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lunoxd/cobalt/internal/auth"
)

func TestMCPServer_Unauthenticated(t *testing.T) {
	srv := NewServer(nil, nil)
	handler := srv.Handler()

	body := `{"jsonrpc": "2.0", "id": 1, "method": "tools/list"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var res JSONRPCResponse
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Error == nil {
		t.Fatalf("expected error for unauthenticated request, got nil")
	}
	if res.Error.Code != CodeUnauthorized {
		t.Errorf("expected error code %d, got %d", CodeUnauthorized, res.Error.Code)
	}
}

func TestMCPServer_Initialize(t *testing.T) {
	srv := NewServer(nil, nil)
	handler := srv.Handler()

	body := `{"jsonrpc": "2.0", "id": "test-init", "method": "initialize"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
	// Add caller identity
	caller := &auth.Identity{
		UserID: "user_test",
		Role:   auth.RoleUser,
		Scopes: []string{auth.ScopeProjectsRead},
	}
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), caller))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var res JSONRPCResponse
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Error != nil {
		t.Fatalf("unexpected error: %v", res.Error.Message)
	}

	resultMap, ok := res.Result.(map[string]any)
	if !ok {
		t.Fatalf("expected result map, got %T", res.Result)
	}
	if resultMap["protocolVersion"] != "2024-11-05" {
		t.Errorf("expected protocolVersion 2024-11-05, got %v", resultMap["protocolVersion"])
	}
}

func TestMCPServer_ScopeEnforcement(t *testing.T) {
	srv := NewServer(nil, nil)
	handler := srv.Handler()

	// Caller only has database:read, but calls insert_row (which requires database:write)
	caller := &auth.Identity{
		UserID: "readonly_user",
		Role:   auth.RoleUser,
		Scopes: []string{auth.ScopeDatabaseRead},
	}

	callPayload := map[string]any{
		"jsonrpc": "2.0",
		"id":      42,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "insert_row",
			"arguments": map[string]any{
				"table_name": "projects",
				"data":       map[string]any{"name": "test"},
			},
		},
	}
	payloadBytes, _ := json.Marshal(callPayload)

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBuffer(payloadBytes))
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), caller))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var res JSONRPCResponse
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.Error == nil {
		t.Fatalf("expected forbidden error for lacking database:write scope, got success")
	}
	if res.Error.Code != CodeForbidden {
		t.Errorf("expected CodeForbidden (-32001), got %d", res.Error.Code)
	}
}
