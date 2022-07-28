package data

import (
	_ "embed"
)

//go:embed raw/gems.min.json.gz
var gemsMinJSONGz []byte
var Gems map[string]*Gem

func init() {
	Gems = FromJSONGz[map[string]*Gem](gemsMinJSONGz)
}
