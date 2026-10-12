package pob

import (
	"regexp"
	"slices"
	"strconv"
)

var classAscendancies = map[ClassID][]AscendancyName{
	Scion:    {"Ascendant", "Reliquarian", "Luminary"},
	Marauder: {"Juggernaut", "Berserker", "Chieftain"},
	Ranger:   {"Warden", "Deadeye", "Pathfinder"},
	Witch:    {"Occultist", "Elementalist", "Necromancer"},
	Duelist:  {"Slayer", "Gladiator", "Champion"},
	Templar:  {"Inquisitor", "Hierophant", "Guardian"},
	Shadow:   {"Assassin", "Trickster", "Saboteur"},
}

func ClassAscendancyNames() map[ClassID][]AscendancyName {
	return classAscendancies
}

func ClassIDByName(name ClassName) (ClassID, bool) {
	for id, n := range ClassNameByID {
		if n == name {
			return id, true
		}
	}
	return 0, false
}

func AscendancyIndex(classID ClassID, name AscendancyName) AscendancyID {
	if i := slices.Index(classAscendancies[classID], name); i >= 0 {
		return AscendancyID(i + 1)
	}
	return 0
}

func AscendancyNameOf(classID ClassID, ascendClassID AscendancyID) AscendancyName {
	list := classAscendancies[classID]
	if ascendClassID < 1 || int(ascendClassID) > len(list) {
		return "None"
	}
	return list[ascendClassID-1]
}

func (b *PathOfBuilding) ActiveSpecIndex() int {
	if len(b.Tree.Specs) == 0 {
		return -1
	}
	return min(max(b.Tree.ActiveSpec, 1), len(b.Tree.Specs)) - 1
}

func (b *PathOfBuilding) activeSpec() *Spec {
	if i := b.ActiveSpecIndex(); i >= 0 {
		return &b.Tree.Specs[i]
	}
	return nil
}

var specNodeRe = regexp.MustCompile(`\d+`)

//pob:block:src/Classes/PassiveSpec.lua:138-141
func SpecNodes(spec Spec) []int64 {
	out := make([]int64, 0, 100)
	for _, s := range specNodeRe.FindAllString(spec.NodesAttr, -1) {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			out = append(out, n)
		}
	}
	return out
}

//pob:block:src/Classes/TreeTab.lua:489
func SpecTitle(spec Spec) string {
	if spec.Title == nil {
		return "Default"
	}
	return *spec.Title
}

//pob:src/Classes/TreeTab.lua:543-579
func (b *PathOfBuilding) SetActiveSpec(specID int) {
	if len(b.Tree.Specs) == 0 {
		return
	}
	b.Tree.ActiveSpec = max(min(specID, len(b.Tree.Specs)), 1)
	spec := b.Tree.Specs[b.Tree.ActiveSpec-1]
	b.Build.PassiveNodes = SpecNodes(spec)
	b.Build.ClassName = ClassNameByID[spec.ClassID]
	b.Build.AscendClassName = AscendancyNameOf(spec.ClassID, spec.AscendClassID)
}

//pob:block:src/Classes/PassiveSpecListControl.lua:52-53
func (b *PathOfBuilding) RenameSpec(specID int, title string) {
	if specID < 1 || specID > len(b.Tree.Specs) {
		return
	}
	b.Tree.Specs[specID-1].Title = &title
}

//pob:block:src/Classes/PassiveSpecListControl.lua:13-20,src/Classes/PassiveSpecListControl.lua:52-59
func (b *PathOfBuilding) CopySpec(specID int, title string) {
	if specID < 1 || specID > len(b.Tree.Specs) {
		return
	}
	from := b.Tree.Specs[specID-1]
	two := 2
	spec := Spec{
		ClassID:                  from.ClassID,
		AscendClassID:            from.AscendClassID,
		SecondaryAscendClassID:   from.SecondaryAscendClassID,
		ClusterHashFormatVersion: &two,
		Title:                    &title,
		TreeVersion:              from.TreeVersion,
		NodesAttr:                from.NodesAttr,
		MasteryEffects:           from.MasteryEffects,
		URL:                      from.URL,
		Sockets:                  slices.Clone(from.Sockets),
		Overrides:                slices.Clone(from.Overrides),
	}
	if spec.Sockets == nil {
		spec.Sockets = []Socket{}
	}
	if spec.Overrides == nil {
		spec.Overrides = []Override{}
	}
	b.Tree.Specs = append(b.Tree.Specs, spec)
}

//pob:block:src/Classes/PassiveSpecListControl.lua:55-59
func (b *PathOfBuilding) AddSpec(spec Spec) {
	b.Tree.Specs = append(b.Tree.Specs, spec)
}

//pob:block:src/Classes/PassiveSpecListControl.lua:96-105
func (b *PathOfBuilding) DeleteSpec(specID int) {
	if len(b.Tree.Specs) <= 1 || specID < 1 || specID > len(b.Tree.Specs) {
		return
	}
	active := b.ActiveSpecIndex() + 1
	b.Tree.Specs = slices.Delete(b.Tree.Specs, specID-1, specID)
	switch {
	case specID == active:
		b.SetActiveSpec(max(1, specID-1))
	case specID < active:
		b.Tree.ActiveSpec = active - 1
	}
}

//pob:block:src/Classes/ItemSlotControl.lua:62-64,src/Classes/PassiveSpec.lua:215-221
func (b *PathOfBuilding) SocketJewel(nodeID int, itemID int) {
	spec := b.activeSpec()
	if spec == nil {
		return
	}
	spec.Sockets = slices.DeleteFunc(spec.Sockets, func(s Socket) bool { return s.NodeID == nodeID })
	if itemID > 0 {
		spec.Sockets = append(spec.Sockets, Socket{NodeID: nodeID, ItemID: itemID})
	}
}

func (b *PathOfBuilding) SocketedJewel(nodeID int) int {
	if spec := b.activeSpec(); spec != nil {
		for _, s := range spec.Sockets {
			if s.NodeID == nodeID {
				return s.ItemID
			}
		}
	}
	return 0
}

//pob:src/Classes/CompareTab.lua:1331-1368
func (b *PathOfBuilding) CopyCompareSpec(from *PathOfBuilding, andUse bool) {
	i := from.ActiveSpecIndex()
	if i < 0 {
		return
	}
	src := from.Tree.Specs[i]
	title := SpecTitle(src) + " (Compared)"
	spec := Spec{
		ClassID:                  src.ClassID,
		AscendClassID:            src.AscendClassID,
		SecondaryAscendClassID:   src.SecondaryAscendClassID,
		ClusterHashFormatVersion: src.ClusterHashFormatVersion,
		Title:                    &title,
		TreeVersion:              src.TreeVersion,
		NodesAttr:                src.NodesAttr,
		MasteryEffects:           src.MasteryEffects,
		Sockets:                  []Socket{},
		Overrides:                slices.Clone(src.Overrides),
	}
	if spec.Overrides == nil {
		spec.Overrides = []Override{}
	}
	b.Tree.Specs = append(b.Tree.Specs, spec)
	if andUse {
		b.SetActiveSpec(len(b.Tree.Specs))
	}
}
