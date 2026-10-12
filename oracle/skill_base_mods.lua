
local outPath = arg[1]
assert(outPath, "usage: skill_base_mods.lua <out.json>")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local function scanSourceLines()
	local locs = {}
	local dir = "Data/Skills/"
	local p = io.popen("ls " .. dir)
	for file in p:lines() do
		if file:match("%.lua$") then
			local id = nil
			local n = 0
			for line in io.lines(dir .. file) do
				n = n + 1
				local skillId = line:match('^skills%["([^"]+)"%]%s*=')
				if skillId then
					id = skillId
				elseif id and line:match("^\tbaseMods%s*=") then
					locs[id] = dir .. file .. ":" .. n
					id = nil
				end
			end
		end
	end
	p:close()
	return locs
end

local locs = scanSourceLines()

local function encodeValue(v)
	local t = type(v)
	if t == "table" then
		local out = { __table = true }
		for k, sub in pairs(v) do
			out[tostring(k)] = encodeValue(sub)
		end
		return out
	elseif t == "boolean" then
		return { __bool = v }
	elseif t == "number" then
		return { __number = v }
	elseif t == "string" then
		return { __string = v }
	end
	return { __other = tostring(v) }
end

local function encodeMod(m)
	local out = {
		name = m.name,
		type = m.type,
		flags = m.flags or 0,
		keywordFlags = m.keywordFlags or 0,
		value = encodeValue(m.value),
		tags = {},
	}
	for _, tag in ipairs(m) do
		table.insert(out.tags, encodeValue(tag))
	end
	local extra = {}
	for k in pairs(m) do
		if type(k) == "string" and k ~= "name" and k ~= "type" and k ~= "value"
			and k ~= "flags" and k ~= "keywordFlags" and k ~= "source" then
			table.insert(extra, k)
		end
	end
	if #extra > 0 then
		table.sort(extra)
		out.extraKeys = extra
	end
	return out
end

local skills = {}
local count, modCount = 0, 0
for skillId, grantedEffect in pairs(data.skills) do
	if grantedEffect.baseMods and next(grantedEffect.baseMods) then
		local mods = {}
		for _, m in ipairs(grantedEffect.baseMods) do
			if m.name then
				table.insert(mods, encodeMod(m))
			else
				for _, sub in ipairs(m) do
					local e = encodeMod(sub)
					e.nested = true
					table.insert(mods, e)
				end
			end
		end
		skills[skillId] = {
			loc = locs[skillId],
			support = grantedEffect.support and true or false,
			mods = mods,
		}
		count = count + 1
		modCount = modCount + #mods
	end
end

local skillTypeNames = {}
for name, n in pairs(SkillType) do
	skillTypeNames[tostring(n)] = name
end

local function flagBits(t)
	local out = {}
	for name, value in pairs(t) do
		if type(value) == "number" then
			out[name] = value
		end
	end
	return out
end

local fh = assert(io.open(outPath, "w"))
fh:write(json.encode({
	skillTypeNames = skillTypeNames,
	modFlagBits = flagBits(ModFlag),
	keywordFlagBits = flagBits(KeywordFlag),
	skillCount = count,
	modCount = modCount,
	skills = skills,
}, { indent = true }))
fh:close()
io.stderr:write(string.format("oracle: %d skills with baseMods, %d mods -> %s\n", count, modCount, outPath))
