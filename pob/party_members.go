package pob

import (
	"encoding/xml"
	"fmt"
)

func (b *PathOfBuilding) PartyMembers() []PartyMember {
	return b.Party.Members
}

func (b *PathOfBuilding) AddPartyMember(xmlText string, destination string, appendMode bool) error {
	if err := b.ImportParty(xmlText, destination, appendMode); err != nil {
		return err
	}
	var doc struct {
		Build struct {
			ClassName  string `xml:"className,attr"`
			Ascendancy string `xml:"ascendClassName,attr"`
			Level      int    `xml:"level,attr"`
		} `xml:"Build"`
		Party []struct {
			ExportedBuffs []PartyBuffs `xml:"ExportedBuffs"`
		} `xml:"Party"`
	}
	if err := xml.Unmarshal([]byte(xmlText), &doc); err != nil {
		return fmt.Errorf("error loading the build code: %w", err)
	}
	m := PartyMember{
		Name:        doc.Build.Ascendancy,
		ClassName:   doc.Build.ClassName,
		Ascendancy:  doc.Build.Ascendancy,
		Level:       doc.Build.Level,
		Destination: destination,
		Active:      true,
	}
	if m.Name == "" || m.Name == "None" {
		m.Name = m.ClassName
	}
	if len(doc.Party) > 0 {
		m.Buffs = doc.Party[0].ExportedBuffs
	}
	members := make([]PartyMember, 0, len(b.Party.Members)+1)
	for _, old := range b.Party.Members {
		old.Active = old.Active && appendMode
		members = append(members, old)
	}
	b.Party.Members = append(members, m)
	return nil
}

func (b *PathOfBuilding) SetPartyMemberActive(i int, active bool) {
	if i < 0 || i >= len(b.Party.Members) {
		return
	}
	members := append([]PartyMember(nil), b.Party.Members...)
	members[i].Active = active
	b.Party.Members = members
	b.rebuildPartyFromMembers()
}

func (b *PathOfBuilding) RemovePartyMember(i int) {
	if i < 0 || i >= len(b.Party.Members) {
		return
	}
	members := append([]PartyMember(nil), b.Party.Members[:i]...)
	b.Party.Members = append(members, b.Party.Members[i+1:]...)
	b.rebuildPartyFromMembers()
}

func (b *PathOfBuilding) rebuildPartyFromMembers() {
	b.ClearParty("All")
	for _, m := range b.Party.Members {
		if !m.Active {
			continue
		}
		doc := struct {
			XMLName xml.Name `xml:"PathOfBuilding"`
			Party   struct {
				Buffs []PartyBuffs `xml:"ExportedBuffs"`
			} `xml:"Party"`
		}{}
		doc.Party.Buffs = m.Buffs
		out, err := xml.Marshal(doc)
		if err != nil {
			continue
		}
		_ = b.ImportParty(string(out), m.Destination, true)
	}
}
