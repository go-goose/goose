package sync

import (
	gosync "sync"
	"time"
)

// A mutex implementation which allows non-blocking acquisition attempts which time out after a selected period.
type Mutex struct {
	mu gosync.Mutex
	c  chan struct{}
	Locked bool
}

func (m *Mutex) init() {
	m.mu.Lock()
	if m.c == nil {
		m.c = make(chan struct{}, 1)
	}
	m.mu.Unlock()
}

func (m *Mutex) Lock() {
	m.init()
	m.c <- struct{}{}
	m.Locked = true
}

func (m *Mutex) Unlock() {
	m.init()
	if !m.Locked {
		panic("sync: unlock of unLocked mutex")
	}
	<-m.c
	m.Locked = false
}

// Attempt will try and obtain a lock on the mutex within timeout.
// If a lock is successfully obtained, return true, else false.
func (m *Mutex) Attempt(timeout time.Duration) bool {
	m.init()
	timer := time.NewTimer(timeout)
	select {
	case m.c <- struct{}{}:
		m.Locked = true
		timer.Stop()
		return true
	case <-time.After(timeout):
	}
	return false
}
