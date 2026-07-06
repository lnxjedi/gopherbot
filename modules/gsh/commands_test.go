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
		tmp,
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

func TestRunScriptTailShorthandReadsPipedExternalOutput(t *testing.T) {
	tmp := t.TempDir()
	writeTempScript(t, tmp, "emit-lines", `#!/bin/sh
printf 'one\n'
printf 'two\n'
printf 'three\n'
`)
	script := writeTempScript(t, tmp, "tail-shorthand.gsh", `#!/bin/sh
output=$(emit-lines | tail -1)
printf 'last=%s\n' "$output"
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	ret, err := runScript(
		script,
		"tail-shorthand-test",
		tmp,
		[]string{
			"PATH=" + tmp + string(os.PathListSeparator) + os.Getenv("PATH"),
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
	if got != "last=three" {
		t.Fatalf("tail shorthand output = %q, want %q; stderr=%q", got, "last=three", stderr.String())
	}
}

func TestRunScriptJqBuiltinCLICompatibility(t *testing.T) {
	tmp := t.TempDir()
	script := writeTempScript(t, tmp, "jq-compat.gsh", `#!/bin/sh
mkdir -p data mods || exit 10
cat > data/app.json <<'JSON'
{"metadata":{"annotations":{"remoteWorkload":"interactive-spot","remoteNodeSize":"large"}},"spec":{"source":{"helm":{"parameters":[{"name":"nodesize","value":"small"}]},"targetRevision":"prod","path":"argocd/remote-devel"}}}
JSON
cat > data/list.json <<'JSON'
{"n":1}
{"n":2}
JSON
printf 'alpha\nbeta\n' > data/raw.txt
printf '{"n":3}\n{"n":4}\n' > data/slurp.json
printf '{"fromFile":"ok"}\n' > data/vars.json
cat > data/filter.jq <<'EOF'
.fromFile
EOF
cat > mods/inc.jq <<'EOF'
def bump: . + 1;
EOF

workload=$(jq -r --arg annotation remoteWorkload --arg parameter workload --arg default_value interactive '
  def present: select(. != null and . != "");
  first([
    ((.metadata.annotations // {})[$annotation]),
    (.spec.source.helm.parameters[]? | select(.name == $parameter) | .value),
    $default_value
  ][] | present) // ""
' data/app.json) || exit 11

node=$(jq -r --arg annotation missing --arg parameter nodesize --arg default_value medium '
  [((.metadata.annotations // {})[$annotation]), (.spec.source.helm.parameters[]? | select(.name == $parameter) | .value), $default_value]
  | map(select(. != null and . != "")) | .[0] // ""
' data/app.json) || exit 12

argjson=$(jq -nr --argjson obj '{"a":2}' '$obj.a + 1') || exit 13
args=$(jq -nr --arg name astro --argjson count 7 '$ARGS.named.name + ":" + ($ARGS.named.count|tostring) + ":" + ($ARGS.positional|join(","))' --args one two) || exit 14
jsonargs=$(jq -cn '[ $ARGS.positional[0].x ]' --jsonargs '{"x":1}') || exit 15
slurp=$(jq -c -s '[.[].n] | add' data/list.json) || exit 16
raw=$(jq -R -s -r 'split("\n")[:-1] | join("|")' data/raw.txt) || exit 17
filter=$(jq -r -f data/filter.jq data/vars.json) || exit 18
slurpfile=$(jq -nr --slurpfile docs data/list.json '$docs | map(.n) | add') || exit 19
rawfile=$(jq -nr --rawfile text data/raw.txt '$text | split("\n")[0]') || exit 20
module=$(jq -nr -L mods 'include "inc"; 41 | bump') || exit 21
env_result=$(export JQ_GSH_ENV=visible; jq -nr 'env.JQ_GSH_ENV + ":" + $ENV.JQ_GSH_ENV') || exit 22
input_result=$(printf '{"v":5}\n' | jq -nr 'input.v + 1') || exit 23
filename=$(jq -r 'input_filename' data/app.json) || exit 24
stream=$(jq -c --stream 'select(length == 2 and .[0][-1] == "remoteWorkload")' data/app.json) || exit 25
yaml=$(printf 'a: 9\n' | jq --yaml-input -r '.a') || exit 26
compact=$(jq -c '.metadata.annotations | {remoteWorkload}' data/app.json) || exit 27
raw_join=$(printf '"a"\n"b"\n' | jq -rj '.') || exit 28

if jq -ne 'false' >/dev/null 2>/dev/null; then
  exit 29
fi
if ! jq -ne 'true' >/dev/null 2>/dev/null; then
  exit 30
fi

printf 'workload=%s node=%s argjson=%s args=%s jsonargs=%s slurp=%s raw=%s filter=%s slurpfile=%s rawfile=%s module=%s env=%s input=%s filename=%s stream=%s yaml=%s compact=%s rawjoin=%s\n' \
  "$workload" "$node" "$argjson" "$args" "$jsonargs" "$slurp" "$raw" "$filter" "$slurpfile" "$rawfile" "$module" "$env_result" "$input_result" "$filename" "$stream" "$yaml" "$compact" "$raw_join"
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	ret, err := runScript(
		script,
		"jq-compat-test",
		tmp,
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
		t.Fatalf("runScript() ret = %v, want %v; stdout=%q stderr=%q", ret, robot.Normal, stdout.String(), stderr.String())
	}
	got := strings.TrimSpace(stdout.String())
	want := `workload=interactive-spot node=small argjson=3 args=astro:7:one,two jsonargs=[1] slurp=3 raw=alpha|beta filter=ok slurpfile=3 rawfile=alpha module=42 env=visible:visible input=6 filename=data/app.json stream=[["metadata","annotations","remoteWorkload"],"interactive-spot"] yaml=9 compact={"remoteWorkload":"interactive-spot"} rawjoin=ab`
	if got != want {
		t.Fatalf("jq compat output = %q, want %q; stderr=%q", got, want, stderr.String())
	}
}

func TestRunScriptUsesWorkDirInsteadOfScriptDir(t *testing.T) {
	home := t.TempDir()
	scriptDir := filepath.Join(home, "jobs")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("creating script dir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(home, "custom"), 0o755); err != nil {
		t.Fatalf("creating custom dir: %v", err)
	}
	outPath := filepath.Join(home, "cwd.txt")
	script := writeTempScript(t, scriptDir, "install-libs.gsh", `#!/bin/sh
pwd > "$OUT_PATH"
cd custom
pwd >> "$OUT_PATH"
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	ret, err := runScript(
		script,
		"install-libs-test",
		home,
		[]string{
			"OUT_PATH=" + outPath,
			"GOPHER_INSTALLDIR=" + home,
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
	gotBytes, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading cwd output: %v", err)
	}
	got := strings.Split(strings.TrimSpace(string(gotBytes)), "\n")
	if len(got) != 2 {
		t.Fatalf("cwd output lines = %#v, want two lines", got)
	}
	if got[0] != home {
		t.Fatalf("initial cwd = %q, want %q", got[0], home)
	}
	if got[1] != filepath.Join(home, "custom") {
		t.Fatalf("cwd after cd custom = %q, want %q", got[1], filepath.Join(home, "custom"))
	}
}

func TestParseLogLevelSupportsNumericAndNamedLevels(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  robot.LogLevel
	}{
		{name: "numeric audit", input: "3", want: robot.Audit},
		{name: "named trace", input: "Trace", want: robot.Trace},
		{name: "named debug lower", input: "debug", want: robot.Debug},
		{name: "named info", input: "Info", want: robot.Info},
		{name: "named audit", input: "Audit", want: robot.Audit},
		{name: "named warn", input: "Warn", want: robot.Warn},
		{name: "named warning", input: "Warning", want: robot.Warn},
		{name: "named error", input: "Error", want: robot.Error},
		{name: "named fatal matches external shell behavior", input: "Fatal", want: robot.Error},
		{name: "unknown matches external shell behavior", input: "NoSuchLevel", want: robot.Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseLogLevel(tt.input); got != tt.want {
				t.Fatalf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
