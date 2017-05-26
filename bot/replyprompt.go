package bot

import (
	"fmt"
	"regexp"
	"sync"
	"time"
)

const replyTimeout = 45 * time.Second

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

// a reply matcher is used as the key in the replys map
type replyMatcher struct {
	user, channel string // Only one reply at a time can be requested for a given user/channel combination
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

// PromptForReply lets a plugin send a prompt string and temporarily register a
// regex for a reply expected to an multi-step command when the robot needs
// more info. If the regular expression matches, it returns the matched text and
// RetVal = Ok.
// If the regular expression doesn't match, it returns an empty string
// with one of the following RetVals:
//	Interrupted - the user issued a new command that ran or canceled with '-'
//  UseDefaultValue - user supplied a single "=", meaning "use the default value"
//	ReplyNotMatched - didn't successfully match for any reason
//	MatcherNotFound - the regexId didn't correspond to a valid regex
//	TimeoutExpired - the user didn't respond within the timeout window
//
// Plugin authors can define regex's for regexId's in the plugin's JSON config,
// with the restriction that the regexId must start with a lowercase letter.
// A pre-definied regex from the following list can also be used:
// 	Email
//	Domain - an alpha-numeric domain name
//	OTP - a 6-digit one-time password code
//	IPAddr
//	SimpleString - Characters commonly found in most english sentences, doesn't
//    include special characters like @, {, etc.
//	YesNo
func (r *Robot) PromptForReply(regexID string, prompt string) (string, RetVal) {
	var rep string
	var ret RetVal
	for i := 0; i < 3; i++ {
		rep, ret = r.promptInternal(false, regexID, prompt)
		if ret == RetryPrompt {
			continue
		}
		return rep, ret
	}
	if ret == RetryPrompt {
		return rep, Interrupted
	}
	return rep, ret
}

// PromptUserForReply is identical to PromptForReply, but directs the message
// to the user.
func (r *Robot) PromptUserForReply(regexID string, prompt string) (string, RetVal) {
	var rep string
	var ret RetVal
	for i := 0; i < 3; i++ {
		rep, ret = r.promptInternal(true, regexID, prompt)
		if ret == RetryPrompt {
			continue
		}
		return rep, ret
	}
	if ret == RetryPrompt {
		return rep, Interrupted
	}
	return rep, ret
}

// promptInternal can return 'RetryPrompt'
func (r *Robot) promptInternal(promptUser bool, regexID string, prompt string) (string, RetVal) {
	matcher := replyMatcher{
		user:    r.User,
		channel: r.Channel,
	}
	var rep replyWaiter
	plugin := currentPlugins.getPluginByID(r.pluginID)
	if stockRepliesRe.MatchString(regexID) {
		rep.re = stockReplies[regexID]
	} else {
		for _, matcher := range plugin.ReplyMatchers {
			if matcher.Label == regexID {
				rep.re = matcher.re
				break
			} else if matcher.Command == regexID {
				rep.re = matcher.re
			}
		}
	}
	if rep.re == nil {
		r.Log(Error, fmt.Sprintf("Unable to resolve a reply matcher for plugin %s, regexID %s", plugin.name, regexID))
		return "", MatcherNotFound
	}
	rep.replyChannel = make(chan reply)

	replies.Lock()
	// See if there's already a continuation in progress for this Robot:user,channel,
	// and if so append to the list of waiters.
	waiters, exists := replies.m[matcher]
	if exists {
		r.Log(Debug, fmt.Sprintf("Delaying prompt \"%s\" and appending to the list of waiters for matcher: %q", prompt, matcher))
		waiters = append(waiters, rep)
		replies.m[matcher] = waiters
		replies.Unlock()
	} else {
		r.Log(Debug, fmt.Sprintf("Prompting for \"%s \" and creating reply waiters list and prompting for matcher: %q", prompt, matcher))
		waiters = make([]replyWaiter, 1, 2)
		waiters[0] = rep
		replies.m[matcher] = waiters
		replies.Unlock()
		if promptUser {
			r.Reply(prompt)
		} else {
			r.Say(prompt)
		}
	}
	var replied reply
	select {
	case <-time.After(replyTimeout):
		Log(Warn, fmt.Sprintf("Timed out waiting for a reply to regex \"%s\" in channel: %s", regexID, r.Channel))
		replies.Lock()
		waitlist, found := replies.m[matcher]
		if found {
			// reply timed out, free up this matcher for later reply requests
			delete(replies.m, matcher)
			replies.Unlock()
			Log(Debug, fmt.Sprintf("Timeout expired waiting for reply to: %s", prompt))
			// let other waiters know to retry
			for i, rep := range waitlist {
				if i != 0 {
					Log(Debug, "Sending retryPrompt to waiters on primary waiter timeout")
					rep.replyChannel <- reply{false, retryPrompt, ""}
				}
			}
			// matched=false, timedOut=true
			return "", TimeoutExpired
		} else {
			// race: we got a reply at the timeout deadline, and lost the race
			// to delete the entry, so we read the reply as if the timeout hadn't
			// expired.
			replies.Unlock()
			replied = <-rep.replyChannel
		}
	case replied = <-rep.replyChannel:
	}
	if replied.disposition == replyInterrupted {
		return "", Interrupted
	}
	if replied.disposition == retryPrompt {
		return "", RetryPrompt
	}
	// Note: the replies.m[] entry is deleted in handleMessage
	if !replied.matched {
		if replied.rep == "=" {
			return "", UseDefaultValue
		}
		if replied.rep == "-" {
			return "", Interrupted
		}
		return "", ReplyNotMatched
	}
	return replied.rep, Ok
}
