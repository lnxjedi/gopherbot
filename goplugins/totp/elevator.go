package totp

import (
	"bytes"
	"encoding/base32"
	"fmt"
	"math/rand"
	"sync"
	"time"

	otp "github.com/dgryski/dgoogauth"
	"github.com/lnxjedi/gopherbot/robot"
)

var timeoutLock sync.RWMutex
var lastElevate = make(map[string]time.Time)
var random = rand.New(rand.NewSource(time.Now().UnixNano()))

type timeoutType int

const (
	idle timeoutType = iota
	absolute
)

type config struct {
	TimeoutSeconds int
	tf64           float64
	TimeoutType    string
	tt             timeoutType
}

var cfg config

func checkOTP(r robot.Robot, code string) (bool, robot.TaskRetVal) {
	m := r.GetMessage()
	var userOTP otp.OTPConfig
	lock, exists, ret := r.CheckoutDatum(m.User, &userOTP, true)
	if ret != robot.Ok {
		r.CheckinDatum(m.User, lock)
		return false, robot.MechanismFail
	}
	if !exists {
		r.CheckinDatum(m.User, lock)
		return false, robot.MechanismFail
	}
	valid, err := userOTP.Authenticate(code)
	if err != nil {
		r.Log(robot.Error, "Problem authenticating launch code for user %s: %v", m.User, err)
		r.CheckinDatum(m.User, lock)
		return false, robot.MechanismFail
	}
	ret = r.UpdateDatum(m.User, lock, &userOTP)
	if ret != robot.Ok {
		r.Log(robot.Error, "Problem updating OTP for %s, failing", m.User)
		return false, robot.MechanismFail
	}
	return valid, robot.Success
}

func getcode(r robot.Robot, immediate bool) (retval robot.TaskRetVal) {
	m := r.GetMessage()
	dm := ""
	if m.Channel != "" {
		dm = " - I'll message you directly"
	}
	if immediate {
		r.Say("This command requires immediate elevation" + dm)
	} else {
		r.Say("This command requires elevation" + dm)
	}
	r.Pause(1)
	rep, ret := r.Direct().PromptForReply("OTP", "Please provide your totp launch code")
	if ret != robot.Ok {
		rep, ret = r.Direct().PromptForReply("OTP", "Try again? I need a 6-digit launch code")
	}
	if ret == robot.Ok {
		ok, ret := checkOTP(r, rep)
		if ret != robot.Success {
			r.Direct().Say("There were technical issues validating your code, ask an administrator to check the log")
			return robot.MechanismFail
		}
		if ok {
			return robot.Success
		}
		r.Direct().Say("Invalid code")
		return robot.Fail
	}
	r.Log(robot.Error, "User \"%s\" failed to respond to TOTP token prompt", m.User)
	return robot.Fail
}

func elevate(r robot.Robot, command string, args ...string) (retval robot.TaskRetVal) {
	m := r.GetMessage()
	switch command {
	case "send":
		var userOTP otp.OTPConfig
		updated := false
		lock, exists, ret := r.CheckoutDatum(m.User, &userOTP, true)
		if ret != robot.Ok {
			r.Say("Yikes! - Something went wrong with my brain, have an admin check my log")
			return
		}
		defer func() {
			if updated {
				ret = r.UpdateDatum(m.User, lock, &userOTP)
				if ret != robot.Ok {
					r.Log(robot.Error, "Couldn't save OTP config")
					r.Reply("Good grief, I'm having trouble remembering your launch codes - have somebody check my log")
				}
			} else {
				// Well-behaved plugins will always do a CheckinDatum when the datum hasn't been updated,
				// in case there's another thread waiting.
				r.CheckinDatum(m.User, lock)
			}
		}()
		if exists {
			r.Reply("I've already sent you the launch codes, contact an administrator if you're having problems")
			return
		}
		otpb := make([]byte, 10)
		random.Read(otpb)
		userOTP.Secret = base32.StdEncoding.EncodeToString(otpb)
		userOTP.WindowSize = 2
		userOTP.DisallowReuse = []int{}
		var codeMail bytes.Buffer
		fmt.Fprintf(&codeMail, "For your authenticator:\n%s\n", userOTP.Secret)
		// Sending email takes longer than the timeout, so we check it in and check
		// out again after.
		r.CheckinDatum(m.User, lock)
		if ret = r.Email("Your launch codes - if you print this email, please chew it up and swallow it", &codeMail); ret != robot.Ok {
			r.Reply("There was a problem sending your launch codes, contact an administrator")
			return
		}
		lock, _, ret = r.CheckoutDatum(m.User, &userOTP, true)
		updated = true
		r.Reply("I've emailed your launch codes - please delete it promptly")
		return
	case "elevate":
		immediate := false
		switch args[0] {
		case "true", "True", "t", "T", "Yes", "yes", "Y":
			immediate = true
		}
		cfg := &config{}
		r.GetTaskConfig(&cfg)
		if cfg.TimeoutType == "absolute" {
			cfg.tt = absolute
		}
		cfg.tf64 = float64(cfg.TimeoutSeconds)
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
				if diff.Seconds() > cfg.tf64 {
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
		if retval == robot.Success && cfg.tt == idle {
			timeoutLock.Lock()
			lastElevate[m.User] = now
			timeoutLock.Unlock()
		} else if retval == robot.Success && ask && cfg.tt == absolute {
			timeoutLock.Lock()
			lastElevate[m.User] = now
			timeoutLock.Unlock()
		}
		return
	}
	return
}

var totphandler = robot.PluginHandler{
	Handler: elevate,
	Config:  &config{},
}
