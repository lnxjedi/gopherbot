package bot

import (
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func TestSetFormat(t *testing.T) {
	tests := []struct {
		in   string
		want robot.MessageFormat
	}{
		{in: "raw", want: robot.Raw},
		{in: "fixed", want: robot.Fixed},
		{in: "variable", want: robot.Variable},
		{in: "BasicMarkdown", want: robot.BasicMarkdown},
		{in: "basic_markdown", want: robot.BasicMarkdown},
		{in: "basic-markdown", want: robot.BasicMarkdown},
		{in: "unknown-format", want: robot.BasicMarkdown},
	}

	for _, tc := range tests {
		if got := setFormat(tc.in); got != tc.want {
			t.Fatalf("setFormat(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
