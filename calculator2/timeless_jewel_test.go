package calculator2

import (
	"os"
	"strings"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/pob2"
)

// The expected values are PoB-PoE2's for the same file loaded headless: build.spec.nodes[id].dn and sd,
// and mainEnv.modDB:Sum("BASE", nil, "Tribute"). The abyss jewel conquers the keystone, the allocated
// attribute node and a small passive in its radius; the Kalguur one only the keystone. 1207 is outside
// the radius.
func TestTimelessJewelConquersNodes(t *testing.T) {
	raw, err := os.ReadFile("testdata/timeless_abyss.xml")
	testza.AssertNoError(t, err)
	kalguur := strings.NewReplacer(
		"Undying Hate", "Heroic Tragedy",
		"Glorifying the defilement of 19567 souls in tribute to Amanamu", "Remembrancing 7699 songworthy deeds by the line of Medved",
		"Conquered by the Abyssals", "Conquered by the Kalguur",
	).Replace(string(raw))
	testza.AssertNotEqual(t, string(raw), kalguur)

	for _, tc := range []struct {
		name    string
		raw     string
		nodes   map[int]string
		tribute float64
	}{
		{"abyss", string(raw), map[int]string{
			44017: "Sacrifice of Flesh: When taking damage from Hits, 20% of Life Loss is prevented, then 150% of Life Loss prevented this way is Lost over 4 seconds",
			57703: "Strength: +5 to Strength| |+3 to Tribute",
			18073: "Tribute: +5 to Tribute",
			1207:  "Dexterity: +5 to Dexterity",
		}, 3},
		{"kalguur", kalguur, map[int]string{
			44017: "Circular Teachings: Gain no inherent bonus from Dexterity|1% increased Armour per 2 Dexterity",
			57703: "Strength: +5 to Strength",
			18073: "Attack Damage: 12% increased Attack Damage",
			1207:  "Dexterity: +5 to Dexterity",
		}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			build, err := pob2.ParseBuild([]byte(tc.raw))
			testza.AssertNoError(t, err)
			tree := loadTestTree(t)
			env, err := BuildOutput(build, Options{Tree: tree, Items: NewItemsTab(build, tree, nil)})
			testza.AssertNoError(t, err)
			for id, want := range tc.nodes {
				node := env.Spec.Nodes[id]
				testza.AssertEqual(t, want, node.Dn+": "+strings.Join(node.Sd, "|"), id)
			}
			testza.AssertEqual(t, tc.tribute, env.ModDB.Sum("BASE", nil, "Tribute"))
			testza.AssertEqual(t, 60.0, env.Player.Output["Str"])
		})
	}
}
