package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp-experiment/types"
)

// re-use one HTTP client (best practice)
var httpClient = &http.Client{
	Timeout: 45 * time.Second,
}

// searchClips calls the Eluvio Search API and decodes JSON into clipResponse.
func searchClips(ctx context.Context, urlStr string, authToken string) (*types.ClipResponse, *http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")

	// Auth header only if provided
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, fmt.Errorf("reading search API response body failed: %w", err)
	}
	bodyStr := string(body)
	if len(bodyStr) > 500 {
		bodyStr = bodyStr[:500] + "...(truncated)"
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp, fmt.Errorf("search API returned %s: %s", resp.Status, string(body))
	}

	// Handle empty or whitespace-only body: treat as "no results"
	if len(strings.TrimSpace(bodyStr)) == 0 {
		// Adjust this to match your real clipResponse type / zero value
		return &types.ClipResponse{}, resp, nil
	}

	var out types.ClipResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, resp, fmt.Errorf("decode error: %w, body=%s", err, bodyStr)
	}

	return &out, resp, nil
}

// toolError creates an MCP error result and logs the details.
// err can be nil if you only want to log a message.
func toolError(userMessage string, err error) (*mcp.CallToolResult, any, error) {
	if err != nil {
		log.Printf("[tool error] %s: %v", userMessage, err)
	} else {
		log.Printf("[tool error] %s", userMessage)
	}

	// For users, keep it reasonably high-level while still informative.
	text := userMessage
	if err != nil {
		text = fmt.Sprintf("%s: %v", userMessage, err)
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
		IsError: true,
	}, nil, nil
}

// recoverMiddleware prevents panics from crashing the server and returns HTTP 500 instead.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic in HTTP handler: %v", rec)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs basic request info for debugging.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("HTTP %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
