package exif_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/soypat/exif"
)

func ExampleLocal() {
	fp, err := os.Open("testdata/large.tiff")
	if err != nil {
		panic(err)
	}

	// var buf [1024]byte
	// _, _ = io.ReadFull(fp, buf[:])
	// fmt.Printf("%x\n\n", buf[:200])
	// return
	var dec exif.LazyDecoder
	err = dec.Decode(fp)
	if err != nil {
		panic(err)
	}
	ifds, err := dec.MakeIFDs(fp, func(ifd, size int, id exif.ID) bool {
		return true
	})
	offset := dec.EndOfApp1()
	fmt.Println(offset)
	for _, ifd := range ifds {
		fmt.Println(ifd.Group)
		for _, tag := range ifd.Tags {
			str := strings.Trim(tag.String(), "\x00")
			fmt.Println("\t" + str)
		}
	}

	// Output:
	// None.
}
