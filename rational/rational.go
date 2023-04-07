package rational

import (
	"encoding/binary"
	"errors"
	"strconv"
)

// Rational is implemented by rational numbers in this package.
type Rational interface {
	Fraction() (numerator, denominator int)
}

var (
	errZeroDenominator = errors.New("zero denominator")
	errShortBuf        = errors.New("buffer too short")
)

func NewI64(numerator, denominator int) I64 {
	if denominator == 0 {
		panic(errZeroDenominator)
	}
	return I64{num: int32(numerator), denMinusOne: int32(denominator) - 1}
}

func NewU64(numerator, denominator uint) U64 {
	if denominator == 0 {
		panic(errZeroDenominator)
	}
	return U64{num: uint32(numerator), denMinusOne: uint32(denominator) - 1}
}

type I64 struct {
	num         int32
	denMinusOne int32
}

type U64 struct {
	num         uint32
	denMinusOne uint32
}

func (u U64) Float() float64 {
	return float64(u.num) / float64(u.denMinusOne+1)
}

func (u I64) Float() float64 {
	return float64(u.num) / float64(u.denMinusOne+1)
}

func DecodeU64(order binary.ByteOrder, b []byte) (U64, error) {
	if len(b) < 8 {
		return U64{}, errShortBuf
	}
	denominator := order.Uint32(b[4:])
	if denominator == 0 {
		return U64{}, errZeroDenominator
	}
	numerator := order.Uint32(b)
	return U64{denMinusOne: denominator - 1, num: numerator}, nil
}

func DecodeI64(order binary.ByteOrder, b []byte) (I64, error) {
	if len(b) < 8 {
		return I64{}, errShortBuf
	}
	denominator := int32(order.Uint32(b[4:]))
	if denominator == 0 {
		return I64{}, errZeroDenominator
	}
	numerator := int32(order.Uint32(b))
	return I64{denMinusOne: denominator - 1, num: numerator}, nil
}

func (i I64) Fraction() (numerator, denominator int) {
	return int(i.num), int(i.denMinusOne + 1)
}

func (i U64) Fraction() (numerator, denominator int) {
	return int(i.num), int(i.denMinusOne + 1)
}

func (i I64) String() string {
	num, den := i.Fraction()
	return strconv.Itoa(int(num)) + "/" + strconv.Itoa(int(den))
}

func (i U64) String() string {
	num, den := i.Fraction()
	return strconv.FormatUint(uint64(num), 10) + "/" + strconv.FormatUint(uint64(den), 10)
}
