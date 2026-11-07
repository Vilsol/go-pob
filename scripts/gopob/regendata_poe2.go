package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// rdGeneratorsPoE2 is `regen-data --game poe2`'s table: PoB-PoE2's own loaded data tables, dumped by
// scripts/gopob/regendata_poe2.lua after Data.lua has run, over go-pob-data's 0.5 tables where a row
// has a gamedatadump dump.
var rdGeneratorsPoE2 = []rdGenerator{
	{
		name:     "poe2-bases",
		source:   rdGameDataOverlay,
		out:      "data/poe2/bases_generated.go",
		lua:      "scripts/gopob/regendata_poe2.lua",
		dump:     "poe2-item-bases",
		luaArgs:  []string{"bases"},
		emit:     "poe2-data-emit",
		emitArgs: []string{"bases"},
		needsPob: true,
		note:     "data.itemBases after Data.lua's load (src/Data/Bases/*.lua). Game data FROM go-pob-data: `gamedatadump poe2-item-bases` runs PoB-PoE2's Export/Scripts/bases.lua over 0.5's json tables, .it files and stat descriptions with the pin's Export/Bases templates, and regendata_poe2.lua loads its Data/Bases files in place of the pin's (GAMEDATA_DUMP)",
	},
	{
		name:     "poe2-runes",
		source:   rdGameDataOverlay,
		out:      "data/poe2/runes_generated.go",
		lua:      "scripts/gopob/regendata_poe2.lua",
		dump:     "poe2-mod-runes",
		luaArgs:  []string{"runes"},
		emit:     "poe2-data-emit",
		emitArgs: []string{"runes"},
		needsPob: true,
		note:     "data.itemMods.Runes (src/Data/ModRunes.lua): runes, soul cores, idols, abyssal eyes, by slot; tradeHashes dropped. Game data FROM go-pob-data: `gamedatadump poe2-mod-runes` runs PoB-PoE2's Export/Scripts/soulcores.lua over 0.5's soul core tables and stat descriptions with the pin's Export/Bases/soulcore.txt, equal to the pin's up to the order of each tradeHashes table, and regendata_poe2.lua loads it in place of the pin's (GAMEDATA_DUMP)",
	},
	{
		name:     "poe2-constants",
		source:   rdGameDataOverlay,
		out:      "data/poe2/constants_generated.go",
		lua:      "scripts/gopob/regendata_poe2.lua",
		dump:     "poe2-misc-data",
		luaArgs:  []string{"constants"},
		emit:     "poe2-data-emit",
		emitArgs: []string{"constants"},
		needsPob: true,
		note:     "data.misc (Data.lua's magic numbers, computed) and src/Data/Misc.lua's constant maps and monster level tables. Game data FROM go-pob-data: `gamedatadump poe2-misc-data` writes src/Data/Misc.lua (PoB-PoE2's Export/Scripts/miscdata.lua over 0.5's json tables and .ot files), byte-identical to the pin's, and regendata_poe2.lua loads it in place of the pin's (GAMEDATA_DUMP); data.misc stays Data.lua's own literals computed over it",
	},
	{
		name:     "poe2-gems",
		source:   rdGameDataOverlay,
		out:      "data/poe2/gems_generated.go",
		lua:      "scripts/gopob/regendata_poe2.lua",
		dump:     "poe2-gems",
		luaArgs:  []string{"gems"},
		emit:     "poe2-data-emit",
		emitArgs: []string{"gems"},
		needsPob: true,
		note:     "data.gems after setupGem (src/Data/Gems.lua), with each granted effect's name and support flag. Game data FROM go-pob-data: `gamedatadump poe2-gems` runs the Gems.lua half of PoB-PoE2's Export/Scripts/skills.lua over 0.5's skill tables with the pin's Export/Skills templates, byte-identical to the pin's, and regendata_poe2.lua loads it in place of the pin's (GAMEDATA_DUMP); the granted effects' names and support flags stay the pin's Data/Skills",
	},
	{
		name:     "poe2-legion",
		source:   rdGameData,
		out:      "data/poe2/legion_generated.go",
		dump:     "poe2-legion-passives",
		emit:     "poe2-data-emit",
		emitArgs: []string{"legion"},
		note:     "the nodes of src/Data/TimelessJewelData/LegionPassives.lua, which PassiveTree loads as tree.legion: id, name, stat lines and type flags, in the file's order. FROM go-pob-data: `gamedatadump poe2-legion-passives` ports PoB-PoE2's Export/Scripts/legionPassives.lua and Export/statdesc.lua over 0.5's AlternatePassiveSkills, Stats and passive_skill_stat_descriptions; every node and addition equals the pin's file (`mise run poe2-gamedata-check`)",
	},
	rdGeneratorPoE2ModCache,
	rdGeneratorPoE2Quests,
	rdGeneratorsPoE2Skills[0],
	rdGeneratorsPoE2Skills[1],
	rdGeneratorsPoE2Skills[2],
	rdGeneratorsPoE2Skills[3],
	rdGeneratorsPoE2Bosses[0],
	rdGeneratorsPoE2Bosses[1],
	rdGeneratorsPoE2Items[0],
	rdGeneratorsPoE2Items[1],
	rdGeneratorsPoE2Items[2],
	rdGeneratorsPoE2Items[3],
	rdGeneratorsPoE2Items[4],
}

const p2eUsage = "usage: poe2-data-emit <dump.json> <out.go> <bases|runes|constants|gems|legion|bosses|boss-skills|runeforging|item-db|item-mods|item-mods-exclusive|mod-scalability>"

func poe2DataEmit(args []string) int {
	if len(args) != 3 {
		return oeUsage(p2eUsage)
	}
	body, err := os.ReadFile(args[0]) //nolint:gosec // the dump path is the caller's argument
	if err != nil {
		return oeRefuse(fmt.Sprintf("%s is not readable: %s", args[0], errnoText(err)))
	}
	var out string
	switch args[2] {
	case "bases":
		out, err = p2eBases(body)
	case "runes":
		out, err = p2eRunes(body)
	case "constants":
		out, err = p2eConstants(body)
	case "gems":
		out, err = p2eGems(body)
	case "legion":
		out, err = p2eLegion(body)
	case "bosses":
		out, err = p2eBosses(body)
	case "boss-skills":
		out, err = p2eBossSkills(body)
	case "runeforging":
		out, err = p2eRuneforging(body)
	case "item-db":
		out, err = p2eItemDB(body)
	case "item-mods":
		out, err = p2eItemMods(body)
	case "item-mods-exclusive":
		out, err = p2eItemModsExclusive(body)
	case "mod-scalability":
		out, err = p2eModScalability(body)
	default:
		return oeUsage(p2eUsage)
	}
	if err != nil {
		return oeRefuse(fmt.Sprintf("%s: %s", args[0], err))
	}
	return oeWriteGo(args[1], out)
}

func p2eDecode(body []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("does not match the PoE2 schema (a field PoB-PoE2 added needs a column here): %w", err)
	}
	return nil
}

// p2ePresent refuses a present zero on a field PoB tests with `if x then`: the Go type reads absent
// as 0, which only stays lossless while no row carries a real 0 there.
func p2ePresent(row, field string, v *float64) (float64, error) {
	if v == nil {
		return 0, nil
	}
	if *v == 0 {
		return 0, fmt.Errorf("%s has %s = 0, which PoB tells apart from nil; give the field a presence flag", row, field)
	}
	return *v, nil
}

func p2eNonEmpty(row, field string, v *string) (string, error) {
	if v == nil {
		return "", nil
	}
	if *v == "" {
		return "", fmt.Errorf("%s has %s = \"\", which PoB tells apart from nil; give the field a presence flag", row, field)
	}
	return *v, nil
}

// p2eHeader names the table's own pipeline: a dump from go-pob-data's tables, PoB-PoE2's Data.lua loaded over
// one, or the pinned PoB-PoE2 alone.
func p2eHeader(b *strings.Builder, table, cite string) {
	var dump, lua string
	for _, g := range rdGeneratorsPoE2 {
		if g.name == "poe2-"+table {
			dump, lua = g.dump, g.lua
		}
	}
	emit := "`gopob poe2-data-emit " + table + "`"
	switch {
	case dump != "" && lua != "":
		b.WriteString("// Code generated by " + lua + " + " + emit + " from the pinned PoB-PoE2 loading `gamedatadump " + dump + "`'s go-pob-data tables. DO NOT EDIT.\n")
	case dump != "":
		b.WriteString("// Code generated by `gamedatadump " + dump + "` + " + emit + " from go-pob-data's tables. DO NOT EDIT.\n")
	default:
		b.WriteString("// Code generated by scripts/gopob/regendata_poe2.lua + " + emit + " from the pinned PoB-PoE2. DO NOT EDIT.\n")
	}
	b.WriteString("// Regenerate with `mise run data-regen --game poe2 -- --only poe2-" + table + "`.\n//\n")
	b.WriteString("//pob2:" + cite + "\n\npackage poe2\n\n")
}

func p2eFloat(v float64) string { return strconv.FormatFloat(v, 'g', -1, 64) }

func p2eStrings(v []string) string {
	if v == nil {
		return "nil"
	}
	parts := make([]string, len(v))
	for i, s := range v {
		parts[i] = strconv.Quote(s)
	}
	return "[]string{" + strings.Join(parts, ", ") + "}"
}

func p2eFloats(v []float64) string {
	if v == nil {
		return "nil"
	}
	parts := make([]string, len(v))
	for i, f := range v {
		parts[i] = p2eFloat(f)
	}
	return "[]float64{" + strings.Join(parts, ", ") + "}"
}

// p2eFields writes `Name: value,` for every non-zero field, in the order given.
type p2eFields struct{ parts []string }

func (f *p2eFields) str(name, v string) {
	if v != "" {
		f.parts = append(f.parts, name+": "+strconv.Quote(v))
	}
}

func (f *p2eFields) num(name string, v float64) {
	if v != 0 {
		f.parts = append(f.parts, name+": "+p2eFloat(v))
	}
}

func (f *p2eFields) flag(name string, v bool) {
	if v {
		f.parts = append(f.parts, name+": true")
	}
}

func (f *p2eFields) raw(name, v string) {
	if v != "nil" && v != "" {
		f.parts = append(f.parts, name+": "+v)
	}
}

func (f *p2eFields) String() string { return "{" + strings.Join(f.parts, ", ") + "}" }

type p2eBaseJSON struct {
	Name                           string     `json:"name"`
	Type                           string     `json:"type"`
	SubType                        *string    `json:"subType"`
	Hidden                         bool       `json:"hidden"`
	SocketLimit                    float64    `json:"socketLimit"`
	Quality                        *float64   `json:"quality"`
	Spirit                         float64    `json:"spirit"`
	CharmLimit                     *float64   `json:"charmLimit"`
	GrantedSkillsHaveNoReservation bool       `json:"grantedSkillsHaveNoReservation"`
	Implicit                       *string    `json:"implicit"`
	ImplicitModTypes               [][]string `json:"implicitModTypes"`
	Tags                           []string   `json:"tags"`
	VariantList                    []string   `json:"variantList"`
	Req                            *struct {
		Level *float64 `json:"level"`
		Str   float64  `json:"str"`
		Dex   float64  `json:"dex"`
		Int   float64  `json:"int"`
	} `json:"req"`
	Armour *struct {
		Armour          float64  `json:"Armour"`
		Evasion         float64  `json:"Evasion"`
		EnergyShield    float64  `json:"EnergyShield"`
		Ward            float64  `json:"Ward"`
		MovementPenalty *float64 `json:"MovementPenalty"`
		BlockChance     *float64 `json:"BlockChance"`
	} `json:"armour"`
	Weapon *struct {
		PhysicalMin    float64  `json:"PhysicalMin"`
		PhysicalMax    float64  `json:"PhysicalMax"`
		FireMin        float64  `json:"FireMin"`
		FireMax        float64  `json:"FireMax"`
		ColdMin        float64  `json:"ColdMin"`
		ColdMax        float64  `json:"ColdMax"`
		LightningMin   float64  `json:"LightningMin"`
		LightningMax   float64  `json:"LightningMax"`
		ChaosMin       float64  `json:"ChaosMin"`
		ChaosMax       float64  `json:"ChaosMax"`
		CritChanceBase float64  `json:"CritChanceBase"`
		AttackRateBase float64  `json:"AttackRateBase"`
		Range          float64  `json:"Range"`
		ReloadTimeBase *float64 `json:"ReloadTimeBase"`
	} `json:"weapon"`
	Flask *struct {
		Life        *float64 `json:"life"`
		Mana        *float64 `json:"mana"`
		Duration    float64  `json:"duration"`
		ChargesMax  float64  `json:"chargesMax"`
		ChargesUsed float64  `json:"chargesUsed"`
	} `json:"flask"`
	Charm *struct {
		Duration    float64  `json:"duration"`
		ChargesMax  float64  `json:"chargesMax"`
		ChargesUsed float64  `json:"chargesUsed"`
		Buff        []string `json:"buff"`
	} `json:"charm"`
}

func p2eBaseRow(r p2eBaseJSON) (string, error) {
	f := &p2eFields{}
	f.str("Name", r.Name)
	f.str("Type", r.Type)
	sub, err := p2eNonEmpty(r.Name, "subType", r.SubType)
	if err != nil {
		return "", err
	}
	f.str("SubType", sub)
	f.flag("Hidden", r.Hidden)
	f.num("SocketLimit", r.SocketLimit)
	q, err := p2ePresent(r.Name, "quality", r.Quality)
	if err != nil {
		return "", err
	}
	f.num("Quality", q)
	f.num("Spirit", r.Spirit)
	if r.CharmLimit != nil {
		f.num("CharmLimit", *r.CharmLimit)
		f.flag("HasCharmLimit", true)
	}
	f.flag("GrantedSkillsHaveNoReservation", r.GrantedSkillsHaveNoReservation)
	imp, err := p2eNonEmpty(r.Name, "implicit", r.Implicit)
	if err != nil {
		return "", err
	}
	f.str("Implicit", imp)
	if r.ImplicitModTypes != nil {
		groups := make([]string, len(r.ImplicitModTypes))
		for i, g := range r.ImplicitModTypes {
			groups[i] = p2eStrings(g)
		}
		f.raw("ImplicitModTypes", "[][]string{"+strings.Join(groups, ", ")+"}")
	}
	f.raw("Tags", p2eStrings(r.Tags))
	f.raw("VariantList", p2eStrings(r.VariantList))
	if r.Req != nil {
		lvl, err := p2ePresent(r.Name, "req.level", r.Req.Level)
		if err != nil {
			return "", err
		}
		req := &p2eFields{}
		req.num("Level", lvl)
		req.num("Str", r.Req.Str)
		req.num("Dex", r.Req.Dex)
		req.num("Int", r.Req.Int)
		if len(req.parts) > 0 {
			f.raw("Req", "BaseReq"+req.String())
		}
	}
	if a := r.Armour; a != nil {
		mp, err := p2ePresent(r.Name, "armour.MovementPenalty", a.MovementPenalty)
		if err != nil {
			return "", err
		}
		bc, err := p2ePresent(r.Name, "armour.BlockChance", a.BlockChance)
		if err != nil {
			return "", err
		}
		af := &p2eFields{}
		af.num("Armour", a.Armour)
		af.num("Evasion", a.Evasion)
		af.num("EnergyShield", a.EnergyShield)
		af.num("Ward", a.Ward)
		af.num("MovementPenalty", mp)
		af.num("BlockChance", bc)
		f.raw("Armour", "&BaseArmour"+af.String())
	}
	if w := r.Weapon; w != nil {
		rt, err := p2ePresent(r.Name, "weapon.ReloadTimeBase", w.ReloadTimeBase)
		if err != nil {
			return "", err
		}
		wf := &p2eFields{}
		for _, kv := range []struct {
			k string
			v float64
		}{{"PhysicalMin", w.PhysicalMin}, {"PhysicalMax", w.PhysicalMax}, {"FireMin", w.FireMin}, {"FireMax", w.FireMax},
			{"ColdMin", w.ColdMin}, {"ColdMax", w.ColdMax}, {"LightningMin", w.LightningMin}, {"LightningMax", w.LightningMax},
			{"ChaosMin", w.ChaosMin}, {"ChaosMax", w.ChaosMax}, {"CritChanceBase", w.CritChanceBase},
			{"AttackRateBase", w.AttackRateBase}, {"Range", w.Range}, {"ReloadTimeBase", rt}} {
			wf.num(kv.k, kv.v)
		}
		f.raw("Weapon", "&BaseWeapon"+wf.String())
	}
	if fl := r.Flask; fl != nil {
		life, err := p2ePresent(r.Name, "flask.life", fl.Life)
		if err != nil {
			return "", err
		}
		mana, err := p2ePresent(r.Name, "flask.mana", fl.Mana)
		if err != nil {
			return "", err
		}
		ff := &p2eFields{}
		ff.num("Life", life)
		ff.num("Mana", mana)
		ff.num("Duration", fl.Duration)
		ff.num("ChargesMax", fl.ChargesMax)
		ff.num("ChargesUsed", fl.ChargesUsed)
		f.raw("Flask", "&BaseFlask"+ff.String())
	}
	if c := r.Charm; c != nil {
		cf := &p2eFields{}
		cf.num("Duration", c.Duration)
		cf.num("ChargesMax", c.ChargesMax)
		cf.num("ChargesUsed", c.ChargesUsed)
		cf.raw("Buff", p2eStrings(c.Buff))
		f.raw("Charm", "&BaseCharm"+cf.String())
	}
	return f.String(), nil
}

func p2eBases(body []byte) (string, error) {
	var rows []p2eBaseJSON
	if err := p2eDecode(body, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("holds no item bases")
	}
	if !sort.SliceIsSorted(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name }) {
		return "", fmt.Errorf("the bases are not sorted by name; BaseByName binary-searches them")
	}
	var b strings.Builder
	p2eHeader(&b, "bases", "src/Modules/Data.lua:1108-1113")
	fmt.Fprintf(&b, "// ItemBases is data.itemBases, %d bases sorted by name.\nvar ItemBases = []ItemBase{\n", len(rows))
	for _, r := range rows {
		row, err := p2eBaseRow(r)
		if err != nil {
			return "", err
		}
		b.WriteString("\t" + row + ",\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

type p2eRuneJSON struct {
	Rune      string    `json:"rune"`
	Slot      string    `json:"slot"`
	Type      string    `json:"type"`
	Lines     []string  `json:"lines"`
	StatOrder []float64 `json:"statOrder"`
	Bonded    *struct {
		Lines     []string  `json:"lines"`
		StatOrder []float64 `json:"statOrder"`
	} `json:"bonded"`
	Limit                          *float64 `json:"limit"`
	LimitID                        *string  `json:"limitId"`
	LocalMod                       bool     `json:"localMod"`
	LevelReq                       float64  `json:"levelReq"`
	CanSocketInUniqueItems         bool     `json:"canSocketInUniqueItems"`
	CanSocketInJewellery           bool     `json:"canSocketInJewellery"`
	CanSocketInCorruptedSanctified bool     `json:"canSocketInCorruptedSanctified"`
	CanSocketInChakraSlots         bool     `json:"canSocketInChakraSlots"`
	IsSocketBound                  bool     `json:"isSocketBound"`
}

func p2eRunes(body []byte) (string, error) {
	var rows []p2eRuneJSON
	if err := p2eDecode(body, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("holds no rune mods")
	}
	if !sort.SliceIsSorted(rows, func(i, j int) bool {
		return rows[i].Rune < rows[j].Rune || rows[i].Rune == rows[j].Rune && rows[i].Slot < rows[j].Slot
	}) {
		return "", fmt.Errorf("the rune mods are not sorted by rune and slot; RuneMods binary-searches them")
	}
	var b strings.Builder
	p2eHeader(&b, "runes", "src/Modules/Data.lua:699-699")
	fmt.Fprintf(&b, "// RuneMods is data.itemMods.Runes flattened to one row per rune and slot, %d rows sorted by both.\nvar RuneMods = []RuneMod{\n", len(rows))
	for _, r := range rows {
		id := r.Rune + "/" + r.Slot
		limit, err := p2ePresent(id, "limit", r.Limit)
		if err != nil {
			return "", err
		}
		limitID, err := p2eNonEmpty(id, "limitId", r.LimitID)
		if err != nil {
			return "", err
		}
		f := &p2eFields{}
		f.str("Rune", r.Rune)
		f.str("Slot", r.Slot)
		f.str("Type", r.Type)
		f.raw("Lines", p2eStrings(r.Lines))
		f.raw("StatOrder", p2eFloats(r.StatOrder))
		if r.Bonded != nil {
			f.raw("Bonded", p2eStrings(r.Bonded.Lines))
			f.raw("BondedStatOrder", p2eFloats(r.Bonded.StatOrder))
		}
		f.num("Limit", limit)
		f.str("LimitID", limitID)
		f.flag("LocalMod", r.LocalMod)
		f.num("LevelReq", r.LevelReq)
		f.flag("CanSocketInUniqueItems", r.CanSocketInUniqueItems)
		f.flag("CanSocketInJewellery", r.CanSocketInJewellery)
		f.flag("CanSocketInCorruptedSanctified", r.CanSocketInCorruptedSanctified)
		f.flag("CanSocketInChakraSlots", r.CanSocketInChakraSlots)
		f.flag("IsSocketBound", r.IsSocketBound)
		b.WriteString("\t" + f.String() + ",\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

var p2eConstantMaps = []struct{ key, goName, doc string }{
	{"misc", "Misc", "data.misc, Data.lua's magic numbers as PoB computes them"},
	{"gameConstants", "GameConstants", "data.gameConstants (src/Data/Misc.lua)"},
	{"characterConstants", "CharacterConstants", "data.characterConstants (src/Data/Misc.lua), its numeric entries"},
	{"monsterConstants", "MonsterConstants", "data.monsterConstants (src/Data/Misc.lua)"},
	{"playerMinionIntrinsicStats", "PlayerMinionIntrinsicStats", "data.playerMinionIntrinsicStats (src/Data/Misc.lua)"},
}

var p2eConstantTables = []struct{ key, goName string }{
	{"monsterEvasionTable", "MonsterEvasionTable"},
	{"monsterAccuracyTable", "MonsterAccuracyTable"},
	{"monsterLifeTable", "MonsterLifeTable"},
	{"monsterAllyLifeTable", "MonsterAllyLifeTable"},
	{"monsterDamageTable", "MonsterDamageTable"},
	{"monsterAllyDamageTable", "MonsterAllyDamageTable"},
	{"monsterArmourTable", "MonsterArmourTable"},
	{"monsterAilmentThresholdTable", "MonsterAilmentThresholdTable"},
	{"monsterPoiseThresholdTable", "MonsterPoiseThresholdTable"},
	{"minionLevelTable", "MinionLevelTable"},
}

var p2eConstantPairTables = []struct{ key, goName, doc string }{
	{"hollowPalmAddedPhys", "HollowPalmAddedPhys", "{ min, max } added physical damage"},
}

func p2eConstants(body []byte) (string, error) {
	var dump map[string]json.RawMessage
	if err := json.Unmarshal(body, &dump); err != nil {
		return "", fmt.Errorf("is not a JSON object: %w", err)
	}
	if known := len(p2eConstantMaps) + len(p2eConstantTables) + len(p2eConstantPairTables) + 1; len(dump) != known {
		return "", fmt.Errorf("holds %d tables, the emitter knows %d", len(dump), known)
	}
	var b strings.Builder
	p2eHeader(&b, "constants", "src/Modules/Data.lua:112-118")
	for _, m := range p2eConstantMaps {
		var vals map[string]any
		if err := json.Unmarshal(dump[m.key], &vals); err != nil || len(vals) == 0 {
			return "", fmt.Errorf("%s is not a non-empty JSON object", m.key)
		}
		var nums, flags []string
		for _, k := range insSortedKeys(rdAnyKeys(vals)) {
			switch v := vals[k].(type) {
			case float64:
				nums = append(nums, "\t"+strconv.Quote(k)+": "+p2eFloat(v)+",\n")
			case bool:
				flags = append(flags, "\t"+strconv.Quote(k)+": "+strconv.FormatBool(v)+",\n")
			default:
				return "", fmt.Errorf("%s.%s is a %T, not a number or a boolean", m.key, k, v)
			}
		}
		fmt.Fprintf(&b, "// %s is %s.\nvar %s = map[string]float64{\n%s}\n\n", m.goName, m.doc, m.goName, strings.Join(nums, ""))
		if len(flags) > 0 {
			fmt.Fprintf(&b, "// %sFlags is the boolean entries of %s.\nvar %sFlags = map[string]bool{\n%s}\n\n", m.goName, m.key, m.goName, strings.Join(flags, ""))
		}
	}
	for _, t := range p2eConstantTables {
		var vals []float64
		if err := json.Unmarshal(dump[t.key], &vals); err != nil || len(vals) == 0 {
			return "", fmt.Errorf("%s is not a non-empty array of numbers", t.key)
		}
		fmt.Fprintf(&b, "// %s is data.%s, indexed by level - 1.\nvar %s = %s\n\n", t.goName, t.key, t.goName, p2eFloats(vals))
	}
	for _, t := range p2eConstantPairTables {
		var vals [][]float64
		if err := json.Unmarshal(dump[t.key], &vals); err != nil || len(vals) == 0 {
			return "", fmt.Errorf("%s is not a non-empty array of number pairs", t.key)
		}
		rows := make([]string, len(vals))
		for i, v := range vals {
			if len(v) != 2 {
				return "", fmt.Errorf("%s[%d] holds %d numbers, not 2", t.key, i+1, len(v))
			}
			rows[i] = "{" + p2eFloat(v[0]) + ", " + p2eFloat(v[1]) + "}"
		}
		fmt.Fprintf(&b, "// %s is data.%s, %s indexed by level - 1.\nvar %s = [][2]float64{%s}\n\n", t.goName, t.key, t.doc, t.goName, strings.Join(rows, ", "))
	}
	var totemMult map[string]float64
	if err := json.Unmarshal(dump["totemLifeMult"], &totemMult); err != nil || len(totemMult) == 0 {
		return "", fmt.Errorf("totemLifeMult is not a non-empty JSON object of numbers")
	}
	ids := make([]int, 0, len(totemMult))
	for k := range totemMult {
		id, err := strconv.Atoi(k)
		if err != nil {
			return "", fmt.Errorf("totemLifeMult key %q is not a skill totem id", k)
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	b.WriteString("// TotemLifeMult is data.totemLifeMult (src/Data/Misc.lua), the life multiplier keyed by skillTotemId.\nvar TotemLifeMult = map[int]float64{\n")
	for _, id := range ids {
		fmt.Fprintf(&b, "\t%d: %s,\n", id, p2eFloat(totemMult[strconv.Itoa(id)]))
	}
	b.WriteString("}\n")
	return strings.TrimRight(b.String(), "\n") + "\n", nil
}

func rdAnyKeys(m map[string]any) map[string]bool {
	out := make(map[string]bool, len(m))
	for k := range m {
		out[k] = true
	}
	return out
}

type p2eGemJSON struct {
	ID                         string    `json:"id"`
	Name                       string    `json:"name"`
	GameID                     string    `json:"gameId"`
	VariantID                  string    `json:"variantId"`
	BaseTypeName               string    `json:"baseTypeName"`
	GrantedEffectID            string    `json:"grantedEffectId"`
	GrantedEffectName          string    `json:"grantedEffectName"`
	Support                    bool      `json:"support"`
	AdditionalGrantedEffectID1 string    `json:"additionalGrantedEffectId1"`
	AdditionalGrantedEffectID2 string    `json:"additionalGrantedEffectId2"`
	AdditionalGrantedEffectID3 string    `json:"additionalGrantedEffectId3"`
	AdditionalStatSet1         string    `json:"additionalStatSet1"`
	AdditionalStatSet2         string    `json:"additionalStatSet2"`
	AdditionalStatSet3         string    `json:"additionalStatSet3"`
	GrantedEffectDisplayOrder  []float64 `json:"grantedEffectDisplayOrder"`
	SecondaryEffectName        string    `json:"secondaryEffectName"`
	GemType                    string    `json:"gemType"`
	GemFamily                  string    `json:"gemFamily"`
	TagString                  string    `json:"tagString"`
	Tags                       []string  `json:"tags"`
	WeaponRequirements         string    `json:"weaponRequirements"`
	ReqStr                     float64   `json:"reqStr"`
	ReqDex                     float64   `json:"reqDex"`
	ReqInt                     float64   `json:"reqInt"`
	Tier                       float64   `json:"Tier"`
	NaturalMaxLevel            float64   `json:"naturalMaxLevel"`
}

// p2eNumbered turns PoB's x1, x2, x3 fields into a list, refusing a gap: setupGem's while loop
// stops at the first missing index, so anything after a gap is never read.
func p2eNumbered(row, field string, vals ...string) ([]string, error) {
	var out []string
	for i, v := range vals {
		if v == "" {
			for _, rest := range vals[i:] {
				if rest != "" {
					return nil, fmt.Errorf("%s has a gap in %sN", row, field)
				}
			}
			break
		}
		out = append(out, v)
	}
	return out, nil
}

func p2eGems(body []byte) (string, error) {
	var rows []p2eGemJSON
	if err := p2eDecode(body, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("holds no gems")
	}
	if !sort.SliceIsSorted(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID }) {
		return "", fmt.Errorf("the gems are not sorted by id; GemByID binary-searches them")
	}
	var b strings.Builder
	p2eHeader(&b, "gems", "src/Modules/Data.lua:968-1051")
	fmt.Fprintf(&b, "// Gems is data.gems, %d gems sorted by id.\nvar Gems = []Gem{\n", len(rows))
	for _, r := range rows {
		effects, err := p2eNumbered(r.ID, "additionalGrantedEffectId", r.AdditionalGrantedEffectID1, r.AdditionalGrantedEffectID2, r.AdditionalGrantedEffectID3)
		if err != nil {
			return "", err
		}
		sets, err := p2eNumbered(r.ID, "additionalStatSet", r.AdditionalStatSet1, r.AdditionalStatSet2, r.AdditionalStatSet3)
		if err != nil {
			return "", err
		}
		f := &p2eFields{}
		f.str("ID", r.ID)
		f.str("Name", r.Name)
		f.str("GameID", r.GameID)
		f.str("VariantID", r.VariantID)
		f.str("BaseTypeName", r.BaseTypeName)
		f.str("GrantedEffectID", r.GrantedEffectID)
		f.str("GrantedEffectName", r.GrantedEffectName)
		f.flag("Support", r.Support)
		f.raw("AdditionalGrantedEffectIDs", p2eStrings(effects))
		f.raw("AdditionalStatSets", p2eStrings(sets))
		f.raw("GrantedEffectDisplayOrder", p2eFloats(r.GrantedEffectDisplayOrder))
		f.str("SecondaryEffectName", r.SecondaryEffectName)
		f.str("GemType", r.GemType)
		f.str("GemFamily", r.GemFamily)
		f.str("TagString", r.TagString)
		f.raw("Tags", p2eStrings(r.Tags))
		f.str("WeaponRequirements", r.WeaponRequirements)
		f.num("ReqStr", r.ReqStr)
		f.num("ReqDex", r.ReqDex)
		f.num("ReqInt", r.ReqInt)
		f.num("Tier", r.Tier)
		f.num("NaturalMaxLevel", r.NaturalMaxLevel)
		b.WriteString("\t" + f.String() + ",\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}

type p2eLegionJSON struct {
	ID  string   `json:"id"`
	Dn  string   `json:"dn"`
	Sd  []string `json:"sd"`
	Ks  bool     `json:"ks"`
	Not bool     `json:"not"`
	M   bool     `json:"m"`
}

func p2eLegion(body []byte) (string, error) {
	var rows []p2eLegionJSON
	if err := p2eDecode(body, &rows); err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("no legion nodes")
	}
	var b strings.Builder
	p2eHeader(&b, "legion", "src/Data/TimelessJewelData/LegionPassives.lua")
	b.WriteString("// LegionNodes is tree.legion.nodes, " + strconv.Itoa(len(rows)) + " nodes in PoB's order (legionNodes[i] is LegionNodes[i-1]).\n")
	b.WriteString("var LegionNodes = []LegionNode{\n")
	for i, r := range rows {
		if r.ID == "" {
			return "", fmt.Errorf("legion node %d has no id", i+1)
		}
		f := &p2eFields{}
		f.str("ID", r.ID)
		f.str("Dn", r.Dn)
		f.raw("Sd", p2eStrings(r.Sd))
		f.flag("Ks", r.Ks)
		f.flag("Not", r.Not)
		f.flag("M", r.M)
		b.WriteString("\t" + f.String() + ",\n")
	}
	b.WriteString("}\n")
	return b.String(), nil
}
