package calculator

import (
	"math"
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

//pob:block:src/Modules/Data.lua:125-172
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

//pob:block:src/Modules/Data.lua:207-228,src/Classes/TreeTab.lua:260-265
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

//pob:block:src/Modules/Data.lua:123
func NodePowerStats() []PowerStat {
	return append([]PowerStat{{Label: "Offence/Defence", Combined: true}}, PowerStats()...)
}

//pob:src/Classes/CalcsTab.lua:741-751
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

//pob:src/Modules/Data.lua:179-205
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

	tree    *data.Tree
	alloc   map[int64]bool
	granted map[int64]bool
	paths   map[int64]int64
	roots   []int64

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
	taskNodes []int64
}

func (b *PowerBuilder) treeNode(id int64) data.Node {
	return b.tree.Nodes[strconv.FormatInt(id, 10)]
}

//pob:block:src/Classes/PassiveTree.lua:521-552
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

//pob:block:src/Classes/PassiveSpec.lua:44
func specNode(tree *data.Tree, n data.Node) bool {
	if n.Group == nil || (n.IsProxy != nil && *n.IsProxy) || (n.ExpansionJewel != nil && n.ExpansionJewel.Parent != nil) {
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

func (b *PowerBuilder) depends(id int64) []int64 {
	return append([]int64{id}, data.TreeVersions[data.LatestTreeVersion].CalculateDependentNodes(b.misc.build.Build.PassiveNodes, id, b.roots)...)
}

//pob:block:src/Classes/CalcsTab.lua:489-597
func (m *MiscCalculator) NewPowerBuilder(stat PowerStat, maxDepth int) *PowerBuilder {
	b := &PowerBuilder{
		misc: m, stat: stat, maxDepth: maxDepth, base: powerStatValue(m.base, stat),
		tree:     data.TreeVersions[data.LatestTreeVersion].Tree(),
		alloc:    map[int64]bool{},
		granted:  map[int64]bool{},
		keys:     map[int64]string{},
		power:    map[int64]*nodePower{},
		mastery:  map[int64]map[int64]*nodePower{},
		cluster:  map[int64]float64{},
		cache:    map[string]float64{},
		combined: map[string][2]float64{},
	}
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
	b.paths = data.TreeVersions[data.LatestTreeVersion].CalculateAllocationPaths(m.build.Build.PassiveNodes, b.roots)
	selections := map[int64]int64{}
	for _, s := range activeMasterySelections(m.build) {
		selections[s.Node] = s.Effect
	}

	distanceMap := map[int][]int64{}
	var masteryNodes []int64
	for key, n := range b.tree.Nodes {
		if !specNode(b.tree, n) {
			continue
		}
		id, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			continue
		}
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
		if key := statsModKey(b.treeNode(id).Stats); key != "" && !b.granted[id] {
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

//pob:block:src/Classes/PassiveTree.lua:814-824
func statsModKey(stats []string) string {
	ml := moddb.NewModList()
	processStats(&Environment{}, ml, "", "", stats)
	if len(ml.Mods()) == 0 {
		return ""
	}
	return strings.Join(stats, "\n")
}

func (b *PowerBuilder) modKey(id int64, selections map[int64]int64) string {
	n := b.treeNode(id)
	if !isMasteryNode(n) {
		return statsModKey(n.Stats)
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

//pob:block:src/Modules/Calcs.lua:135-145
func (b *PowerBuilder) run(override *CalcOverride) *Environment {
	if b.stat.Stat == "FullDPS" {
		return b.misc.Calc(override)
	}
	var env *Environment
	if b.spec == nil || !sameInputs(b.specInputs, b.misc.build.Config.Inputs) {
		var player, enemy, minion moddb.ModStoreFuncs
		inputs := cloneInputs(b.misc.build.Config.Inputs)
		env, player, enemy, minion = initEnvSpec(b.misc.build, b.misc.cache, OutputModeCalculator, false, override, nil)
		b.spec = &calcSpecEnv{cachedPlayerDB: player, cachedEnemyDB: enemy, cachedMinionDB: minion, flatDB: true,
			clusterNodes: env.clusterNodes, clusterNodesSet: true}
		b.specInputs = inputs
	} else {
		env, _, _, _ = initEnvSpecKeep(b.misc.build, b.misc.cache, OutputModeCalculator, false, override, b.spec, false)
	}
	PerformCalc(env)
	return env
}

//pob:block:src/Classes/CalcsTab.lua:606-609
func (b *PowerBuilder) calc(key string, override *CalcOverride) float64 {
	if b.stat.Combined {
		v, _ := b.calcCombined(key, override)
		return v
	}
	v, ok := b.cache[key]
	if !ok {
		v = powerStatValue(b.run(override), b.stat)
		b.cache[key] = v
	}
	return v - b.base
}

//pob:block:src/Classes/CalcsTab.lua:540-549
func (b *PowerBuilder) calcCombined(key string, override *CalcOverride) (float64, float64) {
	v, ok := b.combined[key]
	if !ok {
		v[0], v[1] = combinedOffDefStat(b.run(override), b.misc.base)
		b.combined[key] = v
	}
	return v[0], v[1]
}

func (b *PowerBuilder) powerOf(env *Environment) float64 {
	return powerStatValue(env, b.stat) - b.base
}

//pob:block:src/Classes/CalcsTab.lua:604-646
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
	} else {
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

//pob:block:src/Classes/CalcsTab.lua:658-703
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
	ep.single, ep.hasSingle = b.calc("effect:"+strings.Join(effect.Stats, "\n"), &CalcOverride{AddNodes: []int64{id}}), true
	path := allocationPath(b.paths, id)
	if path != nil && (n.AscendancyName == nil || *n.AscendancyName == "") {
		ep.path, ep.hasPath = ep.single, true
		if len(path) > 1 {
			ep.path = b.powerOf(b.run(&CalcOverride{AddNodes: path}))
		}
	}
	p := b.power[id]
	p.single, p.hasSingle = max(p.single, ep.single), true
	p.path, p.hasPath = max(p.path, ep.path), true
}

//pob:block:src/Classes/PassiveTree.lua:784-792
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

//pob:block:src/Classes/TreeTab.lua:1090-1145
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
			report = append(report, PowerReportEntry{ID: id, Name: name, Type: typ, Power: p.single, PathPower: p.path / float64(d),
				PathDist: d, Allocated: isAlloc, Stats: n.Stats, Offence: p.single, Defence: p.defence})
		case "Mastery":
			d := pathDist()
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
				report = append(report, PowerReportEntry{ID: id, Name: name + ": " + strings.Join(label, " / "), Type: typ,
					Power: ep.single, PathPower: pathPower / float64(d), PathDist: d, Allocated: isAlloc, Stats: label})
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
