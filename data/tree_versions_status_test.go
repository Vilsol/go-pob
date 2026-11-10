//go:build !js

package data

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"

	"github.com/Vilsol/go-pob/data/assetcache"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

type statusTransport struct {
	status int
	body   []byte
	calls  int
}

func (s *statusTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	s.calls++
	return &http.Response{
		StatusCode: s.status,
		Status:     fmt.Sprintf("%d %s", s.status, http.StatusText(s.status)),
		Body:       io.NopCloser(bytes.NewReader(s.body)),
		Request:    r,
	}, nil
}

func TestRawTreeRefusesAnErrorResponseAndDoesNotCacheIt(t *testing.T) {
	t.Setenv(storage.CacheDirEnv, t.TempDir())
	storage.InitializeStorage(nil, nil, nil, nil, nil, nil)
	t.Cleanup(func() { storage.InitializeStorage(nil, nil, nil, nil, nil, nil) })

	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			transport := &statusTransport{status: status, body: []byte("<html>error page</html>")}
			old := http.DefaultClient.Transport
			http.DefaultClient.Transport = transport
			t.Cleanup(func() { http.DefaultClient.Transport = old })

			v := &TreeVersionData{Display: fmt.Sprintf("status_%d", status)}
			key := assetcache.CacheKey(raw.TreeURL(v.Display))

			var recovered any
			func() {
				defer func() { recovered = recover() }()
				v.RawTree()
			}()

			if transport.calls == 0 {
				t.Fatal("RawTree did not reach the network")
			}
			if recovered == nil {
				t.Fatalf("RawTree accepted a %d response as tree data", status)
			}
			if !strings.Contains(fmt.Sprint(recovered), fmt.Sprint(status)) {
				t.Errorf("the error does not name the status %d: %v", status, recovered)
			}
			if storage.Get().ExistsInCache(key) {
				t.Errorf("the %d response body was cached as the tree", status)
			}
			if v.RawLoaded() {
				t.Error("RawLoaded reports a tree that did not load")
			}
		})
	}
}

func TestRawTreeCachesAGoodResponse(t *testing.T) {
	t.Setenv(storage.CacheDirEnv, t.TempDir())
	storage.InitializeStorage(nil, nil, nil, nil, nil, nil)
	t.Cleanup(func() { storage.InitializeStorage(nil, nil, nil, nil, nil, nil) })

	var body bytes.Buffer
	w := brotli.NewWriter(&body)
	_, _ = w.Write([]byte(`{"tree":"ok"}`))
	_ = w.Close()

	transport := &statusTransport{status: http.StatusOK, body: body.Bytes()}
	old := http.DefaultClient.Transport
	http.DefaultClient.Transport = transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })

	v := &TreeVersionData{Display: "status_200"}
	if got := string(v.RawTree()); got != `{"tree":"ok"}` {
		t.Fatalf("RawTree = %q", got)
	}
	if !storage.Get().ExistsInCache(assetcache.CacheKey(raw.TreeURL(v.Display))) {
		t.Error("a good response was not cached")
	}
}

func TestRawTreeRefusesATruncatedTreeAndDoesNotKeepIt(t *testing.T) {
	t.Setenv(storage.CacheDirEnv, t.TempDir())
	storage.InitializeStorage(nil, nil, nil, nil, nil, nil)
	t.Cleanup(func() { storage.InitializeStorage(nil, nil, nil, nil, nil, nil) })

	var body bytes.Buffer
	w := brotli.NewWriterLevel(&body, 0)
	_, _ = w.Write(bytes.Repeat([]byte(`{"tree":"partial"}`), 1<<14))
	_ = w.Close()
	truncated := body.Bytes()[:body.Len()/2]

	transport := &statusTransport{status: http.StatusOK, body: truncated}
	old := http.DefaultClient.Transport
	http.DefaultClient.Transport = transport
	t.Cleanup(func() { http.DefaultClient.Transport = old })

	v := &TreeVersionData{Display: "status_truncated"}
	for attempt := 1; attempt <= 2; attempt++ {
		var got []byte
		var recovered any
		func() {
			defer func() { recovered = recover() }()
			got = v.RawTree()
		}()
		if recovered == nil {
			t.Fatalf("attempt %d: RawTree returned %d bytes of a truncated tree", attempt, len(got))
		}
	}
	if transport.calls != 2 {
		t.Errorf("the second attempt did not refetch: %d calls", transport.calls)
	}
	if storage.Get().ExistsInCache(assetcache.CacheKey(raw.TreeURL(v.Display))) {
		t.Error("a truncated tree was cached")
	}
	if v.RawLoaded() {
		t.Error("RawLoaded reports a tree that did not load")
	}
}
