package calculator2

import (
	"fmt"
	"sync"

	"github.com/Vilsol/go-pob/data/poe2"
	"github.com/Vilsol/go-pob/data/tree2"
)

type progressAct struct {
	Level       int
	QuestPoints int
}

// PoE2's data.questRewards has no Cruel quests, so a quest's act is its Act plus one.
//
//pob2:block:src/Modules/Build.lua:83-102
var progressActs = sync.OnceValue(func() []progressAct {
	acts := []progressAct{{Level: 1, QuestPoints: 0}}
	for _, quest := range poe2.QuestRewards {
		if quest.QuestPoints == 0 {
			continue
		}
		act := int(quest.Act) + 1
		if act > len(acts) {
			acts = append(acts, progressAct{Level: int(quest.AreaLevel), QuestPoints: int(quest.QuestPoints) + acts[act-2].QuestPoints})
		} else {
			acts[act-1].QuestPoints += int(quest.QuestPoints)
			acts[act-1].Level = max(acts[act-1].Level, int(quest.AreaLevel))
		}
	}
	return acts
})

// PlayerProgress is what EstimatePlayerProgress shows in the top bar's point display and adds to the warnings.
type PlayerProgress struct {
	Level          int
	Act            string
	Extra          int
	NormalPassives int
	UsedMax        int
	WeaponSet1Used int
	WeaponSet2Used int
	WeaponSetMax   int
	AscUsed        int
	AscMax         int
	Warnings       []string
}

func EstimatePlayerProgress(spec *tree2.PassiveSpec, output map[string]float64) PlayerProgress {
	pointsUsed, ascUsed, secondaryAscUsed, _, weaponSet1Used, weaponSet2Used := spec.CountAllocNodes()
	return estimatePlayerProgress(pointsUsed, ascUsed, secondaryAscUsed, weaponSet1Used, weaponSet2Used, output)
}

//pob2:src/Modules/Build.lua:1025-1103
func estimatePlayerProgress(pointsUsed, ascUsed, secondaryAscUsed, weaponSet1Used, weaponSet2Used int, output map[string]float64) PlayerProgress {
	extra := int(output["ExtraPoints"])
	acts := progressActs()
	maxActs := len(acts)
	maxWeaponSets := acts[maxActs-1].QuestPoints
	extraWeaponSets := int(output["PassivePointsToWeaponSetPoints"])
	usedMax, ascMax, secondaryAscMax, level, act := 99+maxWeaponSets+extra, 8, 8, 1, 0

	for {
		act++
		level = min(max(pointsUsed+1-acts[act-1].QuestPoints-extra-min(weaponSet1Used, weaponSet2Used), acts[act-1].Level), 100)
		if act == maxActs || level <= acts[act].Level {
			break
		}
	}

	p := PlayerProgress{
		Level: level, Extra: extra, UsedMax: usedMax, AscUsed: ascUsed, AscMax: ascMax,
		WeaponSet1Used: weaponSet1Used, WeaponSet2Used: weaponSet2Used, WeaponSetMax: maxWeaponSets + extraWeaponSets,
	}
	warn := func(line string) {
		for _, l := range p.Warnings {
			if l == line {
				return
			}
		}
		p.Warnings = append(p.Warnings, line)
	}

	p.NormalPassives = pointsUsed - min(weaponSet1Used, weaponSet2Used)
	if p.NormalPassives > usedMax {
		warn("You have too many passive points allocated")
	}
	if ascUsed > ascMax {
		warn("You have too many ascendancy points allocated")
	}
	if secondaryAscUsed > secondaryAscMax {
		warn("You have too many secondary ascendancy points allocated")
	}

	warningsWeaponSet := false
	if weaponSet1Used > p.WeaponSetMax {
		warningsWeaponSet = true
		warn(fmt.Sprintf("You have allocated %d too many weapon set 1 passives", abs(p.WeaponSetMax-weaponSet1Used)))
	}
	if weaponSet2Used > p.WeaponSetMax {
		warningsWeaponSet = true
		warn(fmt.Sprintf("You have allocated %d too many weapon set 2 passives", abs(p.WeaponSetMax-weaponSet2Used)))
	}
	if !warningsWeaponSet && weaponSet1Used != weaponSet2Used {
		set := 2
		if weaponSet1Used < weaponSet2Used {
			set = 1
		}
		warn(fmt.Sprintf("You have %d Weapon set %d passives available", abs(weaponSet2Used-weaponSet1Used), set))
	}

	if act == maxActs {
		p.Act = "Endgame"
	} else {
		p.Act = fmt.Sprintf("Act %d", act)
	}
	return p
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
