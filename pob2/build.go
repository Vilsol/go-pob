package pob2

import (
	"fmt"
	"strconv"
)

const RootElem = "PathOfBuilding2"

//pob2:src/Modules/Build.lua:2691-2729
func ParseBuild(xmlText []byte) (*PathOfBuilding2, error) {
	dbXML, err := ParseXML(string(xmlText))
	if err != nil {
		return nil, err
	}
	if len(dbXML.Kids) == 0 || dbXML.Kids[0].Elem != RootElem {
		return nil, fmt.Errorf("'%s' root element missing", RootElem)
	}
	var build PathOfBuilding2
	if err := decodeNode(dbXML.Kids[0], &build); err != nil {
		return nil, fmt.Errorf("failed to parse build: %w", err)
	}
	return &build, nil
}

//pob2:src/Modules/Build.lua:2746-2770
func SerializeBuild(build *PathOfBuilding2) ([]byte, error) {
	node, err := encodeNode(RootElem, build)
	if err != nil {
		return nil, err
	}
	text, err := ComposeXML(node)
	if err != nil {
		return nil, fmt.Errorf("error saving build: %w", err)
	}
	return []byte(text), nil
}

// NodeIDs reads a comma-joined node list the way Load walks it, with gmatch("%d+").
//
//pob2:block:src/Classes/PassiveSpec.lua:215-217
func NodeIDs(list string) []int {
	var out []int
	for i := 0; i < len(list); {
		if list[i] < '0' || list[i] > '9' {
			i++
			continue
		}
		j := i
		for j < len(list) && list[j] >= '0' && list[j] <= '9' {
			j++
		}
		if n, err := strconv.Atoi(list[i:j]); err == nil {
			out = append(out, n)
		}
		i = j
	}
	return out
}
