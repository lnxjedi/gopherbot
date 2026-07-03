package bot

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dump an expanded, but not parsed, configuration file for troubleshooting yaml errors.
func cliDump(which, file string, unredactedSecrets bool) {
	var base string
	var custom bool
	switch which {
	case "installed":
		base = installPath
	case "configured":
		custom = true
		base = configPath
	}
	secretMode := configSecretRedact
	if unredactedSecrets {
		secretMode = configSecretRequire
	}
	restoreSecretMode := beginConfigSecretResolution(secretMode)
	defer restoreSecretMode()

	cfgfile := filepath.Join(base, "conf", file)
	raw, err := os.ReadFile(cfgfile)
	if err != nil {
		fmt.Printf("Reading '%s': %v\n", cfgfile, err)
		os.Exit(1)
	}
	if custom {
		variables, err := loadConfigVariables()
		if err != nil {
			fmt.Printf("Loading custom variables: %v\n", err)
			os.Exit(1)
		}
		setActiveConfigVariables(variables)
	} else {
		setActiveConfigVariables(newConfigVariableSet())
	}
	dir := filepath.Dir(filepath.Join("conf", file))
	expanded, err := expand(dir, custom, raw)
	if err != nil {
		fmt.Printf("Expanding '%s': %v\n", cfgfile, err)
		os.Exit(1)
	}
	if configSecretRedactionUsed() {
		fmt.Fprintln(os.Stderr, `Info: secret template values redacted; use --unredacted-secrets to print decrypted secrets`)
	}
	fmt.Println(string(expanded))
}
