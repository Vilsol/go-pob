-- usage (from PoB-PoE2's src/): luajit poe2_gem_search.lua <out.json>
-- Records the gem list PoB-PoE2's gem picker builds for each search and support filter: GemSelectControl's
-- PopulateGemList, FilterSupport and BuildList, sorted by name (no DPS sort, nothing it can support).
local selfPath, outPath = arg[0], arg[1]
assert(outPath, "usage: poe2_gem_search.lua <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local tab = build.skillsTab
tab.sortGemsByDPS = false
local control = tab.gemSlots[1].nameSpec

local queries = { "", "fire", "fireball", "FB", "ice", "strike", ":fire:-spell", ":active:cold", ":aoe:int", "aoe", "active",
	"support", "str", "lineage", "zzz", "elemental focus", "Lightning" }
local out = {}
for _, show in ipairs({ "ALL", "LINEAGE", "NORMAL" }) do
	for _, legacy in ipairs({ false, true }) do
		tab.showSupportGemTypes = show
		tab.showLegacyGems = legacy
		control:PopulateGemList()
		control.sortCache = { canSupport = { }, dps = { } }
		local results = { }
		for _, q in ipairs(queries) do
			control:BuildList(q)
			local names = { }
			for _, gemId in ipairs(control.list) do
				if gemId ~= "" then
					names[#names + 1] = control.gems[gemId].name
				end
			end
			results[#results + 1] = { query = q, names = names }
		end
		out[#out + 1] = { showSupport = show, showLegacy = legacy, results = results }
	end
end

local o = assert(io.open(outPath, "w"))
o:write(json.encode(stable(out), { indent = true }))
o:close()
