package exif

type Type uint8

const (
	TypeString Type = iota
	TypeUndef
	TypeUint8
	TypeInt8
	TypeUint16
	TypeInt16
	TypeUint32
	TypeInt32
	TypeFloat32
	TypeFloat64
	TypeURational64
	TypeRational64
	typeIgnore
)

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

type Group uint8

const (
	GroupNone Group = iota
	GroupInteropIFD
	GroupIFD0
	GroupExifIFD
	GroupSubIFD
)
