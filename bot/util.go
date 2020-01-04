package bot

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	godebug "runtime/debug"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

var idRegex = regexp.MustCompile(`^<(.*)>$`)

// ExtractID is a utility function to check a user/channel string against
// the pattern '<internalID>' and if it matches return the internalID,true;
// otherwise return the unmodified string,false.
func (h handler) ExtractID(u string) (string, bool) {
	matches := idRegex.FindStringSubmatch(u)
	if len(matches) > 0 {
		return matches[1], true
	}
	return u, false
}

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

func bracket(s string) string {
	return "<" + s + ">"
}

func checkPanic(w *worker, s string) {
	if rcv := recover(); rcv != nil {
		Log(robot.Error, "PANIC from '%s': %s\nStack trace:%s", s, rcv, godebug.Stack())
		w.Reply("OUCH! It looks like you found a bug - please ask an admin to check the log and give them this string: '%s'", s)
		time.Sleep(2 * time.Second)
		os.Exit(1)
	}
}

func checkDirectory(cpath string) (string, bool) {
	if len(cpath) == 0 {
		return "", true
	}
	var filePath string
	if filepath.IsAbs(cpath) {
		filePath = filepath.Clean(cpath)
	} else {
		filePath = cpath
	}
	ds, err := os.Stat(filePath)
	if err != nil {
		Log(robot.Debug, "Checking os.Stat for dir '%s' from wd '%s': %v", cpath, homePath, err)
		return "", false
	}
	if ds.Mode().IsDir() {
		return filePath, true
	}
	Log(robot.Debug, "IsDir() for dir '%s' from wd '%s' returned false", cpath, homePath)
	return "", false
}

// getObjectPath looks for an object first in the custom config dir, then
// the install dir.
func getObjectPath(path string) (opath string, err error) {
	if filepath.IsAbs(path) {
		opath = path
		_, err = os.Stat(opath)
		if err == nil {
			Log(robot.Debug, "Using fully specified path to object: %s", opath)
			return opath, nil
		}
		err = fmt.Errorf("Invalid path for object: %s (%v)", opath, err)
		Log(robot.Error, err.Error())
		return "", err
	}
	if len(configPath) > 0 {
		opath = filepath.Join(configPath, path)
		_, err = os.Stat(opath)
		if err == nil {
			Log(robot.Debug, "Loading object from configPath: %s", opath)
			return opath, nil
		}
	}
	opath = filepath.Join(installPath, path)
	if _, err = os.Stat(opath); err == nil {
		Log(robot.Debug, "Loading object from installPath: %s", opath)
		return opath, nil
	}
	return "", err
}

func setFormat(format string) robot.MessageFormat {
	format = strings.ToLower(format)
	switch format {
	case "fixed":
		return robot.Fixed
	case "variable":
		return robot.Variable
	case "raw":
		return robot.Raw
	default:
		Log(robot.Error, "Unknown message format '%s', defaulting to 'raw'", format)
		return robot.Raw
	}
}

// getProtocol takes a string name of the protocol and returns the constant and
// the name of the loadable module, if any.
func getProtocol(proto string) (robot.Protocol, string) {
	proto = strings.ToLower(proto)
	switch proto {
	case "slack":
		return robot.Slack, "slack"
	case "term", "terminal":
		return robot.Terminal, "terminal"
	case "nullconn":
		return robot.Null, "nullconn"
	case "rocket":
		return robot.Rocket, "rocket"
	default:
		return robot.Test, ""
	}
}

func updateRegexes() {
	currentCfg.RLock()
	name := currentCfg.botinfo.UserName
	protoMention := currentCfg.botinfo.protoMention
	alias := currentCfg.alias
	currentCfg.RUnlock()
	pre, post, bare, errpre, errpost, errbare := updateRegexesWrapped(name, protoMention, alias)
	if errpre != nil {
		Log(robot.Error, "Compiling pre regex: %s", errpre)
	}
	if pre != nil {
		Log(robot.Debug, "Setting pre regex to: %s", pre)
	}
	if errpost != nil {
		Log(robot.Error, "Compiling post regex: %s", errpost)
	}
	if post != nil {
		Log(robot.Debug, "Setting post regex to: %s", post)
	}
	if errbare != nil {
		Log(robot.Error, "Compiling bare regex: %s", errbare)
	}
	if bare != nil {
		Log(robot.Debug, "Setting bare regex to: %s", bare)
	}
	regexes.Lock()
	regexes.preRegex = pre
	regexes.postRegex = post
	regexes.bareRegex = bare
	regexes.Unlock()
}

// TODO: write unit test. The regexes produced shouldn't be checked, but rather
// whether given strings do or don't match them. Note: this code is partially
// tested in TestBotName
func updateRegexesWrapped(name, mention string, alias rune) (pre, post, bare *regexp.Regexp, errpre, errpost, errbare error) {
	pre = nil
	post = nil
	if alias == 0 && len(name) == 0 {
		Log(robot.Error, "Robot has no name or alias, and will only respond to direct messages")
		return
	}
	preString := `^`
	names := []string{}
	barenames := []string{}
	if alias != 0 {
		if strings.ContainsRune(string(escapeAliases), alias) {
			names = append(names, `\`+string(alias))
			barenames = append(barenames, `\`+string(alias))
		} else {
			names = append(names, string(alias))
			barenames = append(barenames, string(alias))
		}
	}
	if len(name) > 0 {
		if len(mention) > 0 {
			names = append(names, `(?i:`+name+`)[:, ]`)
			barenames = append(barenames, `(?i:`+name+`)`)
		} else {
			names = append(names, `@?`+name+`[:, ]`)
			barenames = append(barenames, `@?`+name)
		}
	}
	if len(mention) > 0 {
		names = append(names, `@`+mention+`[:, ]`)
		barenames = append(barenames, `@`+mention)
	}
	preString += `^(?i:` + strings.Join(names, "|") + `\s*)(.*)$`
	pre, errpre = regexp.Compile(preString)
	// NOTE: the preString regex matches a bare alias, but not a bare name
	if len(name) > 0 {
		postString := `^([^,@]+),\s+(?i:@?` + name + `)([.?!])?$`
		post, errpost = regexp.Compile(postString)
		bareString := `^@?(?i:` + strings.Join(barenames, "|") + `)$`
		bare, errbare = regexp.Compile(bareString)
	}
	return
}
