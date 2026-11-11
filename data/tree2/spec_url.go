package tree2

import (
	"encoding/base64"
	"fmt"
	"slices"
	"strings"
)

// luaBase64Decode is runtime/lua/base64.lua's decode: characters outside the alphabet and '=' are dropped, and a
// trailing partial byte is lost.
func luaBase64Decode(data string) []byte {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var out []byte
	acc, bits := 0, 0
	for i := range len(data) {
		f := strings.IndexByte(alphabet, data[i])
		if f < 0 {
			continue
		}
		acc = acc<<6 | f
		bits += 6
		if bits >= 8 {
			bits -= 8
			out = append(out, byte(acc>>bits)) //nolint:gosec // one byte of the accumulator
			acc &= 1<<bits - 1
		}
	}
	return out
}

// treeLinkBytes is url:gsub("^.+/",""):gsub("-","+"):gsub("_","/") decoded.
func treeLinkBytes(url string) []byte {
	if i := strings.LastIndexByte(url, '/'); i > 0 {
		url = url[i+1:]
	}
	return luaBase64Decode(strings.NewReplacer("-", "+", "_", "/").Replace(url))
}

//pob2:src/Classes/PassiveSpec.lua:442-459
func (s *PassiveSpec) AllocateDecodedNodes(nodes []byte, isCluster bool, endian string) {
	for i := 0; i+1 < len(nodes); i += 2 {
		var id int
		if endian == "big" {
			id = int(nodes[i])*256 + int(nodes[i+1])
		} else {
			id = int(nodes[i]) + int(nodes[i+1])*256
		}
		if isCluster {
			id += 65536
		}
		if node, ok := s.Nodes[id]; ok {
			node.Alloc = true
			s.AllocNodes[id] = node
		}
	}
}

// AllocateMasteryEffects finds no effect: PassiveTree.lua:208 leaves tree.masteryEffects empty in PoB-PoE2, so every
// entry deallocates its node. Go must differ: a node id the tree lacks is skipped where Lua raises.
//
//pob2:src/Classes/PassiveSpec.lua:461-502
func (s *PassiveSpec) AllocateMasteryEffects(masteryEffects []byte, endian string) {
	for i := 0; i+3 < len(masteryEffects); i += 4 {
		var id int
		if endian == "big" {
			id = int(masteryEffects[i+2])*256 + int(masteryEffects[i+3])
		} else {
			id = int(masteryEffects[i]) + int(masteryEffects[i+1])*256
			if node, ok := s.Nodes[id]; ok {
				node.Alloc = true
				s.AllocNodes[id] = node
			}
		}
		delete(s.AllocNodes, id)
		if node, ok := s.Nodes[id]; ok {
			node.Alloc = false
		}
	}
}

// DecodePoePlannerTreeVersion is DecodePoePlannerURL(url, true): the link's tree version, or the error message. The
// decode proper is not ported: every version it accepts is a PoE1 tree, which PoB-PoE2 cannot load.
//
//pob2:src/Classes/PassiveSpec.lua:505-547
func DecodePoePlannerTreeVersion(url string) (string, error) {
	byteToInt := func(bytes []byte, start int) int {
		return int(bytes[start-1]) + int(bytes[start])*256
	}
	translatePoepToGggTreeVersion := func(minor int) int {
		treeVersions := map[int]int{31: 24, 29: 23, 27: 22, 26: 21, 25: 20, 24: 19, 23: 18}
		if v, ok := treeVersions[minor]; ok {
			return v
		}
		return -1
	}
	b := treeLinkBytes(url)
	if len(b) < 15 {
		return "", fmt.Errorf("Invalid tree link (unrecognised format).") //nolint:staticcheck // PoB-PoE2's message
	}
	majorVersion := byteToInt(b, 4)
	minorVersion := translatePoepToGggTreeVersion(byteToInt(b, 6))
	if minorVersion < 0 {
		return "", fmt.Errorf("Invalid tree version found in link.") //nolint:staticcheck // PoB-PoE2's message
	}
	return fmt.Sprintf("%d_%d", majorVersion, minorVersion), nil
}

//pob2:src/Classes/PassiveSpec.lua:590-633
func (s *PassiveSpec) DecodeURL(url string) error {
	b := treeLinkBytes(url)
	if len(b) < 6 {
		return fmt.Errorf("Invalid tree link (unrecognised format)") //nolint:staticcheck // PoB-PoE2's message
	}
	ver := int(b[0])*16777216 + int(b[1])*65536 + int(b[2])*256 + int(b[3])
	if ver > 6 {
		return fmt.Errorf("Invalid tree link (unknown version number '%d')", ver) //nolint:staticcheck // PoB-PoE2's message
	}
	classID := int(b[4])
	ascendancyIDs := 0
	if ver >= 4 {
		ascendancyIDs = int(b[5])
	}
	ascendClassID := ascendancyIDs & 3
	secondaryAscendClassID := (ascendancyIDs & 12) >> 2
	if _, ok := s.Tree.Classes[classID]; !ok {
		return fmt.Errorf("Invalid tree link (bad class ID '%d')", classID) //nolint:staticcheck // PoB-PoE2's message
	}
	s.ResetNodes()
	s.SelectClass(classID)
	s.SelectAscendClass(ascendClassID)
	s.SelectSecondaryAscendClass(secondaryAscendClassID)

	// b:sub(i, j) is 1-based and inclusive, a negative j counting from the end
	sub := func(i, j int) []byte {
		if j < 0 {
			j = len(b) + j + 1
		}
		i, j = max(i, 1), min(j, len(b))
		if i > j {
			return nil
		}
		return b[i-1 : j]
	}
	byteAt := func(i int) int {
		if i < 1 || i > len(b) {
			return 0
		}
		return int(b[i-1])
	}
	nodesStart := 7
	if ver >= 4 {
		nodesStart = 8
	}
	nodesEnd := -1
	if ver >= 5 {
		nodesEnd = 7 + byteAt(7)*2
	}
	s.AllocateDecodedNodes(sub(nodesStart, nodesEnd), false, "big")

	if ver < 5 {
		return nil
	}

	clusterStart := nodesEnd + 1
	clusterEnd := clusterStart + byteAt(clusterStart)*2
	s.AllocateDecodedNodes(sub(clusterStart+1, clusterEnd), true, "big")

	if ver < 6 {
		return nil
	}

	masteryStart := clusterEnd + 1
	masteryEnd := masteryStart + byteAt(masteryStart)*4
	s.AllocateMasteryEffects(sub(masteryStart+1, masteryEnd), "big")
	return nil
}

//pob2:src/Classes/PassiveSpec.lua:637-686
func (s *PassiveSpec) EncodeURL(prefix string) string {
	nodeCount := 0
	clusterCount := 0
	masteryCount := 0

	var clusterNodeIDs []byte
	var masteryNodeIDs []byte

	encodeNodeIDList := make([]int, 0, len(s.AllocNodes))
	for id := range s.AllocNodes {
		encodeNodeIDList = append(encodeNodeIDList, id)
	}
	slices.Sort(encodeNodeIDList)
	var nodes []byte
	for _, id := range encodeNodeIDList {
		node := s.AllocNodes[id]
		if node.Type != "ClassStart" && node.Type != "AscendClassStart" && id < 65536 && nodeCount < 255 {
			nodes = append(nodes, byte(id/256), byte(id%256)) //nolint:gosec // id is below 65536
			nodeCount++
			if effectID, ok := s.MasterySelections[id]; ok {
				masteryNodeIDs = append(masteryNodeIDs, byte(effectID/256), byte(effectID%256), byte(id/256), byte(id%256)) //nolint:gosec // ids below 65536
				masteryCount++
			}
		} else if id >= 65536 {
			clusterID := id - 65536
			clusterNodeIDs = append(clusterNodeIDs, byte(clusterID/256), byte(clusterID%256)) //nolint:gosec // cluster ids below 65536
			clusterCount++
		}
	}
	a := make([]byte, 0, 9+len(nodes)+len(clusterNodeIDs)+len(masteryNodeIDs))
	// curSecondaryAscendClassId stays nil: the PoE2 tree has no alternate_ascendancies
	a = append(a, 0, 0, 0, 6, byte(s.CurClassID), byte(s.CurAscendClassID)) //nolint:gosec // class ids are below 256
	a = append(a, byte(nodeCount))                                          //nolint:gosec // at most 255
	a = append(a, nodes...)

	a = append(a, byte(clusterCount)) //nolint:gosec // the PoE2 tree has no cluster nodes
	a = append(a, clusterNodeIDs...)

	a = append(a, byte(masteryCount)) //nolint:gosec // the PoE2 tree has no mastery effects
	a = append(a, masteryNodeIDs...)

	return prefix + base64.URLEncoding.EncodeToString(a)
}
