-- poe2configcorpus.lua options <out.tsv>
-- poe2configcorpus.lua visible <out dir> <build.xml>...
-- Run from PoB-PoE2's src/. "options" lists ConfigOptions.lua's varList; "visible" loads each build
-- and writes <out dir>/<build>.txt: one var per line whose control the Configuration tab shows.
local mode = arg[1]
assert(mode == "options" or mode == "visible", "usage: options <out.tsv> | visible <outDir> <xml>...")

dofile("HeadlessWrapper.lua")

local function clean(s)
	return (tostring(s):gsub("%^x%x%x%x%x%x%x", ""):gsub("%^%d", ""):gsub("[\t\r\n]", " "))
end

local function typed(v)
	if type(v) == "number" then
		return "n:" .. string.format("%.17g", v)
	elseif type(v) == "boolean" then
		return "b:" .. tostring(v)
	elseif v == nil then
		return ""
	end
	return "s:" .. clean(v)
end

local function joined(v)
	if v == nil then
		return ""
	elseif type(v) == "table" then
		local parts = {}
		for _, x in ipairs(v) do
			parts[#parts + 1] = clean(x)
		end
		return table.concat(parts, "\31")
	end
	return clean(v)
end

local predicateKeys = { "ifCond", "ifEnemyCond", "ifMinionCond", "ifMult", "ifEnemyMult", "ifStat", "ifEnemyStat",
	"ifFlag", "ifPlayerFlag", "ifMinionFlag", "ifMod", "ifSkill", "ifSkillType", "ifSkillFlag", "ifSkillData",
	"ifGemFamily", "ifTagType", "ifOption", "ifNode", "implyCond", "implyCondList", "implyMinionCond", "implyEnemyCond" }

if mode == "options" then
	local varList = LoadModule("Modules/ConfigOptions")
	local out = assert(io.open(arg[2], "w"))
	out:write("index\tvar\ttype\tdefaultState\tdefaultIndex\tlist\tpredicates\tifSkill\tapplyLine\tlabel\n")
	for i, v in ipairs(varList) do
		if v.var then
			local list = {}
			for _, e in ipairs(v.list or {}) do
				list[#list + 1] = typed(e.val)
			end
			local preds = {}
			for _, k in ipairs(predicateKeys) do
				if v[k] ~= nil then
					preds[#preds + 1] = k .. "=" .. joined(v[k]):gsub("\31", "|")
				end
			end
			local line = v.apply and debug.getinfo(v.apply, "S").linedefined or 0
			out:write(table.concat({ i, clean(v.var), clean(v.type), typed(v.defaultState), v.defaultIndex or "",
				table.concat(list, "\31"), table.concat(preds, ";"), joined(v.ifSkill), line, clean(v.label or "") }, "\t"), "\n")
		end
	end
	out:close()
	return
end

local outDir = arg[2]
for i = 3, #arg do
	local path = arg[i]
	local name = path:match("([^/]+)%.xml$")
	local fh = assert(io.open(path, "r"))
	local xmlText = fh:read("*a")
	fh:close()
	local lines = {}
	local ok, err = pcall(function()
		if GlobalCache and GlobalCache.cachedData then
			wipeGlobalCache()
		end
		loadBuildFromXML(xmlText, name)
		runCallback("OnFrame")
		for var, control in pairs(build.configTab.varControls) do
			local shown = control.shown
			if type(shown) == "function" then
				shown = shown(control)
			end
			if shown then
				lines[#lines + 1] = var
			end
		end
	end)
	table.sort(lines)
	local tmp = outDir .. "/" .. name .. ".tmp"
	local out = assert(io.open(tmp, "w"))
	if ok then
		out:write(table.concat(lines, "\n"), #lines > 0 and "\n" or "")
	else
		out:write("#error\t", clean(err), "\n")
	end
	out:close()
	assert(os.rename(tmp, outDir .. "/" .. name .. ".txt"))
end
