package tree2

import (
	"encoding/xml"
	"html"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// buildItems is the item side of a build file reduced to what the spec reads: jewel radius and jewel
// data per item, the active set's slots, and "Allocates" lines. Test code only: the item parser is the calc's.
type buildItems struct {
	items    map[int]*Item
	allocate map[int][]GrantedPassive
	slots    []int
}

type itemsXML struct {
	Items struct {
		ActiveItemSet string `xml:"activeItemSet,attr"`
		Items         []struct {
			ID   string `xml:"id,attr"`
			Text string `xml:",chardata"`
		} `xml:"Item"`
		Sets []struct {
			ID    string `xml:"id,attr"`
			Slots []struct {
				Name   string `xml:"name,attr"`
				ItemID string `xml:"itemId,attr"`
			} `xml:"Slot"`
		} `xml:"ItemSet"`
	} `xml:"Items"`
}

var (
	itemTagPrefixRE = regexp.MustCompile(`^(?:\{[^}]*\})+`)
	fromNothingRE   = regexp.MustCompile(`^passives in radius of ([a-z\s']+) can be allocated without being connected to your tree$`)
	allocatesLineRE = regexp.MustCompile(`^allocates (.+)$`)
	sinisterLineRE  = regexp.MustCompile(`^allocates (\d+) sinister jewel sockets?$`)
	altStartRE      = regexp.MustCompile(`^can allocate passive(?:s| skills) from the ([a-z]+)'s starting point$`)
	ringRE          = regexp.MustCompile(`^(?:only )?affects passives in (.+) ring$`)
	upgradeRadiusRE = regexp.MustCompile(`^upgrades radius to (.+)$`)
)

//pob2:block:src/Modules/ModParser.lua:5604-5619
var (
	ringRadiusIndex = map[string]int{"very small": 5, "small": 6, "medium-small": 7, "medium": 8, "medium-large": 9, "large": 10, "very large": 11, "massive": 12}
	// "affects" (not "only affects") has no very small or medium-small row.
	ringNeedsOnly         = map[string]bool{"very small": true, "medium-small": true, "medium-large": true}
	timeLostRadiusUpgrade = map[string]int{"medium": 2, "large": 3, "very large": 4}
)

// radiusFromLines returns jewelData.radiusIndex and jewelData.timeLostJewelRadiusOverride (0 = nil).
func radiusFromLines(lines []string) (int, int) {
	radiusIndex, override := 0, 0
	for _, l := range lines {
		if m := ringRE.FindStringSubmatch(l); m != nil {
			if idx, ok := ringRadiusIndex[m[1]]; ok && (strings.HasPrefix(l, "only ") || !ringNeedsOnly[m[1]]) {
				radiusIndex = idx
			}
		}
		if m := upgradeRadiusRE.FindStringSubmatch(l); m != nil {
			override = timeLostRadiusUpgrade[m[1]]
		}
	}
	return radiusIndex, override
}

//pob2:block:src/Modules/ModParser.lua:5599-5603
func jewelDataFromLines(lines []string) *JewelData {
	var jd *JewelData
	get := func() *JewelData {
		if jd == nil {
			jd = &JewelData{FromNothingKeystones: map[string]bool{}}
		}
		return jd
	}
	for i := range lines {
		for _, cand := range []string{lines[i], strings.Join(lines[i:min(i+2, len(lines))], " ")} {
			if cand == "passives in radius can be allocated without being connected to your tree" {
				get().IntuitiveLeapLike = true
			}
			if m := fromNothingRE.FindStringSubmatch(cand); m != nil {
				get().FromNothingKeystone = m[1]
				get().FromNothingKeystones[m[1]] = true
			}
			//pob2:block:src/Modules/ModParser.lua:2786-2787
			if m := altStartRE.FindStringSubmatch(cand); m != nil {
				get().AlternateClassStart = strings.ToUpper(m[1][:1]) + m[1][1:]
			}
		}
	}
	return jd
}

func readBuildItems(t *testing.T, tree *PassiveTree, raw []byte) buildItems {
	t.Helper()
	var doc itemsXML
	if err := xml.Unmarshal(raw, &doc); err != nil { //nolint:gosec // a local build file
		t.Fatal(err)
	}
	out := buildItems{items: map[int]*Item{}, allocate: map[int][]GrantedPassive{}}
	for _, it := range doc.Items.Items {
		id, err := strconv.Atoi(it.ID)
		if err != nil {
			continue
		}
		var lines []string
		radiusIndex, isJewel, deferRadius := 0, false, false
		for _, l := range strings.Split(it.Text, "\n") {
			l = strings.TrimSpace(html.UnescapeString(l))
			if v, ok := strings.CutPrefix(l, "Radius: "); ok {
				isJewel = true
				deferRadius = strings.HasPrefix(v, "Variable")
				//pob2:block:src/Classes/Item.lua:830-842
				for i, r := range tree.JewelRadius {
					if v == r.Label {
						radiusIndex = i + 1
						break
					}
				}
				continue
			}
			lines = append(lines, strings.ToLower(itemTagPrefixRE.ReplaceAllString(l, "")))
		}
		//pob2:block:src/Modules/ModParser.lua:5964-5965
		for _, l := range lines {
			if m := sinisterLineRE.FindStringSubmatch(l); m != nil {
				n, _ := strconv.Atoi(m[1])
				out.allocate[id] = append(out.allocate[id], GrantedPassive{Table: true, Type: "SinisterJewelSockets", Count: n})
			} else if m := allocatesLineRE.FindStringSubmatch(l); m != nil {
				out.allocate[id] = append(out.allocate[id], GrantedPassive{Name: m[1]})
			}
		}
		//pob2:block:src/Classes/Item.lua:1820-1825
		ringIndex, override := radiusFromLines(lines)
		if deferRadius {
			radiusIndex = ringIndex
		}
		if override != 0 {
			radiusIndex = override
		}
		item := &Item{JewelRadiusIndex: radiusIndex, JewelData: jewelDataFromLines(lines)}
		if isJewel && item.JewelData == nil {
			item.JewelData = &JewelData{FromNothingKeystones: map[string]bool{}}
		}
		out.items[id] = item
	}
	active := doc.Items.ActiveItemSet
	for _, set := range doc.Items.Sets {
		if set.ID != active {
			continue
		}
		for _, sl := range set.Slots {
			if id, err := strconv.Atoi(sl.ItemID); err == nil && id > 0 && !strings.HasSuffix(sl.Name, " Swap") {
				out.slots = append(out.slots, id)
			}
		}
	}
	return out
}

// grantSockets emulates CollectGrantedPassiveNodesFromItems + SetGrantedPassiveNodes for the
// equipped items and the jewels in allocated sockets, without IsItemValidForSlot or jewel limits.
func grantSockets(spec *PassiveSpec, bi buildItems) {
	granted := map[int]*SpecNode{}
	alloc := func(id int) bool { return spec.AllocNodes[id] != nil || granted[id] != nil }
	for changed, safety := true, 0; changed && safety < 8; safety++ {
		changed = false
		ids := append([]int(nil), bi.slots...)
		for _, socketID := range sortedKeys(spec.Jewels) {
			if alloc(socketID) {
				ids = append(ids, spec.Jewels[socketID])
			}
		}
		for _, itemID := range ids {
			for _, passive := range bi.allocate[itemID] {
				for _, n := range spec.ResolveGrantedPassiveNodes(passive) {
					if n.Type == "Socket" && granted[n.ID] == nil {
						granted[n.ID] = n
						changed = true
					}
				}
			}
		}
	}
	spec.SetGrantedPassiveNodes(granted)
}
