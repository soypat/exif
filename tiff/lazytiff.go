package tiff

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unsafe"

	"github.com/soypat/exif"
)

type TiffLazy struct {
	dirs       []lazydir
	order      binary.ByteOrder
	baseOffset int64
}

func (lt *TiffLazy) MakeTags(r io.ReaderAt, f func(_ exif.IFD, id exif.ID) bool) ([]exif.Tag, error) {
	r = &offsetReaderAt{r: r, offset: lt.baseOffset}
	var tags []exif.Tag
	for _, dir := range lt.dirs {
		for _, tag := range dir.Tags {
			if tag.length != 0 || !f(exif.IFD{}, tag.ID) {
				continue // skip tag.
			}
			sz := tag.Type.Size()
			v, err := exif.EvaluateData(tag.Type, lt.order, tag.arrayptr()[:sz])
			if err != nil {
				return nil, err
			}
			newtag, err := exif.NewTag(tag.ID, v)
			if err != nil && newtag.Value() == nil {
				continue // bad data
			}
			tags = append(tags, newtag)
		}
	}
	return tags, nil
}

func LazyDecode(r io.ReaderAt) (lt *TiffLazy, err error) {
	lt = new(TiffLazy)
	var buf [8]byte
	n, err := r.ReadAt(buf[:], 0)
	if err != nil {
		return nil, err
	}
	if n != len(buf) {
		return nil, errors.New("wanted to read 10 starting bytes, only read " + strconv.Itoa(n))
	}
	start := string(buf[:2])
	if start == "\xff\xd8" {
		// start of image found.
		r.ReadAt(buf[:], 12)
		r = &offsetReaderAt{r: r, offset: 12}
		lt.baseOffset = 12
	}
	var order binary.ByteOrder
	switch string(buf[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return nil, errors.New("failed reading TIFF byte order")
	}
	lt.order = order
	//
	specialMarker := order.Uint16(buf[2:])
	if specialMarker != 42 {
		return nil, errors.New("failed to find special marker")
	}
	// read offset to first IFD and load them.
	offset := int64(order.Uint32(buf[4:]))
	for offset != 0 {
		d, next, err := decodeDir(r, offset, order)
		if err != nil {
			return nil, err
		}
		if next == offset {
			return nil, errors.New("recursive dir")
		}
		offset = next
		lt.dirs = append(lt.dirs, d)
	}
	return lt, nil
}

type lazydir struct {
	Tags []lazytag
}

type offsetReaderAt struct {
	r      io.ReaderAt
	offset int64
}

func (or *offsetReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	return or.r.ReadAt(p, off+or.offset)
}

func decodeDir(r io.ReaderAt, offset int64, order binary.ByteOrder) (d lazydir, nextOffset int64, err error) {
	var buf [32]byte
	n, err := r.ReadAt(buf[:2], offset)
	if err != nil {
		return d, 0, fmt.Errorf("while seeking offset at %d: %w", n, err)
	}
	if n != 2 {
		return d, 0, errors.New("expected read 2 bytes at offset, got " + strconv.Itoa(n))
	}
	nTags := order.Uint16(buf[:2])
	// load tags
	totalOffset := offset + 2
	for n := 0; n < int(nTags); n++ {
		t, err := decodeTag(r, totalOffset, order)
		if err != nil {
			return d, 0, err
		}
		d.Tags = append(d.Tags, t)
		totalOffset += 12 // size of tag field.
	}
	n, err = r.ReadAt(buf[:4], totalOffset)
	if err != nil {
		return d, 0, err
	}
	if n != 4 {
		return d, 0, errors.New("read less than wanted")
	}
	nextOffset = int64(order.Uint32(buf[:4]))
	return d, nextOffset, nil
}

type lazytag struct {
	ID            exif.ID
	Type          exif.Type
	offsetOrValue uint32
	length        uint32
}

func (lt *lazytag) arrayptr() *[4]byte {
	return (*[4]byte)(unsafe.Pointer(&lt.offsetOrValue))
}

func decodeTag(r io.ReaderAt, offset int64, order binary.ByteOrder) (tg lazytag, err error) {
	var buf [12]byte
	n, err := r.ReadAt(buf[:], offset)
	if err != nil {
		return tg, err
	}
	if n != len(buf) {
		return tg, errors.New("reading tag got short read (" + strconv.Itoa(n) + ")")
	}
	tg.ID = exif.ID(order.Uint16(buf[0:]))
	tg.Type = exif.Type(order.Uint16(buf[2:]))
	// if tg.ID.Type() != tg.Type {
	//   err = fmt.Errorf("type mismatch for tag ID %q(%#x), got %s, expected %s", tg.ID.String(), uint16(tg.ID), tg.Type.String(), tg.ID.Type().String())
	// }
	count := order.Uint32(buf[4:])
	if count == 1<<32-1 {
		return tg, errors.New("invalid count offset in tag")
	}
	sz := tg.Type.Size()
	if sz == 0 || sz > 8 {
		return tg, errors.New("invalid tag type: " + strconv.Itoa(int(tg.Type)))
	}
	length := count * uint32(sz)
	valueBuf := buf[8:12]
	if length > 4 {
		tg.offsetOrValue = order.Uint32(valueBuf)
		tg.length = length

	} else {
		arr := tg.arrayptr()
		copy(arr[:], valueBuf)
		_ = arr // Place breakpoints for debugging.
	}
	return tg, nil
}
