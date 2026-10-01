package value

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

// Kind is a stable type name used by expression and host-function signatures.
type Kind string

const (
	Any     Kind = "any"
	Missing Kind = "missing"
	Null    Kind = "null"
	Bool    Kind = "bool"
	String  Kind = "string"
	Int     Kind = "int"
	Uint    Kind = "uint"
	Decimal Kind = "decimal"
	Float   Kind = "float"
	List    Kind = "list"
	Object  Kind = "object"
	Rich    Kind = "rich"
)

// MissingValue represents a field that does not exist. It is deliberately
// distinct from an explicitly present null value.
type MissingValue struct{}

func (MissingValue) String() string { return "missing" }

// DecimalValue stores an exact rational value parsed from base-10 input.
// Arithmetic is exact; rendering uses a bounded decimal representation only
// when a rational has no finite base-10 expansion.
type DecimalValue struct {
	rat *big.Rat
}

func ParseDecimal(source string) (DecimalValue, error) {
	text := strings.TrimSpace(source)
	if text == "" {
		return DecimalValue{}, fmt.Errorf("empty decimal")
	}
	base, exponentText, hasExponent := strings.Cut(text, "e")
	if !hasExponent {
		base, exponentText, hasExponent = strings.Cut(text, "E")
	}
	rat, ok := new(big.Rat).SetString(base)
	if !ok {
		return DecimalValue{}, fmt.Errorf("invalid decimal %q", source)
	}
	if hasExponent {
		exponent, err := strconv.Atoi(exponentText)
		if err != nil || exponent < -10_000 || exponent > 10_000 {
			return DecimalValue{}, fmt.Errorf("invalid decimal exponent in %q", source)
		}
		power := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs(exponent))), nil)
		factor := new(big.Rat).SetInt(power)
		if exponent < 0 {
			rat.Quo(rat, factor)
		} else {
			rat.Mul(rat, factor)
		}
	}
	return DecimalValue{rat: rat}, nil
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func DecimalFromInt(value int64) DecimalValue {
	return DecimalValue{rat: new(big.Rat).SetInt64(value)}
}

func DecimalFromUint(value uint64) DecimalValue {
	integer := new(big.Int).SetUint64(value)
	return DecimalValue{rat: new(big.Rat).SetInt(integer)}
}

func (d DecimalValue) Rat() *big.Rat {
	if d.rat == nil {
		return new(big.Rat)
	}
	return new(big.Rat).Set(d.rat)
}

func (d DecimalValue) String() string {
	rat := d.Rat()
	denominator := new(big.Int).Set(rat.Denom())
	two, five := big.NewInt(2), big.NewInt(5)
	scale := 0
	for new(big.Int).Mod(denominator, two).Sign() == 0 {
		denominator.Div(denominator, two)
		scale++
	}
	for new(big.Int).Mod(denominator, five).Sign() == 0 {
		denominator.Div(denominator, five)
		scale++
	}
	if denominator.Cmp(big.NewInt(1)) == 0 {
		text := rat.FloatString(scale)
		if strings.Contains(text, ".") {
			text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
		}
		return text
	}
	text := strings.TrimRight(strings.TrimRight(rat.FloatString(18), "0"), ".")
	if text == "-0" || text == "" {
		return "0"
	}
	return text
}

func (d DecimalValue) Add(other DecimalValue) DecimalValue {
	return DecimalValue{rat: new(big.Rat).Add(d.Rat(), other.Rat())}
}

func (d DecimalValue) Sub(other DecimalValue) DecimalValue {
	return DecimalValue{rat: new(big.Rat).Sub(d.Rat(), other.Rat())}
}

func (d DecimalValue) Mul(other DecimalValue) DecimalValue {
	return DecimalValue{rat: new(big.Rat).Mul(d.Rat(), other.Rat())}
}

func (d DecimalValue) Quo(other DecimalValue) (DecimalValue, error) {
	if other.Rat().Sign() == 0 {
		return DecimalValue{}, fmt.Errorf("division by zero")
	}
	return DecimalValue{rat: new(big.Rat).Quo(d.Rat(), other.Rat())}, nil
}

func (d DecimalValue) Neg() DecimalValue {
	return DecimalValue{rat: new(big.Rat).Neg(d.Rat())}
}

func (d DecimalValue) Cmp(other DecimalValue) int { return d.Rat().Cmp(other.Rat()) }
