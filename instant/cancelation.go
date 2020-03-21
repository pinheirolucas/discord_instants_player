package instant

import "sync"

type CancelationManager struct {
	sync.Mutex
	tokens map[string]chan bool
}

func NewCancelationManager() *CancelationManager {
	m := &CancelationManager{
		tokens: make(map[string]chan bool),
	}

	return m
}

func (m *CancelationManager) Register(token string) (func() bool, func() bool) {
	m.Lock()
	defer m.Unlock()

	m.tokens[token] = make(chan bool, 1)

	cancel := func() bool {
		return m.Cancel(token)
	}

	stop := func() bool {
		return m.Stop(token)
	}

	return cancel, stop
}

// Stop clear the channel after sending a message to it
func (m *CancelationManager) Stop(token string) bool {
	m.Lock()
	defer m.Unlock()

	c, ok := m.tokens[token]
	if !ok {
		return false
	}

	c <- true

	close(c)
	delete(m.tokens, token)
	return true
}

// Cancel the channel to stop receiving messages
func (m *CancelationManager) Cancel(token string) bool {
	m.Lock()
	defer m.Unlock()

	c, ok := m.tokens[token]
	if !ok {
		return false
	}

	close(c)
	delete(m.tokens, token)
	return true
}
