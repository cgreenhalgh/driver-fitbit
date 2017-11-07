package main

import (
	"sync"
)

// startup state
var started = false
var startupLock = &sync.Mutex{}
var startupCond = sync.NewCond(startupLock)

// check if started (non-blocking)
func IsStarted() bool {
	startupLock.Lock()
	defer startupLock.Unlock()
	return started
}

// wait until started (blocking)
func WaitUntilStarted() {
	startupLock.Lock()
	defer startupLock.Unlock()
	for !started {
		startupCond.Wait()
	}
}

// signal started. Will release all waiting go routines.
func SignalStarted() {
	startupLock.Lock()
	defer startupLock.Unlock()
	started = true
	startupCond.Broadcast()
}
