package calculator

import (
	"testing"

	"github.com/MarvinJWendt/testza"
)

func TestBareSkillNamePrefixParses(t *testing.T) {
	for _, line := range []string{
		"wrath 45% increased aura effect",
		"hatred 45% increased aura effect",
		"zealotry 45% increased aura effect",
		"smite 45% increased aura effect",
		"hatred 40% increased aura effect",
	} {
		testza.AssertTrue(t, goParsed(line), "%q must parse: ModParser.lua:3583", line)
	}

	testza.AssertTrue(t, goParsed("grace reserves no mana"), "ModParser.lua:3583")

	testza.AssertTrue(t, goParsed("purity of elements has 200% reduced aura effect"),
		"the :3584 control must keep parsing")
}
