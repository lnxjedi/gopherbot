package bot

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

/* Technical notes on the waiter implementation
 - or -
Why retryPrompt is sent to all waiters, instead of just the head of a queue

After spending a good deal of one morning re-writing waiters as a proper queue,
I realized the problem with that implementation. Each script plugin is
posting JSON to a port and waiting for a reply, and most script libraries will
timeout waiting after no more than a minute (which is why the default replyTimeout
is 45 seconds). If we queue up all waiters, and the user doesn't reply to the first
waiter (or second), then the second waiter in the queue might not get a reply
for a 90 seconds - by which time the script would crash. To be certain that
every waiting plugin gets some kind of return value within 60 seconds, we just
send retryPrompt to all waiters, and let them race to be first.

Long-running interactive prompts on local connectors (ssh/terminal) can opt
into a longer timeout, but the queueing model remains unchanged.

Realize this isn't as bad as it might seem; the list of waiters is per
user/channel combination, so this kind of thing only happens if a single user
is absolutely going crazy with firing off interactive commands in a single
channel.

The moral of the story: don't bother implementing a queue for reply waiters,
and think hard before doing things any differently.
*/

const replyTimeout = 45 * time.Second
const interactivePromptTimeout = 42 * time.Minute

var promptShutdown = struct {
	sync.Mutex
	ch     chan struct{}
	closed bool
}{
	ch:     make(chan struct{}),
	closed: false,
}

type replyDisposition int

const (
	replied          replyDisposition = iota
	replyInterrupted                  // user started another command or canceled
	retryPrompt                       // another prompt was in progress
)

// a replyWaiter is used when a plugin is waiting for a reply
type replyWaiter struct {
	re           *regexp.Regexp // The regular expression the reply needs to match
	replyChannel chan reply     // The channel to send the reply to when it is received
}

// a reply matcher is used as the key in the replies map
type replyMatcher struct {
	protocol, user, channel, thread string // Only one reply at a time can be requested for a given protocol/user/channel/thread combination
}

// a reply is sent over the replyWaiter channel when a user replies
type reply struct {
	matched     bool             // true if the regex matched
	disposition replyDisposition // replied, interrupted, retry
	rep         string           // text of the reply
}

var replies = struct {
	m map[replyMatcher][]replyWaiter
	sync.Mutex
}{
	make(map[replyMatcher][]replyWaiter),
	sync.Mutex{},
}

type stockReply struct {
	repTag   string
	repRegex string
}

var stockRepliesRegex = `^[A-Z]`
var stockRepliesRe *regexp.Regexp

var stockReplies = make(map[string]*regexp.Regexp)

var stockReplyList = []stockReply{
	{"Email", `[\w-\.]+@(?:[\w-]+\.)+[\w-]{2,4}`},
	{"Domain", `(?:[\w-]+\.)+[\w-]{2,4}`},
	{"OTP", `\d{6}`},
	//	{ "IPaddr", `[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}` }
	{"IPaddr", `(?:(?:0|1[0-9]{0,2}|2[0-9]?|2[0-4][0-9]|25[0-5]|[3-9][0-9]?)\.){3}(?:0|1[0-9]{0,2}|2[0-9]?|2[0-4][0-9]|25[0-5]|[3-9][0-9]?)`},
	{"SimpleString", `[-\w .,_'"?!]+`},
	{"YesNo", `(?i:yes|no|Y|N)`},
}

func init() {
	stockRepliesRe = regexp.MustCompile(stockRepliesRegex)
	for _, sr := range stockReplyList {
		stockReplies[sr.repTag] = regexp.MustCompile(`^\s*` + sr.repRegex + `\s*$`)
	}
}

func resetPromptShutdownSignal() {
	promptShutdown.Lock()
	promptShutdown.ch = make(chan struct{})
	promptShutdown.closed = false
	promptShutdown.Unlock()
}

func triggerPromptShutdownSignal() {
	promptShutdown.Lock()
	if !promptShutdown.closed && promptShutdown.ch != nil {
		close(promptShutdown.ch)
		promptShutdown.closed = true
	}
	promptShutdown.Unlock()
}

func getPromptShutdownSignal() <-chan struct{} {
	promptShutdown.Lock()
	defer promptShutdown.Unlock()
	return promptShutdown.ch
}

func isInteractivePromptProtocol(protocol string) bool {
	switch normalizeProtocolName(protocol) {
	case "ssh", "terminal":
		return true
	}
	return false
}

func isInterpreterTask(task *Task) bool {
	if task == nil {
		return false
	}
	if task.taskType == taskGo {
		return true
	}
	if task.taskType != taskExternal {
		return false
	}
	path := strings.ToLower(strings.TrimSpace(task.Path))
	return strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".lua") || strings.HasSuffix(path, ".js")
}

func promptTimeoutForContext(r Robot, task *Task) time.Duration {
	protocol := protocolFromIncoming(r.Incoming, r.Protocol)
	if !isInteractivePromptProtocol(protocol) {
		return replyTimeout
	}
	if !isInterpreterTask(task) {
		return replyTimeout
	}
	return interactivePromptTimeout
}

// see robot/robot.go
func (r Robot) PromptForReply(regexID string, prompt string, v ...interface{}) (string, robot.RetVal) {
	var rep string
	var ret robot.RetVal
	if len(v) > 0 {
		prompt = fmt.Sprintf(prompt, v...)
	}
	for i := 0; i < 3; i++ {
		var thread string
		if r.Incoming.ThreadedMessage {
			thread = r.Incoming.ThreadID
		}
		rep, ret = r.promptInternal(regexID, r.User, r.Channel, thread, prompt)
		if ret == robot.RetryPrompt {
			continue
		}
		return rep, ret
	}
	if ret == robot.RetryPrompt {
		return rep, robot.Interrupted
	}
	return rep, ret
}

// see robot/robot.go
func (r Robot) PromptThreadForReply(regexID string, prompt string, v ...interface{}) (string, robot.RetVal) {
	var rep string
	var ret robot.RetVal
	if len(v) > 0 {
		prompt = fmt.Sprintf(prompt, v...)
	}
	for i := 0; i < 3; i++ {
		rep, ret = r.promptInternal(regexID, r.User, r.Channel, r.Incoming.ThreadID, prompt)
		if ret == robot.RetryPrompt {
			continue
		}
		return rep, ret
	}
	if ret == robot.RetryPrompt {
		return rep, robot.Interrupted
	}
	return rep, ret
}

// see robot/robot.go
func (r Robot) PromptUserForReply(regexID string, user string, prompt string, v ...interface{}) (string, robot.RetVal) {
	var rep string
	var ret robot.RetVal
	if len(v) > 0 {
		prompt = fmt.Sprintf(prompt, v...)
	}
	for i := 0; i < 3; i++ {
		rep, ret = r.promptInternal(regexID, user, "", "", prompt)
		if ret == robot.RetryPrompt {
			continue
		}
		return rep, ret
	}
	if ret == robot.RetryPrompt {
		return rep, robot.Interrupted
	}
	return rep, ret
}

// see robot/robot.go
func (r Robot) PromptUserChannelForReply(regexID string, user string, channel string, prompt string, v ...interface{}) (string, robot.RetVal) {
	var rep string
	var ret robot.RetVal
	if len(v) > 0 {
		prompt = fmt.Sprintf(prompt, v...)
	}
	for i := 0; i < 3; i++ {
		rep, ret = r.promptInternal(regexID, user, channel, "", prompt)
		if ret == robot.RetryPrompt {
			continue
		}
		return rep, ret
	}
	if ret == robot.RetryPrompt {
		return rep, robot.Interrupted
	}
	return rep, ret
}

// PromptUserChannelThreadForReply prompts a specific user in a given channel
// and thread.
func (r Robot) PromptUserChannelThreadForReply(regexID string, user, channel, thread string, prompt string, v ...interface{}) (string, robot.RetVal) {
	var rep string
	var ret robot.RetVal
	if len(v) > 0 {
		prompt = fmt.Sprintf(prompt, v...)
	}
	for i := 0; i < 3; i++ {
		rep, ret = r.promptInternal(regexID, user, channel, thread, prompt)
		if ret == robot.RetryPrompt {
			continue
		}
		return rep, ret
	}
	if ret == robot.RetryPrompt {
		return rep, robot.Interrupted
	}
	return rep, ret
}

// promptInternal can return 'RetryPrompt'
func (r Robot) promptInternal(regexID, user, channel, thread, prompt string) (string, robot.RetVal) {
	protocol := protocolFromIncoming(r.Incoming, r.Protocol)
	matcher := replyMatcher{
		protocol: protocol,
		user:     user,
		channel:  channel,
		thread:   thread,
	}
	var rep replyWaiter
	task, _, job := getTask(r.currentTask)
	isJob := job != nil
	waitTimeout := promptTimeoutForContext(r, task)
	shutdownSignal := getPromptShutdownSignal()
	select {
	case <-shutdownSignal:
		return "", robot.Interrupted
	default:
	}
	if stockRepliesRe.MatchString(regexID) {
		rep.re = stockReplies[regexID]
	} else {
		var rm []InputMatcher
		if isJob {
			rm = job.Arguments
		} else {
			rm = task.ReplyMatchers
		}
		for _, matcher := range rm {
			if matcher.Label == regexID {
				rep.re = matcher.re
				break
			} else if matcher.Command == regexID {
				rep.re = matcher.re
				break
			}
		}
	}
	if rep.re == nil {
		Log(robot.Error, "Unable to resolve a reply matcher for plugin %s, regexID %s (protocol: %s)", task.name, regexID, protocol)
		return "", robot.MatcherNotFound
	}
	rep.replyChannel = make(chan reply, 1)

	replies.Lock()
	// See if there's already a continuation in progress for this Robot:user,channel,
	// and if so append to the list of waiters.
	waiters, exists := replies.m[matcher]
	if exists {
		Log(robot.Debug, "Delaying prompt \"%s\" and appending to the list of waiters for matcher: %q (protocol: %s)", prompt, matcher, protocol)
		waiters = append(waiters, rep)
		replies.m[matcher] = waiters
		replies.Unlock()
	} else {
		Log(robot.Debug, "Prompting for \"%s\" and creating reply waiters list and prompting for matcher: %q (protocol: %s)", prompt, matcher, protocol)
		puser := (&r).tryResolveUser(user)
		var ret robot.RetVal
		if channel == "" {
			ret = interfaces.SendProtocolUserMessage(puser, prompt, r.Format, r.Incoming)
		} else {
			ret = interfaces.SendProtocolUserChannelThreadMessage(puser, user, channel, thread, prompt, r.Format, r.Incoming)
		}
		if ret != robot.Ok {
			replies.Unlock()
			return "", ret
		}
		waiters = make([]replyWaiter, 1, 2)
		waiters[0] = rep
		replies.m[matcher] = waiters
		replies.Unlock()
	}
	var replied reply
	select {
	case <-time.After(waitTimeout):
		Log(robot.Warn, "Timed out waiting for a reply to regex \"%s\" on protocol '%s' in channel '%s' (timeout: %s)", regexID, protocol, channel, waitTimeout)
		replies.Lock()
		waitlist, found := replies.m[matcher]
		if found {
			// reply timed out, free up this matcher for later reply requests
			delete(replies.m, matcher)
			replies.Unlock()
			Log(robot.Debug, "Timeout expired waiting for reply to: %s (protocol: %s)", prompt, protocol)
			// let other waiters know to retry
			for i, rep := range waitlist {
				if i != 0 {
					Log(robot.Debug, "Sending retryPrompt to waiters on primary waiter timeout (protocol: %s)", protocol)
					rep.replyChannel <- reply{false, retryPrompt, ""}
				}
			}
			// matched=false, timedOut=true
			return "", robot.TimeoutExpired
		}
		// race: we got a reply at the timeout deadline, and lost the race
		// to delete the entry, so we read the reply as if the timeout hadn't
		// expired.
		replies.Unlock()
		replied = <-rep.replyChannel
	case <-shutdownSignal:
		replies.Lock()
		if _, found := replies.m[matcher]; found {
			delete(replies.m, matcher)
		}
		replies.Unlock()
		return "", robot.Interrupted
	case replied = <-rep.replyChannel:
	}
	if replied.disposition == replyInterrupted {
		return "", robot.Interrupted
	}
	if replied.disposition == retryPrompt {
		return "", robot.RetryPrompt
	}
	// Note: the replies.m[] entry is deleted in handleMessage
	if !replied.matched {
		if replied.rep == "=" {
			return "", robot.UseDefaultValue
		}
		if replied.rep == "-" {
			return "", robot.Interrupted
		}
		return "", robot.ReplyNotMatched
	}
	return replied.rep, robot.Ok
}
