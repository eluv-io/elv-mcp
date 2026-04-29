//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	async "github.com/qluvio/elv-mcp/tasks/async"
)

func TestTaskStatus_Integration(t *testing.T) {
	_ = loadIntegrationConfig(t) // kept for symmetry; not strictly needed here

	var id string

	id = async.StartAsyncTask(context.Background(), func(ctx context.Context) (any, error) {
		for i := 0; i <= 100; i += 25 {
			if err := async.ReportProgress(id, i, "integration progress"); err != nil {
				return nil, err
			}
			time.Sleep(40 * time.Millisecond)
		}
		return map[string]any{"ok": true}, nil
	})

	var snap *async.Snapshot

	for i := 0; i < 20; i++ {
		_, out, err := async.AsynchWorker(
			context.Background(),
			&mcp.CallToolRequest{},
			async.TaskStatusArgs{TaskID: id},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		snap = out

		if snap.Status == async.StatusRunning {
			if snap.Result == nil {
				t.Fatalf("expected progress result while running")
			}
			if _, ok := snap.Result["progress"]; !ok {
				t.Fatalf("expected progress field in result while running")
			}
		}

		if snap.Status == async.StatusCompleted {
			break
		}

		time.Sleep(30 * time.Millisecond)
	}

	if snap.Status != async.StatusCompleted {
		t.Fatalf("task did not complete in time: %+v", snap)
	}
	if snap.Result == nil {
		t.Fatalf("expected final result payload")
	}
	if ok, _ := snap.Result["ok"].(bool); !ok {
		t.Fatalf("expected final result to contain ok=true, got %#v", snap.Result)
	}
}
