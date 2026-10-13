local selfPath = arg[0]
local mode, target = arg[1], arg[2]
assert(mode and target, "usage: config_flip.lua options <out.json> | dump <jobs.tsv>")

local stable = dofile((selfPath:match("^(.*)[/\\]") or ".") .. "/json_stable.lua")

dofile("HeadlessWrapper.lua")

local json = require("dkjson")

local predicateKeys = {
	"ifNode", "ifOption", "ifCond", "ifMinionCond", "ifEnemyCond", "ifCondTrue", "ifMult", "ifEnemyMult",
	"ifStat", "ifEnemyStat", "ifTagType", "ifFlag", "ifMod", "ifSkill", "ifSkillFlag", "ifSkillData",
	"includeTransfigured", "implyCond", "implyCondList", "implyMinionCond", "implyEnemyCond",
}

local function writeJSON(path, value)
	local tmp = path .. ".tmp"
	local f = assert(io.open(tmp, "w"))
	f:write(json.encode(stable(value), { indent = true }))
	f:close()
	assert(os.rename(tmp, path))
end

if mode == "options" then
	local varList = LoadModule("Modules/ConfigOptions")
	local out = {}
	local section
	for _, varData in ipairs(varList) do
		if varData.section then
			section = varData.section
		elseif varData.var then
			local rec = { var = varData.var, type = varData.type, section = section, label = varData.label }
			rec.defaultState = varData.defaultState
			rec.defaultIndex = varData.defaultIndex
			rec.defaultPlaceholderState = varData.defaultPlaceholderState
			for _, k in ipairs(predicateKeys) do
				rec[k] = varData[k]
			end
			if varData.list then
				rec.list = {}
				for j, entry in ipairs(varData.list) do
					rec.list[j] = { val = entry.val, label = entry.label }
				end
			end
			rec.hasApply = varData.apply ~= nil
			if varData.apply then
				rec.applyLine = debug.getinfo(varData.apply, "S").linedefined
				rec.applyLastLine = debug.getinfo(varData.apply, "S").lastlinedefined
			end
			out[#out + 1] = rec
		end
	end
	if #out < 100 then
		io.stderr:write("config_flip: REFUSING -- ConfigOptions yielded " .. #out .. " options\n")
		os.exit(1)
	end
	writeJSON(target, out)
	io.stderr:write(string.format("config_flip: %d options -> %s\n", #out, target))
	os.exit(0)
end

assert(mode == "dump", "unknown mode " .. tostring(mode))

local staticFromItem, staticFromItemCount = {}, 0
for id, granted in pairs(data.skills) do
	if granted.fromItem then
		staticFromItem[id] = true
		staticFromItemCount = staticFromItemCount + 1
	end
end
if staticFromItemCount == 0 then
	io.stderr:write("config_flip: REFUSING -- no static fromItem rows; the snapshot ran before the data loaded\n")
	os.exit(1)
end
local function resetFromItem()
	for id, granted in pairs(data.skills) do
		if staticFromItem[id] then
			granted.fromItem = true
		else
			granted.fromItem = nil
		end
	end
end

local keystoneSource, keystoneSourceCount, snapshottedTrees = {}, 0, {}
local function snapshotKeystoneSources(tree)
	if not tree or snapshottedTrees[tree] then
		return
	end
	snapshottedTrees[tree] = true
	local function record(mod)
		if keystoneSource[mod] == nil then
			keystoneSource[mod] = mod.source or false
			keystoneSourceCount = keystoneSourceCount + 1
		end
	end
	for _, node in pairs(tree.keystoneMap or {}) do
		for _, mod in ipairs(node.modList or {}) do
			record(mod)
			if type(mod.value) == "table" and mod.value.mod then
				record(mod.value.mod)
			end
		end
	end
end
for _, tree in pairs(main.tree) do
	snapshotKeystoneSources(tree)
end
local loadTree = main.LoadTree
function main:LoadTree(treeVersion)
	local tree = loadTree(self, treeVersion)
	snapshotKeystoneSources(tree)
	return tree
end
if keystoneSourceCount == 0 then
	io.stderr:write("config_flip: REFUSING -- no keystone mods snapshotted; the tree was not built yet\n")
	os.exit(1)
end
local function resetKeystoneSources()
	for mod, source in pairs(keystoneSource) do
		mod.source = source or nil
	end
end

local function scalars(t)
	local out = {}
	if type(t) ~= "table" then
		return out
	end
	for k, v in pairs(t) do
		local vt = type(v)
		if type(k) == "string" and (vt == "number" or vt == "string" or vt == "boolean") then
			out[k] = v
		end
	end
	return out
end

local jobs = {}
for line in io.lines(target) do
	local xmlPath, outPath, vis = line:match("^([^\t]+)\t([^\t]+)\t([01])$")
	if xmlPath then
		jobs[#jobs + 1] = { xml = xmlPath, out = outPath, vis = vis == "1" }
	end
end

local done, skipped = 0, 0
for _, job in ipairs(jobs) do
	local existing = io.open(job.out, "r")
	if existing then
		existing:close()
		skipped = skipped + 1
	else
		local fh = assert(io.open(job.xml, "r"))
		local xmlText = fh:read("*a")
		fh:close()
		local entry = {}
		local ok, err = pcall(function()
			wipeGlobalCache()
			resetFromItem()
			resetKeystoneSources()
			loadBuildFromXML(xmlText, job.xml:match("[^/]+$"))
			runCallback("OnFrame")

			local mainEnv = build.calcsTab.mainEnv
			entry.output = scalars(mainEnv.player.output)
			entry.outputMainHand = mainEnv.player.output.MainHand and scalars(mainEnv.player.output.MainHand) or nil
			entry.outputOffHand = mainEnv.player.output.OffHand and scalars(mainEnv.player.output.OffHand) or nil
			entry.enemyOutput = mainEnv.enemy and scalars(mainEnv.enemy.output) or nil
			local minion = mainEnv.minion
			entry.minionOutput = minion and scalars(minion.output) or nil
			entry.minionOutputMainHand = minion and minion.output and minion.output.MainHand
				and scalars(minion.output.MainHand) or nil
			entry.minionOutputOffHand = minion and minion.output and minion.output.OffHand
				and scalars(minion.output.OffHand) or nil

			if job.vis then
				local configTab = build.configTab
				local input = configTab.configSets[configTab.activeConfigSetId].input
				local visible, nonDefault = {}, {}
				for var, control in pairs(configTab.varControls) do
					local cur = input[var]
					if cur ~= nil and cur ~= configTab:GetDefaultState(var, type(cur)) then
						nonDefault[var] = cur
					end
					if type(control.shown) == "function" then
						local okShown, shown = pcall(control.shown)
						if okShown and shown then
							visible[#visible + 1] = var
						end
					elseif control.shown ~= false then
						visible[#visible + 1] = var
					end
				end
				table.sort(visible)
				entry.visible = visible
				entry.nonDefault = nonDefault
			end
		end)
		if not ok then
			entry.error = tostring(err)
		end
		writeJSON(job.out, entry)
		done = done + 1
	end
end
io.stderr:write(string.format("config_flip: dumped %d, skipped %d existing, of %d jobs\n", done, skipped, #jobs))
