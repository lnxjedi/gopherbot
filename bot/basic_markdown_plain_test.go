package bot

import "testing"

func TestRenderBasicMarkdownPlain(t *testing.T) {
	in := "**Deploy status:** *rollback in progress*\nSee [runbook](https://example.com/runbook)\nEscaped: \\*not bold\\* and \\`not code\\` and \\@alice\nInline: `kubectl get pods`"
	got := renderBasicMarkdownPlain(in)
	want := "Deploy status: rollback in progress\nSee runbook (https://example.com/runbook)\nEscaped: *not bold* and `not code` and @alice\nInline: kubectl get pods"
	if got != want {
		t.Fatalf("renderBasicMarkdownPlain() = %q, want %q", got, want)
	}
}

func TestRenderBasicMarkdownPlainFencedCode(t *testing.T) {
	in := "Before\n```yaml\napiVersion: v1\nkind: Pod\n```\nAfter"
	got := renderBasicMarkdownPlain(in)
	want := "Before\n\napiVersion: v1\nkind: Pod\n\nAfter"
	if got != want {
		t.Fatalf("renderBasicMarkdownPlain() = %q, want %q", got, want)
	}
}
