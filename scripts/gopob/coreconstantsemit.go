package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

const ccnUsage = "usage: core_constants_emit.py <dump.json> <out.go>"

type ccnRefusalError struct{ msg string }

func (e *ccnRefusalError) Error() string { return e.msg }

func ccnRefuse(msg string) int {
	fmt.Fprintf(os.Stderr, "REFUSING: %s\n", msg)
	return 3
}

func ccnErr(format string, args ...any) error {
	return &ccnRefusalError{fmt.Sprintf(format, args...)}
}

func ccnNum(exact string) string {
	v, err := strconv.ParseFloat(strings.TrimSpace(exact), 64)
	if err != nil {
		return exact
	}
	if v == float64(int64(v)) && v < (1<<53) && v > -(1<<53) {
		return strconv.FormatInt(int64(v), 10)
	}
	return pyFloat(v)
}

func ccnGoStr(s string) string {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return `""`
	}
	return strings.TrimRight(b.String(), "\n")
}

func ccnExact(tbl any) map[string]any {
	t, _ := tbl.(map[string]any)
	m, _ := t["exact"].(map[string]any)
	return m
}

func ccnRows(tbl any) map[string]any {
	t, _ := tbl.(map[string]any)
	m, _ := t["rows"].(map[string]any)
	return m
}

func ccnRowNum(row any, field, where string) (string, error) {
	r, _ := row.(map[string]any)
	ex, _ := r["exact"].(map[string]any)
	v, ok := ex[field]
	if !ok || v == nil {
		return "", ccnErr("%s has no numeric field %s; PoB moved or renamed it", where, pyRepr(field))
	}
	return ccnNum(oeStr(v)), nil
}

func ccnValues(row any) map[string]any {
	r, _ := row.(map[string]any)
	v, _ := r["values"].(map[string]any)
	return v
}

var ccnIdentSplit = regexp.MustCompile(`[^A-Za-z0-9]+`)

func ccnCharacterConstantIdent(key string) string {
	s := strings.ReplaceAll(strings.ReplaceAll(key, "%", "Percent"), "+", "Plus")
	out := ""
	for _, p := range ccnIdentSplit.Split(s, -1) {
		if p == "" {
			continue
		}
		out += strings.ToUpper(p[:1]) + p[1:]
	}
	return out
}

func ccnSortedKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ccnEmitMisc(tbl any) ([]string, error) {
	exact := ccnExact(tbl)
	mapped := map[string]bool{}
	for _, row := range ccnMisc {
		mapped[row[0]] = true
	}
	unknown := []string{}
	for _, k := range ccnSortedKeys(exact) {
		if !mapped[k] {
			if _, ok := ccnMiscUnported[k]; !ok {
				unknown = append(unknown, k)
			}
		}
	}
	if len(unknown) > 0 {
		return nil, ccnErr("data.misc carries keys this emitter neither maps nor declares "+
			"unported:\n  %s\n\nEmitting anyway would drop them silently, which is exactly how "+
			"WardRechargeDelay sat one character wrong for months. Add each to MISC (with its Go "+
			"name) or to MISC_UNPORTED (with why nothing reads it).", strings.Join(unknown, "\n  "))
	}
	removed := map[string]bool{}
	for _, row := range ccnMiscRemoved {
		removed[row[0]] = true
	}
	absent := []string{}
	for _, row := range ccnMisc {
		if _, ok := exact[row[0]]; !ok && !removed[row[0]] {
			absent = append(absent, row[0])
		}
	}
	sort.Strings(absent)
	if len(absent) > 0 {
		return nil, ccnErr("these mapped data.misc keys are absent from the pinned PoB:\n  %s\n\n"+
			"PoB removed or renamed them. Emitting would leave the Go constants referring to "+
			"nothing; decide whether each is now ours-only or simply gone.",
			strings.Join(absent, "\n  "))
	}

	out := []string{
		`// The data.misc constants (src/Modules/Data.lua:446-503), PoB's own "magic numbers".`,
		"//",
		"// ⚠️ THE BOSS MULTIPLIERS ARE PoB's COMPUTED VALUES, NOT ITS EXPRESSIONS. PoB writes",
		"// `stdBossDPSMult = 4 / 4.40`; transcribing that expression into Go is not the same",
		"// number, because Go evaluates an untyped constant expression in exact arithmetic and",
		"// rounds ONCE while Lua rounds each operand and divides at runtime. The two agree here",
		"// and disagree by one ULP on `1 / 0.033` -- enough to move m_ceil(x * rate) by a whole",
		"// server tick. Emitting the measured value makes that a non-question.",
		"const (",
	}
	for _, row := range ccnMisc {
		lit := ccnNum(oeStr(exact[row[0]]))
		if row[2] != "" {
			lit = row[2] + "(" + lit + ")"
		}
		out = append(out, "\t"+row[1]+" = "+lit)
	}
	for _, row := range ccnMiscRemoved {
		if _, ok := exact[row[0]]; ok {
			return nil, ccnErr("%s is declared MISC_REMOVED_UPSTREAM but the pinned PoB HAS it. "+
				"Move it back to MISC -- a frozen value silently overriding a live one is worse "+
				"than either.", row[0])
		}
		out = append(out, "", "\t// ⚠️ FROZEN. "+row[3], "\t"+row[1]+" = "+ccnNum(row[2]))
	}
	return append(out, ")", ""), nil
}

func ccnEmitCharacterConstants(tbl any) ([]string, error) {
	exact := ccnExact(tbl)
	t, _ := tbl.(map[string]any)
	if dropped, _ := t["dropped"].([]any); len(dropped) > 0 {
		return nil, ccnErr("data.characterConstants carries non-scalar values this emitter cannot "+
			"render:\n  %s\n\nEvery key in this table has always been a plain number. Model the "+
			"new shape rather than dropping it.", strings.Join(mdStrList(dropped), "\n  "))
	}
	idents := map[string]string{}
	for _, key := range ccnSortedKeys(exact) {
		ident := ccnCharacterConstantIdent(key)
		if prev, ok := idents[ident]; ok {
			return nil, ccnErr("two characterConstants keys map to the Go identifier %s:\n  %s\n  "+
				"%s\n\nThe mapping is mechanical, so a collision means two stat ids differ only in "+
				"punctuation. Disambiguate here rather than letting one silently win.",
				pyRepr(ident), pyRepr(prev), pyRepr(key))
		}
		idents[ident] = key
	}
	if len(exact) != ccnCharacterConstantKeyCount {
		return nil, ccnErr("data.characterConstants has %d keys; this emitter was written against "+
			"%d.\n\nThat is not a failure -- GGG adds and removes stats -- but the count is pinned "+
			"so the change is DECIDED rather than absorbed. Update CHARACTER_CONSTANT_KEY_COUNT "+
			"once you have looked at what moved.", len(exact), ccnCharacterConstantKeyCount)
	}
	out := []string{
		"// data.characterConstants (src/Data/Misc.lua:62-152) -- the game constants GGG ships and",
		"// PoB reads by their raw stat ids. 57 sites across src/Modules read one.",
		"//",
		"// ⚠️ ALL 89 ARE EMITTED, INCLUDING THE 40 NOTHING READS. A key with no consumer today is",
		"// the key that gains one at the next pin, and one omitted here is one the drift gate can",
		"// never see change. The Go names are derived mechanically from the stat ids and checked",
		"// for collisions by the emitter.",
		"const (",
	}
	names := make([]string, 0, len(idents))
	for ident := range idents {
		names = append(names, ident)
	}
	sort.Strings(names)
	for _, ident := range names {
		key := idents[ident]
		out = append(out, "\t// "+key, "\t"+ident+" = "+ccnNum(oeStr(exact[key])))
	}
	return append(out, ")", ""), nil
}

func ccnEmitMonster(name, goVar string, tbl any) ([]string, error) {
	t, _ := tbl.(map[string]any)
	values, _ := t["exact"].([]any)
	if len(values) == 0 {
		return nil, ccnErr("data.%s dumped empty", name)
	}
	parts := []string{"0"}
	for _, v := range values {
		parts = append(parts, ccnNum(oeStr(v)))
	}
	return []string{"var " + goVar + " = []float64{" + strings.Join(parts, ", ") + "}", ""}, nil
}

func ccnEmitLevelMult(name, goVar string, tbl any) ([]string, error) {
	t, _ := tbl.(map[string]any)
	values, _ := t["exact"].(map[string]any)
	if len(values) == 0 {
		return nil, ccnErr("data.%s dumped empty", name)
	}
	keys := make([]int, 0, len(values))
	for k := range values {
		n, err := strconv.Atoi(k)
		if err != nil {
			return nil, ccnErr("data.%s has a non-integer key %s", name, pyRepr(k))
		}
		keys = append(keys, n)
	}
	sort.Ints(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, strconv.Itoa(k)+": "+ccnNum(oeStr(values[strconv.Itoa(k)])))
	}
	return []string{
		"var " + goVar + " = map[int]float64{" + strings.Join(parts, ", ") + "}", "",
	}, nil
}

func ccnEmitNonDamaging(tbl any) ([]string, error) {
	rows := ccnRows(tbl)
	known := map[string]string{}
	for _, row := range ccnAilment {
		known[row[0]] = row[1]
	}
	unknown := []string{}
	for _, k := range ccnSortedKeys(rows) {
		if _, ok := known[k]; !ok {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		return nil, ccnErr("data.nonDamagingAilment has rows this emitter cannot name: %s",
			pyReprValue(mdAnyList(unknown)))
	}
	absent := []string{}
	for _, row := range ccnAilment {
		if _, ok := rows[row[0]]; !ok {
			absent = append(absent, row[0])
		}
	}
	sort.Strings(absent)
	if len(absent) > 0 {
		return nil, ccnErr("data.nonDamagingAilment is missing rows this emitter expects: %s",
			pyReprValue(mdAnyList(absent)))
	}
	out := []string{
		"// NonDamagingAilments is data.nonDamagingAilment (src/Modules/Data.lua:324-331).",
		"//",
		"// ⚠️ A NIL FIELD AND A ZERO FIELD ARE DIFFERENT THINGS, WHICH IS WHY Default AND Duration",
		"// ARE POINTERS. Freeze carries `default = nil` and `duration = nil` in the Lua; every",
		"// other row carries numbers. Lua distinguishes nil from 0 and Go collapses them, so a",
		`// plain float64 here would silently turn "Freeze has no default" into "Freeze defaults`,
		`// to 0". The dump omits a nil field entirely, and absence is read as nil below.`,
		"var NonDamagingAilments = map[Ailment]NonDamagingAilmentData{",
	}
	for _, ar := range ccnAilment {
		pobName, goName := ar[0], ar[1]
		row := rows[pobName]
		vals := ccnValues(row)
		assoc := vals["associatedType"]
		dt, ok := ccnDamageTypeConst[oeStr(assoc)]
		if !ok || assoc == nil {
			return nil, ccnErr("nonDamagingAilment.%s has associatedType %s, unmapped",
				pobName, pyReprValue(assoc))
		}
		alt := "false"
		if oeTruthy(vals["alt"]) {
			alt = "true"
		}
		out = append(out, "\t"+goName+": {",
			"\t\tAssociatedType: "+dt+",",
			"\t\tAlt:            "+alt+",")
		r, _ := row.(map[string]any)
		ex, _ := r["exact"].(map[string]any)
		for _, pair := range [][2]string{{"Default", "default"}, {"Duration", "duration"}} {
			if v, ok := ex[pair[1]]; ok {
				out = append(out, "\t\t"+pair[0]+": utils.Ptr[float64]("+ccnNum(oeStr(v))+"),")
			} else {
				out = append(out, "\t\t"+pair[0]+": nil,")
			}
		}
		for _, pair := range [][2]string{{"Min", "min"}, {"Max", "max"}, {"Precision", "precision"}} {
			n, err := ccnRowNum(row, pair[1], "nonDamagingAilment."+pobName)
			if err != nil {
				return nil, err
			}
			out = append(out, "\t\t"+pair[0]+": "+n+",")
		}
		out = append(out, "\t},")
	}
	return append(out, "}", ""), nil
}

func mdAnyList(xs []string) []any {
	out := make([]any, len(xs))
	for i, s := range xs {
		out[i] = s
	}
	return out
}

func ccnEmitKeystones(tbl any) ([]string, error) {
	t, _ := tbl.(map[string]any)
	names, _ := t["strings"].([]any)
	if len(names) == 0 {
		return nil, ccnErr("data.keystones dumped empty")
	}
	out := []string{
		"// Keystones is data.keystones (src/Modules/Data.lua:264-318): every keystone name the mod",
		"// parser accepts as a mod line of its own. ModParser.lua:3476-3478 turns each into a",
		`// mod("Keystone", "LIST", name) entry in specialModList, which is how items, Scourge and`,
		"// Recombinator mods grant a keystone by name. PoB's own order is kept.",
		"var Keystones = []string{",
	}
	for _, n := range names {
		out = append(out, "\t"+ccnGoStr(oeStr(n))+",")
	}
	return append(out, "}", ""), nil
}

func ccnEmitJewelRadius(tbl any) ([]string, error) {
	t, _ := tbl.(map[string]any)
	rows, _ := t["rows"].([]any)
	if len(rows) == 0 {
		return nil, ccnErr("data.jewelRadius dumped empty")
	}
	out := []string{
		"// JewelRadius is data.jewelRadius (src/Modules/Data.lua:148-173).",
		"//",
		"// ⚠️ PoB SELECTS IT BY TREE VERSION AT LOAD TIME; IT IS NOT A LITERAL. data.jewelRadii",
		`// holds a "3_15" and a "3_16" table and setJewelRadiiGlobally picks one. The dumper`,
		"// reads the RESOLVED data.jewelRadius, so this is the table the pinned PoB actually",
		"// grades with rather than whichever branch a transcriber happened to copy.",
		"var JewelRadius = []JewelRadiusData{",
	}
	for i, row := range rows {
		vals := ccnValues(row)
		where := fmt.Sprintf("jewelRadius[%d]", i+1)
		for _, field := range []string{"col", "label"} {
			if _, ok := vals[field]; !ok {
				return nil, ccnErr("%s has no %s", where, pyRepr(field))
			}
		}
		inner, err := ccnRowNum(row, "inner", where)
		if err != nil {
			return nil, err
		}
		outer, err := ccnRowNum(row, "outer", where)
		if err != nil {
			return nil, err
		}
		out = append(out, fmt.Sprintf("\t{Inner: %s, Outer: %s, Col: %s, Label: %s},",
			inner, outer, ccnGoStr(oeStr(vals["col"])), ccnGoStr(oeStr(vals["label"]))))
	}
	return append(out, "}", ""), nil
}

func ccnEmitUnarmed(tbl any) ([]string, error) {
	rows := ccnRows(tbl)
	weaponClass := map[string]string{}
	for _, row := range ccnWeaponClass {
		weaponClass[row[0]] = row[1]
	}
	out := []string{
		"// UnarmedWeaponData is data.unarmedWeaponData (src/Modules/Data.lua:225-233), indexed by",
		"// PoB's class id 0-6 — the same numbering pob.ClassID uses.",
		"//",
		"// ⚠️ THE float64 CONVERSIONS ARE LOAD-BEARING AND NOT DECORATION. item_setup.go:701 feeds",
		"// these through utils.Set, which sets a struct field by REFLECTION, so the value's dynamic",
		"// type is what lands in SkillData. An unwrapped `0` would be an int in the interface{} and",
		"// the field write would fail or panic instead of producing a zero CritChance.",
		"var UnarmedWeaponData = map[pob.ClassID]map[string]interface{}{",
	}
	for i, className := range ccnUnarmedClass {
		row, ok := rows[strconv.Itoa(i)]
		if !ok || row == nil {
			return nil, ccnErr("data.unarmedWeaponData has no class index %d (%s)", i, className)
		}
		vals := ccnValues(row)
		wt, ok := vals["type"]
		goClass, mapped := weaponClass[oeStr(wt)]
		if !ok || !mapped {
			return nil, ccnErr("unarmedWeaponData[%d].type = %s, unmapped", i, pyReprValue(wt))
		}
		where := fmt.Sprintf("unarmedWeaponData[%d]", i)
		fields := []string{`"Type": ` + goClass}
		for _, f := range []string{"AttackRate", "CritChance", "PhysicalMin", "PhysicalMax"} {
			n, err := ccnRowNum(row, f, where)
			if err != nil {
				return nil, err
			}
			fields = append(fields, fmt.Sprintf(`"%s": float64(%s)`, f, n))
		}
		out = append(out, fmt.Sprintf("\tpob.%s: {%s},", className, strings.Join(fields, ", ")))
	}
	return append(out, "}", ""), nil
}

func ccnEmitWeaponTypes(tbl any) ([]string, error) {
	rows := ccnRows(tbl)
	known := map[string]string{}
	for _, row := range ccnWeaponClass {
		known[row[0]] = row[1]
	}
	unknown := []string{}
	for _, k := range ccnSortedKeys(rows) {
		if _, ok := known[k]; !ok {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		return nil, ccnErr("data.weaponTypeInfo has classes this emitter cannot name: %s",
			pyReprValue(mdAnyList(unknown)))
	}
	absent := []string{}
	for _, row := range ccnWeaponClass {
		if _, ok := rows[row[0]]; !ok {
			absent = append(absent, row[0])
		}
	}
	sort.Strings(absent)
	if len(absent) > 0 {
		return nil, ccnErr("data.weaponTypeInfo is missing classes this emitter expects: %s",
			pyReprValue(mdAnyList(absent)))
	}
	out := []string{
		"// WeaponTypes is data.weaponTypeInfo (src/Modules/Data.lua:208-224).",
		"//",
		"// ModFlag has NO PoB counterpart: PoB tests the `flag` STRING where go-pob's ModDB filters",
		"// on a bitmask, so the bit is derived from `flag` by oracle/core_constants_emit.py's",
		"// WEAPON_MOD_FLAG map rather than transcribed. `Label` is present on two rows only —",
		"// Sceptre and Thrusting One Handed Sword — and is the empty string on the other thirteen,",
		"// which is what PoB's absent field means here.",
		"var WeaponTypes = map[ItemClassName]*WeaponTypeInfo{",
	}
	for _, wr := range ccnWeaponClass {
		pobName, goName := wr[0], wr[1]
		vals := ccnValues(rows[pobName])
		flag, has := vals["flag"]
		mf, mapped := ccnWeaponModFlag[oeStr(flag)]
		if !has || !mapped {
			return nil, ccnErr("weaponTypeInfo[%s].flag = %s, no mod.MFlag mapped",
				pyRepr(pobName), pyReprValue(flag))
		}
		oneHand, melee := "false", "false"
		if oeTruthy(vals["oneHand"]) {
			oneHand = "true"
		}
		if oeTruthy(vals["melee"]) {
			melee = "true"
		}
		parts := []string{
			"OneHand: " + oneHand,
			"Melee: " + melee,
			"Flag: " + ccnGoStr(oeStr(flag)),
		}
		if oeTruthy(vals["label"]) {
			parts = append(parts, "Label: "+ccnGoStr(oeStr(vals["label"])))
		}
		parts = append(parts, "ModFlag: "+mf)
		out = append(out, "\t"+goName+": {"+strings.Join(parts, ", ")+"},")
	}
	return append(out, "}", ""), nil
}

func ccnEmitBossStats(tbl any) ([]string, error) {
	exact := ccnExact(tbl)
	known := map[string]bool{}
	for _, f := range ccnBossStats {
		known[f] = true
	}
	unknown := []string{}
	for _, k := range ccnSortedKeys(exact) {
		if !known[k] {
			unknown = append(unknown, k)
		}
	}
	if len(unknown) > 0 {
		return nil, ccnErr("data.bossStats carries means this emitter does not know:\n  %s\n\n"+
			"PoB grew a boss statistic. Add it to BOSS_STATS and give it a consumer, or it will "+
			"be dropped silently.", strings.Join(unknown, "\n  "))
	}
	absent := []string{}
	for _, f := range ccnBossStats {
		if _, ok := exact[f]; !ok {
			absent = append(absent, f)
		}
	}
	sort.Strings(absent)
	if len(absent) > 0 {
		return nil, ccnErr("these data.bossStats means are absent from the pinned PoB:\n  %s\n\n"+
			"PoB removed or renamed them; the Go fields would refer to nothing.",
			strings.Join(absent, "\n  "))
	}
	out := []string{
		"// BossStats is data.bossStats (src/Modules/Data.lua:505-530): the mean armourMult and",
		"// evasionMult over Data/Bosses, once across all 22 bosses and once across the 7 isUber",
		"// ones, each offset by 100 so a call site reads it as a percentage.",
		"//",
		"// ⚠️ DERIVED AT LOAD TIME, NOT TRANSCRIBED. There is no such table in the Lua source --",
		"// only the loop that builds it -- so the value here is what the pinned PoB computed.",
		"//",
		"// Its consumers are the enemyArmour/enemyEvasion placeholders on the Pinnacle and Uber",
		"// branches of the enemyIsBoss handler (ConfigOptions.lua:1554-1555, :1587-1588). The None",
		"// and Boss branches use the raw monster table with no mean at all.",
		"var BossStats = struct {",
	}
	for _, f := range ccnBossStats {
		out = append(out, "\t"+f+" float64")
	}
	out = append(out, "}{")
	for _, f := range ccnBossStats {
		out = append(out, "\t"+f+": "+ccnNum(oeStr(exact[f]))+",")
	}
	return append(out, "}", ""), nil
}

func coreConstantsEmit(args []string) int {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, ccnUsage)
		return 2
	}
	raw, rerr := os.ReadFile(args[0]) //nolint:gosec // the operator's dump path, as in the original
	if rerr != nil {
		return ccnRefuse(fmt.Sprintf("the dump (%s) is not readable: %s", args[0], errnoText(rerr)))
	}
	if !utf8.Valid(raw) {
		return ccnRefuse(fmt.Sprintf("the dump (%s) is not valid JSON", args[0]))
	}
	var top any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&top); err != nil {
		return ccnRefuse(fmt.Sprintf("the dump (%s) is not valid JSON", args[0]))
	}
	dump, isObj := top.(map[string]any)
	if !isObj {
		return ccnRefuse(fmt.Sprintf("the dump (%s) is not a JSON object", args[0]))
	}
	tables, _ := dump["tables"].(map[string]any)
	if _, present := dump["tables"]; present && tables == nil {
		if oeTruthy(dump["tables"]) {
			return ccnRefuse(fmt.Sprintf("the dump (%s)'s 'tables' is not a JSON object", args[0]))
		}
	}
	if tables == nil {
		tables = map[string]any{}
	}

	expected := []string{"nonDamagingAilment", "keystones", "jewelRadius", "unarmedWeaponData",
		"weaponTypeInfo", "misc", "bossStats", "characterConstants"}
	for _, row := range ccnMonsterTables {
		expected = append(expected, row[0])
	}
	for _, row := range ccnLevelMultTables {
		expected = append(expected, row[0])
	}
	missing := []string{}
	for _, name := range expected {
		if _, ok := tables[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return ccnRefuse("the dump is missing tables this emitter needs:\n  " +
			strings.Join(missing, "\n  ") +
			"\n\nRe-run oracle/core_constants.lua against the pinned PoB.")
	}

	shape := map[string]string{
		"misc": "exact", "characterConstants": "exact", "bossStats": "exact",
		"nonDamagingAilment": "rows", "jewelRadius": "rows", "unarmedWeaponData": "rows",
		"weaponTypeInfo": "rows", "keystones": "strings",
	}
	for _, row := range ccnMonsterTables {
		shape[row[0]] = "exact"
	}
	for _, row := range ccnLevelMultTables {
		shape[row[0]] = "exact"
	}
	sorted_ := append([]string{}, expected...)
	sort.Strings(sorted_)
	for _, name := range sorted_ {
		want := shape[name]
		tbl, ok := tables[name].(map[string]any)
		if !ok {
			return ccnRefuse(fmt.Sprintf("the dump's table %s has no %s; the dumper wrote a shape "+
				"this emitter cannot read. Re-run oracle/core_constants.lua.",
				pyRepr(name), pyRepr(want)))
		}
		if _, has := tbl[want]; !has {
			return ccnRefuse(fmt.Sprintf("the dump's table %s has no %s; the dumper wrote a shape "+
				"this emitter cannot read. Re-run oracle/core_constants.lua.",
				pyRepr(name), pyRepr(want)))
		}
	}

	lines := []string{strings.NewReplacer(
		"{version}", oeStr(mdGet(dump, "pobVersion", "unknown")),
		"{tree}", oeStr(mdGet(dump, "treeVersion", "unknown")),
	).Replace(ccnHeader)}

	type section func() ([]string, error)
	steps := []section{
		func() ([]string, error) { return ccnEmitMisc(tables["misc"]) },
		func() ([]string, error) { return ccnEmitCharacterConstants(tables["characterConstants"]) },
	}
	for _, s := range steps {
		part, err := s()
		if err != nil {
			return ccnRefuse(err.Error())
		}
		lines = append(lines, part...)
	}
	lines = append(lines,
		"// The six monster scaling tables (src/Data/Misc.lua, loaded into data.* by Modules/Data.lua).",
		"// Every one carries a LEADING 0 that PoB has no element for, so `table[level]` indexes the",
		"// same way it does in the Lua instead of subtracting 1 at every call site.",
		"")
	for _, row := range ccnMonsterTables {
		part, err := ccnEmitMonster(row[0], row[1], tables[row[0]])
		if err != nil {
			//go-only: THIS SITE HAS NO LITERAL AND THE SWEEP BLAMES THE NEXT ONE — the exact
			return ccnRefuse(err.Error())
		}
		lines = append(lines, part...)
	}
	lines = append(lines,
		"// The map-level scaling tables (src/Data/Misc.lua:469-473, from MonsterMapDifficulty.dat).",
		"// SPARSE — keyed by MAP LEVEL, not by index from 1 — so they are maps and carry no leading",
		"// 0 sentinel. Every reader is `table[level] or 1`, which a Go map's zero value cannot spell:",
		"// read them through a presence check, never as a bare index.",
		"")
	for _, row := range ccnLevelMultTables {
		part, err := ccnEmitLevelMult(row[0], row[1], tables[row[0]])
		if err != nil {
			return ccnRefuse(err.Error())
		}
		lines = append(lines, part...)
	}
	rest := []section{
		func() ([]string, error) { return ccnEmitNonDamaging(tables["nonDamagingAilment"]) },
		func() ([]string, error) { return ccnEmitKeystones(tables["keystones"]) },
		func() ([]string, error) { return ccnEmitJewelRadius(tables["jewelRadius"]) },
		func() ([]string, error) { return ccnEmitUnarmed(tables["unarmedWeaponData"]) },
		func() ([]string, error) { return ccnEmitWeaponTypes(tables["weaponTypeInfo"]) },
		func() ([]string, error) { return ccnEmitBossStats(tables["bossStats"]) },
	}
	for _, s := range rest {
		part, err := s()
		if err != nil {
			return ccnRefuse(err.Error())
		}
		lines = append(lines, part...)
	}

	if err := os.WriteFile(args[1], []byte(strings.Join(lines, "\n")), 0o644); err != nil { //nolint:gosec // the original's mode
		return ccnRefuse(fmt.Sprintf("%s is not writable: %s", args[1], errnoText(err)))
	}
	fmt.Fprintf(os.Stderr, "wrote %s: %d misc constants, %d monster tables, %d ailments, "+
		"%d keystones, %d jewel radii, %d weapon types\n",
		args[1], len(ccnMisc), len(ccnMonsterTables),
		len(ccnRows(tables["nonDamagingAilment"])),
		ccnLen(tables["keystones"], "strings"),
		ccnLen(tables["jewelRadius"], "rows"),
		len(ccnRows(tables["weaponTypeInfo"])))
	return 0
}

func ccnLen(tbl any, key string) int {
	t, _ := tbl.(map[string]any)
	a, _ := t[key].([]any)
	return len(a)
}
