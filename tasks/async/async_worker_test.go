package tasks

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestAsynchWorker_MissingID(t *testing.T) {
	_, _, err := AsynchWorker(context.Background(), &mcp.CallToolRequest{}, TaskStatusArgs{})
	if err == nil {
		t.Fatalf("expected error for missing task_id")
	}
}

func TestAsynchWorker_UnknownID(t *testing.T) {
	_, _, err := AsynchWorker(context.Background(), &mcp.CallToolRequest{}, TaskStatusArgs{
		TaskID: "task_does_not_exist",
	})
	if err == nil {
		t.Fatalf("expected error for unknown task_id")
	}
}

func TestAsynchWorker_CompletedTask(t *testing.T) {
	id := StartAsyncTask(context.Background(), func(ctx context.Context) (any, error) {
		return map[string]any{"ok": true}, nil
	})

	snap := waitForStatus(t, id, StatusCompleted, 200*time.Millisecond)

	if snap.Status != StatusCompleted {
		t.Fatalf("expected complete snapshot")
	}

	_, out, err := AsynchWorker(context.Background(), &mcp.CallToolRequest{}, TaskStatusArgs{
		TaskID: id,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Status != StatusCompleted {
		t.Fatalf("expected completed, got %s", out.Status)
	}
	if out.Result == nil {
		t.Fatalf("expected result payload")
	}
}
