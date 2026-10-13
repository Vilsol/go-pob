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

func (p *SidebarPass) Breakdown(minion bool, stat string) []string {
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
	if e := actor.Breakdown.Data[stat]; e != nil {
		return e.Lines
	}
	return nil
}
