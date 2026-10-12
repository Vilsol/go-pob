local src = assert(arg[1], "usage: luajit delete_spec.lua <PathOfBuilding/src>")

local classes = {}
function newClass(name, ...)
	local cls = {}
	cls.__index = cls
	classes[name] = cls
	return cls
end
function isValueInArray(tbl, val)
	for i, v in ipairs(tbl) do
		if val == v then
			return i
		end
	end
end
main = {
	OpenConfirmPopup = function(_, _, _, _, onConfirm)
		onConfirm()
	end,
}

dofile(src .. "/Classes/PassiveSpecListControl.lua")
local SpecListClass = classes["PassiveSpecListControl"]

for n = 1, 5 do
	for active = 1, n do
		for deleted = 1, n do
			local list = {}
			for i = 1, n do
				list[i] = { title = tostring(i) }
			end
			local treeTab = { activeSpec = active, specList = list }
			treeTab.build = {
				spec = list[active],
				SyncLoadouts = function() end,
				itemsTab = { controls = { specSelect = { SetList = function() end } } },
			}
			function treeTab:SetActiveSpec(specId)
				self.activeSpec = math.min(specId, #self.specList)
				self.build.spec = self.specList[self.activeSpec]
			end
			local self = setmetatable({ list = list, treeTab = treeTab }, SpecListClass)
			self:OnSelDelete(deleted, list[deleted])
			local order = {}
			for i, spec in ipairs(list) do
				order[i] = spec.title
			end
			print(table.concat({ n, active, deleted, table.concat(order, ","), treeTab.activeSpec }, "\t"))
		end
	end
end
