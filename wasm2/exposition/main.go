//go:build js || bindings2

// The manifest is hidden from the PoE1 generator, which loads every package of the module natively;
// `task generate2` runs crystalline with the bindings2 tag.
package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/calculator2"
	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/storage"
)

//go:generate go tool crystalline -app go-pob2 -dir ../.. -js-out ../../frontend/src/lib/types2/index.js -ts-out ../../frontend/src/lib/types2/index.d.ts -banner "/* eslint-disable */" ./wasm2/exposition

//crystalline:exports
func Exports(r bind.Registry) {
	r.Type(Calculator{}, bind.AsPromise("Calculate"))
	r.Type(Outputs{}, bind.Plain())
	r.Type(calculator2.SocketGroupView{}, bind.Plain())
	r.Type(calculator2.ConfigValue{}, bind.Plain())
	r.Type(calculator2.ConfigRelevance{}, bind.Plain())
	r.Type(storage.DirEntry{}, bind.Plain())

	r.Func(ParseBuild)
	r.Value("TreeVersion", TreeVersion)

	r.Func(config.InitLogging)
	r.Func(storage.InitializeStorage)
}
