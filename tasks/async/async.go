package tasks

import "context"

// AsyncResult is the shape returned to MCP tasks when they
// choose to run asynchronously using the high-level wrapper.
//
// It is intentionally minimal: the client receives a task_id and
// an initial status of "pending".
type AsyncResult struct {
	TaskID string `json:"task_id"`
	Status Status `json:"status"`
}

// StartAsync is the low-level explicit API.
//
// Tasks call this when they want full control over how the
// async task is created and how the response is shaped.
//
// Example:
//
//	id := tasks.StartAsync(ctx, func(ctx context.Context) (any, error) {
//	    return doWork(ctx, args)
//	})
func StartAsync(ctx context.Context, fn func(context.Context) (any, error)) string {
	return StartAsyncTask(ctx, fn)
}

// Async is the high-level ergonomic wrapper.
//
// It creates an async task and returns a ready-to-serialize struct
// that tasks can return directly to the MCP client.
//
// Example:
//
//	result := tasks.Async(ctx, func(ctx) (any, error) {
//	    return SearchHandler(ctx, req, args, cfg)
//	})
//	return nil, result, nil
func Async(ctx context.Context, fn func(context.Context) (any, error)) AsyncResult {
	id := StartAsyncTask(ctx, fn)
	return AsyncResult{
		TaskID: id,
		Status: StatusPending,
	}
}
