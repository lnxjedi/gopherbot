// +build test

package bot

import (
	"path"
	"runtime"

	"github.com/lnxjedi/robot"
)

var events = make(chan Event, 16)

// shove an event in to the buffered channel for later retrieval by an
// integration test
func emit(e Event) {
	_, file, line, _ := runtime.Caller(1)
	select {
	case events <- e:
		Log(robot.Debug, "Event recorded: %s in %s, line %d", e, path.Base(file), line)
	default:
		Log(robot.Debug, "Event channel buffer full, didn't record: %s in %s, line %d", e, file, line)
	}
}

// GetEvents called by integration tests
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

// GetEventStrings called by terminal connector
func (h handler) GetEventStrings() *[]string {
	ev := make([]string, 0)
loop:
	for {
		select {
		case e := <-events:
			ev = append(ev, e.String())
		default:
			break loop
		}
	}
	return &ev
}
