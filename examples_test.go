package exif_test

import (
	"fmt"
	"os"
	"strings"

	"github.com/soypat/exif"
	"github.com/soypat/exif/exifid"
)

func ExampleLazyDecoder() {
	fp, err := os.Open("testdata/app1jpeg.bin")
	if err != nil {
		panic(err)
	}
	var decoder exif.LazyDecoder
	err = decoder.Decode(fp)
	if err != nil {
		panic(err)
	}
	ifds, err := decoder.MakeIFDs(fp, func(ifd, size int, id exif.ID) bool {
		return size < 128 // Tags less than a kilobyte in size.
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
	// 	ImageWidth (uint32): 2048
	// 	ImageHeight (uint32): 1536
	// 	Make (string): RaspberryPi
	// 	Model (string): RP_imx477
	// 	XResolution (rational): 72/1
	// 	YResolution (rational): 72/1
	// 	ResolutionUnit (uint16): 2
	// 	ModifyDate (string): 2023:01:04 14:18:34
	// 	YCbCrPositioning (uint16): 1
	// 	ExifOffset (unknown): 192
	// IFD1:
	// 	ImageWidth (uint32): 64
	// 	ImageHeight (uint32): 48
	// 	Compression (uint16): 6
	// 	XResolution (rational): 72/1
	// 	YResolution (rational): 72/1
	// 	ResolutionUnit (uint16): 2
	// 	ThumbnailOffset (uint32): 958
	// 	ThumbnailLength (uint32): 24576
	// SubIFD:
	// 	ExposureTime (rational): 39636/1000000
	// 	ExposureProgram (uint16): 3
	// 	ISO (uint16): 6
	// 	ExifVersion (undefined): [48]
	// 	DateTimeOriginal (string): 2023:01:04 14:18:34
	// 	CreateDate (string): 2023:01:04 14:18:34
	// 	ComponentsConfiguration (undefined): [1]
	// 	ShutterSpeedValue (urational): 4657045/1000000
	// 	BrightnessValue (urational): 0
	// 	MeteringMode (uint16): 2
	// 	Flash (uint16): 0
	// 	FlashpixVersion (undefined): [48]
	// 	ColorSpace (uint16): 1
	// 	ExifImageWidth (uint16): 2048
	// 	ExifImageHeight (uint16): 1536
	// 	InteropOffset (unknown): 822
	// 	ExposureMode (uint16): 0
	// 	WhiteBalance (uint16): 0
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
