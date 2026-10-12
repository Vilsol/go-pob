package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/pob"
)

// TestSkillDistanceFallsBackToThePlaceholder pins CalcActiveSkill.lua:446-450.
func TestSkillDistanceFallsBackToThePlaceholder(t *testing.T) {
	cases := []struct {
		name        string
		option      string
		placeholder float64
		inputs      []pob.Input
		places      []pob.Input
		want        float64
	}{
		{"melee absent everywhere takes the DECLARED DEFAULT", "meleeDistance", 15, nil, nil, 15},
		{"projectile absent everywhere takes the DECLARED DEFAULT", "projectileDistance", 40, nil, nil, 40},
		{
			"an XML placeholder beats the declared default",
			"meleeDistance", 15, nil,
			[]pob.Input{{Name: "meleeDistance", Number: floatPtr(22)}},
			22,
		},
		{
			"an explicit zero WINS and does not fall through",
			"meleeDistance", 15,
			[]pob.Input{{Name: "meleeDistance", Number: floatPtr(0)}}, nil,
			0,
		},
		{
			"a real value wins",
			"meleeDistance", 15,
			[]pob.Input{{Name: "meleeDistance", Number: floatPtr(7)}}, nil,
			7,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := &pob.PathOfBuilding{Config: pob.Config{Inputs: c.inputs, Placeholders: c.places}}
			if got := skillDistanceOption(b, c.option, c.placeholder); got != c.want {
				t.Fatalf("%s: got %v, want %v", c.option, got, c.want)
			}
		})
	}
}

func floatPtr(f float64) *float64 { return &f }
