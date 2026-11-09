package calculator2

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

// sFormat is string.format as the oracle's LuaJIT prints it: %d truncates, %s of a number is tostring, %g
// defaults to six significant digits, and a decimal tie rounds away from zero.
func sFormat(format string, args ...any) string {
	var out strings.Builder
	arg := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			out.WriteByte(format[i])
			continue
		}
		start := i
		i++
		for i < len(format) && strings.IndexByte("+- #0", format[i]) >= 0 {
			i++
		}
		for i < len(format) && format[i] >= '0' && format[i] <= '9' {
			i++
		}
		prec := -1
		if i < len(format) && format[i] == '.' {
			prec = 0
			for i++; i < len(format) && format[i] >= '0' && format[i] <= '9'; i++ {
				prec = prec*10 + int(format[i]-'0')
			}
		}
		if i >= len(format) {
			out.WriteString(format[start:])
			break
		}
		verb := format[i]
		spec := format[start:i]
		if verb == '%' {
			out.WriteByte('%')
			continue
		}
		var a any
		if arg < len(args) {
			a = args[arg]
		}
		arg++
		v, isNum := formatArg(a)
		if isNum && strings.IndexByte("fFeEgG", verb) >= 0 && (math.IsInf(v, 0) || math.IsNaN(v)) {
			width := spec
			if i := strings.IndexByte(spec, '.'); i >= 0 {
				width = spec[:i]
			}
			out.WriteString(fmt.Sprintf(width+"s", cNonFinite(v)))
			continue
		}
		switch verb {
		case 'd', 'i':
			out.WriteString(fmt.Sprintf(spec+"d", truncX86(v)))
		case 'f', 'F', 'e', 'E':
			p := prec
			if p < 0 {
				p = 6
			}
			if verb == 'f' || verb == 'F' {
				v = bumpTie(v, p)
			}
			out.WriteString(fmt.Sprintf(spec+string(verb), v))
		case 'g', 'G':
			if prec < 0 {
				spec += ".6"
				prec = 6
			}
			p := max(prec, 1)
			if v != 0 && !math.IsInf(v, 0) && !math.IsNaN(v) {
				exp := int(math.Floor(math.Log10(math.Abs(v))))
				v = bumpTie(v, p-1-exp)
			}
			out.WriteString(fmt.Sprintf(spec+string(verb), v))
		case 's':
			if isNum {
				out.WriteString(fmt.Sprintf(spec+"s", luaToString(v)))
			} else {
				out.WriteString(fmt.Sprintf(spec+"s", a))
			}
		default:
			out.WriteString(fmt.Sprintf(spec+string(verb), a))
		}
	}
	return out.String()
}

// truncX86 is the oracle's x86 cvttsd2si: NaN and anything outside int64 give math.MinInt64, where Go's int64(v)
// gives 0 on arm64 and wasm.
func truncX86(v float64) int64 {
	if math.IsNaN(v) || v >= math.MaxInt64 || v < math.MinInt64 {
		return math.MinInt64
	}
	return int64(v)
}

// cNonFinite is how C's printf spells a non-finite number.
func cNonFinite(v float64) string {
	switch {
	case math.IsNaN(v):
		return "nan"
	case v > 0:
		return "inf"
	}
	return "-inf"
}

func formatArg(a any) (float64, bool) {
	switch v := a.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case bool:
		return 0, false
	}
	return 0, false
}

func bumpTie(v float64, prec int) float64 {
	if prec < 0 || prec > 20 || math.IsNaN(v) || math.IsInf(v, 0) {
		return v
	}
	scaled := float64(math.Abs(v) * math.Pow10(prec))
	if math.Abs(scaled-math.Floor(scaled)-0.5) > 1e-6 {
		return v
	}
	exact := new(big.Float).SetPrec(2048).SetFloat64(math.Abs(v))
	exact.Mul(exact, new(big.Float).SetPrec(2048).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(prec)), nil)))
	whole, _ := exact.Int(nil)
	frac := exact.Sub(exact, new(big.Float).SetPrec(2048).SetInt(whole))
	if frac.Cmp(big.NewFloat(0.5)) != 0 {
		return v
	}
	return math.Nextafter(v, math.Copysign(math.Inf(1), v))
}

// BreakdownSlot is one row of a breakdown's slots table.
type BreakdownSlot struct {
	Base float64
	// Inc and More are "" where the Lua leaves them nil.
	Inc        string
	More       string
	Total      string
	Source     string
	SourceName string
	// Item is actor.itemList[source] when the row names no sourceName.
	Item *Item
}

// BreakdownReservation is one row of a breakdown's reservations table; "" is nil.
type BreakdownReservation struct {
	SkillName      string
	Base           string
	Mult           string
	More           string
	Inc            string
	Efficiency     string
	EfficiencyMore string
	Count          string
	Total          string
}

// BreakdownCol is one colList entry.
type BreakdownCol struct {
	Label string
	Key   string
}

// BreakdownEntry is one breakdown table: its array part is Lines, the rest its named fields.
type BreakdownEntry struct {
	Lines        []string
	Slots        []BreakdownSlot
	Reservations []BreakdownReservation
	ColList      []BreakdownCol
	RowList      []map[string]string
	Label        string
	Footer       string
	// Source marks an ailment breakdown, whose rows CalcBreakdownControl sorts.
	Source    bool
	Radius    float64
	HasRadius bool
	// BreakdownSource is breakdownSource: the breakdown whose Calcs cell the sidebar shows for this one.
	BreakdownSource string
	// DamageTypes is damageTypes: a hit damage breakdown's rows by damage type, a missing key an empty cell.
	DamageTypes []map[string]string
	// ModList is modList: the mods the Calcs tab's SkillBuffs and SkillDebuffs lists show, with their values.
	ModList []BreakdownMod
}

// BreakdownMod is one modList entry; List is a LIST mod's value, Value a number or 1 for a true flag.
type BreakdownMod struct {
	Mod   mod.Mod
	Value float64
	List  any
}

// appendEvalMods appends the mods whose value db evaluates to something other than nil, false or 0.
func appendEvalMods(out []BreakdownMod, db *moddb.ModDB, mods []mod.Mod) []BreakdownMod {
	for _, m := range mods {
		value := db.EvalMod(m, nil)
		switch {
		case value == nil:
		case value.Type() == mod.ModValueMultiTypeFloat:
			if value.Float() != 0 {
				out = append(out, BreakdownMod{Mod: m, Value: value.Float()})
			}
		case value.Type() == mod.ModValueMultiTypeFlag:
			if value.Flag() {
				out = append(out, BreakdownMod{Mod: m, Value: 1})
			}
		default:
			out = append(out, BreakdownMod{Mod: m, Value: 1, List: value.List()})
		}
	}
	return out
}

// Breakdown is the table CalcBreakdown.lua returns: the generators over one actor, and the entries the calc
// modules write into it by stat name.
//
//pob2:src/Modules/CalcBreakdown.lua:16-18
type Breakdown struct {
	modDB  moddb.ModStoreFuncs
	output map[string]float64
	actor  *Actor

	Entries map[string]*BreakdownEntry
	// MainHand and OffHand are breakdown.MainHand/OffHand, the attack passes' own breakdowns.
	MainHand *Breakdown
	OffHand  *Breakdown
}

func newBreakdown(modDB moddb.ModStoreFuncs, output map[string]float64, actor *Actor) *Breakdown {
	return &Breakdown{modDB: modDB, output: output, actor: actor, Entries: map[string]*BreakdownEntry{}}
}

// set is `breakdown[key] = { lines... }`.
func (b *Breakdown) set(key string, lines ...string) *BreakdownEntry {
	e := &BreakdownEntry{Lines: lines}
	b.Entries[key] = e
	return e
}

// setEntry is `breakdown[key] = e`; a nil e clears it.
func (b *Breakdown) setEntry(key string, e *BreakdownEntry) {
	if e == nil {
		delete(b.Entries, key)
		return
	}
	b.Entries[key] = e
}

// insert is t_insert(breakdown[key], line); Lua errors on a nil table, so the entry must exist.
func (b *Breakdown) insert(key string, lines ...string) {
	e := b.Entries[key]
	e.Lines = append(e.Lines, lines...)
}

// simple is breakdown.simple: the base of the first name with extraBase, then the increased and more multipliers,
// nil where nothing scales it.
//
//pob2:src/Modules/CalcBreakdown.lua:47-70
func (b *Breakdown) simple(extraBase float64, cfg *moddb.ListCfg, total float64, names ...string) *BreakdownEntry {
	base := b.modDB.Sum(mod.TypeBase, cfg, names[0])
	if base+extraBase != 0 {
		inc := b.modDB.Sum(mod.TypeIncrease, cfg, names...)
		more := b.modDB.More(cfg, names...)
		if inc != 0 || more != 1 || (base != 0 && extraBase != 0) {
			var out []string
			if base != 0 && extraBase != 0 {
				out = append(out, sFormat("(%g + %g) ^8(base)", extraBase, base))
			} else {
				out = append(out, sFormat("%g ^8(base)", base+extraBase))
			}
			if inc != 0 {
				out = append(out, sFormat("x %.2f ^8(increased/reduced)", 1+inc/100))
			}
			if more != 1 {
				out = append(out, sFormat("x %.2f ^8(more/less)", more))
			}
			out = append(out, sFormat("= %g", total))
			return &BreakdownEntry{Lines: out}
		}
	}
	return nil
}

// areaBreakpoints are calcRadiusBreakpoints' four results; a nil *areaBreakpoints is their nil.
type areaBreakpoints struct{ inc, more, red, less float64 }

// area is breakdown.area; label "" is nil, radiusMod 1 is nil.
//
//pob2:src/Modules/CalcBreakdown.lua:98-115
func (b *Breakdown) area(base, areaMod, total float64, bp *areaBreakpoints, label string, radiusMod float64) *BreakdownEntry {
	var out []string
	if label != "" {
		out = append(out, label)
	}
	if base != total {
		out = append(out, sFormat("%.1fm ^8(base radius)", base/10))
		out = append(out, sFormat("x %.2f ^8(square root of area of effect modifier)", math.Floor(100*math.Sqrt(areaMod))/100))
		if radiusMod != 1 {
			out = append(out, sFormat("x %.2f ^8(final radius modifier)", radiusMod))
		}
		out = append(out, sFormat("= %.1fm", total/10))
	}
	if bp != nil {
		out = append(out, sFormat("^8Next 0.1m breakpoint: %d%% increased AoE / a %d%% more AoE multiplier", bp.inc, bp.more))
		out = append(out, sFormat("^8Previous 0.1m breakpoint: %d%% reduced AoE / a %d%% less AoE multiplier", bp.red, bp.less))
	}
	return &BreakdownEntry{Lines: out, Radius: total, HasRadius: true}
}

// leech is breakdown.leech. The Lua reads a skillCfg it never defines, so the rate modifier has no cfg.
//
//pob2:src/Modules/CalcBreakdown.lua:209-263
func (b *Breakdown) leech(instant, instantRate, instances, pool float64, rate string, maxRate, dur, instantLeechProportion, hitRate float64) *BreakdownEntry {
	var out []string
	leechRateBase := miscConstant("LeechRateBase")
	if truthy(b.actor.MainSkill.SkillData["showAverage"]) {
		if instant > 0 {
			if instantLeechProportion != 1 {
				out = append(out, sFormat("Instant Leech: %.1f ^8(%d%% x %.1f)", instant, instantLeechProportion*100, dur*pool*leechRateBase/(1-instantLeechProportion)))
			} else {
				out = append(out, sFormat("Instant Leech: %.1f", instant))
			}
		}
		if instances > 0 {
			out = append(out, "Total leeched per instance:",
				sFormat("%d ^8(size of leech destination pool)", pool),
				sFormat("x %.2f ^8(base leech rate is %d%% per second)", leechRateBase, 100*leechRateBase))
			rateMod := calcLibMod(b.modDB, nil, rate)
			if rateMod != 1 {
				out = append(out, sFormat("x %.2f ^8(leech rate modifier)", rateMod))
			}
			out = append(out, sFormat("x %.2fs ^8(instance duration)", dur), sFormat("= %.1f", pool*leechRateBase*rateMod*dur))
		}
	} else {
		if instantRate > 0 {
			if instantLeechProportion != 1 {
				out = append(out, sFormat("Instant Leech: %.1f ^8(%d%% x %.1f)", instant, instantLeechProportion*100, dur*pool*leechRateBase/(1-instantLeechProportion)))
			} else {
				out = append(out, sFormat("Instant Leech: %.1f", instant))
			}
			out = append(out, sFormat("Instant Leech per second: %.1f ^8(%.1f x %.2f)", instantRate, instant, hitRate))
		}
		if instances > 0 {
			out = append(out, "Rate per instance:",
				sFormat("%d ^8(size of leech destination pool)", pool),
				sFormat("x %.2f ^8(base leech rate is %d%% per second)", leechRateBase, 100*leechRateBase))
			rateMod := calcLibMod(b.modDB, nil, rate)
			if rateMod != 1 {
				out = append(out, sFormat("x %.2f ^8(leech rate modifier)", rateMod))
			}
			out = append(out, sFormat("= %.1f ^8per second", pool*leechRateBase*rateMod),
				"Maximum leech rate against one target:",
				sFormat("%.1f", pool*leechRateBase*rateMod),
				sFormat("x %.2f ^8(average instances)", instances))
			total := pool * leechRateBase * rateMod * instances
			out = append(out, sFormat("= %.1f ^8per second", total))
			if total <= maxRate {
				out = append(out, sFormat("Time to reach max: %.1fs", dur))
			}
			out = append(out, sFormat("Leech rate cap: %.1f", maxRate))
			if total > maxRate {
				out = append(out, sFormat("Time to reach cap: %.1fs", dur/total*maxRate))
			}
		}
	}
	return &BreakdownEntry{Lines: out}
}

// mod is breakdown.mod: the increased and more multipliers of names, nil where either is neutral.
//
//pob2:src/Modules/CalcBreakdown.lua:72-82
func (b *Breakdown) mod(modList moddb.ModStoreFuncs, cfg *moddb.ListCfg, names ...string) *BreakdownEntry {
	inc := modList.Sum(mod.TypeIncrease, cfg, names...)
	more := modList.More(cfg, names...)
	if inc != 0 && more != 1 {
		return &BreakdownEntry{Lines: []string{
			sFormat("%.2f ^8(increased/reduced)", 1+inc/100),
			sFormat("x %.2f ^8(more/less)", more),
			sFormat("= %.2f", (1+inc/100)*more),
		}}
	}
	return nil
}

// chainLink is one { format, value, args... } multiChain entry; a nil Value is the Lua's nil.
type chainLink struct {
	Format string
	Value  *float64
	Args   []any
}

func (l chainLink) format() string {
	return sFormat(l.Format, append([]any{*l.Value}, l.Args...)...)
}

func link(format string, value float64) chainLink {
	return chainLink{Format: format, Value: &value}
}

type multiChainSpec struct {
	Label   string
	Base    *chainLink
	Links   []chainLink
	Total   string
	NoTotal bool
}

//pob2:src/Modules/CalcBreakdown.lua:20-45
func (b *Breakdown) multiChain(out []string, chain multiChainSpec) ([]string, int) {
	var base *float64
	if chain.Base != nil {
		base = chain.Base.Value
	}
	multiplier := 1.0
	lines := 0
	if chain.Label != "" {
		out = append(out, chain.Label)
	}
	if base != nil {
		out = append(out, chain.Base.format())
	}
	if base == nil || *base != 0 {
		for _, mult := range chain.Links {
			if mult.Value != nil && *mult.Value != 1 {
				multiplier *= *mult.Value
				out = append(out, "x "+mult.format())
				lines++
			}
		}
	}
	if chain.Total != "" {
		out = append(out, chain.Total)
	} else if !chain.NoTotal && (lines > 0 && base != nil) || (lines > 1 && base == nil) {
		b := 1.0
		if base != nil {
			b = *base
		}
		out = append(out, sFormat("= %.2f", multiplier*b))
	}
	return out, lines
}

// luaMin is LuaJIT's math.min of two numbers: b where a is NaN, NaN where only b is.
func luaMin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// effMult is breakdown.effMult; Lua's optional minPen, effectiveResist, normalHitResist and invertedHitResist are
// always given here.
//
//pob2:src/Modules/CalcBreakdown.lua:117-171
func (b *Breakdown) effMult(damageType string, resist, pen, taken, mult, takenMore float64, sourceRes string, useRes bool, invertChance, minPen, effectiveResist, normalHitResist, invertedHitResist float64) []string {
	out := []string{}
	resistForm := "resistance"
	if damageType == "Physical" {
		resistForm = "physical damage reduction"
	}
	resistLabel := resistForm

	if sourceRes != "" && sourceRes != damageType {
		out = append(out, sFormat("Enemy %s: %d%% ^8(%s)", resistLabel, resist, sourceRes))
	} else if resist != 0 {
		out = append(out, sFormat("Enemy %s: %d%%", resistLabel, resist))
	}
	if invertChance != 0 && useRes {
		out = append(out, "Effective resistance:",
			sFormat("%g%% ^8(non-inverted hit after penetration)", normalHitResist),
			sFormat("%g%% ^8(inverted hit after penetration)", invertedHitResist),
			sFormat("= %g%% ^8(weighted average from %.0f%% inversion chance)", effectiveResist, invertChance*100))
	} else if pen != 0 || !useRes {
		out = append(out, "Effective resistance:", sFormat("%d%% ^8(resistance)", resist))
		if pen < 0 {
			out = append(out, sFormat("+ %d%% ^8(penetration)", -pen))
		} else if pen > 0 {
			out = append(out, sFormat("- %d%% ^8(penetration)", pen))
		}
		switch {
		case !useRes:
			out = append(out, sFormat("x %d%% ^8(resistance ignored)", 0), sFormat("= %d%%", 0))
		case resist <= minPen:
			out = append(out, sFormat("= %d%% ^8(negative resistance unaffected by penetration)", resist))
		case resist-pen < minPen:
			out = append(out, sFormat("= %d%% ^8(penetration cannot bring resistances below %d%%)", math.Max(resist-pen, minPen), minPen))
		default:
			out = append(out, sFormat("= %d%%", resist-pen))
		}
	}
	if useRes {
		r := link("%.2f ^8(%s)", 1-effectiveResist/100)
		r.Args = []any{resistForm}
		out, _ = b.multiChain(out, multiChainSpec{
			Label: "Effective DPS modifier:",
			Links: []chainLink{r, link("%.2f ^8(increased/reduced damage taken)", 1+taken/100), link("%.2f ^8(more/less damage taken)", takenMore)},
			Total: sFormat("= %.3f", mult),
		})
	} else {
		out = append(out, "Effective DPS modifier:", sFormat("= %.3f ^8(increased/reduced damage taken)", mult))
	}
	return out
}

//pob2:src/Modules/CalcBreakdown.lua:84-96
func (b *Breakdown) slot(source, sourceName string, cfg *moddb.ListCfg, base float64, total *float64, names ...string) {
	inc := b.modDB.Sum(mod.TypeIncrease, cfg, names...)
	more := b.modDB.More(cfg, names...)
	row := BreakdownSlot{Base: base, Source: source, SourceName: sourceName}
	if inc != 0 {
		row.Inc = sFormat(" x %.2f", 1+inc/100)
	}
	if more != 1 {
		row.More = sFormat(" x %.2f", more)
	}
	if total != nil {
		row.Total = sFormat("%.2f", *total)
	} else {
		row.Total = sFormat("%.2f", base*(1+inc/100)*more)
	}
	if sourceName == "" {
		row.Item = b.actor.ItemList[source]
	}
	e := b.Entries[names[0]]
	e.Slots = append(e.Slots, row)
}

// rawSlot is a slots row the Lua builds inline rather than through breakdown.slot.
func rawSlot(base, inc, more float64, source string, item *Item) BreakdownSlot {
	row := BreakdownSlot{Base: base, Total: sFormat("%.2f", base*(1+inc/100)*more), Source: source, Item: item}
	if inc != 0 {
		row.Inc = sFormat(" x %.2f", 1+inc/100)
	}
	if more != 1 {
		row.More = sFormat(" x %.2f", more)
	}
	return row
}
