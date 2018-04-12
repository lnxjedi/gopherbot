// +build !test

package bot

func emit(e Event) {
	// noop - see emit_test.go
}

func GetEvents() *[]Event {
	return &[]Event{}
}
