package pob

import (
	"encoding/xml"
	"fmt"
	"strings"
)

var PartyBufferNames = []string{"PartyMemberStats", "Aura", "Curse", "Warcry Skills", "Link Skills", "EnemyConditions", "EnemyMods"}

var PartyDestinations = []string{"All", "Party Member Stats", "Aura", "Curse", "Warcry Skills", "Link Skills", "EnemyConditions", "EnemyMods"}

func partyInDestination(destination string, name string) bool {
	if destination == "All" {
		return true
	}
	if name == "PartyMemberStats" {
		return destination == "Party Member Stats"
	}
	return destination == name
}

func (b *PathOfBuilding) PartyBuffers() map[string]string {
	out := make(map[string]string, len(PartyBufferNames))
	for _, name := range PartyBufferNames {
		out[name] = ""
	}
	for _, node := range b.Party.ImportedBuffs {
		if _, ok := out[node.Name]; ok {
			out[node.Name] = strings.Trim(node.Text, NotesEdgeSpace)
		}
	}
	return out
}

func (p Party) Applied() []PartyBuffs {
	if p.applied == nil {
		return p.ImportedBuffs
	}
	out := make([]PartyBuffs, 0, len(PartyBufferNames))
	for _, name := range PartyBufferNames {
		if text := p.applied[name]; text != "" {
			out = append(out, PartyBuffs{Name: name, Text: text})
		}
	}
	return out
}

func (b *PathOfBuilding) PartyEffectsApplied() bool {
	if b.Party.applied == nil {
		return true
	}
	bufs := b.PartyBuffers()
	for _, name := range PartyBufferNames {
		if b.Party.applied[name] != bufs[name] {
			return false
		}
	}
	return true
}

func (b *PathOfBuilding) appliedCopy() map[string]string {
	src := b.Party.applied
	if src == nil {
		src = b.PartyBuffers()
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

//pob:src/Classes/PartyTab.lua:575-611
func (b *PathOfBuilding) setPartyBuffers(bufs map[string]string) {
	nodes := make([]PartyBuffs, 0, len(PartyBufferNames))
	for _, name := range PartyBufferNames {
		if bufs[name] != "" {
			nodes = append(nodes, PartyBuffs{Name: name, Text: bufs[name]})
		}
	}
	b.Party.ImportedBuffs = nodes
}

func (b *PathOfBuilding) SetPartyBuffer(name string, text string) {
	applied := b.appliedCopy()
	bufs := b.PartyBuffers()
	if _, ok := bufs[name]; !ok {
		return
	}
	bufs[name] = text
	b.setPartyBuffers(bufs)
	b.Party.applied = applied
}

func (b *PathOfBuilding) PartyOptions() (string, bool, bool) {
	destination := b.Party.Destination
	if destination == "" {
		destination = "All"
	}
	return destination, b.Party.Append == "true", b.Party.ShowAdvanceTools == "true"
}

//pob:block:src/Classes/PartyTab.lua:662-666
func (b *PathOfBuilding) SetPartyOptions(destination string, appendMode bool, showAdvanceTools bool) {
	b.Party.Destination = destination
	b.Party.Append = "false"
	if appendMode {
		b.Party.Append = "true"
	}
	b.Party.ShowAdvanceTools = "false"
	if showAdvanceTools {
		b.Party.ShowAdvanceTools = "true"
	}
}

//pob:src/Classes/PartyTab.lua:82-118
func clearPartyText(bufs map[string]string, applied map[string]string, destination string) {
	for _, name := range PartyBufferNames {
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

//pob:src/Classes/PartyTab.lua:339-344
func (b *PathOfBuilding) ClearParty(destination string) {
	applied := b.appliedCopy()
	bufs := b.PartyBuffers()
	clearPartyText(bufs, applied, destination)
	applied["EnemyConditions"] = ""
	applied["EnemyMods"] = ""
	b.setPartyBuffers(bufs)
	b.Party.applied = applied
}

//pob:src/Classes/PartyTab.lua:353-360
func (b *PathOfBuilding) DisablePartyEffects() {
	b.Party.applied = map[string]string{}
}

func (b *PathOfBuilding) PartyEffectsDisabled() bool {
	return b.Party.applied != nil && len(b.Party.applied) == 0
}

func (b *PathOfBuilding) WithoutPartyEffects() *PathOfBuilding {
	out := *b
	out.Party.applied = map[string]string{}
	return &out
}

//pob:src/Classes/PartyTab.lua:363-377
func (b *PathOfBuilding) RebuildParty() {
	b.Party.applied = nil
}

type partyExport struct {
	XMLName xml.Name
	Party   []struct {
		ExportedBuffs []struct {
			Name *string `xml:"name,attr"`
			Text string  `xml:",chardata"`
		} `xml:"ExportedBuffs"`
	} `xml:"Party"`
}

//pob:src/Classes/PartyTab.lua:162-285
func (b *PathOfBuilding) ImportParty(xmlText string, destination string, appendMode bool) error {
	applied := b.appliedCopy()
	bufs := b.PartyBuffers()
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
		b.setPartyBuffers(bufs)
		b.Party.applied = applied
	}

	var doc partyExport
	if err := xml.Unmarshal([]byte(xmlText), &doc); err != nil {
		commit()
		return fmt.Errorf("error loading the build code: %w", err)
	}
	if doc.XMLName.Local != "PathOfBuilding" {
		commit()
		return fmt.Errorf("'PathOfBuilding' root element missing")
	}
	if len(doc.Party) > 0 {
		for _, node := range doc.Party[0].ExportedBuffs {
			if node.Name == nil || !partyInDestination(destination, *node.Name) {
				continue
			}
			name := *node.Name
			text := strings.Trim(node.Text, NotesEdgeSpace)
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

//pob:block:src/Classes/ImportTab.lua:747-748
func (b *PathOfBuilding) ExportParty() bool {
	return b.Import != nil && b.Import.ExportParty
}

//pob:src/Classes/ImportTab.lua:507-510
func (b *PathOfBuilding) SetExportParty(on bool) {
	if b.Import == nil {
		b.Import = &Import{}
	}
	b.Import.ExportParty = on
	if !on {
		b.Party.ExportedBuffs = []PartyBuffs{}
	}
}

//pob:block:src/Classes/PartyTab.lua:612-653
func (b *PathOfBuilding) SetExportedBuffs(texts map[string]string) {
	b.Party.ExportedBuffs = []PartyBuffs{}
	for _, name := range []string{"PartyMemberStats", "Aura", "Curse", "Warcry Skills", "Link Skills", "EnemyConditions", "EnemyMods"} {
		if texts[name] != "" {
			b.Party.ExportedBuffs = append(b.Party.ExportedBuffs, PartyBuffs{Name: name, Text: texts[name]})
		}
	}
}
