package calculator

import (
	"strings"

	raw2 "github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

//pob:block:src/Classes/SkillsTab.lua:242-288
func OptimiseSockets(build *pob.PathOfBuilding, slotName string) bool {
	id := 0
	if len(build.Items.ItemSets) > 0 {
		id = setSlotItemID(build.Items.ItemSets[activeItemSetIndex(build.Items)], slotName)
	}
	text, ok := buildItemText(build, id)
	if !ok {
		return false
	}
	item := ParseItem(text)
	if item.Base == nil || item.Base.SocketLimit == 0 {
		return false
	}

	abyssalSocketCount := 0
	for _, socket := range item.Sockets {
		if socket.Color == "A" {
			abyssalSocketCount++
		}
	}

	groupCount := 0
	var sockets []ItemSocket
	maxSockets := item.Base.SocketLimit - abyssalSocketCount
	colours := []string{"R", "G", "B"}
	if set := activeSkillSetIndex(build.Skills); set >= 0 {
		for _, group := range build.Skills.SkillSets[set].Skills {
			if group.Slot != slotName {
				continue
			}
			for _, gem := range group.Gems {
				skillID := gem.SkillID
				if skillID == "" {
					skillID = raw2.Gems[gem.GemID].GrantedEffectID
				}
				grantedEffect, ok := raw2.PoBSkills[skillID]
				if ok && maxSockets > 0 {
					gemColour := "W"
					if grantedEffect.Color >= 1 && grantedEffect.Color <= len(colours) {
						gemColour = colours[grantedEffect.Color-1]
					}
					sockets = append(sockets, ItemSocket{Color: gemColour, Group: groupCount})
					maxSockets--
				}
			}
			groupCount++
		}
	}
	for range abyssalSocketCount {
		groupCount++
		sockets = append(sockets, ItemSocket{Color: "A", Group: groupCount})
	}

	edited, err := EditItem(text, ItemEdit{Kind: "sockets", Text: socketsText(sockets)})
	if err != nil || strings.TrimSpace(edited) == "" {
		return false
	}
	return build.SetItemText(id, edited)
}
