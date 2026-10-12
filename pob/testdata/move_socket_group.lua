local src = assert(arg[1], "usage: luajit move_socket_group.lua <PathOfBuilding/src>")

local classes = {}
function newClass(name, ...)
	local cls = {}
	cls.__index = cls
	classes[name] = cls
	return cls
end
function NewImageHandle()
	return {}
end
function IsKeyDown()
	return false
end

dofile(src .. "/Classes/ListControl.lua")
dofile(src .. "/Classes/SkillListControl.lua")
local ListClass = classes["ListControl"]
local SkillListClass = classes["SkillListControl"]
setmetatable(SkillListClass, { __index = ListClass })

local no = function()
	return false
end
local yes = function()
	return true
end

for n = 1, 5 do
	for from = 1, n do
		for slot = 1, n + 1 do
			for main = 0, n do
				for skillNumber = 1, n do
					local list = {}
					for i = 1, n do
						list[i] = i
					end
					local build = { mainSocketGroup = main, calcsTab = { input = { skill_number = skillNumber } } }
					local self = setmetatable({
						list = list,
						selIndex = from,
						selValue = list[from],
						selDragActive = true,
						selDragIndex = slot,
						controls = { scrollBarV = { IsScrollDownKey = no, IsScrollUpKey = no } },
						IsShown = yes,
						IsEnabled = yes,
						skillsTab = { build = build, AddUndoState = function() end },
					}, SkillListClass)
					ListClass.OnKeyUp(self, "LEFTBUTTON")
					print(table.concat({ n, from, slot, main, skillNumber, table.concat(list, ","), build.mainSocketGroup, build.calcsTab.input.skill_number }, "\t"))
				end
			end
		end
	end
end
