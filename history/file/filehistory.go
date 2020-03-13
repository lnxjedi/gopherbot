// Package filehistory is a simple file-backed implementation for bot plugin
// and job histories.
package filehistory

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/lnxjedi/robot"
)

var historyPath string
var handler robot.Handler

// TODO: move to bot.historyStdFlags
const logFlags = log.LstdFlags

type historyConfig struct {
	Directory string `yaml:"Directory"` // path to histories
	URLPrefix string `yaml:"URLPrefix"` // Optional URL prefix corresponding to the Directory
	// If LocalPort set, passed to http.ListenAndServe to serve static files
	LocalPort string `yaml:"LocalPort"`
}

type historyFile struct {
	l    *log.Logger
	f    *os.File
	path string
	keep bool
}

// Log takes a line of text and stores it in the history file
func (hf *historyFile) Log(line string) {
	hf.l.Println(line)
}

// Section creates a new named section in the history file, for separating
// output from jobs/plugins in a pipeline
func (hf *historyFile) Line(line string) {
	hf.l.SetFlags(0)
	hf.l.Println(line)
	hf.l.SetFlags(logFlags)
}

// Close sets the logger output to discard and closes the log file
func (hf *historyFile) Close() {
	hf.l.SetOutput(ioutil.Discard)
	hf.f.Close()
}

// Finalize removes the log if needed
func (hf *historyFile) Finalize() {
	if hf.keep {
		return
	}
	if rerr := os.Remove(hf.path); rerr != nil {
		handler.Log(robot.Error, "Removing %s: %v", hf.path, rerr)
	}
}

var fhc historyConfig

// NewLog initializes and returns a historyFile, as well as cleaning up old
// logs.
func (fhc *historyConfig) NewLog(tag string, index, maxHistories int) (robot.HistoryLogger, error) {
	tag = strings.Replace(tag, `\`, ":", -1)
	tag = strings.Replace(tag, `/`, ":", -1)
	dirPath := path.Join(fhc.Directory, tag)
	filePath := path.Join(dirPath, fmt.Sprintf("run-%d.log", index))
	handler.RaisePriv("creating new log for " + tag)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("Error creating history directory '%s': %v", dirPath, err)
	}
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("Error creating history file '%s': %v", filePath, err)
	}
	keep := maxHistories != 0
	hl := log.New(file, "", logFlags)
	hf := &historyFile{
		hl,
		file,
		filePath,
		keep,
	}
	if index-maxHistories >= 0 {
		for i := index - maxHistories; i >= 0; i-- {
			rmPath := path.Join(dirPath, fmt.Sprintf("run-%d.log", i))
			_, err := os.Stat(rmPath)
			if err != nil {
				break
			}
			rerr := os.Remove(rmPath)
			if rerr != nil {
				handler.Log(robot.Error, "Error removing old log file '%s': %v", rmPath, rerr)
				// assume it's pointless to keep trying to delete files
				break
			}
		}
	}
	return hf, nil
}

// GetLog returns an io.Reader
func (fhc *historyConfig) GetLog(tag string, index int) (io.Reader, error) {
	tag = strings.Replace(tag, `\`, ":", -1)
	tag = strings.Replace(tag, `/`, ":", -1)
	dirPath := path.Join(fhc.Directory, tag)
	filePath := path.Join(dirPath, fmt.Sprintf("run-%d.log", index))
	return os.Open(filePath)
}

// GetLogURL returns the permanent link to the history
func (fhc *historyConfig) GetLogURL(tag string, index int) (string, bool) {
	if len(fhc.URLPrefix) == 0 {
		return "", false
	}
	tag = strings.Replace(tag, `\`, ":", -1)
	tag = strings.Replace(tag, `/`, ":", -1)
	prefix := strings.TrimRight(fhc.URLPrefix, "/")
	htmlPath := fmt.Sprintf("%s/%s/run-%d.log", prefix, tag, index)
	return htmlPath, true
}

// MakeLogURL publishes a history to a URL and returns the URL
func (fhc *historyConfig) MakeLogURL(tag string, index int) (string, bool) {
	return "", false
}

func provider(r robot.Handler) robot.HistoryProvider {
	handler = r
	handler.GetHistoryConfig(&fhc)
	if len(fhc.Directory) == 0 {
		handler.Log(robot.Error, "HistoryConfig missing value for Directory required by 'file' history provider")
		return nil
	}
	historyPath = fhc.Directory
	handler.RaisePriv("initializing file history")
	if err := r.GetDirectory(historyPath); err != nil {
		handler.Log(robot.Error, "Checking history directory '%s': %v", historyPath, err)
		return nil
	}
	if len(fhc.LocalPort) > 0 {
		go func() {
			handler.Log(robot.Info, "Starting fileserver listener for file history provider")
			log.Fatal(http.ListenAndServe(fhc.LocalPort, http.FileServer(http.Dir(historyPath))))
		}()
	}
	if len(fhc.LocalPort) > 0 {
		handler.Log(robot.Info, "Initialized file history provider with directory: '%s'; serving on: '%s'", historyPath, fhc.LocalPort)
	} else {
		handler.Log(robot.Info, "Initialized file history provider with directory: '%s'", historyPath)
	}
	return &fhc
}
