// +build test

package bot

// shove an event in to the buffered channel for later retrieval by an
// integration test
func emit(e Event) {
	select {
	case robot.events <- e:
	default:
	}
}

// Called by integration tests
func GetEvents() *[]Event {
	ev := make([]Event, 0)
loop:
	for {
		select {
		case e := <-robot.events:
			ev = append(ev, e)
		default:
			break loop
		}
	}
	return &ev
}
