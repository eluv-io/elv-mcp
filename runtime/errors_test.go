package runtime_test

import (
	"errors"
	"testing"

	"github.com/qluvio/elv-mcp/runtime"
)

func TestMCPError(t *testing.T) {
	err := errors.New("boom")

	res, payload, outErr := runtime.MCPError(err)

	if outErr != err {
		t.Fatalf("MCPError must return the same error")
	}

	if res == nil {
		t.Fatalf("CallToolResult must not be nil")
	}

	if !res.IsError {
		t.Fatalf("IsError must be true")
	}

	if payload != nil {
		t.Fatalf("payload must be nil on error")
	}
}
