local src = assert(io.open(arg[1] .. "/Modules/BuildSiteTools.lua")):read("*a")
local list = src:match("(buildSites%.websiteList = {.-\n})\n")
buildSites = {}
assert(loadstring(list))()
local links = {
  Maxroll = "https://maxroll.gg/poe/pob/abc123",
  PoBCodes = "https://pob.codes/b/abc-def",
  POBBin = "https://pobb.in/aBcD1234",
  PoeNinja = { "https://poe.ninja/pob/Zx9", "https://poe.ninja/poe1/pob/Zx9" },
  pastebin = "https://pastebin.com/Qw3rTy",
  pastebinProxy = "https://pastebinp.com/Qw3rTy",
  rentry = "https://rentry.co/myBuild",
  PoEDB = "https://poedb.tw/pob/xyz",
}
for _, site in ipairs(buildSites.websiteList) do
  local l = links[site.id]
  for _, link in ipairs(type(l) == "table" and l or { l }) do
    print(site.label .. "\t" .. link .. "\t" .. (link:gsub(site.regexURL, site.downloadURL)))
  end
end
