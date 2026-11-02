package calculator2

import "sort"

// BreakdownTable is one TABLE section CalcBreakdownControl draws; a row's missing key is an empty cell.
type BreakdownTable struct {
	Label   string
	Footer  string
	Columns []BreakdownCol
	Rows    []map[string]string
}

// BreakdownView is the sections CalcBreakdownControl:AddBreakdownSection makes of one breakdown: its text lines,
// its radius (HasRadius) and its tables, in the order it draws them.
type BreakdownView struct {
	Lines     []string
	Radius    float64
	HasRadius bool
	Tables    []BreakdownTable
	// GearTable is the slots table a gearOnly cell draws in place of the full one, nil without slots.
	GearTable *BreakdownTable
}

//pob2:block:src/Classes/CalcBreakdownControl.lua:207-219
var reservationCols = []BreakdownCol{
	{"Skill", "skillName"}, {"Base", "base"}, {"MCM", "mult"}, {"More/less", "more"}, {"Inc/red", "inc"},
	{"Efficiency", "efficiency"}, {"Efficiency More/less", "efficiencyMore"}, {"Count", "count"}, {"Reservation", "total"},
}

//pob2:block:src/Classes/CalcBreakdownControl.lua:246-250
var gearSlotCols = []BreakdownCol{{"Value", "base"}, {"Source", "source"}, {"Name", "sourceLabel"}}

//pob2:block:src/Classes/CalcBreakdownControl.lua:259-266
var slotCols = []BreakdownCol{
	{"Base", "base"}, {"Inc/red", "inc"}, {"More/less", "more"}, {"Total", "total"}, {"Source", "source"}, {"Name", "sourceLabel"},
}

// colorCodes[rarity] for an item row's sourceLabel.
//
//pob2:src/Data/Global.lua:8-12
var rarityColors = map[string]string{
	"NORMAL": "^xC8C8C8", "MAGIC": "^x8888FF", "RARE": "^xFFFF77", "UNIQUE": "^xAF6025", "RELIC": "^x60C060",
}

// View is AddBreakdownSection for the breakdown name a CalcSections cell gives, without its modList section. ok is
// false where the actor has no such breakdown.
//
//pob2:src/Classes/CalcBreakdownControl.lua:147-297
func (b *Breakdown) View(name string) (BreakdownView, bool) {
	if b == nil {
		return BreakdownView{}, false
	}
	e := b.Entries[name]
	if e == nil {
		return BreakdownView{}, false
	}
	var v BreakdownView
	if len(e.Lines) > 0 {
		v.Lines = append([]string(nil), e.Lines...)
	}
	if e.HasRadius {
		v.Radius, v.HasRadius = e.Radius, true
	}
	if len(e.RowList) > 0 {
		v.Tables = append(v.Tables, BreakdownTable{Label: e.Label, Footer: e.Footer, Columns: e.ColList, Rows: e.RowList})
	}
	if len(e.Reservations) > 0 {
		rows := make([]map[string]string, 0, len(e.Reservations))
		for _, r := range e.Reservations {
			rows = append(rows, nonEmpty(map[string]string{
				"skillName": r.SkillName, "base": r.Base, "mult": r.Mult, "more": r.More, "inc": r.Inc,
				"efficiency": r.Efficiency, "efficiencyMore": r.EfficiencyMore, "count": r.Count, "total": r.Total,
			}))
		}
		v.Tables = append(v.Tables, BreakdownTable{Columns: reservationCols, Rows: rows})
	}
	if len(e.Slots) > 0 {
		slots := append([]BreakdownSlot(nil), e.Slots...)
		sort.SliceStable(slots, func(i, j int) bool { return slots[i].Base > slots[j].Base })
		rows := make([]map[string]string, 0, len(slots))
		gear := []map[string]string{}
		for _, s := range slots {
			label := s.SourceName
			if s.Item != nil {
				label = rarityColors[s.Item.Rarity] + s.Item.Name
			}
			row := nonEmpty(map[string]string{
				"base": luaToString(s.Base), "inc": s.Inc, "more": s.More, "total": s.Total, "source": s.Source, "sourceLabel": label,
			})
			rows = append(rows, row)
			if s.Item != nil && s.Item.ArmourData != nil {
				gear = append(gear, nonEmpty(map[string]string{"base": row["base"], "source": s.Source, "sourceLabel": label}))
			}
		}
		v.Tables = append(v.Tables, BreakdownTable{Columns: slotCols, Rows: rows})
		v.GearTable = &BreakdownTable{Columns: gearSlotCols, Rows: gear}
	}
	if v.Lines == nil && !v.HasRadius && v.Tables == nil {
		return v, false
	}
	return v, true
}

func nonEmpty(m map[string]string) map[string]string {
	for k, v := range m {
		if v == "" {
			delete(m, k)
		}
	}
	return m
}

// Views is View for every breakdown the actor holds, a MainHand or OffHand one keyed "MainHand.<name>".
func (b *Breakdown) Views() map[string]BreakdownView {
	if b == nil {
		return nil
	}
	out := make(map[string]BreakdownView, len(b.Entries))
	for k := range b.Entries {
		if v, ok := b.View(k); ok {
			out[k] = v
		}
	}
	for prefix, hand := range map[string]*Breakdown{"MainHand.": b.MainHand, "OffHand.": b.OffHand} {
		for k, v := range hand.Views() {
			out[prefix+k] = v
		}
	}
	return out
}
