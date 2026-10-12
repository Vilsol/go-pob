package data

import (
	"strings"
	"testing"
)

// TestAscendancyMapCarriesMultipleChoiceOptions pins the SECOND arm of PassiveTree.lua:574-576.
func TestAscendancyMapCarriesMultipleChoiceOptions(t *testing.T) {
	version := TreeVersions[LatestTreeVersion]
	if version == nil {
		t.Skipf("NOT-RUN: no tree registered for %q", LatestTreeVersion)
	}
	nodes := version.Tree().Nodes
	byID := version.AscendancyMap()

	if len(nodes) < 1000 || len(byID) < 50 {
		t.Fatalf("REFUSING: %d tree nodes and %d ascendancyMap entries; the tree did not load and "+
			"every assertion below would pass by not being reached", len(nodes), len(byID))
	}

	ids := map[string]string{}
	for name, id := range byID {
		ids[id] = name
	}

	var missing, options int
	for id, node := range nodes {
		if node.Name == nil || node.AscendancyName == nil {
			continue
		}
		if node.IsMultipleChoiceOption == nil || !*node.IsMultipleChoiceOption {
			continue
		}
		if *node.AscendancyName == "Reliquarian" || *node.AscendancyName == "Luminary" {
			continue
		}
		if node.IsNotable != nil && *node.IsNotable {
			continue
		}
		options++
		if _, ok := byID[strings.ToLower(*node.Name)]; !ok {
			missing++
			if missing <= 5 {
				t.Errorf("MISSING FROM ascendancyMap: node %s %q (%s) is an isMultipleChoiceOption "+
					"and PassiveTree.lua:576's second arm puts it in ascendancyMap. Forbidden "+
					"Flame/Flesh resolves its grant BY NAME through this map, so an absent entry "+
					"silently grants nothing.", id, *node.Name, *node.AscendancyName)
			}
		}
	}

	if options < 10 {
		t.Fatalf("REFUSING: only %d non-notable multiple-choice ascendancy options found in the "+
			"tree; this test has no subject and its silence means nothing", options)
	}
	if missing > 0 {
		t.Errorf("%d of %d multiple-choice ascendancy options are absent from ascendancyMap", missing, options)
	}

	if got := byID["knife in the back"]; got != "21264" {
		t.Errorf("ascendancyMap[\"knife in the back\"] = %q, want \"21264\" (Assassin, "+
			"isMultipleChoiceOption). This is the node 39 corpus builds grant through Forbidden "+
			"Flame/Flesh.", got)
	}
	t.Logf("ascendancyMap: %d entries, %d non-notable multiple-choice options all present", len(byID), options)
}
