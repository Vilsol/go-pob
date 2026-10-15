local outPath = arg[1]
local files = {}
for i = 2, #arg do
	files[#files + 1] = arg[i]
end
if not pcall(require, "zlib") then
	package.preload.zlib = function()
		local refuse = function() error("lua-zlib is not installed (mise run bootstrap in the PoB checkout)") end
		return { deflate = refuse, inflate = refuse }
	end
end
if not pcall(require, "lua-utf8") then
	package.preload["lua-utf8"] = function() return string end
end

local construct = dofile((arg[0]:match("^(.*)[/\\]") or ".") .. "/pob_compat.lua").construct
dofile("HeadlessWrapper.lua")
local json = require("dkjson")

local candidates = {
	"Rarity: RARE\nCandidate Amulet\nOnyx Amulet\n+10 to all Attributes",
	"Rarity: RARE\nCandidate Helm\nHubris Circlet\nQuality: 3\n+50 to maximum Life",
	"Rarity: RARE\nCandidate Plate\nGlorious Plate\n+50 to maximum Life",
	"Rarity: MAGIC\nFingerless Silk Gloves of the Whale",
	"Rarity: NORMAL\nTwo-Toned Boots",
	"Rarity: RARE\nCandidate Bow\nThicket Bow\nAdds 10 to 20 Physical Damage",
	"Rarity: RARE\nShaped Helm\nHubris Circlet\nShaper Item\n+50 to maximum Life",
	"Rarity: RARE\nCorrupt Plate\nGlorious Plate\n+50 to maximum Life\nCorrupted",
	"Rarity: RARE\nCandidate Ring\nTwo-Stone Ring\n+50 to maximum Life",
}

local function lines(list)
	local out = {}
	for _, l in ipairs(list) do
		out[#out + 1] = l.line
	end
	return out
end

local result = {}
for _, path in ipairs(files) do
	local fh = assert(io.open(path, "r"))
	local xmlText = fh:read("*a")
	fh:close()
	local name = path:match("([^/]+)$")
	local ok, err = pcall(function()
		wipeGlobalCache()
		loadBuildFromXML(xmlText, name)
		runCallback("OnFrame")
	end)
	assert(ok and build.buildName == name, "REFUSING: " .. name .. ": " .. tostring(err))
	local entry = {}
	for k, raw in ipairs(candidates) do
		local newItem = construct("Item", raw)
		assert(newItem.base, "candidate " .. k .. " has no base")
		build.itemsTab:CopyAnointsAndEldritchImplicits(newItem, true, false)
		newItem:NormaliseQuality()
		entry[k] = {
			enchant = lines(newItem.enchantModLines),
			implicit = lines(newItem.implicitModLines),
			cleansing = newItem.cleansing or false,
			tangle = newItem.tangle or false,
			quality = newItem.quality or 0,
		}
	end
	result[name] = { candidates = candidates, items = entry }
end
local out = assert(io.open(outPath, "w"))
out:write(json.encode(result, { keyorder = {} }))
out:close()
print("wrote " .. outPath .. " (" .. #files .. " builds)")
