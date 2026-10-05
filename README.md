# Cobalt ⚡

> **Lightweight, production-ready backend platform for ZenCompiler built with Go + PostgreSQL.**

Cobalt replaces external heavy backend dependencies (such as Supabase) with a single, ultra-fast Go binary designed to run smoothly on small VPS instances (such as a 6 vCPU / 8 GB RAM VPS) with minimal resource usage (< 20MB idle RAM).

---

## 🏛️ Architecture

```
                    ZenCompiler
                        │
                        ▼
                 Next.js Frontend
                        │
                   HTTPS / REST
                        │
                        ▼
              ┌───────────────────┐
              │    Go Backend     │
              │     (Cobalt)      │
              │                   │
              │ REST API          │
              │ Business Logic    │
              │ PostgreSQL Pool   │
              │ API Keys (SHA256) │
              │ First-Class MCP   │
              │ Webhooks (HMAC)   │
              │ In-Process Jobs   │
              └─────────┬─────────┘
                        │
                        ▼
                   PostgreSQL
Separate system:
              Go Backend
                   │
                   ▼
              Worker VPS (Docker / Code execution)
```

> [!NOTE]
> Cobalt handles data persistence, business logic, authorization, database inspection, API keys, MCP tooling, and integrations. Code execution remains isolated in the external Worker VPS sandbox.

---

## ✨ Features

- **Blazing Fast Go Service**: Built using idiomatic Go, standard library HTTP stack with Chi router, and `pgx/v5` connection pooling.
- **Embedded Migrations**: Auto-migrates database schemas on startup via Go's `embed.FS` without requiring external migration CLI binaries.
- **Pluggable Auth Abstraction**: Validates Bearer tokens and API keys; decouple identity providers (Supabase, Clerk, Auth0, or custom JWT/headers) without touching business logic.
- **Projects CRUD Engine**: User-isolated project workspace management, slug generation, parameterization, and authorization.
- **Admin Database Inspector**:
  - List tables with row estimates and storage sizes
  - Inspect table schema, data types, nullability, defaults, primary keys, and indexes
  - Browse live table rows with pagination and custom sorting
  - Safe parameterized row insertion, updates, and deletion
  - High-level database metrics (cache hit ratio, active connections, total tables, size)
- **First-Class MCP Server**: Model Context Protocol JSON-RPC 2.0 server (`POST /mcp` and `GET /mcp`) with strict scope enforcement (`database:read`, `database:write`, `projects:read`, `projects:write`, `admin`). **No raw arbitrary SQL execution tool exposed**.
- **Cryptographic API Key Management**:
  - Generates secure `cb_live_` prefixed API keys
  - Plaintext shown **once** upon creation
  - Stores only SHA-256 hashes in database
  - Scoped permissions, expiration dates, and revocation tracking
- **Extensible Webhook System**: Signature verification (HMAC SHA-256), idempotency tracking with `webhook_events`, and asynchronous background dispatch.
- **Lightweight Background Worker**: In-process worker pool with retry queues and graceful drain on shutdown (zero Kafka/Redis required for V1).
- **Security Hardened**: Token-bucket rate limiting, 2MB request body size limits, CORS headers, security headers, parameterized SQL, context cancellation timeouts, and structured error responses.
- **Shadcn Next.js Admin Dashboard**: Modern dark-mode dashboard for inspecting tables, browsing live data, managing API keys, exploring projects, testing MCP tools, and viewing system health.

---

## 🚀 Quickstart

### 1. Run with Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/lunoxd/cobalt.git
cd cobalt

# Copy environment template
cp .env.example .env

# Start Cobalt Backend, PostgreSQL 16, and Admin Dashboard
docker compose up -d --build
```

- **Backend API**: `http://localhost:8080`
- **Health Check**: `http://localhost:8080/health`
- **Readiness Check**: `http://localhost:8080/ready`
- **Admin Dashboard**: `http://localhost:3000`

---

### 2. Local Development

#### Prerequisites
- Go 1.24+
- PostgreSQL 16 (or local instance)
- Node.js 20+

```bash
# 1. Start your local PostgreSQL instance:
# createdb cobalt

# 2. Configure environment:
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/cobalt?sslmode=disable"
export ADMIN_API_KEY="cb_admin_dev_secret"
export PORT="8080"

# 3. Run backend tests:
go test -v ./...

# 4. Start the Go backend:
go run ./cmd/server

# 5. In another terminal, start the Next.js Dashboard:
cd dashboard
npm install
npm run dev
```

The admin dashboard will be live at `http://localhost:3000`.

---

## 🤖 MCP (Model Context Protocol) Integration

Cobalt exposes a first-class MCP server at `http://localhost:8080/mcp`.

### Available MCP Tools & Scopes

| Tool | Required Scope | Description |
|------|----------------|-------------|
| `list_tables` | `database:read` | List public tables with row counts and sizes |
| `describe_table` | `database:read` | Table schema: columns, data types, indexes, PKs |
| `list_columns` | `database:read` | List columns and data types for a table |
| `list_indexes` | `database:read` | List indexes and uniqueness for a table |
| `get_rows` | `database:read` | Safely browse rows with limit, offset, and sort |
| `insert_row` | `database:write` | Safely insert a single row with parameterized data |
| `update_row` | `database:write` | Safely update a single row by primary key |
| `delete_row` | `database:write` | Safely delete a single row by primary key |
| `database_stats` | `database:read` | Database metrics, size, and connection count |
| `list_projects` | `projects:read` | List projects for the authenticated user |
| `get_project` | `projects:read` | Fetch a single project by UUID |

### AI Agent Configuration

Add Cobalt to your `claude_desktop_config.json` or Antigravity/Cursor MCP configuration:

```json
{
  "mcpServers": {
    "cobalt": {
      "url": "http://localhost:8080/mcp",
      "headers": {
        "Authorization": "Bearer cb_live_xxxxxxxxxxxxxxxxxxxxxxxx"
      }
    }
  }
}
```

---

## 📡 REST API Reference

### Health Endpoints

- `GET /health` - Liveness check (`200 OK`)
- `GET /ready` - Readiness check with PostgreSQL pool ping (`200 OK` or `503 Service Unavailable`)

### Projects API (`/api/projects`)

Requires header: `Authorization: Bearer <TOKEN>` or `X-API-Key: <KEY>`

- `GET /api/projects?limit=20&offset=0` - List user projects
- `POST /api/projects` - Create new project
  ```json
  {
    "name": "Rust WebAssembly Compiler",
    "description": "ZenCompiler project",
    "language": "rust",
    "code_content": "fn main() { println!(\"Hello Cobalt\"); }",
    "is_public": false
  }
  ```
- `GET /api/projects/:id` - Fetch project by UUID (accessible if owner, public, or admin)
- `PATCH /api/projects/:id` - Update project
- `DELETE /api/projects/:id` - Delete project

### Admin Inspector API (`/api/admin/inspector`)

Requires admin authorization: `Authorization: Bearer <ADMIN_API_KEY>`

- `GET /api/admin/stats` - System memory, Go runtime, pool stats
- `GET /api/admin/inspector/stats` - PostgreSQL database metrics
- `GET /api/admin/inspector/tables` - List all tables
- `GET /api/admin/inspector/tables/:table` - Describe table schema
- `GET /api/admin/inspector/tables/:table/rows?limit=25&offset=0` - Browse table rows
- `POST /api/admin/inspector/tables/:table/rows` - Insert row
- `PATCH /api/admin/inspector/tables/:table/rows` - Update row by PK
- `DELETE /api/admin/inspector/tables/:table/rows?pk_column=id&pk_value=...` - Delete row

### API Keys Management (`/api/admin/api-keys`)

- `GET /api/admin/api-keys` - List created API keys (prefix, scopes, status)
- `POST /api/admin/api-keys` - Create new API key
  ```json
  {
    "name": "Next.js Frontend Client",
    "scopes": ["projects:read", "projects:write"],
    "expires_in": "720h"
  }
  ```
- `DELETE /api/admin/api-keys/:id` - Revoke an API key

### Webhooks (`/webhooks/:provider`)

- `POST /webhooks/razorpay` - Receives Razorpay webhook with `X-Razorpay-Signature`
- `POST /webhooks/generic` - Receives webhook with `X-Hub-Signature-256`

---

## 🔒 Security Model

- **No Plaintext Passwords / Auth Independence**: External providers handle authentication. Cobalt validates tokens and extracts identity.
- **Hashed API Keys**: Keys are hashed with SHA-256. Plaintext is only visible once upon generation.
- **Strict Role Separation**:
  - `User`: Can only read/modify owned projects.
  - `Admin`: Accesses inspector, system stats, and all projects.
  - `MCP Client`: Restricted to explicitly granted scopes (`database:read`, `projects:read`, etc.).
- **No Arbitrary SQL in MCP**: AI agents cannot execute arbitrary SQL queries; all operations go through parameterized inspector tools.
- **Protection**: In-memory token bucket rate limiting (50 req/sec, burst 100), 2MB max request payload, and strict security headers.

---

## 💾 Backup & Restore

### Automated PostgreSQL Backup
```bash
# Dump the database
docker exec -t cobalt-postgres pg_dump -U postgres -d cobalt -F c -b -v -f /tmp/cobalt_backup.dump

# Copy to host
docker cp cobalt-postgres:/tmp/cobalt_backup.dump ./cobalt_backup_$(date +%Y%m%d_%H%M%S).dump
```

### Restore from Backup
```bash
# Copy into container
docker cp ./cobalt_backup.dump cobalt-postgres:/tmp/cobalt_backup.dump

# Restore into database
docker exec -t cobalt-postgres pg_restore -U postgres -d cobalt -c -v /tmp/cobalt_backup.dump
```

---

## 🚢 Coolify Deployment

1. Create a new service in Coolify and select **Docker Compose**.
2. Point Coolify to the repository: `https://github.com/lunoxd/cobalt`.
3. Set the environment variables in the Coolify UI:
   - `ADMIN_API_KEY`: A strong random string (e.g. `openssl rand -hex 32`)
   - `WEBHOOK_SECRET`: A strong random string
   - `POSTGRES_PASSWORD`: A strong random database password
4. Click **Deploy**. Coolify will build the Go service, mount persistent PostgreSQL volume, and launch the service with auto-healing.

---

## 📄 License

MIT License. Built for ZenCompiler.
