# Eluvio MCP Server — Tools Reference  

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

## 2.4 `content.get_metadata`

Retrieve the public user metadata for a Fabric content object.

### Arguments (GetMetadataArgs)

{
  "content_id": "string"
}

- **content_id** *(string, required)* — Content hash, ID, or write token.

### Returns (GetMetadataResult)

``` json
{
  "data": {
    "... dynamic metadata fields ...": "values"
  }
}
```

## 2.5 `content.get_offerings`

Summarize all offerings for a Fabric content object, including duration, video/audio/subtitle tracks, DRM, and playout formats.

### Arguments (GetOfferingsArgs)

{
  "content_id": "string"
}

- **content_id** *(string, required)* — Content hash, ID, or write token.

### Returns (GetOfferingsResult)

A map keyed by offering name:

``` json
{
  "offerings": {
    "<offering_name>": {
      "duration_seconds": 0.0,
      "duration_human": "string",
      "video": {
        "codec": "string",
        "width": 0,
        "height": 0,
        "frame_rate": "string",
        "aspect_ratio": "string",
        "hdr": {},
        "bitrate": 0,
        "default_for_media_type": false
      },
      "video_tracks": [
        {
          "codec": "string",
          "width": 0,
          "height": 0,
          "frame_rate": "string",
          "aspect_ratio": "string",
          "hdr": {},
          "bitrate": 0,
          "default_for_media_type": false
        }
      ],
      "audio_tracks": [
        {
          "label": "string",
          "language": "string",
          "channels": 0,
          "layout": "string",
          "codec": "string",
          "bitrate": 0,
          "default_for_media_type": false
        }
      ],
      "subtitle_tracks": [
        {
          "label": "string",
          "language": "string",
          "codec": "string",
          "forced": true,
          "hearing_impaired": false,
          "default_for_media_type": false
        }
      ],
      "drm": {
        "optional": false,
        "schemes": ["string"]
      },
      "playout_formats": [
        {
          "name": "string",
          "drm": "string",
          "protocol": "string"
        }
      ],
      "ready": false
    }
  }
}
```

### Behavioral Notes

- Returns **all offerings**, keyed by offering name.  
- Selects a **primary video track** using deterministic rules (default flag → resolution → bitrate).  
- Includes **all video tracks**, **all audio tracks**, and **all subtitle tracks**.  
- Extracts DRM schemes from both `playout.drm_keys` and `playout.streams.*.encryption_schemes`.  
- Ignores mezzanine‑internal fields (prep specs, billing items, part hashes, etc.).


### Behavioral Notes

- Returns the full JSON object from `/struct/meta/user/public`.  
- Only `name` and `description` are guaranteed fields; all others are dynamic.  
- Fails if `content_id` is missing, tenant context is missing, or Fabric API calls fail.  


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

# **3.5 `tagger_cleanup_jobs`**

Delete Tagger jobs for a Fabric content object.

Supports two modes:

1. **Delete a specific job** (when `job_id` is provided)  
2. **Cleanup all final‑state jobs** for a content ID (when `job_id` is omitted)

Final states:

- `completed`
- `failed`
- `canceled`

---

## Arguments (TaggerCleanupJobsArgs)

``` json
{
  "qid": "string",
  "job_id": "string (optional)"
}
```

### Field definitions

- **qid** *(string, required)* — Fabric content ID.
- **job_id** *(string, optional)* — If provided, delete only this job.

---

## Returns (TaggerCleanupJobsResult)

``` json
{
  "qid": "string",
  "deleted_job_ids": ["string"],
  "total_jobs": 0,
  "final_jobs": 0
}
```

### Field definitions

- **qid** — Content ID targeted.
- **deleted_job_ids** — List of job IDs successfully deleted.
- **total_jobs** — Total jobs returned by job‑status API.
- **final_jobs** — Number of jobs in a final state.

---

## Behavioral Notes

- If `job_id` is provided:
  - Only that job is deleted.
  - No job‑status listing is performed.

- If `job_id` is omitted:
  - The tool calls  
    `GET {AITaggerUrl}/{qid}/job-status?authorization=...`
  - Deletes only jobs in final states.

- Uses editor‑signed tokens for authorization.

- MCP error contract:
  - On error → `CallToolResult.IsError = true`, payload = null  
  - On success → payload = `TaggerCleanupJobsResult`
---
## 3.6 `tag_content`

Start one or more Tagger jobs on a Fabric content object.

Use this tool for **general tagging requests** or **multi‑model workflows**.  
Use specialized tools such as `tag_chapters` when the user explicitly requests those workflows.

### Arguments (TagContentArgs)

``` json
{
  "qid": "string",
  "options": {
    "destination_qid": "string",
    "replace": false,
    "max_fetch_retries": 0,
    "scope": {
      "key": "value"
    }
  },
  "jobs": [
    {
      "model": "string",
      "model_params": {
        "key": "value"
      },
      "overrides": {
        "destination_qid": "string",
        "replace": false,
        "max_fetch_retries": 0,
        "scope": {
          "key": "value"
        }
      }
    }
  ],
  "synchronous": false
}
```

#### Field definitions

- **qid** *(string, required)*  
  Fabric content identifier (QID, hash, or write token).

- **options** *(object, optional — TaggerOptions)*  
  Global options applied to **all** jobs in the request:
  - **destination_qid** *(string, optional)* — Where tags should be written.  
    If omitted, tags are written back to the source content.
  - **replace** *(bool, optional)* — Whether to overwrite existing tags on the destination track(s).  
    - `true` → existing tags may be replaced.  
    - `false` (default) → tags are appended/merged.
  - **max_fetch_retries** *(int, optional)* — Maximum number of retries when fetching content parts before failing.
  - **scope** *(object, optional)* — Tagging scope, such as:
    - time ranges  
    - subset of assets  
    - livestream parameters  
    The exact shape is model‑ and deployment‑specific.

- **jobs** *(array, required — TagJobSpec[])*  
  List of individual job specifications. Each job has:
  - **model** *(string, required)* — Model identifier to run (e.g., `"shot"`, `"asr"`, `"chapters"`).  
    Must be one of the supported Tagger models.
  - **model_params** *(object, optional)* — Model‑specific parameters, e.g.:  
    ``` json
    { "fps": 1.5 }
    ```
  - **overrides** *(object, optional — TaggerOptions)* — Per‑job overrides of the global `options`:
    - **destination_qid** *(string, optional)*  
    - **replace** *(bool, optional)*  
    - **max_fetch_retries** *(int, optional)*  
    - **scope** *(object, optional)*  

- **synchronous** *(bool, optional)*  
  - `true` → wait for all jobs to complete and return final statuses.  
  - `false` or omitted → start jobs and return an async `task_id` for polling via `task_status`.

### Sync Result (TagContentSyncResult)

Returned when `synchronous` is `true`.

``` json
{
  "jobs": [
    {
      "job_id": "string",
      "model": "string",
      "status": "string",
      "time_running": 0.0,
      "tagging_progress": "string",
      "missing_tags": ["string"],
      "failed": ["string"]
    }
  ]
}
```

#### TagJobStatus fields

- **job_id** *(string)* — Tagger job identifier.  
- **model** *(string)* — Model name used for this job.  
- **status** *(string)* — Current job status (e.g., `"running"`, `"completed"`, `"failed"`).  
- **time_running** *(number)* — Time the job has been running, in seconds.  
- **tagging_progress** *(string)* — Progress summary (e.g., `"1/3"`).  
- **missing_tags** *(string[], optional)* — Tags that could not be produced.  
- **failed** *(string[], optional)* — Failed content parts or error reasons.

### Async Result (TagContentAsyncResult)

Returned when `synchronous` is `false` or omitted.

``` json
{
  "task_id": "string"
}
```

#### Field definitions

- **task_id** *(string)* — Identifier for the async task, to be polled via the `task_status` tool.

### Behavioral Notes

- This tool **does not perform dependency resolution**; it runs exactly the jobs specified.  
- Use this tool for **flexible or multi‑model tagging** when the user describes specific models or workflows.  
- Do **not** invent model names or parameters; if unclear, ask the user or describe what is required.  
- If required inputs (e.g., `qid`, `jobs`) are missing, respond with which fields are required.

---

## 3.7 `stop_tagging`

Stop tagging jobs for a Fabric content object.

Use this tool only when the user explicitly asks to **stop** or **cancel** tagging.

### Arguments (TaggerStopArgs)

``` json
{
  "qid": "string",
  "model": "string"
}
```

#### Field definitions

- **qid** *(string, required)*  
  Fabric content identifier whose jobs should be stopped.

- **model** *(string, optional)*  
  - If provided → stop only jobs for this model.  
  - If omitted → stop **all** running jobs for the given `qid`.

### Result (TaggerStopResult)

``` json
{
  "jobs": [
    {
      "job_id": "string",
      "message": "string"
    }
  ]
}
```

#### TagStopStatus fields

- **job_id** *(string)* — Identifier of the job that was targeted.  
- **message** *(string)* — Human‑readable status message (e.g., `"stopped"`, `"already completed"`).

### Behavioral Notes

- This operation is **always synchronous**.  
- Stopping a job typically transitions it to a **canceled** or **stopped** state in Tagger.  
- Use only when the user clearly requests interruption or cancellation.  
- If `qid` is missing, clearly state that it is required.

---

## 3.8 `tag_status`

Retrieve tagging status for a Fabric content object, optionally filtered by model.

Use this tool when the user asks for **tagging progress**, **current status**, or **completion state**.

### Arguments (TaggerTagStatusArgs)

``` json
{
  "qid": "string",
  "model": "string"
}
```

#### Field definitions

- **qid** *(string, required)*  
  Fabric content identifier whose tagging status should be inspected.

- **model** *(string, optional)*  
  If provided, return status only for this model; otherwise return status for all models.

### Summary Result (TagStatusSummaryResponse)

When requesting overall status (no specific model detail):

``` json
{
  "statuses": [
    {
      "model": "string",
      "track": "string",
      "last_run": "string",
      "percent_complete": 0.0
    }
  ]
}
```

#### TagStatusSummary fields

- **model** *(string)* — Model name (e.g., `"asr"`, `"chapters"`).  
- **track** *(string)* — Tag track associated with the model.  
- **last_run** *(string)* — Timestamp of the last run (ISO‑8601 string).  
- **percent_complete** *(number)* — Completion percentage for this model on the content.

### Detailed Model Result (TagStatusModelDetail)

When inspecting a specific model in detail:

``` json
{
  "summary": {
    "model": "string",
    "track": "string",
    "last_run": "string",
    "percent_complete": 0.0,
    "num_content_parts": 0
  },
  "jobs": [
    {
      "time_ran": "string",
      "params": {
        "key": "value"
      },
      "status": {
        "key": "value"
      },
      "upload": {
        "key": "value"
      }
    }
  ]
}
```

#### TagStatusModelSummary fields

- **model** *(string)* — Model name.  
- **track** *(string)* — Tag track name.  
- **last_run** *(string)* — Timestamp of the last run.  
- **percent_complete** *(number)* — Completion percentage.  
- **num_content_parts** *(int)* — Number of content parts considered.

#### TagStatusJobDetail fields

- **time_ran** *(string)* — Timestamp when this job ran.  
- **params** *(object)* — Parameters used for this job (mirrors Tagger job params).  
- **status** *(object)* — Raw job status details from Tagger.  
- **upload** *(object)* — Raw upload status details from Tagger.

### Behavioral Notes

- Use this tool instead of guessing whether tagging has completed.  
- Supports both **high‑level summaries** and **detailed per‑job information**.  
- If `qid` is missing, clearly state that it is required.

---

# 4. Tagstore Tools

## 4.1 `tagstore_create_track`

Create a TagStore track for a Fabric content object.

Use this tool when the user asks to **create a new TagStore track**.

### Arguments (TagStoreCreateTrackArgs)

``` json
{
  "qid": "string",
  "track": "string",
  "label": "string",
  "color": "string",
  "description": "string"
}
```

#### Field definitions

- **qid** *(string, required)*  
  Fabric content identifier for which the track will be created.

- **track** *(string, required)*  
  Unique internal track name (e.g., `"speech_to_text"`, `"chapters"`).

- **label** *(string, optional)*  
  Human‑readable track label shown in UIs.

- **color** *(string, optional)*  
  Hex color code used for visual representation (e.g., `"#FF0000"`).

- **description** *(string, optional)*  
  Free‑form description of the track’s purpose.

### Result (TagStoreCreateTrackResult)

``` json
{
  "qid": "string",
  "track": "string",
  "track_id": "string",
  "message": "string",
  "created": true
}
```

#### Field definitions

- **qid** *(string)* — Content identifier targeted.  
- **track** *(string)* — Track name requested.  
- **track_id** *(string)* — Unique TagStore track identifier created.  
- **message** *(string)* — Human‑readable confirmation or status message.  
- **created** *(bool)* — Indicates whether the track was created successfully.

### Behavioral Notes

- Fails if a track with the same name already exists for the given `qid`.  
- Use only when the user explicitly requests track creation.  
- If required inputs are missing, state which ones are required.

---

## 4.2 `tagstore_delete_track`

Delete a TagStore track for a Fabric content object, including all associated batches and tags.

Use this tool only when the user explicitly asks to **delete** a TagStore track.

### Arguments (TagStoreDeleteTrackArgs)

``` json
{
  "qid": "string",
  "track": "string"
}
```

#### Field definitions

- **qid** *(string, required)*  
  Fabric content identifier whose track should be deleted.

- **track** *(string, required)*  
  Track name to delete.

### Result (TagStoreDeleteTrackResult)

``` json
{
  "qid": "string",
  "track": "string",
  "deleted": true
}
```

#### Field definitions

- **qid** *(string)* — Content identifier targeted.  
- **track** *(string)* — Track name requested for deletion.  
- **deleted** *(bool)* — Indicates whether the track was successfully deleted.

### Behavioral Notes

- This is a **destructive operation**; it removes the track and all associated TagStore data.  
- Do not use this tool for inspection or status checks.  
- If required inputs are missing, state which ones are required.

---
## 4.3 `tagstore_list_tracks`

List all TagStore tracks for a Fabric content object, returning **full track metadata**.

This tool queries the TagStore API:

- `GET {TagStoreUrl}/{qid}/tracks?authorization=...`

and returns the complete list of tracks associated with the content.

---

### Arguments (TagStoreListTracksArgs)

``` json
{
  "qid": "string"
}
```

#### Field definitions

- **qid** *(string, required)* — Fabric content ID whose tracks should be listed.

---

### Returns (TagStoreListTracksResult)

``` json
{
  "qid": "string",
  "tracks": [
    {
      "id": "string",
      "qid": "string",
      "name": "string",
      "label": "string",
      "color": "string",
      "description": "string"
    }
  ]
}
```

#### Track object fields (TagStoreTrack)

- **id** *(string)* — Unique TagStore track identifier.  
- **qid** *(string)* — Content ID the track belongs to.  
- **name** *(string)* — Internal track name (e.g., `"speech_to_text"`).  
- **label** *(string)* — Human‑readable label.  
- **color** *(string)* — Hex color used in UI.  
- **description** *(string)* — Optional descriptive text.

---

### Behavioral Notes

- Returns **all tracks** for the content, not just names.  
- Uses an **editor‑signed token** for authorization.  
- MCP error contract:
  - On error → `CallToolResult.IsError = true`, payload = null  
  - On success → payload = `TagStoreListTracksResult`  
- Fails if:
  - `qid` is missing  
  - Tenant context is missing  
  - TagStore returns non‑200 status codes  
