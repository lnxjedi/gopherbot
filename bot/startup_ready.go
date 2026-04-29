package bot

import (
	"context"
	"sync"
)

var robotInitializedState = struct {
	sync.Mutex
	ch     chan struct{}
	closed bool
}{
	ch: make(chan struct{}),
}

func resetRobotInitializedSignal() {
	robotInitializedState.Lock()
	robotInitializedState.ch = make(chan struct{})
	robotInitializedState.closed = false
	robotInitializedState.Unlock()
}

func signalRobotInitialized() {
	robotInitializedState.Lock()
	defer robotInitializedState.Unlock()
	if robotInitializedState.closed {
		return
	}
	close(robotInitializedState.ch)
	robotInitializedState.closed = true
}

func WaitForRobotInitialized(ctx context.Context) error {
	robotInitializedState.Lock()
	ch := robotInitializedState.ch
	closed := robotInitializedState.closed
	robotInitializedState.Unlock()
	if closed {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch:
		return nil
	}
}
