package tasks

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

// waitForStatus polls until the task reaches the expected status or times out.
func waitForStatus(t *testing.T, id string, expected Status, timeout time.Duration) Snapshot {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		snap, ok := GetSnapshot(id)
		log.Printf("waitForStatus: got snapshot %+v (ok=%v)", snap.Status, ok)
		if !ok {
			t.Fatalf("task %s not found", id)
		}
		if snap.Status == expected {
			return snap
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("task %s did not reach status %s in time", id, expected)
	return Snapshot{}
}

func TestManager_CompletesSuccessfully(t *testing.T) {
	id := StartAsyncTask(context.Background(), func(ctx context.Context) (any, error) {
		time.Sleep(20 * time.Millisecond)
		return map[string]any{"ok": true}, nil
	})

	snap := waitForStatus(t, id, StatusCompleted, 200*time.Millisecond)

	if snap.Result == nil {
		t.Fatalf("expected result payload")
	}
	if snap.Error != "" {
		t.Fatalf("expected no error, got: %s", snap.Error)
	}
}

func TestManager_Fails(t *testing.T) {
	id := StartAsyncTask(context.Background(), func(ctx context.Context) (any, error) {
		time.Sleep(10 * time.Millisecond)
		return nil, fmt.Errorf("boom")
	})

	snap := waitForStatus(t, id, StatusFailed, 200*time.Millisecond)

	if snap.Error == "" {
		t.Fatalf("expected user-facing error message")
	}
	if snap.Result != nil {
		t.Fatalf("expected no result on failure")
	}
}

func TestManager_ConcurrentTasks(t *testing.T) {
	ids := []string{
		StartAsyncTask(context.Background(), func(ctx context.Context) (any, error) {
			time.Sleep(30 * time.Millisecond)
			return "a", nil
		}),
		StartAsyncTask(context.Background(), func(ctx context.Context) (any, error) {
			time.Sleep(10 * time.Millisecond)
			return "b", nil
		}),
	}

	for _, id := range ids {
		snap := waitForStatus(t, id, StatusCompleted, 300*time.Millisecond)
		if snap.Result == nil {
			t.Fatalf("task %s missing result", id)
		}
	}
}

// existing waitForStatus helper assumed

func TestManager_TimestampsConsistent(t *testing.T) {
	id := StartAsyncTask(context.Background(), func(ctx context.Context) (any, error) {
		time.Sleep(20 * time.Millisecond)
		return "done", nil
	})

	snap := waitForStatus(t, id, StatusCompleted, 300*time.Millisecond)

	if snap.CreatedAt.IsZero() {
		t.Fatalf("expected CreatedAt to be set")
	}
	if snap.UpdatedAt.IsZero() {
		t.Fatalf("expected UpdatedAt to be set")
	}
	if snap.UpdatedAt.Before(snap.CreatedAt) {
		t.Fatalf("expected UpdatedAt >= CreatedAt, got CreatedAt=%v UpdatedAt=%v", snap.CreatedAt, snap.UpdatedAt)
	}
}

func TestManager_LongRunningTaskProgressUpdates(t *testing.T) {
	var id string

	id = StartAsyncTask(context.Background(), func(ctx context.Context) (any, error) {
		for i := 0; i < 4; i++ {
			if err := ReportProgress(id, i*25, "working"); err != nil {
				return nil, err
			}
			time.Sleep(30 * time.Millisecond)
		}
		return "done", nil
	})

	seen := make(map[time.Time]struct{})
	deadline := time.Now().Add(1 * time.Second)

	for time.Now().Before(deadline) {
		snap, ok := GetSnapshot(id)
		if !ok {
			t.Fatalf("task %s not found", id)
		}

		seen[snap.UpdatedAt] = struct{}{}

		if snap.Status == StatusCompleted {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if len(seen) < 3 {
		t.Fatalf("expected at least 3 distinct UpdatedAt values, got %d", len(seen))
	}

	snap := waitForStatus(t, id, StatusCompleted, 200*time.Millisecond)
	if snap.Result == nil {
		t.Fatalf("expected final result")
	}
}

func TestManager_FailingTaskLifecycle(t *testing.T) {
	var id string

	id = StartAsyncTask(context.Background(), func(ctx context.Context) (any, error) {
		if err := ReportProgress(id, 10, "starting"); err != nil {
			return nil, err
		}
		time.Sleep(30 * time.Millisecond)
		if err := ReportProgress(id, 50, "halfway"); err != nil {
			return nil, err
		}
		time.Sleep(30 * time.Millisecond)
		return nil, fmt.Errorf("simulated failure")
	})

	snap := waitForStatus(t, id, StatusFailed, 500*time.Millisecond)

	if snap.Status != StatusFailed {
		t.Fatalf("expected failed, got %s", snap.Status)
	}
	if snap.Error == "" {
		t.Fatalf("expected error message on failure")
	}
	if snap.Result != nil {
		t.Fatalf("expected no result on failure, got %#v", snap.Result)
	}
	if snap.CreatedAt.IsZero() || snap.UpdatedAt.IsZero() {
		t.Fatalf("expected timestamps to be set")
	}
	if snap.UpdatedAt.Before(snap.CreatedAt) {
		t.Fatalf("expected UpdatedAt >= CreatedAt, got CreatedAt=%v UpdatedAt=%v", snap.CreatedAt, snap.UpdatedAt)
	}
}

func TestManager_ProgressReporting(t *testing.T) {
	var id string

	// Start a task that reports progress several times
	id = StartAsyncTask(context.Background(), func(ctx context.Context) (any, error) {
		for i := 0; i <= 50; i += 25 {
			if err := ReportProgress(id, i, fmt.Sprintf("step %d", i)); err != nil {
				return nil, err
			}
			time.Sleep(40 * time.Millisecond)
		}

		// Final result
		return map[string]any{"ok": true}, nil
	})

	// Track distinct UpdatedAt timestamps while running
	seenUpdates := make(map[time.Time]struct{})

	deadline := time.Now().Add(1 * time.Second)

	var snap Snapshot
	for time.Now().Before(deadline) {
		s, ok := GetSnapshot(id)
		if !ok {
			t.Fatalf("task %s not found", id)
		}
		snap = s

		// While running, progress must be present
		if snap.Status == StatusRunning {
			if snap.Result == nil {
				t.Fatalf("expected progress result while running")
			}
			if _, ok := snap.Result["progress"]; !ok {
				t.Fatalf("expected progress field in result while running")
			}
			if _, ok := snap.Result["message"]; !ok {
				t.Fatalf("expected message field in result while running")
			}

			seenUpdates[snap.UpdatedAt] = struct{}{}
		}

		if snap.Status == StatusCompleted {
			break
		}

		time.Sleep(20 * time.Millisecond)
	}

	// Ensure task completed
	if snap.Status != StatusCompleted {
		t.Fatalf("task did not complete in time: %+v", snap)
	}

	// Ensure multiple progress updates occurred
	if len(seenUpdates) < 2 {
		t.Fatalf("expected at least 2 progress updates, got %d", len(seenUpdates))
	}

	// Final result must overwrite progress
	if snap.Result == nil {
		t.Fatalf("expected final result payload")
	}
	if _, ok := snap.Result["progress"]; ok {
		t.Fatalf("final result should not contain progress field")
	}
	if ok, _ := snap.Result["ok"].(bool); !ok {
		t.Fatalf("expected final result to contain ok=true, got %#v", snap.Result)
	}
}
