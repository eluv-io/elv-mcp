package tasks

import (
	"time"
)

// Status represents the lifecycle state of an asynchronous task.
//
// The state machine is intentionally minimal and stable:
//   - StatusPending:   task has been created but not yet started
//   - StatusRunning:   task is currently executing
//   - StatusCompleted: task finished successfully and Result is available
//   - StatusFailed:    task terminated with a user-facing error
//
// Additional states (e.g. cancelled) can be introduced later without
// breaking existing clients, as long as the string values remain stable.
type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
)

// AsynchTask represents a single asynchronous unit of work managed by the
// in-memory task manager.
//
// AsynchTasks are created by tasks that want to offload long-running
// work to a background goroutine while returning a task identifier
// to the MCP client.
//
// Result is stored as map[string]any so the task manager remains
// JSON-safe and MCP-compatible while still being agnostic of
// domain-specific types.
type AsynchTask struct {
	ID        string         `json:"id"`
	Status    Status         `json:"status"`
	Result    map[string]any `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// Snapshot is an immutable view of a AsynchTask suitable for returning
// to callers without exposing internal pointers or synchronization
// concerns.
type Snapshot struct {
	ID        string         `json:"id"`
	Status    Status         `json:"status"`
	Result    map[string]any `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// snapshotFromAsynchTask creates a value copy of the given AsynchTask suitable
// for returning to external callers. It must be called while the
// caller holds the manager's lock.
func snapshotFromAsynchTask(t *AsynchTask) Snapshot {
	if t == nil {
		return Snapshot{}
	}
	return Snapshot{
		ID:        t.ID,
		Status:    t.Status,
		Result:    t.Result,
		Error:     t.Error,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}
