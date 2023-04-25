package exif

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unsafe"
)

type LazyDecoder struct {
	dirs       []lazydir
	order      binary.ByteOrder
	baseOffset int64
	app1Size   [2]byte
	buf        [8]byte
}

// MakeIFDs processes the collected tags in the LazyDecoder (obtained from a previous call to Decode)
// and creates the corresponding EXIF tags.
// If a nil reader is passed into MakeIFDs then only the tags which have a
// lazy in-memory representation will be returned.
// The callback passed in to MakeIFDs will decide if a tag is made or skipped depending
// on whether the call returns true or false. The callback has the ifd level, size in bytes
// and the tag's ID to decide whether to create the tag and allocate memory for it.
func (lt *LazyDecoder) MakeIFDs(r io.ReaderAt, fn func(ifd, size int, id ID) bool) ([]IFD, error) {
	if fn == nil {
		return nil, errors.New("nil callback")
	}
	if r != nil {
		r = newOffsetReaderAt(r, lt.baseOffset, nil)
	}
	var ifds []IFD
	for ifd, dir := range lt.dirs {
		tags := make([]Tag, 0, len(dir.Tags))
		for i, lztag := range dir.Tags {
			sz := lztag.size()
			if r == nil && lztag.dataOffset() != 0 {
				continue // Nil reader means no way to read from file.
			}
			if !fn(ifd, sz, lztag.ID) {
				continue // User decides to skip tag.
			}
			tag, err := lt.getTag(r, lztag)
			if err != nil {
				// Return correctly generated tags up to the point of failure.
				return append(ifds, IFD{Tags: tags[:i], Group: dir.Group}), err
			}
			tags = append(tags, tag)
		}
		ifds = append(ifds, IFD{Tags: tags, Group: dir.Group})
	}
	return ifds, nil
}

func (lt *LazyDecoder) getTag(r io.ReaderAt, lztag lazytag) (tag Tag, err error) {
	if dataOffset := lztag.dataOffset(); dataOffset != 0 {
		// 8-byte values or variable length value are stored at an offset position.
		var data []byte
		if lztag.length == 8 && lztag.Type != TypeUndefined {
			data = lt.buf[:8]
		} else {
			data = make([]byte, lztag.length)
		}

		n, err := r.ReadAt(data, int64(dataOffset))
		if err != nil {
			return Tag{}, fmt.Errorf("reading %d/%d exif data at %#x: %w", n, lztag.length, dataOffset, err)
		}
		if n != int(lztag.length) {
			return Tag{}, errors.New("incomplete read")
		}
		v, err := DecodeTypeData(lztag.ID.Type(), lt.order, data)
		if err != nil {
			return Tag{}, err
		}
		tag = Tag{ID: lztag.ID, data: v}

	} else {
		// 1, 2 or 4 byte length values, stored in place.
		sz := lztag.Type.Size()
		v, err := DecodeTypeData(lztag.Type, lt.order, lztag.arrayptr()[:sz])
		if err != nil {
			return Tag{}, err
		}
		tag = Tag{ID: lztag.ID, data: v}
	}
	return tag, nil
}

func (lt *LazyDecoder) GetTag(r io.ReaderAt, ifdLevel int, id ID) (_ Tag, err error) {
	switch {
	case len(lt.dirs) == 0:
		err = errors.New("decoder empty: did decoding succeed?")
	case ifdLevel > len(lt.dirs):
		err = errors.New("IFD level exceeds available levels")
	}
	if err != nil {
		return Tag{}, err
	}
	ifdTags := lt.dirs[ifdLevel].Tags
	for _, lztag := range ifdTags {
		if lztag.ID == id {
			if r == nil && lztag.dataOffset() != 0 {
				return Tag{}, errors.New("need non-nil reader to read tag " + id.String())
			}
			return lt.getTag(r, lztag)
		}
	}
	return Tag{}, errors.New("tag ID not found in IFD")
}

// Decode marshals exif data in r lazily. It only stores values that have a
// constrained in-memory representation.
func (lt *LazyDecoder) Decode(r io.ReaderAt) (err error) {
	*lt = LazyDecoder{}
	var buf [8]byte
	n, err := r.ReadAt(buf[:], 0)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return errors.New("wanted to read 10 starting bytes, only read " + strconv.Itoa(n))
	}
	start := string(buf[:2])
	if start == "\xff\xd8" {
		// start of image found.
		copy(lt.app1Size[:2], buf[4:])
		r.ReadAt(buf[:], 12)
		r = newOffsetReaderAt(r, 12, nil)
		lt.baseOffset = 12
	}
	var order binary.ByteOrder
	switch string(buf[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return errors.New("failed reading EXIF byte order")
	}
	lt.order = order
	//
	specialMarker := order.Uint16(buf[2:])
	if specialMarker != 42 {
		return errors.New("failed to find special marker")
	}
	// read offset to first IFD and load them.
	offset := int64(order.Uint32(buf[4:]))
	if offset == 0 {
		return errors.New("zero IFD0 offset")
	}
	group := GroupIFD0
	for offset != 0 {
		d, next, err := decodeDir(r, offset, order)
		if err != nil {
			return err
		}
		if next == offset {
			return errors.New("recursive dir")
		}
		offset = next
		if group < GroupSubIFD {
			d.Group = group
			group++
		}
		lt.dirs = append(lt.dirs, d)
	}
	var subIFDOffset uint32
	for _, tag := range lt.dirs[0].Tags {
		if tag.ID == 0x8769 && tag.dataOffset() == 0 { // Check ExifOffset ID.
			subIFDOffset = lt.order.Uint32(tag.arrayptr()[:4])
		}
	}
	if subIFDOffset == 0 {
		return nil
	}

	offset = int64(subIFDOffset)
	for offset != 0 {
		d, next, err := decodeDir(r, offset, order)
		if err != nil {
			return err
		}
		if next == offset {
			return errors.New("recursive dir")
		}
		offset = next
		d.Group = GroupSubIFD
		lt.dirs = append(lt.dirs, d)
	}
	return nil
}

// EndOfApp1 returns the end of the APP1 segment with EXIF metadata.
// This is only set when decoding images and not
// just pure EXIF data.
func (e *LazyDecoder) EndOfApp1() int64 {
	return int64(e.order.Uint16(e.app1Size[:])) + 4 // App1 length does not include SOI and APP1 markers (4 bytes).
}

type lazydir struct {
	Tags  []lazytag
	Group Group
}

func decodeDir(r io.ReaderAt, offset int64, order binary.ByteOrder) (d lazydir, nextOffset int64, err error) {
	var buf [32]byte
	n, err := r.ReadAt(buf[:2], offset)
	if err != nil {
		return d, 0, fmt.Errorf("while seeking offset at %d: %w", offset, err)
	}
	if n != 2 {
		return d, 0, errors.New("expected read 2 bytes at offset, got " + strconv.Itoa(n))
	}
	nTags := order.Uint16(buf[:2])
	d.Tags = make([]lazytag, nTags)
	// load tags
	totalOffset := offset + 2
	for n := 0; n < int(nTags); n++ {
		t, err := decodeTag(r, totalOffset, order)
		if err != nil {
			return d, 0, err
		}
		d.Tags[n] = t
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
	offsetOrValue uint32
	ID            ID
	Type          Type
	// Size in bytes of field if array, else 0.
	length int
}

func (lt *lazytag) dataOffset() uint32 {
	if lt.length == 0 {
		return 0 // No data, only value.
	}
	return lt.offsetOrValue
}

func (lt *lazytag) size() int {
	if lt.length == 0 {
		return int(lt.Type.Size())
	}
	return lt.length
}

func (lt *lazytag) arrayptr() *[4]byte {
	return (*[4]byte)(unsafe.Pointer(&lt.offsetOrValue))
}

func decodeTag(r io.ReaderAt, offset int64, order binary.ByteOrder) (lztag lazytag, err error) {
	var buf [12]byte
	n, err := r.ReadAt(buf[:], offset)
	if err != nil {
		return lztag, err
	}
	if n != len(buf) {
		return lztag, errors.New("reading tag got short read (" + strconv.Itoa(n) + ")")
	}
	lztag.ID = ID(order.Uint16(buf[0:]))
	lztag.Type = Type(order.Uint16(buf[2:]))
	count := order.Uint32(buf[4:])
	if count == 1<<32-1 {
		return lztag, errors.New("invalid count offset in tag")
	}
	sz := lztag.Type.Size()
	if sz == 0 || sz > 8 {
		return lztag, errors.New("invalid tag type: " + strconv.Itoa(int(lztag.Type)))
	}
	length := int(count) * int(sz)
	valueBuf := buf[8:12]
	if length > 4 {
		lztag.offsetOrValue = order.Uint32(valueBuf)
		lztag.length = length

	} else {
		arr := lztag.arrayptr()
		copy(arr[:], valueBuf)
		_ = arr // Place breakpoints for debugging.
	}
	return lztag, nil
}

type offsetReaderAt struct {
	r         io.ReaderAt
	offset    int64
	buf       []byte
	bufOffset int64
}

func (or *offsetReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	off += or.offset // Work in underlying reader coordinates from here on out.
	if len(p) < len(or.buf) {
		// Only perform fast in-memory copy for buffers smaller than underlying buffer.
		end := off + int64(len(p))
		bufStart, bufEnd := or.buflims()
		if off >= bufStart && end <= bufEnd {
			start := off - bufStart
			n := copy(p, or.buf[start:start+int64(len(p))])
			return n, nil
		}
		// If we miss the buffer then we reload file contents into memory.
		nn, err := or.r.ReadAt(or.buf[:cap(or.buf)], off)
		if err != nil && nn < len(p) {
			return nn, err // Read less than capacity of p.
		}
		or.buf = or.buf[:nn]
		or.bufOffset = off
		return copy(p, or.buf[:len(p)]), nil
	}
	return or.r.ReadAt(p, off)
}

func newOffsetReaderAt(r io.ReaderAt, baseOffset int64, buf []byte) *offsetReaderAt {
	const bufSize = 64
	if buf == nil {
		buf = make([]byte, bufSize)
	}
	return &offsetReaderAt{
		r:         r,
		buf:       buf,
		bufOffset: -1,
		offset:    baseOffset,
	}
}

func (or *offsetReaderAt) buflims() (start, end int64) {
	if or.bufOffset < 0 {
		return 0, 0
	}
	return or.bufOffset, or.bufOffset + int64(len(or.buf))
}
