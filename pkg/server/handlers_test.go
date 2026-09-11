package server

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pinheirolucas/discord_instants_player/pkg/fsutil"
	"github.com/pinheirolucas/discord_instants_player/pkg/instant"
)

// seedCache points the shared fsutil cache at a temp dir holding a fixture for
// each link, so the handlers resolve clips without any network access.
func seedCache(t *testing.T, links ...string) {
	t.Helper()

	dir := t.TempDir()
	previous := fsutil.Default
	fsutil.Default = &fsutil.Cache{Dir: dir}
	t.Cleanup(func() { fsutil.Default = previous })

	mp3, err := os.ReadFile(filepath.Join("testdata", "clip.mp3"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	for _, link := range links {
		name := filepath.Join(dir, fmt.Sprintf("%x.mp3", md5.Sum([]byte(link))))
		if err := os.WriteFile(name, mp3, 0o644); err != nil {
			t.Fatalf("writing cache fixture: %v", err)
		}
	}
}

func decodeBody(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var out map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	return out
}

func TestHandlePlayRejectsAMissingURL(t *testing.T) {
	s := New(instant.NewPlayer())

	rec := httptest.NewRecorder()
	s.handlePlay(rec, httptest.NewRequest(http.MethodGet, "/play", nil))

	if got := decodeBody(t, rec)["label"]; got != "empty_url" {
		t.Errorf("label = %v, want empty_url", got)
	}
}

func TestHandlePlayReturnsTheClipAsADataURI(t *testing.T) {
	const link = "https://example.com/a.mp3"
	seedCache(t, link)

	s := New(instant.NewPlayer())

	rec := httptest.NewRecorder()
	s.handlePlay(rec, httptest.NewRequest(http.MethodGet, "/play?url="+link, nil))

	data, ok := decodeBody(t, rec)["data"].(map[string]any)
	if !ok {
		t.Fatalf("response had no data object: %s", rec.Body.String())
	}
	if data["exists"] != true {
		t.Errorf("exists = %v, want true", data["exists"])
	}
	content, _ := data["content"].(string)
	if !strings.HasPrefix(content, "data:audio/mp3;base64,") {
		t.Errorf("content = %.40q, want an mp3 data URI", content)
	}
	// The whole clip must survive, not just the part left after type sniffing.
	if len(content) < 1000 {
		t.Errorf("content is only %d chars — the clip looks truncated", len(content))
	}
}

func TestHandleBotPlayRejectsAnInvalidBody(t *testing.T) {
	s := New(instant.NewPlayer())

	rec := httptest.NewRecorder()
	s.handleBotPlay(rec, httptest.NewRequest(http.MethodPost, "/bot/play", strings.NewReader("not json")))

	if got := decodeBody(t, rec)["label"]; got != "invalid_body" {
		t.Errorf("label = %v, want invalid_body", got)
	}
}

func TestHandleBotPlayRejectsAnInvalidURL(t *testing.T) {
	s := New(instant.NewPlayer())

	rec := httptest.NewRecorder()
	s.handleBotPlay(rec, httptest.NewRequest(http.MethodPost, "/bot/play", strings.NewReader(`{"url":"not a url"}`)))

	if got := decodeBody(t, rec)["label"]; got != "invalid_url" {
		t.Errorf("label = %v, want invalid_url", got)
	}
}

// TestHandleBotPlayReturnsOnlyOneResponseForUnsupportedAudio guards against a
// missing return after the fsutil.ErrUnsuportedAudioFormat case: without it,
// control falls out of the switch and into writeSuccessResponse, appending a
// second JSON object onto the same body.
func TestHandleBotPlayReturnsOnlyOneResponseForUnsupportedAudio(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not an mp3 file"))
	}))
	t.Cleanup(upstream.Close)

	previous := fsutil.Default
	fsutil.Default = &fsutil.Cache{Client: upstream.Client(), Dir: t.TempDir()}
	t.Cleanup(func() { fsutil.Default = previous })

	link := upstream.URL + "/a.mp3"

	s := New(instant.NewPlayer())

	rec := httptest.NewRecorder()
	s.handleBotPlay(rec, httptest.NewRequest(http.MethodPost, "/bot/play", strings.NewReader(`{"url":"`+link+`"}`)))

	dec := json.NewDecoder(bytes.NewReader(rec.Body.Bytes()))
	var out map[string]any
	if err := dec.Decode(&out); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if out["label"] != "unsuported_audio_format" {
		t.Errorf("label = %v, want unsuported_audio_format", out["label"])
	}
	if dec.More() {
		t.Errorf("response body carries more than one JSON object: %s", rec.Body.String())
	}
}

func TestHandleBotPlayReturnsTheExitReasonWhenPlaybackEnds(t *testing.T) {
	const link = "https://example.com/b.mp3"
	seedCache(t, link)

	player := instant.NewPlayer()
	s := New(player)

	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		s.handleBotPlay(rec, httptest.NewRequest(http.MethodPost, "/bot/play", strings.NewReader(`{"url":"`+link+`"}`)))
		close(done)
	}()

	// Stand in for the bot loop: take the queued path, then report completion.
	player.GetNextPlay()
	player.End()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("handleBotPlay did not return after End()")
	}

	data, ok := decodeBody(t, rec)["data"].(map[string]any)
	if !ok {
		t.Fatalf("response had no data object: %s", rec.Body.String())
	}
	if data["exitReason"] != "end" {
		t.Errorf("exitReason = %v, want end", data["exitReason"])
	}
}

func TestHandleBotStopReleasesAnInFlightPlay(t *testing.T) {
	const link = "https://example.com/c.mp3"
	seedCache(t, link)

	player := instant.NewPlayer()
	s := New(player)

	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		s.handleBotPlay(rec, httptest.NewRequest(http.MethodPost, "/bot/play", strings.NewReader(`{"url":"`+link+`"}`)))
		close(done)
	}()

	player.GetNextPlay()

	s.handleBotStop(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/bot/stop", nil))

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("POST /bot/stop did not release the in-flight /bot/play")
	}
	<-player.StopChan

	data, _ := decodeBody(t, rec)["data"].(map[string]any)
	if data["exitReason"] != "stop" {
		t.Errorf("exitReason = %v, want stop", data["exitReason"])
	}
}

func TestHandleBotStopIsSafeWhenNothingIsPlaying(t *testing.T) {
	s := New(instant.NewPlayer())

	rec := httptest.NewRecorder()
	s.handleBotStop(rec, httptest.NewRequest(http.MethodPost, "/bot/stop", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}
