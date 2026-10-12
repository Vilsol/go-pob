
local FALSE = setmetatable({}, {
	__tojson = function()
		return "false"
	end,
})

local function keyLess(a, b)
	local ta, tb = type(a), type(b)
	if ta ~= tb then
		return ta < tb
	end
	if ta == "number" or ta == "string" then
		return a < b
	end
	return tostring(a) < tostring(b)
end

local function stable(value)
	if type(value) ~= "table" then
		return value
	end
	local keys = {}
	for k in pairs(value) do
		keys[#keys + 1] = k
	end
	for _, k in ipairs(keys) do
		local v = value[k]
		if v == false then
			value[k] = FALSE
		elseif type(v) == "table" then
			stable(v)
		end
	end
	table.sort(keys, keyLess)
	local mt = getmetatable(value) or {}
	mt.__jsonorder = keys
	setmetatable(value, mt)
	return value
end

return stable
