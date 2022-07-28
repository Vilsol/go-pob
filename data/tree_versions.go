package data

type TreeVersion string

const (
	TreeVersion3_10 = TreeVersion("3_10")
	TreeVersion3_11 = TreeVersion("3_11")
	TreeVersion3_12 = TreeVersion("3_12")
	TreeVersion3_13 = TreeVersion("3_13")
	TreeVersion3_14 = TreeVersion("3_14")
	TreeVersion3_15 = TreeVersion("3_15")
	TreeVersion3_16 = TreeVersion("3_16")
	TreeVersion3_17 = TreeVersion("3_17")
	TreeVersion3_18 = TreeVersion("3_18")
)

const LatestTreeVersion = TreeVersion3_18
const DefaultTreeVersion = TreeVersion3_10

type TreeVersionData struct {
	Display string
	Num     float64
	URL     string
	Tree    *Tree
}

var TreeVersions = make(map[TreeVersion]TreeVersionData)
