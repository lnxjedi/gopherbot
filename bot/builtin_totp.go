package bot

import (
	"fmt"
	"sync"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
	"github.com/pquerna/otp/totp"
)

var timeoutLock sync.RWMutex
var lastElevate = make(map[string]time.Time)

type timeoutType int

const (
	idle timeoutType = iota
	absolute
)

type totp_user struct {
	User, Secret string
}

type totp_config struct {
	TimeoutSeconds int
	tf64           float64
	TimeoutType    string
	tt             timeoutType
	Users          []totp_user
}

var totpCfg = &totp_config{}
var totpUsers = make(map[string]string)

func init() {
	RegisterPlugin("builtin-totp", robot.PluginHandler{Handler: totp_elevate, Config: totpCfg})
}

func checkOTP(r robot.Robot, code string) (bool, robot.TaskRetVal) {
	m := r.GetMessage()
	secret, exists := totpUsers[m.User]
	if !exists {
		return false, robot.MechanismFail
	}
	m.Channel = ""
	lastValid := r.Recall("lastTOTP", false)
	if lastValid == code {
		r.Log(robot.Warn, "User %s attempted to re-use a TOTP code", m.User)
		return false, robot.Fail
	}
	valid := totp.Validate(code, secret)
	if valid {
		r.Remember("lastTOTP", code, false)
	}
	return valid, robot.Success
}

func getcode(gr robot.Robot, immediate bool) (retval robot.TaskRetVal) {
	m := gr.GetMessage()
	r := gr.(Robot)
	botFull := r.cfg.botinfo.FullName
	var prompt string
	if immediate {
		prompt = fmt.Sprintf("This command requires immediate elevation, please provide a TOTP code for '%s':", botFull)
	} else {
		prompt = fmt.Sprintf("This command requires elevation, please provide a TOTP code for '%s':", botFull)
	}
	rep, ret := r.PromptForReply("OTP", prompt)
	if ret != robot.Ok {
		rep, ret = r.Direct().PromptForReply("OTP", "Try again? I need a 6-digit launch code")
	}
	if ret == robot.Ok {
		ok, ret := checkOTP(r, rep)
		if ret != robot.Success {
			r.Say("There were technical issues validating your code, ask an administrator to check the log")
			return robot.MechanismFail
		}
		if ok {
			return robot.Success
		}
		r.Say("Invalid code")
		return robot.Fail
	}
	r.Log(robot.Error, "User \"%s\" failed to respond to TOTP token prompt", m.User)
	return robot.Fail
}

func totp_elevate(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	m := r.GetMessage()
	switch command {
	case "init":
		r.GetTaskConfig(&totpCfg)
		for _, user := range totpCfg.Users {
			totpUsers[user.User] = user.Secret
		}
	case "check":
		valid, ret := checkOTP(r, args[0])
		if ret != robot.Success {
			r.Say("Dang! I had a system problem verifying your code")
			return
		}
		if valid {
			r.Say("Looks good - you're ready to wreak havoc!")
		} else {
			r.Say("Sorry, that's not a valid code")
		}
	case "elevate":
		immediate := false
		switch args[0] {
		case "true", "True", "t", "T", "Yes", "yes", "Y":
			immediate = true
		}
		if totpCfg.TimeoutType == "absolute" {
			totpCfg.tt = absolute
		}
		totpCfg.tf64 = float64(totpCfg.TimeoutSeconds)
		now := time.Now().UTC()
		ask := false
		if immediate {
			retval = getcode(r, immediate)
		} else {
			timeoutLock.RLock()
			le, ok := lastElevate[m.User]
			timeoutLock.RUnlock()
			if ok {
				diff := now.Sub(le)
				if diff.Seconds() > totpCfg.tf64 {
					ask = true
				} else {
					retval = robot.Success
				}
			} else {
				ask = true
			}
			if ask {
				retval = getcode(r, immediate)
			}
		}
		if retval == robot.Success && totpCfg.tt == idle {
			timeoutLock.Lock()
			lastElevate[m.User] = now
			timeoutLock.Unlock()
		} else if retval == robot.Success && ask && totpCfg.tt == absolute {
			timeoutLock.Lock()
			lastElevate[m.User] = now
			timeoutLock.Unlock()
		}
		return
	}
	return
}
