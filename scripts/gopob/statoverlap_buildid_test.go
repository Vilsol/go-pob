package main

import "testing"

// TestBuildIDRegexKeepsNonASCIINames pins a silent data loss in a tool that reports and never asserts.
func TestBuildIDRegexKeepsNonASCIINames(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{"CJK display name", "3767-ladder-Foo_4542.\u67da\u67da\u67da\u59b9\u4e0d\u958b\u639b",
			"3767-ladder-Foo_4542.\u67da\u67da\u67da\u59b9\u4e0d\u958b\u639b"},
		{"cyrillic", "2076-ladder-Привет_1.Тест", "2076-ladder-Привет_1.Тест"},
		{"plain ascii is unchanged", "0001-necromancer-unknown", "0001-necromancer-unknown"},
		{"accented latin", "0450-ladder-José_9.Ñandú", "0450-ladder-José_9.Ñandú"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := reBuildID.FindString(tc.in)
			if got != tc.want {
				t.Errorf("truncated at the first non-ASCII byte:\n  got  %q\n  want %q", got, tc.want)
			}
		})
	}

	t.Run("stops at whitespace", func(t *testing.T) {
		if got := reBuildID.FindString("0001-abc.xml  ratio 2.00"); got != "0001-abc.xml" {
			t.Errorf("the id ran past its name: %q", got)
		}
	})

	t.Run("a five-digit index keeps its leading digit", func(t *testing.T) {
		for _, tc := range []struct{ in, want string }{
			{"10000-ladder-Flyarx_5815.xml", "10000-ladder-Flyarx_5815.xml"},
			{"10000-ladder-Flyarx_5815.Перепогружение.xml",
				"10000-ladder-Flyarx_5815.Перепогружение.xml"},
			{"9999-ladder-Flyarx_5815.witrgfxdscx.xml", "9999-ladder-Flyarx_5815.witrgfxdscx.xml"},
			{"100000-ladder-Foo.xml", "100000-ladder-Foo.xml"},
		} {
			if got := reBuildID.FindString(tc.in); got != tc.want {
				t.Errorf("index truncated:\n  got  %q\n  want %q", got, tc.want)
			}
		}
	})

	t.Run("truncation collisions are gone", func(t *testing.T) {
		a := reBuildID.FindString("2076-ladder-Привет")
		b := reBuildID.FindString("2076-ladder-Мир")
		if a == b {
			t.Errorf("two distinct builds collapsed to one id %q", a)
		}
	})
}
