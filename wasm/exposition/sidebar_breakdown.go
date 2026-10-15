package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func SidebarBreakdownExports(r bind.Registry) {
	r.Type(SidebarPass{})

	r.Func(NewSidebarPass)
}

type SidebarPass struct {
	env *calculator.Environment
}

func NewSidebarPass(build *pob.PathOfBuilding) *SidebarPass {
	calc := calculator.NewCalculator(*build)
	calc.AlignCalcsSkillToMain = true
	return &SidebarPass{env: calc.BuildOutput(calculator.OutputModeCalcs)}
}

func (p *SidebarPass) breakdown(minion bool) map[string]*calculator.Entry {
	if p.env == nil {
		return nil
	}
	actor := p.env.Player
	if minion {
		if p.env.Minion == nil {
			return nil
		}
		actor = p.env.Minion.Actor
	}
	if actor == nil || actor.Breakdown == nil {
		return nil
	}
	return actor.Breakdown.GetData()
}

//pob:src/Modules/Build.lua:1872-1908
func (p *SidebarPass) StatBreakdownKey(minion bool, stat, childStat string, override string, hasOverride bool, modNames bool) (string, bool) {
	bd := p.breakdown(minion)
	if bd == nil {
		return "", false
	}
	var key string
	switch {
	case hasOverride:
		key = override
	case childStat != "":
		if bd[stat+"."+childStat] == nil {
			return "", false
		}
		key = stat + "." + childStat
	case bd[stat] != nil:
		key = stat
	default:
		return "", false
	}
	if modNames {
		return key, true
	}
	e := bd[key]
	if e != nil && (len(e.Lines) > 0 || e.Radius != 0 || len(e.Rows) > 0 || len(e.Reservations) > 0 || len(e.DamageTypes) > 0 || len(e.Slots) > 0) {
		return key, true
	}
	return "", false
}

func (p *SidebarPass) Entry(minion bool, key string) *calculator.Entry {
	return p.breakdown(minion)[key]
}

func (p *SidebarPass) ModSection(minion bool, sd calculator.ModSectionData) []calculator.ModSectionRow {
	return p.env.CalcsModSection(sd, minion)
}
