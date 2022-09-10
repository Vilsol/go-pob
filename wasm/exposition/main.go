package exposition

import (
	"github.com/Vilsol/crystalline"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/cache"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

func Expose() *crystalline.Exposer {
	e := crystalline.NewExposer("go-pob")

	e.ExposeFuncOrPanic(pob.DecodeDecompress)
	e.ExposeFuncOrPanic(pob.CompressEncode)

	e.ExposeFuncOrPanic(builds.ParseBuild)
	e.ExposeFuncOrPanic(builds.ParseBuildStr)

	e.ExposeFuncOrPanic(calculator.NewCalculator)
	e.ExposeFuncOrPanicPromise(raw.InitializeAll)
	e.ExposeFuncOrPanic(cache.InitializeDiskCache)
	e.ExposeFuncOrPanic(config.InitLogging)

	e.ExposeFuncOrPanic(GetSkillGems)
	e.ExposeFuncOrPanicPromise(GetRawTree)
	e.ExposeFuncOrPanic(GetStatByIndex)
	e.ExposeFuncOrPanic(CalculateTreePath)

	return e
}
