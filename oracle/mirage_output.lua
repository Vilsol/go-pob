
local buildPath, outPath = arg[1], arg[2]
assert(buildPath and outPath, "usage: mirage_output.lua <build.xml> <out.json>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function scalars(t)
	local out = {}
	if type(t) ~= "table" then
		return nil
	end
	for k, v in pairs(t) do
		local vt = type(v)
		if type(k) == "string" and (vt == "number" or vt == "string" or vt == "boolean") then
			out[k] = v
		end
	end
	return out
end

local fh = assert(io.open(buildPath, "r"))
local xmlText = fh:read("*a")
fh:close()

local name = buildPath:match("([^/]+)$")

wipeGlobalCache()
loadBuildFromXML(xmlText, name)
runCallback("OnFrame")

local mainEnv = build.calcsTab.mainEnv
local skill = mainEnv.player.mainSkill
local entry = { build = name }

entry.mainSkill = skill and skill.activeEffect and skill.activeEffect.grantedEffect
	and skill.activeEffect.grantedEffect.name

entry.playerOutput = {
	MirageDPS = mainEnv.player.output.MirageDPS,
	CombinedDPS = mainEnv.player.output.CombinedDPS,
	TotalDPS = mainEnv.player.output.TotalDPS,
	TotalDot = mainEnv.player.output.TotalDot,
	IgniteDPS = mainEnv.player.output.IgniteDPS,
	BleedDPS = mainEnv.player.output.BleedDPS,
	PoisonDPS = mainEnv.player.output.PoisonDPS,
}
entry.DotCanStack = skill and skill.skillFlags and skill.skillFlags.DotCanStack or false

if not skill then
	entry.error = "build has no main skill"
elseif not skill.mirage then
	entry.error = "main skill has no mirage table: not triggeredByMirageArcher, or "
		.. "CalcPerform.lua:2496-2497 found no usable skill (infoMessage2 = "
		.. tostring(skill.infoMessage2) .. ")"
else
	entry.mirage = {
		count = skill.mirage.count,
		name = skill.mirage.name,
		skillPartName = skill.mirage.skillPartName,
		output = scalars(skill.mirage.output),
	}
	if not entry.mirage.output then
		entry.error = "mirage table present but mirage.output is nil"
	end
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode(entry, { indent = true }))
out:close()
io.stderr:write("oracle: wrote " .. name .. " -> " .. outPath
	.. (entry.error and (" (" .. entry.error .. ")") or "") .. "\n")
