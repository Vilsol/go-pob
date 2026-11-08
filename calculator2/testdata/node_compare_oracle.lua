-- Loads each build and builds the tree's node tooltip (stat differences on) for a spread of nodes, one JSON row per
-- node: the tooltip's lines from the stat difference separator on, and every calculation the tooltip ran (its
-- addNodes / removeNodes ids and the numeric outputs that differ from the base pass, the minion's too), after a row per
-- build with the calculator's base pass and the main skill's flags.
-- usage (from PoB-PoE2's src/): node_compare_oracle.lua <out.jsonl> <build.xml>...
local outPath = arg[1]
assert(outPath and arg[2], "usage: node_compare_oracle.lua <out.jsonl> <build.xml>...")
local xmlPaths = {}
for i = 2, #arg do
	xmlPaths[#xmlPaths + 1] = arg[i]
end

dofile("HeadlessWrapper.lua")

local function encodeString(s)
	return '"' .. s:gsub('[%c"\\\128-\255]', function(c)
		local map = { ['"'] = '\\"', ["\\"] = "\\\\", ["\n"] = "\\n", ["\t"] = "\\t", ["\r"] = "\\r" }
		return map[c] or string.format("\\u%04x", c:byte())
	end) .. '"'
end
local function encodeNumber(v)
	if v ~= v or v == math.huge or v == -math.huge then
		return '"' .. tostring(v) .. '"'
	end
	return string.format("%.17g", v)
end
-- the numeric outputs, or with base those that differ from it and the base's that it lacks (as null)
local function encodeOutput(output, base)
	local keys, seen = {}, {}
	for k, v in pairs(output or {}) do
		if type(v) == "number" and (not base or base[k] ~= v) then
			keys[#keys + 1] = k
			seen[k] = true
		end
	end
	for k, v in pairs(base or {}) do
		if type(v) == "number" and not seen[k] and type((output or {})[k]) ~= "number" then
			keys[#keys + 1] = k
		end
	end
	table.sort(keys)
	local parts = {}
	for i, k in ipairs(keys) do
		local v = (output or {})[k]
		parts[i] = encodeString(k) .. ":" .. (type(v) == "number" and encodeNumber(v) or "null")
	end
	return "{" .. table.concat(parts, ",") .. "}"
end
local function encodeFlags(flags)
	local keys = {}
	for k, v in pairs(flags or {}) do
		if v == true then
			keys[#keys + 1] = encodeString(k) .. ":true"
		end
	end
	table.sort(keys)
	return "{" .. table.concat(keys, ",") .. "}"
end
local function encodeIds(set)
	local ids, byId = {}, {}
	for k in pairs(set or {}) do
		if type(k) == "table" then
			ids[#ids + 1] = k.id
		else
			byId[#byId + 1] = k
		end
	end
	table.sort(ids)
	table.sort(byId)
	return "[" .. table.concat(ids, ",") .. "]", "[" .. table.concat(byId, ",") .. "]"
end
local function readFile(path)
	local fh = assert(io.open(path, "r"))
	local text = fh:read("*a")
	fh:close()
	return text
end

local out = assert(io.open(outPath, "w"))
for _, xmlPath in ipairs(xmlPaths) do
	local name = xmlPath:match("([^/]+)%.xml$")
	loadBuildFromXML(readFile(xmlPath), name)
	local build = __mainObject__.main.modes.BUILD
	build.buildFlag = true
	runCallback("OnFrame")
	local spec = build.spec
	local viewer = build.treeTab.viewer
	viewer.showStatDifferences = true

	local calls
	local getMiscCalculator = build.calcsTab.GetMiscCalculator
	local calcFunc0, calcBase = getMiscCalculator(build.calcsTab)
	build.calcsTab.GetMiscCalculator = function()
		return function(override, ...)
			local output = calcFunc0(override, ...)
			calls[#calls + 1] = { override = override, output = output }
			return output
		end, calcBase
	end

	local env = build.calcsTab.mainEnv
	out:write('{"build":', encodeString(name), ',"base":', encodeOutput(calcBase), ',"baseMinion":', encodeOutput(calcBase.Minion),
		',"skillFlags":', encodeFlags(env.player.mainSkill.activeEffect.statSet.skillFlags), ',"minion":',
		tostring(env.player.mainSkill.minion ~= nil), "}\n")

	-- a spread of nodes: unallocated ones with a path (some far), allocated ones (some with dependants), granted ones
	local ids = {}
	for id in pairs(spec.nodes) do
		ids[#ids + 1] = id
	end
	table.sort(ids)
	local add, far, remove, removeMany, granted = {}, {}, {}, {}, {}
	for _, id in ipairs(ids) do
		local node = spec.nodes[id]
		if node.type ~= "ClassStart" and node.type ~= "AscendClassStart" and node.modKey ~= "" then
			if env.grantedPassives[id] and not node.alloc then
				granted[#granted + 1] = node
			elseif node.alloc then
				if #node.depends > 1 then
					removeMany[#removeMany + 1] = node
				else
					remove[#remove + 1] = node
				end
			elseif node.path then
				if #node.path > 2 then
					far[#far + 1] = node
				else
					add[#add + 1] = node
				end
			end
		end
	end
	-- PassiveTreeView.lua:923-926
	local incSmallPassiveSkillEffect = 0
	for _, n in pairs(spec.allocNodes) do
		incSmallPassiveSkillEffect = incSmallPassiveSkillEffect + n.modList:Sum("INC", nil, "SmallPassiveSkillEffect")
	end
	local picked = {}
	local function pick(list, n)
		for k = 1, math.min(n, #list) do
			picked[#picked + 1] = list[math.floor((k - 1) * #list / math.min(n, #list)) + 1]
		end
	end
	pick(add, 4)
	pick(far, 3)
	pick(remove, 3)
	pick(removeMany, 3)
	pick(granted, 2)

	for _, node in ipairs(picked) do
		calls = {}
		local tooltip = new("Tooltip"):Tooltip()
		viewer:AddNodeTooltip(tooltip, node, build, incSmallPassiveSkillEffect)
		local lines, on = {}, false
		for _, line in ipairs(tooltip.lines) do
			local text = line.text or ""
			if text:match("will give you:") or text:match("^%^7No changes") or text:match("granted by an item") then
				on = true
			end
			if on and text ~= "" then
				lines[#lines + 1] = encodeString(text)
			end
			if text:match("Tip: Press Ctrl%+D") then
				on = false
			end
			if text:match("points to node") or text:match("To reach this node") or text:match("points gained from") or text:match("Gold ") then
				lines[#lines + 1] = encodeString(text)
			end
		end
		local calcParts = {}
		for k, c in ipairs(calls) do
			local o = c.override or {}
			local addIds = encodeIds(o.addNodes)
			local removeIds, removeById = encodeIds(o.removeNodes)
			calcParts[k] = '{"add":' .. addIds .. ',"remove":' .. removeIds .. ',"removeIds":' .. removeById ..
				',"output":' .. encodeOutput(c.output, calcBase) .. ',"minion":' .. encodeOutput(c.output.Minion, calcBase.Minion) .. "}"
		end
		out:write('{"build":', encodeString(name), ',"node":', node.id, ',"alloc":', tostring(node.alloc and true or false),
			',"lines":[', table.concat(lines, ","), '],"calcs":[', table.concat(calcParts, ","), "]}\n")
	end
	build.calcsTab.GetMiscCalculator = getMiscCalculator
end
out:close()
