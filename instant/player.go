package instant

import (
	"fmt"
	"sync"

	"github.com/pinheirolucas/discord_instants_player/fsutil"
	"github.com/pkg/errors"
)

var ErrInvalidLink = errors.New("invalid link")

type Player struct {
	sync.Mutex

	playing bool

	playChan     chan string
	endChan      chan bool
	internalStop chan bool
	StopChan     chan bool
}

func NewPlayer() *Player {
	return &Player{
		playChan:     make(chan string, 1),
		endChan:      make(chan bool, 1),
		internalStop: make(chan bool, 1),
		StopChan:     make(chan bool, 1),
	}
}

func (p *Player) Close() {
	close(p.playChan)
	close(p.endChan)
	close(p.StopChan)
}

func (p *Player) Play(link string) (string, error) {
	if !IsLinkValid(link) {
		return "", ErrInvalidLink
	}

	f, err := fsutil.GetFromCache(link)
	if err != nil {
		return "", err
	}
	defer f.Close()

	p.Stop()
	p.playChan <- f.Name()
	p.playing = true

	select {
	case <-p.endChan:
		fmt.Println("ended")
		return "end", nil
	case <-p.internalStop:
		fmt.Println("stoped")
		return "stop", nil
	}
}

func (p *Player) Stop() {
	if !p.playing {
		return
	}

	p.StopChan <- true
	p.internalStop <- true
	p.playing = false
}

func (p *Player) End() {
	if !p.playing {
		return
	}

	p.endChan <- p.playing
	p.playing = false
}

func (p *Player) GetNextPlay() string {
	return <-p.playChan
}
