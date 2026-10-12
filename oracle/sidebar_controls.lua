local paths = { unpack(arg) }
dofile("HeadlessWrapper.lua")

local function str(v)
	if v == nil then return "null" end
	if type(v) == "boolean" then return tostring(v) end
	if type(v) == "number" then return tostring(v) end
	return '"' .. tostring(v):gsub('\\', '\\\\'):gsub('"', '\\"'):gsub('\n', '\\n') .. '"'
end

local function labels(list)
	local out = {}
	for _, v in ipairs(list or {}) do
		out[#out + 1] = str(type(v) == "table" and (v.label or v.val) or v)
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

	local c = build.controls
	local used, ascUsed, secondaryAscUsed, sockets = build.spec:CountAllocNodes()
	local fields = {
		'"file":' .. str(paths[i]:match("[^/]+$")),
		'"used":' .. used, '"ascUsed":' .. ascUsed, '"secondaryAscUsed":' .. secondaryAscUsed, '"sockets":' .. sockets,
		'"extraPoints":' .. str(build.calcsTab.mainOutput.ExtraPoints or 0),
		'"pointStr":' .. str(c.pointDisplay.str),
		'"pointReq":' .. str(c.pointDisplay.req),
		'"skills":' .. labels(c.mainSkill.list),
		'"skillShown":' .. str(c.mainSkill.shown ~= false),
		'"skillEnabled":' .. str(c.mainSkill.enabled == true),
		'"mainActiveSkill":' .. str(c.mainSkill.selIndex),
		'"partShown":' .. str(c.mainSkillPart.shown == true),
		'"parts":' .. labels(c.mainSkillPart.shown and c.mainSkillPart.list or {}),
		'"skillPart":' .. str(c.mainSkillPart.shown and c.mainSkillPart.selIndex or 0),
		'"stageShown":' .. str(c.mainSkillStageCount.shown == true),
		'"stageCount":' .. str(c.mainSkillStageCount.shown and c.mainSkillStageCount.buf or ""),
		'"mineShown":' .. str(c.mainSkillMineCount.shown == true),
		'"mineCount":' .. str(c.mainSkillMineCount.shown and c.mainSkillMineCount.buf or ""),
		'"minionShown":' .. str(c.mainSkillMinion.shown == true),
		'"minionEnabled":' .. str(c.mainSkillMinion.shown and c.mainSkillMinion.enabled == true),
		'"minionLibraryShown":' .. str(c.mainSkillMinionLibrary.shown == true),
		'"minions":' .. labels(c.mainSkillMinion.shown and c.mainSkillMinion.list or {}),
		'"minion":' .. str(c.mainSkillMinion.shown and c.mainSkillMinion.selIndex or 0),
		'"minionSkillShown":' .. str(c.mainSkillMinionSkill.shown == true),
		'"minionSkillEnabled":' .. str(c.mainSkillMinionSkill.shown and c.mainSkillMinionSkill.enabled == true),
		'"minionSkills":' .. labels(c.mainSkillMinionSkill.shown and c.mainSkillMinionSkill.list or {}),
		'"minionSkill":' .. str(c.mainSkillMinionSkill.shown and c.mainSkillMinionSkill.selIndex or 0),
	}
	print("{" .. table.concat(fields, ",") .. "}")
end
