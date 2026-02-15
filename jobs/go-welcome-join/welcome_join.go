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

	statusActive    = "active"
	statusCompleted = "completed"

	stageScaffolded      = "scaffolded"
	stageAwaitingRepoURL = "awaiting-repository-url"
	stageRepoReady       = "repository-ready"
)

var joinMessageRe = regexp.MustCompile(`(?i:^@([a-z][a-z0-9_-]{0,31}) has joined #([a-z0-9_-]+)$)`)

type welcomeAction int

const (
	actionNone welcomeAction = iota
	actionInitialWelcome
	actionResumeGeneral
	actionResumeRepository
)

type setupState struct {
	Sessions map[string]setupSession `json:"sessions"`
}

type setupSession struct {
	Status        string `json:"status"`
	Stage         string `json:"stage"`
	CanonicalUser string `json:"canonicalUser"`
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
	name := strings.TrimSpace(r.GetBotAttribute("name").String())
	if name == "" {
		name = "floyd"
	}

	action := resolveAction(user)
	switch action {
	case actionResumeRepository:
		r.Pause(0.5)
		r.Say("Welcome back, @%s", user)
		r.Pause(0.5)
		r.Say("Type '%snew robot repo' to continue where we left off", alias)
		return robot.Normal
	case actionResumeGeneral:
		r.Pause(0.5)
		r.Say("@%s Welcome back - I found onboarding progress in '%s'", user, setupStateFile)
		r.Pause(0.5)
		r.Say("@%s To continue setup, type '%snew robot resume' (or '%snew robot cancel' to reset).", user, alias, alias)
		return robot.Normal
	case actionInitialWelcome:
		r.Pause(0.5)
		r.Say("Welcome to the *Gopherbot* ssh connector, @%s! Since no configuration was detected, you're connected to '%s', the default robot.", user, name)
		r.Pause(0.5)
		r.Say("If you've started the robot by mistake, just hit ctrl-D to exit and try 'gopherbot --help'; otherwise feel free to play around with the default robot - you can start by typing 'help'. If you'd like to start configuring a new robot, type: '%snew robot'.", alias)
		return robot.Normal
	}

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

func resolveAction(user string) welcomeAction {
	body, err := os.ReadFile(setupStateFile)
	if err != nil {
		if os.IsNotExist(err) {
			if user == "alice" {
				return actionInitialWelcome
			}
			return actionNone
		}
		return actionNone
	}
	if strings.TrimSpace(string(body)) == "" {
		if user == "alice" {
			return actionInitialWelcome
		}
		return actionNone
	}

	var state setupState
	if err := json.Unmarshal(body, &state); err != nil {
		return actionNone
	}
	if state.Sessions == nil {
		if user == "alice" {
			return actionInitialWelcome
		}
		return actionNone
	}

	session, found := state.Sessions[user]
	if !found {
		for _, candidate := range state.Sessions {
			if normalizeName(candidate.CanonicalUser) == user {
				session = candidate
				found = true
				break
			}
		}
	}
	if !found {
		if user == "alice" {
			return actionInitialWelcome
		}
		return actionNone
	}

	status := strings.ToLower(strings.TrimSpace(session.Status))
	stage := strings.ToLower(strings.TrimSpace(session.Stage))
	if status == statusCompleted && stage == stageRepoReady {
		return actionNone
	}
	if status == statusActive && (stage == stageAwaitingRepoURL || stage == stageScaffolded) {
		return actionResumeRepository
	}
	if status == statusActive {
		return actionResumeGeneral
	}
	if status == "" && stage == "" {
		if user == "alice" {
			return actionInitialWelcome
		}
		return actionNone
	}
	return actionResumeGeneral
}
