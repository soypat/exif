package exif

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Tag ID	Tag Name	Writable	Group	Values / Notes
//
//go:embed exif.txt
var txt []byte

func ExampleGeneration() {
	fp, _ := os.Create("tagdefinitions.go")
	defer fp.Close()
	// fp, _ = os.Open(os.DevNull)
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
		str := fmt.Sprintf("\t%0#4x: {Name: %q, Type: %d, flags: %x, arrayLen: [2]int{%d, %d}",
			tag.ID, tag.Tagname, tp, flag, arraylen[0], arraylen[1])
		fmt.Fprint(fp, str)
		fmt.Fprintf(fp, ", ID: %0#4x", tag.ID)
		fp.WriteString("},\n")
		_ = grp

		// fmt.Fprintf(fp, "\t%+v %d %d\n", tag.Writable, tp, flag)
	}
	fmt.Fprint(fp, "}\n")
	genExifid()
	fmt.Println(time.Now()) // so that generate runs.
	// Output:
	// None.
}

func parseType(s string) (tp Type, flags flags, arrayLen [2]int) {
	arrayLen = arrayLenInvalid // default value is -1, -1.
	if s == "-" || len(s) == 0 {
		return 0, 0, arrayLen
	}
	if s == "no" {
		return 0, 0, arrayLen
	}
	unsafe := strings.ContainsRune(s, '!')
	protected := strings.ContainsRune(s, '*')
	avoid := strings.ContainsRune(s, '/')
	writeConstrained := strings.ContainsRune(s, '~')
	mandatory := strings.ContainsRune(s, ':')

	flagCount := b2u8(unsafe) + b2u8(avoid) + b2u8(mandatory) + b2u8(writeConstrained) + b2u8(protected)
	typeString := s[:len(s)-int(flagCount)]
	var arrayLengthMin = -1
	var arrayLengthMax = 1
	arrayStart := strings.IndexByte(s, '[')
	if arrayStart > 0 {
		lenString := s[arrayStart+1 : strings.IndexByte(s, ']')]
		dotIdx := strings.IndexByte(lenString, '.')
		if dotIdx >= 0 {
			lenString = lenString[dotIdx+1:]
			arrayLengthMin = 0
		}

		if lenString == "n" {
			arrayLengthMax = 0 // undefined length
		} else {
			v, err := strconv.Atoi(lenString)
			if err != nil || v == 0 {
				panic(fmt.Sprintf("unknwon type string %q from parsed %q: %s", typeString, s, err))
			}
			arrayLengthMin = v
		}
		typeString = typeString[:arrayStart]
	}
	isSigned := typeString[len(typeString)-1] == 's'
	if isSigned || typeString[len(typeString)-1] == 'u' {
		typeString = typeString[:len(typeString)-1]
	}
	signedAdd := Type(b2u8(isSigned)) * 5
	switch typeString {
	case "undef":
		tp = TypeUndefined
	case "int16":
		tp = TypeUint16
	case "int32":
		tp = TypeUint32 + signedAdd
	case "int8":
		tp = TypeUint8 + signedAdd
	case "rational64":
		tp = TypeURational64 + signedAdd
	case "double":
		tp = TypeFloat64
	case "float":
		tp = TypeFloat32
	case "string":
		tp = TypeString
	default:
		panic(fmt.Sprintf("unknwon type string %q from parsed %q", typeString, s))
	}
	return tp, newflags(unsafe, protected, avoid, writeConstrained, mandatory),
		[2]int{arrayLengthMin, arrayLengthMax}
}

type TagPreproces struct {
	Tagname  string
	ID       uint16
	Writable string
	Group    string
	Values   []string
}

func genExifid() {
	os.Mkdir("exifid", 0777)
	fp, err := os.Create("exifid/exifid.go")
	if err != nil {
		panic(err)
	}
	defer fp.Close()
	fp.WriteString(`package exifid

import "github.com/soypat/exif"

// All Exif field/tag IDs.
const (
`)
	var tagslice tagdefs
	written := make(map[string]struct{})
	maxLen := 0
	for _, tag := range tags {
		if _, ok := written[tag.Name]; !ok && !strings.ContainsAny(tag.Name, "-?") {
			written[tag.Name] = struct{}{}
			tagslice = append(tagslice, tag)
			if len(tag.Name) > maxLen {
				maxLen = len(tag.Name)
			}
		}
	}
	sort.Sort(tagslice)
	fmtString := "\t%-" + strconv.Itoa(maxLen) + "s exif.ID = %0#4x\n"
	for _, tag := range tagslice {
		fmt.Fprintf(fp, fmtString, tag.Name, uint16(tag.ID))
		written[tag.Name] = struct{}{}
	}
	fp.WriteString(")\n")
}

type tagdefs []tagdef

func (a tagdefs) Len() int           { return len(a) }
func (a tagdefs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a tagdefs) Less(i, j int) bool { return a[i].ID < a[j].ID }
