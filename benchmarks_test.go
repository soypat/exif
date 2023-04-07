package exif_test

import (
	"os"
	"testing"

	"github.com/soypat/exif"
)

const (
	largeImageName = "testdata/large.tiff"
	smallImageName = "testdata/sample1.tiff"
)

func BenchmarkThisPackage_SmallImage(b *testing.B) {
	fp, err := os.Open(smallImageName)
	if err != nil {
		b.Fatal(err)
	}
	var decoder exif.LazyDecoder
	for i := 0; i < b.N; i++ {
		err := decoder.Decode(fp)
		if err != nil {
			b.Fatal(err)
		}
		_, err = decoder.MakeIFDs(fp, func(ifd, size int, id exif.ID) bool {
			return size <= 4
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
