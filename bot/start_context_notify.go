package bot

import (
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

// notifyPipelineStartContext mirrors a status message back to the pipeline
// origin context captured in GOPHER_START_* when available. It returns
// attempted=false when no origin context exists or when current and origin
// routing targets are effectively the same.
func notifyPipelineStartContext(r Robot, message string) (attempted bool, ret robot.RetVal) {
	startProtocol := strings.TrimSpace(r.GetParameter("GOPHER_START_PROTOCOL"))
	startChannel := strings.TrimSpace(r.GetParameter("GOPHER_START_CHANNEL"))
	startUser := strings.TrimSpace(r.GetParameter("GOPHER_START_USER"))
	if startProtocol == "" && startChannel == "" {
		return false, robot.Ok
	}

	currentChannel := strings.TrimSpace(r.Channel)
	currentProtocol := strings.TrimSpace(protocolFromIncoming(r.Incoming, r.Protocol))
	if startUser == "" {
		startUser = strings.TrimSpace(r.User)
	}

	sameChannel := startChannel != "" && strings.EqualFold(startChannel, currentChannel)
	sameProtocol := startProtocol == "" || currentProtocol == "" || strings.EqualFold(startProtocol, currentProtocol)
	if sameChannel && sameProtocol {
		return false, robot.Ok
	}

	if startChannel == "" {
		if startUser == "" {
			return false, robot.MissingArguments
		}
		if startProtocol != "" {
			return true, r.SendProtocolUserChannelMessage(startProtocol, startUser, "", message)
		}
		return true, r.SendUserMessage(startUser, message)
	}

	if startProtocol != "" {
		if startUser != "" {
			if ret := r.SendProtocolUserChannelMessage(startProtocol, startUser, startChannel, message); ret == robot.Ok {
				return true, robot.Ok
			}
		}
		return true, r.SendProtocolUserChannelMessage(startProtocol, "", startChannel, message)
	}

	if startUser != "" {
		if ret := r.SendUserChannelMessage(startUser, startChannel, message); ret == robot.Ok {
			return true, robot.Ok
		}
	}
	return true, r.SendChannelMessage(startChannel, message)
}
