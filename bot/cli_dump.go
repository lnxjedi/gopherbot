package bot

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dump and expanded, but not parsed, configuration file - for troubleshooting yaml errors
func cliDump(which, file string) {
	var base string
	var custom bool
	switch which {
	case "installed":
		base = installPath
	case "configured":
		custom = true
		base = configPath
	}
	cfgfile := filepath.Join(base, "conf", file)
	raw, err := os.ReadFile(cfgfile)
	if err != nil {
		fmt.Printf("Reading '%s': %v\n", cfgfile, err)
		os.Exit(1)
	}
	dir := filepath.Dir(filepath.Join("conf", file))
	expanded, err := expand(dir, custom, raw)
	if err != nil {
		fmt.Printf("Expanding '%s': %v\n", cfgfile, err)
		os.Exit(1)
	}
	fmt.Println(string(expanded))
	os.Exit(0)
}
