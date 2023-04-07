package exif_test

import (
	"os"
	"testing"

	dsoprea "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
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
			return true
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDsoprea_SmallImage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		rawExif, err := dsoprea.SearchFileAndExtractExif(smallImageName)
		if err != nil {
			b.Fatal(err)
		}
		mapping, _ := exifcommon.NewIfdMappingWithStandard()
		ti := dsoprea.NewTagIndex()
		_, index, err := dsoprea.Collect(mapping, ti, rawExif)
		if err != nil {
			b.Fatal(err)
		}
		err = index.RootIfd.EnumerateTagsRecursively(func(i *dsoprea.Ifd, ite *dsoprea.IfdTagEntry) error {
			return nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
