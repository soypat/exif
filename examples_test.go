package exif_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/soypat/exif"
	"github.com/soypat/exif/exifid"
)

func ExampleLazyDecoder() {
	fp, err := os.Open("testdata/sample1.tiff")
	if err != nil {
		panic(err)
	}
	var decoder exif.LazyDecoder
	err = decoder.Decode(fp)
	if err != nil {
		panic(err)
	}
	ifds, err := decoder.MakeIFDs(fp, func(ifd, size int, id exif.ID) bool {
		return size < 1024 // Tags less than a kilobyte in size.
	})
	if err != nil {
		panic(err)
	}
	for _, ifd := range ifds {
		fmt.Printf("%s:\n", ifd.Group.String())
		for _, tag := range ifd.Tags {
			fmt.Println("\t" + strings.Trim(tag.String(), "\x00"))
		}
	}
	//Output:
	// IFD0:
	// 	ImageWidth (uint32): 1728
	// 	ImageHeight (uint32): 2376
	// 	BitsPerSample (uint16): 1
	// 	Compression (uint16): 4
	// 	PhotometricInterpretation (uint16): 0
	// 	FillOrder (uint16): 2
	// 	DocumentName (string): Standard Input
	// 	ImageDescription (string): converted PBM file
	// 	StripOffsets (unknown): 8
	// 	Orientation (uint16): 1
	// 	SamplesPerPixel (uint16): 1
	// 	RowsPerStrip (uint32): 2376
	// 	StripByteCounts (unknown): 18112
	// 	XResolution (rational): 2000000/10000
	// 	YResolution (rational): 2000000/10000
	// 	PlanarConfiguration (uint16): 1
	// 	ResolutionUnit (uint16): 2
}

func ExampleLazyDecoder_onlyWords() {
	fp, err := os.Open("testdata/sample1.tiff")
	if err != nil {
		panic(err)
	}
	var decoder exif.LazyDecoder
	err = decoder.Decode(fp)
	if err != nil {
		panic(err)
	}
	// Here we are passing in a nil reader, so decoder will only process
	// tags which have a lazy in-memory representation.
	ifds, err := decoder.MakeIFDs(nil, func(ifd, size int, id exif.ID) bool {
		// Make all encountered tags except ExifOffset.
		return true && id != exifid.ExifOffset
	})
	if err != nil {
		panic(err)
	}
	for _, ifd := range ifds {
		fmt.Printf("%s:\n", ifd.Group.String())
		for _, tag := range ifd.Tags {
			fmt.Println("\t" + strings.Trim(tag.String(), "\x00"))
		}
	}
	// IFD0:
	//	ImageWidth (uint32): 1728
	//	ImageHeight (uint32): 2376
	//	BitsPerSample (uint16): 1
	//	Compression (uint16): 4
	//	PhotometricInterpretation (uint16): 0
	//	FillOrder (uint16): 2
	//	StripOffsets (unknown): 8
	//	Orientation (uint16): 1
	//	SamplesPerPixel (uint16): 1
	//	RowsPerStrip (uint32): 2376
	//	StripByteCounts (unknown): 18112
	//	PlanarConfiguration (uint16): 1
	//	ResolutionUnit (uint16): 2
}
