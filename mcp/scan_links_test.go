package mcp

import (
	"fmt"
	"strings"
	"testing"
)

// TestScanLinkUp: a five-link rare body armour whose five-gem group fills it offers a sixth link with each support
// that can support the group, priced at the bench's link cost difference plus the support; applying a row links the
// sixth socket in the support's colour (blue for Arcane Surge) and adds the support to the group.
func TestScanLinkUp(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = bookOf(t, map[string]string{
		"Currency": `{"lines":[{"currencyTypeName":"Divine Orb","chaosEquivalent":150},{"currencyTypeName":"Orb of Fusing","chaosEquivalent":0.2}]}`,
		"SkillGem": `{"lines":[
			{"name":"Controlled Destruction Support","gemLevel":20,"gemQuality":20,"chaosValue":5,"listingCount":10},
			{"name":"Spell Echo Support","gemLevel":20,"gemQuality":20,"chaosValue":7,"listingCount":10},
			{"name":"Added Cold Damage Support","gemLevel":20,"gemQuality":20,"chaosValue":3,"listingCount":10},
			{"name":"Elemental Focus Support","gemLevel":20,"gemQuality":20,"chaosValue":4,"listingCount":10},
			{"name":"Arcane Surge Support","gemLevel":20,"gemQuality":0,"chaosValue":2,"listingCount":10},
			{"name":"Freezing Pulse","gemLevel":20,"gemQuality":20,"chaosValue":1,"listingCount":10}]}`,
	})
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	body := "Rarity: RARE\nDoom Shell\nAstral Plate\nItem Level: 84\nSockets: R-R-R-R-R\nImplicits: 1\n+12% to all Elemental Resistances\n+90 to maximum Life"
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_text", "slot": "Body Armour", "text": body})})
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "slot": "Body Armour", "gems": []string{"Freezing Pulse", "Controlled Destruction", "Spell Echo", "Added Cold Damage", "Elemental Focus"}, "level": 20, "quality": 20},
		map[string]any{"op": "set_main", "group": 1})})
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"links"}, "rank_by": "gain_per_div"})
	got := rows(res)
	if len(got) != 1 {
		t.Fatalf("link-up rows %v (empty: %v)", got, res["empty_parts"])
	}
	r := got[0]
	bench6, _ := benchLinkChaos(s.opts.Prices, 6)
	bench5, _ := benchLinkChaos(s.opts.Prices, 5)
	if r["id"] != "linkup|Body Armour|1|Arcane Surge" || r["adds"] != "Arcane Surge" || num(r, "gain") <= 0 || !near(num(r, "cost"), bench6-bench5+2) || !strings.Contains(r["name"].(string), "6L with Arcane Surge") {
		t.Fatalf("row %v, want Arcane Surge at %v", r, bench6-bench5+2)
	}
	applied := call(t, s, "scan_build", map[string]any{"handle": h, "apply": r["id"]})
	if applied["revision"] == nil {
		t.Fatalf("apply %v: %v", r["id"], applied)
	}
	g := (*groupsOf(mustBuild(t, s, h)))[0]
	if len(g.Gems) != 6 || g.Gems[5].NameSpec != "Arcane Surge" {
		t.Fatalf("applied group %v", g.Gems)
	}
	if txt := fmt.Sprint(call(t, s, "get_build", map[string]any{"handle": h, "section": "items"})); !strings.Contains(txt, "R-R-R-R-R-B") {
		t.Errorf("applied item is not six-linked with a blue socket: %v", txt)
	}
	if got := rows(call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"links"}})); len(got) != 0 {
		t.Errorf("a six-link offers another link: %v", got)
	}
}
