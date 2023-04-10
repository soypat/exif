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
	defer fp.Close()
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
	// Output:
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

func ExampleLazyDecoder_onlyInMemoryTags() {
	fp, err := os.Open("testdata/sample1.tiff")
	if err != nil {
		panic(err)
	}
	defer fp.Close()
	var decoder exif.LazyDecoder
	err = decoder.Decode(fp)
	if err != nil {
		panic(err)
	}
	// Here we are passing in a nil reader, so decoder will only process
	// tags which have a lazy in-memory representation.
	// This will avoid creating string tags like titles, maker notes, GPS etc.
	ifds, err := decoder.MakeIFDs(nil, func(ifd, size int, id exif.ID) bool {
		// We could also modify the condition in the callback
		// To make certain tags are the only ones created.
		return true
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
	// Output:
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

func ExampleLazyDecoder_GetTag() {
	fp, err := os.Open("testdata/sample1.tiff")
	if err != nil {
		panic(err)
	}
	defer fp.Close()
	var decoder exif.LazyDecoder
	err = decoder.Decode(fp)
	if err != nil {
		panic(err)
	}

	// We can get tags from the decoder once the file has been decoded.
	// The decoder does not require the file as an argument for tags
	// that have a in-memory lazy representation. If the tag needs the reader
	// to be parsed then an error will be returned by GetTag.
	widthTag, errW := decoder.GetTag(fp, 0, exifid.ImageWidth)
	heightTag, errH := decoder.GetTag(fp, 0, exifid.ImageHeight)
	if errW != nil || errH != nil {
		panic("expected width or height tags in image")
	}
	fmt.Println(widthTag, heightTag)

	// One can assert the integer type using the Int and MustInt methods of a tag.
	compressionTag, err := decoder.GetTag(fp, 0, exifid.Compression)
	if err != nil {
		panic("compression tag not found")
	}
	fmt.Printf("the compression is %d\n", compressionTag.MustInt())

	// You can also generically print values with the Value method.
	// In this case we need to pass in the file since the XResolution
	// tag does not have a in-memory representation in the lazy decoder.
	// This may change in the future. Best practice is to always pass in the reader.
	xResTag, err := decoder.GetTag(fp, 0, exifid.XResolution)
	if err != nil {
		panic("compression tag not found")
	}
	fmt.Printf("the x resolution is %v", xResTag.Value())
	// Output:
	// ImageWidth (uint32): 1728 ImageHeight (uint32): 2376
	// the compression is 4
	// the x resolution is 2000000/10000
}
