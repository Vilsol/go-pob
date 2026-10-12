package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/storage"
)

const benchMagicItem = `Rarity: MAGIC
Wide Silver Flask of the Eagle
Unique ID: a9954821c5ebd9324adfad1f081a3bce7c5ac5880bdd49119467f6877886e39f
Item Level: 85
Quality: 0
LevelReq: 49
Implicits: 0
+16 to Maximum Charges
12% increased Attack Speed during Flask effect`

func BenchmarkParseMagicItem(b *testing.B) {
	if err := poe.InitializeAll(raw.DataContext(b.Context()), raw.LatestVersion, raw.AssetLoaderWrapper{Storage: storage.Get()}, nil); err != nil {
		b.Fatal(err)
	}

	ItemBases()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		if item := ParseItem(benchMagicItem); item.BaseName != "Silver Flask" {
			b.Fatalf("unexpected base name %q", item.BaseName)
		}
	}
}

// BenchmarkParseBuildItems parses every item of the largest build, what opening its Items page costs.
func BenchmarkParseBuildItems(b *testing.B) {
	initBenchData(b)
	build := loadHeavyBuild(b, "0001-necromancer-unknown.xml")
	ItemBases()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, it := range build.Items.Items {
			ParseItem(it.Text)
		}
	}
}
