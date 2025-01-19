package calculator

import (
	"context"
	"os"
	"testing"

	"github.com/MarvinJWendt/testza"

	"github.com/Vilsol/go-pob-data/poe"
	"github.com/Vilsol/go-pob/cache"

	"github.com/Vilsol/go-pob/builds"
	"github.com/Vilsol/go-pob/config"
	"github.com/Vilsol/go-pob/data/raw"
	"github.com/Vilsol/go-pob/pob"
)

func init() {
	config.InitLogging(false)

	if err := poe.InitializeAll(context.Background(), raw.LatestVersion, cache.Disk(), nil); err != nil {
		panic(err)
	}
}

type testdata struct {
	name        string
	buildData   string
	baseDamage  map[OutTable]map[string]float64
	skillDamage []skillGroup
}

type skillGroup struct {
	name        string
	socketGroup int
	damage      map[string]float64
}

// These two functions check for partial map equality - only keys present in the expected param will be checked.
func assertMapEqual[M ~map[K]V, K comparable, V any](t *testing.T, expected, got M) {
	for calc, want := range expected {
		testza.AssertEqual(t, want, got[calc], calc)
	}
}

func assertNestedMapEqual[M ~map[K]map[L]V, K, L comparable, V any](t *testing.T, expected, got M) {
	for k := range expected {
		assertMapEqual(t, expected[k], got[k])
	}
}

func TestOutput(t *testing.T) {
	tc := []testdata{
		{
			name:      "Fireball",
			buildData: "../testdata/builds/Fireball.xml",
			baseDamage: map[OutTable]map[string]float64{
				OutTableMainHand: {
					"TotalMin":      0.9523809523809523,
					"TotalMax":      2.8571428571428568,
					"AverageHit":    1.9047619047619047,
					"AverageDamage": 1.8857142857142855,
					"TotalDPS":      2.2628571428571425,
				},
			},
			skillDamage: []skillGroup{
				{
					name:        "Fireball level 1",
					socketGroup: 2,
					damage: map[string]float64{
						"TotalMin":      9,
						"TotalMax":      14,
						"AverageHit":    11.844999999999999,
						"AverageDamage": 11.845,
						"TotalDPS":      15.793333333333333,
					},
				},
				{
					name:        "Fireball level 20",
					socketGroup: 3,
					damage: map[string]float64{
						"TotalMin":      1640,
						"TotalMax":      2460,
						"AverageHit":    2111.5,
						"AverageDamage": 2111.5,
						"TotalDPS":      2815.333333333333,
					},
				},
				{
					name:        "Fireball level 1 Added Cold level 1",
					socketGroup: 4,
					damage: map[string]float64{
						"TotalMin":      24,
						"TotalMax":      36,
						"AverageHit":    30.9,
						"AverageDamage": 30.9,
						"TotalDPS":      41.199999999999996,
					},
				},
				{
					name:        "Fireball level 20 Added Cold level 1",
					socketGroup: 5,
					damage: map[string]float64{
						"TotalMin":      1655,
						"TotalMax":      2482,
						"AverageHit":    2130.555,
						"AverageDamage": 2130.555,
						"TotalDPS":      2840.74,
					},
				},
				{
					name:        "Fireball level 20 Added Cold level 20",
					socketGroup: 6,
					damage: map[string]float64{
						"TotalMin":      2202,
						"TotalMax":      3304,
						"AverageHit":    2835.5899999999997,
						"AverageDamage": 2835.5899999999992,
						"TotalDPS":      3780.7866666666655,
					},
				},
			},
		},
	}

	for _, test := range tc {
		t.Run(test.name, func(t *testing.T) {
			d, err := os.ReadFile("../testdata/builds/Fireball.xml")
			if err != nil {
				t.Fatal(err)
			}
			build, err := builds.ParseBuild(d)
			if err != nil {
				t.Fatal(err)
			}

			t.Run("Unarmed", func(t *testing.T) {
				// Test without skills
				if test.baseDamage != nil {
					skills := build.Skills.SkillSets
					build.Skills.SkillSets = []pob.SkillSet{}
					env := NewCalculator(*build).BuildOutput(OutputModeMain)
					assertNestedMapEqual(t, test.baseDamage, env.Player.OutputTable)
					build.Skills.SkillSets = skills
				}
			})

			for _, sg := range test.skillDamage {
				t.Run(sg.name, func(t *testing.T) {
					sgbuild := build.WithMainSocketGroup(sg.socketGroup)
					env := NewCalculator(*sgbuild).BuildOutput(OutputModeMain)
					assertMapEqual(t, sg.damage, env.Player.Output)
				})
			}
		})
	}
}
