package importproxy

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestLivePoE2: a PoB-PoE2 site link downloads a PathOfBuilding2 build; skips without IMPORT_LIVE or a network.
func TestLivePoE2(t *testing.T) {
	if os.Getenv("IMPORT_LIVE") == "" {
		t.Skip("IMPORT_LIVE unset")
	}
	srv := httptest.NewServer(Handler(Deps{Client: &http.Client{Timeout: 30 * time.Second}}))
	defer srv.Close()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL+"/api/build?site=PoeNinja2&id=abc", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode == http.StatusBadGateway && strings.HasSuffix(string(body), "is unreachable") {
		t.Skipf("offline: %s", body)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("%d %s", res.StatusCode, body)
	}
	raw, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(strings.TrimRight(strings.TrimSpace(string(body)), "="))
	if err != nil {
		t.Fatal(err)
	}
	z, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	xml, err := io.ReadAll(z)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(xml, []byte("<PathOfBuilding2>")) {
		t.Fatalf("not a PoE2 build: %.120s", xml)
	}
}
