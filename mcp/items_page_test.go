package mcp

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

func TestGetBuildItemsPagesEveryItem(t *testing.T) {
	for _, detail := range []bool{false, true} {
		pageItems(t, detail)
	}
}

func pageItems(t *testing.T, detail bool) {
	s := testServer(t)
	h := call(t, s, "import_build", map[string]any{"source": fixture("7.xml")})["handle"].(string)
	want := len(equippedItems(mustParse(t, s, h)))
	seen := map[string]int{}
	cursor := ""
	pages := 0
	sawUnequipped := false
	limit := capSection
	if detail {
		limit = capSectionDetail
	}
	first := call(t, s, "get_build", map[string]any{"handle": h, "section": "items", "detail": detail})
	if !detail && first["truncated"] != true {
		t.Fatalf("detail=%v: the fixture fits one page; it no longer tests paging", detail)
	}
	for ; ; pages++ {
		args := map[string]any{"handle": h, "section": "items", "detail": detail}
		if cursor != "" {
			args["cursor"] = cursor
		}
		res := assertSize(t, s, "get_build", args, limit)
		for _, it := range res["items"].([]any) {
			seen[it.(map[string]any)["slot"].(string)]++
		}
		if res["unequipped"] != nil {
			sawUnequipped = true
		}
		next, ok := res["next_cursor"].(string)
		if !ok {
			break
		}
		if res["truncated"] != true || res["omitted"].(float64) <= 0 {
			t.Fatalf("a truncated page does not say how many items it omitted: %v", keys(res))
		}
		cursor = next
	}
	if len(seen) != want || sawUnequipped != detail {
		t.Fatalf("saw %d of %d items over %d pages (unequipped listed: %v)", len(seen), want, pages+1, sawUnequipped)
	}
	for slot, n := range seen {
		if n != 1 {
			t.Fatalf("%s seen %d times", slot, n)
		}
	}
}

func mustParse(t *testing.T, s *Server, h string) *pob.PathOfBuilding {
	t.Helper()
	hd, e := s.get(h)
	if e != nil {
		t.Fatal(e.Message)
	}
	b, e := hd.parseRev(hd.cur)
	if e != nil {
		t.Fatal(e.Message)
	}
	return b
}
