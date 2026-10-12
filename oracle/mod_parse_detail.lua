local selfPath = arg[0]
local linesPath = arg[1]
assert(linesPath, "usage: mod_parse_detail.lua <lines.txt>")

local wanted = {}
for line in assert(io.open(linesPath, "r")):lines() do
	if line ~= "" and line:sub(1, 1) ~= "#" then table.insert(wanted, line) end
end

dofile("HeadlessWrapper.lua")

local function fmtval(v)
	if type(v) == "table" then
		if v.name then return "mod<" .. tostring(v.name) .. " " .. tostring(v.type) .. " " .. tostring(v.value) .. ">" end
		local parts = {}
		for k, x in pairs(v) do parts[#parts + 1] = tostring(k) .. "=" .. tostring(type(x) == "table" and "{...}" or x) end
		table.sort(parts)
		return "{" .. table.concat(parts, " ") .. "}"
	end
	return tostring(v)
end

for _, line in ipairs(wanted) do
	modLib.parseModCache[line] = nil
	local modList, extra = modLib.parseMod(line)
	print("LINE  " .. line)
	print("  extra=" .. tostring(extra) .. "  count=" .. tostring(modList and #modList or 0))
	if modList then
		for i, m in ipairs(modList) do
			local tags = {}
			for _, t in ipairs(m) do
				local parts = {}
				for k, v in pairs(t) do parts[#parts + 1] = k .. "=" .. fmtval(v) end
				table.sort(parts)
				tags[#tags + 1] = "[" .. table.concat(parts, " ") .. "]"
			end
			print(string.format("  #%d name=%s type=%s value=%s flags=%s keywordFlags=%s source=%s %s",
				i, tostring(m.name), tostring(m.type), fmtval(m.value),
				tostring(m.flags), tostring(m.keywordFlags), tostring(m.source),
				table.concat(tags, " ")))
		end
	end
	print("")
end
