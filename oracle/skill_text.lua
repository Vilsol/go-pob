
local selfPath = arg[0]
local outPath = arg[1]
assert(outPath, "usage: skill_text.lua <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

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
local pinLoadfile = loadfile
function loadfile(fileName, ...)
	if overlay[fileName] then
		return loadstring(overlay[fileName], "@" .. fileName)
	end
	return pinLoadfile(fileName, ...)
end

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local effects = {}
local effectCount = 0
for id, effect in pairs(data.skills) do
	local flavour
	if effect.flavourText then
		flavour = {}
		for i, line in ipairs(effect.flavourText) do
			flavour[i] = line
		end
	end
	local quality
	if effect.qualityStats and effect.qualityStats[1] then
		quality = {}
		for i, stat in ipairs(effect.qualityStats) do
			quality[i] = { stat = stat[1], value = stat[2] }
		end
	end
	effects[id] = {
		qualityStats = quality,
		name = effect.name,
		description = effect.description,
		castTime = effect.castTime,
		scope = effect.statDescriptionScope,
		flavourText = flavour,
		legacy = effect.legacy and true or nil,
	}
	effectCount = effectCount + 1
end
assert(effectCount > 1000, "REFUSING: only " .. effectCount .. " granted effects; Data/Skills did not load")

local gems = {}
for _, gem in pairs(data.gems) do
	gems[gem.grantedEffectId] = {
		tagString = gem.tagString,
		secondaryEffectName = gem.secondaryEffectName,
		vaalGem = gem.vaalGem and true or nil,
	}
end

local out = assert(io.open(outPath, "w"))
local gold = {}
for i, v in ipairs(data.goldRespecPrices) do
	gold[i] = v
end
assert(#gold >= 100, "REFUSING: data.goldRespecPrices has " .. #gold .. " levels")
out:write(json.encode(stable({ pobVersion = tostring(launch.versionNumber or "unknown"), effects = effects, gems = gems, goldRespecPrices = gold }), { indent = true }))
out:close()
io.stderr:write(string.format("skill-text: %d effects, %d gems written to %s\n", effectCount, #(function() local n = {} for k in pairs(gems) do n[#n + 1] = k end return n end)(), outPath))
