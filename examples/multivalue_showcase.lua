local function pack(...)
  return ...
end

local function pair()
  return "left", "right"
end

local ok, message = pcall(function()
  error("boom")
end)

local a, b, c = pack(1, pair())
local x, y = (pair()), pair()
local list = { "head", pair() }
local b1, b2 = string.byte("AZ", 1, 2)

print("assign", a, b, c)
print("paren", x, y)
print("pcall", ok, message)
print("table", table.concat(list, "|"))
print("byte", b1, b2)
