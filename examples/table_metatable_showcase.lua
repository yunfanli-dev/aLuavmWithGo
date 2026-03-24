local store = {}
local table_key_a = {}
local table_key_b = {}

local function make_key(label)
  return function()
    return label
  end
end

local fn_key_a = make_key("a")
local fn_key_b = make_key("b")

store[table_key_a] = "table-a"
store[table_key_b] = "table-b"
rawset(store, fn_key_a, "fn-a")
rawset(store, fn_key_b, "fn-b")

local fallback = {}
setmetatable(fallback, {
  __index = function(_, key)
    if key == "missing" then
      return "from-chain"
    end
  end,
})

local writes = {}
local target = {}
setmetatable(target, {
  __index = fallback,
  __newindex = writes,
})

target.answer = 42

print("identity", store[table_key_a], store[table_key_b], rawget(store, fn_key_a), rawget(store, fn_key_b))
print("chain", target.missing, writes.answer)
