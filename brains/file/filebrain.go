package fileBrain

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/parsley42/gopherbot/bot"
)

var brainPath string
var robot bot.Handler

// go-yaml assumes lowercase keys, so we need to use a struct tag for camel case
type brainConfig struct {
	BrainDirectory string `yaml:"BrainDirectory"` // path to brain files
}

var fb brainConfig

func (fb *brainConfig) Store(k string, b []byte) error {
	datumPath := brainPath + "/" + k
	if err := ioutil.WriteFile(datumPath, b, 0644); err != nil {
		return fmt.Errorf("Writing datum \"%s\": %v", datumPath, err)
	}
	return nil
}

func (fb *brainConfig) Retrieve(k string) (datum []byte, exists bool, err error) {
	datumPath := brainPath + "/" + k
	if _, err := os.Stat(datumPath); err == nil {
		exists = true
	}
	datum, err = ioutil.ReadFile(datumPath)
	if err != nil {
		err = fmt.Errorf("Error reading file \"%s\": %v", datumPath, err)
	}
	return
}

// The file brain doesn't need the logger, but other brains might
func provider(r bot.Handler, _ *log.Logger) bot.SimpleBrain {
	robot = r
	robot.GetBrainConfig(&fb)
	if byte(fb.BrainDirectory[0]) == byte("/"[0]) {
		brainPath = fb.BrainDirectory
	} else {
		brainPath = robot.GetLocalPath() + "/" + fb.BrainDirectory
	}
	bd, err := os.Stat(brainPath)
	if err != nil {
		robot.Log(bot.Fatal, fmt.Sprintf("Checking brain directory \"%s\": %v", err))
	}
	if !bd.Mode().IsDir() {
		robot.Log(bot.Fatal, fmt.Sprintf("Checking brain directory: \"%s\" isn't a directory", brainPath))
	}
	return &fb
}

func init() {
	bot.RegisterSimpleBrain("file", provider)
}
