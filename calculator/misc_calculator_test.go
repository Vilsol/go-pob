package calculator

import (
	"math"
	"slices"
	"strconv"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/storage"
)

var mainOnlyOutputs = map[string]bool{
	"ExtraPoints": true, "Spec:LifeInc": true, "Spec:ManaInc": true, "Spec:ArmourInc": true,
	"Spec:EvasionInc": true, "Spec:EnergyShieldInc": true,
	"FlaskEffect": true, "FlaskChargeGen": true, "LifeFlaskChargeGen": true, "ManaFlaskChargeGen": true,
	"UtilityFlaskChargeGen": true, "FlaskChargeOnCritChance": true, "TinctureEffect": true, "TinctureLimit": true,
	"IgniteDotMulti": true, "BleedDotMulti": true, "PoisonDotMulti": true,
}

func treeRoots(build *pob.PathOfBuilding) []int64 {
	tree := data.TreeVersions[data.LatestTreeVersion].Tree()
	start := tree.Nodes[classStartNodeID(tree, NewPassiveSpec(build, data.LatestTreeVersion).ClassID)]
	roots := make([]int64, 0)
	for _, id := range append(append([]string{}, start.Out...), start.In...) {
		if n, ok := tree.Nodes[id]; ok && n.AscendancyName == nil && n.Skill != nil {
			roots = append(roots, *n.Skill)
		}
	}
	for _, id := range build.Build.PassiveNodes {
		if n, ok := tree.Nodes[strconv.FormatInt(id, 10)]; ok && n.IsAscendancyStart != nil && *n.IsAscendancyStart {
			roots = append(roots, id)
		}
	}
	return roots
}

func isPlainNode(n data.Node) bool {
	return n.Name != nil && n.ClassStartIndex == nil && (n.IsMastery == nil || !*n.IsMastery) &&
		(n.IsAscendancyStart == nil || !*n.IsAscendancyStart) && n.AscendancyName == nil
}

func pathTo(hops map[int64]int64, target int64) []int64 {
	path := []int64{target}
	for range len(hops) {
		next, ok := hops[target]
		if !ok {
			return nil
		}
		if next == -1 {
			return path
		}
		path = append(path, next)
		target = next
	}
	return nil
}

type whatIfCase struct {
	name   string
	add    []int64
	remove []int64
}

func whatIfCases(t *testing.T, build *pob.PathOfBuilding, pathLen int) []whatIfCase {
	t.Helper()
	version := data.TreeVersions[data.LatestTreeVersion]
	tree := version.Tree()
	active := slices.Clone(build.Build.PassiveNodes)
	roots := treeRoots(build)
	hops := version.CalculateAllocationPaths(active, roots)

	ids := make([]int64, 0, len(hops))
	for id := range hops {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	var cases []whatIfCase
	for _, id := range ids {
		n := tree.Nodes[strconv.FormatInt(id, 10)]
		if !isPlainNode(n) || !data.IsTreeNotable(n) {
			continue
		}
		if p := pathTo(hops, id); len(p) == pathLen {
			cases = append(cases, whatIfCase{name: "allocate " + *n.Name, add: p})
			break
		}
	}

	slices.Sort(active)
	for _, id := range active {
		n, ok := tree.Nodes[strconv.FormatInt(id, 10)]
		if !ok || !isPlainNode(n) || !data.IsTreeNotable(n) || slices.Contains(roots, id) {
			continue
		}
		rest := slices.DeleteFunc(slices.Clone(active), func(a int64) bool { return a == id })
		cut := version.CalculatePrunableNodes(rest, roots)
		cases = append(cases, whatIfCase{name: "deallocate " + *n.Name, remove: append([]int64{id}, cut...)})
		break
	}
	if len(cases) != 2 {
		t.Fatalf("REFUSING: found %d of the 2 cases (path of %d, a deallocatable notable)", len(cases), pathLen)
	}
	return cases
}

func outputsAgree(a, b map[string]float64) (string, bool) {
	for k, av := range a {
		if mainOnlyOutputs[k] {
			continue
		}
		bv, ok := b[k]
		if !ok {
			return k + " missing from the recompute", false
		}
		if av != bv && !(math.IsNaN(av) && math.IsNaN(bv)) && math.Abs(av-bv) > 1e-9*math.Max(math.Abs(av), math.Abs(bv)) {
			return k + ": what-if " + strconv.FormatFloat(av, 'g', -1, 64) + ", recompute " + strconv.FormatFloat(bv, 'g', -1, 64), false
		}
	}
	for k := range b {
		if _, ok := a[k]; !ok && !mainOnlyOutputs[k] {
			return k + " missing from the what-if", false
		}
	}
	return "", true
}

func changedOutputs(a, b map[string]float64) int {
	n := 0
	for k, av := range a {
		if bv := b[k]; av != bv && !(math.IsNaN(av) && math.IsNaN(bv)) {
			n++
		}
	}
	return n
}

// TestMiscCalculatorEqualsRecompute: a node what-if (override addNodes/removeNodes) must equal the build recalculated after the tree page's own mutator.
func TestMiscCalculatorEqualsRecompute(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	for _, file := range []string{
		"0339-reliquarian-shield-crush-of-the-chieftain-2.xml",
		"0008-necromancer-vaal-summon-skeletons.xml",
		"0280-elementalist-fireball.xml",
		"13276-ladder-cuba_5717.cubamrc.xml",
		"4788-ladder-GeordiMcSnuff_1894.GeordiFlameOne.xml",
		"1591-necromancer-summon-holy-relic-of-conviction-15.xml",
		"0028-occultist-forbidden-rite-of-soul-sacrifice.xml",
		"0136-deadeye-elemental-hit-of-the-spectrum-2.xml",
	} {
		t.Run(file, func(t *testing.T) {
			build := loadCorpusBuild(t, file)
			misc := GetMiscCalculator(build)
			before := slices.Clone(build.Build.PassiveNodes)

			for _, c := range whatIfCases(t, build, 3) {
				whatIf := misc.Calc(&CalcOverride{AddNodes: c.add, RemoveNodes: c.remove})
				if !slices.Equal(build.Build.PassiveNodes, before) {
					t.Fatalf("%s: the what-if mutated the build's nodes", c.name)
				}

				mutated := loadCorpusBuild(t, file)
				if len(c.add) > 0 {
					mutated.AllocateNodes(c.add)
				}
				if len(c.remove) > 0 {
					mutated.DeallocateNodes(c.remove)
				}
				recompute := (&Calculator{PoB: mutated}).BuildOutput(OutputModeMain)

				if n := changedOutputs(whatIf.Player.Output, misc.Base().Player.Output); n == 0 {
					t.Errorf("%s: the what-if changed no output — the override did not land", c.name)
				}
				if where, ok := outputsAgree(whatIf.Player.Output, recompute.Player.Output); !ok {
					t.Errorf("%s (%d nodes): player output differs from a recompute: %s", c.name, len(c.add)+len(c.remove), where)
				}
				if (whatIf.Minion == nil) != (recompute.Minion == nil) {
					t.Errorf("%s: minion presence differs from a recompute", c.name)
				} else if whatIf.Minion != nil {
					if where, ok := outputsAgree(whatIf.Minion.Output, recompute.Minion.Output); !ok {
						t.Errorf("%s: minion output differs from a recompute: %s", c.name, where)
					}
				}
			}

			if where, ok := outputsAgree(misc.Base().Player.Output, (&Calculator{PoB: loadCorpusBuild(t, file)}).BuildOutput(OutputModeMain).Player.Output); !ok {
				t.Errorf("calcBase differs from the build's own output: %s", where)
			}
		})
	}
}

// TestMiscCalculatorRemovesGrantedPassive: the tree tooltip's "granted by an item" case removes a passive the build never allocated (CalcSetup.lua:1291).
func TestMiscCalculatorRemovesGrantedPassive(t *testing.T) {
	if err := poe.InitializeAll(raw.DataContext(t.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		t.Fatal(err)
	}

	checked := 0
	for _, file := range []string{
		"0004-trickster-lightning-tendrils-of-eccentricity.xml",
		"0005-occultist-vaal-double-strike.xml",
		"0006-elementalist-storm-burst.xml",
	} {
		build := loadCorpusBuild(t, file)
		misc := GetMiscCalculator(build)
		for id := range misc.Base().GrantedPassives {
			n, err := strconv.ParseInt(id, 10, 64)
			if err != nil || slices.Contains(build.Build.PassiveNodes, n) {
				continue
			}
			checked++
			removed := misc.Calc(&CalcOverride{RemoveNodes: []int64{n}})
			if _, still := removed.GrantedPassives[id]; still {
				t.Errorf("%s: granted passive %s is still granted after removing it", file, id)
			}
			if _, still := removed.AllocatedNodes[id]; still {
				t.Errorf("%s: granted passive %s is still allocated after removing it", file, id)
			}
			if changedOutputs(removed.Player.Output, misc.Base().Player.Output) == 0 {
				t.Errorf("%s: removing granted passive %s changed no output", file, id)
			}
		}
	}
	if checked == 0 {
		t.Fatal("REFUSING: none of the builds has a granted passive")
	}
}
