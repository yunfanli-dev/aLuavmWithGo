local iterations = 1000
local sum = 0

local start_ms = clock_ms()

for i = 1, iterations do
  sum = sum + i
end

local finish_ms = clock_ms()
local elapsed_ms = finish_ms - start_ms

print("iterations", iterations)
print("sum", sum)
print("elapsed_ms", elapsed_ms)
