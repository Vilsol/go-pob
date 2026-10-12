
local selfPath = arg[0]
local casesPath, outPath = arg[1], arg[2]
assert(casesPath and outPath, "usage: item_tools.lua <cases.json> <out.json>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local f = assert(io.open(casesPath, "r"))
local cases = assert(json.decode(f:read("*a")))
f:close()

local function caseText(c)
	if c.textBytes then
		local bytes = {}
		for i, b in ipairs(c.textBytes) do
			bytes[i] = string.char(b)
		end
		return table.concat(bytes)
	end
	return c.text
end

local required = {
	{ name = "itemLib.applyValueScalar", fn = itemLib.applyValueScalar },
	{ name = "itemLib.applyRange", fn = itemLib.applyRange },
	{ name = "sanitiseText (Common.lua, a GLOBAL -- it was itemLib.sanitiseItemText)", fn = sanitiseText },
}
for _, r in ipairs(required) do
	assert(type(r.fn) == "function",
		"REFUSING: " .. r.name .. " is not a function in the pinned PoB; the oracle cannot ground this port")
end

local handlers = {
	applyValueScalar = function(c)
		return { out = itemLib.applyValueScalar(c.line, c.valueScalar, nil, c.numbers, nil) }
	end,
	applyRange = function(c)
		return { out = itemLib.applyRange(c.line, c.range, c.valueScalar, nil) }
	end,
	sanitiseItemText = function(c)
		return { out = sanitiseText(caseText(c)) }
	end,
}

local results = {}
for i, c in ipairs(cases) do
	local h = assert(handlers[c.fn],
		"REFUSING: no handler for fn " .. tostring(c.fn) .. " -- this oracle cannot ground it")
	local ok, res = pcall(h, c)
	if not ok then
		res = { error = tostring(res) }
	end
	res.fn = c.fn
	res.name = c.name
	results[i] = res
end

local out = assert(io.open(outPath, "w"))
out:write(json.encode(stable(results), { indent = true }))
out:close()
print("oracle: wrote " .. #results .. " results to " .. outPath)
