"use client";

import React, { useState, useEffect } from "react";
import {
  Database,
  Key,
  FolderGit2,
  Cpu,
  Server,
  Activity,
  Layers,
  Search,
  Plus,
  Trash2,
  RefreshCw,
  Copy,
  Check,
  Code2,
  Terminal,
  Shield,
  Eye,
  ExternalLink,
  ChevronRight,
  HardDrive
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import * as api from "@/lib/api";

type TabType = "overview" | "database" | "apikeys" | "projects" | "mcp" | "system";
type DbSubTab = "tables" | "schema" | "data";

export default function Dashboard() {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [dbSubTab, setDbSubTab] = useState<DbSubTab>("tables");

  // System Stats
  const [stats, setStats] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Database Inspector state
  const [tables, setTables] = useState<any[]>([]);
  const [selectedTable, setSelectedTable] = useState<string>("");
  const [tableDetail, setTableDetail] = useState<any>(null);
  const [rowsResult, setRowsResult] = useState<any>(null);
  const [dataLimit] = useState(20);
  const [dataOffset, setDataOffset] = useState(0);

  // API Keys state
  const [apiKeys, setApiKeys] = useState<any[]>([]);
  const [newKeyName, setNewKeyName] = useState("");
  const [newKeyScopes, setNewKeyScopes] = useState<string[]>([
    "projects:read",
    "projects:write",
  ]);
  const [createdPlaintextKey, setCreatedPlaintextKey] = useState<string | null>(null);
  const [isCopied, setIsCopied] = useState(false);

  // Projects state
  const [projectsList, setProjectsList] = useState<any[]>([]);
  const [selectedProject, setSelectedProject] = useState<any>(null);

  // MCP state
  const [mcpInfo, setMcpInfo] = useState<any>(null);
  const [selectedMcpTool, setSelectedMcpTool] = useState<string>("database_stats");
  const [mcpArgsInput, setMcpArgsInput] = useState<string>("{}");
  const [mcpExecutionResult, setMcpExecutionResult] = useState<any>(null);
  const [mcpLoading, setMcpLoading] = useState(false);

  // Auth token state
  const [adminToken, setAdminToken] = useState<string>("cb_admin_secret_key_change_in_prod");
  const [showTokenModal, setShowTokenModal] = useState(false);

  useEffect(() => {
    if (typeof window !== "undefined") {
      const stored = localStorage.getItem("cobalt_admin_key");
      if (stored) setAdminToken(stored);
    }
  }, []);

  const saveToken = (token: string) => {
    setAdminToken(token);
    if (typeof window !== "undefined") {
      localStorage.setItem("cobalt_admin_key", token);
    }
    setShowTokenModal(false);
    refreshData();
  };

  // Load initial stats
  const refreshData = async () => {
    setLoading(true);
    setError(null);
    try {
      const s = await api.getSystemStats();
      setStats(s);

      const tbls = await api.listTables();
      setTables(tbls?.data || []);
      if (tbls?.data?.length && !selectedTable) {
        setSelectedTable(tbls.data[0].name);
      }
    } catch (err: any) {
      setError(err.message || "Failed to load dashboard data");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refreshData();
  }, []);

  // Fetch table schema or data when selectedTable or dbSubTab changes
  useEffect(() => {
    if (activeTab === "database" && selectedTable) {
      if (dbSubTab === "schema") {
        api.describeTable(selectedTable)
          .then((res) => setTableDetail(res?.data || null))
          .catch((err) => setError(err.message));
      } else if (dbSubTab === "data") {
        api.getTableRows(selectedTable, dataLimit, dataOffset)
          .then((res) => setRowsResult(res || null))
          .catch((err) => setError(err.message));
      }
    }
  }, [activeTab, dbSubTab, selectedTable, dataOffset]);

  // Load API keys
  useEffect(() => {
    if (activeTab === "apikeys") {
      api.listAPIKeys()
        .then((res) => setApiKeys(res?.data || []))
        .catch((err) => setError(err.message));
    }
  }, [activeTab]);

  // Load Projects
  useEffect(() => {
    if (activeTab === "projects") {
      api.listProjects(50, 0)
        .then((res) => setProjectsList(res?.data || []))
        .catch((err) => setError(err.message));
    }
  }, [activeTab]);

  // Load MCP Info
  useEffect(() => {
    if (activeTab === "mcp") {
      api.getMCPInfo()
        .then((res) => setMcpInfo(res))
        .catch((err) => setError(err.message));
    }
  }, [activeTab]);

  const handleCreateAPIKey = async () => {
    if (!newKeyName.trim()) return;
    try {
      const res = await api.createAPIKey({
        name: newKeyName.trim(),
        scopes: newKeyScopes,
      });
      setCreatedPlaintextKey(res.data.plaintext_key);
      setNewKeyName("");
      const updated = await api.listAPIKeys();
      setApiKeys(updated?.data || []);
    } catch (err: any) {
      alert("Error creating API key: " + err.message);
    }
  };

  const handleRevokeKey = async (id: string) => {
    if (!confirm("Are you sure you want to revoke this API key? This action is immediate and cannot be undone.")) return;
    try {
      await api.revokeAPIKey(id);
      const updated = await api.listAPIKeys();
      setApiKeys(updated?.data || []);
    } catch (err: any) {
      alert("Error revoking key: " + err.message);
    }
  };

  const handleExecuteMCP = async () => {
    setMcpLoading(true);
    setMcpExecutionResult(null);
    try {
      let parsedArgs = {};
      if (mcpArgsInput.trim()) {
        parsedArgs = JSON.parse(mcpArgsInput);
      }
      const res = await api.callMCPTool(selectedMcpTool, parsedArgs);
      setMcpExecutionResult(res);
    } catch (err: any) {
      setMcpExecutionResult({ error: err.message });
    } finally {
      setMcpLoading(false);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setIsCopied(true);
    setTimeout(() => setIsCopied(false), 2000);
  };

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100 flex">
      {/* Sidebar Navigation */}
      <aside className="w-64 border-r border-zinc-850 bg-zinc-950/80 p-5 flex flex-col justify-between">
        <div>
          {/* Cobalt Brand */}
          <div className="flex items-center gap-3 px-2 py-3 mb-6">
            <div className="h-8 w-8 rounded-lg bg-blue-600 flex items-center justify-center font-bold text-white shadow-md shadow-blue-500/20">
              C
            </div>
            <div>
              <div className="font-bold tracking-tight text-white flex items-center gap-1.5 text-base">
                Cobalt
                <Badge variant="secondary" className="text-[10px] px-1.5 py-0">v1.0</Badge>
              </div>
              <div className="text-xs text-zinc-400">ZenCompiler Engine</div>
            </div>
          </div>

          {/* Navigation Links */}
          <nav className="space-y-1 text-sm font-medium">
            <button
              onClick={() => setActiveTab("overview")}
              className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                activeTab === "overview"
                  ? "bg-zinc-800 text-white font-semibold"
                  : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900"
              }`}
            >
              <Activity className="h-4 w-4 text-blue-400" />
              Overview
            </button>

            <button
              onClick={() => setActiveTab("database")}
              className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                activeTab === "database"
                  ? "bg-zinc-800 text-white font-semibold"
                  : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900"
              }`}
            >
              <Database className="h-4 w-4 text-emerald-400" />
              Database
            </button>

            <button
              onClick={() => setActiveTab("apikeys")}
              className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                activeTab === "apikeys"
                  ? "bg-zinc-800 text-white font-semibold"
                  : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900"
              }`}
            >
              <Key className="h-4 w-4 text-amber-400" />
              API Keys
            </button>

            <button
              onClick={() => setActiveTab("projects")}
              className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                activeTab === "projects"
                  ? "bg-zinc-800 text-white font-semibold"
                  : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900"
              }`}
            >
              <FolderGit2 className="h-4 w-4 text-indigo-400" />
              Projects
            </button>

            <button
              onClick={() => setActiveTab("mcp")}
              className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                activeTab === "mcp"
                  ? "bg-zinc-800 text-white font-semibold"
                  : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900"
              }`}
            >
              <Cpu className="h-4 w-4 text-purple-400" />
              MCP Server
            </button>

            <button
              onClick={() => setActiveTab("system")}
              className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${
                activeTab === "system"
                  ? "bg-zinc-800 text-white font-semibold"
                  : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900"
              }`}
            >
              <Server className="h-4 w-4 text-cyan-400" />
              System
            </button>
          </nav>
        </div>

        {/* Auth settings & Status */}
        <div className="pt-4 border-t border-zinc-850">
          <button
            onClick={() => setShowTokenModal(true)}
            className="w-full flex items-center justify-between px-3 py-2 rounded-lg bg-zinc-900 hover:bg-zinc-850 text-xs text-zinc-300 transition-colors"
          >
            <div className="flex items-center gap-2 truncate">
              <Shield className="h-3.5 w-3.5 text-blue-400" />
              <span className="truncate">Auth: {adminToken ? "Configured" : "None"}</span>
            </div>
            <span className="text-[10px] text-zinc-500 underline">Change</span>
          </button>
        </div>
      </aside>

      {/* Main Content Area */}
      <main className="flex-1 flex flex-col min-w-0 overflow-y-auto">
        {/* Top Header */}
        <header className="h-16 border-b border-zinc-850 px-8 flex items-center justify-between bg-zinc-950/60 backdrop-blur-md sticky top-0 z-10">
          <div className="flex items-center gap-3">
            <h1 className="text-lg font-semibold capitalize text-zinc-100">{activeTab}</h1>
            {loading && <RefreshCw className="h-4 w-4 animate-spin text-zinc-400" />}
          </div>

          <div className="flex items-center gap-3">
            <Button variant="outline" size="sm" onClick={refreshData} disabled={loading} className="gap-2">
              <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
              Sync
            </Button>
          </div>
        </header>

        {/* Content Container */}
        <div className="p-8 max-w-7xl w-full mx-auto space-y-6">
          {error && (
            <div className="p-4 rounded-lg bg-red-950/40 border border-red-800/60 text-red-300 text-sm flex items-center justify-between">
              <span>{error}</span>
              <button onClick={() => setError(null)} className="text-red-400 hover:text-red-200">×</button>
            </div>
          )}

          {/* OVERVIEW TAB */}
          {activeTab === "overview" && (
            <div className="space-y-6">
              {/* Metric Cards */}
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <Card>
                  <CardHeader className="flex flex-row items-center justify-between pb-2">
                    <CardTitle className="text-xs font-medium text-zinc-400 uppercase tracking-wider">Database Size</CardTitle>
                    <HardDrive className="h-4 w-4 text-emerald-400" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{stats?.database?.total_size || "—"}</div>
                    <p className="text-xs text-zinc-500 mt-1">{stats?.database?.database_name || "PostgreSQL"}</p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between pb-2">
                    <CardTitle className="text-xs font-medium text-zinc-400 uppercase tracking-wider">Total Tables</CardTitle>
                    <Database className="h-4 w-4 text-blue-400" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{tables?.length ?? stats?.database?.total_tables ?? 0}</div>
                    <p className="text-xs text-zinc-500 mt-1">{stats?.database?.total_indexes ?? 0} indexes</p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between pb-2">
                    <CardTitle className="text-xs font-medium text-zinc-400 uppercase tracking-wider">Active Conns</CardTitle>
                    <Activity className="h-4 w-4 text-amber-400" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{stats?.database?.active_connections ?? 1}</div>
                    <p className="text-xs text-zinc-500 mt-1">Pool: {stats?.pool?.acquired_conns ?? 0}/{stats?.pool?.max_conns ?? 25}</p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader className="flex flex-row items-center justify-between pb-2">
                    <CardTitle className="text-xs font-medium text-zinc-400 uppercase tracking-wider">System Memory</CardTitle>
                    <Cpu className="h-4 w-4 text-purple-400" />
                  </CardHeader>
                  <CardContent>
                    <div className="text-2xl font-bold">{stats?.system?.alloc_mb ?? 0} MB</div>
                    <p className="text-xs text-zinc-500 mt-1">{stats?.system?.goroutines ?? 0} goroutines</p>
                  </CardContent>
                </Card>
              </div>

              {/* Architecture Map & Quick Links */}
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <Card>
                  <CardHeader>
                    <CardTitle>Architecture Status</CardTitle>
                    <CardDescription>ZenCompiler decoupled infrastructure</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4 text-sm">
                    <div className="flex items-center justify-between p-3 rounded-lg bg-zinc-900 border border-zinc-800">
                      <div className="flex items-center gap-3">
                        <div className="h-2.5 w-2.5 rounded-full bg-emerald-500"></div>
                        <div>
                          <div className="font-semibold text-zinc-200">Go Backend (Cobalt)</div>
                          <div className="text-xs text-zinc-400">REST API, MCP, Webhooks &amp; Auth</div>
                        </div>
                      </div>
                      <Badge variant="success">Online</Badge>
                    </div>

                    <div className="flex items-center justify-between p-3 rounded-lg bg-zinc-900 border border-zinc-800">
                      <div className="flex items-center gap-3">
                        <div className="h-2.5 w-2.5 rounded-full bg-emerald-500"></div>
                        <div>
                          <div className="font-semibold text-zinc-200">PostgreSQL Database</div>
                          <div className="text-xs text-zinc-400">Managed pgxpool with persistent storage</div>
                        </div>
                      </div>
                      <Badge variant="success">Connected</Badge>
                    </div>

                    <div className="flex items-center justify-between p-3 rounded-lg bg-zinc-900 border border-zinc-800">
                      <div className="flex items-center gap-3">
                        <div className="h-2.5 w-2.5 rounded-full bg-blue-500"></div>
                        <div>
                          <div className="font-semibold text-zinc-200">MCP Protocol Server</div>
                          <div className="text-xs text-zinc-400">JSON-RPC 2.0 with scope enforcement</div>
                        </div>
                      </div>
                      <Badge variant="default">Active</Badge>
                    </div>

                    <div className="flex items-center justify-between p-3 rounded-lg bg-zinc-900 border border-zinc-800">
                      <div className="flex items-center gap-3">
                        <div className="h-2.5 w-2.5 rounded-full bg-amber-500"></div>
                        <div>
                          <div className="font-semibold text-zinc-200">Worker VPS (Separate)</div>
                          <div className="text-xs text-zinc-400">Docker Code Execution Sandbox</div>
                        </div>
                      </div>
                      <Badge variant="secondary">External</Badge>
                    </div>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle>Recent Tables</CardTitle>
                    <CardDescription>Managed schemas in PostgreSQL</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="divide-y divide-zinc-800">
                      {tables.map((t) => (
                        <div key={t.name} className="py-2.5 flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <Layers className="h-4 w-4 text-zinc-400" />
                            <span className="font-mono text-sm text-zinc-200">{t.name}</span>
                          </div>
                          <div className="flex items-center gap-3 text-xs text-zinc-400">
                            <span>{t.estimated_rows} rows</span>
                            <Badge variant="outline">{t.pretty_size}</Badge>
                          </div>
                        </div>
                      ))}
                      {tables.length === 0 && (
                        <div className="py-8 text-center text-zinc-500 text-sm">No tables detected.</div>
                      )}
                    </div>
                  </CardContent>
                </Card>
              </div>
            </div>
          )}

          {/* DATABASE TAB */}
          {activeTab === "database" && (
            <div className="space-y-6">
              {/* Database Subnav */}
              <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
                <div className="flex gap-2">
                  <Button
                    variant={dbSubTab === "tables" ? "default" : "ghost"}
                    size="sm"
                    onClick={() => setDbSubTab("tables")}
                  >
                    Tables
                  </Button>
                  <Button
                    variant={dbSubTab === "schema" ? "default" : "ghost"}
                    size="sm"
                    onClick={() => setDbSubTab("schema")}
                  >
                    Schema
                  </Button>
                  <Button
                    variant={dbSubTab === "data" ? "default" : "ghost"}
                    size="sm"
                    onClick={() => setDbSubTab("data")}
                  >
                    Data Browser
                  </Button>
                </div>

                {/* Table Picker */}
                <div className="flex items-center gap-2">
                  <span className="text-xs text-zinc-400">Selected Table:</span>
                  <select
                    value={selectedTable}
                    onChange={(e) => setSelectedTable(e.target.value)}
                    className="bg-zinc-900 border border-zinc-700 text-zinc-100 text-xs rounded-md px-2.5 py-1.5 font-mono focus:outline-none focus:ring-1 focus:ring-blue-500"
                  >
                    {tables.map((t) => (
                      <option key={t.name} value={t.name}>
                        {t.name}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              {/* Subtab: TABLES */}
              {dbSubTab === "tables" && (
                <Card>
                  <CardHeader>
                    <CardTitle>Database Tables</CardTitle>
                    <CardDescription>Overview of tables, estimated row counts, and storage</CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="overflow-x-auto">
                      <table className="w-full text-left text-sm text-zinc-300">
                        <thead className="border-b border-zinc-800 text-xs font-semibold text-zinc-400 uppercase">
                          <tr>
                            <th className="py-3 px-4">Table Name</th>
                            <th className="py-3 px-4">Estimated Rows</th>
                            <th className="py-3 px-4">Total Bytes</th>
                            <th className="py-3 px-4">Size</th>
                            <th className="py-3 px-4 text-right">Actions</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-zinc-800/60 font-mono text-xs">
                          {tables.map((t) => (
                            <tr key={t.name} className="hover:bg-zinc-900/40 transition-colors">
                              <td className="py-3 px-4 font-semibold text-zinc-100">{t.name}</td>
                              <td className="py-3 px-4">{t.estimated_rows}</td>
                              <td className="py-3 px-4 text-zinc-400">{t.total_bytes}</td>
                              <td className="py-3 px-4">
                                <Badge variant="secondary">{t.pretty_size}</Badge>
                              </td>
                              <td className="py-3 px-4 text-right space-x-2 font-sans">
                                <Button
                                  variant="outline"
                                  size="sm"
                                  onClick={() => {
                                    setSelectedTable(t.name);
                                    setDbSubTab("schema");
                                  }}
                                >
                                  Schema
                                </Button>
                                <Button
                                  variant="default"
                                  size="sm"
                                  onClick={() => {
                                    setSelectedTable(t.name);
                                    setDbSubTab("data");
                                  }}
                                >
                                  Browse Data
                                </Button>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* Subtab: SCHEMA */}
              {dbSubTab === "schema" && (
                <div className="space-y-6">
                  <Card>
                    <CardHeader>
                      <CardTitle className="font-mono">{selectedTable} Columns</CardTitle>
                      <CardDescription>Field types, nullability, and keys</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="overflow-x-auto">
                        <table className="w-full text-left text-sm">
                          <thead className="border-b border-zinc-800 text-xs font-semibold text-zinc-400 uppercase">
                            <tr>
                              <th className="py-2.5 px-4">Column</th>
                              <th className="py-2.5 px-4">Type</th>
                              <th className="py-2.5 px-4">Nullable</th>
                              <th className="py-2.5 px-4">Default</th>
                              <th className="py-2.5 px-4">Key</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-zinc-800/60 font-mono text-xs">
                            {tableDetail?.columns?.map((c: any) => (
                              <tr key={c.name}>
                                <td className="py-2.5 px-4 font-semibold text-zinc-100">{c.name}</td>
                                <td className="py-2.5 px-4 text-blue-400">{c.data_type}</td>
                                <td className="py-2.5 px-4 text-zinc-400">{c.is_nullable ? "YES" : "NO"}</td>
                                <td className="py-2.5 px-4 text-zinc-500 truncate max-w-xs">{c.column_default || "—"}</td>
                                <td className="py-2.5 px-4 font-sans">
                                  {c.is_primary_key && <Badge variant="default">PK</Badge>}
                                </td>
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    </CardContent>
                  </Card>

                  {tableDetail?.indexes?.length > 0 && (
                    <Card>
                      <CardHeader>
                        <CardTitle>Indexes</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="space-y-2">
                          {tableDetail.indexes.map((idx: any) => (
                            <div key={idx.name} className="p-3 rounded-lg bg-zinc-900 border border-zinc-800 flex items-center justify-between text-xs font-mono">
                              <div>
                                <span className="font-semibold text-zinc-200">{idx.name}</span>
                                <div className="text-zinc-500 text-[11px] mt-0.5">{idx.columns}</div>
                              </div>
                              <div className="flex gap-2 font-sans">
                                {idx.is_unique && <Badge variant="secondary">Unique</Badge>}
                                {idx.is_primary && <Badge variant="default">Primary</Badge>}
                              </div>
                            </div>
                          ))}
                        </div>
                      </CardContent>
                    </Card>
                  )}
                </div>
              )}

              {/* Subtab: DATA BROWSER */}
              {dbSubTab === "data" && (
                <Card>
                  <CardHeader className="flex flex-row items-center justify-between">
                    <div>
                      <CardTitle className="font-mono">{selectedTable} Rows</CardTitle>
                      <CardDescription>
                        Total: {rowsResult?.total || 0} records
                      </CardDescription>
                    </div>
                  </CardHeader>
                  <CardContent>
                    {rowsResult?.rows && rowsResult.rows.length > 0 ? (
                      <div className="overflow-x-auto">
                        <table className="w-full text-left text-xs font-mono">
                          <thead className="border-b border-zinc-800 text-zinc-400 uppercase">
                            <tr>
                              {Object.keys(rowsResult.rows[0]).map((key) => (
                                <th key={key} className="py-2.5 px-3 whitespace-nowrap">{key}</th>
                              ))}
                            </tr>
                          </thead>
                          <tbody className="divide-y divide-zinc-800/60">
                            {rowsResult.rows.map((row: any, i: number) => (
                              <tr key={i} className="hover:bg-zinc-900/60 transition-colors">
                                {Object.keys(rowsResult.rows[0]).map((key) => {
                                  const val = row[key];
                                  const displayVal = typeof val === "object" ? JSON.stringify(val) : String(val ?? "");
                                  return (
                                    <td key={key} className="py-2.5 px-3 max-w-xs truncate text-zinc-300">
                                      {displayVal}
                                    </td>
                                  );
                                })}
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    ) : (
                      <div className="py-12 text-center text-zinc-500 text-sm">No rows found in {selectedTable}.</div>
                    )}

                    {/* Pagination */}
                    <div className="flex items-center justify-between pt-4 border-t border-zinc-800 text-xs text-zinc-400">
                      <div>
                        Offset: {dataOffset} | Showing up to {dataLimit}
                      </div>
                      <div className="flex gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={dataOffset === 0}
                          onClick={() => setDataOffset(Math.max(0, dataOffset - dataLimit))}
                        >
                          Previous
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          disabled={!rowsResult || dataOffset + dataLimit >= rowsResult.total}
                          onClick={() => setDataOffset(dataOffset + dataLimit)}
                        >
                          Next
                        </Button>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              )}
            </div>
          )}

          {/* API KEYS TAB */}
          {activeTab === "apikeys" && (
            <div className="space-y-6">
              {/* Plaintext Key Copy Banner */}
              {createdPlaintextKey && (
                <div className="p-4 rounded-xl bg-blue-950/60 border border-blue-800 text-blue-200 space-y-2">
                  <div className="font-semibold flex items-center justify-between">
                    <span>New API Key Created! Save this key now:</span>
                    <Button size="sm" variant="default" onClick={() => copyToClipboard(createdPlaintextKey)} className="gap-1.5">
                      {isCopied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                      {isCopied ? "Copied" : "Copy Key"}
                    </Button>
                  </div>
                  <div className="p-2.5 rounded bg-zinc-950 font-mono text-sm text-emerald-400 break-all border border-blue-900">
                    {createdPlaintextKey}
                  </div>
                  <p className="text-xs text-blue-300">
                    Warning: For security reasons, this plaintext key is never stored and will never be shown again.
                  </p>
                </div>
              )}

              {/* Create API Key Box */}
              <Card>
                <CardHeader>
                  <CardTitle>Create API Key</CardTitle>
                  <CardDescription>Generate keys for frontend apps, workers, or AI agents</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="flex gap-3">
                    <Input
                      placeholder="Key Name (e.g. Next.js Production, Claude Agent)"
                      value={newKeyName}
                      onChange={(e) => setNewKeyName(e.target.value)}
                      className="flex-1"
                    />
                    <Button onClick={handleCreateAPIKey} className="gap-1.5">
                      <Plus className="h-4 w-4" />
                      Generate Key
                    </Button>
                  </div>

                  <div>
                    <span className="text-xs text-zinc-400 font-medium block mb-2">Permission Scopes:</span>
                    <div className="flex flex-wrap gap-2 text-xs">
                      {[
                        "database:read",
                        "database:write",
                        "projects:read",
                        "projects:write",
                        "admin",
                      ].map((sc) => {
                        const checked = newKeyScopes.includes(sc);
                        return (
                          <button
                            key={sc}
                            type="button"
                            onClick={() => {
                              if (checked) {
                                setNewKeyScopes(newKeyScopes.filter((s) => s !== sc));
                              } else {
                                setNewKeyScopes([...newKeyScopes, sc]);
                              }
                            }}
                            className={`px-3 py-1.5 rounded-md border font-mono transition-colors ${
                              checked
                                ? "bg-blue-600/30 border-blue-500 text-blue-300"
                                : "bg-zinc-900 border-zinc-800 text-zinc-500 hover:border-zinc-700"
                            }`}
                          >
                            {sc}
                          </button>
                        );
                      })}
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Active API Keys List */}
              <Card>
                <CardHeader>
                  <CardTitle>Active &amp; Revoked API Keys</CardTitle>
                  <CardDescription>Keys stored as cryptographic SHA-256 hashes</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm">
                      <thead className="border-b border-zinc-850 text-xs font-semibold text-zinc-400 uppercase">
                        <tr>
                          <th className="py-3 px-4">Name</th>
                          <th className="py-3 px-4">Prefix</th>
                          <th className="py-3 px-4">Scopes</th>
                          <th className="py-3 px-4">Last Used</th>
                          <th className="py-3 px-4">Status</th>
                          <th className="py-3 px-4 text-right">Action</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-zinc-850/60 font-mono text-xs">
                        {apiKeys.map((k) => (
                          <tr key={k.id} className="hover:bg-zinc-900/40 transition-colors">
                            <td className="py-3 px-4 font-sans font-medium text-zinc-200">{k.name}</td>
                            <td className="py-3 px-4 text-zinc-400">{k.key_prefix}</td>
                            <td className="py-3 px-4 font-sans">
                              <div className="flex flex-wrap gap-1">
                                {k.scopes?.map((s: string) => (
                                  <Badge key={s} variant="secondary" className="text-[10px] py-0">{s}</Badge>
                                ))}
                              </div>
                            </td>
                            <td className="py-3 px-4 text-zinc-500 font-sans">
                              {k.last_used_at ? new Date(k.last_used_at).toLocaleDateString() : "Never"}
                            </td>
                            <td className="py-3 px-4 font-sans">
                              {k.revoked_at ? (
                                <Badge variant="destructive">Revoked</Badge>
                              ) : (
                                <Badge variant="success">Active</Badge>
                              )}
                            </td>
                            <td className="py-3 px-4 text-right font-sans">
                              {!k.revoked_at && (
                                <Button
                                  variant="destructive"
                                  size="sm"
                                  onClick={() => handleRevokeKey(k.id)}
                                >
                                  Revoke
                                </Button>
                              )}
                            </td>
                          </tr>
                        ))}
                        {apiKeys.length === 0 && (
                          <tr>
                            <td colSpan={6} className="py-8 text-center text-zinc-500 font-sans">
                              No API keys found.
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}

          {/* PROJECTS TAB */}
          {activeTab === "projects" && (
            <div className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle>ZenCompiler Projects</CardTitle>
                  <CardDescription>Code repositories and metadata owned by users</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    {projectsList.map((p) => (
                      <div
                        key={p.id}
                        onClick={() => setSelectedProject(p)}
                        className={`p-4 rounded-xl border transition-all cursor-pointer ${
                          selectedProject?.id === p.id
                            ? "bg-zinc-900 border-blue-500 shadow-md shadow-blue-500/10"
                            : "bg-zinc-950/70 border-zinc-800 hover:border-zinc-700"
                        }`}
                      >
                        <div className="flex items-center justify-between mb-2">
                          <h4 className="font-semibold text-zinc-100">{p.name}</h4>
                          <Badge variant="secondary">{p.language}</Badge>
                        </div>
                        <p className="text-xs text-zinc-400 mb-3 line-clamp-2">{p.description || "No description provided."}</p>
                        <div className="flex items-center justify-between text-[11px] text-zinc-500 font-mono">
                          <span>Owner: {p.owner_id}</span>
                          <span>{new Date(p.created_at).toLocaleDateString()}</span>
                        </div>
                      </div>
                    ))}
                    {projectsList.length === 0 && (
                      <div className="col-span-2 py-12 text-center text-zinc-500 text-sm">
                        No projects found. Create one via <code className="text-zinc-300">POST /api/projects</code>.
                      </div>
                    )}
                  </div>

                  {/* Project Code & Metadata Drawer */}
                  {selectedProject && (
                    <div className="mt-6 p-4 rounded-xl bg-zinc-900/90 border border-zinc-800 space-y-4">
                      <div className="flex items-center justify-between">
                        <div className="font-mono text-sm font-semibold text-zinc-200">
                          {selectedProject.name} <span className="text-zinc-500">({selectedProject.id})</span>
                        </div>
                        <Button variant="ghost" size="sm" onClick={() => setSelectedProject(null)}>Close</Button>
                      </div>

                      {/* Code Content */}
                      <div>
                        <div className="text-xs text-zinc-400 font-semibold mb-1">Code Content:</div>
                        <pre className="p-3 rounded-lg bg-zinc-950 border border-zinc-850 font-mono text-xs text-zinc-300 overflow-x-auto max-h-60">
                          {selectedProject.code_content || "// Empty code content"}
                        </pre>
                      </div>

                      {/* Metadata */}
                      <div>
                        <div className="text-xs text-zinc-400 font-semibold mb-1">Metadata JSON:</div>
                        <pre className="p-3 rounded-lg bg-zinc-950 border border-zinc-850 font-mono text-xs text-zinc-400 overflow-x-auto">
                          {JSON.stringify(selectedProject.metadata, null, 2)}
                        </pre>
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>
          )}

          {/* MCP TAB */}
          {activeTab === "mcp" && (
            <div className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                {/* Registered MCP Tools */}
                <Card>
                  <CardHeader>
                    <CardTitle>Registered MCP Tools</CardTitle>
                    <CardDescription>Tools exposed to AI agents via JSON-RPC 2.0</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    {mcpInfo?.tools?.map((t: any) => (
                      <div
                        key={t.name}
                        onClick={() => setSelectedMcpTool(t.name)}
                        className={`p-3 rounded-lg border transition-all cursor-pointer ${
                          selectedMcpTool === t.name
                            ? "bg-zinc-900 border-purple-500"
                            : "bg-zinc-950 border-zinc-800 hover:border-zinc-700"
                        }`}
                      >
                        <div className="flex items-center justify-between mb-1">
                          <span className="font-mono text-sm font-semibold text-purple-400">{t.name}</span>
                          <Badge variant="outline">{t.required_scope}</Badge>
                        </div>
                        <p className="text-xs text-zinc-400">{t.description}</p>
                      </div>
                    ))}
                  </CardContent>
                </Card>

                {/* MCP Tool Runner Playground */}
                <Card>
                  <CardHeader>
                    <CardTitle>MCP Tool Playground</CardTitle>
                    <CardDescription>Execute tool call directly against <code className="text-zinc-300">/mcp</code></CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div>
                      <label className="text-xs text-zinc-400 font-medium block mb-1">Selected Tool:</label>
                      <Input value={selectedMcpTool} readOnly className="font-mono text-xs bg-zinc-950" />
                    </div>

                    <div>
                      <label className="text-xs text-zinc-400 font-medium block mb-1">Tool Arguments (JSON):</label>
                      <textarea
                        value={mcpArgsInput}
                        onChange={(e) => setMcpArgsInput(e.target.value)}
                        rows={4}
                        className="w-full rounded-md border border-zinc-700 bg-zinc-950 p-2.5 font-mono text-xs text-zinc-200 focus:outline-none focus:ring-1 focus:ring-purple-500"
                        placeholder="{}"
                      />
                    </div>

                    <Button
                      onClick={handleExecuteMCP}
                      disabled={mcpLoading}
                      className="w-full bg-purple-600 hover:bg-purple-500 gap-2"
                    >
                      <Terminal className="h-4 w-4" />
                      {mcpLoading ? "Calling Tool..." : "Execute MCP Tool"}
                    </Button>

                    {mcpExecutionResult && (
                      <div className="mt-4">
                        <div className="text-xs text-zinc-400 font-medium mb-1">Response:</div>
                        <pre className="p-3 rounded-lg bg-zinc-950 border border-zinc-800 font-mono text-xs text-zinc-300 overflow-x-auto max-h-72">
                          {JSON.stringify(mcpExecutionResult, null, 2)}
                        </pre>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </div>
            </div>
          )}

          {/* SYSTEM TAB */}
          {activeTab === "system" && (
            <div className="space-y-6">
              <Card>
                <CardHeader>
                  <CardTitle>Cobalt Platform Runtime</CardTitle>
                  <CardDescription>Server status, garbage collector, and database connection pool</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
                    <div className="p-4 rounded-lg bg-zinc-900 border border-zinc-800">
                      <div className="text-xs text-zinc-500">Go Version</div>
                      <div className="text-base font-semibold text-zinc-200 font-mono mt-1">{stats?.system?.go_version || "—"}</div>
                    </div>
                    <div className="p-4 rounded-lg bg-zinc-900 border border-zinc-800">
                      <div className="text-xs text-zinc-500">Uptime</div>
                      <div className="text-base font-semibold text-zinc-200 font-mono mt-1">{stats?.system?.uptime || "—"}</div>
                    </div>
                    <div className="p-4 rounded-lg bg-zinc-900 border border-zinc-800">
                      <div className="text-xs text-zinc-500">Heap Alloc</div>
                      <div className="text-base font-semibold text-zinc-200 font-mono mt-1">{stats?.system?.alloc_mb || 0} MB</div>
                    </div>
                    <div className="p-4 rounded-lg bg-zinc-900 border border-zinc-800">
                      <div className="text-xs text-zinc-500">GC Cycles</div>
                      <div className="text-base font-semibold text-zinc-200 font-mono mt-1">{stats?.system?.num_gc || 0}</div>
                    </div>
                  </div>

                  <div className="p-4 rounded-lg bg-zinc-950 border border-zinc-850 space-y-2">
                    <div className="font-semibold text-xs text-zinc-300">Raw Diagnostic Metrics:</div>
                    <pre className="font-mono text-xs text-zinc-400 overflow-x-auto">
                      {JSON.stringify(stats, null, 2)}
                    </pre>
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
        </div>
      </main>

      {/* Auth Token Configuration Modal */}
      {showTokenModal && (
        <div className="fixed inset-0 bg-black/70 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-zinc-900 border border-zinc-800 rounded-xl max-w-md w-full p-6 space-y-4">
            <h3 className="font-semibold text-lg text-zinc-100">Configure Admin Token</h3>
            <p className="text-xs text-zinc-400">
              Enter your Cobalt Admin API key to authorize database inspection, API key generation, and system operations.
            </p>
            <Input
              type="password"
              value={adminToken}
              onChange={(e) => setAdminToken(e.target.value)}
              placeholder="cb_admin_..."
            />
            <div className="flex justify-end gap-2 pt-2">
              <Button variant="ghost" size="sm" onClick={() => setShowTokenModal(false)}>Cancel</Button>
              <Button size="sm" onClick={() => saveToken(adminToken)}>Save Token</Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
