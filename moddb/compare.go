package moddb

import (
	"reflect"

	"github.com/Vilsol/go-pob/mod"
)

//pob:src/Modules/ModTools.lua:121-134
func CompareModParams(a, b mod.Mod) bool {
	if a.Name() != b.Name() ||
		a.Type() != b.Type() ||
		a.Flags() != b.Flags() ||
		a.KeywordFlags() != b.KeywordFlags() {
		return false
	}

	aTags := a.Tags()
	bTags := b.Tags()
	if len(aTags) != len(bTags) {
		return false
	}
	if len(aTags) == 0 {
		return true
	}

	return reflect.DeepEqual(aTags, bTags)
}
