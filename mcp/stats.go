package mcp

import (
	_ "embed"
	"encoding/json"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/Vilsol/go-pob/calculator"
)

// display_stats.json and minion_display_stats.json are BuildDisplayStats.lua's lists, written by
// oracle/display_stats.lua from the pinned PoB (mise run mcp-data).
//
//go:embed display_stats.json
var displayStatsJSON []byte

//go:embed minion_display_stats.json
var minionDisplayStatsJSON []byte

type displayStat struct {
	Stat     string   `json:"stat"`
	Label    string   `json:"label"`
	ModNames []string `json:"modNames"`
}

type statInfo struct {
	Key      string
	Label    string
	Group    string
	ModNames []string
}

const mainDPS = "MainDPS"

var summaryKeys = []string{
	mainDPS, "FullDPS", "Speed", "CritChance", "HitChance", "Life", "EnergyShield", "Mana", "LifeUnreserved",
	"ManaUnreserved", "TotalEHP", "Armour", "Evasion", "FireResist", "ColdResist", "LightningResist", "ChaosResist",
	"BlockChance", "SpellSuppressionChance", "Str", "Dex", "Int",
}

var statGroups = map[string][]string{
	"offence": {mainDPS, "FullDPS", "CombinedDPS", "TotalDPS", "AverageHit", "AverageDamage", "Speed", "HitChance",
		"CritChance", "CritMultiplier", "TotalDot", "BleedDPS", "IgniteDPS", "PoisonDPS", "WithImpaleDPS",
		"ImpaleDPS", "CullingDPS", "ManaCost", "LifeCost", "ESCost", "ManaPerSecondCost", "AreaOfEffectRadius"},
	"defence": {"Life", "LifeUnreserved", "EnergyShield", "Mana", "ManaUnreserved", "Ward", "TotalEHP",
		"PhysicalMaximumHitTaken", "FireMaximumHitTaken", "ColdMaximumHitTaken", "LightningMaximumHitTaken",
		"ChaosMaximumHitTaken", "Armour", "PhysicalDamageReduction", "Evasion", "MeleeEvadeChance", "BlockChance",
		"SpellBlockChance", "SpellSuppressionChance", "AttackDodgeChance", "SpellDodgeChance", "MeleeAvoidChance",
		"SpellAvoidChance", "ProjectileAvoidChance"},
	"resist": {"FireResist", "FireResistOverCap", "FireResistTotal", "ColdResist", "ColdResistOverCap",
		"ColdResistTotal", "LightningResist", "LightningResistOverCap", "LightningResistTotal", "ChaosResist",
		"ChaosResistOverCap", "ChaosResistTotal"},
	"attributes": {"Str", "Dex", "Int", "ReqStr", "ReqDex", "ReqInt"},
	"charges": {"PowerCharges", "PowerChargesMax", "FrenzyCharges", "FrenzyChargesMax", "EnduranceCharges",
		"EnduranceChargesMax"},
	"minion": {"MinionCombinedDPS", "MinionTotalDPS", "MinionSpeed", "MinionLife", "MinionEnergyShield",
		"MinionFireResist", "MinionColdResist", "MinionLightningResist", "MinionChaosResist"},
	"recovery": {"LifeRegenRecovery", "LifeLeechGainRate", "LifeRecoverable", "ManaRegenRecovery",
		"ManaLeechGainRate", "EnergyShieldRegenRecovery", "EnergyShieldLeechGainRate", "EnergyShieldRecoveryCap"},
}

var groupNames = []string{"offence", "defence", "resist", "attributes", "charges", "minion", "recovery"}

var statTable = sync.OnceValue(func() map[string]*statInfo {
	out := map[string]*statInfo{}
	add := func(key, label string, modNames []string) {
		if key == "" {
			return
		}
		if s, ok := out[key]; ok {
			if s.Label == "" {
				s.Label = label
			}
			if len(s.ModNames) == 0 {
				s.ModNames = modNames
			}
			return
		}
		out[key] = &statInfo{Key: key, Label: label, ModNames: modNames}
	}
	var player, minion []displayStat
	_ = json.Unmarshal(displayStatsJSON, &player)
	_ = json.Unmarshal(minionDisplayStatsJSON, &minion)
	for _, d := range player {
		add(d.Stat, d.Label, d.ModNames)
	}
	for _, d := range minion {
		add("Minion"+d.Stat, "Minion "+d.Label, d.ModNames)
	}
	for _, p := range calculator.PowerStats() {
		add(p.Stat, p.Label, nil)
	}
	add(mainDPS, "Main skill DPS (player + minion Combined DPS)", nil)
	for _, g := range groupNames {
		for _, k := range statGroups[g] {
			add(k, "", nil)
			if out[k].Group == "" {
				out[k].Group = g
			}
		}
	}
	for k, s := range out {
		if s.Label == "" {
			s.Label = k
		}
		if s.Group == "" && strings.HasPrefix(k, "Minion") {
			s.Group = "minion"
		}
	}
	return out
})

func statKeys() []string {
	keys := make([]string, 0, len(statTable()))
	for k := range statTable() {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// statValue reads one stat key from a calculated environment; ok is false when neither actor has it.
func statValue(env *calculator.Environment, key string) (float64, bool) {
	if env == nil || env.Player == nil {
		return 0, false
	}
	out := env.Player.Output
	var minion map[string]float64
	if env.Minion != nil {
		minion = env.Minion.Output
	}
	if key == mainDPS {
		return out["CombinedDPS"] + minion["CombinedDPS"], true
	}
	if v, ok := out[key]; ok {
		return v, true
	}
	if m, ok := strings.CutPrefix(key, "Minion"); ok && minion != nil {
		if v, ok := minion[m]; ok {
			return v, true
		}
	}
	return 0, false
}

func knownStat(env *calculator.Environment, key string) bool {
	if _, ok := statTable()[key]; ok {
		return true
	}
	_, ok := statValue(env, key)
	return ok
}

// round keeps integers for |v| >= 100 and one decimal below, per the output contract.
func round(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if math.Abs(v) >= 100 {
		return math.Round(v)
	}
	return math.Round(v*10) / 10
}

func pct(before, after float64) *float64 {
	if before == 0 {
		return nil
	}
	p := math.Round((after-before)/math.Abs(before)*1000) / 10
	return &p
}

type deltaRow struct {
	Stat   string   `json:"stat"`
	Before float64  `json:"before"`
	After  float64  `json:"after"`
	Pct    *float64 `json:"pct"`
}

func alwaysShown(key string) bool {
	return strings.HasSuffix(key, "Resist") || strings.Contains(key, "Charges") || key == "points"
}

// statDelta compares the watch set between two environments: rows that moved >= 0.5% (any move for resistances
// and charges), sorted by |pct| desc, capped at limit; must lists keys that are always included.
func statDelta(a, b *calculator.Environment, keys []string, must []string, limit int) ([]deltaRow, int) {
	var rows []deltaRow
	seen := map[string]bool{}
	mustSet := map[string]bool{}
	for _, k := range must {
		mustSet[k] = true
	}
	for _, k := range slices.Concat(must, keys) {
		if seen[k] {
			continue
		}
		seen[k] = true
		va, _ := statValue(a, k)
		vb, _ := statValue(b, k)
		ra, rb := round(va), round(vb)
		if !mustSet[k] {
			if ra == rb {
				continue
			}
			if !alwaysShown(k) && math.Abs(vb-va) < 0.005*math.Abs(va) {
				continue
			}
		}
		rows = append(rows, deltaRow{Stat: k, Before: ra, After: rb, Pct: pct(va, vb)})
	}
	mag := func(r deltaRow) float64 {
		if mustSet[r.Stat] {
			return math.Inf(1)
		}
		if r.Pct == nil {
			return 1e9
		}
		return math.Abs(*r.Pct)
	}
	sort.SliceStable(rows, func(i, j int) bool { return mag(rows[i]) > mag(rows[j]) })
	more := 0
	if len(rows) > limit {
		more = len(rows) - limit
		rows = rows[:limit]
	}
	return rows, more
}

func statsOf(env *calculator.Environment, keys []string) map[string]float64 {
	out := make(map[string]float64, len(keys))
	for _, k := range keys {
		if v, ok := statValue(env, k); ok {
			out[k] = round(v)
		}
	}
	return out
}
