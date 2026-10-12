package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/pob"
)

func TreeSetExports(r bind.Registry) {
	r.Type(calculator.TreeSpecRow{}, bind.Plain())
	r.Type(TreeSpecList{}, bind.Plain())
	r.Type(calculator.TreeSpecPoints{}, bind.Plain())

	r.Func(GetTreeSpecs, bind.AsPromise())
	r.Func(GetTreeSpecsPoints, bind.AsPromise())
	r.Func(SetActiveTreeSpec, bind.AsPromise())
	r.Func(NewTreeSpec, bind.AsPromise())
	r.Func(DeleteTreeSpec, bind.AsPromise())
}

type TreeSpecList struct {
	Active int
	Specs  []calculator.TreeSpecRow
}

func GetTreeSpecs(build *pob.PathOfBuilding) TreeSpecList {
	return TreeSpecList{Active: build.ActiveSpecIndex() + 1, Specs: calculator.TreeSpecs(build)}
}

func GetTreeSpecsPoints(build *pob.PathOfBuilding) []calculator.TreeSpecPoints {
	return calculator.TreeSpecsPoints(build)
}

func SetActiveTreeSpec(build *pob.PathOfBuilding, specID int) {
	build.SetActiveSpec(specID)
}

func NewTreeSpec(build *pob.PathOfBuilding, title string) {
	calculator.NewTreeSpec(build, title)
}

func DeleteTreeSpec(build *pob.PathOfBuilding, specID int) {
	build.DeleteSpec(specID)
}
