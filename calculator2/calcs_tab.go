package calculator2

import (
	"github.com/Vilsol/go-pob/pob2"
)

// CalcsInputs is calcsTab.input as the Calcs tab reads it: the socket group it shows (1-based), its buff mode and
// whether it shows the minion.
type CalcsInputs struct {
	SkillNumber int
	BuffMode    string
	ShowMinion  bool
}

func calcsInput(build *pob2.PathOfBuilding2, name string) *pob2.Input {
	for i := range build.Calcs.Inputs {
		if build.Calcs.Inputs[i].Name == name {
			return &build.Calcs.Inputs[i]
		}
	}
	return nil
}

// readCalcsInputs is calcsTab.input after the tab's constructor and Load.
//
//pob2:block:src/Classes/CalcsTab.lua:31-33
//pob2:block:src/Classes/CalcsTab.lua:200-217
func readCalcsInputs(build *pob2.PathOfBuilding2) CalcsInputs {
	in := CalcsInputs{SkillNumber: 1, BuffMode: "EFFECTIVE"}
	if v := calcsInput(build, "skill_number"); v != nil && v.Number != nil {
		in.SkillNumber = int(*v.Number)
	}
	if v := calcsInput(build, "misc_buffMode"); v != nil && v.String != nil {
		in.BuffMode = *v.String
	}
	if v := calcsInput(build, "showMinion"); v != nil && v.Boolean != nil {
		in.ShowMinion = *v.Boolean
	}
	return in
}

func ReadCalcsInputs(build *pob2.PathOfBuilding2) CalcsInputs {
	return readCalcsInputs(build)
}

func setCalcsInput(build *pob2.PathOfBuilding2, in pob2.Input) {
	if v := calcsInput(build, in.Name); v != nil {
		*v = in
		return
	}
	build.Calcs.Inputs = append(build.Calcs.Inputs, in)
}

// SetCalcsInputNumber, SetCalcsInputString and SetCalcsInputBool are the tab's controls writing self.input[name].
//
//pob2:block:src/Classes/CalcsTab.lua:43-46
func SetCalcsInputNumber(build *pob2.PathOfBuilding2, name string, value float64) {
	setCalcsInput(build, pob2.Input{Name: name, Number: &value})
}

//pob2:block:src/Classes/CalcsTab.lua:165-168
func SetCalcsInputString(build *pob2.PathOfBuilding2, name string, value string) {
	setCalcsInput(build, pob2.Input{Name: name, String: &value})
}

//pob2:block:src/Classes/CalcsTab.lua:102-105
func SetCalcsInputBool(build *pob2.PathOfBuilding2, name string, value bool) {
	setCalcsInput(build, pob2.Input{Name: name, Boolean: &value})
}

// SetMainActiveSkillCalcs is the tab's Active Skill dropdown: the shown socket group's mainActiveSkillCalcs.
//
//pob2:block:src/Classes/CalcsTab.lua:58-63
func SetMainActiveSkillCalcs(build *pob2.PathOfBuilding2, env *Environment, index int) error {
	group, err := editGroup(build, env, readCalcsInputs(build).SkillNumber)
	if err != nil {
		return err
	}
	group.MainActiveSkillCalcs = pob2.Nil[int]{V: index}
	return nil
}

// BuildCalcsOutput is calcs.buildOutput(build, "CALCS") after the main pass: the Calcs tab's own calculation over the
// socket group list the main pass left, with the tab's lists. main and used are calcsTab.mainEnv and its tables.
//
//pob2:block:src/Classes/CalcsTab.lua:512-513
//pob2:block:src/Modules/Calcs.lua:518-531
func BuildCalcsOutput(main *Environment, used *UsedTables) *Environment {
	env := newWeaponSetEnv(main, "CALCS", calcOverride{})
	initEnv(env)
	perform(env)
	setFullDPS(env, calcFullDPS(env, "CALCULATOR", calcOverride{}))
	calcsBuffLists(env, used.MultipliersUsed)
	return env
}

// CalcsMainSkill is RefreshSkillSelectControls(controls, input.skill_number, "Calcs")'s Active Skill dropdown: the
// shown group's displaySkillListCalcs and its mainActiveSkillCalcs.
//
//pob2:block:src/Modules/Build.lua:2078-2090
func CalcsMainSkill(env *Environment) ([]string, int) {
	skills := []string{}
	if env.MainSocketGroup < 1 || env.MainSocketGroup > len(env.SocketGroupList) {
		return skills, 0
	}
	group := env.SocketGroupList[env.MainSocketGroup-1]
	// displaySkillListCalcs is the group's socketGroupSkillList, CalcSetup.lua:2432-2434
	if group.UsingSkillSet == env.WeaponSet {
		for _, activeSkill := range env.Player.ActiveSkillList {
			if activeSkill.SocketGroup == group {
				skills = append(skills, getActiveSkillDisplayName(activeSkill))
			}
		}
	}
	mainActiveSkill := group.MainActiveSkillCalcs
	if mainActiveSkill == 0 {
		mainActiveSkill = 1
	}
	return skills, mainActiveSkill
}

// SyncCalcsSkillSelection points a compared build's Calcs tab at its main socket group and active skill, as loading it
// does.
//
//pob2:block:src/Classes/CompareEntry.lua:232-238
func SyncCalcsSkillSelection(build *pob2.PathOfBuilding2) {
	SetCalcsInputNumber(build, "skill_number", float64(build.Build.MainSocketGroup))
	mainGroup := savedSocketGroup(build, build.Build.MainSocketGroup)
	if mainGroup == nil {
		return
	}
	mainGroup.MainActiveSkillCalcs = mainGroup.MainActiveSkill
}
