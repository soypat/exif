package exif

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"
)

// Tag ID	Tag Name	Writable	Group	Values / Notes
//
//go:embed exif.txt
var txt []byte

func TestTags(t *testing.T) {
	fp, _ := os.Create("tagdefinitions.go")
	scn := bufio.NewScanner(bytes.NewReader(txt))
	var tags []TagPreproces
	var currentType TagPreproces
	for scn.Scan() {
		line := scn.Text()
		if len(line) < 6 {
			log.Println("skipping line", line)
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		if line[0] == '\t' {
			currentType.Values = append(currentType.Values, strings.TrimSpace(line))
			continue
		}
		if line[0:2] == "0x" {
			v, err := strconv.ParseUint(fields[0][2:], 16, 16)
			if err != nil {
				continue
			}
			tags = append(tags, currentType)
			currentType = TagPreproces{}
			currentType.Writable = fields[2]
			currentType.Group = fields[3]
			currentType.Tagname = fields[1]
			currentType.ID = uint16(v)
			if len(fields) > 4 {
				currentType.Values = append(currentType.Values, fields[4])
			}
			continue
		}

	}
	tags = tags[1:] // first type is empty
	fmt.Fprint(fp, `package exif

var tags = map[uint16]tagdef{
`)
	for i := range tags {
		tag := tags[i]
		tp, flag, arraylen := parseType(tag.Writable)
		var grp Group
		switch tag.Group {
		case "IFD0":
			grp = GroupIFD0
		case "ExifIFD":
			grp = GroupExifIFD
		case "InteropIFD":
			grp = GroupInteropIFD
		case "SubIFD":
			grp = GroupSubIFD
		default:
			grp = GroupNone
		}
		fmt.Fprintf(fp, "\t%0#4x: {Name: %q, Type: %0#x, ID: %0#x, flags: %d, arrayLen: [2]int{%d, %d}, Group: %d},\n",
			tag.ID, tag.Tagname, tp, tag.ID, flag, arraylen[0], arraylen[1], grp)

		// fmt.Fprintf(fp, "\t%+v %d %d\n", tag.Writable, tp, flag)
	}
	fmt.Fprint(fp, "}\n")
}

func parseType(s string) (tp Type, flags flags, arrayLen [2]int) {
	if s == "-" || len(s) == 0 {
		return typeIgnore, 0, arrayLen
	}
	if s == "no" {
		return TypeUndef, 0, arrayLen
	}
	unsafe := strings.ContainsRune(s, '!')
	protected := strings.ContainsRune(s, '*')
	avoid := strings.ContainsRune(s, '/')
	writeConstrained := strings.ContainsRune(s, '~')
	mandatory := strings.ContainsRune(s, ':')

	flagCount := b2u8(unsafe) + b2u8(avoid) + b2u8(mandatory) + b2u8(writeConstrained) + b2u8(protected)
	typeString := s[:len(s)-int(flagCount)]
	var arrayLength = 1
	var arrayLength2 = -1
	arrayStart := strings.IndexByte(s, '[')
	if arrayStart > 0 {
		lenString := s[arrayStart+1 : strings.IndexByte(s, ']')]
		dotIdx := strings.IndexByte(lenString, '.')
		if dotIdx >= 0 {
			lenString = lenString[dotIdx+1:]
			arrayLength2 = 0
		}

		if lenString == "n" {
			arrayLength = 0 // undefined length
		} else {
			v, err := strconv.Atoi(lenString)
			if err != nil {
				panic(fmt.Sprintf("unknwon type string %q from parsed %q: %s", typeString, s, err))
			}
			arrayLength = v
		}
		typeString = typeString[:arrayStart]
	}
	isSigned := typeString[len(typeString)-1] == 's'
	if isSigned || typeString[len(typeString)-1] == 'u' {
		typeString = typeString[:len(typeString)-1]
	}
	signedAdd := Type(b2u8(isSigned))
	switch typeString {
	case "undef":
		tp = TypeUndef
	case "int16":
		tp = TypeUint16 + signedAdd
	case "int32":
		tp = TypeUint32 + signedAdd
	case "int8":
		tp = TypeUint8 + signedAdd
	case "double":

		tp = TypeFloat64
	case "float":
		tp = TypeFloat32
	case "string":
		tp = TypeString
	case "rational64":
		tp = TypeRational64 + signedAdd
	default:
		panic(fmt.Sprintf("unknwon type string %q from parsed %q", typeString, s))
	}
	return tp, newflags(unsafe, protected, avoid, writeConstrained, mandatory),
		[2]int{arrayLength, arrayLength2}
}

type TagPreproces struct {
	Tagname  string
	ID       uint16
	Writable string
	Group    string
	Values   []string
}
