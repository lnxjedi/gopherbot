package bot

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lnxjedi/gopherbot/robot"
)

// func template(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
// 	r := m.(Robot)
// 	return
// }

// rotatelog (task rotate-log); rotate the log file when logging to file
func rotatelog(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	ext := ""
	if len(args) == 1 {
		ext = args[0]
	}
	return logRotate(ext)
}

// pause just adds a pause to the pipeline
func pause(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	if len(args) != 1 {
		m.Log(robot.Warn, "Pause called with wrong number or args, not pausing")
		return
	}
	seconds, err := strconv.Atoi(args[0])
	if err != nil {
		m.Log(robot.Warn, "Unable to parse integer argument for pause, not pausing")
		return
	}
	time.Sleep(time.Duration(seconds) * time.Second)
	return
}

// logtail - task tail-log; get the last 2k of pipeline log
func logtail(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	w := getLockedWorker(r.tid)
	hist := w.histName
	idx := w.runIndex
	w.Unlock()
	var buffer []byte
	retval, buffer = getLogTail(hist, idx)
	if retval == robot.Normal {
		r.Fixed().Say(string(buffer))
	}
	return
}

// sendmsg - task send-message just sends a message to the job/plugin channel
// functionally equivalent to status/say (bash tasks)
func sendmsg(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	r := m.GetMessage()
	if len(args) == 0 {
		m.Log(robot.Warn, "Empty status message")
		return
	}
	full := strings.Join(args, " ")
	ret := m.Say(full)
	if ret != robot.Ok {
		m.Log(robot.Error, "Failed sending message '%s' in channel '%s', return code: %d (%s)", full, r.Channel, ret, ret)
	}
	return
}

// logmail - task email-log; send the job log to one or more email
// addresses.
func logmail(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	if len(args) == 0 {
		defaultMail := false
		if len(r.Message.User) > 0 {
			sa := r.GetSenderAttribute("email")
			if sa.RetVal == robot.Ok {
				defaultMail = true
				args = []string{sa.Attribute}
			}
		}
		if !defaultMail {
			m.Log(robot.Error, "Email-log called with no addresses")
			return robot.Fail
		}
	}
	w := getLockedWorker(r.tid)
	hist := w.histName
	idx := w.runIndex
	w.Unlock()
	var buff []byte
	retval, buff = getLogMail(hist, idx)
	if retval != robot.Normal {
		return
	}
	body := new(bytes.Buffer)
	body.Write([]byte("<pre>\n"))
	body.Write(buff)
	body.Write([]byte("\n</pre>"))
	subject := fmt.Sprintf("Log for pipeline '%s', run %d", hist, idx)
	var ret robot.RetVal = robot.Ok
	for _, addr := range args {
		check := r.EmailAddress(addr, subject, body, true)
		if check != robot.Ok {
			ret = check
		}
	}
	if ret != robot.Ok {
		r.Log(robot.Error, "There was a problem emailing one or more pipeline logs, contact an administrator: %s", ret)
		return robot.Fail
	}
	return
}

func restart(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	pn := r.pipeName
	state.Lock()
	if state.shuttingDown {
		state.Unlock()
		Log(robot.Warn, "Restart triggered in pipeline '%s' with shutdown already in progress", pn)
		return
	}
	running := state.pipelinesRunning - 1
	state.shuttingDown = true
	state.restart = true
	state.Unlock()
	r.Log(robot.Info, "Restart triggered in pipeline '%s' with %d pipelines running (including this one)", pn, running)
	go stop()
	return
}

func quit(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	pn := r.pipeName
	state.Lock()
	if state.shuttingDown {
		state.Unlock()
		Log(robot.Warn, "Quit triggered in pipeline '%s' with shutdown already in progress", pn)
		return
	}
	running := state.pipelinesRunning - 1
	state.shuttingDown = true
	state.Unlock()
	r.Log(robot.Info, "Quit triggered in pipeline '%s' with %d pipelines running (including this one)", pn, running)
	go stop()
	return
}

func pauseBrainTask(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	w := getLockedWorker(r.tid)
	w.Unlock()
	resume := make(chan struct{})
	brainLocks.Lock()
	brainLocks.locks[w.id] = resume
	brainLocks.Unlock()
	pauseBrain(w.id, resume)
	return
}

func resume(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
	r := m.(Robot)
	w := getLockedWorker(r.tid)
	w.Unlock()
	brainLocks.Lock()
	if resume, ok := brainLocks.locks[w.id]; ok {
		close(resume)
		delete(brainLocks.locks, w.id)
	}
	brainLocks.Unlock()
	return
}

func init() {
	robot.RegisterTask("email-log", true, robot.TaskHandler{Handler: logmail})
	robot.RegisterTask("pause-brain", true, robot.TaskHandler{Handler: pauseBrainTask})
	robot.RegisterTask("pause", false, robot.TaskHandler{Handler: pause})
	robot.RegisterTask("restart-robot", true, robot.TaskHandler{Handler: restart})
	robot.RegisterTask("resume-brain", true, robot.TaskHandler{Handler: resume})
	robot.RegisterTask("robot-quit", true, robot.TaskHandler{Handler: quit})
	robot.RegisterTask("rotate-log", true, robot.TaskHandler{Handler: rotatelog})
	robot.RegisterTask("send-message", false, robot.TaskHandler{Handler: sendmsg})
	robot.RegisterTask("tail-log", false, robot.TaskHandler{Handler: logtail})
}
