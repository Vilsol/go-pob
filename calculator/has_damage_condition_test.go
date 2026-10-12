package calculator

import (
	"os"
	"testing"

	"github.com/Vilsol/go-pob/data"
)

// TestHasDamageConditionsAreProduced pins CalcOffence.lua:3618.
//
//	skillModList:NewMod("Condition:"..damageType.."HasDamage", "FLAG", true, "Config")
func TestHasDamageConditionsAreProduced(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	for _, tc := range []struct {
		build      string
		damageType data.DamageType
	}{
		{"0781-hierophant-fireball.xml", data.DamageTypeFire},
		{"0600-saboteur-glacial-cascade.xml", data.DamageTypeCold},
		{"0421-hierophant-vaal-arc.xml", data.DamageTypeLightning},
	} {
		t.Run(tc.build, func(t *testing.T) {
			env := corpusEnvForLoadedData(t, tc.build)
			skillModList := env.Player.MainSkill.SkillModList

			anyHighest := false
			for _, dt := range data.DamageType("").Values() {
				if skillModList.Flag(nil, "Condition:"+string(dt)+"IsHighestDamageType") {
					anyHighest = true
					break
				}
			}
			if !anyHighest {
				t.Fatalf("REFUSING: %s sets no Condition:<T>IsHighestDamageType at all — the "+
					"control moved with the subject, so this probe is reading the wrong store "+
					"rather than finding a missing condition", tc.build)
			}

			if env.Player.Output[string(tc.damageType)+"HitAverage"] == 0 &&
				env.Player.OutputTable[OutTableMainHand][string(tc.damageType)+"HitAverage"] == 0 {
				t.Fatalf("REFUSING: %s deals no %s hit damage, so :3616's guard never fires and "+
					"this fixture cannot see the condition", tc.build, tc.damageType)
			}

			name := "Condition:" + string(tc.damageType) + "HasDamage"
			if !skillModList.Flag(nil, name) {
				t.Errorf("%s: %s is not set; mod_parser.go:1095-1102 reads it on five rows and "+
					"nothing in this tree writes it", tc.build, name)
			}
		})
	}
}
