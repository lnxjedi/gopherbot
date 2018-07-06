package bot

func init() {
	RegisterPlugin("builtInhistory", PluginHandler{DefaultConfig: `---`, Handler: jobhistory})
}

func jobhistory(bot *Robot, command string, args ...string) (retval TaskRetVal) {
	switch command {
	case "init":
		return
	}
	return
}
