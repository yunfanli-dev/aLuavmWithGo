local obj = {
  value = 40
}

function obj:add(step)
  return self.value + step
end

local result = obj:add(2)
print("result:", result)

return result
