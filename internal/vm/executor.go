package vm

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/yunfanli-dev/aLuavmWithGo/internal/ir"
)

type executionResult struct {
	// returnValues 保存当前语句块或函数执行显式产生的返回值列表。
	// 它会在 `return`、函数调用收尾或部分控制流节点中向上层传递。
	returnValues []Value
	// breakLoop 标记当前执行结果是否代表一次 `break` 控制流跳出。
	// 外层循环节点会消费这个标记，普通语句块则需要继续向上传递。
	breakLoop bool
}

type envFrame struct {
	// env 保存当前活跃调用帧绑定的环境表。
	// 未命中的全局名读写会回落到这里。
	env *table
	// userFunction 指向当前调用帧对应的 Lua 函数对象。
	// 这样 `setfenv(level, env)` 改帧环境时，也能同步改回函数对象本身。
	userFunction *userFunction
	// nativeFunction 指向当前调用帧对应的 native 函数对象。
	// 这让 `setfenv(fn, env)` 和栈级 `setfenv(level, env)` 的观察结果保持一致。
	nativeFunction *nativeFunction
}

type executor struct {
	// state 指向当前执行绑定的运行时状态。
	// 它主要用于 `_G` 全局环境桥接和少量跨 helper 的运行时协作。
	state *State
	// envFrames 按调用栈顺序记录当前活跃执行帧。
	// 最后一个元素总是当前执行帧；`getfenv` / `setfenv` 会依赖这条栈做最小 level 解析。
	envFrames []envFrame
	// env 保存当前执行帧绑定的最小环境表。
	// 未命中的全局名读写会回落到这里，以便支持最小 `getfenv` / `setfenv`。
	env *table
	// scopes 维护当前执行栈可见的局部作用域链。
	// 最后一个元素总是当前最内层作用域，变量查找和赋值都依赖这条链路。
	scopes []map[string]*valueCell
	// varargs 保存当前函数调用收到的额外可变参数。
	// 只有在声明了 `...` 的函数作用域内，这些值才会被相关表达式读取。
	varargs []Value
	// stepLimit 记录本次执行允许使用的最大步数预算。
	// 非正数表示不启用限制，正数时会和 remainingSteps 一起工作。
	stepLimit int
	// remainingSteps 保存当前执行还剩多少步可以消耗。
	// 每执行一条受计步保护的语句或表达式路径时都会逐步递减。
	remainingSteps int
	// ctx 持有当前执行绑定的上下文对象。
	// 执行过程中会定期检查它，以支持超时和宿主主动取消。
	ctx context.Context
}

// executeProgram 执行当前 IR 子集程序，并返回脚本显式产生的返回值。
// 它会创建执行器、驱动语句求值，并把最终 `return` 结果整理成统一结构返回。
func executeProgram(ctx context.Context, state *State, program *ir.Program) (*executionResult, error) {
	return executeProgramWithEnv(ctx, state, program, state.globalEnv)
}

// executeProgramWithEnv 执行当前 IR 子集程序，并允许调用方指定线程环境表。
// 顶层执行默认使用 `_G`，`require` 等链路则可以把调用者线程环境继续传下去。
func executeProgramWithEnv(ctx context.Context, state *State, program *ir.Program, rootEnv *table) (*executionResult, error) {
	if program == nil {
		return nil, fmt.Errorf("execute nil IR program")
	}
	if state == nil {
		return nil, fmt.Errorf("execute nil VM state")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	exec := newExecutorWithEnv(ctx, state, rootEnv)

	for _, statement := range program.Statements {
		result, done, err := exec.executeStatement(statement)
		if err != nil {
			return nil, err
		}

		if done {
			if result.breakLoop {
				return nil, fmt.Errorf("break outside loop")
			}

			return result, nil
		}
	}

	return &executionResult{}, nil
}

// newExecutor 基于当前 State 和上下文创建一份执行器实例。
// 常规脚本执行和 `require` 触发的 preload loader 都会复用它。
func newExecutor(ctx context.Context, state *State) *executor {
	return newExecutorWithEnv(ctx, state, state.globalEnv)
}

// newExecutorWithEnv 基于当前 State、上下文和线程环境创建一份执行器实例。
// 这样嵌套 `require` 等链路就能继续沿用调用者线程环境，而不是总掉回根 `_G`。
func newExecutorWithEnv(ctx context.Context, state *State, rootEnv *table) *executor {
	if rootEnv == nil {
		rootEnv = state.globalEnv
	}

	return &executor{
		state:          state,
		envFrames:      []envFrame{{env: rootEnv}},
		env:            rootEnv,
		scopes:         []map[string]*valueCell{{}},
		stepLimit:      state.stepLimit,
		remainingSteps: state.stepLimit,
		ctx:            ctx,
	}
}

func (e *executor) executeStatement(statement ir.Statement) (*executionResult, bool, error) {
	if err := e.consumeStep(); err != nil {
		return nil, false, err
	}

	switch node := statement.(type) {
	case *ir.CallStatement:
		_, err := e.evaluateCallExpression(node.Call)
		return nil, false, err
	case *ir.AssignStatement:
		values, err := e.evaluateExpressionList(node.Values)
		if err != nil {
			return nil, false, err
		}

		for index, target := range node.Targets {
			value := NilValue()
			if index < len(values) {
				value = values[index]
			}

			if err := e.assignTarget(target, value); err != nil {
				return nil, false, err
			}
		}

		return nil, false, nil
	case *ir.LocalAssignStatement:
		values, err := e.evaluateExpressionList(node.Values)
		if err != nil {
			return nil, false, err
		}

		for index, name := range node.Names {
			value := NilValue()
			if index < len(values) {
				value = values[index]
			}

			e.defineLocal(name, value)
		}

		return nil, false, nil
	case *ir.DoStatement:
		return e.executeBlock(node.Body)
	case *ir.BreakStatement:
		return &executionResult{breakLoop: true}, true, nil
	case *ir.FunctionDeclarationStatement:
		e.assign(node.Name, e.makeUserFunctionValue(node.Name, node.Parameters, node.IsVararg, node.Body))

		return nil, false, nil
	case *ir.LocalFunctionDeclarationStatement:
		placeholder := NilValue()
		e.defineLocal(node.Name, placeholder)
		functionValue := e.makeUserFunctionValue(node.Name, node.Parameters, node.IsVararg, node.Body)
		e.assign(node.Name, functionValue)

		if userFn, ok := functionValue.Data.(*userFunction); ok {
			if cell, ok := e.currentScope()[node.Name]; ok {
				userFn.captured[node.Name] = cell
			}
		}

		return nil, false, nil
	case *ir.IfStatement:
		return e.executeIfStatement(node)
	case *ir.WhileStatement:
		return e.executeWhileStatement(node)
	case *ir.RepeatStatement:
		return e.executeRepeatStatement(node)
	case *ir.NumericForStatement:
		return e.executeNumericForStatement(node)
	case *ir.GenericForStatement:
		return e.executeGenericForStatement(node)
	case *ir.ReturnStatement:
		values, err := e.evaluateExpressionList(node.Values)
		if err != nil {
			return nil, false, err
		}

		return &executionResult{returnValues: values}, true, nil
	default:
		return nil, false, fmt.Errorf("execute unsupported IR statement %T", statement)
	}
}

func (e *executor) executeIfStatement(statement *ir.IfStatement) (*executionResult, bool, error) {
	for _, clause := range statement.Clauses {
		condition, err := e.evaluateExpression(clause.Condition)
		if err != nil {
			return nil, false, err
		}

		if isTruthy(condition) {
			return e.executeBlock(clause.Body)
		}
	}

	return e.executeBlock(statement.ElseBody)
}

func (e *executor) executeWhileStatement(statement *ir.WhileStatement) (*executionResult, bool, error) {
	for {
		condition, err := e.evaluateExpression(statement.Condition)
		if err != nil {
			return nil, false, err
		}

		if !isTruthy(condition) {
			return nil, false, nil
		}

		result, done, err := e.executeBlock(statement.Body)
		if err != nil {
			return nil, false, err
		}

		if done {
			if result.breakLoop {
				return nil, false, nil
			}

			return result, true, nil
		}
	}
}

// executeRepeatStatement 执行 `repeat ... until` 循环。
// 终止条件会在循环体作用域内求值，以保持与 Lua 作用域规则一致。
func (e *executor) executeRepeatStatement(statement *ir.RepeatStatement) (*executionResult, bool, error) {
	for {
		e.pushScope()
		for _, child := range statement.Body {
			result, done, err := e.executeStatement(child)
			if err != nil {
				e.popScope()
				return nil, false, err
			}

			if done {
				if result.breakLoop {
					e.popScope()
					return nil, false, nil
				}

				e.popScope()
				return result, true, nil
			}
		}

		condition, err := e.evaluateExpression(statement.Condition)
		e.popScope()
		if err != nil {
			return nil, false, err
		}

		if isTruthy(condition) {
			return nil, false, nil
		}
	}
}

// executeNumericForStatement 执行当前支持的数值 for 循环。
// 它会先求值起始值、终止值和步长，再按 Lua 风格边界规则推进循环变量。
func (e *executor) executeNumericForStatement(statement *ir.NumericForStatement) (*executionResult, bool, error) {
	startValue, err := e.evaluateExpression(statement.Start)
	if err != nil {
		return nil, false, err
	}

	limitValue, err := e.evaluateExpression(statement.Limit)
	if err != nil {
		return nil, false, err
	}

	stepValue, err := e.evaluateExpression(statement.Step)
	if err != nil {
		return nil, false, err
	}

	current, err := requireNumber(startValue, "for")
	if err != nil {
		return nil, false, err
	}

	limit, err := requireNumber(limitValue, "for")
	if err != nil {
		return nil, false, err
	}

	step, err := requireNumber(stepValue, "for")
	if err != nil {
		return nil, false, err
	}

	if step == 0 {
		return nil, false, fmt.Errorf("numeric for step cannot be zero")
	}

	e.pushScope()
	defer e.popScope()
	e.defineLocal(statement.Name, Value{Type: ValueTypeNumber, Data: current})

	for numericForContinues(current, limit, step) {
		e.assign(statement.Name, Value{Type: ValueTypeNumber, Data: current})
		result, done, err := e.executeBlock(statement.Body)
		if err != nil {
			return nil, false, err
		}

		if done {
			if result.breakLoop {
				return nil, false, nil
			}

			return result, true, nil
		}

		current += step
	}

	return nil, false, nil
}

// executeGenericForStatement 执行当前支持的 generic for 循环。
// 它基于 iterator、state 和 control value 这三元组驱动循环展开。
func (e *executor) executeGenericForStatement(statement *ir.GenericForStatement) (*executionResult, bool, error) {
	iteratorValues, err := e.evaluateExpressionList(statement.Iterators)
	if err != nil {
		return nil, false, err
	}

	iterator := NilValue()
	stateValue := NilValue()
	controlValue := NilValue()
	if len(iteratorValues) > 0 {
		iterator = iteratorValues[0]
	}
	if len(iteratorValues) > 1 {
		stateValue = iteratorValues[1]
	}
	if len(iteratorValues) > 2 {
		controlValue = iteratorValues[2]
	}

	for {
		returnValues, err := e.callFunctionValue(iterator, []Value{stateValue, controlValue})
		if err != nil {
			return nil, false, err
		}

		if len(returnValues) == 0 || returnValues[0].Type == ValueTypeNil {
			return nil, false, nil
		}

		controlValue = returnValues[0]

		e.pushScope()
		for index, name := range statement.Names {
			value := NilValue()
			if index < len(returnValues) {
				value = returnValues[index]
			}

			e.defineLocal(name, value)
		}

		result, done, err := e.executeBlock(statement.Body)
		e.popScope()
		if err != nil {
			return nil, false, err
		}

		if done {
			if result.breakLoop {
				return nil, false, nil
			}

			return result, true, nil
		}
	}
}

func (e *executor) executeBlock(statements []ir.Statement) (*executionResult, bool, error) {
	e.pushScope()
	defer e.popScope()

	for _, statement := range statements {
		result, done, err := e.executeStatement(statement)
		if err != nil {
			return nil, false, err
		}

		if done {
			return result, true, nil
		}
	}

	return nil, false, nil
}

func (e *executor) evaluateExpressionList(expressions []ir.Expression) ([]Value, error) {
	values := make([]Value, 0, len(expressions))
	for index, expression := range expressions {
		expanded, err := e.evaluateExpressionValues(expression, index == len(expressions)-1)
		if err != nil {
			return nil, err
		}

		values = append(values, expanded...)
	}

	return values, nil
}

func (e *executor) evaluateExpressionValues(expression ir.Expression, expandCall bool) ([]Value, error) {
	if expandCall {
		if call, ok := expression.(*ir.CallExpression); ok {
			return e.evaluateCallExpressionValues(call)
		}

		if _, ok := expression.(*ir.VarargExpression); ok {
			return append([]Value(nil), e.varargs...), nil
		}
	}

	value, err := e.evaluateExpression(expression)
	if err != nil {
		return nil, err
	}

	return []Value{value}, nil
}

func (e *executor) evaluateExpression(expression ir.Expression) (Value, error) {
	if err := e.consumeStep(); err != nil {
		return NilValue(), err
	}

	switch node := expression.(type) {
	case *ir.IdentifierExpression:
		value, ok := e.lookup(node.Name)
		if !ok {
			return NilValue(), nil
		}

		return value, nil
	case *ir.NilExpression:
		return NilValue(), nil
	case *ir.BooleanExpression:
		return Value{Type: ValueTypeBoolean, Data: node.Value}, nil
	case *ir.NumberExpression:
		number, err := parseNumberLiteral(node.Literal)
		if err != nil {
			return NilValue(), fmt.Errorf("parse number literal %q: %w", node.Literal, err)
		}

		return Value{Type: ValueTypeNumber, Data: number}, nil
	case *ir.StringExpression:
		return Value{Type: ValueTypeString, Data: node.Value}, nil
	case *ir.CallExpression:
		return e.evaluateCallExpression(node)
	case *ir.IndexExpression:
		target, err := e.evaluateExpression(node.Target)
		if err != nil {
			return NilValue(), err
		}

		if target.Type != ValueTypeTable {
			return NilValue(), fmt.Errorf("attempt to index non-table value of type %s", target.Type)
		}

		index, err := e.evaluateExpression(node.Index)
		if err != nil {
			return NilValue(), err
		}

		tableValue, ok := target.Data.(*table)
		if !ok {
			return NilValue(), fmt.Errorf("invalid table payload %T", target.Data)
		}

		return e.readTableIndex(tableValue, index)
	case *ir.TableConstructorExpression:
		tableValue := newTable()
		for index, field := range node.Fields {
			key, err := e.evaluateExpression(field.Key)
			if err != nil {
				return NilValue(), err
			}

			if field.IsListField && index == len(node.Fields)-1 {
				values, err := e.evaluateExpressionValues(field.Value, true)
				if err != nil {
					return NilValue(), err
				}

				baseIndex, err := requireNumber(key, "table constructor list field")
				if err != nil {
					return NilValue(), err
				}

				if len(values) == 0 {
					values = []Value{NilValue()}
				}

				for offset, value := range values {
					listKey := Value{Type: ValueTypeNumber, Data: baseIndex + float64(offset)}
					if err := tableValue.set(listKey, value); err != nil {
						return NilValue(), err
					}
				}

				continue
			}

			value, err := e.evaluateExpression(field.Value)
			if err != nil {
				return NilValue(), err
			}

			if err := tableValue.set(key, value); err != nil {
				return NilValue(), err
			}
		}

		return Value{Type: ValueTypeTable, Data: tableValue}, nil
	case *ir.FunctionExpression:
		return e.makeUserFunctionValue("", node.Parameters, node.IsVararg, node.Body), nil
	case *ir.VarargExpression:
		if len(e.varargs) == 0 {
			return NilValue(), nil
		}

		return e.varargs[0], nil
	case *ir.ParenthesizedExpression:
		return e.evaluateExpression(node.Inner)
	case *ir.UnaryExpression:
		return e.evaluateUnaryExpression(node)
	case *ir.BinaryExpression:
		return e.evaluateBinaryExpression(node)
	default:
		return NilValue(), fmt.Errorf("evaluate unsupported IR expression %T", expression)
	}
}

// consumeStep 在启用步数预算时扣减当前脚本的剩余执行步数。
// 当预算耗尽时会直接返回错误，用于阻止明显的无限循环长期占用执行线程。
func (e *executor) consumeStep() error {
	if e.ctx != nil {
		if err := e.ctx.Err(); err != nil {
			return err
		}
	}

	if e.stepLimit <= 0 {
		return nil
	}

	if e.remainingSteps <= 0 {
		return fmt.Errorf("execution step limit exceeded")
	}

	e.remainingSteps--
	return nil
}

func (e *executor) makeUserFunctionValue(name string, parameters []string, isVararg bool, body []ir.Statement) Value {
	return Value{
		Type: ValueTypeFunction,
		Data: &userFunction{
			name:       name,
			parameters: append([]string(nil), parameters...),
			isVararg:   isVararg,
			body:       append([]ir.Statement(nil), body...),
			captured:   e.snapshotVisibleCells(),
			env:        e.env,
		},
	}
}

func (e *executor) assignTarget(target ir.Expression, value Value) error {
	switch node := target.(type) {
	case *ir.IdentifierExpression:
		e.assign(node.Name, value)
		return nil
	case *ir.IndexExpression:
		targetValue, err := e.evaluateExpression(node.Target)
		if err != nil {
			return err
		}

		if targetValue.Type != ValueTypeTable {
			return fmt.Errorf("attempt to index-assign non-table value of type %s", targetValue.Type)
		}

		index, err := e.evaluateExpression(node.Index)
		if err != nil {
			return err
		}

		tableValue, ok := targetValue.Data.(*table)
		if !ok {
			return fmt.Errorf("invalid table payload %T", targetValue.Data)
		}

		return e.writeTableIndex(tableValue, index, value)
	default:
		return fmt.Errorf("unsupported assignment target %T", target)
	}
}

// readTableIndex 读取一个 table 字段，并在需要时应用最小 `__index` 元方法回退。
// 这条路径同时服务于方括号索引和点语法字段读取，并会阻止明显的链式回退环。
func (e *executor) readTableIndex(tableValue *table, index Value) (Value, error) {
	return e.readTableIndexWithVisited(tableValue, index, make(map[*table]struct{}))
}

// readTableIndexWithVisited 在读取 table 字段时携带一份已访问表集合。
// 这样当 `__index` table 回退形成自引用或环时，可以及时报错而不是无限递归。
func (e *executor) readTableIndexWithVisited(tableValue *table, index Value, visited map[*table]struct{}) (Value, error) {
	if e.state != nil && e.state.isGlobalEnv(tableValue) && index.Type == ValueTypeString {
		if value, ok := e.state.lookupGlobalValue(index.Data.(string)); ok {
			return value, nil
		}

		return NilValue(), nil
	}

	if _, exists := visited[tableValue]; exists {
		return NilValue(), fmt.Errorf("loop in table __index chain")
	}

	visited[tableValue] = struct{}{}

	value, exists, err := tableValue.get(index)
	if err != nil {
		return NilValue(), err
	}

	if exists {
		return value, nil
	}

	metatable := tableValue.getMetatable()
	if metatable == nil {
		return NilValue(), nil
	}

	metaIndex, exists, err := metatable.get(Value{Type: ValueTypeString, Data: "__index"})
	if err != nil {
		return NilValue(), err
	}

	if !exists {
		return NilValue(), nil
	}

	switch metaIndex.Type {
	case ValueTypeTable:
		fallbackTable, ok := metaIndex.Data.(*table)
		if !ok {
			return NilValue(), fmt.Errorf("invalid __index table payload %T", metaIndex.Data)
		}

		return e.readTableIndexWithVisited(fallbackTable, index, visited)
	case ValueTypeFunction:
		returnValues, err := e.callFunctionValue(metaIndex, []Value{
			{Type: ValueTypeTable, Data: tableValue},
			index,
		})
		if err != nil {
			return NilValue(), err
		}

		if len(returnValues) == 0 {
			return NilValue(), nil
		}

		return returnValues[0], nil
	default:
		// TODO: 后续按需要补齐 Lua 5.1 更完整的 `__index` 链式形态。
		// 当前如果元方法值既不是 table 也不是 function，则直接按非法索引目标报错。
		return NilValue(), fmt.Errorf("attempt to index non-table __index value of type %s", metaIndex.Type)
	}
}

// writeTableIndex 写入一个 table 字段，并在需要时应用最小 `__newindex` 元方法回退。
// 这条路径统一处理直接赋值和可能触发的元方法转发，并会阻止明显的链式回退环。
func (e *executor) writeTableIndex(tableValue *table, index Value, value Value) error {
	return e.writeTableIndexWithVisited(tableValue, index, value, make(map[*table]struct{}))
}

// writeTableIndexWithVisited 在写入 table 字段时携带一份已访问表集合。
// 这样当 `__newindex` table 回退形成自引用或环时，可以及时报错而不是无限递归。
func (e *executor) writeTableIndexWithVisited(tableValue *table, index Value, value Value, visited map[*table]struct{}) error {
	if e.state != nil && e.state.isGlobalEnv(tableValue) && index.Type == ValueTypeString {
		e.state.setGlobalValue(index.Data.(string), value)
		return nil
	}

	if _, exists := visited[tableValue]; exists {
		return fmt.Errorf("loop in table __newindex chain")
	}

	visited[tableValue] = struct{}{}

	_, exists, err := tableValue.get(index)
	if err != nil {
		return err
	}

	if exists {
		return tableValue.set(index, value)
	}

	metatable := tableValue.getMetatable()
	if metatable == nil {
		return tableValue.set(index, value)
	}

	metaNewIndex, exists, err := metatable.get(Value{Type: ValueTypeString, Data: "__newindex"})
	if err != nil {
		return err
	}

	if !exists {
		return tableValue.set(index, value)
	}

	switch metaNewIndex.Type {
	case ValueTypeTable:
		fallbackTable, ok := metaNewIndex.Data.(*table)
		if !ok {
			return fmt.Errorf("invalid __newindex table payload %T", metaNewIndex.Data)
		}

		return e.writeTableIndexWithVisited(fallbackTable, index, value, visited)
	case ValueTypeFunction:
		_, err := e.callFunctionValue(metaNewIndex, []Value{
			{Type: ValueTypeTable, Data: tableValue},
			index,
			value,
		})
		return err
	default:
		// TODO: 后续按需要补齐 Lua 5.1 更完整的 `__newindex` 链式形态。
		// 当前如果元方法值既不是 table 也不是 function，则直接按非法赋值目标报错。
		return fmt.Errorf("attempt to index-assign non-table __newindex value of type %s", metaNewIndex.Type)
	}
}

func (e *executor) evaluateCallExpression(expression *ir.CallExpression) (Value, error) {
	values, err := e.evaluateCallExpressionValues(expression)
	if err != nil {
		return NilValue(), err
	}

	if len(values) == 0 {
		return NilValue(), nil
	}

	return values[0], nil
}

func (e *executor) evaluateCallExpressionValues(expression *ir.CallExpression) ([]Value, error) {
	if expression.Receiver != nil {
		receiver, err := e.evaluateExpression(expression.Receiver)
		if err != nil {
			return nil, err
		}

		method, err := e.lookupMethod(receiver, expression.Method)
		if err != nil {
			return nil, err
		}

		arguments, err := e.evaluateExpressionList(expression.Arguments)
		if err != nil {
			return nil, err
		}

		callArgs := make([]Value, 0, len(arguments)+1)
		callArgs = append(callArgs, receiver)
		callArgs = append(callArgs, arguments...)
		return e.callFunctionValue(method, callArgs)
	}

	callee, err := e.evaluateExpression(expression.Callee)
	if err != nil {
		return nil, err
	}

	arguments, err := e.evaluateExpressionList(expression.Arguments)
	if err != nil {
		return nil, err
	}

	return e.callFunctionValue(callee, arguments)
}

// lookupMethod 基于现有索引语义从接收者上解析一个方法名。
// 这让 `obj:method(...)` 可以复用 table 读取和 metatable 回退的既有逻辑。
func (e *executor) lookupMethod(receiver Value, method string) (Value, error) {
	if receiver.Type != ValueTypeTable {
		return NilValue(), fmt.Errorf("attempt to call method %q on non-table value of type %s", method, receiver.Type)
	}

	tableValue, ok := receiver.Data.(*table)
	if !ok {
		return NilValue(), fmt.Errorf("invalid table payload %T", receiver.Data)
	}

	return e.readTableIndex(tableValue, Value{Type: ValueTypeString, Data: method})
}

func (e *executor) pushScope() {
	e.scopes = append(e.scopes, map[string]*valueCell{})
}

func (e *executor) currentScope() map[string]*valueCell {
	return e.scopes[len(e.scopes)-1]
}

// currentEnv 返回当前执行帧绑定的最小环境表。
// 当执行器上还没有显式环境时，会回退到运行时的 `_G`。
func (e *executor) currentEnv() *table {
	if len(e.envFrames) > 0 {
		return e.envFrames[len(e.envFrames)-1].env
	}
	if e.env != nil {
		return e.env
	}
	if e.state != nil {
		return e.state.globalEnv
	}

	return nil
}

// threadEnv 返回当前执行线程绑定的最小全局环境表。
// `getfenv(0)` / `setfenv(0, ...)` 会通过这条路径观察和改写线程级环境。
func (e *executor) threadEnv() *table {
	if len(e.envFrames) > 0 {
		return e.envFrames[0].env
	}

	return e.currentEnv()
}

// setThreadEnv 改写当前执行线程绑定的最小全局环境表。
// 当当前只剩顶层 chunk 一层活动帧时，也会同步刷新当前帧环境。
func (e *executor) setThreadEnv(env *table) {
	if len(e.envFrames) == 0 {
		e.env = env
		return
	}

	e.envFrames[0].env = env
	e.env = e.currentEnv()
}

// setCurrentEnv 改写当前执行帧绑定的环境表，并同步更新环境栈顶部。
// `module(...)` 这类会直接切当前 chunk 环境的路径会复用它，保持 `getfenv(1)` 观察一致。
func (e *executor) setCurrentEnv(env *table) {
	if len(e.envFrames) == 0 {
		e.env = env
		return
	}

	e.envFrames[len(e.envFrames)-1].env = env
	e.syncFrameFunctionEnv(len(e.envFrames) - 1)
	e.env = env
}

// envByLevel 按最小 `getfenv` / `setfenv` 规则读取当前活跃调用栈上的某一层环境。
// level=0 和 level=1 都指向当前执行帧；更大的 level 会继续向外层调用者回退。
func (e *executor) envByLevel(level int) (*table, error) {
	if level < 0 {
		return nil, fmt.Errorf("environment level must be non-negative")
	}

	if len(e.envFrames) == 0 {
		return e.currentEnv(), nil
	}

	if level <= 1 {
		return e.currentEnv(), nil
	}

	index := len(e.envFrames) - level
	if index < 0 || index >= len(e.envFrames) {
		return nil, fmt.Errorf("environment level %d out of range", level)
	}

	return e.envFrames[index].env, nil
}

// setEnvByLevel 按最小 `setfenv` 规则改写当前活跃调用栈上的某一层环境。
// level=0 和 level=1 会改当前执行帧；更大的 level 会改外层调用者环境。
func (e *executor) setEnvByLevel(level int, env *table) error {
	if level < 0 {
		return fmt.Errorf("environment level must be non-negative")
	}

	if len(e.envFrames) == 0 {
		e.env = env
		return nil
	}

	index := len(e.envFrames) - 1
	if level > 1 {
		index = len(e.envFrames) - level
	}

	if index < 0 || index >= len(e.envFrames) {
		return fmt.Errorf("environment level %d out of range", level)
	}

	e.envFrames[index].env = env
	e.syncFrameFunctionEnv(index)
	e.env = e.currentEnv()
	return nil
}

// syncFrameFunctionEnv 把某个活跃调用帧的环境同步回对应函数对象。
// 这样栈级 `setfenv(level, env)` 不会只影响当前一次调用，而会持续影响后续调用。
func (e *executor) syncFrameFunctionEnv(index int) {
	if index < 0 || index >= len(e.envFrames) {
		return
	}

	frame := e.envFrames[index]
	if frame.userFunction != nil {
		frame.userFunction.env = frame.env
	}
	if frame.nativeFunction != nil {
		frame.nativeFunction.env = frame.env
	}
}

// syncActiveFunctionFrames 把某个函数对象的新环境同步到当前活跃调用栈里对应的帧。
// 这样 `setfenv(fn, env)` 命中“当前正在执行的函数”时，本次调用后续的全局读写也会立刻看到新环境。
func (e *executor) syncActiveFunctionFrames(target Value, env *table) {
	for index := range e.envFrames {
		frame := &e.envFrames[index]
		switch functionValue := target.Data.(type) {
		case *userFunction:
			if frame.userFunction == functionValue {
				frame.env = env
			}
		case *nativeFunction:
			if frame.nativeFunction == functionValue {
				frame.env = env
			}
		}
	}

	e.env = e.currentEnv()
}

// setFunctionValueEnv 改写函数对象绑定的环境表，并同步当前活跃调用栈里命中的同一函数对象。
// 这样 `setfenv(fn, env)` 不会只影响未来调用，当前这次调用后半段也会立即沿用新环境。
func (e *executor) setFunctionValueEnv(value Value, env *table) error {
	if err := setFunctionEnvironment(value, env); err != nil {
		return err
	}

	e.syncActiveFunctionFrames(value, env)
	return nil
}

func (e *executor) popScope() {
	if len(e.scopes) <= 1 {
		e.scopes[0] = map[string]*valueCell{}
		return
	}

	e.scopes = e.scopes[:len(e.scopes)-1]
}

func (e *executor) callFunctionValue(callee Value, arguments []Value) ([]Value, error) {
	return e.callFunctionValueWithVisited(callee, arguments, make(map[*table]struct{}))
}

// callFunctionValueWithVisited 调用一个运行时可调用值，并跟踪 `__call` 链上已经访问过的 table。
// 这样当 `__call` 元方法形成自引用或环时，可以及时报错而不是无限递归。
func (e *executor) callFunctionValueWithVisited(callee Value, arguments []Value, visited map[*table]struct{}) ([]Value, error) {
	if callee.Type != ValueTypeFunction {
		if callee.Type == ValueTypeTable {
			tableValue, ok := callee.Data.(*table)
			if !ok {
				return nil, fmt.Errorf("invalid table payload %T", callee.Data)
			}

			if _, exists := visited[tableValue]; exists {
				return nil, fmt.Errorf("loop in table __call chain")
			}

			visited[tableValue] = struct{}{}

			metaCall, exists, err := e.lookupMetamethod(callee, "__call")
			if err != nil {
				return nil, err
			}

			if exists {
				callArgs := make([]Value, 0, len(arguments)+1)
				callArgs = append(callArgs, Value{Type: ValueTypeTable, Data: tableValue})
				callArgs = append(callArgs, arguments...)
				return e.callFunctionValueWithVisited(metaCall, callArgs, visited)
			}
		}

		return nil, fmt.Errorf("attempt to call non-function value of type %s", callee.Type)
	}

	switch functionValue := callee.Data.(type) {
	case *userFunction:
		return e.callUserFunction(functionValue, arguments)
	case *nativeFunction:
		return e.callNativeFunction(functionValue, arguments)
	default:
		return nil, fmt.Errorf("invalid function payload %T", callee.Data)
	}
}

func (e *executor) callUserFunction(functionValue *userFunction, arguments []Value) ([]Value, error) {
	if functionValue == nil {
		return nil, fmt.Errorf("call nil function")
	}

	savedScopes := e.scopes
	savedVarargs := e.varargs
	capturedScope := copyCellMap(functionValue.captured)
	frameEnv := functionValue.env
	if frameEnv == nil && e.state != nil {
		frameEnv = e.state.globalEnv
	}
	e.envFrames = append(e.envFrames, envFrame{
		env:          frameEnv,
		userFunction: functionValue,
	})
	e.env = frameEnv
	e.scopes = []map[string]*valueCell{capturedScope, {}}
	defer func() {
		e.scopes = savedScopes
		e.varargs = savedVarargs
		if len(e.envFrames) > 0 {
			e.envFrames = e.envFrames[:len(e.envFrames)-1]
		}
		e.env = e.currentEnv()
	}()

	for index, parameter := range functionValue.parameters {
		value := NilValue()
		if index < len(arguments) {
			value = arguments[index]
		}

		e.defineLocal(parameter, value)
	}

	if functionValue.isVararg && len(arguments) > len(functionValue.parameters) {
		e.varargs = append([]Value(nil), arguments[len(functionValue.parameters):]...)
	} else {
		e.varargs = nil
	}

	for _, statement := range functionValue.body {
		result, done, err := e.executeStatement(statement)
		if err != nil {
			return nil, err
		}

		if done {
			return append([]Value(nil), result.returnValues...), nil
		}
	}

	return nil, nil
}

func (e *executor) callNativeFunction(functionValue *nativeFunction, arguments []Value) ([]Value, error) {
	if functionValue == nil || (functionValue.fn == nil && functionValue.contextualImpl == nil) {
		return nil, fmt.Errorf("call nil native function")
	}

	frameEnv := functionValue.env
	if frameEnv == nil && e.state != nil {
		frameEnv = e.state.globalEnv
	}
	e.envFrames = append(e.envFrames, envFrame{
		env:            frameEnv,
		nativeFunction: functionValue,
	})
	e.env = frameEnv
	defer func() {
		if len(e.envFrames) > 0 {
			e.envFrames = e.envFrames[:len(e.envFrames)-1]
		}
		e.env = e.currentEnv()
	}()

	if functionValue.contextualImpl != nil {
		return functionValue.contextualImpl(e, arguments)
	}

	return functionValue.fn(arguments)
}

// lookupMetamethod 从当前运行时值上解析一个受支持的 metatable 字段。
// 当前实现只覆盖已落地的最小元方法集合。
func (e *executor) lookupMetamethod(value Value, field string) (Value, bool, error) {
	if value.Type != ValueTypeTable {
		return NilValue(), false, nil
	}

	tableValue, ok := value.Data.(*table)
	if !ok {
		return NilValue(), false, fmt.Errorf("invalid table payload %T", value.Data)
	}

	metatable := tableValue.getMetatable()
	if metatable == nil {
		return NilValue(), false, nil
	}

	return metatable.get(Value{Type: ValueTypeString, Data: field})
}

// valueToString 把一个运行时值渲染成字符串。
// 对 table 会优先尝试最小 `__tostring` 元方法钩子，再回退到默认格式。
func (e *executor) valueToString(value Value) (string, error) {
	metaToString, exists, err := e.lookupMetamethod(value, "__tostring")
	if err != nil {
		return "", err
	}

	if exists {
		returnValues, err := e.callFunctionValue(metaToString, []Value{value})
		if err != nil {
			return "", err
		}

		if len(returnValues) == 0 {
			return "", nil
		}

		return valueToString(returnValues[0]), nil
	}

	return valueToString(value), nil
}

func (e *executor) defineLocal(name string, value Value) {
	e.scopes[len(e.scopes)-1][name] = &valueCell{value: value}
}

func (e *executor) assign(name string, value Value) {
	for index := len(e.scopes) - 1; index >= 0; index-- {
		if cell, ok := e.scopes[index][name]; ok {
			cell.value = value
			return
		}
	}

	// 未声明名称的普通赋值会回落到全局环境。
	// 如果当前函数绑定了自定义环境，则会优先写入那份环境表。
	if e.env != nil {
		if err := e.writeTableIndex(e.env, Value{Type: ValueTypeString, Data: name}, value); err == nil {
			return
		}
	}

	if e.state != nil {
		e.state.setGlobalValue(name, value)
		return
	}

	e.scopes[0][name] = &valueCell{value: value}
}

func (e *executor) lookup(name string) (Value, bool) {
	for index := len(e.scopes) - 1; index >= 0; index-- {
		cell, ok := e.scopes[index][name]
		if ok {
			return cell.value, true
		}
	}

	if e.env != nil {
		value, err := e.readTableIndex(e.env, Value{Type: ValueTypeString, Data: name})
		if err == nil && value.Type != ValueTypeNil {
			return value, true
		}
	}

	return NilValue(), false
}

func (e *executor) snapshotVisibleCells() map[string]*valueCell {
	snapshot := make(map[string]*valueCell)
	for _, scope := range e.scopes {
		for name, cell := range scope {
			snapshot[name] = cell
		}
	}

	return snapshot
}

func copyCellMap(input map[string]*valueCell) map[string]*valueCell {
	if input == nil {
		return map[string]*valueCell{}
	}

	output := make(map[string]*valueCell, len(input))
	for key, value := range input {
		output[key] = value
	}

	return output
}

func (e *executor) evaluateUnaryExpression(expression *ir.UnaryExpression) (Value, error) {
	operand, err := e.evaluateExpression(expression.Operand)
	if err != nil {
		return NilValue(), err
	}

	switch expression.Operator {
	case "-":
		number, err := requireNumber(operand, "unary '-'")
		if err != nil {
			return e.evaluateUnaryMetamethod(operand, "__unm", err)
		}

		return Value{Type: ValueTypeNumber, Data: -number}, nil
	case "not":
		return Value{Type: ValueTypeBoolean, Data: !isTruthy(operand)}, nil
	case "#":
		if operand.Type == ValueTypeString {
			return Value{Type: ValueTypeNumber, Data: float64(len(operand.Data.(string)))}, nil
		}

		if operand.Type == ValueTypeTable {
			tableValue, ok := operand.Data.(*table)
			if !ok {
				return NilValue(), fmt.Errorf("invalid table payload %T", operand.Data)
			}

			length, err := tableValue.borderLength()
			if err != nil {
				return NilValue(), err
			}

			return Value{Type: ValueTypeNumber, Data: float64(length)}, nil
		}

		return NilValue(), fmt.Errorf("operator '#' expects string or table operand, got %s", operand.Type)
	default:
		return NilValue(), fmt.Errorf("unsupported unary operator %q", expression.Operator)
	}
}

func (e *executor) evaluateBinaryExpression(expression *ir.BinaryExpression) (Value, error) {
	switch expression.Operator {
	case "and":
		left, err := e.evaluateExpression(expression.Left)
		if err != nil {
			return NilValue(), err
		}

		if !isTruthy(left) {
			return left, nil
		}

		return e.evaluateExpression(expression.Right)
	case "or":
		left, err := e.evaluateExpression(expression.Left)
		if err != nil {
			return NilValue(), err
		}

		if isTruthy(left) {
			return left, nil
		}

		return e.evaluateExpression(expression.Right)
	}

	left, err := e.evaluateExpression(expression.Left)
	if err != nil {
		return NilValue(), err
	}

	right, err := e.evaluateExpression(expression.Right)
	if err != nil {
		return NilValue(), err
	}

	switch expression.Operator {
	case "+":
		return e.evaluateBinaryArithmetic(left, right, expression.Operator, "__add", func(a, b float64) float64 { return a + b })
	case "-":
		return e.evaluateBinaryArithmetic(left, right, expression.Operator, "__sub", func(a, b float64) float64 { return a - b })
	case "*":
		return e.evaluateBinaryArithmetic(left, right, expression.Operator, "__mul", func(a, b float64) float64 { return a * b })
	case "/":
		return e.evaluateBinaryArithmetic(left, right, expression.Operator, "__div", func(a, b float64) float64 { return a / b })
	case "%":
		return e.evaluateBinaryArithmetic(left, right, expression.Operator, "__mod", math.Mod)
	case "^":
		return e.evaluateBinaryArithmetic(left, right, expression.Operator, "__pow", math.Pow)
	case "..":
		// Lua 5.1 的原生拼接只接受字符串和数字；
		// 任一侧超出这个范围时，都应该回退到 `__concat` 或直接报错。
		if isStringLikeValue(left) && isStringLikeValue(right) {
			return Value{Type: ValueTypeString, Data: valueToString(left) + valueToString(right)}, nil
		}

		return e.evaluateBinaryMetamethod(left, right, "__concat", fmt.Errorf("operator %q expects string-like operands, got %s and %s", expression.Operator, left.Type, right.Type))
	case "<":
		return e.evaluateOrderedComparison(left, right, expression.Operator, expression.Operator, "__lt", func(a, b float64) bool { return a < b })
	case "<=":
		return e.evaluateOrderedComparison(left, right, expression.Operator, expression.Operator, "__le", func(a, b float64) bool { return a <= b })
	case ">":
		return e.evaluateOrderedComparison(right, left, "<", expression.Operator, "__lt", func(a, b float64) bool { return a < b })
	case ">=":
		return e.evaluateOrderedComparison(right, left, "<=", expression.Operator, "__le", func(a, b float64) bool { return a <= b })
	case "==":
		return e.evaluateEquality(left, right)
	case "~=":
		value, err := e.evaluateEquality(left, right)
		if err != nil {
			return NilValue(), err
		}

		return Value{Type: ValueTypeBoolean, Data: !value.Data.(bool)}, nil
	default:
		return NilValue(), fmt.Errorf("unsupported binary operator %q", expression.Operator)
	}
}

// evaluateUnaryMetamethod 在一元运算直接类型检查失败时尝试走元方法回退。
// 当前只对已支持的一元元方法做最小兼容。
func (e *executor) evaluateUnaryMetamethod(operand Value, metamethod string, cause error) (Value, error) {
	metaFn, exists, err := e.lookupMetamethod(operand, metamethod)
	if err != nil {
		return NilValue(), err
	}

	if !exists {
		return NilValue(), cause
	}

	returnValues, err := e.callFunctionValue(metaFn, []Value{operand})
	if err != nil {
		return NilValue(), err
	}

	if len(returnValues) == 0 {
		return NilValue(), nil
	}

	return returnValues[0], nil
}

// evaluateBinaryArithmetic 先尝试直接做数值算术运算。
// 如果操作数不满足直接计算条件，再回退到已支持的二元元方法路径。
func (e *executor) evaluateBinaryArithmetic(left, right Value, operator string, metamethod string, fn func(float64, float64) float64) (Value, error) {
	value, err := numericBinary(left, right, operator, fn)
	if err == nil {
		return value, nil
	}

	return e.evaluateBinaryMetamethod(left, right, metamethod, err)
}

// evaluateBinaryMetamethod 按“先左后右”的顺序尝试解析并调用二元元方法。
// 这与 Lua 常见的双操作数元方法查找顺序保持一致。
func (e *executor) evaluateBinaryMetamethod(left, right Value, metamethod string, cause error) (Value, error) {
	metaFn, exists, err := e.lookupBinaryMetamethod(left, right, metamethod)
	if err != nil {
		return NilValue(), err
	}

	if !exists {
		if cause == nil {
			return NilValue(), nil
		}

		return NilValue(), cause
	}

	returnValues, err := e.callFunctionValue(metaFn, []Value{left, right})
	if err != nil {
		return NilValue(), err
	}

	if len(returnValues) == 0 {
		return NilValue(), nil
	}

	return returnValues[0], nil
}

// evaluateSharedBinaryMetamethod 只在两侧共享同一个二元元方法时才执行调用。
// 这主要用于 Lua 5.1 的比较元方法路径，如 `__lt` / `__le`。
func (e *executor) evaluateSharedBinaryMetamethod(left, right Value, metamethod string, cause error) (Value, error) {
	metaFn, exists, err := e.lookupSharedBinaryMetamethod(left, right, metamethod)
	if err != nil {
		return NilValue(), err
	}

	if !exists {
		if cause == nil {
			return NilValue(), nil
		}

		return NilValue(), cause
	}

	returnValues, err := e.callFunctionValue(metaFn, []Value{left, right})
	if err != nil {
		return NilValue(), err
	}

	if len(returnValues) == 0 {
		return NilValue(), nil
	}

	return returnValues[0], nil
}

// evaluateOrderedComparison 先尝试直接做数字或字符串的有序比较，再回退到已支持的比较元方法。
// directOperator 表示当前实际执行的基础比较方向，errorOperator 用于保留原始运算符错误文本。
func (e *executor) evaluateOrderedComparison(left, right Value, directOperator string, errorOperator string, metamethod string, fn func(float64, float64) bool) (Value, error) {
	value, err := comparisonBinary(left, right, directOperator, errorOperator, fn)
	if err == nil {
		return value, nil
	}

	if metamethod == "__le" {
		value, sharedErr := e.evaluateSharedBinaryMetamethod(left, right, "__le", nil)
		if sharedErr != nil {
			return NilValue(), sharedErr
		}

		if value.Type != ValueTypeNil {
			return value, nil
		}

		value, ok, fallbackErr := e.evaluateLessEqualFallback(left, right)
		if fallbackErr != nil {
			return NilValue(), fallbackErr
		}

		if ok {
			return value, nil
		}
	}

	return e.evaluateSharedBinaryMetamethod(left, right, metamethod, err)
}

// evaluateLessEqualFallback 为 `<=` 提供 Lua 5.1 风格的最小 `__lt` 反向回退。
// 当 `__le` 缺失时，会尝试计算 `not (right < left)`。
func (e *executor) evaluateLessEqualFallback(left, right Value) (Value, bool, error) {
	value, err := e.evaluateSharedBinaryMetamethod(right, left, "__lt", nil)
	if err != nil {
		return NilValue(), false, err
	}

	if value.Type == ValueTypeNil {
		return NilValue(), false, nil
	}

	return Value{Type: ValueTypeBoolean, Data: !isTruthy(value)}, true, nil
}

// evaluateEquality 先执行原始相等性判断，再在需要时回退到最小 `__eq` 元方法路径。
// 当前元方法相等性主要面向 table 值，并要求两侧共享同一个 `__eq` 元方法。
func (e *executor) evaluateEquality(left, right Value) (Value, error) {
	if valuesEqual(left, right) {
		return Value{Type: ValueTypeBoolean, Data: true}, nil
	}

	if left.Type != right.Type || left.Type != ValueTypeTable {
		return Value{Type: ValueTypeBoolean, Data: false}, nil
	}

	metaFn, exists, err := e.lookupSharedEqualityMetamethod(left, right)
	if err != nil {
		return NilValue(), err
	}

	if !exists {
		return Value{Type: ValueTypeBoolean, Data: false}, nil
	}

	returnValues, err := e.callFunctionValue(metaFn, []Value{left, right})
	if err != nil {
		return NilValue(), err
	}

	if len(returnValues) == 0 || returnValues[0].Type == ValueTypeNil {
		return Value{Type: ValueTypeBoolean, Data: false}, nil
	}

	return Value{Type: ValueTypeBoolean, Data: isTruthy(returnValues[0])}, nil
}

// lookupSharedEqualityMetamethod 查找可用于 `__eq` 的共享元方法。
// 按 Lua 5.1 的最小规则，只有两侧都声明了同一个 `__eq` 值时，才允许触发元方法比较。
func (e *executor) lookupSharedEqualityMetamethod(left, right Value) (Value, bool, error) {
	leftMetaFn, leftExists, err := e.lookupMetamethod(left, "__eq")
	if err != nil {
		return NilValue(), false, err
	}

	if !leftExists {
		return NilValue(), false, nil
	}

	rightMetaFn, rightExists, err := e.lookupMetamethod(right, "__eq")
	if err != nil {
		return NilValue(), false, err
	}

	if !rightExists || !valuesEqual(leftMetaFn, rightMetaFn) {
		return NilValue(), false, nil
	}

	return leftMetaFn, true, nil
}

// lookupSharedBinaryMetamethod 查找可用于有序比较的共享二元元方法。
// 按 Lua 5.1 的最小规则，只有两侧都声明了同一个元方法值时，才允许触发比较元方法。
func (e *executor) lookupSharedBinaryMetamethod(left, right Value, field string) (Value, bool, error) {
	if left.Type != right.Type || left.Type != ValueTypeTable {
		return NilValue(), false, nil
	}

	leftMetaFn, leftExists, err := e.lookupMetamethod(left, field)
	if err != nil {
		return NilValue(), false, err
	}

	if !leftExists {
		return NilValue(), false, nil
	}

	rightMetaFn, rightExists, err := e.lookupMetamethod(right, field)
	if err != nil {
		return NilValue(), false, err
	}

	if !rightExists || !valuesEqual(leftMetaFn, rightMetaFn) {
		return NilValue(), false, nil
	}

	return leftMetaFn, true, nil
}

// lookupBinaryMetamethod 按先左后右的顺序查找二元元方法入口。
// 这样可以把 Lua 的常见元方法分派规则集中收敛在一个位置处理。
func (e *executor) lookupBinaryMetamethod(left, right Value, field string) (Value, bool, error) {
	metaFn, exists, err := e.lookupMetamethod(left, field)
	if err != nil {
		return NilValue(), false, err
	}

	if exists {
		return metaFn, true, nil
	}

	return e.lookupMetamethod(right, field)
}

func numericBinary(left, right Value, operator string, fn func(float64, float64) float64) (Value, error) {
	leftNumber, err := requireNumber(left, operator)
	if err != nil {
		return NilValue(), err
	}

	rightNumber, err := requireNumber(right, operator)
	if err != nil {
		return NilValue(), err
	}

	return Value{Type: ValueTypeNumber, Data: fn(leftNumber, rightNumber)}, nil
}

func comparisonBinary(left, right Value, directOperator string, errorOperator string, fn func(float64, float64) bool) (Value, error) {
	if left.Type == ValueTypeString && right.Type == ValueTypeString {
		leftText, leftOK := left.Data.(string)
		rightText, rightOK := right.Data.(string)
		if !leftOK || !rightOK {
			return NilValue(), fmt.Errorf("invalid string comparison payloads %T and %T", left.Data, right.Data)
		}

		switch directOperator {
		case "<":
			return Value{Type: ValueTypeBoolean, Data: leftText < rightText}, nil
		case "<=":
			return Value{Type: ValueTypeBoolean, Data: leftText <= rightText}, nil
		}
	}

	leftNumber, err := requireStrictNumber(left, errorOperator)
	if err != nil {
		return NilValue(), err
	}

	rightNumber, err := requireStrictNumber(right, errorOperator)
	if err != nil {
		return NilValue(), err
	}

	return Value{Type: ValueTypeBoolean, Data: fn(leftNumber, rightNumber)}, nil
}

// numericForContinues 根据步长正负应用 Lua 风格的数值 for 边界判断。
// 正步长使用 `<=`，负步长使用 `>=`，从而决定循环是否继续。
func numericForContinues(current float64, limit float64, step float64) bool {
	if step > 0 {
		return current <= limit
	}

	return current >= limit
}

// requireNumber 按 Lua 5.1 的最小数值强转规则读取一个可参与数值运算的值。
// 当前会接受原生 number，以及可被基础浮点解析接受的字符串。
func requireNumber(value Value, operator string) (float64, error) {
	switch value.Type {
	case ValueTypeNumber:
		number, ok := value.Data.(float64)
		if !ok {
			return 0, fmt.Errorf("operator %q received invalid number payload %T", operator, value.Data)
		}

		return number, nil
	case ValueTypeString:
		text, ok := value.Data.(string)
		if !ok {
			return 0, fmt.Errorf("operator %q received invalid string payload %T", operator, value.Data)
		}

		number, err := strconv.ParseFloat(strings.TrimSpace(text), 64)
		if err != nil {
			return 0, fmt.Errorf("operator %q expects number operand, got %s", operator, value.Type)
		}

		return number, nil
	default:
		return 0, fmt.Errorf("operator %q expects number operand, got %s", operator, value.Type)
	}
}

// requireStrictNumber 只接受原生 number，不做字符串到数值的强转。
// 关系比较会用它保留 Lua 5.1“数字和字符串不能混比”的最小规则。
func requireStrictNumber(value Value, operator string) (float64, error) {
	if value.Type != ValueTypeNumber {
		return 0, fmt.Errorf("operator %q expects number operand, got %s", operator, value.Type)
	}

	number, ok := value.Data.(float64)
	if !ok {
		return 0, fmt.Errorf("operator %q received invalid number payload %T", operator, value.Data)
	}

	return number, nil
}

// isStringLikeValue 判断当前值是否能直接参与 Lua 5.1 的原生字符串拼接。
// 当前只把字符串和数字视为可直接拼接的基础值类型。
func isStringLikeValue(value Value) bool {
	return value.Type == ValueTypeString || value.Type == ValueTypeNumber
}

func parseNumberLiteral(literal string) (float64, error) {
	if strings.HasPrefix(literal, "0x") || strings.HasPrefix(literal, "0X") {
		number, err := strconv.ParseUint(literal[2:], 16, 64)
		if err != nil {
			return 0, err
		}

		return float64(number), nil
	}

	return strconv.ParseFloat(literal, 64)
}

func isTruthy(value Value) bool {
	switch value.Type {
	case ValueTypeNil:
		return false
	case ValueTypeBoolean:
		booleanValue, _ := value.Data.(bool)
		return booleanValue
	default:
		return true
	}
}

func valuesEqual(left, right Value) bool {
	if left.Type != right.Type {
		return false
	}

	switch left.Type {
	case ValueTypeNil:
		return true
	default:
		return left.Data == right.Data
	}
}

func valueToString(value Value) string {
	switch value.Type {
	case ValueTypeNil:
		return "nil"
	case ValueTypeBoolean:
		if value.Data.(bool) {
			return "true"
		}

		return "false"
	case ValueTypeNumber:
		return strconv.FormatFloat(value.Data.(float64), 'f', -1, 64)
	case ValueTypeString:
		return value.Data.(string)
	default:
		return fmt.Sprintf("%v", value.Data)
	}
}
