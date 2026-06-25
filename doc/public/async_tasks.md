# Asynchronous Tasks  
Long‑Running Operations in the Eluvio MCP Server

Some operations in the Eluvio MCP Server may take several seconds or minutes to complete. To support these workflows safely and predictably, the server provides a built‑in **asynchronous task system**.

This document explains:

- What async tasks are  
- How to start them  
- How to poll for progress  
- How results are returned  
- How cancellation works  
- Best practices for LLM agents  

---

# 1. Overview

Async tasks allow long‑running operations to execute **in the background** while the LLM continues interacting with the user.

Tools that support async execution include:

- `tag_content`  
- `tag_chapters`  
- `tag_characters`  

These tools accept a `synchronous` flag:

- `synchronous: true` → wait for completion  
- `synchronous: false` → return a `task_id` immediately  

The LLM can then poll the task using `task_status`.

---

# 2. Starting an Async Task

To start an async task, call a tool with:

``` json
{
  "name": "tag_content",
  "arguments": {
    "qid": "hq__123",
    "jobs": [{ "model": "scene" }],
    "synchronous": false
  }
}
```

### Example Response

``` json
{
  "task_id": "task_abc123"
}
```

The task is now running in the background.

---

# 3. Polling Task Status

Use the `task_status` tool to check progress.

### Example

``` json
{
  "name": "task_status",
  "arguments": {
    "task_id": "task_abc123"
  }
}
```

### Example Response (in progress)

``` json
{
  "state": "running",
  "progress": {
    "percent": 42.5,
    "message": "Tagging video segments..."
  }
}
```

### Example Response (completed)

``` json
{
  "state": "completed",
  "result": {
    "jobs": [
      {
        "model": "scene",
        "status": "success",
        "tagging_progress": "100%"
      }
    ]
  }
}
```

### Example Response (failed)

``` json
{
  "state": "failed",
  "error": "Model execution failed"
}
```

---

# 4. Task Lifecycle

Async tasks follow a predictable lifecycle:

1. **pending**  
   Task created but not yet started.

2. **running**  
   Worker is actively processing the task.

3. **completed**  
   Final result is available under `result`.

4. **failed**  
   Error occurred; details in `error`.

5. **canceled**  
   Task was stopped (e.g., via `stop_tagging`).

---

# 5. Progress Reporting

Workers may provide progress updates such as:

- Percent complete  
- Human‑readable messages  
- Model‑specific metadata  

Example:

``` json
{
  "state": "running",
  "progress": {
    "percent": 67.2,
    "message": "Uploading tag batches..."
  }
}
```

Progress fields are optional and may vary by task type.

---

# 6. Synchronous Mode

If `synchronous: true` is passed:

- The tool blocks until all jobs complete  
- The final result is returned directly  
- No `task_id` is created  

Example:

``` json
{
  "name": "tag_content",
  "arguments": {
    "qid": "hq__123",
    "jobs": [{ "model": "scene" }],
    "synchronous": true
  }
}
```

### Example Response

``` json
{
  "jobs": [
    {
      "model": "scene",
      "status": "success",
      "tagging_progress": "100%"
    }
  ]
}
```

---

# 7. Cancellation

Some async tasks can be canceled using:

- `stop_tagging` (Tagger workflows)

Example:

``` json
{
  "name": "stop_tagging",
  "arguments": {
    "qid": "hq__123"
  }
}
```

If the task is canceled:

``` json
{
  "state": "canceled",
  "message": "Tagging stopped by user request"
}
```

---

# 8. Best Practices for LLM Agents

### ✔️ Use async mode for long operations  
Tagging workflows may take minutes.

### ✔️ Poll periodically using `task_status`  
Do not assume completion.

### ✔️ Surface progress to the user  
Show percent complete or messages when available.

### ✔️ Avoid unnecessary polling  
Reasonable intervals: 1–3 seconds.

### ✔️ Respect cancellation  
If the user says “stop”, call `stop_tagging`.

### ✔️ Never guess task completion  
Always check `task_status`.

---

# 9. Summary

The async task system enables:

- Long‑running operations  
- Non‑blocking workflows  
- Progress reporting  
- Reliable polling  
- Safe cancellation  

Tools that support async mode return a `task_id`, which can be monitored using `task_status`.

For details on specific tools, see `tools_reference.md`.

