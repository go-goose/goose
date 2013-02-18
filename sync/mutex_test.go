package sync

import (
	"testing"
	"time"
)

func TestAcquireSuccess(t *testing.T) {
	var m Mutex
	m.Lock()
	go func() {
		time.Sleep(time.Duration(5) * time.Millisecond)
		m.Unlock()
	}()
	success := m.Attempt(time.Duration(6) * time.Millisecond)
	if !success {
		t.Fail()
	}
	m.Unlock()
}

func assertAcquireFail(t *testing.T, m *Mutex) {
	m.Lock()
	go func() {
		time.Sleep(time.Duration(5) * time.Millisecond)
		m.Unlock()
	}()
	success := m.Attempt(time.Duration(1) * time.Millisecond)
	if success {
		t.Fail()
	}
}

func TestAcquireFail(t *testing.T) {
	var m Mutex
	assertAcquireFail(t, &m)
}

func TestSecondAttemptAfterFail(t *testing.T) {
	var m Mutex
	assertAcquireFail(t, &m)
	// The lock is still held so the following attempt fails.
	success := m.Attempt(time.Duration(1) * time.Millisecond)
	if success {
		t.Fail()
	}
}

func TestAcquireAfterFail(t *testing.T) {
	var m Mutex
	assertAcquireFail(t, &m)
	success := m.Attempt(time.Duration(5) * time.Millisecond)
	if !success {
		t.Fail()
	}
	m.Unlock()
}

func TestMutexPanic(t *testing.T) {
	defer (func() {
		if recover() == nil {
			t.Fatalf("unlock of unlocked mutex did not panic")
		}
	})()
	var m Mutex
	m.Lock()
	m.Unlock()
	m.Unlock()
}
