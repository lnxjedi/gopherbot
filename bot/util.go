package bot

import (
	"fmt"
	"io/fs"
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
func getObjectPath(path string) (opath string, info fs.FileInfo, err error) {
	if filepath.IsAbs(path) {
		opath = path
		info, err = os.Stat(opath)
		if err == nil {
			Log(robot.Debug, "Using fully specified path to object: %s", opath)
			return opath, info, nil
		}
		err = fmt.Errorf("invalid path for object: %s (%v)", opath, err)
		Log(robot.Error, err.Error())
		return "", nil, err
	}
	if len(configPath) > 0 {
		opath = filepath.Join(configPath, path)
		info, err = os.Stat(opath)
		if err == nil {
			Log(robot.Debug, "Loading object from configPath: %s", opath)
			return opath, info, nil
		}
	}
	opath = filepath.Join(installPath, path)
	if info, err = os.Stat(opath); err == nil {
		Log(robot.Debug, "Loading object from installPath: %s", opath)
		return opath, info, nil
	}
	return "", nil, err
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

// getProtocol takes a string name of the protocol and returns the constant.
func getProtocol(proto string) robot.Protocol {
	proto = strings.ToLower(proto)
	switch proto {
	case "slack":
		return robot.Slack
	case "term", "terminal":
		return robot.Terminal
	case "nullconn":
		return robot.Null
	case "rocket":
		return robot.Rocket
	case "ssh":
		return robot.SSH
	default:
		return robot.Test
	}
}
