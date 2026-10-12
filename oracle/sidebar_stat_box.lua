local paths = { unpack(arg) }
dofile("HeadlessWrapper.lua")

local function str(v)
	if v == nil then return "null" end
	if type(v) == "boolean" then return tostring(v) end
	if type(v) == "number" then
		if v == math.huge then return '"inf"' end
		if v ~= v then return '"nan"' end
		return string.format("%.17g", v)
	end
	return '"' .. tostring(v):gsub('\\', '\\\\'):gsub('"', '\\"'):gsub('\n', '\\n') .. '"'
end

local function strList(list)
	local out = {}
	for _, v in ipairs(list or {}) do
		out[#out + 1] = str(v)
	end
	return "[" .. table.concat(out, ",") .. "]"
end

local function rows(list)
	local out = {}
	for _, row in ipairs(list) do
		local cells = {}
		for i = 1, #row do
			cells[#cells + 1] = str(row[i])
		end
		out[#out + 1] = '{"height":' .. str(row.height) .. ',"center":' .. str(row.align == "CENTER_X") .. ',"cells":[' .. table.concat(cells, ",") .. "]}"
	end
	return "[" .. table.concat(out, ",") .. "]"
end

local function skillDPS(list)
	local out = {}
	for _, s in ipairs(list or {}) do
		out[#out + 1] = "{" .. table.concat({
			'"name":' .. str(s.name), '"dps":' .. str(s.dps), '"count":' .. str(s.count),
			'"trigger":' .. str(s.trigger), '"skillPart":' .. str(s.skillPart), '"source":' .. str(s.source),
		}, ",") .. "}"
	end
	return "[" .. table.concat(out, ",") .. "]"
end

for i = 1, #paths do
	local fh = assert(io.open(paths[i], "r"))
	local xmlText = fh:read("*a")
	fh:close()
	wipeGlobalCache()
	loadBuildFromXML(xmlText, paths[i])
	runCallback("OnFrame")

	local env = build.calcsTab.mainEnv
	local output = build.calcsTab.mainOutput
	local warnings = {}
	for _, v in pairs(build.controls.warnings.lines) do
		warnings[#warnings + 1] = v
	end
	local fields = {
		'"file":' .. str(paths[i]:match("[^/]+$")),
		'"minion":' .. str(env.minion ~= nil),
		'"statBox":' .. rows(build.controls.statBox.list),
		'"warnings":' .. strList(warnings),
		'"skillDPS":' .. skillDPS(output.SkillDPS),
		'"energyShieldProtectsMana":' .. str(output.EnergyShieldProtectsMana == true),
		'"lifeCostWarningList":' .. strList(output.LifeCostWarningList),
		'"manaCostWarningList":' .. strList(output.ManaCostWarningList),
		'"rageCostWarningList":' .. strList(output.RageCostWarningList),
		'"esCostWarningList":' .. strList(output.ESCostWarningList),
		'"lifePercentCostWarningList":' .. strList(output.LifePercentCostPercentCostWarningList),
		'"manaPercentCostWarningList":' .. strList(output.ManaPercentCostPercentCostWarningList),
		'"costWarning":' .. strList((function()
			local l = {}
			for _, k in ipairs({ "LifeCost", "ManaCost", "RageCost", "ESCost" }) do
				if output[k .. "Warning"] then l[#l + 1] = k end
			end
			return l
		end)()),
		'"jewelLimitWarning":' .. strList(env.itemWarnings.jewelLimitWarning),
		'"socketLimitWarning":' .. strList(env.itemWarnings.socketLimitWarning),
		'"missingAnointWarning":' .. strList(env.itemWarnings.missingAnointWarning),
	}
	print("{" .. table.concat(fields, ",") .. "}")
end
