package calculator

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	tjcalc "github.com/Vilsol/timeless-jewels/calculator"
	tjdata "github.com/Vilsol/timeless-jewels/data"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/moddb"
	"github.com/Vilsol/go-pob/pob"
)

type TimelessJewelType struct {
	Label string
	Name  string
	ID    int
}

//pob:block:src/Classes/TreeTab.lua:1187-1193
var TimelessJewelTypes = []TimelessJewelType{
	{Label: "Glorious Vanity", Name: "vaal", ID: 1},
	{Label: "Lethal Pride", Name: "karui", ID: 2},
	{Label: "Brutal Restraint", Name: "maraketh", ID: 3},
	{Label: "Militant Faith", Name: "templar", ID: 4},
	{Label: "Elegant Hubris", Name: "eternal", ID: 5},
	{Label: "Heroic Tragedy", Name: "kalguur", ID: 6},
}

//pob:block:src/Classes/TreeTab.lua:1211-1247
var TimelessConquerorTypes = map[int][]string{
	1: {"Any", "Doryani (Corrupted Soul)", "Xibaqua (Divine Flesh)", "Ahuana (Immortal Ambition)"},
	2: {"Any", "Kaom (Strength of Blood)", "Rakiata (Tempered by War)", "Akoya (Chainbreaker)"},
	3: {"Any", "Asenath (Dance with Death)", "Nasima (Second Sight)", "Balbala (The Traitor)"},
	4: {"Any", "Avarius (Power of Purpose)", "Dominus (Inner Conviction)", "Maxarius (Transcendence)"},
	5: {"Any", "Cadiro (Supreme Decadence)", "Victario (Supreme Grandstanding)", "Caspiro (Supreme Ostentation)"},
	6: {"Any", "Vorana (Black Scythe Training)", "Uhtred (Celestial Mathematics)", "Medved (The Unbreaking Circle)"},
}

//pob:block:src/Classes/TreeTab.lua:1265-1282
var TimelessDevotionVariants = []string{
	"Any", "Totem Damage", "Brand Damage", "Channelling Damage", "Area Damage", "Elemental Damage",
	"Elemental Resistances", "Effect of non-Damaging Ailments", "Elemental Ailment Duration", "Duration of Curses",
	"Minion Attack and Cast Speed", "Minions Accuracy Rating", "Mana Regen", "Skill Cost", "Non-Curse Aura Effect",
	"Defences from Shield",
}

//pob:block:src/Modules/Data.lua:900-925
var (
	timelessJewelSeedMin = map[int]int{1: 100, 2: 10000, 3: 500, 4: 2000, 5: 2000 / 20, 6: 100}
	timelessJewelSeedMax = map[int]int{1: 8000, 2: 18000, 3: 8000, 4: 10000, 5: 160000 / 20, 6: 8000}
)

//pob:block:src/Classes/TreeTab.lua:1169-1186
var (
	timelessIgnoredMods = []string{"Might of the Vaal", "Legacy of the Vaal", "Strength", "Add Strength", "Dex", "Add Dexterity", "Devotion", "Price of Glory", "Ward"}
	timelessTotalMods   = map[int]string{2: "Strength", 3: "Dexterity", 4: "Devotion"}
	timelessTotalModIDs = map[string]map[string]bool{
		"total_strength":  {"karui_notable_add_strength": true, "karui_attribute_strength": true, "karui_small_strength": true},
		"total_dexterity": {"maraketh_notable_add_dexterity": true, "maraketh_attribute_dex": true, "maraketh_small_dex": true},
		"total_devotion":  {"templar_notable_devotion": true, "templar_devotion_node": true, "templar_small_devotion": true},
	}
	timelessReverseTotalModIDs = map[string]bool{
		"karui_notable_add_strength": true, "karui_attribute_strength": true, "karui_small_strength": true,
		"maraketh_notable_add_dexterity": true, "maraketh_attribute_dex": true, "maraketh_small_dex": true,
		"templar_notable_devotion": true, "templar_devotion_node": true, "templar_small_devotion": true,
	}
)

type TimelessSocket struct {
	Label    string
	Keystone string
	ID       int64
}

//pob:block:src/Classes/TreeTab.lua:1295-1336
func TimelessSockets(build *pob.PathOfBuilding) []TimelessSocket {
	tree := specTree()
	alloc := allocSet(build, tree)
	var out []TimelessSocket
	for key, n := range tree.Nodes {
		if !specNode(tree, n) || n.IsJewelSocket == nil || !*n.IsJewelSocket || (n.Name != nil && *n.Name == "Charm Socket") {
			continue
		}
		id, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			continue
		}
		keystone := "Unknown"
		switch id {
		case 26725:
			keystone = "Marauder"
		case 54127:
			keystone = "Duelist"
		case 7960:
			keystone = "Templar/Witch"
		default:
			sx, sy, _ := tree.NodePosition(id)
			minDistance := math.Inf(1)
			for other := range tree.NodesInRadiusOf(id, 2) {
				o := tree.Nodes[strconv.FormatInt(other, 10)]
				if o.IsKeystone == nil || !*o.IsKeystone {
					continue
				}
				x, y, _ := tree.NodePosition(other)
				if d := math.Sqrt(math.Pow(x-sx, 2) + math.Pow(y-sy, 2)); d < minDistance {
					keystone = timelessName(o.Name)
					minDistance = d
				}
			}
		}
		label := keystone + ": " + key
		if alloc[id] {
			label = "# " + label
		}
		out = append(out, TimelessSocket{Label: label, Keystone: keystone, ID: id})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return append([]TimelessSocket{{Label: "All Sockets", Keystone: "Multi-Socket Search", ID: -1}}, out...)
}

func timelessName(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

type TimelessMod struct {
	Label        string
	ID           string
	Descriptions []string
	TotalMod     bool
}

//pob:block:src/Export/Scripts/legionPassives.lua:162-168
func timelessLegionFlags() map[string][2]bool {
	out := map[string][2]bool{}
	for _, s := range tjdata.AlternatePassiveSkills {
		out[legionNodeRowID(s)] = [2]bool{slices.Contains(s.PassiveType, 3), slices.Contains(s.PassiveType, 4)}
	}
	return out
}

//pob:block:src/Classes/TreeTab.lua:1349-1402
func TimelessMods(jewelType int) []TimelessMod {
	jt, ok := timelessJewelType(jewelType)
	if !ok || raw.EnsureTimelessSources() != nil {
		return nil
	}
	flags := timelessLegionFlags()
	prefix := jt.Name + "_"
	var mods, small []TimelessMod
	for id, node := range data.LegionNodes {
		f := flags[id]
		if !strings.HasPrefix(id, prefix) || len(id) == len(prefix) || strings.HasPrefix(id, "abyss_special_ascendancy_notable_") ||
			slices.Contains(timelessIgnoredMods, node.DN) || f[1] {
			continue
		}
		if f[0] {
			label := node.DN + "                                                " + firstOf(node.Stats)
			if len(node.Stats) > 1 {
				label += " " + node.Stats[1]
			}
			mods = append(mods, TimelessMod{Label: label, ID: id, Descriptions: slices.Clone(node.Stats)})
		} else {
			small = append(small, TimelessMod{Label: node.DN, ID: id, Descriptions: slices.Clone(node.Stats)})
		}
	}
	if jt.Name != "vaal" {
		for id, add := range data.LegionAdditions {
			if strings.HasPrefix(id, prefix) && len(id) > len(prefix) && !slices.Contains(timelessIgnoredMods, add.DN) {
				mods = append(mods, TimelessMod{Label: add.DN, ID: id, Descriptions: slices.Clone(add.Stats)})
			}
		}
	}
	byLabel := func(a, b TimelessMod) int {
		if c := strings.Compare(a.Label, b.Label); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	}
	slices.SortFunc(mods, byLabel)
	slices.SortFunc(small, byLabel)
	if total, ok := timelessTotalMods[jewelType]; ok {
		mods = append([]TimelessMod{{
			Label:        "Total " + total,
			Descriptions: []string{"This is a hybrid node containing all additions to " + total},
			ID:           "total_" + strings.ToLower(total),
			TotalMod:     true,
		}}, mods...)
	}
	mods = append([]TimelessMod{{Label: "..."}}, mods...)
	return append(mods, small...)
}

func firstOf(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func timelessJewelType(id int) (TimelessJewelType, bool) {
	for _, t := range TimelessJewelTypes {
		if t.ID == id {
			return t, true
		}
	}
	return TimelessJewelType{}, false
}

var timelessSearchSplit = regexp.MustCompile(`[^,\s]+`)

//pob:block:src/Classes/TreeTab.lua:1420-1444
func ParseTimelessSearchList(text string) [][]string {
	lines := strings.FieldsFunc(text, func(r rune) bool { return r == '\r' || r == '\n' })
	rows := make([][]string, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, timelessSearchSplit.FindAllString(line, -1))
	}
	return rows
}

type TimelessSearchQuery struct {
	JewelType             int
	ConquerorType         int
	SocketID              int64
	SocketFilter          bool
	SocketFilterDistance  int
	ProtectedNodes        []string
	SearchList            string
	SearchListFallback    string
	TotalMinimumWeight    float64
	HasTotalMinimumWeight bool
}

type TimelessDesiredNode struct {
	ID          string
	DisplayName string
	DesiredIdx  int
}

type TimelessNodeMatch struct {
	ID              string
	TargetNodeNames []string
	TotalWeight     float64
}

type TimelessSearchResult struct {
	Label       string
	Seed        int
	Total       float64
	SocketID    int64
	SocketLabel string
	Nodes       []TimelessNodeMatch
}

type TimelessSearchResults struct {
	Results []TimelessSearchResult
	Desired []TimelessDesiredNode
}

type timelessDesired struct {
	nodeWeight, nodeWeight2 float64
	displayName             string
	desiredIdx              int
}

type timelessResultNode struct {
	targetNodeNames []string
	totalWeight     float64
}

type timelessSocketSearch struct {
	resultNodes map[int]map[string]*timelessResultNode
	seedWeights map[int]float64
	desired     map[string]*timelessDesired
}

type TimelessSearcher struct {
	m       *MiscCalculator
	q       TimelessSearchQuery
	sockets []TimelessSocket
	next    int
	out     TimelessSearchResults
}

func (m *MiscCalculator) NewTimelessSearcher(q TimelessSearchQuery) *TimelessSearcher {
	if _, ok := timelessJewelType(q.JewelType); !ok || raw.EnsureTimelessSources() != nil {
		return nil
	}
	s := &TimelessSearcher{m: m, q: q}
	if q.SocketID == -1 {
		s.sockets = TimelessSockets(m.build)[1:]
	} else {
		s.sockets = []TimelessSocket{{ID: q.SocketID}}
	}
	return s
}

//pob:block:src/Classes/TreeTab.lua:2794-2859
func (s *TimelessSearcher) Step() float64 {
	if s.next < len(s.sockets) {
		socket := s.sockets[s.next]
		s.next++
		if r := s.m.timelessSearchSocket(s.q, socket.ID); r != nil {
			s.out.Desired = timelessDesiredList(r.desired)
			if s.q.SocketID == -1 {
				s.out.Results = append(s.out.Results, timelessFormatResults(s.q, r, &socket)...)
			} else {
				s.out.Results = timelessFormatResults(s.q, r, nil)
			}
		}
		if s.next == len(s.sockets) {
			if s.q.SocketID == -1 {
				sort.SliceStable(s.out.Results, func(i, j int) bool {
					a, b := s.out.Results[i], s.out.Results[j]
					if a.Total == b.Total {
						return a.SocketLabel < b.SocketLabel
					}
					return a.Total > b.Total
				})
			} else {
				sort.SliceStable(s.out.Results, func(i, j int) bool { return s.out.Results[i].Total > s.out.Results[j].Total })
			}
		}
	}
	if len(s.sockets) == 0 {
		return 1
	}
	return float64(s.next) / float64(len(s.sockets))
}

func (s *TimelessSearcher) Results() TimelessSearchResults {
	return s.out
}

func (m *MiscCalculator) TimelessSearch(q TimelessSearchQuery) TimelessSearchResults {
	s := m.NewTimelessSearcher(q)
	if s == nil {
		return TimelessSearchResults{}
	}
	for s.Step() < 1 {
	}
	return s.Results()
}

func timelessDesiredList(desired map[string]*timelessDesired) []TimelessDesiredNode {
	out := make([]TimelessDesiredNode, 0, len(desired))
	for id, d := range desired {
		out = append(out, TimelessDesiredNode{ID: id, DisplayName: d.displayName, DesiredIdx: d.desiredIdx})
	}
	slices.SortFunc(out, func(a, b TimelessDesiredNode) int { return a.DesiredIdx - b.DesiredIdx })
	return out
}

type timelessCell struct {
	s     string
	n     float64
	isNum bool
}

func (c timelessCell) present() bool {
	return c.isNum || c.s != ""
}

func (c timelessCell) num() (float64, bool) {
	if c.isNum {
		return c.n, true
	}
	return luaToNumber(c.s)
}

func timelessNumOr(c timelessCell, ok bool, fallback float64) float64 {
	if !ok {
		return fallback
	}
	if v, ok := c.num(); ok {
		return v
	}
	return fallback
}

func timelessRoll(node data.LegionNode, pos int, rolls tjdata.StatRolls) (float64, bool) {
	if pos >= len(node.Rolls) {
		return 0, false
	}
	idx := node.Rolls[pos].Index
	if idx < 1 || idx > len(rolls) {
		return 0, true
	}
	return float64(rolls[idx-1]), true
}

func timelessConqueror(jewelType tjdata.JewelType) tjdata.Conqueror {
	names := make([]string, 0, len(tjdata.TimelessJewelConquerors[jewelType]))
	for c := range tjdata.TimelessJewelConquerors[jewelType] {
		names = append(names, string(c))
	}
	slices.Sort(names)
	if len(names) == 0 {
		return ""
	}
	return tjdata.Conqueror(names[0])
}

//pob:block:src/Classes/TreeTab.lua:2419-2706
func (m *MiscCalculator) timelessSearchSocket(q TimelessSearchQuery, socketID int64) *timelessSocketSearch {
	tree := specTree()
	socket, ok := tree.Nodes[strconv.FormatInt(socketID, 10)]
	if !ok || socket.IsJewelSocket == nil || !*socket.IsJewelSocket {
		return nil
	}
	jewelTypeID := q.JewelType
	radiusSet := tree.NodesInRadiusOf(socketID, 2)
	radiusNodes := make([]int64, 0, len(radiusSet))
	for id := range radiusSet {
		radiusNodes = append(radiusNodes, id)
	}
	slices.Sort(radiusNodes)

	allocatedNodes := map[int64]bool{}
	unAllocatedNodesDistance := map[int64]int{}
	targetNodes := []int64{}
	attributeSmalls, otherSmalls := 0, 0
	desiredNodes := map[string]*timelessDesired{}
	type minimumWeight struct {
		reqNode string
		weight  float64
	}
	var minimumWeights []minimumWeight
	resultNodes := map[int]map[string]*timelessResultNode{}
	rootNodes := map[int64]bool{}
	desiredIdx := 0

	searchListTbl := ParseTimelessSearchList(q.SearchList)
	searchListFallbackTbl := ParseTimelessSearchList(q.SearchListFallback)
	found := map[string]bool{}
	var combined [][]string
	for _, row := range searchListTbl {
		found[firstOf(row)] = true
		combined = append(combined, slices.Clone(row))
	}
	for _, row := range searchListFallbackTbl {
		if !found[firstOf(row)] {
			combined = append(combined, slices.Clone(row))
		}
	}

	protectedNodesCount := len(q.ProtectedNodes)
	allocatedNodesInRadiusCount := 0
	if q.SocketID != -1 {
		allocatedNodesInRadiusCount = m.timelessAllocatedNotablesInRadius(q.SocketID)
	}

	for _, row := range combined {
		if len(row) <= 1 {
			continue
		}
		cells := make([]timelessCell, len(row))
		for i, s := range row {
			cells[i] = timelessCell{s: s}
		}
		at := func(i int) (timelessCell, bool) {
			if i < len(cells) && cells[i].present() {
				return cells[i], true
			}
			return timelessCell{}, false
		}
		set := func(i int, v float64) {
			for len(cells) <= i {
				cells = append(cells, timelessCell{})
			}
			cells[i] = timelessCell{n: v, isNum: true}
		}
		id := cells[0].s
		displayName := ""
		hasName := false
		singleStat := false
		if total, ok := timelessTotalMods[jewelTypeID]; ok && id == "total_"+strings.ToLower(total) {
			id = "totalStat"
			displayName, hasName = total, true
		}
		if !hasName {
			if node, ok := data.LegionNodes[id]; ok {
				if jewelTypeID > 1 && jewelTypeID < 7 {
					singleStat = true
				}
				displayName, hasName = strings.Join(node.Stats, " + "), true
			}
		}
		if !hasName {
			if add, ok := data.LegionAdditions[id]; ok {
				singleStat = jewelTypeID < 7
				displayName, hasName = strings.Join(add.Stats, " + "), true
			}
		}
		if !hasName {
			continue
		}
		for i := 0; i < len(cells) && cells[i].present(); i++ {
			val := cells[i]
			if singleStat && i == 1 {
				c1, ok1 := at(1)
				c2, ok2 := at(2)
				set(1, timelessNumOr(c1, ok1, timelessNumOr(c2, ok2, 1)))
			}
			if !val.isNum && val.s == "required" {
				v := 0.0
				if singleStat && i == 1 {
					v, _ = cells[1].num()
				}
				set(i, v)
				if c3, ok := at(3); !ok || timelessNumOr(c3, true, 0) < 0.001 {
					set(3, 0.001)
				}
			}
		}
		if c3, ok := at(3); ok {
			if w, _ := c3.num(); w > 0 {
				minimumWeights = append(minimumWeights, minimumWeight{reqNode: id, weight: w})
			}
		}
		if jewelTypeID == 4 && protectedNodesCount > 0 && protectedNodesCount < allocatedNodesInRadiusCount && len(combined) < allocatedNodesInRadiusCount {
			minimumWeights = append(minimumWeights, minimumWeight{reqNode: id, weight: 1})
		}
		c1, ok1 := at(1)
		c2, ok2 := at(2)
		d := &timelessDesired{
			nodeWeight:  timelessNumOr(c1, ok1, 0.001),
			nodeWeight2: timelessNumOr(c2, ok2, 0.001),
			displayName: displayName,
		}
		if prev, ok := desiredNodes[id]; ok {
			d.desiredIdx = prev.desiredIdx
		} else {
			desiredIdx++
			d.desiredIdx = desiredIdx
		}
		desiredNodes[id] = d
	}

	for key, n := range tree.Nodes {
		if n.ClassStartIndex != nil {
			if id, err := strconv.ParseInt(key, 10, 64); err == nil {
				rootNodes[id] = true
			}
		}
	}

	nodeOf := func(id int64) data.Node { return tree.Nodes[strconv.FormatInt(id, 10)] }
	isTrue := func(b *bool) bool { return b != nil && *b }

	if q.SocketFilter {
		alloc := allocSet(m.build, tree)
		var paths map[int64]int64
		if q.SocketFilterDistance > 0 {
			paths = m.timelessPaths(tree)
		}
		for _, id := range radiusNodes {
			_, granted := m.base.GrantedPassives[strconv.FormatInt(id, 10)]
			allocatedNodes[id] = granted || alloc[id]
			if q.SocketFilterDistance > 0 {
				unAllocatedNodesDistance[id] = 1000
				if alloc[id] {
					unAllocatedNodesDistance[id] = 0
				} else if p := allocationPath(paths, id); p != nil {
					unAllocatedNodesDistance[id] = len(p)
				}
			}
		}
	}

	for _, id := range radiusNodes {
		n := nodeOf(id)
		if rootNodes[id] || isTrue(n.IsJewelSocket) || isTrue(n.IsKeystone) {
			continue
		}
		if q.SocketFilter && !allocatedNodes[id] && !(q.SocketFilterDistance > 0 && unAllocatedNodesDistance[id] <= q.SocketFilterDistance) {
			continue
		}
		if isTrue(n.IsNotable) || jewelTypeID == 1 {
			targetNodes = append(targetNodes, id)
		} else if desiredNodes["totalStat"] != nil {
			if slices.Contains([]string{"Strength", "Intelligence", "Dexterity"}, timelessName(n.Name)) {
				attributeSmalls++
			} else {
				otherSmalls++
			}
		}
	}

	jewelType := tjdata.JewelType(jewelTypeID) //nolint:gosec // 1..6 checked by the caller
	conqueror := timelessConqueror(jewelType)
	seedWeights := map[int]float64{}
	seedMultiplier := 1
	if jewelTypeID == 5 {
		seedMultiplier = 20
	}
	skillKeys := make(map[int64]uint32, len(targetNodes))
	for _, id := range targetNodes {
		if s := raw.PassiveSkillByGraphID(id); s != nil && s.Key >= 0 {
			skillKeys[id] = uint32(s.Key) //nolint:gosec // guarded above
		}
	}
	protected := func(id int64) bool { return slices.Contains(q.ProtectedNodes, timelessName(nodeOf(id).Name)) }
	add := func(seed int, legionID string, targetNode int64, weight float64) {
		r := resultNodes[seed][legionID]
		if r == nil {
			r = &timelessResultNode{}
			resultNodes[seed][legionID] = r
		}
		r.totalWeight += float64(weight)
		r.targetNodeNames = append(r.targetNodeNames, timelessName(nodeOf(targetNode).Name))
		seedWeights[seed] += float64(weight)
	}

	for curSeed := timelessJewelSeedMin[jewelTypeID] * seedMultiplier; curSeed <= timelessJewelSeedMax[jewelTypeID]*seedMultiplier; curSeed += seedMultiplier {
		seedWeights[curSeed] = 0
		resultNodes[curSeed] = map[string]*timelessResultNode{}
		for _, targetNode := range targetNodes {
			key, ok := skillKeys[targetNode]
			if !ok {
				continue
			}
			info := tjcalc.Calculate(key, uint32(curSeed), jewelType, conqueror) //nolint:gosec // seeds are positive
			replaced := info.AlternatePassiveSkill != nil
			if !replaced && len(info.AlternatePassiveAdditionInformations) == 0 {
				continue
			}
			curNodeID := ""
			if jewelTypeID == 4 && protected(targetNode) {
				if replaced {
					delete(resultNodes, curSeed)
					break
				}
				if desiredNodes["totalStat"] == nil {
					desiredNodes["totalStat"] = &timelessDesired{nodeWeight: 0.1, displayName: "Devotion", desiredIdx: desiredIdx + 1}
				}
			}
			var curNode *data.LegionNode
			switch {
			case replaced && !protected(targetNode):
				if node, ok := data.LegionNodes[legionNodeRowID(info.AlternatePassiveSkill)]; ok {
					curNode, curNodeID = &node, node.ID
				}
			case !replaced:
				if a := info.AlternatePassiveAdditionInformations[0].AlternatePassiveAddition; a != nil {
					if node, ok := data.LegionAdditions[legionRowID(a.ID)]; ok {
						curNode, curNodeID = &node, node.ID
					}
				}
			default:
				curNodeID = ""
			}
			if desiredNodes["totalStat"] != nil && timelessReverseTotalModIDs[curNodeID] {
				curNodeID = "totalStat"
			}
			if jewelTypeID == 1 {
				if len(info.AlternatePassiveAdditionInformations) == 0 {
					if d := desiredNodes[curNodeID]; d != nil && curNode != nil {
						r1, _ := timelessRoll(*curNode, 0, info.StatRolls)
						weight := float64(d.nodeWeight * r1)
						if r2, ok := timelessRoll(*curNode, 1, info.StatRolls); ok {
							weight += float64(d.nodeWeight2 * r2)
						}
						add(curSeed, curNodeID, targetNode, weight)
					}
				} else {
					for _, a := range info.AlternatePassiveAdditionInformations {
						if a.AlternatePassiveAddition == nil {
							continue
						}
						node, ok := data.LegionAdditions[legionRowID(a.AlternatePassiveAddition.ID)]
						if !ok {
							continue
						}
						if d := desiredNodes[node.ID]; d != nil {
							add(curSeed, node.ID, targetNode, d.nodeWeight*float64(a.StatRolls[0]))
						}
					}
				}
			} else if d := desiredNodes[curNodeID]; d != nil && curNodeID != "" {
				add(curSeed, curNodeID, targetNode, d.nodeWeight)
			}
		}
		if resultNodes[curSeed] != nil && desiredNodes["totalStat"] != nil {
			r := resultNodes[curSeed]["totalStat"]
			if r == nil {
				r = &timelessResultNode{}
				resultNodes[curSeed]["totalStat"] = r
			}
			var addedWeight float64
			if jewelTypeID == 4 {
				addedWeight = float64(desiredNodes["totalStat"].nodeWeight * float64(5*otherSmalls+10*attributeSmalls))
				addedWeight += float64(r.totalWeight * 4)
			} else {
				addedWeight = float64(desiredNodes["totalStat"].nodeWeight * float64(4*otherSmalls+2*attributeSmalls))
				addedWeight += float64(r.totalWeight * 19)
			}
			r.totalWeight += addedWeight
			seedWeights[curSeed] += addedWeight
		}
		if resultNodes[curSeed] != nil {
			for _, mw := range minimumWeights {
				w := 0.0
				if r := resultNodes[curSeed][mw.reqNode]; r != nil {
					w = r.totalWeight
				}
				if w < mw.weight {
					delete(resultNodes, curSeed)
					break
				}
			}
		}
	}
	return &timelessSocketSearch{resultNodes: resultNodes, seedWeights: seedWeights, desired: desiredNodes}
}

//pob:block:src/Classes/TreeTab.lua:1558-1588
func (m *MiscCalculator) timelessAllocatedNotablesInRadius(socketID int64) int {
	tree := specTree()
	alloc := allocSet(m.build, tree)
	count := 0
	for id := range tree.NodesInRadiusOf(socketID, 2) {
		_, granted := m.base.GrantedPassives[strconv.FormatInt(id, 10)]
		n := tree.Nodes[strconv.FormatInt(id, 10)]
		if (granted || alloc[id]) && n.IsNotable != nil && *n.IsNotable {
			count++
		}
	}
	return count
}

//pob:block:src/Classes/TreeTab.lua:1558-1588
func (m *MiscCalculator) TimelessProtectableNodes(socketID int64) []TimelessMod {
	tree := specTree()
	if _, ok := tree.Nodes[strconv.FormatInt(socketID, 10)]; !ok || socketID == -1 {
		return nil
	}
	alloc := allocSet(m.build, tree)
	var out []TimelessMod
	for id := range tree.NodesInRadiusOf(socketID, 2) {
		_, granted := m.base.GrantedPassives[strconv.FormatInt(id, 10)]
		n := tree.Nodes[strconv.FormatInt(id, 10)]
		if (granted || alloc[id]) && n.IsNotable != nil && *n.IsNotable {
			out = append(out, TimelessMod{Label: timelessName(n.Name), Descriptions: slices.Clone(n.Stats)})
		}
	}
	slices.SortFunc(out, func(a, b TimelessMod) int { return strings.Compare(a.Label, b.Label) })
	return out
}

func (m *MiscCalculator) timelessPaths(tree *data.Tree) map[int64]int64 {
	classID, _ := pob.ClassIDByName(m.build.Build.ClassName)
	return data.TreeVersions[data.LatestTreeVersion].CalculateAllocationPaths(m.build.Build.PassiveNodes, specRoots(tree, classID))
}

var (
	timelessFmtZero      = regexp.MustCompile(` 0`)
	timelessFmtDotZero   = regexp.MustCompile(`(?s).0 `)
	timelessFmtSpaceDot  = regexp.MustCompile(`  \.`)
	timelessFmtTrailZero = regexp.MustCompile(`\.([0-9])0`)
)

//pob:block:src/Classes/TreeTab.lua:2708-2722
func timelessFormatSearchValue(input float64) string {
	s := " " + fmt.Sprintf("%06.2f", input)
	s = timelessFmtZero.ReplaceAllString(s, "   ")
	s = timelessFmtZero.ReplaceAllString(s, "   ")
	s = timelessFmtDotZero.ReplaceAllString(s, "    ")
	s = timelessFmtDotZero.ReplaceAllString(s, "    ")
	s = timelessFmtSpaceDot.ReplaceAllString(s, "0.")
	return timelessFmtTrailZero.ReplaceAllString(s, ".${1}  ")
}

//pob:block:src/Classes/TreeTab.lua:2724-2777
func timelessFormatResults(q TimelessSearchQuery, r *timelessSocketSearch, socket *TimelessSocket) []TimelessSearchResult {
	seeds := make([]int, 0, len(r.resultNodes))
	for seed := range r.resultNodes {
		seeds = append(seeds, seed)
	}
	slices.Sort(seeds)
	var out []TimelessSearchResult
	for _, seed := range seeds {
		seedData := r.resultNodes[seed]
		passesMin := !q.HasTotalMinimumWeight || r.seedWeights[seed] >= q.TotalMinimumWeight
		if r.seedWeights[seed] <= 0 || !passesMin {
			continue
		}
		prefix := ""
		if socket != nil {
			prefix = socket.Label + " | "
		}
		res := TimelessSearchResult{Label: prefix + strconv.Itoa(seed) + ":", Seed: seed, Total: r.seedWeights[seed]}
		if socket != nil {
			res.SocketID = socket.ID
			res.SocketLabel = socket.Label
		}
		switch {
		case q.JewelType == 1 || q.JewelType == 3 || q.JewelType >= 7:
			if seed < 1000 {
				res.Label = "  " + res.Label
			}
		case q.JewelType == 4:
			if seed < 10000 {
				res.Label = "  " + res.Label
			}
		default:
			if seed < 10000 {
				res.Label = "    " + res.Label
			} else if seed < 100000 {
				res.Label = "  " + res.Label
			}
		}
		sorted := map[int]string{}
		for legionID, d := range r.desired {
			if hit := seedData[legionID]; hit != nil {
				if d.desiredIdx == 8 {
					sorted[8] = " ..."
				} else if d.desiredIdx < 8 {
					sorted[d.desiredIdx] = timelessFormatSearchValue(hit.totalWeight)
				}
				names := slices.Clone(hit.targetNodeNames)
				res.Nodes = append(res.Nodes, TimelessNodeMatch{ID: legionID, TargetNodeNames: names, TotalWeight: hit.totalWeight})
			} else if d.desiredIdx < 8 {
				sorted[d.desiredIdx] = "     0     "
			}
		}
		slices.SortFunc(res.Nodes, func(a, b TimelessNodeMatch) int { return r.desired[a.ID].desiredIdx - r.desired[b.ID].desiredIdx })
		for i := 1; ; i++ {
			s, ok := sorted[i]
			if !ok {
				break
			}
			res.Label += s
		}
		out = append(out, res)
	}
	return out
}

//pob:block:src/Classes/TreeTab.lua:2097-2106
func TimelessFallbackModes() []PowerStat {
	return PowerStats()
}

func timelessLegionOrder() ([]string, []string) {
	nodes := make([]string, 0, len(tjdata.AlternatePassiveSkills))
	for _, s := range tjdata.AlternatePassiveSkills {
		nodes = append(nodes, legionNodeRowID(s))
	}
	additions := make([]string, 0, len(tjdata.AlternatePassiveAdditions))
	for _, a := range tjdata.AlternatePassiveAdditions {
		additions = append(additions, legionRowID(a.ID))
	}
	return nodes, additions
}

//pob:block:src/Classes/TreeTab.lua:1929-2094
func (m *MiscCalculator) TimelessFallbackWeights(jewelType int, stat string) (string, bool) {
	if jewelType == 1 || raw.EnsureTimelessSources() != nil {
		return "", false
	}
	var powerStat *PowerStat
	for _, s := range TimelessFallbackModes() {
		if s.Stat == stat {
			powerStat = &s
			break
		}
	}
	if powerStat == nil {
		return "", false
	}
	nodeOrder, additionOrder := timelessLegionOrder()
	parsed := func(id string, lines []string) []mod.Mod {
		list := moddb.NewModList()
		processStats(&Environment{}, list, id, "legion", lines)
		return list.Mods()
	}
	type fallbackNode struct {
		id      string
		mods    []mod.Mod
		divisor float64
	}
	var nodes []fallbackNode
	for _, modNode := range TimelessMods(jewelType) {
		if modNode.ID == "" {
			continue
		}
		var n *fallbackNode
		for _, id := range nodeOrder {
			legionNode, ok := data.LegionNodes[id]
			if !ok || (id != modNode.ID && !timelessTotalModIDs[modNode.ID][id]) {
				continue
			}
			n = &fallbackNode{id: modNode.ID, mods: parsed(id, legionNode.Stats)}
			if modNode.TotalMod && len(n.mods) > 0 {
				n.divisor = n.mods[0].Value().Float()
			}
			break
		}
		if n == nil {
			for _, id := range additionOrder {
				add, ok := data.LegionAdditions[id]
				if !ok || (id != modNode.ID && !timelessTotalModIDs[modNode.ID][id]) {
					continue
				}
				n = &fallbackNode{id: modNode.ID, divisor: 1}
				if entry := ParseMod(firstOf(add.Stats), false); entry != nil {
					n.mods = entry.ModList
				}
				if modNode.TotalMod && len(n.mods) > 0 {
					n.divisor = n.mods[0].Value().Float()
				}
				break
			}
		}
		if n != nil {
			nodes = append(nodes, *n)
		}
	}

	basePower := powerStatValue(m.base, *powerStat)
	var rows []string
	for _, n := range nodes {
		weight := 0.0
		if basePower != 0 {
			nodePower := powerStatValue(m.Calc(&CalcOverride{AddMods: n.mods}), *powerStat)
			divisor := n.divisor
			if divisor == 0 {
				divisor = 1
			}
			weight = (nodePower - basePower) / math.Abs(basePower) / divisor
		}
		if weight != 0 {
			rows = append(rows, n.id+", "+luaNumber(luaRound(weight*100, 3))+", "+luaNumber(luaRound(0, 3))+", 0")
		}
	}
	return strings.Join(rows, "\n"), true
}
