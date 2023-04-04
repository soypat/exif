package exif

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/soypat/exif/rational"
)

//go:generate go run generate_tagdefinitions.go

// TODO
type IFD struct{}

type Tag struct {
	ID   ID
	data any
}

func (t Tag) String() string {
	return fmt.Sprintf("%s (%s): %v", t.ID.String(), t.ID.Type().String(), t.Value())
}

func (t Tag) Value() any {
	return t.data
}

func NewTag(id ID, value any) (_ Tag, err error) {
	inputValueType := Type(0)
	switch value.(type) {
	case uint8:
		inputValueType = TypeUint8
	case uint16:
		inputValueType = TypeUint16
	case uint32:
		inputValueType = TypeUint32
	case int8:
		inputValueType = TypeInt8
	case int16:
		inputValueType = TypeInt16
	case int32:
		inputValueType = TypeInt32
	case float32:
		inputValueType = TypeFloat32
	case float64:
		inputValueType = TypeFloat64
	case string:
		inputValueType = TypeString
	default:
		return Tag{}, fmt.Errorf("unhandled type %T", value)
	}
	if inputValueType != id.Type() {
		//
		err = fmt.Errorf("mismatch between value type %s and %q type %s", inputValueType.String(), id.String(), id.Type().String())
	}
	return Tag{ID: id, data: value}, err
}

type Type uint16

const (
	_ = iota
	TypeUint8
	TypeString
	TypeUint16
	TypeUint32
	TypeURational64
	TypeInt8
	TypeUndefined
	TypeInt16
	TypeInt32
	TypeRational64
	TypeFloat32
	TypeFloat64
)

type Group uint8

const (
	GroupNone Group = iota
	GroupInteropIFD
	GroupIFD0
	GroupExifIFD
	GroupSubIFD
)

func (tp Type) Size() (s uint8) {
	switch tp {
	case TypeInt8, TypeUint8, TypeString, TypeUndefined:
		s = 1
	case TypeUint16, TypeInt16:
		s = 2
	case TypeUint32, TypeInt32, TypeFloat32:
		s = 4
	case TypeRational64, TypeFloat64, TypeURational64:
		s = 8
	default:
		s = 0 // Default value will be 0.
	}
	return s
}

func (tp Type) String() (s string) {
	switch tp {
	case TypeUint8:
		s = "uint8"
	case TypeString:
		s = "string"
	case TypeUint16:
		s = "uint16"
	case TypeUint32:
		s = "uint32"
	case TypeUndefined:
		s = "undefined"
	case TypeInt16:
		s = "int16"
	case TypeInt32:
		s = "int32"
	case TypeInt8:
		s = "int8"
	case TypeFloat32:
		s = "float32"
	case TypeFloat64:
		s = "float64"
	case TypeRational64:
		s = "urational"
	case TypeURational64:
		s = "rational"
	default:
		s = "unknown"
	}
	return s
}

type ID uint16

func (id ID) String() string {
	return tags[uint16(id)].Name
}

func (id ID) Type() Type {
	return tags[uint16(id)].Type
}

func (id ID) Group() Group {
	return tags[uint16(id)].Group
}

func (id ID) IsMandatory() bool {
	return tags[uint16(id)].flags.IsMandatory()
}

type tagdef struct {
	Name     string
	arrayLen [2]int
	ID       uint16
	Type     Type
	flags    flags
	Group    Group
}

func newflags(unsafe, protected, avoid, writeConstrained, mandatory bool) flags {
	return flags(b2u8(mandatory) | b2u8(unsafe)<<1 | b2u8(protected)<<2 |
		b2u8(avoid)<<3 | b2u8(writeConstrained)<<4)
}

type flags uint8

func (f flags) IsMandatory() bool        { return f&1 != 0 }
func (f flags) IsUnsafe() bool           { return f&(1<<1) != 0 }
func (f flags) IsProtected() bool        { return f&(1<<2) != 0 }
func (f flags) Avoid() bool              { return f&(1<<3) != 0 }
func (f flags) IsWriteConstrained() bool { return f&(1<<4) != 0 }

func b2u8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}

func EvaluateData(tp Type, order binary.ByteOrder, data []byte) (v any, err error) {
	sz := tp.Size()
	if sz == 0 {
		return nil, errors.New("invalid type")
	}
	count := len(data) / int(sz)
	if count == 0 || len(data)%int(sz) != 0 {
		return nil, errors.New("bad byte buffer size for type")
	}
	if tp == TypeString {
		return string(data), nil
	} else if tp == TypeUndefined {
		return data, nil
	}
	if count > 1 {
		return nil, errors.New("slices not implemented yet")
	}
	switch tp {
	case TypeUint8:
		v = data[0]
	case TypeUint16:
		v = order.Uint16(data[:2])
	case TypeUint32:
		v = order.Uint32(data[:4])
	case TypeInt8:
		v = int8(data[0])
	case TypeInt16:
		v = int16(order.Uint16(data[:2]))
	case TypeInt32:
		v = int32(order.Uint32(data[:4]))
	case TypeFloat32:
		v = math.Float32frombits(order.Uint32(data[:4]))
	case TypeFloat64:
		v = math.Float64frombits(order.Uint64(data[:8]))
	case TypeRational64:
		v, err = rational.DecodeI64(order, data[:8])
	case TypeURational64:
		v, err = rational.DecodeU64(order, data[:8])
	default:
		return nil, errors.New("unsupported data type: " + tp.String())
	}
	return v, err
}
