package exif_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/soypat/exif"
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
	ifds, err := decoder.MakeIFDs(fp, func(_ exif.IFD, id exif.ID) bool {
		return true // Make all tags.
	})
	if err != nil {
		panic(err)
	}
	for i, ifd := range ifds {
		fmt.Printf("ifd%d:\n", i)
		for _, tag := range ifd.Tags {
			fmt.Println("\t" + strings.Trim(tag.String(), "\x00"))
		}
	}
	//Output:
	// ifd0:
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
	// 	XResolution (urational): 2000000/10000
	// 	YResolution (urational): 2000000/10000
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
	ifds, err := decoder.MakeIFDs(nil, func(_ exif.IFD, id exif.ID) bool {
		return true // Make all tags.
	})
	if err != nil {
		panic(err)
	}
	for i, ifd := range ifds {
		fmt.Printf("ifd%d:\n", i)
		for _, tag := range ifd.Tags {
			fmt.Println("\t" + strings.Trim(tag.String(), "\x00"))
		}
	}
	// ifd0:
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
