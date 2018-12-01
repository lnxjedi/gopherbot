package bot

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	godebug "runtime/debug"
	"strings"
	"time"
)

const escapeAliases = `*+^$?\[]{}`
const aliases = `&!;:-%#@~<>/`

var hostName, binDirectory string

func init() {
	var err error
	// Installpath is where the default config and stock external
	// plugins are.
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	binDirectory, err = filepath.Abs(filepath.Dir(ex))
	if err != nil {
		panic(err)
	}
}

func checkPanic(r *Robot, s string) {
	if rcv := recover(); rcv != nil {
		Log(Error, fmt.Sprintf("PANIC from '%s': %s\nStack trace:%s", s, rcv, godebug.Stack()))
		r.Reply(fmt.Sprintf("OUCH! It looks like you found a bug - please ask an admin to check the log and give them this string: '%s'", s))
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}
}

func checkDirectory(cpath string) (string, bool) {
	if len(cpath) == 0 {
		return "", true
	}
	var filePath string
	if path.IsAbs(cpath) {
		filePath = path.Clean(cpath)
	} else {
		filePath = cpath
	}
	ds, err := os.Stat(filePath)
	if err != nil {
		return "", false
	}
	if ds.Mode().IsDir() {
		return filePath, true
	}
	return "", false
}

func (r *Robot) setFormat(format string) MessageFormat {
	format = strings.ToLower(format)
	switch format {
	case "fixed":
		return Fixed
	case "variable":
		return Variable
	case "raw":
		return Raw
	default:
		Log(Error, fmt.Sprintf("Unknown message format '%s', defaulting to 'raw'", format))
		return Raw
	}
}

func setProtocol(proto string) Protocol {
	proto = strings.ToLower(proto)
	switch proto {
	case "slack":
		return Slack
	case "term", "terminal":
		return Terminal
	default:
		return Test
	}
}

func updateRegexes() {
	botCfg.RLock()
	name := botCfg.name
	alias := botCfg.alias
	botCfg.RUnlock()
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
	botCfg.Lock()
	botCfg.preRegex = pre
	botCfg.postRegex = post
	botCfg.bareRegex = bare
	botCfg.Unlock()
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
	// NOTE: the preString regex matches a bare alias, but not a bare name
	if len(name) > 0 {
		postString := `^([^,@]+),?\s+(?i:@?` + name + `)([.?!])?$`
		post, errpost = regexp.Compile(postString)
		bareString := `^@?` + name + `$`
		bare, errbare = regexp.Compile(bareString)
	}
	return
}
