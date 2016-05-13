package bot

import (
	"encoding/json"
	"log"
	"regexp"
)

var brains map[string]func(l Logger, conf json.RawMessage) interface{} = make(map[string]func(Logger, json.RawMessage) interface{})

const keyRegex = `[\w:]+` // keys can ony be word chars + separator (:)
var keyRe = regexp.MustCompile(keyRegex)

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
