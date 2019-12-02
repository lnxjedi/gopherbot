// +build !test

package bot

func emit(e Event) {
	// noop - see emit_test.go
}

// GetEvents lets the test harness figure out what happened
func GetEvents() *[]Event {
	return &[]Event{}
}

// GetEventStrings for terminal connector
func (h handler) GetEventStrings() *[]string {
	return &[]string{}
}
