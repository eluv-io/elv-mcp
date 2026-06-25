package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPGet_EmptyBodyOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// empty body, 200 OK
	}))
	defer srv.Close()

	body, resp, err := HTTPGet(context.Background(), srv.URL, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if len(body) != 0 {
		t.Fatalf("expected empty body, got %d bytes", len(body))
	}
}

func TestHTTPGet_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	}))
	defer srv.Close()

	_, _, err := HTTPGet(context.Background(), srv.URL, nil)
	if err == nil {
		t.Fatalf("expected error for non-2xx status")
	}
}
