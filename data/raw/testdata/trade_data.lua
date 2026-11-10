-- usage (from PoB's src/): luajit trade_data.lua
-- Prints Data/TradeSiteStats.lua and Data/StatDescriptions/stat_descriptions.lua's described entries as JSON.
local json = require("dkjson")
local descs = {}
for _, entry in ipairs(dofile("Data/StatDescriptions/stat_descriptions.lua")) do
	if entry[1] then
		local forms = {}
		for _, form in ipairs(entry[1]) do
			local flags = {}
			for _, flag in ipairs(form) do
				table.insert(flags, { k = flag.k, v = flag.v })
			end
			table.insert(forms, { text = form.text, limit = form.limit, flags = flags })
		end
		table.insert(descs, { stats = entry.stats, forms = forms })
	end
end
io.write(json.encode({ tradeStats = dofile("Data/TradeSiteStats.lua"), statDescriptions = descs }))
