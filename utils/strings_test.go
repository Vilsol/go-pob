package utils

import "testing"

func TestCapitalEach(t *testing.T) {
	tests := []struct {
		str  string
		want string
	}{
		{
			str:  "hello world",
			want: "Hello World",
		},
	}
	for _, tt := range tests {
		t.Run(tt.str, func(t *testing.T) {
			if got := CapitalEach(tt.str); got != tt.want {
				t.Errorf("CapitalEach() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestEscapeGGGString grades against luajit running Common.lua:1048-1054 on the same inputs.
func TestEscapeGGGString(t *testing.T) {
	for in, want := range map[string]string{
		"Spell Critical Strike Chance [BifurcatedCrit|Bifurcates]":                          "Spell Critical Strike Chance Bifurcates",
		"Gain [SpiritInfusion|Spirit Infusion] every 0.5 seconds while Channelling a Spell": "Gain Spirit Infusion every 0.5 seconds while Channelling a Spell",
		"Triggers Level 20 [BoneFeeders|Summon Bone Feeders] when Allocated":                "Triggers Level 20 Summon Bone Feeders when Allocated",
		"[Critical] and [A|B] then [C|D]":                                                   "Critical and B then D",
		"<red>{Blood} and <x>{y}":                                                           "Blood and y",
		"plain [unterminated":                                                               "plain [unterminated",
		"[A|B|C]":                                                                           "[A|B|C]",
		"[A|B] x [C] y":                                                                     "B x C y",
	} {
		if got := EscapeGGGString(in); got != want {
			t.Errorf("EscapeGGGString(%q) = %q, want %q", in, got, want)
		}
	}
}
