package exposition

// ExportParty is the Import tab's "Export Support" check box.
//
//pob2:block:src/Classes/ImportTab.lua:563-564
func (c *Calculator) ExportParty() bool {
	return c.build.Import.ExportParty
}

//pob2:block:src/Classes/ImportTab.lua:138-141
func (c *Calculator) SetExportParty(on bool) {
	c.build.Import.ExportParty = on
}

// PartyExports is the export the last calculation wrote, by the name PartyTab:Save gives each part; empty while
// Export Support is off.
func (c *Calculator) PartyExports() map[string]string {
	out := make(map[string]string, len(c.build.Party.ExportedBuffs))
	for _, b := range c.build.Party.ExportedBuffs {
		if b.Name != nil {
			out[*b.Name] = b.Text
		}
	}
	return out
}
