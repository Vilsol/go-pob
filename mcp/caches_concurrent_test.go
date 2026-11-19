package mcp

import (
	"fmt"
	"testing"
)

// Servers share the package's tree index, gem list and warning cache; run under -race with this test alone so the
// lazily built caches start cold.
func TestConcurrentServersShareCaches(t *testing.T) {
	for i := range 4 {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			s := testServer(t)
			h := importFixture(t, s)
			call(t, s, "get_build", map[string]any{"handle": h, "section": "warnings"})
			call(t, s, "search_data", map[string]any{"kind": "node", "query": "maximum life", "handle": h})
			call(t, s, "search_data", map[string]any{"kind": "support", "query": "added", "max_level_req": 10})
		})
	}
}
