# Eluvio MCP Server — Tools Reference  
Authoritative specification generated directly from Go task definitions.

This document describes every MCP tool exposed by the Eluvio MCP Server, including:

- Tool name  
- Purpose  
- Exact argument schema (from Go structs)  
- Exact result schema (from Go structs)  
- Behavioral notes  

All schemas come directly from the task definitions in `./tasks/*/*_task.go`.

---

# 1. Async Task Tools

## 1.1 `task_status`

Retrieve the status or final result of an asynchronous task.

### Arguments (TaskStatusArgs)

``` json
{
  "task_id": "string"
}
``` 

- **task_id** *(string, required)* — Identifier returned by a previous async tool.

### Returns

`*mcp.CallToolResult` plus a `Snapshot` containing the async task state.

---

# 2. Fabric Tools

## 2.1 `refresh_clips`

Refresh authentication tokens in previously returned clip or thumbnail URLs.

### Arguments (RefreshClipsArgs)

``` json
{
  "contents": [
    {
      "... ClipItem fields ...": "see ClipItem struct"
    }
  ]
}
``` 

- **contents** *(array of ClipItem, required)* — Items whose URLs need refreshing.

### Returns

`ClipResponse` — refreshed URLs for each clip.

---

## 2.2 `search_images`

Search for images using text or an uploaded reference image.

### Arguments (SearchImagesArgs)

``` json
{
  "collection_id": "string",
  "query": "string",
  "image_path": "string",
  "image": "string"
}
``` 

- **collection_id** *(string, optional)* — Override search collection.
- **query** *(string, optional)* — Text search query.
- **image** *(string, optional)* — Uploaded file (LibreChat → MCP SDK → temp file).
- **image_path** *(string, optional)* — Resolved local path (worker uses this).

### Rules

- Provide **either** `query` **or** `image`.
- If both are empty → do not call.
- If `image` is present, the server maps it to `image_path`.

### Returns

Worker‑specific image search results.

---

## 2.3 `search_clips`

Search for video clips using Fabric search.

### Arguments (SearchClipsArgs)

``` json
{
  "terms": "string",
  "search_fields": ["string"],
  "display_fields": ["string"],
  "semantic": "string",
  "start": 0,
  "limit": 20,
  "max_total": 100,
  "debug": false,
  "clips": true,
  "clips_include_source_tags": true,
  "thumbnails": true
}
``` 

- **terms** *(string, required)* — Search text.
- **search_fields** *(string[], optional)* — Fields to search.
- **display_fields** *(string[], optional)* — Fields to return.
- **semantic** *(string, optional)* — Semantic search mode.
- **start** *(int, optional, default 0)* — Pagination offset.
- **limit** *(int, optional, default 20)* — Max results.
- **max_total** *(int, optional, default 100)* — Cap on total hits.
- **debug** *(bool, optional, default false)* — Verbose output.
- **clips** *(bool, optional, default true)* — Include clip results.
- **clips_include_source_tags** *(bool, optional, default true)* — Include source tags.
- **thumbnails** *(bool, optional, default true)* — Include thumbnails.

### Returns

`ClipResponse` — clip metadata and signed URLs.

---

# 3. Tagger Tools

## 3.1 `tag_content`

Start one or more Tagger jobs (frame, celeb, speaker, etc.).

### Arguments (TagContentArgs)

``` json
{
  "qid": "string",
  "options": {
    "destination_qid": "string",
    "replace": false,
    "max_fetch_retries": 0,
    "scope": { "key": "value" }
  },
  "jobs": [
    {
      "model": "string",
      "model_params": { "key": "value" },
      "overrides": {
        "destination_qid": "string",
        "replace": false,
        "max_fetch_retries": 0,
        "scope": { "key": "value" }
      }
    }
  ],
  "synchronous": false
}
``` 

#### Field definitions

- **qid** *(string, required)* — Fabric content ID.
- **options** *(TaggerOptions, optional)* — Global options.
- **jobs** *(array of TagJobSpec, required)* — Individual model jobs.
- **synchronous** *(bool, optional)*  
  - true → wait for completion  
  - false → return async task ID

### Result (sync)

``` json
{
  "jobs": [ TagJobStatus ]
}
``` 

### Result (async)

``` json
{
  "task_id": "string"
}
``` 

### Supporting Types

#### TaggerOptions

``` json
{
  "destination_qid": "string",
  "replace": false,
  "max_fetch_retries": 0,
  "scope": { "key": "value" }
}
``` 

#### TagJobSpec

``` json
{
  "model": "string",
  "model_params": { "key": "value" },
  "overrides": TaggerOptions
}
``` 

#### TagJobStatus

``` json
{
  "model": "string",
  "status": "string",
  "time_running": 0.0,
  "tagging_progress": "string",
  "missing_tags": ["string"],
  "failed": ["string"]
}
``` 

---

## 3.2 `tag_chapters`

High‑level workflow for chapter tagging with dependency resolution.

### Arguments (ChaptersTaggingArgs)

``` json
{
  "qid": "string",
  "auto_run_dependencies": false,
  "synchronous": false,
  "options": TaggerOptions
}
``` 

### Sync Result (ChaptersTaggingSyncResult)

``` json
{
  "jobs": [ TagJobStatus ],
  "auto_ran_dependencies": ["string"]
}
``` 

### Async Result (ChaptersTaggingAsyncResult)

``` json
{
  "task_id": "string"
}
``` 

---

## 3.3 `tag_characters`

High‑level workflow for character tagging with dependency resolution.

### Arguments (CharacterTaggingArgs)

``` json
{
  "qid": "string",
  "auto_run_dependencies": false,
  "synchronous": false,
  "options": TaggerOptions
}
``` 

### Sync Result (CharacterTaggingSyncResult)

``` json
{
  "jobs": [ TagJobStatus ],
  "auto_ran_dependencies": ["string"]
}
``` 

### Async Result (CharacterTaggingAsyncResult)

``` json
{
  "task_id": "string"
}
``` 

---

## 3.4 `list_models`

List available Tagger models.

### Arguments (ListModelsArgs)

``` json
{}
``` 

(no parameters)

### Returns (ModelsResponse)

``` json
{
  "models": [
    {
      "name": "string",
      "description": "string",
      "type": "string",
      "tag_tracks": [
        {
          "name": "string",
          "label": "string"
        }
      ]
    }
  ]
}
``` 

---

# Summary

This reference is generated **directly from your Go task definitions** and contains:

- Exact argument schemas  
- Exact result schemas  
- No hallucinated fields  
- No missing fields  
- No inferred defaults beyond what the code documents  

If you later update any task structs, upload the updated files and this document can be regenerated with the same precision.

