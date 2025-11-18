package abi

import (
	"bytes"
	"testing"
)

func BenchmarkEncodeUint64(b *testing.B) {
	cases := []struct {
		name string
		v    uint64
	}{
		{"Small", 1},
		{"Medium", 123456789},
		{"Large", 1<<63 - 1},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_ = EncodeUint64(tc.v)
			}
		})
	}
}

func BenchmarkDecodeUint64(b *testing.B) {
	cases := []struct {
		name string
		data []byte
	}{
		{"Small", EncodeUint64(1)},
		{"Medium", EncodeUint64(123456789)},
		{"Large", EncodeUint64(1<<63 - 1)},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_, _ = DecodeUint64(tc.data)
			}
		})
	}
}

func BenchmarkEncodeSliceOfBytes(b *testing.B) {
	cases := []struct {
		name string
		data [][]byte
	}{
		{
			"Small-3-items",
			[][]byte{
				[]byte("a"),
				[]byte("bb"),
				[]byte("ccc"),
			},
		},
		{
			"Medium-10-items",
			func() [][]byte {
				out := make([][]byte, 10)
				for i := range out {
					out[i] = bytes.Repeat([]byte{1}, 32)
				}
				return out
			}(),
		},
		{
			"Large-100-items",
			func() [][]byte {
				out := make([][]byte, 100)
				for i := range out {
					out[i] = bytes.Repeat([]byte{2}, 64)
				}
				return out
			}(),
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				_, _ = EncodeSliceOfBytes(tc.data)
			}
		})
	}
}

func BenchmarkDecodeSliceOfBytes(b *testing.B) {
	makeEnc := func(size, itemSize int) []byte {
		arr := make([][]byte, size)
		for i := range arr {
			arr[i] = bytes.Repeat([]byte{1}, itemSize)
		}
		enc, _ := EncodeSliceOfBytes(arr)
		return enc
	}

	cases := []struct {
		name string
		data []byte
	}{
		{"Small-3-items", makeEnc(3, 8)},
		{"Medium-10-items", makeEnc(10, 32)},
		{"Large-100-items", makeEnc(100, 64)},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = DecodeSliceOfBytes(tc.data)
			}
		})
	}
}

// Helper to generate a tuple of n uint64 elements
func makeUint64Tuple(n int) []EncoderFunc {
	tuple := make([]EncoderFunc, n)
	for i := 0; i < n; i++ {
		v := uint64(i + 1)
		tuple[i] = func() (EncoderResult, error) {
			return EncoderResult{data: EncodeUint64(v)}, nil
		}
	}
	return tuple
}

func BenchmarkEncodeTuple(b *testing.B) {
	cases := []struct {
		name      string
		numFields int
	}{
		{"Small-3-elements", 3},
		{"Medium-10-elements", 10},
		{"Large-50-elements", 50},
		{"VeryLarge-100-elements", 100},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			tuple := makeUint64Tuple(tc.numFields)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := EncodeTuple(tuple...)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDecodeTuple(b *testing.B) {
	// helper to build encoded tuple with n uint64 values
	makeEncoded := func(n int) []byte {
		encs := make([]EncoderFunc, n)
		for i := range n {
			encs[i] = EncodeTupleFuncUint64(uint64(i + 1))
		}
		out, _ := EncodeTuple(encs...)
		return out
	}

	cases := []struct {
		name string
		data []byte
	}{
		{"Small-3-elements", makeEncoded(3)},
		{"Medium-10-elements", makeEncoded(10)},
		{"Large-50-elements", makeEncoded(50)},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			dec := make([]DecoderFunc, len(tc.data)/32)
			for i := range dec {
				var v uint64
				dec[i] = DecodeTupleFuncUint64(&v)
			}
			b.ResetTimer()

			for b.Loop() {
				_ = DecodeTuple(tc.data, dec...)
			}
		})
	}
}

func BenchmarkEncodeTupleFuncUint64(b *testing.B) {
	cases := []struct {
		name string
		v    uint64
	}{
		{"Small", 1},
		{"Medium", 123456},
		{"Large", 1<<63 - 1},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			f := EncodeTupleFuncUint64(tc.v)
			b.ResetTimer()

			for b.Loop() {
				_, _ = f()
			}
		})
	}
}

func BenchmarkEncodeTupleFuncBytes(b *testing.B) {
	cases := []struct {
		name string
		data []byte
	}{
		{"Small-8B", bytes.Repeat([]byte{1}, 8)},
		{"Medium-32B", bytes.Repeat([]byte{2}, 32)},
		{"Large-128B", bytes.Repeat([]byte{3}, 128)},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			f := EncodeTupleFuncBytes(tc.data)
			b.ResetTimer()

			for b.Loop() {
				_, _ = f()
			}
		})
	}
}

func BenchmarkDecodeTupleFuncUint64(b *testing.B) {
	cases := []struct {
		name string
		data []byte
	}{
		{"Small", EncodeUint64(1)},
		{"Medium", EncodeUint64(123456789)},
		{"Large", EncodeUint64(1<<63 - 1)},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			var out uint64
			f := DecodeTupleFuncUint64(&out)
			full := tc.data
			b.ResetTimer()

			for b.Loop() {
				_ = f(tc.data, full)
			}
		})
	}
}

func BenchmarkDecodeTupleFuncBytes(b *testing.B) {
	// helper to build encoded data for one bytes element
	makeEncoded := func(v []byte) (cur, full []byte) {
		enc := EncodeTupleFuncBytes(v)
		full, _ = EncodeTuple(enc)
		cur = full[:32]
		return
	}

	cases := []struct {
		name string
		data []byte
	}{
		{"Small-8B", bytes.Repeat([]byte{1}, 8)},
		{"Medium-32B", bytes.Repeat([]byte{2}, 32)},
		{"Large-128B", bytes.Repeat([]byte{3}, 128)},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			var out []byte
			cur, full := makeEncoded(tc.data)
			f := DecodeTupleFuncBytes(&out)
			b.ResetTimer()

			for b.Loop() {
				_ = f(cur, full)
			}
		})
	}
}
