package mcp

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// gemsBuild is a new Templar whose main group holds Freezing Pulse and Controlled Destruction at level, quality.
func gemsBuild(t *testing.T, s *Server, level, quality int) string {
	t.Helper()
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Freezing Pulse", "Controlled Destruction"}}, map[string]any{"op": "set_main", "group": 1},
		map[string]any{"op": "set_gem_level", "group": 1, "gem": "Freezing Pulse", "level": level, "quality": quality},
		map[string]any{"op": "set_gem_level", "group": 1, "gem": "Controlled Destruction", "level": level, "quality": quality})})
	return h
}

func gemResult(res map[string]any, id string) map[string]any {
	for _, r := range res["results"].([]any) {
		if r.(map[string]any)["id"] == id {
			return r.(map[string]any)
		}
	}
	return nil
}

// TestScanGemsUpgrades: a 19/0 gem offers level 20, its corrupted 21 and quality 20 and 23, and no downgrade; the
// scan's rows each gain.
func TestScanGemsUpgrades(t *testing.T) {
	s := testServer(t)
	h := gemsBuild(t, s, 19, 0)
	tries := gemTries(1, 0, (*groupsOf(mustBuild(t, s, h)))[0].Gems[0], 100)
	ids := make([]string, 0, len(tries))
	for _, tr := range tries {
		ids = append(ids, tr.id)
	}
	for _, id := range []string{"gemlevel|1|1|20", "gemlevel|1|1|21", "gemquality|1|1|20", "gemquality|1|1|23"} {
		if !slices.Contains(ids, id) {
			t.Fatalf("%s not tried: %v", id, ids)
		}
	}
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"gems"}, "per_slot": 5})
	if len(res["results"].([]any)) == 0 {
		t.Fatal("no rows")
	}
	for _, r := range res["results"].([]any) {
		if r := r.(map[string]any); r["part"] != "gems" || num(r, "gain") <= 0 || r["downgrade"] != nil {
			t.Fatalf("row %v", r)
		}
	}
}

// TestScanGemsDowngrades: with downgrades, a 21/20 gem offers 20/20, kept though it loses DPS, priced as what it saves;
// a 20/23 gem offers 20/20; apply commits the row's level. Without downgrades, an upgrade scan lists none.
func TestScanGemsDowngrades(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = budgetBook(t, true)
	h := gemsBuild(t, s, 21, 20)
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"gems"}, "allow_unknown_price": true, "rank_by": "gain_per_div"})
	for _, r := range res["results"].([]any) {
		if r.(map[string]any)["downgrade"] != nil {
			t.Fatalf("downgrade in an upgrade scan: %v", r)
		}
	}
	res = call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"gems"}, "allow_unknown_price": true, "rank_by": "gain_per_div", "downgrades": true})
	r := gemResult(res, "gemlevel|1|2|20")
	if r == nil || r["downgrade"] != true || num(r, "gain") >= 0 || r["cost"] != -895.0 || !strings.Contains(r["name"].(string), "21/20 -> 20/20") {
		t.Fatalf("Controlled Destruction 21/20 -> 20/20: %v in %v", r, resultNames(res))
	}
	if gemResult(res, "gemquality|1|2|23") != nil {
		t.Fatal("21/20 offered quality 23: a gem is corrupted once")
	}
	saved := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"gems"}, "rank_by": "savings"})
	if r := gemResult(saved, "gemlevel|1|2|20"); r == nil || r["cost"] != -895.0 || num(r, "savings_per_pct") <= 0 {
		t.Fatalf("rank_by savings keeps the 21/20 -> 20/20 downgrade with its saving: %v in %v", r, resultNames(saved))
	}
	call(t, s, "scan_build", map[string]any{"handle": h, "apply": "gemlevel|1|2|20"})
	g := (*groupsOf(mustBuild(t, s, h)))[0].Gems[1]
	if g.Level != 20 || g.Quality != 20 {
		t.Fatalf("applied gem %d/%d", g.Level, g.Quality)
	}

	h = gemsBuild(t, s, 20, 23)
	res = call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"gems"}, "downgrades": true})
	if r := gemResult(res, "gemquality|1|1|20"); r == nil || r["downgrade"] != true {
		t.Fatalf("Freezing Pulse 20/23 -> 20/20: %v in %v", r, resultNames(res))
	}
}

func mustBuild(t *testing.T, s *Server, h string) *pob.PathOfBuilding {
	t.Helper()
	hh, e := s.get(h)
	if e != nil {
		t.Fatal(e)
	}
	b, e := hh.parseRev(hh.cur)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

// TestScanGemsDowngradeSurvivesWorstCase: with a combat condition set, rows rank by worst_gain, which drops what gains
// nothing; a downgrade stays.
func TestScanGemsDowngradeSurvivesWorstCase(t *testing.T) {
	s := testServer(t)
	h := gemsBuild(t, s, 21, 20)
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "buffOnslaught", "value": true})})
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"gems"}, "downgrades": true})
	if res["worst_case"] == nil {
		t.Fatalf("no worst case: %v", res)
	}
	if r := gemResult(res, "gemlevel|1|2|20"); r == nil || r["downgrade"] != true {
		t.Fatalf("Controlled Destruction 21/20 -> 20/20: %v in %v", r, resultNames(res))
	}
}

// TestScanGemsLevelAndQuality: a 19/0 gem also offers level and quality raised together, 20/20, the corrupted 21/20 and
// 20/23, never the illegal 21/23; apply commits both.
func TestScanGemsLevelAndQuality(t *testing.T) {
	s := testServer(t)
	h := gemsBuild(t, s, 19, 0)
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"gems"}, "per_slot": 5})
	for _, id := range []string{"gemlevel|1|1|20|20", "gemlevel|1|1|21|20", "gemlevel|1|1|20|23"} {
		if r := gemResult(res, id); r == nil || num(r, "gain") <= 0 {
			t.Fatalf("%s: %v in %v", id, r, resultNames(res))
		}
	}
	if gemResult(res, "gemlevel|1|1|21|23") != nil {
		t.Fatal("offered 21/23: a gem is corrupted once")
	}
	call(t, s, "scan_build", map[string]any{"handle": h, "apply": "gemlevel|1|2|21|20"})
	if g := (*groupsOf(mustBuild(t, s, h)))[0].Gems[1]; g.Level != 21 || g.Quality != 20 {
		t.Fatalf("applied gem %d/%d", g.Level, g.Quality)
	}
}

// TestScanGemsPerGemByGainPerDiv: under rank_by gain_per_div, per_slot keeps each gem's best rows, so a gem's dear
// upgrade is not cut for another gem's cheap ones in the same group.
func TestScanGemsPerGemByGainPerDiv(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = bookOf(t, map[string]string{
		"Currency": `{"lines":[{"currencyTypeName":"Divine Orb","chaosEquivalent":150}]}`,
		"SkillGem": `{"lines":[
			{"name":"Freezing Pulse","gemLevel":20,"gemQuality":20,"chaosValue":1,"listingCount":10},
			{"name":"Freezing Pulse","gemLevel":21,"gemQuality":20,"corrupted":true,"chaosValue":2,"listingCount":10},
			{"name":"Freezing Pulse","gemLevel":20,"gemQuality":23,"corrupted":true,"chaosValue":3,"listingCount":10},
			{"name":"Controlled Destruction Support","gemLevel":20,"gemQuality":20,"chaosValue":5,"listingCount":10},
			{"name":"Controlled Destruction Support","gemLevel":21,"gemQuality":20,"corrupted":true,"chaosValue":900,"listingCount":10}]}`,
	})
	h := gemsBuild(t, s, 20, 20)
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"gems"}, "rank_by": "gain_per_div"})
	if r := gemResult(res, "gemlevel|1|2|21"); r == nil || r["cost"] != 895.0 || r["slot"] != "group 1" {
		t.Fatalf("Controlled Destruction 20/20 -> 21/20 cut: %v in %v", r, resultNames(res))
	}
	if n := len(res["results"].([]any)); n != 4 {
		t.Fatalf("want two rows per gem, got %v", resultNames(res))
	}
}

// TestScanGemsLevelCap: a level 30 character is offered no gem level it cannot use; the top level offered is the
// highest whose requirement it meets.
func TestScanGemsLevelCap(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar", "level": 30})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "add_group", "gems": []string{"Freezing Pulse", "Controlled Destruction"}, "level": 5, "quality": 0}, map[string]any{"op": "set_main", "group": 1})})
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"gems"}, "per_slot": 5})
	top := 0
	for _, r := range res["results"].([]any) {
		parts := strings.Split(r.(map[string]any)["id"].(string), "|")
		if parts[0] != "gemlevel" || parts[2] != "1" {
			continue
		}
		lv, _ := strconv.Atoi(parts[3])
		if req := gemLevelReq("FreezingPulse", lv); req > 30 {
			t.Fatalf("level %d needs character level %d: %v", lv, req, resultNames(res))
		}
		top = max(top, lv)
	}
	if top <= 5 || gemLevelReq("FreezingPulse", top+1) <= 30 {
		t.Fatalf("top level offered %d, not the highest a level 30 character can use: %v", top, resultNames(res))
	}
}

// TestScanGemsIdleQuality: with downgrades, Blind's 20 quality (blind effect, which the build does not use) is offered
// back to 0, kept as a downgrade that loses nothing; Controlled Destruction's (spell damage) is not.
func TestScanGemsIdleQuality(t *testing.T) {
	s := testServer(t)
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Freezing Pulse", "Controlled Destruction", "Blind"}, "level": 20, "quality": 20},
		map[string]any{"op": "set_main", "group": 1})})
	res := call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"gems"}, "downgrades": true})
	r := gemResult(res, "gemquality|1|3|0")
	if r == nil || r["downgrade"] != true || num(r, "gain") != 0 {
		t.Fatalf("Blind 20/20 -> 20/0: %v in %v", r, resultNames(res))
	}
	if gemResult(res, "gemquality|1|2|0") != nil {
		t.Fatalf("Controlled Destruction's quality raises spell damage: %v", resultNames(res))
	}
	applied := call(t, s, "scan_build", map[string]any{"handle": h, "apply": "gemquality|1|3|0", "dry_run": true})
	if !strings.Contains(fmt.Sprint(applied["applied"]), "quality 0") {
		t.Fatalf("apply: %v", applied["applied"])
	}
}

// TestScanGemsByDefault: a scan without parts ranks gem levels and quality too.
func TestScanGemsByDefault(t *testing.T) {
	s := testServer(t)
	h := gemsBuild(t, s, 19, 0)
	res := call(t, s, "scan_build", map[string]any{"handle": h, "slots": []string{"Amulet"}})
	for _, r := range res["results"].([]any) {
		if r.(map[string]any)["part"] == "gems" {
			return
		}
	}
	t.Fatalf("no gems rows by default: %v", resultNames(res))
}
