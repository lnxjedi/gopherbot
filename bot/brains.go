package bot

import (
	"encoding/json"
	"log"
	"regexp"
	"sync"
	"time"
)

// Maximum amount of time that a plugin can hold a lock on a datum.
// If the plugin tries to UpdateDatum() after the timeout expires, it'll
// get an error.
const lockTimeout = time.Second

var brains map[string]func(l Logger, conf json.RawMessage) interface{} = make(map[string]func(Logger, json.RawMessage) interface{})

type datum struct {
	done chan struct{} // signal that the datum has been checked
}

var datumLock sync.RWMutex
var datumLocks map[string]datum = make(map[string]datum)

const keyRegex = `[\w:]+` // keys can ony be word chars + separator (:)
var keyRe = regexp.MustCompile(keyRegex)

// Checkout returns the datum for the key, or blocks if it's already checked
// out. A datum can
func (r *Robot) CheckOut(key string) []byte {
	b := r.robot
	datumLock.RLock()
	d, ok := datumLocks[key]
	if ok { // there's already a goroutine with a lock

	}
	return nil
}

// RegisterBrain allows brain implementations to register a function with a named
// brain type that returns an XXXBrain interface (currently only SimpleBrain).
// When the bot initializes, it will look for a function registered under the configured
// "Brain" in gopherbot.json, then pass in rawJSON config and get back an interface.
// This can only be called from a brain provider's init() function(s). Pass in a Logger
// so the brain can log error messages if needed.
func RegisterBrain(name string, provider func(Logger, json.RawMessage) interface{}) {
	if stopRegistrations {
		return
	}
	if brains[name] != nil {
		log.Fatal("Attempted registration of duplicate brain provider name:", name)
	}
	brains[name] = provider
}
