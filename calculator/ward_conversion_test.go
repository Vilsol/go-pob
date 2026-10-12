package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

var wardConversionBuilds = []string{
	"0888-reliquarian-phase-run.xml",
	"1344-slayer-reave-3.xml",
	"1412-gladiator-flame-surge-of-combusting-2.xml",
	"1606-chieftain-treeskill-4.xml",
}

// TestBodyArmourWardConversionEntersWard pins the converted quantity itself.
func TestBodyArmourWardConversionEntersWard(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, build := range wardConversionBuilds {
		t.Run(build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, build)
			modDB := env.Player.ModDB

			if !modDB.Flag(nil, "ConvertBodyArmourArmourEvasionToWard") {
				t.Fatalf("premise broken: %s no longer sets ConvertBodyArmourArmourEvasionToWard, "+
					"so this case cannot exercise the conversion", build)
			}
			pct := modDB.Sum(mod.TypeBase, nil, "BodyArmourArmourEvasionToWardPercent")
			if pct <= 0 {
				t.Fatalf("premise broken: %s converts %v%%, so the conversion is a no-op and the "+
					"assertion below would pass vacuously", build, pct)
			}

			item := env.Player.ItemList["Body Armour"]
			if item == nil || item.ArmourData == nil {
				t.Fatalf("premise broken: %s has no body armour to convert from", build)
			}
			rawArmour, rawEvasion := item.ArmourData.Armour, item.ArmourData.Evasion
			if rawArmour <= 0 && rawEvasion <= 0 {
				t.Fatalf("premise broken: %s equips a body armour with no armour AND no evasion "+
					"(%v/%v), so nothing converts and the assertion is vacuous", build, rawArmour, rawEvasion)
			}

			conversion := pct / 100
			if conversion > 1 {
				conversion = 1
			}
			converted := (rawArmour + rawEvasion) * conversion

			got := env.Player.Output["Ward"]
			if got < converted {
				t.Errorf("%s: Ward = %v, which is BELOW the %v this body armour converts into it "+
					"(%v armour + %v evasion at %v%%). The conversion is not reaching wardBase.",
					build, got, converted, rawArmour, rawEvasion, pct)
			}
		})
	}
}
