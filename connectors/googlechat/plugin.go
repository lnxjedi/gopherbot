package googlechat

import (
	"strings"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

var googleChatRuntime struct {
	sync.RWMutex
	connector *googleChatConnector
}

func init() {
	robot.RegisterPlugin("googlechatutil", robot.PluginHandler{Handler: googleChatPlugin})
}

func setActiveGoogleChatConnector(gc *googleChatConnector) {
	googleChatRuntime.Lock()
	googleChatRuntime.connector = gc
	googleChatRuntime.Unlock()
}

func clearActiveGoogleChatConnector(gc *googleChatConnector) {
	googleChatRuntime.Lock()
	if googleChatRuntime.connector == gc {
		googleChatRuntime.connector = nil
	}
	googleChatRuntime.Unlock()
}

func activeGoogleChatConnector() *googleChatConnector {
	googleChatRuntime.RLock()
	defer googleChatRuntime.RUnlock()
	return googleChatRuntime.connector
}

func googleChatPlugin(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	switch command {
	case "init":
		return
	case "googlevalidaterobot":
		msg := r.GetMessage()
		if msg == nil || msg.Incoming == nil {
			r.Say("This command needs a live incoming message context.")
			return
		}
		if !msg.Incoming.ValidatedUser {
			r.Say("This command requires a validated administrator account.")
			return
		}
		if !msg.Incoming.DirectMessage && !msg.Incoming.HiddenMessage {
			r.Say("This command is only available in direct messages or hidden messages.")
			return
		}
		connector := activeGoogleChatConnector()
		if connector == nil {
			r.Say("The Google Chat connector is not running.")
			return
		}
		if selfID := strings.TrimSpace(connector.CurrentSelfID()); selfID != "" {
			r.Reply("Google Chat SelfID is already known: %s", selfID)
			return
		}
		code, resultCh, err := connector.IssueRobotValidation()
		if err != nil {
			connector.Log(robot.Error, "Issuing Google Chat robot validation code: %v", err)
			r.Say("I couldn't issue a Google Chat robot validation code right now.")
			return
		}
		if resultCh == nil {
			r.Reply("Google Chat SelfID is already known: %s", code)
			return
		}
		r.Reply("Google Chat robot validation code: %s (expires in about 42 seconds). Mention me in Google Chat with this code and I'll confirm there, then send the discovered numeric SelfID back here.", code)
		timer := time.NewTimer(robotValidationTTL)
		defer timer.Stop()
		select {
		case result := <-resultCh:
			if result.AckSpace != "" {
				ret := connector.sendMessage(result.AckSpace, "", result.AckThread, "Code accepted", robot.Variable, nil)
				if ret != robot.Ok {
					connector.Log(robot.Warn, "Google Chat robot validation acknowledgement failed in %q: %s", result.AckSpace, ret)
				}
			}
			r.MessageFormat(robot.BasicMarkdown).Reply("Google Chat robot validation received: bot internal ID is `%s`", result.BotID)
		case <-timer.C:
			connector.CancelRobotValidation(code)
			r.Reply("Google Chat robot validation timed out waiting for code %s.", code)
		}
	}
	return
}
