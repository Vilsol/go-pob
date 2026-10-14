local path, pad, names = arg[1], tonumber(arg[2]), arg[3]
local junk = {}
for i = 1, pad do junk[i] = { tostring(i) .. "x" } end
dofile("HeadlessWrapper.lua")
local fh = assert(io.open(path, "r")); local x = fh:read("*a"); fh:close()
loadBuildFromXML(x, path)
runCallback("OnFrame")
local env = build.calcsTab.mainEnv
for spec in names:gmatch("[^,]+") do
  local actor, name = spec:match("^(%u+):(.+)$")
  local db = actor == "ENEMY" and env.enemyDB or env.player.modDB
  local out = {}
  for _, m in ipairs(db.mods[name] or {}) do out[#out+1] = tostring(m.source) .. "/" .. tostring(m.value) end
  print("ORDER\t" .. spec .. "\t" .. table.concat(out, " | "))
end
