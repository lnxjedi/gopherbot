package bot

import (
	"sync"
	"time"
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
// or false when called outside of a thread, or the thread is already
// subscribed. When false, an Error log message is generated.
