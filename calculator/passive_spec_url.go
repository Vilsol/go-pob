package calculator

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

func u16(v int64) []byte {
	return []byte{byte(v >> 8 & 0xff), byte(v & 0xff)} //nolint:gosec // masked to a byte
}

func specClassIDs(spec pob.Spec) (int, int, int) {
	secondary := 0
	if spec.SecondaryAscendClassID != nil {
		secondary = *spec.SecondaryAscendClassID
	}
	return int(spec.ClassID), int(spec.AscendClassID), secondary
}

//pob:src/Classes/PassiveSpec.lua:514-559
func EncodeTreeURL(build *pob.PathOfBuilding, prefix string) string {
	i := build.ActiveSpecIndex()
	if i < 0 {
		return ""
	}
	spec := build.Tree.Specs[i]
	tree := specTree()
	selections := map[int64]int64{}
	for _, s := range pob.SpecMasterySelections(spec) {
		selections[s.Node] = s.Effect
	}
	classID, ascendID, secondaryID := specClassIDs(spec)

	var nodes, cluster, mastery []byte
	nodeCount, clusterCount, masteryCount := 0, 0, 0
	seen := map[int64]bool{}
	for _, id := range build.Build.PassiveNodes {
		if seen[id] {
			continue
		}
		seen[id] = true
		if id >= 65536 {
			c := id - 65536
			cluster = append(cluster, u16(c)...)
			clusterCount++
			continue
		}
		node, ok := tree.Nodes[nodeKey(id)]
		if !ok || isClassStartNode(node) || isAscendClassStartNode(node) || nodeCount >= 255 {
			continue
		}
		nodes = append(nodes, u16(id)...)
		nodeCount++
		if effect, ok := selections[id]; ok {
			mastery = append(append(mastery, u16(effect)...), u16(id)...)
			masteryCount++
		}
	}
	a := make([]byte, 0, 10+len(nodes)+len(cluster)+len(mastery))
	a = append(a, 0, 0, 0, 6, byte(classID), byte(secondaryID<<2|ascendID)) //nolint:gosec // ids below 256
	a = append(a, byte(nodeCount))
	a = append(a, nodes...)
	a = append(a, byte(clusterCount))
	a = append(a, cluster...)
	a = append(a, byte(masteryCount))
	a = append(a, mastery...)
	return prefix + base64.URLEncoding.EncodeToString(a)
}

type DecodedTree struct {
	ClassID          int
	AscendClassID    int
	SecondaryAscend  int
	Nodes            []int64
	MasterySelection map[int64]int64
	Version          pob.TreeVersion
}

func decodeTreeLinkBytes(url string) ([]byte, bool) {
	s := url[strings.LastIndex(url, "/")+1:]
	s = strings.NewReplacer("-", "+", "_", "/").Replace(s)
	s = strings.TrimRight(s, "=")
	b, err := base64.RawStdEncoding.DecodeString(s)
	return b, err == nil
}

//pob:src/Classes/PassiveSpec.lua:469-512
func DecodeTreeURL(url string) (DecodedTree, error) {
	b, ok := decodeTreeLinkBytes(url)
	if !ok || len(b) < 6 {
		return DecodedTree{}, fmt.Errorf("Invalid tree link (unrecognised format)") //nolint:staticcheck // PoB's message
	}
	ver := int(b[0])<<24 | int(b[1])<<16 | int(b[2])<<8 | int(b[3])
	if ver > 6 {
		return DecodedTree{}, fmt.Errorf("Invalid tree link (unknown version number '%d')", ver) //nolint:staticcheck // PoB's message
	}
	tree := specTree()
	out := DecodedTree{ClassID: int(b[4]), MasterySelection: map[int64]int64{}}
	if ver >= 4 {
		out.AscendClassID = int(b[5]) & 3
		out.SecondaryAscend = (int(b[5]) & 12) >> 2
	}
	if out.ClassID >= len(tree.Classes) {
		return DecodedTree{}, fmt.Errorf("Invalid tree link (bad class ID '%d')", out.ClassID) //nolint:staticcheck // PoB's message
	}

	alloc := map[int64]bool{}
	add := func(id int64) {
		if _, ok := tree.Nodes[nodeKey(id)]; ok && !alloc[id] {
			alloc[id] = true
			out.Nodes = append(out.Nodes, id)
		}
	}
	for _, id := range decodedStarts(tree, out) {
		add(id)
	}

	sub := func(i, j int) []byte {
		if j < 0 {
			j = len(b) + j + 1
		}
		i = max(i, 1)
		j = min(j, len(b))
		if i > j {
			return nil
		}
		return b[i-1 : j]
	}
	byteAt := func(i int) int {
		if i < 1 || i > len(b) {
			return 0
		}
		return int(b[i-1])
	}
	pairs := func(raw []byte, cluster bool) {
		for i := 0; i+1 < len(raw); i += 2 {
			id := int64(raw[i])<<8 | int64(raw[i+1])
			if cluster {
				id += 65536
			}
			add(id)
		}
	}

	nodesStart := 7
	if ver >= 4 {
		nodesStart = 8
	}
	nodesEnd := -1
	if ver >= 5 {
		nodesEnd = 7 + byteAt(7)*2
	}
	pairs(sub(nodesStart, nodesEnd), false)
	if ver < 5 {
		return out, nil
	}
	clusterStart := nodesEnd + 1
	clusterEnd := clusterStart + byteAt(clusterStart)*2
	pairs(sub(clusterStart+1, clusterEnd), true)
	if ver < 6 {
		return out, nil
	}
	masteryStart := clusterEnd + 1
	masteryEnd := masteryStart + byteAt(masteryStart)*4
	raw := sub(masteryStart+1, masteryEnd)
	for i := 0; i+3 < len(raw); i += 4 {
		effect := int64(raw[i])<<8 | int64(raw[i+1])
		id := int64(raw[i+2])<<8 | int64(raw[i+3])
		if !alloc[id] {
			continue
		}
		if masteryHasEffect(tree, id, effect) {
			out.MasterySelection[id] = effect
		} else {
			delete(alloc, id)
			out.Nodes = removeID(out.Nodes, id)
		}
	}
	return out, nil
}

//pob:block:src/Classes/PassiveSpec.lua:577-580,src/Classes/PassiveSpec.lua:608-613,src/Classes/PassiveSpec.lua:640-650
func decodedStarts(tree *data.Tree, d DecodedTree) []int64 {
	var out []int64
	if id, err := strconv.ParseInt(classStartNodeID(tree, pob.ClassID(d.ClassID)), 10, 64); err == nil {
		out = append(out, id)
	}
	if name := pob.AscendancyNameOf(pob.ClassID(d.ClassID), pob.AscendancyID(d.AscendClassID)); name != "None" {
		if id, err := strconv.ParseInt(classAscendancyStartNodeID(tree, pob.ClassID(d.ClassID), name), 10, 64); err == nil {
			out = append(out, id)
		}
	}
	if d.SecondaryAscend > 0 && d.SecondaryAscend <= len(tree.AlternateAscendancies) {
		if alt := tree.AlternateAscendancies[d.SecondaryAscend-1].ID; !legacyAlternateAscendancies[string(alt)] {
			if id, err := strconv.ParseInt(ascendancyStartNodeID(tree, alt), 10, 64); err == nil {
				out = append(out, id)
			}
		}
	}
	return out
}

func masteryHasEffect(tree *data.Tree, id, effect int64) bool {
	for _, e := range tree.Nodes[nodeKey(id)].MasteryEffects {
		if e.Effect == effect {
			return true
		}
	}
	return false
}

func removeID(ids []int64, id int64) []int64 {
	out := ids[:0]
	for _, x := range ids {
		if x != id {
			out = append(out, x)
		}
	}
	return out
}

var legacyAlternateAscendancies = map[string]bool{"Warden": true, "Warlock": true, "Primalist": true}

var (
	treeLinkVersionRe   = regexp.MustCompile(`tree/([0-9]+)\.([0-9]+)\.([0-9]+)/`)
	treeLinkAlternateRe = regexp.MustCompile(`tree/([a-z]+-?[a-z]*)`)
)

//pob:block:src/Classes/TreeTab.lua:727-741
func validateTreeVersion(alternate string, major, minor string) pob.TreeVersion {
	suffix := ""
	if alternate != "" {
		suffix = "_" + strings.ReplaceAll(alternate, "-", "_")
	}
	version := string(data.LatestTreeVersion) + suffix
	if major != "" && minor != "" {
		ma, _ := strconv.Atoi(major)
		mi, _ := strconv.Atoi(minor)
		version = fmt.Sprintf("%d_%d", ma, mi) + suffix
	}
	if _, ok := data.TreeVersions[pob.TreeVersion(version)]; ok {
		return pob.TreeVersion(version)
	}
	return data.LatestTreeVersion
}

func submatch(re *regexp.Regexp, s string, i int) string {
	if m := re.FindStringSubmatch(s); m != nil && i < len(m) {
		return m[i]
	}
	return ""
}

//pob:block:src/Classes/TreeTab.lua:796-802
func treeLinkVersion(url string) pob.TreeVersion {
	return validateTreeVersion(submatch(treeLinkAlternateRe, url, 1), submatch(treeLinkVersionRe, url, 1), submatch(treeLinkVersionRe, url, 2))
}

var (
	siteVersionRe        = regexp.MustCompile(`/\?v=.+#`)
	skillTreeVersionRe   = regexp.MustCompile(`/\?v=([0-9]+)\.([0-9]+)\.([0-9]+)-?\w?-?\w?#`)
	skillTreeAlternateRe = regexp.MustCompile(`-([a-z]+-?[a-z]*)#`)
)

//pob:block:src/GameVersions.lua:228-260
var poePlannerVersions = map[int]string{
	51: "3_29", 50: "3_29_ruthless", 47: "3_28", 46: "3_28_ruthless", 45: "3_28_alternate", 44: "3_28_ruthless_alternate",
	43: "3_27", 42: "3_27_ruthless", 37: "3_26", 36: "3_26_ruthless", 35: "3_25_alternate", 34: "3_25_ruthless_alternate",
	33: "3_25", 32: "3_25_ruthless", 31: "3_24", 30: "3_23_ruthless", 29: "3_23", 28: "3_22_ruthless", 27: "3_22",
	26: "3_21", 25: "3_20", 24: "3_19", 23: "3_18", 22: "3_17", 21: "3_16", 19: "3_15", 18: "3_14", 17: "3_13",
	16: "3_12", 15: "3_11",
}

//pob:src/Classes/PassiveSpec.lua:365-466
func DecodePoePlannerURL(url string) (DecodedTree, error) {
	b, ok := decodeTreeLinkBytes(url)
	if !ok || len(b) < 15 {
		return DecodedTree{}, fmt.Errorf("Invalid tree link (unrecognised format).") //nolint:staticcheck // PoB's message
	}
	u16At := func(i int) int {
		if i < 1 || i+1 > len(b) {
			return 0
		}
		return int(b[i-1]) | int(b[i])<<8
	}
	name, known := poePlannerVersions[u16At(7)]
	if !known {
		return DecodedTree{}, fmt.Errorf("Invalid tree version found in link.") //nolint:staticcheck // PoB's message
	}
	tree := specTree()
	out := DecodedTree{ClassID: int(b[8]), AscendClassID: int(b[9]), MasterySelection: map[int64]int64{}, Version: data.LatestTreeVersion}
	if _, ok := data.TreeVersions[pob.TreeVersion(name)]; ok {
		out.Version = pob.TreeVersion(name)
	}
	if out.ClassID >= len(tree.Classes) {
		return DecodedTree{}, fmt.Errorf("Invalid tree link (bad class ID '%d')", out.ClassID) //nolint:staticcheck // PoB's message
	}
	alloc := map[int64]bool{}
	add := func(id int64) {
		if _, ok := tree.Nodes[nodeKey(id)]; ok && !alloc[id] {
			alloc[id] = true
			out.Nodes = append(out.Nodes, id)
		}
	}
	for _, id := range decodedStarts(tree, out) {
		add(id)
	}
	list := func(idx, width int) ([]byte, int) {
		count := u16At(idx)
		end := idx + 2 + count*width
		lo, hi := idx+2, min(end-1, len(b))
		if lo > hi {
			return nil, end
		}
		return b[lo-1 : hi], end
	}
	pairs := func(raw []byte, cluster bool) {
		for i := 0; i+1 < len(raw); i += 2 {
			id := int64(raw[i]) | int64(raw[i+1])<<8
			if cluster {
				id += 65536
			}
			add(id)
		}
	}
	nodes, idx := list(12, 2)
	pairs(nodes, false)
	cluster, idx := list(idx, 2)
	pairs(cluster, true)
	ascendancy, idx := list(idx, 2)
	pairs(ascendancy, false)
	mastery, _ := list(idx, 4)
	for i := 0; i+3 < len(mastery); i += 4 {
		id := int64(mastery[i]) | int64(mastery[i+1])<<8
		effect := int64(mastery[i+2]) | int64(mastery[i+3])<<8
		add(id)
		if !alloc[id] {
			continue
		}
		if masteryHasEffect(tree, id, effect) {
			out.MasterySelection[id] = effect
		} else {
			delete(alloc, id)
			out.Nodes = removeID(out.Nodes, id)
		}
	}
	return out, nil
}

//pob:block:src/Classes/TreeTab.lua:759-803
func decodeTreeLink(url string) (DecodedTree, error) {
	switch {
	case strings.Contains(url, "poeurl.com/"):
		return DecodedTree{}, fmt.Errorf("PoEURL links are not resolved here: open it and import the link it leads to")
	case strings.Contains(url, "poeplanner.com/"):
		return DecodePoePlannerURL(siteVersionRe.ReplaceAllString(url, "/"))
	case strings.Contains(url, "poeskilltree.com/"):
		d, err := DecodeTreeURL(siteVersionRe.ReplaceAllString(url, "/"))
		d.Version = validateTreeVersion(submatch(skillTreeAlternateRe, url, 1), submatch(skillTreeVersionRe, url, 1), submatch(skillTreeVersionRe, url, 2))
		return d, err
	default:
		d, err := DecodeTreeURL(url)
		d.Version = treeLinkVersion(url)
		return d, err
	}
}

//pob:block:src/Classes/TreeTab.lua:708-726,src/Classes/TreeTab.lua:796-803
func ImportTreeURL(build *pob.PathOfBuilding, title string, url string) error {
	decoded, err := decodeTreeLink(url)
	if err != nil {
		return err
	}
	nodes := make([]string, 0, len(decoded.Nodes))
	for _, id := range decoded.Nodes {
		nodes = append(nodes, strconv.FormatInt(id, 10))
	}
	selections := make([]pob.MasterySelection, 0, len(decoded.MasterySelection))
	for _, id := range decoded.Nodes {
		if effect, ok := decoded.MasterySelection[id]; ok {
			selections = append(selections, pob.MasterySelection{Node: id, Effect: effect})
		}
	}
	two := 2
	secondary := decoded.SecondaryAscend
	build.AddSpec(pob.Spec{
		ClassID:                  pob.ClassID(decoded.ClassID),
		AscendClassID:            pob.AscendancyID(decoded.AscendClassID),
		SecondaryAscendClassID:   &secondary,
		ClusterHashFormatVersion: &two,
		Title:                    &title,
		TreeVersion:              decoded.Version,
		NodesAttr:                strings.Join(nodes, ","),
		MasteryEffects:           pob.FormatMasterySelections(selections),
		Sockets:                  []pob.Socket{},
		Overrides:                []pob.Override{},
	})
	build.SetActiveSpec(len(build.Tree.Specs))
	return nil
}
