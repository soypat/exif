package exif

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"

	"github.com/soypat/exif/rational"
)

//go:generate go run ./cmd/codegen

// IFD or Image File Directory
type IFD struct {
	Tags  []Tag
	Group Group
}

// Tag represents an EXIF field and the contained data in the field.
type Tag struct {
	ID   ID
	data any
}

// String returns a human readable representation of the tag and its value.
func (t Tag) String() string {
	desc, err := t.Describe()
	if err != nil {
		return fmt.Sprintf("!ERR %s: %v", t.ID.String(), err.Error())
	}
	return fmt.Sprintf("%s (%s): %v", t.ID.String(), t.ID.Type().String(), desc)
}

// Describe returns a human-readable description of the value contained in the
// EXIF tag. It converts the tag's numeric value to a more meaningful textual
// representation based on the EXIF specification.
//
// In case of an issue with the tag's value or ID correspondence, Describe
// returns an error detailing the problem.
func (t Tag) Describe() (description string, err error) {
	if t.data == nil {
		return "", errors.New("nil tag value")
	}
	tagdef, ok := getTagdef(t.ID)
	if !ok {
		return "", errors.New("unknown tag ID")
	}
	tp := t.ID.Type()
	if tp == 0 {
		tp = tagdef.Type
	}
	switch {
	case tp.IsFloat():
		v, err := t.Float()
		if err != nil {
			return "", err
		}
		description = strconv.FormatFloat(v, 'g', 6, 64)

	case tp.IsInt():
		v, err := t.Int()
		if err != nil {
			return "", err
		}
		// Case where the ID represents an enum.
		if len(tagdef.enum) > 0 {
			v, err := toInt(t.data)
			if err != nil {
				return "", fmt.Errorf("%v <unexpected %T type>", t.data, t.data)
			}
			for i, enum := range tagdef.enum {
				if enum == v {
					return tagdef.enumString[i], nil
				}
			}
			return "", fmt.Errorf("%d <unexpected value of Exif enum>", v)
		}
		// Just an integer.
		description = strconv.FormatInt(v, 10)

	case tp.IsRational():
		v, err := t.Rational()
		if err != nil {
			return "", err
		}
		stringer := v.(fmt.Stringer)
		description = stringer.String()

	case tp.IsBytes():
		v, err := t.Bytes()
		if err != nil {
			return "", err
		}
		if tp == TypeString {
			description = string(v)
		} else {
			description = fmt.Sprintf("%q", v)
		}
	case tp == 0: // Unknown type.
		// Some tags have this label. They are usually offset
		description = fmt.Sprintf("%v", t.data)
	default:
		return "", fmt.Errorf("unknown Exif type code (%d)", uint16(tp))
	}

	return description, nil
}

// Value returns the value contained in the tag. An uninitialized tag will return nil.
func (t Tag) Value() any {
	return t.data
}

// NewTag creates a new tag with the underlying value.
// It returns an error if the resulting tag would be malformed.
// i.e: mismatched type between value and what would be expected with tag's ID.
func NewTag(id ID, value any) (_ Tag, err error) {
	inputValueType := Type(0)
	v, err := toInt(value)
	if err == nil {
		inputValueType = TypeInt32
		value = v
	}
	if v, ok := value.(float32); ok {
		inputValueType = TypeFloat64
		value = float64(v)
	}
	idTp := id.Type()
	if inputValueType.IsInt() != idTp.IsInt() ||
		inputValueType.IsFloat() != idTp.IsFloat() {
		return Tag{}, fmt.Errorf("mismatch between value type %s and %q type %s", inputValueType.String(), id.String(), id.Type().String())
	}

	return Tag{ID: id, data: value}, nil
}

// Type is the set of all types one may encounter when parsing EXIF data.
type Type uint16

const (
	_ = iota
	// TypeUint8 can be found as Byte type in EXIF spec.
	TypeUint8
	// TypeString a.k.a. ASCII.
	TypeString
	TypeUint16
	TypeUint32
	// Unsigned rational type.
	TypeURational64
	TypeInt8
	TypeUndefined
	TypeInt16
	TypeInt32
	// Signed rational type.
	TypeRational64
	TypeFloat32
	// TypeFloat64 can be found as the double type in EXIF spec.
	TypeFloat64
)

// Group represents the IFD group.
type Group uint8

const (
	GroupNone Group = iota
	// IFD of the main image. Usually contains ExifOffset tag which points to the ExifSubIFD.
	GroupIFD0
	// IFD of the thumbnail.
	GroupIFD1
	// IFD containing digicam's information such as shutter speed, focal length etc.
	GroupSubIFD
	GroupExifIFD
	GroupInteropIFD
)

// String returns a human readable representation of the IFD group. i.e: IFD0, IFD1, SubIFD.
func (g Group) String() (s string) {
	switch g {
	case GroupIFD0:
		s = "IFD0"
	case GroupIFD1:
		s = "IFD1"
	case GroupSubIFD:
		s = "SubIFD"
	case GroupExifIFD:
		s = "ExifIFD"
	case GroupInteropIFD:
		s = "InteropIFD"
	default:
		s = "<unknown IFD group>"
	}
	return s
}

// Size returns the size in bytes of the type. Can be 1, 2, 4, or 8 for valid types. 0 otherwise.
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
		s = 0 // Invalid type.
	}
	return s
}

// String returns a Go-like representation of the type.
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

// String returns a camel case human readable representation of the ID.
func (id ID) String() string {
	tag, ok := getTagdef(id)
	if !ok {
		return "<unknown EXIF ID>"
	}
	return tag.Name
}

// Type returns the type of data the ID field would contain.
func (id ID) Type() Type {
	tg, _ := getTagdef(id)
	return tg.Type
}

// IsMandatory returns true if the tag is specified as mandatory in the EXIF spec.
func (id ID) IsMandatory() bool {
	tg, _ := getTagdef(id)
	return tg.flags.IsMandatory()
}

// IsStaticSize returns true if the ids data array size is of constrained length/size.
func (id ID) IsStaticSize() bool {
	tg, _ := getTagdef(id)
	return tg.arrayLen[1] != 0
}

type tagdef struct {
	Name       string
	arrayLen   [2]int
	ID         ID
	Type       Type
	flags      flags
	enum       []int64
	enumString []string
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

// DecodeTypeData takes raw EXIF byte slice data and interprets it according to
// the Type tp and the byte order. It returns an empty interface containing the
// interpreted value if err is nil. This function should be used for tags of
// constrained length.
// It may return any of the following types:
//   - int64 for integers.
//   - float64 for floats.
//   - rational.U64 for unsigned rational numbers.
//   - rational.I64 for signed rational numbers.
//   - []byte for undefined type (identical to input data).
//   - string for String (ASCII) type which is just string(data).
func DecodeTypeData(tp Type, order binary.ByteOrder, data []byte) (v any, err error) {
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
		v = int64(data[0])
	case TypeUint16:
		v = int64(order.Uint16(data[:2]))
	case TypeUint32:
		v = int64(order.Uint32(data[:4]))
	case TypeInt8:
		v = int64(int8(data[0]))
	case TypeInt16:
		v = int64(int16(order.Uint16(data[:2])))
	case TypeInt32:
		v = int64(int32(order.Uint32(data[:4])))
	case TypeFloat32:
		v = float64(math.Float32frombits(order.Uint32(data[:4])))
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

// IsInt returns true if tp is a signed or unsigned integer type.
func (tp Type) IsInt() bool {
	return tp == TypeInt8 || tp == TypeInt16 || tp == TypeInt32 ||
		tp == TypeUint8 || tp == TypeUint16 || tp == TypeUint32
}

// IsFloat returns true if tp is of float32 (single) or float64 (double) type.
func (tp Type) IsFloat() bool {
	return tp == TypeFloat32 || tp == TypeFloat64
}

// IsRational returns true if tp is of unsigned or signed rational type.
func (tp Type) IsRational() bool {
	return tp == TypeRational64 || tp == TypeURational64
}

// IsBytes returns true if tp is of string or undefined (blob/binary/byte) data type.
func (tp Type) IsBytes() bool {
	return tp == TypeString || tp == TypeUndefined
}

// Bytes returns the bytes contained in the tag value if the tag is of
// the TypeString or TypeUndefined Exif tag type.
func (tag Tag) Bytes() (v []byte, err error) {
	tp := tag.ID.Type()
	if tp != TypeString && tp != TypeUndefined {
		return nil, errors.New("Bytes undefined for type " + tp.String())
	}
	switch c := tag.data.(type) {
	case nil:
		// Do nothing. TODO(soypat): Should this be an error?
	case string:
		v = []byte(c)
	case []byte:
		v = c
	default:
		err = fmt.Errorf("mismatching type in Tag of type %s: %T", tp.String(), tag.data)
	}
	return v, err
}

// Int returns the integer value contained in the tag if the value is of integer type.
// This function returns an error if the ID of the tag does not match a integer type
// (signed or unsigned) or if the type contained is not a integer type.
func (tag Tag) Int() (int64, error) {
	if !tag.ID.Type().IsInt() {
		return 0, errors.New("exif ID is not of integer type")
	}
	if tag.data == nil {
		return 0, errors.New("nil tag value")
	}
	v, ok := tag.data.(int64)
	if ok {
		return v, nil
	}
	v, err := toInt(tag.data)
	if err == nil {
		return v, nil
	}
	return 0, fmt.Errorf("tag did not contain integer type %T (%s)", tag.data, err)
}

// MustInt returns the integer contained in the tag's value.
// It is a wrapper around Int that panics if Int returns an error.
func (tag Tag) MustInt() int64 {
	i, err := tag.Int()
	if err != nil {
		panic(err)
	}
	return i
}

// Float returns the float32 or float64 value contained in the tag if the value is of float type.
// This function returns an error if the ID of the tag does not match a float type or if the type
// contained is not a float type.
func (tag Tag) Float() (float64, error) {
	if !tag.ID.Type().IsFloat() {
		return 0, errors.New("exif ID is not of float type")
	}
	if tag.data == nil {
		return 0, errors.New("nil tag value")
	}
	v, ok := tag.data.(float64)
	if ok {
		return v, nil
	}
	v32, ok := tag.data.(float32)
	if ok {
		return float64(v32), nil
	}
	return 0, fmt.Errorf("tag did not contain float type: %T", tag.data)
}

// MustFloat returns the float contained in the tag's value.
// It is a wrapper around Float that panics if Float returns an error.
func (tag Tag) MustFloat() float64 {
	f64, err := tag.Float()
	if err != nil {
		panic(err)
	}
	return f64
}

// Rational returns the underlying rational number contained in the tag if
// the value implements the [rational.Rational] interface.
// This function returns an error if the ID of the tag does not match a rational
// type or if the type contained does not implement the rational.Rational interface.
func (tag Tag) Rational() (rational.Rational, error) {
	if !tag.ID.Type().IsRational() {
		return nil, errors.New("exif ID is not of rational type")
	}
	if tag.data == nil {
		return nil, errors.New("nil tag value")
	}
	v, ok := tag.data.(rational.Rational)
	if !ok {
		return nil, fmt.Errorf("tag did not contain a rational type: %T", tag.data)
	}
	return v, nil
}

// MustRational returns the rational number contained in the tag's value.
// It is a wrapper around Rational that panics if Rational returns an error.
func (tag Tag) MustRational() rational.Rational {
	rat, err := tag.Rational()
	if err != nil {
		panic(err)
	}
	return rat
}

func toInt(v any) (ret int64, _ error) {
	switch c := v.(type) {
	case int8:
		ret = int64(c)
	case int16:
		ret = int64(c)
	case int32:
		ret = int64(c)
	case uint8:
		ret = int64(c)
	case uint16:
		ret = int64(c)
	case uint32:
		ret = int64(c)
	case int64:
		ret = c
	case int:
		ret = int64(c)
	case uint:
		if c > math.MaxInt64 {
			return 0, errors.New("uint overflows int64")
		}
		ret = int64(c)
	case uint64:
		if c > math.MaxInt64 {
			return 0, errors.New("uint64 overflows int64")
		}
		ret = int64(c)
	default:
		return 0, errors.New("value is not of integer type")
	}
	return ret, nil
}

// tag file generation flags.
var (
	arrayLenInvalid = [2]int{-1, -1}
)

//go:inline
func getTagdef(id ID) (tagdef, bool) {
	tag, ok := tags[uint16(id)]
	return tag, ok
	// if int(id) > len(tags) {
	// 	return tagdef{}, false
	// }
	// return tags[id], true
}

func stringTagInt(id ID, value int64) string {
	tag, ok := getTagdef(id)
	if !ok || len(tag.enum) == 0 {
		return strconv.FormatInt(value, 10)
	}
	for i, v := range tag.enum {
		if v == value {
			return tag.enumString[i]
		}
	}
	return strconv.FormatInt(value, 10)
}

// FindStartOffset searches the given reader for the start of the EXIF metadata,
// and a buffer to read the reader's contents into.
// It returns the offset of the start of the metadata and any error encountered.
//
// The returned offset points to the first byte of the EXIF metadata in the file
// which is the byte ordering.
// The caller can use this offset to create a new reader that starts at the EXIF metadata
// using [io.NewSectionReader].
//
// Note that this function assumes that the file format is compatible with the EXIF
// standard and that the metadata start is indicated by the "Exif\x00\x00" pattern.
// If the file does not contain EXIF metadata or uses a different format, this function
// may return an error or an incorrect offset.
//
// If the argument buffer is nil one will be automatically allocated.
func FindStartOffset(rd io.ReaderAt, buffer []byte) (startOffset int64, err error) {
	const (
		pattern    = "Exif\x00\x00"
		patternLen = len(pattern)
	)
	// Attempt to perform feeling-lucky quick search.
	var arr [32]byte
	n, err := rd.ReadAt(arr[:], 0)
	if err != nil {
		return -1, err
	}
	idx := bytes.Index(arr[:n], []byte(pattern))
	if idx >= 0 {
		return int64(idx + patternLen), nil // Quick return case.
	}

	// Perform long search.
	if buffer == nil {
		buffer = make([]byte, 16*1024)
	}
	n = 0
	for i := int64(0); ; i += int64(n - patternLen + 1) {
		n, err = rd.ReadAt(buffer, i)
		if n < 2*patternLen {
			i -= 2 * int64(patternLen)
			n, err = rd.ReadAt(buffer, i)
			if n < 2*patternLen {
				return -1, errors.New("extraordinary error: short buffer reads")
			}
		}
		idx := bytes.Index(buffer[:n], []byte(pattern))
		if idx >= 0 {
			return i + int64(idx+patternLen), nil
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return -1, err
		}
	}
	return -1, errors.New("did not find exif metadata start pattern")
}
