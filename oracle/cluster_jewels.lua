
local outPath = arg[1]
assert(outPath, "usage: cluster_jewels.lua <out.json>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

assert(data.clusterJewels, "REFUSING: the pinned PoB does not expose data.clusterJewels")
assert(data.clusterJewels.jewels, "REFUSING: data.clusterJewels has no .jewels")
assert(data.clusterJewels.keystones, "REFUSING: data.clusterJewels has no .keystones")
assert(data.clusterJewels.notableSortOrder, "REFUSING: data.clusterJewels has no .notableSortOrder")
assert(data.clusterJewels.orbitOffsets, "REFUSING: data.clusterJewels has no .orbitOffsets")

local jewelFields = {
	size = "string", sizeIndex = "number", minNodes = "number", maxNodes = "number",
	totalIndicies = "number", smallIndicies = "array", notableIndicies = "array",
	socketIndicies = "array", skills = "skills",
}
local skillFields = {
	name = "string", icon = "string", masteryIcon = "string", tag = "string",
	stats = "array", enchant = "array",
}

local unknown = {}

local function copyArray(t)
	local out = {}
	for i = 1, #t do
		out[i] = t[i]
	end
	return out
end

local function checkFields(where, row, known)
	for k in pairs(row) do
		if not known[k] then
			unknown[#unknown + 1] = where .. "." .. tostring(k)
		end
	end
end

local jewels = {}
for name, jewel in pairs(data.clusterJewels.jewels) do
	checkFields("jewels." .. name, jewel, jewelFields)
	local skills = {}
	for skillKey, skill in pairs(jewel.skills or {}) do
		checkFields("jewels." .. name .. ".skills." .. skillKey, skill, skillFields)
		skills[tostring(skillKey)] = {
			name = skill.name,
			icon = skill.icon,
			masteryIcon = skill.masteryIcon,
			tag = skill.tag,
			stats = copyArray(skill.stats or {}),
			enchant = copyArray(skill.enchant or {}),
		}
	end
	jewels[tostring(name)] = {
		size = jewel.size,
		sizeIndex = jewel.sizeIndex,
		minNodes = jewel.minNodes,
		maxNodes = jewel.maxNodes,
		totalIndicies = jewel.totalIndicies,
		smallIndicies = copyArray(jewel.smallIndicies or {}),
		notableIndicies = copyArray(jewel.notableIndicies or {}),
		socketIndicies = copyArray(jewel.socketIndicies or {}),
		skills = skills,
	}
end

if #unknown > 0 then
	io.stderr:write("REFUSING: PoB carries fields this emitter does not model:\n")
	for _, f in ipairs(unknown) do
		io.stderr:write("  " .. f .. "\n")
	end
	io.stderr:write("Emitting anyway would silently drop them from data/cluster_jewels.go, and " ..
		"nothing downstream would notice. Model them here and in cluster_jewels_emit.py.\n")
	os.exit(3)
end

local orbitOffsets = {}
for proxy, starts in pairs(data.clusterJewels.orbitOffsets) do
	local row = {}
	for sizeIndex, start in pairs(starts) do
		row[tostring(sizeIndex)] = start
	end
	orbitOffsets[tostring(proxy)] = row
end

local out = {
	pobVersion = tostring(launch.versionNumber or "unknown"),
	jewels = jewels,
	keystones = copyArray(data.clusterJewels.keystones),
	notableSortOrder = data.clusterJewels.notableSortOrder,
	orbitOffsets = orbitOffsets,
}

local file = assert(io.open(outPath, "w"))
file:write(json.encode(out, { indent = true }))
file:close()

local n = 0
for _ in pairs(jewels) do n = n + 1 end
io.stderr:write(string.format("wrote %s (PoB %s, %d jewels, %d keystones)\n",
	outPath, out.pobVersion, n, #out.keystones))
