package exposition

import (
	"slices"

	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

func TreeVersionExports(r bind.Registry) {
	r.Type(TreeVersionOption{}, bind.Plain())

	r.Func(GetTreeVersionOptions)
	r.Func(ConvertTreeSpec, bind.AsPromise())
	r.Func(ConvertAllTreeSpecs, bind.AsPromise())
}

type TreeVersionOption struct {
	Value string
	Label string
}

func GetTreeVersionOptions() []TreeVersionOption {
	versions := make([]pob.TreeVersion, 0, len(data.TreeVersions))
	for v := range data.TreeVersions {
		versions = append(versions, v)
	}
	slices.SortFunc(versions, func(a, b pob.TreeVersion) int {
		switch na, nb := data.TreeVersions[a].Num, data.TreeVersions[b].Num; {
		case na < nb:
			return -1
		case na > nb:
			return 1
		}
		return 0
	})
	out := make([]TreeVersionOption, 0, len(versions))
	for _, v := range versions {
		out = append(out, TreeVersionOption{Value: string(v), Label: data.TreeVersions[v].Display})
	}
	return out
}

func ConvertTreeSpec(build *pob.PathOfBuilding, version string, replace bool) string {
	if err := calculator.ConvertSpecToVersion(build, pob.TreeVersion(version), replace); err != nil {
		return err.Error()
	}
	return ""
}

func ConvertAllTreeSpecs(build *pob.PathOfBuilding, version string) string {
	if err := calculator.ConvertAllSpecsToVersion(build, pob.TreeVersion(version)); err != nil {
		return err.Error()
	}
	return ""
}
