package data

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func treeServer(t *testing.T, fails int32, fail func(w http.ResponseWriter)) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if n.Add(1) <= fails {
			fail(w)
			return
		}
		_, _ = w.Write([]byte("tree"))
	}))
	t.Cleanup(srv.Close)
	old := treeFetchBackoff
	treeFetchBackoff = time.Millisecond
	t.Cleanup(func() { treeFetchBackoff = old })
	return srv, &n
}

// TestFetchTreeRetriesAResetAndA5xx: a dropped connection or a 502 is tried again; a 404 is not.
func TestFetchTreeRetriesAResetAndA5xx(t *testing.T) {
	srv, n := treeServer(t, 1, func(w http.ResponseWriter) {
		if conn, _, err := w.(http.Hijacker).Hijack(); err == nil {
			_ = conn.Close()
		}
	})
	if b, err := fetchTree(context.Background(), srv.URL); err != nil || string(b) != "tree" || n.Load() != 2 {
		t.Fatalf("reset: %q %v after %d requests", b, err, n.Load())
	}

	srv, n = treeServer(t, 2, func(w http.ResponseWriter) { w.WriteHeader(http.StatusBadGateway) })
	if b, err := fetchTree(context.Background(), srv.URL); err != nil || string(b) != "tree" || n.Load() != 3 {
		t.Fatalf("502: %q %v after %d requests", b, err, n.Load())
	}

	srv, n = treeServer(t, 100, func(w http.ResponseWriter) { w.WriteHeader(http.StatusNotFound) })
	if _, err := fetchTree(context.Background(), srv.URL); err == nil || n.Load() != 1 {
		t.Fatalf("404: %v after %d requests", err, n.Load())
	}
}
