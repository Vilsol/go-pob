package calculator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob/cache"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/raw"
)

var enabled = false

func init() {
	config.InitLogging(false)

	if err := poe.InitializeAll(context.Background(), raw.LatestVersion, cache.Disk(), nil); err != nil {
		panic(err)
	}
}

func TestManyBuilds(t *testing.T) {
	if !enabled {
		t.SkipNow()
	}

	dir, err := os.ReadDir("../testdata/many-builds")
	if err != nil {
		panic(err)
	}

	for _, entry := range dir {
		if strings.HasSuffix(entry.Name(), ".xml") {
			t.Run(strings.TrimSuffix(entry.Name(), ".xml"), func(t *testing.T) {
				file, err := os.ReadFile(filepath.Join("../testdata/many-builds", entry.Name()))
				testza.AssertNoError(t, err)

				build, err := builds.ParseBuild(file)
				testza.AssertNoError(t, err)

				calculator := &Calculator{PoB: build}
				env := calculator.BuildOutput(OutputModeMain)

				for _, stat := range build.Build.PlayerStats {
					testza.AssertEqual(t, stat.Value, env.Player.OutputTable[OutTableMainHand][stat.Stat], stat.Stat)
				}
			})
		}
	}
}
