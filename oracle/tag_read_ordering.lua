local xmlPath = arg[1]
assert(xmlPath, "usage: tag_read_ordering.lua <build.xml> [statToTraceback]")
local traceStat = arg[2]

dofile("HeadlessWrapper.lua")

local first = {}
local order = {}
local traces = {}

local patched = 0
for _, class in pairs(common.classes) do
	if rawget(class, "GetStat") then
		local orig = rawget(class, "GetStat")
		rawset(class, "GetStat", function(self, stat, cfg)
			local actor = self.actor
			if actor then
				local seen = first[actor]
				if not seen then
					seen = {}
					first[actor] = seen
					order[#order + 1] = actor
				end
				if seen[stat] == nil then
					local out = actor.output
					local v = out and out[stat]
					if v == nil then
						seen[stat] = "absent"
						if stat == traceStat then
							traces[#traces + 1] = debug.traceback("", 2)
						end
					else
						seen[stat] = "present:" .. tostring(v)
					end
				end
			end
			return orig(self, stat, cfg)
		end)
		patched = patched + 1
	end
end
assert(patched > 0, "REFUSING: no class in common.classes carries GetStat — the hook installed " ..
	"nothing and an empty report would read exactly like a build with no early reads")
print(string.format("HOOK installed on %d class(es)", patched))

local fh = assert(io.open(xmlPath, "r"))
local xmlText = fh:read("*a")
fh:close()

wipeGlobalCache()
loadBuildFromXML(xmlText, xmlPath)
runCallback("OnFrame")

local env = build.calcsTab.mainEnv

local function label(actor)
	if actor == env.player then return "player" end
	if actor == env.enemy then return "enemy" end
	if actor == env.minion then return "minion" end
	if env.player and actor == env.player.enemy then return "player.enemy" end
	if env.minion and actor == env.minion.enemy then return "minion.enemy" end
	return "other"
end

print(string.format("BUILD %s", xmlPath))
print("")
print("=== FIRST GetStat read per actor+stat, and whether output carried the key yet ===")
local anyAbsent = 0
for _, actor in ipairs(order) do
	local seen = first[actor]
	local stats = {}
	for stat in pairs(seen) do stats[#stats + 1] = stat end
	table.sort(stats)
	for _, stat in ipairs(stats) do
		local state = seen[stat]
		local final = actor.output and actor.output[stat]
		if state == "absent" then
			anyAbsent = anyAbsent + 1
			print(string.format("ABSENT   %-14s %-32s final=%s", label(actor), stat, tostring(final)))
		else
			local read = state:sub(9)
			if tostring(final) ~= read then
				print(string.format("STALE    %-14s %-32s read=%s final=%s", label(actor), stat, read, tostring(final)))
			end
		end
	end
end
print(string.format("ABSENT reads: %d", anyAbsent))

if traceStat then
	print("")
	print(string.format("=== tracebacks for ABSENT first reads of %q (%d) ===", traceStat, #traces))
	for _, tb in ipairs(traces) do
		print(tb)
	end
end
