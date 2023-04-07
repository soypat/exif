package main

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"

	"github.com/soypat/exif"
	"github.com/soypat/exif/exifid"
)

func main() {
	fp, err := os.Open("../testdata/large.tiff")
	if err != nil {
		panic(err)
	}
	_, err = Decode(fp)
	if err != nil {
		panic(err)
	}
}

type TIFF struct {
	r             io.ReaderAt
	width, height int
	app1          int64
}

func Decode(r io.ReaderAt) (*TIFF, error) {
	var dec exif.LazyDecoder
	err := dec.Decode(r)
	if err != nil {
		return nil, err
	}
	app1 := dec.EndOfApp1()
	if app1 == 0 {
		return nil, errors.New("got zero APP1 offset")
	}
	heightTag, err := dec.GetTag(r, 0, exifid.ImageHeight)
	if err != nil {
		return nil, err
	}
	widthTag, err := dec.GetTag(r, 0, exifid.ImageWidth)
	if err != nil {
		return nil, err
	}
	height, _ := heightTag.Int()
	width, _ := widthTag.Int()
	ycbcr, err := dec.GetTag(r, 0, exifid.YCbCrPositioning)

	fmt.Println(height, width)
	image.Image
	return &TIFF{r: r, app1: app1, width: int(width), height: int(height)}, nil
}

func (tf *TIFF) Bounds() image.Rectangle {
	return image.Rect(0, 0, tf.width, tf.height)
}

func (tf *TIFF) ColorModel() color.Model {
	return color.YCbCrModel
}
