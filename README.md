# Eluvio Content Fabric MCP Server

The **Eluvio Content Fabric MCP Server** exposes Fabric search, tagging, and TagStore operations through the **Model Context Protocol (MCP)**.  
It enables MCP‑enabled LLMs—such as ChatGPT, Claude Desktop, LibreChat, and custom MCP clients—to interact with Fabric content using safe, structured, deterministic tools.

This repository contains:

- The MCP server implementation  
- Task + Worker definitions  
- Async task manager  
- Fabric search and URL refresh tools  
- Tagger workflows (content, chapters, characters)  
- TagStore management tools  
- Public documentation under `./doc/public`  

---

# Provided Tools

### Fabric Search & URL Management
- **`search_clips`** — Search the Eluvio Content Fabric and return signed, ready‑to‑play clip results  
- **`refresh_clips`** — Refresh expired authentication tokens in previously returned clip URLs  
- **`search_images`** — Search for images using text or an uploaded reference image  

### Tagger Workflows
- **`tag_content`** — Run one or more Tagger models (frame, celeb, speaker, etc.)  
- **`tag_chapters`** — High‑level chapter tagging workflow with dependency resolution  
- **`tag_characters`** — High‑level character tagging workflow with dependency resolution  
- **`list_models`** — List available Tagger models and their tag tracks  

### Async Task Management
- **`task_status`** — Query the status or final result of an asynchronous task  

All tool schemas are documented in `./doc/public/tools_reference.md`.

---

# MCP Tool Architecture

The server uses a **Task + Worker** architecture that cleanly separates:

- **Task** → metadata, registration, schema, description  
- **Worker** → business logic  

This ensures deterministic MCP behavior and testable logic.

## 1. Task (metadata + registration)

Located in:
```
tasks/<domain>/*_task.go
```

A Task:

- Defines the tool name  
- Provides the human‑readable description  
- Registers the tool with the MCP server  
- Wires the tool to its Worker  

Tasks contain **no business logic**.

## 2. Worker (business logic)

Located alongside the task, e.g.:

```
tasks/fabric/search_worker.go  
tasks/fabric/refresh_worker.go  
tasks/taggers/tagger_start_worker.go  
```

A Worker:

- Validates arguments  
- Calls the underlying Eluvio APIs  
- Constructs and returns the tool result  

Workers are pure logic and can be tested independently.

---

# Tool Registration

All tasks self‑register via `init()` and are loaded automatically:

``` go
import _ "github.com/qluvio/elv-mcp/tasks/all"
```

This pattern ensures:

- No manual registry maintenance  
- Adding a new tool = add a Task + Worker file  

---

# Developer Setup

## Install Go
https://go.dev/dl/

## Install Ngrok (for MCP tunneling)
https://ngrok.com/download

or:

```bash
brew install --cask ngrok
```

## Clone the repository

```bash
git clone git@github.com:eluv-io/elv-mcp.git
cd elv-mcp
```

---

# Configure `config.yaml`

Copy the sample:

```bash
cp config-sample.yaml config.yaml
```

Edit `config.yaml`:

``` yaml
log:
  level: info
  formatter: text
  file:
    filename: elvmcp.log
    maxsize: 100
    maxbackups: 5

server:
  port: 8181
  oauth_issuer: https://<your-auth-provider>
  resource_url: https://<your-public-hostname>

fabric:
  search_base_url: "https://ai.contentfabric.io"
  image_base_url: "https://host-76-74-91-7.contentfabric.io"
  vid_base_url: "https://embed.v3.contentfabric.io"
  eth_url: "https://host-76-74-34-194.contentfabric.io/eth/"
  qspace_id: "ispc2RUoRe9eR2v33HARQUVSp1rYXzw1"

tenants:
  - id: studio-a
    users:
      - "user|abc123"
      - "user|def456"
    fabric:
      private_key: "0x..."
      qlibid_index: "ilib..."
      qid_index: "iq__..."

  - id: studio-b
    users:
      - "user|xyz789"
    fabric:
      private_key: "0x..."
      qlibid_index: "ilib..."
      qid_index: "iq__..."
```

### Finding a user's `sub`

The `sub` claim is the unique user identifier in the JWT issued by your OAuth2 provider.  
Decode any access token (e.g., via https://jwt.io) or inspect your provider’s dashboard.

---

# Config Reference

### `server` section

| Field | Description |
|-------|-------------|
| `port` | TCP port the MCP server listens on |
| `oauth_issuer` | OAuth2 issuer URL |
| `resource_url` | Public URL representing this MCP server |

### `fabric` section (shared)

| Field | Description |
|-------|-------------|
| `search_base_url` | Base URL for Fabric search |
| `image_base_url` | Base URL for thumbnails |
| `vid_base_url` | Base URL for video playback |
| `eth_url` | Ethereum/Fabric node URL |
| `qspace_id` | QSpace ID |

### `tenants` list

| Field | Description |
|-------|-------------|
| `id` | Tenant name |
| `users` | OAuth2 `sub` values mapped to this tenant |
| `fabric.private_key` | Private key for signing Fabric requests |
| `fabric.qlibid_index` | QLib ID for this tenant |
| `fabric.qid_index` | QID for this tenant |

Unknown users → `403 Forbidden`.

---

# Run the MCP Server

```bash
go run ./cmd/elvmcpd
```

---

# Start Ngrok Tunnel

```bash
ngrok http 8181
```

Copy the public URL, e.g.:

```
https://cool-ngrok-url.ngrok.io
```

---

# Connect to Your LLM

In your MCP‑enabled LLM (Claude, ChatGPT MCP, LibreChat, etc.), add a connector pointing to your Ngrok URL.

Example:

```
https://cool-ngrok-url.ngrok.io
```

---

# Prompt Example

```text
Use the Fabric tool to search for {topic or scene} clips.
Return the top {number} results and display them as clickable thumbnails using Markdown.

Each result must include a thumbnail image URL embedded inside a clickable link:
[![Description](THUMBNAIL_URL)](VIDEO_URL)

Do not return HTML tags, only Markdown.
```

---

# MCP Tool Details (Example: `search_clips`)

### Input Structure

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `terms` | string | required | Search query |
| `search_fields` | []string | optional | Override search fields |
| `display_fields` | []string | optional | Fields to return |
| `semantic` | string | optional | Semantic search |
| `start` | int | 0 | Pagination offset |
| `limit` | int | 20 | Page size |
| `max_total` | int | 100 | Max results |
| `debug` | bool | false | Debug output |
| `clips` | *bool | true | Include clips |
| `clips_include_source_tags` | *bool | true | Include metadata |
| `thumbnails` | *bool | true | Include thumbnails |

---

# Documentation

### Public documentation (`./doc/public`)

- **Overview & index** — [`./doc/public/README.md`](./doc/public/README.md)  
- **Architecture** — [`./doc/public/architecture.md`](./doc/public/architecture.md)  
- **Integration guide** — [`./doc/public/integration_guide.md`](./doc/public/integration_guide.md)  
- **Authentication model** — [`./doc/public/authentication.md`](./doc/public/authentication.md)  
- **Tools reference** — [`./doc/public/tools_reference.md`](./doc/public/tools_reference.md)  
- **Async tasks** — [`./doc/public/async_tasks.md`](./doc/public/async_tasks.md)  
- **Developing tasks** — [`./doc/public/developing_tasks.md`](./doc/public/developing_tasks.md)  

### Internal / API documentation (`./doc`)

- **API reference** — [`./doc/API.md`](./doc/API.md)  
- **Async task internals** — [`./doc/async_task.md`](./doc/async_task.md)  

---

# Development Notes

- HTTP client is reused for connection pooling  
- Video URL builder handles `hq__` / `iq__` hashes  
- Thumbnail URL builder prevents duplicate `/q/`  
- Fabric access tokens added to both header and query param  
- OAuth2 Authorization Code Flow handled by the MCP client  
- Server validates tokens and maps `sub` → tenant  
- Each tenant has its own private key, search index, and token cache  
- State‑channel and editor‑signed tokens are generated lazily and cached  
- All tools require authentication  

For full API documentation, see:  
https://docs.eluv.io/docs/getting-started/resources/
