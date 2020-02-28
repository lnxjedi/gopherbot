// +build !test

package bot

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
	output := fmt.Sprintf("%s: %s\n", ch, msg)
	if f != robot.Fixed {
		output = Wrap(output, tc.width)
		tc.reader.Write([]byte(output)[0 : len(output)-1])
	} else {
		tc.reader.Write([]byte(output))
	}
	return robot.Ok
}
