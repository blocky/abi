// Package abi provides a minimal Go library for ABI encoding and decoding
// without reflection or code generation.
package abi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
)

func isNonZero(b []byte) bool {
	return slices.ContainsFunc(b, func(b byte) bool { return b != 0 })
}

// EncodeUint64 encodes a uint64 to 32-byte ABI format. It is the inverse
// operation of DecodeUint64.
func EncodeUint64(v uint64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, v)
	return append(make([]byte, 24), buf...)
}

// DecodeUint64 decodes ABI bytes back to uint64. It is the inverse operation
// of EncodeUint64.
func DecodeUint64(v []byte) (uint64, error) {
	if len(v) != 32 {
		return 0, errors.New("uint64 encoding must contain 32 bytes")
	}

	padding, data := v[:24], v[24:]
	if isNonZero(padding) {
		return 0, fmt.Errorf("padding contains non-zero values")
	}

	return binary.BigEndian.Uint64(data), nil
}

func padRight(data []byte, length int) ([]byte, error) {
	if length < len(data) {
		format := "length %d smaller than input %d"
		return nil, fmt.Errorf(format, length, len(data))
	}

	padded := make([]byte, length)
	copy(padded, data)
	return padded, nil
}

// SliceHeader returns the header for a slice of bytes.
func SliceHeader() []byte {
	return append(make([]byte, 31), 0x20)
}

func nextMultipleOf32(n int) int {
	remainder := n % 32
	return n + (32-remainder)%32
}

// EncodeBytes encodes a byte slice (in the go sense) to a bytes type
// (in the evm sense).  It is the inverse operation of DecodeBytes.
func EncodeBytes(v []byte) ([]byte, error) {
	vLen := len(v)
	head := EncodeUint64(uint64(vLen))
	tail, err := padRight(v, nextMultipleOf32(vLen))
	if err != nil {
		return nil, fmt.Errorf("padding, %w", err)
	}

	return append(head, tail...), nil
}

// DecodeBytes decodes a byte slice (in the go sense) from an
// abi encoding of Bytes (in the evm sense).  It is the inverse operation
// of EncodeBytes.
func DecodeBytes(abiEncoded []byte) ([]byte, error) {
	// We specify a few names to help understand the layout.
	// Note that the '|' is not part of the layout, it is just a visual aid.
	// | head (32 bytes) | tail (padded to a multiple of 32 bytes) |
	//
	// Restricting our view to just the tail we have:
	// tail = | data | padding |
	//
	// the value of head is an integer that tells us how many bytes of
	// tail are data
	//
	// note that because the head is 32 bytes and the tail is padded
	// to a multiple of 32 bytes, a valid input must always
	// have a length that is a multiple of 32.
	const headLen = uint64(32)
	abiEncodedLen := uint64(len(abiEncoded))
	switch {
	case abiEncodedLen < headLen:
		return nil, errors.New("not long enough to have a head")
	case abiEncodedLen%32 != 0:
		return nil, fmt.Errorf("invalid length '%d' not 32-byte aligned", abiEncodedLen)
	}

	// unpack the abi encoded data
	head := abiEncoded[:headLen]
	tail := abiEncoded[headLen:]

	// unpack the head
	dataLen, err := DecodeUint64(head)
	if err != nil {
		return nil, fmt.Errorf("decoding data length, %w", err)
	}

	// validate the content in the head
	if dataLen > uint64(len(tail)) {
		return nil, fmt.Errorf("length in head is out of range")
	}

	// unpack the tail
	data := tail[:dataLen]
	padding := tail[dataLen:]

	// validate the content in the tail
	switch {
	case len(padding) >= 32:
		return nil, fmt.Errorf("invalid padding length '%d'", len(padding))
	case isNonZero(padding):
		return nil, fmt.Errorf("padding contains non-zero values")
	}

	dst := make([]byte, dataLen)
	copy(dst, data)
	return dst, nil
}

// EncodeSliceOfBytes encodes a slice of byte arrays (in the go sense) to a
// bytes type (in the evm sense).  It is the inverse operation of
// DecodeSliceOfBytes.
func EncodeSliceOfBytes(v [][]byte) ([]byte, error) {
	var head, tail bytes.Buffer

	// collect the data needed for the head
	vLen := uint64(len(v))

	// write the head
	head.Write(SliceHeader())
	head.Write(EncodeUint64(vLen))

	// Compute the initial offset.
	// The 32*vLen are for the start locations of each element in the slice.
	// We will add this data to the head as we build up the tail.
	offset := uint64(32 * vLen)
	for i, vi := range v {
		head.Write(EncodeUint64(offset))
		encoded, err := EncodeBytes(vi)
		if err != nil {
			return nil, fmt.Errorf("encoding element %d, %w", i, err)
		}

		offset += uint64(len(encoded))
		tail.Write(encoded)
	}

	return append(head.Bytes(), tail.Bytes()...), nil
}

// DecodeSliceOfBytes decodes a slice of byte arrays (in the go sense) from an
// abi encoding of Bytes (in the evm sense).  It is the inverse operation
// of EncodeSliceOfBytes.
func DecodeSliceOfBytes(abiEncoded []byte) ([][]byte, error) {
	// We specify a few names to help understand the layout.
	// Note that the '|' is not part of the layout, it is just a visual aid.
	//
	// Assume that we encoded a slice of k bytes.
	// | head 64 byte | tail (padded to a multiple of 32 bytes) |
	//
	// Restricting our view to just the head we have
	// head = | type (32 bytes) | num elts 32 bytes) |
	//
	// Restricting our view to just the tail we have
	// tail = | offsets (32*k bytes) | elements (each 32-byte aligned) |
	//
	// Restricting our view to just the elements we have
	// elements = | elt1 | elt2 | ... | eltk |
	// where each elt is aligned to 32 bytes.
	//
	// note that because the head is 64 the offsets are 32*k bytes
	// and each element is padded to a multiple of 32 bytes,
	// a valid input must always have a length that is a multiple of 32.

	headLen := uint64(64)
	abiEncodedLen := uint64(len(abiEncoded))
	switch {
	case abiEncodedLen < headLen:
		return nil, errors.New("not long enough to have a head")
	case abiEncodedLen%32 != 0:
		return nil, fmt.Errorf("invalid length '%d' not 32-byte aligned", abiEncodedLen)
	}

	// unpack the abi encoded data
	head := abiEncoded[:headLen]
	tail := abiEncoded[headLen:]
	tailLen := uint64(len(tail))

	// unpack the head
	typeBytes := head[:32]
	eltCountBytes := head[32:]
	eltCount, err := DecodeUint64(eltCountBytes)
	if err != nil {
		return nil, fmt.Errorf("decoding element count, %w", err)
	}

	// validate the head data
	offsetsLen := 32 * eltCount
	switch {
	case !bytes.Equal(typeBytes, SliceHeader()):
		return nil, errors.New("not a slice type")
	case offsetsLen > tailLen:
		return nil, fmt.Errorf("tail too short for %d elements", eltCount)
	}

	// unpack the offsets
	offsets := make([]uint64, eltCount+1)
	for i := range eltCount {
		offset, err := DecodeUint64(tail[i*32 : (i+1)*32])
		switch {
		case err != nil:
			return nil, fmt.Errorf("decoding offset for index %d, %w", i, err)
		case offset >= tailLen:
			return nil, fmt.Errorf("offset at index %d out of bounds", i)
		}

		offsets[i] = offset
	}
	offsets[eltCount] = tailLen

	// use offsets to read and decode each encoded byte array
	results := make([][]byte, eltCount)
	for i := range eltCount {
		start := offsets[i]
		end := offsets[i+1]
		if start >= end {
			return nil, fmt.Errorf("start %d greater than end %d", start, end)
		}

		results[i], err = DecodeBytes(tail[start:end])
		if err != nil {
			return nil, fmt.Errorf("decoding element %d, %w", i, err)
		}
	}

	return results, nil
}

// EncoderResult is the result of encoding a single element.  It is intended
// to be used as the return value of an EncoderFunc. While it is exported,
// it is not intended to be used directly by users as it is part of the
// glue for the TupleEncoder and TupleDecoder.
type EncoderResult struct {
	indirect bool
	data     []byte
}

// EncoderFunc is a function that encodes a single element.  It works in
// concert with the TupleEncoder to encode a tuple.
type EncoderFunc func() (EncoderResult, error)

// EncodeTuple encodes a tuple of elements.  While one can use the EncodeTuple
// function directly, because of its simpler interface, it is recommended to
// use the TupleEncoder instead.
func EncodeTuple(encoders ...EncoderFunc) ([]byte, error) {
	var head, tail bytes.Buffer

	offset := uint64(32 * len(encoders))
	for _, encode := range encoders {
		result, err := encode()
		if err != nil {
			return nil, fmt.Errorf("encoding: %w", err)
		}

		if !result.indirect {
			head.Write(result.data)
			continue
		}

		head.Write(EncodeUint64(offset))
		tail.Write(result.data)
		offset += uint64(len(result.data))
	}

	return append(head.Bytes(), tail.Bytes()...), nil
}

// EncodeTupleFuncUint64 encodes a uint64 as the k-th element of a tuple.
func EncodeTupleFuncUint64(v uint64) EncoderFunc {
	return func() (EncoderResult, error) {
		data := EncodeUint64(v)
		return EncoderResult{indirect: false, data: data}, nil
	}
}

// EncodeTupleFuncBytes encodes a byte slice as the k-th element of a tuple.
func EncodeTupleFuncBytes(v []byte) EncoderFunc {
	return func() (EncoderResult, error) {
		data, err := EncodeBytes(v)
		if err != nil {
			return EncoderResult{}, fmt.Errorf("encoding: %w", err)
		}

		return EncoderResult{indirect: true, data: data}, nil
	}
}

// TupleEncoder is a helper for encoding a tuple of elements.  The struct
// is used in building a fluent API for encoding a tuple.
type TupleEncoder struct {
	encoders []EncoderFunc
}

// NewTupleEncoder creates a new TupleEncoder.
func NewTupleEncoder() *TupleEncoder {
	return &TupleEncoder{
		encoders: []EncoderFunc{},
	}
}

// Uint64 encodes a uint64 as the k-th element of a tuple.
func (e *TupleEncoder) Uint64(v uint64) *TupleEncoder {
	encoder := EncodeTupleFuncUint64(v)
	e.encoders = append(e.encoders, encoder)
	return e
}

// Bytes encodes a byte slice as the k-th element of a tuple.
func (e *TupleEncoder) Bytes(v []byte) *TupleEncoder {
	encoder := EncodeTupleFuncBytes(v)
	e.encoders = append(e.encoders, encoder)
	return e
}

// Encode encodes the tuple.
func (e *TupleEncoder) Encode() ([]byte, error) {
	return EncodeTuple(e.encoders...)
}

// DecoderFunc is a function that decodes a single element.  It works in
// concert with the TupleDecoder to decode a tuple.
type DecoderFunc func(cur, full []byte) error

// DecodeTuple decodes a tuple of elements.  While one can use the DecodeTuple
// function directly, because of its simpler interface, it is recommended to
// use the TupleDecoder instead.
func DecodeTuple(data []byte, decoders ...DecoderFunc) error {
	// We specify a few names to help understand the layout.
	// Note that the '|' is not part of the layout, it is just a visual aid.
	//
	// Assume that we have encoded a k-tuple.
	// | head (32*k bytes) | tail (32-bytes aligned) |
	//
	// Restricting our view to just the head we have
	// head = | elt1 | elt2 | ... | eltk |
	// where each elt is aligned to 32 bytes.
	//
	// For each element, we have that either it is a value, such as
	// a 64-bit integer, or it is an offset to a value, such as bytes.
	// For a element that is a value, we can decode it directly,
	// for a element that is an offset, we need  to do additional work.
	// Either way, that additional work is decided by the specific decoder.
	switch {
	case len(decoders) == 0:
		return errors.New("no decoders provided")
	case len(data) < 32*len(decoders):
		return errors.New("not long enough to support all decoders")
	}

	for i, decode := range decoders {
		cur := data[i*32 : (i+1)*32]
		err := decode(cur, data)
		if err != nil {
			return fmt.Errorf("decoding element %d: %w", i, err)
		}
	}
	return nil
}

// DecodeTupleFuncUint64 decodes a uint64 as the k-th element of a tuple.
func DecodeTupleFuncUint64(v *uint64) DecoderFunc {
	return func(cur, full []byte) error {
		vv, err := DecodeUint64(cur[:])
		if err != nil {
			return fmt.Errorf("decoding: %w", err)
		}

		*v = vv
		return nil
	}
}

// DecodeTupleFuncBytes decodes a byte slice as the k-th element of a tuple.
func DecodeTupleFuncBytes(v *[]byte) DecoderFunc {
	return func(cur, full []byte) error {
		// We specify a few names to help understand the layout.
		// Note that the '|' is not part of the layout, it is just a visual aid.
		//
		// Assume that we are processing the k-th element of an n-tuple
		// and so our input of full is
		// | head (32*n bytes) | tail (32-bytes aligned) |
		//
		// Restricting our view to just the head we have
		// | elt1 | elt2 | ... | eltk | elt(k+1) | ... | eltn |
		// where each elt is aligned to 32 bytes.
		//
		// We expect that cur is bytes of eltk
		// those bytes will tell us the offset into full where
		// we find the start of the bytes that we need to decode.
		//
		// Recall that bytes are encoded such that the first 32 bytes
		// are the length of the data followed by the data itself,
		// padded to 32 bytes.  First, we will get the byte count
		// so that we know which slice from full to decode.
		// And then decode using some helper functions.

		offset, err := DecodeUint64(cur)
		switch {
		case err != nil:
			return fmt.Errorf("decoding offset: %w", err)
		case offset+32 > uint64(len(full)):
			return fmt.Errorf("offset+32 out of bounds")
		}

		byteCountBytes := full[offset : offset+32]
		byteCount, err := DecodeUint64(byteCountBytes)
		if err != nil {
			return fmt.Errorf("decoding length : %w", err)
		}

		alignedByteCount := nextMultipleOf32(int(byteCount))
		start := int(offset)
		end := start + 32 + alignedByteCount
		if end > len(full) {
			return fmt.Errorf("end is out of bounds")
		}

		alignedBytes := full[start:end]
		vv, err := DecodeBytes(alignedBytes)
		if err != nil {
			return fmt.Errorf("decoding bytes: %w", err)
		}

		*v = vv
		return nil
	}
}

// TupleDecoder is a helper for decoding a tuple of elements.  The struct
// is used in building a fluent API for decoding a tuple.
type TupleDecoder struct {
	decoders []DecoderFunc
}

// NewTupleDecoder creates a new TupleDecoder.
func NewTupleDecoder() *TupleDecoder {
	return &TupleDecoder{
		decoders: []DecoderFunc{},
	}
}

// Decode decodes the tuple.
func (d *TupleDecoder) Decode(data []byte) error {
	return DecodeTuple(data, d.decoders...)
}

// Uint64 decodes a uint64 as the k-th element of a tuple.
func (d *TupleDecoder) Uint64(v *uint64) *TupleDecoder {
	decoder := DecodeTupleFuncUint64(v)
	d.decoders = append(d.decoders, decoder)
	return d
}

// Bytes decodes a byte slice as the k-th element of a tuple.
func (d *TupleDecoder) Bytes(v *[]byte) *TupleDecoder {
	decoder := DecodeTupleFuncBytes(v)
	d.decoders = append(d.decoders, decoder)
	return d
}
