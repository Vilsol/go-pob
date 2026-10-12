
local outFile = arg[1]

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

local typeName = {}
for name, value in pairs(SkillType) do
	typeName[value] = name
end

local function scanDuplicates()
	local dupes = {}
	local dir = "Data/Skills/"
	local files = {}
	for path in pairs(overlay) do
		local file = path:match("^Data/Skills/(.+)$")
		if file then
			files[#files + 1] = file
		end
	end
	table.sort(files)
	for _, file in ipairs(files) do
		if file:match("%.lua$") then
			local id, seen = nil, {}
			local n = 0
			for line in (overlay[dir .. file] .. "\n"):gmatch("([^\n]*)\n") do
				n = n + 1
				local skillId = line:match('^skills%["([^"]+)"%]%s*=')
				if skillId then
					id = skillId
				elseif id and line:match("^\taddSkillTypes%s*=") then
					seen[id] = seen[id] or {}
					table.insert(seen[id], { file = "src/" .. dir .. file, line = n, text = (line:gsub("^%s+", "")) })
				end
			end
			for skillId, hits in pairs(seen) do
				if #hits > 1 then
					dupes[skillId] = hits
				end
			end
		end
	end
	return dupes
end

local duplicates = scanDuplicates()

local order, entries, lines = {}, {}, {}
local unknown = {}
for id, skill in pairs(data.skills) do
	if skill.support and skill.addSkillTypes and next(skill.addSkillTypes) then
		local names = {}
		for _, skillType in pairs(skill.addSkillTypes) do
			local name = typeName[skillType]
			if not name then
				unknown[#unknown + 1] = id .. " -> " .. tostring(skillType)
			else
				names[#names + 1] = name
			end
		end
		table.sort(names)
		order[#order + 1] = id
		entries[id] = names
		lines[#lines + 1] = id .. "\t" .. table.concat(names, ",")
	end
end
table.sort(order)
table.sort(lines)

if #unknown > 0 then
	io.stderr:write("REFUSING: addSkillTypes carries numbers with no name in Data/Global.lua:\n")
	for _, u in ipairs(unknown) do
		io.stderr:write("  " .. u .. "\n")
	end
	os.exit(3)
end
assert(#order > 0, "REFUSING: no support in the pinned PoB declares addSkillTypes")

local function setNames(t, id, field)
	local names = {}
	for skillType, present in pairs(t) do
		if present then
			local name = typeName[skillType]
			if not name then
				unknown[#unknown + 1] = id .. "." .. field .. " -> " .. tostring(skillType)
			else
				names[#names + 1] = name
			end
		end
	end
	table.sort(names)
	return names
end
local function exprNames(t, id, field)
	local names = {}
	for _, skillType in ipairs(t) do
		local name = typeName[skillType]
		if not name then
			unknown[#unknown + 1] = id .. "." .. field .. " -> " .. tostring(skillType)
		else
			names[#names + 1] = name
		end
	end
	return names
end
local skillTypes, excludeSkillTypes, requireSkillTypes = {}, {}, {}
for id, skill in pairs(data.skills) do
	if skill.skillTypes then
		skillTypes[id] = setNames(skill.skillTypes, id, "skillTypes")
	end
	if skill.support then
		excludeSkillTypes[id] = exprNames(skill.excludeSkillTypes or {}, id, "excludeSkillTypes")
		requireSkillTypes[id] = exprNames(skill.requireSkillTypes or {}, id, "requireSkillTypes")
	end
end
if #unknown > 0 then
	io.stderr:write("REFUSING: a skill-type table carries numbers with no name in Data/Global.lua:\n")
	for _, u in ipairs(unknown) do
		io.stderr:write("  " .. u .. "\n")
	end
	os.exit(3)
end

local fh = assert(io.open(assert(outFile, "usage: skill_add_skill_types.lua <out.json>"), "w"))
fh:write(json.encode({
	pobVersion = tostring(launch.versionNumber or "unknown"),
	order = order,
	entries = entries,
	duplicates = duplicates,
	skillTypes = skillTypes,
	excludeSkillTypes = excludeSkillTypes,
	requireSkillTypes = requireSkillTypes,
}, { indent = true }))
fh:close()

for _, line in ipairs(lines) do
	io.stderr:write(line, "\n")
end

io.stderr:write(("%d support(s) declare addSkillTypes\n"):format(#lines))
