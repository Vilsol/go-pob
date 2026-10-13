package raw


// SkillPart is one entry of PoB's hand-written per-skill `parts` table
// (src/Data/Skills/*.lua). PoB keys a part's flag overrides by skill-flag name in the same
// table as `name`, and CalcActiveSkill.lua:206-212 walks it setting the flag for `true` and
// CLEARING it for `false`; Flags keeps the two apart because a Go map read cannot.
type SkillPart struct {
	Name  string
	Flags map[string]bool
}

// SkillParts mirrors `grantedEffect.parts`, keyed by granted-effect id. It exists for the
// same reason as SkillBaseFlags: the table is hand-written PoB data that go-pob-data does
// not ship. PoB reads it at src/Modules/CalcActiveSkill.lua:196 to pick the part the build
// selected (`<Gem skillPart="N">`), apply that part's flag overrides, and set the
// `multiPart` flag. Flag names are PoB's raw strings — data/raw cannot import calculator,
// whose SkillFlag is a string type holding exactly these values.
//
// Generated from PoB v2.18.1 by oracle/skill_parts.py.

var SkillParts = map[string][]SkillPart{
	// src/Data/Skills/sup_str.lua:19
	"AlphaSupport": {
		{Name: "Three"},
	},
	// src/Data/Skills/act_int.lua:3
	"Autopulse": {
		{Name: "Deep Indent", Flags: map[string]bool{"spell": true}},
	},
	// src/Data/Skills/other.lua:13
	"ControlChar": {
		{Name: "BellInside"},
	},
	// src/Data/Skills/sup_str.lua:11
	"MarrowSupport": {
		{Name: "Two"},
	},
	// src/Data/Skills/sup_dex.lua:10
	"NbspSkill": {
		{Name: "Nbsp Name"},
	},
	// src/Data/Skills/sup_str.lua:3
	"NimbusSupport": {
		{Name: "One"},
	},
	// src/Data/Skills/sup_int.lua:14
	"OrbSupport": {
		{Name: "Lone"},
	},
	// src/Data/Skills/sup_int.lua:3
	"PrismSupport": {
		{Name: "First"},
		{Name: "Second"},
	},
	// src/Data/Skills/sup_dex.lua:3
	"QuiverSupport": {
		{Name: "Support Part"},
	},
	// src/Data/Skills/spectre.lua:3
	"RavenGhost": {
		{Name: "Haunt"},
	},
	// src/Data/Skills/other.lua:3
	"SableAura": {
		{Name: "Aura", Flags: map[string]bool{"aura": true, "buff": false}},
	},
	// src/Data/Skills/minion.lua:3
	"TangleSummon": {
		{Name: "Alpha"},
		{Name: "Beta"},
	},
	// src/Data/Skills/act_dex.lua:28
	"TiedId": {
		{Name: "From act_dex.lua"},
	},
	// src/Data/Skills/minion.lua:14
	"TiedId": {
		{Name: "From minion.lua"},
	},
	// src/Data/Skills/sup_int.lua:22
	"TiedId": {
		{Name: "From sup_int.lua"},
	},
	// src/Data/Skills/glove.lua:8
	"UmbralGrasp": {
		{Name: "Backslash \\\\b Kept"},
	},
	// src/Data/Skills/act_str.lua:12
	"VaultCharge": {
		{Name: "Charge"},
		{Name: "Impact"},
	},
	// src/Data/Skills/act_str.lua:3
	"WarcrySlam": {
		{Name: "Melee", Flags: map[string]bool{"attack": true}},
	},
	// src/Data/Skills/act_int.lua:13
	"XrayBolt": {
		{Name: "Only"},
	},
	// src/Data/Skills/act_dex.lua:16
	"YankeeShot": {
		{Name: "Single"},
		{Name: "Volley", Flags: map[string]bool{"totem": true}},
	},
	// src/Data/Skills/act_dex.lua:3
	"ZetaStrike": {
		{Name: "Slash"},
		{Name: "Thrust", Flags: map[string]bool{"trap": true, "mine": false}},
	},
}
