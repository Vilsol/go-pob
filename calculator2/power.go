package calculator2

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data/tree2"
	"github.com/Vilsol/go-pob/mod"
)

// PowerStat is a data.powerStatList entry. Negate is its transform (the "Taken ... dmg" stats).
type PowerStat struct {
	Stat           string
	Label          string
	CombinedOffDef bool
	IgnoreForNodes bool
	IgnoreForItems bool
	ReverseSort    bool
	ItemField      string
	Negate         bool
}

// PowerStatList is data.powerStatList; the Name entry's transform strips a leading "The ".
//
//pob2:block:src/Modules/Data.lua:131-184
var PowerStatList = []PowerStat{
	{Label: "Offence/Defence", CombinedOffDef: true, IgnoreForItems: true},
	{Label: "Name", ItemField: "Name", IgnoreForNodes: true, ReverseSort: true},
	{Stat: "FullDPS", Label: "Full DPS"},
	{Stat: "CombinedDPS", Label: "Combined DPS"},
	{Stat: "TotalDPS", Label: "Hit DPS"},
	{Stat: "WithImpaleDPS", Label: "Impale + Hit DPS"},
	{Stat: "AverageDamage", Label: "Average Hit"},
	{Stat: "Speed", Label: "Attack/Cast Speed"},
	{Stat: "TotalDot", Label: "DoT DPS"},
	{Stat: "BleedDPS", Label: "Bleed DPS"},
	{Stat: "IgniteDPS", Label: "Ignite DPS"},
	{Stat: "PoisonDPS", Label: "Poison DPS"},
	{Stat: "Life", Label: "Life"},
	{Stat: "LifeRegen", Label: "Life regen"},
	{Stat: "LifeLeechRate", Label: "Life leech"},
	{Stat: "Armour", Label: "Armour"},
	{Stat: "Evasion", Label: "Evasion"},
	{Stat: "EnergyShield", Label: "Energy Shield"},
	{Stat: "EnergyShieldRecoveryCap", Label: "Recoverable ES"},
	{Stat: "EnergyShieldRegen", Label: "Energy Shield regen"},
	{Stat: "EnergyShieldLeechRate", Label: "Energy Shield leech"},
	{Stat: "Mana", Label: "Mana"},
	{Stat: "ManaRegen", Label: "Mana regen"},
	{Stat: "ManaLeechRate", Label: "Mana leech"},
	{Stat: "Ward", Label: "Runic Ward"},
	{Stat: "Spirit", Label: "Spirit"},
	{Stat: "Str", Label: "Strength"},
	{Stat: "Dex", Label: "Dexterity"},
	{Stat: "Int", Label: "Intelligence"},
	{Stat: "TotalAttr", Label: "Total Attributes"},
	{Stat: "MeleeAvoidChance", Label: "Melee avoid chance"},
	{Stat: "SpellAvoidChance", Label: "Spell avoid chance"},
	{Stat: "ProjectileAvoidChance", Label: "Projectile avoid chance"},
	{Stat: "TotalEHP", Label: "Effective Hit Pool"},
	{Stat: "SecondMinimalMaximumHitTaken", Label: "Eff. Maximum Hit Taken"},
	{Stat: "PhysicalTakenHit", Label: "Taken Phys dmg", Negate: true},
	{Stat: "LightningTakenHit", Label: "Taken Lightning dmg", Negate: true},
	{Stat: "ColdTakenHit", Label: "Taken Cold dmg", Negate: true},
	{Stat: "FireTakenHit", Label: "Taken Fire dmg", Negate: true},
	{Stat: "ChaosTakenHit", Label: "Taken Chaos dmg", Negate: true},
	{Stat: "CritChance", Label: "Crit Chance"},
	{Stat: "CritMultiplier", Label: "Crit Multiplier"},
	{Stat: "BleedChance", Label: "Bleed Chance"},
	{Stat: "FreezeChance", Label: "Freeze Chance"},
	{Stat: "IgniteChance", Label: "Ignite Chance"},
	{Stat: "ShockChance", Label: "Shock Chance"},
	{Stat: "EffectiveMovementSpeedMod", Label: "Move speed"},
	{Stat: "LightRadiusMod", Label: "Light Radius"},
	{Stat: "BlockChance", Label: "Block Chance"},
	{Stat: "SpellBlockChance", Label: "Spell Block Chance"},
	{Stat: "SpellSuppressionChance", Label: "Spell Suppression Chance"},
	{Stat: "EffectiveLootRarityMod", Label: "Rarity of Items found"},
}

// PowerOutput is a calculation's output table and its minion's, as calcFunc returns it.
type PowerOutput struct {
	Output map[string]float64
	Minion map[string]float64
}

func powerOutput(env *Environment) PowerOutput {
	out := PowerOutput{Output: env.Player.Output}
	if main := env.Player.MainSkill; main != nil && main.Minion != nil {
		out.Minion = main.Minion.Output
	}
	return out
}

// GetFromOutput is data.powerStatList.GetFromOutput; output.FullDPS is never set, as calcs.calcFullDPS is not ported.
//
//pob2:block:src/Modules/Data.lua:190-216
func (s *PowerStat) GetFromOutput(o PowerOutput) float64 {
	v := s.entry(o)
	if s.Negate {
		return -v
	}
	return v
}

func (s *PowerStat) entry(o PowerOutput) float64 {
	if s.Stat == "FullDPS" {
		return o.Output["CombinedDPS"] + o.Minion["CombinedDPS"]
	}
	if minionStat, ok := strings.CutPrefix(s.Stat, "Minion"); ok && minionStat != "" {
		return o.Minion[minionStat]
	}
	if s.Stat == "AverageDamage" || s.Stat == "TotalDot" || strings.Contains(s.Stat, "DPS") {
		return o.Output[s.Stat] + o.Minion[s.Stat]
	}
	return o.Output[s.Stat]
}

//pob2:block:src/Classes/CalcsTab.lua:711-715
func calculatePowerStat(selection *PowerStat, original, modified PowerOutput) float64 {
	return selection.GetFromOutput(original) - selection.GetFromOutput(modified)
}

// orNil is output.X where a nil output reads as the fallback field.
func orNil(o map[string]float64, key, fallback string) float64 {
	if v, ok := o[key]; ok {
		return v
	}
	return o[fallback]
}

//pob2:block:src/Classes/CalcsTab.lua:717-727
func calculateCombinedOffDefStat(original, modified PowerOutput) (float64, float64) {
	o, m := original.Output, modified.Output
	esO, esM := orNil(o, "EnergyShieldRecoveryCap", "EnergyShield"), orNil(m, "EnergyShieldRecoveryCap", "EnergyShield")
	defence := (o["LifeUnreserved"]-m["LifeUnreserved"])/math.Max(3000, m["Life"]) +
		(o["Armour"]-m["Armour"])/math.Max(10000, m["Armour"]) +
		(esO-esM)/math.Max(3000, esM) +
		(o["Evasion"]-m["Evasion"])/math.Max(10000, m["Evasion"]) +
		(o["LifeRegenRecovery"]-m["LifeRegenRecovery"])/500 +
		(o["EnergyShieldRegenRecovery"]-m["EnergyShieldRegenRecovery"])/1000
	modifiedDps := m["CombinedDPS"] + modified.Minion["CombinedDPS"]
	dpsIncr := o["CombinedDPS"] + original.Minion["CombinedDPS"] - modifiedDps
	return dpsIncr / modifiedDps, defence
}

// NodePower is node.power: what allocating (or, for an allocated node, removing) the node and its path is worth.
// The Has flags tell a field PoB leaves nil from a zero.
type NodePower struct {
	Distance              int
	SingleStat, PathPower float64
	Offence, Defence      float64
	HasSingle, HasPath    bool
	HasOffDef             bool
}

// PowerMax is calcsTab.powerMax.
type PowerMax struct {
	SingleStat, Offence, OffencePerPoint, Defence, DefencePerPoint float64
}

// PowerBuilder is CalcsTab:PowerBuilder run in steps: Step calculates the next nodes, nearest first.
type PowerBuilder struct {
	m        *MiscCalculator
	stat     *PowerStat
	maxDepth int
	base     PowerOutput
	granted  map[int]bool
	cache    map[string]PowerOutput
	modKeys  map[int]string
	tasks    []powerTask
	next     int

	// Power is each node's power, by id; Max is the largest over the reachable non-ascendancy nodes.
	Power map[int]*NodePower
	Max   PowerMax
	// Total is the number of nodes within the depth, the progress denominator.
	Total int
}

type powerTask struct {
	node     *tree2.SpecNode
	distance int
}

// nodeModKey is node.modKey: one entry per mod the node's fully parsed lines give, "" for a node with none.
//
//pob2:block:src/Classes/PassiveTree.lua:519-521
func (b *PowerBuilder) nodeModKey(node *tree2.SpecNode) string {
	if k, ok := b.modKeys[node.ID]; ok {
		return k
	}
	var sb strings.Builder
	for _, m := range b.m.main.nodeModList(node).Mods() {
		sb.WriteString("[")
		sb.WriteString(mod.FormatValue(m.Value()))
		sb.WriteString(" = ")
		sb.WriteString(m.Name())
		sb.WriteString(",")
		sb.WriteString(string(m.Type()))
		sb.WriteString(",")
		sb.WriteString(strconv.FormatInt(int64(m.Flags()), 10))
		sb.WriteString(",")
		sb.WriteString(strconv.FormatInt(int64(m.KeywordFlags()), 10))
		for _, t := range m.Tags() {
			sb.WriteString(",")
			sb.WriteString(mod.FormatValue(t))
		}
		sb.WriteString("]")
	}
	b.modKeys[node.ID] = sb.String()
	return b.modKeys[node.ID]
}

// NewPowerBuilder is the first part of CalcsTab:PowerBuilder: every node's distance and the nodes to calculate.
// stat is nil or the Offence/Defence entry for the combined power; maxDepth 0 is no limit.
//
//pob2:block:src/Classes/CalcsTab.lua:551-616
func NewPowerBuilder(m *MiscCalculator, stat *PowerStat, maxDepth int) *PowerBuilder {
	b := &PowerBuilder{
		m: m, stat: stat, maxDepth: maxDepth, base: powerOutput(m.Base), granted: m.main.GrantedPassives,
		cache: map[string]PowerOutput{}, modKeys: map[int]string{}, Power: map[int]*NodePower{},
	}
	spec := m.main.Spec
	distanceMap := map[int][]*tree2.SpecNode{}
	// PoB walks spec.nodes with pairs(); ids are walked sorted here.
	for _, nodeID := range sortedNodeIDs(spec.Nodes) {
		node := spec.Nodes[nodeID]
		if b.nodeModKey(node) == "" || b.granted[nodeID] {
			continue
		}
		hiddenByLockedAscendancyNode := false
		if node.UnlockConstraint != nil {
			for _, unlockNodeID := range node.UnlockConstraint.Nodes {
				if unlockNode := spec.Nodes[unlockNodeID]; unlockNode != nil && unlockNode.AscendancyName != "" && !unlockNode.Alloc {
					hiddenByLockedAscendancyNode = true
					break
				}
			}
		}
		if hiddenByLockedAscendancyNode {
			continue
		}
		dist := node.PathDist
		for _, leap := range node.IntuitiveLeapLikesAffecting {
			if leap.Alloc {
				dist = max(min(leap.PathDist, dist), 1)
			}
		}
		distanceMap[dist] = append(distanceMap[dist], node)
		b.Power[nodeID] = &NodePower{Distance: dist}
		if maxDepth == 0 || dist <= maxDepth {
			b.Total++
		}
	}
	distances := make([]int, 0, len(distanceMap))
	for d := range distanceMap {
		distances = append(distances, d)
	}
	sort.Ints(distances)
	for _, d := range distances {
		if maxDepth != 0 && maxDepth < d {
			break
		}
		for _, node := range distanceMap[d] {
			b.tasks = append(b.tasks, powerTask{node: node, distance: d})
		}
	}
	return b
}

func (b *PowerBuilder) calc(key string, o NodeOverride) PowerOutput {
	if out, ok := b.cache[key]; ok {
		return out
	}
	out := powerOutput(b.m.CalcNodes(o))
	b.cache[key] = out
	return out
}

func (b *PowerBuilder) singleStat() bool {
	return b.stat != nil && b.stat.Stat != "" && !b.stat.IgnoreForNodes
}

// Step calculates up to n nodes and reports whether every node is done.
//
//pob2:block:src/Classes/CalcsTab.lua:617-680
func (b *PowerBuilder) Step(n int) bool {
	for ; n > 0 && b.next < len(b.tasks); n-- {
		t := b.tasks[b.next]
		b.next++
		node, distance := t.node, t.distance
		power := b.Power[node.ID]
		modKey := b.nodeModKey(node)
		switch {
		case !node.Alloc:
			output := b.calc(modKey, NodeOverride{AddNodes: map[*tree2.SpecNode]bool{node: true}})
			if b.singleStat() {
				power.SingleStat, power.HasSingle = calculatePowerStat(b.stat, output, b.base), true
				if node.HasPath && node.AscendancyName == "" {
					b.Max.SingleStat = math.Max(b.Max.SingleStat, power.SingleStat)
					power.PathPower, power.HasPath = power.SingleStat, true
					if distance > 1 {
						pathNodes := make(map[*tree2.SpecNode]bool, len(node.Path))
						for _, p := range node.Path {
							pathNodes[p] = true
						}
						power.PathPower = calculatePowerStat(b.stat, powerOutput(b.m.CalcNodes(NodeOverride{AddNodes: pathNodes})), b.base)
					}
				}
			} else if b.stat == nil || !b.stat.IgnoreForNodes {
				power.Offence, power.Defence = calculateCombinedOffDefStat(output, b.base)
				power.HasOffDef = true
				power.SingleStat, power.HasSingle = power.Offence, true
				if node.HasPath && node.AscendancyName == "" {
					b.Max.Offence = math.Max(b.Max.Offence, power.Offence)
					b.Max.Defence = math.Max(b.Max.Defence, power.Defence)
					b.Max.OffencePerPoint = math.Max(b.Max.OffencePerPoint, power.Offence/float64(distance))
					b.Max.DefencePerPoint = math.Max(b.Max.DefencePerPoint, power.Defence/float64(distance))
				}
			}
		default:
			output := b.calc(modKey+"_remove", NodeOverride{RemoveNodes: map[*tree2.SpecNode]bool{node: true}})
			if b.singleStat() {
				power.SingleStat, power.HasSingle = calculatePowerStat(b.stat, output, b.base), true
				if node.AscendancyName == "" {
					power.PathPower, power.HasPath = power.SingleStat, true
					if len(node.Depends) > 1 {
						pathNodes := make(map[*tree2.SpecNode]bool, len(node.Depends))
						for _, p := range node.Depends {
							pathNodes[p] = true
						}
						power.PathPower = calculatePowerStat(b.stat, powerOutput(b.m.CalcNodes(NodeOverride{RemoveNodes: pathNodes})), b.base)
					}
				}
			}
		}
	}
	return b.next >= len(b.tasks)
}

// Done is the number of nodes calculated so far.
func (b *PowerBuilder) Done() int { return b.next }

// Tasks is the number of nodes the builder calculates.
func (b *PowerBuilder) Tasks() int { return len(b.tasks) }

// PowerReportRow is one TreeTab:BuildPowerReportList row before its display stat's scale and format: Power is the
// node's power and PathPower its path's divided by PathDist.
type PowerReportRow struct {
	ID        int
	Name      string
	Type      string
	Allocated bool
	Power     float64
	PathPower float64
	PathDist  int
}

// PowerReportList is the power report's rows for the stat the builder ran, unsorted; nil for the combined power.
// Cluster notables are left out: the PoE2 tree has none.
//
//pob2:block:src/Classes/TreeTab.lua:932-1005
func (b *PowerBuilder) PowerReportList() []PowerReportRow {
	if b.stat == nil || b.stat.Stat == "" {
		return nil
	}
	spec := b.m.main.Spec
	var report []PowerReportRow
	for _, nodeID := range sortedNodeIDs(spec.Nodes) {
		node := spec.Nodes[nodeID]
		isAlloc := node.Alloc || b.granted[nodeID]
		if (node.Type != "Normal" && node.Type != "Keystone" && node.Type != "Notable") || node.AscendancyName != "" {
			continue
		}
		power, ok := b.Power[nodeID]
		if !ok {
			power = &NodePower{}
		}
		var pathDist int
		switch {
		case isAlloc:
			pathDist = max(len(node.Depends), 1)
		case ok:
			pathDist = power.Distance
		default:
			pathDist = max(len(node.Path), 1)
		}
		report = append(report, PowerReportRow{
			ID: nodeID, Name: node.Dn, Type: node.Type, Allocated: isAlloc,
			Power: power.SingleStat, PathPower: power.PathPower / float64(pathDist), PathDist: pathDist,
		})
	}
	return report
}
