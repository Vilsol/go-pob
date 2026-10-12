package calculator

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
)

// TestItemDisablerRowsParse pins the seven CanNotUseItem rows: ModParser.lua:3054-3055 (the Ritualist Bloodline pair) and :5205-5209 (Misc).
func TestItemDisablerRowsParse(t *testing.T) {
	type want struct {
		slotName string
		exclude  string
		slotNum  int
	}
	cases := map[string][]want{
		"Can't use Amulets":             {{slotName: "Amulet"}},
		"Can't use Belts":               {{slotName: "Belt"}},
		"Can't use Chest armour":        {{slotName: "Body Armour"}},
		"Can't use Helmets":             {{slotName: "Helmet"}},
		"Can't use Helmet":              {{slotName: "Helmet"}},
		"Can't use Flask in Fifth Slot": {{slotName: "Flask 5", exclude: "Tincture"}},
		"Can't use other Rings":         {{slotName: "Ring 2", slotNum: 1}, {slotName: "Ring 1", slotNum: 2}},
		"Uses both hand slots":          {{slotName: "Weapon 2", slotNum: 1}, {slotName: "Weapon 1", slotNum: 2}},
	}

	for line, wants := range cases {
		t.Run(line, func(t *testing.T) {
			entry := ParseMod(line, false)
			if strings.Trim(entry.Extra, " ") != "" {
				t.Fatalf("unparsed remainder %q", entry.Extra)
			}
			if len(entry.ModList) != len(wants) {
				t.Fatalf("got %d mods, want %d", len(entry.ModList), len(wants))
			}
			for i, w := range wants {
				m := entry.ModList[i]
				if m.Name() != "CanNotUseItem" {
					t.Fatalf("mod %d: name %q, want CanNotUseItem", i, m.Name())
				}
				if m.Type() != mod.TypeDisablerFlag {
					t.Fatalf("mod %d: type %q, want %q", i, m.Type(), mod.TypeDisablerFlag)
				}
				if v := m.Value(); v == nil || v.Float() != 1 {
					t.Fatalf("mod %d: value %v, want 1", i, m.Value())
				}
				var di *mod.DisablesItemTag
				var sn *mod.SlotNumberTag
				for _, tg := range m.Tags() {
					switch tt := tg.(type) {
					case *mod.DisablesItemTag:
						di = tt
					case *mod.SlotNumberTag:
						sn = tt
					}
				}
				if di == nil {
					t.Fatalf("mod %d: no DisablesItem tag", i)
				}
				if di.TagSlotName != w.slotName {
					t.Fatalf("mod %d: slotName %q, want %q", i, di.TagSlotName, w.slotName)
				}
				if di.TagExcludeItemType != w.exclude {
					t.Fatalf("mod %d: excludeItemType %q, want %q", i, di.TagExcludeItemType, w.exclude)
				}
				if w.slotNum == 0 {
					if sn != nil {
						t.Fatalf("mod %d: unexpected SlotNumber tag %d", i, sn.N)
					}
				} else if sn == nil || sn.N != w.slotNum {
					t.Fatalf("mod %d: SlotNumber %v, want %d", i, sn, w.slotNum)
				}
			}
		})
	}
}

// TestResolveDisablerChains pins CalcSetup.lua:879-904's alternation and its cycle guard.
func TestResolveDisablerChains(t *testing.T) {
	cases := []struct {
		name      string
		disablers map[string]string
		want      []string
	}{
		{
			name:      "single",
			disablers: map[string]string{"Tree:12354": "Amulet"},
			want:      []string{"Amulet"},
		},
		{
			name:      "three link chain disables every other",
			disablers: map[string]string{"Helmet": "Body Armour", "Body Armour": "Gloves", "Gloves": "Boots"},
			want:      []string{"Body Armour", "Boots"},
		},
		{
			name:      "two independent chains",
			disablers: map[string]string{"Tree:1": "Amulet", "Helmet": "Body Armour"},
			want:      []string{"Amulet", "Body Armour"},
		},
		{
			name:      "mutual pair kills one side",
			disablers: map[string]string{"Ring 1": "Ring 2", "Ring 2": "Ring 1"},
			want:      []string{"Ring 1"},
		},
		{
			name:      "three cycle terminates",
			disablers: map[string]string{"A": "B", "B": "C", "C": "A"},
			want:      []string{"A", "C"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			inverse := make(map[string]string, len(c.disablers))
			for disabler, slot := range c.disablers {
				inverse[slot] = disabler
			}
			got := resolveDisablerChains(c.disablers, inverse)
			var names []string
			for slot := range got {
				names = append(names, slot)
			}
			sort.Strings(names)
			if !reflect.DeepEqual(names, c.want) {
				t.Fatalf("disabled %v, want %v", names, c.want)
			}
		})
	}
}

// TestDisabledItemSlotsMatchPoB grades the end-to-end pass against PoB's own dumps.
func TestDisabledItemSlotsMatchPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	cases := map[string][]string{
		"2668-ladder-helias61_8746.Somersby_Grusha": {"Amulet"},
		"3319-ladder-Tomikpomik_4685.GLAVKABAN":     {"Amulet"},
		"6803-ladder-aotarraid_2633.MirrorOfFlame":  {"Amulet"},
		"7282-ladder-grandonbs_2408.XPENGEXTREME":   {"Body Armour", "Ring 1"},
		"0006-elementalist-storm-burst":             nil,
	}

	for build, want := range cases {
		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			if len(env.Player.ItemList) == 0 {
				t.Fatalf("REFUSING: %s equipped no items at all", build)
			}
			for _, slotName := range want {
				if item := env.Player.ItemList[slotName]; item != nil {
					t.Errorf("%s survived; PoB drops it", slotName)
				}
			}
			disabled := make(map[string]bool, len(want))
			for _, s := range want {
				disabled[s] = true
			}
			for _, slotName := range []string{"Amulet", "Helmet", "Body Armour", "Gloves", "Boots", "Belt", "Ring 1", "Ring 2", "Weapon 1"} {
				if disabled[slotName] {
					continue
				}
				if _, filled := equippedSlotIDs(t, build)[slotName]; !filled {
					continue
				}
				if env.Player.ItemList[slotName] == nil {
					t.Errorf("%s went missing and nothing disables it", slotName)
				}
			}
		})
	}
}

// TestItemDisablerRespectsIgnoreConfig covers CalcSetup.lua:841's gate, `ignoreItemDisablers`.
func TestItemDisablerRespectsIgnoreConfig(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	const buildName = "2668-ladder-helias61_8746.Somersby_Grusha"
	rawXML, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), buildName+".xml"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	build, err := builds.ParseBuild(rawXML)
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	on := true
	build.Config.Inputs = append(build.Config.Inputs, pob.Input{Name: "ignoreItemDisablers", Boolean: &on})

	env := BuildOutputWithCache(&Calculator{PoB: build}, OutputModeMain, corpusCacheForTest(t))
	if env.Player.ItemList["Amulet"] == nil {
		t.Fatal("the amulet is still disabled with ignoreItemDisablers set")
	}
}

func equippedSlotIDs(t *testing.T, build string) map[string]int {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(corpusDirForLoadedData(t), build+".xml"))
	if err != nil {
		t.Fatalf("REFUSING: %v", err)
	}
	text := string(raw)
	active := "1"
	if m := activeItemSetRe.FindStringSubmatch(text); m != nil && m[1] != "" {
		active = m[1]
	}
	set := itemSetRe.FindString(text)
	for _, candidate := range itemSetRe.FindAllString(text, -1) {
		if m := itemSetIDRe.FindStringSubmatch(candidate); m != nil && m[1] == active {
			set = candidate
			break
		}
	}
	out := map[string]int{}
	for _, elem := range slotElemRe.FindAllString(set, -1) {
		name := slotNameRe.FindStringSubmatch(elem)
		id := slotIDRe.FindStringSubmatch(elem)
		if name == nil || id == nil {
			continue
		}
		if n, _ := strconv.Atoi(id[1]); n != 0 {
			out[name[1]] = n
		}
	}
	return out
}

var (
	activeItemSetRe = regexp.MustCompile(`activeItemSet="([^"]*)"`)
	itemSetRe       = regexp.MustCompile(`(?s)<ItemSet [^>]*>.*?</ItemSet>`)
	itemSetIDRe     = regexp.MustCompile(`<ItemSet [^>]*id="([^"]+)"`)
	slotElemRe      = regexp.MustCompile(`<Slot [^>]*/>`)
	slotNameRe      = regexp.MustCompile(`name="([^"]+)"`)
	slotIDRe        = regexp.MustCompile(`itemId="(\d+)"`)
)
