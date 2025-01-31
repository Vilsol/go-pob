package calculator

import (
	"fmt"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/utils"
)

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
	Base  string
	Label string
	Total string
	Items []BMultiChainItem
}

type BReservation struct {
	SkillName  string
	Base       string
	Mult       *string
	More       *string
	Inc        *string
	Efficiency *string
	Total      string
}

type BDamageType struct {
	Source  string
	ConvSrc *string
	Total   string
	ConvDst *string
	Base    *string
	Inc     *string
	More    *string
}

type Entry struct {
	Columns      []BCol
	Rows         []map[string]string
	Lines        []string
	Slots        []BSlot
	Reservations []BReservation
	Label        string
	DamageTypes  []BDamageType
}

type Breakdown struct {
	ModDB  *moddb.ModDB
	Output map[string]float64
	Actor  *Actor

	Data map[string]*Entry
}

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

func (b *Breakdown) GetData() map[string]*Entry {
	return b.Data
}

func (b *Breakdown) AddSlot(key string, slot ...BSlot) {
	if _, ok := b.Data[key]; !ok {
		b.Data[key] = &Entry{}
	}

	b.Data[key].Slots = append(b.Data[key].Slots, slot...)
}

func (b *Breakdown) Slot(source string, sourceName *string, cfg *moddb.ListCfg, base float64, total *float64, keys ...string) {
	inc := b.ModDB.Sum("INC", cfg, keys...)
	more := b.ModDB.More(cfg, keys...)

	for _, key := range keys {
		b.AddSlot(key, BSlot{
			Base:       base,
			Inc:        utils.Ternary(inc != 0, fmt.Sprintf(" x %.2f", 1+inc/100), ""),
			More:       utils.Ternary(more != 1, fmt.Sprintf(" x %.2f", more), ""),
			Total:      fmt.Sprintf("%.2f", utils.Ternary(total != nil, *total, base*(1+inc/100)*more)),
			Source:     source,
			SourceName: sourceName,
			Item:       b.Actor.ItemList[source],
		})
	}
}

func (b *Breakdown) Simple(extraBasePtr *float64, cfg *moddb.ListCfg, total float64, key string) {
	extraBase := utils.Ternary(extraBasePtr != nil, *extraBasePtr, 0)
	base := b.ModDB.Sum(mod.TypeBase, cfg, key)
	if base+extraBase != 0 {
		inc := b.ModDB.Sum(mod.TypeIncrease, cfg, key)
		more := b.ModDB.More(cfg, key)
		if inc != 0 || more != 1 || (base != 0 && extraBase != 0) {
			lines := make([]string, 0)

			if base != 0 && extraBase != 0 {
				lines = append(lines, fmt.Sprintf("(%g + %g) ^8(base)", extraBase, base))
			} else {
				lines = append(lines, fmt.Sprintf("%g ^8(base)", base+extraBase))
			}

			if inc != 0 {
				lines = append(lines, fmt.Sprintf("x %.2f ^8(increased/reduced)", 1+inc/100))
			}

			if more != 1 {
				lines = append(lines, fmt.Sprintf("x %.2f ^8(more/less)", more))
			}

			b.AddLine(key, lines...)
		}
	}
}

func (b *Breakdown) MultiChain(key string, chains ...BMultiChain) {
	for _, chain := range chains {
		base := chain.Base
		lines := 0
		for _, mult := range chain.Items {
			if mult.Value != 1 {
				if lines == 0 {
					if base != "" {
						if chain.Label != "" {
							b.AddLine(key, chain.Label)
						}
						b.AddLine(key, base)
						b.AddLine(key, "x "+fmt.Sprintf(mult.Format, mult.Value))
						lines = 2
					} else {
						base = fmt.Sprintf(mult.Format, mult.Value)
					}
				} else {
					b.AddLine(key, "x "+fmt.Sprintf(mult.Format, mult.Value))
					lines = lines + 1
				}
			}
			if lines > 0 {
				b.AddLine(key, chain.Total)
			}
		}
	}
}

func (b *Breakdown) Mod(key string, modList moddb.ModStoreFuncs, cfg *moddb.ListCfg, names ...string) {
	inc := modList.Sum(mod.TypeIncrease, cfg, names...)
	more := modList.More(cfg, names...)
	if inc != 0 && more != 1 {
		b.AddLine(
			key,
			fmt.Sprintf("%.2f ^8(increased/reduced)", 1+inc/100),
			fmt.Sprintf("x %.2f ^8(more/less)", more),
			fmt.Sprintf("= %.2f", (1+inc/100)*more),
		)
	}
}

func (b *Breakdown) EffMult(key string, damageType data.DamageType, resist float64, pen float64, taken float64, mult float64, takenMore float64, sourceRes data.DamageType, useRes bool) {
	resistForm := utils.Ternary(damageType == data.DamageTypePhysical, "physical damage reduction", "resistance")
	if sourceRes != "" && sourceRes != damageType {
		b.AddLine(key, fmt.Sprintf("Enemy %s: %.2f%% ^8(%s)", resistForm, resist, sourceRes))
	} else if resist != 0 {
		b.AddLine(key, fmt.Sprintf("Enemy %s: %.2f%%", resistForm, resist))
	}

	if pen != 0 || !useRes {
		b.AddLine(key, "Effective resistance:")
		b.AddLine(key, fmt.Sprintf("%.2f%% ^8(resistance)", resist))
		if pen < 0 {
			b.AddLine(key, fmt.Sprintf("+ %.2f%% ^8(penetration)", -pen))
		} else if pen > 0 {
			b.AddLine(key, fmt.Sprintf("- %.2f%% ^8(penetration)", pen))
		}

		if !useRes {
			b.AddLine(key, fmt.Sprintf("x %d%% ^8(resistance ignored)", 0))
			b.AddLine(key, fmt.Sprintf("= %d%%", (0)))
		} else {
			b.AddLine(key, fmt.Sprintf("= %.2f%%", (resist-pen)))
		}
	}

	if useRes {
		b.MultiChain(key, BMultiChain{
			Label: "Effective DPS modifier:",
			Total: fmt.Sprintf("= %.3f", mult),
			Items: []BMultiChainItem{
				{"%.2f ^8(" + resistForm + ")", 1 - (resist-pen)/100},
				{"%.2f ^8(increased/reduced damage taken)", 1 + taken/100},
				{"%.2f ^8(more/less damage taken)", takenMore},
			},
		})
	} else {
		b.AddLine(key, "Effective DPS modifier:")
		b.AddLine(key, fmt.Sprintf("= %.3f ^8(increased/reduced damage taken)", mult))
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
