// Package memBrain is a trivial memory-based implementation of the bot.SimpleBrain
// interface, which gives the robot a place to store it's memories. Memories
// are lost when the robot stops, so this is mainly only useful for testing.
package memBrain

import (
	"fmt"
	"log"
	"sync"

	"github.com/lnxjedi/gopherbot/bot"
)

var robot bot.Handler

// Note on locking: shouldn't be needed. The API grants RW access via lock token
// to a single plugin at a time.
type memBrain struct {
	memories map[string][]byte
	sync.Mutex
}

func (mb *memBrain) Store(k string, b []byte) error {
	mb.Lock()
	mb.memories[k] = b
	mb.Unlock()
	return nil
}

func (mb *memBrain) Retrieve(k string) ([]byte, bool, error) {
	mb.Lock()
	datum, exists := mb.memories[k]
	mb.Unlock()
	if exists {
		return datum, true, nil
	} else { // Memory doesn't exist yet
		robot.Log(bot.Info, fmt.Sprintf("Retrieve called on non-existing key \"%s\"", k))
		return datum, false, nil
	}
}

// The file brain doesn't need the logger, but other brains might
func provider(r bot.Handler, _ *log.Logger) bot.SimpleBrain {
	robot = r

	mb := &memBrain{
		memories: make(map[string][]byte),
	}
	return mb
}

func init() {
	bot.RegisterSimpleBrain("mem", provider)
}
