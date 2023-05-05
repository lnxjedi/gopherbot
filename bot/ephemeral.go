package bot

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

const ephemeralMemKey = "bot:_ephemeral-memories"

// ephemeral memories held im RAM that expire after a time
type ephemeralMemory struct {
	memory    string
	timestamp time.Time
}

type memoryContext struct {
	key, user, channel, thread string
}

type eMemories struct {
	m map[memoryContext]ephemeralMemory
	sync.Mutex
}

var ephemeralMemories = eMemories{
	m: make(map[memoryContext]ephemeralMemory),
}

func (em *eMemories) MarshalJSON() ([]byte, error) {
	em.Lock()
	defer em.Unlock()

	tempMap := make(map[string]ephemeralMemory)
	for k, v := range em.m {
		keyString := fmt.Sprintf("%s{|}%s{|}%s{|}%s", k.key, k.user, k.channel, k.thread)
		tempMap[keyString] = v
	}

	return json.Marshal(tempMap)
}

func (em *eMemories) UnmarshalJSON(data []byte) error {
	var tempMap map[string]ephemeralMemory
	err := json.Unmarshal(data, &tempMap)
	if err != nil {
		return err
	}

	em.Lock()
	defer em.Unlock()

	em.m = make(map[memoryContext]ephemeralMemory)
	for k, v := range tempMap {
		var key memoryContext
		_, err := fmt.Sscanf(k, "%s{|}%s{|}%s{|}%s", &key.key, &key.user, &key.channel, &key.thread)
		if err != nil {
			return err
		}
		em.m[key] = v
	}

	return nil
}

func restoreEphemeralMemories() {
	// Restore subscriptions and ephemeral memories
	var storedMemories eMemories
	sm_tok, sm_exists, sm_ret := checkoutDatum(ephemeralMemKey, &storedMemories, true)
	if sm_ret == robot.Ok {
		if sm_exists {
			if len(storedMemories.m) > 0 {
				Log(robot.Info, "Restored '%d' ephemeral memories from long-term memory", len(storedMemories.m))
				now := time.Now()
				for _, m := range storedMemories.m {
					m.timestamp = now
				}
				ephemeralMemories.m = storedMemories.m
			} else {
				Log(robot.Info, "Restoring ephemeral memories from long-term memory: zero-length map")
				checkinDatum(ephemeralMemKey, sm_tok)
			}
		} else {
			Log(robot.Info, "Restoring ephemeral memories from long-term memory: memory doesn't exist")
			checkinDatum(ephemeralMemKey, sm_tok)
		}
	} else {
		Log(robot.Error, "Restoring ephemeral memories from long-term memory: error '%s' getting datum", sm_ret)
	}
}
