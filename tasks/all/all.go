package all

// Blank imports ensure that each task package's init() function runs,
// causing all tasks to self-register into the global registry.
import (
	elog "github.com/eluv-io/log-go"
	_ "github.com/qluvio/elv-mcp/tasks/async"
	_ "github.com/qluvio/elv-mcp/tasks/fabric"
	_ "github.com/qluvio/elv-mcp/tasks/taggers"
	_ "github.com/qluvio/elv-mcp/tasks/tagstore"
	// Add future task packages here
)

func init() {
	elog.Info("tasks/all: init() running — blank imports should trigger task init()")
}
