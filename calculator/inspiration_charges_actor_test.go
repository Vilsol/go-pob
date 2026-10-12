package calculator

import (
	"os"
	"testing"
)

// TestInspirationChargesAreWrittenOnThePlayerActorOnly pins CalcPerform.lua:1060-1062.
//
//	if actor == env.player then
//	    output.InspirationCharges = modDB:Override(nil, "InspirationCharges") or output.InspirationChargesMax
//	end
func TestInspirationChargesAreWrittenOnThePlayerActorOnly(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	oracle, err := oracleForLoadedData()
	if err != nil {
		t.Fatalf("REFUSING: cannot load the oracle blob: %v", err)
	}
	_, names := corpusNamesForLoadedData(t)

	var decisive, enemyWrong, playerSeen, playerWrong int

	for _, name := range names {
		rec, ok := oracle[name]
		if !ok || rec.TreeSources < oracleMinTreeSources {
			continue
		}
		want, has := rec.EnemyOutput["InspirationCharges"]
		if !has {
			continue
		}
		env := corpusEnvForLoadedData(t, name)
		if env.Enemy == nil {
			continue
		}
		if env.Enemy.Output["InspirationChargesMax"] <= 0 {
			continue
		}
		decisive++
		if !oracleClose(env.Enemy.Output["InspirationCharges"], want, oracleDumpTolerance) {
			enemyWrong++
		}
		if pw, hasPlayer := rec.Output["InspirationCharges"]; hasPlayer {
			playerSeen++
			if !oracleClose(env.Player.Output["InspirationCharges"], pw, oracleDumpTolerance) {
				playerWrong++
			}
		}
		if decisive >= 200 {
			break
		}
	}

	if decisive == 0 {
		t.Fatal("REFUSING: no graded build gives the enemy a nonzero InspirationChargesMax, so the " +
			"actor-identity branch decides nothing and this test no longer measures its mechanism")
	}
	if playerSeen == 0 {
		t.Fatal("REFUSING: the blob publishes no player InspirationCharges, so the two-sided control " +
			"is absent and a fix that simply deleted the write would pass")
	}
	if enemyWrong > 0 {
		t.Errorf("enemy InspirationCharges disagrees with PoB on %d of %d decisive builds", enemyWrong, decisive)
	}
	if playerWrong > 0 {
		t.Errorf("player InspirationCharges disagrees with PoB on %d of %d builds", playerWrong, playerSeen)
	}
}
