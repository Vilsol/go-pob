package exposition

import (
	"github.com/Vilsol/crystalline"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/cache"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/raw"
)

func Expose() *crystalline.Exposer {
	e := crystalline.NewExposer("go-pob")

	e.ExposeFuncOrPanic(builds.ParseBuild)
	e.ExposeFuncOrPanic(calculator.NewCalculator)
	e.ExposeFuncOrPanicPromise(raw.InitializeAll)
	e.ExposeFuncOrPanic(cache.InitializeDiskCache)
	e.ExposeFuncOrPanic(config.InitLogging)

	return e
}
