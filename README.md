# exif
Exchangeable image file format tools for Go. 

This library is at least 200 times faster for extracting EXIF data from a small 
image when compared to [go-exif](https://github.com/dsoprea/go-exif) and can be up to
thousands of times faster for images in the size of megabytes. See benchmarks below.

- The root directory contains common EXIF functions and data types.
- The `tiff` directory contains a TIFF image parser that uses lazy loading.
    - TODO: Implement the `image.Image` interface using a cache for low memory requirement lazy loading
- The `rational` directory contains signed and unsigned 64bit rational number types


Benchmarks- comparisons with the popular go-exif library.
- Small image: 15kB TIFF
- Large image: 19.6MB TIFF

```
goos: linux
goarch: amd64
pkg: github.com/soypat/exif
cpu: 12th Gen Intel(R) Core(TM) i5-12400F
BenchmarkThisPackage_SmallImage-12    	  136064	      8411 ns/op	    2457 B/op	      57 allocs/op
BenchmarkDsoprea_SmallImage-12        	     798	   1650455 ns/op	  659437 B/op	   11252 allocs/op
BenchmarkThisPackage_LargeImage-12    	  127800	      8593 ns/op	    1977 B/op	      50 allocs/op
BenchmarkDsoprea_LargeImage-12        	      52	  23897506 ns/op	123760580 B/op	   11588 allocs/op
PASS
coverage: 41.6% of statements
```
<details><summary>Benchmark code</summary>

I have removed the following benchmark from this package since dsoprea's Go 
package has two high severity security issues and github's dependabot was
bothering me some. You are free to run it on your computer and compare with the
benchmarks under [`benchmarks_test.go`](./benchmarks_test.go) 

```go
package exif_test

import (
	"os"
	"testing"

	dsoprea "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

const (
	smallImageName = "testdata/sample1.tiff"
)

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
```

</details>

## Example
Example of usage of this library. We read a single tag value for the `XResolution` tag
and then print out all the Exif tags in the file under the Image File Directories. In this
case we only have IFD0.

```go
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

	// Read a single tag from the decoded tags.
	xResTag, err := decoder.GetTag(fp, 0, exifid.XResolution)
	if err != nil {
		panic("compression tag not found")
	}
	// One can also use the Rational, Float and Int methods to obtain 
	// statically typed tag values. The Value method returns interface{} type.
	fmt.Printf("the x resolution is %v", xResTag.Value())

	// Generate all tags of a certain constrained size using a callback.
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
```

Outputs:
```
the x resolution is 2000000/10000
IFD0:
	ImageWidth (uint32): 1728
	ImageHeight (uint32): 2376
	BitsPerSample (uint16): 1
	Compression (uint16): 4
	PhotometricInterpretation (uint16): 0
	FillOrder (uint16): 2
	DocumentName (string): Standard Input
	ImageDescription (string): converted PBM file
	StripOffsets (unknown): 8
	Orientation (uint16): 1
	SamplesPerPixel (uint16): 1
	RowsPerStrip (uint32): 2376
	StripByteCounts (unknown): 18112
	XResolution (rational): 2000000/10000
	YResolution (rational): 2000000/10000
	PlanarConfiguration (uint16): 1
	ResolutionUnit (uint16): 2
```