// Package opusaudio decodes cached mp3 clips to PCM with a pure-Go
// decoder and encodes them to Opus on demand for disgo's voice package.
//
// disgo pulls audio rather than accepting a push channel: its AudioSender
// calls OpusFrameProvider.ProvideOpusFrame() on its own 20ms clock, handling
// the speaking indicator and silence frames itself. mp3OpusProvider only
// has to hand back the next encoded frame (or io.EOF once the clip, or a
// stop signal, ends it).
package opusaudio

import (
	"bufio"
	"io"
	"os"
	"sync"

	"github.com/disgoorg/disgo/voice"
	"github.com/hajimehoshi/go-mp3"
	"github.com/pion/opus"
)

// Technically the below settings can be adjusted however that poses
// a lot of other problems that are not handled well at this time.
// These below values seem to provide the best overall performance
const (
	channels  int = 2                   // 1 for mono, 2 for stereo
	frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // uint16 size of each audio frame
	maxBytes  int = (frameSize * 2) * 2 // max size of opus data
)

// OnError gets called by opusaudio when an error is encountered.
// By default logs to STDERR
var OnError = func(str string, err error) {
	prefix := "opusaudio: " + str

	if err != nil {
		os.Stderr.WriteString(prefix + ": " + err.Error() + "\n")
	} else {
		os.Stderr.WriteString(prefix + "\n")
	}
}

// mp3OpusProvider implements voice.OpusFrameProvider over an mp3 file
// decoded to raw PCM and resampled to frameRate, encoding each frame to
// Opus as disgo's AudioSender pulls it.
type mp3OpusProvider struct {
	file    *os.File
	pcm     *bufio.Reader
	encoder *opus.Encoder
	stop    <-chan bool

	closeOnce sync.Once
	done      chan struct{}
}

// newMp3OpusProvider opens filename and decodes it to PCM as it's read.
// The returned done channel closes once playback ends, naturally or via
// stop, so callers can block until it's over.
func newMp3OpusProvider(filename string, stop <-chan bool) (*mp3OpusProvider, <-chan struct{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}

	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}

	resampled := NewResampler(decoder, decoder.SampleRate(), frameRate, channels)

	encoder, err := opus.NewEncoder(
		opus.WithChannels(channels),
		opus.WithApplication(opus.ApplicationAudio),
		opus.WithBitrate(96000),
		opus.WithComplexity(10),
		opus.WithVBR(true),
	)
	if err != nil {
		_ = file.Close()
		return nil, nil, err
	}

	p := &mp3OpusProvider{
		file:    file,
		pcm:     bufio.NewReaderSize(resampled, 16384),
		encoder: encoder,
		stop:    stop,
		done:    make(chan struct{}),
	}

	return p, p.done, nil
}

// ProvideOpusFrame implements voice.OpusFrameProvider.
func (p *mp3OpusProvider) ProvideOpusFrame() ([]byte, error) {
	select {
	case <-p.stop:
		p.finish()
		return nil, io.EOF
	default:
	}

	pcm := make([]byte, frameSize*channels*2)
	if _, err := io.ReadFull(p.pcm, pcm); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			OnError("error reading decoded mp3 PCM", err)
		}
		p.finish()
		return nil, io.EOF
	}

	out := make([]byte, maxBytes)
	n, err := p.encoder.Encode(pcm, out)
	if err != nil {
		OnError("encoding error", err)
		p.finish()
		return nil, io.EOF
	}

	return out[:n], nil
}

// Close implements voice.OpusFrameProvider. disgo calls it when the
// provider is replaced by a new one or the Conn is closed.
func (p *mp3OpusProvider) Close() {
	p.finish()
}

func (p *mp3OpusProvider) finish() {
	p.closeOnce.Do(func() {
		_ = p.file.Close()
		close(p.done)
	})
}

// PlayAudioFile plays filename over the given voice.Conn and blocks until
// playback ends or stop is signalled.
func PlayAudioFile(conn voice.Conn, filename string, stop <-chan bool) {
	provider, done, err := newMp3OpusProvider(filename, stop)
	if err != nil {
		OnError("failed to decode mp3", err)
		return
	}

	conn.SetOpusFrameProvider(provider)
	<-done
}
