package calculator2

import (
	"reflect"
	"testing"
)

// Expected values are PoB-PoE2's Build.lua acts table and EstimatePlayerProgress run under luajit on the same counts.
func TestEstimatePlayerProgress(t *testing.T) {
	cases := []struct {
		used, asc, secAsc, ws1, ws2 int
		extra, extraWS              float64
		want                        PlayerProgress
	}{
		{0, 0, 0, 0, 0, 0, 0, PlayerProgress{Level: 1, Act: "Act 1", NormalPassives: 0, UsedMax: 123, WeaponSetMax: 24, AscMax: 8}},
		{10, 0, 0, 0, 0, 0, 0, PlayerProgress{Level: 11, Act: "Act 1", NormalPassives: 10, UsedMax: 123, WeaponSetMax: 24, AscMax: 8}},
		{30, 2, 0, 0, 0, 0, 0, PlayerProgress{Level: 27, Act: "Act 2", NormalPassives: 30, UsedMax: 123, WeaponSetMax: 24, AscUsed: 2, AscMax: 8}},
		{60, 4, 0, 4, 2, 0, 0, PlayerProgress{Level: 47, Act: "Act 4", NormalPassives: 58, UsedMax: 123, WeaponSet1Used: 4, WeaponSet2Used: 2, WeaponSetMax: 24, AscUsed: 4, AscMax: 8,
			Warnings: []string{"You have 2 Weapon set 2 passives available"}}},
		{90, 6, 0, 8, 8, 0, 0, PlayerProgress{Level: 62, Act: "Endgame", NormalPassives: 82, UsedMax: 123, WeaponSet1Used: 8, WeaponSet2Used: 8, WeaponSetMax: 24, AscUsed: 6, AscMax: 8}},
		{120, 8, 0, 24, 24, 2, 0, PlayerProgress{Level: 71, Act: "Endgame", Extra: 2, NormalPassives: 96, UsedMax: 125, WeaponSet1Used: 24, WeaponSet2Used: 24, WeaponSetMax: 24, AscUsed: 8, AscMax: 8}},
		{140, 10, 0, 26, 20, 0, 0, PlayerProgress{Level: 97, Act: "Endgame", NormalPassives: 120, UsedMax: 123, WeaponSet1Used: 26, WeaponSet2Used: 20, WeaponSetMax: 24, AscUsed: 10, AscMax: 8,
			Warnings: []string{"You have too many ascendancy points allocated", "You have allocated 2 too many weapon set 1 passives"}}},
		{150, 8, 0, 26, 28, 4, 2, PlayerProgress{Level: 97, Act: "Endgame", Extra: 4, NormalPassives: 124, UsedMax: 127, WeaponSet1Used: 26, WeaponSet2Used: 28, WeaponSetMax: 26, AscUsed: 8, AscMax: 8,
			Warnings: []string{"You have allocated 2 too many weapon set 2 passives"}}},
		{140, 8, 0, 0, 0, 0, 0, PlayerProgress{Level: 100, Act: "Endgame", NormalPassives: 140, UsedMax: 123, WeaponSetMax: 24, AscUsed: 8, AscMax: 8,
			Warnings: []string{"You have too many passive points allocated"}}},
	}
	for _, c := range cases {
		got := estimatePlayerProgress(c.used, c.asc, c.secAsc, c.ws1, c.ws2, map[string]float64{"ExtraPoints": c.extra, "PassivePointsToWeaponSetPoints": c.extraWS})
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("used %d asc %d ws %d/%d extra %v/%v:\n got %+v\nwant %+v", c.used, c.asc, c.ws1, c.ws2, c.extra, c.extraWS, got, c.want)
		}
	}
}
