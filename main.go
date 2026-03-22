package main

import (
	"log"
	"net/http"

	"github.com/qluvio/elv-mcp-experiment/auth"
	"github.com/qluvio/elv-mcp-experiment/mcpserver"
	"github.com/qluvio/elv-mcp-experiment/types"
)

func main() {
	cfg, err := types.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	// Creating StateChannel token
	stateToken, err := auth.FetchStateChannel("")
	if err != nil {
		log.Fatalf("Failed to fetch state token: %v", err)
	}
	cfg.SCToken = stateToken
	log.Println("Token returned:", stateToken)

	server := mcpserver.NewServer(cfg)
	mux := mcpserver.NewHTTPMux(server, cfg)

	addr := ":8080"
	log.Printf("MCP server listening on http://localhost%s/mcp", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}

}
