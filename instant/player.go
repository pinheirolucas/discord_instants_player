package instant

import (
	"strings"
	"sync"

	"github.com/pinheirolucas/discord_instants_player/fsutil"
	"github.com/pkg/errors"
)

var ErrInvalidLink = errors.New("invalid link")

type Player struct {
	sync.Mutex

	playing string

	StopChan  chan bool
	playChan  chan string
	startChan chan string
	endChan   chan string
}

func NewPlayer() *Player {
	return &Player{
		StopChan:  make(chan bool, 1),
		playChan:  make(chan string, 1),
		startChan: make(chan string, 1),
		endChan:   make(chan string, 1),
	}
}

func (p *Player) Close() {
	close(p.playChan)
	close(p.StopChan)
}

func (p *Player) Play(link string) error {
	if !IsLinkValid(link) {
		return ErrInvalidLink
	}

	f, err := fsutil.GetFromCache(link)
	if err != nil {
		return err
	}
	defer f.Close()

	p.Lock()
	defer p.Unlock()

	if isPlaying(p.playing) {
		p.StopChan <- true
	}

	p.playChan <- f.Name()
	p.startChan <- link
	p.playing = f.Name()
	return nil
}

func (p *Player) Stop() {
	p.Lock()
	defer p.Unlock()

	if !isPlaying(p.playing) {
		return
	}

	p.StopChan <- true
	p.endChan <- p.playing
	p.playing = ""
}

func (p *Player) End() {
	p.Lock()
	defer p.Unlock()

	if !isPlaying(p.playing) {
		return
	}

	p.endChan <- p.playing
	p.playing = ""
}

func (p *Player) GetNextStarted() string {
	return <-p.startChan
}

func (p *Player) GetNextEnded() string {
	return <-p.endChan
}

func (p *Player) GetNextPlay() string {
	return <-p.playChan
}

func isPlaying(url string) bool {
	return strings.TrimSpace(url) != ""
}
