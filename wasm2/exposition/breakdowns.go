package exposition

import "github.com/Vilsol/go-pob/calculator2"

// CalcsBreakdownCol is one column of a breakdown table.
type CalcsBreakdownCol struct {
	Label string
	Key   string
}

// CalcsBreakdownTable is one table of a breakdown; a row's missing key is an empty cell.
type CalcsBreakdownTable struct {
	Label   string
	Footer  string
	Columns []CalcsBreakdownCol
	Rows    []map[string]string
}

// CalcsBreakdown is what CalcBreakdownControl shows of one breakdown: its lines, its radius (0 for none) and its
// tables. A gearOnly cell draws GearTable in place of the slots table, where Gear is set.
type CalcsBreakdown struct {
	Lines     []string
	Radius    float64
	Tables    []CalcsBreakdownTable
	Gear      bool
	GearTable CalcsBreakdownTable
}

// CalcsBreakdowns is every breakdown the last calculation's player holds, by the name a Calcs cell gives.
func (c *Calculator) CalcsBreakdowns() (map[string]CalcsBreakdown, error) {
	if c.env == nil {
		return nil, notCalculatedError{}
	}
	views := c.env.Player.Breakdown.Views()
	out := make(map[string]CalcsBreakdown, len(views))
	for name, v := range views {
		out[name] = calcsBreakdownOf(v)
	}
	return out, nil
}

func calcsBreakdownOf(v calculator2.BreakdownView) CalcsBreakdown {
	bd := CalcsBreakdown{Lines: v.Lines, Radius: v.Radius}
	for _, t := range v.Tables {
		bd.Tables = append(bd.Tables, calcsBreakdownTableOf(t))
	}
	if v.GearTable != nil {
		bd.Gear, bd.GearTable = true, calcsBreakdownTableOf(*v.GearTable)
	}
	return bd
}

func calcsBreakdownTableOf(t calculator2.BreakdownTable) CalcsBreakdownTable {
	cols := make([]CalcsBreakdownCol, len(t.Columns))
	for i, c := range t.Columns {
		cols[i] = CalcsBreakdownCol{Label: c.Label, Key: c.Key}
	}
	return CalcsBreakdownTable{Label: t.Label, Footer: t.Footer, Columns: cols, Rows: t.Rows}
}
