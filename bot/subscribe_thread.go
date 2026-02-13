package bot

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

/*
Thread subscriptions allow a plugin to Subscribe() to the current thread,
so that all future messages to the thread which aren't commands will be sent
to the plugin with a command of "subscribed". This is meant to replace message
matchers that match all messages.
*/

/*
NOTE on Marshalling, Unmarshalling and locks:
The Go linter is complaining about copying locks, but in reality we're not using the lock
that's being copied anyway.
*/

const subscriptionMemKey = "bot:_subscriptions"

// Plugins can subscribe to a thread in a channel. This struct is used as the
// key in the subscriptions map.
type subscriptionMatcher struct {
	protocol, channel, thread string
}

type subscriber struct {
	Plugin    string    // the plugin subscribed
	Timestamp time.Time // for expiring after subscriptionTimeout
}

type tSubs struct {
	m     map[subscriptionMatcher]subscriber
	dirty bool
	sync.Mutex
}

var subscriptions = tSubs{
	m: make(map[subscriptionMatcher]subscriber),
}

// the lock should be held on entry and released after return
func (s tSubs) MarshalJSON() ([]byte, error) {
	tempMap := make(map[string]subscriber)
	for k, v := range s.m {
		keyString := fmt.Sprintf("%s{|}%s{|}%s", k.protocol, k.channel, k.thread)
		tempMap[keyString] = v
	}
	return json.Marshal(tempMap)
}

// No locking needed - called before multi-threaded
func (s *tSubs) UnmarshalJSON(data []byte) error {
	var tempMap map[string]subscriber
	err := json.Unmarshal(data, &tempMap)
	if err != nil {
		return err
	}

	Log(robot.Debug, "Unmarshalled data: %v", tempMap)

	s.m = make(map[subscriptionMatcher]subscriber)
	for k, v := range tempMap {
		parts := strings.SplitN(k, "{|}", 3)
		switch len(parts) {
		case 2:
			// Legacy format: channel{|}thread
			key := subscriptionMatcher{protocol: "", channel: parts[0], thread: parts[1]}
			s.m[key] = v
		case 3:
			key := subscriptionMatcher{protocol: normalizeProtocolName(parts[0]), channel: parts[1], thread: parts[2]}
			s.m[key] = v
		default:
			return fmt.Errorf("invalid key string format: %s", k)
		}
	}

	return nil
}

func restoreSubscriptions() {
	var storedSubscriptions tSubs
	ss_tok, ss_exists, ss_ret := checkoutDatum(subscriptionMemKey, &storedSubscriptions, true)
	if ss_ret == robot.Ok {
		if ss_exists {
			if len(storedSubscriptions.m) > 0 {
				Log(robot.Info, "Restored '%d' subscriptions from long-term memory", len(storedSubscriptions.m))
				now := time.Now()
				for key, s := range storedSubscriptions.m {
					s.Timestamp = now
					storedSubscriptions.m[key] = s
				}
				subscriptions.m = storedSubscriptions.m
			} else {
				Log(robot.Info, "Restoring subscriptions from long-term memory: zero-length map")
				checkinDatum(subscriptionMemKey, ss_tok)
			}
		} else {
			Log(robot.Info, "Restoring suscriptions from long-term memory: memory doesn't exist")
			checkinDatum(subscriptionMemKey, ss_tok)
		}
	} else {
		Log(robot.Error, "Restoring suscriptions from long-term memory: error '%s' getting datum", ss_ret)
	}
}

func saveSubscriptions() {
	var storedSubscriptions tSubs
	ss_tok, _, ss_ret := checkoutDatum(subscriptionMemKey, &storedSubscriptions, true)
	if ss_ret == robot.Ok {
		subscriptions.Lock()
		storedSubscriptions.m = subscriptions.m
		subscriptions.dirty = false
		ret := updateDatum(subscriptionMemKey, ss_tok, storedSubscriptions)
		// NOTE: Hold the lock until after serializing - the
		// storedSubscriptions assignment doesn't copy.
		subscriptions.Unlock()
		if ret == robot.Ok {
			Log(robot.Debug, "Successfully saved '%d' long-term subscription memories", len(storedSubscriptions.m))
		} else {
			Log(robot.Error, "Error '%s' updating long-term subscription memory", ret)
		}
	} else {
		Log(robot.Error, "Saving suscriptions to long-term memory: error '%s' getting datum", ss_ret)
	}
}

// Subscribe allows a plugin to subscribe to it's current thread and
// receive all future responses. It returns a boolean - true on success,
// or false when a thread is already subscribed, or when called by anything
// other than a plugin. When false, an Error log message is generated.
func (r Robot) Subscribe() (success bool) {
	w := getLockedWorker(r.tid)
	defer w.Unlock()
	protocol := protocolFromIncoming(w.Incoming, w.Protocol)
	task, plugin, _ := getTask(r.currentTask)
	if plugin == nil {
		w.Log(robot.Error, "Subscribe called by non-plugin task '%s'", task.name)
		return false
	}
	subscriptionSpec := subscriptionMatcher{protocol: protocol, channel: w.Channel, thread: w.Incoming.ThreadID}
	legacySpec := subscriptionMatcher{channel: w.Channel, thread: w.Incoming.ThreadID}
	subscriptions.Lock()
	defer subscriptions.Unlock()
	if subscription, ok := subscriptions.m[subscriptionSpec]; ok {
		if task.name != subscription.Plugin {
			w.Log(robot.Error, "Subscribe - plugin '%s' failed subscribing on protocol '%s' to thread '%s' in channel '%s', subscription already held by plugin '%s'", task.name, protocol, w.Incoming.ThreadID, w.Channel, subscription.Plugin)
			return false
		} else {
			w.Log(robot.Debug, "Subscribe - already subscribed; plugin '%s' subscribing on protocol '%s' to thread '%s' in channel '%s'", task.name, protocol, w.Incoming.ThreadID, w.Channel)
			return true
		}
	}
	// Backward-compatibility path for restored pre-protocol keys.
	if protocol != "" {
		if subscription, ok := subscriptions.m[legacySpec]; ok {
			if task.name != subscription.Plugin {
				w.Log(robot.Error, "Subscribe - plugin '%s' failed subscribing on protocol '%s' to thread '%s' in channel '%s', legacy subscription held by plugin '%s'", task.name, protocol, w.Incoming.ThreadID, w.Channel, subscription.Plugin)
				return false
			}
			delete(subscriptions.m, legacySpec)
			subscriptions.m[subscriptionSpec] = subscriber{
				Plugin:    task.name,
				Timestamp: time.Now(),
			}
			subscriptions.dirty = true
			w.Log(robot.Debug, "Subscribe - migrated legacy subscription for plugin '%s' onto protocol '%s' to thread '%s' in channel '%s'", task.name, protocol, w.Incoming.ThreadID, w.Channel)
			return true
		}
	}
	subscriptions.m[subscriptionSpec] = subscriber{
		Plugin:    task.name,
		Timestamp: time.Now(),
	}
	subscriptions.dirty = true
	w.Log(robot.Debug, "Subscribe - plugin '%s' successfully subscribed on protocol '%s' to thread '%s' in channel '%s'", task.name, protocol, w.Incoming.ThreadID, w.Channel)
	return true
}

// Unsubscribe unsubscribes from the thread, returning true on success
// or false if no subscription was found. Generally, the return value
// can be ignored.
func (r Robot) Unsubscribe() (success bool) {
	w := getLockedWorker(r.tid)
	defer w.Unlock()
	protocol := protocolFromIncoming(w.Incoming, w.Protocol)
	task, plugin, _ := getTask(r.currentTask)
	if plugin == nil {
		w.Log(robot.Error, "Unsubscribe called by non-plugin task '%s'", task.name)
		return false
	}
	subscriptionSpec := subscriptionMatcher{protocol: protocol, channel: w.Channel, thread: w.Incoming.ThreadID}
	legacySpec := subscriptionMatcher{channel: w.Channel, thread: w.Incoming.ThreadID}
	subscriptions.Lock()
	defer subscriptions.Unlock()
	if subscription, ok := subscriptions.m[subscriptionSpec]; ok {
		if task.name != subscription.Plugin {
			w.Log(robot.Error, "Unsubscribe - plugin '%s' failed unsubscribing on protocol '%s' from thread '%s' in channel '%s', subscription held by other plugin '%s'", task.name, protocol, w.Incoming.ThreadID, w.Channel, subscription.Plugin)
			return false
		} else {
			w.Log(robot.Debug, "Unsubscribe - plugin '%s' unsubscribing on protocol '%s' from thread '%s' in channel '%s'", task.name, protocol, w.Incoming.ThreadID, w.Channel)
			delete(subscriptions.m, subscriptionSpec)
			subscriptions.dirty = true
			return true
		}
	}
	if protocol != "" {
		if subscription, ok := subscriptions.m[legacySpec]; ok {
			if task.name != subscription.Plugin {
				w.Log(robot.Error, "Unsubscribe - plugin '%s' failed unsubscribing on protocol '%s' from thread '%s' in channel '%s', legacy subscription held by other plugin '%s'", task.name, protocol, w.Incoming.ThreadID, w.Channel, subscription.Plugin)
				return false
			}
			w.Log(robot.Debug, "Unsubscribe - plugin '%s' unsubscribing legacy entry on protocol '%s' from thread '%s' in channel '%s'", task.name, protocol, w.Incoming.ThreadID, w.Channel)
			delete(subscriptions.m, legacySpec)
			subscriptions.dirty = true
			return true
		}
	}
	w.Log(robot.Warn, "Unsubscribe - plugin '%s' not subscribed on protocol '%s' to thread '%s' in channel '%s'", task.name, protocol, w.Incoming.ThreadID, w.Channel)
	return true
}

func lookupSubscriptionLocked(protocol, channel, thread string) (subscriptionMatcher, subscriber, bool) {
	spec := subscriptionMatcher{protocol: normalizeProtocolName(protocol), channel: channel, thread: thread}
	if subscription, ok := subscriptions.m[spec]; ok {
		return spec, subscription, true
	}
	legacy := subscriptionMatcher{channel: channel, thread: thread}
	if spec.protocol != "" {
		if subscription, ok := subscriptions.m[legacy]; ok {
			return legacy, subscription, true
		}
	}
	return subscriptionMatcher{}, subscriber{}, false
}

// expireSubscriptions is called by the brainTicker
func expireSubscriptions(now time.Time) bool {
	subscriptions.Lock()
	for subscription, subscriber := range subscriptions.m {
		if now.Sub(subscriber.Timestamp) > threadMemoryDuration {
			delete(subscriptions.m, subscription)
			subscriptions.dirty = true
			protocol := subscription.protocol
			if protocol == "" {
				protocol = "unknown"
			}
			Log(robot.Debug, "Subscribe - expiring subscription for plugin '%s' on protocol '%s' to thread '%s' in channel '%s'", subscriber.Plugin, protocol, subscription.thread, subscription.channel)
		}
	}
	updated := subscriptions.dirty
	subscriptions.Unlock()
	return updated
}
