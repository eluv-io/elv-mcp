package main

import (
	"net/http"

	elog "github.com/eluv-io/log-go"

	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/mcpserver"
	_ "github.com/qluvio/elv-mcp/tasks/all"
)

var log = elog.Get("/main")

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("failed to load config", err)
	}

	server := mcpserver.NewServer(cfg)
	mux := mcpserver.NewHTTPMux(server, cfg)

	addr := ":8181"
	log.Info("MCP server listening", "addr", "http://localhost"+addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal("http server failed", err)
	}
}
