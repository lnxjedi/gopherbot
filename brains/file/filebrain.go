package fileBrain

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/parsley42/gopherbot/bot"
)

var brainPath string
var logger bot.Logger

type brainConfig struct {
	BrainDirectory string // path to brain files
}

var fb brainConfig

func (fb *brainConfig) Store(k string, b []byte) error {
	datumPath := brainPath + "/" + k
	if err := ioutil.WriteFile(datumPath, b, 0644); err != nil {
		logger.Log(bot.Error, fmt.Sprintf("Writing datum \"%s\": %v", datumPath, err))
		return err
	}
	return nil
}

func (fb *brainConfig) Retrieve(k string) ([]byte, bool) {
	datumPath := brainPath + "/" + k
	datum, err := ioutil.ReadFile(datumPath)
	if err != nil {
		logger.Log(bot.Debug, "Error reading file \"%s\": %v", datumPath, err)
		return []byte(""), false
	}
	return datum, true
}

func provider(l bot.Logger, j json.RawMessage) bot.SimpleBrain {
	json.Unmarshal(j, &fb)
	if byte(fb.BrainDirectory[0]) == byte("/"[0]) {
		brainPath = fb.BrainDirectory
	} else {
		brainPath = os.Getenv("GOPHER_LOCALDIR") + "/" + fb.BrainDirectory
	}
	bd, err := os.Stat(brainPath)
	if err != nil {
		log.Fatalf("Checking brain directory \"%s\": %v", err)
	}
	if !bd.Mode().IsDir() {
		log.Fatalf("Checking brain directory: \"%s\" isn't a directory", brainPath)
	}
	logger = l
	return &fb
}

func init() {
	bot.RegisterBrain("file", provider)
}
