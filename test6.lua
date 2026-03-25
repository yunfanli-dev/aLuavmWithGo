local M = {}

function M.add(a, b)

  local iterations = 10
  local sum = 0
  
  local start_ms = clock_ms()
  local nt = {}
  for i = 1, iterations do
      nt[i] = i
      sum = sum + i
      print("iteration", i, "sum", nt[i])
  end
  
  local finish_ms = clock_ms()
  local elapsed_ms = finish_ms - start_ms
  print("start_ms", start_ms)
  print("finish_ms", finish_ms)
  print("iterations", iterations)
  print("sum", sum)
  print("elapsed_ms", elapsed_ms)
  print(math.max(-111, 1))
  return a + b
end

return M
