package expr

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/nawafinity/go-namat/internal/value"
)

// Function is a typed host function callable by template expressions.
type Function struct {
	Signature Signature
	Call      func(context.Context, ...any) (any, error)
}

// Context holds immutable root data, lexical variables, registered functions,
// and an evaluation budget. SharedSteps makes the budget cumulative across
// multiple programs in one render.
type Context struct {
	Context     context.Context
	Root        any
	Variables   map[string]any
	Functions   map[string]Function
	MaxSteps    int
	SharedSteps *int
	steps       int
}

func (c *Context) step() error {
	if c == nil {
		return nil
	}
	steps := &c.steps
	if c.SharedSteps != nil {
		steps = c.SharedSteps
	}
	*steps++
	if c.MaxSteps > 0 && *steps > c.MaxSteps {
		return fmt.Errorf("namat expression: evaluation step limit exceeded (%d)", c.MaxSteps)
	}
	if c.Context != nil {
		return c.Context.Err()
	}
	return nil
}

func (p *Program) Eval(ctx *Context) (any, error) {
	if p == nil || p.root == nil {
		return nil, errors.New("namat expression: empty program")
	}
	if ctx == nil {
		ctx = &Context{}
	}
	if ctx.SharedSteps == nil {
		ctx.steps = 0
	}
	return p.root.eval(ctx)
}

type node interface {
	eval(*Context) (any, error)
}

type literalNode struct{ value any }

func (n literalNode) eval(ctx *Context) (any, error) {
	if err := ctx.step(); err != nil {
		return nil, err
	}
	return n.value, nil
}

type arrayNode struct{ values []node }

func (n arrayNode) eval(ctx *Context) (any, error) {
	if err := ctx.step(); err != nil {
		return nil, err
	}
	items := make([]any, len(n.values))
	for index, item := range n.values {
		resolved, err := item.eval(ctx)
		if err != nil {
			return nil, err
		}
		items[index] = resolved
	}
	return items, nil
}

type objectEntry struct {
	key   string
	value node
}

type objectNode struct{ entries []objectEntry }

func (n objectNode) eval(ctx *Context) (any, error) {
	if err := ctx.step(); err != nil {
		return nil, err
	}
	result := make(map[string]any, len(n.entries))
	for _, entry := range n.entries {
		resolved, err := entry.value.eval(ctx)
		if err != nil {
			return nil, err
		}
		result[entry.key] = resolved
	}
	return result, nil
}

type identifierNode struct {
	name string
	pos  int
}

func (n identifierNode) eval(ctx *Context) (any, error) {
	if err := ctx.step(); err != nil {
		return nil, err
	}
	if ctx.Variables != nil {
		if resolved, ok := ctx.Variables[n.name]; ok {
			return resolved, nil
		}
	}
	if resolved, ok := lookup(ctx.Root, n.name); ok {
		return resolved, nil
	}
	candidates := append(keysOf(ctx.Root), mapKeys(ctx.Variables)...)
	return nil, unknownNameError("identifier", n.name, candidates)
}

type memberNode struct {
	target   node
	key      node
	optional bool
}

func (n memberNode) eval(ctx *Context) (any, error) {
	if err := ctx.step(); err != nil {
		return nil, err
	}
	target, err := n.target.eval(ctx)
	if err != nil {
		if n.optional {
			return value.MissingValue{}, nil
		}
		return nil, err
	}
	if isMissing(target) || isNil(target) {
		if n.optional {
			return value.MissingValue{}, nil
		}
		return nil, errors.New("namat expression: cannot access a property of missing or null")
	}
	key, err := n.key.eval(ctx)
	if err != nil {
		return nil, err
	}
	resolved, ok := lookup(target, key)
	if ok {
		return resolved, nil
	}
	if n.optional {
		return value.MissingValue{}, nil
	}
	name := fmt.Sprint(key)
	return nil, unknownNameError("property", name, keysOf(target))
}

type callNode struct {
	name string
	args []node
	pos  int
}

func (n callNode) eval(ctx *Context) (any, error) {
	if err := ctx.step(); err != nil {
		return nil, err
	}
	function, ok := ctx.Functions[n.name]
	if !ok || function.Call == nil {
		return nil, unknownNameError("function", n.name, functionNames(ctx.Functions))
	}
	arguments := make([]any, len(n.args))
	for index, argument := range n.args {
		resolved, err := argument.eval(ctx)
		if err != nil {
			return nil, err
		}
		arguments[index] = resolved
	}
	if err := validateRuntimeArguments(n.name, arguments, function.Signature); err != nil {
		return nil, err
	}
	arguments = normalizeRuntimeArguments(arguments, function.Signature)
	callContext := ctx.Context
	if callContext == nil {
		callContext = context.Background()
	}
	result, err := function.Call(callContext, arguments...)
	if err != nil {
		return nil, fmt.Errorf("namat expression: function %s: %w", n.name, err)
	}
	if expected := function.Signature.Returns; expected != "" && expected != value.Any && expected != value.Rich {
		if actual := KindOf(result); actual != expected {
			return nil, fmt.Errorf("namat expression: function %q declared %s result but returned %s", n.name, expected, actual)
		}
	}
	if err := ctx.step(); err != nil {
		return nil, err
	}
	return result, nil
}

type unaryNode struct {
	op    string
	value node
}

func (n unaryNode) eval(ctx *Context) (any, error) {
	if err := ctx.step(); err != nil {
		return nil, err
	}
	resolved, err := n.value.eval(ctx)
	if err != nil {
		return nil, err
	}
	switch n.op {
	case "!":
		if KindOf(resolved) != value.Bool {
			return nil, fmt.Errorf("namat expression: ! expects bool, got %s", kindName(resolved))
		}
		return !boolOf(resolved), nil
	case "+":
		if !isNumeric(resolved) {
			return nil, fmt.Errorf("namat expression: unary + expects a number, got %s", kindName(resolved))
		}
		return resolved, nil
	case "-":
		return negate(resolved)
	default:
		return nil, fmt.Errorf("namat expression: unsupported unary operator %q", n.op)
	}
}

type binaryNode struct {
	op          string
	left, right node
}

func (n binaryNode) eval(ctx *Context) (any, error) {
	if err := ctx.step(); err != nil {
		return nil, err
	}
	left, err := n.left.eval(ctx)
	if err != nil {
		return nil, err
	}
	switch n.op {
	case "??":
		if !isMissing(left) && !isNil(left) {
			return left, nil
		}
	case "||", "&&":
		if KindOf(left) != value.Bool {
			return nil, fmt.Errorf("namat expression: %s expects bool operands, got %s", n.op, kindName(left))
		}
		boolean := boolOf(left)
		if n.op == "||" && boolean {
			return true, nil
		}
		if n.op == "&&" && !boolean {
			return false, nil
		}
	}
	right, err := n.right.eval(ctx)
	if err != nil {
		return nil, err
	}
	switch n.op {
	case "??":
		return right, nil
	case "||", "&&":
		if KindOf(right) != value.Bool {
			return nil, fmt.Errorf("namat expression: %s expects bool operands, got %s", n.op, kindName(right))
		}
		return boolOf(right), nil
	case "+":
		if KindOf(left) == value.String {
			if KindOf(right) != value.String {
				return nil, fmt.Errorf("namat expression: + cannot combine string and %s", kindName(right))
			}
			return stringOf(left) + stringOf(right), nil
		}
		return numericOperation(n.op, left, right)
	case "-", "*", "/", "%":
		return numericOperation(n.op, left, right)
	case "==", "!=":
		equal, err := equalValues(left, right)
		if err != nil {
			return nil, err
		}
		if n.op == "!=" {
			return !equal, nil
		}
		return equal, nil
	case ">", ">=", "<", "<=":
		return compareValues(n.op, left, right)
	default:
		return nil, fmt.Errorf("namat expression: unsupported operator %q", n.op)
	}
}

func validateRuntimeArguments(name string, arguments []any, signature Signature) error {
	if (!signature.Variadic && len(arguments) != len(signature.Params)) || (signature.Variadic && len(arguments) < len(signature.Params)) {
		return fmt.Errorf("namat expression: function %q expects %d arguments, got %d", name, len(signature.Params), len(arguments))
	}
	for index, expected := range signature.Params {
		if expected == value.Any {
			continue
		}
		actual := KindOf(arguments[index])
		if actual != expected {
			return fmt.Errorf("namat expression: function %q argument %d expects %s, got %s", name, index+1, expected, actual)
		}
	}
	return nil
}

func normalizeRuntimeArguments(arguments []any, signature Signature) []any {
	for index, expected := range signature.Params {
		if index >= len(arguments) || expected == value.Any || expected == value.Rich {
			continue
		}
		switch expected {
		case value.Bool:
			arguments[index] = boolOf(arguments[index])
		case value.String:
			arguments[index] = stringOf(arguments[index])
		case value.Int:
			arguments[index] = int64Of(arguments[index])
		case value.Uint:
			arguments[index] = uint64Of(arguments[index])
		case value.Float:
			arguments[index] = float64Of(arguments[index])
		case value.Decimal:
			arguments[index], _ = exactDecimal(arguments[index])
		}
	}
	return arguments
}

func KindOf(input any) value.Kind {
	if isMissing(input) {
		return value.Missing
	}
	if isNil(input) {
		return value.Null
	}
	reflected := reflect.ValueOf(input)
	for reflected.IsValid() && (reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Interface) {
		if reflected.IsNil() {
			return value.Null
		}
		reflected = reflected.Elem()
	}
	if !reflected.IsValid() {
		return value.Null
	}
	if reflected.CanInterface() {
		if _, ok := reflected.Interface().(value.DecimalValue); ok {
			return value.Decimal
		}
	}
	switch reflected.Kind() {
	case reflect.Bool:
		return value.Bool
	case reflect.String:
		return value.String
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Uint
	case reflect.Float32, reflect.Float64:
		return value.Float
	case reflect.Array, reflect.Slice:
		return value.List
	case reflect.Map, reflect.Struct:
		return value.Object
	default:
		return value.Any
	}
}

func AsBool(input any) (bool, bool) {
	if KindOf(input) != value.Bool {
		return false, false
	}
	return boolOf(input), true
}

func kindName(input any) value.Kind { return KindOf(input) }

func isNumeric(input any) bool {
	kind := KindOf(input)
	return kind == value.Int || kind == value.Uint || kind == value.Decimal || kind == value.Float
}

func numericOperation(operator string, left, right any) (any, error) {
	leftKind, rightKind := KindOf(left), KindOf(right)
	if !isNumeric(left) || !isNumeric(right) {
		return nil, fmt.Errorf("namat expression: %s expects numeric operands, got %s and %s", operator, leftKind, rightKind)
	}
	if leftKind == value.Float || rightKind == value.Float {
		if leftKind != value.Float || rightKind != value.Float {
			return nil, fmt.Errorf("namat expression: exact and floating-point numbers cannot be mixed")
		}
		l, r := float64Of(left), float64Of(right)
		switch operator {
		case "+":
			return l + r, nil
		case "-":
			return l - r, nil
		case "*":
			return l * r, nil
		case "/":
			if r == 0 {
				return nil, errors.New("namat expression: division by zero")
			}
			return l / r, nil
		case "%":
			return nil, errors.New("namat expression: modulo does not accept floating-point operands")
		}
	}
	if leftKind == value.Decimal || rightKind == value.Decimal || operator == "/" {
		leftDecimal, err := exactDecimal(left)
		if err != nil {
			return nil, err
		}
		rightDecimal, err := exactDecimal(right)
		if err != nil {
			return nil, err
		}
		switch operator {
		case "+":
			return leftDecimal.Add(rightDecimal), nil
		case "-":
			return leftDecimal.Sub(rightDecimal), nil
		case "*":
			return leftDecimal.Mul(rightDecimal), nil
		case "/":
			result, err := leftDecimal.Quo(rightDecimal)
			if err != nil {
				return nil, fmt.Errorf("namat expression: %w", err)
			}
			return result, nil
		case "%":
			return nil, errors.New("namat expression: modulo accepts integer operands only")
		}
	}
	if leftKind != rightKind {
		return nil, fmt.Errorf("namat expression: signed and unsigned integers cannot be mixed")
	}
	if leftKind == value.Int {
		l, r := int64Of(left), int64Of(right)
		switch operator {
		case "+":
			if (r > 0 && l > math.MaxInt64-r) || (r < 0 && l < math.MinInt64-r) {
				return nil, errors.New("namat expression: integer overflow")
			}
			return l + r, nil
		case "-":
			if (r < 0 && l > math.MaxInt64+r) || (r > 0 && l < math.MinInt64+r) {
				return nil, errors.New("namat expression: integer overflow")
			}
			return l - r, nil
		case "*":
			if l != 0 && (l == math.MinInt64 && r == -1 || r != 0 && (l*r)/r != l) {
				return nil, errors.New("namat expression: integer overflow")
			}
			return l * r, nil
		case "%":
			if r == 0 {
				return nil, errors.New("namat expression: modulo by zero")
			}
			return l % r, nil
		}
	}
	l, r := uint64Of(left), uint64Of(right)
	switch operator {
	case "+":
		if math.MaxUint64-l < r {
			return nil, errors.New("namat expression: unsigned integer overflow")
		}
		return l + r, nil
	case "-":
		if l < r {
			return nil, errors.New("namat expression: unsigned integer underflow")
		}
		return l - r, nil
	case "*":
		if r != 0 && l > math.MaxUint64/r {
			return nil, errors.New("namat expression: unsigned integer overflow")
		}
		return l * r, nil
	case "%":
		if r == 0 {
			return nil, errors.New("namat expression: modulo by zero")
		}
		return l % r, nil
	}
	return nil, fmt.Errorf("namat expression: unsupported numeric operator %q", operator)
}

func negate(input any) (any, error) {
	switch KindOf(input) {
	case value.Int:
		integer := int64Of(input)
		if integer == math.MinInt64 {
			return nil, errors.New("namat expression: integer overflow")
		}
		return -integer, nil
	case value.Decimal:
		return indirect(reflect.ValueOf(input)).Interface().(value.DecimalValue).Neg(), nil
	case value.Float:
		return -float64Of(input), nil
	default:
		return nil, fmt.Errorf("namat expression: unary - expects a signed number, got %s", kindName(input))
	}
}

func equalValues(left, right any) (bool, error) {
	leftKind, rightKind := KindOf(left), KindOf(right)
	if leftKind == value.Missing || rightKind == value.Missing {
		if leftKind != rightKind {
			return false, fmt.Errorf("namat expression: cannot compare %s and %s", leftKind, rightKind)
		}
		return true, nil
	}
	if leftKind == value.Null || rightKind == value.Null {
		if leftKind != rightKind {
			return false, fmt.Errorf("namat expression: cannot compare %s and %s", leftKind, rightKind)
		}
		return true, nil
	}
	if isExactNumberKind(leftKind) && isExactNumberKind(rightKind) {
		leftDecimal, _ := exactDecimal(left)
		rightDecimal, _ := exactDecimal(right)
		return leftDecimal.Cmp(rightDecimal) == 0, nil
	}
	if leftKind != rightKind {
		return false, fmt.Errorf("namat expression: cannot compare %s and %s", leftKind, rightKind)
	}
	if leftKind == value.Float {
		return float64Of(left) == float64Of(right), nil
	}
	if leftKind == value.String {
		return stringOf(left) == stringOf(right), nil
	}
	if leftKind == value.Bool {
		return boolOf(left) == boolOf(right), nil
	}
	leftValue, rightValue := indirect(reflect.ValueOf(left)), indirect(reflect.ValueOf(right))
	return reflect.DeepEqual(leftValue.Interface(), rightValue.Interface()), nil
}

func compareValues(operator string, left, right any) (bool, error) {
	leftKind, rightKind := KindOf(left), KindOf(right)
	comparison := 0
	if isExactNumberKind(leftKind) && isExactNumberKind(rightKind) {
		leftDecimal, _ := exactDecimal(left)
		rightDecimal, _ := exactDecimal(right)
		comparison = leftDecimal.Cmp(rightDecimal)
	} else if leftKind == value.Float && rightKind == value.Float {
		l, r := float64Of(left), float64Of(right)
		if math.IsNaN(l) || math.IsNaN(r) {
			return false, errors.New("namat expression: NaN is not orderable")
		}
		if l < r {
			comparison = -1
		} else if l > r {
			comparison = 1
		}
	} else if leftKind == value.String {
		if rightKind != value.String {
			return false, fmt.Errorf("namat expression: cannot order %s and %s", leftKind, rightKind)
		}
		comparison = strings.Compare(stringOf(left), stringOf(right))
	} else {
		return false, fmt.Errorf("namat expression: cannot order %s and %s", leftKind, rightKind)
	}
	switch operator {
	case ">":
		return comparison > 0, nil
	case ">=":
		return comparison >= 0, nil
	case "<":
		return comparison < 0, nil
	case "<=":
		return comparison <= 0, nil
	default:
		return false, fmt.Errorf("namat expression: unsupported comparison %q", operator)
	}
}

func isExactNumberKind(kind value.Kind) bool {
	return kind == value.Int || kind == value.Uint || kind == value.Decimal
}

func exactDecimal(input any) (value.DecimalValue, error) {
	switch KindOf(input) {
	case value.Int:
		return value.DecimalFromInt(int64Of(input)), nil
	case value.Uint:
		return value.DecimalFromUint(uint64Of(input)), nil
	case value.Decimal:
		return indirect(reflect.ValueOf(input)).Interface().(value.DecimalValue), nil
	default:
		return value.DecimalValue{}, fmt.Errorf("namat expression: %s is not an exact number", kindName(input))
	}
}

func int64Of(input any) int64 {
	reflected := indirect(reflect.ValueOf(input))
	return reflected.Int()
}

func uint64Of(input any) uint64 {
	reflected := indirect(reflect.ValueOf(input))
	return reflected.Uint()
}

func float64Of(input any) float64 {
	reflected := indirect(reflect.ValueOf(input))
	return reflected.Convert(reflect.TypeOf(float64(0))).Float()
}

func boolOf(input any) bool { return indirect(reflect.ValueOf(input)).Bool() }

func stringOf(input any) string { return indirect(reflect.ValueOf(input)).String() }

func indirect(reflected reflect.Value) reflect.Value {
	for reflected.IsValid() && (reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Interface) {
		reflected = reflected.Elem()
	}
	return reflected
}

func lookup(target any, key any) (any, bool) {
	if isNil(target) || isMissing(target) {
		return nil, false
	}
	reflected := reflect.ValueOf(target)
	for reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Interface {
		if reflected.IsNil() {
			return nil, false
		}
		reflected = reflected.Elem()
	}
	switch reflected.Kind() {
	case reflect.Map:
		keyValue := reflect.ValueOf(key)
		if keyValue.IsValid() && keyValue.Type().AssignableTo(reflected.Type().Key()) {
			resolved := reflected.MapIndex(keyValue)
			if resolved.IsValid() {
				return resolved.Interface(), true
			}
		}
		if reflected.Type().Key().Kind() == reflect.String {
			name, ok := key.(string)
			if !ok {
				return nil, false
			}
			resolved := reflected.MapIndex(reflect.ValueOf(name).Convert(reflected.Type().Key()))
			if resolved.IsValid() {
				return resolved.Interface(), true
			}
		}
	case reflect.Struct:
		name, ok := key.(string)
		if !ok {
			return nil, false
		}
		typeOf := reflected.Type()
		for index := 0; index < reflected.NumField(); index++ {
			field := typeOf.Field(index)
			jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
			lookupName := field.Name
			if jsonName != "" && jsonName != "-" {
				lookupName = jsonName
			}
			if lookupName == name && reflected.Field(index).CanInterface() {
				return reflected.Field(index).Interface(), true
			}
		}
	case reflect.Array, reflect.Slice:
		index, ok := indexValue(key)
		if !ok || index < 0 || index >= reflected.Len() {
			return nil, false
		}
		return reflected.Index(index).Interface(), true
	case reflect.String:
		index, ok := indexValue(key)
		runes := []rune(reflected.String())
		if !ok || index < 0 || index >= len(runes) {
			return nil, false
		}
		return string(runes[index]), true
	}
	return nil, false
}

func indexValue(input any) (int, bool) {
	switch KindOf(input) {
	case value.Int:
		integer := int64Of(input)
		if integer < 0 || integer > int64(math.MaxInt) {
			return 0, false
		}
		return int(integer), true
	case value.Uint:
		unsigned := uint64Of(input)
		if unsigned > uint64(math.MaxInt) {
			return 0, false
		}
		return int(unsigned), true
	default:
		return 0, false
	}
}

func isMissing(input any) bool {
	_, ok := input.(value.MissingValue)
	return ok
}

func isNil(input any) bool {
	if input == nil {
		return true
	}
	reflected := reflect.ValueOf(input)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

func keysOf(input any) []string {
	if isNil(input) || isMissing(input) {
		return nil
	}
	reflected := reflect.ValueOf(input)
	for reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Interface {
		if reflected.IsNil() {
			return nil
		}
		reflected = reflected.Elem()
	}
	var result []string
	switch reflected.Kind() {
	case reflect.Map:
		if reflected.Type().Key().Kind() == reflect.String {
			iterator := reflected.MapRange()
			for iterator.Next() {
				result = append(result, iterator.Key().String())
			}
		}
	case reflect.Struct:
		typeOf := reflected.Type()
		for index := 0; index < reflected.NumField(); index++ {
			field := typeOf.Field(index)
			if !reflected.Field(index).CanInterface() {
				continue
			}
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			if name == "" {
				name = field.Name
			}
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

func mapKeys(input map[string]any) []string {
	result := make([]string, 0, len(input))
	for key := range input {
		result = append(result, key)
	}
	return result
}

func functionNames(input map[string]Function) []string {
	result := make([]string, 0, len(input))
	for name := range input {
		result = append(result, name)
	}
	return result
}

func unknownNameError(kind, name string, candidates []string) error {
	message := fmt.Sprintf("namat expression: unknown %s %q", kind, name)
	if suggestion := closestName(name, candidates); suggestion != "" {
		message += fmt.Sprintf("; did you mean %q?", suggestion)
	}
	return errors.New(message)
}

func closestName(name string, candidates []string) string {
	best, bestDistance := "", 3
	for _, candidate := range candidates {
		distance := editDistance(name, candidate)
		if distance < bestDistance {
			best, bestDistance = candidate, distance
		}
	}
	return best
}

func editDistance(left, right string) int {
	a, b := []rune(left), []rune(right)
	previous := make([]int, len(b)+1)
	for index := range previous {
		previous[index] = index
	}
	for i, leftRune := range a {
		current := make([]int, len(b)+1)
		current[0] = i + 1
		for j, rightRune := range b {
			cost := 0
			if leftRune != rightRune {
				cost = 1
			}
			current[j+1] = min(current[j]+1, previous[j+1]+1, previous[j]+cost)
		}
		previous = current
	}
	return previous[len(b)]
}

func Format(input any) (string, error) {
	if isMissing(input) {
		return "", errors.New("namat expression: missing value cannot be rendered; use default")
	}
	if isNil(input) {
		return "", nil
	}
	reflected := indirect(reflect.ValueOf(input))
	if reflected.IsValid() && reflected.CanInterface() {
		input = reflected.Interface()
	}
	switch typed := input.(type) {
	case string:
		return typed, nil
	case value.DecimalValue:
		return typed.String(), nil
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), nil
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32), nil
	default:
		return fmt.Sprint(typed), nil
	}
}
