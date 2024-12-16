package bot

// memhistory provides a trivial history provider that only keeps
// active (un-Finalize()'d) histories in 64k buffers

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/gopherbot/v2/modules/linebuffer"
)

type memlogentry struct {
	tag string
	idx int
}

type memlog struct {
	entry memlogentry
	log   *linebuffer.Buffer
}

type memHistLog struct {
	logs map[memlogentry]memlog
	sync.Mutex
}

type memHistoryConfig struct {
	BufferSize, MaxLineLength int
	Truncated                 string
}

var mhc memHistoryConfig

var memHistories *memHistLog

// Log writes a timestamped line to the buffer
func (m memlog) Log(line string) {
	tsLine := fmt.Sprintf("%s %s", time.Now().Format("Jan 2 15:04:05"), line)
	m.log.WriteLine(tsLine)
}

// Line writes a bare line to a buffer
func (m memlog) Line(line string) {
	m.log.WriteLine(line)
}

// Close closes the log against further writes
func (m memlog) Close() {
	m.log.Close()
}

// Finalize removes the log from the lookup map
func (m memlog) Finalize() {
	memHistories.Lock()
	defer memHistories.Unlock()
	delete(memHistories.logs, m.entry)
}

// NewHistory returns a lineBuffer based history logger
func (h *memHistLog) NewLog(tag string, index, maxHistories int) (robot.HistoryLogger, error) {
	lb := linebuffer.New(mhc.BufferSize, mhc.MaxLineLength, mhc.Truncated)
	entry := memlogentry{tag, index}
	ml := memlog{entry, lb}
	memHistories.Lock()
	defer memHistories.Unlock()
	memHistories.logs[entry] = ml
	return ml, nil
}

// GetLogURL does nothing for mem logs
func (h *memHistLog) GetLogURL(tag string, index int) (string, bool) {
	return "", false
}

// MakeLogURL does nothing for mem logs
func (h *memHistLog) MakeLogURL(tag string, index int) (string, bool) {
	return "", false
}

// GetHistory returns a reader for the log if it exists
func (h *memHistLog) GetLog(tag string, index int) (io.Reader, error) {
	entry := memlogentry{tag, index}
	memHistories.Lock()
	defer memHistories.Unlock()
	mh, ok := memHistories.logs[entry]
	if !ok {
		return nil, errors.New("not found")
	}
	mr, err := mh.log.Reader()
	if err != nil {
		mr = mh.log.Snapshot()
	}
	return mr, nil
}

func mhprovider(r robot.Handler) robot.HistoryProvider {
	r.GetHistoryConfig(&mhc)
	if mhc.BufferSize < 4096 {
		mhc.BufferSize = 4096
	}
	if mhc.MaxLineLength < 1024 {
		mhc.MaxLineLength = 1024
	}
	if mhc.Truncated == "" {
		mhc.Truncated = "<... truncated>"
	}
	memHistories = &memHistLog{
		make(map[memlogentry]memlog),
		sync.Mutex{},
	}
	return memHistories
}

func init() {
	RegisterHistoryProvider("mem", mhprovider)
}
