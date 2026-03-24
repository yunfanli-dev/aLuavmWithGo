local function step(state, control)
  local next_index = control + 1
  if next_index > state.limit then
    return nil
  end

  return next_index, state.values[next_index]
end

local total = 0
for index, value in step, { limit = 3, values = { 10, 20, 30 } }, 0 do
  total = total + index + value
end

local parts = {}
for part in string.gmatch("aba", "a") do
  table.insert(parts, part)
end

print("custom_iter", total)
print("gmatch_iter", table.concat(parts, "|"))
