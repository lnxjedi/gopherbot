package bot

import (
	"fmt"
	"regexp"
	"strings"
)

const escape_aliases = `*+|^$?\[]{}`
const aliases = `&!;:=-%#@~<>/`

func setFormat(format string) MessageFormat {
	switch format {
	case "fixed":
		return Fixed
	default:
		return Variable
	}
}

func updateRegexes() {
	b.lock.RLock()
	alias := b.alias
	b.lock.RUnlock()
	if b.alias == 0 {
		return
	}
	if ! strings.ContainsRune(string(aliases + escape_aliases), alias) {
		Log(Warn, "Invalid alias specified, ignoring. Must be one of: %s%s", escape_aliases, aliases)
		return
	}
	preString := `^(?i:`
	if strings.ContainsRune(string(escape_aliases), alias) {
		preString += `\` + string(alias) + "|"
	} else {
		preString += string(alias) + "|"
	}
	preString += `@?` + b.name + `[:,]{0,1}\s*)(.+)$`
	re, err := regexp.Compile(preString)
	if err == nil {
		Log(Debug, "Setting preString regex to", preString)
		b.lock.Lock()
		b.preRegex = re
		b.lock.Unlock()
	} else {
		Log(Error, fmt.Sprintf("Error compiling robot name regex: %s", preString))
	}
	postString := `^([^,@]+),?\s*(?i:@?` + b.name + `)([.?!])?$`
	re, err = regexp.Compile(postString)
	if err == nil {
		Log(Debug, "Setting postString regex to", postString)
		b.lock.Lock()
		b.postRegex = re
		b.lock.Unlock()
	} else {
		Log(Error, fmt.Sprintf("Error compiling robot name regex: %s", postString))
	}
}
