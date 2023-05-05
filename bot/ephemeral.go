package bot

import (
	"encoding/json"
	"fmt"
	"strings"
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

// the lock should be held on entry and released after return
func (em eMemories) MarshalJSON() ([]byte, error) {
	tempMap := make(map[string]ephemeralMemory)
	for k, v := range em.m {
		// No need to thrash the brain with lastMsg memories
		if k.key == lastMsgKey {
			continue
		}
		keyString := fmt.Sprintf("%s{|}%s{|}%s{|}%s", k.key, k.user, k.channel, k.thread)
		tempMap[keyString] = v
	}
	return json.Marshal(tempMap)
}

// No locking needed - called before multi-threaded
func (e *eMemories) UnmarshalJSON(data []byte) error {
	var tempMap map[string]ephemeralMemory
	err := json.Unmarshal(data, &tempMap)
	if err != nil {
		return err
	}
	e.m = make(map[memoryContext]ephemeralMemory)
	for k, v := range tempMap {
		parts := strings.SplitN(k, "{|}", 4)
		if len(parts) != 4 {
			return fmt.Errorf("invalid key string format: %s", k)
		}
		key := memoryContext{
			key:     parts[0],
			user:    parts[1],
			channel: parts[2],
			thread:  parts[3],
		}
		e.m[key] = v
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

// NOTE: subscriptions is already locked on entry, and unlocks after exit
func saveEphemeralMemories() {
	var storedEphemeralMemories eMemories
	sm_tok, _, sm_ret := checkoutDatum(ephemeralMemKey, &storedEphemeralMemories, true)
	if sm_ret == robot.Ok {
		storedEphemeralMemories.m = ephemeralMemories.m
		ret := updateDatum(ephemeralMemKey, sm_tok, storedEphemeralMemories)
		if ret == robot.Ok {
			Log(robot.Debug, "Successfully saved '%d' ephemeral memories to long-term memory", len(storedEphemeralMemories.m))
		} else {
			Log(robot.Error, "Error '%s' updating ephemeral memories in long-term memory", ret)
		}
	} else {
		Log(robot.Error, "Saving ephemeral memories to long-term memory: error '%s' getting datum", sm_ret)
	}
}
