package bot

import (
	"fmt"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"
)

const escape_aliases = `*+|^$?\[]{}`
const aliases = `&!;:-%#@~<>/`

func checkPanic(bot *Robot, s string) {
	if r := recover(); r != nil {
		Log(Error, fmt.Sprintf("PANIC from \"%s\": %s\nStack trace:%s", s, r, debug.Stack()))
		bot.Reply(fmt.Sprintf("OUCH! It looks like you found a bug - please ask an admin to check the log and give them this string: \"%s\"", s))
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}
}

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
	preString := `^(?i:`
	if b.alias != 0 {
		if strings.ContainsRune(string(escape_aliases), alias) {
			preString += `\` + string(alias) + "|"
		} else {
			preString += string(alias) + "|"
		}
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
