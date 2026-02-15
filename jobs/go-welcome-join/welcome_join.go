package main

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

const (
	setupStateFile = ".setup-state"
	statusActive   = "active"
	stageRepoReady = "repository-ready"
)

var joinMessageRe = regexp.MustCompile(`(?i:^@([a-z][a-z0-9_-]{0,31}) has joined #([a-z0-9_-]+)$)`)

type setupState struct {
	Sessions map[string]setupSession `json:"sessions"`
}

type setupSession struct {
	Status string `json:"status"`
	Stage  string `json:"stage"`
}

func JobHandler(r robot.Robot, args ...string) robot.TaskRetVal {
	user, _ := joinedUser(args, r.GetMessage())
	if user == "" {
		return robot.Normal
	}

	alias := strings.TrimSpace(r.GetBotAttribute("alias").String())
	if alias == "" {
		alias = ";"
	}

	if resume, ok := shouldResumeSetup(user); ok && resume {
		r.Say("@%s Welcome back. I found onboarding progress in '%s'.", user, setupStateFile)
		r.Pause(0.8)
		r.Say("@%s To continue setup, type '%snew robot resume' (or '%snew robot cancel' to reset).", user, alias, alias)
		return robot.Normal
	}

	name := strings.TrimSpace(r.GetBotAttribute("name").String())
	if name == "" {
		name = "floyd"
	}
	r.Say("@%s Welcome to the *Gopherbot* ssh connector. You're connected to '%s', the default robot.", user, name)
	r.Pause(1.2)
	r.Say("@%s Start with '%shelp'. To begin configuring a new robot, type '%snew robot'.", user, alias, alias)

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

func shouldResumeSetup(user string) (resume bool, ok bool) {
	body, err := os.ReadFile(setupStateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, true
		}
		return false, false
	}
	if strings.TrimSpace(string(body)) == "" {
		return false, true
	}

	var state setupState
	if err := json.Unmarshal(body, &state); err != nil {
		return false, false
	}
	if state.Sessions == nil {
		return false, true
	}
	session, found := state.Sessions[user]
	if !found {
		return false, true
	}

	status := strings.ToLower(strings.TrimSpace(session.Status))
	stage := strings.ToLower(strings.TrimSpace(session.Stage))
	if status == statusActive {
		return true, true
	}
	if status == "completed" && stage == stageRepoReady {
		return false, true
	}
	if status == "" && stage == "" {
		return false, true
	}
	return true, true
}
