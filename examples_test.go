package exif_test

import (
	"fmt"
	"io"
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
	// We find the EXIF metdata start.
	// We can also choose to directly pass in the
	// reader to the decoder though it may not work for all images.
	offset, err := exif.FindStartOffset(fp, nil)
	if err != nil {
		panic(err)
	}
	// Limit file reading with third argument to NewSectionReader
	// If we don't want to limit the reader we can just pass in an arbitrarily large number.
	rd := io.NewSectionReader(fp, offset, offset+9999999)

	var decoder exif.LazyDecoder
	err = decoder.Decode(rd)
	if err != nil {
		panic(err)
	}
	ifds, err := decoder.MakeIFDs(rd, func(ifd, size int, id exif.ID) bool {
		return size < 128 // Only parse tags less than 128 bytes in size.
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
	// 	ResolutionUnit (uint16): inches
	// 	ModifyDate (string): 2023:01:04 14:18:34
	// 	YCbCrPositioning (uint16): Centered
	// 	ExifOffset (unknown): 192
	// IFD1:
	// 	ImageWidth (uint32): 64
	// 	ImageHeight (uint32): 48
	// 	Compression (uint16): 6
	// 	XResolution (rational): 72/1
	// 	YResolution (rational): 72/1
	// 	ResolutionUnit (uint16): inches
	// 	ThumbnailOffset (uint32): 958
	// 	ThumbnailLength (uint32): 24576
	// SubIFD:
	// 	ExposureTime (rational): 39636/1000000
	// 	ExposureProgram (uint16): Aperture-priority AE
	// 	ISO (uint16): 6
	// 	ExifVersion (undefined): [48]
	// 	DateTimeOriginal (string): 2023:01:04 14:18:34
	// 	CreateDate (string): 2023:01:04 14:18:34
	// 	ComponentsConfiguration (undefined): [1]
	// 	ShutterSpeedValue (urational): 4657045/1000000
	// 	BrightnessValue (urational): 0
	// 	MeteringMode (uint16): Center-weighted average
	// 	Flash (uint16): 0
	// 	FlashpixVersion (undefined): [48]
	// 	ColorSpace (uint16): sRGB
	// 	ExifImageWidth (uint16): 2048
	// 	ExifImageHeight (uint16): 1536
	// 	InteropOffset (unknown): 822
	// 	ExposureMode (uint16): Auto
	// 	WhiteBalance (uint16): Auto
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
	// Output:
	// IFD0:
	// 	ImageWidth (uint32): 1728
	// 	ImageHeight (uint32): 2376
	// 	BitsPerSample (uint16): 1
	// 	Compression (uint16): 4
	// 	PhotometricInterpretation (uint16): WhiteIsZero
	// 	FillOrder (uint16): Reversed
	// 	StripOffsets (unknown): 8
	// 	Orientation (uint16): Horizontal (normal)
	// 	SamplesPerPixel (uint16): 1
	// 	RowsPerStrip (uint32): 2376
	// 	StripByteCounts (unknown): 18112
	// 	PlanarConfiguration (uint16): Chunky
	// 	ResolutionUnit (uint16): inches
}
