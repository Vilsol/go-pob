package calculator2

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob2"
)

// poe2SkillModListFloors are the skills-off builds whose main skill's baseSkillModList agrees with
// PoB-PoE2's dump mod for mod (name, type, value, flags, keyword flags, tag types), per corpus.
var poe2SkillModListFloors = map[string]int{
	"skills-off": 8723,
	"skills-on":  8719,
}

type poe2SkillModListDumpMod struct {
	Value any      `json:"value"`
	Type  string   `json:"type"`
	Flags string   `json:"flags"`
	KW    string   `json:"kw"`
	Tags  []string `json:"tags"`
}

type poe2SkillModListDump struct {
	// BaseSkillModList is an empty JSON array when the list is empty.
	BaseSkillModList json.RawMessage `json:"baseSkillModList"`
	Meta             struct {
		MainSkill *string `json:"mainSkill"`
	} `json:"meta"`
}

var poe2SkillDataValueRE = regexp.MustCompile(`^\{key=([^/}]*)(?:/value=([^}]*))?\}$`)

//pob2:block:src/Data/Global.lua:270-306
var keywordFlagByName = map[string]mod.KeywordFlag{
	"Aura": KeywordFlagAura, "Curse": KeywordFlagCurse, "Warcry": KeywordFlagWarcry, "Movement": KeywordFlagMovement,
	"Physical": KeywordFlagPhysical, "Fire": KeywordFlagFire, "Cold": KeywordFlagCold, "Lightning": KeywordFlagLightning,
	"Chaos": KeywordFlagChaos, "Vaal": KeywordFlagVaal, "Bow": KeywordFlagBow, "Arrow": KeywordFlagArrow,
	"Trap": KeywordFlagTrap, "Mine": KeywordFlagMine, "Totem": KeywordFlagTotem, "Minion": KeywordFlagMinion,
	"Attack": KeywordFlagAttack, "Spell": KeywordFlagSpell, "Hit": KeywordFlagHit, "Ailment": KeywordFlagAilment,
	"Brand": KeywordFlagBrand, "Poison": KeywordFlagPoison, "Bleed": KeywordFlagBleed, "Ignite": KeywordFlagIgnite,
	"PhysicalDot": KeywordFlagPhysicalDot, "LightningDot": KeywordFlagLightningDot, "ColdDot": KeywordFlagColdDot,
	"FireDot": KeywordFlagFireDot, "ChaosDot": KeywordFlagChaosDot, "MatchAll": KeywordFlagMatchAll,
}

// formatPoE2Flags is modLib.formatFlags over ModFlag or KeywordFlag.
func formatPoE2Flags[F ~int | ~int64 | ~uint64](flags F, names map[string]F) string {
	if flags == 0 {
		return ""
	}
	var out []string
	for name, v := range names {
		if flags&v == v {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

func poe2ModValueKey(name string, v any) string {
	switch x := v.(type) {
	case float64:
		return strconv.FormatFloat(math.Round(x*1e6)/1e6, 'g', -1, 64)
	case bool:
		if x {
			return "1"
		}
		return "0"
	case string:
		if name == "SkillData" {
			if m := poe2SkillDataValueRE.FindStringSubmatch(x); m != nil {
				val := m[2]
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					val = strconv.FormatFloat(math.Round(f*1e6)/1e6, 'g', -1, 64)
				}
				switch val {
				case "true":
					val = "1"
				case "false", "":
					val = "0"
				}
				return m[1] + "=" + val
			}
		}
		return "list"
	}
	return "?"
}

func ourModKey(m mod.Mod) string {
	name := m.Name()
	var value any
	v := m.Value()
	switch {
	case m.Type() == mod.TypeFlag:
		value = v.Flag()
	case m.Type() == mod.TypeList:
		value = "list"
		if sd, ok := v.ValueList.(mod.SkillData); ok {
			s := sd.SetValue()
			switch x := s.(type) {
			case float64:
				value = fmt.Sprintf("{key=%s/value=%s}", sd.Key, strconv.FormatFloat(x, 'g', -1, 64))
			case string:
				value = fmt.Sprintf("{key=%s/value=%s}", sd.Key, x)
			}
		}
	default:
		value = v.Float()
	}
	tags := make([]string, 0, len(m.Tags()))
	for _, t := range m.Tags() {
		tags = append(tags, string(t.Type()))
	}
	return strings.Join([]string{name, string(m.Type()), poe2ModValueKey(name, value),
		formatPoE2Flags(m.Flags(), modFlagByName), formatPoE2Flags(m.KeywordFlags(), keywordFlagByName), strings.Join(tags, ",")}, "|")
}

func dumpModKey(name string, d poe2SkillModListDumpMod) string {
	flags, kw := d.Flags, d.KW
	if flags == "-" {
		flags = ""
	}
	if kw == "-" {
		kw = ""
	}
	value := d.Value
	if d.Type == "FLAG" {
		// A FLAG mod is read for truth; mergeLevelMod leaves the stat's number in it.
		b, isBool := value.(bool)
		value = !isBool || b
	}
	return strings.Join([]string{name, d.Type, poe2ModValueKey(name, value), flags, kw, strings.Join(d.Tags, ",")}, "|")
}

// TestCorpusPoE2SkillModListVsDumps compares, per build, the main skill's baseSkillModList (its own mods,
// not its parent's) with the one PoB-PoE2's dump records, as multisets of mod keys.
func TestCorpusPoE2SkillModListVsDumps(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("CORPUS=true reads the local PoE2 corpus")
	}
	oracleDir := filepath.Join("..", "testdata", "oracle-poe2-0.5")
	realOracle, err := filepath.EvalSymlinks(oracleDir)
	if err != nil {
		t.Skipf("no PoE2 oracle at %s", oracleDir)
	}
	root := filepath.Dir(realOracle)
	tree := loadTestTree(t)
	limit, _ := strconv.Atoi(os.Getenv("LIMIT"))
	for _, c := range []struct{ name, corpus, dumps string }{
		{"skills-off", "corpus-poe2-0.5-skillsoff", "dumps-skillsoff"},
		{"skills-on", "corpus-poe2-0.5", "dumps"},
	} {
		corpusDir := filepath.Join("..", "testdata", c.corpus)
		dumpDir := filepath.Join(root, c.dumps)
		entries, err := os.ReadDir(dumpDir)
		if err != nil {
			t.Skipf("no PoE2 dumps at %s", dumpDir)
		}
		var builds []string
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".json") {
				builds = append(builds, strings.TrimSuffix(e.Name(), ".json"))
			}
		}
		sort.Strings(builds)
		if limit > 0 && len(builds) > limit {
			builds = builds[:limit]
		}
		type result struct {
			err             error
			missing, extra  []string
			mainOK, matched bool
			wantN, gotN     int
		}
		results := make([]result, len(builds))
		work := make(chan int)
		var wg sync.WaitGroup
		for range poe2Workers() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range work {
					results[i] = func() result {
						var res result
						raw, err := os.ReadFile(filepath.Join(dumpDir, builds[i]+".json"))
						if err != nil {
							return result{err: err}
						}
						var d poe2SkillModListDump
						if err := json.Unmarshal(raw, &d); err != nil {
							return result{err: err}
						}
						xml, err := os.ReadFile(filepath.Join(corpusDir, builds[i]+".xml"))
						if err != nil {
							return result{err: err}
						}
						build, err := pob2.ParseBuild(xml)
						if err != nil {
							return result{err: err}
						}
						var env *Environment
						func() {
							defer func() {
								if r := recover(); r != nil {
									err = fmt.Errorf("panic: %v", r)
								}
							}()
							env, err = BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
						}()
						if err != nil {
							return result{err: err}
						}
						var dumped map[string][]poe2SkillModListDumpMod
						if len(d.BaseSkillModList) > 0 && d.BaseSkillModList[0] == '{' {
							if err := json.Unmarshal(d.BaseSkillModList, &dumped); err != nil {
								return result{err: err}
							}
						}
						want := map[string]int{}
						for name, mods := range dumped {
							for _, m := range mods {
								want[dumpModKey(name, m)]++
								res.wantN++
							}
						}
						got := map[string]int{}
						if m := env.Player.MainSkill; m != nil {
							res.mainOK = d.Meta.MainSkill != nil && m.ActiveEffect.GrantedEffect.Name == *d.Meta.MainSkill
							if m.BaseSkillModList != nil {
								for _, x := range m.BaseSkillModList.Mods() {
									got[ourModKey(x)]++
									res.gotN++
								}
							}
						}
						for k, n := range want {
							for range n - got[k] {
								res.missing = append(res.missing, k)
							}
						}
						for k, n := range got {
							for range n - want[k] {
								res.extra = append(res.extra, k)
							}
						}
						res.matched = len(res.missing) == 0 && len(res.extra) == 0
						return res
					}()
				}
			}()
		}
		for i := range builds {
			work <- i
		}
		close(work)
		wg.Wait()

		var seen, failed, matched, mainOK, matchedMainOK, wantN, okN int
		missByName, extraByName := map[string]int{}, map[string]int{}
		var lines []string
		for i, r := range results {
			if r.err != nil {
				failed++
				if failed <= 5 {
					t.Logf("%s %s: %v", c.name, builds[i], r.err)
				}
				continue
			}
			seen++
			wantN += r.wantN
			okN += r.wantN - len(r.missing)
			if r.mainOK {
				mainOK++
			}
			if r.matched {
				matched++
				if r.mainOK {
					matchedMainOK++
				}
				continue
			}
			for _, k := range r.missing {
				missByName[strings.SplitN(k, "|", 2)[0]]++
				lines = append(lines, builds[i]+"\tmissing\t"+k)
			}
			for _, k := range r.extra {
				extraByName[strings.SplitN(k, "|", 2)[0]]++
				lines = append(lines, builds[i]+"\textra\t"+k)
			}
		}
		t.Logf("=== %s: %d builds, %d failed; baseSkillModList agrees on %d / %d builds (%d / %d with the right main skill); %d / %d dumped mods found",
			c.name, len(builds), failed, matched, seen, matchedMainOK, mainOK, okN, wantN)
		for _, tally := range []struct {
			what string
			m    map[string]int
		}{{"missing", missByName}, {"extra", extraByName}} {
			type kv struct {
				k string
				n int
			}
			var rows []kv
			for k, n := range tally.m {
				rows = append(rows, kv{k, n})
			}
			sort.Slice(rows, func(a, b int) bool {
				if rows[a].n != rows[b].n {
					return rows[a].n > rows[b].n
				}
				return rows[a].k < rows[b].k
			})
			for _, r := range rows[:min(len(rows), 15)] {
				t.Logf("  %-7s %6d  %s", tally.what, r.n, r.k)
			}
		}
		if out := os.Getenv("WORKLIST"); out != "" {
			_ = os.MkdirAll(out, 0o750)
			_ = os.WriteFile(filepath.Join(out, "skillmodlist-"+c.name+".tsv"), []byte(strings.Join(lines, "\n")+"\n"), 0o600)
		}
		if limit == 0 && matched < poe2SkillModListFloors[c.name] {
			t.Errorf("%s: baseSkillModList agrees on %d builds, floor %d", c.name, matched, poe2SkillModListFloors[c.name])
		}
	}
}
