package exif_test

import (
	"fmt"
	"os"

	"github.com/soypat/exif"
	"github.com/soypat/exif/tiff"
)

func ExampleLazyTiff() {
	fp, err := os.Open("testdata/sample1.tiff")
	if err != nil {
		panic(err)
	}
	ltiff, err := tiff.LazyDecode(fp)
	if err != nil {
		panic(err)
	}
	tags, err := ltiff.MakeTags(fp, func(_ exif.IFD, id exif.ID) bool {
		return true // Make all tags.
	})
	if err != nil {
		panic(err)
	}
	for _, tag := range tags {
		fmt.Println(tag)
	}
	//Output:
	// ImageWidth (uint32): 1728
	// ImageHeight (uint32): 2376
	// BitsPerSample (uint16): 1
	// Compression (uint16): 4
	// PhotometricInterpretation (uint16): 0
	// FillOrder (uint16): 2
	// StripOffsets (unknown): 8
	// Orientation (uint16): 1
	// SamplesPerPixel (uint16): 1
	// RowsPerStrip (uint32): 2376
	// StripByteCounts (unknown): 18112
	// PlanarConfiguration (uint16): 1
	// ResolutionUnit (uint16): 2
}
