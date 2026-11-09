package exposition

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/calculator2"
	"github.com/Vilsol/go-pob/pob2"
)

// partyBufferNames are the party tab's text boxes by the name Save gives them.
var partyBufferNames = []string{"PartyMemberStats", "Aura", "Curse", "Warcry Skills", "Link Skills", "EnemyConditions", "EnemyMods"}

// partyMemberElem keeps the party page's imported members beside the party tab's buffers, in an element
// PoB-PoE2 does not read.
const partyMemberElem = "Member"

//pob2:block:src/Classes/PartyTab.lua:39
func partyInDestination(destination string, name string) bool {
	if destination == "All" {
		return true
	}
	if name == "PartyMemberStats" {
		return destination == "Party Member Stats"
	}
	return destination == name
}

// PartyBuffers is each party tab text box's text.
func (c *Calculator) PartyBuffers() map[string]string {
	out := make(map[string]string, len(partyBufferNames))
	for _, name := range partyBufferNames {
		out[name] = ""
	}
	for _, node := range c.build.Party.ImportedBuffs {
		if node.Name == nil {
			continue
		}
		if _, ok := out[*node.Name]; ok {
			out[*node.Name] = strings.Trim(node.Text, notesEdgeSpace)
		}
	}
	return out
}

// partyApplied is the buffers the party tab's actor was last built from: the saved ones until an edit.
func (c *Calculator) partyApplied() []calculator2.PartyBuffer {
	if c.partyAppliedText == nil {
		return calculator2.ImportedPartyBuffers(c.build.Party)
	}
	out := make([]calculator2.PartyBuffer, 0, len(partyBufferNames))
	for _, name := range partyBufferNames {
		if text := c.partyAppliedText[name]; text != "" {
			out = append(out, calculator2.PartyBuffer{Name: name, Text: text})
		}
	}
	return out
}

func (c *Calculator) partyOptions() calculator2.Options {
	return calculator2.Options{Party: c.partyApplied(), PartyApplied: true}
}

// PartyEffectsApplied is whether every text box is what the calculation reads.
func (c *Calculator) PartyEffectsApplied() bool {
	if c.partyAppliedText == nil {
		return true
	}
	bufs := c.PartyBuffers()
	for _, name := range partyBufferNames {
		if c.partyAppliedText[name] != bufs[name] {
			return false
		}
	}
	return true
}

func (c *Calculator) appliedCopy() map[string]string {
	src := c.partyAppliedText
	if src == nil {
		src = c.PartyBuffers()
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

//pob2:src/Classes/PartyTab.lua:575-611
func (c *Calculator) setPartyBuffers(bufs map[string]string) {
	nodes := make([]pob2.PartyBuffs, 0, len(partyBufferNames))
	for _, name := range partyBufferNames {
		if bufs[name] != "" {
			n := name
			nodes = append(nodes, pob2.PartyBuffs{Name: &n, Text: bufs[name]})
		}
	}
	c.build.Party.ImportedBuffs = nodes
}

// SetPartyBuffer edits one text box; the calculation keeps reading what was applied.
func (c *Calculator) SetPartyBuffer(name string, text string) {
	applied := c.appliedCopy()
	bufs := c.PartyBuffers()
	if _, ok := bufs[name]; !ok {
		return
	}
	bufs[name] = text
	c.setPartyBuffers(bufs)
	c.partyAppliedText = applied
}

// PartyOptions is the import destination, the append toggle and the advanced tools toggle.
//
//pob2:block:src/Classes/PartyTab.lua:568-570
func (c *Calculator) PartyOptions() (string, bool, bool) {
	destination := "All"
	if d := c.build.Party.Destination; d != nil && *d != "" {
		destination = *d
	}
	return destination, c.build.Party.Append, c.build.Party.ShowAdvanceTools
}

//pob2:block:src/Classes/PartyTab.lua:662-666
func (c *Calculator) SetPartyOptions(destination string, appendMode bool, showAdvanceTools bool) {
	c.build.Party.Destination = &destination
	c.build.Party.Append = appendMode
	c.build.Party.ShowAdvanceTools = showAdvanceTools
}

//pob2:src/Classes/PartyTab.lua:82-118
func clearPartyText(bufs map[string]string, applied map[string]string, destination string) {
	for _, name := range partyBufferNames {
		if !partyInDestination(destination, name) {
			continue
		}
		bufs[name] = ""
		switch name {
		case "Aura", "Curse", "Warcry Skills", "Link Skills":
			applied[name] = ""
		}
	}
}

//pob2:block:src/Classes/PartyTab.lua:339-344
func (c *Calculator) ClearParty(destination string) {
	applied := c.appliedCopy()
	bufs := c.PartyBuffers()
	clearPartyText(bufs, applied, destination)
	applied["EnemyConditions"] = ""
	applied["EnemyMods"] = ""
	c.setPartyBuffers(bufs)
	c.partyAppliedText = applied
}

//pob2:block:src/Classes/PartyTab.lua:353-360
func (c *Calculator) DisablePartyEffects() {
	c.partyAppliedText = map[string]string{}
}

func (c *Calculator) PartyEffectsDisabled() bool {
	return c.partyAppliedText != nil && len(c.partyAppliedText) == 0
}

//pob2:block:src/Classes/PartyTab.lua:363-377
func (c *Calculator) RebuildParty() {
	c.partyAppliedText = nil
}

// partyExportedBuffs is the ExportedBuffs of an import code's Party element, in file order.
func partyExportedBuffs(root *pob2.Node) []pob2.PartyBuffs {
	var out []pob2.PartyBuffs
	for _, tabNode := range root.Kids {
		if tabNode.Elem != "Party" {
			continue
		}
		for _, node := range tabNode.Kids {
			if node.Elem != "ExportedBuffs" {
				continue
			}
			var name *string
			if v, ok := node.Attrib["name"]; ok {
				name = &v
			}
			var text strings.Builder
			for _, kid := range node.Kids {
				if kid.Elem == "" {
					text.WriteString(kid.Text)
				}
			}
			out = append(out, pob2.PartyBuffs{Name: name, Text: text.String()})
		}
		break
	}
	return out
}

// partyCodeRoot is the root element of an import code's XML.
//
//pob2:block:src/Classes/PartyTab.lua:196-203
func partyCodeRoot(xmlText string) (*pob2.Node, error) {
	dbXML, err := pob2.ParseXML(xmlText)
	if err != nil {
		return nil, fmt.Errorf("error loading the build code: %w", err)
	}
	if len(dbXML.Kids) == 0 || dbXML.Kids[0].Elem != pob2.RootElem {
		return nil, fmt.Errorf("'%s' root element missing", pob2.RootElem)
	}
	return dbXML.Kids[0], nil
}

// ImportParty reads another build's exported buffs into the text boxes the destination names, appending or
// replacing, and applies them.
//
//pob2:src/Classes/PartyTab.lua:162-285
func (c *Calculator) ImportParty(xmlText string, destination string, appendMode bool) error {
	applied := c.appliedCopy()
	bufs := c.PartyBuffers()
	var currentCurseBuffer, currentLinkBuffer *string
	if !appendMode {
		clearPartyText(bufs, applied, destination)
	} else {
		if partyInDestination(destination, "Aura") {
			applied["Aura"] = ""
		}
		if partyInDestination(destination, "Curse") {
			cur := bufs["Curse"]
			currentCurseBuffer = &cur
			bufs["Curse"] = ""
			applied["Curse"] = ""
		}
		if partyInDestination(destination, "Warcry Skills") {
			applied["Warcry Skills"] = ""
		}
		if partyInDestination(destination, "Link Skills") {
			cur := bufs["Link Skills"]
			currentLinkBuffer = &cur
			bufs["Link Skills"] = ""
			applied["Link Skills"] = ""
		}
	}
	commit := func() {
		c.setPartyBuffers(bufs)
		c.partyAppliedText = applied
	}

	root, err := partyCodeRoot(xmlText)
	if err != nil {
		commit()
		return err
	}
	hasParty := false
	for _, kid := range root.Kids {
		if kid.Elem == "Party" {
			hasParty = true
			break
		}
	}
	if hasParty {
		for _, node := range partyExportedBuffs(root) {
			if node.Name == nil || !partyInDestination(destination, *node.Name) {
				continue
			}
			name := *node.Name
			if _, ok := bufs[name]; !ok {
				continue
			}
			text := strings.Trim(node.Text, notesEdgeSpace)
			if bufs[name] != "" {
				text = bufs[name] + "\n" + text
			}
			switch name {
			case "Curse":
				if currentCurseBuffer != nil && text == "--- Curse Limit ---\n1" {
					text = *currentCurseBuffer
				}
			case "Link Skills":
				if currentLinkBuffer != nil && text == "" {
					text = *currentLinkBuffer
				}
			}
			bufs[name] = text
			if name != "EnemyConditions" && name != "EnemyMods" {
				applied[name] = text
			}
		}
		if destination == "All" || destination == "EnemyConditions" || destination == "EnemyMods" {
			applied["EnemyConditions"] = bufs["EnemyConditions"]
			applied["EnemyMods"] = bufs["EnemyMods"]
		}
	}
	commit()
	return nil
}

// PartySummaries is the party tab's summary of what each text box applies.
//
//pob2:block:src/Classes/PartyTab.lua:897-980
func (c *Calculator) PartySummaries() map[string][]calculator2.PartySummaryEntry {
	return calculator2.PartySummaries(calculator2.ImportedPartyBuffers(c.build.Party))
}

// PartyImpactRow is one stat with and without the party's effects.
type PartyImpactRow struct {
	Stat    string
	With    float64
	Without float64
}

// PartyImpact calculates the build without the party's effects and pairs each stat with the main calculation's.
func (c *Calculator) PartyImpact(stats []string) ([]PartyImpactRow, error) {
	if c.env == nil {
		if _, err := c.Calculate(); err != nil {
			return nil, err
		}
	}
	t, err := loadTree()
	if err != nil {
		return nil, err
	}
	opts := calculator2.Options{Tree: t, Items: calculator2.NewItemsTab(c.build, t, nil), PartyApplied: true}
	// the export is the main calculation's, not this one's
	exported := c.build.Party.ExportedBuffs
	without, err := calculator2.BuildOutput(c.build, opts)
	c.build.Party.ExportedBuffs = exported
	if err != nil {
		return nil, err
	}
	out := make([]PartyImpactRow, 0, len(stats))
	for _, stat := range stats {
		out = append(out, PartyImpactRow{Stat: stat, With: c.env.Player.Output[stat], Without: without.Player.Output[stat]})
	}
	return out, nil
}

// PartyMember is one imported member: who it is, where its buffs went and whether they are applied.
type PartyMember struct {
	Name        string
	ClassName   string
	Ascendancy  string
	Level       int
	Destination string
	Active      bool
	Gives       []string
	buffs       []pob2.PartyBuffs
}

func memberOf(n *pob2.Node) PartyMember {
	level, _ := strconv.Atoi(n.Attrib["level"])
	m := PartyMember{
		Name:        n.Attrib["name"],
		ClassName:   n.Attrib["className"],
		Ascendancy:  n.Attrib["ascendClassName"],
		Level:       level,
		Destination: n.Attrib["destination"],
		Active:      n.Attrib["active"] == "true",
	}
	for _, kid := range n.Kids {
		if kid.Elem != "ExportedBuffs" {
			continue
		}
		name := kid.Attrib["name"]
		var text strings.Builder
		for _, t := range kid.Kids {
			if t.Elem == "" {
				text.WriteString(t.Text)
			}
		}
		m.buffs = append(m.buffs, pob2.PartyBuffs{Name: &name, Text: text.String()})
		if strings.TrimSpace(text.String()) != "" {
			m.Gives = append(m.Gives, name)
		}
	}
	return m
}

func (m PartyMember) node() *pob2.Node {
	n := &pob2.Node{Elem: partyMemberElem, Attrib: map[string]string{
		"name":            m.Name,
		"className":       m.ClassName,
		"ascendClassName": m.Ascendancy,
		"level":           strconv.Itoa(m.Level),
		"destination":     m.Destination,
		"active":          strconv.FormatBool(m.Active),
	}}
	for _, b := range m.buffs {
		name := ""
		if b.Name != nil {
			name = *b.Name
		}
		n.Kids = append(n.Kids, &pob2.Node{Elem: "ExportedBuffs", Attrib: map[string]string{"name": name}, Kids: []*pob2.Node{{Text: b.Text}}})
	}
	return n
}

// PartyMembers is the members imported on the party page, in import order.
func (c *Calculator) PartyMembers() []PartyMember {
	var out []PartyMember
	for _, kid := range c.build.Party.ExtraKids {
		if kid.Elem == partyMemberElem {
			out = append(out, memberOf(kid))
		}
	}
	return out
}

func (c *Calculator) setPartyMembers(members []PartyMember) {
	kids := make([]*pob2.Node, 0, len(c.build.Party.ExtraKids))
	for _, kid := range c.build.Party.ExtraKids {
		if kid.Elem != partyMemberElem {
			kids = append(kids, kid)
		}
	}
	for _, m := range members {
		kids = append(kids, m.node())
	}
	c.build.Party.ExtraKids = kids
}

// AddPartyMember imports a build code's party export as ImportParty does and keeps the build as a member.
func (c *Calculator) AddPartyMember(xmlText string, destination string, appendMode bool) error {
	if err := c.ImportParty(xmlText, destination, appendMode); err != nil {
		return err
	}
	root, err := partyCodeRoot(xmlText)
	if err != nil {
		return err
	}
	m := PartyMember{Destination: destination, Active: true}
	for _, kid := range root.Kids {
		if kid.Elem == "Build" {
			m.ClassName = kid.Attrib["className"]
			m.Ascendancy = kid.Attrib["ascendClassName"]
			m.Level, _ = strconv.Atoi(kid.Attrib["level"])
			break
		}
	}
	m.Name = m.Ascendancy
	if m.Name == "" || m.Name == "None" {
		m.Name = m.ClassName
	}
	m.buffs = partyExportedBuffs(root)
	for _, b := range m.buffs {
		if b.Name != nil && strings.TrimSpace(b.Text) != "" {
			m.Gives = append(m.Gives, *b.Name)
		}
	}
	members := c.PartyMembers()
	for i := range members {
		members[i].Active = members[i].Active && appendMode
	}
	c.setPartyMembers(append(members, m))
	return nil
}

func (c *Calculator) SetPartyMemberActive(i int, active bool) {
	members := c.PartyMembers()
	if i < 0 || i >= len(members) {
		return
	}
	members[i].Active = active
	c.setPartyMembers(members)
	c.rebuildPartyFromMembers(members)
}

func (c *Calculator) RemovePartyMember(i int) {
	members := c.PartyMembers()
	if i < 0 || i >= len(members) {
		return
	}
	members = append(members[:i], members[i+1:]...)
	c.setPartyMembers(members)
	c.rebuildPartyFromMembers(members)
}

func (c *Calculator) rebuildPartyFromMembers(members []PartyMember) {
	c.ClearParty("All")
	for _, m := range members {
		if !m.Active {
			continue
		}
		party := &pob2.Node{Elem: "Party"}
		for _, b := range m.buffs {
			name := ""
			if b.Name != nil {
				name = *b.Name
			}
			party.Kids = append(party.Kids, &pob2.Node{Elem: "ExportedBuffs", Attrib: map[string]string{"name": name}, Kids: []*pob2.Node{{Text: b.Text}}})
		}
		xmlText, err := pob2.ComposeXML(&pob2.Node{Elem: pob2.RootElem, Attrib: map[string]string{}, Kids: []*pob2.Node{party}})
		if err != nil {
			continue
		}
		_ = c.ImportParty(xmlText, m.Destination, true)
	}
}
