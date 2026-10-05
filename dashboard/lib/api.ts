// API Client for the Cobalt Go backend

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export function getHeaders(token?: string) {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  const effectiveToken = token || (typeof window !== "undefined" ? localStorage.getItem("cobalt_admin_key") : null) || "cb_admin_secret_key_change_in_prod";
  if (effectiveToken) {
    headers["Authorization"] = `Bearer ${effectiveToken}`;
  }
  return headers;
}

export async function fetchWithAuth(path: string, options: RequestInit = {}) {
  const headers = getHeaders();
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      ...headers,
      ...options.headers,
    },
  });

  if (!res.ok) {
    const errorText = await res.text();
    let errorJson;
    try {
      errorJson = JSON.parse(errorText);
    } catch {
      // not JSON
    }
    throw new Error(errorJson?.error?.message || errorJson?.error || `Request failed with status ${res.status}`);
  }

  return res.json();
}

// System stats
export async function getSystemStats() {
  return fetchWithAuth("/api/admin/stats");
}

// Database Inspector
export async function listTables() {
  return fetchWithAuth("/api/admin/inspector/tables");
}

export async function describeTable(table: string) {
  return fetchWithAuth(`/api/admin/inspector/tables/${table}`);
}

export async function getTableRows(table: string, limit = 25, offset = 0, orderBy?: string, sort?: string) {
  const params = new URLSearchParams({
    limit: limit.toString(),
    offset: offset.toString(),
  });
  if (orderBy) params.set("order_by", orderBy);
  if (sort) params.set("sort", sort);
  return fetchWithAuth(`/api/admin/inspector/tables/${table}/rows?${params.toString()}`);
}

export async function insertTableRow(table: string, data: Record<string, any>) {
  return fetchWithAuth(`/api/admin/inspector/tables/${table}/rows`, {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function deleteTableRow(table: string, pkColumn: string, pkValue: string) {
  return fetchWithAuth(`/api/admin/inspector/tables/${table}/rows?pk_column=${encodeURIComponent(pkColumn)}&pk_value=${encodeURIComponent(pkValue)}`, {
    method: "DELETE",
  });
}

// API Keys
export async function listAPIKeys() {
  return fetchWithAuth("/api/admin/api-keys");
}

export async function createAPIKey(data: { name: string; owner_id?: string; scopes: string[]; expires_in?: string }) {
  return fetchWithAuth("/api/admin/api-keys", {
    method: "POST",
    body: JSON.stringify(data),
  });
}

export async function revokeAPIKey(id: string) {
  return fetchWithAuth(`/api/admin/api-keys/${id}`, {
    method: "DELETE",
  });
}

// Projects
export async function listProjects(limit = 20, offset = 0) {
  return fetchWithAuth(`/api/projects?limit=${limit}&offset=${offset}`);
}

// MCP Info & Tool Call
export async function getMCPInfo() {
  return fetchWithAuth("/mcp");
}

export async function callMCPTool(name: string, args: Record<string, any>) {
  return fetchWithAuth("/mcp", {
    method: "POST",
    body: JSON.stringify({
      jsonrpc: "2.0",
      id: Date.now(),
      method: "tools/call",
      params: {
        name,
        arguments: args,
      },
    }),
  });
}
