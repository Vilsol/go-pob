package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
	raw2 "github.com/Vilsol/go-pob/data/raw"
)

// TestMinionsAreUndamagableResolvesInGameData: THIS TEST EXISTS BECAUSE TWO LIVE COMMENTS SAID THE OPPOSITE, AND BOTH WERE BELIEVED.
func TestMinionsAreUndamagableResolvesInGameData(t *testing.T) {
	requireGameData(t, "3.29")

	if len(poe.ActiveSkillTypesByID) == 0 {
		t.Fatalf("REFUSING: ActiveSkillTypesByID is empty; the game data did not load and every " +
			"answer below would read as 'absent'")
	}
	if _, ok := poe.ActiveSkillTypesByID["ZZZNotASkillType"]; ok {
		t.Fatalf("REFUSING: a nonsense id resolved, so this lookup cannot distinguish present " +
			"from absent")
	}

	if _, ok := poe.ActiveSkillTypesByID[string(data.SkillTypeMinionsAreUndamagable)]; !ok {
		t.Fatalf("%s does not resolve in game data, so BOTH guards written against it are silent "+
			"no-ops", data.SkillTypeMinionsAreUndamagable)
	}

	var creates, undamagable, both int
	for _, as := range poe.ActiveSkills {
		types := make(map[data.SkillType]bool)
		for _, st := range as.GetActiveSkillTypes() {
			if st != nil {
				types[data.SkillType(st.ID)] = true
			}
		}
		c := types[data.SkillTypeCreatesMinion]
		u := types[data.SkillTypeMinionsAreUndamagable]
		if c {
			creates++
		}
		if u {
			undamagable++
		}
		if c && u {
			both++
		}
	}

	if creates < 50 {
		t.Fatalf("REFUSING: only %d skills carry CreatesMinion; the type query is broken and the "+
			"count below means nothing", creates)
	}
	if both == 0 {
		t.Fatalf("MinionsAreUndamagable resolves but NO CreatesMinion skill carries it, so both " +
			"guards are unreachable and the port is undemonstrable")
	}
	t.Logf("loaded %s: CreatesMinion=%d MinionsAreUndamagable=%d both=%d",
		raw2.LatestVersion, creates, undamagable, both)

	if both >= creates {
		t.Fatalf("every CreatesMinion skill (%d of %d) is undamagable, which would make both "+
			"guards unconditional", both, creates)
	}
}
