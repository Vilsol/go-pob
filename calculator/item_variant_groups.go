package calculator

import (
	"maps"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

var (
	versionSpecRe      = regexp.MustCompile(`\{version:([^}]*)\}`)
	groupSpecRe        = regexp.MustCompile(`\{group:([^}]*)\}`)
	variantGroupSpecRe = regexp.MustCompile(`^(\d+)\s*=\s*(\d+)$`)
)

func idSet(spec string) map[int]bool {
	out := map[int]bool{}
	for _, d := range variantDigitRe.FindAllString(spec, -1) {
		if v, err := strconv.Atoi(d); err == nil {
			out[v] = true
		}
	}
	return out
}

//pob:block:src/Classes/Item.lua:888-910
func (i *Item) registerVariantGroups(ml ModLine) {
	if ml.VariantGroupList == nil || ml.VariantList == nil {
		return
	}
	if i.VariantGroups == nil {
		i.VariantGroups = map[int]map[int]map[int]bool{}
	}
	for groupID := range ml.VariantGroupList {
		group := i.VariantGroups[groupID]
		if group == nil {
			group = map[int]map[int]bool{}
			i.VariantGroups[groupID] = group
		}
		for variantID := range ml.VariantList {
			if variantID < 1 || variantID > len(i.VariantList) {
				continue
			}
			versions := group[variantID]
			if versions == nil {
				versions = map[int]bool{}
				group[variantID] = versions
			}
			if ml.VersionList == nil {
				versions[0] = true
				continue
			}
			for versionID := range ml.VersionList {
				if versionID >= 1 && versionID <= len(i.VersionList) {
					versions[versionID] = true
				}
			}
		}
	}
}

//pob:src/Classes/Item.lua:335-339
func (i *Item) isVariantGroupOptionEligible(groupID, variantID int) bool {
	versions := i.VariantGroups[groupID][variantID]
	return versions != nil && (versions[0] || (i.SelectedVersion != nil && versions[*i.SelectedVersion]))
}

//pob:src/Classes/Item.lua:341-363
func (i *Item) VariantGroupOptions(groupID int, excludeSelected bool) []int {
	var options []int
	if i.VariantGroups[groupID] == nil {
		return options
	}
	used := map[int]bool{}
	if excludeSelected {
		for _, other := range slices.Sorted(maps.Keys(i.VariantGroups)) {
			if other == groupID {
				continue
			}
			if v, ok := i.VariantGroupSelections[other]; ok && i.isVariantGroupOptionEligible(other, v) {
				used[v] = true
			}
		}
	}
	for variantID := 1; variantID <= len(i.VariantList); variantID++ {
		if i.isVariantGroupOptionEligible(groupID, variantID) && !used[variantID] {
			options = append(options, variantID)
		}
	}
	return options
}

//pob:src/Classes/Item.lua:365-403
func (i *Item) normaliseVariantSelections() {
	if len(i.VersionList) > 0 {
		v := len(i.VersionList)
		if i.SelectedVersion != nil {
			v = min(len(i.VersionList), *i.SelectedVersion)
		}
		i.SelectedVersion = intPtr(max(1, v))
	} else {
		i.SelectedVersion = nil
	}
	if i.VariantGroupSelections == nil {
		i.VariantGroupSelections = map[int]int{}
	}
	for groupID := range i.VariantGroupSelections {
		if i.VariantGroups[groupID] == nil {
			delete(i.VariantGroupSelections, groupID)
		}
	}
	used := map[int]bool{}
	var needsSelection []int
	for _, groupID := range slices.Sorted(maps.Keys(i.VariantGroups)) {
		if len(i.VariantGroupOptions(groupID, false)) == 0 {
			continue
		}
		if selected, ok := i.VariantGroupSelections[groupID]; ok && i.isVariantGroupOptionEligible(groupID, selected) && !used[selected] {
			used[selected] = true
		} else {
			needsSelection = append(needsSelection, groupID)
		}
	}
	for _, groupID := range needsSelection {
		delete(i.VariantGroupSelections, groupID)
		for _, variantID := range i.VariantGroupOptions(groupID, false) {
			if !used[variantID] {
				i.VariantGroupSelections[groupID] = variantID
				used[variantID] = true
				break
			}
		}
	}
}

//pob:block:src/Classes/Item.lua:2082-2098
func (i *Item) checkModLineVariantGroups(ml ModLine) bool {
	if ml.VersionList != nil && (i.SelectedVersion == nil || !ml.VersionList[*i.SelectedVersion]) {
		return false
	}
	if ml.VariantGroupList != nil {
		if ml.VariantList == nil {
			return false
		}
		for groupID := range ml.VariantGroupList {
			if selected, ok := i.VariantGroupSelections[groupID]; ok && ml.VariantList[selected] {
				return true
			}
		}
		return false
	}
	return ml.VariantList == nil
}

//pob:block:src/Classes/Item.lua:1889-1907
func (i *Item) variantGroupRawLines() []string {
	var out []string
	for _, name := range i.VersionList {
		out = append(out, "Version: "+name)
	}
	if len(i.VersionList) > 0 && i.SelectedVersion != nil {
		out = append(out, "Selected Version: "+strconv.Itoa(*i.SelectedVersion))
	}
	return out
}

func (i *Item) variantGroupSelectionLines() []string {
	var out []string
	for _, groupID := range slices.Sorted(maps.Keys(i.VariantGroups)) {
		if v, ok := i.VariantGroupSelections[groupID]; ok {
			out = append(out, "Selected Variant Group: "+strconv.Itoa(groupID)+"="+strconv.Itoa(v))
		}
	}
	return out
}

//pob:block:src/Classes/ItemsTab.lua:4412-4421
func (i *Item) selectedGroupVariants() []string {
	var out []string
	for _, groupID := range slices.Sorted(maps.Keys(i.VariantGroups)) {
		if v, ok := i.VariantGroupSelections[groupID]; ok && i.isVariantGroupOptionEligible(groupID, v) {
			out = append(out, i.VariantList[v-1])
		}
	}
	return out
}

func idSpec(set map[int]bool) string {
	ids := slices.Sorted(maps.Keys(set))
	parts := make([]string, len(ids))
	for k, id := range ids {
		parts[k] = strconv.Itoa(id)
	}
	return strings.Join(parts, ",")
}
