// Package memBrain is a trivial memory-based implementation of the bot.SimpleBrain
// interface, which gives the robot a place to store it's memories. Memories
// are lost when the robot stops, so this is mainly only useful for testing.
package memBrain

import (
	"fmt"
	"log"

	"github.com/lnxjedi/gopherbot/bot"
)

var robot bot.Handler

// NOTE: brains shouldn't need to do their own locking. See bot/brain.go
type memBrain struct {
	memories map[string]*[]byte
}

func (mb *memBrain) Store(k string, b *[]byte) error {
	mb.memories[k] = b
	return nil
}

func (mb *memBrain) Retrieve(k string) (*[]byte, bool, error) {
	datum, exists := mb.memories[k]
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
		memories: make(map[string]*[]byte),
	}
	return mb
}

func init() {
	bot.RegisterSimpleBrain("mem", provider)
}
