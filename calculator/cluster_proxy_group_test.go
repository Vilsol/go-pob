package calculator

import "testing"

// TestClusterSubgraphKeepsProxyGroup: a v2 cluster hash spec builds a medium jewel in a large socket on the large proxy group.
func TestClusterSubgraphKeepsProxyGroup(t *testing.T) {
	requireGameData(t, "3.29")
	env := corpusEnvForLoadedData(t, "1243-elementalist-penance-brand-33.xml")
	for k, want := range map[string]float64{"Spec:ArmourInc": 38, "Armour": 4252, "BlockChance": 73} {
		if got := env.Player.Output[k]; got != want {
			t.Errorf("%s = %v, want %v (PoB)", k, got, want)
		}
	}
}
