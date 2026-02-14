package main

import (
	"fmt"

	"github.com/lnxjedi/gopherbot/robot"
)

var defaultConfig = []byte(`
---
Help:
- Keywords: [ "say", "ask" ]
  Helptext:
  - "(bot), say everything - full test of Say*/Reply*/Send* methods"
- Keywords: [ "config" ]
  Helptext:
  - "(bot), go-config - exercise GetTaskConfig + RandomString"
- Keywords: [ "subscribe" ]
  Helptext:
  - "(bot), go-subscribe - exercise Subscribe/Unsubscribe"
- Keywords: [ "prompt" ]
  Helptext:
  - "(bot), go-prompts - exercise Prompt* methods (user/channel/thread variants)"
- Keywords: [ "memory" ]
  Helptext:
  - "(bot), go-memory-seed/go-memory-check/go-memory-thread-check - exercise Remember*/Recall context behavior"
  - "(bot), go-memory-datum-seed/go-memory-datum-check/go-memory-datum-checkin - exercise CheckoutDatum/UpdateDatum/CheckinDatum"
- Keywords: [ "identity", "parameter" ]
  Helptext:
  - "(bot), go-identity - exercise Get*Attribute + Set/GetParameter"
  - "(bot), go-parameter-addtask - SetParameter + AddTask pipeline visibility"
- Keywords: [ "pipeline", "admin", "elevate" ]
  Helptext:
  - "(bot), go-pipeline-ok/go-pipeline-fail/go-spawn-job - exercise pipeline-control methods"
  - "(bot), go-admin-check/go-elevate-check - exercise CheckAdmin + Elevate"
CommandMatchers:
- Regex: (?i:say everything)
  Command: sendmsg
- Regex: (?i:go-config)
  Command: configtest
- Regex: (?i:go-subscribe)
  Command: subscribe
- Regex: (?i:go-prompts)
  Command: prompts
- Regex: (?i:go-memory-seed)
  Command: memoryseed
- Regex: (?i:go-memory-check)
  Command: memorycheck
- Regex: (?i:go-memory-thread-check)
  Command: memorythreadcheck
- Regex: (?i:go-memory-datum-seed)
  Command: memorydatumseed
- Regex: (?i:go-memory-datum-check)
  Command: memorydatumcheck
- Regex: (?i:go-memory-datum-checkin)
  Command: memorydatumcheckin
- Regex: (?i:go-identity)
  Command: identity
- Regex: (?i:go-parameter-addtask)
  Command: parameteraddtask
- Regex: (?i:go-pipeline-ok)
  Command: pipelineok
- Regex: (?i:go-pipeline-fail)
  Command: pipelinefail
- Regex: (?i:go-spawn-job)
  Command: spawnjob
- Regex: (?i:go-admin-check)
  Command: admincheck
- Regex: (?i:go-elevate-check)
  Command: elevatecheck
- Regex: (?i:pc-add-cmd)
  Command: pipeaddcmd
- Regex: (?i:pc-final-cmd)
  Command: pipefinalcmd
- Regex: (?i:pc-fail-cmd)
  Command: pipefailcmd
AllowedHiddenCommands:
- sendmsg
Config:
  Openings:
  - "Not completely random 1"
  - "Not completely random 2"
`)

type goFullConfig struct {
	Openings []string
}

func showMemory(v string) string {
	if v == "" {
		return "<empty>"
	}
	return v
}

func Configure() *[]byte {
	return &defaultConfig
}

func PluginHandler(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	switch command {
	case "init":
		return robot.Normal
	case "sendmsg":
		msg := r.GetMessage()
		if msg == nil {
			return robot.Fail
		}
		r.Say("Regular Say")
		r.SayThread("SayThread, yeah")
		r.Reply("Regular Reply")
		r.ReplyThread("Reply in thread, yo")
		r.SendChannelMessage(msg.Channel, "Sending to the channel: %s", msg.Channel)
		r.SendUserMessage(msg.User, "Sending this message to user: %s", msg.User)
		r.SendUserChannelMessage(msg.User, msg.Channel, "Sending to user '%s' in channel: %s", msg.User, msg.Channel)
		r.SendChannelThreadMessage(msg.Channel, "0xDEADBEEF", "Sending to channel '%s' in thread: 0xDEADBEEF", msg.Channel)
		r.SendUserChannelThreadMessage(msg.User, msg.Channel, "0xDEADBEEF", "Sending to user '%s' in channel '%s' in thread: 0xDEADBEEF", msg.User, msg.Channel)
		return robot.Normal
	case "configtest":
		var cfg goFullConfig
		if ret := r.GetTaskConfig(&cfg); ret != robot.Ok {
			r.Say("No config available")
			return robot.Fail
		}
		r.Say(r.RandomString(cfg.Openings))
		return robot.Normal
	case "subscribe":
		sub := r.Subscribe()
		unsub := r.Unsubscribe()
		r.Say(fmt.Sprintf("SUBSCRIBE FLOW: %t/%t", sub, unsub))
		return robot.Normal
	case "prompts":
		msg := r.GetMessage()
		if msg == nil {
			return robot.Fail
		}
		p1, ret1 := r.PromptForReply("SimpleString", "Codename check: pick a mission codename.")
		if ret1 != robot.Ok {
			r.Say("PROMPT FLOW FAILED 1:%s", ret1)
			return robot.Fail
		}
		p2, ret2 := r.PromptThreadForReply("SimpleString", "Thread check: pick a favorite snack for launch.")
		if ret2 != robot.Ok {
			r.Say("PROMPT FLOW FAILED 2:%s", ret2)
			return robot.Fail
		}
		p3, ret3 := r.PromptUserForReply("SimpleString", msg.User, "DM check: name a secret moon base.")
		if ret3 != robot.Ok {
			r.Say("PROMPT FLOW FAILED 3:%s", ret3)
			return robot.Fail
		}
		p4, ret4 := r.PromptUserChannelForReply("SimpleString", msg.User, msg.Channel, "Channel check: describe launch weather in two words.")
		if ret4 != robot.Ok {
			r.Say("PROMPT FLOW FAILED 4:%s", ret4)
			return robot.Fail
		}
		threadID := "0xDEADBEEF"
		if msg.Incoming != nil && msg.Incoming.ThreadID != "" {
			threadID = msg.Incoming.ThreadID
		}
		p5, ret5 := r.PromptUserChannelThreadForReply("SimpleString", msg.User, msg.Channel, threadID, "Thread rally: choose a backup call sign.")
		if ret5 != robot.Ok {
			r.Say("PROMPT FLOW FAILED 5:%s", ret5)
			return robot.Fail
		}
		r.Say("PROMPT FLOW OK: %s | %s | %s | %s | %s", p1, p2, p3, p4, p5)
		return robot.Normal
	case "memoryseed":
		r.Remember("launch_snack", "saffron noodles", false)
		r.Remember("launch_snack", "solar soup", true)
		r.RememberContext("pad", "orbital-7")
		r.RememberThread("thread_note", "delta thread", false)
		r.RememberContextThread("mission", "aurora mission")
		r.Say("MEMORY SEED: done")
		return robot.Normal
	case "memorycheck":
		localMem := r.Recall("launch_snack", false)
		sharedMem := r.Recall("launch_snack", true)
		ctx := r.Recall("context:pad", false)
		threadMem := r.Recall("thread_note", false)
		threadCtx := r.Recall("context:mission", false)
		r.Say("MEMORY CHECK: local=%s shared=%s ctx=%s thread=%s threadctx=%s",
			showMemory(localMem), showMemory(sharedMem), showMemory(ctx), showMemory(threadMem), showMemory(threadCtx))
		return robot.Normal
	case "memorythreadcheck":
		localMem := r.Recall("launch_snack", false)
		sharedMem := r.Recall("launch_snack", true)
		ctx := r.Recall("context:pad", false)
		threadMem := r.Recall("thread_note", false)
		threadCtx := r.Recall("context:mission", false)
		r.Say("MEMORY THREAD CHECK: local=%s shared=%s ctx=%s thread=%s threadctx=%s",
			showMemory(localMem), showMemory(sharedMem), showMemory(ctx), showMemory(threadMem), showMemory(threadCtx))
		return robot.Normal
	case "memorydatumseed":
		memory := map[string]interface{}{}
		lockToken, _, retVal := r.CheckoutDatum("launch_manifest", &memory, true)
		if retVal != robot.Ok {
			r.Say("MEMORY DATUM SEED FAILED: %s", retVal)
			return robot.Fail
		}
		memory["mission"] = "opal-orbit"
		memory["vehicle"] = "heron-7"
		memory["status"] = "go"
		updateRet := r.UpdateDatum("launch_manifest", lockToken, memory)
		if updateRet != robot.Ok {
			r.Say("MEMORY DATUM SEED FAILED: %s", updateRet)
			return robot.Fail
		}
		r.Say("MEMORY DATUM SEED: update=%s", updateRet)
		return robot.Normal
	case "memorydatumcheck":
		memory := map[string]interface{}{}
		_, exists, retVal := r.CheckoutDatum("launch_manifest", &memory, false)
		if retVal != robot.Ok {
			r.Say("MEMORY DATUM CHECK FAILED: %s", retVal)
			return robot.Fail
		}
		mission := "<empty>"
		vehicle := "<empty>"
		status := "<empty>"
		if exists {
			if v, ok := memory["mission"]; ok {
				mission = fmt.Sprintf("%v", v)
			}
			if v, ok := memory["vehicle"]; ok {
				vehicle = fmt.Sprintf("%v", v)
			}
			if v, ok := memory["status"]; ok {
				status = fmt.Sprintf("%v", v)
			}
		}
		r.Say("MEMORY DATUM CHECK: mission=%s vehicle=%s status=%s", mission, vehicle, status)
		return robot.Normal
	case "memorydatumcheckin":
		memory := map[string]interface{}{}
		lockToken, exists, retVal := r.CheckoutDatum("launch_manifest", &memory, true)
		if retVal != robot.Ok {
			r.Say("MEMORY DATUM CHECKIN FAILED: %s", retVal)
			return robot.Fail
		}
		tokenPresent := lockToken != ""
		r.CheckinDatum("launch_manifest", lockToken)
		r.Say("MEMORY DATUM CHECKIN: exists=%t token=%t ret=Ok", exists, tokenPresent)
		return robot.Normal
	case "identity":
		botName := r.GetBotAttribute("name")
		senderFirst := r.GetSenderAttribute("firstName")
		bobFirst := r.GetUserAttribute("bob", "firstName")
		setOK := r.SetParameter("launch_phase", "phase-amber")
		phase := r.GetParameter("definitely_missing_param")
		r.Say("IDENTITY CHECK: bot=%s/%s sender=%s/%s bob=%s/%s set=%t param=%s",
			showMemory(botName.Attribute), botName.RetVal,
			showMemory(senderFirst.Attribute), senderFirst.RetVal,
			showMemory(bobFirst.Attribute), bobFirst.RetVal,
			setOK, showMemory(phase))
		return robot.Normal
	case "parameteraddtask":
		if !r.SetParameter("PIPELINE_SENTINEL", "nebula-42") {
			r.Say("SETPARAM ADDTASK: set=false")
			return robot.Fail
		}
		if r.AddTask("param-show") != robot.Ok {
			r.Say("SETPARAM ADDTASK: queue=false")
			return robot.Fail
		}
		r.Say("SETPARAM ADDTASK: queued")
		return robot.Normal
	case "pipeaddcmd":
		r.Say("PIPE ADD COMMAND: ran")
		return robot.Normal
	case "pipefinalcmd":
		r.Say("PIPE FINAL COMMAND: ran")
		return robot.Normal
	case "pipefailcmd":
		r.Say("PIPE FAIL COMMAND: ran")
		return robot.Normal
	case "pipelineok":
		if r.AddTask("pipeline-note", "add-task") != robot.Ok {
			r.Say("PIPELINE OK: addtask failed")
			return robot.Fail
		}
		if r.AddJob("pipe-job", "job-step") != robot.Ok {
			r.Say("PIPELINE OK: addjob failed")
			return robot.Fail
		}
		if r.AddCommand("gofull", "pc-add-cmd") != robot.Ok {
			r.Say("PIPELINE OK: addcommand failed")
			return robot.Fail
		}
		if r.FinalTask("pipeline-note", "final-task") != robot.Ok {
			r.Say("PIPELINE OK: finaltask failed")
			return robot.Fail
		}
		if r.FinalCommand("gofull", "pc-final-cmd") != robot.Ok {
			r.Say("PIPELINE OK: finalcommand failed")
			return robot.Fail
		}
		r.Say("PIPELINE OK: queued")
		return robot.Normal
	case "pipelinefail":
		if r.FailTask("pipeline-note", "fail-task") != robot.Ok {
			r.Say("PIPELINE FAIL: failtask failed")
			return robot.Fail
		}
		if r.FailCommand("gofull", "pc-fail-cmd") != robot.Ok {
			r.Say("PIPELINE FAIL: failcommand failed")
			return robot.Fail
		}
		r.Say("PIPELINE FAIL: armed")
		return robot.Fail
	case "spawnjob":
		if r.SpawnJob("pipe-spawn-job", "spawn-step") != robot.Ok {
			r.Say("SPAWN JOB: queue=false")
			return robot.Fail
		}
		return robot.Normal
	case "admincheck":
		r.Say("ADMIN CHECK: %t", r.CheckAdmin())
		return robot.Normal
	case "elevatecheck":
		r.Say("ELEVATE CHECK: %t", r.Elevate(true))
		return robot.Normal
	default:
		return robot.Fail
	}
}
