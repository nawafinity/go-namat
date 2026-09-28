package expr

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// Function is a safe host function callable by template expressions.
type Function func(args ...any) (any, error)

// Context holds root data, local variables, and registered functions.
type Context struct {
	Root      any
	Variables map[string]any
	Functions map[string]Function
}

// Eval evaluates a compiled expression.
func (p *Program) Eval(ctx *Context) (any, error) {
	if p == nil || p.root == nil {
		return nil, errors.New("namat expression: empty program")
	}
	if ctx == nil {
		ctx = &Context{}
	}
	return p.root.eval(ctx)
}

type node interface {
	eval(*Context) (any, error)
}

type literalNode struct{ value any }

func (n literalNode) eval(*Context) (any, error) { return n.value, nil }

type arrayNode struct{ values []node }

func (n arrayNode) eval(ctx *Context) (any, error) {
	values := make([]any, len(n.values))
	for index, value := range n.values {
		resolved, err := value.eval(ctx)
		if err != nil {
			return nil, err
		}
		values[index] = resolved
	}
	return values, nil
}

type objectEntry struct {
	key   string
	value node
}

type objectNode struct{ entries []objectEntry }

func (n objectNode) eval(ctx *Context) (any, error) {
	value := make(map[string]any, len(n.entries))
	for _, entry := range n.entries {
		resolved, err := entry.value.eval(ctx)
		if err != nil {
			return nil, err
		}
		value[entry.key] = resolved
	}
	return value, nil
}

type conditionalNode struct {
	condition node
	whenTrue  node
	whenFalse node
}

func (n conditionalNode) eval(ctx *Context) (any, error) {
	condition, err := n.condition.eval(ctx)
	if err != nil {
		return nil, err
	}
	if truthy(condition) {
		return n.whenTrue.eval(ctx)
	}
	return n.whenFalse.eval(ctx)
}

type identifierNode struct{ name string }

type unknownIdentifierError struct{ name string }

func (e unknownIdentifierError) Error() string {
	return fmt.Sprintf("namat expression: unknown identifier %q", e.name)
}

func (n identifierNode) eval(ctx *Context) (any, error) {
	if ctx.Variables != nil {
		if value, ok := ctx.Variables[n.name]; ok {
			return value, nil
		}
	}
	if ctx.Functions != nil {
		if fn, ok := ctx.Functions[n.name]; ok {
			return fn, nil
		}
	}
	value, ok := lookup(ctx.Root, n.name)
	if !ok {
		return nil, unknownIdentifierError{name: n.name}
	}
	return value, nil
}

type memberNode struct {
	target   node
	key      node
	optional bool
}

func (n memberNode) eval(ctx *Context) (any, error) {
	target, err := n.target.eval(ctx)
	if err != nil {
		var unknown unknownIdentifierError
		if n.optional && errors.As(err, &unknown) {
			return nil, nil
		}
		return nil, err
	}
	if isNil(target) {
		if n.optional {
			return nil, nil
		}
		return nil, errors.New("namat expression: cannot access a property of null")
	}
	key, err := n.key.eval(ctx)
	if err != nil {
		return nil, err
	}
	value, ok := lookup(target, key)
	if !ok {
		if member, found := builtinMember(target, stringify(key)); found {
			return member, nil
		}
		if n.optional {
			return nil, nil
		}
		return nil, fmt.Errorf("namat expression: property %v not found", key)
	}
	return value, nil
}

func builtinMember(target any, name string) (any, bool) {
	v := reflect.ValueOf(target)
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}
	if !v.IsValid() {
		return nil, false
	}
	switch v.Kind() {
	case reflect.String:
		text := v.String()
		switch name {
		case "length":
			return len([]rune(text)), true
		case "trim":
			return Function(func(args ...any) (any, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("trim expects no arguments")
				}
				return strings.TrimSpace(text), nil
			}), true
		case "toUpperCase", "upper":
			return Function(func(args ...any) (any, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("%s expects no arguments", name)
				}
				return strings.ToUpper(text), nil
			}), true
		case "toLowerCase", "lower":
			return Function(func(args ...any) (any, error) {
				if len(args) != 0 {
					return nil, fmt.Errorf("%s expects no arguments", name)
				}
				return strings.ToLower(text), nil
			}), true
		case "contains", "includes":
			return Function(func(args ...any) (any, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("%s expects one argument", name)
				}
				return strings.Contains(text, stringify(args[0])), nil
			}), true
		case "startsWith":
			return Function(func(args ...any) (any, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("startsWith expects one argument")
				}
				return strings.HasPrefix(text, stringify(args[0])), nil
			}), true
		case "endsWith":
			return Function(func(args ...any) (any, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("endsWith expects one argument")
				}
				return strings.HasSuffix(text, stringify(args[0])), nil
			}), true
		case "slice":
			return Function(func(args ...any) (any, error) {
				if len(args) < 1 || len(args) > 2 {
					return nil, fmt.Errorf("slice expects one or two arguments")
				}
				runes := []rune(text)
				start, ok := integer(args[0])
				if !ok {
					return nil, fmt.Errorf("slice start must be an integer")
				}
				end := len(runes)
				if len(args) == 2 {
					end, ok = integer(args[1])
					if !ok {
						return nil, fmt.Errorf("slice end must be an integer")
					}
				}
				start, end = normalizeSliceBounds(start, end, len(runes))
				return string(runes[start:end]), nil
			}), true
		}
	case reflect.Array, reflect.Slice:
		switch name {
		case "length":
			return v.Len(), true
		case "join":
			return Function(func(args ...any) (any, error) {
				if len(args) > 1 {
					return nil, fmt.Errorf("join expects zero or one argument")
				}
				separator := ","
				if len(args) == 1 {
					separator = stringify(args[0])
				}
				items := make([]string, v.Len())
				for index := 0; index < v.Len(); index++ {
					items[index] = stringify(v.Index(index).Interface())
				}
				return strings.Join(items, separator), nil
			}), true
		case "includes", "contains":
			return Function(func(args ...any) (any, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("%s expects one argument", name)
				}
				for index := 0; index < v.Len(); index++ {
					if equal(v.Index(index).Interface(), args[0]) {
						return true, nil
					}
				}
				return false, nil
			}), true
		}
	case reflect.Map:
		if name == "length" {
			return v.Len(), true
		}
	}
	return nil, false
}

func normalizeSliceBounds(start, end, length int) (int, int) {
	if start < 0 {
		start = length + start
	}
	if end < 0 {
		end = length + end
	}
	if start < 0 {
		start = 0
	}
	if start > length {
		start = length
	}
	if end < start {
		end = start
	}
	if end > length {
		end = length
	}
	return start, end
}

type callNode struct {
	callee node
	args   []node
}

func (n callNode) eval(ctx *Context) (any, error) {
	callee, err := n.callee.eval(ctx)
	if err != nil {
		return nil, err
	}
	args := make([]any, len(n.args))
	for i, argNode := range n.args {
		args[i], err = argNode.eval(ctx)
		if err != nil {
			return nil, err
		}
	}
	if fn, ok := callee.(Function); ok {
		return fn(args...)
	}
	return callReflect(callee, args)
}

type unaryNode struct {
	op    string
	value node
}

func (n unaryNode) eval(ctx *Context) (any, error) {
	value, err := n.value.eval(ctx)
	if err != nil {
		return nil, err
	}
	switch n.op {
	case "!":
		return !truthy(value), nil
	case "-":
		number, ok := number(value)
		if !ok {
			return nil, fmt.Errorf("namat expression: unary - expects a number, got %T", value)
		}
		return -number, nil
	case "+":
		number, ok := number(value)
		if !ok {
			return nil, fmt.Errorf("namat expression: unary + expects a number, got %T", value)
		}
		return number, nil
	default:
		return nil, fmt.Errorf("namat expression: unsupported unary operator %q", n.op)
	}
}

type binaryNode struct {
	op          string
	left, right node
}

func (n binaryNode) eval(ctx *Context) (any, error) {
	left, err := n.left.eval(ctx)
	if err != nil {
		return nil, err
	}
	switch n.op {
	case "??":
		if !isNil(left) {
			return left, nil
		}
	case "||":
		if truthy(left) {
			return left, nil
		}
	case "&&":
		if !truthy(left) {
			return left, nil
		}
	}
	right, err := n.right.eval(ctx)
	if err != nil {
		return nil, err
	}
	switch n.op {
	case "??", "||", "&&":
		return right, nil
	case "+":
		if l, ok := number(left); ok {
			if r, ok := number(right); ok {
				return l + r, nil
			}
		}
		return stringify(left) + stringify(right), nil
	case "-", "*", "/", "%":
		l, lok := number(left)
		r, rok := number(right)
		if !lok || !rok {
			return nil, fmt.Errorf("namat expression: %s expects numbers", n.op)
		}
		switch n.op {
		case "-":
			return l - r, nil
		case "*":
			return l * r, nil
		case "/":
			if r == 0 {
				return nil, errors.New("namat expression: division by zero")
			}
			return l / r, nil
		default:
			if r == 0 {
				return nil, errors.New("namat expression: modulo by zero")
			}
			return math.Mod(l, r), nil
		}
	case "==", "===":
		return equal(left, right), nil
	case "!=", "!==":
		return !equal(left, right), nil
	case ">", ">=", "<", "<=":
		return compare(n.op, left, right)
	default:
		return nil, fmt.Errorf("namat expression: unsupported operator %q", n.op)
	}
}

type templateNode struct{ text string }

func (n templateNode) eval(ctx *Context) (any, error) {
	var out strings.Builder
	for i := 0; i < len(n.text); {
		start := strings.Index(n.text[i:], "${")
		if start < 0 {
			out.WriteString(n.text[i:])
			break
		}
		start += i
		out.WriteString(n.text[i:start])
		end, err := findTemplateEnd(n.text, start+2)
		if err != nil {
			return nil, err
		}
		program, err := Compile(n.text[start+2 : end])
		if err != nil {
			return nil, err
		}
		value, err := program.Eval(ctx)
		if err != nil {
			return nil, err
		}
		out.WriteString(stringify(value))
		i = end + 1
	}
	return out.String(), nil
}

func findTemplateEnd(text string, start int) (int, error) {
	depth := 0
	quote := byte(0)
	for i := start; i < len(text); i++ {
		ch := text[i]
		if quote != 0 {
			if ch == '\\' {
				i++
				continue
			}
			if ch == quote {
				quote = 0
			}
			continue
		}
		if ch == '\'' || ch == '"' || ch == '`' {
			quote = ch
			continue
		}
		switch ch {
		case '{':
			depth++
		case '}':
			if depth == 0 {
				return i, nil
			}
			depth--
		}
	}
	return 0, errors.New("namat expression: unterminated template interpolation")
}

func lookup(target any, key any) (any, bool) {
	if isNil(target) {
		return nil, false
	}
	v := reflect.ValueOf(target)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Map:
		keyValue := reflect.ValueOf(key)
		if keyValue.IsValid() && keyValue.Type().AssignableTo(v.Type().Key()) {
			value := v.MapIndex(keyValue)
			if value.IsValid() {
				return value.Interface(), true
			}
		}
		if v.Type().Key().Kind() == reflect.String {
			value := v.MapIndex(reflect.ValueOf(stringify(key)).Convert(v.Type().Key()))
			if value.IsValid() {
				return value.Interface(), true
			}
		}
	case reflect.Struct:
		name := stringify(key)
		typeOf := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := typeOf.Field(i)
			jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
			if field.Name == name || strings.EqualFold(field.Name, name) || (jsonName != "" && jsonName != "-" && jsonName == name) {
				value := v.Field(i)
				if value.CanInterface() {
					return value.Interface(), true
				}
			}
		}
	case reflect.Slice, reflect.Array:
		index, ok := integer(key)
		if !ok || index < 0 || index >= v.Len() {
			return nil, false
		}
		return v.Index(index).Interface(), true
	case reflect.String:
		runes := []rune(v.String())
		index, ok := integer(key)
		if !ok || index < 0 || index >= len(runes) {
			return nil, false
		}
		return string(runes[index]), true
	}
	return nil, false
}

func callReflect(callee any, args []any) (any, error) {
	v := reflect.ValueOf(callee)
	if !v.IsValid() || v.Kind() != reflect.Func {
		return nil, fmt.Errorf("namat expression: %T is not callable", callee)
	}
	t := v.Type()
	minimum := t.NumIn()
	if t.IsVariadic() {
		minimum--
	}
	if (!t.IsVariadic() && len(args) != t.NumIn()) || (t.IsVariadic() && len(args) < minimum) {
		return nil, fmt.Errorf("namat expression: function expects %d arguments, got %d", t.NumIn(), len(args))
	}
	values := make([]reflect.Value, len(args))
	for i, arg := range args {
		parameterIndex := i
		if t.IsVariadic() && i >= t.NumIn()-1 {
			parameterIndex = t.NumIn() - 1
		}
		parameter := t.In(parameterIndex)
		if t.IsVariadic() && i >= t.NumIn()-1 {
			parameter = parameter.Elem()
		}
		value := reflect.ValueOf(arg)
		if !value.IsValid() {
			values[i] = reflect.Zero(parameter)
		} else if value.Type().AssignableTo(parameter) {
			values[i] = value
		} else if value.Type().ConvertibleTo(parameter) {
			values[i] = value.Convert(parameter)
		} else {
			return nil, fmt.Errorf("namat expression: argument %d has type %T, expected %s", i+1, arg, parameter)
		}
	}
	results := v.Call(values)
	if len(results) == 0 {
		return nil, nil
	}
	if len(results) > 1 {
		if err, ok := results[len(results)-1].Interface().(error); ok && err != nil {
			return nil, err
		}
	}
	return results[0].Interface(), nil
}

func truthy(value any) bool {
	if isNil(value) {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != ""
	default:
		if number, ok := number(value); ok {
			return number != 0 && !math.IsNaN(number)
		}
		return true
	}
}

func number(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func integer(value any) (int, bool) {
	if n, ok := number(value); ok && n == math.Trunc(n) {
		return int(n), true
	}
	if text, ok := value.(string); ok {
		n, err := strconv.Atoi(text)
		return n, err == nil
	}
	return 0, false
}

func equal(left, right any) bool {
	if l, ok := number(left); ok {
		if r, ok := number(right); ok {
			return l == r
		}
	}
	return reflect.DeepEqual(left, right)
}

func compare(op string, left, right any) (bool, error) {
	if l, ok := number(left); ok {
		if r, ok := number(right); ok {
			switch op {
			case ">":
				return l > r, nil
			case ">=":
				return l >= r, nil
			case "<":
				return l < r, nil
			default:
				return l <= r, nil
			}
		}
	}
	l, r := stringify(left), stringify(right)
	switch op {
	case ">":
		return l > r, nil
	case ">=":
		return l >= r, nil
	case "<":
		return l < r, nil
	case "<=":
		return l <= r, nil
	default:
		return false, fmt.Errorf("namat expression: unsupported comparison %q", op)
	}
}

func stringify(value any) string {
	if isNil(value) {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	default:
		return fmt.Sprint(v)
	}
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}
