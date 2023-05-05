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

const subscriptionTimeout = 14 * 24 * time.Hour

const subscriptionMemKey = "bot:_subscriptions"

// Plugins can subscribe to a thread in a channel. This struct is used as the
// key in the subscriptions map.
type subscriptionMatcher struct {
	channel, thread string
}

type subscriber struct {
	Plugin    string    // the plugin subscribed
	Timestamp time.Time // for expiring after subscriptionTimeout
}

type tSubs struct {
	m map[subscriptionMatcher]subscriber
	sync.Mutex
}

var subscriptions = tSubs{
	m: make(map[subscriptionMatcher]subscriber),
}

// the lock should be held on entry and released after return
func (s tSubs) MarshalJSON() ([]byte, error) {
	tempMap := make(map[string]subscriber)
	for k, v := range s.m {
		keyString := fmt.Sprintf("%s{|}%s", k.channel, k.thread)
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
		parts := strings.SplitN(k, "{|}", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid key string format: %s", k)
		}
		key := subscriptionMatcher{channel: parts[0], thread: parts[1]}
		s.m[key] = v
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
				for _, s := range storedSubscriptions.m {
					s.Timestamp = now
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

// NOTE: subscriptions is already locked on entry, and unlocks after exit
func saveSubscriptions() {
	var storedSubscriptions tSubs
	ss_tok, _, ss_ret := checkoutDatum(subscriptionMemKey, &storedSubscriptions, true)
	if ss_ret == robot.Ok {
		storedSubscriptions.m = subscriptions.m
		ret := updateDatum(subscriptionMemKey, ss_tok, storedSubscriptions)
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
	task, plugin, _ := getTask(r.currentTask)
	if plugin == nil {
		r.Log(robot.Error, "Subscribe called by non-plugin task '%s'", task.name)
		return false
	}
	subscriptionSpec := subscriptionMatcher{w.Channel, w.ThreadID}
	subscriptions.Lock()
	defer subscriptions.Unlock()
	if subscription, ok := subscriptions.m[subscriptionSpec]; ok {
		if task.name != subscription.Plugin {
			r.Log(robot.Error, "Subscribe - plugin '%s' failed subscribing to thread '%s' in channel '%s', subscription already held by plugin '%s'", task.name, w.ThreadID, w.Channel, subscription.Plugin)
			return false
		} else {
			r.Log(robot.Debug, "Subscribe - already subscribed; plugin '%s' subscribing to thread '%s' in channel '%s'", task.name, w.ThreadID, w.Channel)
			return true
		}
	}
	subscriptions.m[subscriptionSpec] = subscriber{
		Plugin:    task.name,
		Timestamp: time.Now(),
	}
	saveSubscriptions()
	r.Log(robot.Debug, "Subscribe - plugin '%s' successfully subscribed to thread '%s' in channel '%s'", task.name, w.ThreadID, w.Channel)
	return true
}

// Unsubscribe unsubscribes from the thread, returning true on success
// or false if no subscription was found. Generally, the return value
// can be ignored.
func (r Robot) Unsubscribe() (success bool) {
	w := getLockedWorker(r.tid)
	defer w.Unlock()
	task, plugin, _ := getTask(r.currentTask)
	if plugin == nil {
		r.Log(robot.Error, "Unsubscribe called by non-plugin task '%s'", task.name)
		return false
	}
	subscriptionSpec := subscriptionMatcher{w.Channel, w.ThreadID}
	subscriptions.Lock()
	defer subscriptions.Unlock()
	if subscription, ok := subscriptions.m[subscriptionSpec]; ok {
		if task.name != subscription.Plugin {
			r.Log(robot.Error, "Unsubscribe - plugin '%s' failed subscribing to thread '%s' in channel '%s', subscription held by other plugin '%s'", task.name, w.ThreadID, w.Channel, subscription.Plugin)
			return false
		} else {
			r.Log(robot.Debug, "Unsubscribe - plugin '%s' unsubscribing from thread '%s' in channel '%s'", task.name, w.ThreadID, w.Channel)
			delete(subscriptions.m, subscriptionSpec)
			saveSubscriptions()
			return true
		}
	}
	r.Log(robot.Warn, "Unsubscribe - plugin '%s' not subscribed to thread '%s' in channel '%s'", task.name, w.ThreadID, w.Channel)
	return true
}

// expireSubscriptions is called by the brainTicker
func expireSubscriptions(now time.Time) {
	modified := false
	subscriptions.Lock()
	defer subscriptions.Unlock()
	for subscription, subscriber := range subscriptions.m {
		if now.Sub(subscriber.Timestamp) > subscriptionTimeout {
			delete(subscriptions.m, subscription)
			modified = true
			Log(robot.Debug, "Subscribe - expiring subscription for plugin '%s' to thread '%s' in channel '%s'", subscriber.Plugin, subscription.thread, subscription.channel)
		}
	}
	if modified {
		saveSubscriptions()
	}
}
