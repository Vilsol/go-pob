package exposition

import (
	"runtime/debug"

	"github.com/Vilsol/crystalline/bind"
	"github.com/Vilsol/go-pob-data/poe"
	"github.com/tinylib/msgp/msgp"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/calculator"
	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

//go:generate go tool crystalline -app go-pob -dir ../.. -js-out ../../frontend/src/lib/types/index.js -ts-out ../../frontend/src/lib/types/index.d.ts -banner "/* eslint-disable */" ./...

//crystalline:exports
func Exports(r bind.Registry) {
	r.Type(calculator.Calculator{}, bind.AsPromise("BuildOutput"))
	r.Type(calculator.Breakdown{}, bind.PlainResult("GetData"))

	r.Type(storage.DirEntry{}, bind.Plain())
	r.Type(SkillGem{}, bind.Plain())
	r.Type(GemPart{}, bind.Plain())
	r.Type(ElementWrapper{}, bind.Plain())
	r.Type(pob.Gem{}, bind.Plain())
	r.Type(pob.Input{}, bind.Plain())
	r.Type(StatOutputs{}, bind.Plain())
	r.Type(MiscCalculator{}, bind.AsPromise("CalcNodes", "CompareItem", "CalcUnequip", "CalcToggleFlask", "AddItem", "ReplaceItem"))
	r.Type(calculator.ConfigRelevance{}, bind.Plain())
	r.Type(ModLine{}, bind.Plain())

	r.Type(msgp.Reader{}, bind.Without("ReadComplex64", "ReadComplex128"))
	r.Type(msgp.Writer{}, bind.Without("WriteComplex64", "WriteComplex128"))

	r.Type(pob.Skills{}, bind.Without("UnmarshalXML"))

	r.Type(poe.SkillGem{}, bind.Without("GetGemVariants", "GetLevelProgression"))
	r.Type(poe.BaseItemType{}, bind.Without("GetEnchantMods", "GetItemVisualIdentity"))
	r.Type(poe.ActiveSkill{}, bind.Without("GetSkillSpecificStats", "GetSecondarySkillSpecificStats"))

	r.Type(pob.Build{}, bind.Plain())
	r.Type(pob.Tree{}, bind.Plain())
	r.Type(pob.Calcs{}, bind.Plain())
	r.Type(pob.Items{}, bind.Plain())
	r.Type(pob.TreeView{}, bind.Plain())
	r.Type(pob.Config{}, bind.Plain())

	r.Func(pob.DecodeDecompress)
	r.Func(pob.CompressEncode)

	r.Func(builds.ParseBuild)
	r.Func(builds.ParseBuildStr)
	r.Func(builds.ParseBuildOver)
	r.Func(builds.SerializeBuild)
	r.Func(builds.SerializeBuildWithStats)
	r.Func(builds.EmptyBuild)
	r.Func(builds.SavedBuildSummary)

	r.Func(calculator.NewCalculator)
	r.Func(raw.InitializeAll, bind.AsPromise())
	r.Func(raw.LoadDeferred, bind.AsPromise())
	r.Func(raw.GameDataVersion)
	r.Func(config.InitLogging)

	r.Func(storage.InitializeStorage)
	r.Func(storage.ListBuilds, bind.AsPromise())
	r.Func(storage.NewFolder, bind.AsPromise())
	r.Func(storage.GetBuild, bind.AsPromise())
	r.Func(storage.SetBuild, bind.AsPromise())
	r.Func(storage.DeleteBuild, bind.AsPromise())

	r.Func(GetSkillGems)
	r.Func(CalcTreeVersion)
	r.Func(GetRawTree, bind.AsPromise())
	r.Func(GetStatByIndex)
	r.Func(CalculateAllocationPaths)
	r.Func(CalculatePrunableNodes)
	r.Func(CalculateDependentNodes)
	r.Type(data.SubgraphNode{}, bind.Plain())
	r.Func(CalculateDependentNodesWith)
	r.Func(GetMiscCalculator, bind.AsPromise())
	r.Func(MiscCalculatorFor)
	r.Func(ParseModLines)
	r.Func(ModBrowserList)
	r.Func(ModBrowserSupported)
	r.Func(ModBrowserSources)
	r.Func(ConfigPlaceholders)

	r.Type(calculator.ComparePowerCategories{}, bind.Plain())
	r.Type(calculator.ComparePowerEntry{}, bind.Plain())
	r.Type(calculator.ComparePowerBuilder{}, bind.Without("Step"))
	r.Func(calculator.NewComparePowerBuilder)

	r.Func(SetCalcTabElements)
	r.Func(DefaultGem)
	r.Func(GemError)
	r.Func(LiveHeapMB)

	info, _ := debug.ReadBuildInfo()
	r.Value("BuildInfo", info, bind.InNamespace("pob"))
}

func DefaultGem(gemID string, nameSpec string, skillID string, level int, quality int) pob.Gem {
	count := 1
	return pob.Gem{
		GemID:         gemID,
		NameSpec:      nameSpec,
		SkillID:       skillID,
		Level:         level,
		Quality:       quality,
		QualityID:     "Default",
		Count:         &count,
		Enabled:       true,
		EnableGlobal1: true,
		EnableGlobal2: true,
	}
}
