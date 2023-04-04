# exif
Exchangeable image file format tools for Go. 

This library is at least 200 times faster for extracting EXIF data from a small 
image when compared to [go-exif](https://github.com/dsoprea/go-exif) and can be up to
thousands of times faster for images in the size of megabytes. See benchmarks below.

_This is a work in progress._

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