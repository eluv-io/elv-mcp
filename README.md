# Eluvio Search MCP Server

This tool lets MCP-enabled LLMs (Claude, OpenAI, etc.) search for clips using the **Eluvio Content SearchAI** and return **ready-to-play video clip results**.
Built in **Go**, it signs thumbnails and video URLs automatically and delivers structured search data.

**Provided Tool**
- `search_clips`

**Output Includes**
- Start/end timestamps for each clip
- Signed video URLs (direct playback)
- Signed thumbnail URLs
- JSON Response

# Setup
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
git clone git@github.com:qluvio/elv-mcp-experiment.git
cd elv-mcp-experiment
go mod tidy
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
  resource_url: https://<your-public-hostname>/mcp

fabric:
  qlibid_index: "iq__..."
  qid_index: "hq__..."
  search_base_url: "https://hosted-search.example/api"
  image_base_url: "https://images.fabric.example"
  vid_base_url: "https://videos.fabric.example"
  eth_url: "https://host.svc.eluv.io/eth/"
  qspace_id: "ispc__..."

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

Or using make:

```bash
make run
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

`<NGROK_LINK>/mcp`

Example:
`https://cool-ngrok-url.ngrok.io/mcp`


## Prompt Example
```text
Use the Fabric tool to search for {topic or scene} clips.
Return the top {number} results and display them as clickable thumbnails using Markdown.
Each result must include:

A thumbnail image URL embedded inside a clickable link,
formatted exactly like this: [![MovieTitle or description](THUMBNAIL_URL)](VIDEO_URL)

Do not return HTML tags, only Markdown.
```


## MCP Tool Details

### Tool Name
search_clips

### Input Structure (Args)

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

## Development Notes

- HTTP client is reused (connection reuse optimization)
- Video URL builder extracts `hq__` / `iq__` hashes
- Thumbnail URL builder prevents duplicate `/q/`
- Adds access tokens to both:
    - Header (Bearer)
    - Query param (`authorization=`)
- OAuth2/JWT authorization support via Ory
- State channel token is fetched on startup for signing
- For full documentation, consult Eluvio Search AI API docs.

## Done!
You now have an MCP-powered Eluvio Search Tool with full clip & thumbnail support.
Happy hunting!
