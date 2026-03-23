package gsh

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lnxjedi/gopherbot/robot"
)

func writeTempScript(t *testing.T, dir, name, contents string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatalf("writing temp script: %v", err)
	}
	return path
}

func TestRunScriptUtilityBuiltins(t *testing.T) {
	tmp := t.TempDir()
	script := writeTempScript(t, tmp, "utilities.gsh", `#!/bin/sh
tmpdir=$(mktemp -d "$GOPHER_WORKSPACE/shfull.XXXXXX") || exit 10
mkdir -p "$tmpdir/a" || exit 11
printf 'beta\nalpha\nbeta\n' > "$tmpdir/a/input.txt"
cp "$tmpdir/a/input.txt" "$tmpdir/a/copy.txt" || exit 12
mv "$tmpdir/a/copy.txt" "$tmpdir/a/moved.txt" || exit 13
touch "$tmpdir/a/marker.txt" || exit 14
printf 'ship' | base64 > "$tmpdir/a/encoded.txt"
decoded=$(base64 -d "$tmpdir/a/encoded.txt") || exit 15
printf '{"phase":"go"}\n' > "$tmpdir/a/data.json"
jq_phase=$(jq -r '.phase' "$tmpdir/a/data.json") || exit 18
gzip "$tmpdir/a/moved.txt" || exit 16
gunzip "$tmpdir/a/moved.txt.gz" || exit 17
head_line=$(head -n 1 "$tmpdir/a/moved.txt")
tail_line=$(tail -n 1 "$tmpdir/a/moved.txt")
line_info=$(wc -l "$tmpdir/a/moved.txt")
set -- $line_info
line_count=$1
uniq_lines=$(cat "$tmpdir/a/moved.txt" | sort | uniq)
printf 'head=%s tail=%s lines=%s decode=%s jq=%s uniq=%s\n' "$head_line" "$tail_line" "$line_count" "$decoded" "$jq_phase" "$(printf '%s' "$uniq_lines" | tr '\n' ',')"
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	ret, err := runScript(
		script,
		"utilities-test",
		[]string{
			"GOPHER_WORKSPACE=" + tmp,
			"GOPHER_INSTALLDIR=" + tmp,
		},
		nil,
		nil,
		nil,
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatalf("runScript() error = %v; stderr=%q", err, stderr.String())
	}
	if ret != robot.Normal {
		t.Fatalf("runScript() ret = %v, want %v; stderr=%q", ret, robot.Normal, stderr.String())
	}
	got := strings.TrimSpace(stdout.String())
	want := "head=beta tail=beta lines=3 decode=ship jq=go uniq=alpha,beta"
	if got != want {
		t.Fatalf("utility output = %q, want %q", got, want)
	}
}
