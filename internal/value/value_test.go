package value

import "testing"

func TestDecimalExactArithmeticAndFormatting(t *testing.T) {
	a, err := ParseDecimal("0.10")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := ParseDecimal("0.20")
	if got := a.Add(b).String(); got != "0.3" {
		t.Fatalf("sum = %q", got)
	}
	if got := b.Sub(a).String(); got != "0.1" {
		t.Fatalf("difference = %q", got)
	}
	if got := a.Mul(b).String(); got != "0.02" {
		t.Fatalf("product = %q", got)
	}
	quotient, err := b.Quo(a)
	if err != nil || quotient.String() != "2" {
		t.Fatalf("quotient = %q, %v", quotient.String(), err)
	}
	third, _ := ParseDecimal("1/3")
	if got := third.String(); got != "0.333333333333333333" {
		t.Fatalf("third = %q", got)
	}
	if a.Neg().String() != "-0.1" || a.Cmp(b) >= 0 {
		t.Fatal("negation or comparison failed")
	}
}

func TestDecimalConstructionAndErrors(t *testing.T) {
	if _, err := ParseDecimal(""); err == nil {
		t.Fatal("empty decimal succeeded")
	}
	if _, err := ParseDecimal("not-a-number"); err == nil {
		t.Fatal("invalid decimal succeeded")
	}
	if exponent, err := ParseDecimal("1.25e3"); err != nil || exponent.String() != "1250" {
		t.Fatalf("exponent decimal = %q, %v", exponent.String(), err)
	}
	if exponent, err := ParseDecimal("2E-3"); err != nil || exponent.String() != "0.002" {
		t.Fatalf("negative exponent decimal = %q, %v", exponent.String(), err)
	}
	if _, err := ParseDecimal("1e10001"); err == nil {
		t.Fatal("oversized exponent succeeded")
	}
	zero := DecimalValue{}
	if zero.String() != "0" || zero.Rat().Sign() != 0 {
		t.Fatal("zero value is not usable")
	}
	if DecimalFromInt(-2).String() != "-2" || DecimalFromUint(3).String() != "3" {
		t.Fatal("integer conversion failed")
	}
	if _, err := DecimalFromInt(1).Quo(zero); err == nil {
		t.Fatal("division by zero succeeded")
	}
	if (MissingValue{}).String() != "missing" {
		t.Fatal("missing value text changed")
	}
}
