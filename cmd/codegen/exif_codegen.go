package main

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
	_ "unsafe"

	"github.com/soypat/exif"
)

// Tag ID	Tag Name	Writable	Group	Values / Notes
//
//go:embed exif.txt
var txt []byte

func main() {
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
		var grp exif.Group
		switch tag.Group {
		case "IFD0":
			grp = exif.GroupIFD0
		case "ExifIFD":
			grp = exif.GroupExifIFD
		case "InteropIFD":
			grp = exif.GroupInteropIFD
		case "SubIFD":
			grp = exif.GroupSubIFD
		default:
			grp = exif.GroupNone
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
	genExifid(tags)
	fmt.Println(time.Now()) // so that generate runs.
	// Output:
	// None.
}

func parseType(s string) (tp exif.Type, flags uint8, arrayLen [2]int) {
	arrayLen = [2]int{-1, -1} // default value is -1, -1.
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
	signedAdd := exif.Type(b2u8(isSigned)) * 5
	switch typeString {
	case "undef":
		tp = exif.TypeUndefined
	case "int16":
		tp = exif.TypeUint16
	case "int32":
		tp = exif.TypeUint32 + signedAdd
	case "int8":
		tp = exif.TypeUint8 + signedAdd
	case "rational64":
		tp = exif.TypeURational64 + signedAdd
	case "double":
		tp = exif.TypeFloat64
	case "float":
		tp = exif.TypeFloat32
	case "string":
		tp = exif.TypeString
	default:
		panic(fmt.Sprintf("unknwon type string %q from parsed %q", typeString, s))
	}
	flags = newflags(unsafe, protected, avoid, writeConstrained, mandatory)
	return tp, flags,
		[2]int{arrayLengthMin, arrayLengthMax}
}

type TagPreproces struct {
	Tagname  string
	ID       uint16
	Writable string
	Group    string
	Values   []string
}

func genExifid(tags tagPs) {
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
	// Sort for consistent results.
	sort.Sort(tags)
	var tagslice tagPs

	for _, tag := range tags {
		if !strings.ContainsAny(tag.Tagname, "-?") {
			tagslice = append(tagslice, tag)
		}
	}

	// Delete duplicated entries
	written := make(map[string]struct{})
	maxLen := 0
	var uniqTagSlice tagPs
	for _, tag := range tagslice {
		_, ok := written[tag.Tagname]
		if !ok {
			uniqTagSlice = append(uniqTagSlice, tag)
			written[tag.Tagname] = struct{}{}
			if len(tag.Tagname) > maxLen {
				maxLen = len(tag.Tagname)
			}
		}
	}
	fmtString := "\t%-" + strconv.Itoa(maxLen) + "s exif.ID = %0#4x\n"
	for _, tag := range uniqTagSlice {
		fmt.Fprintf(fp, fmtString, tag.Tagname, uint16(tag.ID))
		written[tag.Tagname] = struct{}{}
	}
	fp.WriteString(")\n")
}

type tagPs []TagPreproces

func (a tagPs) Len() int           { return len(a) }
func (a tagPs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a tagPs) Less(i, j int) bool { return a[i].ID < a[j].ID }

//go:linkname newflags github.com/soypat/exif.newflags
func newflags(unsafe, protected, avoid, writeConstrained, mandatory bool) uint8

func b2u8(b bool) uint8 {
	if b {
		return 1
	}
	return 0
}
