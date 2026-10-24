package builds

import (
	"bytes"
	"encoding/xml"
	"strings"
)

type Game int

const (
	GameUnknown Game = iota
	GamePoE1
	GamePoE2
	// GamePoE2Legacy is a PoE2 save from before PoB-PoE2 renamed its root; PoB-PoE2 refuses it.
	GamePoE2Legacy
)

// PoE2 targetVersions are 0_x, PoE1's 2_6 and 3_0.
//
//pob2:block:src/Modules/Build.lua:2700-2703,src/GameVersions.lua:4-6
func DetectGame(raw []byte) Game {
	d := xml.NewDecoder(bytes.NewReader(raw))
	for {
		tok, err := d.Token()
		if err != nil {
			return GameUnknown
		}
		start, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "PathOfBuilding2":
			return GamePoE2
		case "PathOfBuilding":
			if _, b, ok := element(raw, "Build"); ok {
				for _, a := range b.Attr {
					if a.Name.Local == "targetVersion" && strings.HasPrefix(a.Value, "0_") {
						return GamePoE2Legacy
					}
				}
			}
			return GamePoE1
		}
		return GameUnknown
	}
}
