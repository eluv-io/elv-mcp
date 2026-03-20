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
cd <YourRepo>/elv-mcp-experiment
git mod tidy
```
## Set Environment Variables

| Variable | Description |
|---------|-------------|
| `SEARCH_BASE_URL` | Base search endpoint URL |
| `QLIBID_INDEX` | QLib QID for search queries |
| `QID_INDEX` | QID for search queries |
| `INDEX_AUTH_TOKEN` | Bearer auth token for search |
| `IMAGE_BASE_URL` | Base URL for thumbnails |
| `QAUTH_TOKEN` | Video authorization token |
| `VID_BASE_URL` | Base URL for constructed video URLs |
### Example: Set `.env` variables
export SEARCH_BASE_URL="https://hosted-search.example/api"  
export QLIBID_INDEX="iq__123..."  
export QID_INDEX="hq__abc..."  
export INDEX_AUTH_TOKEN="eyJhbGciOiJI..."  
export IMAGE_BASE_URL="https://images.fabric.example"  
export QAUTH_TOKEN="ht__456..."  
export VID_BASE_URL="https://videos.fabric.example"

## Run the MCP server
go run .

## Start Ngrok tunnel
ngrok http 8080

## Copy the URL output, for example:
https://cool-ngrok-url.ngrok.io

## Connect to Your LLM

In your MCP-enabled LLM (Claude, ChatGPT MCP, etc.), add a connector:

<NGROK_LINK>/mcp

Example:
https://cool-ngrok-url.ngrok.io/mcp


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
- For full documentation, consult Eluvio Search AI API docs.
## Done!
You now have an MCP-powered Eluvio Search Tool with full clip & thumbnail support. 🎬  
Happy hunting!





Publish to Github MCP Registry

- Security
- How are curently consumers accessing the search AI? what is the verification process?
- how do they get auth tokens etc?
- wider functionality?
- better search more interpreting
- maybe get metadata if you want 
- where do we put limits to that
- 