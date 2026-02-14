package bot

import (
	"os"
	"strings"
)

var sensitiveChildEnv = map[string]struct{}{
	"GOPHER_ENCRYPTION_KEY": {},
	"GOPHER_DEPLOY_KEY":     {},
	"GOPHER_HOST_KEYS":      {},
}

func sanitizedChildEnvironment(extra ...string) []string {
	out := make([]string, 0, len(os.Environ())+len(extra))
	for _, envVar := range os.Environ() {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) == 0 {
			continue
		}
		if _, sensitive := sensitiveChildEnv[parts[0]]; sensitive {
			continue
		}
		out = append(out, envVar)
	}
	out = append(out, extra...)
	return out
}
