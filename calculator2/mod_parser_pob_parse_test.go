package calculator2

import (
	"testing"
)

// Expected rows are PoB-PoE2's live modLib.parseMod output, written in ModCache.lua's row syntax.
func TestParseModMatchesPoBLiveParse(t *testing.T) {
	rows := map[string]string{
		"Fissure Skills have +2 to Limit":                              `{{[1]={[1]={skillType=164,type="SkillType"},flags=0,keywordFlags=0,name="AdditionalCooldownUses",type="BASE",value="2"}},nil}`,
		"Storm Skills have +1 to Limit":                                `{{[1]={[1]={skillType=254,type="SkillType"},flags=0,keywordFlags=0,name="AdditionalCooldownUses",type="BASE",value="1"}},nil}`,
		"Orb Skills have +1 to Limit":                                  `{{[1]={[1]={skillType=101,type="SkillType"},flags=0,keywordFlags=0,name="AdditionalCooldownUses",type="BASE",value="1"}},nil}`,
		"Bogus Skills have +1 to Limit":                                `{nil,nil}`,
		"Skills have +1 to Limit":                                      `{{[1]={flags=0,keywordFlags=0,name="AdditionalCooldownUses",type="BASE",value=1}},nil}`,
		"Temporary Minion Skills have +1 to Limit of Minions summoned": `{{[1]={[1]={baseFlag="duration",neg=true,type="BaseFlag"},flags=0,keywordFlags=0,name="ActiveMinionLimit",type="BASE",value=1}},nil}`,

		"270% increased Spell Damage with Spells that cost Life":              `{{[1]={[1]={statList={[1]="LifeCost",[2]="LifePerSecondCost"},threshold=1,type="StatThreshold"},flags=2,keywordFlags=131072,name="Damage",type="INC",value=270}},nil}`,
		"10% increased Damage with Skills that cost Life":                     `{{[1]={[1]={statList={[1]="LifeCost",[2]="LifePerSecondCost"},threshold=1,type="StatThreshold"},flags=0,keywordFlags=0,name="Damage",type="INC",value=10}},nil}`,
		"Spells which cost Life Gain 108% of Damage as Extra Physical Damage": `{{[1]={[1]={statList={[1]="LifeCost",[2]="LifePerSecondCost"},threshold=1,type="StatThreshold"},flags=0,keywordFlags=131072,name="DamageGainAsPhysical",type="BASE",value=108}},nil}`,

		"Abyssal Wasting also applies -15% to Lightning Resistance": `{{[1]={flags=0,keywordFlags=0,name="AbyssalWastingAlsoGrants",type="LIST",value={applyToEnemy=true,mod={flags=0,keywordFlags=0,name="LightningResist",type="BASE",value=-15}}}},nil}`,
		"Abyssal Wasting also applies -14% to Cold Resistance":      `{{[1]={flags=0,keywordFlags=0,name="AbyssalWastingAlsoGrants",type="LIST",value={applyToEnemy=true,mod={flags=0,keywordFlags=0,name="ColdResist",type="BASE",value=-14}}}},nil}`,
		"Abyssal Wasting also applies -15% to Chaos Resistance":     `{{[1]={flags=0,keywordFlags=0,name="AbyssalWastingAlsoGrants",type="LIST",value={applyToEnemy=true,mod={flags=0,keywordFlags=0,name="ChaosResist",type="BASE",value=-15}}}},nil}`,
		"Abyssal Wasting also applies -15% to Bogus Resistance":     `{nil,nil}`,

		"Has 10 to 15 Physical damage, +4 to +5 per Boss's Face Broken": `{{[1]={flags=0,keywordFlags=0,name="FacebreakerPhysicalMin",type="BASE",value=10},[2]={flags=0,keywordFlags=0,name="FacebreakerPhysicalMax",type="BASE",value=15},[3]={[1]={type="Multiplier",var="BossFaceBroken"},flags=0,keywordFlags=0,name="FacebreakerPhysicalMin",type="BASE",value=4},[4]={[1]={type="Multiplier",var="BossFaceBroken"},flags=0,keywordFlags=0,name="FacebreakerPhysicalMax",type="BASE",value=5}},nil}`,
		"Has 9 to 14 Fire damage, +3 to +5 per Boss's Face Broken":      `{{[1]={flags=0,keywordFlags=0,name="FacebreakerFireMin",type="BASE",value=9},[2]={flags=0,keywordFlags=0,name="FacebreakerFireMax",type="BASE",value=14},[3]={[1]={type="Multiplier",var="BossFaceBroken"},flags=0,keywordFlags=0,name="FacebreakerFireMin",type="BASE",value=3},[4]={[1]={type="Multiplier",var="BossFaceBroken"},flags=0,keywordFlags=0,name="FacebreakerFireMax",type="BASE",value=5}},nil}`,
	}
	for line, row := range rows {
		rhs := (&mpLuaParser{s: row}).value()
		var pob []mpcMod
		pobWhole := rhs.pos[0].kind == 't'
		if pobWhole {
			for _, m := range rhs.pos[0].pos {
				pob = append(pob, mpcPobMod(m))
			}
		}
		modList, extra := ParseMod(line)
		var ours []mpcMod
		for _, m := range modList {
			ours = append(ours, mpcOurMod(m))
		}
		oursWhole := modList != nil && extra == ""
		if c := mpcClassify(pob, pobWhole, ours, oursWhole); c != "" || (!pobWhole && len(ours) > 0) {
			t.Errorf("%q [%s]\n    pob:  %s\n    ours: %s ⟨%s⟩", line, c, mpcField(pob, mpcMod.full), mpcField(ours, mpcMod.full), extra)
		}
	}
}
