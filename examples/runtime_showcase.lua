local values = { 3, 1, 2 }
table.sort(values)

local object = { base = 40 }

function object:add(x)
  return self.base + x
end

local mt = {
  __index = function(_, key)
    if key == "fallback" then
      return "from_meta"
    end
  end,
  __tostring = function(table_value)
    return "object:" .. table_value.base
  end,
}

setmetatable(object, mt)

local joined = table.concat(values, ",")
local reversed = string.reverse("stressed")
local upper = string.upper("lua")
local rounded = math.floor(math.sqrt(81))
local random_pick

math.randomseed(5)
random_pick = math.random(10)

print("sort", joined)
print("method", object:add(2))
print("meta_index", object.fallback)
print("meta_tostring", tostring(object))
print("string", reversed, upper)
print("math", rounded, random_pick)
