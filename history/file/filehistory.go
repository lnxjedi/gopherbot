// Package fileHistory is a simple file-backed implementation for bot plugin
// and job histories.
package fileHistory

import (
	//	"io/ioutil"
	"fmt"
	"log"
	"os"

	"github.com/lnxjedi/gopherbot/bot"
)

var historyPath string
var robot bot.Handler

type historyConfig struct {
	Directory string `yaml:Directory"` // path to histories
}

type historyFile struct {
	l *log.Logger
}

// Log takes a line of text and stores it in the history file
func (hf *historyFile) Log(line string) {

}

// Close sets the logger output to discard and closes the log file
func (hf *historyFile) Close() {

}

var fhc historyConfig

func (fhc *historyConfig) NewHistory(tag string, index, maxHistories int) bot.HistoryLogger {
	hl := log.New(os.Stdout, "", log.LstdFlags)
	return &historyFile{
		hl,
	}
}

// The file brain doesn't need the logger, but other brains might
func provider(r bot.Handler) bot.HistoryProvider {
	robot = r
	robot.GetHistoryConfig(&fhc)
	if byte(fhc.Directory[0]) == byte("/"[0]) {
		historyPath = fhc.Directory
	} else {
		historyPath = robot.GetConfigPath() + "/" + fhc.Directory
	}
	hd, err := os.Stat(historyPath)
	if err != nil {
		robot.Log(bot.Fatal, fmt.Sprintf("Checking history directory '%s': %v", historyPath, err))
	}
	if !hd.Mode().IsDir() {
		robot.Log(bot.Fatal, fmt.Sprintf("Checking history directory: '%s' isn't a directory", historyPath))
	}
	robot.Log(bot.Info, fmt.Sprintf("Initialized file history provider with directory: '%s'", historyPath))
	return &fhc
}

func init() {
	bot.RegisterHistoryProvider("file", provider)
}
