package runtime

import (
	elog "github.com/eluv-io/log-go"
)

// Log is the shared logger used by tasks and server code.
var Log = elog.Get("/runtime")
