package mcp

import (
	"slices"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/calculator"
)

// shieldBuild is a Templar casting Fireball from a wand with Shield Charge socketed in a shield.
func shieldBuild(t *testing.T, s *Server) string {
	t.Helper()
	h := call(t, s, "new_build", map[string]any{"class": "Templar"})["handle"].(string)
	call(t, s, "edit_items", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "equip_unique", "name": "Void Battery", "slot": "Weapon 1"},
		map[string]any{"op": "equip_unique", "name": "Lioneye's Remorse", "slot": "Weapon 2"})})
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(
		map[string]any{"op": "add_group", "gems": []string{"Fireball"}, "slot": "Weapon 1"},
		map[string]any{"op": "add_group", "gems": []string{"Shield Charge"}, "slot": "Weapon 2"},
		map[string]any{"op": "set_main", "group": 1})})
	return h
}

// TestRankUniquesFlagsDisabledSkills: a wand in the off-hand disables Shield Charge ("This skill requires a Shield");
// such a change is flagged with the skill, and a wand that gains more than every shield still ranks after them.
func TestRankUniquesFlagsDisabledSkills(t *testing.T) {
	s := testServer(t)
	h := shieldBuild(t, s)
	hd, _ := s.get(h)
	b, _ := hd.parseRev(hd.cur)
	env, _ := hd.envAt(hd.cur)
	text, _ := calculator.ItemDBText(calculator.ItemDBUnique, "Shade of Solaris, Sage Wand")
	c := cloneBuild(b)
	calculator.EquipItem(c, text, "Weapon 2")
	if flags := realismFlags(b, c, env, calcEnv(c)); !slices.ContainsFunc(flags, func(f string) bool { return strings.HasPrefix(f, "disables: Shield Charge") }) {
		t.Fatalf("wand in the off-hand: %v", flags)
	}
	list := rows(call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "uniques", "slot": "Weapon 2", "stat": "MainDPS", "limit": 25}))
	wand := dryDelta(t, s, "edit_items", h, mainDPS, map[string]any{"op": "equip_unique", "name": "Shade of Solaris", "slot": "Weapon 2"})
	if len(list) == 0 || wand <= list[0]["gain"].(float64) {
		t.Fatalf("the wand gains %v, top row %v", wand, list)
	}
	for _, r := range list {
		if strings.Contains(r["name"].(string), "Wand") && !flagged(r, "disables: ", "Shield Charge") {
			t.Fatalf("wand row ranks clean: %v", r)
		}
	}
	if strings.Contains(list[0]["name"].(string), "Wand") {
		t.Fatalf("a wand ranks first: %v", list[0])
	}
}

// TestRankUniquesFlagsSocketsTheBuildAlreadyOverfills: with four gems in a three-socket shield, every three-socket
// replacement still holds too many; its row says so though the overflow is not new.
func TestRankUniquesFlagsSocketsTheBuildAlreadyOverfills(t *testing.T) {
	s := testServer(t)
	h := shieldBuild(t, s)
	call(t, s, "edit_skills", map[string]any{"handle": h, "ops": ops(map[string]any{"op": "set_gems", "group": 2, "gems": []string{"Shield Charge", "Faster Attacks", "Fortify", "Onslaught"}})})
	list := rows(call(t, s, "rank_candidates", map[string]any{"handle": h, "kind": "uniques", "slot": "Weapon 2", "stat": "MainDPS", "limit": 25}))
	if len(list) == 0 {
		t.Fatal("no rows")
	}
	for _, r := range list {
		if r["sockets"].(float64) < 4 && !flagged(r, "sockets: 4 gems in", "Weapon 2") {
			t.Fatalf("%v holds %v sockets for 4 gems: %v", r["name"], r["sockets"], r)
		}
	}
}
