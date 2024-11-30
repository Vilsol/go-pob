package calculator

import (
	"strconv"
	"strings"

	"github.com/Vilsol/go-pob-data/poe"

	"github.com/Vilsol/go-pob/data"
	"github.com/Vilsol/go-pob/mod"
	"github.com/Vilsol/go-pob/pob"
	"github.com/Vilsol/go-pob/utils"
)

func InitEnv(build *pob.PathOfBuilding, envCache *EnvironmentCache, mode OutputMode) (*Environment, ModStoreFuncs, ModStoreFuncs, ModStoreFuncs) {
	env := &Environment{}
	env.Cache = envCache

	// TODO This come from the build at some point but below it's not so not sure what to do here yet
	currentTreeVersion := data.LatestTreeVersion

	// Clear Node Mod cache if tree has changed
	if env.Cache.TreeVersion != currentTreeVersion {
		env.Cache.modsForNodes = make(map[string]ModList, len(data.TreeVersions[currentTreeVersion].Tree().Nodes))
		env.Cache.TreeVersion = currentTreeVersion
	}

	env.DebugErrors = make([]string, 0)
	env.Build = build
	env.Mode = mode
	env.Spec = NewPassiveSpec(build, data.LatestTreeVersion)

	env.ModDB = NewModDB()
	env.EnemyModDB = NewModDB()
	env.ItemModDB = NewModDB()

	// TODO m_max(1, m_min(100, env.configInput.enemyLevel and env.configInput.enemyLevel or env.configPlaceholder["enemyLevel"] or m_min(env.build.characterLevel, data.misc.MaxEnemyLevel)))
	env.EnemyLevel = 1

	env.Player = &Actor{
		ModDB:           env.ModDB,
		Level:           build.Build.Level,
		ActiveSkillList: make([]*ActiveSkill, 0),
	}

	env.Enemy = &Actor{
		ModDB: env.EnemyModDB,
		Level: env.EnemyLevel,
	}

	env.ModDB.Actor = env.Player
	env.EnemyModDB.Actor = env.Enemy

	env.Player.Enemy = env.Enemy
	env.Enemy.Enemy = env.Player

	env.RequirementsTableItems = make(map[string]interface{})
	env.RequirementsTableGems = make([]*RequirementsTableGems, 0)

	env.RadiusJewelList = make(map[string]interface{})
	env.ExtraRadiusNodeList = make(map[string]interface{})
	env.GrantedSkills = make(map[string]interface{})
	env.GrantedSkillsNodes = make(map[string]interface{})
	env.GrantedSkillsItems = make(map[string]interface{})
	env.Flasks = make(map[string]interface{})

	env.GrantedPassives = make(map[string]interface{})

	env.AuxSkillList = make(map[string]interface{})

	buffMode := BuffModeEffective
	if mode == OutputModeCalcs {
		// TODO env.calcsInput.misc_buffMode
	}

	switch buffMode {
	case BuffModeEffective:
		env.ModeEffective = true
		fallthrough
	case BuffModeCombat:
		env.ModeCombat = true
		fallthrough
	case BuffModeBuffed:
		env.ModeBuffs = true
	}

	classStats := env.Spec.Class()

	env.ModDB.AddMod(mod.NewFloat("Str", mod.TypeBase, float64(classStats.BaseStr)).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("Dex", mod.TypeBase, float64(classStats.BaseDex)).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("Int", mod.TypeBase, float64(classStats.BaseInt)).Source("Base"))

	env.ModDB.Multipliers["Level"] = max(1, min(100, float64(build.Build.Level)))
	initModDB(env, env.ModDB)

	env.ModDB.AddMod(mod.NewFloat("Life", mod.TypeBase, 12).Source("Base").Tag(mod.Multiplier("Level").Base(38)))
	env.ModDB.AddMod(mod.NewFloat("Mana", mod.TypeBase, 6).Source("Base").Tag(mod.Multiplier("Level").Base(34)))
	env.ModDB.AddMod(mod.NewFloat("ManaRegen", mod.TypeBase, 0.0175).Source("Base").Tag(mod.PerStat(1, "Mana")))
	env.ModDB.AddMod(mod.NewFloat("Devotion", mod.TypeBase, 0).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("Evasion", mod.TypeBase, 15).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("Accuracy", mod.TypeBase, 2).Source("Base").Tag(mod.Multiplier("Level").Base(-2)))
	env.ModDB.AddMod(mod.NewFloat("CritMultiplier", mod.TypeBase, 50).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("DotMultiplier", mod.TypeBase, 50).Source("Base").Tag(mod.Condition("CriticalStrike")))
	env.ModDB.AddMod(mod.NewFloat("FireResist", mod.TypeBase /* TODO env.configInput.resistancePenalty or */, -60).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("ColdResist", mod.TypeBase /* TODO env.configInput.resistancePenalty or */, -60).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("LightningResist", mod.TypeBase /* TODO env.configInput.resistancePenalty or */, -60).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("ChaosResist", mod.TypeBase /* TODO env.configInput.resistancePenalty or */, -60).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("TotemFireResist", mod.TypeBase, 40).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("TotemColdResist", mod.TypeBase, 40).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("TotemLightningResist", mod.TypeBase, 40).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("TotemChaosResist", mod.TypeBase, 20).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("CritChance", mod.TypeIncrease, 40).Source("Base").Tag(mod.Multiplier("PowerCharge").Base(0)))
	env.ModDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 4).Source("Base").Tag(mod.Multiplier("FrenzyCharge").Base(0)))
	env.ModDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, 4).Source("Base").Tag(mod.Multiplier("FrenzyCharge").Base(0)))
	env.ModDB.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, 4).Source("Base").Tag(mod.Multiplier("EnduranceCharge").Base(0)))
	env.ModDB.AddMod(mod.NewFloat("ElementalResist", mod.TypeBase, 4).Source("Base").Tag(mod.Multiplier("EnduranceCharge").Base(0)))
	env.ModDB.AddMod(mod.NewFloat("Multiplier:RageEffect", mod.TypeBase, 1).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 1).Source("Base").Flag(mod.MFlagAttack).Tag(mod.Multiplier("Rage").Base(0)).Tag(mod.Multiplier("RageEffect").Base(0)))
	env.ModDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 1).Source("Base").Flag(mod.MFlagAttack).Tag(mod.Multiplier("Rage").Base(0).Div(2)).Tag(mod.Multiplier("RageEffect").Base(0)))
	env.ModDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, 1).Source("Base").Tag(mod.Multiplier("Rage").Base(0).Div(5)).Tag(mod.Multiplier("RageEffect").Base(0)))
	env.ModDB.AddMod(mod.NewFloat("MaximumRage", mod.TypeBase, 50).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("Multiplier:GaleForce", mod.TypeBase, 0).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("MaximumGaleForce", mod.TypeBase, 10).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("MaximumFortification", mod.TypeBase, 20).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("Multiplier:IntensityLimit", mod.TypeBase, 3).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("Damage", mod.TypeIncrease, 2).Source("Base").Tag(mod.Multiplier("Rampage").Base(0).Limit(50).Div(20)))
	env.ModDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, 1).Source("Base").Tag(mod.Multiplier("Rampage").Base(0).Limit(50).Div(20)))
	env.ModDB.AddMod(mod.NewFloat("ActiveTrapLimit", mod.TypeBase, 15).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("ActiveMineLimit", mod.TypeBase, 15).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("ActiveBrandLimit", mod.TypeBase, 3).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("EnemyCurseLimit", mod.TypeBase, 1).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("SocketedCursesHexLimitValue", mod.TypeBase, 1).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("ProjectileCount", mod.TypeBase, 1).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("Speed", mod.TypeMore, 10).Source("Base").Flag(mod.MFlagAttack).Tag(mod.Condition("DualWielding")))
	env.ModDB.AddMod(mod.NewFloat("BlockChance", mod.TypeBase, 15).Source("Base").Tag(mod.Condition("DualWielding")).Tag(mod.Condition("NoInherentBlock").Neg(true)))
	env.ModDB.AddMod(mod.NewFloat("Damage", mod.TypeMore, 200).Source("Base").KeywordFlag(mod.KeywordFlagBleed).Tag(mod.ActorCondition("enemy", "Moving")).Tag(mod.Condition("NoExtraBleedDamageToMovingEnemy").Neg(true)))
	env.ModDB.AddMod(mod.NewFlag("Condition:BloodStance", true).Source("Base").Tag(mod.Condition("SandStance").Neg(true)))
	env.ModDB.AddMod(mod.NewFlag("Condition:PrideMinEffect", true).Source("Base").Tag(mod.Condition("PrideMaxEffect").Neg(true)))
	env.ModDB.AddMod(mod.NewFloat("PerBrutalTripleDamageChance", mod.TypeBase, 3).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("PerAfflictionAilmentDamage", mod.TypeBase, 8).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("PerAfflictionNonDamageEffect", mod.TypeBase, 8).Source("Base"))
	env.ModDB.AddMod(mod.NewFloat("Multiplier:AllocatedNotable", mod.TypeBase, float64(env.Spec.AllocatedNotableCount)))
	env.ModDB.AddMod(mod.NewFloat("Multiplier:AllocatedMastery", mod.TypeBase, float64(env.Spec.AllocatedMasteryCount)))

	// Bandit mods
	switch build.GetStringOption("bandit") {
	case "Alira":
		env.ModDB.AddMod(mod.NewFloat("ManaRegen", mod.TypeBase, 5).Source("Bandit"))
		env.ModDB.AddMod(mod.NewFloat("CritMultiplier", mod.TypeBase, 20).Source("Bandit"))
		env.ModDB.AddMod(mod.NewFloat("ElementalResist", mod.TypeBase, 15).Source("Bandit"))
	case "Kraityn":
		env.ModDB.AddMod(mod.NewFloat("Speed", mod.TypeIncrease, 6).Source("Bandit"))
		env.ModDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, 6).Source("Bandit"))
		for _, ailment := range data.ElementalAilment("").Values() {
			env.ModDB.AddMod(mod.NewFloat("Avoid"+string(ailment), mod.TypeBase, 10).Source("Bandit"))
		}
	case "Oak":
		env.ModDB.AddMod(mod.NewFloat("LifeRegenPercent", mod.TypeBase, 1).Source("Bandit"))
		env.ModDB.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, 2).Source("Bandit"))
		env.ModDB.AddMod(mod.NewFloat("PhysicalDamage", mod.TypeIncrease, 20).Source("Bandit"))
	default:
		env.ModDB.AddMod(mod.NewFloat("ExtraPoints", mod.TypeBase, 2).Source("Bandit"))
	}

	/*
		TODO Implement pantheon mods
		-- Add Pantheon mods
		local parser = modLib.parseMod
		-- Major Gods
		if env.configInput.pantheonMajorGod ~= "None" then
			local majorGod = env.data.pantheons[env.configInput.pantheonMajorGod]
			pantheon.applySoulMod(modDB, parser, majorGod)
		end
		-- Minor Gods
		if env.configInput.pantheonMinorGod ~= "None" then
			local minorGod = env.data.pantheons[env.configInput.pantheonMinorGod]
			pantheon.applySoulMod(modDB, parser, minorGod)
		end
	*/

	// Initialise enemy modifier database
	initModDB(env, env.EnemyModDB)
	env.EnemyModDB.AddMod(mod.NewFloat("Accuracy", mod.TypeBase, data.MonsterAccuracyTable[env.EnemyLevel]).Source("Base"))
	env.EnemyModDB.AddMod(mod.NewFloat("Evasion", mod.TypeBase, data.MonsterEvasionTable[env.EnemyLevel]).Source("Base"))
	env.EnemyModDB.AddMod(mod.NewFloat("Armour", mod.TypeBase, data.MonsterArmourTable[env.EnemyLevel]).Source("Base"))

	// Add mods from the config tab
	confModList := NewModList()
	confEnemyModList := NewModList()
	for _, input := range build.Config.Inputs {
		if config, ok := configurations[input.Name]; ok {
			var value interface{}
			if input.String != nil {
				value = *input.String
			} else if input.Boolean != nil {
				value = *input.Boolean
			} else if input.Number != nil {
				value = *input.Number
			}
			config(value, confModList, confEnemyModList)
		}
	}
	env.ModDB.AddList(confModList)
	env.EnemyModDB.AddList(confEnemyModList)

	cachedPlayerDB := env.ModDB.Clone()
	cachedEnemyDB := env.EnemyModDB.Clone()
	cachedMinionDB := env.Minion.Clone()

	var tree = data.TreeVersions[data.LatestTreeVersion].Tree()
	env.AllocatedNodes = make(map[string]data.Node)
	/* *
	// TODO
	if override.addNodes or override.removeNodes then
		nodes = { }
		if override.addNodes then
			for node in pairs(override.addNodes) do
				nodes[node.id] = node
			end
		end
		for _, node in pairs(env.spec.allocNodes) do
			if not override.removeNodes or not override.removeNodes[node] then
				nodes[node.id] = node
			end
		end
	end
	/* */
	for _, id := range env.Build.Build.PassiveNodes {
		var strId = strconv.FormatInt(id, 10)
		env.AllocatedNodes[strId] = tree.Nodes[strId]
	}

	/*
		TODO -- Build and merge item modifiers, and create list of radius jewels
		for _, slot in pairs(build.itemsTab.orderedSlots) do
			local slotName = slot.slotName
			local item
			if slotName == override.repSlotName then
				item = override.repItem
			elseif override.repItem and override.repSlotName:match("^Weapon 1") and slotName:match("^Weapon 2") and
			(override.repItem.base.type == "Staff" or override.repItem.base.type == "Two Handed Sword" or override.repItem.base.type == "Two Handed Axe" or override.repItem.base.type == "Two Handed Mace"
			or (override.repItem.base.type == "Bow" and item and item.base.type ~= "Quiver")) then
				item = nil
			elseif slot.nodeId and override.spec then
				item = build.itemsTab.items[env.spec.jewels[slot.nodeId]]
			else
				item = build.itemsTab.items[slot.selItemId]
			end
			if item then
				-- Find skills granted by this item
				for _, skill in ipairs(item.grantedSkills) do
					local grantedSkill = copyTable(skill)
					grantedSkill.sourceItem = item
					grantedSkill.slotName = slotName
					t_insert(env.grantedSkillsItems, grantedSkill)
				end
			end
			if slot.weaponSet and slot.weaponSet ~= (build.itemsTab.activeItemSet.useSecondWeaponSet and 2 or 1) then
				item = nil
			end
			if slot.weaponSet == 2 and build.itemsTab.activeItemSet.useSecondWeaponSet then
				slotName = slotName:gsub(" Swap","")
			end
			if slot.nodeId then
				-- Slot is a jewel socket, check if socket is allocated
				if not env.allocNodes[slot.nodeId] then
					item = nil
				elseif item and item.jewelRadiusIndex then
					-- Jewel has a radius, add it to the list
					local funcList = item.jewelData.funcList or { { type = "Self", func = function(node, out, data)
						-- Default function just tallies all stats in radius
						if node then
							for _, stat in pairs({"Str","Dex","Int"}) do
								data[stat] = (data[stat] or 0) + out:Sum("BASE", nil, stat)
							end
						end
					end } }
					for _, func in ipairs(funcList) do
						local node = env.spec.nodes[slot.nodeId]
						t_insert(env.radiusJewelList, {
							nodes = node.nodesInRadius and node.nodesInRadius[item.jewelRadiusIndex] or { },
							func = func.func,
							type = func.type,
							item = item,
							nodeId = slot.nodeId,
							attributes = node.attributesInRadius and node.attributesInRadius[item.jewelRadiusIndex] or { },
							data = { }
						})
						if func.type ~= "Self" and node.nodesInRadius then
							-- Add nearby unallocated nodes to the extra node list
							for nodeId, node in pairs(node.nodesInRadius[item.jewelRadiusIndex]) do
								if not env.allocNodes[nodeId] then
									env.extraRadiusNodeList[nodeId] = env.spec.nodes[nodeId]
								end
							end
						end
					end
				end
			end
			if item and item.type == "Flask" then
				if slot.active then
					env.flasks[item] = true
				end
				if item.base.subType == "Life" then
					local highestLifeRecovery = env.itemModDB.multipliers["LifeFlaskRecovery"] or 0
					if item.flaskData.lifeTotal > highestLifeRecovery then
						env.itemModDB.multipliers["LifeFlaskRecovery"] = item.flaskData.lifeTotal
					end
				end
				item = nil
			end
			local scale = 1
			if item and item.type == "Jewel" and item.base.subType == "Abyss" and slot.parentSlot then
				-- Check if the item in the parent slot has enough Abyssal Sockets
				local parentItem = env.player.itemList[slot.parentSlot.slotName]
				if not parentItem or parentItem.abyssalSocketCount < slot.slotNum then
					item = nil
				else
					scale = parentItem.socketedJewelEffectModifier
				end
			end
			if slot.nodeId and item and item.type == "Jewel" and item.jewelData and item.jewelData.jewelIncEffectFromClassStart then
				local node = env.spec.nodes[slot.nodeId]
				if node and node.distanceToClassStart then
					scale = scale + node.distanceToClassStart * (item.jewelData.jewelIncEffectFromClassStart / 100)
				end
			end
			if item then
				env.player.itemList[slotName] = item
				-- Merge mods for this item
				local srcList = item.modList or item.slotModList[slot.slotNum]
				if item.requirements and not accelerate.requirementsItems then
					t_insert(env.requirementsTableItems, {
						source = "Item",
						sourceItem = item,
						sourceSlot = slotName,
						Str = item.requirements.strMod,
						Dex = item.requirements.dexMod,
						Int = item.requirements.intMod,
					})
				end
				if item.type == "Jewel" and item.base.subType == "Abyss" then
					-- Update Abyss Jewel conditions/multipliers
					local cond = "Have"..item.baseName:gsub(" ","")
					if not env.itemModDB.conditions[cond] then
						env.itemModDB.conditions[cond] = true
						env.itemModDB.multipliers["AbyssJewelType"] = (env.itemModDB.multipliers["AbyssJewelType"] or 0) + 1
					end
					if slot.parentSlot then
						env.itemModDB.conditions[cond.."In"..slot.parentSlot.slotName] = true
					end
					env.itemModDB.multipliers["AbyssJewel"] = (env.itemModDB.multipliers["AbyssJewel"] or 0) + 1
					env.itemModDB.multipliers[item.baseName:gsub(" ","")] = (env.itemModDB.multipliers[item.baseName:gsub(" ","")] or 0) + 1
				end
				if item.type == "Shield" and env.allocNodes[45175] and env.allocNodes[45175].dn == "Necromantic Aegis" then
					-- Special handling for Necromantic Aegis
					env.aegisModList = new("ModList")
					for _, mod in ipairs(srcList) do
						-- Filter out mods that apply to socketed gems, or which add supports
						local add = true
						for _, tag in ipairs(mod) do
							if tag.type == "SocketedIn" then
								add = false
								break
							end
						end
						if add then
							env.aegisModList:ScaleAddMod(mod, scale)
						else
							env.itemModDB:ScaleAddMod(mod, scale)
						end
					end
				elseif (slotName == "Weapon 1" or slotName == "Weapon 2") and modDB:Flag(nil, "Condition:EnergyBladeActive") then
					local type = env.player.itemList[slotName] and env.player.itemList[slotName].weaponData and env.player.itemList[slotName].weaponData[1].type
					local info = env.data.weaponTypeInfo[type]
					if info and type ~= "Bow" then
						local name = info.oneHand and "Energy Blade One Handed" or "Energy Blade Two Handed"
						local item = new("Item")
						item.name = name
						item.base = data.itemBases[name]
						item.baseName = name
						item.classRequirementModLines = { }
						item.buffModLines = { }
						item.enchantModLines = { }
						item.scourgeModLines = { }
						item.implicitModLines = { }
						item.explicitModLines = { }
						item.quality = 0
						item.rarity = "NORMAL"
						if item.baseName.implicit then
							local implicitIndex = 1
							for line in item.baseName.implicit:gmatch("[^\n]+") do
								local modList, extra = modLib.parseMod(line)
								t_insert(item.implicitModLines, { line = line, extra = extra, modList = modList or { }, modTags = item.baseName.implicitModTypes and item.baseName.implicitModTypes[implicitIndex] or { } })
								implicitIndex = implicitIndex + 1
							end
						end
						item:NormaliseQuality()
						item:BuildAndParseRaw()
						env.player.itemList[slotName] = item
					else
						env.itemModDB:ScaleAddList(srcList, scale)
					end
				elseif slotName == "Weapon 1" and item.name == "The Iron Mass, Gladius" then
					-- Special handling for The Iron Mass
					env.theIronMass = new("ModList")
					for _, mod in ipairs(srcList) do
						-- Filter out mods that apply to socketed gems, or which add supports
						local add = true
						for _, tag in ipairs(mod) do
							if tag.type == "SocketedIn" then
								add = false
								break
							end
						end
						if add then
							env.theIronMass:ScaleAddMod(mod, scale)
						end
						-- Add all the stats to player as well
						env.itemModDB:ScaleAddMod(mod, scale)
					end
				elseif slotName == "Weapon 1" and item.grantedSkills[1] and item.grantedSkills[1].skillId == "UniqueAnimateWeapon" then
					-- Special handling for The Dancing Dervish
					env.weaponModList1 = new("ModList")
					for _, mod in ipairs(srcList) do
						-- Filter out mods that apply to socketed gems, or which add supports
						local add = true
						for _, tag in ipairs(mod) do
							if tag.type == "SocketedIn" then
								add = false
								break
							end
						end
						if add then
							env.weaponModList1:ScaleAddMod(mod, scale)
						else
							env.itemModDB:ScaleAddMod(mod, scale)
						end
					end
				else
					env.itemModDB:ScaleAddList(srcList, scale)
				end
				-- set conditions on restricted items
				if item.classRestriction then
					env.itemModDB.conditions[item.title:gsub(" ", "")] = item.classRestriction
				end
				if item.type ~= "Jewel" and item.type ~= "Flask" then
					-- Update item counts
					local key
					if item.rarity == "UNIQUE" or item.rarity == "RELIC" then
						key = "UniqueItem"
					elseif item.rarity == "RARE" then
						key = "RareItem"
					elseif item.rarity == "MAGIC" then
						key = "MagicItem"
					else
						key = "NormalItem"
					end
					env.itemModDB.multipliers[key] = (env.itemModDB.multipliers[key] or 0) + 1
					env.itemModDB.conditions[key .. "In" .. slotName] = true
					if item.corrupted then
						env.itemModDB.multipliers.CorruptedItem = (env.itemModDB.multipliers.CorruptedItem or 0) + 1
					else
						env.itemModDB.multipliers.NonCorruptedItem = (env.itemModDB.multipliers.NonCorruptedItem or 0) + 1
					end
					if item.shaper then
						env.itemModDB.multipliers.ShaperItem = (env.itemModDB.multipliers.ShaperItem or 0) + 1
						env.itemModDB.conditions["ShaperItemIn"..slotName] = true
					else
						env.itemModDB.multipliers.NonShaperItem = (env.itemModDB.multipliers.NonShaperItem or 0) + 1
					end
					if item.elder then
						env.itemModDB.multipliers.ElderItem = (env.itemModDB.multipliers.ElderItem or 0) + 1
						env.itemModDB.conditions["ElderItemIn"..slotName] = true
					else
						env.itemModDB.multipliers.NonElderItem = (env.itemModDB.multipliers.NonElderItem or 0) + 1
					end
					if item.shaper or item.elder then
						env.itemModDB.multipliers.ShaperOrElderItem = (env.itemModDB.multipliers.ShaperOrElderItem or 0) + 1
					end
				end
			end
		end
	*/

	/*
		TODO -- Merge env.itemModDB with env.ModDB
		mergeDB(env.modDB, env.itemModDB)
	*/

	/*
		TODO Flask Override
		if override.toggleFlask then
			if env.flasks[override.toggleFlask] then
				env.flasks[override.toggleFlask] = nil
			else
				env.flasks[override.toggleFlask] = true
			end
		end
	*/

	/*
		TODO -- Add granted passives (e.g., amulet anoints)
		for _, passive in pairs(env.modDB:List(nil, "GrantedPassive")) do
			local node = env.spec.tree.notableMap[passive]
			if node and (not override.removeNodes or not override.removeNodes[node.id]) then
				env.allocNodes[node.id] = node
				env.grantedPassives[node.id] = true
			end
		end
	*/

	/*
		TODO -- Add granted ascendancy node (e.g., Forbidden Flame/Flesh combo)
		local matchedName = { }
		for _, ascTbl in pairs(env.modDB:List(nil, "GrantedAscendancyNode")) do
			local name = ascTbl.name
			if matchedName[name] and matchedName[name].side ~= ascTbl.side and matchedName[name].matched == false then
				matchedName[name].matched = true
				local node = env.spec.tree.ascendancyMap[name]
				if node and (not override.removeNodes or not override.removeNodes[node.id]) then
					if env.itemModDB.conditions["ForbiddenFlesh"] == env.spec.curClassName and env.itemModDB.conditions["ForbiddenFlame"] == env.spec.curClassName then
						env.allocNodes[node.id] = node
						env.grantedPassives[node.id] = true
					end
				end
			else
				matchedName[name] = { side = ascTbl.side, matched = false }
			end
		end
	*/

	env.ModDB.AddList(buildModListForNodeList(env, env.AllocatedNodes))

	/*
		TODO -- Find skills granted by tree nodes
		for _, node in pairs(env.allocNodes) do
			for _, skill in ipairs(node.grantedSkills) do
				local grantedSkill = copyTable(skill)
				grantedSkill.sourceNode = node
				t_insert(env.grantedSkillsNodes, grantedSkill)
			end
		end
	*/

	/*
		TODO -- Merge Granted Skills Tables
		env.grantedSkills = tableConcat(env.grantedSkillsNodes, env.grantedSkillsItems)
	*/

	// Skills
	if env.Mode == OutputModeMain {
		/*
			TODO -- Process extra skills granted by items or tree nodes
			local markList = wipeTable(tempTable1)
			for _, grantedSkill in ipairs(env.grantedSkills) do
				TODO -- Check if a matching group already exists
				local group
				for index, socketGroup in pairs(build.skillsTab.socketGroupList) do
					if socketGroup.source == grantedSkill.source and socketGroup.slot == grantedSkill.slotName then
						if socketGroup.gemList[1] and socketGroup.gemList[1].skillId == grantedSkill.skillId and socketGroup.gemList[1].level == grantedSkill.level then
							group = socketGroup
							markList[socketGroup] = true
							break
						end
					end
				end
				if not group then
					TODO -- Create a new group for this skill
					group = { label = "", enabled = true, gemList = { }, source = grantedSkill.source, slot = grantedSkill.slotName }
					t_insert(build.skillsTab.socketGroupList, group)
					markList[group] = true
				end

				TODO -- Update the group
				group.sourceItem = grantedSkill.sourceItem
				group.sourcacceleeNode = grantedSkill.sourceNode
				local activeGemInstance = group.gemList[1] or {
					skillId = grantedSkill.skillId,
					quality = 0,
					enabled = true,
				}
				activeGemInstance.gemId = nil
				activeGemInstance.level = grantedSkill.level
				activeGemInstance.enableGlobal1 = true
				if grantedSkill.triggered then
					activeGemInstance.triggered = grantedSkill.triggered
				end
				wipeTable(group.gemList)
				t_insert(group.gemList, activeGemInstance)
				if grantedSkill.noSupports then
					group.noSupports = true
				else
					for _, socketGroup in pairs(build.skillsTab.socketGroupList) do
						TODO -- Look for other groups that are socketed in the item
						if socketGroup.slot == grantedSkill.slotName and not socketGroup.source then
							TODO -- Add all support gems to the skill's group
							for _, gemInstance in ipairs(socketGroup.gemList) do
								if gemInstance.gemData and gemInstance.gemData.grantedEffect.support then
									t_insert(group.gemList, gemInstance)
								end
							end
						end
					end
				end
				build.skillsTab:ProcessSocketGroup(group)
			end

			TODO -- Remove any socket groups that no longer have a matching item
			local i = 1
			while build.skillsTab.socketGroupList[i] do
				local socketGroup = build.skillsTab.socketGroupList[i]
				if socketGroup.source and not markList[socketGroup] then
					t_remove(build.skillsTab.socketGroupList, i)
					if build.skillsTab.displayGroup == socketGroup then
						build.skillsTab.displayGroup = nil
					end
				else
					i = i + 1
				end
			end
		*/
	}

	env.Player.WeaponData1 = utils.CopyMap(data.UnarmedWeaponData[data.ClassIDs[env.Spec.ClassName]])
	//if _, ok := env.Player.ItemList["Weapon 1"]; ok {
	// TODO Weapon 1 Data
	// env.player.itemList["Weapon 1"].weaponData and env.player.itemList["Weapon 1"].weaponData[1]
	//}

	if utils.HasTrue(env.Player.WeaponData1, "countsAsDualWielding") {
		// TODO
		// env.player.weaponData2 = env.player.itemList["Weapon 1"].weaponData[2]
	} else {
		// TODO
		// env.player.weaponData2 = env.player.itemList["Weapon 2"] and env.player.itemList["Weapon 2"].weaponData and env.player.itemList["Weapon 2"].weaponData[2] or { }
		env.Player.WeaponData2 = make(map[string]interface{})
	}

	/*
		TODO -- Get the weapon data tables for the equipped weapons
		if env.player.weaponData1.countsAsDualWielding then
			env.player.weaponData2 = env.player.itemList["Weapon 1"].weaponData[2]
		else
			env.player.weaponData2 = env.player.itemList["Weapon 2"] and env.player.itemList["Weapon 2"].weaponData and env.player.itemList["Weapon 2"].weaponData[2] or { }
		end
	*/

	if env.Mode == OutputModeCalcs {

	} else {
		selectedSkillSet := build.Skills.ActiveSkillSet - 1
		if selectedSkillSet < 0 {
			selectedSkillSet = 0
		}

		skillCount := 0
		if len(build.Skills.SkillSets) > selectedSkillSet {
			skillCount = len(build.Skills.SkillSets[selectedSkillSet].Skills)
		}
		build.Build.MainSocketGroup = min(max(skillCount, 1), build.Build.MainSocketGroup) - 1
		env.MainSocketGroup = build.Build.MainSocketGroup
	}

	/*
		TODO -- Determine main skill group
		if env.mode == "CALCS" then
			env.calcsInput.skill_number = m_min(m_max(#build.skillsTab.socketGroupList, 1), env.calcsInput.skill_number or 1)
			env.mainSocketGroup = env.calcsInput.skill_number
		else
			build.mainSocketGroup = m_min(m_max(#build.skillsTab.socketGroupList, 1), build.mainSocketGroup or 1)
			env.mainSocketGroup = build.mainSocketGroup
		end
	*/

	// Build list of active skills
	groupCfg := &ListCfg{}

	// Below we re-order the socket group list in order to support modifiers introduced in 3.16
	// which allow a Shield (Weapon 2) to link to a Main Hand and an Amulet to link to a Body Armour
	// as we need their support gems and effects to be processed before we cross-link them to those slots
	selectedSkillSet := build.Skills.ActiveSkillSet - 1
	if selectedSkillSet < 0 {
		selectedSkillSet = 0
	}

	var indexOrder []int
	if selectedSkillSet < len(build.Skills.SkillSets) {
		indexOrder = make([]int, len(build.Skills.SkillSets[selectedSkillSet].Skills))
		for i, socketGroup := range build.Skills.SkillSets[selectedSkillSet].Skills {
			if socketGroup.Slot == "Amulet" || socketGroup.Slot == "Weapon 2" {
				indexOrder = append([]int{i}, indexOrder...)
			} else {
				indexOrder = append(indexOrder, i)
			}
		}
	}

	crossLinkedSupportList := make(map[string]interface{})
	for _, index := range indexOrder {
		socketGroup := build.Skills.SkillSets[selectedSkillSet].Skills[index]
		socketGroupSkillList := make([]*ActiveSkill, 0)
		var slot interface{}
		if socketGroup.Slot != "" {
			// TODO
			// slot = build.itemsTab.slots[socketGroup.slot]
		}

		socketGroup.SlotEnabled = slot == nil
		// TODO
		// socketGroup.slotEnabled = not slot or not slot.weaponSet or slot.weaponSet == (build.itemsTab.activeItemSet.useSecondWeaponSet and 2 or 1)
		if index == env.MainSocketGroup || (socketGroup.Enabled && socketGroup.SlotEnabled) {
			if socketGroup.Slot != "" {
				groupCfg.SlotName = strings.Replace(socketGroup.Slot, " Swap", "", -1)
			}

			propertyModList := utils.CastSlice[mod.GemProperty](env.ModDB.List(groupCfg, "GemProperty"))

			// Build list of supports for this socket group
			supportList := make([]*GemEffect, 0)
			if socketGroup.Source == nil {
				// Add extra supports from the item this group is socketed in
				for _, value := range env.ModDB.List(groupCfg, "ExtraSupport") {
					_ = value
					/*
						TODO
						local grantedEffect = env.data.skills[value.skillId]
						-- Some skill gems share the same name as support gems, e.g. Barrage.
						-- Since a support gem is expected here, if the first lookup returns a skill, then
						-- prepending "Support" to the skillId will find the support version of the gem.
						if grantedEffect and not grantedEffect.support then
							grantedEffect = env.data.skills["Support"..value.skillId]
						end
						if grantedEffect then
							t_insert(supportList, {
								grantedEffect = grantedEffect,
								level = value.level,
								quality = 0,
								enabled = true,
							})
						end
					*/
				}
			}

			if _, ok := crossLinkedSupportList[socketGroup.Slot]; ok {
				_ = ok
				/*
					TODO
					for _, supportItem in ipairs(crossLinkedSupportList[socketGroup.slot]) do
						t_insert(supportList, supportItem)
					end
				*/
			}

			for _, gemInstance := range socketGroup.Gems {
				// Add support gems from this group
				if env.Mode == OutputModeMain {
					// TODO
					//gemInstance.DisplayEffect = nil
					//gemInstance.SupportEffect = nil
				}

				if gemInstance.Enabled {
					baseItemType := poe.BaseItemTypeByIDMap[gemInstance.GemID]
					if baseItemType == nil {
						continue
					}

					gemData := baseItemType.SkillGem()

					processGrantedEffect := func(grantedEffect *poe.GrantedEffect) {
						if grantedEffect == nil || !grantedEffect.IsSupport {
							return
						}

						var temp = gemInstance
						supportEffect := &GemEffect{
							GrantedEffect: &GrantedEffect{
								Raw: grantedEffect,
							},
							Level:        gemInstance.Level,
							Quality:      gemInstance.Quality,
							QualityID:    gemInstance.QualityID,
							SrcInstance:  &temp,
							GemData:      gemData,
							Superseded:   false,
							IsSupporting: make(map[*pob.Gem]bool),
							Values:       make(map[string]float64),
						}

						if env.Mode == OutputModeMain {
							// TODO
							//gemInstance.DisplayEffect = supportEffect
							//gemInstance.SupportEffect = supportEffect
						}

						if gemData != nil {
							for _, value := range propertyModList {
								match := true

								if len(value.KeywordList) > 0 {
									for _, keyword := range value.KeywordList {
										if !CalcGemIsType(supportEffect.GemData, keyword) {
											match = false
											break
										}
									}
								} else if !CalcGemIsType(supportEffect.GemData, *value.Keyword) {
									match = false
								}

								if match {
									supportEffect.Values[value.Key] = supportEffect.Values[value.Key] + value.Value
								}
							}
						}

						// Validate support gem level in case there is no active skill (and no full calculation)
						CalcValidateGemLevel(supportEffect)

						add := true
						for index, otherSupport := range supportList {
							// Check if there's another support with the same name already present
							if grantedEffect == otherSupport.GrantedEffect.Raw {
								add = false
								if supportEffect.Level > otherSupport.Level || (supportEffect.Level == otherSupport.Level && supportEffect.Quality > otherSupport.Quality) {
									if env.Mode == OutputModeMain {
										otherSupport.Superseded = true
									}
									supportList[index] = supportEffect
								} else {
									supportEffect.Superseded = true
								}
								break
							} else if grantedEffect.PlusVersionOf != nil && otherSupport.GrantedEffect.Raw.PlusVersionOf != nil && *grantedEffect.PlusVersionOf == *otherSupport.GrantedEffect.Raw.PlusVersionOf {
								add = false
								if env.Mode == OutputModeMain {
									otherSupport.Superseded = true
								}
								supportList[index] = supportEffect
							} else if otherSupport.GrantedEffect.Raw.PlusVersionOf != nil && *otherSupport.GrantedEffect.Raw.PlusVersionOf == grantedEffect.Key {
								add = false
								supportEffect.Superseded = true
							}
						}

						if add {
							supportList = append(supportList, supportEffect)
						}
					}

					_ = processGrantedEffect // TODO Remove

					if gemData != nil {
						processGrantedEffect(gemData.GetGrantedEffect())
						processGrantedEffect(gemData.GetSecondaryGrantedEffect())
					} else {
						// TODO processGrantedEffect(gemInstance.grantedEffect)
					}

					// Store extra supports for other items that are linked
					for _, value := range env.ModDB.List(groupCfg, "LinkedSupport") {
						_ = value
						/*
							// TODO LinkedSupport
							crossLinkedSupportList[value.targetSlotName] = { }
							for _, supportItem in ipairs(supportList) do
								t_insert(crossLinkedSupportList[value.targetSlotName], supportItem)
							end
						*/
					}
				}
			}

			// Create active skills
			for _, gemInstance := range socketGroup.Gems {
				baseItem := poe.BaseItemTypeByIDMap[gemInstance.GemID]
				if baseItem == nil {
					continue
				}
				gemData := baseItem.SkillGem()
				grantedEffectList := gemData.GetGrantedEffects()

				if gemInstance.Enabled && grantedEffectList != nil && len(grantedEffectList) > 0 {
					for index, grantedEffect := range grantedEffectList {
						globalEnable := gemInstance.EnableGlobal1
						if index == 2 {
							globalEnable = gemInstance.EnableGlobal2
						}

						if !grantedEffect.IsSupport && (!grantedEffect.HasGlobalEffect() || globalEnable) {
							baseFlags, skillTypes := TypesToFlagsAndTypes(grantedEffect.GetActiveSkill().GetActiveSkillTypes())

							temp := gemInstance
							activeEffect := &GemEffect{
								GrantedEffect: &GrantedEffect{
									Raw:        grantedEffect,
									Parts:      nil, // TODO Parts
									SkillTypes: skillTypes,
									BaseFlags:  baseFlags,
								},
								Level:       gemInstance.Level,
								Quality:     gemInstance.Quality,
								QualityID:   gemInstance.QualityID,
								SrcInstance: &temp,
								GemData:     gemData,
							}

							if gemData != nil {
								for _, value := range propertyModList {
									match := false
									if value.KeywordList != nil {
										match = true
										for _, keyword := range value.KeywordList {
											if !CalcGemIsType(activeEffect.GemData, keyword) {
												match = false
												break
											}
										}
									} else {
										match = CalcGemIsType(activeEffect.GemData, *value.Keyword)
									}

									if match {
										_ = match
										// TODO
										// activeEffect[value.key] = (activeEffect[value.key] or 0) + value.value
									}
								}
							}

							if env.Mode == OutputModeMain {
								// TODO
								//gemInstance.DisplayEffect = activeEffect
							}

							activeSkill := CreateActiveSkill(activeEffect, supportList, env.Player, &socketGroup, nil)
							if gemData != nil {
								activeSkill.SlotName = groupCfg.SlotName
							}

							socketGroupSkillList = append(socketGroupSkillList, activeSkill)
							env.Player.ActiveSkillList = append(env.Player.ActiveSkillList, activeSkill)
						}
					}

					if gemData != nil {
						env.RequirementsTableGems = append(env.RequirementsTableGems, &RequirementsTableGems{
							Source:    "Gem",
							SourceGem: gemInstance,
							Str:       gemData.Str,
							Dex:       gemData.Dex,
							Int:       gemData.Int,
						})
					}
				}
			}

			if index == env.MainSocketGroup && len(socketGroupSkillList) > 0 {
				// Select the main skill from this socket group
				activeSkillIndex := 0
				if env.Mode == OutputModeCalcs {
					socketGroup.MainActiveSkillCalcs = min(len(socketGroupSkillList)-1, socketGroup.MainActiveSkillCalcs)
					activeSkillIndex = socketGroup.MainActiveSkillCalcs
				} else {
					activeSkillIndex = min(len(socketGroupSkillList)-1, socketGroup.MainActiveSkill)
					if env.Mode == OutputModeMain {
						socketGroup.MainActiveSkill = activeSkillIndex
					}
				}
				env.Player.MainSkill = socketGroupSkillList[activeSkillIndex]
			}
		}

		if env.Mode == OutputModeMain {
			// Create display label for the socket group if the user didn't specify one
			if socketGroup.Label != "" {
				socketGroup.DisplayLabel = socketGroup.Label
			} else {
				DisplayLabel := ""
				for _, gemInstance := range socketGroup.Gems {
					baseItem := poe.BaseItemTypeByIDMap[gemInstance.GemID]
					if baseItem == nil {
						continue
					}
					gemData := baseItem.SkillGem()
					grantedEffect := gemData.GetGrantedEffect()
					if grantedEffect != nil && !grantedEffect.IsSupport && gemInstance.Enabled {
						if DisplayLabel != "" {
							DisplayLabel += ", "
						}
						DisplayLabel += grantedEffect.GetActiveSkill().DisplayedName
					}
				}
				if DisplayLabel == "" {
					DisplayLabel = "<No active skills>"
				}
				socketGroup.DisplayLabel = DisplayLabel
			}

			// Save the active skill list for display in the socket group tooltip
			socketGroup.DisplaySkillList = socketGroupSkillList
		} else if env.Mode == OutputModeCalcs {
			socketGroup.DisplaySkillListCalcs = socketGroupSkillList
		}
	}

	if env.Player.MainSkill == nil {
		// Add a default main skill if none are specified
		playerMelee := poe.GrantedEffectByID("PlayerMelee")
		baseFlags, skillTypes := TypesToFlagsAndTypes(playerMelee.GetActiveSkill().GetActiveSkillTypes())
		defaultEffect := &GemEffect{
			GrantedEffect: &GrantedEffect{
				Raw:        playerMelee,
				Parts:      nil, // TODO Parts
				SkillTypes: skillTypes,
				BaseFlags:  baseFlags,
			},
		}
		env.Player.MainSkill = CreateActiveSkill(defaultEffect, []*GemEffect{}, env.Player, nil, nil)
		env.Player.ActiveSkillList = append(env.Player.ActiveSkillList, env.Player.MainSkill)
	}

	// Build skill modifier lists
	for _, activeSkill := range env.Player.ActiveSkillList {
		CalcBuildActiveSkillModList(env, activeSkill)
	}

	/*
		TODO -- Merge Requirements Tables
		env.requirementsTable = tableConcat(env.requirementsTableItems, env.requirementsTableGems)
	*/

	return env, cachedPlayerDB, cachedEnemyDB, cachedMinionDB
}

func initModDB(env *Environment, modDB *ModDB) {
	modDB.AddMod(mod.NewFloat("FireResistMax", mod.TypeBase, 75).Source("Base"))
	modDB.AddMod(mod.NewFloat("ColdResistMax", mod.TypeBase, 75).Source("Base"))
	modDB.AddMod(mod.NewFloat("LightningResistMax", mod.TypeBase, 75).Source("Base"))
	modDB.AddMod(mod.NewFloat("ChaosResistMax", mod.TypeBase, 75).Source("Base"))
	modDB.AddMod(mod.NewFloat("TotemFireResistMax", mod.TypeBase, 75).Source("Base"))
	modDB.AddMod(mod.NewFloat("TotemColdResistMax", mod.TypeBase, 75).Source("Base"))
	modDB.AddMod(mod.NewFloat("TotemLightningResistMax", mod.TypeBase, 75).Source("Base"))
	modDB.AddMod(mod.NewFloat("TotemChaosResistMax", mod.TypeBase, 75).Source("Base"))
	modDB.AddMod(mod.NewFloat("BlockChanceMax", mod.TypeBase, 75).Source("Base"))
	modDB.AddMod(mod.NewFloat("SpellBlockChanceMax", mod.TypeBase, 75).Source("Base"))
	modDB.AddMod(mod.NewFloat("SpellDodgeChanceMax", mod.TypeBase, 75).Source("Base"))
	modDB.AddMod(mod.NewFloat("PowerChargesMax", mod.TypeBase, 3).Source("Base"))
	modDB.AddMod(mod.NewFloat("FrenzyChargesMax", mod.TypeBase, 3).Source("Base"))
	modDB.AddMod(mod.NewFloat("EnduranceChargesMax", mod.TypeBase, 3).Source("Base"))
	modDB.AddMod(mod.NewFloat("SiphoningChargesMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("ChallengerChargesMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("BlitzChargesMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("InspirationChargesMax", mod.TypeBase, 5).Source("Base"))
	modDB.AddMod(mod.NewFloat("CrabBarriersMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("BrutalChargesMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("AbsorptionChargesMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("AfflictionChargesMax", mod.TypeBase, 0).Source("Base"))
	modDB.AddMod(mod.NewFloat("BloodChargesMax", mod.TypeBase, 5).Source("Base"))
	modDB.AddMod(mod.NewFloat("MaxLifeLeechRate", mod.TypeBase, 20).Source("Base"))
	modDB.AddMod(mod.NewFloat("MaxManaLeechRate", mod.TypeBase, 20).Source("Base"))
	modDB.AddMod(mod.NewFloat("ImpaleStacksMax", mod.TypeBase, 5).Source("Base"))
	modDB.AddMod(mod.NewFloat("Multiplier:VirulenceStacksMax", mod.TypeBase, 40).Source("Base"))
	modDB.AddMod(mod.NewFloat("BleedStacksMax", mod.TypeBase, 1).Source("Base"))
	modDB.AddMod(mod.NewFloat("MaxEnergyShieldLeechRate", mod.TypeBase, 10).Source("Base"))
	modDB.AddMod(mod.NewFloat("MaxLifeLeechInstance", mod.TypeBase, 10).Source("Base"))
	modDB.AddMod(mod.NewFloat("MaxManaLeechInstance", mod.TypeBase, 10).Source("Base"))
	modDB.AddMod(mod.NewFloat("MaxEnergyShieldLeechInstance", mod.TypeBase, 10).Source("Base"))
	modDB.AddMod(mod.NewFloat("TrapThrowingTime", mod.TypeBase, 0.6).Source("Base"))
	modDB.AddMod(mod.NewFloat("MineLayingTime", mod.TypeBase, 0.3).Source("Base"))
	modDB.AddMod(mod.NewFloat("WarcryCastTime", mod.TypeBase, 0.8).Source("Base"))
	modDB.AddMod(mod.NewFloat("TotemPlacementTime", mod.TypeBase, 0.6).Source("Base"))
	modDB.AddMod(mod.NewFloat("BallistaPlacementTime", mod.TypeBase, 0.35).Source("Base"))
	modDB.AddMod(mod.NewFloat("ActiveTotemLimit", mod.TypeBase, 1).Source("Base"))

	modDB.AddMod(mod.NewFloat("MovementSpeed", mod.TypeIncrease, -30).Source("Base").Tag(mod.Condition("Maimed")))
	modDB.AddMod(mod.NewFloat("DamageTaken", mod.TypeIncrease, 10).Source("Base").Flag(mod.MFlagAttack).Tag(mod.Condition("Intimidated")))

	modDB.AddMod(mod.NewFlag("Condition:Burning", true).Source("Base").Tag(mod.IgnoreCond()).Tag(mod.Condition("Ignited")))
	modDB.AddMod(mod.NewFlag("Condition:Chilled", true).Source("Base").Tag(mod.IgnoreCond()).Tag(mod.Condition("Frozen")))
	modDB.AddMod(mod.NewFlag("Condition:Poisoned", true).Source("Base").Tag(mod.IgnoreCond()).Tag(mod.MultiplierThreshold("PoisonStack").Threshold(1)))

	modDB.AddMod(mod.NewFlag("Blind", true).Source("Base").Tag(mod.Condition("Blinded")))
	modDB.AddMod(mod.NewFlag("Chill", true).Source("Base").Tag(mod.Condition("Chilled")))
	modDB.AddMod(mod.NewFlag("Freeze", true).Source("Base").Tag(mod.Condition("Frozen")))
	modDB.AddMod(mod.NewFlag("Fortify", true).Source("Base").Tag(mod.Condition("Fortify")))
	modDB.AddMod(mod.NewFlag("Fortified", true).Source("Base").Tag(mod.Condition("Fortified")))
	modDB.AddMod(mod.NewFlag("Fanaticism", true).Source("Base").Tag(mod.Condition("Fanaticism")))
	modDB.AddMod(mod.NewFlag("Onslaught", true).Source("Base").Tag(mod.Condition("Onslaught")))
	modDB.AddMod(mod.NewFlag("UnholyMight", true).Source("Base").Tag(mod.Condition("UnholyMight")))
	modDB.AddMod(mod.NewFlag("Tailwind", true).Source("Base").Tag(mod.Condition("Tailwind")))
	modDB.AddMod(mod.NewFlag("Adrenaline", true).Source("Base").Tag(mod.Condition("Adrenaline")))
	modDB.AddMod(mod.NewFlag("AlchemistsGenius", true).Source("Base").Tag(mod.Condition("AlchemistsGenius")))
	modDB.AddMod(mod.NewFlag("LuckyHits", true).Source("Base").Tag(mod.Condition("LuckyHits")))
	modDB.AddMod(mod.NewFlag("Convergence", true).Source("Base").Tag(mod.Condition("Convergence")))

	modDB.AddMod(mod.NewFloat("PhysicalDamageReduction", mod.TypeBase, -15).Source(mod.SourceBase).Tag(mod.Condition("Crushed")))

	modDB.Conditions["Buffed"] = env.ModeBuffs
	modDB.Conditions["Combat"] = env.ModeCombat
	modDB.Conditions["Effective"] = env.ModeEffective
}
