package bot

import (
	"os"
	"strings"
)

func sanitizedChildEnvironment(extra ...string) []string {
	out := make([]string, 0, len(os.Environ())+len(extra))
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 0 {
			continue
		}
		if strings.HasPrefix(parts[0], "GOPHER_") || parts[0] == gopherEnvHandoffFDEnv {
			continue
		}
		out = append(out, envVar)
	}
	out = append(out, extra...)
	return out
}
