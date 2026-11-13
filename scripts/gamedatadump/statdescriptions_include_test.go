package main

import "testing"

// 3.30 stat description files include each other as .csd; earlier PoE1 files as .txt. Both name the same file.
func TestIncludeReAcceptsTxtAndCsd(t *testing.T) {
	for inc, want := range map[string]string{
		`Metadata/StatDescriptions/stat_descriptions.txt`:            "stat_descriptions",
		`Metadata/StatDescriptions/stat_descriptions.csd`:            "stat_descriptions",
		`Metadata/StatDescriptions/skill_stat_descriptions_base.csd`: "skill_stat_descriptions_base",
	} {
		m := includeRe.FindStringSubmatch(inc)
		if m == nil || m[1] != want {
			t.Errorf("%s: got %v, want %q", inc, m, want)
		}
	}
	for _, inc := range []string{`Data/StatDescriptions/x.csd`, `Metadata/StatDescriptions/x.dat`} {
		if includeRe.MatchString(inc) {
			t.Errorf("%s matched", inc)
		}
	}
}
