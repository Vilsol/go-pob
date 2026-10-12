local paths = { unpack(arg) }
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local sections = LoadModule("Modules/CalcSections")

local entries = { }
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

local function strip(s)
	return s and tostring(s) or ""
end

for i = 1, #paths do
	local fh = assert(io.open(paths[i], "r"))
	local xmlText = fh:read("*a")
	fh:close()
	wipeGlobalCache()
	loadBuildFromXML(xmlText, paths[i])
	runCallback("OnFrame")
	local breakdown = build.calcsTab.controls.breakdown
	local seen = { }
	for _, entry in ipairs(entries) do
		local modName = type(entry.modName) == "table" and entry.modName or { entry.modName }
		local key = table.concat(modName, ",") .. "|" .. strip(entry.modType) .. "|" .. strip(entry.cfg) .. "|" .. strip(entry.modSource) .. "|" .. tostring(entry.enemy or false) .. "|" .. strip(entry.actor)
		if not seen[key] then
			seen[key] = true
			breakdown.sectionList = { }
			breakdown:AddModSection(entry)
			local section = breakdown.sectionList[1]
			if section then
				local rows = { }
				for _, row in ipairs(section.rowList) do
					local totals
					if row.sourceTooltip then
						totals = { }
						row.sourceTooltip({ AddLine = function(_, _, text) totals[#totals + 1] = text end })
					end
					rows[#rows + 1] = { value = strip(row.displayValue), name = strip(row.name), flags = strip(row.flags), tags = strip(row.tags),
						source = strip(row.source), sourceName = strip(row.sourceName), totals = totals }
				end
				print(json.encode({ file = paths[i]:match("[^/]+$"), modName = modName, multi = type(entry.modName) == "table",
					modType = entry.modType, cfg = entry.cfg, modSource = entry.modSource, enemy = entry.enemy or false, actor = entry.actor, rows = rows }))
			end
		end
	end
end
