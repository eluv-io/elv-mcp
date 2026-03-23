package main

import (
	"net/http"

	elog "github.com/eluv-io/log-go"

	"github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/mcpserver"
	"github.com/qluvio/elv-mcp/types"
)

var log = elog.Get("/main")

func main() {
	cfg, err := types.LoadConfig()
	if err != nil {
		log.Fatal("failed to load config", err)
	}
	// Creating StateChannel token
	stateToken, err := auth.FetchStateChannel(cfg, "")
	if err != nil {
		log.Fatal("failed to fetch state token", err)
	}
	cfg.SCToken = stateToken
	log.Info("state token fetched", "token", stateToken)

	server := mcpserver.NewServer(cfg)
	mux := mcpserver.NewHTTPMux(server, cfg)

	addr := ":8080"
	log.Info("MCP server listening", "addr", "http://localhost"+addr+"/mcp")
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal("http server failed", err)
	}
}
