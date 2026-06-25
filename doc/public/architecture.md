# Architecture  
Internal Design of the Eluvio MCP Server

This document describes the internal architecture of the **Eluvio MCP Server**, including configuration loading, authentication, tenant mapping, Fabric integration, tool execution, async task processing, and the HTTP/MCP interface.

The goal is to provide a clear understanding of how the server works internally and how each subsystem interacts with the others.

---

# 1. High‑Level Overview

The Eluvio MCP Server exposes a Model Context Protocol (MCP) endpoint that allows LLM clients to:

- Search Fabric content  
- Tag content using AI Tagger workflows  
- Manage TagStore tracks  
- Execute long‑running tasks asynchronously  

The server is composed of:

1. **Configuration Loader** (`config.LoadConfig`)
2. **Tenant Registry** (maps OAuth `sub` → tenant)
3. **Fabric Integration Layer**
4. **MCP Server Core** (`mcpserver.NewServer`)
5. **HTTP Interface** (`mcpserver.NewHTTPMux`)
6. **Tool Handlers**
7. **Async Task Manager**
8. **Logging Subsystem**

The server is configured entirely via `config.yaml`.

---

# 2. Configuration System

Configuration is loaded at startup via:

``` go
cfg, err := config.LoadConfig()
```

This reads `config.yaml` and populates a `Config` struct.

## 2.1 Server Configuration

``` yaml
server:
  port: 8181
  oauth_issuer: https://auth.example.com
  resource_url: https://mcp.example.com
```

### Meaning:

- **port** — TCP port the server listens on  
- **oauth_issuer** — OAuth2 provider used for token validation  
- **resource_url** — Identifier for this MCP server as a protected resource  

The server does **not** store OAuth client secrets or perform token exchange.

---

## 2.2 Fabric Configuration

``` yaml
fabric:
  qlibid_index: "ilibXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
  qid_index: "iq__XXXXXXXXXXXXXXXXXXXXXXXXXXXX"
  search_base_url: "https://ai.example.com"
  image_base_url: "https://images.example.com"
  vid_base_url: "https://videos.example.com"
  eth_url: "https://eth.example.com"
  qspace_id: "ispXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
```

These values define shared infrastructure endpoints for:

- Search  
- Image retrieval  
- Video retrieval  
- Ethereum signing  
- QSpace  

Defaults are applied if optional fields are omitted.

---

## 2.3 Tenant Registry

Tenants define per‑tenant Fabric credentials and user mappings:

``` yaml
tenants:
  - id: example
    users:
      - "00000000-0000-0000-0000-000000000000"
    fabric:
      private_key: "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
      qlibid_index: "ilibXXXXXXXXXXXXXXXXXXXXXXXXXXXX"
      qid_index: "iq__XXXXXXXXXXXXXXXXXXXXXXXXXXXX"
```

### Tenant model:

- Each tenant has:
  - A private key  
  - Fabric index IDs  
  - Optional search collection ID  
- Each user is identified by OAuth `sub`  
- A user belongs to exactly one tenant  

### Internal representation:

``` go
type TenantRegistry struct {
    byUser map[string]*Tenant
}
```

The registry is built at startup.

---

# 3. Authentication Architecture

The server uses **OAuth2 Authorization Code Flow (no PKCE)**, but:

### ✔ The MCP client performs OAuth2  
### ✔ The server only validates tokens  

Token validation steps:

1. Fetch JWKS from `oauth_issuer`
2. Verify signature
3. Verify issuer claim
4. Verify expiration
5. Extract `sub`
6. Map `sub` → tenant

If the user is not found in the tenant registry, the request is rejected.

---

# 4. MCP Server Core

The MCP server is created via:

``` go
server := mcpserver.NewServer(cfg)
```

This initializes:

- Tool registry  
- Async task manager  
- Fabric clients  
- Tagger clients  
- TagStore clients  
- Authentication middleware  

The server exposes all MCP tools defined under:

```
/tasks/all
```


which registers:

- Search tools  
- Tagging tools  
- TagStore tools  
- Async task tools  

---

# 5. HTTP Interface

The HTTP interface is created via:

``` go
mux := mcpserver.NewHTTPMux(server, cfg)
```

This sets up:

- `/mcp` — MCP endpoint  
- Authentication middleware  
- JSON‑RPC framing  
- Streaming support (if enabled)  

The server listens on:

``` go
http.ListenAndServe(fmt.Sprintf(":%d", cfg.Server.Port), mux)
```

---

# 6. Tool Execution Pipeline

When an MCP client calls a tool:

1. **HTTP request arrives** at `/mcp`
2. **Authentication middleware** validates the token
3. **Tenant lookup** determines Fabric credentials
4. **MCP server** parses the JSON‑RPC request
5. **Tool handler** is invoked
6. **Fabric or Tagger operations** are executed
7. **Response** is returned to the client

### Tool categories:

- **Search Tools**
  - `search_clips`
  - `search_images`
  - `refresh_clip_urls`

- **Tagger Tools**
  - `tag_content`
  - `tag_chapters`
  - `tag_characters`
  - `tag_status`
  - `cancel_tagging`
  - `list_models`

- **TagStore Tools**
  - `create_track`
  - `delete_track`

- **Async Task Tools**
  - `task_status`
  - `task_cancel`

---

# 7. Async Task Architecture

Long‑running operations (e.g., tagging workflows) run asynchronously.

### Components:

- **Task Manager**
- **Task Registry**
- **Worker Pool**
- **Status Store**

### Workflow:

1. Client calls a tool that starts an async task  
2. Server returns a `task_id`  
3. Worker executes the task in the background  
4. Client polls using `task_status`  
5. Task may complete, fail, or be cancelled  

Tasks are registered via:

```
import _ "github.com/qluvio/elv-mcp/tasks/all"
```


which auto‑registers all task types.

---

# 8. Fabric Integration

Each tenant has:

- A private key  
- QLib index ID  
- Q index ID  
- Optional search collection ID  

The server:

- Signs Fabric requests using the tenant’s private key  
- Caches Fabric tokens per tenant  
- Uses mutexes to protect token access  

Fabric operations include:

- Clip search  
- Image search  
- Video URL generation  
- Tagging workflows  
- TagStore operations  

---

# 9. Logging Architecture

Logging is configured via:

``` yaml
log:
  level: info
  formatter: text
  file:
    filename: /var/log/elvmcp.log
    maxsize: 100
    maxbackups: 5
```

The server uses `elog` for:

- Structured logging  
- Named loggers  
- File rotation (via Lumberjack)  

---

# 10. Summary

The Eluvio MCP Server architecture consists of:

- A configuration‑driven startup model  
- OAuth2 token validation (client performs OAuth2)  
- Tenant‑based Fabric credential mapping  
- A modular MCP tool registry  
- A robust async task system  
- A clean HTTP/MCP interface  
- Strong logging and security protections  

This architecture enables safe, deterministic, multi‑tenant access to Eluvio Fabric through LLM applications.
