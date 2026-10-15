
local outPath = arg[1]
assert(outPath, "usage: skill_hidden_levels.lua <out.json>")

local overlayPath = assert(os.getenv("GAMEDATA_DUMP"),
	"REFUSING: GAMEDATA_DUMP is not set; the game data comes from `gamedatadump skills-data`")
local overlay
do
	local f = assert(io.open(overlayPath, "r"), "REFUSING: cannot read " .. overlayPath)
	overlay = assert(require("dkjson").decode(f:read("*a")), "REFUSING: " .. overlayPath .. " is not JSON")
	f:close()
	assert(overlay["Data/Skills/act_str.lua"] and overlay["Data/Gems.lua"] and overlay["Data/Misc.lua"],
		"REFUSING: " .. overlayPath .. " carries no Data/Skills, Data/Gems.lua or Data/Misc.lua")
end
local compat = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua")
local pinLoadfile = loadfile
function loadfile(fileName, ...)
	if overlay[fileName] then
		return loadstring(compat.adaptOverlay(fileName, overlay[fileName]), "@" .. fileName)
	end
	return pinLoadfile(fileName, ...)
end

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local WANTED = {
	{
		id = "EmptyActionSpellWarlordGrandmaster",
		note = '"Arena Master\'s Presence". Reached from the Arena Master spectre\'s skillList ' ..
			"(Data/Spectres.lua, spectre_data.go). buffAllies + buffMinions, so its four values " ..
			"land on the player as well as the minion.\n" ..
			"go-pob-data's ActiveSkillTypes for this effect is EMPTY, and the Buff member is " ..
			"load-bearing: ConfigOptions.lua's `raiseSpectreEnableBuffs` (default checked) tags " ..
			"its `enable` payload with SkillType Buff, so without it the skill never enables and " ..
			"CalcPerform.lua skips the buff entirely.",
	},
	{
		id = "DTTHellscapeStabWeb",
		note = '"Thunder Web". Reached from the Pale Seraphim spectre\'s skillList ' ..
			"(Data/Spectres.lua, spectre_data.go). Its last two stats carry GlobalEffect Debuff " ..
			"mods (skill_stat_maps.go), so they land on the ENEMY.\n" ..
			"This entry is a SHORT stat list rather than an absent one, which is why it needed " ..
			"finding twice. go-pob-data HAS three of PoB's five stats, at the right values and " ..
			"in PoB's order, and drops exactly the two that carry mods -- so the skill built, " ..
			"held plausible data and published no buff at all.",
	},
	{
		id = "DelveMonsterEnfeebleOnHit",
		note = '"Enfeeble", a hidden hex reached from the Delve Wraith spectre\'s skillList ' ..
			"(Data/Spectres.lua, spectre_data.go). Its mod-bearing stats carry GlobalEffect " ..
			"Curse mods, so they land on the ENEMY through the minion-curse arm at " ..
			"CalcPerform.lua:1991-2001.\n" ..
			"⚠️ THE LEVEL KEY IS 8 AND THAT IS NOT A TYPO. PoB writes `levels = { [8] = {...} }`, " ..
			"so `#` and ipairs both see an empty sequence: createMinionSkills leaves " ..
			"activeEffect.level at 1, and calcLib.validateGemLevel's last arm (CalcTools.lua:56, " ..
			"`next(grantedEffect.levels)`) is what recovers 8. Our CalcValidateGemLevel ports " ..
			"that arm. Keying this at 1 would be a fitted value that happens to work.\n" ..
			"A level row SHORTER than the stat list is PoB's own shape, not a dump error: " ..
			"`level[index] or 1` (CalcTools.lua:166) supplies the tail, and calc_tools.go ports " ..
			"that fallback, so the missing values are deliberately not written out.",
	},
	{
		id = "HarvestNessaCrabScreechDebuff",
		note = '"Frigid Roar". Reached from the Primal Crushclaw spectre\'s skillList ' ..
			"(Data/Spectres.lua, spectre_data.go). go-pob-data has NO granted effect for it, so " ..
			"it also needs a pobOnlyGrantedEffects shell (calc_active_skill.go); its one stat " ..
			"carries a GlobalEffect Debuff mod, so it lands on the ENEMY.",
	},
}

local skillTypeNames = {}
for name, n in pairs(SkillType) do
	skillTypeNames[tostring(n)] = name
end

local entries, order = {}, {}
local missing = {}
for _, want in ipairs(WANTED) do
	local ge = data.skills[want.id]
	if not ge then
		missing[#missing + 1] = want.id
	else
		if not ge.stats or #ge.stats == 0 then
			io.stderr:write("REFUSING: " .. want.id .. " has no `stats` in the pinned PoB. This " ..
				"table exists to supply exactly that, so an empty one is a no-op that compiles.\n")
			os.exit(3)
		end
		if not ge.levels or not next(ge.levels) then
			io.stderr:write("REFUSING: " .. want.id .. " has no `levels` in the pinned PoB.\n")
			os.exit(3)
		end

		local stats = {}
		for i = 1, #ge.stats do
			stats[i] = ge.stats[i]
		end

		local typeNums = {}
		for n in pairs(ge.skillTypes or {}) do
			typeNums[#typeNums + 1] = n
		end
		table.sort(typeNums)
		local types = {}
		for _, n in ipairs(typeNums) do
			local name = skillTypeNames[tostring(n)]
			if not name then
				io.stderr:write("REFUSING: " .. want.id .. " carries SkillType " .. tostring(n) ..
					", which has no name in Data/Global.lua.\n")
				os.exit(3)
			end
			types[#types + 1] = name
		end

		local levels = {}
		for key, row in pairs(ge.levels) do
			local vals = {}
			for i = 1, #row do
				vals[i] = row[i]
			end
			levels[tostring(key)] = vals
		end

		entries[want.id] = { note = want.note, stats = stats, skillTypes = types, levels = levels }
		order[#order + 1] = want.id
	end
end

if #missing > 0 then
	io.stderr:write("REFUSING: the pinned PoB has no hidden skill(s): " ..
		table.concat(missing, ", ") .. "\nEither upstream renamed or removed them -- in which " ..
		"case the entry must GO, not be silently skipped -- or the id is a typo. Emitting the " ..
		"rest would drop a row nothing downstream would miss.\n")
	os.exit(3)
end

local fh = assert(io.open(outPath, "w"))
fh:write(json.encode({
	pobVersion = tostring(launch.versionNumber or "unknown"),
	order = order,
	entries = entries,
}, { indent = true }))
fh:close()

io.stderr:write(string.format("wrote %s (PoB %s, %d hidden skills)\n",
	outPath, tostring(launch.versionNumber or "unknown"), #order))
