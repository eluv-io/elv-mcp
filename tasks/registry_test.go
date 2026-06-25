package tasks_test

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/mcpserver"
	"github.com/qluvio/elv-mcp/tasks"
	_ "github.com/qluvio/elv-mcp/tasks/all" // triggers auto-registration
)

// ────────────────────────────────────────────────────────────────────────────────
//
//	Test 1: Auto‑registration works
//
// ────────────────────────────────────────────────────────────────────────────────
//
// This test verifies that importing Tasks/all triggers all Task
// init() functions and that tasks.All() returns the full set.
func TestTaskAutoRegistration(t *testing.T) {
	got := tasks.All()

	if len(got) == 0 {
		t.Fatalf("expected at least one task, got none")
	}

	names := map[string]bool{}
	for _, p := range got {
		names[p.Name()] = true
	}

	// Assert known tasks exist
	if !names["search_clips"] {
		t.Errorf("search_clips task not registered")
	}
	if !names["refresh_clips"] {
		t.Errorf("refresh_clips task not registered")
	}
}

// ────────────────────────────────────────────────────────────────────────────────
//
//	Test 2: All tasks have valid metadata
//
// ────────────────────────────────────────────────────────────────────────────────
//
// Ensures each task declares:
//
//   - a non‑empty Name()
//   - a non‑empty Description()
//
// This protects documentation quality and prevents silent registration issues.
func TestAllTasksHaveMetadata(t *testing.T) {
	for _, p := range tasks.All() {
		if p.Name() == "" {
			t.Errorf("task %T has empty Name()", p)
		}
		if p.Description() == "" {
			t.Errorf("task %s has empty Description()", p.Name())
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────────
//
//	Test 3: Task names must be unique
//
// ────────────────────────────────────────────────────────────────────────────────
//
// Duplicate task names would cause tool registration collisions.
// This test ensures each task declares a unique Name().
func TestTaskNamesAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range tasks.All() {
		name := p.Name()
		if seen[name] {
			t.Errorf("duplicate task name detected: %s", name)
		}
		seen[name] = true
	}
}

// ────────────────────────────────────────────────────────────────────────────────
//
//	Test 4: RegisterAll calls Register() on all tasks
//
// ────────────────────────────────────────────────────────────────────────────────
//
// This test verifies the registry wiring in isolation:
//
//   - Wrap each real task in a tracking wrapper.
//   - Ensure RegisterAll(server) calls Register() on every one of them.
type trackingTask struct {
	tasks.Task
	called bool
}

func (tp *trackingTask) Register(server *mcp.Server, cfg *config.Config) {
	tp.called = true
	tp.Task.Register(server, cfg)
}

func TestRegisterAllCallsRegisterOnAllTasks(t *testing.T) {
	cfg := &config.Config{}
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)

	wrapped := []tasks.Task{}
	for _, p := range tasks.All() {
		wrapped = append(wrapped, &trackingTask{Task: p})
	}

	reg := mcpserver.NewToolRegistry(cfg, wrapped...) // adjust if needed
	reg.RegisterAll(server)

	for _, p := range wrapped {
		tp := p.(*trackingTask)
		if !tp.called {
			t.Errorf("task %s.Register was not called", tp.Name())
		}
	}
}

// ────────────────────────────────────────────────────────────────────────────────
//
//	Test 5: Task.Register does not panic
//
// ────────────────────────────────────────────────────────────────────────────────
//
// The MCP SDK does not expose a way to inspect installed tools, but we can
// still verify that calling Register() on each task does not panic.
func TestTaskRegisterDoesNotPanic(t *testing.T) {
	cfg := &config.Config{}

	for _, p := range tasks.All() {
		server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)

		t.Run(p.Name(), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("task %s.Register panicked: %v", p.Name(), r)
				}
			}()
			println("Tool", "name", p.Name(), "description", p.Description())
			p.Register(server, cfg)
		})
	}
}
