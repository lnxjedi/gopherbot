package failreport

import (
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

func init() {
	robot.RegisterTask("fail-report", true, robot.TaskHandler{
		Handler: failReportTask,
	})
}

func notifyStartContext(r robot.Robot, message string) {
	startProtocol := strings.TrimSpace(r.GetParameter("GOPHER_START_PROTOCOL"))
	startChannel := strings.TrimSpace(r.GetParameter("GOPHER_START_CHANNEL"))
	startUser := strings.TrimSpace(r.GetParameter("GOPHER_START_USER"))
	if startProtocol == "" && startChannel == "" {
		return
	}

	msg := r.GetMessage()
	currentChannel := ""
	currentProtocol := ""
	if msg != nil {
		currentChannel = strings.TrimSpace(msg.Channel)
		if msg.Incoming != nil {
			currentProtocol = strings.TrimSpace(msg.Incoming.Protocol)
		}
		if startUser == "" {
			startUser = strings.TrimSpace(msg.User)
		}
	}

	sameChannel := startChannel != "" && strings.EqualFold(startChannel, currentChannel)
	sameProtocol := startProtocol == "" || currentProtocol == "" || strings.EqualFold(startProtocol, currentProtocol)
	if sameChannel && sameProtocol {
		return
	}

	if startChannel == "" {
		if startUser == "" {
			return
		}
		if startProtocol != "" {
			_ = r.SendProtocolUserChannelMessage(startProtocol, startUser, "", message)
			return
		}
		_ = r.SendUserMessage(startUser, message)
		return
	}

	if startProtocol != "" {
		if startUser != "" {
			if ret := r.SendProtocolUserChannelMessage(startProtocol, startUser, startChannel, message); ret == robot.Ok {
				return
			}
		}
		_ = r.SendProtocolUserChannelMessage(startProtocol, "", startChannel, message)
		return
	}

	if startUser != "" {
		if ret := r.SendUserChannelMessage(startUser, startChannel, message); ret == robot.Ok {
			return
		}
	}
	_ = r.SendChannelMessage(startChannel, message)
}

func failReportTask(r robot.Robot, args ...string) (retval robot.TaskRetVal) {
	pipename := strings.TrimSpace(r.GetParameter("GOPHER_PIPE_NAME"))
	op := strings.ToLower(strings.TrimSpace(r.GetParameter("GIT_OPERATION")))
	targetBranch := strings.TrimSpace(r.GetParameter("GIT_TARGET_BRANCH"))
	errMsg := strings.TrimSpace(r.GetParameter("GIT_ERROR"))
	status := ""

	if errMsg == "" {
		finalTask := strings.TrimSpace(r.GetParameter("GOPHER_FINAL_TASK"))
		failStr := strings.TrimSpace(r.GetParameter("GOPHER_FAIL_STRING"))
		if finalTask != "" && failStr != "" {
			errMsg = fmt.Sprintf("task '%s' failed (%s)", finalTask, failStr)
		} else if finalTask != "" {
			errMsg = fmt.Sprintf("task '%s' failed", finalTask)
		} else if failStr != "" {
			errMsg = failStr
		} else {
			errMsg = "unknown error"
		}
	}

	switch op {
	case "update":
		status = fmt.Sprintf("Git update failed: %s", errMsg)
	case "switch":
		if targetBranch == "" {
			status = fmt.Sprintf("Git branch switch failed: %s", errMsg)
			break
		}
		status = fmt.Sprintf("Git branch switch to '%s' failed: %s", targetBranch, errMsg)
	default:
		if pipename == "" {
			pipename = "(unknown)"
		}
		status = fmt.Sprintf("Pipeline '%s' failed: %s", pipename, errMsg)
	}
	r.Say(status)
	notifyStartContext(r, status)
	return robot.Normal
}
