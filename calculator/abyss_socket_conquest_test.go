package calculator

import (
	"os"
	"sort"
	"strconv"
	"testing"

	tjdata "github.com/Vilsol/timeless-jewels/data"
)

var abyss1852PoBConquered = []int{
	476, 2355, 5065, 5152, 6043, 6718, 6884, 8426, 12033, 12236, 12795, 15438, 18302, 20402, 20551,
	24383, 24544, 25933, 27140, 27301, 27422, 28498, 28728, 30691, 30733, 31033, 31080, 31928, 32739,
	34009, 34666, 34959, 35436, 36047, 38777, 39768, 40867, 40907, 42086, 43374, 43412, 45341, 48438,
	49178, 49412, 49538, 50264, 50340, 50360, 53558, 57900, 58449, 59861, 60031, 61198, 63048, 64816,
	65308,
}

// TestAbyssSocketConquestsMatchPoB: PassiveSpec.lua:1117-1143 for a socket-keyed Abyss jewel.
func TestAbyssSocketConquestsMatchPoB(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "1852-ladder-Nyiou_7641.Nyu_Clutch")
	sockets := abyssSockets(env)
	if len(sockets) != 1 || sockets[0].jewelType != tjdata.AbyssTecrod {
		t.Fatalf("want one Tecrod socket, got %+v", sockets)
	}

	rewrites := nodeRewrites(env)
	ours := make([]int, 0, len(abyss1852PoBConquered))
	for id, r := range rewrites {
		if r.stamped {
			n, err := strconv.Atoi(id)
			if err != nil {
				t.Fatal(err)
			}
			ours = append(ours, n)
		}
	}
	sort.Ints(ours)
	if len(ours) != len(abyss1852PoBConquered) {
		t.Fatalf("stamped %d nodes, PoB conquers %d:\nours %v\npob  %v", len(ours), len(abyss1852PoBConquered), ours, abyss1852PoBConquered)
	}
	for i := range ours {
		if ours[i] != abyss1852PoBConquered[i] {
			t.Fatalf("stamped set differs at %d:\nours %v\npob  %v", i, ours, abyss1852PoBConquered)
		}
	}

	keystone := rewrites["40907"].effectiveReplacement()
	if keystone == nil || keystone.ID != "abyss_murderous_keystone" {
		t.Fatalf("40907 (Unwavering Stance) must become abyss_murderous_keystone, got %+v", keystone)
	}
}

// TestAbyssSocketConquestsLeaveRadiusJewelsAlone: Control: a build whose only timeless jewels are radius jewels.
func TestAbyssSocketConquestsLeaveRadiusJewelsAlone(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")

	env := corpusEnvForLoadedData(t, "0006-elementalist-storm-burst")
	if s := abyssSockets(env); len(s) != 0 {
		t.Fatalf("0006 carries no Abyss jewel, got %+v", s)
	}
	jewels := socketedLegionJewels(env)
	if len(jewels) == 0 {
		t.Fatal("REFUSING: 0006 no longer carries a radius timeless jewel, so this controls nothing")
	}

	inRadius := make(map[string]bool)
	radiusNodes := env.Spec.Tree().JewelRadiusNodes()
	for _, j := range jewels {
		for _, id := range radiusNodes[j.socketNodeID][j.radiusIndex] {
			inRadius[strconv.FormatInt(id, 10)] = true
		}
	}
	stamped := 0
	for id, r := range nodeRewrites(env) {
		if !r.stamped {
			continue
		}
		stamped++
		if !inRadius[id] {
			t.Errorf("node %s is stamped outside every radius jewel's radius", id)
		}
	}
	if stamped == 0 {
		t.Fatal("REFUSING: nothing stamped, so this controls nothing")
	}
}
