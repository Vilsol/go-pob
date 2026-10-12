package data

import (
	"fmt"
	"testing"

	"github.com/Vilsol/go-pob/utils"
)

// TestMonsterAllyLifeTableMatchesPinnedPoB pins every entry of `data.monsterAllyLifeTable`.
func TestMonsterAllyLifeTableMatchesPinnedPoB(t *testing.T) {
	want := []float64{0, 15, 16, 18, 20, 22, 24, 27, 29, 32, 35, 38, 41, 44, 48, 52, 56, 60, 65, 70, 75, 81, 87, 93, 99, 106, 114, 121, 130, 138, 148, 158, 168, 179, 190, 203, 215, 229, 244, 259, 275, 292, 310, 328, 348, 369, 392, 415, 440, 465, 493, 522, 552, 584, 618, 653, 691, 730, 772, 816, 862, 910, 961, 1015, 1072, 1131, 1194, 1260, 1329, 1402, 1478, 1559, 1644, 1733, 1827, 1926, 2029, 2138, 2253, 2373, 2500, 2633, 2773, 2919, 3074, 3236, 3406, 3585, 3773, 3970, 4178, 4395, 4624, 4864, 5116, 5381, 5659, 5951, 6257, 6578, 6916}

	if len(MonsterAllyLifeTable) != len(want) {
		t.Fatalf("length %d, want %d (100 Lua entries plus the leading 0 pad)",
			len(MonsterAllyLifeTable), len(want))
	}
	for i := range want {
		if MonsterAllyLifeTable[i] != want[i] {
			t.Errorf("index %d: have %g, want %g", i, MonsterAllyLifeTable[i], want[i])
		}
	}
}

// TestNonDamagingAilmentsMatchPinnedPoB pins all six rows of `data.nonDamagingAilment`.
func TestNonDamagingAilmentsMatchPinnedPoB(t *testing.T) {
	want := map[Ailment]NonDamagingAilmentData{
		AilmentChill: {
			AssociatedType: DamageTypeCold, Alt: false,
			Default: utils.Ptr[float64](10), Min: 5, Max: 30, Precision: 0, Duration: utils.Ptr[float64](2),
		},
		AilmentFreeze: {
			AssociatedType: DamageTypeCold, Alt: false,
			Default: nil, Min: 0.3, Max: 3, Precision: 2, Duration: nil,
		},
		AilmentShock: {
			AssociatedType: DamageTypeLightning, Alt: false,
			Default: utils.Ptr[float64](15), Min: 5, Max: 50, Precision: 0, Duration: utils.Ptr[float64](2),
		},
		AilmentScorch: {
			AssociatedType: DamageTypeFire, Alt: true,
			Default: utils.Ptr[float64](10), Min: 0, Max: 30, Precision: 0, Duration: utils.Ptr[float64](4),
		},
		AilmentBrittle: {
			AssociatedType: DamageTypeCold, Alt: true,
			Default: utils.Ptr[float64](2), Min: 0, Max: 6, Precision: 2, Duration: utils.Ptr[float64](4),
		},
		AilmentSap: {
			AssociatedType: DamageTypeLightning, Alt: true,
			Default: utils.Ptr[float64](6), Min: 0, Max: 20, Precision: 0, Duration: utils.Ptr[float64](4),
		},
	}

	if len(NonDamagingAilments) != len(want) {
		t.Fatalf("%d ailments, want %d", len(NonDamagingAilments), len(want))
	}

	for ailment, w := range want {
		got, ok := NonDamagingAilments[ailment]
		if !ok {
			t.Errorf("%s: missing from NonDamagingAilments", ailment)
			continue
		}
		if got.AssociatedType != w.AssociatedType || got.Alt != w.Alt ||
			!eqFloatPtr(got.Default, w.Default) || got.Min != w.Min || got.Max != w.Max ||
			got.Precision != w.Precision || !eqFloatPtr(got.Duration, w.Duration) {
			t.Errorf("%s:\n  have %s\n  want %s", ailment, showAilment(got), showAilment(w))
		}
	}
}

func eqFloatPtr(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func showAilment(a NonDamagingAilmentData) string {
	show := func(p *float64) string {
		if p == nil {
			return "nil"
		}
		return fmt.Sprintf("%g", *p)
	}
	return fmt.Sprintf("{type=%s alt=%v default=%s min=%g max=%g precision=%g duration=%s}",
		a.AssociatedType, a.Alt, show(a.Default), a.Min, a.Max, a.Precision, show(a.Duration))
}
