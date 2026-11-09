-- usage (from PoB-PoE2's src/): luajit poe2_mod_sections.lua <out dir> <build.xml>...
-- Records, for every distinct CalcSections.lua modName entry, the mod table CalcBreakdownControl:AddModSection builds
-- for it over the Calcs tab calculation's player: each row's drawn cells and its source tooltip's lines. Writes
-- <out dir>/<build>.json, keyed by modName list|modType|cfg|modSource|enemy|actor; multi is a table modName, and an
-- entry with no rows holds none.
local selfPath, outDir = arg[0], arg[1]
local paths = { select(2, unpack(arg)) }
assert(outDir and #paths > 0, "usage: poe2_mod_sections.lua <out dir> <build.xml>...")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")
local sections = LoadModule("Modules/CalcSections")

local entries = {}
for _, section in ipairs(sections) do
	for _, sub in ipairs(section[5]) do
		for _, row in ipairs(sub.data) do
			for _, col in ipairs(row) do
				if type(col) == "table" then
					for _, entry in ipairs(col) do
						if type(entry) == "table" and entry.modName then
							entries[#entries + 1] = entry
						end
					end
				end
			end
		end
	end
end
assert(#entries > 100, "REFUSING: only " .. #entries .. " modName entries")

local function str(s)
	return s and tostring(s) or ""
end

for i = 1, #paths do
	local fh = assert(io.open(paths[i], "r"))
	local xmlText = fh:read("*a")
	fh:close()
	wipeGlobalCache()
	loadBuildFromXML(xmlText, paths[i])
	runCallback("OnFrame")
	local control = build.calcsTab.controls.breakdown
	control.envName = "calcsEnv"
	local out = {}
	for _, entry in ipairs(entries) do
		local modName = type(entry.modName) == "table" and entry.modName or { entry.modName }
		local key = table.concat(modName, ",") .. "|" .. str(entry.modType) .. "|" .. str(entry.cfg) .. "|" .. str(entry.modSource)
			.. "|" .. tostring(entry.enemy or false) .. "|" .. str(entry.actor)
		if out[key] == nil then
			local multi = type(entry.modName) == "table"
			control.sectionList = {}
			local ok, err = pcall(control.AddModSection, control, entry)
			local section = control.sectionList[1]
			if not ok then
				out[key] = { multi = multi, error = tostring(err) }
			else
				out[key] = { multi = multi }
			end
			if ok and section then
				local rows = {}
				for _, row in ipairs(section.rowList) do
					local totals
					if row.sourceTooltip then
						totals = {}
						row.sourceTooltip({ AddLine = function(_, _, text) totals[#totals + 1] = tostring(text) end })
					end
					rows[#rows + 1] = { value = str(row.displayValue), name = str(row.name), flags = str(row.flags), tags = str(row.tags),
						source = str(row.source), sourceName = str(row.sourceName), totals = totals }
				end
				out[key].rows = rows
			end
		end
	end
	local name = paths[i]:match("([^/]+)%.xml$")
	local o = assert(io.open(outDir .. "/" .. name .. ".json", "w"))
	o:write(json.encode(stable(out), { indent = true }))
	o:write("\n")
	o:close()
end
