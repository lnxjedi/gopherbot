// +build test

package bot

import (
	"fmt"
	"path"
	"runtime"
)

var events = make(chan Event, 16)

// shove an event in to the buffered channel for later retrieval by an
// integration test
func emit(e Event) {
	_, file, line, _ := runtime.Caller(1)
	select {
	case events <- e:
		Log(Debug, fmt.Sprintf("Event recorded: %s in %s, line %d", e, path.Base(file), line))
	default:
		Log(Debug, fmt.Sprintf("Event channel buffer full, didn't record: %s in %s, line %d", e, file, line))
	}
}

// Called by integration tests
func GetEvents() *[]Event {
	ev := make([]Event, 0)
loop:
	for {
		select {
		case e := <-events:
			ev = append(ev, e)
		default:
			break loop
		}
	}
	return &ev
}
