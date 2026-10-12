package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

//crystalline:exports
func TreeLinkExports(r bind.Registry) {
	r.Func(ExportTreeLink, bind.AsPromise())
	r.Func(ImportTreeLink, bind.AsPromise())
}

func ExportTreeLink(build *pob.PathOfBuilding) string {
	prefix := data.TreeVersions[data.LatestTreeVersion].URL
	if i := build.ActiveSpecIndex(); i >= 0 {
		if v, ok := data.TreeVersions[build.Tree.Specs[i].TreeVersion]; ok {
			prefix = v.URL
		}
	}
	return calculator.EncodeTreeURL(build, prefix)
}

func ImportTreeLink(build *pob.PathOfBuilding, title string, link string) string {
	if err := calculator.ImportTreeURL(build, title, link); err != nil {
		return err.Error()
	}
	return ""
}
