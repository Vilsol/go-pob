package builds

import (
	"encoding/xml"
	"fmt"
	"regexp"

	"github.com/Vilsol/go-pob/pob"
)

var nilCleanupRegex = regexp.MustCompile(`\w+?="nil"`)

func ParseBuildStr(rawXML string) (*pob.PathOfBuilding, error) {
	return ParseBuild([]byte(rawXML))
}

func ParseBuild(rawXML []byte) (*pob.PathOfBuilding, error) {
	clean := nilCleanupRegex.ReplaceAllLiteral(rawXML, []byte{})
	var build pob.PathOfBuilding
	err := xml.Unmarshal(clean, &build)
	if err != nil {
		return nil, fmt.Errorf("failed to parse build as xml: %w", err)
	}
	return &build, nil
}
