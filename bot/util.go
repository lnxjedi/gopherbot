package bot

import (
	"fmt"
	"os"
	"regexp"
	godebug "runtime/debug"
	"strings"
	"time"
)

const escapeAliases = `*+^$?\[]{}`
const aliases = `&!;:-%#@~<>/`

func checkPanic(bot *Robot, s string) {
	if r := recover(); r != nil {
		Log(Error, fmt.Sprintf("PANIC from \"%s\": %s\nStack trace:%s", s, r, godebug.Stack()))
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
	name := robot.name
	alias := robot.alias
	robot.RUnlock()
	pre, post, bare, errpre, errpost, errbare := updateRegexesWrapped(name, alias)
	if errpre != nil {
		Log(Error, fmt.Sprintf("Error compiling pre regex: %s", errpre))
	}
	if pre != nil {
		Log(Debug, fmt.Sprintf("Setting pre regex to: %s", pre))
	}
	if errpost != nil {
		Log(Error, fmt.Sprintf("Error compiling post regex: %s", errpost))
	}
	if post != nil {
		Log(Debug, fmt.Sprintf("Setting post regex to: %s", post))
	}
	if errbare != nil {
		Log(Error, fmt.Sprintf("Error compiling bare regex: %s", errbare))
	}
	if bare != nil {
		Log(Debug, fmt.Sprintf("Setting bare regex to: %s", bare))
	}
	robot.Lock()
	robot.preRegex = pre
	robot.postRegex = post
	robot.bareRegex = bare
	robot.Unlock()
}

// TODO: write unit test. The regexes produced shouldn't be checked, but rather
// whether given strings do or don't match them. Note: this code is partially
// tested in TestBotName
func updateRegexesWrapped(name string, alias rune) (pre, post, bare *regexp.Regexp, errpre, errpost, errbare error) {
	pre = nil
	post = nil
	if alias == 0 && len(name) == 0 {
		Log(Error, "Robot has no name or alias, and can't be spoken to")
		return
	}
	preString := `^(?i:`
	if alias != 0 {
		if strings.ContainsRune(string(escapeAliases), alias) {
			preString += `\` + string(alias)
		} else {
			preString += string(alias)
		}
	}
	// If both name and alias present, combine with an '|' (or)
	if alias != 0 && len(name) > 0 {
		preString += `|`
	}
	if len(name) > 0 {
		preString += `@?` + name + `[:, ]`
	}
	preString += `\s*)(.*)$`
	pre, errpre = regexp.Compile(preString)
	if len(name) > 0 {
		postString := `^([^,@]+),?\s+(?i:@?` + name + `)([.?!])?$`
		post, errpost = regexp.Compile(postString)
		bareString := `^@?` + name + `$`
		bare, errbare = regexp.Compile(bareString)
	}
	return
}
