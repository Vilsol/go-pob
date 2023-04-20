package raw

import raw2 "github.com/Vilsol/go-pob-data/raw"

type Tag struct {
	raw2.Tag
}

var Tags []*Tag

func InitializeTags(version string) error {
	return InitHelper(version, "Tags", &Tags, nil)
}
