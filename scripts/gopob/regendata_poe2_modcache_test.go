package main

import (
	"strings"
	"testing"
)

func TestPoE2ModCacheEmit(t *testing.T) {
	out, err := p2mcEmit([]byte(`[{"line":"b","mods":null,"extra":"b "},{"line":"a","mods":[{"name":"Life","type":"BASE","value":5,"flags":0,"keywordFlags":0,"source":null,"tags":[]}],"extra":null}]`))
	if err != nil {
		t.Fatal(err)
	}
	a := strings.Index(out, `{Line: "a", Mods: `)
	b := strings.Index(out, `{Line: "b", Extra: "b ", HasExtra: true}`)
	if a < 0 || b < 0 || a > b {
		t.Fatalf("rows missing or unsorted:\n%s", out)
	}
	for _, tc := range []struct{ name, body, want string }{
		{"unknown field", `[{"line":"a","mods":null,"extra":null,"x":1}]`, "schema"},
		{"duplicate", `[{"line":"a","mods":null,"extra":null},{"line":"a","mods":null,"extra":null}]`, "twice"},
		{"empty", `[]`, "no rows"},
	} {
		if _, err := p2mcEmit([]byte(tc.body)); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want an error containing %q", tc.name, err, tc.want)
		}
	}
}
