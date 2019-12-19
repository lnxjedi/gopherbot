// +build !test

package terminal

import (
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

func (tc *termConnector) sendMessage(ch, msg string, f robot.MessageFormat) (ret robot.RetVal) {
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
	tc.reader.Write([]byte(fmt.Sprintf("%s: %s\n", ch, msg)))
	return robot.Ok
}
