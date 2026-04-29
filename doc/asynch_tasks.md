# Async Task Progress Reporting (MCP-Compatible)

This document describes how long‑running MCP tasks can report progress
back to the client (e.g., ChatGPT) using the Eluvio MCP Server’s asynchronous
task manager.

The Model Context Protocol (MCP) does **not** define a dedicated `progress`
field in the task status schema. However, MCP explicitly allows arbitrary
structured JSON inside the `result` field **even while the task is still
running**.

This means tasks can report progress by updating the task’s `result`
field with percentage and message information. MCP clients poll the
`task_status` tool and automatically display progress updates.

---

## Overview

Long‑running tasks should:

1. Launch work asynchronously using `tasks.StartAsyncTask` or `tasks.Async`.
2. Periodically call `tasks.ReportProgress(taskID, percent, message)` to
   update progress.
3. Return their final result normally from the async function.

The task manager will:

- Update the task’s `result` field with progress information.
- Update `updated_at` on every progress update.
- Keep `status = "running"` until the task completes or fails.
- Preserve any additional fields in the result (merge behavior).
- Ensure MCP clients see progress updates via `task_status`.

---

## API: `ReportProgress`

```
func ReportProgress(id string, percent int, message string) error
```

### Parameters

- `id`  
  The task ID returned by `StartAsyncTask` or `Async`.

- `percent`  
  An integer from 0 to 100 representing completion percentage.

- `message`  
  A human‑readable description of the current phase or activity.

### Behavior

- Updates the task’s `result` field with:
  - `progress`: integer percentage
  - `message`: optional description
- Merges into any existing result fields.
- Updates `updated_at`.
- Does **not** change the task’s status.
- Does **not** complete the task.
- Does **not** overwrite the final result.

---

## Example: Long‑Running Primitive

```
id := tasks.StartAsyncTask(ctx, func(ctx context.Context) (any, error) {
    for i := 0; i <= 100; i += 10 {
        tasks.ReportProgress(id, i, fmt.Sprintf("Processing %d%%", i))
        time.Sleep(200 * time.Millisecond)
    }

    // Final result returned when the task completes
    return map[string]any{
        "ok": true,
        "summary": "Processing completed successfully",
    }, nil
})
```

### What the client sees via `task_status`

While running:

```
{
  "status": "running",
  "result": {
    "progress": 40,
    "message": "Processing 40%"
  }
}
```

When complete:

```
{
  "status": "completed",
  "result": {
    "ok": true,
    "summary": "Processing completed successfully"
  }
}
```

---

## How MCP Clients Interpret Progress

MCP clients (including ChatGPT):

- Poll the `task_status` tool automatically.
- Display progress bars or incremental updates based on the JSON.
- Update the UI whenever `progress` or `message` changes.
- Stop polling when `status` becomes `completed` or `failed`.

This makes progress reporting a first‑class user experience without requiring
any MCP protocol extensions.

---

## Best Practices

- Call `ReportProgress` frequently for long tasks (every 100–500ms).
- Always include a meaningful `message`.
- Avoid jumping backwards in percentage.
- Keep progress updates lightweight (no large payloads).
- Do not mutate the final result inside progress updates.

---

## Error Handling

If a task fails:

- The task manager sets `status = "failed"`.
- The `error` field contains a user‑safe message.
- The `result` field is cleared.
- `updated_at` is updated.

Example:

```
{
  "status": "failed",
  "error": "transcoding failed: invalid input file"
}
```

---

## Summary

Progress reporting in MCP is achieved by updating the task’s `result` field
while the task is still running. The Eluvio MCP Server provides a clean,
ergonomic API (`ReportProgress`) that tasks can call to keep clients
informed of long‑running operations.

This approach is fully MCP‑compliant, client‑friendly, and requires no
protocol extensions.
