package builds

import (
	"encoding/xml"
	"regexp"

	"github.com/pkg/errors"

	"github.com/Vilsol/go-pob/pob"
)

var nilCleanupRegex = regexp.MustCompile(`\w+?="nil"`)

func ParseBuild(rawXML []byte) (*pob.PathOfBuilding, error) {
	clean := nilCleanupRegex.ReplaceAllLiteral(rawXML, []byte{})
	var build pob.PathOfBuilding
	err := xml.Unmarshal(clean, &build)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse build as xml")
	}
	return &build, nil
}
