local compat = {}

function compat.construct(className, ...)
	if debug.getinfo(new, "u").isvararg then
		return new(className, ...)
	end
	local obj = new(className)
	return obj[className](obj, ...)
end

function compat.loadDataModule(name, ...)
	local ret = LoadModule(name, ...)
	if type(ret) == "function" then
		return ret(...)
	end
	return ret
end

local varargLine = "\nlocal ([%w_, ]+) = %.%.%.%s*\n"

function compat.adaptOverlay(fileName, text)
	local names = ("\n" .. text):match(varargLine)
	if not names then
		return text
	end
	local f = io.open(fileName, "r")
	if not f then
		return text
	end
	local pinText = "\n" .. f:read("*a")
	f:close()
	if pinText:match(varargLine) then
		return text
	end
	local body
	if pinText:match("\nreturn function%(") then
		body = ("\n" .. text):gsub(varargLine, "\nreturn function(" .. names .. ")\n", 1)
		return body:sub(2) .. "\nend\n"
	end
	body = ("\n" .. text):gsub(varargLine, "\nlocal " .. names .. " = {}\n", 1)
	return body:sub(2) .. "\nreturn " .. names .. "\n"
end

function compat.loadBuildDisplayStats()
	local ret, minion, extra = LoadModule("Modules/BuildDisplayStats")
	if ret.displayStats then
		return ret.displayStats, ret.minionDisplayStats, ret.extraSaveStats
	end
	return ret, minion, extra
end

return compat
