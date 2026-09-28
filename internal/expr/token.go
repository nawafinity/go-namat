package expr

import "fmt"

type tokenKind uint8

const (
	tokenEOF tokenKind = iota
	tokenIdentifier
	tokenNumber
	tokenString
	tokenTemplate
	tokenLParen
	tokenRParen
	tokenLBracket
	tokenRBracket
	tokenComma
	tokenDot
	tokenOptionalDot
	tokenOperator
	tokenLBrace
	tokenRBrace
	tokenColon
	tokenQuestion
)

type token struct {
	kind tokenKind
	text string
	pos  int
}

func (t token) String() string {
	return fmt.Sprintf("%q at %d", t.text, t.pos)
}
