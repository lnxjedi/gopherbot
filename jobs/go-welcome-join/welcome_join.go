package main

import (
	"regexp"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
	"gopherbot.internal/lib/newrobotflow"
)

var joinMessageRe = regexp.MustCompile(`(?i:^@([a-z][a-z0-9_-]{0,31}) has joined #([a-z0-9_-]+)$)`)

func JobHandler(r robot.Robot, args ...string) robot.TaskRetVal {
	user, channel := joinedUser(args, r.GetMessage())
	if user == "" || channel == "" {
		return robot.Normal
	}
	if newrobotflow.HasAnySetupState() {
		return robot.Normal
	}
	if user != "alice" {
		return robot.Normal
	}

	alias := strings.TrimSpace(r.GetBotAttribute("alias").String())
	if alias == "" {
		alias = ";"
	}
	name := strings.TrimSpace(r.GetBotAttribute("name").String())
	if name == "" {
		name = "floyd"
	}

	send := r.MessageFormat(robot.BasicMarkdown)
	send.Pause(newrobotflow.SetupInitialGreetingPauseSeconds)
	send.SendChannelMessage(channel, "Welcome to the *Gopherbot* ssh connector, @%s! Since no configuration was detected, you're connected to '%s', the default robot.", user, name)
	send.Pause(newrobotflow.SetupParagraphReadPauseSeconds)
	send.SendChannelMessage(channel, "If you've started the robot by mistake, just hit ctrl-D to exit and try 'gopherbot --help'; otherwise feel free to play around with the default robot - you can start by typing 'help'. If you'd like to start configuring a new robot, type: '%snew robot'.", alias)
	return robot.Normal
}

func joinedUser(args []string, m *robot.Message) (string, string) {
	if len(args) >= 2 {
		return normalizeName(args[0]), normalizeName(args[1])
	}
	if m == nil || m.Incoming == nil {
		return "", ""
	}
	matches := joinMessageRe.FindStringSubmatch(strings.TrimSpace(m.Incoming.MessageText))
	if len(matches) < 3 {
		return "", ""
	}
	return normalizeName(matches[1]), normalizeName(matches[2])
}

func normalizeName(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}
