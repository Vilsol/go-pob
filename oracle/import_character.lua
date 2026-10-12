
local inDir, outDir = arg[1], arg[2]
assert(inDir and outDir, "usage: import_character.lua <inDir> <outDir>")

dofile("HeadlessWrapper.lua")

local dkjson = require("dkjson")
local lfs = require("lfs")
lfs.mkdir(outDir)

local function readFile(path)
	local fh = io.open(path, "r")
	if not fh then
		return nil
	end
	local text = fh:read("*a")
	fh:close()
	return text
end

local minAllocatedNodes = 5

local files = {}
for file in lfs.dir(inDir) do
	local id = file:match("^(.+)%.items%.json$")
	if id then
		files[#files + 1] = id
	end
end
table.sort(files)

local counts = { ok = 0, noPassives = 0, noItems = 0, treeTooSmall = 0, itemsLost = 0, saveFailed = 0, decodeFailed = 0 }
local failures = {}

local function fail(class, id, detail)
	counts[class] = counts[class] + 1
	failures[#failures + 1] = string.format("%-14s %s%s", class, id, detail and ("  " .. detail) or "")
end

for _, id in ipairs(files) do
	local itemsText = readFile(inDir .. "/" .. id .. ".items.json")
	local passivesText = readFile(inDir .. "/" .. id .. ".passives.json")

	if not passivesText then
		fail("noPassives", id)
		goto continue
	end

	local items = itemsText and dkjson.decode(itemsText)
	local passives = dkjson.decode(passivesText)
	if not items or not passives or not items.character then
		fail("decodeFailed", id)
		goto continue
	end

	if #(items.items or {}) == 0 then
		fail("noItems", id)
		goto continue
	end

	do
		local charData = {
			name = items.character.name,
			league = items.character.league or "Standard",
			level = items.character.level,
			class = items.character.class,
			realm = items.character.realm,
		}
		charData.passives = passives
		charData.jewels = passives.items
		charData.equipment = items.items
		charData.guardian = items.guardian

		charData.passives.bandit_choice = charData.passives.bandit_choice or "None"
		charData.passives.pantheon_major = charData.passives.pantheon_major or "None"
		charData.passives.pantheon_minor = charData.passives.pantheon_minor or "None"

		newBuild()

		build.importTab:ImportPassiveTreeAndJewels(charData, false)
		build.importTab:ImportItemsAndSkills(charData, true, true, false)
		runCallback("OnFrame")

		local allocated = 0
		for _ in pairs(build.spec.allocNodes or {}) do
			allocated = allocated + 1
		end
		local wanted = #(passives.hashes or {})
		if allocated < minAllocatedNodes or (wanted > 0 and allocated < wanted / 2) then
			fail("treeTooSmall", id, string.format("allocated %d of %d hashes", allocated, wanted))
			goto continue
		end

		local imported = 0
		for _ in pairs(build.itemsTab.items or {}) do
			imported = imported + 1
		end
		if imported == 0 then
			fail("itemsLost", id, string.format("%d equipment entries produced 0 items", #items.items))
			goto continue
		end

		local xml = build:SaveDB("import")
		if not xml or not xml:match("<PathOfBuilding>") then
			fail("saveFailed", id)
			goto continue
		end

		local out = io.open(outDir .. "/" .. id .. ".xml", "w")
		if not out then
			fail("saveFailed", id, "could not open output")
			goto continue
		end
		out:write(xml)
		out:close()

		counts.ok = counts.ok + 1
		print(string.format("  ok  %-28s %-14s lvl %-3s  %d nodes  %d items",
			id, charData.class or "?", tostring(charData.level), allocated, imported))
	end

	::continue::
end

print("")
print(string.format("characters   %d", #files))
print(string.format("  ok           %d", counts.ok))
for _, class in ipairs({ "noItems", "noPassives", "decodeFailed", "treeTooSmall", "itemsLost", "saveFailed" }) do
	if counts[class] > 0 then
		print(string.format("  %-13s%d", class, counts[class]))
	end
end
if #failures > 0 then
	print("")
	print("failures:")
	for _, line in ipairs(failures) do
		print("  " .. line)
	end
end

if #files > 0 and counts.ok == 0 then
	print("")
	print("REFUSING: " .. #files .. " characters in, 0 builds out — see the failure classes above")
	os.exit(1)
end
