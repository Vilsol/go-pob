package utils

import (
	"testing"
)

type foldTarget struct {
	RadiusExtra          float64
	AuraCannotAffectSelf bool
	MinionLevel          float64
	ManaReservationFlat  *float64
	Name                 string
}

func TestSetResolvesLowercaseFirstKey(t *testing.T) {
	tgt := &foldTarget{}

	Set(tgt, "radiusExtra", float64(12))
	if tgt.RadiusExtra != 12 {
		t.Errorf("radiusExtra: got %v, want 12", tgt.RadiusExtra)
	}

	Set(tgt, "auraCannotAffectSelf", float64(1))
	if !tgt.AuraCannotAffectSelf {
		t.Errorf("auraCannotAffectSelf: got false, want true")
	}

	Set(tgt, "minionLevel", float64(20))
	if tgt.MinionLevel != 20 {
		t.Errorf("minionLevel: got %v, want 20", tgt.MinionLevel)
	}
}

func TestSetStillPrefersExactKey(t *testing.T) {
	tgt := &foldTarget{}
	Set(tgt, "RadiusExtra", float64(7))
	if tgt.RadiusExtra != 7 {
		t.Errorf("RadiusExtra: got %v, want 7", tgt.RadiusExtra)
	}
}

func TestGetOrResolvesLowercaseFirstKey(t *testing.T) {
	tgt := &foldTarget{RadiusExtra: 5}
	if got := GetOr(tgt, "radiusExtra", float64(0)); got != 5 {
		t.Errorf("radiusExtra: got %v, want 5", got)
	}
}

// TestGetOrDereferencesPointerField: PoB reads grantedEffectLevel.manaReservationFlat as a plain number (CalcOffence.lua:1223).
func TestGetOrDereferencesPointerField(t *testing.T) {
	v := 42.0
	tgt := &foldTarget{ManaReservationFlat: &v}
	if got := GetOr(tgt, "manaReservationFlat", float64(0)); got != 42 {
		t.Errorf("manaReservationFlat: got %v, want 42", got)
	}

	if got := GetOr(&foldTarget{RadiusExtra: 1}, "manaReservationFlat", float64(0)); got != 0 {
		t.Errorf("nil pointer field: got %v, want 0", got)
	}
}

// TestSetAndGetOrDoNotPanicOnTypeMismatch: A resolved field whose type cannot hold the value must be counted and skipped.
func TestSetAndGetOrDoNotPanicOnTypeMismatch(t *testing.T) {
	ResetDroppedKeys()
	defer ResetDroppedKeys()

	tgt := &foldTarget{Name: "x"}
	Set(tgt, "name", float64(3))
	if tgt.Name != "x" {
		t.Errorf("Name was overwritten with a float: %q", tgt.Name)
	}

	if got := GetOr(tgt, "name", float64(0)); got != 0 {
		t.Errorf("mismatched read: got %v, want 0", got)
	}

	total, _ := DroppedKeys()
	if total != 2 {
		t.Errorf("dropped count: got %d, want 2", total)
	}
}

func TestDroppedKeysCountsUnresolvableKeys(t *testing.T) {
	ResetDroppedKeys()
	defer ResetDroppedKeys()

	tgt := &foldTarget{RadiusExtra: 1}
	Set(tgt, "noSuchField", float64(1))
	Set(tgt, "noSuchField", float64(1))
	_ = GetOr(tgt, "alsoMissing", float64(0))

	total, byKey := DroppedKeys()
	if total != 3 {
		t.Errorf("total: got %d, want 3", total)
	}
	if len(byKey) != 2 {
		t.Fatalf("distinct: got %d, want 2", len(byKey))
	}
	if byKey[0].Key != "noSuchField" || byKey[0].Op != "set" || byKey[0].Count != 2 {
		t.Errorf("top entry: got %+v", byKey[0])
	}
}
