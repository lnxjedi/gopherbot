package updatereport

import (
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

func init() {
	robot.RegisterTask("update-report", true, robot.TaskHandler{
		Handler: updateReportTask,
	})
}

func branchStatusSuffix(r robot.Robot, branch string) string {
	if strings.EqualFold(r.GetParameter("GOPHER_CUSTOM_BRANCH_IS_DEFAULT"), "true") {
		return " (default branch)"
	}
	defaultBranch := strings.TrimSpace(r.GetParameter("GOPHER_CUSTOM_DEFAULT_BRANCH"))
	if defaultBranch != "" && defaultBranch != branch {
		return fmt.Sprintf(" (default: '%s')", defaultBranch)
	}
	return ""
}

func updateNonDefaultWarning(r robot.Robot, branch string) string {
	if branch == "" {
		return ""
	}
	if strings.EqualFold(r.GetParameter("GOPHER_CUSTOM_BRANCH_IS_DEFAULT"), "true") {
		return ""
	}
	defaultBranch := strings.TrimSpace(r.GetParameter("GOPHER_CUSTOM_DEFAULT_BRANCH"))
	if defaultBranch == "" || strings.EqualFold(defaultBranch, branch) {
		return ""
	}
	return fmt.Sprintf("Warning: update/reload ran on non-default branch '%s' (default branch is '%s').", branch, defaultBranch)
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

func updateReportTask(r robot.Robot, args ...string) (retval robot.TaskRetVal) {
	operation := strings.ToLower(strings.TrimSpace(r.GetParameter("GIT_OPERATION")))
	if operation == "" {
		operation = "update"
	}

	status := ""
	branch := strings.TrimSpace(r.GetParameter("GOPHER_CUSTOM_BRANCH"))
	switch operation {
	case "switch":
		if branch == "" {
			target := strings.TrimSpace(r.GetParameter("GIT_TARGET_BRANCH"))
			if target == "" {
				status = "Successfully switched robot's git repository branch and pulled latest changes (active branch unavailable)"
				break
			}
			status = fmt.Sprintf("Successfully switched robot's git repository branch to '%s' and pulled latest changes (active branch unavailable)", target)
			break
		}
		status = fmt.Sprintf("Successfully switched robot's git repository to branch '%s' and pulled latest changes%s", branch, branchStatusSuffix(r, branch))
	default:
		if branch == "" {
			status = "Successfully updated robot's git repository (branch unavailable)"
		} else {
			status = fmt.Sprintf("Successfully updated robot's git repository on branch '%s'%s", branch, branchStatusSuffix(r, branch))
		}
	}
	r.Say(status)
	notifyStartContext(r, status)
	if operation == "update" {
		if warning := updateNonDefaultWarning(r, branch); warning != "" {
			r.Say(warning)
			notifyStartContext(r, warning)
		}
	}
	return robot.Normal
}
