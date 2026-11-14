package mcp

//pob:block:src/Modules/Build.lua:729-764
var acts = []struct{ level, questPoints int }{
	{1, 0}, {12, 2}, {22, 4}, {32, 6}, {40, 7}, {44, 9}, {50, 12}, {54, 15}, {60, 18}, {64, 20}, {67, 23},
}

//pob:src/Modules/Build.lua:766-769
func actExtra(act, extra int) int {
	if act > 2 {
		return extra
	}
	return 0
}

type progress struct {
	Level    int
	Warnings []string
}

//pob:src/Modules/Build.lua:898-927
func estimatePlayerProgress(pointsUsed, ascUsed, secondaryAscUsed, extra int) progress {
	usedMax, ascMax, secondaryAscMax, level, act := 99+23+extra, 8, 8, 1, 0
	for {
		act++
		level = min(max(pointsUsed+1-acts[act-1].questPoints-actExtra(act, extra), acts[act-1].level), 100)
		if act == 11 || level <= acts[act].level {
			break
		}
	}
	var warnings []string
	if pointsUsed > usedMax {
		warnings = append(warnings, "You have too many passive points allocated")
	}
	if ascUsed > ascMax {
		warnings = append(warnings, "You have too many ascendancy points allocated")
	}
	if secondaryAscUsed > secondaryAscMax {
		warnings = append(warnings, "You have too many secondary ascendancy points allocated")
	}
	return progress{Level: level, Warnings: warnings}
}
