local obj = {
  value = 10
}

function obj:add(step)
  self.value = self.value + step
  return self.value
end

local function makeCounter(start)
  local current = start
  return function(step)
    current = current + step
    return current
  end
end

function sum3(a, b, c)
  return a + b + c
end

function explode()
  error("boom")
end

local counter = makeCounter(0)
local a = counter(1)
local b = counter(2)

local ok1, x, y = pcall(function()
  return 7, 8
end)

local ok2, err = pcall(explode)

local values = { 1, 2, 3 }
local total = 0
for i, v in ipairs(values) do
  total = total + v
end

local meta = {
  __index = function(_, key)
    return "missing:" .. key
  end,
  __tostring = function(value)
    return "obj(" .. value.value .. ")"
  end
}

setmetatable(obj, meta)

local s1 = obj:add(5)
local s2 = tostring(obj)
local fallback = obj.name

local mixed = sum3(0, counter(3), obj:add(1))

print("counter:", a, b)
print("pcall success:", ok1, x, y)
print("pcall fail:", ok2, err)
print("table total:", total)
print("metatable:", s2, fallback)
print("mixed:", mixed)

return a, b, ok1, x, y, ok2, err, total, s1, s2, fallback, mixed
