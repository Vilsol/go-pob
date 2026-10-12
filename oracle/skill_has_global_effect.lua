
local outPath = arg[1]
assert(outPath, "usage: skill_has_global_effect.lua <out.json>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

assert(data.skills, "REFUSING: the pinned PoB does not expose data.skills")

local ids, total = {}, 0
for id, grantedEffect in pairs(data.skills) do
	total = total + 1
	if grantedEffect.hasGlobalEffect then
		ids[#ids + 1] = tostring(id)
	end
end

assert(total > 0, "REFUSING: data.skills is empty")
assert(#ids > 0, string.format(
	"REFUSING: not one of %d granted effects carries hasGlobalEffect. processMod " ..
	"(Modules/Data.lua:66-77) sets it during boot, so a zero here means the flag moved or " ..
	"this dump ran before the data loaded -- not that PoB has no global-effect skills.", total))

table.sort(ids)

local out = {
	pobVersion = tostring(launch.versionNumber or "unknown"),
	skillCount = total,
	ids = ids,
}

local file = assert(io.open(outPath, "w"), "REFUSING: cannot write " .. outPath)
file:write(json.encode(out, { indent = true }))
file:close()

io.stderr:write(string.format("wrote %s (PoB %s, %d of %d granted effects flagged)\n",
	outPath, out.pobVersion, #ids, total))
