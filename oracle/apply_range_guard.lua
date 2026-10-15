
local guard = {}

local minValues = 10

function guard.install()
	local applyRange = itemLib.applyRange
	local prefixMatches = {}
	local function anyKeyStartsWith(prefix)
		local hit = prefixMatches[prefix]
		if hit == nil then
			hit = false
			for key in pairs(data.modScalability) do
				if key:sub(1, #prefix) == prefix then
					hit = true
					break
				end
			end
			prefixMatches[prefix] = hit
		end
		return hit
	end
	itemLib.applyRange = function(line, range, valueScalar, baseValueScalar)
		if not line:find("-", 1, true) and (not valueScalar or valueScalar == 1) and (not baseValueScalar or baseValueScalar == 1) then
			local stripped, n = line:gsub("%d+%.?%d*", "#")
			if n >= minValues and not anyKeyStartsWith((stripped:match("^([^#]*)#"):gsub("%+$", ""))) then
				return line
			end
		end
		return applyRange(line, range, valueScalar, baseValueScalar)
	end
end

return guard
