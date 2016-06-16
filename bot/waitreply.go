package bot

import (
	"fmt"
	"regexp"
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
	matched bool   // true if the regex matched
	rep     string // text of the reply if matched=true, else ""
}

var replies = make(map[replyMatcher]replyWaiter)

type stockReply struct {
	repTag   string
	repRegex string
}

var stockRepliesRegex string = `^[A-Z]`
var stockRepliesRe *regexp.Regexp

var stockReplies = make(map[string]*regexp.Regexp)

var stockReplyList = []stockReply{
	{"Email", `[\w-\.]+@(?:[\w-]+\.)+[\w-]{2,4}`},
	{"Domain", `(?:[\w-]+\.)+[\w-]{2,4}`},
	{"OTP", `\d{6}`},
	//	{ "IPaddr", `[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}` }
	{"IPaddr", `(?:(?:0|1[0-9]{0,2}|2[0-9]?|2[0-4][0-9]|25[0-5]|[3-9][0-9]?)\.){3}(?:0|1[0-9]{0,2}|2[0-9]?|2[0-4][0-9]|25[0-5]|[3-9][0-9]?)`},
	{"SimpleString", `[\w -.,]+`},
	{"YesNo", `(?i:yes|no|Y|N)`},
}

func init() {
	stockRepliesRe = regexp.MustCompile(stockRepliesRegex)
	for _, sr := range stockReplyList {
		stockReplies[sr.repTag] = regexp.MustCompile(sr.repRegex)
	}
}

/* WaitForReply lets a plugin temporarily register a regex for a reply
expected to an multi-step command when the robot needs more info. It
returns whatever text the user replied with, together with a BotRetVal
which may have the following flags set:
ReplyInProgress - couldn't wait for a reply, already one in progress
ReplyNotMatched - didn't successfully match for any reason
MatcherNotFound - the regexId didn't correspond to a valid regex
TimeoutExpired - the user didn't respond within the timeout window given

Plugin authors can define regex's for regexId's in the plugin's JSON config,
with the restriction that the regexId must start with a lowercase letter.
A pre-definied regex from the following list can also be used:
Email
Domain - an alpha-numeric domain name
OTP - a 6-digit one-time password code
IPAddr
SimpleString - letters, numbers, spaces, dots, dashes, underscores, and commas
YesNo
*/
func (r *Robot) WaitForReply(regexId string, timeout int) (replyText string, ret BotRetVal) {
	matcher := replyMatcher{
		user:    r.User,
		channel: r.Channel,
	}
	// We don't immediately defer an unlock because this function blocks on the
	// reply channel - so we need to Unlock() at every error return point.
	botLock.Lock()
	// See if there's already a continuation in progress for this Robot:user,channel,
	rep, exists := replies[matcher]
	if exists {
		ret = ReplyInProgress
		r.Log(Warn, fmt.Errorf("A reply is already being waited on for user %s in channel %s", r.User, r.Channel))
		botLock.Unlock()
		return "", ret
	}
	b.lock.RLock()
	plugin := plugins[plugIDmap[r.pluginID]]
	plugName := plugin.Name
	if stockRepliesRe.MatchString(regexId) {
		rep.re = stockReplies[regexId]
	} else {
		for _, matcher := range plugin.ReplyMatchers {
			if matcher.Command == regexId {
				rep.re = matcher.re
				break
			}
		}
	}
	b.lock.RUnlock()
	if rep.re == nil {
		r.Log(Error, fmt.Sprintf("Unable to resolve a reply matcher for plugin %s, regexID %s", plugin.Name, regexId))
		botLock.Unlock()
		ret = MatcherNotFound
		return "", ret
	}
	rep.replyChannel = make(chan reply)
	r.Log(Trace, fmt.Sprintf("Adding matcher to replies: %q", matcher))
	replies[matcher] = rep
	// Now that we've added the reply to the map, unlock the bot so we can block
	// on the channel for a reply.
	botLock.Unlock()
	// Start a goroutine to delete the reply request if it still exists after a minute.
	// If it's matched in the meantime, it should get deleted at that point.
	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		Log(Warn, fmt.Sprintf("Plugin \"%s\" timed out waiting for a reply to regex \"%s\"", plugName, regexId))
		botLock.Lock()
		// reply timed out, free up this matcher for later reply requests
		delete(replies, matcher)
		botLock.Unlock()
		// matched=false, timedOut=true
		ret = TimeoutExpired
		return "", ret
	case replied, _ := <-rep.replyChannel:
		// Note: the replies[] entry is deleted in handleMessage
		if !replied.matched {
			ret = ReplyNotMatched
		}
		return replied.rep, ret
	}
}
