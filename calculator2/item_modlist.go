package calculator2

import (
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob/mod"
)

// calcLocal sums and removes the local mods of one name and type. FLAG reads through calcLocalFlag.
//
//pob2:src/Classes/Item.lua:2424-2451
func calcLocal(modList *[]mod.Mod, name string, modType mod.Type, flags mod.MFlag) float64 {
	result := 0.0
	if modType == mod.TypeMore {
		result = 1
	}
	kept := (*modList)[:0]
	for _, m := range *modList {
		if isLocalMod(m, name, modType, flags) {
			if modType == mod.TypeMore {
				result *= (100 + m.Value().Float()) / 100
			} else {
				result += m.Value().Float()
			}
			continue
		}
		kept = append(kept, m)
	}
	clear((*modList)[len(kept):])
	*modList = kept
	return result
}

//pob2:src/Classes/Item.lua:2424-2451
func calcLocalFlag(modList *[]mod.Mod, name string, flags mod.MFlag) bool {
	result := false
	kept := (*modList)[:0]
	for _, m := range *modList {
		if isLocalMod(m, name, mod.TypeFlag, flags) {
			result = result || m.Value().Flag()
			continue
		}
		kept = append(kept, m)
	}
	clear((*modList)[len(kept):])
	*modList = kept
	return result
}

// luaTags is the tags ipairs(mod) visits: those before the first nil. A mod table can hold a tag at
// [2] with none at [1] (ModCache has such a line), and every ipairs walk then sees no tags.
func luaTags(m mod.Mod) []mod.Tag {
	tags := m.Tags()
	for n, t := range tags {
		if t == nil {
			return tags[:n]
		}
	}
	return tags
}

//pob2:block:src/Classes/Item.lua:2436-2436
func isLocalMod(m mod.Mod, name string, modType mod.Type, flags mod.MFlag) bool {
	if m.Name() != name || m.Type() != modType || m.Flags() != flags || m.KeywordFlags() != 0 {
		return false
	}
	tags := luaTags(m)
	return len(tags) == 0 || tags[0].Type() == mod.TypeInSlot
}

// listValues is modList:List(nil, name) over a plain mod list. Only untagged mods are read: a
// tagged LIST mod would need a mod store to evaluate it, and no item list carries one.
func listValues(modList []mod.Mod, name string) []any {
	var out []any
	for _, m := range modList {
		if m.Name() == name && m.Type() == mod.TypeList && len(luaTags(m)) == 0 && m.Value() != nil {
			out = append(out, m.Value().ValueList)
		}
	}
	return out
}

// flagValue is modList:Flag(nil, name) over a plain mod list, with listValues' tag rule.
func flagValue(modList []mod.Mod, name string) bool {
	for _, m := range modList {
		if m.Name() == name && m.Type() == mod.TypeFlag && len(luaTags(m)) == 0 && m.Value() != nil && m.Value().Flag() {
			return true
		}
	}
	return false
}

// keyValue reads a { key = ..., value = ... } list value.
func keyValue(v any) (string, any, bool) {
	if lt, ok := v.(LuaTable); ok {
		k, ok := lt["key"].(string)
		return k, lt["value"], ok
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return "", nil, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return "", nil, false
	}
	k, val := rv.FieldByName("Key"), rv.FieldByName("Value")
	if !k.IsValid() || k.Kind() != reflect.String || !val.IsValid() {
		return "", nil, false
	}
	return k.String(), val.Interface(), true
}

func toFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case bool:
		return 0, false
	}
	return 0, false
}

// nestedModField is the value.mod field of a list value, if it has one.
func nestedModField(m mod.Mod) (reflect.Value, bool) {
	if m.Value() == nil || m.Value().Type() != mod.ModValueMultiTypeList {
		return reflect.Value{}, false
	}
	rv := reflect.ValueOf(m.Value().ValueList)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return reflect.Value{}, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	f := rv.FieldByName("Mod")
	if !f.IsValid() || f.IsNil() {
		return reflect.Value{}, false
	}
	if _, ok := f.Interface().(mod.Mod); !ok {
		return reflect.Value{}, false
	}
	return f, true
}

// nestedMod is `type(mod.value) == "table" and mod.value.mod or mod`.
func nestedMod(m mod.Mod) mod.Mod {
	if f, ok := nestedModField(m); ok {
		inner, _ := f.Interface().(mod.Mod)
		return inner
	}
	return m
}

// withNestedMod returns a deep copy of m whose value.mod is replaced by fn(copy of value.mod).
func withNestedMod(m mod.Mod, fn func(mod.Mod) mod.Mod) mod.Mod {
	c := m.Clone()
	if _, ok := nestedModField(c); !ok {
		return c
	}
	rv := reflect.ValueOf(c.Value().ValueList)
	ptr := rv.Kind() == reflect.Pointer
	if ptr {
		rv = rv.Elem()
	}
	cp := reflect.New(rv.Type()).Elem()
	cp.Set(rv)
	inner, _ := cp.FieldByName("Mod").Interface().(mod.Mod)
	cp.FieldByName("Mod").Set(reflect.ValueOf(fn(inner.Clone())))
	if ptr {
		c.Value().ValueList = cp.Addr().Interface()
	} else {
		c.Value().ValueList = cp.Interface()
	}
	return c
}

// setSource is modLib.setSource on a copy: Lua mutates the mod table the mod line holds, so the
// caller stores the copy back into the line.
//
//pob2:src/Modules/ModTools.lua:277-283
func setSource(m mod.Mod, source string) mod.Mod {
	c := withNestedMod(m, func(inner mod.Mod) mod.Mod { return inner.Source(mod.Source(source)) })
	return c.Source(mod.Source(source))
}

func tagIsUnscalable(tag mod.Tag) bool {
	rv := reflect.ValueOf(tag)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return false
	}
	for _, name := range []string{"UnscalableTag", "Unscalable"} {
		if f := rv.FieldByName(name); f.IsValid() && f.Kind() == reflect.Bool && f.Bool() {
			return true
		}
	}
	return false
}

// scaledKeyed scales a list value that names its scaled field (value.keyOfScaledMod), read by
// reflection: an exported KeyOfScaledMod string naming an exported float64 field.
//
//pob2:block:src/Classes/ModStore.lua:98-104
func scaledKeyed(m mod.Mod, scale float64) (mod.Mod, bool) {
	if m.Value() == nil || m.Value().Type() != mod.ModValueMultiTypeList {
		return nil, false
	}
	rv := reflect.ValueOf(m.Value().ValueList)
	ptr := rv.Kind() == reflect.Pointer
	if ptr {
		if rv.IsNil() {
			return nil, false
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, false
	}
	kf := rv.FieldByName("KeyOfScaledMod")
	if !kf.IsValid() || kf.Kind() != reflect.String || kf.String() == "" {
		return nil, false
	}
	name := kf.String()
	fieldName := strings.ToUpper(name[:1]) + name[1:]
	if vf := rv.FieldByName(fieldName); !vf.IsValid() || vf.Kind() != reflect.Float64 {
		return nil, false
	}
	c := m.Clone()
	cp := reflect.New(rv.Type()).Elem()
	cp.Set(rv)
	vf := cp.FieldByName(fieldName)
	if key := cp.FieldByName("Key"); key.IsValid() && key.Kind() == reflect.String && key.String() == "level" {
		vf.SetFloat(math.Floor(vf.Float() * scale))
	} else {
		vf.SetFloat(round(vf.Float()*scale, 2))
	}
	if ptr {
		c.Value().ValueList = cp.Addr().Interface()
	} else {
		c.Value().ValueList = cp.Interface()
	}
	return c, true
}

// scaleAddMod is ModStore:ScaleAddMod without roundToNearest (no item caller passes it).
//
//pob2:src/Classes/ModStore.lua:82-119
func scaleAddMod(list *[]mod.Mod, m mod.Mod, scale float64) {
	unscalable := false
	for _, tag := range luaTags(m) {
		if tagIsUnscalable(tag) {
			unscalable = true
			break
		}
	}
	if scale == 1 || unscalable {
		*list = append(*list, m)
		return
	}
	scaleValue := func(sub mod.Mod) mod.Mod {
		if sub.Value() == nil || sub.Value().Type() != mod.ModValueMultiTypeFloat {
			return sub
		}
		v := sub.Value().Float()
		var precision *int
		if p, ok := highPrecisionMods[sub.Name()][sub.Type()]; ok {
			precision = &p
		} else if math.Floor(v) != v {
			p := defaultHighPrecision
			precision = &p
		}
		if precision != nil {
			power := math.Pow(10, float64(*precision))
			sub.Value().SetFloat(math.Floor(v*scale*power) / power)
		} else {
			sub.Value().SetFloat(math.Trunc(round(v*scale, 2)))
		}
		return sub
	}
	if _, ok := nestedModField(m); ok {
		*list = append(*list, withNestedMod(m, scaleValue))
		return
	}
	if scaled, ok := scaledKeyed(m, scale); ok {
		*list = append(*list, scaled)
		return
	}
	*list = append(*list, scaleValue(m.Clone()))
}

func slotPlaceholderReplacer(slotName string, slotNum int) *strings.Replacer {
	hand, other := "OffHand", "1"
	if slotNum == 1 {
		hand, other = "MainHand", "2"
	}
	return strings.NewReplacer("{SlotName}", slotName, "{Hand}", hand, "{OtherSlotNum}", other)
}

// rewriteTagStrings applies the {SlotName}/{Hand}/{OtherSlotNum} substitution to every string
// field of a tag, returning a copy when one changed.
//
//pob2:block:src/Classes/Item.lua:2470-2476
func rewriteTagStrings(tag mod.Tag, r *strings.Replacer) (mod.Tag, bool) {
	v := reflect.ValueOf(tag)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return tag, false
	}
	src := v.Elem()
	var dst reflect.Value
	field := func(idx int) reflect.Value {
		if !dst.IsValid() {
			dst = reflect.New(src.Type())
			dst.Elem().Set(src)
		}
		return dst.Elem().Field(idx)
	}
	for idx := 0; idx < src.NumField(); idx++ {
		if src.Type().Field(idx).PkgPath != "" {
			continue
		}
		f := src.Field(idx)
		switch f.Kind() {
		case reflect.String:
			if s := f.String(); r.Replace(s) != s {
				field(idx).SetString(r.Replace(s))
			}
		case reflect.Pointer:
			if f.IsNil() || f.Elem().Kind() != reflect.String {
				continue
			}
			if s := f.Elem().String(); r.Replace(s) != s {
				p := reflect.New(f.Type().Elem())
				p.Elem().SetString(r.Replace(s))
				field(idx).Set(p)
			}
		case reflect.Slice:
			if f.Type().Elem().Kind() != reflect.String {
				continue
			}
			var replaced reflect.Value
			for j := 0; j < f.Len(); j++ {
				s := f.Index(j).String()
				if r.Replace(s) == s {
					continue
				}
				if !replaced.IsValid() {
					replaced = reflect.MakeSlice(f.Type(), f.Len(), f.Len())
					reflect.Copy(replaced, f)
				}
				replaced.Index(j).SetString(r.Replace(s))
			}
			if replaced.IsValid() {
				field(idx).Set(replaced)
			}
		}
	}
	if !dst.IsValid() {
		return tag, false
	}
	out, _ := dst.Interface().(mod.Tag)
	return out, true
}

func tagSlotNum(tag mod.Tag) (int, bool) {
	switch t := tag.(type) {
	case *mod.SlotNumberTag:
		return t.N, true
	case mod.SlotNumberTag:
		return t.N, true
	case *mod.InSlotTag:
		return t.N, true
	case mod.InSlotTag:
		return t.N, true
	}
	return 0, false
}

func floatOr(p *float64, d float64) float64 {
	if p != nil {
		return *p
	}
	return d
}

// BuildModListForSlotNum builds the item's mods for one slot (slotNum 0 is Lua's nil), applying
// the local mods and quality.
//
//pob2:src/Classes/Item.lua:2454-2714
//nolint:gocyclo,maintidx // mirrors the Lua
func (i *Item) BuildModListForSlotNum(baseList []mod.Mod, slotNum int) []mod.Mod {
	slotName := i.GetPrimarySlot()
	if slotNum == 2 {
		slotName = strings.ReplaceAll(slotName, "1", "2")
	}
	replacer := slotPlaceholderReplacer(slotName, slotNum)
	modList := make([]mod.Mod, 0, len(baseList)+2)
	for _, baseMod := range baseList {
		m := baseMod.Clone()
		add := true
		for idx, tag := range luaTags(m) {
			if n, ok := tagSlotNum(tag); ok && n != slotNum {
				add = false
				break
			}
			if rewritten, changed := rewriteTagStrings(tag, replacer); changed {
				m.Tags()[idx] = rewritten
			}
		}
		if add {
			modList = append(modList, m)
		}
	}
	craftedQuality := calcLocal(&modList, "Quality", mod.TypeBase, 0)
	if i.CraftedQuality == nil || craftedQuality != *i.CraftedQuality {
		if i.CraftedQuality != nil {
			i.Quality = floatp(floatOr(i.Quality, 0) - *i.CraftedQuality + craftedQuality)
		}
		i.CraftedQuality = floatp(craftedQuality)
	}
	if i.Quality != nil {
		modList = append(modList, mod.NewFloat("Multiplier:QualityOn"+slotName, mod.TypeBase, *i.Quality).Source("Quality"))
	}
	quality := floatOr(i.Quality, 0)
	if i.SpiritValue != nil {
		spiritBase := i.Base.Spirit + calcLocal(&modList, "Spirit", mod.TypeBase, 0)
		spiritInc := calcLocal(&modList, "Spirit", mod.TypeIncrease, 0)
		i.SpiritValue = floatp(round(spiritBase * (1 + spiritInc/100)))
	}
	if i.CharmLimit != nil {
		i.CharmLimit = floatp(i.Base.CharmLimit + calcLocal(&modList, "CharmLimit", mod.TypeBase, 0))
	}
	switch {
	case i.Base.Weapon != nil:
		//pob2:block:src/Classes/Item.lua:2501-2560
		w := i.Base.Weapon
		weaponData := map[string]float64{}
		i.WeaponData[slotNum] = weaponData
		weaponData["AttackSpeedInc"] = calcLocal(&modList, "Speed", mod.TypeIncrease, mod.MFlagAttack) + math.Floor(quality/8*calcLocal(&modList, "AlternateQualityLocalAttackSpeedPer8Quality", mod.TypeIncrease, 0))
		weaponData["AttackRate"] = round(w.AttackRateBase*(1+weaponData["AttackSpeedInc"]/100), 2)
		weaponData["rangeBonus"] = calcLocal(&modList, "WeaponRange", mod.TypeBase, 0) + float64(10*calcLocal(&modList, "WeaponRangeMetre", mod.TypeBase, 0)) + math.Floor(quality/10*calcLocal(&modList, "AlternateQualityLocalWeaponRangePer10Quality", mod.TypeBase, 0))
		weaponData["range"] = w.Range + weaponData["rangeBonus"]
		if w.ReloadTimeBase != 0 {
			weaponData["ReloadSpeedInc"] = calcLocal(&modList, "ReloadSpeed", mod.TypeIncrease, mod.MFlagAttack) + weaponData["AttackSpeedInc"]
			weaponData["ReloadTime"] = round(w.ReloadTimeBase/(1+weaponData["ReloadSpeedInc"]/100), 2)
		}
		localIncEle := calcLocal(&modList, "LocalElementalDamage", mod.TypeIncrease, 0)
		baseDmg := map[string][2]float64{
			"Physical": {w.PhysicalMin, w.PhysicalMax}, "Lightning": {w.LightningMin, w.LightningMax},
			"Cold": {w.ColdMin, w.ColdMax}, "Fire": {w.FireMin, w.FireMax}, "Chaos": {w.ChaosMin, w.ChaosMax},
		}
		for _, dmgType := range dmgTypeList {
			lo := baseDmg[dmgType][0] + calcLocal(&modList, dmgType+"Min", mod.TypeBase, 0)
			hi := baseDmg[dmgType][1] + calcLocal(&modList, dmgType+"Max", mod.TypeBase, 0)
			if dmgType == "Physical" {
				physInc := calcLocal(&modList, "PhysicalDamage", mod.TypeIncrease, 0)
				qualityScalar := quality
				if calcLocal(&modList, "AlternateQualityWeapon", mod.TypeBase, 0) > 0 {
					qualityScalar = 0
				}
				lo = round(lo * (1 + physInc/100) * (1 + qualityScalar/100))
				hi = round(hi * (1 + physInc/100) * (1 + qualityScalar/100))
			} else if dmgType != "Chaos" {
				localInc := calcLocal(&modList, "Local"+dmgType+"Damage", mod.TypeIncrease, 0) + localIncEle
				lo = round(lo * (1 + localInc/100))
				hi = round(hi * (1 + localInc/100))
			}
			if lo > 0 && hi > 0 {
				weaponData[dmgType+"Min"] = lo
				weaponData[dmgType+"Max"] = hi
				dps := float64((lo + hi) / 2 * weaponData["AttackRate"])
				weaponData[dmgType+"DPS"] = dps
				if dmgType != "Physical" && dmgType != "Chaos" {
					weaponData["ElementalDPS"] += dps
				}
			}
		}
		weaponData["CritChance"] = round((w.CritChanceBase+calcLocal(&modList, "CritChance", mod.TypeBase, 0))*(1+(calcLocal(&modList, "CritChance", mod.TypeIncrease, 0)+math.Floor(quality/4*calcLocal(&modList, "AlternateQualityLocalCritChancePer4Quality", mod.TypeIncrease, 0)))/100), 2)
		for _, v := range listValues(modList, "WeaponData") {
			if k, val, ok := keyValue(v); ok {
				if f, ok := toFloat(val); ok {
					weaponData[k] = f
				} else if str, ok := val.(string); ok {
					if i.WeaponDataStr == nil {
						i.WeaponDataStr = map[int]map[string]string{}
					}
					if i.WeaponDataStr[slotNum] == nil {
						i.WeaponDataStr[slotNum] = map[string]string{}
					}
					i.WeaponDataStr[slotNum][k] = str
				}
			}
		}
		hand := "OffHandAttack"
		if slotNum == 1 {
			hand = "MainHandAttack"
		}
		for idx, m := range modList {
			name, tags := m.Name(), luaTags(m)
			if ((name == "Accuracy" && m.Flags() == 0) || (name == "CritMultiplier" && m.Flags() == 0) || (name == "ImpaleChance" && m.Flags() != mod.MFlagSpell) ||
				((name == "LifeOnHit" || name == "ManaOnHit") && m.Flags() == mod.MFlagAttack) ||
				((name == "PhysicalDamageLifeLeech" || name == "PhysicalDamageManaLeech") && m.Flags() == mod.MFlagAttack)) &&
				(m.KeywordFlags() == 0 || m.KeywordFlags() == mod.KeywordFlagAttack) && len(tags) == 0 {
				c := m.Clone()
				if len(c.Tags()) > 0 {
					c.Tags()[0] = mod.Condition(hand)
				} else {
					c.Tag(mod.Condition(hand))
				}
				modList[idx] = c
			} else if (name == "PoisonChance" || name == "BleedChance") && m.Flags() != mod.MFlagSpell && (len(tags) == 0 || (len(tags) == 1 && isConditionVar(tags[0], "CriticalStrike"))) {
				c := m.Clone()
				c.Tag(mod.Condition(hand))
				modList[idx] = c
			}
		}
		weaponData["TotalDPS"] = 0
		for _, dmgType := range dmgTypeList {
			weaponData["TotalDPS"] += weaponData[dmgType+"DPS"]
		}
	case i.Base.Armour != nil:
		//pob2:block:src/Classes/Item.lua:2561-2606
		a := i.Base.Armour
		armourData := i.ArmourData
		armourBase := calcLocal(&modList, "Armour", mod.TypeBase, 0) + a.Armour
		armourEvasionBase := calcLocal(&modList, "ArmourAndEvasion", mod.TypeBase, 0)
		evasionBase := calcLocal(&modList, "Evasion", mod.TypeBase, 0) + a.Evasion
		evasionEnergyShieldBase := calcLocal(&modList, "EvasionAndEnergyShield", mod.TypeBase, 0)
		energyShieldBase := calcLocal(&modList, "EnergyShield", mod.TypeBase, 0) + a.EnergyShield
		armourEnergyShieldBase := calcLocal(&modList, "ArmourAndEnergyShield", mod.TypeBase, 0)
		wardBase := calcLocal(&modList, "Ward", mod.TypeBase, 0) + a.Ward
		evasionPerLevel := calcLocal(&modList, "EvasionPerLevel", mod.TypeBase, 0)
		energyShieldPerLevel := calcLocal(&modList, "EnergyShieldPerLevel", mod.TypeBase, 0)
		wardPerLevel := calcLocal(&modList, "WardPerLevel", mod.TypeBase, 0)
		armourInc := calcLocal(&modList, "Armour", mod.TypeIncrease, 0)
		armourEvasionInc := calcLocal(&modList, "ArmourAndEvasion", mod.TypeIncrease, 0)
		evasionInc := calcLocal(&modList, "Evasion", mod.TypeIncrease, 0)
		evasionEnergyShieldInc := calcLocal(&modList, "EvasionAndEnergyShield", mod.TypeIncrease, 0)
		energyShieldInc := calcLocal(&modList, "EnergyShield", mod.TypeIncrease, 0)
		wardInc := calcLocal(&modList, "Ward", mod.TypeIncrease, 0)
		armourEnergyShieldInc := calcLocal(&modList, "ArmourAndEnergyShield", mod.TypeIncrease, 0)
		defencesInc := calcLocal(&modList, "Defences", mod.TypeIncrease, 0)
		qualityScalar := quality
		if calcLocal(&modList, "AlternateQualityArmour", mod.TypeBase, 0) > 0 {
			qualityScalar = 0
		}
		armourData["ArmourBase"] = a.Armour
		armourData["Armour"] = round((armourBase + armourEvasionBase + armourEnergyShieldBase) * (1 + (armourInc+armourEvasionInc+armourEnergyShieldInc+defencesInc)/100) * (1 + (qualityScalar / 100)))
		armourData["EvasionBase"] = a.Evasion
		armourData["Evasion"] = round((evasionBase + armourEvasionBase + evasionEnergyShieldBase) * (1 + (evasionInc+armourEvasionInc+evasionEnergyShieldInc+defencesInc)/100) * (1 + (qualityScalar / 100)))
		armourData["EnergyShieldBase"] = a.EnergyShield
		armourData["EnergyShield"] = round((energyShieldBase + evasionEnergyShieldBase + armourEnergyShieldBase) * (1 + (energyShieldInc+armourEnergyShieldInc+evasionEnergyShieldInc+defencesInc)/100) * (1 + (qualityScalar / 100)))
		armourData["WardBase"] = a.Ward
		armourData["Ward"] = round(wardBase * (1 + (wardInc+defencesInc)/100) * (1 + (qualityScalar / 100)))
		armourData["EvasionPerLevel"] = evasionPerLevel * (1 + (evasionInc+armourEvasionInc+evasionEnergyShieldInc+defencesInc)/100) * (1 + (qualityScalar / 100))
		armourData["EnergyShieldPerLevel"] = energyShieldPerLevel * (1 + (energyShieldInc+armourEnergyShieldInc+evasionEnergyShieldInc+defencesInc)/100) * (1 + (qualityScalar / 100))
		armourData["WardPerLevel"] = wardPerLevel * (1 + (wardInc+defencesInc)/100) * (1 + (qualityScalar / 100))
		if a.BlockChance != 0 {
			armourData["BlockChance"] = math.Floor((a.BlockChance + calcLocal(&modList, "BlockChance", mod.TypeBase, 0)) * (1 + calcLocal(&modList, "BlockChance", mod.TypeIncrease, 0)/100))
		}
		if a.MovementPenalty != 0 {
			modList = append(modList, mod.NewFloat("MovementSpeed", mod.TypeBase, -a.MovementPenalty).Source(mod.Source(i.modSourceName)).Tag(mod.Condition("IgnoreMovementPenalties").Neg(true)))
		}
		for _, v := range listValues(modList, "ArmourData") {
			if k, val, ok := keyValue(v); ok {
				if f, ok := toFloat(val); ok {
					armourData[k] = f
				}
			}
		}
	case i.Base.Flask != nil:
		//pob2:block:src/Classes/Item.lua:2607-2641
		f := i.Base.Flask
		flaskData := i.FlaskData
		flaskBase := append([]mod.Mod(nil), baseList...)
		durationInc := calcLocal(&modList, "Duration", mod.TypeIncrease, 0)
		durationMore := calcLocal(&modList, "Duration", mod.TypeMore, 0)
		if f.Life != 0 || f.Mana != 0 {
			flaskData["instantPerc"] = calcLocal(&modList, "FlaskInstantRecovery", mod.TypeBase, 0)
			recoveryMod := 1 + calcLocal(&modList, "FlaskRecovery", mod.TypeIncrease, 0)/100
			rateMod := 1 + calcLocal(&modList, "FlaskRecoveryRate", mod.TypeIncrease, 0)/100
			flaskData["duration"] = round(f.Duration*(1+durationInc/100)/rateMod*durationMore, 1)
			if f.Life != 0 {
				flaskData["lifeBase"] = f.Life * (1 + quality/100) * recoveryMod
				flaskData["lifeInstant"] = flaskData["lifeBase"] * flaskData["instantPerc"] / 100
				flaskData["lifeGradual"] = flaskData["lifeBase"] * (1 - flaskData["instantPerc"]/100)
				flaskData["lifeTotal"] = flaskData["lifeInstant"] + flaskData["lifeGradual"]
				flaskData["lifeAdditional"] = calcLocal(&modList, "FlaskAdditionalLifeRecovery", mod.TypeBase, 0)
				if calcLocalFlag(&flaskBase, "LifeFlaskEffectNotRemoved", 0) {
					flaskData["lifeEffectNotRemoved"] = 1
				}
			}
			if f.Mana != 0 {
				flaskData["manaBase"] = f.Mana * (1 + quality/100) * recoveryMod
				flaskData["manaInstant"] = flaskData["manaBase"] * flaskData["instantPerc"] / 100
				flaskData["manaGradual"] = flaskData["manaBase"] * (1 - flaskData["instantPerc"]/100)
				flaskData["manaTotal"] = flaskData["manaInstant"] + flaskData["manaGradual"]
				if calcLocalFlag(&flaskBase, "ManaFlaskEffectNotRemoved", 0) {
					flaskData["manaEffectNotRemoved"] = 1
				}
			}
		}
		flaskData["chargesMax"] = (f.ChargesMax + calcLocal(&modList, "FlaskCharges", mod.TypeBase, 0)) * (1 + calcLocal(&modList, "FlaskCharges", mod.TypeIncrease, 0)/100)
		flaskData["chargesUsed"] = math.Floor(f.ChargesUsed * (1 + calcLocal(&modList, "FlaskChargesUsed", mod.TypeIncrease, 0)/100))
		flaskData["gainBase"] = calcLocal(&modList, "FlaskChargesGenerated", mod.TypeBase, 0)
		flaskData["gainInc"] = calcLocal(&modList, "FlaskChargesGained", mod.TypeIncrease, 0)
		flaskData["gainMod"] = 1 + calcLocal(&modList, "FlaskChargeRecovery", mod.TypeIncrease, 0)/100
		flaskData["effectInc"] = calcLocal(&modList, "FlaskEffect", mod.TypeIncrease, 0) + calcLocal(&modList, "LocalEffect", mod.TypeIncrease, 0)
		for _, v := range listValues(modList, "FlaskData") {
			if k, val, ok := keyValue(v); ok {
				if fv, ok := toFloat(val); ok {
					flaskData[k] = fv
				}
			}
		}
	case i.Base.Charm != nil:
		//pob2:block:src/Classes/Item.lua:2642-2655
		c := i.Base.Charm
		charmData := i.CharmData
		durationInc := calcLocal(&modList, "Duration", mod.TypeIncrease, 0)
		durationMore := calcLocal(&modList, "Duration", mod.TypeMore, 0)
		charmData["duration"] = round(c.Duration*(1+durationInc/100)*(1+quality/100)*durationMore, 1)
		charmData["chargesMax"] = (c.ChargesMax + calcLocal(&modList, "FlaskCharges", mod.TypeBase, 0)) * (1 + calcLocal(&modList, "FlaskCharges", mod.TypeIncrease, 0)/100)
		charmData["chargesUsed"] = math.Floor(c.ChargesUsed * (1 + calcLocal(&modList, "FlaskChargesUsed", mod.TypeIncrease, 0)/100))
		charmData["gainBase"] = calcLocal(&modList, "FlaskChargesGenerated", mod.TypeBase, 0)
		charmData["gainInc"] = calcLocal(&modList, "FlaskChargesGained", mod.TypeIncrease, 0)
		charmData["gainMod"] = 1 + calcLocal(&modList, "FlaskChargeRecovery", mod.TypeIncrease, 0)/100
		charmData["effectInc"] = calcLocal(&modList, "CharmEffect", mod.TypeIncrease, 0) + calcLocal(&modList, "LocalEffect", mod.TypeIncrease, 0)
		for _, v := range listValues(modList, "CharmData") {
			if k, val, ok := keyValue(v); ok {
				if fv, ok := toFloat(val); ok {
					charmData[k] = fv
				}
			}
		}
	case i.Type == "Jewel":
		//pob2:block:src/Classes/Item.lua:2656-2684
		if strings.Contains(i.Name, "Grand Spectrum") {
			spectrumMod := mod.NewFloat("Multiplier:GrandSpectrum", mod.TypeBase, 1).Source(mod.Source(i.Name))
			modList = append(modList, spectrumMod, mod.NewList("MinionModifier", mod.MinionModifier{Mod: spectrumMod}).Source(mod.Source(i.Name)))
		}
		jewelData := i.JewelData
		funcList := listValues(modList, "JewelFunc")
		if funcList != nil {
			jewelData["funcList"] = funcList
		} else {
			delete(jewelData, "funcList")
		}
		for _, v := range listValues(modList, "JewelData") {
			if k, val, ok := keyValue(v); ok {
				jewelData[k] = val
			}
		}
		for _, v := range listValues(modList, "AlternateClassStart") {
			jewelData["alternateClassStart"] = v
		}
		fromNothing := map[string]any{}
		for _, v := range listValues(modList, "FromNothingKeystones") {
			if k, val, ok := keyValue(v); ok {
				fromNothing[k] = val
			}
		}
		jewelData["fromNothingKeystones"] = fromNothing
	}
	return modList
}

func isConditionVar(tag mod.Tag, v string) bool {
	if c, ok := tag.(*mod.ConditionTag); ok {
		return len(c.VarList) == 1 && c.VarList[0] == v
	}
	return false
}

//pob2:src/Classes/Item.lua:2716-2725
func (i *Item) BuildModListsForSlots(baseList []mod.Mod) {
	if i.Base.Weapon != nil || i.Base.Type == "Wand" || i.Base.Type == "Sceptre" || i.Base.Type == "Staff" || i.Type == "Ring" {
		n := 2
		if i.Type == "Ring" {
			n = 3
		}
		i.ModList = nil
		i.SlotModList = make([][]mod.Mod, n)
		for s := 1; s <= n; s++ {
			i.SlotModList[s-1] = i.BuildModListForSlotNum(baseList, s)
		}
		return
	}
	i.SlotModList = nil
	i.ModList = i.BuildModListForSlotNum(baseList, 0)
}

//pob2:src/Classes/Item.lua:2727-2737
func getRangedModList(item *Item, ml *ModLine) []mod.Mod {
	if ml.Range == nil || !rangedModLineRe.MatchString(ml.Line) {
		return nil
	}
	line := applyRange(strings.ReplaceAll(ml.Line, "\n", " "), *ml.Range, nil, ml.ValueScalar, ml.CorruptedRange, item.modSource)
	list, extra := item.modSource(line)
	if isZeroValueLine(line) {
		return []mod.Mod{}
	}
	if extra != "" {
		return nil
	}
	return list
}

var rangedModLineRe = regexp.MustCompile(`\((-?\d+\.?\d*)-(-?\d+\.?\d*)\)`)

// BuildModList builds the item's base mod list and the per-slot lists.
//
//pob2:src/Classes/Item.lua:2739-2908
//nolint:gocyclo // mirrors the Lua
func (i *Item) BuildModList() {
	if i.Base == nil {
		return
	}
	var baseList []mod.Mod
	switch {
	case i.Base.Weapon != nil:
		i.WeaponData = map[int]map[string]float64{}
		i.WeaponDataStr = nil
	case i.Base.Armour != nil:
		if i.ArmourData == nil {
			i.ArmourData = map[string]float64{}
		}
	case i.Base.Flask != nil:
		i.FlaskData = map[string]float64{}
		i.BuffModList = []mod.Mod{}
	case i.Base.Charm != nil:
		i.CharmData = map[string]float64{}
		i.BuffModList = []mod.Mod{}
	case i.Type == "Jewel":
		i.JewelData = map[string]any{}
	}
	i.RangeLines = nil
	i.modSourceName = "Item:" + strconv.Itoa(i.ID) + ":" + i.Name
	for _, ml := range i.BuffModLines {
		if ml.Extra == "" && i.CheckModLineVariant(ml) {
			for idx, m := range ml.ModList {
				ml.ModList[idx] = setSource(m, i.modSourceName)
				i.BuffModList = append(i.BuffModList, ml.ModList[idx])
			}
		}
	}
	processModLine := func(ml *ModLine) {
		ml.BondedModList = nil
		if ml.Disabled {
			return
		}
		variantCount := i.GetModLineVariantCount(ml)
		if variantCount <= 0 {
			return
		}
		if strings.Contains(ml.Line, "Requires Class") {
			if m := requiresClassRe.FindStringSubmatch(variantStripRe.ReplaceAllString(ml.Line, "")); m != nil {
				i.ClassRestriction = m[1]
			} else {
				i.ClassRestriction = ""
			}
		}
		if ml.SocketedAugmentTypeOverride != "" {
			i.SocketedAugmentTypeOverride = ml.SocketedAugmentTypeOverride
		} else if ml.SocketedSoulCoreType != "" {
			i.SocketedSoulCoreTypes[ml.SocketedSoulCoreType] = true
		}
		if ml.Extra != "" {
			return
		}
		target := &baseList
		if ml.Bonded {
			ml.BondedModList = []mod.Mod{}
			target = &ml.BondedModList
		}
		if ranged := getRangedModList(i, ml); ranged != nil {
			ml.ModList = ranged
			i.RangeLines = append(i.RangeLines, ml)
		}
		for idx, m := range ml.ModList {
			ml.ModList[idx] = setSource(m, i.modSourceName)
			for n := 0; n < variantCount; n++ {
				*target = append(*target, ml.ModList[idx])
			}
		}
		if len(ml.ModTags) > 0 {
			i.HasModTags = true
		}
	}
	i.SocketedAugmentTypeOverride = ""
	i.SocketedSoulCoreTypes = map[string]bool{}
	for _, lines := range [][]*ModLine{i.EnchantModLines, i.RuneModLines, i.ClassRequirementModLines, i.ImplicitModLines, i.ExplicitModLines} {
		for _, ml := range lines {
			processModLine(ml)
		}
	}
	i.SocketedIdolsUseBondedModifiers = calcLocalFlag(&baseList, "SocketedIdolsUseBondedModifiers", 0)
	i.SocketedSoulCoreEffectModifier = calcLocal(&baseList, "SocketedSoulCoreEffect", mod.TypeIncrease, 0) / 100
	i.SocketedRuneEffectModifier = calcLocal(&baseList, "SocketedRuneEffect", mod.TypeIncrease, 0) / 100
	i.SocketedAugmentItemEffectModifier = calcLocal(&baseList, "SocketedAugmentItemEffect", mod.TypeIncrease, 0) / 100
	if len(i.RuneModLines) > 0 {
		i.ApplySocketedRuneDisplayScalars()
	}
	for _, ml := range i.RuneModLines {
		effectModifier := i.SocketedAugmentItemEffectModifier
		switch ml.AugmentType {
		case "SoulCore":
			effectModifier += i.SocketedSoulCoreEffectModifier
		case "Rune":
			effectModifier += i.SocketedRuneEffectModifier
		}
		target := &baseList
		if ml.Bonded {
			if ml.BondedModList == nil {
				continue
			}
			target = &ml.BondedModList
		}
		if effectModifier != 0 && i.CheckModLineVariant(ml) && ml.Extra == "" && !ml.SocketedRuneEffectAlreadyApplied {
			for _, m := range ml.ModList {
				scaleAddMod(target, m, effectModifier)
			}
		}
	}
	i.GrantedSkill = nil
	for _, v := range listValues(baseList, "ExtraSkill") {
		skill, ok := v.(mod.ExtraSkill)
		if p, isPtr := v.(*mod.ExtraSkill); isPtr {
			skill, ok = *p, true
		}
		if !ok || skill.SkillName == "Unknown" {
			continue
		}
		i.GrantedSkill = append(i.GrantedSkill, ItemGrantedSkill{
			SkillID: skill.SkillID, Level: skill.Level, NoSupports: skill.NoSupports,
			NoReservation: i.Base.GrantedSkillsHaveNoReservation, Source: i.modSourceName,
			Triggered: skill.Triggered, TriggerChance: skill.TriggerChance,
		})
	}
	if flagValue(baseList, "JewelSocketRestriction") {
		i.CanSocketJewelBase = map[string]bool{
			"Diamond":  calcLocalFlag(&baseList, "CanSocketJewelBaseDiamond", 0),
			"Sapphire": calcLocalFlag(&baseList, "CanSocketJewelBaseSapphire", 0),
			"Emerald":  calcLocalFlag(&baseList, "CanSocketJewelBaseEmerald", 0),
			"Ruby":     calcLocalFlag(&baseList, "CanSocketJewelBaseRuby", 0),
		}
	}
	if i.Name == "Tabula Rasa, Simple Robe" || i.Name == "Skin of the Loyal, Simple Robe" || i.Name == "Skin of the Lords, Simple Robe" || i.Name == "The Apostate, Cabalist Regalia" {
		baseList = append(baseList, mod.NewList("ArmourData", mod.ArmourData{Key: "EnergyShield", Value: 0}))
		i.Requirements.Int = 0
	}
	if i.Name == "Geofri's Sanctuary, Revered Vestments" {
		baseList = append(baseList, mod.NewList("ArmourData", mod.ArmourData{Key: "EnergyShield", Value: 0}))
	}
	r := &i.Requirements
	switch {
	case calcLocalFlag(&baseList, "NoAttributeRequirements", 0):
		r.StrMod, r.DexMod, r.IntMod = floatp(0), floatp(0), floatp(0)
	case calcLocalFlag(&baseList, "AttributeRequirementsConverted", 0):
		strConversion := calcLocal(&baseList, "AttributeRequirementsConvertedToStrength", mod.TypeBase, 0) / 100
		dexConversion := calcLocal(&baseList, "AttributeRequirementsConvertedToDexterity", mod.TypeBase, 0) / 100
		intConversion := calcLocal(&baseList, "AttributeRequirementsConvertedToIntelligence", mod.TypeBase, 0) / 100
		r.IntBase = floatp(float64(intConversion*(r.Str+r.Dex)) + (r.Int + calcLocal(&baseList, "IntRequirement", mod.TypeBase, 0)) - float64(r.Int*(strConversion+dexConversion)))
		r.IntMod = floatp(math.Floor(*r.IntBase * (1 + calcLocal(&baseList, "IntRequirement", mod.TypeIncrease, 0)/100)))
		r.DexBase = floatp(float64(dexConversion*(r.Str+r.Int)) + (r.Dex + calcLocal(&baseList, "DexRequirement", mod.TypeBase, 0)) - float64(r.Dex*(strConversion+intConversion)))
		r.DexMod = floatp(math.Floor(*r.DexBase * (1 + calcLocal(&baseList, "DexRequirement", mod.TypeIncrease, 0)/100)))
		r.StrBase = floatp(float64(strConversion*(r.Int+r.Dex)) + (r.Str + calcLocal(&baseList, "StrRequirement", mod.TypeBase, 0)) - float64(r.Str*(dexConversion+intConversion)))
		r.StrMod = floatp(math.Floor(*r.StrBase * (1 + calcLocal(&baseList, "StrRequirement", mod.TypeIncrease, 0)/100)))
	default:
		r.StrMod = floatp(math.Floor((r.Str + calcLocal(&baseList, "StrRequirement", mod.TypeBase, 0)) * (1 + calcLocal(&baseList, "StrRequirement", mod.TypeIncrease, 0)/100)))
		r.DexMod = floatp(math.Floor((r.Dex + calcLocal(&baseList, "DexRequirement", mod.TypeBase, 0)) * (1 + calcLocal(&baseList, "DexRequirement", mod.TypeIncrease, 0)/100)))
		r.IntMod = floatp(math.Floor((r.Int + calcLocal(&baseList, "IntRequirement", mod.TypeBase, 0)) * (1 + calcLocal(&baseList, "IntRequirement", mod.TypeIncrease, 0)/100)))
	}
	if i.ItemSocketCount > 0 {
		sockets := make([]ItemSocket, i.ItemSocketCount)
		for n := range sockets {
			sockets[n] = ItemSocket{Group: n + 1}
		}
		i.Sockets = sockets
	}
	i.SocketedJewelEffectModifier = 1 + calcLocal(&baseList, "SocketedJewelEffect", mod.TypeIncrease, 0)/100
	i.BaseModList = baseList
	i.BuildModListsForSlots(baseList)
	i.activeBondedSet, i.activeBonded = true, ""
}

// GetActiveModListForSlotNum returns the item's mods for a slot (slotNum 0 is Lua's nil), adding the
// Bonded mods the global unlock (canUseBonded) or the item's own Idol unlock enables.
//
//pob2:src/Classes/Item.lua:2232-2254
func (i *Item) GetActiveModListForSlotNum(slotNum int, canUseBonded bool) []mod.Mod {
	bondedState := ""
	if canUseBonded {
		bondedState = "all"
	} else if i.SocketedIdolsUseBondedModifiers {
		bondedState = "idol"
	}
	if !i.activeBondedSet || i.activeBonded != bondedState {
		baseList := i.BaseModList
		var activeBaseList []mod.Mod
		if bondedState != "" {
			for _, ml := range i.RuneModLines {
				canUseBondedMod := ml.Bonded && (bondedState == "all" || ml.AugmentType == "Idol")
				if canUseBondedMod && len(ml.BondedModList) > 0 {
					activeBaseList = append(activeBaseList, ml.BondedModList...)
				}
			}
		}
		if activeBaseList != nil {
			baseList = append(activeBaseList, baseList...)
		}
		i.BuildModListsForSlots(baseList)
		i.activeBondedSet = true
		i.activeBonded = bondedState
	}
	if i.ModList != nil || slotNum < 1 || slotNum > len(i.SlotModList) {
		return i.ModList
	}
	return i.SlotModList[slotNum-1]
}
