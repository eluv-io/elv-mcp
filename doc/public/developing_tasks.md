# Developing Tasks  
Contributor Guide for Extending the Eluvio MCP Server

This document explains how to develop new tasks for the Eluvio MCP Server.  
It is intended for contributors who want to add new capabilities, integrate new Fabric APIs, or extend existing workflows.

It covers:

- Task vs Worker architecture  
- How to define a new task  
- How to define input/output schemas  
- How to register tasks  
- How to implement workers  
- How to support async tasks  
- How to test tasks  
- Best practices for LLM‑safe tool design  

---

# 1. Architecture Overview

Every MCP tool exposed by the server is implemented using **two layers**:

1. **Task**  
   - Defines the MCP tool  
   - Provides the description  
   - Defines input schema  
   - Registers the tool  
   - Delegates to a worker  

2. **Worker**  
   - Implements business logic  
   - Calls Fabric / Tagger / TagStore APIs  
   - Normalizes results  
   - Returns MCP‑friendly output  

Tasks contain *no business logic*.  
Workers contain *no MCP wiring*.

This separation ensures:

- Clean, testable code  
- Predictable tool behavior  
- Easy extensibility  

---

# 2. Creating a New Task

A task is a Go struct that implements:

- `Name() string`  
- `Description() string`  
- `Register(server, cfg)`  

And defines:

- Input argument struct  
- Output struct (optional)  

### Example Task Skeleton

``` go
type MyTaskArgs struct {
    QID string `json:"qid"`
}

type MyTaskResult struct {
    Message string `json:"message"`
}

type MyTask struct{}

func NewMyTask() *MyTask { return &MyTask{} }

func (MyTask) Name() string { return "my_task" }

func (MyTask) Description() string {
    return "Describe what this tool does..."
}

func (MyTask) Register(server *mcp.Server, cfg *config.Config) {
    mcp.AddTool(
        server,
        &mcp.Tool{
            Name:        MyTask{}.Name(),
            Description: MyTask{}.Description(),
        },
        func(ctx context.Context, req *mcp.CallToolRequest, args MyTaskArgs) (*mcp.CallToolResult, any, error) {
            return MyTaskWorker(ctx, req, args, cfg)
        },
    )
}
```

### Required conventions

- Task names must be **snake_case**  
- Descriptions must be **LLM‑friendly**  
- Required parameters must be explicitly documented  
- Destructive operations must include warnings  

---

# 3. Writing LLM‑Safe Descriptions

Tool descriptions are critical: they determine how LLMs choose tools.

### Guidelines

✔️ Describe **exactly when** the tool should be used  
✔️ Describe **when NOT** to use it  
✔️ List **required parameters**  
✔️ List **optional parameters**  
✔️ Include **rules** for safe usage  
✔️ Avoid ambiguity  
✔️ Avoid implementation details  

### Example (good)

```
Use this tool when the user explicitly asks to delete a TagStore track.

Required:
 • qid
 • track

Rules:
 • This is a destructive operation; do not call unless the user clearly requests deletion.
```

---

# 4. Defining Input Schemas

Input schemas are generated from Go structs.

### Example

``` go
type Args struct {
    QID   string  `json:"qid"`
    Limit *int    `json:"limit,omitempty"`
    Query *string `json:"query,omitempty"`
}
```

### File Uploads

For tools like `search_images`, define:

```
"type": "string",
"format": "file",
"contentMediaType": "application/octet-stream"
```

This is done using a custom schema override in the task registration.

---

# 5. Implementing Workers

Workers contain all business logic.

### Worker Signature

``` go
func Worker(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args Args,
    cfg *config.Config,
) (*mcp.CallToolResult, any, error)
```

### Worker Responsibilities

- Validate arguments  
- Call Fabric / Tagger / TagStore APIs  
- Normalize responses  
- Return structured results  
- Handle errors gracefully  

Workers must **never** modify MCP server state.

---

# 6. Supporting Async Tasks

Async tasks are used for long‑running operations.

### To support async mode:

1. Accept `synchronous` in args  
2. If `false`, create a task via the async manager  
3. Return `{ "task_id": "..." }`  
4. Worker runs in background  
5. Use `task_status` for polling  

### Example (conceptual)

``` go
if !args.Synchronous {
    id := async.Start(...)
    return async.TaskStarted(id)
}
```

See `async_tasks.md` for full details.

---

# 7. Testing Tasks

Tests should cover:

### 1. Schema validation  
- Missing required fields  
- Invalid types  
- Optional fields  

### 2. Worker logic  
- Successful execution  
- Error handling  
- Edge cases  

### 3. Async behavior  
- Task creation  
- Progress updates  
- Completion  

### 4. LLM‑safety  
- Description correctness  
- No ambiguous instructions  

---

# 8. Best Practices

### ✔️ Keep tasks thin  
Tasks should only:
- Define schema  
- Provide description  
- Register tool  
- Call worker  

### ✔️ Keep workers pure  
Workers should:
- Contain all logic  
- Avoid side effects  
- Be testable  

### ✔️ Validate required fields  
If a required field is missing:
- Return a clear error  
- Do not attempt execution  

### ✔️ Avoid guessing user intent  
Tools must only run when explicitly requested.

### ✔️ Use consistent naming  
- snake_case for tool names  
- CamelCase for Go structs  

### ✔️ Document destructive operations  
E.g., `tagstore_delete_track`.

---

# 9. Summary

To add a new task:

1. Create a task struct  
2. Define input/output structs  
3. Write a clear, LLM‑safe description  
4. Register the task  
5. Implement the worker  
6. Add async support if needed  
7. Write tests  
8. Update public documentation  

This ensures new tools integrate cleanly into the MCP server and behave predictably when used by LLMs.

