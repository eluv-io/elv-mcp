// Package version provides version information
package version

//go:generate bash -c "../scripts/generate-version-info.sh > version-info.go"

import "time"

var date string
var full string

func init() {
	// if you see compile errors here, run `go generate github.com/qluvio/elv-mcp-experiment/version`
	cd := time.Unix(commit_date, 0).UTC()
	date = cd.Format(time.RFC3339)
	if version != "" {
		full = version + " " + date
	} else if branch != "" {
		full = branch + "@" + revision + " " + date
	} else {
		full = `N/A - run 'go generate github.com/qluvio/elv-mcp-experiment/version'`
	}
}

func Version() string {
	return version
}

func Full() string {
	return full
}
