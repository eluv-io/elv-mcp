# Asynchronous Task Support

This module provides first‑class asynchronous task execution for the MCP server.  
It enables tasks to return immediately with a `task_id` while work continues in the background.

---

## Overview

Some operations (batch clip processing, metadata extraction, content analysis) may take longer than the typical MCP round‑trip. To support these workflows, the server exposes a lightweight asynchronous task system:

- Tasks run in the background.
- Each task has a unique ID.
- Clients can poll for status using the `task_status` task.
- Results are stored in memory until retrieved.

---

## Task Lifecycle

A task moves through the following states:

| State       | Meaning |
|-------------|---------|
| `pending`   | Task created but not yet started |
| `running`   | Task is executing in a background goroutine |
| `completed` | Task finished successfully and result is available |
| `failed`    | Task encountered an error |

---

## Creating a Task

Any task can create an asynchronous task using:

```go
taskID := tasks.StartAsyncTask(ctx, func(ctx context.Context) (any, error) {
    // long-running work here
})
```

The task should return:

```json
{
  "task_id": "<id>",
  "status": "pending"
}
```

---

## Checking Task Status

The `task_status` task accepts:

```json
{ "task_id": "<id>" }
```

And returns:

```json
{
  "task_id": "<id>",
  "status": "running|completed|failed",
  "result": { ... }   // only when completed
}
```

---

## Asynchronous Tasks
When implementing an asynchronous task, refer to this document

- [Async Task Manager & Progress Reporting](tasks/async_tasks.md)

---

## Thread Safety

The task manager uses a mutex to ensure safe concurrent access.  
All tasks are stored in memory and remain available for the lifetime of the server process.

---

## Future Extensions

Still to implement:

- Task cancellation  
- TTL‑based cleanup  
- Persistent task storage  
- Distributed execution 
- Progress reporting  
