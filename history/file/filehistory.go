// Package fileHistory is a simple file-backed implementation for bot plugin
// and job histories.
package fileHistory

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/lnxjedi/gopherbot/bot"
)

var historyPath string
var robot bot.Handler

// TODO: move to bot.historyStdFlags
const logFlags = log.LstdFlags

type historyConfig struct {
	Directory string `yaml:Directory"` // path to histories
}

type historyFile struct {
	l *log.Logger
	f *os.File
}

// Log takes a line of text and stores it in the history file
func (hf *historyFile) Log(line string) {
	hf.l.Println(line)
}

// TODO: This belongs in the Robot as a generic method - move
// Section creates a new named section in the history file, for separating
// output from jobs/plugins in a pipeline
func (hf *historyFile) Section(task, desc string) {
	hf.l.SetFlags(0)
	hf.l.Println("*** " + task + " - " + desc)
	hf.l.SetFlags(logFlags)
}

// Close sets the logger output to discard and closes the log file
func (hf *historyFile) Close() {
	hf.l.SetOutput(ioutil.Discard)
	hf.f.Close()
}

var fhc historyConfig

// NewHistory initializes and returns a historyFile, as well as cleaning up old
// logs.
func (fhc *historyConfig) NewHistory(tag string, index, maxHistories int) (bot.HistoryLogger, error) {
	tag = strings.Replace(tag, `\`, ":", -1)
	tag = strings.Replace(tag, `/`, ":", -1)
	dirPath := path.Join(fhc.Directory, tag)
	filePath := path.Join(dirPath, fmt.Sprintf("%s-%d.log", tag, index))
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("Error creating history directory '%s': %v", dirPath, err)
	}
	if file, err := os.Create(filePath); err != nil {
		return nil, fmt.Errorf("Error creating history file '%s': %v", filePath, err)
	} else {
		hl := log.New(file, "", logFlags)
		hf := &historyFile{
			hl,
			file,
		}
		if index-maxHistories >= 0 {
			for i := index - maxHistories; i >= 0; i-- {
				rmPath := path.Join(dirPath, fmt.Sprintf("%s-%d.log", tag, i))
				_, err := os.Stat(rmPath)
				if err != nil {
					break
				}
				rerr := os.Remove(rmPath)
				if rerr != nil {
					robot.Log(bot.Error, fmt.Sprintf("Error removing old log file '%s': %v", rmPath, rerr))
					// assume it's pointless to keep trying to delete files
					break
				}
			}
		}
		return hf, nil
	}
}

// GetHistory returns an io.Reader
func (fhc *historyConfig) GetHistory(tag string, index int) (io.Reader, error) {
	tag = strings.Replace(tag, `\`, ":", -1)
	tag = strings.Replace(tag, `/`, ":", -1)
	dirPath := path.Join(fhc.Directory, tag)
	filePath := path.Join(dirPath, fmt.Sprintf("%s-%d.log", tag, index))
	return os.Open(filePath)
}

func provider(r bot.Handler) bot.HistoryProvider {
	robot = r
	robot.GetHistoryConfig(&fhc)
	if len(fhc.Directory) == 0 {
		robot.Log(bot.Fatal, "HistoryConfig missing value for Directory required by 'file' history provider")
	}
	historyPath = fhc.Directory
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
