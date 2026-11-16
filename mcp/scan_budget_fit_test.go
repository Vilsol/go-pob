package mcp

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/mcp/prices"
	"github.com/Vilsol/go-pob/pob"
)

// fitBook prices the bench's links, the six supports at 21/20 corrupted dear and at 20/20 cheap.
func fitBook(t *testing.T) *prices.Book {
	lines := ""
	for _, g := range []string{"Controlled Destruction Support", "Spell Echo Support", "Added Cold Damage Support", "Elemental Focus Support", "Cold Penetration Support"} {
		lines += fmt.Sprintf(`{"name":%q,"gemLevel":21,"gemQuality":20,"corrupted":true,"chaosValue":100,"listingCount":10},{"name":%q,"gemLevel":20,"gemQuality":20,"chaosValue":5,"listingCount":10},`, g, g)
	}
	lines += `{"name":"Freezing Pulse","gemLevel":20,"gemQuality":20,"chaosValue":1,"listingCount":10}`
	return bookOf(t, map[string]string{
		"Currency": `{"lines":[{"currencyTypeName":"Divine Orb","chaosEquivalent":150},{"currencyTypeName":"Orb of Fusing","chaosEquivalent":0.2}]}`,
		"SkillGem": `{"lines":[` + lines + `]}`,
	})
}

var supportLevelRe = regexp.MustCompile(`(gemId="Metadata/Items/Gems/SupportGem[^"]*"[^>]*?) level="20"`)

// fitBuild is a Templar with a six-linked rare body armour holding Freezing Pulse and five corrupted 21/20 supports.
func fitBuild(t *testing.T, s *Server) string {
	t.Helper()
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	body := "Rarity: RARE\nDoom Shell\nSimple Robe\nItem Level: 84\nSockets: B-B-B-B-B-B\nImplicits: 0\n+4% to all Elemental Resistances\n+90 to maximum Life"
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "equip_text", "slot": "Body Armour", "text": body})})
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "slot": "Body Armour", "gems": []string{"Freezing Pulse", "Controlled Destruction", "Spell Echo", "Added Cold Damage", "Elemental Focus", "Cold Penetration"}, "level": 20, "quality": 20},
		map[string]any{"op": "set_main", "group": 1})})
	xml := call(t, s, "export_build", map[string]any{"handle": h, "format": "xml"})["xml"].(string)
	xml = supportLevelRe.ReplaceAllString(xml, `$1 level="21"`)
	return call(t, s, "import_build", map[string]any{"source": xml})["handle"].(string)
}

func fitPlanAt(t *testing.T, s *Server, h string, limit float64) map[string]any {
	t.Helper()
	return call(t, s, "scan_build", map[string]any{"handle": h, "parts": []string{"links", "gems"}, "per_slot": 5, "cost_budget": limit, "fit_budget": true})["fit_budget"].(map[string]any)
}

// gemOfPick is the group|gem a gem or links row id changes.
func gemOfPick(id string) string {
	p := strings.Split(id, "|")
	if p[0] == "links" {
		return p[2] + "|" + p[3]
	}
	return p[1] + "|" + p[2]
}

// TestFitBudgetSets: three gem downgrades in one group that lose under 1% each beat a five-link that loses a third of
// the DPS; no gem takes two changes; an unreachable cap answers can't fit with no picks.
func TestFitBudgetSets(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = fitBook(t)
	h := fitBuild(t, s)
	build := 802.0

	plan := fitPlanAt(t, s, h, build-250)
	picks, _ := plan["picks"].([]any)
	if plan["fits"] != true || len(picks) != 3 || plan["loss_pct"].(float64) > 3 {
		t.Fatalf("need 250: %v", plan)
	}
	for _, p := range picks {
		if !strings.HasPrefix(p.(string), "gemlevel|1|") {
			t.Fatalf("need 250 picked %v: %v", p, plan)
		}
	}

	plan = fitPlanAt(t, s, h, build-600)
	picks, _ = plan["picks"].([]any)
	seen := map[string]bool{}
	for _, p := range picks {
		g := gemOfPick(p.(string))
		if seen[g] {
			t.Fatalf("need 600: two changes to gem %s: %v", g, plan)
		}
		seen[g] = true
	}
	if plan["fits"] != true || plan["build_chaos_after"].(float64) > build-600 {
		t.Fatalf("need 600: %v", plan)
	}

	plan = fitPlanAt(t, s, h, 10)
	note, _ := plan["note"].(string)
	if picks, _ := plan["picks"].([]any); plan["fits"] != false || len(picks) != 0 || !strings.Contains(note, "can't fit") || plan["closest_chaos"] == nil {
		t.Fatalf("need 792: %v", plan)
	}
}

// fitCtx is scan_build's context for h under an absolute cap, as fit_budget runs it.
func fitCtx(t *testing.T, s *Server, h string, limit float64) (*scanCtx, *realism) {
	t.Helper()
	hd, _ := s.get(h)
	b, _ := hd.parseRev(hd.cur)
	env, _ := hd.envAt(hd.cur)
	ps, _ := powerStat(env, mainDPS)
	cb, e := s.newCostBudget(budgetIn{CostBudget: []byte(fmt.Sprint(limit)), FitBudget: true}, b, false)
	if e != nil {
		t.Fatal(e)
	}
	misc := calculator.GetMiscCalculator(b)
	sc := &scanCtx{b: b, env: env, misc: misc, key: mainDPS, ps: ps, base: statOf(env, mainDPS), room: headroom(env), roll: defaultRoll, cb: cb}
	return sc, newRealism(b, env, mainDPS, time.Now().Add(time.Minute))
}

// TestFitBudgetSetBigLoss: a pick that tips the set into big_loss (here the third, standing in for nine picks that
// together lose 40% EHP) is dropped and the set chosen again, and the skip is counted.
func TestFitBudgetSetBigLoss(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = fitBook(t)
	h := fitBuild(t, s)
	got := rows(call(t, s, "scan_build", map[string]any{"handle": h, "rank_by": "savings", "parts": []string{"links", "gems"}, "per_slot": 5}))
	sc, rz := fitCtx(t, s, h, 802-250)
	plan := sc.fitSets(got, rz, func(c *pob.PathOfBuilding, _ *calculator.Environment) (string, any) {
		if len(partsOf(c).gems) > 0 && countDowngraded(c) >= 3 {
			return "set_big_loss", nil
		}
		return "", nil
	})
	picks, _ := plan["picks"].([]any)
	if plan["fits"] != true || plan["skipped"].(map[string]int)["set_big_loss"] == 0 || len(picks) == 0 {
		t.Fatalf("plan %v", plan)
	}
	c := cloneBuild(sc.b)
	for _, p := range picks {
		ch, _, e := scanChange(c, p.(string))
		if e != nil {
			t.Fatal(e)
		}
		ch(c)
	}
	if countDowngraded(c) >= 3 {
		t.Fatalf("plan keeps a set the check refused: %v", plan)
	}
}

// countDowngraded counts the enabled support gems below level 21.
func countDowngraded(b *pob.PathOfBuilding) int {
	n := 0
	for _, g := range *groupsOf(b) {
		for _, gem := range g.Gems {
			if gem.Enabled && gemIsSupport(gem) && gem.Level < 21 {
				n++
			}
		}
	}
	return n
}

// TestFitCheckRefillOutsideTheSet: a set that replaces the only rare with open suffixes, leaving a deeper resistance
// gap, is refused, though that rare's suffixes would refill the gap on the current build; a set that keeps it passes.
func TestFitCheckRefillOutsideTheSet(t *testing.T) {
	offline(t)
	s := testServer(t)
	s.opts.Prices = fitBook(t)
	h := fitBuild(t, s)
	call(t, s, "edit_build", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_config", "var": "customMods", "value": "+131% to all Elemental Resistances"})})
	sc, _ := fitCtx(t, s, h, 1)
	text, _ := calculator.ItemDBText(calculator.ItemDBUnique, "Kaom's Heart")
	c := cloneBuild(sc.b)
	uniqueChange(text, "Body Armour")(c)
	after := calcEnv(c)
	gap := resistGap(sc.room, headroom(after))
	if len(gap) == 0 {
		t.Fatalf("no gap: room %v after %v", sc.room, headroom(after))
	}
	if r := refill(resistItems(sc.b, sc.roll.rng), sc.room, map[string]any{"resist_gap": gap}, sc.roll); fmt.Sprint(r) == "not from the open suffix slots" {
		t.Fatalf("control: the current build's open suffixes refill nothing: %v", r)
	}
	if why, _ := sc.fitCheck(c, after); why != "set_resist_gap" {
		t.Fatalf("fitCheck on the replaced body: %q", why)
	}
	keep := cloneBuild(sc.b)
	gemLevelChange(1, 1, 20, 20)(keep)
	if why, _ := sc.fitCheck(keep, calcEnv(keep)); why != "" {
		t.Fatalf("fitCheck on a downgrade: %q", why)
	}
}
