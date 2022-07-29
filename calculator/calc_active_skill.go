package calculator

// CreateActiveSkill Create an active skill using the given active gem and list of support gems
// It will determine the base flag set, and check which of the support gems can support this skill
func CreateActiveSkill(activeEffect *ActiveEffect, supportList []interface{}, actor *Actor, socketGroup interface{}, summonSkill interface{}) *ActiveSkill {
	activeSkill := &ActiveSkill{
		ActiveEffect: activeEffect,
		SupportList:  supportList,
		SkillData:    make(map[string]interface{}),
		Actor:        actor,
		SocketGroup:  socketGroup,
		SummonSkill:  summonSkill,
	}

	/*
		TODO -- Initialise skill types
		activeSkill.skillTypes = copyTable(activeGrantedEffect.skillTypes)
		if activeGrantedEffect.minionSkillTypes then
			activeSkill.minionSkillTypes = copyTable(activeGrantedEffect.minionSkillTypes)
		end
	*/
	/*
		TODO -- Initialise skill flag set ('attack', 'projectile', etc)
		local skillFlags = copyTable(activeGrantedEffect.baseFlags)
		activeSkill.skillFlags = skillFlags
		skillFlags.hit = skillFlags.hit or activeSkill.skillTypes[SkillType.Attack] or activeSkill.skillTypes[SkillType.Damage] or activeSkill.skillTypes[SkillType.Projectile]
	*/
	/*
		TODO -- Process support skills
		activeSkill.effectList = { activeEffect }
		for _, supportEffect in ipairs(supportList) do
			-- Pass 1: Add skill types from compatible supports
			if calcLib.canGrantedEffectSupportActiveSkill(supportEffect.grantedEffect, activeSkill) then
				for _, skillType in pairs(supportEffect.grantedEffect.addSkillTypes) do
					activeSkill.skillTypes[skillType] = true
				end
			end
		end
		for _, supportEffect in ipairs(supportList) do
			-- Pass 2: Add all compatible supports
			if calcLib.canGrantedEffectSupportActiveSkill(supportEffect.grantedEffect, activeSkill) then
				t_insert(activeSkill.effectList, supportEffect)
				if supportEffect.isSupporting and activeEffect.srcInstance then
					supportEffect.isSupporting[activeEffect.srcInstance] = true
				end
				if supportEffect.grantedEffect.addFlags and not summonSkill then
					-- Support skill adds flags to supported skills (eg. Remote Mine adds 'mine')
					for k in pairs(supportEffect.grantedEffect.addFlags) do
						skillFlags[k] = true
					end
				end
			end
		end
	*/

	return activeSkill
}
