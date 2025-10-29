package abi_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/blocky/abi"
)

func nZeros(n int) []byte {
	return make([]byte, n)
}

func TestEncodeUint64(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		input := uint64(3)
		want := append(nZeros(31), 3)
		//when
		got := abi.EncodeUint64(input)
		//them
		assert.Equal(t, want, got)
	})
}

func TestDecodeUint64(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		input := append(nZeros(31), 3)
		want := uint64(3)
		// when
		got, err := abi.DecodeUint64(input)
		require.NoError(t, err)
		//then
		assert.Equal(t, want, got)
	})

	t.Run("not 32 bytes, too short", func(t *testing.T) {
		// given
		input := []byte("20-bytes-xxxxxxxxxxx")
		// when
		_, err := abi.DecodeUint64(input)
		// then
		assert.ErrorContains(t, err, "must contain 32 bytes")
	})

	t.Run("not 32 bytes, too long", func(t *testing.T) {
		// given
		input := []byte("40-bytes-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
		// when
		_, err := abi.DecodeUint64(input)
		// then
		assert.ErrorContains(t, err, "must contain 32 bytes")
	})

	t.Run("bad padding", func(t *testing.T) {
		// given
		input := append(nZeros(31), 3)
		input[0] = 1
		// when
		_, err := abi.DecodeUint64(input)
		// then
		assert.ErrorContains(t, err, "padding contains non-zero values")
	})
}

func TestEncodeDecodeUint64Roundtrip(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		input := uint64(3)
		// when
		data := abi.EncodeUint64(input)
		got, err := abi.DecodeUint64(data)
		require.NoError(t, err)
		// then
		assert.Equal(t, input, got)
	})
}

func abiEncodeAByte(v byte) []byte {
	want := append(nZeros(31), 1)      // there is 1 element
	want = append(want, v)             // the element
	want = append(want, nZeros(31)...) // padding
	return want
}

func TestEncodeBytes(t *testing.T) {

	t.Run("happy path", func(t *testing.T) {
		// given
		input := byte(93)
		want := abiEncodeAByte(input)

		// when
		got, err := abi.EncodeBytes([]byte{input})
		require.NoError(t, err)

		// then
		assert.Equal(t, want, got)
	})

	t.Run("empty", func(t *testing.T) {
		// given
		input := []byte{}
		want := nZeros(32)

		// when
		got, err := abi.EncodeBytes(input)
		require.NoError(t, err)

		// then
		assert.Equal(t, want, got)
	})
}

func TestDecodeBytes(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		want := []byte{93}
		input := abiEncodeAByte(want[0])

		// when
		got, err := abi.DecodeBytes(input)
		require.NoError(t, err)

		// then
		assert.Equal(t, want, got)
	})

	t.Run("empty", func(t *testing.T) {
		// given
		input := nZeros(32)
		want := []byte{}

		// when
		got, err := abi.DecodeBytes(input)
		require.NoError(t, err)

		// then
		assert.Equal(t, want, got)
	})

	t.Run("too short to have a header", func(t *testing.T) {
		// given
		input := []byte("too-short")
		// when
		_, err := abi.DecodeBytes(input)
		// then
		assert.ErrorContains(t, err, "not long enough to have a head")
	})

	t.Run("not 32-byte aligned", func(t *testing.T) {
		// given
		input, err := abi.EncodeBytes([]byte("some-bytes"))
		require.NoError(t, err)
		input = append(input, nZeros(22)...)
		// when
		_, err = abi.DecodeBytes(input)
		// then
		assert.ErrorContains(t, err, "not 32-byte aligned")
	})

	t.Run("length in header is invalid", func(t *testing.T) {
		// given
		input, err := abi.EncodeBytes([]byte("some-bytes"))
		require.NoError(t, err)
		// byte [0,32) encode the length of the array.
		// The length should be 24 0s followed by a binary encoding
		// of the length of the payload.
		// So we set a byte that is supposed to be zero to 1,
		// which is not a valid encoding.
		input[4] = 1

		// when
		_, err = abi.DecodeBytes(input)

		// then
		assert.ErrorContains(t, err, "decoding data length")
	})

	t.Run("length in header is out of range", func(t *testing.T) {
		// given
		bodyLen := 32
		// set the length of the payload
		input := abi.EncodeUint64(uint64(bodyLen + 1))
		// set the body to be smaller than the length specified in the header
		input = append(input, nZeros(bodyLen)...)

		// when
		_, err := abi.DecodeBytes(input)

		// then
		assert.ErrorContains(t, err, "length in head is out of range")
	})

	t.Run("padding unexpected length too short", func(t *testing.T) {
		// given
		input := abi.EncodeUint64(1)
		input = append(input, 3)
		input = append(input, nZeros(22)...)

		// when
		_, err := abi.DecodeBytes(input)

		// then
		assert.ErrorContains(t, err, "not 32-byte aligned")
	})

	t.Run("padding unexpected length too long 32-bytes", func(t *testing.T) {
		// given
		input := abi.EncodeUint64(32)
		input = append(input, []byte("32-bytes-xxxxxxxxxxxxxxxxxxxxxxx")...)
		input = append(input, nZeros(32)...)

		// when
		_, err := abi.DecodeBytes(input)

		// then
		assert.ErrorContains(t, err, "invalid padding length")
	})

	t.Run("padding unexpected length too long", func(t *testing.T) {
		// given
		input := abi.EncodeUint64(1)
		input = append(input, 3)
		input = append(input, nZeros(31)...)
		input = append(input, nZeros(32)...)

		// when
		_, err := abi.DecodeBytes(input)

		// then
		assert.ErrorContains(t, err, "invalid padding length")
	})

	t.Run("padding has non-zero values", func(t *testing.T) {
		// given
		input := abi.EncodeUint64(1)
		input = append(input, 3)
		// next we tack on the correct amount of padding (31 bytes)
		// but because we put a non-zero value in the padding, it is not valid
		input = append(input, nZeros(30)...)
		input = append(input, 7)

		// when
		_, err := abi.DecodeBytes(input)

		// then
		assert.ErrorContains(t, err, "padding contains non-zero values")
	})
}

func TestEncodeDecodeBytesRoundTrip(t *testing.T) {
	for name, input := range map[string][]byte{
		"empty":       {},
		"one-byte":    {1},
		"a-few-bytes": []byte("hello"),
		"multi-lines": []byte("40-bytes-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"),
	} {
		t.Run(name, func(t *testing.T) {
			// when
			encoded, err := abi.EncodeBytes(input)
			require.NoError(t, err)

			got, err := abi.DecodeBytes(encoded)
			require.NoError(t, err)

			// then
			assert.Equal(t, input, got)
		})
	}
}

func TestEncodeSliceOfBytes(t *testing.T) {
	for _, tc := range testData.sliceOfBytes {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, err := abi.EncodeSliceOfBytes(tc.native)
			require.NoError(t, err)

			// then
			assert.Equal(t, tc.encoded, got)
		})
	}
}

func TestDecodeSliceOfBytes(t *testing.T) {
	someBytes := [][]byte{[]byte("some-bytes")}

	for _, tc := range testData.sliceOfBytes {
		t.Run(tc.name, func(t *testing.T) {
			// when
			got, err := abi.DecodeSliceOfBytes(tc.encoded)
			require.NoError(t, err)

			// then
			assert.Equal(t, tc.native, got)
		})
	}

	t.Run("too short to have a header", func(t *testing.T) {
		// given
		input := []byte("too-short")
		// when
		_, err := abi.DecodeSliceOfBytes(input)
		// then
		assert.ErrorContains(t, err, "not long enough to have a head")
	})

	t.Run("not 32-byte aligned", func(t *testing.T) {
		// given
		input, err := abi.EncodeSliceOfBytes(someBytes)
		require.NoError(t, err)
		input = append(input, nZeros(22)...)
		// when
		_, err = abi.DecodeSliceOfBytes(input)
		// then
		assert.ErrorContains(t, err, "not 32-byte aligned")
	})

	t.Run("length in header is invalid", func(t *testing.T) {
		// given
		input, err := abi.EncodeSliceOfBytes(someBytes)
		require.NoError(t, err)
		// byte [32,64) encode the length of the array.
		// The length should be 24 0s followed by a binary encoding
		// of the length of the payload.
		// So we set a byte that is supposed to be zero to 1,
		// which is not a valid encoding.
		input[38] = 1

		// when
		_, err = abi.DecodeSliceOfBytes(input)

		// then
		assert.ErrorContains(t, err, "decoding element count")
	})

	t.Run("type is not a slice", func(t *testing.T) {
		// given
		input, err := abi.EncodeSliceOfBytes(someBytes)
		require.NoError(t, err)
		// byte [0,32) encode the type.
		// The value should be 30 0s followed by a 2 followed by a 0.
		// So we set a byte that is supposed to be zero to 1,
		// which is not a valid encoding.
		input[2] = 1

		// when
		_, err = abi.DecodeSliceOfBytes(input)

		// then
		assert.ErrorContains(t, err, "not a slice type")
	})

	t.Run("too many elements for length of tail", func(t *testing.T) {
		// given
		// setup for a slice with 2 elements but only put enough data for 1
		input := abi.SliceHeader()
		input = append(input, abi.EncodeUint64(2)...)
		// set the body to be smaller than the length specified in the header
		input = append(input, nZeros(32)...)

		// when
		_, err := abi.DecodeSliceOfBytes(input)

		// then
		assert.ErrorContains(t, err, "tail too short for 2 elements")
	})

	t.Run("offset is invalid", func(t *testing.T) {
		// given
		input, err := abi.EncodeSliceOfBytes(someBytes)
		require.NoError(t, err)
		// bytes [0, 64) encode head
		// bytes [64, 96) encode the offset
		// set the offest so that it is not a valid uint64
		input[64] = 1

		// when
		_, err = abi.DecodeSliceOfBytes(input)

		// then
		assert.ErrorContains(t, err, "decoding offset for index 0")
	})

	t.Run("offsets reversed", func(t *testing.T) {
		// given
		input, err := abi.EncodeSliceOfBytes([][]byte{
			[]byte("first"),
			[]byte("second"),
		})
		require.NoError(t, err)
		// bytes [0, 64) encode head
		// bytes [64, 96) encode the offset of "first"
		// bytes [96, 128) encode the offset of "second"
		// swap first and second
		tmp := bytes.Buffer{}
		tmp.Write(input[64:96])
		firstOffset := input[64:96]
		secondOffset := input[96:128]
		copy(firstOffset, secondOffset)
		copy(secondOffset, tmp.Bytes())

		// when
		_, err = abi.DecodeSliceOfBytes(input)

		// then
		assert.ErrorContains(t, err, "greater than end")
	})

	t.Run("bad encoding of bytes", func(t *testing.T) {
		// given
		input, err := abi.EncodeSliceOfBytes(someBytes)
		require.NoError(t, err)
		// add on extra padding
		input = append(input, nZeros(32)...)

		// when
		_, err = abi.DecodeSliceOfBytes(input)

		// then
		assert.ErrorContains(t, err, "decoding element")
	})
}

func TestEncodeDecodeSliceOfBytesRoundTrip(t *testing.T) {
	for _, tc := range testData.sliceOfBytes {
		t.Run(tc.name, func(t *testing.T) {
			// when
			encoded, err := abi.EncodeSliceOfBytes(tc.native)
			require.NoError(t, err)

			got, err := abi.DecodeSliceOfBytes(encoded)
			require.NoError(t, err)

			// then
			assert.Equal(t, tc.native, got)
		})
	}
}

func TestEncodeDecodeTupleRoundTrip(t *testing.T) {
	for _, tc := range testData.allInts {
		t.Run(tc.name, func(t *testing.T) {
			// when
			input := tc.native
			encoded, err := abi.EncodeTuple(
				abi.EncodeTupleFuncUint64(input.Val1),
				abi.EncodeTupleFuncUint64(input.Val2),
				abi.EncodeTupleFuncUint64(input.Val3),
			)
			require.NoError(t, err)
			require.Equal(t, tc.encoded, encoded)

			var got AllInts
			err = abi.DecodeTuple(encoded,
				abi.DecodeTupleFuncUint64(&got.Val1),
				abi.DecodeTupleFuncUint64(&got.Val2),
				abi.DecodeTupleFuncUint64(&got.Val3),
			)
			require.NoError(t, err)

			// then
			assert.Equal(t, input, got)
		})
	}

	for _, tc := range testData.intAndBytes {
		t.Run(tc.name, func(t *testing.T) {
			// when
			input := tc.native
			encoded, err := abi.EncodeTuple(
				abi.EncodeTupleFuncUint64(input.Int1),
				abi.EncodeTupleFuncBytes(input.Bytes1),
				abi.EncodeTupleFuncBytes(input.Bytes2),
			)
			require.NoError(t, err)
			require.Equal(t, tc.encoded, encoded)

			var got IntAndBytes
			err = abi.NewTupleDecoder().
				Uint64(&got.Int1).
				Bytes(&got.Bytes1).
				Bytes(&got.Bytes2).
				Decode(encoded)
			require.NoError(t, err)

			// then
			assert.Equal(t, input, got)
		})
	}
}

func TestDecodeTuple(t *testing.T) {
	// for happy path see round trip test

	t.Run("no decoders provided", func(t *testing.T) {
		// given
		input := []byte("too-short")
		// when
		err := abi.DecodeTuple(input)
		// then
		assert.ErrorContains(t, err, "no decoders provided")
	})

	t.Run("too short to support all decoders", func(t *testing.T) {
		// given
		input := []byte("too-short")
		// when
		err := abi.DecodeTuple(input, abi.DecodeTupleFuncUint64(nil))
		// then
		assert.ErrorContains(t, err, "not long enough to support all decoders")
	})
}

func TestDecodeTupleFuncBytes(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		// given
		want := byte(93)
		input := abi.EncodeUint64(32)
		input = append(input, abiEncodeAByte(want)...)
		// when
		got := []byte{}
		f := abi.DecodeTupleFuncBytes(&got)
		err := f(input[0:32], input)
		require.NoError(t, err)
		// then
		assert.Equal(t, []byte{want}, got)
	})

	t.Run("beginning of offset out of bounds", func(t *testing.T) {
		// given
		input := abi.EncodeUint64(100)
		input = append(input, abiEncodeAByte(7)...)
		f := abi.DecodeTupleFuncBytes(nil)
		// when
		err := f(input[0:32], input)
		// then
		assert.ErrorContains(t, err, "offset+32 out of bounds")
	})

	t.Run("end of offset out of bounds", func(t *testing.T) {
		// given
		input := abi.EncodeUint64(90)
		input = append(input, abiEncodeAByte(7)...)
		f := abi.DecodeTupleFuncBytes(nil)
		// when
		err := f(input[0:32], input)
		// then
		assert.ErrorContains(t, err, "offset+32 out of bounds")
	})

	t.Run("offset not valid", func(t *testing.T) {
		// given
		input := abi.EncodeUint64(32)
		input = append(input, []byte("32-bytes-xxxxxxxxxxxxxxxxxxxxxxx")...)
		input = append(input, nZeros(32)...)
		f := abi.DecodeTupleFuncBytes(nil)
		// when
		err := f(input[0:32], input)
		// then
		assert.ErrorContains(t, err, "decoding length")
	})

	t.Run("end of offset out of bounds", func(t *testing.T) {
		// given
		input := abi.EncodeUint64(32)
		input = append(input, abiEncodeAByte(7)...)
		f := abi.DecodeTupleFuncBytes(nil)
		// when
		err := f(input[0:32], input[:len(input)-1])
		// then
		assert.ErrorContains(t, err, "end is out of bounds")
	})

	t.Run("bytes are invalid", func(t *testing.T) {
		// given
		input := abi.EncodeUint64(32)
		input = append(input, abiEncodeAByte(7)...)
		input[len(input)-1] = 1
		f := abi.DecodeTupleFuncBytes(nil)
		// when
		err := f(input[0:32], input)
		// then
		assert.ErrorContains(t, err, "decoding bytes")
	})
}

func TestTupleEncoderDecoder_RoundTrip(t *testing.T) {
	for _, tc := range testData.allInts {
		t.Run(tc.name, func(t *testing.T) {
			// when
			input := tc.native
			encoded, err := abi.NewTupleEncoder().
				Uint64(input.Val1).
				Uint64(input.Val2).
				Uint64(input.Val3).
				Encode()
			require.NoError(t, err)
			require.Equal(t, tc.encoded, encoded)

			var got AllInts
			err = abi.NewTupleDecoder().
				Uint64(&got.Val1).
				Uint64(&got.Val2).
				Uint64(&got.Val3).
				Decode(encoded)
			require.NoError(t, err)

			// then
			assert.Equal(t, input, got)
		})
	}

	for _, tc := range testData.intAndBytes {
		t.Run(tc.name, func(t *testing.T) {
			// when
			input := tc.native
			encoded, err := abi.NewTupleEncoder().
				Uint64(input.Int1).
				Bytes(input.Bytes1).
				Bytes(input.Bytes2).
				Encode()
			require.NoError(t, err)
			require.Equal(t, tc.encoded, encoded)

			var got IntAndBytes
			err = abi.NewTupleDecoder().
				Uint64(&got.Int1).
				Bytes(&got.Bytes1).
				Bytes(&got.Bytes2).
				Decode(encoded)
			require.NoError(t, err)

			// then
			assert.Equal(t, input, got)
		})
	}
}
