package main

import (
	"regexp"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/lnxjedi/gopherbot/v2/lib/newrobotflow"
)

var joinMessageRe = regexp.MustCompile(`(?i:^@([a-z][a-z0-9_-]{0,31}) has joined #([a-z0-9_-]+)$)`)

func JobHandler(r robot.Robot, args ...string) robot.TaskRetVal {
	user, channel := joinedUser(args, r.GetMessage())
	if user == "" || channel == "" {
		return robot.Normal
	}
	newrobotflow.HandleResumeJoin(r, user, channel, "ssh")
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
