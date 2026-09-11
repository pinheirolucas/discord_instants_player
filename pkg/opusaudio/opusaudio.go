// Package opusaudio decodes clips to PCM via ffmpeg and encodes them to
// Opus on demand for disgo's voice package.
//
// disgo pulls audio rather than accepting a push channel: its AudioSender
// calls OpusFrameProvider.ProvideOpusFrame() on its own 20ms clock, handling
// the speaking indicator and silence frames itself. ffmpegOpusProvider only
// has to hand back the next encoded frame (or io.EOF once the clip, or a
// stop signal, ends it).
package opusaudio

import (
	"bufio"
	"encoding/binary"
	"io"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/disgoorg/disgo/voice"
	"layeh.com/gopus"
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

// ffmpegOpusProvider implements voice.OpusFrameProvider over an ffmpeg
// process decoding one clip to raw PCM, encoding each frame to Opus as
// disgo's AudioSender pulls it.
type ffmpegOpusProvider struct {
	cmd     *exec.Cmd
	stdout  *bufio.Reader
	encoder *gopus.Encoder
	stop    <-chan bool

	closeOnce sync.Once
	done      chan struct{}
}

// newFfmpegOpusProvider starts ffmpeg decoding filename to PCM. The returned
// done channel closes once playback ends, naturally or via stop, so callers
// can block until it's over.
func newFfmpegOpusProvider(filename string, stop <-chan bool) (*ffmpegOpusProvider, <-chan struct{}, error) {
	cmd := exec.Command("ffmpeg", "-i", filename, "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	encoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
	if err != nil {
		_ = cmd.Process.Kill()
		return nil, nil, err
	}

	p := &ffmpegOpusProvider{
		cmd:     cmd,
		stdout:  bufio.NewReaderSize(stdout, 16384),
		encoder: encoder,
		stop:    stop,
		done:    make(chan struct{}),
	}

	return p, p.done, nil
}

// ProvideOpusFrame implements voice.OpusFrameProvider.
func (p *ffmpegOpusProvider) ProvideOpusFrame() ([]byte, error) {
	select {
	case <-p.stop:
		p.finish()
		return nil, io.EOF
	default:
	}

	pcm := make([]int16, frameSize*channels)
	if err := binary.Read(p.stdout, binary.LittleEndian, &pcm); err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			OnError("error reading from ffmpeg stdout", err)
		}
		p.finish()
		return nil, io.EOF
	}

	opus, err := p.encoder.Encode(pcm, frameSize, maxBytes)
	if err != nil {
		OnError("encoding error", err)
		p.finish()
		return nil, io.EOF
	}

	return opus, nil
}

// Close implements voice.OpusFrameProvider. disgo calls it when the
// provider is replaced by a new one or the Conn is closed.
func (p *ffmpegOpusProvider) Close() {
	p.finish()
}

func (p *ffmpegOpusProvider) finish() {
	p.closeOnce.Do(func() {
		_ = p.cmd.Process.Kill()
		close(p.done)
	})
}

// PlayAudioFile plays filename over the given voice.Conn and blocks until
// playback ends or stop is signalled.
func PlayAudioFile(conn voice.Conn, filename string, stop <-chan bool) {
	provider, done, err := newFfmpegOpusProvider(filename, stop)
	if err != nil {
		OnError("failed to start ffmpeg", err)
		return
	}

	conn.SetOpusFrameProvider(provider)
	<-done
}
