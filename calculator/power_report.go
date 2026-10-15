package calculator

import (
	"container/heap"
	"maps"
	"math"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

type PowerStat struct {
	Stat     string
	Label    string
	Negate   bool
	Combined bool
}

//pob:block:src/Modules/Data.lua:132-179
var powerStatList = []PowerStat{
	{"FullDPS", "Full DPS", false, false}, {"CombinedDPS", "Combined DPS", false, false}, {"TotalDPS", "Hit DPS", false, false},
	{"WithImpaleDPS", "Impale + Hit DPS", false, false}, {"AverageDamage", "Average Hit", false, false}, {"Speed", "Attack/Cast Speed", false, false},
	{"TotalDot", "DoT DPS", false, false}, {"BleedDPS", "Bleed DPS", false, false}, {"IgniteDPS", "Ignite DPS", false, false},
	{"PoisonDPS", "Poison DPS", false, false}, {"Life", "Life", false, false}, {"LifeRegen", "Life regen", false, false},
	{"LifeLeechRate", "Life leech", false, false}, {"Armour", "Armour", false, false}, {"Evasion", "Evasion", false, false},
	{"EnergyShield", "Energy Shield", false, false}, {"EnergyShieldRecoveryCap", "Recoverable ES", false, false},
	{"EnergyShieldRegen", "Energy Shield regen", false, false}, {"EnergyShieldLeechRate", "Energy Shield leech", false, false},
	{"Mana", "Mana", false, false}, {"ManaRegen", "Mana regen", false, false}, {"ManaLeechRate", "Mana leech", false, false},
	{"Ward", "Ward", false, false}, {"Str", "Strength", false, false}, {"Dex", "Dexterity", false, false}, {"Int", "Intelligence", false, false},
	{"TotalAttr", "Total Attributes", false, false}, {"MeleeAvoidChance", "Melee avoid chance", false, false},
	{"SpellAvoidChance", "Spell avoid chance", false, false}, {"ProjectileAvoidChance", "Projectile avoid chance", false, false},
	{"TotalEHP", "Effective Hit Pool", false, false}, {"SecondMinimalMaximumHitTaken", "Eff. Maximum Hit Taken", false, false},
	{"PhysicalTakenHit", "Taken Phys dmg", true, false}, {"LightningTakenHit", "Taken Lightning dmg", true, false},
	{"ColdTakenHit", "Taken Cold dmg", true, false}, {"FireTakenHit", "Taken Fire dmg", true, false},
	{"ChaosTakenHit", "Taken Chaos dmg", true, false}, {"CritChance", "Crit Chance", false, false},
	{"CritMultiplier", "Crit Multiplier", false, false}, {"BleedChance", "Bleed Chance", false, false},
	{"FreezeChance", "Freeze Chance", false, false}, {"IgniteChance", "Ignite Chance", false, false},
	{"ShockChance", "Shock Chance", false, false}, {"EffectiveMovementSpeedMod", "Move speed", false, false},
	{"LightRadiusMod", "Light Radius", false, false}, {"BlockChance", "Block Chance", false, false},
	{"SpellBlockChance", "Spell Block Chance", false, false}, {"SpellSuppressionChance", "Spell Suppression Chance", false, false},
}

//pob:block:src/Modules/Data.lua:214-235,src/Classes/TreeTab.lua:264-269
func PowerStats() []PowerStat {
	notMinion := map[string]bool{"AverageDamage": true, "TotalDot": true, "Str": true, "Dex": true, "Int": true, "Spirit": true,
		"EffectiveLootRarityMod": true, "LightRadiusMod": true}
	out := slices.Clone(powerStatList)
	for _, s := range powerStatList {
		if strings.Contains(s.Stat, "DPS") || notMinion[s.Stat] {
			continue
		}
		out = append(out, PowerStat{Stat: "Minion" + s.Stat, Label: "Minion " + s.Label, Negate: s.Negate})
	}
	return out
}

//pob:block:src/Modules/Data.lua:130
func NodePowerStats() []PowerStat {
	return append([]PowerStat{{Label: "Offence/Defence", Combined: true}}, PowerStats()...)
}

//pob:src/Classes/CalcsTab.lua:748-758
func combinedOffDefStat(original, modified *Environment) (float64, float64) {
	o, m := original.Player.Output, modified.Player.Output
	es := func(out map[string]float64) float64 {
		if v, ok := out["EnergyShieldRecoveryCap"]; ok {
			return v
		}
		return out["EnergyShield"]
	}
	defence := (o["LifeUnreserved"]-m["LifeUnreserved"])/math.Max(3000, m["Life"]) +
		(o["Armour"]-m["Armour"])/math.Max(10000, m["Armour"]) +
		(es(o)-es(m))/math.Max(3000, es(m)) +
		(o["Evasion"]-m["Evasion"])/math.Max(10000, m["Evasion"]) +
		(o["LifeRegenRecovery"]-m["LifeRegenRecovery"])/500 +
		(o["EnergyShieldRegenRecovery"]-m["EnergyShieldRegenRecovery"])/1000
	minionDPS := func(env *Environment) float64 {
		if env.Minion != nil {
			return env.Minion.Output["CombinedDPS"]
		}
		return 0
	}
	modifiedDps := m["CombinedDPS"] + minionDPS(modified)
	dpsIncr := o["CombinedDPS"] + minionDPS(original) - modifiedDps
	return dpsIncr / modifiedDps, defence
}

//pob:src/Modules/Data.lua:186-212
func powerStatValue(env *Environment, stat PowerStat) float64 {
	out := env.Player.Output
	var minion map[string]float64
	if env.Minion != nil {
		minion = env.Minion.Output
	}
	entry := func() float64 {
		if stat.Stat == "FullDPS" {
			if v, ok := out[stat.Stat]; ok {
				return v
			}
			return out["CombinedDPS"] + minion["CombinedDPS"]
		}
		if m, ok := strings.CutPrefix(stat.Stat, "Minion"); ok {
			return minion[m]
		}
		if stat.Stat == "AverageDamage" || stat.Stat == "TotalDot" || strings.Contains(stat.Stat, "DPS") {
			return out[stat.Stat] + minion[stat.Stat]
		}
		return out[stat.Stat]
	}
	if stat.Negate {
		return -entry()
	}
	return entry()
}

type PowerReportEntry struct {
	ID        int64
	Name      string
	Type      string
	Power     float64
	PathPower float64
	PathDist  int
	Allocated bool
	Stats     []string
	Cluster   bool
	Offence   float64
	Defence   float64
	Effect    int64
	Picked    bool
}

type nodePower struct {
	single, path       float64
	hasSingle, hasPath bool
	distance           int
	defence            float64
}

type PowerBuilder struct {
	misc     *MiscCalculator
	stat     PowerStat
	maxDepth int
	base     float64

	tree      *data.Tree
	alloc     map[int64]bool
	granted   map[int64]bool
	paths     map[int64]int64
	roots     []int64
	rewrites  map[string]nodeRewrite
	dependsOf map[int64][]int64
	subgraph  map[int64]data.Node
	graph     *powerGraph

	nodes        []int64
	keys         map[int64]string
	power        map[int64]*nodePower
	mastery      map[int64]map[int64]*nodePower
	clusterNodes []int64
	cluster      map[int64]float64
	cache        map[string]float64
	combined     map[string][2]float64
	spec         *calcSpecEnv
	specInputs   []pob.Input

	tasks     []func()
	next      int
	calcs     int
	taskNodes []int64
	last      *Environment
	spare     spareOutputs
}

func (b *PowerBuilder) treeNode(id int64) data.Node {
	if n, ok := b.tree.Nodes[strconv.FormatInt(id, 10)]; ok {
		return n
	}
	return b.subgraph[id]
}

//pob:block:src/Classes/PassiveTree.lua:546-577
func powerNodeType(n data.Node) string {
	switch {
	case isClassStartNode(n):
		return "ClassStart"
	case isAscendClassStartNode(n):
		return "AscendClassStart"
	case isMasteryNode(n):
		return "Mastery"
	case n.IsJewelSocket != nil && *n.IsJewelSocket:
		return "Socket"
	case n.IsKeystone != nil && *n.IsKeystone:
		return "Keystone"
	case n.IsNotable != nil && *n.IsNotable:
		return "Notable"
	}
	return "Normal"
}

//pob:block:src/Classes/PassiveSpec.lua:54,src/Classes/PassiveTree.lua:132-163
func specNode(tree *data.Tree, n data.Node) bool {
	if n.Group == nil || (n.IsProxy != nil && *n.IsProxy) || (n.ExpansionJewel != nil && n.ExpansionJewel.Parent != nil) {
		return false
	}
	if n.AscendancyName != nil && legacyAlternateAscendancies[*n.AscendancyName] &&
		slices.ContainsFunc(tree.AlternateAscendancies, func(a data.Ascendancy) bool { return string(a.ID) == *n.AscendancyName }) {
		return false
	}
	g, ok := tree.Groups[strconv.FormatInt(*n.Group, 10)]
	return ok && (g.IsProxy == nil || !*g.IsProxy)
}

func allocationPath(paths map[int64]int64, id int64) []int64 {
	path := []int64{id}
	for range len(paths) + 1 {
		hop, ok := paths[id]
		if !ok {
			return nil
		}
		if hop == -1 || paths[hop] == -1 {
			return path
		}
		path = append(path, hop)
		id = hop
	}
	return nil
}

//pob:block:src/Classes/PassiveSpec.lua:1470-1624
func (b *PowerBuilder) depends(id int64) []int64 {
	if len(b.dependsOf) == 0 {
		extra := map[int64][]int64{}
		for key, n := range b.misc.base.clusterNodes {
			if c, err := strconv.ParseInt(key, 10, 64); err == nil {
				for _, out := range n.Out {
					if other, err := strconv.ParseInt(out, 10, 64); err == nil {
						extra[c] = append(extra[c], other)
						extra[other] = append(extra[other], c)
					}
				}
			}
		}
		version := data.TreeVersions[data.LatestTreeVersion]
		node := func(id int64) (data.DependsNode, bool) {
			d, ok := version.SpecDependsNode(id)
			if !ok {
				n, sub := b.subgraph[id]
				if !sub {
					return d, false
				}
				d = data.DependsNode{Type: powerNodeType(n)}
			}
			links := slices.Clone(extra[id])
			slices.Sort(links)
			for _, other := range links {
				if !slices.Contains(d.Linked, other) {
					d.Linked = append(d.Linked, other)
				}
			}
			return d, true
		}
		alloc := allocSet(b.misc.build, b.tree)
		b.dependsOf = data.SpecDepends(slices.Collect(maps.Keys(alloc)), node, leapJewels(b.misc.build, b.tree, alloc))
	}
	if d, ok := b.dependsOf[id]; ok {
		return d
	}
	return []int64{id}
}

//pob:block:src/Classes/CalcsTab.lua:496-604
func (m *MiscCalculator) NewPowerBuilder(stat PowerStat, maxDepth int) *PowerBuilder {
	b := &PowerBuilder{
		misc: m, stat: stat, maxDepth: maxDepth,
		tree:      data.TreeVersions[data.LatestTreeVersion].Tree(),
		alloc:     map[int64]bool{},
		granted:   map[int64]bool{},
		dependsOf: map[int64][]int64{},
		keys:      map[int64]string{},
		power:     map[int64]*nodePower{},
		mastery:   map[int64]map[int64]*nodePower{},
		cluster:   map[int64]float64{},
		cache:     map[string]float64{},
		combined:  map[string][2]float64{},
	}
	b.base = b.value(m.base)
	for _, id := range m.build.Build.PassiveNodes {
		b.alloc[id] = true
	}
	for id := range m.base.GrantedPassives {
		if n, err := strconv.ParseInt(id, 10, 64); err == nil {
			b.granted[n] = true
		}
	}
	if classID, ok := pob.ClassIDByName(m.build.Build.ClassName); ok {
		start := b.tree.Nodes[classStartNodeID(b.tree, classID)]
		for _, id := range linkedNodeIDs(start) {
			if n, ok := b.tree.Nodes[id]; ok && n.Skill != nil && n.AscendancyName == nil {
				b.roots = append(b.roots, *n.Skill)
			}
		}
	}
	for key, n := range m.base.clusterNodes {
		if _, ok := b.tree.Nodes[key]; ok {
			continue
		}
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			if b.subgraph == nil {
				b.subgraph = map[int64]data.Node{}
			}
			b.subgraph[id] = n
		}
	}
	if b.subgraph != nil {
		b.graph = newPowerGraph(b.tree, m.base.clusterNodes)
	}
	//pob:block:src/Classes/PassiveSpec.lua:1169
	leap := map[int64]bool{}
	for id := range leapHeld(m.build, b.tree, allocSet(m.build, b.tree)) {
		if n := b.treeNode(id); powerNodeType(n) != "ClassStart" && powerNodeType(n) != "Socket" && (n.AscendancyName == nil || *n.AscendancyName == "") {
			leap[id] = true
		}
	}
	//pob:block:src/Classes/PassiveSpec.lua:1627-1692
	seeds := m.build.Build.PassiveNodes
	if len(leap) > 0 {
		roots := slices.Clone(b.roots)
		for _, id := range seeds {
			if n := b.treeNode(id); n.IsAscendancyStart != nil && *n.IsAscendancyStart {
				roots = append(roots, id)
			}
		}
		var unconnected []int64
		if b.graph != nil {
			unconnected = b.graph.prunable(seeds, roots)
		} else {
			unconnected = data.TreeVersions[data.LatestTreeVersion].CalculatePrunableNodes(seeds, roots)
		}
		seeds = slices.DeleteFunc(slices.Clone(seeds), func(id int64) bool { return leap[id] && slices.Contains(unconnected, id) })
	}
	if b.graph != nil {
		b.paths = b.graph.allocationPaths(seeds, b.roots)
	} else {
		b.paths = data.TreeVersions[data.LatestTreeVersion].CalculateAllocationPaths(seeds, b.roots)
	}
	b.rewrites = nodeRewrites(m.base)
	selections := map[int64]int64{}
	for _, s := range activeMasterySelections(m.build) {
		selections[s.Node] = s.Effect
	}

	distanceMap := map[int][]int64{}
	var masteryNodes []int64
	specNodes := map[int64]data.Node{}
	for key, n := range b.tree.Nodes {
		if !specNode(b.tree, n) {
			continue
		}
		if id, err := strconv.ParseInt(key, 10, 64); err == nil {
			specNodes[id] = n
		}
	}
	maps.Copy(specNodes, b.subgraph)
	for id, n := range specNodes {
		b.nodes = append(b.nodes, id)
		b.power[id] = &nodePower{}
		b.keys[id] = b.modKey(id, selections)
		if b.keys[id] == "" || b.granted[id] {
			continue
		}
		pathDist := 1000
		if b.alloc[id] {
			pathDist = 0
		} else if p := allocationPath(b.paths, id); p != nil {
			pathDist = len(p)
		}
		if isMasteryNode(n) && !b.alloc[id] {
			if maxDepth == 0 || pathDist <= maxDepth {
				masteryNodes = append(masteryNodes, id)
			}
			continue
		}
		//pob:block:src/Classes/CalcsTab.lua:579-584,src/Classes/PassiveSpec.lua:1169
		if leap[id] {
			pathDist = 1
		}
		b.power[id].distance = pathDist
		distanceMap[pathDist] = append(distanceMap[pathDist], id)
	}
	slices.Sort(b.nodes)
	distances := make([]int, 0, len(distanceMap))
	for d := range distanceMap {
		distances = append(distances, d)
	}
	sort.Ints(distances)
	for _, d := range distances {
		if maxDepth != 0 && maxDepth < d {
			break
		}
		ids := distanceMap[d]
		slices.Sort(ids)
		for _, id := range ids {
			b.tasks = append(b.tasks, func() { b.nodeTask(id, d, selections) })
			b.taskNodes = append(b.taskNodes, id)
		}
	}
	slices.Sort(masteryNodes)
	for _, id := range masteryNodes {
		for _, effect := range b.treeNode(id).MasteryEffects {
			if held, ok := heldBy(selections, effect.Effect); ok && held != id {
				continue
			}
			if statsModKey(effect.Stats) == "" {
				continue
			}
			b.tasks = append(b.tasks, func() { b.masteryTask(id, effect) })
			b.taskNodes = append(b.taskNodes, id)
		}
	}
	byName := map[string]int64{}
	for _, n := range b.tree.Nodes {
		if n.Group != nil || n.Name == nil || n.Skill == nil || !((n.IsNotable != nil && *n.IsNotable) || (n.IsKeystone != nil && *n.IsKeystone)) ||
			(n.AscendancyName != nil && legacyAlternateAscendancies[*n.AscendancyName]) {
			continue
		}
		if id, ok := byName[*n.Name]; !ok || *n.Skill < id {
			byName[*n.Name] = *n.Skill
		}
	}
	for _, id := range byName {
		if !b.alloc[id] {
			b.clusterNodes = append(b.clusterNodes, id)
		}
	}
	slices.Sort(b.clusterNodes)
	for _, id := range b.clusterNodes {
		if key := statsModKey(b.treeNode(id).Stats); key != "" && !b.granted[id] && !stat.Combined {
			b.tasks = append(b.tasks, func() { b.cluster[id] = b.calc(key, &CalcOverride{AddNodes: []int64{id}}) })
			b.taskNodes = append(b.taskNodes, id)
		}
	}
	return b
}

func heldBy(selections map[int64]int64, effect int64) (int64, bool) {
	for node, e := range selections {
		if e == effect {
			return node, true
		}
	}
	return 0, false
}

//pob:block:src/Classes/PassiveTree.lua:840-850
func statsModKey(stats []string) string {
	ml := moddb.NewModList()
	processStats(&Environment{}, ml, "", "", stats)
	if len(ml.Mods()) == 0 {
		return ""
	}
	return strings.Join(stats, "\n")
}

//pob:block:src/Classes/PassiveSpec.lua:1697-1706,src/Classes/PassiveSpec.lua:2468-2476
func (b *PowerBuilder) specNodeStats(id int64) (string, []string) {
	n := b.treeNode(id)
	name := ""
	if n.Name != nil {
		name = *n.Name
	}
	r, ok := b.rewrites[strconv.FormatInt(id, 10)]
	if !ok {
		return name, n.Stats
	}
	stats := n.Stats
	if rep := r.effectiveReplacement(); rep != nil {
		name, stats = rep.DN, rep.Stats
	}
	for _, line := range r.addStats {
		if line != "" {
			stats = append(slices.Clip(stats), line)
		}
	}
	return name, stats
}

func (b *PowerBuilder) modKey(id int64, selections map[int64]int64) string {
	n := b.treeNode(id)
	if !isMasteryNode(n) {
		_, stats := b.specNodeStats(id)
		return statsModKey(stats)
	}
	var lines []string
	for _, e := range n.MasteryEffects {
		if !b.alloc[id] || selections[id] == e.Effect {
			lines = append(lines, e.Stats...)
		}
	}
	if key := statsModKey(lines); key != "" {
		return "mastery:" + key
	}
	return ""
}

//pob:block:src/Modules/Calcs.lua:101-111
func (b *PowerBuilder) run(override *CalcOverride) *Environment {
	b.calcs++
	if b.last != nil {
		b.spare, b.last = spareOutputsOf(b.last), nil
	}
	runtime.Gosched()
	fullDPS := b.stat.Stat == "FullDPS" && b.misc.usedFullDPS
	if b.spec == nil || !sameInputs(b.specInputs, b.misc.build.Config.Inputs) {
		inputs := cloneInputs(b.misc.build.Config.Inputs)
		env, player, enemy, minion := initEnvSpec(b.misc.build, b.misc.cache, OutputModeCalculator, false, override, nil)
		b.spec = &calcSpecEnv{cachedPlayerDB: player, cachedEnemyDB: enemy, cachedMinionDB: minion, flatDB: true,
			clusterNodes: env.clusterNodes, clusterNodesSet: true, distanceToClassStart: env.Spec.DistanceToClassStart}
		b.specInputs = inputs
		PerformCalc(env)
		if fullDPS {
			b.rollUp(env, env.override)
		}
		b.last = env
		return env
	}
	if fullDPS {
		env := &Environment{Player: &Actor{Output: map[string]float64{}}}
		b.rollUp(env, override)
		return env
	}
	env, _, _, _ := initEnvSpecKeep(b.misc.build, b.misc.cache, OutputModeCalculator, false, override, b.spec, false)
	env.spare, b.spare = b.spare, spareOutputs{}
	PerformCalc(env)
	b.last = env
	return env
}

//pob:block:src/Modules/Calcs.lua:107-110
func (b *PowerBuilder) rollUp(env *Environment, override *CalcOverride) {
	fullDPS, fullDotDPS, _ := calcFullDPS(&Calculator{PoB: b.misc.build}, env, OutputModeCalculator, b.misc.cache, override,
		&calcSpecEnv{cachedPlayerDB: b.spec.cachedPlayerDB, cachedEnemyDB: b.spec.cachedEnemyDB, cachedMinionDB: b.spec.cachedMinionDB,
			clusterNodes: b.spec.clusterNodes, clusterNodesSet: true, spare: &b.spare, distanceToClassStart: b.spec.distanceToClassStart})
	env.Player.Output["FullDPS"] = fullDPS
	env.Player.Output["FullDotDPS"] = fullDotDPS
}

//pob:block:src/Classes/CalcsTab.lua:613-616
func (b *PowerBuilder) calc(key string, override *CalcOverride) float64 {
	if b.stat.Combined {
		v, _ := b.calcCombined(key, override)
		return v
	}
	v, ok := b.cache[key]
	if !ok {
		v = b.value(b.run(override))
		b.cache[key] = v
	}
	return v - b.base
}

//pob:block:src/Classes/CalcsTab.lua:547-556
func (b *PowerBuilder) calcCombined(key string, override *CalcOverride) (float64, float64) {
	v, ok := b.combined[key]
	if !ok {
		v[0], v[1] = combinedOffDefStat(b.run(override), b.misc.base)
		b.combined[key] = v
	}
	return v[0], v[1]
}

func (b *PowerBuilder) powerOf(env *Environment) float64 {
	return b.value(env) - b.base
}

func (b *PowerBuilder) value(env *Environment) float64 {
	if b.stat.Stat == "FullDPS" && !b.misc.usedFullDPS {
		return powerStatValue(env, PowerStat{Stat: "CombinedDPS"})
	}
	return powerStatValue(env, b.stat)
}

//pob:block:src/Classes/CalcsTab.lua:611-653
func (b *PowerBuilder) nodeTask(id int64, distance int, selections map[int64]int64) {
	n := b.treeNode(id)
	p := b.power[id]
	key := b.keys[id]
	ascendancy := n.AscendancyName != nil && *n.AscendancyName != ""
	if !b.alloc[id] {
		if b.stat.Combined {
			p.single, p.defence = b.calcCombined(key, &CalcOverride{AddNodes: []int64{id}})
			p.hasSingle = true
		} else {
			p.single, p.hasSingle = b.calc(key, &CalcOverride{AddNodes: []int64{id}}), true
		}
		if path := allocationPath(b.paths, id); path != nil && !ascendancy && !b.stat.Combined {
			p.path, p.hasPath = p.single, true
			if distance > 1 {
				p.path = b.powerOf(b.run(&CalcOverride{AddNodes: path}))
			}
		}
	} else if !b.stat.Combined {
		p.single, p.hasSingle = b.calc(key+"_remove", &CalcOverride{RemoveNodes: []int64{id}}), true
		if !ascendancy {
			p.path, p.hasPath = p.single, true
			if depends := b.depends(id); len(depends) > 1 {
				p.path = b.powerOf(b.run(&CalcOverride{RemoveNodes: depends}))
			}
		}
	}
	if isMasteryNode(n) {
		if effect, ok := selections[id]; ok {
			b.mastery[id] = map[int64]*nodePower{effect: {single: p.single, path: p.path, hasSingle: p.hasSingle, hasPath: p.hasPath}}
		}
	}
}

//pob:block:src/Classes/CalcsTab.lua:665-710
func (b *PowerBuilder) masteryTask(id int64, effect data.MasteryEffect) {
	build := b.misc.build
	spec := &build.Tree.Specs[build.ActiveSpecIndex()]
	saved := spec.MasteryEffects
	build.SetMasteryEffect(id, effect.Effect)
	defer func() { spec.MasteryEffects = saved }()

	n := b.treeNode(id)
	ep := &nodePower{}
	if b.mastery[id] == nil {
		b.mastery[id] = map[int64]*nodePower{}
	}
	b.mastery[id][effect.Effect] = ep
	p := b.power[id]
	if b.stat.Combined {
		ep.single, ep.defence = b.calcCombined("effect:"+strings.Join(effect.Stats, "\n"), &CalcOverride{AddNodes: []int64{id}})
		ep.hasSingle = true
		p.single, p.defence, p.hasSingle = max(p.single, ep.single), max(p.defence, ep.defence), true
		return
	}
	ep.single, ep.hasSingle = b.calc("effect:"+strings.Join(effect.Stats, "\n"), &CalcOverride{AddNodes: []int64{id}}), true
	path := allocationPath(b.paths, id)
	if path != nil && (n.AscendancyName == nil || *n.AscendancyName == "") {
		ep.path, ep.hasPath = ep.single, true
		if len(path) > 1 {
			ep.path = b.powerOf(b.run(&CalcOverride{AddNodes: path}))
		}
	}
	p.single, p.hasSingle = max(p.single, ep.single), true
	p.path, p.hasPath = max(p.path, ep.path), true
}

//pob:block:src/Classes/PassiveTree.lua:810-818
func splitStatLines(stats []string) []string {
	var out []string
	for _, s := range stats {
		for _, part := range strings.Split(s, "\n") {
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

func (b *PowerBuilder) Step(budget time.Duration) float64 {
	start := time.Now()
	for b.next < len(b.tasks) {
		b.tasks[b.next]()
		b.next++
		if time.Since(start) > budget {
			break
		}
	}
	return b.Progress()
}

func (b *PowerBuilder) Progress() float64 {
	if len(b.tasks) == 0 {
		return 1
	}
	return float64(b.next) / float64(len(b.tasks))
}

func (b *PowerBuilder) Current() int64 {
	if b.next < len(b.taskNodes) {
		return b.taskNodes[b.next]
	}
	return 0
}

//pob:block:src/Classes/TreeTab.lua:1099-1154
func (b *PowerBuilder) Report() []PowerReportEntry {
	selections := map[int64]int64{}
	for _, s := range activeMasterySelections(b.misc.build) {
		selections[s.Node] = s.Effect
	}
	var report []PowerReportEntry
	for _, id := range b.nodes {
		n := b.treeNode(id)
		p := b.power[id]
		isAlloc := b.alloc[id] || b.granted[id]
		if n.AscendancyName != nil && *n.AscendancyName != "" {
			continue
		}
		pathDist := func() int {
			if b.alloc[id] {
				return len(b.depends(id))
			}
			if isAlloc {
				return 1
			}
			if p.distance != 0 {
				return p.distance
			}
			return max(1, len(allocationPath(b.paths, id)))
		}
		name := ""
		if n.Name != nil {
			name = *n.Name
		}
		switch typ := powerNodeType(n); typ {
		case "Normal", "Keystone", "Notable":
			d := pathDist()
			name, stats := b.specNodeStats(id)
			report = append(report, PowerReportEntry{ID: id, Name: name, Type: typ, Power: p.single, PathPower: p.path / float64(d),
				PathDist: d, Allocated: isAlloc, Stats: stats, Offence: p.single, Defence: p.defence})
		case "Mastery":
			d := pathDist()
			first := len(report)
			picked, best := -1, 0.0
			for _, effect := range n.MasteryEffects {
				ep := b.mastery[id][effect.Effect]
				if ep == nil {
					continue
				}
				label := splitStatLines(effect.Stats)
				pathPower := ep.single
				if ep.hasPath {
					pathPower = ep.path
				}
				if !isAlloc && ep.single > best {
					picked, best = len(report), ep.single
				}
				report = append(report, PowerReportEntry{ID: id, Name: name + ": " + strings.Join(label, " / "), Type: typ,
					Power: ep.single, PathPower: pathPower / float64(d), PathDist: d, Allocated: isAlloc, Stats: label, Offence: ep.single, Defence: ep.defence,
					Effect: effect.Effect})
			}
			if picked >= first {
				report[picked].Picked = true
			}
		}
	}
	for _, id := range b.clusterNodes {
		n := b.treeNode(id)
		report = append(report, PowerReportEntry{ID: id, Name: *n.Name, Type: powerNodeType(n), Power: b.cluster[id], Stats: n.Stats, Cluster: true})
	}
	sort.SliceStable(report, func(i, j int) bool { return report[i].Power > report[j].Power })
	return report
}

func cloneInputs(inputs []pob.Input) []pob.Input {
	out := make([]pob.Input, len(inputs))
	for i, in := range inputs {
		out[i] = pob.Input{Name: in.Name, Boolean: clonePtr(in.Boolean), Number: clonePtr(in.Number), String: clonePtr(in.String)}
	}
	return out
}

func sameInputs(a, b []pob.Input) bool {
	return slices.EqualFunc(a, b, func(x, y pob.Input) bool {
		return x.Name == y.Name && samePtr(x.Boolean, y.Boolean) && samePtr(x.Number, y.Number) && samePtr(x.String, y.String)
	})
}

func samePtr[T comparable](x, y *T) bool {
	if x == nil || y == nil {
		return x == y
	}
	return *x == *y
}

type powerGraph struct {
	tree          *data.Tree
	adjacency     map[int64][]int64
	blocked       map[int64]bool
	masteryParent map[int64]bool
	subgraph      map[int64]bool
}

func newPowerGraph(tree *data.Tree, clusterNodes map[string]data.Node) *powerGraph {
	g := &powerGraph{tree: tree, adjacency: map[int64][]int64{}, blocked: map[int64]bool{}, masteryParent: map[int64]bool{},
		subgraph: map[int64]bool{}}
	link := func(from, to int64) {
		if !slices.Contains(g.adjacency[from], to) {
			g.adjacency[from] = append(g.adjacency[from], to)
		}
	}
	for _, n := range tree.Nodes {
		if n.Skill == nil {
			continue
		}
		if n.ClassStartIndex != nil || (n.IsAscendancyStart != nil && *n.IsAscendancyStart) {
			g.blocked[*n.Skill] = true
		}
		if n.IsMastery != nil && *n.IsMastery {
			g.masteryParent[*n.Skill] = true
		}
		for _, out := range n.Out {
			target, ok := tree.Nodes[out]
			id, err := strconv.ParseInt(out, 10, 64)
			if !ok || err != nil || target.ClassStartIndex != nil {
				continue
			}
			if (target.AscendancyName == nil) != (n.AscendancyName == nil) ||
				(target.AscendancyName != nil && *target.AscendancyName != *n.AscendancyName) {
				continue
			}
			link(*n.Skill, id)
			if target.IsMastery == nil || !*target.IsMastery {
				link(id, *n.Skill)
			}
		}
	}
	for key, n := range clusterNodes {
		id, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			continue
		}
		if _, ok := tree.Nodes[key]; !ok {
			g.subgraph[id] = true
		}
		for _, out := range n.Out {
			if other, err := strconv.ParseInt(out, 10, 64); err == nil {
				link(id, other)
				link(other, id)
			}
		}
	}
	return g
}

//pob:block:src/Classes/PassiveSpec.lua:54
func (g *powerGraph) inSpec(id int64) bool {
	if g.subgraph[id] {
		return true
	}
	n, ok := g.tree.Nodes[strconv.FormatInt(id, 10)]
	if !ok || n.Group == nil || (n.IsProxy != nil && *n.IsProxy) {
		return false
	}
	if group, ok := g.tree.Groups[strconv.FormatInt(*n.Group, 10)]; ok && group.IsProxy != nil && *group.IsProxy {
		return false
	}
	return n.ExpansionJewel == nil || n.ExpansionJewel.Parent == nil
}

type pathFrontier struct {
	ids       []int64
	distances map[int64]int64
}

func (f *pathFrontier) Len() int { return len(f.ids) }
func (f *pathFrontier) Less(i, j int) bool {
	if di, dj := f.distances[f.ids[i]], f.distances[f.ids[j]]; di != dj {
		return di < dj
	}
	return f.ids[i] < f.ids[j]
}
func (f *pathFrontier) Swap(i, j int) { f.ids[i], f.ids[j] = f.ids[j], f.ids[i] }
func (f *pathFrontier) Push(x any)    { f.ids = append(f.ids, x.(int64)) }
func (f *pathFrontier) Pop() any {
	x := f.ids[len(f.ids)-1]
	f.ids = f.ids[:len(f.ids)-1]
	return x
}

//pob:src/Classes/PassiveSpec.lua:548-585
func (g *powerGraph) allocationPaths(activeNodes []int64, rootNodes []int64) map[int64]int64 {
	hops := map[int64]int64{}
	frontier := &pathFrontier{distances: map[int64]int64{}}
	distances := frontier.distances
	for _, id := range activeNodes {
		hops[id] = -1
		distances[id] = 0
		frontier.ids = append(frontier.ids, id)
	}
	for _, root := range rootNodes {
		if _, active := distances[root]; active {
			continue
		}
		hops[root] = -1
		if n, ok := g.tree.Nodes[strconv.FormatInt(root, 10)]; ok {
			for _, other := range slices.Concat(n.Out, n.In) {
				start, ok := g.tree.Nodes[other]
				if id, err := strconv.ParseInt(other, 10, 64); ok && err == nil && start.ClassStartIndex != nil {
					hops[root] = id
					hops[id] = -1
					break
				}
			}
		}
		distances[root] = 1
		frontier.ids = append(frontier.ids, root)
	}
	heap.Init(frontier)
	for frontier.Len() > 0 {
		current := heap.Pop(frontier).(int64)
		if g.masteryParent[current] {
			continue
		}
		for _, next := range g.adjacency[current] {
			if g.blocked[next] || !g.inSpec(next) {
				continue
			}
			if _, seen := distances[next]; seen {
				continue
			}
			distances[next] = distances[current] + 1
			hops[next] = current
			heap.Push(frontier, next)
		}
	}
	return hops
}

//pob:block:src/Classes/PassiveSpec.lua:1472-1529
func (g *powerGraph) dependentNodes(activeNodes []int64, target int64, rootNodes []int64) []int64 {
	rootsFor := func(list []int64) []int64 {
		out := slices.Clone(rootNodes)
		for _, id := range list {
			if n, ok := g.tree.Nodes[strconv.FormatInt(id, 10)]; ok && n.IsAscendancyStart != nil && *n.IsAscendancyStart {
				out = append(out, id)
			}
		}
		return out
	}
	rest := slices.DeleteFunc(slices.Clone(activeNodes), func(id int64) bool { return id == target })
	before := map[int64]bool{}
	for _, id := range g.prunable(activeNodes, rootsFor(activeNodes)) {
		before[id] = true
	}
	var out []int64
	for _, id := range g.prunable(rest, rootsFor(rest)) {
		if !before[id] {
			out = append(out, id)
		}
	}
	return out
}

//pob:src/Classes/PassiveSpec.lua:868-899
func (g *powerGraph) prunable(activeNodes []int64, rootNodes []int64) []int64 {
	active := map[int64]bool{}
	for _, id := range activeNodes {
		active[id] = true
	}
	var stack []int64
	for _, root := range rootNodes {
		if active[root] {
			stack = append(stack, root)
		}
	}
	visited := map[int64]bool{}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if visited[current] {
			continue
		}
		visited[current] = true
		if g.masteryParent[current] {
			continue
		}
		for _, next := range g.adjacency[current] {
			if active[next] && !visited[next] {
				stack = append(stack, next)
			}
		}
	}
	var out []int64
	for _, id := range activeNodes {
		if !visited[id] {
			out = append(out, id)
		}
	}
	return out
}
