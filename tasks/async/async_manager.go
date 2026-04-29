package tasks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager is a simple in-memory registry for asynchronous tasks.
//
// It is intentionally minimal and process-local:
//   - tasks are stored in memory only
//   - tasks live for the lifetime of the server process
//   - there is no persistence, sharding, or distribution
//
// This is sufficient for MCP use cases where the client is expected
// to poll for task status within the same server session.
type Manager struct {
	mu    sync.RWMutex
	tasks map[string]*AsynchTask
}

// defaultManager is a process-wide singleton used by helper functions
// when a caller does not need explicit control over task manager
// lifecycle. This keeps integration simple for tasks that just
// want to "fire and forget" a background task.
var defaultManager = NewManager()

// NewManager constructs a new, empty Manager instance.
//
// In most cases, callers should use the package-level helpers that
// delegate to the defaultManager. NewManager is exposed primarily
// for tests or advanced scenarios.
func NewManager() *Manager {
	return &Manager{
		tasks: make(map[string]*AsynchTask),
	}
}

// StartAsyncTask registers a new task and executes the provided
// function in a background goroutine.
//
// The function fn is responsible for performing the long-running
// work. When fn returns, the task's Status and Result/Error fields
// are updated accordingly.
//
// The returned task ID can be sent back to the MCP client, which
// can later query task status via a dedicated primitive.
func StartAsyncTask(ctx context.Context, fn func(context.Context) (any, error)) string {
	return defaultManager.StartAsyncTask(ctx, fn)
}

// GetSnapshot returns an immutable snapshot of the task with the
// given ID, if it exists. The boolean return indicates whether the
// task was found.
//
// Callers should treat a missing task as either an invalid ID or
// an expired/cleaned-up task, depending on future retention policy.
func GetSnapshot(id string) (Snapshot, bool) {
	return defaultManager.GetSnapshot(id)
}

func ReportProgress(id string, percent int, message string) error {
	return defaultManager.ReportProgress(id, percent, message)
}

// ReportProgress updates the task's result with progress information and
// refreshes the UpdatedAt timestamp without changing the task status.
//
// This is MCP-compatible progress reporting: clients poll task_status and
// see the updated result while status remains "running".
//
// Fields:
//   - percent: 0–100
//   - message: optional human-readable description
//
// Behavior:
//   - merges into existing Result map (does not discard other fields)
//   - updates UpdatedAt
//   - does NOT change Status
//   - does NOT complete the task
func (m *Manager) ReportProgress(id string, percent int, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", id)
	}

	if task.Result == nil {
		task.Result = make(map[string]any)
	}

	// Merge into existing result
	task.Result["progress"] = percent
	if message != "" {
		task.Result["message"] = message
	}

	task.UpdatedAt = time.Now()
	return nil
}

// StartAsyncTask registers a new task on this Manager and executes
// the provided function in a background goroutine.
//
// The task is created in StatusPending, then transitioned to
// StatusRunning when the goroutine starts, and finally to either
// StatusCompleted or StatusFailed when fn returns.
func (m *Manager) StartAsyncTask(ctx context.Context, fn func(context.Context) (any, error)) string {
	if fn == nil {
		// Defensive: avoid creating a task that can never complete.
		panic("tasks: StartAsyncTask called with nil function")
	}

	id := uuid.NewString()
	now := time.Now().UTC()

	m.mu.Lock()
	m.tasks[id] = &AsynchTask{
		ID:        id,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.mu.Unlock()

	// Launch the worker goroutine. We intentionally do not derive
	// a new context here; the caller is responsible for providing
	// an appropriate context (with timeout/cancellation) if needed.
	go m.runTask(ctx, id, fn)

	return id
}

// GetSnapshot returns a value copy of the task with the given ID.
// If no such task exists, the second return value is false.
func (m *Manager) GetSnapshot(id string) (Snapshot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	t, ok := m.tasks[id]
	if !ok {
		return Snapshot{}, false
	}
	return snapshotFromAsynchTask(t), true
}

// runTask is the internal worker that transitions a task through
// its lifecycle and captures the result of fn.
//
// It is executed in a background goroutine and must be robust to
// panics and context cancellation.
func (m *Manager) runTask(ctx context.Context, id string, fn func(context.Context) (any, error)) {
	// Transition to running (no result yet)
	m.updateStatus(id, StatusRunning, "", nil)

	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in async task %s: %v", id, r)
			m.updateStatus(id, StatusFailed, err.Error(), nil)
		}
	}()

	// Execute user function
	result, err := fn(ctx)
	if err != nil {
		m.updateStatus(id, StatusFailed, err.Error(), nil)
		return
	}

	// Normalize final result into map[string]any so it fits Task.Result.
	var final map[string]any
	switch v := result.(type) {
	case nil:
		final = nil
	case map[string]any:
		final = v
	default:
		// Wrap non-map results under a well-known key.
		final = map[string]any{
			"result": v,
		}
	}

	// Final transition to completed
	m.updateStatus(id, StatusCompleted, "", final)
}

// updateStatus updates the status, error, and final result of a task.
// This is used only for terminal states (completed or failed).
//
// IMPORTANT:
//   - For progress updates, use ReportProgress instead.
//   - This replaces the entire Result map (final output).
func (m *Manager) updateStatus(id string, status Status, errMsg string, result map[string]any) {
	now := time.Now().UTC()

	m.mu.Lock()
	defer m.mu.Unlock()

	t, ok := m.tasks[id]
	if !ok {
		return
	}

	t.Status = status
	t.Error = errMsg

	// Replace the entire result map (final output)
	if result != nil {
		t.Result = result
	} else {
		t.Result = nil
	}

	t.UpdatedAt = now
}
