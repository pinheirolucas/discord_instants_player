package instant

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Play resolves the link through fsutil.GetFromCache, which returns the cached
// file directly when it already exists — so seeding ~/.instants keeps these
// tests off the network entirely. fsutil finds the cache via os.UserHomeDir(),
// which reads HOME.
func seedCache(t *testing.T, links ...string) []string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".instants")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating cache dir: %v", err)
	}

	paths := make([]string, 0, len(links))
	for _, link := range links {
		path := filepath.Join(dir, fmt.Sprintf("%x.mp3", md5.Sum([]byte(link))))
		if err := os.WriteFile(path, []byte("not really an mp3"), 0o644); err != nil {
			t.Fatalf("writing cache fixture: %v", err)
		}
		paths = append(paths, path)
	}

	return paths
}

// Play blocks until the consumer signals End or Stop, so every case here runs it
// in a goroutine and reports back over a channel. waitFor keeps a stuck player
// from hanging the suite.
func waitFor(t *testing.T, ch <-chan string) string {
	t.Helper()

	select {
	case v := <-ch:
		return v
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Play to return")
		return ""
	}
}

func playAsync(p *Player, link string) (<-chan string, <-chan error) {
	reason := make(chan string, 1)
	errc := make(chan error, 1)

	go func() {
		r, err := p.Play(link)
		errc <- err
		reason <- r
	}()

	return reason, errc
}

func TestPlayRejectsInvalidLinkBeforeAnyIO(t *testing.T) {
	p := NewPlayer()
	defer p.Close()

	reason, err := p.Play("not a url")

	if err != ErrInvalidLink {
		t.Errorf("err = %v, want ErrInvalidLink", err)
	}
	if reason != "" {
		t.Errorf("reason = %q, want empty", reason)
	}
}

func TestStopIsANoOpWhenNothingIsPlaying(t *testing.T) {
	p := NewPlayer()
	defer p.Close()

	// StopChan has capacity 1; a Stop that wrongly published here would leave a
	// stale value behind and desynchronise the next real playback.
	p.Stop()

	select {
	case v := <-p.StopChan:
		t.Fatalf("Stop() published %v to StopChan with nothing playing", v)
	default:
	}
}

func TestEndIsANoOpWhenNothingIsPlaying(t *testing.T) {
	p := NewPlayer()
	defer p.Close()

	p.End()

	select {
	case v := <-p.endChan:
		t.Fatalf("End() published %v to endChan with nothing playing", v)
	default:
	}
}

func TestGetNextPlayReceivesThePathAndEndCompletesPlayback(t *testing.T) {
	p := NewPlayer()
	defer p.Close()

	path := seedCache(t, "https://example.com/a.mp3")[0]

	reason, errc := playAsync(p, "https://example.com/a.mp3")

	if got := p.GetNextPlay(); got != path {
		t.Errorf("GetNextPlay() = %q, want the cached file path %q", got, path)
	}

	p.End()

	if err := <-errc; err != nil {
		t.Fatalf("Play returned error: %v", err)
	}
	if r := waitFor(t, reason); r != "end" {
		t.Errorf("Play returned %q, want \"end\"", r)
	}
}

func TestStopMidPlaybackReturnsStopAndSignalsStopChan(t *testing.T) {
	p := NewPlayer()
	defer p.Close()

	seedCache(t, "https://example.com/b.mp3")

	reason, errc := playAsync(p, "https://example.com/b.mp3")
	p.GetNextPlay()

	p.Stop()

	select {
	case <-p.StopChan:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not signal StopChan, which is what stops the audio stream")
	}

	if err := <-errc; err != nil {
		t.Fatalf("Play returned error: %v", err)
	}
	if r := waitFor(t, reason); r != "stop" {
		t.Errorf("Play returned %q, want \"stop\"", r)
	}
}

func TestPlayWhileAlreadyPlayingStopsTheFirstClip(t *testing.T) {
	p := NewPlayer()
	defer p.Close()

	seedCache(t, "https://example.com/first.mp3", "https://example.com/second.mp3")

	firstReason, firstErr := playAsync(p, "https://example.com/first.mp3")
	p.GetNextPlay()

	secondReason, secondErr := playAsync(p, "https://example.com/second.mp3")

	// The first Play is released with "stop" by the second one.
	if err := <-firstErr; err != nil {
		t.Fatalf("first Play returned error: %v", err)
	}
	if r := waitFor(t, firstReason); r != "stop" {
		t.Errorf("first Play returned %q, want \"stop\"", r)
	}

	<-p.StopChan
	p.GetNextPlay()
	p.End()

	if err := <-secondErr; err != nil {
		t.Fatalf("second Play returned error: %v", err)
	}
	if r := waitFor(t, secondReason); r != "end" {
		t.Errorf("second Play returned %q, want \"end\"", r)
	}
}
