package exposition

import (
	"github.com/Vilsol/crystalline"

	"runtime/debug"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

func Expose() *crystalline.Exposer {
	e := crystalline.NewExposer("go-pob")

	crystalline.MarkPromise("calculator.Calculator", "BuildOutput")

	crystalline.MarkIgnored("msgp.Reader", "ReadComplex64")
	crystalline.MarkIgnored("msgp.Reader", "ReadComplex128")

	crystalline.MarkIgnored("msgp.Writer", "WriteComplex64")
	crystalline.MarkIgnored("msgp.Writer", "WriteComplex128")

	e.ExposeFuncOrPanic(pob.DecodeDecompress)
	e.ExposeFuncOrPanic(pob.CompressEncode)

	e.ExposeFuncOrPanic(builds.ParseBuild)
	e.ExposeFuncOrPanic(builds.ParseBuildStr)
	e.ExposeFuncOrPanic(builds.SerializeBuild)
	e.ExposeFuncOrPanic(builds.EmptyBuild)

	e.ExposeFuncOrPanic(calculator.NewCalculator)
	e.ExposeFuncOrPanicPromise(raw.InitializeAll)
	e.ExposeFuncOrPanic(config.InitLogging)

	e.ExposeFuncOrPanic(storage.InitializeStorage)
	e.ExposeFuncOrPanicPromise(storage.ListBuilds)
	e.ExposeFuncOrPanicPromise(storage.NewFolder)
	e.ExposeFuncOrPanicPromise(storage.GetBuild)
	e.ExposeFuncOrPanicPromise(storage.SetBuild)
	e.ExposeFuncOrPanicPromise(storage.DeleteBuild)

	e.ExposeFuncOrPanic(GetSkillGems)
	e.ExposeFuncOrPanicPromise(GetRawTree)
	e.ExposeFuncOrPanic(GetStatByIndex)
	e.ExposeFuncOrPanic(CalculateAllocationPaths)
	e.ExposeFuncOrPanic(CalculatePrunableNodes)

	e.ExposeFuncOrPanic(SetCalcTabElements)

	info, _ := debug.ReadBuildInfo()
	e.ExposeOrPanic(info, "pob", "BuildInfo")

	return e
}
