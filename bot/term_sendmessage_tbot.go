//go:build test
// +build test

package bot

import (
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

func (tc *termConnector) sendMessage(user, ch, thr, msg string, f robot.MessageFormat, incomingMsg *robot.ConnectorMessage) (ret robot.RetVal) {
	tc.checkSendSelf(ch, thr, msg, f)
	found := false
	tc.RLock()
	if strings.HasPrefix(ch, "(dm:") {
		found = true
	} else {
		for _, channel := range tc.channels {
			if channel == ch {
				found = true
				break
			}
		}
	}
	tc.RUnlock()
	if !found {
		tc.Log(robot.Error, "Channel not found:", ch)
		return robot.ChannelNotFound
	}
	threadID := ""
	if len(thr) > 0 {
		threadID = fmt.Sprintf("(%s)", thr)
		tc.Lock()
		tc.lastThread = thr
		tc.Unlock()
	}
	user, _ = tc.ExtractID(user)
	if incomingMsg.HiddenMessage &&
		(user == "" || user == incomingMsg.UserID) {
		msg = "(" + msg + ")"
	}
	switch f {
	case robot.Fixed:
		msg = strings.ToUpper(msg)
	case robot.Variable:
		msg = strings.ToLower(msg)
	}
	if startMode == "test-dev" {
		// Log(robot.Info, "TEST/OUTGOING: %s", tbot.FormatOutgoing(user, ch, msg, thr))
	}
	tc.reader.Write([]byte(fmt.Sprintf("%s%s: %s\n", ch, threadID, msg)))
	return robot.Ok
}
