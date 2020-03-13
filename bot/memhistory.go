package bot

// memhistory provides a trivial history provider that only keeps
// active (un-Finalize()'d) histories in 64k buffers

import (
	"errors"
	"io"
	"sync"

	"github.com/lnxjedi/robot"
)

const logSize = 65536
const maxLogLine = 8092
const trunc = "<... truncated>"

type memlogentry struct {
	tag string
	idx int
}

type memlog struct {
	entry memlogentry
	log   *lineBuffer
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

// Log writes a line to the buffer
func (m memlog) Log(line string) {
	m.log.writeLine(line)
}

// Section adds a new section to the log
func (m memlog) Section(task, desc string) {
	m.log.writeLine("*** " + task + " - " + desc)
}

// Close closes the log against further writes
func (m memlog) Close() {
	m.log.close()
}

// Finalize removes the log from the lookup map
func (m memlog) Finalize() {
	memHistories.Lock()
	defer memHistories.Unlock()
	delete(memHistories.logs, m.entry)
}

// NewHistory returns a lineBuffer based history logger
func (h *memHistLog) NewLog(tag string, index, maxHistories int) (robot.HistoryLogger, error) {
	lb := newlineBuffer(mhc.BufferSize, mhc.MaxLineLength, mhc.Truncated)
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
		return nil, errors.New("Not found")
	}
	mr, err := mh.log.getReader()
	if err != nil {
		mr = mh.log.copyReader()
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
