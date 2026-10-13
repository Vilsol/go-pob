package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func TreeTattooNodesExports(r bind.Registry) {
	r.Type(TattooNodeTip{}, bind.Plain())

	r.Func(TreeTattooNodes)
}

type TattooNodeTip struct {
	NodeID   int
	Name     string
	Stats    []string
	Reminder []string
}

//pob:block:src/Classes/PassiveSpec.lua:1614-1635
func TreeTattooNodes(build *pob.PathOfBuilding) []TattooNodeTip {
	var out []TattooNodeTip
	i := build.ActiveSpecIndex()
	if i < 0 {
		return out
	}
	for _, o := range build.Tree.Specs[i].Overrides {
		n, ok := data.TattooNodes[o.DN]
		if !ok {
			continue
		}
		tip := TattooNodeTip{NodeID: o.NodeID, Name: n.DN, Stats: n.Stats}
		if t := data.TattooTargets[o.DN]; t.ReminderText != "" {
			tip.Reminder = []string{t.ReminderText}
		}
		out = append(out, tip)
	}
	return out
}
