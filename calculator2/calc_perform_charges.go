package calculator2

import (
	"math"
	"strings"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
)

func overrideOr(modDB *moddb.ModDB, name string, fallback float64) float64 {
	if o := modDB.Override(nil, name); o != nil {
		return o.Float()
	}
	return fallback
}

func flagOr(cond bool, a, b float64) float64 {
	if cond {
		return a
	}
	return b
}

// No party is ported, so env.partyMembers' PartyMemberMaximumEnduranceChargesEqualToYours reads unset.
//
//pob2:src/Modules/CalcPerform.lua:862-1016
func doActorCharges(env *Environment, actor *Actor) {
	modDB := actor.ModDB
	output := actor.Output

	output["PowerChargesMin"] = math.Max(modDB.Sum(mod.TypeBase, nil, "PowerChargesMin"), 0)
	output["PowerChargesMax"] = overrideOr(modDB, "PowerChargesMax", math.Max(modDB.Sum(mod.TypeBase, nil, "PowerChargesMax"), 0))
	output["PowerChargesDuration"] = math.Floor(modDB.Sum(mod.TypeBase, nil, "ChargeDuration") * calcLibMod(modDB, nil, "PowerChargesDuration", "ChargeDuration"))
	if modDB.Flag(nil, "MaximumFrenzyChargesIsMaximumPowerCharges") {
		source := modDB.Mods["MaximumFrenzyChargesIsMaximumPowerCharges"][0].GetSource()
		prefix := mod.Source(strings.SplitN(string(source), ":", 2)[0])
		if !modDB.HasMod(mod.TypeOverride, &moddb.ListCfg{Source: &prefix}, "FrenzyChargesMax") {
			modDB.AddMod(newMod("FrenzyChargesMax", mod.TypeOverride, output["PowerChargesMax"], source))
		}
	}
	output["FrenzyChargesMin"] = math.Max(modDB.Sum(mod.TypeBase, nil, "FrenzyChargesMin"), 0)
	output["FrenzyChargesMax"] = overrideOr(modDB, "FrenzyChargesMax", math.Max(flagOr(modDB.Flag(nil, "MaximumFrenzyChargesIsMaximumPowerCharges"), output["PowerChargesMax"], modDB.Sum(mod.TypeBase, nil, "FrenzyChargesMax")), 0))
	output["FrenzyChargesDuration"] = math.Floor(modDB.Sum(mod.TypeBase, nil, "ChargeDuration") * calcLibMod(modDB, nil, "FrenzyChargesDuration", "ChargeDuration"))
	if modDB.Flag(nil, "MaximumEnduranceChargesIsMaximumFrenzyCharges") {
		source := modDB.Mods["MaximumEnduranceChargesIsMaximumFrenzyCharges"][0].GetSource()
		prefix := mod.Source(strings.SplitN(string(source), ":", 2)[0])
		if !modDB.HasMod(mod.TypeOverride, &moddb.ListCfg{Source: &prefix}, "EnduranceChargesMax") {
			modDB.AddMod(newMod("EnduranceChargesMax", mod.TypeOverride, output["FrenzyChargesMax"], source))
		}
	}
	output["EnduranceChargesMin"] = math.Max(modDB.Sum(mod.TypeBase, nil, "EnduranceChargesMin"), 0)
	output["EnduranceChargesMax"] = math.Max(overrideOr(modDB, "EnduranceChargesMax", flagOr(modDB.Flag(nil, "MaximumEnduranceChargesIsMaximumFrenzyCharges"), output["FrenzyChargesMax"], modDB.Sum(mod.TypeBase, nil, "EnduranceChargesMax"))), 0)
	output["EnduranceChargesDuration"] = math.Floor(modDB.Sum(mod.TypeBase, nil, "ChargeDuration") * calcLibMod(modDB, nil, "EnduranceChargesDuration", "ChargeDuration"))
	output["SiphoningChargesMax"] = math.Max(modDB.Sum(mod.TypeBase, nil, "SiphoningChargesMax"), 0)
	output["ChallengerChargesMax"] = math.Max(modDB.Sum(mod.TypeBase, nil, "ChallengerChargesMax"), 0)
	output["BlitzChargesMax"] = math.Max(modDB.Sum(mod.TypeBase, nil, "BlitzChargesMax"), 0)
	output["InspirationChargesMax"] = math.Max(modDB.Sum(mod.TypeBase, nil, "InspirationChargesMax"), 0)
	output["CrabBarriersMax"] = math.Max(modDB.Sum(mod.TypeBase, nil, "CrabBarriersMax"), 0)
	output["BrutalChargesMin"] = math.Max(flagOr(modDB.Flag(nil, "MinimumEnduranceChargesEqualsMinimumBrutalCharges"), flagOr(modDB.Flag(nil, "MinimumEnduranceChargesIsMaximumEnduranceCharges"), output["EnduranceChargesMax"], output["EnduranceChargesMin"]), 0), 0)
	output["BrutalChargesMax"] = math.Max(flagOr(modDB.Flag(nil, "MaximumEnduranceChargesEqualsMaximumBrutalCharges"), output["EnduranceChargesMax"], 0), 0)
	output["AbsorptionChargesMin"] = math.Max(flagOr(modDB.Flag(nil, "MinimumPowerChargesEqualsMinimumAbsorptionCharges"), flagOr(modDB.Flag(nil, "MinimumPowerChargesIsMaximumPowerCharges"), output["PowerChargesMax"], output["PowerChargesMin"]), 0), 0)
	output["AbsorptionChargesMax"] = math.Max(flagOr(modDB.Flag(nil, "MaximumPowerChargesEqualsMaximumAbsorptionCharges"), output["PowerChargesMax"], 0), 0)
	output["AfflictionChargesMin"] = math.Max(flagOr(modDB.Flag(nil, "MinimumFrenzyChargesEqualsMinimumAfflictionCharges"), flagOr(modDB.Flag(nil, "MinimumFrenzyChargesIsMaximumFrenzyCharges"), output["FrenzyChargesMax"], output["FrenzyChargesMin"]), 0), 0)
	output["AfflictionChargesMax"] = math.Max(flagOr(modDB.Flag(nil, "MaximumFrenzyChargesEqualsMaximumAfflictionCharges"), output["FrenzyChargesMax"], 0), 0)
	output["BloodChargesMax"] = math.Max(modDB.Sum(mod.TypeBase, nil, "BloodChargesMax"), 0)
	output["SpiritChargesMax"] = math.Max(modDB.Sum(mod.TypeBase, nil, "SpiritChargesMax"), 0)

	// Initialize Charges
	output["PowerCharges"] = 0
	output["FrenzyCharges"] = 0
	output["EnduranceCharges"] = 0
	output["SiphoningCharges"] = 0
	output["ChallengerCharges"] = 0
	output["BlitzCharges"] = 0
	output["InspirationCharges"] = 0
	output["GhostShrouds"] = 0
	output["BrutalCharges"] = 0
	output["AbsorptionCharges"] = 0
	output["AfflictionCharges"] = 0
	output["BloodCharges"] = 0
	output["SpiritCharges"] = 0

	// Conditionally over-write Charge values
	if modDB.Flag(nil, "MinimumFrenzyChargesIsMaximumFrenzyCharges") {
		output["FrenzyChargesMin"] = output["FrenzyChargesMax"]
	}
	if modDB.Flag(nil, "MinimumEnduranceChargesIsMaximumEnduranceCharges") {
		output["EnduranceChargesMin"] = output["EnduranceChargesMax"]
	}
	if modDB.Flag(nil, "MinimumPowerChargesIsMaximumPowerCharges") {
		output["PowerChargesMin"] = output["PowerChargesMax"]
	}
	if modDB.Flag(nil, "Condition:UsePowerCharges") {
		output["PowerCharges"] = overrideOr(modDB, "PowerCharges", output["PowerChargesMax"])
	}
	if modDB.Flag(nil, "PowerChargesConvertToAbsorptionCharges") {
		// we max with possible Power Charge Override from Config since Absorption Charges won't have their own config entry
		// and are converted from Power Charges
		output["AbsorptionCharges"] = math.Max(output["PowerCharges"], math.Min(output["AbsorptionChargesMax"], output["AbsorptionChargesMin"]))
		output["PowerCharges"] = 0
	} else {
		output["PowerCharges"] = math.Max(output["PowerCharges"], math.Min(output["PowerChargesMax"], output["PowerChargesMin"]))
	}
	removablePowerCharges := math.Max(output["PowerCharges"]-output["PowerChargesMin"], 0)
	extraConsumablePowerCharges := 0.0
	if removablePowerCharges > 0 {
		var skillCfg *moddb.ListCfg
		if actor.MainSkill != nil {
			skillCfg = actor.MainSkill.SkillCfg
		}
		extraConsumablePowerCharges = modDB.Sum(mod.TypeBase, skillCfg, "Multiplier:ExtraConsumablePowerCharges")
	}
	output["RemovablePowerCharges"] = removablePowerCharges + extraConsumablePowerCharges
	if modDB.Flag(nil, "Condition:UseFrenzyCharges") {
		output["FrenzyCharges"] = overrideOr(modDB, "FrenzyCharges", output["FrenzyChargesMax"])
	}
	if modDB.Flag(nil, "FrenzyChargesConvertToAfflictionCharges") {
		// we max with possible Power Charge Override from Config since Absorption Charges won't have their own config entry
		// and are converted from Power Charges
		output["AfflictionCharges"] = math.Max(output["FrenzyCharges"], math.Min(output["AfflictionChargesMax"], output["AfflictionChargesMin"]))
		output["FrenzyCharges"] = 0
	} else {
		output["FrenzyCharges"] = math.Max(output["FrenzyCharges"], math.Min(output["FrenzyChargesMax"], output["FrenzyChargesMin"]))
	}
	output["RemovableFrenzyCharges"] = math.Max(output["FrenzyCharges"]-output["FrenzyChargesMin"], 0)
	if modDB.Flag(nil, "Condition:UseEnduranceCharges") {
		output["EnduranceCharges"] = overrideOr(modDB, "EnduranceCharges", output["EnduranceChargesMax"])
	}
	if modDB.Flag(nil, "EnduranceChargesConvertToBrutalCharges") {
		// we max with possible Endurance Charge Override from Config since Brutal Charges won't have their own config entry
		// and are converted from Endurance Charges
		output["BrutalCharges"] = math.Max(output["EnduranceCharges"], math.Min(output["BrutalChargesMax"], output["BrutalChargesMin"]))
		output["EnduranceCharges"] = 0
	} else {
		output["EnduranceCharges"] = math.Max(output["EnduranceCharges"], math.Min(output["EnduranceChargesMax"], output["EnduranceChargesMin"]))
	}
	output["RemovableEnduranceCharges"] = math.Max(output["EnduranceCharges"]-output["EnduranceChargesMin"], 0)
	if modDB.Flag(nil, "UseSiphoningCharges") {
		output["SiphoningCharges"] = overrideOr(modDB, "SiphoningCharges", output["SiphoningChargesMax"])
	}
	if modDB.Flag(nil, "UseChallengerCharges") {
		output["ChallengerCharges"] = overrideOr(modDB, "ChallengerCharges", output["ChallengerChargesMax"])
	}
	if modDB.Flag(nil, "UseBlitzCharges") {
		output["BlitzCharges"] = overrideOr(modDB, "BlitzCharges", output["BlitzChargesMax"])
	}
	if actor == env.Player {
		output["InspirationCharges"] = overrideOr(modDB, "InspirationCharges", output["InspirationChargesMax"])
	}
	if modDB.Flag(nil, "UseGhostShrouds") {
		output["GhostShrouds"] = overrideOr(modDB, "GhostShrouds", 3)
	}
	output["BloodCharges"] = math.Min(overrideOr(modDB, "BloodCharges", output["BloodChargesMax"]), output["BloodChargesMax"])
	output["SpiritCharges"] = math.Min(overrideOr(modDB, "SpiritCharges", 0), output["SpiritChargesMax"])

	output["CrabBarriers"] = math.Min(overrideOr(modDB, "CrabBarriers", output["CrabBarriersMax"]), output["CrabBarriersMax"])
	if modDB.Flag(nil, "HaveMaximumPowerCharges") {
		output["PowerCharges"] = output["PowerChargesMax"]
	}
	if modDB.Flag(nil, "HaveMaximumFrenzyCharges") {
		output["FrenzyCharges"] = output["FrenzyChargesMax"]
	}
	if modDB.Flag(nil, "HaveMaximumEnduranceCharges") {
		output["EnduranceCharges"] = output["EnduranceChargesMax"]
	}
	output["TotalCharges"] = output["PowerCharges"] + output["FrenzyCharges"] + output["EnduranceCharges"]
	output["RemovableTotalCharges"] = output["RemovableEnduranceCharges"] + output["RemovableFrenzyCharges"] + output["RemovablePowerCharges"]
	modDB.Multipliers["PowerCharge"] = output["PowerCharges"]
	modDB.Multipliers["PowerChargeMax"] = output["PowerChargesMax"]
	modDB.Multipliers["RemovablePowerCharge"] = output["RemovablePowerCharges"]
	modDB.Multipliers["FrenzyCharge"] = output["FrenzyCharges"]
	modDB.Multipliers["RemovableFrenzyCharge"] = output["RemovableFrenzyCharges"]
	modDB.Multipliers["EnduranceCharge"] = output["EnduranceCharges"]
	modDB.Multipliers["RemovableEnduranceCharge"] = output["RemovableEnduranceCharges"]
	modDB.Multipliers["TotalCharges"] = output["TotalCharges"]
	modDB.Multipliers["RemovableTotalCharges"] = output["RemovableTotalCharges"]
	modDB.Multipliers["SiphoningCharge"] = output["SiphoningCharges"]
	modDB.Multipliers["ChallengerCharge"] = output["ChallengerCharges"]
	modDB.Multipliers["BlitzCharge"] = output["BlitzCharges"]
	modDB.Multipliers["InspirationCharge"] = output["InspirationCharges"]
	modDB.Multipliers["GhostShroud"] = output["GhostShrouds"]
	modDB.Multipliers["CrabBarrier"] = output["CrabBarriers"]
	modDB.Multipliers["BrutalCharge"] = output["BrutalCharges"]
	modDB.Multipliers["AbsorptionCharge"] = output["AbsorptionCharges"]
	modDB.Multipliers["AfflictionCharge"] = output["AfflictionCharges"]
	modDB.Multipliers["BloodCharge"] = output["BloodCharges"]
	modDB.Multipliers["SpiritCharge"] = output["SpiritCharges"]
}
