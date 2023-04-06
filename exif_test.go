package exif

import (
	"encoding/binary"
	"testing"
)

func TestDecodeTypeData_integers(t *testing.T) {
	testCases := []struct {
		desc     string
		data     []byte
		tp       Type
		order    binary.ByteOrder
		expected int64
	}{
		{
			desc:     "int8",
			data:     []byte{0x7f},
			tp:       TypeInt8,
			expected: 0x7f,
		},
		{
			desc:     "byte",
			data:     []byte{0x7f},
			tp:       TypeUint8,
			expected: 0x7f,
		},
		{
			desc:     "uint32",
			data:     []byte{0xfe, 0xed, 0xbe, 0xad},
			tp:       TypeUint32,
			order:    binary.BigEndian,
			expected: 0xfeedbead,
		},
	}
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			v, err := DecodeTypeData(tC.tp, tC.order, tC.data)
			if err != nil {
				t.Fatal(err)
			}
			if v != tC.expected {
				t.Errorf("mismatch between %v and %v", v, tC.expected)
			}
		})
	}
}
