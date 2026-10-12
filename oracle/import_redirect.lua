local outPath = arg[1]
dofile("HeadlessWrapper.lua")
local json = require("dkjson")
local cases = {
  "https://www.youtube.com/redirect?event=video_description&redir_token=abc&q=https%3A%2F%2Fpobb.in%2FAbC-12_x&v=xyz",
  "  ?https://www.google.com/url?sa=t&url=x&q=https%3A%2F%2Fpastebin.com%2FqwErTy12&usg=AOv  ",
  "https://www.google.com/url?q=https://poe.ninja/pob/1a2b%2B3",
  "https://www.youtube.com/redirect?event=x",
  "https://pobb.in/plain",
}
local out = {}
for i, buf in ipairs(cases) do
  local urlText = buf:gsub("^[%s?]+", ""):gsub("[%s?]+$", "")
  if urlText:match("youtube%.com/redirect%?") or urlText:match("google%.com/url%?") then
    local nested_url = urlText:gsub(".*[?&]q=([^&]+).*", "%1")
    urlText = UrlDecode(nested_url)
  end
  out[i] = { input = buf, url = urlText }
end
local f = assert(io.open(outPath, "w"))
f:write(json.encode(out))
f:close()
