package bot

import (
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

const subscriptionTimeout = 14 * 24 * time.Hour

// Plugins can subscribe to a thread in a channel. This struct is used as the
// key in the subscriptions map.
type subscriptionMatcher struct {
	channel, thread string
}

type subscriber struct {
	plugin    string    // the plugin subscribed
	timestamp time.Time // for expiring after subscriptionTimeout
}

var subscriptions = struct {
	m map[subscriptionMatcher]subscriber
	sync.Mutex
}{
	make(map[subscriptionMatcher]subscriber),
	sync.Mutex{},
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
		if task.name != subscription.plugin {
			r.Log(robot.Error, "Subscribe - plugin '%s' failed subscribing to thread '%s' in channel '%s', subscription already held by plugin '%s'", task.name, w.ThreadID, w.Channel, subscription.plugin)
			return false
		} else {
			r.Log(robot.Debug, "Subscribe - already subscribed; plugin '%s' subscribing to thread '%s' in channel '%s'", task.name, w.ThreadID, w.Channel)
			return true
		}
	}
	subscriptions.m[subscriptionSpec] = subscriber{
		plugin:    task.name,
		timestamp: time.Now(),
	}
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
		if task.name != subscription.plugin {
			r.Log(robot.Error, "Unsubscribe - plugin '%s' failed subscribing to thread '%s' in channel '%s', subscription held by other plugin '%s'", task.name, w.ThreadID, w.Channel, subscription.plugin)
			return false
		} else {
			r.Log(robot.Debug, "Unsubscribe - plugin '%s' unsubscribing from thread '%s' in channel '%s'", task.name, w.ThreadID, w.Channel)
			delete(subscriptions.m, subscriptionSpec)
			return true
		}
	}
	r.Log(robot.Warn, "Unsubscribe - plugin '%s' not subscribed to thread '%s' in channel '%s'", task.name, w.ThreadID, w.Channel)
	return true
}

// expireSubscriptions is called by the brainTicker
func expireSubscriptions(now time.Time) {
	subscriptions.Lock()
	defer subscriptions.Unlock()
	for subscription, subscriber := range subscriptions.m {
		if now.Sub(subscriber.timestamp) > subscriptionTimeout {
			delete(subscriptions.m, subscription)
			Log(robot.Debug, "Subscribe - expiring subscription for plugin '%s' to thread '%s' in channel '%s'", subscriber.plugin, subscription.thread, subscription.channel)
		}
	}
}
