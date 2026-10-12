package calculator

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/pob"
)

//pob:src/Classes/TreeTab.lua:588-612
func ConvertSpecToVersion(build *pob.PathOfBuilding, version pob.TreeVersion, remove bool) error {
	target, ok := data.TreeVersions[version]
	if !ok {
		return fmt.Errorf("no tree version %q", version)
	}
	i := build.ActiveSpecIndex()
	if i < 0 {
		return fmt.Errorf("the build has no tree")
	}
	tree := target.Tree()
	spec := build.Tree.Specs[i]
	selected := map[int64]bool{}
	for _, s := range pob.SpecMasterySelections(spec) {
		if s.Effect < 65536 {
			selected[s.Node] = true
		}
	}
	var nodes []string
	for _, id := range pob.SpecNodes(spec) {
		node, known := tree.Nodes[nodeKey(id)]
		if id < 65536 && (!known || (isMasteryNode(node) && !selected[id])) {
			continue
		}
		nodes = append(nodes, strconv.FormatInt(id, 10))
	}
	converted := spec
	converted.TreeVersion = version
	converted.NodesAttr = strings.Join(nodes, ",")
	converted.Sockets = slices.Clone(spec.Sockets)
	converted.Overrides = slices.Clone(spec.Overrides)
	converted.URL = ""
	build.Tree.Specs = slices.Insert(build.Tree.Specs, i+1, converted)
	build.SetActiveSpec(i + 2)
	pruneOrphansOn(build, tree)
	if remove {
		build.Tree.Specs = slices.Delete(build.Tree.Specs, i, i+1)
		build.SetActiveSpec(i + 1)
	}
	return nil
}

//pob:src/Classes/TreeTab.lua:614-627
func ConvertAllSpecsToVersion(build *pob.PathOfBuilding, version pob.TreeVersion) error {
	active := build.ActiveSpecIndex() + 1
	versions := make([]pob.TreeVersion, 0, len(build.Tree.Specs))
	for _, s := range build.Tree.Specs {
		versions = append(versions, s.TreeVersion)
	}
	for index, v := range versions {
		if v != version {
			build.SetActiveSpec(index + 1)
			if err := ConvertSpecToVersion(build, version, true); err != nil {
				return err
			}
		}
	}
	build.SetActiveSpec(active)
	return nil
}
