package calculator

import (
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

func luaSprintf(format string, args ...any) string {
	arg := 0
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
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
		if i >= len(format) || format[i] == '%' {
			continue
		}
		if arg < len(args) && (format[i] == 'f' || format[i] == 'F') {
			if v, ok := args[arg].(float64); ok {
				if prec < 0 {
					prec = 6
				}
				if isDecimalTie(v, prec) {
					args[arg] = math.Nextafter(v, math.Copysign(math.Inf(1), v))
				}
			}
		}
		arg++
	}
	return fmt.Sprintf(format, args...)
}

func isDecimalTie(v float64, prec int) bool {
	if math.IsNaN(v) || math.IsInf(v, 0) || prec > 20 {
		return false
	}
	scaled := float64(math.Abs(v) * math.Pow10(prec))
	if math.Abs(scaled-math.Floor(scaled)-0.5) > 1e-6 {
		return false
	}
	exact := new(big.Float).SetPrec(2048).SetFloat64(math.Abs(v))
	exact.Mul(exact, new(big.Float).SetPrec(2048).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(prec)), nil)))
	whole, _ := exact.Int(nil)
	frac := exact.Sub(exact, new(big.Float).SetPrec(2048).SetInt(whole))
	return frac.Cmp(big.NewFloat(0.5)) == 0
}

type BCol struct {
	Label string
	Key   string
}

type BSlot struct {
	Base       float64
	Inc        string
	More       string
	Total      string
	Source     string
	SourceName *string
	Item       *ItemData
}

type BMultiChainItem struct {
	Format string
	Value  float64
}

type BMultiChain struct {
	Base    *BMultiChainItem
	Label   string
	Total   string
	NoTotal bool
	Items   []BMultiChainItem
}

type BReservation struct {
	SkillName      string
	Base           string
	Mult           *string
	More           *string
	Inc            *string
	Efficiency     *string
	EfficiencyMore *string
	Total          string
}

type BDamageType struct {
	Source  string
	ConvSrc *string
	Total   string
	ConvDst *string
	GainDst *string
	Base    *string
	Inc     *string
	More    *string
}

type Entry struct {
	Columns         []BCol
	Rows            []map[string]string
	Lines           []string
	Slots           []BSlot
	Reservations    []BReservation
	Label           string
	DamageTypes     []BDamageType
	Radius          float64
	BreakdownSource string
}

type Breakdown struct {
	ModDB  *moddb.ModDB
	Output map[string]float64
	Actor  *Actor

	Data map[string]*Entry

	mainHand *Breakdown
	offHand  *Breakdown
	modStore moddb.ModStoreFuncs
}

//pob:block:src/Modules/CalcBreakdown.lua:13-15
func NewBreakdown(modDB *moddb.ModDB, output map[string]float64, actor *Actor) *Breakdown {
	return &Breakdown{
		ModDB:  modDB,
		Output: output,
		Actor:  actor,
		Data:   make(map[string]*Entry),
	}
}

func (b *Breakdown) AddCol(key string, cols ...BCol) {
	if _, ok := b.Data[key]; !ok {
		b.Data[key] = &Entry{}
	}

	b.Data[key].Columns = append(b.Data[key].Columns, cols...)
}

func (b *Breakdown) AddRow(key string, rows ...map[string]string) {
	if _, ok := b.Data[key]; !ok {
		b.Data[key] = &Entry{}
	}

	b.Data[key].Rows = append(b.Data[key].Rows, rows...)
}

func (b *Breakdown) AddLine(key string, s ...string) {
	if _, ok := b.Data[key]; !ok {
		b.Data[key] = &Entry{}
	}

	b.Data[key].Lines = append(b.Data[key].Lines, s...)
}

var goNonFiniteRe = regexp.MustCompile(`[+-]Inf|NaN`)

func cNonFinite(s string) string {
	return goNonFiniteRe.ReplaceAllStringFunc(s, func(m string) string {
		switch m {
		case "+Inf":
			return "inf"
		case "-Inf":
			return "-inf"
		}
		return "nan"
	})
}

func (b *Breakdown) cFloatSpelling() {
	if b == nil {
		return
	}
	b.mainHand.cFloatSpelling()
	b.offHand.cFloatSpelling()
	for _, e := range b.Data {
		for i, l := range e.Lines {
			e.Lines[i] = cNonFinite(l)
		}
		for _, row := range e.Rows {
			for k, v := range row {
				row[k] = cNonFinite(v)
			}
		}
	}
}

//pob:src/Modules/CalcBreakdown.lua:95-112
func (b *Breakdown) area(key string, base float64, areaMod float64, total float64, bp *radiusBreakpoints, label string, radiusMod float64) {
	var out []string
	if label != "" {
		out = append(out, label)
	}
	if base != total {
		out = append(out,
			luaSprintf("%.1fm ^8(base radius)", base/10),
			luaSprintf("x %.2f ^8(square root of area of effect modifier)", math.Floor(100*math.Sqrt(areaMod))/100),
		)
		if radiusMod != 1 {
			out = append(out, luaSprintf("x %.2f ^8(final radius modifier)", radiusMod))
		}
		out = append(out, luaSprintf("= %.1fm", total/10))
	}
	if bp != nil {
		out = append(out,
			luaSprintf("^8Next 0.1m breakpoint: %d%% increased AoE / a %d%% more AoE multiplier", bp.inc, bp.more),
			luaSprintf("^8Previous 0.1m breakpoint: %d%% reduced AoE / a %d%% less AoE multiplier", bp.red, bp.less),
		)
	}
	b.Data[key] = &Entry{Lines: out, Radius: total}
}

//pob:src/Modules/CalcBreakdown.lua:162-173
func (b *Breakdown) Dot(key string, baseVal, inc, more, mult, rate, aura, effMult, total float64) {
	b.Data[key] = &Entry{}
	b.MultiChain(key, BMultiChain{
		Base: &BMultiChainItem{Format: "%.1f ^8(base damage per second)", Value: baseVal},
		Items: []BMultiChainItem{
			{Format: "%.2f ^8(increased/reduced)", Value: 1 + inc/100},
			{Format: "%.2f ^8(more/less)", Value: more},
			{Format: "%.2f ^8(multiplier)", Value: 1 + mult/100},
			{Format: "%.2f ^8(rate modifier)", Value: rate},
			{Format: "%.3f ^8(aura effect modifier)", Value: aura},
			{Format: "%.3f ^8(effective DPS modifier)", Value: effMult},
		},
		Total: luaSprintf("= %.1f ^8per second", total),
	})
}

//pob:src/Modules/CalcBreakdown.lua:175-196
func breakdownCritDot(dotMulti, critMulti, dotChance, critChance float64) []string {
	combined := float64(dotMulti*dotChance) + float64(critMulti*critChance)
	var out []string
	if dotChance > 0 {
		out = append(out,
			"Contribution from Non-crits:",
			luaSprintf("%.2f ^8(dot multiplier for non-crits)", dotMulti),
			luaSprintf("x %.4f ^8(portion of instances created by non-crits)", dotChance),
			luaSprintf("= %.2f", dotMulti*dotChance),
		)
	}
	if critChance > 0 {
		out = append(out,
			"Contribution from Crits:",
			luaSprintf("%.2f ^8(dot multiplier for crits)", critMulti),
			luaSprintf("x %.4f ^8(portion of instances created by crits)", critChance),
			luaSprintf("= %.2f", critMulti*critChance),
		)
	}
	if (dotChance > 0 && critChance > 0) && (dotMulti != critMulti) {
		out = append(out,
			"Effective DoT Multiplier:",
			luaSprintf("%.2f + %.2f", dotMulti*dotChance, critMulti*critChance),
			luaSprintf("= %.2f", combined),
		)
	}
	return out
}

//pob:src/Modules/CalcBreakdown.lua:198-252
func (b *Breakdown) leech(instant, instantRate, instances, pool float64, rate string, maxRate, dur, instantLeechProportion, hitRate float64) []string {
	var out []string
	if b.Actor.MainSkill.SkillData.ShowAverage {
		if instant > 0 {
			if instantLeechProportion != 1 {
				out = append(out, luaSprintf("Instant Leech: %.1f ^8(%d%% x %.1f)", instant, luaInt(instantLeechProportion*100), dur*pool*data.LeechRateBase/(1-instantLeechProportion)))
			} else {
				out = append(out, luaSprintf("Instant Leech: %.1f", instant))
			}
		}
		if instances > 0 {
			out = append(out,
				"Total leeched per instance:",
				luaSprintf("%d ^8(size of leech destination pool)", luaInt(pool)),
				luaSprintf("x %.2f ^8(base leech rate is %d%% per second)", data.LeechRateBase, luaInt(100*data.LeechRateBase)),
			)
			rateMod := CalcMod(b.store(), nil, rate)
			if rateMod != 1 {
				out = append(out, luaSprintf("x %.2f ^8(leech rate modifier)", rateMod))
			}
			out = append(out,
				luaSprintf("x %.2fs ^8(instance duration)", dur),
				luaSprintf("= %.1f", pool*data.LeechRateBase*rateMod*dur),
			)
		}
	} else {
		if instantRate > 0 {
			if instantLeechProportion != 1 {
				out = append(out, luaSprintf("Instant Leech: %.1f ^8(%d%% x %.1f)", instant, luaInt(instantLeechProportion*100), dur*pool*data.LeechRateBase/(1-instantLeechProportion)))
			} else {
				out = append(out, luaSprintf("Instant Leech: %.1f", instant))
			}
			out = append(out, luaSprintf("Instant Leech per second: %.1f ^8(%.1f x %.2f)", instantRate, instant, hitRate))
		}
		if instances > 0 {
			out = append(out,
				"Rate per instance:",
				luaSprintf("%d ^8(size of leech destination pool)", luaInt(pool)),
				luaSprintf("x %.2f ^8(base leech rate is %d%% per second)", data.LeechRateBase, luaInt(100*data.LeechRateBase)),
			)
			rateMod := CalcMod(b.store(), nil, rate)
			if rateMod != 1 {
				out = append(out, luaSprintf("x %.2f ^8(leech rate modifier)", rateMod))
			}
			out = append(out,
				luaSprintf("= %.1f ^8per second", pool*data.LeechRateBase*rateMod),
				"Maximum leech rate against one target:",
				luaSprintf("%.1f", pool*data.LeechRateBase*rateMod),
				luaSprintf("x %.2f ^8(average instances)", instances),
			)
			total := pool * data.LeechRateBase * rateMod * instances
			out = append(out, luaSprintf("= %.1f ^8per second", total))
			if total <= maxRate {
				out = append(out, luaSprintf("Time to reach max: %.1fs", dur))
			}
			out = append(out, luaSprintf("Leech rate cap: %.1f", maxRate))
			if total > maxRate {
				out = append(out, luaSprintf("Time to reach cap: %.1fs", dur/total*maxRate))
			}
		}
	}
	return out
}

func (b *Breakdown) GetData() map[string]*Entry {
	if b.mainHand == nil && b.offHand == nil {
		return b.Data
	}
	out := make(map[string]*Entry, len(b.Data))
	for k, e := range b.Data {
		out[k] = e
	}
	for _, hand := range []struct {
		prefix string
		bd     *Breakdown
	}{{"MainHand.", b.mainHand}, {"OffHand.", b.offHand}} {
		if hand.bd != nil {
			for k, e := range hand.bd.Data {
				out[hand.prefix+k] = e
			}
		}
	}
	return out
}

//pob:block:src/Modules/CalcOffence.lua:2006,src/Modules/CalcOffence.lua:2023
func newHandBreakdown(skillModList moddb.ModStoreFuncs, output map[string]float64) *Breakdown {
	return &Breakdown{Output: output, Data: make(map[string]*Entry), modStore: skillModList}
}

func (b *Breakdown) store() moddb.ModStoreFuncs {
	if b.modStore != nil {
		return b.modStore
	}
	return b.ModDB
}

func (b *Breakdown) AddSlot(key string, slot ...BSlot) {
	if _, ok := b.Data[key]; !ok {
		b.Data[key] = &Entry{}
	}

	b.Data[key].Slots = append(b.Data[key].Slots, slot...)
}

//pob:src/Modules/CalcBreakdown.lua:81-93
func (b *Breakdown) Slot(source string, sourceName *string, cfg *moddb.ListCfg, base float64, total *float64, keys ...string) {
	inc := b.store().Sum("INC", cfg, keys...)
	more := b.store().More(cfg, keys...)

	for _, key := range keys {
		b.AddSlot(key, BSlot{
			Base:       base,
			Inc:        utils.Ternary(inc != 0, luaSprintf(" x %.2f", 1+inc/100), ""),
			More:       utils.Ternary(more != 1, luaSprintf(" x %.2f", more), ""),
			Total:      luaSprintf("%.2f", utils.UnwrapOrF(total, base*(1+inc/100)*more)),
			Source:     source,
			SourceName: sourceName,
			Item:       b.Actor.ItemList[source],
		})
	}
}

//pob:src/Modules/CalcBreakdown.lua:44-67
func (b *Breakdown) Simple(extraBasePtr *float64, cfg *moddb.ListCfg, total float64, key string) {
	extraBase := utils.UnwrapOrF(extraBasePtr, 0)
	base := b.store().Sum(mod.TypeBase, cfg, key)
	if base+extraBase != 0 {
		inc := b.store().Sum(mod.TypeIncrease, cfg, key)
		more := b.store().More(cfg, key)
		if inc != 0 || more != 1 || (base != 0 && extraBase != 0) {
			lines := make([]string, 0)

			if base != 0 && extraBase != 0 {
				lines = append(lines, luaSprintf("(%g + %g) ^8(base)", extraBase, base))
			} else {
				lines = append(lines, luaSprintf("%g ^8(base)", base+extraBase))
			}

			if inc != 0 {
				lines = append(lines, luaSprintf("x %.2f ^8(increased/reduced)", 1+inc/100))
			}

			if more != 1 {
				lines = append(lines, luaSprintf("x %.2f ^8(more/less)", more))
			}
			lines = append(lines, luaSprintf("= %g", total))

			b.Data[key] = &Entry{Lines: lines}
			return
		}
	}
	delete(b.Data, key)
}

//pob:src/Modules/CalcBreakdown.lua:17-42
func (b *Breakdown) MultiChain(key string, chains ...BMultiChain) {
	for _, chain := range chains {
		multiplier := float64(1)
		lines := 0
		if chain.Label != "" {
			b.AddLine(key, chain.Label)
		}
		if chain.Base != nil {
			b.AddLine(key, luaSprintf(chain.Base.Format, chain.Base.Value))
		}
		if chain.Base == nil || chain.Base.Value != 0 {
			for _, mult := range chain.Items {
				if mult.Value != 1 {
					multiplier *= mult.Value
					b.AddLine(key, "x "+luaSprintf(mult.Format, mult.Value))
					lines++
				}
			}
		}
		switch {
		case chain.Total != "":
			b.AddLine(key, chain.Total)
		case (!chain.NoTotal && lines > 0 && chain.Base != nil) || (lines > 1 && chain.Base == nil):
			base := float64(1)
			if chain.Base != nil {
				base = chain.Base.Value
			}
			b.AddLine(key, luaSprintf("= %.2f", multiplier*base))
		}
	}
}

//pob:src/Modules/CalcBreakdown.lua:69-79
func (b *Breakdown) Mod(key string, modList moddb.ModStoreFuncs, cfg *moddb.ListCfg, names ...string) {
	inc := modList.Sum(mod.TypeIncrease, cfg, names...)
	more := modList.More(cfg, names...)
	if inc != 0 || more != 1 {
		b.Data[key] = &Entry{Lines: []string{
			luaSprintf("%.2f ^8(increased/reduced)", 1+inc/100),
			luaSprintf("x %.2f ^8(more/less)", more),
			luaSprintf("= %.2f", (1+inc/100)*more),
		}}
		return
	}
	delete(b.Data, key)
}

//pob:src/Modules/CalcBreakdown.lua:114-160
func (b *Breakdown) EffMult(key string, damageType data.DamageType, resist float64, pen float64, taken float64, mult float64, takenMore float64, sourceRes data.DamageType, useRes bool, invertChance float64, effectiveResist *float64) {
	b.Data[key] = &Entry{}
	resistForm := utils.Ternary(damageType == data.DamageTypePhysical, "physical damage reduction", "resistance")
	effRes := resist - pen
	if effectiveResist != nil {
		effRes = *effectiveResist
	}

	if sourceRes != "" && sourceRes != damageType {
		b.AddLine(key, luaSprintf("Enemy %s: %d%% ^8(%s)", resistForm, luaInt(resist), sourceRes))
	} else if resist != 0 {
		b.AddLine(key, luaSprintf("Enemy %s: %d%%", resistForm, luaInt(resist)))
	}

	switch {
	case invertChance != 0 && useRes:
		normalResist := resist - pen
		invertedResist := -resist - pen
		b.AddLine(key, "Effective resistance:")
		b.AddLine(key, luaSprintf("%.6g%% ^8(non-inverted hit after penetration)", normalResist))
		b.AddLine(key, luaSprintf("%.6g%% ^8(inverted hit after penetration)", invertedResist))
		b.AddLine(key, luaSprintf("= %.6g%% ^8(weighted average from %.0f%% inversion chance)", effRes, invertChance*100))
	case pen != 0 || !useRes:
		b.AddLine(key, "Effective resistance:")
		b.AddLine(key, luaSprintf("%d%% ^8(resistance)", luaInt(resist)))
		if pen < 0 {
			b.AddLine(key, luaSprintf("+ %d%% ^8(penetration)", luaInt(-pen)))
		} else if pen > 0 {
			b.AddLine(key, luaSprintf("- %d%% ^8(penetration)", luaInt(pen)))
		}

		if !useRes {
			b.AddLine(key, luaSprintf("x %d%% ^8(resistance ignored)", 0))
			b.AddLine(key, luaSprintf("= %d%%", 0))
		} else {
			b.AddLine(key, luaSprintf("= %d%%", luaInt(resist-pen)))
		}
	}

	if useRes {
		b.MultiChain(key, BMultiChain{
			Label: "Effective DPS modifier:",
			Total: luaSprintf("= %.3f", mult),
			Items: []BMultiChainItem{
				{"%.2f ^8(" + resistForm + ")", 1 - effRes/100},
				{"%.2f ^8(increased/reduced damage taken)", 1 + taken/100},
				{"%.2f ^8(more/less damage taken)", takenMore},
			},
		})
	} else {
		b.AddLine(key, "Effective DPS modifier:")
		b.AddLine(key, luaSprintf("= %.3f ^8(increased/reduced damage taken)", mult))
	}
}

func (b *Breakdown) Reservation(key string, reservation ...BReservation) {
	if _, ok := b.Data[key]; !ok {
		b.Data[key] = &Entry{}
	}

	b.Data[key].Reservations = append(b.Data[key].Reservations, reservation...)
}

func (b *Breakdown) SetLabel(key string, label string) {
	if _, ok := b.Data[key]; !ok {
		b.Data[key] = &Entry{}
	}

	b.Data[key].Label = label
}

func (b *Breakdown) DamageType(key string, damageTypes ...BDamageType) {
	if _, ok := b.Data[key]; !ok {
		b.Data[key] = &Entry{}
	}

	b.Data[key].DamageTypes = append(b.Data[key].DamageTypes, damageTypes...)
}

func (b *Breakdown) Has(key string) bool {
	_, ok := b.Data[key]
	return ok
}

func setBreakdownEntry(b *Breakdown, key string, entries ...*Entry) {
	for _, e := range entries {
		if e != nil {
			b.Data[key] = e
			return
		}
	}
	delete(b.Data, key)
}
