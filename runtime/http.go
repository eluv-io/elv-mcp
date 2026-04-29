package runtime

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	elog "github.com/eluv-io/log-go"
)

var log = elog.Get("/http")

// shared HTTP client
var httpClient = &http.Client{
	Timeout: 45 * time.Second,
}

// HTTPGet performs a generic HTTP GET with optional headers.
// It returns the raw response body and the http.Response.
// Non-2xx responses are returned as errors including status and body.
func HTTPGet(ctx context.Context, urlStr string, headers map[string]string) ([]byte, *http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, fmt.Errorf("reading HTTP response body failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Avoid including the full response body in the error to prevent leaking sensitive data
		log.Error("HTTP error", resp.StatusCode, string(body))
		return body, resp, fmt.Errorf("HTTP %s: %s", resp.Status, "Check logs for details")
	}

	return body, resp, nil
}
