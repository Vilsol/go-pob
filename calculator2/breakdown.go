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
			out.WriteString(fmt.Sprintf(spec+"d", int64(v)))
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
	scaled := math.Abs(v) * math.Pow10(prec)
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
