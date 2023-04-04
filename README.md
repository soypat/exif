# exif
Exchangeable image file format tools for Go.

_This is a work in progress._

- The root directory contains common EXIF functions and data types.
- The `tiff` directory contains a TIFF image parser that uses lazy loading.
    - TODO: Implement the `image.Image` interface using a cache for low memory requirement lazy loading
- The `rational` directory contains signed and unsigned 64bit rational number types

