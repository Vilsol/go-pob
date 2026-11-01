package poe2

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type spotKey struct{ kind, key, field string }

func spotGoSide() (map[spotKey]string, map[string]int) {
	m := map[spotKey]string{}
	num := func(kind, key, field string, v float64) {
		m[spotKey{kind, key, field}] = strconv.FormatFloat(v, 'g', -1, 64)
	}
	str := func(kind, key, field, v string) {
		if v != "" {
			m[spotKey{kind, key, field}] = v
		}
	}
	for _, b := range ItemBases {
		str("base", b.Name, "type", b.Type)
		str("base", b.Name, "subType", b.SubType)
		str("base", b.Name, "implicit", b.Implicit)
		num("base", b.Name, "quality", b.Quality)
		num("base", b.Name, "socketLimit", b.SocketLimit)
		num("base", b.Name, "req.level", b.Req.Level)
		num("base", b.Name, "req.str", b.Req.Str)
		num("base", b.Name, "req.dex", b.Req.Dex)
		num("base", b.Name, "req.int", b.Req.Int)
		if a := b.Armour; a != nil {
			for f, v := range map[string]float64{"Armour": a.Armour, "Evasion": a.Evasion, "EnergyShield": a.EnergyShield,
				"Ward": a.Ward, "BlockChance": a.BlockChance, "MovementPenalty": a.MovementPenalty} {
				num("base", b.Name, "armour."+f, v)
			}
		}
		if w := b.Weapon; w != nil {
			for f, v := range map[string]float64{"PhysicalMin": w.PhysicalMin, "PhysicalMax": w.PhysicalMax, "CritChanceBase": w.CritChanceBase,
				"AttackRateBase": w.AttackRateBase, "Range": w.Range, "ReloadTimeBase": w.ReloadTimeBase} {
				num("base", b.Name, "weapon."+f, v)
			}
		}
	}
	for _, r := range RuneMods {
		key := r.Rune + "/" + r.Slot
		str("rune", key, "type", r.Type)
		str("rune", key, "lines", strings.Join(r.Lines, "|"))
		num("rune", key, "levelReq", r.LevelReq)
		num("rune", key, "limit", r.Limit)
	}
	consts := 0
	for t, vals := range map[string]map[string]float64{"misc": Misc, "gameConstants": GameConstants,
		"characterConstants": CharacterConstants, "monsterConstants": MonsterConstants} {
		for k, v := range vals {
			consts++
			num("const", t, k, v)
		}
	}
	for k, v := range CharacterConstantsFlags {
		consts++
		m[spotKey{"const", "characterConstants", k}] = strconv.FormatBool(v)
	}
	for i, v := range MonsterLifeTable {
		consts++
		num("const", "monsterLifeTable", strconv.Itoa(i+1), v)
	}
	for id, v := range TotemLifeMult {
		consts++
		num("const", "totemLifeMult", strconv.Itoa(id), v)
	}
	for _, g := range Gems {
		str("gem", g.ID, "name", g.Name)
		str("gem", g.ID, "grantedEffectId", g.GrantedEffectID)
		m[spotKey{"gem", g.ID, "support"}] = strconv.FormatBool(g.Support)
		str("gem", g.ID, "tagString", g.TagString)
		num("gem", g.ID, "reqStr", g.ReqStr)
		num("gem", g.ID, "reqDex", g.ReqDex)
		num("gem", g.ID, "reqInt", g.ReqInt)
		num("gem", g.ID, "naturalMaxLevel", g.NaturalMaxLevel)
	}
	for i, l := range LegionNodes {
		key := strconv.Itoa(i + 1)
		str("legion", key, "id", l.ID)
		str("legion", key, "dn", l.Dn)
		str("legion", key, "sd", strings.Join(l.Sd, "|"))
		m[spotKey{"legion", key, "ks"}] = strconv.FormatBool(l.Ks)
		m[spotKey{"legion", key, "not"}] = strconv.FormatBool(l.Not)
		m[spotKey{"legion", key, "m"}] = strconv.FormatBool(l.M)
	}
	return m, map[string]int{"base": len(ItemBases), "rune": len(RuneMods), "const": consts, "gem": len(Gems), "legion": len(LegionNodes)}
}

// spotDiff compares PoB's rows against ours. A Lua nil prints no row and our absent field reads 0,
// "" or false, so absence on either side only disagrees with a non-zero value on the other.
func spotDiff(pob map[spotKey]string, pobCounts map[string]int, ours map[spotKey]string, ourCounts map[string]int) (int, []string) {
	agree, bad := 0, []string(nil)
	same := func(a, b string) bool {
		if a == b {
			return true
		}
		fa, ea := strconv.ParseFloat(a, 64)
		fb, eb := strconv.ParseFloat(b, 64)
		return ea == nil && eb == nil && fa == fb
	}
	for k, v := range pob {
		o, ok := ours[k]
		if !ok && (v == "" || v == "false" || same(v, "0")) || ok && same(v, o) {
			agree++
		} else {
			bad = append(bad, k.kind+" "+k.key+" "+k.field+": PoB "+v+", ours "+o)
		}
	}
	for k, v := range ours {
		if _, ok := pob[k]; !ok && !same(v, "0") && v != "false" {
			bad = append(bad, k.kind+" "+k.key+" "+k.field+": PoB nil, ours "+v)
		}
	}
	for kind, n := range pobCounts {
		if ourCounts[kind] != n {
			bad = append(bad, kind+" rows: PoB "+strconv.Itoa(n)+", ours "+strconv.Itoa(ourCounts[kind]))
		}
	}
	return agree, bad
}

func spotPoB(t *testing.T) (map[spotKey]string, map[string]int) {
	t.Helper()
	if os.Getenv("OPERATOR") != "true" {
		t.Skip("SKIP: runs the PoB-PoE2 pin under luajit; set OPERATOR=true")
	}
	common, err := exec.CommandContext(context.Background(), "git", "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		t.Skipf("SKIP: not in a git checkout: %v", err)
	}
	main := filepath.Dir(strings.TrimSpace(string(common)))
	pob := os.Getenv("POB2_DIR")
	if pob == "" {
		pob = filepath.Join(main, "..", "PathOfBuilding-PoE2")
	}
	luajit := filepath.Join(main, ".luajit-det", "bin", "luajit")
	for _, p := range []string{filepath.Join(pob, "src", "HeadlessWrapper.lua"), luajit} {
		if _, err := os.Stat(p); err != nil {
			t.Skipf("SKIP: %s is absent", p)
		}
	}
	probe, err := filepath.Abs(filepath.Join("testdata", "spot_check.lua"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(context.Background(), "setarch", "-R", luajit, probe)
	cmd.Dir = filepath.Join(pob, "src")
	cmd.Env = append(os.Environ(),
		"LUA_PATH="+pob+"/runtime/lua/?.lua;"+pob+"/runtime/lua/?/init.lua;"+pob+"/.luarocks/share/lua/5.1/?.lua;"+pob+"/.luarocks/share/lua/5.1/?/init.lua;;",
		"LUA_CPATH="+pob+"/.luarocks/lib/lua/5.1/?.so;;")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("the probe failed: %v\n%s", err, stderr.String())
	}
	rows, counts := map[spotKey]string{}, map[string]int{}
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if len(f) != 4 {
			continue
		}
		v := strings.ReplaceAll(f[3], `\n`, "\n")
		switch f[0] {
		case "count":
			n, err := strconv.Atoi(v)
			if err != nil {
				t.Fatalf("bad count row %q", sc.Text())
			}
			counts[f[1]] = n
		case "base", "rune", "const", "gem", "legion":
			rows[spotKey{f[0], f[1], f[2]}] = v
		}
	}
	if len(counts) != 5 || len(rows) == 0 {
		t.Fatalf("the probe printed %d count rows and %d value rows; it stopped early\n%s", len(counts), len(rows), stderr.String())
	}
	return rows, counts
}

func TestGeneratedTablesMatchPoBPoE2(t *testing.T) {
	pob, pobCounts := spotPoB(t)
	ours, ourCounts := spotGoSide()
	agree, bad := spotDiff(pob, pobCounts, ours, ourCounts)
	t.Logf("%d of %d PoB values agree; rows: %v", agree, len(pob), pobCounts)
	for _, b := range bad {
		t.Error(b)
	}

	// The control: the same differential must see a wrong value, a dropped row and an invented field.
	mut := map[spotKey]string{}
	for k, v := range ours {
		mut[k] = v
	}
	mutCounts := map[string]int{}
	for k, v := range ourCounts {
		mutCounts[k] = v
	}
	evasion := spotKey{"base", "Leather Vest", "armour.Evasion"}
	if _, ok := pob[evasion]; !ok {
		t.Fatalf("the control row %v is not in PoB's output", evasion)
	}
	mut[evasion] = "31"
	mutCounts["gem"]--
	mut[spotKey{"base", "Leather Vest", "armour.Armour"}] = "1"
	if _, bad := spotDiff(pob, pobCounts, mut, mutCounts); len(bad) != 3 {
		t.Errorf("the must-mismatch control reports %d disagreements, want 3: %v", len(bad), bad)
	}
}
