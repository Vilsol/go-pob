package calculator

import (
	"math"
	"os"
	"testing"
)

// TestBrandTotalDotScalesByAttachedBrands pins CalcOffence.lua:5787-5789 on the Wintertide Brand builds whose main part is "Manual Stages".
func TestBrandTotalDotScalesByAttachedBrands(t *testing.T) {
	if os.Getenv("CORPUS") != "true" {
		t.Skip("set CORPUS=true")
	}
	requireGameData(t, "3.29")
	for build, want := range map[string]float64{
		"0115-elementalist-wintertide-brand-4.xml": 1425079.9538053,
		"0398-hierophant-wintertide-brand.xml":     1140434.4249364,
	} {
		env := corpusEnvForLoadedData(t, build)
		if got := env.Player.Output["TotalDot"]; math.Abs(got-want) > want*1e-3 {
			t.Errorf("%s TotalDot = %v, PoB has %v", build, got, want)
		}
	}
}
