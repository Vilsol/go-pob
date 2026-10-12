package calculator

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/utils"
)

type DisplayModLine struct {
	Line      string
	Kind      string
	Section   string
	Raw       int
	HasRange  bool
	Range     float64
	Disabled  bool
	Removable bool
	Removal   string
	Mutable   bool
	Mutated   bool
}

type DisplayItemInfo struct {
	Quality         int
	HasQuality      bool
	Variants        []string
	Variant         int
	AltVariants     []int
	VariantGroups   *DisplayVariantGroups
	Sockets         string
	SocketLimit     int
	DefaultColor    string
	CanInfluence    bool
	Influence       []string
	ShowsCatalyst   bool
	Catalyst        int
	CatalystQuality int
	Mods            []DisplayModLine
	Enchants        int
	Anoints         int
	Anointed        []string
	Corruptible     bool
	AddImplicit     bool
	AddCustom       bool
	AddCrucible     bool
	Cluster         *ClusterInfo
}

var (
	modSpecPrefixRe = regexp.MustCompile(`^(\{[^}]*\})+`)
	modTagSuffixRe  = regexp.MustCompile(` \((implicit|crafted|enchant|fractured|scourge|crucible|disabled|mutated|unscalable|vestigial)\)$`)
	rangeTokenRe    = regexp.MustCompile(`\{range:[\d.]+\}`)
)

func rawModText(line string) string {
	line = strings.TrimSpace(line)
	line = modSpecPrefixRe.ReplaceAllString(line, "")
	for modTagSuffixRe.MatchString(line) {
		line = modTagSuffixRe.ReplaceAllString(line, "")
	}
	return line
}

func itemHeaderEnd(lines []string, item *Item) int {
	for k, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "Rarity: ") {
			end := k + 2
			if item.Title != "" {
				end++
			}
			return min(end, len(lines))
		}
	}
	return min(1, len(lines))
}

func foldModRanges(raw string) string {
	if len(parseModRangeElements(raw)) == 0 {
		return raw
	}
	item := ParseItem(raw)
	var lines []string
	for _, l := range splitRaw(modRangeElemRE.ReplaceAllString(raw, "")) {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	next := 0
	for _, list := range [][]ModLine{item.EnchantModLines, item.ScourgeModLines, item.ImplicitModLines, item.ExplicitModLines, item.CrucibleModLines} {
		for _, ml := range list {
			first, _, _ := strings.Cut(ml.Line, "\n")
			for k := next; k < len(lines); k++ {
				if rawModText(lines[k]) != first {
					continue
				}
				next = k + 1
				if ml.Range != nil && !rangeTokenRe.MatchString(lines[k]) {
					lines[k] = "{range:" + strconv.FormatFloat(utils.RoundTo(*ml.Range, 6), 'f', -1, 64) + "}" + strings.TrimSpace(lines[k])
				}
				break
			}
		}
	}
	return strings.Join(lines, "\n")
}

func splitRaw(raw string) []string {
	return strings.Split(strings.ReplaceAll(strings.TrimRight(raw, "\r\n"), "\r\n", "\n"), "\n")
}

func setSpec(lines []string, item *Item, name string, value string, set bool) []string {
	out := make([]string, 0, len(lines)+1)
	at := itemHeaderEnd(lines, item)
	for k, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), name+": ") {
			if k < at {
				at--
			}
			continue
		}
		out = append(out, l)
	}
	if set {
		out = append(out[:at], append([]string{name + ": " + value}, out[at:]...)...)
	}
	return out
}

func displayModLines(lines []string, item *Item) []DisplayModLine {
	var out []DisplayModLine
	next := 0
	add := func(section string, mls []ModLine) {
		for _, ml := range mls {
			if !item.ItemModLineVisible(ml) {
				continue
			}
			text, kind, ok := FormatModLine(ml)
			if !ok {
				continue
			}
			raw := -1
			for k := next; k < len(lines); k++ {
				if rawModText(lines[k]) == ml.Line {
					raw, next = k, k+1
					break
				}
			}
			d := DisplayModLine{Line: text, Kind: string(kind), Section: section, Raw: raw, Disabled: ml.Disabled,
				Mutable: ml.ModID != "" && ml.NewModID != "", Mutated: ml.Mutated}
			if raw >= 0 && rangedModLineRe.MatchString(ml.Line) {
				d.HasRange = true
				d.Range = 0.5
				if ml.Range != nil {
					d.Range = *ml.Range
				}
			}
			//pob:block:src/Classes/ItemsTab.lua:2338-2360
			if (section == "explicit" || section == "crucible") && raw >= 0 && (ml.Custom || ml.Crafted || ml.Crucible) &&
				(item.RareLikeUnique || item.Rarity == RarityMagic || item.Rarity == RarityRare || len(item.CrucibleModLines) > 0) {
				d.Removable = true
				switch {
				case ml.Crafted:
					d.Removal = "Crafted:"
				case ml.Crucible:
					d.Removal = "Crucible:"
				default:
					d.Removal = "Custom:"
				}
			}
			out = append(out, d)
		}
	}
	add("enchant", item.EnchantModLines)
	add("scourge", item.ScourgeModLines)
	add("implicit", item.ImplicitModLines)
	add("explicit", item.ExplicitModLines)
	add("crucible", item.CrucibleModLines)
	return out
}

func visibleModLine(item *Item, index int) *ModLine {
	n := 0
	for _, list := range [][]ModLine{item.EnchantModLines, item.ScourgeModLines, item.ImplicitModLines, item.ExplicitModLines, item.CrucibleModLines} {
		for k := range list {
			if !item.ItemModLineVisible(list[k]) {
				continue
			}
			if _, _, ok := FormatModLine(list[k]); !ok {
				continue
			}
			if n == index {
				return &list[k]
			}
			n++
		}
	}
	return nil
}

type DisplayVariantGroups struct {
	Versions []string
	Version  int
	Groups   []DisplayVariantGroup
}

type DisplayVariantGroup struct {
	Group    int
	Options  []DisplayVariantOption
	Selected int
	Enabled  bool
}

type DisplayVariantOption struct {
	Variant int
	Label   string
}

//pob:src/Classes/ItemsTab.lua:1887-1938
func displayVariantGroups(item *Item) *DisplayVariantGroups {
	out := &DisplayVariantGroups{Versions: item.VersionList}
	if item.SelectedVersion != nil {
		out.Version = *item.SelectedVersion
	}
	for _, groupID := range slices.Sorted(maps.Keys(item.VariantGroups)) {
		if len(item.VariantGroupOptions(groupID, false)) == 0 {
			continue
		}
		if len(out.Groups) == 6 {
			break
		}
		available := item.VariantGroupOptions(groupID, true)
		g := DisplayVariantGroup{Group: groupID, Enabled: len(available) > 1}
		for _, v := range available {
			g.Options = append(g.Options, DisplayVariantOption{Variant: v, Label: item.VariantList[v-1]})
			if item.VariantGroupSelections[groupID] == v {
				g.Selected = v
			}
		}
		if len(g.Options) == 0 {
			g.Options = []DisplayVariantOption{{Label: "No available variants"}}
		}
		out.Groups = append(out.Groups, g)
	}
	return out
}

//pob:block:src/Classes/ItemsTab.lua:378-503,src/Classes/ItemsTab.lua:578-694
func DisplayItem(raw string) (DisplayItemInfo, bool) {
	item := ParseItem(raw)
	if item.Base == nil {
		return DisplayItemInfo{}, false
	}
	info := DisplayItemInfo{Sockets: socketsText(item.Sockets), CanInfluence: item.Base.Type != "Jewel" && item.Base.Type != "Flask" && item.Base.Type != "Tincture"}
	if item.Quality != nil {
		info.Quality, info.HasQuality = *item.Quality, true
	}
	if item.UsesVariantGroups {
		info.VariantGroups = displayVariantGroups(item)
	} else if len(item.VariantList) > 1 {
		info.Variants = item.VariantList
		info.Variant = len(item.VariantList)
		if item.Variant != nil {
			info.Variant = *item.Variant
		}
		for _, alt := range []struct {
			has bool
			sel *int
		}{{item.HasAltVariant, item.VariantAlt}, {item.HasAltVariant2, item.VariantAlt2}, {item.HasAltVariant3, item.VariantAlt3}, {item.HasAltVariant4, item.VariantAlt4}, {item.HasAltVariant5, item.VariantAlt5}} {
			if !alt.has {
				break
			}
			v := 1
			if alt.sel != nil {
				v = *alt.sel
			}
			info.AltVariants = append(info.AltVariants, v)
		}
	}
	info.SocketLimit = max(item.Base.SocketLimit, len(item.Sockets)) - int(item.AbyssalSocketCount)
	info.DefaultColor = item.defaultSocketColor()
	info.Influence = influenceNames(item.Influence)
	//pob:block:src/Classes/ItemsTab.lua:677-679
	hasModTags := false
	for _, ml := range item.ExplicitModLines {
		if len(ml.ModTags) > 0 {
			hasModTags = true
		}
	}
	t := item.Base.Type
	info.ShowsCatalyst = (item.Crafted || hasModTags) && (t == "Amulet" || t == "Ring" || t == "Belt")
	if item.Catalyst != nil {
		info.Catalyst = *item.Catalyst
	}
	if item.CatalystQuality != nil {
		info.CatalystQuality = *item.CatalystQuality
	}
	info.Mods = displayModLines(splitRaw(raw), item)
	info.Enchants = enchantButtons(item)
	info.Anoints = anointButtons(item)
	info.Anointed = ItemAnoints(item)
	info.Corruptible = item.Base.Type != "Flask"
	info.AddImplicit = addImplicitShown(item)
	info.AddCustom = addCustomShown(item)
	info.AddCrucible = addCrucibleShown(item)
	if c, ok := itemClusterInfo(raw, item); ok {
		info.Cluster = &c
	}
	return info, true
}

func influenceNames(i ItemInfluence) []string {
	var out []string
	for _, f := range influenceInfoAll {
		if f.on(i) {
			out = append(out, f.display)
		}
	}
	return out
}

type ItemEdit struct {
	Kind   string
	Index  int
	Value  int
	Second int
	Float  float64
	Text   string
}

var altVariantSpecs = []string{"Selected Variant", "Selected Alt Variant", "Selected Alt Variant Two", "Selected Alt Variant Three", "Selected Alt Variant Four", "Selected Alt Variant Five"}

//pob:block:src/Classes/ItemsTab.lua:391-503,src/Classes/ItemsTab.lua:583-694,src/Classes/ItemsTab.lua:2034-2047,src/Classes/ItemsTab.lua:2361-2371
func EditItem(raw string, e ItemEdit) (string, error) {
	raw = foldModRanges(raw)
	item := ParseItem(raw)
	if item.Base == nil {
		return raw, fmt.Errorf("invalid item: no base type resolved")
	}
	lines := splitRaw(raw)
	switch e.Kind {
	case "quality":
		lines = setSpec(lines, item, "Quality", strconv.Itoa(e.Value), true)
	case "variant":
		if e.Index < 0 || e.Index >= len(altVariantSpecs) {
			return raw, fmt.Errorf("no variant selector %d", e.Index)
		}
		name := altVariantSpecs[e.Index]
		found := false
		for k, l := range lines {
			if strings.HasPrefix(strings.TrimSpace(l), name+": ") {
				lines[k], found = name+": "+strconv.Itoa(e.Value), true
			}
		}
		if !found {
			at := itemHeaderEnd(lines, item)
			for k, l := range lines {
				if l = strings.TrimSpace(l); strings.HasPrefix(l, "Variant: ") || strings.HasPrefix(l, "Has Alt Variant") || strings.HasPrefix(l, "Selected ") {
					at = k + 1
				}
			}
			lines = append(lines[:at], append([]string{name + ": " + strconv.Itoa(e.Value)}, lines[at:]...)...)
		}
	case "sockets":
		lines = setSpec(lines, item, "Sockets", e.Text, e.Text != "")
	case "influence":
		var kept []string
		for _, l := range lines {
			isInfluence := false
			for _, inf := range influenceDisplay {
				if strings.TrimSpace(l) == inf.display+" Item" {
					isInfluence = true
				}
			}
			if !isInfluence {
				kept = append(kept, l)
			}
		}
		at := itemHeaderEnd(kept, item)
		var add []string
		for _, name := range strings.Split(e.Text, ",") {
			for _, inf := range influenceDisplay {
				if name == inf.display && !slices.Contains(add, name+" Item") {
					add = append(add, name+" Item")
				}
			}
		}
		lines = append(kept[:at:at], append(add, kept[at:]...)...)
	case "catalyst":
		var kept []string
		for _, l := range lines {
			if t := strings.TrimSpace(l); strings.HasPrefix(t, "Catalyst: ") || strings.HasPrefix(t, "CatalystQuality: ") || catalystQualitySpecRe.MatchString(t) {
				continue
			}
			kept = append(kept, l)
		}
		lines = kept
		if e.Value > 0 && e.Value <= len(catalystList) {
			q := e.Second
			if q == 0 && item.CatalystQuality == nil {
				q = 20
			} else if q == 0 {
				q = *item.CatalystQuality
			}
			at := itemHeaderEnd(lines, item)
			lines = append(lines[:at], append([]string{"Catalyst: " + catalystList[e.Value-1], "CatalystQuality: " + strconv.Itoa(q)}, lines[at:]...)...)
		}
	case "range", "disable", "remove":
		mods := displayModLines(lines, item)
		if e.Index < 0 || e.Index >= len(mods) || mods[e.Index].Raw < 0 {
			return raw, fmt.Errorf("no mod line %d", e.Index)
		}
		k := mods[e.Index].Raw
		switch e.Kind {
		case "range":
			if !mods[e.Index].HasRange {
				return raw, fmt.Errorf("mod line %d has no range", e.Index)
			}
			v := utils.RoundTo(min(max(e.Float, 0), 1), 6)
			lines[k] = "{range:" + strconv.FormatFloat(v, 'f', -1, 64) + "}" + rangeTokenRe.ReplaceAllString(strings.TrimSpace(lines[k]), "")
		case "disable":
			if mods[e.Index].Disabled {
				lines[k] = strings.TrimSuffix(strings.Replace(strings.TrimSpace(lines[k]), "{disabled}", "", 1), " (disabled)")
			} else {
				lines[k] = "{disabled}" + strings.TrimSpace(lines[k])
			}
		case "remove":
			if !mods[e.Index].Removable {
				return raw, fmt.Errorf("mod line %d is not a custom, crafted or crucible mod", e.Index)
			}
			lines = append(lines[:k], lines[k+1:]...)
		}
	case "version", "variantGroup":
		if !item.UsesVariantGroups {
			return raw, fmt.Errorf("the item has no versions or variant groups")
		}
		if e.Kind == "version" {
			item.SelectedVersion = intPtr(e.Value)
		} else {
			if e.Value == 0 {
				return raw, nil
			}
			item.VariantGroupSelections[e.Index] = e.Value
		}
		item.normaliseVariantSelections()
		return buildRawText(item), nil
	case "mutate":
		mods := displayModLines(lines, item)
		if e.Index < 0 || e.Index >= len(mods) || !mods[e.Index].Mutable {
			return raw, fmt.Errorf("mod line %d has no Foulborn counterpart", e.Index)
		}
		ml := visibleModLine(item, e.Index)
		if ml == nil || !item.mutateMod(ml.ModID, ml.NewModID, !ml.Mutated) {
			return raw, fmt.Errorf("mod line %d has no Foulborn counterpart", e.Index)
		}
		return buildRawText(item), nil
	case "cluster":
		if text, ok := craftClusterText(raw, e.Text, e.Value); ok {
			return text, nil
		}
		return raw, fmt.Errorf("not a crafted cluster jewel")
	case "enchant":
		if text, ok := enchantText(raw, e.Index, e.Text, e.Value == 1); ok {
			return text, nil
		}
		return raw, fmt.Errorf("no enchant slot %d", e.Index)
	case "anoint":
		if text, ok := anointText(raw, e.Index, e.Text); ok {
			return text, nil
		}
		return raw, fmt.Errorf("no anoint slot %d", e.Index)
	default:
		return raw, fmt.Errorf("unknown item edit %q", e.Kind)
	}
	return strings.Join(lines, "\n"), nil
}

type CraftBaseGroup struct {
	Type  string
	Bases []string
}

//pob:block:src/Modules/Data.lua:1224-1254
func CraftBaseGroups() []CraftBaseGroup {
	byType := map[string][]*ItemBase{}
	for _, b := range ItemBases() {
		pb, ok := data.CraftBaseKnown(b.Name)
		if !ok || pb.Hidden {
			continue
		}
		t := pb.Type
		if pb.SubType != "" {
			t += ": " + pb.SubType
		}
		byType[t] = append(byType[t], b)
	}
	out := make([]CraftBaseGroup, 0, len(byType))
	for t, list := range byType {
		slices.SortFunc(list, func(a, b *ItemBase) int {
			if a.Req.Level == b.Req.Level {
				return strings.Compare(a.Name, b.Name)
			}
			return b.Req.Level - a.Req.Level
		})
		g := CraftBaseGroup{Type: t}
		for _, b := range list {
			g.Bases = append(g.Bases, b.Name)
		}
		out = append(out, g)
	}
	slices.SortFunc(out, func(a, b CraftBaseGroup) int { return strings.Compare(a.Type, b.Type) })
	return out
}

//pob:block:src/Classes/ItemsTab.lua:2521-2565
func CraftItemText(rarity int, title string, baseName string) (string, bool) {
	base := ItemBaseByName(baseName)
	if base == nil || rarity < 0 || rarity > 4 {
		return "", false
	}
	if (base.Flask != nil || (base.Type == "Jewel" && base.SubType == "Charm") || base.Type == "Tincture") && rarity == 2 {
		rarity = 1
	}
	rarities := []string{"NORMAL", "MAGIC", "RARE", "UNIQUE", "RELIC"}
	lines := []string{"Rarity: " + rarities[rarity]}
	if rarity >= 2 {
		if strings.TrimSpace(title) == "" {
			title = "New Item"
		}
		lines = append(lines, title)
	}
	lines = append(lines, baseName)
	if rarity == 1 || rarity == 2 {
		lines = append(lines, "Crafted: true")
	}
	switch base.Type {
	case "Amulet", "Belt", "Jewel", "Quiver", "Ring", "Graft":
	default:
		quality := 0
		//pob:block:src/Classes/Item.lua:1644-1652
		if base.Armour != nil || base.Weapon != nil || base.Flask != nil || base.Tincture != nil {
			quality = 20
		}
		lines = append(lines, "Quality: "+strconv.Itoa(quality))
	}
	pb := data.CraftBaseInfo(baseName)
	var implicits []string
	if pb.Implicit != "" {
		for k, line := range strings.Split(pb.Implicit, "\n") {
			if line == "" {
				continue
			}
			if k < len(pb.ImplicitModTypes) && len(pb.ImplicitModTypes[k]) > 0 {
				line = "{tags:" + strings.Join(pb.ImplicitModTypes[k], ",") + "}" + line
			}
			implicits = append(implicits, line)
		}
	}
	lines = append(lines, "Implicits: "+strconv.Itoa(len(implicits)))
	lines = append(lines, implicits...)
	return strings.Join(lines, "\n"), true
}

func socketsText(sockets []ItemSocket) string {
	var b strings.Builder
	for i, so := range sockets {
		if i > 0 {
			if so.Group == sockets[i-1].Group {
				b.WriteByte('-')
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteString(so.Color)
	}
	return b.String()
}
