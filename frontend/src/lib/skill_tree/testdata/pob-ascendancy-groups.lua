local v = arg[1]
local t = dofile((os.getenv("POB_DIR") or "PathOfBuilding").."/src/TreeData/"..v.."/tree.lua")
local rows = {}
for gid, g in pairs(t.groups) do
  local asc
  for _, nid in ipairs(g.nodes or {}) do
    local n = t.nodes[nid] or t.nodes[tonumber(nid)]
    if n and n.ascendancyName then asc = n.ascendancyName end
  end
  if asc then rows[#rows+1] = { gid = tonumber(gid), s = string.format('"%d":{"ascendancy":"%s","x":%s,"y":%s}', gid, asc, tostring(g.x), tostring(g.y)) } end
end
table.sort(rows, function(a, b) return a.gid < b.gid end)
local out = {}
for _, r in ipairs(rows) do out[#out+1] = r.s end
print("{" .. table.concat(out, ",\n") .. "}")
