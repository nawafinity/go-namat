package expr

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nawafinity/go-namat/internal/value"
)

// Signature describes a callable expression function.
type Signature struct {
	Params      []value.Kind
	Variadic    bool
	Returns     value.Kind
	Description string
	Pure        bool
}

// CompileOptions bounds parsing and optionally validates function calls.
type CompileOptions struct {
	MaxBytes  int
	MaxTokens int
	MaxDepth  int
	Functions map[string]Signature
}

// Program is an immutable compiled Namat expression.
type Program struct {
	source string
	root   node
}

func Compile(source string) (*Program, error) {
	return CompileWithOptions(source, CompileOptions{})
}

func CompileWithOptions(source string, options CompileOptions) (*Program, error) {
	if options.MaxBytes > 0 && len(source) > options.MaxBytes {
		return nil, fmt.Errorf("namat expression: byte limit exceeded (%d)", options.MaxBytes)
	}
	tokens, err := lex(source, options.MaxTokens)
	if err != nil {
		return nil, err
	}
	p := parser{tokens: tokens, maxDepth: options.MaxDepth}
	root, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if tok := p.peek(); tok.kind != tokenEOF {
		return nil, fmt.Errorf("namat expression: unexpected token %s", tok)
	}
	if options.Functions != nil {
		if err := validateCalls(root, options.Functions); err != nil {
			return nil, err
		}
	}
	return &Program{source: source, root: root}, nil
}

type parser struct {
	tokens   []token
	pos      int
	depth    int
	maxDepth int
}

func (p *parser) enter() error {
	p.depth++
	if p.maxDepth > 0 && p.depth > p.maxDepth {
		return fmt.Errorf("namat expression: AST depth limit exceeded (%d)", p.maxDepth)
	}
	return nil
}

func (p *parser) leave() { p.depth-- }

func (p *parser) parseExpression(minPrecedence int) (node, error) {
	if err := p.enter(); err != nil {
		return nil, err
	}
	defer p.leave()

	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		tok := p.peek()
		if tok.kind != tokenOperator {
			break
		}
		precedence, ok := binaryPrecedence(tok.text)
		if !ok || precedence < minPrecedence {
			break
		}
		p.next()
		right, err := p.parseExpression(precedence + 1)
		if err != nil {
			return nil, err
		}
		left = binaryNode{op: tok.text, left: left, right: right}
	}
	if minPrecedence == 0 {
		for p.peek().kind == tokenPipe {
			p.next()
			name := p.next()
			if name.kind != tokenIdentifier {
				return nil, fmt.Errorf("namat expression: expected filter name at byte %d", name.pos)
			}
			args := []node{left}
			if p.peek().kind == tokenLParen {
				parsed, err := p.parseArguments()
				if err != nil {
					return nil, err
				}
				args = append(args, parsed...)
			}
			left = callNode{name: name.text, args: args, pos: name.pos}
		}
	}
	return left, nil
}

func (p *parser) parseUnary() (node, error) {
	if tok := p.peek(); tok.kind == tokenOperator && (tok.text == "!" || tok.text == "-" || tok.text == "+") {
		p.next()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return unaryNode{op: tok.text, value: operand}, nil
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() (node, error) {
	result, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		switch p.peek().kind {
		case tokenDot, tokenOptionalDot:
			optional := p.next().kind == tokenOptionalDot
			name := p.next()
			if name.kind != tokenIdentifier {
				return nil, fmt.Errorf("namat expression: expected property name at byte %d", name.pos)
			}
			result = memberNode{target: result, key: literalNode{value: name.text}, optional: optional}
		case tokenLBracket:
			p.next()
			index, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			if tok := p.next(); tok.kind != tokenRBracket {
				return nil, fmt.Errorf("namat expression: expected ] at byte %d", tok.pos)
			}
			result = memberNode{target: result, key: index}
		case tokenLParen:
			identifier, ok := result.(identifierNode)
			if !ok {
				return nil, fmt.Errorf("namat expression: methods and dynamic calls are not supported at byte %d", p.peek().pos)
			}
			args, err := p.parseArguments()
			if err != nil {
				return nil, err
			}
			result = callNode{name: identifier.name, args: args, pos: identifier.pos}
		default:
			return result, nil
		}
	}
}

func (p *parser) parseArguments() ([]node, error) {
	open := p.next()
	if open.kind != tokenLParen {
		return nil, fmt.Errorf("namat expression: expected ( at byte %d", open.pos)
	}
	var args []node
	if p.peek().kind != tokenRParen {
		for {
			argument, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			args = append(args, argument)
			if p.peek().kind != tokenComma {
				break
			}
			p.next()
		}
	}
	if close := p.next(); close.kind != tokenRParen {
		return nil, fmt.Errorf("namat expression: expected ) at byte %d", close.pos)
	}
	return args, nil
}

func (p *parser) parsePrimary() (node, error) {
	tok := p.next()
	switch tok.kind {
	case tokenIdentifier:
		switch tok.text {
		case "true":
			return literalNode{value: true}, nil
		case "false":
			return literalNode{value: false}, nil
		case "null":
			return literalNode{value: nil}, nil
		case "nil", "undefined":
			return nil, fmt.Errorf("namat expression: %q is not a v1 literal; use null at byte %d", tok.text, tok.pos)
		default:
			return identifierNode{name: tok.text, pos: tok.pos}, nil
		}
	case tokenNumber:
		if strings.Contains(tok.text, ".") {
			decimal, err := value.ParseDecimal(tok.text)
			if err != nil {
				return nil, fmt.Errorf("namat expression: %w at byte %d", err, tok.pos)
			}
			return literalNode{value: decimal}, nil
		}
		integer, err := strconv.ParseInt(tok.text, 10, 64)
		if err == nil {
			return literalNode{value: integer}, nil
		}
		unsigned, unsignedErr := strconv.ParseUint(tok.text, 10, 64)
		if unsignedErr != nil {
			return nil, fmt.Errorf("namat expression: integer out of range at byte %d", tok.pos)
		}
		return literalNode{value: unsigned}, nil
	case tokenString:
		return literalNode{value: tok.text}, nil
	case tokenLParen:
		result, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if close := p.next(); close.kind != tokenRParen {
			return nil, fmt.Errorf("namat expression: expected ) at byte %d", close.pos)
		}
		return result, nil
	case tokenLBracket:
		var values []node
		if p.peek().kind != tokenRBracket {
			for {
				item, err := p.parseExpression(0)
				if err != nil {
					return nil, err
				}
				values = append(values, item)
				if p.peek().kind != tokenComma {
					break
				}
				p.next()
			}
		}
		if close := p.next(); close.kind != tokenRBracket {
			return nil, fmt.Errorf("namat expression: expected ] at byte %d", close.pos)
		}
		return arrayNode{values: values}, nil
	case tokenLBrace:
		entries := make([]objectEntry, 0)
		if p.peek().kind != tokenRBrace {
			for {
				key := p.next()
				if key.kind != tokenIdentifier && key.kind != tokenString {
					return nil, fmt.Errorf("namat expression: expected object key at byte %d", key.pos)
				}
				if colon := p.next(); colon.kind != tokenColon {
					return nil, fmt.Errorf("namat expression: expected : at byte %d", colon.pos)
				}
				item, err := p.parseExpression(0)
				if err != nil {
					return nil, err
				}
				entries = append(entries, objectEntry{key: key.text, value: item})
				if p.peek().kind != tokenComma {
					break
				}
				p.next()
			}
		}
		if close := p.next(); close.kind != tokenRBrace {
			return nil, fmt.Errorf("namat expression: expected } at byte %d", close.pos)
		}
		return objectNode{entries: entries}, nil
	default:
		return nil, fmt.Errorf("namat expression: expected value at byte %d", tok.pos)
	}
}

func (p *parser) peek() token {
	if p.pos >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) next() token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func binaryPrecedence(op string) (int, bool) {
	switch op {
	case "??":
		return 1, true
	case "||":
		return 2, true
	case "&&":
		return 3, true
	case "==", "!=":
		return 4, true
	case ">", ">=", "<", "<=":
		return 5, true
	case "+", "-":
		return 6, true
	case "*", "/", "%":
		return 7, true
	default:
		return 0, false
	}
}

func validateCalls(root node, functions map[string]Signature) error {
	var walk func(node) error
	walk = func(current node) error {
		switch typed := current.(type) {
		case arrayNode:
			for _, item := range typed.values {
				if err := walk(item); err != nil {
					return err
				}
			}
		case objectNode:
			for _, entry := range typed.entries {
				if err := walk(entry.value); err != nil {
					return err
				}
			}
		case memberNode:
			if err := walk(typed.target); err != nil {
				return err
			}
			return walk(typed.key)
		case unaryNode:
			return walk(typed.value)
		case binaryNode:
			if err := walk(typed.left); err != nil {
				return err
			}
			return walk(typed.right)
		case callNode:
			signature, ok := functions[typed.name]
			if !ok {
				return fmt.Errorf("namat expression: unknown function %q at byte %d", typed.name, typed.pos)
			}
			if (!signature.Variadic && len(typed.args) != len(signature.Params)) || (signature.Variadic && len(typed.args) < len(signature.Params)) {
				return fmt.Errorf("namat expression: function %q expects %d arguments, got %d", typed.name, len(signature.Params), len(typed.args))
			}
			for _, argument := range typed.args {
				if err := walk(argument); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(root)
}
