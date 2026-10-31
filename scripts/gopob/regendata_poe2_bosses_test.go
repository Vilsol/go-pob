package main

import (
	"strings"
	"testing"
)

func TestPoE2BossEmitRefuses(t *testing.T) {
	const stats = `"bossStats":{"PinnacleArmourMean":1,"PinnacleEvasionMean":1,"UberArmourMean":1,"UberEvasionMean":1}`
	for _, tc := range []struct {
		name, table, body, want string
	}{
		{"no bosses", "bosses", `{"bosses":[],` + stats + `}`, "no bosses"},
		{"boss unknown field", "bosses", `{"bosses":[{"name":"A","rarity":1}],` + stats + `}`, "rarity"},
		{"boss stats short", "bosses", `{"bosses":[{"name":"A"}],"bossStats":{"PinnacleArmourMean":1}}`, "holds 1 entries"},
		{"boss stats renamed", "bosses", `{"bosses":[{"name":"A"}],"bossStats":{"PinnacleArmourMean":1,"PinnacleEvasionMean":1,"UberArmourMean":1,"UberEvasion":1}}`, "no UberEvasionMean"},
		{"no skills", "boss-skills", `[]`, "no boss skills"},
		{"skill unknown field", "boss-skills", `[{"name":"A","DamageMultipliers":{"Fire":[1,0]},"stages":10}]`, "stages"},
		{"skill pair shape", "boss-skills", `[{"name":"A","DamageMultipliers":{"Fire":[1]}}]`, "not 2"},
		{"skill pen string", "boss-skills", `[{"name":"A","DamageMultipliers":{"Fire":[1,0]},"DamagePenetrations":{"FirePen":"x"}}]`, `"x"`},
		{"skill stat string", "boss-skills", `[{"name":"A","DamageMultipliers":{"Fire":[1,0]},"additionalStats":{"base":{"CannotBeBlocked":"yes"}}}]`, `"yes"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.table == "bosses" {
				_, err = p2eBosses([]byte(tc.body))
			} else {
				_, err = p2eBossSkills([]byte(tc.body))
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want an error naming %q", err, tc.want)
			}
		})
	}
}

// TestPoE2BossSkillEmitKeepsAbsence: a declared 0, an absent field, a "" pen and a "flag" stat stay four
// different things in the emitted table.
func TestPoE2BossSkillEmitKeepsAbsence(t *testing.T) {
	out, err := p2eBossSkills([]byte(`[{"name":"A","DamageMultipliers":{"Fire":[1,0.5]},"critChance":0,"DamagePenetrations":{"FirePen":""},"UberDamagePenetrations":{"FirePen":30},"additionalStats":{"uber":{"CannotBeBlocked":"flag","PhysicalDamageSkillConvertToFire":25}}}]`))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`CritChance: BossSkillValue{Value: 0, Set: true}`,
		`DamagePenetrations: map[string]BossSkillValue{"FirePen": {}}`,
		`UberDamagePenetrations: map[string]BossSkillValue{"FirePen": {Value: 30, Set: true}}`,
		`AdditionalStats: &BossSkillAdditionalStats{Uber: map[string]BossSkillStat{"CannotBeBlocked": {IsFlag: true}, "PhysicalDamageSkillConvertToFire": {Value: 25}}}`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("emitted table lacks %s", want)
		}
	}
	for _, absent := range []string{"Speed:", "UberDamageMultiplier:", "Base:"} {
		if strings.Contains(out, absent) {
			t.Errorf("emitted table carries %s for a field PoB has nil", absent)
		}
	}
}
