package exposition

import (
	"github.com/Vilsol/crystalline/bind"

	"github.com/Vilsol/go-pob/data/raw"
)

//crystalline:exports
func ItemArtExports(r bind.Registry) {
	r.Type(ArtQuery{}, bind.Plain())
	r.Func(ItemArtPaths, bind.AsPromise())
}

// ArtQuery names an item (Name, Base, Unique) or, with Gem, a gem by its Name.
type ArtQuery struct {
	Name   string
	Base   string
	Unique bool
	Gem    bool
}

// ItemArtPaths answers each query with its game art path, or "" when the art is unknown.
func ItemArtPaths(queries []ArtQuery) []string {
	out := make([]string, len(queries))
	for i, q := range queries {
		if q.Gem {
			out[i] = raw.GemArtPath(q.Name)
		} else {
			out[i] = raw.ItemArtPath(q.Name, q.Base, q.Unique)
		}
	}
	return out
}
