package calculator

import "testing"

// SkillsTab.lua:1334-1337: a group granted by Custom Modifiers names them as its source.
func TestSocketGroupSourceLineNamesCustomModifiers(t *testing.T) {
	if line, ok := (&Environment{}).socketGroupSourceLine("Config"); !ok || line != "^7Source: "+colorNormal+"Custom Modifiers" {
		t.Errorf("%q %v", line, ok)
	}
}
