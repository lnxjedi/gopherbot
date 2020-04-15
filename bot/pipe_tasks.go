package bot

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/lnxjedi/robot"
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
		m.Log(robot.Warn, "empty status message")
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
	if len(args) == 0 {
		m.Log(robot.Error, "email-log called with no addresses")
		return robot.Fail
	}
	r := m.(Robot)
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

func pause(m robot.Robot, args ...string) (retval robot.TaskRetVal) {
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
	}
	brainLocks.Unlock()
	return
}

func init() {
	RegisterTask("restart-robot", true, robot.TaskHandler{Handler: restart})
	RegisterTask("robot-quit", true, robot.TaskHandler{Handler: quit})
	RegisterTask("rotate-log", true, robot.TaskHandler{Handler: rotatelog})
	RegisterTask("pause-brain", true, robot.TaskHandler{Handler: pause})
	RegisterTask("resume-brain", true, robot.TaskHandler{Handler: resume})
	RegisterTask("send-message", false, robot.TaskHandler{Handler: sendmsg})
	RegisterTask("tail-log", false, robot.TaskHandler{Handler: logtail})
	RegisterTask("email-log", true, robot.TaskHandler{Handler: logmail})
}
