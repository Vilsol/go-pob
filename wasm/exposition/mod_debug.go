package exposition

import (
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

func ModDebugExports(r bind.Registry) {
	r.Func(ModDebugDump, bind.AsPromise())
	r.Func(SetItemModDebug, bind.AsPromise())
}

//pob:src/Data/Global.lua:101-132
var modFlagNames = map[string]mod.MFlag{
	"Attack": mod.MFlagAttack, "Spell": mod.MFlagSpell, "Hit": mod.MFlagHit, "Dot": mod.MFlagDot, "Cast": mod.MFlagCast,
	"Melee": mod.MFlagMelee, "Area": mod.MFlagArea, "Projectile": mod.MFlagProjectile, "SourceMask": mod.MFlagSourceMask,
	"Ailment": mod.MFlagAilment, "MeleeHit": mod.MFlagMeleeHit, "Weapon": mod.MFlagWeapon,
	"Axe": mod.MFlagAxe, "Bow": mod.MFlagBow, "Claw": mod.MFlagClaw, "Dagger": mod.MFlagDagger, "Mace": mod.MFlagMace,
	"Staff": mod.MFlagStaff, "Sword": mod.MFlagSword, "Wand": mod.MFlagWand, "Unarmed": mod.MFlagUnarmed,
	"Fishing": mod.MFlagFishing, "WeaponMelee": mod.MFlagWeaponMelee, "WeaponRanged": mod.MFlagWeaponRanged,
	"Weapon1H": mod.MFlagWeapon1H, "Weapon2H": mod.MFlagWeapon2H, "WeaponMask": mod.MFlagWeaponMask,
}

//pob:src/Modules/ModTools.lua:114-127
func formatFlags[F ~int](flags F, src map[string]F) string {
	var flagNames []string
	for name, val := range src {
		if flags&val == val {
			flagNames = append(flagNames, name)
		}
	}
	slices.Sort(flagNames)
	if len(flagNames) == 0 {
		return "-"
	}
	return strings.Join(flagNames, ",")
}

func formatModFlags(flags mod.MFlag) string { return formatFlags(flags, modFlagNames) }

func formatKeywordFlags(flags mod.KeywordFlag) string {
	return formatFlags(flags, mod.KeywordFlagsByName)
}

//pob:src/Modules/ModTools.lua:129-164
func formatTag(tag mod.Tag) string {
	v := reflect.ValueOf(tag)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	var params []string
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if !f.IsExported() || f.Name == "TagType" || v.Field(i).IsZero() {
				continue
			}
			val := mod.FormatValue(v.Field(i).Interface())
			if k := v.Field(i).Kind(); k == reflect.Slice || k == reflect.Array {
				val = strings.TrimSuffix(strings.TrimPrefix(val, "{"), "}")
				val = "{" + val + "}"
			}
			params = append(params, f.Name+"="+val)
		}
	}
	slices.Sort(params)
	return strings.Join(append([]string{"type=" + string(tag.Type())}, params...), "/")
}

//pob:src/Modules/ModTools.lua:166-172
func formatTags(tags []mod.Tag) string {
	if len(tags) == 0 {
		return "-"
	}
	out := make([]string, len(tags))
	for i, tag := range tags {
		out[i] = formatTag(tag)
	}
	return strings.Join(out, ",")
}

//pob:src/Modules/ModTools.lua:174-203
func formatModValue(v *mod.ModValueMulti) string {
	if v == nil {
		return "nil"
	}
	switch v.Type() {
	case mod.ModValueMultiTypeFlag:
		return strconv.FormatBool(v.ValueFlag)
	case mod.ModValueMultiTypeList:
		return mod.FormatValue(v.ValueList)
	default:
		return strconv.FormatFloat(v.ValueFloat, 'g', 14, 64)
	}
}

//pob:src/Modules/ModTools.lua:205-207
func formatModParams(m mod.Mod) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s", m.Name(), m.Type(), formatModFlags(m.Flags()), formatKeywordFlags(m.KeywordFlags()), formatTags(m.Tags()))
}

//pob:src/Modules/ModTools.lua:209-211
func formatMod(m mod.Mod) string {
	return formatModValue(m.Value()) + " = " + formatModParams(m)
}

func modSource(m mod.Mod) string {
	if s := m.GetSource(); s != "" {
		return string(s)
	}
	return "?"
}

//pob:src/Classes/ModDB.lua:323-358
func printModDB(w io.Writer, db *moddb.ModDB) {
	fmt.Fprintln(w, "=== Modifiers ===")
	modNames := make([]string, 0, len(db.Mods))
	for modName := range db.Mods {
		modNames = append(modNames, modName)
	}
	slices.Sort(modNames)
	for _, modName := range modNames {
		fmt.Fprintf(w, "'%s':\n", modName)
		for _, m := range db.Mods[modName] {
			fmt.Fprintf(w, "\t%s = %s|%s|%s|%s|%s\n", formatModValue(m.Value()), m.Type(), formatModFlags(m.Flags()), formatKeywordFlags(m.KeywordFlags()), formatTags(m.Tags()), modSource(m))
		}
	}
	fmt.Fprintln(w, "=== Conditions ===")
	var nameList []string
	for name, value := range db.Conditions {
		if value {
			nameList = append(nameList, name)
		}
	}
	slices.Sort(nameList)
	for _, name := range nameList {
		fmt.Fprintln(w, name)
	}
	fmt.Fprintln(w, "=== Multipliers ===")
	nameList = nameList[:0]
	for name, value := range db.Multipliers {
		if value > 0 {
			nameList = append(nameList, name)
		}
	}
	slices.Sort(nameList)
	for _, name := range nameList {
		fmt.Fprintf(w, "%s = %d\n", name, int64(db.Multipliers[name]))
	}
}

//pob:src/Classes/ModList.lua:241-245
func printModList(w io.Writer, list *moddb.ModList) {
	for _, m := range list.Mods() {
		fmt.Fprintf(w, "%s|%s\n", formatMod(m), modSource(m))
	}
}

//pob:src/Modules/Common.lua:528-546
func prettyPrintTable(w io.Writer, v reflect.Value, pre string) {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	type entry struct {
		name string
		val  reflect.Value
	}
	var entries []entry
	switch v.Kind() {
	case reflect.Map:
		for _, k := range v.MapKeys() {
			entries = append(entries, entry{fmt.Sprint(k.Interface()), v.MapIndex(k)})
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() && !v.Field(i).IsZero() {
				entries = append(entries, entry{v.Type().Field(i).Name, v.Field(i)})
			}
		}
	default:
		return
	}
	slices.SortFunc(entries, func(a, b entry) int { return strings.Compare(a.name, b.name) })
	for _, e := range entries {
		val := e.val
		for val.Kind() == reflect.Interface && !val.IsNil() {
			val = val.Elem()
		}
		switch val.Kind() {
		case reflect.Map, reflect.Struct, reflect.Pointer:
			prettyPrintTable(w, val, pre+e.name+".")
		case reflect.Float64, reflect.Float32:
			fmt.Fprintf(w, "%s%s = %s\n", pre, e.name, strconv.FormatFloat(val.Float(), 'g', 14, 64))
		default:
			fmt.Fprintf(w, "%s%s = %s\n", pre, e.name, mod.FormatValue(val.Interface()))
		}
	}
}

func printSkillEffects(w io.Writer, skill *calculator.ActiveSkill, indent string) {
	for _, skillEffect := range skill.EffectList {
		fmt.Fprintf(w, "%s%s %d/%d\n", indent, skillEffect.GrantedEffect.Name(), skillEffect.Level, skillEffect.Quality)
	}
}

func ModDebugDump(c *calculator.Calculator) string {
	if c == nil {
		return ""
	}
	return modDebugDump(c.BuildOutput(calculator.OutputModeMain))
}

//pob:src/Modules/Calcs.lua:37-70
func modDebugDump(env *calculator.Environment) string {
	if env == nil || env.ModDB == nil {
		return ""
	}
	var w strings.Builder
	if parent, ok := env.ModDB.Parent.(*moddb.ModDB); ok && parent != nil {
		printModDB(&w, parent)
	}
	printModDB(&w, env.ModDB)
	if env.Minion != nil && env.Minion.Actor != nil && env.Minion.ModDB != nil {
		fmt.Fprintln(&w, "=== Minion Mod DB ===")
		printModDB(&w, env.Minion.ModDB)
	}
	if env.EnemyModDB != nil {
		fmt.Fprintln(&w, "=== Enemy Mod DB ===")
		printModDB(&w, env.EnemyModDB)
	}
	var mainSkill *calculator.ActiveSkill
	if env.Minion != nil && env.Minion.Actor != nil {
		mainSkill = env.Minion.MainSkill
	} else if env.Player != nil {
		mainSkill = env.Player.MainSkill
	}
	if mainSkill != nil {
		fmt.Fprintln(&w, "=== Main Skill ===")
		printSkillEffects(&w, mainSkill, "")
		fmt.Fprintln(&w, "=== Main Skill Flags ===")
		var flags mod.MFlag
		var keywordFlags mod.KeywordFlag
		if mainSkill.SkillCfg != nil {
			flags, keywordFlags = mainSkill.SkillCfg.Flags.Get(), mainSkill.SkillCfg.KeywordFlags.Get()
		}
		fmt.Fprintf(&w, "Mod: %s\n", formatModFlags(flags))
		fmt.Fprintf(&w, "Keyword: %s\n", formatKeywordFlags(keywordFlags))
		fmt.Fprintln(&w, "=== Main Skill Mods ===")
		if mainSkill.SkillModList != nil {
			if parent, ok := mainSkill.SkillModList.Parent.(*moddb.ModList); ok && parent != nil {
				printModList(&w, parent)
			}
			printModList(&w, mainSkill.SkillModList)
		}
		fmt.Fprintln(&w, "=== Main Skill Data ===")
		prettyPrintTable(&w, reflect.ValueOf(mainSkill.SkillData), "")
	}
	fmt.Fprintln(&w, "== Aux Skills ==")
	for i, aux := range env.AuxSkillList {
		fmt.Fprintf(&w, "Skill #%d:\n", i+1)
		printSkillEffects(&w, aux, "  ")
	}
	fmt.Fprintln(&w, "== Output Table ==")
	if env.Player != nil {
		prettyPrintTable(&w, reflect.ValueOf(env.Player.Output), "")
	}
	return w.String()
}

var itemModDebug = false

func SetItemModDebug(on bool) {
	itemModDebug = on
}

//pob:block:src/Classes/ItemsTab.lua:5160-5166
func appendItemModDebug(t *calculator.ItemTooltip, item *calculator.Item) {
	if !itemModDebug {
		return
	}
	if n := len(t.Lines); n == 0 || !t.Lines[n-1].Separator {
		t.Lines = append(t.Lines, calculator.TooltipLine{Separator: true})
	}
	for _, m := range item.Mods {
		t.Lines = append(t.Lines, calculator.TooltipLine{Text: "^7" + formatMod(m)})
	}
}
