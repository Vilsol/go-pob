package builds

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/Vilsol/go-pob/pob"
)

var nilAttrSuffix = []byte(`="nil"`)

func isWordByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func stripNilAttrs(raw []byte) []byte {
	if !bytes.Contains(raw, nilAttrSuffix) {
		return raw
	}

	out := make([]byte, 0, len(raw))
	pos := 0
	for {
		i := bytes.Index(raw[pos:], nilAttrSuffix)
		if i < 0 {
			return append(out, raw[pos:]...)
		}

		at := pos + i
		start := at
		for start > pos && isWordByte(raw[start-1]) {
			start--
		}

		if start == at {
			out = append(out, raw[pos:at+len(nilAttrSuffix)]...)
		} else {
			out = append(out, raw[pos:start]...)
		}

		pos = at + len(nilAttrSuffix)
	}
}

var pobGemIDtoGameGemIDs = map[string]string{
	"Metadata/Items/Gems/Smite":                 "Metadata/Items/Gems/SkillGemSmite",
	"Metadata/Items/Gems/ConsecratedPath":       "Metadata/Items/Gems/SkillGemConsecratedPath",
	"Metadata/Items/Gems/VaalAncestralWarchief": "Metadata/Items/Gems/SkillGemVaalAncestralWarchief",
	"Metadata/Items/Gems/HeraldOfAgony":         "Metadata/Items/Gems/SkillGemHeraldOfAgony",
	"Metadata/Items/Gems/HeraldOfPurity":        "Metadata/Items/Gems/SkillGemHeraldOfPurity",
	"Metadata/Items/Gems/ScourgeArrow":          "Metadata/Items/Gems/SkillGemScourgeArrow",
	"Metadata/Items/Gems/RainOfSpores":          "Metadata/Items/Gems/SkillGemToxicRain",
	"Metadata/Items/Gems/SummonRelic":           "Metadata/Items/Gems/SkillGemSummonRelic",
}

func PoBGemIDRewrites() map[string]string {
	out := make(map[string]string, len(pobGemIDtoGameGemIDs))
	for k, v := range pobGemIDtoGameGemIDs {
		out[k] = v
	}
	return out
}

func ParseBuildStr(rawXML string) (*pob.PathOfBuilding, error) {
	return ParseBuild([]byte(rawXML))
}

func ParseBuild(rawXML []byte) (*pob.PathOfBuilding, error) {
	clean := stripNilAttrs(rawXML)
	var build pob.PathOfBuilding
	build.TreeView.ShowStatDifferences = true
	err := xml.Unmarshal(clean, &build)
	if err != nil {
		return nil, fmt.Errorf("failed to parse build as xml: %w", err)
	}

	finishSkills(&build)
	build.Notes = strings.Trim(build.Notes, pob.NotesEdgeSpace)
	resolveActiveConfigSet(&build)

	return finishPassiveNodes(&build)
}

func finishSkills(build *pob.PathOfBuilding) {
	for i, set := range build.Skills.SkillSets {
		for j, skill := range set.Skills {
			for k, gem := range skill.Gems {
				if gameID, ok := pobGemIDtoGameGemIDs[gem.GemID]; ok {
					build.Skills.SkillSets[i].Skills[j].Gems[k].GemID = gameID
				}
			}
		}
	}
	resolveSkillSources(build)
}

func finishPassiveNodes(build *pob.PathOfBuilding) (*pob.PathOfBuilding, error) {
	build.Build.PassiveNodes = make([]int64, 0, 100)

	spec, err := activeSpec(build)
	if err != nil {
		return nil, err
	}

	if spec.NodesAttr != "" {
		var nodeStrs = strings.Split(spec.NodesAttr, ",")
		for _, str := range nodeStrs {
			var num, err = strconv.ParseInt(str, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("spec has some non-integer nodes: %s", spec.NodesAttr)
			}
			build.Build.PassiveNodes = append(build.Build.PassiveNodes, num)
		}
	}

	return build, nil
}

func activeSpec(build *pob.PathOfBuilding) (*pob.Spec, error) {
	if len(build.Tree.Specs) == 0 {
		return nil, fmt.Errorf("build has no <Spec> under <Tree>: there is no passive tree to read")
	}

	idx := build.Tree.ActiveSpec
	if idx < 1 {
		idx = 1
	}
	if idx > len(build.Tree.Specs) {
		idx = len(build.Tree.Specs)
	}
	return &build.Tree.Specs[idx-1], nil
}

var serializedSize atomic.Int64

func SerializeBuild(build *pob.PathOfBuilding) ([]byte, error) {
	size := int(serializedSize.Load())
	w := &pobEntityWriter{out: make([]byte, 0, size+size/64)}
	enc := xml.NewEncoder(w)
	if err := enc.Encode(build); err != nil {
		return nil, fmt.Errorf("failed serializing build: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("failed serializing build: %w", err)
	}
	out := w.finish()
	serializedSize.Store(int64(len(out)))
	return out, nil
}

var pobEntities = [...]struct{ from, to string }{{"&#xA;", "\n"}, {"&#x9;", "\t"}, {"&#34;", "&quot;"}, {"&#39;", "&apos;"}}

const pobEntityLen = 5

func replacePoBEntities(b []byte) []byte {
	return appendPoBEntities(make([]byte, 0, len(b)+len(b)/64), b)
}

func appendPoBEntities(out, b []byte) []byte {
	for {
		i := bytes.IndexByte(b, '&')
		if i < 0 {
			return append(out, b...)
		}
		out = append(out, b[:i]...)
		b = b[i:]
		matched := false
		for _, e := range pobEntities {
			if len(b) >= len(e.from) && string(b[:len(e.from)]) == e.from {
				out = append(out, e.to...)
				b = b[len(e.from):]
				matched = true
				break
			}
		}
		if !matched {
			out = append(out, '&')
			b = b[1:]
		}
	}
}

type pobEntityWriter struct {
	out     []byte
	tail    [pobEntityLen]byte
	tailLen int
}

func (w *pobEntityWriter) Write(p []byte) (int, error) {
	n := len(p)
	if w.tailLen > 0 {
		need := min(pobEntityLen-w.tailLen, len(p))
		var joined [pobEntityLen]byte
		head := append(append(joined[:0], w.tail[:w.tailLen]...), p[:need]...)
		switch {
		case len(head) < pobEntityLen && bytes.IndexByte(p, '&') < 0:
			w.tailLen = copy(w.tail[:], head)
			return n, nil
		case len(head) == pobEntityLen && head[0] == '&' && bytes.IndexByte(head[1:], '&') < 0 && entityAt(head):
			w.out = appendPoBEntities(w.out, head)
			p = p[need:]
		default:
			w.out = append(w.out, w.tail[:w.tailLen]...)
		}
		w.tailLen = 0
	}
	w.keep(p)
	return n, nil
}

func entityAt(b []byte) bool {
	for _, e := range pobEntities {
		if string(b) == e.from {
			return true
		}
	}
	return false
}

func (w *pobEntityWriter) keep(p []byte) {
	cut := len(p)
	if i := bytes.LastIndexByte(p[max(0, len(p)-pobEntityLen+1):], '&'); i >= 0 {
		cut = max(0, len(p)-pobEntityLen+1) + i
	}
	w.out = appendPoBEntities(w.out, p[:cut])
	w.tailLen = copy(w.tail[:], p[cut:])
}

func (w *pobEntityWriter) finish() []byte {
	w.out = append(w.out, w.tail[:w.tailLen]...)
	w.tailLen = 0
	return w.out
}

var NewBuildTreeVersion pob.TreeVersion = "3_18"

func EmptyBuild() *pob.PathOfBuilding {
	return &pob.PathOfBuilding{
		Build: pob.Build{
			ClassName:        pob.ClassNameByID[pob.Scion],
			Level:            1,
			TargetVersion:    "3_0",
			Bandit:           "None",
			PantheonMajorGod: "None",
			PantheonMinorGod: "None",
			MainSocketGroup:  1,
			ViewMode:         "SKILLS",
			AscendClassName:  "None",
		},
		Tree: pob.Tree{
			ActiveSpec: 1,
			Specs: []pob.Spec{
				{
					ClassID:       0,
					AscendClassID: 0,
					TreeVersion:   NewBuildTreeVersion,
				},
			},
		},
		Skills: pob.Skills{
			SortGemsByDPSField:            "CombinedDPS",
			ShowSupportGemTypes:           "ALL",
			MatchGemLevelToCharacterLevel: false,
			ActiveSkillSet:                1,
			SortGemsByDPS:                 true,
			SkillSets: []pob.SkillSet{
				{
					ID: 1,
					Skills: []pob.Skill{
						{
							Enabled: true,
						},
					},
				},
			},
		},
		Items: pob.Items{
			ActiveItemSet: 1,
			ItemSets: []pob.ItemSet{
				{
					ID:    "1",
					Slots: make([]pob.Slot, 0),
				},
			},
		},
	}
}

//pob:block:src/Classes/ConfigTab.lua:837-870,src/Classes/ConfigTab.lua:1289-1295
func resolveActiveConfigSet(build *pob.PathOfBuilding) {
	sets := build.Config.ConfigSets
	if len(sets) == 0 {
		return
	}

	if len(build.Config.Inputs)+len(build.Config.Placeholders)+len(build.Config.CustomModifierBlocks) > 0 {
		one := -1
		for i, set := range sets {
			if set.ID == 1 {
				one = i
				break
			}
		}
		if one < 0 {
			title := "Default"
			sets = append([]pob.ConfigSet{{ID: 1, Title: &title}}, sets...)
			one = 0
		}
		sets[one].Inputs = overlayInputs(sets[one].Inputs, build.Config.Inputs)
		sets[one].Placeholders = overlayInputs(sets[one].Placeholders, build.Config.Placeholders)
		sets[one].CustomModifierBlocks = append(sets[one].CustomModifierBlocks, build.Config.CustomModifierBlocks...)
		build.Config.ConfigSets = sets
	}

	want := build.Config.ActiveConfigSet
	if want == 0 {
		want = 1
	}
	active := -1
	for i, set := range sets {
		if set.ID == want {
			active = i
			break
		}
	}
	if active < 0 {
		active = 0
		build.Config.ActiveConfigSet = sets[0].ID
	}

	build.Config.Inputs = sets[active].Inputs
	build.Config.Placeholders = sets[active].Placeholders
	build.Config.CustomModifierBlocks = sets[active].CustomModifierBlocks
}

func overlayInputs(own, over []pob.Input) []pob.Input {
	out := append([]pob.Input(nil), own...)
	for _, in := range over {
		i := slices.IndexFunc(out, func(o pob.Input) bool { return o.Name == in.Name })
		if i >= 0 {
			out[i] = in
		} else {
			out = append(out, in)
		}
	}
	return out
}

func resolveSkillSources(build *pob.PathOfBuilding) {
	for i := range build.Skills.SkillSets {
		skills := build.Skills.SkillSets[i].Skills
		for j := range skills {
			if skills[j].SourceAttr != nil {
				skills[j].Source = *skills[j].SourceAttr
			}
		}
	}
}

type SavedBuild struct {
	ClassName  string
	Level      int
	Ascendancy string
	MainSkill  string
}

var nonSpaceRe = regexp.MustCompile(`\S`)

func element(raw []byte, name string) (*xml.Decoder, xml.StartElement, bool) {
	tag := []byte("<" + name)
	for from := 0; ; {
		i := bytes.Index(raw[from:], tag)
		if i < 0 {
			return nil, xml.StartElement{}, false
		}
		at := from + i
		end := at + len(tag)
		if end < len(raw) && (raw[end] == '>' || raw[end] == '/' || raw[end] == ' ' || raw[end] == '\t' || raw[end] == '\n' || raw[end] == '\r') {
			d := xml.NewDecoder(bytes.NewReader(raw[at:]))
			tok, err := d.Token()
			if start, ok := tok.(xml.StartElement); ok && err == nil {
				return d, start, true
			}
			return nil, xml.StartElement{}, false
		}
		from = end
	}
}

func SavedBuildSummary(rawXML string) (SavedBuild, error) {
	raw := stripNilAttrs([]byte(rawXML))
	var b struct {
		Build struct {
			ClassName       string `xml:"className,attr"`
			AscendClassName string `xml:"ascendClassName,attr"`
			Level           int    `xml:"level,attr"`
			MainSocketGroup *int   `xml:"mainSocketGroup,attr"`
		}
		Skills pob.Skills
	}
	_, start, ok := element(raw, "Build")
	if !ok {
		return SavedBuild{}, fmt.Errorf("build has no <Build>")
	}
	for _, a := range start.Attr {
		var err error
		switch a.Name.Local {
		case "className":
			b.Build.ClassName = a.Value
		case "ascendClassName":
			b.Build.AscendClassName = a.Value
		case "level":
			b.Build.Level, err = strconv.Atoi(strings.TrimSpace(a.Value))
		case "mainSocketGroup":
			var n int
			n, err = strconv.Atoi(strings.TrimSpace(a.Value))
			b.Build.MainSocketGroup = &n
		}
		if err != nil {
			return SavedBuild{}, fmt.Errorf("failed to parse build as xml: %s: %w", a.Name.Local, err)
		}
	}
	if _, _, ok := element(raw, "Spec"); !ok {
		return SavedBuild{}, fmt.Errorf("build has no <Spec> under <Tree>: there is no passive tree to read")
	}
	if d, start, ok := element(raw, "Skills"); ok {
		if err := d.DecodeElement(&b.Skills, &start); err != nil {
			return SavedBuild{}, fmt.Errorf("failed to parse build as xml: %w", err)
		}
	}

	out := SavedBuild{ClassName: b.Build.ClassName, Level: b.Build.Level}
	if b.Build.AscendClassName != "None" {
		out.Ascendancy = b.Build.AscendClassName
	}

	main := 1
	if b.Build.MainSocketGroup != nil {
		main = *b.Build.MainSocketGroup
	}
	sets := b.Skills.SkillSets
	if len(sets) == 0 {
		return out, nil
	}
	groups := sets[(&pob.PathOfBuilding{Skills: b.Skills}).ActiveSkillSetIndex()].Skills
	if main < 1 || main > len(groups) {
		return out, nil
	}
	group := groups[main-1]
	if nonSpaceRe.MatchString(group.Label) {
		out.MainSkill = group.Label
		return out, nil
	}
	var names []string
	for _, g := range group.Gems {
		support := strings.HasPrefix(g.SkillID, "Support") || strings.Contains(g.GemID, "/SupportGem")
		if g.Enabled && (g.GemID != "" || g.SkillID != "") && !support {
			names = append(names, g.NameSpec)
		}
	}
	out.MainSkill = strings.Join(names, ", ")
	if out.MainSkill == "" {
		out.MainSkill = "<No active skills>"
	}
	return out, nil
}

func SerializeBuildWithStats(build *pob.PathOfBuilding, player []pob.PlayerStat, fullDPS []pob.FullDPSSkill, minion []pob.PlayerStat) ([]byte, error) {
	out := *build
	out.Build.PlayerStats = player
	out.Build.FullDPSSkills = fullDPS
	out.Build.MinionStats = minion
	return SerializeBuild(&out)
}
