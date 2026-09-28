package expr

import (
	"fmt"
	"strconv"
)

// Program is a compiled Namat expression.
type Program struct {
	source string
	root   node
}

// Compile parses an expression once so it can be evaluated repeatedly.
func Compile(source string) (*Program, error) {
	tokens, err := lex(source)
	if err != nil {
		return nil, err
	}
	p := parser{tokens: tokens}
	root, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}
	if tok := p.peek(); tok.kind != tokenEOF {
		return nil, fmt.Errorf("namat expression: unexpected token %s", tok)
	}
	return &Program{source: source, root: root}, nil
}

type parser struct {
	tokens []token
	pos    int
}

func (p *parser) parseExpression(minPrecedence int) (node, error) {
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
	if minPrecedence == 0 && p.peek().kind == tokenQuestion {
		p.next()
		whenTrue, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if colon := p.next(); colon.kind != tokenColon {
			return nil, fmt.Errorf("namat expression: expected : at byte %d", colon.pos)
		}
		whenFalse, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		left = conditionalNode{condition: left, whenTrue: whenTrue, whenFalse: whenFalse}
	}
	return left, nil
}

func (p *parser) parseUnary() (node, error) {
	if tok := p.peek(); tok.kind == tokenOperator && (tok.text == "!" || tok.text == "-" || tok.text == "+") {
		p.next()
		value, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return unaryNode{op: tok.text, value: value}, nil
	}
	return p.parsePostfix()
}

func (p *parser) parsePostfix() (node, error) {
	value, err := p.parsePrimary()
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
			value = memberNode{target: value, key: literalNode{value: name.text}, optional: optional}
		case tokenLBracket:
			p.next()
			index, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			if tok := p.next(); tok.kind != tokenRBracket {
				return nil, fmt.Errorf("namat expression: expected ] at byte %d", tok.pos)
			}
			value = memberNode{target: value, key: index}
		case tokenLParen:
			p.next()
			var args []node
			if p.peek().kind != tokenRParen {
				for {
					arg, err := p.parseExpression(0)
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if p.peek().kind != tokenComma {
						break
					}
					p.next()
				}
			}
			if tok := p.next(); tok.kind != tokenRParen {
				return nil, fmt.Errorf("namat expression: expected ) at byte %d", tok.pos)
			}
			value = callNode{callee: value, args: args}
		default:
			return value, nil
		}
	}
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
		case "null", "nil", "undefined":
			return literalNode{value: nil}, nil
		default:
			return identifierNode{name: tok.text}, nil
		}
	case tokenNumber:
		value, _ := strconv.ParseFloat(tok.text, 64)
		return literalNode{value: value}, nil
	case tokenString:
		return literalNode{value: tok.text}, nil
	case tokenTemplate:
		return templateNode{text: tok.text}, nil
	case tokenLParen:
		value, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		if close := p.next(); close.kind != tokenRParen {
			return nil, fmt.Errorf("namat expression: expected ) at byte %d", close.pos)
		}
		return value, nil
	case tokenLBracket:
		var values []node
		if p.peek().kind != tokenRBracket {
			for {
				value, err := p.parseExpression(0)
				if err != nil {
					return nil, err
				}
				values = append(values, value)
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
				value, err := p.parseExpression(0)
				if err != nil {
					return nil, err
				}
				entries = append(entries, objectEntry{key: key.text, value: value})
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
	case "==", "===", "!=", "!==":
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
