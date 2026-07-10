package bot

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lnxjedi/gopherbot/robot"
)

const userApprovalChoiceMatcher = "approvalChoice"

type userApprovalConfig struct {
	DefaultStrict     *bool                                  `json:"DefaultStrict"`
	FallbackApprovers []string                               `json:"FallbackApprovers"`
	PluginApprovers   map[string]userApprovalPluginApprovers `json:"PluginApprovers"`
}

type userApprovalPluginApprovers struct {
	Approvers    []string `json:"Approvers"`
	Strict       *bool    `json:"Strict"`
	hasApprovers bool
}

func (p *userApprovalPluginApprovers) UnmarshalJSON(data []byte) error {
	var approvers []string
	if err := json.Unmarshal(data, &approvers); err == nil {
		p.Approvers = approvers
		p.hasApprovers = true
		p.Strict = nil
		return nil
	}

	var cfg struct {
		Approvers []string `json:"Approvers"`
		Strict    *bool    `json:"Strict"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	p.Approvers = cfg.Approvers
	p.Strict = cfg.Strict
	p.hasApprovers = cfg.Approvers != nil
	return nil
}

type userApprovalRuntime interface {
	GetTaskConfig(interface{}) robot.RetVal
	GetMessage() *robot.Message
	PromptForReply(regexID string, prompt string, v ...interface{}) (string, robot.RetVal)
	PromptUserForReply(regexID string, user string, prompt string, v ...interface{}) (string, robot.RetVal)
	Say(msg string, v ...interface{}) robot.RetVal
	Log(l robot.LogLevel, m string, v ...interface{}) bool
	pipelineNameForApproval() string
	commandNameForApproval() string
}

func init() {
	robot.RegisterPlugin("builtin-userapproval", robot.PluginHandler{Handler: userApprovalElevate})
}

func (r Robot) pipelineNameForApproval() string {
	return r.pipeName
}

func (r Robot) commandNameForApproval() string {
	return r.plugCommand
}

func userApprovalElevate(gr robot.Robot, command string, args ...string) robot.TaskRetVal {
	if command != "_elevate" {
		return robot.Normal
	}
	r, ok := gr.(userApprovalRuntime)
	if !ok {
		return robot.MechanismFail
	}
	return runUserApproval(r)
}

func runUserApproval(r userApprovalRuntime) robot.TaskRetVal {
	var cfg userApprovalConfig
	if ret := r.GetTaskConfig(&cfg); ret != robot.Ok {
		r.Log(robot.Error, "builtin-userapproval failed to load configuration: %s", ret)
		return robot.MechanismFail
	}

	requester := canonicalApprovalUser(r.GetMessage().User)
	pipeName := strings.TrimSpace(r.pipelineNameForApproval())
	actionName := userApprovalActionName(pipeName, r.commandNameForApproval())
	approval := effectiveApprovalPolicy(cfg, pipeName, requester)
	if !approval.strict && approval.requesterListed {
		r.Log(robot.Audit, "builtin-userapproval auto-approved pipeline '%s' for requester '%s' because requester is a configured approver and strict mode is disabled", pipeName, requester)
		r.Say("Approval granted by %s", requester)
		return robot.Success
	}

	approvers := approval.approvers
	if len(approvers) == 0 {
		r.Log(robot.Error, "builtin-userapproval has no eligible approvers for pipeline '%s', requester '%s'", pipeName, requester)
		r.Say("No eligible approvers are configured for this action")
		return robot.Fail
	}
	if len(approvers) > 26 {
		r.Log(robot.Error, "builtin-userapproval has %d approvers for pipeline '%s'; maximum is 26", len(approvers), pipeName)
		r.Say("Too many approvers are configured for this action")
		return robot.Fail
	}

	choicePrompt := userApprovalChoicePrompt(actionName, approvers)
	choice, ret := r.PromptForReply(userApprovalChoiceMatcher, choicePrompt)
	if ret != robot.Ok {
		r.Log(robot.Warn, "builtin-userapproval requester '%s' did not select an approver for pipeline '%s': %s", requester, pipeName, ret)
		return robot.Fail
	}
	approver, ok := userApprovalApproverForChoice(choice, approvers)
	if !ok {
		r.Log(robot.Warn, "builtin-userapproval requester '%s' selected invalid approver choice '%s' for pipeline '%s'", requester, choice, pipeName)
		r.Say("Invalid approver selection")
		return robot.Fail
	}
	r.Say("Ok, hold on while I ask '%s' for approval...", approver)

	answer, ret := r.PromptUserForReply("YesNo", approver,
		"%s is requesting approval to run command %s - approve? (y/n)",
		requester, actionName)
	if ret != robot.Ok {
		r.Log(robot.Warn, "builtin-userapproval approver '%s' did not respond for requester '%s', pipeline '%s': %s", approver, requester, pipeName, ret)
		r.Say("Approval request to %s did not complete", approver)
		return robot.Fail
	}
	if userApprovalYes(answer) {
		r.Log(robot.Audit, "builtin-userapproval approved pipeline '%s' for requester '%s' by approver '%s'", pipeName, requester, approver)
		r.Say("Approval granted by %s", approver)
		return robot.Success
	}

	r.Log(robot.Audit, "builtin-userapproval denied pipeline '%s' for requester '%s' by approver '%s'", pipeName, requester, approver)
	r.Say("Approval denied by %s", approver)
	return robot.Fail
}

type userApprovalPolicy struct {
	approvers       []string
	strict          bool
	requesterListed bool
}

func effectiveApprovalApprovers(cfg userApprovalConfig, pipeName, requester string) []string {
	return effectiveApprovalPolicy(cfg, pipeName, requester).approvers
}

func effectiveApprovalPolicy(cfg userApprovalConfig, pipeName, requester string) userApprovalPolicy {
	source, strict := effectiveApprovalSource(cfg, pipeName)
	seen := make(map[string]struct{}, len(source))
	approvers := make([]string, 0, len(source))
	requester = canonicalApprovalUser(requester)
	requesterListed := false
	for _, approver := range source {
		approver = canonicalApprovalUser(approver)
		if approver == "" {
			continue
		}
		if approver == requester {
			requesterListed = true
			continue
		}
		if _, exists := seen[approver]; exists {
			continue
		}
		seen[approver] = struct{}{}
		approvers = append(approvers, approver)
	}
	return userApprovalPolicy{
		approvers:       approvers,
		strict:          strict,
		requesterListed: requesterListed,
	}
}

func effectiveApprovalSource(cfg userApprovalConfig, pipeName string) ([]string, bool) {
	strict := true
	if cfg.DefaultStrict != nil {
		strict = *cfg.DefaultStrict
	}
	source := cfg.FallbackApprovers
	if cfg.PluginApprovers != nil {
		if pluginApprovers, ok := cfg.PluginApprovers[pipeName]; ok {
			if pluginApprovers.hasApprovers {
				source = pluginApprovers.Approvers
			}
			if pluginApprovers.Strict != nil {
				strict = *pluginApprovers.Strict
			}
		}
	}
	return source, strict
}

func canonicalApprovalUser(user string) string {
	return strings.ToLower(strings.TrimSpace(user))
}

func userApprovalActionName(pipeName, command string) string {
	pipeName = strings.TrimSpace(pipeName)
	command = strings.TrimSpace(command)
	if pipeName == "" {
		return command
	}
	if command == "" {
		return pipeName
	}
	return pipeName + "/" + command
}

func userApprovalChoicePrompt(actionName string, approvers []string) string {
	choices := make([]string, 0, len(approvers))
	for i, approver := range approvers {
		choices = append(choices, fmt.Sprintf("%c) %s", 'a'+i, approver))
	}
	return fmt.Sprintf("Approval required for command %s. Select one approver: %s", actionName, strings.Join(choices, ", "))
}

func userApprovalApproverForChoice(choice string, approvers []string) (string, bool) {
	choice = strings.TrimSpace(choice)
	if len(choice) != 1 {
		return "", false
	}
	idx := int(choice[0] - 'a')
	if idx < 0 || idx >= len(approvers) {
		return "", false
	}
	return approvers[idx], true
}

func userApprovalYes(answer string) bool {
	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
