package bot

// membrain is a trivial memory-based implementation of the bot.SimpleBrain
// interface, which gives the robot a place to store it's memories. Memories
// are lost when the robot stops, so this is mainly only useful for testing;
// however, if no other brain is configured, membrain is used as the default.

import (
	"github.com/lnxjedi/gopherbot/robot"
)

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
	}
	// Memory doesn't exist yet
	return datum, false, nil
}

// The file brain doesn't need the logger, but other brains might
func provider(r robot.Handler) robot.SimpleBrain {
	mb := &memBrain{
		memories: make(map[string]*[]byte),
	}
	return mb
}

func init() {
	RegisterSimpleBrain("mem", provider)
}
