
local buildsDir, outDir = arg[1], arg[2]
assert(buildsDir and outDir, "usage: dump_reservation.lua <buildsDir> <outDir>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local lfs = require("lfs")

lfs.mkdir(outDir)

local builds = {}
for file in lfs.dir(buildsDir) do
	if file:match("%.xml$") then
		builds[#builds + 1] = file
	end
end
table.sort(builds)

for _, file in ipairs(builds) do
	local fh = assert(io.open(buildsDir .. "/" .. file, "r"))
	local xmlText = fh:read("*a")
	fh:close()

	local entry = { build = file }
	local ok, err = pcall(function()
		loadBuildFromXML(xmlText, file)
		runCallback("OnFrame")

		local env = build.calcsTab.mainEnv
		local output = env.player.output

		entry.output = {
			Mana = output.Mana,
			ManaReserved = output.ManaReserved,
			ManaReservedPercent = output.ManaReservedPercent,
			ManaUnreserved = output.ManaUnreserved,
			Life = output.Life,
			LifeReserved = output.LifeReserved,
			LifeReservedPercent = output.LifeReservedPercent,
			LifeUnreserved = output.LifeUnreserved,
			reserved_ManaBase = env.player.reserved_ManaBase,
			reserved_ManaPercent = env.player.reserved_ManaPercent,
			reserved_LifeBase = env.player.reserved_LifeBase,
			reserved_LifePercent = env.player.reserved_LifePercent,
		}

		local skills = {}
		for _, activeSkill in ipairs(env.player.activeSkillList) do
			local ge = activeSkill.activeEffect and activeSkill.activeEffect.grantedEffect
			local gel = activeSkill.activeEffect and activeSkill.activeEffect.grantedEffectLevel
			local s = {
				name = ge and ge.name,
				id = ge and ge.id,
				hasReservation = activeSkill.skillTypes[SkillType.HasReservation] or false,
				reservationBecomesCost = activeSkill.skillTypes[SkillType.ReservationBecomesCost] or false,
				slotName = activeSkill.slotName,
				socketGroupEnabled = activeSkill.socketGroup and activeSkill.socketGroup.enabled,
				activeMineCount = activeSkill.activeMineCount,
				dataManaFlat = activeSkill.skillData.manaReservationFlat,
				dataManaPercent = activeSkill.skillData.manaReservationPercent,
				levelManaFlat = gel and gel.manaReservationFlat,
				levelManaPercent = gel and gel.manaReservationPercent,
				dataLifeFlat = activeSkill.skillData.lifeReservationFlat,
				dataLifePercent = activeSkill.skillData.lifeReservationPercent,
				levelLifeFlat = gel and gel.lifeReservationFlat,
				levelLifePercent = gel and gel.lifeReservationPercent,
				ManaReservedBase = activeSkill.skillData.ManaReservedBase,
				ManaReservedPercent = activeSkill.skillData.ManaReservedPercent,
				LifeReservedBase = activeSkill.skillData.LifeReservedBase,
				LifeReservedPercent = activeSkill.skillData.LifeReservedPercent,
				ManaReservationFlatForced = activeSkill.skillData.ManaReservationFlatForced,
				ManaReservationPercentForced = activeSkill.skillData.ManaReservationPercentForced,
				LifeReservationFlatForced = activeSkill.skillData.LifeReservationFlatForced,
				LifeReservationPercentForced = activeSkill.skillData.LifeReservationPercentForced,
			}
			if activeSkill.skillTypes[SkillType.HasReservation] then
				local sml, cfg = activeSkill.skillModList, activeSkill.skillCfg
				s.supportManaMultiplier = sml:More(cfg, "SupportManaMultiplier")
				s.bloodMagicReserved = sml:Flag(cfg, "BloodMagicReserved")
				for _, pool in ipairs({ "Mana", "Life" }) do
					s[pool .. "_more"] = sml:More(cfg, pool .. "Reserved", "Reserved")
					s[pool .. "_inc"] = sml:Sum("INC", cfg, pool .. "Reserved", "Reserved")
					s[pool .. "_efficiency"] = sml:Sum("INC", cfg, pool .. "ReservationEfficiency", "ReservationEfficiency")
				end
			end
			skills[#skills + 1] = s
		end
		entry.skills = skills
	end)
	if not ok then
		entry.error = tostring(err)
	end

	local outPath = outDir .. "/" .. file:gsub("%.xml$", ".json")
	local out = assert(io.open(outPath, "w"))
	out:write(json.encode(entry, { indent = true }))
	out:close()

	io.stderr:write(string.format("reservation: %s%s\n", file, entry.error and ("  ERROR " .. entry.error) or ""))
end
