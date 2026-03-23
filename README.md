# Eluvio Content Fabric MCP Server

Allows MCP-enabled LLMs (ChatGPT, Claude, OpenAI, etc.) to access Eluvio Content Fabric functions such as search, clipping and comoposition generation, library content and metadata management and reporting, etc.
Built in **Go**, it signs thumbnails and video URLs automatically and delivers structured search data.

Currently only one tool implemented: search API.  More tools coming soon.

### Tools

#### `search_clips`

  Outputs:
  - Start/end timestamps for each clip
  - Signed video URLs (direct playback)
  - Signed thumbnail URLs
  - Metadata fields: display_title, release_date, ip_title_id

# How to Use

## ChatGPT

### Using the Eluvio MCP server

- Settings - enable "Developer Mode"
- Settings - "Create App"
  - Name: Eluvio Content Fabric
  - MCP URL: https://mcp.svc.eluv.io
  - Authorization: 'oauth'

# Developer Setup

## Install Go
https://go.dev/dl/

## Install Ngrok (for MCP tunneling)
https://ngrok.com/download

or
```bash
brew install --cask ngrok
```

## Clone Github repository

```bash
git clone git@github.com:eluv-io/elv-mcp.git
cd elv-mcp
```

## Configure `config.yaml`

Copy the sample config and fill in your values:

```bash
cp config-sample.yaml config.yaml
```

Edit `config.yaml`:

```yaml
log:
  level: info
  formatter: text
  file:
    filename: elvmcp.log
    maxsize: 100
    maxbackups: 5

server:
  oauth_issuer: https://<your-ory-project>.projects.oryapis.com
  resource_url: https://<your-public-hostname>

fabric:
  qlibid_index: "ilib..."
  qid_index: "iq__..."
  search_base_url: "https://ai.contentfabric.io"
  image_base_url: "https://main.net955305.contentfabric.io"
  vid_base_url: "https://embed.v3.contentfabric.io"
  eth_url: "https://host-76-74-34-194.contentfabric.io/eth/"
  qspace_id: "ispc2RUoRe9eR2v33HARQUVSp1rYXzw1"

dev:
  private_key: "0x..."
```


| Field | Description |
|-------|-------------|
| `server.oauth_issuer` | Ory OAuth2 issuer URL (optional, has default) |
| `server.resource_url` | This server's public URL for OAuth metadata (optional, has default) |
| `fabric.qlibid_index` | QLib ID for the search index |
| `fabric.qid_index` | QID for the search index |
| `fabric.search_base_url` | Base URL for the search API |
| `fabric.image_base_url` | Base URL for thumbnail images |
| `fabric.vid_base_url` | Base URL for video playback |
| `fabric.eth_url` | Ethereum/fabric node URL |
| `fabric.qspace_id` | QSpace ID |
| `dev.private_key` | ECDSA private key (hex) for signing requests |

## Run the MCP server

```bash
go run ./cmd/elvmcpd
```


## Start Ngrok tunnel
```bash
ngrok http 8181
```

or:

```bash
make ngrok
```

## Copy the URL output, for example:
https://cool-ngrok-url.ngrok.io

## Connect to Your LLM

In your MCP-enabled LLM (Claude, ChatGPT MCP, etc.), add a connector:

`<NGROK_LINK>`

Example:
`https://cool-ngrok-url.ngrok.io`


## MCP Tool Details

### `search_clips`

#### Input Structure (Args)

| Field | Type | Default | Notes |
|-------|------|---------|-------|
| `terms` | string | required | Search query |
| `search_fields` | []string | optional | Override search fields |
| `display_fields` | []string | optional | Fields to return |
| `semantic` | string | optional | Semantic search value |
| `start` | int | 0 | Pagination |
| `limit` | int | 20 | Page size |
| `max_total` | int | 100 | Maximum results |
| `debug` | bool | false | Debug info |
| `clips` | *bool | true | Return clips? |
| `clips_include_source_tags` | *bool | true | Include metadata |
| `thumbnails` | *bool | true | Return thumbnails |
