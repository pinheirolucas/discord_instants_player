package opusaudio

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

// encodePCM16 encodes a flat slice of int16 samples (interleaved per
// channel already, if any) as little-endian PCM bytes.
func encodePCM16(samples []int16) []byte {
	out := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(s))
	}
	return out
}

func decodePCM16(b []byte) []int16 {
	out := make([]int16, len(b)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(b[i*2:]))
	}
	return out
}

func TestNewResamplerIdenticalRatesPassthrough(t *testing.T) {
	src := bytes.NewReader(encodePCM16([]int16{1, 2, 3, 4}))

	out := NewResampler(src, 44100, 44100, 2)

	if out != io.Reader(src) {
		t.Fatal("NewResampler should return src unchanged when rates match")
	}
}

func TestResamplerUpsampleInterpolatesValues(t *testing.T) {
	// A single-channel ramp: 0, 10, 20, ..., 90.
	values := make([]int16, 10)
	for i := range values {
		values[i] = int16(i * 10)
	}
	src := bytes.NewReader(encodePCM16(values))

	// ratio = srcRate/dstRate = 0.5: exactly two output samples per
	// source sample, so the math has no rounding to worry about.
	r := NewResampler(src, 24000, 48000, 1)

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	samples := decodePCM16(out)

	// The last source sample can't be interpolated towards (there's no
	// sample beyond it), so a couple of trailing output samples are
	// legitimately not produced.
	if len(samples) < 2*(len(values)-2) {
		t.Fatalf("got %d output samples, want at least %d", len(samples), 2*(len(values)-2))
	}

	for k := 0; k*2+1 < len(samples) && k < len(values)-1; k++ {
		wantEven := values[k]
		wantOdd := (values[k] + values[k+1]) / 2

		if samples[k*2] != wantEven {
			t.Errorf("sample %d = %d, want %d", k*2, samples[k*2], wantEven)
		}
		if samples[k*2+1] != wantOdd {
			t.Errorf("sample %d = %d, want %d", k*2+1, samples[k*2+1], wantOdd)
		}
	}
}

func TestResamplerSampleCountRatio(t *testing.T) {
	const srcCount = 1000
	values := make([]int16, srcCount)
	for i := range values {
		values[i] = int16(i % 100)
	}
	src := bytes.NewReader(encodePCM16(values))

	r := NewResampler(src, 48000, 24000, 1) // ratio = 2.0, downsampling

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	samples := decodePCM16(out)

	want := srcCount / 2
	if diff := want - len(samples); diff < 0 || diff > 2 {
		t.Fatalf("got %d output samples, want approximately %d", len(samples), want)
	}
}

func TestResamplerHandlesEmptySource(t *testing.T) {
	src := bytes.NewReader(nil)
	r := NewResampler(src, 44100, 48000, 2)

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected no output from an empty source, got %d bytes", len(out))
	}
}
