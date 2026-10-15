package importproxy

import (
	"crypto/sha256"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestLive: the Go proxy against the real sites, as frontend/src/lib/import_proxy/live.test.ts runs the TS one; never in CI.
func TestLive(t *testing.T) {
	if os.Getenv("IMPORT_LIVE") == "" {
		t.Skip("IMPORT_LIVE unset")
	}
	srv := httptest.NewServer(Handler(Deps{Client: &http.Client{Timeout: 30 * time.Second}}))
	defer srv.Close()
	for _, path := range []string{
		"/api/build?site=POBBin&id=h-M3H3It2fDw",
		"/api/build?site=pastebin&id=DsS6Qsxa",
		"/api/build?site=Maxroll&id=m5ui4t0x",
		"/api/poe/characters?realm=pc&account=Vilsol%233184",
		"/api/poe/profile?realm=pc&account=Vilsol%233184",
	} {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		t.Logf("%d %s %d bytes sha256 %.12x %q", res.StatusCode, path, len(body), sha256.Sum256(body), trim(string(body[:min(len(body), 60)])))
	}
}
