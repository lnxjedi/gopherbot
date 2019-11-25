// Package fileBrain is a simple file-based implementation of the bot.SimpleBrain
// interface, which gives the robot a place to store it's memories.
package fileBrain

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/lnxjedi/gopherbot/bot"
	"github.com/lnxjedi/gopherbot/robot"
)

var brainPath string
var handler robot.Handler

type brainConfig struct {
	BrainDirectory string `yaml:"BrainDirectory"` // path to brain files
}

var fb brainConfig

func (fb *brainConfig) Store(k string, b *[]byte) error {
	k = strings.Replace(k, `/`, ":", -1)
	k = strings.Replace(k, `\`, ":", -1)
	datumPath := brainPath + "/" + k
	if err := ioutil.WriteFile(datumPath, *b, 0644); err != nil {
		return fmt.Errorf("Writing datum \"%s\": %v", datumPath, err)
	}
	return nil
}

func (fb *brainConfig) Retrieve(k string) (*[]byte, bool, error) {
	k = strings.Replace(k, `/`, ":", -1)
	k = strings.Replace(k, `\`, ":", -1)
	datumPath := brainPath + "/" + k
	if _, err := os.Stat(datumPath); err == nil {
		datum, err := ioutil.ReadFile(datumPath)
		if err != nil {
			err = fmt.Errorf("Error reading file \"%s\": %v", datumPath, err)
			handler.Log(robot.Error, err.Error())
			return nil, false, err
		}
		return &datum, true, nil
	}
	// Memory doesn't exist yet
	return nil, false, nil
}

// The file brain doesn't need the logger, but other brains might
func provider(r robot.Handler) robot.SimpleBrain {
	handler = r
	handler.GetBrainConfig(&fb)
	if len(fb.BrainDirectory) == 0 {
		handler.Log(robot.Fatal, "BrainConfig missing value for BrainDirectory required by 'file' brain")
	}
	brainPath = fb.BrainDirectory
	bd, err := os.Stat(brainPath)
	if err != nil {
		handler.Log(robot.Fatal, "Checking brain directory \"%s\": %v", brainPath, err)
	}
	if !bd.Mode().IsDir() {
		handler.Log(robot.Fatal, "Checking brain directory: \"%s\" isn't a directory", brainPath)
	}
	handler.Log(robot.Info, "Initialized file-backed brain with memories directory: '%s'", brainPath)
	return &fb
}

func init() {
	bot.RegisterSimpleBrain("file", provider)
}
