package bot

import (
	"fmt"
	"os"
	"regexp"
	"runtime/debug"
	"strings"
	"time"
)

const escapeAliases = `*+|^$?\[]{}`
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
	robot.RLock()
	alias := robot.alias
	robot.RUnlock()
	preString := `^(?i:`
	if robot.alias != 0 {
		if strings.ContainsRune(string(escapeAliases), alias) {
			preString += `\` + string(alias) + "|"
		} else {
			preString += string(alias) + "|"
		}
	}
	preString += `@?` + robot.name + `[:,]{0,1}\s*)(.*)$`
	re, err := regexp.Compile(preString)
	if err == nil {
		Log(Debug, "Setting preString regex to", preString)
		robot.Lock()
		robot.preRegex = re
		robot.Unlock()
	} else {
		Log(Error, fmt.Sprintf("Error compiling robot name regex: %s", preString))
	}
	postString := `^([^,@]+),?\s*(?i:@?` + robot.name + `)([.?!])?$`
	re, err = regexp.Compile(postString)
	if err == nil {
		Log(Debug, "Setting postString regex to", postString)
		robot.Lock()
		robot.postRegex = re
		robot.Unlock()
	} else {
		Log(Error, fmt.Sprintf("Error compiling robot name regex: %s", postString))
	}
}
