package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob-data/loader"
	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
)

//pob:block:src/Modules/Data.lua:52-67
func luaModBuilders() luaEnv {
	makeSkillMod := func(args []any) (any, error) {
		arg := func(i int) any {
			if i < len(args) {
				return args[i]
			}
			return nil
		}
		t := newLuaTable()
		t.set("name", arg(0))
		t.set("type", arg(1))
		t.set("value", arg(2))
		flags, keywordFlags := arg(3), arg(4)
		if flags == nil {
			flags = 0.0
		}
		if keywordFlags == nil {
			keywordFlags = 0.0
		}
		t.set("flags", flags)
		t.set("keywordFlags", keywordFlags)
		if len(args) > 5 {
			t.arr = append(t.arr, args[5:]...)
		}
		for len(t.arr) > 0 && t.arr[len(t.arr)-1] == nil {
			t.arr = t.arr[:len(t.arr)-1]
		}
		return t, nil
	}
	tail := func(args []any, from int) []any {
		if len(args) > from {
			return args[from:]
		}
		return nil
	}
	first := func(args []any) any {
		if len(args) > 0 {
			return args[0]
		}
		return nil
	}
	makeFlagMod := func(args []any) (any, error) {
		return makeSkillMod(append([]any{first(args), "FLAG", true, 0.0, 0.0}, tail(args, 1)...))
	}
	makeSkillDataMod := func(args []any) (any, error) {
		data := newLuaTable()
		data.set("key", first(args))
		if len(args) > 1 {
			data.set("value", args[1])
		}
		return makeSkillMod(append([]any{"SkillData", "LIST", data, 0.0, 0.0}, tail(args, 2)...))
	}
	bor := func(args []any) (any, error) {
		var v uint64
		for _, a := range args {
			f, ok := a.(float64)
			if !ok {
				return nil, fmt.Errorf("bit.bor of a non-number")
			}
			v |= uint64(f)
		}
		return float64(v), nil
	}
	return luaEnv{
		"mod":   luaFunc(makeSkillMod),
		"flag":  luaFunc(makeFlagMod),
		"skill": luaFunc(makeSkillDataMod),
		"bit":   map[string]any{"bor": luaFunc(bor)},
	}
}

var globalFlagRe = regexp.MustCompile(`^(ModFlag|KeywordFlag)\.(\w+)\s*=\s*0x([0-9A-Fa-f]+)`)

func globalFlags(pobDir string) (map[string]map[string]float64, error) {
	b, err := os.ReadFile(filepath.Join(pobDir, "src", "Data", "Global.lua")) //nolint:gosec // the pinned PoB checkout
	if err != nil {
		return nil, fmt.Errorf("Global.lua: %w", err)
	}
	out := map[string]map[string]float64{"ModFlag": {}, "KeywordFlag": {}}
	for _, line := range strings.Split(string(b), "\n") {
		if m := globalFlagRe.FindStringSubmatch(line); m != nil {
			v, err := strconv.ParseUint(m[3], 16, 64)
			if err != nil {
				return nil, fmt.Errorf("Global.lua %s.%s: %w", m[1], m[2], err)
			}
			out[m[1]][m[2]] = float64(v)
		}
	}
	_, skillTypes, ok := strings.Cut(string(b), "\nSkillType = ")
	if !ok {
		return nil, fmt.Errorf("Data/Global.lua defines no SkillType")
	}
	toks, err := luaLex(skillTypes)
	if err != nil {
		return nil, fmt.Errorf("Data/Global.lua SkillType: %w", err)
	}
	v, err := (&luaParser{toks: toks}).expr()
	if err != nil {
		return nil, fmt.Errorf("Data/Global.lua SkillType: %w", err)
	}
	out["SkillType"] = map[string]float64{}
	if t, ok := v.(*luaTable); ok {
		for k, x := range t.hash {
			if f, ok := x.(float64); ok {
				out["SkillType"][k] = f
			}
		}
	}
	if len(out["ModFlag"]) == 0 || len(out["KeywordFlag"]) == 0 || len(out["SkillType"]) == 0 {
		return nil, fmt.Errorf("Data/Global.lua defines no ModFlag, KeywordFlag or SkillType")
	}
	return out, nil
}

func luaFlagEnv(flags map[string]map[string]float64) luaEnv {
	env := luaModBuilders()
	for table, values := range flags {
		m := map[string]any{}
		for k, v := range values {
			m[k] = v
		}
		env[table] = m
	}
	return env
}

//pob:block:src/Export/Scripts/minions.lua:17-18
func loadSkillStatMap(pobDir string, env luaEnv) (*luaTable, error) {
	b, err := os.ReadFile(filepath.Join(pobDir, "src", "Data", "SkillStatMap.lua")) //nolint:gosec // the pinned PoB checkout
	if err != nil {
		return nil, fmt.Errorf("SkillStatMap.lua: %w", err)
	}
	toks, err := luaLex(string(b))
	if err != nil {
		return nil, fmt.Errorf("SkillStatMap.lua: %w", err)
	}
	p := &luaParser{toks: toks, env: env}
	for {
		for p.peek().kind != "eof" && !(p.peek().kind == "name" && p.peek().text == "return") {
			p.next()
		}
		p.next()
		if !(p.peek().kind == "name" && p.peek().text == "function") {
			break
		}
	}
	v, err := p.expr()
	if err != nil {
		return nil, fmt.Errorf("SkillStatMap.lua: %w", err)
	}
	t, ok := v.(*luaTable)
	if !ok || len(t.hash) == 0 {
		return nil, fmt.Errorf("SkillStatMap.lua returns no table")
	}
	return t, nil
}

//pob:block:src/Export/Scripts/minions.lua:76-93
var itemClassMap = map[string]string{
	"Claw":                     "Claw",
	"Dagger":                   "Dagger",
	"Wand":                     "Wand",
	"One Hand Sword":           "One Handed Sword",
	"Thrusting One Hand Sword": "One Handed Sword",
	"One Hand Axe":             "One Handed Axe",
	"One Hand Mace":            "One Handed Mace",
	"Bow":                      "Bow",
	"Fishing Rod":              "Fishing Rod",
	"Staff":                    "Staff",
	"Two Hand Sword":           "Two Handed Sword",
	"Two Hand Axe":             "Two Handed Axe",
	"Two Hand Mace":            "Two Handed Mace",
	"Shield":                   "Shield",
	"Sceptre":                  "One Handed Mace",
	"Unarmed":                  "None",
}

//pob:src/Export/Scripts/minions.lua:20-39
func luaTableToString(tbl *luaTable, pre string) string {
	s := "{ "
	names := make([]string, 0, len(tbl.hash))
	for name := range tbl.hash {
		names = append(names, name)
	}
	sort.Strings(names)
	for i, name := range names {
		switch v := tbl.hash[name].(type) {
		case *luaTable:
			s += luaTableToString(v, pre+name+".")
		default:
			if i+1 > 1 {
				s += ", "
			}
			s += pre + name + " = "
			switch v := v.(type) {
			case string:
				s += `"` + v + `"`
			case float64:
				s += luaNumStr(v)
			case bool:
				s += strconv.FormatBool(v)
			}
		}
	}
	return s + " }"
}

type otStat struct {
	id    string
	value float64
}

type minionExport struct {
	cache    loader.AssetCache
	statMap  *luaTable
	ot       map[string]string
	missing  []string
	refusals []string
}

var (
	otExtendsRe  = regexp.MustCompile(`extends "(.+)"`)
	otStatLineRe = regexp.MustCompile(`^(.*?)=(.+)$`)
	otSpaceRe    = regexp.MustCompile(`\s+`)
)

//pob:block:src/Export/Scripts/minions.lua:41-74
func (e *minionExport) getOTStats(otFile string, modList []any) []any {
	file := otFile + ".ot"
	text, ok := e.ot[file]
	if !ok {
		b, err := loader.LoadFile(gameDataContext(), raw.LatestVersion, file, e.cache)
		if err != nil {
			e.missing = append(e.missing, file)
			return modList
		}
		text = string(b)
		e.ot[file] = text
	}
	inWantedBlock := false
	for _, line := range strings.FieldsFunc(text, func(r rune) bool { return r == '\r' || r == '\n' }) {
		if m := otExtendsRe.FindStringSubmatch(line); m != nil && m[1] != "Metadata/Monsters/Monster" && m[1] != "nothing" {
			modList = e.getOTStats(m[1], modList)
		}
		switch {
		case strings.HasPrefix(line, "Stats"):
			inWantedBlock = true
		case inWantedBlock && strings.HasPrefix(line, "}"):
			inWantedBlock = false
		case inWantedBlock && strings.Contains(line, "=") && !strings.Contains(line, "//"):
			if m := otStatLineRe.FindStringSubmatch(otSpaceRe.ReplaceAllString(line, "")); m != nil {
				v, ok := luaToNumber(m[2])
				if !ok {
					e.refusals = append(e.refusals, fmt.Sprintf("%s: stat %s = %s is not a number", file, m[1], m[2]))
					continue
				}
				modList = append(modList, otStat{id: m[1], value: v})
			}
		}
	}
	return modList
}

func luaToNumber(s string) (float64, bool) {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		v, err := strconv.ParseUint(s[2:], 16, 64)
		return float64(v), err == nil
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

//pob:block:src/Export/Scripts/minions.lua:143-250
func (e *minionExport) emit(st *minionState, out *strings.Builder) {
	var variety *poe.MonsterVariety
	for _, v := range poe.MonsterVarieties {
		if v.ID == st.varietyID {
			variety = v
			break
		}
	}
	if variety == nil {
		fmt.Fprintf(os.Stderr, "Invalid Variety: %s\n", st.varietyID)
		return
	}
	typ := poe.MonsterTypes[variety.MonsterTypesKey]
	w := func(s string) { _, _ = out.WriteString(s) }
	num := luaNumStr
	w(`minions["` + st.name + `"] = {` + "\n")
	w("\tname = \"" + variety.Name + "\",\n")
	w("\tmonsterTags = { ")
	for _, tag := range variety.TagsKeys {
		w(`"` + poe.Tags[tag].ID + `", `)
	}
	w("},\n")
	if typ.BaseDamageIgnoresAttackSpeed {
		w("\tbaseDamageIgnoresAttackSpeed = true,\n")
	}
	w("\tlife = " + num(float64(variety.LifeMultiplier)/100) + ",\n")
	if typ.AltLife1 {
		w("\tlifeScaling = \"AltLife1\",\n")
	}
	if typ.AltLife2 {
		w("\tlifeScaling = \"AltLife2\",\n")
	}
	if typ.EnergyShield != 0 {
		w("\tenergyShield = " + num(0.4*float64(typ.EnergyShield)/100) + ",\n")
	}
	if typ.Armour != 0 {
		w("\tarmour = " + num(float64(typ.Armour)/100) + ",\n")
	}
	if typ.Evasion != 0 {
		w("\tevasion = " + num(float64(typ.Evasion)/100) + ",\n")
	}
	if typ.Resistances == nil {
		e.refusals = append(e.refusals, variety.ID+": MonsterTypes "+typ.ID+" has no Resistances row")
		return
	}
	res := poe.MonsterResistances[*typ.Resistances]
	w("\tfireResist = " + num(float64(res.FireMerciless)) + ",\n")
	w("\tcoldResist = " + num(float64(res.ColdMerciless)) + ",\n")
	w("\tlightningResist = " + num(float64(res.LightningMerciless)) + ",\n")
	w("\tchaosResist = " + num(float64(res.ChaosMerciless)) + ",\n")
	w("\tdamage = " + num(float64(variety.DamageMultiplier)/100) + ",\n")
	w("\tdamageSpread = " + num(float64(typ.DamageSpread)/100) + ",\n")
	w("\tattackTime = " + num(float64(variety.AttackSpeed)/1000) + ",\n")
	w("\tattackRange = " + num(float64(variety.MaximumAttackDistance)) + ",\n")
	w("\taccuracy = " + num(float64(typ.Accuracy)/100) + ",\n")
	for _, m := range variety.ModsKeys {
		switch poe.Mods[m].ID {
		case "MonsterSpeedAndDamageFixupSmall":
			w("\tdamageFixup = 0.11,\n")
		case "MonsterSpeedAndDamageFixupLarge":
			w("\tdamageFixup = 0.22,\n")
		case "MonsterSpeedAndDamageFixupComplete":
			w("\tdamageFixup = 0.33,\n")
		}
	}
	if k := variety.MainHandItemClassesKey; k != nil {
		if c, ok := itemClassMap[poe.ItemClasses[*k].ID]; ok {
			w("\tweaponType1 = \"" + c + "\",\n")
		}
	}
	if k := variety.OffHandItemClassesKey; k != nil {
		if c, ok := itemClassMap[poe.ItemClasses[*k].ID]; ok {
			w("\tweaponType2 = \"" + c + "\",\n")
		}
	}
	if st.limit != nil {
		w("\tlimit = \"" + *st.limit + "\",\n")
	}
	if st.hostile != nil {
		w("\thostile = " + *st.hostile + ",\n")
	}
	w("\tskillList = {\n")
	for _, g := range variety.GrantedEffectsKeys {
		w("\t\t\"" + poe.GrantedEffects[g].ID + "\",\n")
	}
	for _, s := range st.extraSkillList {
		w("\t\t\"" + s + "\",\n")
	}
	w("\t},\n")

	modList := make([]any, 0, len(variety.ModsKeys)+len(variety.SpecialModsKeys))
	for _, m := range variety.ModsKeys {
		modList = append(modList, poe.Mods[m])
	}
	for _, m := range variety.SpecialModsKeys {
		modList = append(modList, poe.Mods[m])
	}
	if variety.BaseMonsterTypeIndex != "" && variety.BaseMonsterTypeIndex != "Metadata/Monsters/Monster" {
		modList = e.getOTStats(variety.BaseMonsterTypeIndex, modList)
	}
	w("\tmodList = {\n")
	for _, m := range modList {
		var modID string
		var stats []otStat
		switch m := m.(type) {
		case *poe.Mod:
			modID = m.ID
			for _, s := range []struct {
				key *int
				min int
			}{{m.StatsKey1, m.Stat1Min}, {m.StatsKey2, m.Stat2Min}, {m.StatsKey3, m.Stat3Min},
				{m.StatsKey4, m.Stat4Min}, {m.StatsKey5, m.Stat5Min}, {m.StatsKey6, m.Stat6Min}} {
				if s.key != nil {
					stats = append(stats, otStat{id: poe.Stats[*s.key].ID, value: float64(s.min)})
				}
			}
		case otStat:
			modID = m.id
			stats = []otStat{m}
		}
		for _, s := range stats {
			modStats := " [" + s.id + " = " + num(s.value) + "]"
			entry, ok := e.statMap.hash[s.id].(*luaTable)
			if !ok {
				w("\t\t-- " + modID + modStats + "\n")
				continue
			}
			if len(entry.arr) == 0 {
				e.refusals = append(e.refusals, "SkillStatMap "+s.id+" has no mod")
				continue
			}
			newMod, _ := entry.arr[0].(*luaTable)
			if newMod == nil {
				e.refusals = append(e.refusals, "SkillStatMap "+s.id+"[1] is not a mod")
				continue
			}
			value := ""
			switch v := newMod.hash["value"].(type) {
			case nil, bool:
				if sv, ok := entry.hash["value"]; ok {
					f, ok := sv.(float64)
					if !ok {
						e.refusals = append(e.refusals, "SkillStatMap "+s.id+".value is not a number")
						continue
					}
					value = num(f)
				} else {
					mult, div := 1.0, 1.0
					if f, ok := entry.hash["mult"].(float64); ok {
						mult = f
					}
					if f, ok := entry.hash["div"].(float64); ok {
						div = f
					}
					value = num(s.value * mult / div)
				}
			case *luaTable:
				value = luaTableToString(v, "")
			default:
				e.refusals = append(e.refusals, "SkillStatMap "+s.id+"[1].value is neither a table nor a boolean")
				continue
			}
			flags, _ := newMod.hash["flags"].(float64)
			keywordFlags, _ := newMod.hash["keywordFlags"].(float64)
			w(fmt.Sprintf("\t\tmod(\"%s\", \"%s\", %s, %s, %s", newMod.hash["name"], newMod.hash["type"], value, num(flags), num(keywordFlags)))
			for _, extra := range newMod.arr {
				t, ok := extra.(*luaTable)
				if !ok {
					e.refusals = append(e.refusals, "SkillStatMap "+s.id+"[1] carries a tag that is not a table")
					continue
				}
				w(", " + luaTableToString(t, ""))
			}
			w("), -- " + modID + modStats + "\n")
		}
	}
	for _, m := range st.extraModList {
		w("\t\t" + m + ",\n")
	}
	w("\t},\n")
	w("}\n")
}

type minionState struct {
	varietyID, name string
	limit, hostile  *string
	extraModList    []string
	extraSkillList  []string
}

var templateLineRe = regexp.MustCompile(`#([A-Za-z]+) ?(.*)`)

//pob:block:src/Export/Main.lua:112-129
//pob:block:src/Export/Scripts/minions.lua:95-140
//pob:block:src/Export/Scripts/minions.lua:252-260
func (e *minionExport) processTemplate(pobDir, name string) (string, error) {
	b, err := os.ReadFile(filepath.Join(pobDir, "src", "Export", "Minions", name+".txt")) //nolint:gosec // the pinned PoB checkout
	if err != nil {
		return "", fmt.Errorf("template %s: %w", name, err)
	}
	var out strings.Builder
	out.WriteString("-- This file is automatically generated, do not edit!\n")
	st := &minionState{}
	monster := func(args string) {
		*st = minionState{}
		for _, arg := range strings.Fields(args) {
			switch {
			case st.varietyID == "":
				st.varietyID = arg
			case st.name == "":
				if arg == "#" {
					st.name = st.varietyID
				} else {
					st.name = arg
				}
			default:
				st.extraSkillList = append(st.extraSkillList, arg)
			}
		}
		if st.varietyID == "" {
			st.varietyID = args
		}
		if st.name == "" {
			st.name = args
		}
	}
	text := strings.TrimSuffix(string(b), "\n")
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSuffix(line, "\r")
		m := templateLineRe.FindStringSubmatch(line)
		if m == nil {
			out.WriteString(line + "\n")
			continue
		}
		args := m[2]
		switch m[1] {
		case "monster":
			monster(args)
		case "limit":
			st.limit = &args
		case "hostile":
			st.hostile = &args
		case "mod":
			st.extraModList = append(st.extraModList, args)
		case "skill":
			st.extraSkillList = append(st.extraSkillList, args)
		case "emit":
			e.emit(st, &out)
		case "spectre":
			monster(args)
			e.emit(st, &out)
		default:
			fmt.Fprintf(os.Stderr, "Unknown directive '%s'\n", m[1])
		}
	}
	return out.String(), nil
}

var minionDeclRe = regexp.MustCompile(`^minions\["([^"]+)"\]\s*=`)

func loadMinionFile(text string, env luaEnv) (map[string]*luaTable, map[string]int, error) {
	locs := map[string]int{}
	for i, line := range strings.Split(text, "\n") {
		if m := minionDeclRe.FindStringSubmatch(line); m != nil {
			locs[m[1]] = i + 1
		}
	}
	toks, err := luaLex(text)
	if err != nil {
		return nil, nil, err
	}
	p := &luaParser{toks: toks, env: env}
	out := map[string]*luaTable{}
	for p.peek().kind != "eof" {
		t := p.next()
		switch {
		case t.kind == "name" && t.text == "return" && p.peek().text == "function":
			for !p.isOp(")") {
				if p.peek().kind == "eof" {
					return nil, nil, fmt.Errorf("line %d: unterminated function header", t.line)
				}
				p.next()
			}
			p.next()
		case t.kind == "name" && t.text == "return" && p.peek().text == "minions":
			p.next()
		case t.kind == "name" && t.text == "end":
		case t.kind == "name" && t.text == "local" && p.peek().text == "minions" && p.toks[p.pos+2].text == "{":
			for !p.isOp("}") {
				p.next()
			}
			p.next()
		case t.kind == "name" && t.text == "local":
			for !p.isOp("...") {
				if p.peek().kind == "eof" {
					return nil, nil, fmt.Errorf("line %d: unterminated local", t.line)
				}
				p.next()
			}
			p.next()
		case t.kind == "name" && t.text == "minions":
			if err := p.expect("["); err != nil {
				return nil, nil, err
			}
			k := p.next()
			if k.kind != "str" {
				return nil, nil, fmt.Errorf("line %d: minions key is not a string", k.line)
			}
			if err := p.expect("]"); err != nil {
				return nil, nil, err
			}
			if err := p.expect("="); err != nil {
				return nil, nil, err
			}
			v, err := p.expr()
			if err != nil {
				return nil, nil, fmt.Errorf("minions[%q]: %w", k.text, err)
			}
			tbl, ok := v.(*luaTable)
			if !ok {
				return nil, nil, fmt.Errorf("line %d: minions[%q] is not a table", k.line, k.text)
			}
			out[k.text] = tbl
		default:
			return nil, nil, fmt.Errorf("line %d: unexpected %q", t.line, t.text)
		}
	}
	return out, locs, nil
}

var (
	minionDumpFields = map[string]bool{
		"name": true, "life": true, "energyShield": true, "armour": true,
		"fireResist": true, "coldResist": true, "lightningResist": true, "chaosResist": true,
		"damage": true, "damageSpread": true, "damageFixup": true,
		"attackTime": true, "attackRange": true, "accuracy": true,
		"weaponType1": true, "weaponType2": true, "limit": true,
		"skillList": true, "modList": true,
		"evasion": true, "monsterTags": true, "baseDamageIgnoresAttackSpeed": true,
		"lifeScaling": true, "hostile": true,
	}
	minionModFields = map[string]bool{"name": true, "type": true, "value": true, "flags": true, "keywordFlags": true, "source": true}
)

func minionJSON(v any) any {
	switch v := v.(type) {
	case float64:
		return json.Number(luaNumStr(v))
	case *luaTable:
		if len(v.hash) == 0 {
			out := make([]any, len(v.arr))
			for i, x := range v.arr {
				out[i] = minionJSON(x)
			}
			return out
		}
		out := map[string]any{}
		for k, x := range v.hash {
			out[k] = minionJSON(x)
		}
		return out
	}
	return v
}

type minionConverter struct {
	modFlagName, keywordFlagName map[float64]string
	refusals                     []string
}

func (c *minionConverter) refuse(msg string) { c.refusals = append(c.refusals, msg) }

func (c *minionConverter) flagNames(bits float64, byValue map[float64]string, what, where string) []string {
	if bits == 0 {
		return []string{}
	}
	if n, ok := byValue[bits]; ok {
		return []string{n}
	}
	names := []string{}
	rest := uint64(bits)
	for bit := uint64(1); rest > 0 && bit <= 0x80000000; bit *= 2 {
		if rest&bit != 0 {
			n, ok := byValue[float64(bit)]
			if !ok {
				c.refuse(fmt.Sprintf("%s: %s bit 0x%x has no name in src/Data/Global.lua", where, what, bit))
				return nil
			}
			names = append(names, n)
			rest -= bit
		}
	}
	sort.Strings(names)
	return names
}

func (c *minionConverter) convertMod(m *luaTable, where string) map[string]any {
	for k := range m.hash {
		if !minionModFields[k] {
			c.refuse(where + ": mod carries unmodelled field " + k)
		}
	}
	out := map[string]any{"name": m.hash["name"], "modType": m.hash["type"]}
	if s, ok := m.hash["source"]; ok {
		out["source"] = s
	}
	switch v := m.hash["value"].(type) {
	case float64:
		out["valueKind"], out["number"] = "number", minionJSON(v)
	case bool:
		out["valueKind"], out["boolean"] = "boolean", v
	case *luaTable:
		switch {
		case v.hash["key"] != nil:
			for k := range v.hash {
				if k != "key" && k != "value" {
					c.refuse(where + ": SkillData payload carries unmodelled field " + k)
				}
			}
			out["valueKind"], out["dataKey"] = "skillData", v.hash["key"]
			switch dv := v.hash["value"].(type) {
			case float64:
				out["dataValueType"], out["dataNumber"] = "number", minionJSON(dv)
			case bool:
				out["dataValueType"], out["dataBoolean"] = "boolean", dv
			default:
				c.refuse(fmt.Sprintf("%s: SkillData value is a %T, which has no JSON form", where, dv))
			}
		case v.hash["mod"] != nil:
			nested, ok := v.hash["mod"].(*luaTable)
			if !ok {
				c.refuse(where + ": nested mod is not a table")
				return nil
			}
			out["valueKind"] = "nestedMod"
			out["nested"] = c.convertMod(nested, where+" nested")
			for k := range v.hash {
				if k != "mod" {
					c.refuse(where + ": nested-mod payload carries unmodelled field " + k)
				}
			}
		default:
			c.refuse(where + ": mod value is a table, which has no faithful JSON form")
			return nil
		}
	default:
		c.refuse(fmt.Sprintf("%s: mod value is a %T, which has no faithful JSON form", where, v))
		return nil
	}
	flags, _ := m.hash["flags"].(float64)
	keywordFlags, _ := m.hash["keywordFlags"].(float64)
	mf := c.flagNames(flags, c.modFlagName, "ModFlag", where)
	kf := c.flagNames(keywordFlags, c.keywordFlagName, "KeywordFlag", where)
	if mf == nil || kf == nil {
		return nil
	}
	out["flags"], out["keywordFlags"] = mf, kf
	tags := []any{}
	for i, tagAny := range m.arr {
		tag, ok := tagAny.(*luaTable)
		if !ok {
			c.refuse(fmt.Sprintf("%s: mod tag %d is not a table", where, i+1))
			continue
		}
		typ, _ := tag.hash["type"].(string)
		if typ != "" && typ != "Condition" {
			carried := map[string]any{"tagType": typ}
			for k, v := range tag.hash {
				if k == "type" {
					continue
				}
				switch v.(type) {
				case string, float64, bool:
					carried[k] = minionJSON(v)
				default:
					c.refuse(fmt.Sprintf("%s: tag %d field %s is a %T, which has no JSON form", where, i+1, k, v))
				}
			}
			tags = append(tags, carried)
			continue
		}
		if _, ok := tag.hash["var"].(string); typ != "Condition" || !ok {
			c.refuse(fmt.Sprintf("%s: mod tag %d is not a { type = \"Condition\", var = <string> }", where, i+1))
			continue
		}
		for k := range tag.hash {
			if k != "type" && k != "var" {
				c.refuse(where + ": Condition tag carries unmodelled field " + k)
			}
		}
		tags = append(tags, map[string]any{"tagType": typ, "var": tag.hash["var"]})
	}
	out["tags"] = tags
	return out
}

func (c *minionConverter) convert(id string, minion *luaTable, loc int, file string) map[string]any {
	where := "src/Data/" + file + " " + id
	for k := range minion.hash {
		if !minionDumpFields[k] {
			c.refuse(where + ": carries unmodelled field " + k)
		}
	}
	skills, _ := minion.hash["skillList"].(*luaTable)
	mods, _ := minion.hash["modList"].(*luaTable)
	if skills == nil {
		c.refuse(where + ": has no skillList")
		skills = newLuaTable()
	}
	if mods == nil {
		c.refuse(where + ": has no modList")
		mods = newLuaTable()
	}
	out := map[string]any{"id": id, "loc": loc}
	for k, v := range minion.hash {
		switch k {
		case "skillList", "modList", "monsterTags":
		default:
			out[k] = minionJSON(v)
		}
	}
	out["skillList"] = minionJSON(&luaTable{arr: append([]any{}, skills.arr...)})
	tags := []any{}
	if t, ok := minion.hash["monsterTags"].(*luaTable); ok {
		tags = append(tags, minionJSON(&luaTable{arr: t.arr}).([]any)...)
	}
	out["monsterTags"] = tags
	modList := []any{}
	for i, m := range mods.arr {
		mt, ok := m.(*luaTable)
		if !ok {
			c.refuse(fmt.Sprintf("%s modList[%d]: is not a mod", where, i+1))
			continue
		}
		modList = append(modList, c.convertMod(mt, fmt.Sprintf("%s modList[%d]", where, i+1)))
	}
	out["modList"] = modList
	return out
}

//pob:block:src/Modules/Data.lua:1155-1165
func dumpMinionData() (any, error) {
	pobDir := os.Getenv("POB_DIR")
	if pobDir == "" {
		return nil, fmt.Errorf("POB_DIR is not set; the minion set is the pinned PoB's Export/Minions templates")
	}
	ctx, cache := gameDataContext(), assetCache()
	for _, t := range []struct {
		name string
		init func() error
	}{
		{"MonsterVarieties", func() error { return poe.InitializeMonsterVarieties(ctx, raw.LatestVersion, cache) }},
		{"MonsterTypes", func() error { return poe.InitializeMonsterTypes(ctx, raw.LatestVersion, cache) }},
		{"MonsterResistances", func() error { return poe.InitializeMonsterResistances(ctx, raw.LatestVersion, cache) }},
		{"Tags", func() error { return poe.InitializeTags(ctx, raw.LatestVersion, cache) }},
		{"Stats", func() error { return poe.InitializeStats(ctx, raw.LatestVersion, cache) }},
		{"Mods", func() error { return poe.InitializeMods(ctx, raw.LatestVersion, cache) }},
		{"ItemClasses", func() error { return poe.InitializeItemClasses(ctx, raw.LatestVersion, cache) }},
		{"ActiveSkills", func() error { return poe.InitializeActiveSkills(ctx, raw.LatestVersion, cache) }},
		{"GrantedEffects", func() error { return poe.InitializeGrantedEffects(ctx, raw.LatestVersion, cache) }},
	} {
		if err := t.init(); err != nil {
			return nil, fmt.Errorf("%s: %w", t.name, err)
		}
	}
	flags, err := globalFlags(pobDir)
	if err != nil {
		return nil, err
	}
	env := luaFlagEnv(flags)
	statMap, err := loadSkillStatMap(pobDir, env)
	if err != nil {
		return nil, err
	}
	e := &minionExport{cache: cache, statMap: statMap, ot: map[string]string{}}
	files := map[string]string{}
	for _, name := range []string{"Spectres", "Minions"} {
		text, err := e.processTemplate(pobDir, name)
		if err != nil {
			return nil, err
		}
		files[name] = text
		if dir := os.Getenv("MINION_LUA_DIR"); dir != "" {
			if err := os.WriteFile(filepath.Join(dir, name+".lua"), []byte(text), 0o644); err != nil { //nolint:gosec // a debugging copy
				return nil, fmt.Errorf("MINION_LUA_DIR: %w", err)
			}
		}
	}
	for _, f := range e.missing {
		fmt.Fprintf(os.Stderr, "Invalid OT File location: %s\n", f)
	}
	if len(e.refusals) > 0 {
		return nil, fmt.Errorf("the export cannot be written:\n  %s", strings.Join(e.refusals, "\n  "))
	}

	c := &minionConverter{modFlagName: map[float64]string{}, keywordFlagName: map[float64]string{}}
	for n, v := range flags["ModFlag"] {
		c.modFlagName[v] = n
	}
	for n, v := range flags["KeywordFlag"] {
		c.keywordFlagName[v] = n
	}
	minions, minionLocs, err := loadMinionFile(files["Minions"], env)
	if err != nil {
		return nil, fmt.Errorf("Minions.lua: %w", err)
	}
	spectres, spectreLocs, err := loadMinionFile(files["Spectres"], env)
	if err != nil {
		return nil, fmt.Errorf("Spectres.lua: %w", err)
	}
	for _, s := range spectres {
		s.hash["limit"] = "ActiveSpectreLimit"
	}
	stamp := func(m *luaTable) {
		mods, _ := m.hash["modList"].(*luaTable)
		if mods == nil {
			return
		}
		for _, md := range mods.arr {
			if t, ok := md.(*luaTable); ok {
				t.hash["source"] = "Minion:" + fmt.Sprint(m.hash["name"])
			}
		}
	}
	outMinions, outSpectres := map[string]any{}, map[string]any{}
	for id, s := range spectres {
		stamp(s)
		if spectreLocs[id] == 0 {
			c.refuse("src/Data/Spectres.lua: no `minions[\"" + id + "\"] =` line to cite")
		}
		outSpectres[id] = c.convert(id, s, spectreLocs[id], "Spectres.lua")
	}
	for id, m := range minions {
		if spectres[id] != nil {
			continue
		}
		stamp(m)
		if minionLocs[id] == 0 {
			c.refuse("src/Data/Minions.lua: no `minions[\"" + id + "\"] =` line to cite")
		}
		outMinions[id] = c.convert(id, m, minionLocs[id], "Minions.lua")
	}
	if len(c.refusals) > 0 {
		sort.Strings(c.refusals)
		return nil, fmt.Errorf("the loaded data carries shapes this dumper does not model:\n  %s", strings.Join(c.refusals, "\n  "))
	}
	return map[string]any{
		"pobVersion":   "go-pob-data " + raw.LatestVersion,
		"minions":      outMinions,
		"spectres":     outSpectres,
		"minionCount":  len(outMinions),
		"spectreCount": len(outSpectres),
	}, nil
}
