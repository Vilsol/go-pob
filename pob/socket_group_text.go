package pob

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	pasteLabelRe = regexp.MustCompile(`Label: ([^\x00-\x1f\x7f]+)`)
	pasteSlotRe  = regexp.MustCompile(`Slot: ([^\x00-\x1f\x7f]+)`)
	pasteGemRe   = regexp.MustCompile(`([ A-Za-z']+) (\d+)/(\d+) ?([A-Za-z]*) (\d+)`)
	sanitiseTags = regexp.MustCompile(`<[^<>]*>`)
)

//pob:src/Classes/SkillsTab.lua:667-679
func (b *PathOfBuilding) CopySocketGroup(socketGroup int) string {
	set := b.Skills.SkillSets[b.ActiveSkillSetIndex()]
	if socketGroup < 0 || socketGroup >= len(set.Skills) {
		return ""
	}
	group := set.Skills[socketGroup]
	skillText := ""
	if strings.TrimSpace(group.Label) != "" {
		skillText += "Label: " + group.Label + "\r\n"
	}
	if group.Slot != "" {
		skillText += "Slot: " + group.Slot + "\r\n"
	}
	for _, gem := range group.Gems {
		state := "DISABLED"
		if gem.Enabled {
			state = ""
		}
		count := 1
		if gem.Count != nil {
			count = *gem.Count
		}
		skillText += fmt.Sprintf("%s %d/%d %s %d\r\n", gem.NameSpec, gem.Level, gem.Quality, state, count)
	}
	return skillText
}

//pob:src/Classes/SkillsTab.lua:681-713
func (b *PathOfBuilding) PasteSocketGroup(text string) int {
	skillText := sanitiseText(text)
	newGroup := Skill{Enabled: true}
	if m := pasteLabelRe.FindStringSubmatch(skillText); m != nil {
		newGroup.Label = m[1]
	}
	if m := pasteSlotRe.FindStringSubmatch(skillText); m != nil {
		newGroup.Slot = m[1]
	}
	for _, m := range pasteGemRe.FindAllStringSubmatch(skillText, -1) {
		level, err := strconv.Atoi(m[2])
		if err != nil {
			level = 20
		}
		quality, _ := strconv.Atoi(m[3])
		count, err := strconv.Atoi(m[5])
		if err != nil {
			count = 1
		}
		newGroup.Gems = append(newGroup.Gems, Gem{
			NameSpec:      m[1],
			Level:         level,
			Quality:       quality,
			Enabled:       m[4] != "DISABLED",
			Count:         &count,
			EnableGlobal1: true,
			EnableGlobal2: true,
		})
	}
	if len(newGroup.Gems) == 0 {
		return -1
	}
	set := &b.Skills.SkillSets[b.ActiveSkillSetIndex()]
	set.Skills = append(set.Skills, newGroup)
	return len(set.Skills) - 1
}

//pob:src/Modules/Common.lua:251-274
func sanitiseText(text string) string {
	if !strings.ContainsRune(text, '<') && !hasHighByte(text) {
		return text
	}
	text = sanitiseTags.ReplaceAllString(text, "")
	for _, r := range [][2]string{
		{"\u2010", "-"}, {"\u2011", "-"}, {"\u2012", "-"}, {"\u2013", "-"}, {"\u2014", "-"}, {"\u2015", "-"},
		{"\u2212", "-"}, {"\u00e4", "a"}, {"\u00f6", "o"},
		{"\x96", "-"}, {"\x97", "-"}, {"\xe4", "a"}, {"\xf6", "o"},
	} {
		text = strings.ReplaceAll(text, r[0], r[1])
	}
	out := []byte(text)
	for i, c := range out {
		if c >= 0x80 {
			out[i] = '?'
		}
	}
	return string(out)
}

func SanitiseText(text string) string {
	return sanitiseText(text)
}

func hasHighByte(text string) bool {
	for i := 0; i < len(text); i++ {
		if text[i] >= 0x80 {
			return true
		}
	}
	return false
}

//pob:block:src/Modules/Build.lua:1469-1474
func (b *PathOfBuilding) SetSpectres(ids []string) {
	b.Build.Spectres = make([]Spectre, 0, len(ids))
	for _, id := range ids {
		b.Build.Spectres = append(b.Build.Spectres, Spectre{ID: id})
	}
}

func (b *PathOfBuilding) SpectreIDs() []string {
	ids := make([]string, 0, len(b.Build.Spectres))
	for _, s := range b.Build.Spectres {
		ids = append(ids, s.ID)
	}
	return ids
}
