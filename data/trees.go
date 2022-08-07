package data

import _ "embed"

// //go:embed raw/trees/3.10.json.gz
// var tree3_10JSONGz []byte
// var tree3_10 *Tree
//
// //go:embed raw/trees/3.11.json.gz
// var tree3_11JSONGz []byte
// var tree3_11 *Tree
//
// //go:embed raw/trees/3.12.json.gz
// var tree3_12JSONGz []byte
// var tree3_12 *Tree
//
// //go:embed raw/trees/3.13.json.gz
// var tree3_13JSONGz []byte
// var tree3_13 *Tree
//
// //go:embed raw/trees/3.14.json.gz
// var tree3_14JSONGz []byte
// var tree3_14 *Tree
//
// //go:embed raw/trees/3.15.json.gz
// var tree3_15JSONGz []byte
// var tree3_15 *Tree
//
// //go:embed raw/trees/3.16.json.gz
// var tree3_16JSONGz []byte
// var tree3_16 *Tree
//
// //go:embed raw/trees/3.17.json.gz
// var tree3_17JSONGz []byte
// var tree3_17 *Tree
//
//go:embed raw/trees/3.18.json.gz
var tree3_18JSONGz []byte
var tree3_18 *Tree

func init() {
	//tree3_10 = FromJSONGz[*Tree](tree3_10JSONGz)
	//tree3_11 = FromJSONGz[*Tree](tree3_11JSONGz)
	//tree3_12 = FromJSONGz[*Tree](tree3_12JSONGz)
	//tree3_13 = FromJSONGz[*Tree](tree3_13JSONGz)
	//tree3_14 = FromJSONGz[*Tree](tree3_14JSONGz)
	//tree3_15 = FromJSONGz[*Tree](tree3_15JSONGz)
	//tree3_16 = FromJSONGz[*Tree](tree3_16JSONGz)
	//tree3_17 = FromJSONGz[*Tree](tree3_17JSONGz)
	tree3_18 = FromJSONGz[*Tree](tree3_18JSONGz)

	//TreeVersions[TreeVersion3_10] = TreeVersionData{
	//	Display: "3.10",
	//	Num:     3.10,
	//	URL:     "https://www.pathofexile.com/passive-skill-tree/3.10.0/",
	//	Tree:    tree3_10,
	//}
	//
	//TreeVersions[TreeVersion3_11] = TreeVersionData{
	//	Display: "3.11",
	//	Num:     3.11,
	//	URL:     "https://www.pathofexile.com/passive-skill-tree/3.11.0/",
	//	Tree:    tree3_11,
	//}
	//
	//TreeVersions[TreeVersion3_12] = TreeVersionData{
	//	Display: "3.12",
	//	Num:     3.12,
	//	URL:     "https://www.pathofexile.com/passive-skill-tree/3.12.0/",
	//	Tree:    tree3_12,
	//}
	//
	//TreeVersions[TreeVersion3_13] = TreeVersionData{
	//	Display: "3.13",
	//	Num:     3.13,
	//	URL:     "https://www.pathofexile.com/passive-skill-tree/3.13.0/",
	//	Tree:    tree3_13,
	//}
	//
	//TreeVersions[TreeVersion3_14] = TreeVersionData{
	//	Display: "3.14",
	//	Num:     3.14,
	//	URL:     "https://www.pathofexile.com/passive-skill-tree/3.14.0/",
	//	Tree:    tree3_14,
	//}
	//
	//TreeVersions[TreeVersion3_15] = TreeVersionData{
	//	Display: "3.15",
	//	Num:     3.15,
	//	URL:     "https://www.pathofexile.com/passive-skill-tree/3.15.0/",
	//	Tree:    tree3_15,
	//}
	//
	//TreeVersions[TreeVersion3_16] = TreeVersionData{
	//	Display: "3.16",
	//	Num:     3.16,
	//	URL:     "https://www.pathofexile.com/passive-skill-tree/3.16.0/",
	//	Tree:    tree3_16,
	//}
	//
	//TreeVersions[TreeVersion3_17] = TreeVersionData{
	//	Display: "3.17",
	//	Num:     3.17,
	//	URL:     "https://www.pathofexile.com/passive-skill-tree/3.17.0/",
	//	Tree:    tree3_17,
	//}

	TreeVersions[TreeVersion3_18] = TreeVersionData{
		Display: "3.18",
		Num:     3.18,
		URL:     "https://www.pathofexile.com/passive-skill-tree/3.18.0/",
		Tree:    tree3_18,
	}
}
