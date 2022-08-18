package calculator

import (
	"math"

	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/utils"
)

type ListCfg struct {
	Flags        *mod.MFlag
	KeywordFlags *mod.KeywordFlag
	Source       *mod.Source
	SkillStats   map[string]float64
	SkillCond    map[string]bool
	SlotName     string
}

type ModStoreFuncs interface {
	List(cfg *ListCfg, names ...string) []interface{}
	Sum(modType mod.Type, cfg *ListCfg, names ...string) float64
	More(cfg *ListCfg, names ...string) float64
	Flag(cfg *ListCfg, names ...string) bool
	Override(cfg *ListCfg, names ...string) interface{}
	GetMultiplier(variable string, cfg *ListCfg, noMod bool) float64
	GetCondition(variable string, cfg *ListCfg, noMod bool) (bool, bool)
	Clone() ModStoreFuncs
}

type ModStore struct {
	Parent      ModStoreFuncs
	Child       ModStoreFuncs
	Actor       *Actor `json:"-"`
	Multipliers map[string]float64
	Conditions  map[string]bool
}

func NewModStore(parent ModStoreFuncs) *ModStore {
	return &ModStore{
		Parent:      parent,
		Multipliers: make(map[string]float64),
		Conditions:  make(map[string]bool),
	}
}

func (s *ModStore) Clone() *ModStore {
	if s == nil {
		return nil
	}

	var parent ModStoreFuncs = nil
	if s.Parent != nil {
		parent = s.Parent.Clone()
	}

	out := NewModStore(parent)
	out.Actor = s.Actor
	out.Multipliers = utils.CopyMap(out.Multipliers)
	out.Conditions = utils.CopyMap(out.Conditions)
	return out
}

func (s *ModStore) EvalMod(m mod.Mod, cfg *ListCfg) interface{} {
	value := m.Value()

	if len(m.Tags()) == 0 {
		return value
	}

	for _, raw := range m.Tags() {
		switch tag := raw.(type) {
		case *mod.MultiplierTag:
			target := s
			limitTarget := s

			// Allow limiting a self multiplier on a parent multiplier (eg. Agony Crawler on player virulence)
			// This explicit target is necessary because even though the GetMultiplier method does call self.parent.GetMultiplier, it does so with noMod = true,
			// disabling the summation (3rd part): (not noMod and self:Sum("BASE", cfg, multiplierName[var]) or 0)

			/*
				TODO Limit Actor
				if tag.limitActor then
					if self.actor[tag.limitActor] then
						limitTarget = self.actor[tag.limitActor].modDB
					else
						return
					end
				end
			*/

			/*
				TODO Actor
				if tag.actor then
					if self.actor[tag.actor] then
						target = self.actor[tag.actor].modDB
					else
						return
					end
				end
			*/

			base := target.GetMultiplier(tag.Variable, cfg, false)
			/*
				TODO VarList
				if tag.varList then
					for _, var in pairs(tag.varList) do
						base = base + target:GetMultiplier(var, cfg)
					end
				else
					base = target:GetMultiplier(tag.var, cfg)
				end
			*/

			mult := math.Floor(base/tag.Division + 0.0001)
			var limitTotal *float64
			if tag.TagLimit != nil || tag.TagLimitVariable != nil {
				limit := float64(0)
				if tag.TagLimit != nil {
					limit = *tag.TagLimit
				} else {
					limit = limitTarget.GetMultiplier(*tag.TagLimitVariable, cfg, false)
				}

				if tag.LimitTotal {
					limitTotal = &limit
				} else {
					mult = utils.Min(mult, limit)
				}
			}

			if v, ok := value.(float64); ok {
				out := v*mult + tag.Base
				if limitTotal != nil {
					out = utils.Min(out, *limitTotal)
				}
				value = out
			} else {
				/*
					TODO Non-number multiplier
					value = copyTable(value)
					if value.mod then
						value.mod.value = value.mod.value * mult + (tag.base or 0)
						if limitTotal then
							value.mod.value = m_min(value.mod.value, limitTotal)
						end
					else
						value.value = value.value * mult + (tag.base or 0)
						if limitTotal then
							value.value = m_min(value.value, limitTotal)
						end
					end
				*/
			}
		case *mod.MultiplierThresholdTag:
			target := s
			/*
				TODO Actor
				if tag.actor then
					if self.actor[tag.actor] then
						target = self.actor[tag.actor].modDB
					else
						return
					end
				end
			*/

			mult := target.GetMultiplier(tag.Variable, cfg, false)
			/*
				TODO VarList
				if tag.varList then
					for _, var in pairs(tag.varList) do
						mult = mult + target:GetMultiplier(var, cfg)
					end
				else
					mult = target:GetMultiplier(tag.var, cfg)
				end
			*/

			threshold := float64(0)
			if tag.TagThreshold != nil {
				threshold = *tag.TagThreshold
			} else {
				threshold = target.GetMultiplier(*tag.ThresholdVariable, cfg, false)
			}

			if (tag.TagUpper && mult > threshold) || (!tag.TagUpper && mult < threshold) {
				return nil
			}
		case *mod.PerStatTag:
			base := float64(0)
			target := s

			// This functions similar to the above tagTypes in regard to which actor to use, but for PerStat
			// if the actor is 'parent', we don't want to return if we're already using 'parent', just keep using 'self'

			/*
				TODO Actor
				if tag.actor and self.actor[tag.actor] then
					target = self.actor[tag.actor].modDB
				end
			*/

			base = target.GetStat(tag.Stat, cfg)
			/*
				TODO Stat List
				if tag.statList then
					base = 0
					for _, stat in ipairs(tag.statList) do
						base = base + target:GetStat(stat, cfg)
					end
				else
					base = target:GetStat(tag.stat, cfg)
				end
			*/

			mult := math.Floor(base/tag.Divide + 0.0001)
			var limitTotal *float64
			if tag.TagLimit != nil || tag.TagLimitVariable != nil {
				limit := float64(0)
				if tag.TagLimit != nil {
					limit = *tag.TagLimit
				} else {
					limit = s.GetMultiplier(*tag.TagLimitVariable, cfg, false)
				}

				if tag.LimitTotal {
					limitTotal = &limit
				} else {
					mult = utils.Min(mult, limit)
				}
			}

			if v, ok := value.(float64); ok {
				out := v*mult + tag.Base
				if limitTotal != nil {
					out = utils.Min(out, *limitTotal)
				}
				value = out
			} else {
				/*
					TODO Non-number multiplier
					value = copyTable(value)
					if value.mod then
						value.mod.value = value.mod.value * mult + (tag.base or 0)
						if limitTotal then
							value.mod.value = m_min(value.mod.value, limitTotal)
						end
					else
						value.value = value.value * mult + (tag.base or 0)
						if limitTotal then
							value.value = m_min(value.value, limitTotal)
						end
					end
				*/
			}
			/*
				TODO PercentStat
				case *mod.PercentStatTag:
					local base
					if tag.statList then
						base = 0
						for _, stat in ipairs(tag.statList) do
							base = base + self:GetStat(stat, cfg)
						end
					else
						base = self:GetStat(tag.stat, cfg)
					end
					local mult = base * (tag.percent and tag.percent / 100 or 1)
					local limitTotal
					if tag.limit or tag.limitVar then
						local limit = tag.limit or self:GetMultiplier(tag.limitVar, cfg)
						if tag.limitTotal then
							limitTotal = limit
						else
							mult = m_min(mult, limit)
						end
					end
					if type(value) == "table" then
						value = copyTable(value)
						if value.mod then
							value.mod.value = value.mod.value * mult + (tag.base or 0)
							if limitTotal then
								value.mod.value = m_min(value.mod.value, limitTotal)
							end
						else
							value.value = value.value * mult + (tag.base or 0)
							if limitTotal then
								value.value = m_min(value.value, limitTotal)
							end
						end
					else
						value = value * mult + (tag.base or 0)
						if limitTotal then
							value = m_min(value, limitTotal)
						end
					end
			*/
			/*
				TODO StatThreshold
				case *mod.StatThresholdTag:
					local stat
					if tag.statList then
						stat = 0
						for _, stat in ipairs(tag.statList) do
							stat = stat + self:GetStat(stat, cfg)
						end
					else
						stat = self:GetStat(tag.stat, cfg)
					end
					local threshold = tag.threshold or self:GetStat(tag.thresholdStat, cfg)
					if (tag.upper and stat > threshold) or (not tag.upper and stat < threshold) then
						return
					end
			*/
			/*
				TODO DistanceRampTag
				case *mod.DistanceRampTag:
					if not cfg or not cfg.skillDist then
						return
					end
					if cfg.skillDist <= tag.ramp[1][1] then
						value = value * tag.ramp[1][2]
					elseif cfg.skillDist >= tag.ramp[#tag.ramp][1] then
						value = value * tag.ramp[#tag.ramp][2]
					else
						for i, dat in ipairs(tag.ramp) do
							local next = tag.ramp[i+1]
							if cfg.skillDist <= next[1] then
								value = value * (dat[2] + (next[2] - dat[2]) * (cfg.skillDist - dat[1]) / (next[1] - dat[1]))
								break
							end
						end
					end
			*/
			/*
				TODO MeleeProximityTag
				case *mod.MeleeProximityTag:
					if not cfg or not cfg.skillDist then
						return
					end
					-- Max potency is 0-15 units of distance
					if cfg.skillDist <= 15 then
						value = value * tag.ramp[1]
					-- Reduced potency (linear) until 40 units
					elseif cfg.skillDist >= 16 and cfg.skillDist <= 39 then
						value = value * (tag.ramp[1] - ((tag.ramp[1] / 25) * (cfg.skillDist - 15)))
					elseif cfg.skillDist >= 40 then
						value = 0
					end
			*/
			/*
				TODO LimitTag
				case *mod.LimitTag:
					value = m_min(value, tag.limit or self:GetMultiplier(tag.limitVar, cfg))
			*/
		case *mod.ConditionTag:
			match, ok := s.GetCondition(tag.Variable, cfg, false)

			if !ok && cfg != nil && cfg.SkillCond != nil {
				if c, ok := cfg.SkillCond[tag.Variable]; ok {
					match = c
				}
			}

			/*
				TODO VarList
				if tag.varList then
					for _, var in pairs(tag.varList) do
						if self:GetCondition(var, cfg) or (cfg and cfg.skillCond and cfg.skillCond[var]) then
							match = true
							break
						end
					end
				end
			*/

			if tag.Negative {
				match = !match
			}

			if !match {
				return nil
			}
		case *mod.ActorConditionTag:
			target := s

			if tag.Actor != nil {
				// TODO Tag Actor
				// target = self.actor[tag.actor] and self.actor[tag.actor].modDB
			}

			match, ok := target.GetCondition(tag.Variable, cfg, false)

			if !ok && cfg != nil && cfg.SkillCond != nil {
				if c, ok := cfg.SkillCond[tag.Variable]; ok {
					match = c
				}
			}

			/*
				TODO VarList
				if tag.varList then
					for _, var in pairs(tag.varList) do
						if self:GetCondition(var, cfg) or (cfg and cfg.skillCond and cfg.skillCond[var]) then
							match = true
							break
						end
					end
				end
			*/

			if tag.Negative {
				match = !match
			}

			if !match {
				return nil
			}
			/*
				TODO SocketedIn
				case *mod.SocketedInTag:
					if not cfg or tag.slotName ~= cfg.slotName or (tag.keyword and (not cfg or not cfg.skillGem or not calcLib.gemIsType(cfg.skillGem, tag.keyword))) then
						return
					end
			*/
			/*
				TODO SkillName
				case *mod.SkillNameTag:
					local match = false
					local matchName = tag.summonSkill and (cfg and cfg.summonSkillName or "") or (cfg and cfg.skillName)
					if tag.skillNameList then
						for _, name in pairs(tag.skillNameList) do
							if name == matchName then
								match = true
								break
							end
						end
					else
						match = (tag.skillName == matchName)
					end
					if tag.neg then
						match = not match
					end
					if not match then
						return
					end
			*/
			/*
				TODO SkillId
				case *mod.SkillIdTag:
					if not cfg or not cfg.skillGrantedEffect or cfg.skillGrantedEffect.id ~= tag.skillId then
						return
					end
			*/
			/*
				TODO SkillPart
				case *mod.SkillPartTag:
					if not cfg then
						return
					end
					local match = false
					if tag.skillPartList then
						for _, part in ipairs(tag.skillPartList) do
							if part == cfg.skillPart then
								match = true
								break
							end
						end
					else
						match = (tag.skillPart == cfg.skillPart)
					end
					if tag.neg then
						match = not match
					end
					if not match then
						return
					end
			*/
			/*
				TODO SkillType
				case *mod.SkillTypeTag:
					local match = false
					if tag.skillTypeList then
						for _, type in pairs(tag.skillTypeList) do
							if cfg and cfg.skillTypes and cfg.skillTypes[type] then
								match = true
								break
							end
						end
					else
						match = cfg and cfg.skillTypes and cfg.skillTypes[tag.skillType]
					end
					if tag.neg then
						match = not match
					end
					if not match then
						return
					end
			*/
			/*
				TODO SlotName
				case *mod.SlotNameTag:
					if not cfg then
						return
					end
					local match = false
					if tag.slotNameList then
						for _, slot in ipairs(tag.slotNameList) do
							if slot == cfg.slotName then
								match = true
								break
							end
						end
					else
						match = (tag.slotName == cfg.slotName)
					end
					if tag.neg then
						match = not match
					end
					if not match then
						return
					end
			*/
			/*
				TODO ModFlagOr
				case *mod.ModFlagOrTag:
					if not cfg or not cfg.flags then
						return
					end
					if band(cfg.flags, tag.modFlags) == 0 then
						return
					end
			*/
			/*
				TODO KeywordFlagAnd
				case *mod.KeywordFlagAndTag:
					if not cfg or not cfg.keywordFlags then
						return
					end
					if band(cfg.keywordFlags, tag.keywordFlags) ~= tag.keywordFlags then
						return
					end
			*/
		}
	}

	return value
}

func (s *ModStore) GetMultiplier(variable string, cfg *ListCfg, noMod bool) float64 {
	out := float64(0)

	if mul, ok := s.Multipliers[variable]; ok {
		out += mul
	}

	if s.Parent != nil {
		out += s.Parent.GetMultiplier(variable, cfg, true)
	}

	if !noMod {
		// TODO ???
		// out += s.Sum(mod.ModTypeBase, cfg, multiplierName[variable])
	}

	return out
}

func (s *ModStore) GetStat(stat string, cfg *ListCfg) float64 {
	/*
		TODO Mana handling
		if stat == "ManaReservedPercent" then
			local reservedPercentMana = 0
			for _, activeSkill in ipairs(self.actor.activeSkillList) do
				if (activeSkill.skillTypes[SkillType.Aura] and not activeSkill.skillFlags.disable and activeSkill.buffList and activeSkill.buffList[1] and activeSkill.buffList[1].name == cfg.skillName) then
					local manaBase = activeSkill.skillData["ManaReservedBase"] or 0
					reservedPercentMana = manaBase / self.actor.output["Mana"] * 100
					break
				end
			end
			return m_min(reservedPercentMana, 100) --Don't let people get more than 100% reservation for aura effect.
		end
		-- if ReservationEfficiency is -100, ManaUnreserved is nan which breaks everything if Arcane Cloak is enabled
		if stat == "ManaUnreserved" and self.actor.output[stat] ~= self.actor.output[stat] then
			-- 0% reserved = total mana
			return self.actor.output["Mana"]
		elseif stat == "ManaUnreserved" and not self.actor.output[stat] == nil and self.actor.output[stat] < 0 then
			-- This reverse engineers how much mana is unreserved before efficiency for accurate Arcane Cloak calcs
			local reservedPercentBeforeEfficiency = (math.abs(self.actor.output["ManaUnreservedPercent"]) + 100) * ((100 + self.actor["ManaEfficiency"]) / 100)
			return self.actor.output["Mana"] * (math.ceil(reservedPercentBeforeEfficiency) / 100);
		end
	*/

	if v, ok := s.Actor.Output[stat]; ok {
		return v
	}

	if cfg != nil && cfg.SkillStats != nil {
		if v, ok := cfg.SkillStats[stat]; ok {
			return v
		}
	}

	return 0
}

func (s *ModStore) GetCondition(variable string, cfg *ListCfg, noMod bool) (bool, bool) {
	out := false

	if cond, ok := s.Conditions[variable]; ok {
		return cond, true
	}

	if s.Parent != nil {
		var ok bool
		out, ok = s.Parent.GetCondition(variable, cfg, true)
		if ok {
			return out, true
		}
	}

	if !noMod {
		if s.Child.Flag(cfg, "Condition:"+variable) {
			return true, true
		}
	}

	return out, false
}
