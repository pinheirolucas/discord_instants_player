package opusaudio

import (
	"math"
	"testing"

	"github.com/pion/opus"
)

// TestEncodeDecodeRoundTrip encodes a synthetic tone with the same encoder
// settings newFfmpegOpusProvider uses and decodes it back with pion/opus's
// own Decoder, as a cheap guard against an encoder swap silently breaking
// the bitstream. It needs neither ffmpeg nor a live voice connection.
func TestEncodeDecodeRoundTrip(t *testing.T) {
	encoder, err := opus.NewEncoder(
		opus.WithChannels(channels),
		opus.WithApplication(opus.ApplicationAudio),
		opus.WithBitrate(96000),
		opus.WithComplexity(10),
		opus.WithVBR(true),
	)
	if err != nil {
		t.Fatalf("NewEncoder: %v", err)
	}

	pcm := make([]byte, frameSize*channels*2)
	const freqHz = 440.0
	for i := 0; i < frameSize; i++ {
		sample := int16(math.Sin(2*math.Pi*freqHz*float64(i)/float64(frameRate)) * 0.5 * math.MaxInt16)
		for ch := 0; ch < channels; ch++ {
			offset := (i*channels + ch) * 2
			pcm[offset] = byte(sample)
			pcm[offset+1] = byte(sample >> 8)
		}
	}

	encoded := make([]byte, maxBytes)
	n, err := encoder.Encode(pcm, encoded)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if n == 0 {
		t.Fatal("Encode produced zero bytes")
	}
	encoded = encoded[:n]

	decoder := opus.NewDecoder()
	decoded := make([]byte, frameSize*channels*2)
	if _, _, err := decoder.Decode(encoded, decoded); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	nonZero := false
	for _, b := range decoded {
		if b != 0 {
			nonZero = true
			break
		}
	}
	if !nonZero {
		t.Fatal("decoded PCM is all zero")
	}
}
