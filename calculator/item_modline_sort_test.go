package calculator

import (
	"testing"

	"github.com/Vilsol/go-pob/mod"
)

func sumItemMod(i *Item, name string, typ mod.Type) float64 {
	total := 0.0
	for _, m := range i.Mods {
		if m.Name() != name || m.Type() != typ {
			continue
		}
		if v := m.Value(); v != nil && v.Type() == mod.ModValueMultiTypeFloat {
			total += v.Float()
		}
	}
	return total
}

const kaomsRootsLifeOnly = `
			Rarity: UNIQUE
Kaom&apos;s Roots
Titan Greaves
Implicits: 0
{range:0.72}+(150-200) to maximum Life
			<ModRange id="1" range="0.5"/>
`

// TestModRangeWalkReachesTheModList pins ItemsTab.lua:1258-1259.
func TestModRangeWalkReachesTheModList(t *testing.T) {
	loaded := ParseItem(kaomsRootsLifeOnly)
	if loaded.Base == nil {
		t.Fatal("REFUSING: Titan Greaves did not resolve; BuildModList would return early and prove nothing")
	}
	if len(loaded.ExplicitModLines) != 1 {
		t.Fatalf("REFUSING: fixture has %d explicit lines, want 1 -- a sort could then move the id", len(loaded.ExplicitModLines))
	}
	if got := sumItemMod(loaded, "Life", mod.TypeBase); got != 175 {
		t.Errorf("build-loaded Life BASE = %v, want 175 (150 + 0.5*50, the walked range)", got)
	}

	text := &Item{}
	text.ParseRaw(kaomsRootsLifeOnly)
	if got := sumItemMod(text, "Life", mod.TypeBase); got != 186 {
		t.Errorf("text-parsed Life BASE = %v, want 186 (150 + 0.72*50, the inline tag)", got)
	}
}

const militantFaithDominusRaw = `
			Rarity: UNIQUE
Militant Faith
Timeless Jewel
Item Level: 80
Radius: Large
Limited to: 1
Implicits: 0
{range:0.93}Carved to glorify (2000-10000) new faithful converted by High Templar Dominus
Passives in radius are Conquered by the Templars
Historic
{tags:aura}1% increased effect of Non-Curse Auras per 10 Devotion
{tags:mana}Regenerate 0.6 Mana per Second per 10 Devotion
			<ModRange id="1" range="0.93"/>
			<ModRange id="2" range="0.5"/>
			<ModRange id="3" range="0.5"/>
			<ModRange id="4" range="0.5"/>
			<ModRange id="5" range="0.5"/>
`

func explicitLines(i *Item) []string {
	out := make([]string, 0, len(i.ExplicitModLines))
	for _, ml := range i.ExplicitModLines {
		out = append(out, ml.Line)
	}
	return out
}

func assertLineOrder(t *testing.T, what string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s: %d explicit lines, want %d\n got %q\nwant %q", what, len(got), len(want), got, want)
	}
	for k := range want {
		if got[k] != want[k] {
			t.Errorf("%s: explicit[%d] = %q, want %q", what, k+1, got[k], want[k])
		}
	}
}

// TestSortCraftedModLinesMatchesThePin pins Item.lua:1433-1435 on corpus items.
func TestSortCraftedModLinesMatchesThePin(t *testing.T) {
	t.Run("Militant Faith: a line with no ItemExclusive row sinks last", func(t *testing.T) {
		item := ParseItem(militantFaithDominusRaw)
		if !item.AdvancedCopy {
			t.Fatal("REFUSING: fixture is not advanced-copy; the sort cannot run")
		}
		assertLineOrder(t, "Militant Faith", explicitLines(item), []string{
			"Passives in radius are Conquered by the Templars",
			"Regenerate 0.6 Mana per Second per 10 Devotion",
			"1% increased effect of Non-Curse Auras per 10 Devotion",
			"Historic",
			"Carved to glorify (2000-10000) new faithful converted by High Templar Dominus",
		})
		jewel, ok := itemConqueredBy(item)
		if !ok {
			t.Fatal("no conqueredBy JewelData on the item")
		}
		if jewel.ID != 6000 {
			t.Errorf("seed = %d, want 6000 (id5's 0.5 after the sort; 9440 is the inline 0.93)", jewel.ID)
		}
	})

	t.Run("Kaom's Roots: movement speed sinks, the Life roll moves with it", func(t *testing.T) {
		item := ParseItem(kaomsRootsRaw)
		assertLineOrder(t, "Kaom's Roots", explicitLines(item), []string{
			"Has no Sockets",
			"+(120-150) to maximum Life",
			"+(150-200) to maximum Life",
			"Cannot be Knocked Back",
			"Action Speed cannot be modified to below Base Value",
			"Unwavering Stance",
			"+25% movement speed",
		})
		if got := sumItemMod(item, "Life", mod.TypeBase); got != 175 {
			t.Errorf("Life BASE = %v, want 175", got)
		}
	})

	t.Run("Cane of Unravelling: the normalised fallback orders the fixed-value lines", func(t *testing.T) {
		item := ParseItem(caneOfUnravellingRaw)
		assertLineOrder(t, "Cane of Unravelling", explicitLines(item), []string{
			"+(40-55)% to Chaos Damage over Time Multiplier",
			"(60-80)% increased Chaos Damage",
			"(20-30)% increased Chaos Damage",
			"+2 to Level of all Chaos Spell Skill Gems",
			"2% increased Cast Speed per Power Charge",
			"Regenerate 2 Mana per Second per Power Charge",
			"Gain a Power Charge after Spending a total of 200 Mana",
		})
	})

	t.Run("Chimeric Spire: fractured first, crafted and custom last, rare order untouched", func(t *testing.T) {
		item := ParseItem(chimericSpireRaw)
		if !item.AdvancedCopy || item.Rarity != RarityRare {
			t.Fatalf("REFUSING: fixture is advancedCopy=%v rarity=%s, want an advanced-copy RARE", item.AdvancedCopy, item.Rarity)
		}
		assertLineOrder(t, "Chimeric Spire", explicitLines(item), []string{
			"+2 to Level of all Spell Skill Gems",
			"Intangibility: 19%",
			"+4 to Level of all Chaos Spell Skill Gems",
			"199% increased Spell Damage",
			"+64% to Chaos Damage over Time Multiplier",
			"+19% Chance to Block Spell Damage",
			"+(11-13)% Chance to Block Attack Damage",
		})
	})

	t.Run("Militant Faith, all variants: 21 lines with ties keep source order", func(t *testing.T) {
		item := ParseItem(militantFaithAllVariantsRaw)
		assertLineOrder(t, "Militant Faith (variants)", explicitLines(item), []string{
			"Carved to glorify (2000-10000) new faithful converted by High Templar Maxarius",
			"Passives in radius are Conquered by the Templars",
			"4% increased Area Damage per 10 Devotion",
			"Channelling Skills deal 4% increased Damage per 10 Devotion",
			"4% increased Elemental Damage per 10 Devotion",
			"+2% to all Elemental Resistances per 10 Devotion",
			"1% reduced Mana Cost of Skills per 10 Devotion",
			"Regenerate 0.6 Mana per Second per 10 Devotion",
			"Minions have +60 to Accuracy Rating per 10 Devotion",
			"1% increased Minion Attack and Cast Speed per 10 Devotion",
			"1% increased effect of Non-Curse Auras per 10 Devotion",
			"3% increased Effect of non-Damaging Ailments on Enemies per 10 Devotion",
			"4% reduced Duration of Curses on you per 10 Devotion",
			"4% reduced Elemental Ailment Duration on you per 10 Devotion",
			"3% increased Defences from Equipped Shield per 10 Devotion",
			"4% increased Brand Damage per 10 Devotion",
			"4% increased Totem Damage per 10 Devotion",
			"Historic",
			"Carved to glorify (2000-10000) new faithful converted by High Templar Avarius",
			"Carved to glorify (2000-10000) new faithful converted by High Templar Dominus",
			"Carved to glorify (2000-10000) new faithful converted by High Templar Venarius",
		})
	})

	t.Run("the exact key wins over the normalised one", func(t *testing.T) {
		item := ParseItem("Rarity: UNIQUE\nFixture\nTitan Greaves\nImplicits: 0\n{range:0.5}+(30-60) to Evasion Rating\n+25% movement speed\n")
		var got *float64
		for _, ml := range item.ExplicitModLines {
			if ml.Line == "+(30-60) to Evasion Rating" {
				got = ml.Order
			}
		}
		if got == nil || *got != 1574 {
			t.Errorf("Order = %v, want 1574 (the exact row; 1570 is the normalised one)", got)
		}
	})

	t.Run("group 3: custom alone qualifies, and order is ignored inside it", func(t *testing.T) {
		hi, lo := 9000.0, 10.0
		lines := []ModLine{
			{Line: "+(11-13)% Chance to Block Attack Damage", Custom: true},
			{Line: "+19% Chance to Block Spell Damage", Crafted: true, Order: &hi},
			{Line: "+64% to Chaos Damage over Time Multiplier", Crafted: true, Order: &lo},
			{Line: "199% increased Spell Damage"},
		}
		sortCraftedModLines(lines)
		got := make([]string, len(lines))
		for k := range lines {
			got[k] = lines[k].Line
		}
		assertLineOrder(t, "group 3", got, []string{
			"199% increased Spell Damage",
			"+(11-13)% Chance to Block Attack Damage",
			"+19% Chance to Block Spell Damage",
			"+64% to Chaos Damage over Time Multiplier",
		})
	})
}

const caneOfUnravellingRaw = `
			Rarity: UNIQUE
Cane of Unravelling
Ezomyte Staff
Variant: Pre 3.5.0
Variant: Pre 3.11.0
Variant: Pre 3.25.0
Variant: Current
Selected Variant: 4
Quality: 20
Sockets: B-B-B-B-B-B
LevelReq: 60
Implicits: 3
{variant:1,2}+18% Chance to Block Attack Damage while wielding a Staff
{variant:3}+20% Chance to Block Attack Damage while wielding a Staff
{variant:4}+25% Chance to Block Attack Damage while wielding a Staff
{variant:2,3,4}{range:1}+(40-55)% to Chaos Damage over Time Multiplier
{variant:1}{range:0.5}(60-80)% increased Chaos Damage
{variant:2,3,4}{range:1}(20-30)% increased Chaos Damage
2% increased Cast Speed per Power Charge
+2 to Level of all Chaos Spell Skill Gems
Gain a Power Charge after Spending a total of 200 Mana
Regenerate 2 Mana per Second per Power Charge
			<ModRange id="1" range="0.5"/>
			<ModRange id="2" range="0.5"/>
			<ModRange id="3" range="0.5"/>
			<ModRange id="4" range="1"/>
			<ModRange id="5" range="0.5"/>
			<ModRange id="6" range="1"/>
			<ModRange id="7" range="0.5"/>
			<ModRange id="8" range="0.5"/>
			<ModRange id="9" range="0.5"/>
			<ModRange id="10" range="0.5"/>
`

const chimericSpireRaw = `
			Rarity: RARE
Chimeric Spire
Battery Staff
Crafted: true
Prefix: {range:0.5}GlobalSpellGemsLevelTwoHand2
Prefix: {range:0.5}GlobalChaosSpellGemsLevelTwoHand3
Prefix: {range:0.96}SpellDamageOnTwoHandWeapon8
Suffix: {range:0.889}ChaosDamageOverTimeMultiplierTwoHand4__
Suffix: {range:0.5}StaffSpellBlockPercentage6
Suffix: None
Item Level: 85
Quality: 20
Sockets: B-R-B-G-W-B
LevelReq: 70
Implicits: 4
{enchant}{crafted}Quality does not increase Physical Damage
{enchant}{crafted}Grants 1% increased Area of Effect per 4% Quality
Spend Energy Shield before Mana for Costs of Socketed Skills
{range:1}+(140-165) to maximum Energy Shield
Intangibility: 19%
{tags:caster,gem}{prefix}{fractured}+2 to Level of all Spell Skill Gems
{tags:chaos,caster,gem}{prefix}+4 to Level of all Chaos Spell Skill Gems
{tags:damage,caster}{prefix}199% increased Spell Damage
{tags:damage,chaos}{suffix}+64% to Chaos Damage over Time Multiplier
{suffix}+19% Chance to Block Spell Damage
{suffix}{custom}{crafted}{range:0.5}+(11-13)% Chance to Block Attack Damage
Fractured Item
			<ModRange id="1" range="0.5"/>
			<ModRange id="2" range="0.5"/>
			<ModRange id="3" range="0.5"/>
			<ModRange id="4" range="1"/>
			<ModRange id="5" range="0.5"/>
			<ModRange id="6" range="0.5"/>
			<ModRange id="7" range="0.5"/>
			<ModRange id="8" range="0.5"/>
			<ModRange id="9" range="0.5"/>
			<ModRange id="10" range="0.5"/>
			<ModRange id="11" range="0.5"/>
`

const militantFaithAllVariantsRaw = `
			Rarity: UNIQUE
Militant Faith
Timeless Jewel
League: Legion
Variant: Avarius (Power of Purpose)
Variant: Dominus (Inner Conviction)
Variant: Venarius (The Agnostic) (Pre 3.11.0)
Variant: Maxarius (Transcendence)
Variant: Totem Damage
Variant: Brand Damage
Variant: Channelling Damage
Variant: Area Damage
Variant: Elemental Damage
Variant: Elemental Resistances
Variant: Effect of non-Damaging Ailments
Variant: Elemental Ailment Duration
Variant: Duration of Curses
Variant: Minion Attack and Cast Speed
Variant: Minions Accuracy Rating
Variant: Mana Regen
Variant: Skill Cost
Variant: Non-Curse Aura Effect
Variant: Defences from Shield
Selected Variant: 2
Has Alt Variant: true
Selected Alt Variant: 19
Has Alt Variant Two: true
Selected Alt Variant Two: 19
Radius: Large
Limited to: 1
Implicits: 0
{variant:1}{range:0.5}Carved to glorify (2000-10000) new faithful converted by High Templar Avarius
{variant:2}{range:0.5}Carved to glorify (2000-10000) new faithful converted by High Templar Dominus
{variant:3}{range:0.5}Carved to glorify (2000-10000) new faithful converted by High Templar Venarius
{variant:4}{range:0.5}Carved to glorify (2000-10000) new faithful converted by High Templar Maxarius
{variant:8}4% increased Area Damage per 10 Devotion
{variant:7}Channelling Skills deal 4% increased Damage per 10 Devotion
{variant:9}4% increased Elemental Damage per 10 Devotion
{variant:10}+2% to all Elemental Resistances per 10 Devotion
{variant:17}1% reduced Mana Cost of Skills per 10 Devotion
{variant:16}Regenerate 0.6 Mana per Second per 10 Devotion
{variant:15}Minions have +60 to Accuracy Rating per 10 Devotion
{variant:14}1% increased Minion Attack and Cast Speed per 10 Devotion
{variant:18}1% increased effect of Non-Curse Auras per 10 Devotion
{variant:11}3% increased Effect of non-Damaging Ailments on Enemies per 10 Devotion
{variant:13}4% reduced Duration of Curses on you per 10 Devotion
{variant:12}4% reduced Elemental Ailment Duration on you per 10 Devotion
{variant:19}3% increased Defences from Equipped Shield per 10 Devotion
{variant:6}4% increased Brand Damage per 10 Devotion
{variant:5}4% increased Totem Damage per 10 Devotion
Passives in radius are Conquered by the Templars
Historic
			<ModRange id="1" range="0.5"/>
			<ModRange id="2" range="0.5"/>
			<ModRange id="3" range="0.5"/>
			<ModRange id="4" range="0.5"/>
			<ModRange id="5" range="0.5"/>
			<ModRange id="6" range="0.5"/>
			<ModRange id="7" range="0.5"/>
			<ModRange id="8" range="0.5"/>
			<ModRange id="9" range="0.5"/>
			<ModRange id="10" range="0.5"/>
			<ModRange id="11" range="0.5"/>
			<ModRange id="12" range="0.5"/>
			<ModRange id="13" range="0.5"/>
			<ModRange id="14" range="0.5"/>
			<ModRange id="15" range="0.5"/>
			<ModRange id="16" range="0.5"/>
			<ModRange id="17" range="0.5"/>
			<ModRange id="18" range="0.5"/>
			<ModRange id="19" range="0.5"/>
			<ModRange id="20" range="0.5"/>
			<ModRange id="21" range="0.5"/>
`
