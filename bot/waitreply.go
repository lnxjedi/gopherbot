package bot

import (
	"fmt"
	"regexp"
	"sync"
	"time"
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
	matched     bool   // true if the regex matched
	interrupted bool   // true if the user issued another command
	rep         string // text of the reply
}

var replies = struct {
	m map[replyMatcher]replyWaiter
	sync.Mutex
}{
	make(map[replyMatcher]replyWaiter),
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

// WaitForReply lets a plugin temporarily register a regex for a reply
// expected to an multi-step command when the robot needs more info. If
// the regular expression matches, it returns the matched text and
// RetVal = Ok.
// If the regular expression doesn't match, it returns an empty string
// with one of the following RetVals:
//	Interrupted - the user issued a new command that ran
//  UseDefaultValue - user supplied a single "=", meaning "use the default value"
//	ReplyNotMatched - didn't successfully match for any reason
//	MatcherNotFound - the regexId didn't correspond to a valid regex
//	TimeoutExpired - the user didn't respond within the timeout window given
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
func (r *Robot) WaitForReply(regexID string, timeout int) (string, RetVal) {
	matcher := replyMatcher{
		user:    r.User,
		channel: r.Channel,
	}
	var rep replyWaiter
	pluginlist.RLock()
	plugin := pluginlist.p[plugIDmap[r.pluginID]]
	pluginlist.RUnlock()
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
	_, exists := replies.m[matcher]
	if exists { // this should never happen, and should eventually be removed
		panic(fmt.Sprintf("stale replyWaiter found for user %s in channel %s", r.User, r.Channel))
	}
	replies.m[matcher] = rep
	replies.Unlock()
	r.Log(Trace, fmt.Sprintf("Adding matcher to replies: %q", matcher))
	// Start a goroutine to delete the reply request if it still exists after a minute.
	// If it's matched in the meantime, it should get deleted at that point.
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		Log(Warn, fmt.Sprintf("Timed out waiting for a reply to regex \"%s\" in channel: %s", regexID, r.Channel))
		replies.Lock()
		// reply timed out, free up this matcher for later reply requests
		delete(replies.m, matcher)
		replies.Unlock()
		// matched=false, timedOut=true
		return "", TimeoutExpired
	case replied := <-rep.replyChannel:
		if replied.interrupted {
			return "", Interrupted
		}
		// Note: the replies.m[] entry is deleted in handleMessage
		if !replied.matched {
			if replied.rep == "=" {
				return "", UseDefaultValue
			} else {
				return "", ReplyNotMatched
			}
		} else {
			return replied.rep, Ok
		}
	}
}

// WaitForReplyRegex is identical to WaitForReply except that the first argument is
// the regex to compile and use. If the regex doesn't compile an error will be
// logged and ("", MatcherNotFound) will be returned.
func (r *Robot) WaitForReplyRegex(regex string, timeout int) (string, RetVal) {
	matcher := replyMatcher{
		user:    r.User,
		channel: r.Channel,
	}
	var rep replyWaiter
	re, err := regexp.Compile(regex)
	if err != nil {
		r.Log(Error, fmt.Sprintf("Unable to compile regex \"%s\" in WaitForReplyRegex", regex))
		return "", MatcherNotFound
	}
	rep.re = re
	rep.replyChannel = make(chan reply)

	replies.Lock()
	// See if there's already a continuation in progress for this Robot:user,channel,
	_, exists := replies.m[matcher]
	if exists { // this should never happen, and should eventually be removed
		panic(fmt.Sprintf("stale replyWaiter found for user %s in channel %s", r.User, r.Channel))
	}
	replies.m[matcher] = rep
	replies.Unlock()
	r.Log(Trace, fmt.Sprintf("Added matcher to replies: %q", matcher))
	// Start a goroutine to delete the reply request if it still exists after a minute.
	// If it's matched in the meantime, it should get deleted at that point.
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		Log(Warn, fmt.Sprintf("Timed out waiting for a reply to custom regex \"%s\" in channel: %s", regex, r.Channel))
		replies.Lock()
		// reply timed out, free up this matcher for later reply requests
		delete(replies.m, matcher)
		replies.Unlock()
		// matched=false, timedOut=true
		return "", TimeoutExpired
	case replied, _ := <-rep.replyChannel:
		if replied.interrupted {
			return "", Interrupted
		}
		// Note: the replies.m[] entry is deleted in handleMessage
		if !replied.matched {
			if replied.rep == "=" {
				return "", UseDefaultValue
			} else {
				return "", ReplyNotMatched
			}
		} else {
			return replied.rep, Ok
		}
	}
}
