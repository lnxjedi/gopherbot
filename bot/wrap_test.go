package bot

import "testing"

func TestWrapIgnoresANSIEscapeWidth(t *testing.T) {
	in := "\x1b[1mhello world\x1b[22m"
	got := Wrap(in, 5)
	want := "\x1b[1mhello\nworld\x1b[22m\n"
	if got != want {
		t.Fatalf("Wrap() = %q, want %q", got, want)
	}
}

func TestWrapPreservesPlainBehavior(t *testing.T) {
	in := "alpha beta gamma"
	got := Wrap(in, 10)
	want := "alpha beta\ngamma\n"
	if got != want {
		t.Fatalf("Wrap() = %q, want %q", got, want)
	}
}
