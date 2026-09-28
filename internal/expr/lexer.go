package expr

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

type lexer struct {
	source string
	pos    int
}

func lex(source string) ([]token, error) {
	l := lexer{source: source}
	var tokens []token
	for {
		tok, err := l.next()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, tok)
		if tok.kind == tokenEOF {
			return tokens, nil
		}
	}
}

func (l *lexer) next() (token, error) {
	l.skipSpace()
	if l.pos >= len(l.source) {
		return token{kind: tokenEOF, pos: l.pos}, nil
	}
	start := l.pos
	r, size := utf8.DecodeRuneInString(l.source[l.pos:])
	if isIdentifierStart(r) {
		l.pos += size
		for l.pos < len(l.source) {
			r, size = utf8.DecodeRuneInString(l.source[l.pos:])
			if !isIdentifierContinue(r) {
				break
			}
			l.pos += size
		}
		return token{kind: tokenIdentifier, text: l.source[start:l.pos], pos: start}, nil
	}
	if unicode.IsDigit(r) {
		return l.number()
	}
	switch r {
	case '\'', '"':
		return l.quoted(byte(r), tokenString)
	case '`':
		return l.quoted('`', tokenTemplate)
	case '(':
		l.pos += size
		return token{kind: tokenLParen, text: "(", pos: start}, nil
	case ')':
		l.pos += size
		return token{kind: tokenRParen, text: ")", pos: start}, nil
	case '[':
		l.pos += size
		return token{kind: tokenLBracket, text: "[", pos: start}, nil
	case ']':
		l.pos += size
		return token{kind: tokenRBracket, text: "]", pos: start}, nil
	case ',':
		l.pos += size
		return token{kind: tokenComma, text: ",", pos: start}, nil
	case '{':
		l.pos += size
		return token{kind: tokenLBrace, text: "{", pos: start}, nil
	case '}':
		l.pos += size
		return token{kind: tokenRBrace, text: "}", pos: start}, nil
	case ':':
		l.pos += size
		return token{kind: tokenColon, text: ":", pos: start}, nil
	case '.':
		l.pos += size
		return token{kind: tokenDot, text: ".", pos: start}, nil
	}

	operators := []string{"===", "!==", "?.", "??", "||", "&&", "==", "!=", ">=", "<=", "+", "-", "*", "/", "%", "!", ">", "<"}
	remaining := l.source[l.pos:]
	for _, op := range operators {
		if strings.HasPrefix(remaining, op) {
			l.pos += len(op)
			kind := tokenOperator
			if op == "?." {
				kind = tokenOptionalDot
			}
			return token{kind: kind, text: op, pos: start}, nil
		}
	}
	if r == '?' {
		l.pos += size
		return token{kind: tokenQuestion, text: "?", pos: start}, nil
	}
	return token{}, fmt.Errorf("namat expression: unexpected character %q at byte %d", r, start)
}

func (l *lexer) number() (token, error) {
	start := l.pos
	dot := false
	for l.pos < len(l.source) {
		r, size := utf8.DecodeRuneInString(l.source[l.pos:])
		if unicode.IsDigit(r) {
			l.pos += size
			continue
		}
		if r == '.' && !dot {
			dot = true
			l.pos += size
			continue
		}
		break
	}
	text := l.source[start:l.pos]
	if _, err := strconv.ParseFloat(text, 64); err != nil {
		return token{}, fmt.Errorf("namat expression: invalid number %q at byte %d", text, start)
	}
	return token{kind: tokenNumber, text: text, pos: start}, nil
}

func (l *lexer) quoted(quote byte, kind tokenKind) (token, error) {
	start := l.pos
	l.pos++
	var b strings.Builder
	for l.pos < len(l.source) {
		ch := l.source[l.pos]
		l.pos++
		if ch == quote {
			return token{kind: kind, text: b.String(), pos: start}, nil
		}
		if ch == '\\' {
			if l.pos >= len(l.source) {
				break
			}
			next := l.source[l.pos]
			l.pos++
			switch next {
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			case '\\', '\'', '"', '`':
				b.WriteByte(next)
			default:
				b.WriteByte('\\')
				b.WriteByte(next)
			}
			continue
		}
		b.WriteByte(ch)
	}
	return token{}, fmt.Errorf("namat expression: unterminated string at byte %d", start)
}

func (l *lexer) skipSpace() {
	for l.pos < len(l.source) {
		r, size := utf8.DecodeRuneInString(l.source[l.pos:])
		if !unicode.IsSpace(r) {
			return
		}
		l.pos += size
	}
}

func isIdentifierStart(r rune) bool {
	return r == '_' || r == '$' || unicode.IsLetter(r)
}

func isIdentifierContinue(r rune) bool {
	return isIdentifierStart(r) || unicode.IsDigit(r)
}
